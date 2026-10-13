package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	fenfaemail "github.com/openprx/fenfa/internal/email"
	"github.com/openprx/fenfa/internal/store"
)

// trimDomainURL removes trailing slashes from a domain URL.
func trimDomainURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

func decodeUploadAllowedOrigins(raw string) []string {
	var origins []string
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &origins) != nil {
		return []string{}
	}
	return origins
}

func normalizeUploadAllowedOrigins(origins []string) ([]string, error) {
	normalized := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if strings.TrimSpace(origin) == "" {
			continue
		}
		value, err := config.NormalizeOrigin(origin)
		if err != nil {
			return nil, fmt.Errorf("upload_allowed_origins contains an invalid origin: %q", origin)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized, nil
}

// SystemSettingsResponse represents the API response for system settings
type SystemSettingsResponse struct {
	PrimaryDomain    string   `json:"primary_domain"`
	SecondaryDomains []string `json:"secondary_domains"`
	Organization     string   `json:"organization"`
	AdminPath        string   `json:"admin_path"`
	UpdatedAt        string   `json:"updated_at"`

	// Email notification configuration (SMTP password is never returned).
	EmailEnabled    bool   `json:"email_enabled"`
	EmailConfigured bool   `json:"email_configured"`
	SMTPHost        string `json:"smtp_host,omitempty"`
	SMTPPort        int    `json:"smtp_port"`
	SMTPEncryption  string `json:"smtp_encryption"`
	SMTPUsername    string `json:"smtp_username,omitempty"`
	SMTPPasswordSet bool   `json:"smtp_password_set"`
	SMTPFrom        string `json:"smtp_from,omitempty"`
	SMTPTo          string `json:"smtp_to,omitempty"`

	// Storage configuration
	StorageType          string   `json:"storage_type"`
	UploadDomain         string   `json:"upload_domain,omitempty"`
	UploadAllowedOrigins []string `json:"upload_allowed_origins"`

	// S3/R2 configuration (no secrets)
	S3Configured bool   `json:"s3_configured"`
	S3Endpoint   string `json:"s3_endpoint,omitempty"`
	S3Bucket     string `json:"s3_bucket,omitempty"`
	S3PublicURL  string `json:"s3_public_url,omitempty"`

	// Apple API configuration status (not credentials)
	AppleConfigured bool   `json:"apple_configured"`
	AppleKeyID      string `json:"apple_key_id,omitempty"`  // Only show ID, not full credentials
	AppleTeamID     string `json:"apple_team_id,omitempty"` // Only show ID
}

// SystemSettingsUpdateRequest represents the API request for updating system settings
type SystemSettingsUpdateRequest struct {
	PrimaryDomain    string   `json:"primary_domain"`
	SecondaryDomains []string `json:"secondary_domains"`
	Organization     string   `json:"organization"`
	AdminPath        *string  `json:"admin_path,omitempty"`

	// Storage configuration (optional)
	StorageType          *string   `json:"storage_type,omitempty"`
	UploadDomain         *string   `json:"upload_domain,omitempty"`
	UploadAllowedOrigins *[]string `json:"upload_allowed_origins,omitempty"`

	// S3/R2 configuration (optional)
	S3Endpoint  *string `json:"s3_endpoint,omitempty"`
	S3Bucket    *string `json:"s3_bucket,omitempty"`
	S3AccessKey *string `json:"s3_access_key,omitempty"`
	S3SecretKey *string `json:"s3_secret_key,omitempty"`
	S3PublicURL *string `json:"s3_public_url,omitempty"`

	// Apple API credentials (optional)
	AppleKeyID      string `json:"apple_key_id,omitempty"`
	AppleIssuerID   string `json:"apple_issuer_id,omitempty"`
	ApplePrivateKey string `json:"apple_private_key,omitempty"` // PEM format
	AppleTeamID     string `json:"apple_team_id,omitempty"`

	// Email notification configuration (password is only updated when non-empty).
	EmailEnabled   *bool   `json:"email_enabled,omitempty"`
	SMTPHost       *string `json:"smtp_host,omitempty"`
	SMTPPort       *int    `json:"smtp_port,omitempty"`
	SMTPEncryption *string `json:"smtp_encryption,omitempty"`
	SMTPUsername   *string `json:"smtp_username,omitempty"`
	SMTPPassword   *string `json:"smtp_password,omitempty"`
	SMTPFrom       *string `json:"smtp_from,omitempty"`
	SMTPTo         *string `json:"smtp_to,omitempty"`
}

type testEmailSettingsRequest struct {
	SMTPHost       string `json:"smtp_host"`
	SMTPPort       int    `json:"smtp_port"`
	SMTPEncryption string `json:"smtp_encryption"`
	SMTPUsername   string `json:"smtp_username"`
	SMTPPassword   string `json:"smtp_password"`
	SMTPFrom       string `json:"smtp_from"`
	SMTPTo         string `json:"smtp_to"`
}

// GetSystemSettings returns the current system settings
func GetSystemSettings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings store.SystemSettings
		if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"ok": false,
				"error": gin.H{
					"code":    "SETTINGS_NOT_FOUND",
					"message": "System settings not found",
				},
			})
			return
		}

		// Parse secondary domains from JSON string
		var secondaryDomains []string
		if settings.SecondaryDomains != "" {
			_ = json.Unmarshal([]byte(settings.SecondaryDomains), &secondaryDomains)
		}
		if secondaryDomains == nil {
			secondaryDomains = []string{}
		}

		response := systemSettingsResponse(settings, secondaryDomains)

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": response,
		})
	}
}

