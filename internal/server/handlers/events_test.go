package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

func openEventsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := store.AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedVisitVariant(t *testing.T, db *gorm.DB) store.Variant {
	t.Helper()

	variant := store.Variant{
		ID:         "var_visit_windows",
		ProductID:  "prd_visit_windows",
		Platform:   "windows",
		Identifier: "com.example.windows",
		Published:  true,
	}
	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("create variant: %v", err)
	}
	return variant
}

func postEvent(db *gorm.DB, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	PostEvent(db)(c)
	return rr
}

func TestPostEventIncrementsVariantVisitCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openEventsTestDB(t)
	variant := seedVisitVariant(t, db)

	rr := postEvent(db, map[string]any{
		"type":       "visit",
		"variant_id": variant.ID,
		"release_id": "rel_legacy_payload",
		"extra":      map[string]any{"path": "/products/example"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}

	var got store.Variant
	if err := db.Where("id = ?", variant.ID).First(&got).Error; err != nil {
		t.Fatalf("load variant: %v", err)
	}
	if got.VisitCount != 1 {
		t.Fatalf("expected visit_count 1, got %d", got.VisitCount)
	}
}

func TestPostEventIgnoresNonVisitAndUnknownVariant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openEventsTestDB(t)
	variant := seedVisitVariant(t, db)

	payloads := []map[string]any{
		{"type": "click", "variant_id": variant.ID},
		{"type": "upload", "variant_id": variant.ID},
		{"type": "unknown", "variant_id": variant.ID},
		{"type": "visit", "variant_id": "var_missing"},
		{"type": "visit", "variant_id": ""},
	}
	for _, payload := range payloads {
		postEvent(db, payload)
	}

	var got store.Variant
	if err := db.Where("id = ?", variant.ID).First(&got).Error; err != nil {
		t.Fatalf("load variant: %v", err)
	}
	if got.VisitCount != 0 {
		t.Fatalf("expected visit_count 0, got %d", got.VisitCount)
	}
}
