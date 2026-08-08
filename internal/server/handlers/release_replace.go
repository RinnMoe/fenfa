package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
)

// AdminReplaceReleaseFile replaces the installer payload of an existing release.
// The release ID and public URLs stay stable; only file and metadata fields change.
func AdminReplaceReleaseFile(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		releaseID := c.Param("releaseID")
		var release store.Release
		if err := db.Where("id = ?", releaseID).First(&release).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "release not found"}})
			return
		}
		oldStoragePath := release.StoragePath

		ctx, err := loadReleaseVariantContext(db, release)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "variant not found"}})
			return
		}

		file, err := c.FormFile("app_file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "missing app_file"}})
			return
		}

		ext := detectFileExt(file.Filename)
		if !isAllowedVariantFileExt(ctx.Variant.Platform, ext) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "file extension mismatch"}})
			return
		}

		version := strings.TrimSpace(c.PostForm("version"))
		if version == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "version is required"}})
			return
		}
		if len(version) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "version is too long"}})
			return
		}

		var build int64
		if raw := c.PostForm("build"); raw != "" {
			build, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || build < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "invalid build"}})
				return
			}
		}

		channel := strings.TrimSpace(c.PostForm("channel"))
		if len(channel) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "channel is too long"}})
			return
		}
		minOS := strings.TrimSpace(c.PostForm("min_os"))
		if len(minOS) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "min_os is too long"}})
			return
		}
		changelog := c.PostForm("changelog")
		forceUpdate := formBool(c.PostForm("force_update"))
		iconBase64 := c.PostForm("icon_base64")

		var settings store.SystemSettings
		db.Where("id = ?", "default").First(&settings)
		storageType := settings.StorageType
		if storageType == "" {
			storageType = "local"
		}

		baseName := sanitizeFilename(strings.TrimSuffix(filepath.Base(file.Filename), ext))
		if baseName == "" {
			baseName = "app"
		}
		filename := fmt.Sprintf("%s_%s%s", baseName, time.Now().Format("20060102_150405"), ext)

		tmpPath := ""
		finalPath := ""
		var sizeBytes int64
		var sha string
		newStoragePath := ""

		if storageType == "s3" {
			if settings.S3Endpoint == "" || settings.S3Bucket == "" || settings.S3AccessKey == "" || settings.S3SecretKey == "" {
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "S3 storage is not configured"}})
				return
			}
			tmpPath, sizeBytes, sha, err = saveReplaceFileTemp(file, "", "replace-*"+ext)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "IO", "message": err.Error()}})
				return
			}
			defer os.Remove(tmpPath)
			s3Key := fmt.Sprintf("uploads/%s/%s/%s/%s", ctx.Product.ID, ctx.Variant.ID, release.ID, filename)
			newStoragePath = "s3://" + s3Key

			f, err := os.Open(tmpPath)
			if err != nil {
				os.Remove(tmpPath)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "IO", "message": err.Error()}})
				return
			}
			if err := newS3StorageClient(settings).Upload(c.Request.Context(), s3Key, f, contentTypeForExt(ext)); err != nil {
				f.Close()
				os.Remove(tmpPath)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "S3", "message": err.Error()}})
				return
			}
			f.Close()
		} else {
			targetDir := ""
			if release.StoragePath != "" && !isS3StoragePath(release.StoragePath) {
				targetDir = filepath.Dir(release.StoragePath)
			} else {
				targetDir = filepath.Join("uploads", ctx.Product.ID, ctx.Variant.ID, release.ID)
			}
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "IO", "message": err.Error()}})
				return
			}
			tmpPath, sizeBytes, sha, err = saveReplaceFileTemp(file, targetDir, ".replace-*"+ext)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "IO", "message": err.Error()}})
				return
			}
			finalPath = filepath.Join(targetDir, filename)
			if err := os.Rename(tmpPath, finalPath); err != nil {
				os.Remove(tmpPath)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "IO", "message": err.Error()}})
				return
			}
			tmpPath = finalPath
			newStoragePath = filepath.ToSlash(finalPath)
		}

		parsedID := ""
		if ctx.Variant.Platform == "ios" || ctx.Variant.Platform == "android" {
			var parsed gin.H
			var parseErr error
			if ctx.Variant.Platform == "ios" {
				parsed, parseErr = parseIPAFromPath(tmpPath, file.Filename, file.Size)
			} else {
				parsed, parseErr = parseAPKFromPath(tmpPath, file.Filename, file.Size)
			}
			if parseErr != nil {
				cleanupReplacedPayload(db, newStoragePath)
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "PARSE_ERROR", "message": parseErr.Error()}})
				return
			}
			if ctx.Variant.Platform == "ios" {
				parsedID = getString(parsed, "bundle_id")
			} else {
				parsedID = getString(parsed, "package_name")
			}
			if ctx.Variant.Identifier != "" && parsedID != "" && parsedID != ctx.Variant.Identifier {
				cleanupReplacedPayload(db, newStoragePath)
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "new package identifier does not match the variant"}})
				return
			}
		}

		updates := map[string]any{
			"version":      version,
			"build":        build,
			"channel":      channel,
			"min_os":       minOS,
			"changelog":    changelog,
			"force_update": forceUpdate,
			"file_name":    filename,
			"file_ext":     ext,
			"mime_type":    contentTypeForExt(ext),
			"size_bytes":   sizeBytes,
			"sha256":       sha,
			"storage_path": newStoragePath,
		}

		txErr := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&release).Updates(updates).Error; err != nil {
				return err
			}
			if ctx.Variant.Identifier == "" && parsedID != "" {
				if err := tx.Model(&store.Variant{}).Where("id = ?", ctx.Variant.ID).Update("identifier", parsedID).Error; err != nil {
					return err
				}
			}
			if ctx.Variant.Platform == "ios" {
				if err := tx.Where("release_id = ?", release.ID).Delete(&store.ProvisioningProfile{}).Error; err != nil {
					return err
				}
				saveProvisioningProfile(tx, release.ID, tmpPath)
			}
			return nil
		})
		if txErr != nil {
			cleanupReplacedPayload(db, newStoragePath)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": txErr.Error()}})
			return
		}

		// Old payload is removed only after the new one is committed.
		if oldStoragePath != "" && oldStoragePath != newStoragePath {
			deleteStoredPayload(db, oldStoragePath)
		}

		if iconBase64 != "" {
			updateProductIconFromBase64(db, ctx, iconBase64)
		}

		if err := db.Where("id = ?", release.ID).First(&release).Error; err == nil {
			writeReleaseMetaFile(db, release)
		}

		domains := resolvePublicDomains(c, db, cfg)
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{
			"release": adminReleasePayload(release),
			"urls":    releaseActions(productPathKey(ctx.Product), ctx.Variant.Platform, domains, release.ID),
		}})
	}
}