// UpdateSystemSettings updates the system settings
func UpdateSystemSettings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SystemSettingsUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"ok": false,
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid request body",
				},
			})
			return
		}

		var settings store.SystemSettings
		settingsErr := db.Where("id = ?", "default").First(&settings).Error
		if settingsErr != nil && settingsErr != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok": false,
				"error": gin.H{
					"code":    "DATABASE_ERROR",
					"message": "Failed to query settings",
				},
			})
			return
		}

		adminPath := "admin"
		if settingsErr == nil && settings.AdminPath != "" {
			candidate := config.NormalizeAdminPath(settings.AdminPath)
			if config.ValidateAdminPath(candidate) == nil {
				adminPath = candidate
			}
		}
		if req.AdminPath != nil {
			candidate := config.NormalizeAdminPath(*req.AdminPath)
			if err := config.ValidateAdminPath(candidate); err != nil {
				writeSettingsValidationError(c, err)
				return
			}
			adminPath = candidate
		}

		// Normalize domain URLs: trim trailing slashes
		req.PrimaryDomain = trimDomainURL(req.PrimaryDomain)
		for i, d := range req.SecondaryDomains {
			req.SecondaryDomains[i] = trimDomainURL(d)
		}
		if req.UploadDomain != nil {
			v := trimDomainURL(*req.UploadDomain)
			req.UploadDomain = &v
		}
		var uploadAllowedOrigins []string
		if settingsErr == nil {
			uploadAllowedOrigins = decodeUploadAllowedOrigins(settings.UploadAllowedOrigins)
		}
		if req.UploadAllowedOrigins != nil {
			normalized, err := normalizeUploadAllowedOrigins(*req.UploadAllowedOrigins)
			if err != nil {
				writeSettingsValidationError(c, err)
				return
			}
			uploadAllowedOrigins = normalized
		}
		if req.S3PublicURL != nil {
			v := trimDomainURL(*req.S3PublicURL)
			req.S3PublicURL = &v
		}

		// Validate required fields
		if req.PrimaryDomain == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"ok": false,
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Primary domain is required",
				},
			})
			return
		}

		if req.Organization == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"ok": false,
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Organization is required",
				},
			})
			return
		}

		// Convert secondary domains to JSON string
		secondaryDomainsJSON := "[]"
		if req.SecondaryDomains != nil && len(req.SecondaryDomains) > 0 {
			if b, err := json.Marshal(req.SecondaryDomains); err == nil {
				secondaryDomainsJSON = string(b)
			}
		}

		// Update or create settings
		err := settingsErr
		if err == gorm.ErrRecordNotFound {
			// Create new settings
			settings = store.SystemSettings{
				ID:                   "default",
				PrimaryDomain:        req.PrimaryDomain,
				SecondaryDomains:     secondaryDomainsJSON,
				UploadAllowedOrigins: mustMarshalStringList(uploadAllowedOrigins),
				Organization:         req.Organization,
				AdminPath:            adminPath,
				SMTPPort:             fenfaemail.DefaultSMTPPort,
				SMTPEncryption:       fenfaemail.EncryptionStartTLS,
				UpdatedAt:            time.Now(),
				AppleKeyID:           req.AppleKeyID,
				AppleIssuerID:        req.AppleIssuerID,
				ApplePrivateKey:      req.ApplePrivateKey,
				AppleTeamID:          req.AppleTeamID,
			}
			if req.StorageType != nil {
				settings.StorageType = *req.StorageType
			}
			if req.UploadDomain != nil {
				settings.UploadDomain = *req.UploadDomain
			}
			if req.S3Endpoint != nil {
				settings.S3Endpoint = *req.S3Endpoint
			}
			if req.S3Bucket != nil {
				settings.S3Bucket = *req.S3Bucket
			}
			if req.S3AccessKey != nil {
				settings.S3AccessKey = *req.S3AccessKey
			}
			if req.S3SecretKey != nil {
				settings.S3SecretKey = *req.S3SecretKey
			}
			if req.S3PublicURL != nil {
				settings.S3PublicURL = *req.S3PublicURL
			}
			applyEmailSettings(&settings, &req)
			if err := validateEmailSettings(settings); err != nil {
				writeSettingsValidationError(c, err)
				return
			}
			if err := db.Create(&settings).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"ok": false,
					"error": gin.H{
						"code":    "DATABASE_ERROR",
						"message": "Failed to create settings",
					},
				})
				return
			}
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok": false,
				"error": gin.H{
					"code":    "DATABASE_ERROR",
					"message": "Failed to query settings",
				},
			})
			return
		} else {
			// Update existing settings
			emailSettings := settings
			applyEmailSettings(&emailSettings, &req)
			if err := validateEmailSettings(emailSettings); err != nil {
				writeSettingsValidationError(c, err)
				return
			}
			updates := map[string]interface{}{
				"primary_domain":    req.PrimaryDomain,
				"secondary_domains": secondaryDomainsJSON,
				"organization":      req.Organization,
				"admin_path":        adminPath,
				"updated_at":        time.Now(),
			}
			// Only update storage fields if provided
			if req.StorageType != nil {
				updates["storage_type"] = *req.StorageType
			}
			if req.UploadDomain != nil {
				updates["upload_domain"] = *req.UploadDomain
			}
			if req.UploadAllowedOrigins != nil {
				updates["upload_allowed_origins"] = mustMarshalStringList(uploadAllowedOrigins)
			}
			if req.S3Endpoint != nil {
				updates["s3_endpoint"] = *req.S3Endpoint
			}
			if req.S3Bucket != nil {
				updates["s3_bucket"] = *req.S3Bucket
			}
			if req.S3AccessKey != nil {
				updates["s3_access_key"] = *req.S3AccessKey
			}
			if req.S3SecretKey != nil {
				updates["s3_secret_key"] = *req.S3SecretKey
			}
			if req.S3PublicURL != nil {
				updates["s3_public_url"] = *req.S3PublicURL
			}
			addEmailSettingsUpdates(updates, &req, emailSettings)
			// Only update Apple credentials if provided (allow partial updates)
			if req.AppleKeyID != "" {
				updates["apple_key_id"] = req.AppleKeyID
			}
			if req.AppleIssuerID != "" {
				updates["apple_issuer_id"] = req.AppleIssuerID
			}
			if req.ApplePrivateKey != "" {
				updates["apple_private_key"] = req.ApplePrivateKey
			}
			if req.AppleTeamID != "" {
				updates["apple_team_id"] = req.AppleTeamID
			}
			if err := db.Model(&settings).Updates(updates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"ok": false,
					"error": gin.H{
						"code":    "DATABASE_ERROR",
						"message": "Failed to update settings",
					},
				})
				return
			}
		}

		// Re-read settings to get updated values
		db.Where("id = ?", "default").First(&settings)

		// Return updated settings
		var updatedSecondaryDomains []string
		if settings.SecondaryDomains != "" {
			_ = json.Unmarshal([]byte(settings.SecondaryDomains), &updatedSecondaryDomains)
		}
		if updatedSecondaryDomains == nil {
			updatedSecondaryDomains = []string{}
		}

		response := systemSettingsResponse(settings, updatedSecondaryDomains)

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": response,
		})
	}
}

