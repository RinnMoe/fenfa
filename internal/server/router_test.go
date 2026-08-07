package server

import (
	"fmt"
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
	RegisterRoutes(engine, db, cfg)
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
