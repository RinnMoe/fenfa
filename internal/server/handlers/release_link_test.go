package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalURLValidation(t *testing.T) {
	for _, raw := range []string{"", "http://example.com/a.apk", "//example.com/a.apk", "javascript:alert(1)", "https://user:pass@example.com/a.apk", "https://example.com/a.apk#fragment", "https://example.com/a\r\nb", strings.Repeat("x", 4097)} {
		if _, err := validateExternalURL(raw); err == nil {
			t.Errorf("accepted invalid URL %q", raw)
		}
	}
	if got, err := validateExternalURL(" https://cdn.example.com/a.apk?signature=abc "); err != nil || got != "https://cdn.example.com/a.apk?signature=abc" {
		t.Fatalf("valid signed URL: %q %v", got, err)
	}
}

func TestLinkedReleaseLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedAdminProduct(t, db)
	call := func(h gin.HandlerFunc, method, body string, params gin.Params) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rr)
		c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = params
		h(c)
		return rr
	}
	params := gin.Params{{Key: "variantID", Value: variant.ID}}
	create := AdminCreateLinkedRelease(db, config.Default())
	for _, body := range []string{`{"version":"2","external_url":"http://example.com/a.ipa"}`, `{"version":"2","external_url":"https://example.com/a.ipa","size_bytes":-1}`, `{"version":"2","external_url":"https://example.com/a.ipa","sha256":"bad"}`, `{"version":"2","external_url":"https://example.com/a.ipa","file_name":"a.apk"}`} {
		if rr := call(create, "POST", body, params); rr.Code != 400 {
			t.Fatalf("invalid accepted: %d %s", rr.Code, rr.Body)
		}
	}
	rr := call(create, "POST", `{"version":"2.0","build":2,"external_url":"https://cdn.example.com/a.ipa?signature=abc","size_bytes":123}`, params)
	if rr.Code != 201 {
		t.Fatalf("create: %d %s", rr.Code, rr.Body)
	}
	var rel store.Release
	if err := db.Where("version = ?", "2.0").First(&rel).Error; err != nil {
		t.Fatal(err)
	}
	if rel.StoragePath != "" || rel.FileName != "a.ipa" || rel.SizeBytes != 123 {
		t.Fatalf("unexpected metadata: %+v", rel)
	}
	rp := gin.Params{{Key: "releaseID", Value: rel.ID}}
	rr = call(Download(db), "GET", "", rp)
	if rr.Code != 302 || rr.Header().Get("Location") != rel.ExternalURL || rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("redirect: %d %v", rr.Code, rr.Header())
	}
	db.First(&rel, "id = ?", rel.ID)
	if rel.DownloadCount != 1 {
		t.Fatalf("counter: %d", rel.DownloadCount)
	}
	db.Model(&variant).Update("published", false)
	if rr = call(Download(db), "GET", "", rp); rr.Code != 404 {
		t.Fatal("unpublished release downloadable")
	}
	if rr = call(AdminDownloadRelease(db), "GET", "", rp); rr.Code != 302 {
		t.Fatal("admin redirect failed")
	}
	retainedPath := filepath.Join(t.TempDir(), "retained.ipa")
	db.Model(&rel).Updates(map[string]any{"sha256": strings.Repeat("a", 64), "delta_manifest": "old", "storage_path": retainedPath})
	rr = call(AdminUpdateRelease(db), "PUT", `{"external_url":"https://cdn.example.com/b.ipa"}`, rp)
	if rr.Code != 200 {
		t.Fatalf("edit: %d %s", rr.Code, rr.Body)
	}
	db.First(&rel, "id = ?", rel.ID)
	if rel.SHA256 != "" || rel.DeltaManifest != "" || rel.SizeBytes != 0 || rel.StoragePath != retainedPath || rel.DownloadCount != 1 {
		t.Fatalf("invalid edit state: %+v", rel)
	}
	if rr = call(AdminUpdateRelease(db), "PUT", `{"external_url":""}`, rp); rr.Code != 400 {
		t.Fatal("empty URL accepted")
	}
}