func saveReplaceFileTemp(file *multipart.FileHeader, dir, pattern string) (string, int64, string, error) {
	src, err := file.Open()
	if err != nil {
		return "", 0, "", err
	}
	defer src.Close()

	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", 0, "", err
	}
	path := tmp.Name()
	h := sha256.New()
	n, err := io.Copy(tmp, io.TeeReader(src, h))
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
		return "", 0, "", err
	}
	return path, n, hex.EncodeToString(h.Sum(nil)), nil
}

func cleanupReplacedPayload(db *gorm.DB, storagePath string) {
	if storagePath == "" {
		return
	}
	if isS3StoragePath(storagePath) {
		if err := deleteS3Object(context.Background(), db, storagePath); err != nil {
			log.Printf("WARN: failed to delete staged s3 object %s: %v", storagePath, err)
		}
		return
	}
	_ = os.Remove(storagePath)
}

func deleteStoredPayload(db *gorm.DB, storagePath string) {
	if isS3StoragePath(storagePath) {
		if err := deleteS3Object(context.Background(), db, storagePath); err != nil {
			log.Printf("WARN: failed to delete replaced s3 object %s: %v", storagePath, err)
		}
		return
	}
	if err := os.Remove(storagePath); err != nil && !os.IsNotExist(err) {
		log.Printf("WARN: failed to delete replaced file %s: %v", storagePath, err)
	}
}

func updateProductIconFromBase64(db *gorm.DB, ctx releaseVariantContext, iconB64 string) {
	if idx := strings.Index(iconB64, "base64,"); idx >= 0 {
		iconB64 = iconB64[idx+7:]
	}
	data, err := base64.StdEncoding.DecodeString(iconB64)
	if err != nil || len(data) == 0 {
		return
	}
	_ = os.MkdirAll(filepath.Join("uploads", ctx.Product.ID), 0o755)
	iconPath := filepath.ToSlash(filepath.Join("uploads", ctx.Product.ID, "icon.png"))
	if err := os.WriteFile(iconPath, data, 0o644); err != nil {
		return
	}
	_ = db.Model(&store.Product{}).Where("id = ?", ctx.Product.ID).Update("icon_path", iconPath).Error
	if ctx.Variant.LegacyAppID != nil && *ctx.Variant.LegacyAppID != "" {
		_ = db.Model(&store.App{}).Where("id = ?", *ctx.Variant.LegacyAppID).Update("icon_path", iconPath).Error
	}
}

// writeReleaseMetaFile keeps the local release snapshot consistent after edits.
func writeReleaseMetaFile(db *gorm.DB, release store.Release) {
	if release.StoragePath == "" || isS3StoragePath(release.StoragePath) {
		return
	}
	appDir := filepath.Dir(release.StoragePath)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		log.Printf("WARN: failed to create meta dir %s: %v", appDir, err)
		return
	}

	platform := ""
	identifier := ""
	productID := release.VariantID
	variantID := release.VariantID
	legacyAppID := release.AppID
	if ctx, err := loadReleaseVariantContext(db, release); err == nil {
		platform = ctx.Variant.Platform
		identifier = ctx.Variant.Identifier
		productID = ctx.Variant.ProductID
		variantID = ctx.Variant.ID
		if ctx.Variant.LegacyAppID != nil {
			legacyAppID = *ctx.Variant.LegacyAppID
		}
	}

	meta := map[string]any{
		"app_id":     legacyAppID,
		"product_id": productID,
		"variant_id": variantID,
		"release_id": release.ID,
		"platform":   platform,
		"identifier": identifier,
		"version":    release.Version,
		"build":      release.Build,
		"min_os":     release.MinOS,
		"size":       release.SizeBytes,
		"sha256":     release.SHA256,
		"changelog":  release.Changelog,
		"updated_at": time.Now().UTC(),
	}
	if err := os.WriteFile(filepath.Join(appDir, "meta.json"), []byte(metaJSON(meta)), 0o644); err != nil {
		log.Printf("WARN: failed to write meta.json (release=%s): %v", release.ID, err)
	}
}

func metaJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
