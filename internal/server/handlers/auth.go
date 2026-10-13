package handlers

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	fenfauth "github.com/openprx/fenfa/internal/server/auth"
	"github.com/openprx/fenfa/internal/store"
)

type adminCredentialsRequest struct {
	Username             string `json:"username"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation,omitempty"`
	SetupToken           string `json:"setup_token,omitempty"`
}

type adminAccountUpdateRequest struct {
	Username             string `json:"username"`
	CurrentPassword      string `json:"current_password"`
	NewPassword          string `json:"new_password"`
	PasswordConfirmation string `json:"password_confirmation,omitempty"`
}

const (
	adminSetupMaxBodyBytes   = 16 * 1024
	adminSetupRateLimit      = 5
	adminSetupRateWindow     = 10 * time.Minute
	adminLoginMaxBodyBytes   = 16 * 1024
	adminLoginIPRateLimit    = 10
	adminLoginIPRateWindow   = 10 * time.Minute
	adminLoginUserRateLimit  = 5
	adminLoginUserRateWindow = 15 * time.Minute
)

// AdminAuthStatus reports whether the first administrator still needs to be
// created. It intentionally contains no account details.
func AdminAuthStatus(db *gorm.DB, cfgs ...*config.Config) gin.HandlerFunc {
	var cfg *config.Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return func(c *gin.Context) {
		var count int64
		if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load authentication status")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"setup_required": count == 0 && cfg != nil && strings.TrimSpace(cfg.Auth.SetupToken) != "",
			},
		})
	}
}

// AdminSetup creates the first administrator. It is only available when an
// operator has configured a setup token and is accepted once per database.
func AdminSetup(db *gorm.DB, cfgs ...*config.Config) gin.HandlerFunc {
	var cfg *config.Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	limiter := newRateLimiter(adminSetupRateLimit, adminSetupRateWindow)
	return func(c *gin.Context) {
		if cfg == nil || strings.TrimSpace(cfg.Auth.SetupToken) == "" {
			c.Status(http.StatusNotFound)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, adminSetupMaxBodyBytes)
		if allowed, retryAfter := limiter.allow(c.ClientIP(), time.Now()); !allowed {
			writeRateLimitResponse(c, "too many setup attempts", retryAfter)
			return
		}

		var req adminCredentialsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		providedToken := c.GetHeader("X-Setup-Token")
		if strings.TrimSpace(providedToken) == "" {
			providedToken = req.SetupToken
		}
		if !constantTimeTokenEqual(cfg.Auth.SetupToken, providedToken) {
			writeAuthError(c, http.StatusUnauthorized, "SETUP_TOKEN_REQUIRED", "A valid setup token is required")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if err := validateAdminCredentials(req.Username, req.Password); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		if req.PasswordConfirmation != "" && req.PasswordConfirmation != req.Password {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Passwords do not match")
			return
		}

		var count int64
		if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load authentication status")
			return
		}
		if count > 0 {
			writeAuthError(c, http.StatusConflict, "SETUP_COMPLETE", "Admin setup is already complete")
			return
		}

		claimed, err := store.ClaimAdminSetup(db)
		if err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to reserve administrator setup")
			return
		}
		if !claimed {
			writeAuthError(c, http.StatusConflict, "SETUP_COMPLETE", "Admin setup is already complete")
			return
		}

		// Recheck after the atomic claim so an administrator created between the
		// initial count and the claim cannot trigger bcrypt work or a second row.
		if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
			_ = store.ReleaseAdminSetup(db)
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load authentication status")
			return
		}
		if count > 0 {
			writeAuthError(c, http.StatusConflict, "SETUP_COMPLETE", "Admin setup is already complete")
			return
		}

		hash, err := fenfauth.HashPassword(req.Password)
		if err != nil {
			_ = store.ReleaseAdminSetup(db)
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		user := store.AdminUser{
			ID:           fenfauth.NewID("usr_"),
			Username:     req.Username,
			PasswordHash: hash,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.Create(&user).Error; err != nil {
			_ = store.ReleaseAdminSetup(db)
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create administrator")
			return
		}

		writeAdminLoginResponse(c, db, user)
	}
}