func systemSettingsResponse(settings store.SystemSettings, secondaryDomains []string) SystemSettingsResponse {
	storageType := settings.StorageType
	if storageType == "" {
		storageType = "local"
	}
	adminPath := config.NormalizeAdminPath(settings.AdminPath)
	if config.ValidateAdminPath(adminPath) != nil {
		adminPath = "admin"
	}
	smtpConfig := fenfaemail.SMTPConfigFromSettings(settings)
	return SystemSettingsResponse{
		PrimaryDomain:        settings.PrimaryDomain,
		SecondaryDomains:     secondaryDomains,
		Organization:         settings.Organization,
		AdminPath:            adminPath,
		UpdatedAt:            settings.UpdatedAt.Format(time.RFC3339),
		EmailEnabled:         settings.EmailEnabled,
		EmailConfigured:      fenfaemail.ValidateSMTPConfig(smtpConfig) == nil,
		SMTPHost:             smtpConfig.Host,
		SMTPPort:             smtpConfig.Port,
		SMTPEncryption:       smtpConfig.Encryption,
		SMTPUsername:         smtpConfig.Username,
		SMTPPasswordSet:      settings.SMTPPassword != "",
		SMTPFrom:             smtpConfig.From,
		SMTPTo:               smtpConfig.To,
		StorageType:          storageType,
		UploadDomain:         settings.UploadDomain,
		UploadAllowedOrigins: decodeUploadAllowedOrigins(settings.UploadAllowedOrigins),
		S3Configured:         settings.S3Endpoint != "" && settings.S3Bucket != "" && settings.S3AccessKey != "" && settings.S3SecretKey != "",
		S3Endpoint:           settings.S3Endpoint,
		S3Bucket:             settings.S3Bucket,
		S3PublicURL:          settings.S3PublicURL,
		AppleConfigured:      settings.AppleKeyID != "" && settings.AppleIssuerID != "" && settings.ApplePrivateKey != "",
		AppleKeyID:           settings.AppleKeyID,
		AppleTeamID:          settings.AppleTeamID,
	}
}

