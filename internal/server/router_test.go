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
	fenfauth "github.com/openprx/fenfa/internal/server/auth"
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
	cfg.Auth.SetupToken = "setup-token"

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
	setupRequest.Header.Set("X-Setup-Token", cfg.Auth.SetupToken)
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

func TestAdminSetupRejectsMissingToken(t *testing.T) {
	engine, db, _ := newTestRouter(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(`{"username":"admin","password":"correct-horse"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing setup token to return 401, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var count int64
	if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
		t.Fatalf("count admin users: %v", err)
	}
	if count != 0 {
		t.Fatalf("missing setup token must not create an admin, got %d", count)
	}
}

func TestAdminSetupUnavailableWithoutConfiguredToken(t *testing.T) {
	engine, _, cfg := newTestRouter(t)
	cfg.Auth.SetupToken = ""

	statusRecorder := httptest.NewRecorder()
	statusRequest := httptest.NewRequest(http.MethodGet, "/admin/api/auth/status", nil)
	engine.ServeHTTP(statusRecorder, statusRequest)
	if statusRecorder.Code != http.StatusOK || strings.Contains(statusRecorder.Body.String(), `"setup_required":true`) {
		t.Fatalf("setup status must not expose an unconfigured setup form, got %d body=%s", statusRecorder.Code, statusRecorder.Body.String())
	}

	setupRecorder := httptest.NewRecorder()
	setupRequest := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(`{"username":"admin","password":"correct-horse"}`))
	setupRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(setupRecorder, setupRequest)
	if setupRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected setup to be hidden without a configured token, got %d body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
}

func TestAdminSetupReturns409BeforeBcrypt(t *testing.T) {
	engine, db, cfg := newTestRouter(t)
	if err := db.Create(&store.AdminUser{ID: "usr_existing", Username: "existing", PasswordHash: "not-a-bcrypt-hash"}).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(fmt.Sprintf(`{"username":"another","password":%q}`, strings.Repeat("p", 100))))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Setup-Token", cfg.Auth.SetupToken)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected existing admin to return 409 before bcrypt, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminLoginRateLimitAndSuccessResets(t *testing.T) {
	engine, db, _ := newTestRouter(t)
	hash, err := fenfauth.HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	if err := db.Create(&store.AdminUser{ID: "usr_login", Username: "admin", PasswordHash: hash}).Error; err != nil {
		t.Fatalf("seed login admin: %v", err)
	}

	login := func(password string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", strings.NewReader(fmt.Sprintf(`{"username":"admin","password":%q}`, password)))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "192.0.2.55:1234"
		engine.ServeHTTP(recorder, request)
		return recorder
	}
	for i := 0; i < 4; i++ {
		if recorder := login("wrong-password"); recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected failed login %d to return 401, got %d body=%s", i+1, recorder.Code, recorder.Body.String())
		}
	}
	if recorder := login("correct-horse"); recorder.Code != http.StatusOK {
		t.Fatalf("expected valid login after failed attempts to succeed, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := login("wrong-password"); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("successful login should reset account limiter, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	limitedEngine := engine
	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", strings.NewReader(`{"username":"limited","password":"wrong-password"}`))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "192.0.2.56:1234"
		limitedEngine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected failed login %d to return 401, got %d body=%s", i+1, recorder.Code, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", strings.NewReader(`{"username":"limited","password":"wrong-password"}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "192.0.2.56:1234"
	limitedEngine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected sixth failed login to return 429, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminLogoutRevokesUploadSession(t *testing.T) {
	engine, db, cfg := newTestRouter(t)
	setupRecorder := httptest.NewRecorder()
	setupRequest := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(`{"username":"admin","password":"correct-horse"}`))
	setupRequest.Header.Set("Content-Type", "application/json")
	setupRequest.Header.Set("X-Setup-Token", cfg.Auth.SetupToken)
	engine.ServeHTTP(setupRecorder, setupRequest)
	if setupRecorder.Code != http.StatusOK {
		t.Fatalf("setup admin: %d body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
	var payload struct {
		Data struct {
			UploadToken string `json:"upload_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(setupRecorder.Body.Bytes(), &payload); err != nil || payload.Data.UploadToken == "" {
		t.Fatalf("decode setup upload token: %v body=%s", err, setupRecorder.Body.String())
	}
	cookie := setupRecorder.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("setup must return an admin session cookie")
	}

	logoutRecorder := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/admin/api/auth/logout", nil)
	logoutRequest.Header.Set("Cookie", cookie)
	engine.ServeHTTP(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout: %d body=%s", logoutRecorder.Code, logoutRecorder.Body.String())
	}
	if _, err := fenfauth.LookupUploadSession(db, payload.Data.UploadToken); err == nil {
		t.Fatal("expected logout to revoke the upload session")
	}

	uploadRecorder := httptest.NewRecorder()
	uploadRequest := httptest.NewRequest(http.MethodPost, "/upload", nil)
	uploadRequest.Header.Set("X-Auth-Token", payload.Data.UploadToken)
	engine.ServeHTTP(uploadRecorder, uploadRequest)
	if uploadRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected revoked upload token to return 403, got %d body=%s", uploadRecorder.Code, uploadRecorder.Body.String())
	}
}

func TestUploadRejectsUnconfiguredOriginAndCookieOnlyAuth(t *testing.T) {
	engine, db, cfg := newTestRouter(t)
	if err := db.Model(&store.SystemSettings{}).Where("id = ?", "default").Updates(map[string]interface{}{
		"primary_domain": "https://download.example.com",
		"upload_domain":  "https://upload.example.com",
	}).Error; err != nil {
		t.Fatalf("configure upload origins: %v", err)
	}

	for _, method := range []string{http.MethodOptions, http.MethodPost} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/upload", nil)
		request.Header.Set("Origin", "https://evil.example.com")
		request.Header.Set("X-Auth-Token", "upload-token")
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden || recorder.Header().Get("Access-Control-Allow-Origin") != "" || recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Fatalf("expected unconfigured origin to be rejected without CORS headers, got %d headers=%v body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
		}
	}

	allowedRecorder := httptest.NewRecorder()
	allowedRequest := httptest.NewRequest(http.MethodOptions, "/upload", nil)
	allowedRequest.Header.Set("Origin", cfg.Server.PrimaryDomain)
	engine.ServeHTTP(allowedRecorder, allowedRequest)
	if allowedRecorder.Code != http.StatusNoContent || allowedRecorder.Header().Get("Access-Control-Allow-Origin") != cfg.Server.PrimaryDomain {
		t.Fatalf("expected configured origin to receive CORS preflight, got %d headers=%v", allowedRecorder.Code, allowedRecorder.Header())
	}
	if allowedRecorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("upload CORS must not enable browser credentials")
	}

	setupRecorder := httptest.NewRecorder()
	setupRequest := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(`{"username":"admin","password":"correct-horse"}`))
	setupRequest.Header.Set("Content-Type", "application/json")
	setupRequest.Header.Set("X-Setup-Token", cfg.Auth.SetupToken)
	engine.ServeHTTP(setupRecorder, setupRequest)
	if setupRecorder.Code != http.StatusOK {
		t.Fatalf("setup admin: %d body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
	cookie := setupRecorder.Header().Get("Set-Cookie")
	cookieOnlyRecorder := httptest.NewRecorder()
	cookieOnlyRequest := httptest.NewRequest(http.MethodPost, "/upload", nil)
	cookieOnlyRequest.Header.Set("Cookie", cookie)
	engine.ServeHTTP(cookieOnlyRecorder, cookieOnlyRequest)
	if cookieOnlyRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected cookie-only upload to be rejected, got %d body=%s", cookieOnlyRecorder.Code, cookieOnlyRecorder.Body.String())
	}
}

func TestConcurrentAdminSetupCreatesSingleAdmin(t *testing.T) {
	engine, db, cfg := newTestRouter(t)
	statuses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func(index int) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/admin/api/auth/setup", strings.NewReader(`{"username":"admin","password":"correct-horse"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Setup-Token", cfg.Auth.SetupToken)
			request.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", index+60)
			engine.ServeHTTP(recorder, request)
			statuses <- recorder.Code
		}(i)
	}

	first, second := <-statuses, <-statuses
	if !((first == http.StatusOK && second == http.StatusConflict) || (first == http.StatusConflict && second == http.StatusOK)) {
		t.Fatalf("expected one setup success and one conflict, got %d and %d", first, second)
	}
	var count int64
	if err := db.Model(&store.AdminUser{}).Count(&count).Error; err != nil {
		t.Fatalf("count concurrent admins: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one admin after concurrent setup, got %d", count)
	}
}

func TestLinkedReleaseRouteRequiresAdmin(t *testing.T) {
	engine, _, _ := newTestRouter(t)
	for _, token := range []string{"", "upload-token", "admin-token"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/api/variants/var_router/releases/link", strings.NewReader(`{"version":"2.0","external_url":"https://cdn.example.com/app.exe"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Token", token)
		engine.ServeHTTP(rr, req)
		if token == "admin-token" {
			if rr.Code != 201 {
				t.Fatalf("admin create: %d %s", rr.Code, rr.Body)
			}
		} else if rr.Code != 401 && rr.Code != 403 {
			t.Fatalf("non-admin create: %q %d", token, rr.Code)
		}
	}
}
