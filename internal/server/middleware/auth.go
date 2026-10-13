package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	fenfauth "github.com/openprx/fenfa/internal/server/auth"
	"github.com/openprx/fenfa/internal/store"
)

// RequireAdmin authenticates a web-admin session cookie. The legacy admin
// token remains accepted so existing API clients can migrate without downtime.
func RequireAdmin(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return requireSessionOrToken(db, cfg, "admin")
}

// RequireUpload accepts only an upload-scoped bearer token. In particular, an
// admin session cookie is not accepted here, so cross-origin browser uploads
// cannot turn the admin cookie into an upload authorization.
func RequireUpload(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Auth-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "upload token required"}})
			return
		}
		if user, lookupErr := fenfauth.LookupUploadSession(db, token); lookupErr == nil {
			c.Set(fenfauth.UserContextKey, user)
			c.Next()
			return
		}
		if allowedScope(cfg, "upload", token) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "error": gin.H{"code": "FORBIDDEN", "message": "invalid upload token"}})
	}
}

// RequireAdminPath rejects dynamic admin routes that do not match the path
// persisted in system settings. This lets the path change without rebuilding
// the Gin router.
func RequireAdminPath(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := loadSystemSettings(db)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "404 Not Found"}})
			return
		}
		current := config.NormalizeAdminPath(settings.AdminPath)
		if config.ValidateAdminPath(current) != nil {
			current = "admin"
		}
		requested := config.NormalizeAdminPath(c.Param("adminPath"))
		if requested != current {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "404 Not Found"}})
			return
		}
		c.Next()
	}
}

func loadSystemSettings(db *gorm.DB) (store.SystemSettings, error) {
	var settings store.SystemSettings
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		err = db.Where("id = ?", "default").First(&settings).Error
		if err == nil || err == gorm.ErrRecordNotFound {
			return settings, err
		}
		if !strings.Contains(strings.ToLower(err.Error()), "locked") {
			return settings, err
		}
		time.Sleep(25 * time.Millisecond)
	}
	return settings, err
}

// RequireToken is retained for callers that explicitly need the old header
// behavior. New protected routes should use RequireAdmin or RequireUpload.
func RequireToken(cfg *config.Config, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Auth-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "missing token"}})
			return
		}
		if !allowedScope(cfg, scope, token) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "error": gin.H{"code": "FORBIDDEN", "message": "invalid token"}})
			return
		}
		c.Next()
	}
}

func requireSessionOrToken(db *gorm.DB, cfg *config.Config, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rawToken, err := c.Cookie(fenfauth.SessionCookieName); err == nil && rawToken != "" {
			if user, lookupErr := fenfauth.LookupSession(db, rawToken); lookupErr == nil {
				c.Set(fenfauth.UserContextKey, user)
				c.Next()
				return
			}
		}

		token := c.GetHeader("X-Auth-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "admin session required"}})
			return
		}
		if scope == "upload" {
			if user, lookupErr := fenfauth.LookupUploadSession(db, token); lookupErr == nil {
				c.Set(fenfauth.UserContextKey, user)
				c.Next()
				return
			}
		}
		if allowedScope(cfg, scope, token) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "error": gin.H{"code": "FORBIDDEN", "message": "invalid token"}})
	}
}

func allowedScope(cfg *config.Config, scope, token string) bool {
	if cfg == nil || strings.TrimSpace(token) == "" {
		return false
	}
	// Admin scope: only admin tokens
	if scope == "admin" {
		for _, t := range cfg.Auth.AdminTokens {
			if t == token {
				return true
			}
		}
		return false
	}
	// Other scopes (e.g., upload): allow either upload tokens or admin tokens
	for _, t := range cfg.Auth.UploadTokens {
		if t == token {
			return true
		}
	}
	for _, t := range cfg.Auth.AdminTokens {
		if t == token {
			return true
		}
	}
	return false
}
