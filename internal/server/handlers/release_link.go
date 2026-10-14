package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
	"gorm.io/gorm"
)

func validateExternalURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) > 4096 || strings.ContainsAny(raw, "\r\n\t") {
		return "", fmt.Errorf("invalid download URL")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return "", fmt.Errorf("download URL must be an absolute HTTPS URL without credentials or fragments")
	}
	return u.String(), nil
}

type linkedReleaseInput struct {
	URL         string `json:"external_url"`
	Version     string `json:"version"`
	Build       int64  `json:"build"`
	Channel     string `json:"channel"`
	MinOS       string `json:"min_os"`
	Changelog   string `json:"changelog"`
	ForceUpdate bool   `json:"force_update"`
	FileName    string `json:"file_name"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
}

// AdminCreateLinkedRelease records a publisher-supplied link without fetching
// it. Keeping the stable /d URL preserves visibility checks and counters.
func AdminCreateLinkedRelease(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		var input linkedReleaseInput
		if c.ShouldBindJSON(&input) != nil {
			releaseLinkError(c, 400, "invalid release JSON")
			return
		}
		external, err := validateExternalURL(input.URL)
		if err != nil {
			releaseLinkError(c, 400, err.Error())
			return
		}
		input.Version = strings.TrimSpace(input.Version)
		input.Channel = strings.TrimSpace(input.Channel)
		input.SHA256 = strings.TrimSpace(input.SHA256)
		if input.Version == "" || len(input.Version) > 64 || input.Build < 0 || input.SizeBytes < 0 || len(input.Channel) > 64 || len(input.MinOS) > 64 || (input.SHA256 != "" && !validDeltaHash(input.SHA256)) {
			releaseLinkError(c, 400, "invalid version, build, size or SHA256")
			return
		}
		target, err := loadUploadTarget(db, c.Param("variantID"))
		if err != nil {
			releaseLinkError(c, 404, "variant not found")
			return
		}
		u, _ := url.Parse(external)
		filename := strings.TrimSpace(input.FileName)
		if filename == "" {
			filename = path.Base(u.Path)
		}
		ext := detectFileExt(filename)
		if !isAllowedVariantFileExt(target.Variant.Platform, ext) && input.FileName == "" {
			ext = "." + target.Variant.InstallerType
			if !isAllowedVariantFileExt(target.Variant.Platform, ext) {
				ext = map[string]string{"android": ".apk", "ios": ".ipa", "windows": ".exe", "macos": ".dmg", "linux": ".appimage"}[target.Variant.Platform]
			}
			filename = "app" + ext
		}
		if !isAllowedVariantFileExt(target.Variant.Platform, ext) || len(filename) > 255 || path.Base(filename) != filename || strings.ContainsAny(filename, "\\\r\n\t") {
			releaseLinkError(c, 400, "filename does not match variant platform")
			return
		}
		release := store.Release{ID: "rel_" + randHexN(8), VariantID: target.Variant.ID, AppID: target.LegacyAppID, ExternalURL: external, Version: input.Version, Build: input.Build, Channel: input.Channel, MinOS: input.MinOS, Changelog: input.Changelog, ForceUpdate: input.ForceUpdate, FileName: filename, FileExt: ext, MimeType: contentTypeForExt(ext), SizeBytes: input.SizeBytes, SHA256: input.SHA256}
		if err := db.Create(&release).Error; err != nil {
			releaseLinkError(c, 500, "cannot create release")
			return
		}
		domains := resolvePublicDomains(c, db, cfg)
		actions := releaseActions(productPathKey(target.Product), target.Variant.Platform, domains, release.ID)
		c.JSON(201, gin.H{"ok": true, "data": gin.H{"release": adminReleasePayload(release), "urls": gin.H{"page": actions["release_page"]}, "actions": actions}})
	}
}

func releaseLinkError(c *gin.Context, status int, message string) {
	code := "BAD_REQUEST"
	if status == 404 {
		code = "NOT_FOUND"
	}
	if status >= 500 {
		code = "DB_ERROR"
	}
	c.JSON(status, gin.H{"ok": false, "error": gin.H{"code": code, "message": message}})
}
