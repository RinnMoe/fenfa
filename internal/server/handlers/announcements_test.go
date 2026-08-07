package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/openprx/fenfa/internal/store"
)

func TestAnnouncementAdminCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, _ := seedAdminProduct(t, db)

	now := time.Now()
	createPayload := performAdminJSONRequest(t, AdminCreateAnnouncement(db), http.MethodPost, "/admin/api/products/"+product.ID+"/announcements", gin.Params{
		{Key: "productID", Value: product.ID},
	}, map[string]any{
		"title":    "Maintenance",
		"content":  "Service will be upgraded.",
		"enabled":  true,
		"start_at": now.Add(-time.Hour).Format(time.RFC3339),
		"end_at":   now.Add(time.Hour).Format(time.RFC3339),
	})
	created := createPayload["data"].(map[string]any)["announcement"].(map[string]any)
	if created["title"] != "Maintenance" {
		t.Fatalf("expected created announcement title, got %v", created["title"])
	}
	if created["enabled"] != true {
		t.Fatalf("expected created announcement enabled=true, got %v", created["enabled"])
	}

	announcementID := created["id"].(string)
	updatePayload := performAdminJSONRequest(t, AdminUpdateAnnouncement(db), http.MethodPut, "/admin/api/announcements/"+announcementID, gin.Params{
		{Key: "announcementID", Value: announcementID},
	}, map[string]any{
		"title":    "Maintenance Updated",
		"content":  "Service will be upgraded again.",
		"enabled":  false,
		"start_at": nil,
		"end_at":   nil,
	})
	updated := updatePayload["data"].(map[string]any)["announcement"].(map[string]any)
	if updated["title"] != "Maintenance Updated" {
		t.Fatalf("expected updated announcement title, got %v", updated["title"])
	}
	if updated["enabled"] != false {
		t.Fatalf("expected updated announcement enabled=false, got %v", updated["enabled"])
	}

	deletePayload := performAdminJSONRequest(t, AdminDeleteAnnouncement(db), http.MethodDelete, "/admin/api/announcements/"+announcementID, gin.Params{
		{Key: "announcementID", Value: announcementID},
	}, nil)
	if deletePayload["ok"] != true {
		t.Fatalf("expected delete ok response, got %v", deletePayload)
	}

	var count int64
	db.Model(&store.Announcement{}).Where("id = ?", announcementID).Count(&count)
	if count != 0 {
		t.Fatalf("expected announcement to be deleted, got count=%d", count)
	}
}

func TestPublicAnnouncementsFiltersEnabledAndTimeWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, _ := seedAdminProduct(t, db)
	now := time.Now()

	announcements := []store.Announcement{
		{ID: "ann_active", ProductID: product.ID, Title: "Active", Content: "Active content", Enabled: true, StartAt: nil, EndAt: nil, CreatedAt: now.UTC()},
		{ID: "ann_disabled", ProductID: product.ID, Title: "Disabled", Content: "Disabled content", Enabled: false, CreatedAt: now.UTC()},
		{ID: "ann_future", ProductID: product.ID, Title: "Future", Content: "Future content", Enabled: true, StartAt: timePtr(now.Add(2 * time.Hour).UTC()), CreatedAt: now.UTC()},
		{ID: "ann_expired", ProductID: product.ID, Title: "Expired", Content: "Expired content", Enabled: true, EndAt: timePtr(now.Add(-2 * time.Hour).UTC()), CreatedAt: now.UTC()},
	}
	for _, announcement := range announcements {
		shouldDisable := !announcement.Enabled
		if err := db.Select("*").Create(&announcement).Error; err != nil {
			t.Fatalf("create announcement: %v", err)
		}
		if shouldDisable {
			if err := db.Model(&store.Announcement{}).Where("id = ?", announcement.ID).UpdateColumn("enabled", false).Error; err != nil {
				t.Fatalf("set announcement disabled: %v", err)
			}
		}
	}

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/products/"+product.Slug+"/announcements", nil)
	c.Params = gin.Params{{Key: "slug", Value: product.Slug}}
	PublicAnnouncements(db)(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items := payload["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 active announcement, got %d: %v", len(items), items)
	}
	item := items[0].(map[string]any)
	if item["id"] != "ann_active" {
		t.Fatalf("expected active announcement id, got %v", item["id"])
	}
	if _, ok := item["product_id"]; ok {
		t.Fatalf("public announcement should not expose product_id")
	}
}

func TestPublicAnnouncementsEmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, _ := seedAdminProduct(t, db)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/products/"+product.Slug+"/announcements", nil)
	c.Params = gin.Params{{Key: "slug", Value: product.Slug}}
	PublicAnnouncements(db)(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items := payload["data"].(map[string]any)["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("expected empty items, got %v", items)
	}
}

func TestPublicAnnouncementsMissingProduct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/products/missing/announcements", nil)
	c.Params = gin.Params{{Key: "slug", Value: "missing"}}
	PublicAnnouncements(db)(c)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminAnnouncementValidatesProductAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	body, _ := json.Marshal(map[string]any{"title": "Valid", "content": "Valid content", "enabled": true})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/api/products/missing/announcements", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "productID", Value: "missing"}}
	AdminCreateAnnouncement(db)(c)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected missing product 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
