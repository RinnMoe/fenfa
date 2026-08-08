package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
	"gorm.io/gorm"
)

type fakeS3StorageClient struct {
	uploadKeys []string
	deleteKeys []string
	uploadErr  error
	deleteErr  error
}

func (f *fakeS3StorageClient) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	if f.uploadErr != nil {
		return f.uploadErr
	}
	if _, err := io.Copy(io.Discard, body); err != nil {
		return err
	}
	f.uploadKeys = append(f.uploadKeys, key)
	return nil
}

func (f *fakeS3StorageClient) Delete(ctx context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleteKeys = append(f.deleteKeys, key)
	return nil
}

func useFakeS3StorageClient(t *testing.T, fake *fakeS3StorageClient) {
	t.Helper()
	old := newS3StorageClient
	newS3StorageClient = func(store.SystemSettings) s3StorageClient {
		return fake
	}
	t.Cleanup(func() {
		newS3StorageClient = old
	})
}

func setS3StorageSettings(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Model(&store.SystemSettings{}).
		Where("id = ?", "default").
		Updates(map[string]interface{}{
			"storage_type":  "s3",
			"s3_endpoint":   "https://example.com",
			"s3_bucket":     "fenfa-test",
			"s3_access_key": "access",
			"s3_secret_key": "secret",
			"s3_public_url": "https://cdn.example.com",
		}).Error
	if err != nil {
		t.Fatalf("seed s3 settings: %v", err)
	}
}

func TestUploadS3RemovesLocalStaging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUploadWorkdir(t)

	db := openHandlerTestDB(t)
	setS3StorageSettings(t, db)
	cfg := config.Default()
	variant := seedVariant(t, db, "windows", "com.example.s3")
	fake := &fakeS3StorageClient{}
	useFakeS3StorageClient(t, fake)

	req := makeMultipartRequest(t, "/upload", map[string]string{
		"variant_id": variant.ID,
		"version":    "1.0.0",
		"build":      "1",
	}, "demo.msi", []byte("MSI-DEMO"))

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	Upload(db, cfg)(c)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}

	payload := decodeJSON(t, rr)
	releaseData := payload["data"].(map[string]any)["release"].(map[string]any)
	releaseID := releaseData["id"].(string)

	var release store.Release
	if err := db.Where("id = ?", releaseID).Take(&release).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if !strings.HasPrefix(release.StoragePath, "s3://") {
		t.Fatalf("expected s3 storage path, got %q", release.StoragePath)
	}
	if len(fake.uploadKeys) != 1 {
		t.Fatalf("expected 1 s3 upload, got %d", len(fake.uploadKeys))
	}
	if !strings.Contains(fake.uploadKeys[0], releaseID) {
		t.Fatalf("expected uploaded key to contain release id, got %q", fake.uploadKeys[0])
	}

	appDir := filepath.Join("uploads", variant.ProductID, variant.ID, releaseID)
	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		t.Fatalf("expected staged release dir to be removed, stat err=%v", err)
	}
}

func TestUploadS3FailureCleansLocalStaging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUploadWorkdir(t)

	db := openHandlerTestDB(t)
	setS3StorageSettings(t, db)
	cfg := config.Default()
	variant := seedVariant(t, db, "windows", "com.example.s3fail")
	fake := &fakeS3StorageClient{uploadErr: errors.New("upload failed")}
	useFakeS3StorageClient(t, fake)

	req := makeMultipartRequest(t, "/upload", map[string]string{
		"variant_id": variant.ID,
		"version":    "1.0.0",
		"build":      "1",
	}, "demo.msi", []byte("MSI-DEMO"))

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	Upload(db, cfg)(c)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body=%s", rr.Code, rr.Body.String())
	}

	var count int64
	if err := db.Model(&store.Release{}).Where("variant_id = ?", variant.ID).Count(&count).Error; err != nil {
		t.Fatalf("count releases: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no release after failed upload, got %d", count)
	}

	appDir := filepath.Join("uploads", variant.ProductID, variant.ID)
	entries, err := os.ReadDir(appDir)
	if err == nil && len(entries) != 0 {
		t.Fatalf("expected no staged release dirs, got %v", entries)
	}
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read upload parent dir: %v", err)
	}
}

func TestAdminDeleteS3ReleaseCallsS3Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := openAdminProductsTestDB(t)
	setS3StorageSettings(t, db)
	_, variant := seedAdminProduct(t, db)
	fake := &fakeS3StorageClient{}
	useFakeS3StorageClient(t, fake)

	releaseID := "rel_admin_s3_delete"
	storagePath := "s3://uploads/prd_admin/var_admin_ios/rel_admin_s3/app.ipa"
	if err := db.Create(&store.Release{
		ID:          releaseID,
		VariantID:   variant.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
	}).Error; err != nil {
		t.Fatalf("create s3 release: %v", err)
	}

	payload := performAdminJSONRequest(t, AdminDeleteRelease(db), http.MethodDelete, "/admin/api/releases/"+releaseID, gin.Params{
		{Key: "releaseID", Value: releaseID},
	}, nil)
	if !payload["ok"].(bool) {
		t.Fatalf("expected delete ok, got %v", payload)
	}
	if len(fake.deleteKeys) != 1 {
		t.Fatalf("expected 1 s3 delete, got %d", len(fake.deleteKeys))
	}
	expectedKey := "uploads/prd_admin/var_admin_ios/rel_admin_s3/app.ipa"
	if fake.deleteKeys[0] != expectedKey {
		t.Fatalf("expected s3 delete key %q, got %q", expectedKey, fake.deleteKeys[0])
	}
}
