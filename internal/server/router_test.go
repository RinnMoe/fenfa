package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
	web "github.com/openprx/fenfa/internal/web"
)

func newTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := store.AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	cfg := config.Default()
	cfg.Server.PrimaryDomain = "https://download.example.com"
	cfg.Auth.AdminTokens = []string{"admin-token"}
	cfg.Auth.UploadTokens = []string{"upload-token"}

	product := store.Product{ID: "prd_router", Slug: "router-demo", Name: "Router Demo", Published: true}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	variant := store.Variant{ID: "var_router", ProductID: product.ID, Platform: "windows", Identifier: "com.example.router", DisplayName: "Router", Published: true}
	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("create variant: %v", err)
	}
	if err := db.Create(&store.Release{ID: "rel_router", VariantID: variant.ID, Version: "1.0.0", Build: 1, CreatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	engine := gin.New()
	if err := RegisterRoutes(engine, db, cfg); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	return engine, db, cfg
}

func TestPublicV1RoutesDoNotRequireToken(t *testing.T) {
	engine, _, _ := newTestRouter(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/router-demo/announcements", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected announcements 200 without token, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products/router-demo/latest?variant=var_router", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected latest 200 without token, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(`{"product_id":"prd_router","variant_id":"var_router","release_id":"rel_router","category":"bug","content":"Cannot install"}`))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected feedback 201 without token, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestFeedbackRateLimitIgnoresUntrustedForwardedIP(t *testing.T) {
	engine, _, _ := newTestRouter(t)
	body := `{"product_id":"prd_router","variant_id":"var_router","release_id":"rel_router","category":"bug","content":"Cannot install"}`

	for index := 0; index < 5; index++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", index+1))
		req.RemoteAddr = "192.0.2.44:1234"
		engine.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected request %d to succeed, got %d body=%s", index+1, rr.Code, rr.Body.String())
		}
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "198.51.100.99")
	req.RemoteAddr = "192.0.2.44:1234"
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected spoofed forwarded IP to remain rate limited, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestFeedbackAdminRoutesRequireAdminToken(t *testing.T) {
	engine, _, _ := newTestRouter(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/feedback", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected feedback 401 without token, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/api/feedback", nil)
	req.Header.Set("X-Auth-Token", "upload-token")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected feedback 403 for upload token, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/api/feedback", nil)
	req.Header.Set("X-Auth-Token", "admin-token")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected feedback 200 for admin token, got %d body=%s", rr.Code, rr.Body.String())
	}

	for _, test := range []struct {
		name  string
		token string
		code  int
	}{
		{name: "missing token", code: http.StatusUnauthorized},
		{name: "upload token", token: "upload-token", code: http.StatusForbidden},
	} {
		t.Run("email test "+test.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/email/test", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			if test.token != "" {
				req.Header.Set("X-Auth-Token", test.token)
			}
			engine.ServeHTTP(rr, req)
			if rr.Code != test.code {
				t.Fatalf("expected email test status %d, got %d body=%s", test.code, rr.Code, rr.Body.String())
			}
		})
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/admin/api/settings/email/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", "admin-token")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected admin email test request to reach validation, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAnnouncementAdminRoutesRequireAdminToken(t *testing.T) {
	engine, _, _ := newTestRouter(t)

	body := strings.NewReader(`{"title":"Notice","content":"Notice content","enabled":true}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/products/prd_router/announcements", body)
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	body = strings.NewReader(`{"title":"Notice","content":"Notice content","enabled":true}`)
	req = httptest.NewRequest(http.MethodPost, "/admin/api/products/prd_router/announcements", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", "upload-token")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for upload token, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	body = strings.NewReader(`{"title":"Notice","content":"Notice content","enabled":true}`)
	req = httptest.NewRequest(http.MethodPost, "/admin/api/products/prd_router/announcements", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", "admin-token")
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 with admin token, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCustomAdminPathRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := store.AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	cfg := config.Default()
	cfg.Server.AdminPath = "console"
	cfg.Auth.AdminTokens = []string{"admin-token"}

	engine := gin.New()
	tmpl := template.Must(template.ParseFS(web.Templates(), "*.html"))
	engine.SetHTMLTemplate(tmpl)
	if err := RegisterRoutes(engine, db, cfg); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	// Custom admin API path is protected
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/console/api/products", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on custom admin api without token, got %d body=%s", rr.Code, rr.Body.String())
	}

	// Default admin path is no longer registered
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/api/products", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on default admin api when custom path is set, got %d", rr.Code)
	}

	// Custom admin HTML page renders and injects the admin path
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/console", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on custom admin page, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `__FENFA_ADMIN_PATH__ = "console"`) {
		t.Fatalf("expected admin path injection in admin page, got body=%s", rr.Body.String())
	}

	// Unmatched route renders the 404 page
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched route, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "404") {
		t.Fatalf("expected 404 page body, got %s", rr.Body.String())
	}
}

func TestAdminPasswordLoginAndRuntimePathChange(t *testing.T) {
	engine, _, cfg := newTestRouter(t)
	// Verify that the browser session is sufficient and the legacy token is
	// not needed for the new login flow.
	cfg.Auth.AdminTokens = nil

	setupRecorder := httptest.NewRecorder()
	setupRequest := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(`{"username":"admin","password":"correct-horse","password_confirmation":"correct-horse"}`))
	setupRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(setupRecorder, setupRequest)
	if setupRecorder.Code != http.StatusOK {
		t.Fatalf("expected admin setup 200, got %d body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
	cookie := setupRecorder.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("expected admin session cookie after setup")
	}

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", strings.NewReader(`{"username":"admin","password":"correct-horse"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected admin login 200, got %d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}
	var loginPayload struct {
		Data struct {
			UploadToken string `json:"upload_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginPayload); err != nil || loginPayload.Data.UploadToken == "" {
		t.Fatalf("expected upload-scoped session token in login response, body=%s", loginRecorder.Body.String())
	}

	uploadRecorder := httptest.NewRecorder()
	uploadRequest := httptest.NewRequest(http.MethodPost, "/upload", nil)
	uploadRequest.Header.Set("X-Auth-Token", loginPayload.Data.UploadToken)
	engine.ServeHTTP(uploadRecorder, uploadRequest)
	if uploadRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected upload session to reach upload validation, got %d body=%s", uploadRecorder.Code, uploadRecorder.Body.String())
	}

	adminWithUploadTokenRecorder := httptest.NewRecorder()
	adminWithUploadTokenRequest := httptest.NewRequest(http.MethodGet, "/admin/api/products", nil)
	adminWithUploadTokenRequest.Header.Set("X-Auth-Token", loginPayload.Data.UploadToken)
	engine.ServeHTTP(adminWithUploadTokenRecorder, adminWithUploadTokenRequest)
	if adminWithUploadTokenRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected upload session to be rejected by admin API, got %d body=%s", adminWithUploadTokenRecorder.Code, adminWithUploadTokenRecorder.Body.String())
	}

	productsRecorder := httptest.NewRecorder()
	productsRequest := httptest.NewRequest(http.MethodGet, "/admin/api/products", nil)
	productsRequest.Header.Set("Cookie", cookie)
	engine.ServeHTTP(productsRecorder, productsRequest)
	if productsRecorder.Code != http.StatusOK {
		t.Fatalf("expected session-authenticated admin API 200, got %d body=%s", productsRecorder.Code, productsRecorder.Body.String())
	}

	settingsRecorder := httptest.NewRecorder()
	settingsBody := strings.NewReader(`{"primary_domain":"https://download.example.com","secondary_domains":[],"organization":"Fenfa","admin_path":"console"}`)
	settingsRequest := httptest.NewRequest(http.MethodPut, "/admin/api/settings", settingsBody)
	settingsRequest.Header.Set("Content-Type", "application/json")
	settingsRequest.Header.Set("Cookie", cookie)
	engine.ServeHTTP(settingsRecorder, settingsRequest)
	if settingsRecorder.Code != http.StatusOK {
		t.Fatalf("expected admin path update 200, got %d body=%s", settingsRecorder.Code, settingsRecorder.Body.String())
	}

	newPathRecorder := httptest.NewRecorder()
	newPathRequest := httptest.NewRequest(http.MethodGet, "/console/api/products", nil)
	newPathRequest.Header.Set("Cookie", cookie)
	engine.ServeHTTP(newPathRecorder, newPathRequest)
	if newPathRecorder.Code != http.StatusOK {
		t.Fatalf("expected new admin path 200, got %d body=%s", newPathRecorder.Code, newPathRecorder.Body.String())
	}

	oldPathRecorder := httptest.NewRecorder()
	oldPathRequest := httptest.NewRequest(http.MethodGet, "/admin/api/products", nil)
	oldPathRequest.Header.Set("Cookie", cookie)
	engine.ServeHTTP(oldPathRecorder, oldPathRequest)
	if oldPathRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected old admin path 404, got %d body=%s", oldPathRecorder.Code, oldPathRecorder.Body.String())
	}
}
