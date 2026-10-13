package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

const (
	SessionCookieName = "fenfa_admin_session"
	SessionTTL        = 24 * time.Hour
	UploadSessionTTL  = 2 * time.Hour
	UserContextKey    = "fenfa_admin_user"
)

var ErrInvalidSession = errors.New("invalid admin session")

// HashPassword returns a bcrypt password hash suitable for database storage.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password with a stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CreateSession persists a hash of a new opaque session token.
func CreateSession(db *gorm.DB, userID string) (string, time.Time, error) {
	return createSession(db, userID, "admin", SessionTTL)
}

// CreateUploadSession creates an in-memory-only browser credential for upload
// requests sent to a separately configured upload domain.
func CreateUploadSession(db *gorm.DB, userID string) (string, time.Time, error) {
	return createSession(db, userID, "upload", UploadSessionTTL)
}

func createSession(db *gorm.DB, userID, scope string, ttl time.Duration) (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(buf)
	expiresAt := time.Now().Add(ttl)
	session := store.AdminSession{
		TokenHash: sessionHash(rawToken),
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&session).Error; err != nil {
		return "", time.Time{}, fmt.Errorf("create admin session: %w", err)
	}
	return rawToken, expiresAt, nil
}

// LookupSession validates a raw cookie value and returns the associated user.
func LookupSession(db *gorm.DB, rawToken string) (*store.AdminUser, error) {
	return lookupSession(db, rawToken, "admin")
}

// LookupUploadSession validates a browser upload credential without granting
// access to admin APIs.
func LookupUploadSession(db *gorm.DB, rawToken string) (*store.AdminUser, error) {
	return lookupSession(db, rawToken, "upload")
}

func lookupSession(db *gorm.DB, rawToken, requiredScope string) (*store.AdminUser, error) {
	if rawToken == "" {
		return nil, ErrInvalidSession
	}

	var session store.AdminSession
	if err := db.Where("token_hash = ?", sessionHash(rawToken)).First(&session).Error; err != nil {
		return nil, ErrInvalidSession
	}
	// Sessions created before scopes were added are admin sessions.
	if session.Scope == "" {
		session.Scope = "admin"
	}
	if session.Scope != requiredScope {
		return nil, ErrInvalidSession
	}
	if !session.ExpiresAt.After(time.Now()) {
		_ = db.Delete(&session).Error
		return nil, ErrInvalidSession
	}

	var user store.AdminUser
	if err := db.Where("id = ?", session.UserID).First(&user).Error; err != nil {
		return nil, ErrInvalidSession
	}
	return &user, nil
}

// RevokeSession invalidates a single raw session value.
func RevokeSession(db *gorm.DB, rawToken string) {
	if rawToken == "" {
		return
	}
	_ = db.Where("token_hash = ?", sessionHash(rawToken)).Delete(&store.AdminSession{}).Error
}

// RevokeUserSessions invalidates all sessions for a user, for example after a
// password change.
func RevokeUserSessions(db *gorm.DB, userID string) {
	RevokeUserSessionsByScope(db, userID, "")
}

// RevokeUserSessionsByScope invalidates all sessions for a user that have the
// requested scope. An empty scope revokes every session.
func RevokeUserSessionsByScope(db *gorm.DB, userID, scope string) {
	if userID == "" {
		return
	}
	query := db.Where("user_id = ?", userID)
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}
	_ = query.Delete(&store.AdminSession{}).Error
}

// SetSessionCookie sends the session as an HttpOnly same-site cookie.
func SetSessionCookie(c *gin.Context, rawToken string, expiresAt time.Time) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, rawToken, int(time.Until(expiresAt).Seconds()), "/", "", secure, true)
}

// ClearSessionCookie removes the browser's session cookie.
func ClearSessionCookie(c *gin.Context) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, "", -1, "/", "", secure, true)
}

// EnsureBootstrapAdmin creates the first account from deployment-provided
// credentials. If no credentials are supplied, the UI can perform first-run
// setup through the protected-by-empty-database setup endpoint.
func EnsureBootstrapAdmin(db *gorm.DB, username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return fmt.Errorf("admin username and password must be provided together")
	}

	var count int64
	if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if count > 0 {
		return store.MarkAdminSetupClaimed(db)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	user := store.AdminUser{
		ID:           NewID("usr_"),
		Username:     username,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	return store.MarkAdminSetupClaimed(db)
}

func sessionHash(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return fmt.Sprintf("%x", sum[:])
}

// NewID creates a compact random identifier for authentication records.
func NewID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return prefix + "00000000"
	}
	return prefix + fmt.Sprintf("%x", buf)
}