// AdminLogin validates an administrator account and starts a web session.
func AdminLogin(db *gorm.DB) gin.HandlerFunc {
	ipLimiter := newRateLimiter(adminLoginIPRateLimit, adminLoginIPRateWindow)
	usernameLimiter := newRateLimiter(adminLoginUserRateLimit, adminLoginUserRateWindow)
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, adminLoginMaxBodyBytes)
		clientIP := c.ClientIP()
		if allowed, retryAfter := ipLimiter.allow(clientIP, time.Now()); !allowed {
			writeRateLimitResponse(c, "too many login attempts", retryAfter)
			return
		}

		var req adminCredentialsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		usernameKey := strings.ToLower(req.Username)
		if usernameKey != "" {
			if allowed, retryAfter := usernameLimiter.allow(usernameKey, time.Now()); !allowed {
				writeRateLimitResponse(c, "too many login attempts for this account", retryAfter)
				return
			}
		}
		if req.Username == "" || req.Password == "" {
			writeAuthError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
			return
		}

		var user store.AdminUser
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil || !fenfauth.CheckPassword(user.PasswordHash, req.Password) {
			writeAuthError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
			return
		}

		ipLimiter.reset(clientIP)
		usernameLimiter.reset(usernameKey)
		writeAdminLoginResponse(c, db, user)
	}
}

// AdminMe returns the current account represented by the session cookie.
func AdminMe(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := adminUserFromContext(c)
		if !ok {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Admin session required")
			return
		}
		uploadToken, _, err := fenfauth.CreateUploadSession(db, user.ID)
		if err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create upload session")
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"username": user.Username, "upload_token": uploadToken}})
	}
}

// AdminLogout revokes the current session and clears the browser cookie.
func AdminLogout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken := adminSessionCookie(c)
		if user, err := fenfauth.LookupSession(db, rawToken); err == nil {
			fenfauth.RevokeUserSessionsByScope(db, user.ID, "upload")
		}
		fenfauth.RevokeSession(db, rawToken)
		fenfauth.ClearSessionCookie(c)
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"logged_out": true}})
	}
}

// AdminUpdateAccount changes the current administrator's username and/or
// password, then rotates the current session.
func AdminUpdateAccount(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := adminUserFromContext(c)
		if !ok {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Admin session required")
			return
		}

		var req adminAccountUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" {
			req.Username = user.Username
		}
		if err := validateUsername(req.Username); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		if req.NewPassword == "" && req.Username == user.Username {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Username or new password is required")
			return
		}
		if !fenfauth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
			writeAuthError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Current password is incorrect")
			return
		}
		if req.NewPassword != "" {
			if err := validatePassword(req.NewPassword); err != nil {
				writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}
			if req.PasswordConfirmation != "" && req.PasswordConfirmation != req.NewPassword {
				writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Passwords do not match")
				return
			}
		}

		updates := map[string]interface{}{
			"username":   req.Username,
			"updated_at": time.Now(),
		}
		if req.NewPassword != "" {
			hash, err := fenfauth.HashPassword(req.NewPassword)
			if err != nil {
				writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}
			updates["password_hash"] = hash
		}
		if err := db.Model(&store.AdminUser{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				writeAuthError(c, http.StatusConflict, "USERNAME_TAKEN", "Username is already in use")
				return
			}
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update administrator")
			return
		}

		fenfauth.RevokeUserSessions(db, user.ID)
		var updated store.AdminUser
		if err := db.Where("id = ?", user.ID).First(&updated).Error; err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to reload administrator")
			return
		}
		writeAdminLoginResponse(c, db, updated)
	}
}

func writeAdminLoginResponse(c *gin.Context, db *gorm.DB, user store.AdminUser) {
	rawToken, expiresAt, err := fenfauth.CreateSession(db, user.ID)
	if err != nil {
		writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create admin session")
		return
	}
	uploadToken, _, err := fenfauth.CreateUploadSession(db, user.ID)
	if err != nil {
		fenfauth.RevokeSession(db, rawToken)
		writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create upload session")
		return
	}
	fenfauth.SetSessionCookie(c, rawToken, expiresAt)
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": gin.H{
			"username":     user.Username,
			"expires_at":   expiresAt.Format(time.RFC3339),
			"upload_token": uploadToken,
		},
	})
}

func adminUserFromContext(c *gin.Context) (*store.AdminUser, bool) {
	value, exists := c.Get(fenfauth.UserContextKey)
	if !exists {
		return nil, false
	}
	user, ok := value.(*store.AdminUser)
	return user, ok && user != nil
}

func adminSessionCookie(c *gin.Context) string {
	value, _ := c.Cookie(fenfauth.SessionCookieName)
	return value
}

func validateAdminCredentials(username, password string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	return validatePassword(password)
}

func validateUsername(username string) error {
	if username == "" || utf8.RuneCountInString(username) > 128 || strings.ContainsAny(username, "\r\n\t ") {
		return errors.New("username must be 1-128 characters without whitespace")
	}
	return nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func writeAuthError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"ok": false, "error": gin.H{"code": code, "message": message}})
}

func constantTimeTokenEqual(expected, provided string) bool {
	if expected == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
