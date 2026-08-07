package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
)

func performPublicLatest(t *testing.T, handler gin.HandlerFunc, path string, params gin.Params) (int, map[string]any) {
	t.Helper()

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "download.example.com"
	c.Request = req
	c.Params = params

	handler(c)

	var payload map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &payload)
	return rr.Code, payload
}

func TestPublicLatestReleaseReturnsLatestVariantRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openProductsAPITestDB(t)
	cfg := config.Default()
	cfg.Server.PrimaryDomain = "https://download.example.com"
	product, _, windowsVariant := seedProductFixture(t, db)

	older := store.Release{
		ID:          "rel_older",
		VariantID:   windowsVariant.ID,
		Version:     "1.0.0",
		Build:       10,
		Changelog:   "old",
		ForceUpdate: false,
		CreatedAt:   time.Now().Add(-time.Hour),
	}
	newer := store.Release{
		ID:          "rel_newer",
		VariantID:   windowsVariant.ID,
		Version:     "2.0.0",
		Build:       20,
		Changelog:   "new",
		ForceUpdate: true,
		CreatedAt:   time.Now().Add(2 * time.Minute),
	}
	if err := db.Create(&older).Error; err != nil {
		t.Fatalf("create older release: %v", err)
	}
	if err := db.Create(&newer).Error; err != nil {
		t.Fatalf("create newer release: %v", err)
	}

	status, payload := performPublicLatest(t, PublicLatestRelease(db, cfg), "/api/v1/products/"+product.Slug+"/latest?variant="+windowsVariant.ID, gin.Params{
		{Key: "slug", Value: product.Slug},
	})
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%v", status, payload)
	}

	data := payload["data"].(map[string]any)
	variant := data["variant"].(map[string]any)
	if variant["id"] != windowsVariant.ID {
		t.Fatalf("expected variant id %q, got %v", windowsVariant.ID, variant["id"])
	}
	release := data["release"].(map[string]any)
	if release["id"] != newer.ID {
		t.Fatalf("expected latest release %q, got %v", newer.ID, release["id"])
	}
	if release["version"] != "2.0.0" {
		t.Fatalf("expected latest version, got %v", release["version"])
	}
	if release["force_update"] != true {
		t.Fatalf("expected force_update=true, got %v", release["force_update"])
	}
	if release["download_url"] != "https://download.example.com/d/"+newer.ID {
		t.Fatalf("expected download URL, got %v", release["download_url"])
	}
}

func TestPublicLatestReleaseMissingProductOrVariantOrRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openProductsAPITestDB(t)
	cfg := config.Default()
	cfg.Server.PrimaryDomain = "https://download.example.com"
	product, _, _ := seedProductFixture(t, db)

	status, _ := performPublicLatest(t, PublicLatestRelease(db, cfg), "/api/v1/products/missing/latest?variant=var_windows", gin.Params{
		{Key: "slug", Value: "missing"},
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected missing product 404, got %d", status)
	}

	status, _ = performPublicLatest(t, PublicLatestRelease(db, cfg), "/api/v1/products/"+product.Slug+"/latest?variant=missing", gin.Params{
		{Key: "slug", Value: product.Slug},
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected missing variant 404, got %d", status)
	}

	emptyVariant := store.Variant{
		ID:            "var_empty",
		ProductID:     product.ID,
		Platform:      "linux",
		Identifier:    "com.example.empty",
		DisplayName:   "Empty",
		InstallerType: "appimage",
		Published:     true,
	}
	if err := db.Create(&emptyVariant).Error; err != nil {
		t.Fatalf("create empty variant: %v", err)
	}
	status, _ = performPublicLatest(t, PublicLatestRelease(db, cfg), "/api/v1/products/"+product.Slug+"/latest?variant="+emptyVariant.ID, gin.Params{
		{Key: "slug", Value: product.Slug},
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected missing release 404, got %d", status)
	}

	status, _ = performPublicLatest(t, PublicLatestRelease(db, cfg), "/api/v1/products/"+product.Slug+"/latest", gin.Params{
		{Key: "slug", Value: product.Slug},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected multi-variant 400, got %d", status)
	}

}

func TestPublicLatestReleaseWorksForSingleVariantProduct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	cfg := config.Default()
	cfg.Server.PrimaryDomain = "https://download.example.com"
	product, _ := seedAdminProduct(t, db)

	status, payload := performPublicLatest(t, PublicLatestRelease(db, cfg), "/api/v1/products/"+product.Slug+"/latest", gin.Params{
		{Key: "slug", Value: product.Slug},
	})
	if status != http.StatusOK {
		t.Fatalf("expected single variant 200, got %d body=%v", status, payload)
	}
	data := payload["data"].(map[string]any)
	release := data["release"].(map[string]any)
	if release["id"] != "rel_admin_ios" {
		t.Fatalf("expected seeded release id, got %v", release["id"])
	}
}