func mustMarshalStringList(values []string) string {
	b, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func applyEmailSettings(settings *store.SystemSettings, req *SystemSettingsUpdateRequest) {
	if req.EmailEnabled != nil {
		settings.EmailEnabled = *req.EmailEnabled
	}
	if req.SMTPHost != nil {
		settings.SMTPHost = strings.TrimSpace(*req.SMTPHost)
	}
	if req.SMTPPort != nil {
		settings.SMTPPort = *req.SMTPPort
	}
	if req.SMTPEncryption != nil {
		settings.SMTPEncryption = strings.ToLower(strings.TrimSpace(*req.SMTPEncryption))
	}
	if req.SMTPUsername != nil {
		settings.SMTPUsername = strings.TrimSpace(*req.SMTPUsername)
	}
	if req.SMTPPassword != nil && *req.SMTPPassword != "" {
		settings.SMTPPassword = *req.SMTPPassword
	}
	if req.SMTPFrom != nil {
		settings.SMTPFrom = strings.TrimSpace(*req.SMTPFrom)
	}
	if req.SMTPTo != nil {
		settings.SMTPTo = strings.TrimSpace(*req.SMTPTo)
	}
	if settings.SMTPPort == 0 {
		settings.SMTPPort = fenfaemail.DefaultSMTPPort
	}
	if strings.TrimSpace(settings.SMTPEncryption) == "" {
		settings.SMTPEncryption = fenfaemail.EncryptionStartTLS
	}
}

func addEmailSettingsUpdates(updates map[string]interface{}, req *SystemSettingsUpdateRequest, settings store.SystemSettings) {
	if req.EmailEnabled != nil {
		updates["email_enabled"] = settings.EmailEnabled
	}
	if req.SMTPHost != nil {
		updates["smtp_host"] = settings.SMTPHost
	}
	if req.SMTPPort != nil {
		updates["smtp_port"] = settings.SMTPPort
	}
	if req.SMTPEncryption != nil {
		updates["smtp_encryption"] = settings.SMTPEncryption
	}
	if req.SMTPUsername != nil {
		updates["smtp_username"] = settings.SMTPUsername
	}
	if req.SMTPPassword != nil && *req.SMTPPassword != "" {
		updates["smtp_password"] = settings.SMTPPassword
	}
	if req.SMTPFrom != nil {
		updates["smtp_from"] = settings.SMTPFrom
	}
	if req.SMTPTo != nil {
		updates["smtp_to"] = settings.SMTPTo
	}
}

func validateEmailSettings(settings store.SystemSettings) error {
	if !settings.EmailEnabled {
		return nil
	}
	return fenfaemail.ValidateSMTPConfig(fenfaemail.SMTPConfigFromSettings(settings))
}

func writeSettingsValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"ok": false,
		"error": gin.H{
			"code":    "INVALID_REQUEST",
			"message": err.Error(),
		},
	})
}

