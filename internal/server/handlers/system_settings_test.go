package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	fenfaemail "github.com/openprx/fenfa/internal/email"
	"github.com/openprx/fenfa/internal/store"
)

type recordingEmailSender struct {
	config  fenfaemail.SMTPConfig
	message fenfaemail.Message
	err     error
}

func (sender *recordingEmailSender) Send(_ context.Context, config fenfaemail.SMTPConfig, message fenfaemail.Message) error {
	sender.config = config
	sender.message = message
	return sender.err
}

func performSettingsJSONRequest(t *testing.T, handler gin.HandlerFunc, method string, body map[string]any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal settings request: %v", err)
	}
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(method, "/admin/api/settings", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode settings response: %v body=%s", err, rr.Body.String())
	}
	return rr, payload
}

func emailSettingsUpdateBody(password string) map[string]any {
	body := map[string]any{
		"primary_domain":    "https://download.example.com",
		"secondary_domains": []string{},
		"organization":      "Fenfa",
		"email_enabled":     true,
		"smtp_host":         "smtp.example.com",
		"smtp_port":         587,
		"smtp_encryption":   "starttls",
		"smtp_username":     "smtp-user@example.com",
		"smtp_from":         "noreply@example.com",
		"smtp_to":           "admin@example.com",
	}
	if password != "" {
		body["smtp_password"] = password
	}
	return body
}

func TestSystemSettingsEmailRoundTripMasksPasswordAndPreservesBlankUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)

	first, payload := performSettingsJSONRequest(t, UpdateSystemSettings(db), http.MethodPut, emailSettingsUpdateBody("saved-secret"))
	if first.Code != http.StatusOK || payload["ok"] != true {
		t.Fatalf("expected settings update success, got %d body=%v", first.Code, payload)
	}
	data := payload["data"].(map[string]any)
	if _, exposed := data["smtp_password"]; exposed {
		t.Fatalf("SMTP password must not be exposed in response: %v", data)
	}
	if data["smtp_password_set"] != true || data["email_enabled"] != true {
		t.Fatalf("unexpected email response: %v", data)
	}

	var settings store.SystemSettings
	if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.SMTPPassword != "saved-secret" {
		t.Fatalf("expected password to be saved, got %q", settings.SMTPPassword)
	}

	secondBody := emailSettingsUpdateBody("")
	second, _ := performSettingsJSONRequest(t, UpdateSystemSettings(db), http.MethodPut, secondBody)
	if second.Code != http.StatusOK {
		t.Fatalf("expected blank password update to succeed, got %d body=%s", second.Code, second.Body.String())
	}
	if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if settings.SMTPPassword != "saved-secret" {
		t.Fatalf("expected blank password to preserve saved value, got %q", settings.SMTPPassword)
	}
}

func TestSystemSettingsEmailValidationOnlyBlocksEnabledConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)

	body := emailSettingsUpdateBody("")
	body["email_enabled"] = false
	body["smtp_host"] = ""
	body["smtp_from"] = "not-an-email"
	body["smtp_to"] = ""
	rr, _ := performSettingsJSONRequest(t, UpdateSystemSettings(db), http.MethodPut, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected disabled incomplete email settings to save, got %d body=%s", rr.Code, rr.Body.String())
	}

	body["email_enabled"] = true
	rr, payload := performSettingsJSONRequest(t, UpdateSystemSettings(db), http.MethodPut, body)
	if rr.Code != http.StatusBadRequest || payload["ok"] != false {
		t.Fatalf("expected enabled incomplete email settings to fail, got %d body=%v", rr.Code, payload)
	}
}

func TestTestEmailSettingsUsesSavedPasswordAndCurrentForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	if err := db.Model(&store.SystemSettings{}).Where("id = ?", "default").Updates(map[string]any{
		"smtp_password": "saved-secret",
	}).Error; err != nil {
		t.Fatalf("seed saved SMTP password: %v", err)
	}

	sender := &recordingEmailSender{}
	body := map[string]any{
		"smtp_host":       "smtp.current.example.com",
		"smtp_port":       465,
		"smtp_encryption": "ssl",
		"smtp_username":   "current@example.com",
		"smtp_password":   "",
		"smtp_from":       "noreply@example.com",
		"smtp_to":         "admin@example.com",
	}
	rr, payload := performSettingsJSONRequest(t, TestEmailSettings(db, sender), http.MethodPost, body)
	if rr.Code != http.StatusOK || payload["ok"] != true {
		t.Fatalf("expected test email success, got %d body=%v", rr.Code, payload)
	}
	if sender.config.Host != "smtp.current.example.com" || sender.config.Port != 465 || sender.config.Encryption != "ssl" {
		t.Fatalf("test email did not use current form values: %+v", sender.config)
	}
	if sender.config.Password != "saved-secret" {
		t.Fatalf("expected saved password fallback, got %q", sender.config.Password)
	}
	if sender.message.Subject != "[Fenfa] SMTP test email" {
		t.Fatalf("unexpected test subject: %q", sender.message.Subject)
	}
}

func TestTestEmailSettingsReturnsBadGatewayOnSenderFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	sender := &recordingEmailSender{err: errors.New("fake SMTP failure")}
	rr, payload := performSettingsJSONRequest(t, TestEmailSettings(db, sender), http.MethodPost, map[string]any{
		"smtp_host":       "smtp.example.com",
		"smtp_port":       587,
		"smtp_encryption": "starttls",
		"smtp_from":       "noreply@example.com",
		"smtp_to":         "admin@example.com",
	})
	if rr.Code != http.StatusBadGateway || payload["ok"] != false {
		t.Fatalf("expected SMTP failure response, got %d body=%v", rr.Code, payload)
	}
}

func TestSystemSettingsUploadAllowedOriginsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	body := map[string]any{
		"primary_domain":         "https://download.example.com",
		"secondary_domains":      []string{},
		"organization":           "Fenfa",
		"admin_path":             "admin",
		"upload_allowed_origins": []string{"HTTPS://Admin.Example.com/", "https://admin.example.com"},
	}
	rr, payload := performSettingsJSONRequest(t, UpdateSystemSettings(db), http.MethodPut, body)
	if rr.Code != http.StatusOK || payload["ok"] != true {
		t.Fatalf("expected upload origin settings update success, got %d body=%v", rr.Code, payload)
	}
	data := payload["data"].(map[string]any)
	origins, ok := data["upload_allowed_origins"].([]any)
	if !ok || len(origins) != 1 || origins[0] != "https://admin.example.com" {
		t.Fatalf("expected normalized unique upload origins, got %v", data["upload_allowed_origins"])
	}

	var settings store.SystemSettings
	if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.UploadAllowedOrigins != `["https://admin.example.com"]` {
		t.Fatalf("unexpected stored upload origins: %q", settings.UploadAllowedOrigins)
	}

	body["upload_allowed_origins"] = []string{"https://admin.example.com/path"}
	bad, _ := performSettingsJSONRequest(t, UpdateSystemSettings(db), http.MethodPut, body)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid upload origin to return 400, got %d body=%s", bad.Code, bad.Body.String())
	}
}
