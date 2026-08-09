package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/openprx/fenfa/internal/store"
)

func performFeedbackRequest(t *testing.T, handler gin.HandlerFunc, body map[string]any, ip string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal feedback request: %v", err)
	}

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":1234"
	c.Request = req
	handler(c)

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode feedback response: %v body=%s", err, rr.Body.String())
	}
	return rr, payload
}

func TestPublicCreateFeedbackPersistsContextSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, variant := seedAdminProduct(t, db)

	rr, payload := performFeedbackRequest(t, PublicCreateFeedback(db), map[string]any{
		"product_id": product.ID,
		"variant_id": variant.ID,
		"release_id": "rel_admin_ios",
		"category":   "bug",
		"content":    "无法启动",
		"contact":    "example@example.com",
	}, "192.0.2.10")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok response, got %v", payload)
	}

	var feedback store.Feedback
	if err := db.First(&feedback).Error; err != nil {
		t.Fatalf("load feedback: %v", err)
	}
	if feedback.ProductID != product.ID || feedback.VariantID != variant.ID || feedback.ReleaseID != "rel_admin_ios" {
		t.Fatalf("unexpected feedback context: %+v", feedback)
	}
	if feedback.ProductName != product.Name || feedback.VariantPlatform != variant.Platform || feedback.ReleaseVersion != "1.0.0" {
		t.Fatalf("unexpected feedback snapshot: %+v", feedback)
	}
	if feedback.Status != feedbackStatusPending || feedback.Category != "bug" || feedback.Contact != "example@example.com" {
		t.Fatalf("unexpected feedback fields: %+v", feedback)
	}
}

func TestPublicCreateFeedbackValidatesPayloadAndContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, variant := seedAdminProduct(t, db)

	cases := []struct {
		name string
		body map[string]any
		code int
	}{
		{
			name: "missing content",
			body: map[string]any{"product_id": product.ID, "category": "bug"},
			code: http.StatusBadRequest,
		},
		{
			name: "invalid category",
			body: map[string]any{"product_id": product.ID, "category": "question", "content": "hello"},
			code: http.StatusBadRequest,
		},
		{
			name: "content too long",
			body: map[string]any{"product_id": product.ID, "category": "other", "content": strings.Repeat("a", feedbackMaxContentRunes+1)},
			code: http.StatusBadRequest,
		},
		{
			name: "contact too long",
			body: map[string]any{"product_id": product.ID, "category": "other", "content": "hello", "contact": strings.Repeat("a", feedbackMaxContactRunes+1)},
			code: http.StatusBadRequest,
		},
		{
			name: "missing product",
			body: map[string]any{"product_id": "prd_missing", "category": "other", "content": "hello"},
			code: http.StatusNotFound,
		},
		{
			name: "mismatched release",
			body: map[string]any{"product_id": product.ID, "variant_id": variant.ID, "release_id": "rel_missing", "category": "other", "content": "hello"},
			code: http.StatusNotFound,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rr, _ := performFeedbackRequest(t, PublicCreateFeedback(db), testCase.body, "192.0.2.20")
			if rr.Code != testCase.code {
				t.Fatalf("expected status %d, got %d body=%s", testCase.code, rr.Code, rr.Body.String())
			}
		})
	}

	var count int64
	db.Model(&store.Feedback{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected invalid feedback not to be persisted, got %d", count)
	}
}