// TestEmailSettings sends a test message using the current form values. It
// does not persist those values; an empty password reuses the saved password.
func TestEmailSettings(db *gorm.DB, senders ...fenfaemail.Sender) gin.HandlerFunc {
	var sender fenfaemail.Sender = fenfaemail.SMTPMailer{}
	if len(senders) > 0 && senders[0] != nil {
		sender = senders[0]
	}

	return func(c *gin.Context) {
		var req testEmailSettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeSettingsValidationError(c, err)
			return
		}

		var saved store.SystemSettings
		err := db.Where("id = ?", "default").First(&saved).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DATABASE_ERROR", "message": "Failed to load email settings"}})
			return
		}
		if req.SMTPPassword == "" {
			req.SMTPPassword = saved.SMTPPassword
		}

		config := fenfaemail.NormalizeSMTPConfig(fenfaemail.SMTPConfig{
			Host:       req.SMTPHost,
			Port:       req.SMTPPort,
			Encryption: req.SMTPEncryption,
			Username:   req.SMTPUsername,
			Password:   req.SMTPPassword,
			From:       req.SMTPFrom,
			To:         req.SMTPTo,
		})
		if err := fenfaemail.ValidateSMTPConfig(config); err != nil {
			writeSettingsValidationError(c, err)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		if err := sender.Send(ctx, config, fenfaemail.Message{
			Subject: "[Fenfa] SMTP test email",
			Body:    "This is a test email from Fenfa. SMTP settings are working.",
		}); err != nil {
			log.Printf("[email] SMTP test failed")
			c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": gin.H{"code": "EMAIL_SEND_FAILED", "message": "Failed to send test email"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"sent": true}})
	}
}

// GetUploadConfig returns minimal upload configuration for the frontend
func GetUploadConfig(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings store.SystemSettings
		db.Where("id = ?", "default").First(&settings)

		storageType := settings.StorageType
		if storageType == "" {
			storageType = "local"
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"storage_type":  storageType,
				"upload_domain": settings.UploadDomain,
			},
		})
	}
}
