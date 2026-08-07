package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/openprx/fenfa/internal/store"
)

func TestReleaseForceUpdateDefaultsToFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)

	release := store.Release{
		ID:        "rel_default",
		VariantID: "var_default",
		Version:   "1.0.0",
		Build:     1,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	var loaded store.Release
	if err := db.Where("id = ?", release.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if loaded.ForceUpdate {
		t.Fatalf("expected force_update=false by default, got true")
	}
}

func TestReleaseForceUpdateCreateAndRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)

	release := store.Release{
		ID:          "rel_force",
		VariantID:   "var_force",
		Version:     "2.0.0",
		Build:       2,
		ForceUpdate: true,
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	var loaded store.Release
	if err := db.Where("id = ?", release.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if !loaded.ForceUpdate {
		t.Fatalf("expected force_update=true, got false")
	}
}

func TestAdminUpdateReleaseTogglesForceUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedAdminProduct(t, db)
	release := store.Release{
		ID:        "rel_update_force",
		VariantID: variant.ID,
		Version:   "1.0.0",
		Build:     1,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	payload := performAdminJSONRequest(t, AdminUpdateRelease(db), http.MethodPut, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]any{"force_update": true})
	updated := payload["data"].(map[string]any)["release"].(map[string]any)
	if updated["force_update"] != true {
		t.Fatalf("expected force_update=true after update, got %v", updated["force_update"])
	}

	payload = performAdminJSONRequest(t, AdminUpdateRelease(db), http.MethodPut, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]any{"force_update": false})
	updated = payload["data"].(map[string]any)["release"].(map[string]any)
	if updated["force_update"] != false {
		t.Fatalf("expected force_update=false after update, got %v", updated["force_update"])
	}
}
