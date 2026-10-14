package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

// AdminDownloadRelease serves the stored installer regardless of publish state.
func AdminDownloadRelease(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		releaseID := c.Param("releaseID")
		var rel store.Release
		if err := db.Where("id = ?", releaseID).First(&rel).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "release not found"}})
			return
		}

		if rel.ExternalURL != "" {
			if _, err := validateExternalURL(rel.ExternalURL); err != nil {
				c.Status(404)
				return
			}
			c.Header("Cache-Control", "no-store")
			c.Redirect(http.StatusFound, rel.ExternalURL)
			return
		}
		if strings.HasPrefix(rel.StoragePath, "s3://") {
			var settings store.SystemSettings
			if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "S3 storage is not configured"}})
				return
			}
			if settings.S3Endpoint == "" || settings.S3Bucket == "" || settings.S3AccessKey == "" || settings.S3SecretKey == "" {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "S3 storage is not configured"}})
				return
			}
			key := strings.TrimPrefix(rel.StoragePath, "s3://")
			body, err := newS3StorageClient(settings).Get(c.Request.Context(), key)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "file missing"}})
				return
			}
			defer body.Close()
			ext := rel.FileExt
			if ext == "" {
				ext = detectFileExt(key)
			}
			ctype := contentTypeForExt(ext)
			filename := rel.FileName
			if filename == "" {
				filename = filepath.Base(key)
			}
			c.Header("Content-Type", ctype)
			c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
			c.Header("Cache-Control", "no-transform")
			c.DataFromReader(http.StatusOK, -1, ctype, body, nil)
			return
		}

		if _, err := os.Stat(rel.StoragePath); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "file missing"}})
			return
		}

		ext := rel.FileExt
		if ext == "" {
			ext = detectFileExt(rel.StoragePath)
		}
		c.Header("Content-Type", contentTypeForExt(ext))
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(rel.StoragePath)+"\"")
		c.Header("Cache-Control", "no-transform")
		c.File(rel.StoragePath)
	}
}