func TestPublicCreateFeedbackHoneypotAndRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, variant := seedAdminProduct(t, db)
	handler := PublicCreateFeedback(db)

	honeypot, _ := performFeedbackRequest(t, handler, map[string]any{
		"product_id": product.ID,
		"variant_id": variant.ID,
		"category":   "other",
		"content":    "bot",
		"website":    "https://spam.example",
	}, "192.0.2.30")
	if honeypot.Code != http.StatusCreated {
		t.Fatalf("expected honeypot status 201, got %d", honeypot.Code)
	}

	for i := 0; i < feedbackRateLimit; i++ {
		rr, _ := performFeedbackRequest(t, handler, map[string]any{
			"product_id": product.ID,
			"variant_id": variant.ID,
			"category":   "suggestion",
			"content":    "正常反馈",
		}, "192.0.2.31")
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected request %d to succeed, got %d", i+1, rr.Code)
		}
	}
	limited, _ := performFeedbackRequest(t, handler, map[string]any{
		"product_id": product.ID,
		"variant_id": variant.ID,
		"category":   "suggestion",
		"content":    "第六次反馈",
	}, "192.0.2.31")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limit 429, got %d body=%s", limited.Code, limited.Body.String())
	}
	if limited.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header")
	}

	var count int64
	db.Model(&store.Feedback{}).Count(&count)
	if count != feedbackRateLimit {
		t.Fatalf("expected only valid non-honeypot submissions persisted, got %d", count)
	}
}

func TestAdminFeedbackListUpdateAndDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	product, variant := seedAdminProduct(t, db)
	now := time.Now().UTC()
	feedbacks := []store.Feedback{
		{ID: "fb_old", ProductID: product.ID, VariantID: variant.ID, ProductName: product.Name, VariantPlatform: variant.Platform, Category: "bug", Content: "old", Status: feedbackStatusPending, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
		{ID: "fb_new", ProductID: product.ID, VariantID: variant.ID, ProductName: product.Name, VariantPlatform: variant.Platform, Category: "suggestion", Content: "new", Status: feedbackStatusPending, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&feedbacks).Error; err != nil {
		t.Fatalf("create feedback fixtures: %v", err)
	}

	listRR := httptest.NewRecorder()
	listContext, _ := gin.CreateTestContext(listRR)
	listContext.Request = httptest.NewRequest(http.MethodGet, "/admin/api/feedback?page=1&page_size=1&category=suggestion", nil)
	AdminListFeedback(db)(listContext)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", listRR.Code, listRR.Body.String())
	}
	var listPayload map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	data := listPayload["data"].(map[string]any)
	if data["total"].(float64) != 1 || len(data["items"].([]any)) != 1 {
		t.Fatalf("unexpected filtered list: %v", data)
	}
	item := data["items"].([]any)[0].(map[string]any)
	if item["id"] != "fb_new" {
		t.Fatalf("expected newest filtered feedback, got %v", item["id"])
	}

	updatePayload := performAdminJSONRequest(t, AdminUpdateFeedback(db), http.MethodPatch, "/admin/api/feedback/fb_new", gin.Params{{Key: "feedbackID", Value: "fb_new"}}, map[string]any{"status": "processed"})
	updated := updatePayload["data"].(map[string]any)["feedback"].(map[string]any)
	if updated["status"] != feedbackStatusProcessed {
		t.Fatalf("expected processed status, got %v", updated["status"])
	}

	invalidUpdate := httptest.NewRecorder()
	invalidContext, _ := gin.CreateTestContext(invalidUpdate)
	invalidContext.Request = httptest.NewRequest(http.MethodPatch, "/admin/api/feedback/fb_new", bytes.NewReader([]byte(`{"status":"ignored"}`)))
	invalidContext.Request.Header.Set("Content-Type", "application/json")
	invalidContext.Params = gin.Params{{Key: "feedbackID", Value: "fb_new"}}
	AdminUpdateFeedback(db)(invalidContext)
	if invalidUpdate.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status 400, got %d body=%s", invalidUpdate.Code, invalidUpdate.Body.String())
	}

	deletePayload := performAdminJSONRequest(t, AdminDeleteFeedback(db), http.MethodDelete, "/admin/api/feedback/fb_old", gin.Params{{Key: "feedbackID", Value: "fb_old"}}, nil)
	if deletePayload["ok"] != true {
		t.Fatalf("expected delete success, got %v", deletePayload)
	}
	var remaining int64
	db.Model(&store.Feedback{}).Count(&remaining)
	if remaining != 1 {
		t.Fatalf("expected one feedback remaining, got %d", remaining)
	}
}
