package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	fenfauth "github.com/openprx/fenfa/internal/server/auth"
	"github.com/openprx/fenfa/internal/store"
)

type adminCredentialsRequest struct {
	Username             string `json:"username"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation,omitempty"`
}

type adminAccountUpdateRequest struct {
	Username             string `json:"username"`
	CurrentPassword      string `json:"current_password"`
	NewPassword          string `json:"new_password"`
	PasswordConfirmation string `json:"password_confirmation,omitempty"`
}

var errAdminSetupComplete = errors.New("admin setup is already complete")

// AdminAuthStatus reports whether the first administrator still needs to be
// created. It intentionally contains no account details.
func AdminAuthStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var count int64
		if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load authentication status")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"setup_required": count == 0,
			},
		})
	}
}

// AdminSetup creates the first administrator. It is only accepted while the
// database has no admin users, which keeps the endpoint one-time use.
func AdminSetup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminCredentialsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
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

		hash, err := fenfauth.HashPassword(req.Password)
		if err != nil {
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
		err = db.Transaction(func(tx *gorm.DB) error {
			var count int64
			if err := tx.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errAdminSetupComplete
			}
			return tx.Create(&user).Error
		})
		if errors.Is(err, errAdminSetupComplete) {
			writeAuthError(c, http.StatusConflict, "SETUP_COMPLETE", "Admin setup is already complete")
			return
		}
		if err != nil {
			writeAuthError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create administrator")
			return
		}

		writeAdminLoginResponse(c, db, user)
	}
}

// AdminLogin validates an administrator account and starts a web session.
func AdminLogin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminCredentialsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAuthError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || req.Password == "" {
			writeAuthError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
			return
		}

		var user store.AdminUser
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil || !fenfauth.CheckPassword(user.PasswordHash, req.Password) {
			writeAuthError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
			return
		}

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
		fenfauth.RevokeSession(db, adminSessionCookie(c))
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
