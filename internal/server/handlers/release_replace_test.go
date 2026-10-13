package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
)

func TestAdminUpdateReleaseEditsMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedAdminProduct(t, db)
	release := store.Release{
		ID:        "rel_meta_edit",
		VariantID: variant.ID,
		Version:   "1.0.0",
		Build:     1,
		Channel:   "internal",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	payload := performAdminJSONRequest(t, AdminUpdateRelease(db), http.MethodPut, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]any{
		"version":      "2.0.0",
		"build":        20,
		"channel":      "beta",
		"min_os":       "iOS 15.0",
		"changelog":    "fixed",
		"force_update": true,
	})
	updated := payload["data"].(map[string]any)["release"].(map[string]any)
	if updated["version"] != "2.0.0" || updated["build"] != float64(20) {
		t.Fatalf("unexpected version/build in response: %v", updated)
	}
	if updated["channel"] != "beta" || updated["min_os"] != "iOS 15.0" || updated["changelog"] != "fixed" {
		t.Fatalf("unexpected metadata in response: %v", updated)
	}

	var loaded store.Release
	if err := db.Where("id = ?", release.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if loaded.Version != "2.0.0" || loaded.Build != 20 || loaded.ForceUpdate != true {
		t.Fatalf("release was not updated in db: %+v", loaded)
	}
}

func TestAdminUpdateReleaseRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedAdminProduct(t, db)
	release := store.Release{
		ID:        "rel_meta_invalid",
		VariantID: variant.ID,
		Version:   "1.0.0",
		Build:     1,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	cases := []map[string]any{
		{},
		{"version": "  "},
		{"build": -1},
	}
	for _, body := range cases {
		rr := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rr)
		raw, _ := json.Marshal(body)
		c.Request = httptest.NewRequest(http.MethodPut, "/admin/api/releases/"+release.ID, bytes.NewReader(raw))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "releaseID", Value: release.ID}}
		AdminUpdateRelease(db)(c)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %v, got %d body=%s", body, rr.Code, rr.Body.String())
		}
	}
}

func TestReplaceReleaseFileLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedWindowsVariant(t, db)
	oldDir := t.TempDir()
	oldPath := filepath.Join(oldDir, "old.exe")
	if err := os.WriteFile(oldPath, []byte("old-payload"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	release := store.Release{
		ID:            "rel_replace_local",
		DeltaManifest: "previous manifest",
		VariantID:     variant.ID,
		Version:       "1.0.0",
		Build:         1,
		StoragePath:   oldPath,
		FileName:      "old.exe",
		FileExt:       ".exe",
		CreatedAt:     time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	rr, c := replaceMultipartRequest(t, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]string{
		"version":      "1.1.0",
		"build":        "2",
		"channel":      "beta",
		"min_os":       "Windows 10",
		"changelog":    "replaced",
		"force_update": "true",
	}, "app_file", "new.exe", []byte("new-payload"))
	AdminReplaceReleaseFile(db, config.Default())(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	relData := payload["data"].(map[string]any)["release"].(map[string]any)
	if relData["version"] != "1.1.0" || relData["build"] != float64(2) {
		t.Fatalf("unexpected release metadata: %v", relData)
	}

	var loaded store.Release
	if err := db.Where("id = ?", release.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if loaded.StoragePath == oldPath {
		t.Fatalf("storage path was not replaced")
	}
	if loaded.DeltaManifest != "" {
		t.Fatal("replacement retained stale delta manifest")
	}
	if _, err := os.Stat(loaded.StoragePath); err != nil {
		t.Fatalf("new file missing: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old file was not removed: %v", err)
	}
	if loaded.SizeBytes != int64(len("new-payload")) {
		t.Fatalf("unexpected size %d", loaded.SizeBytes)
	}
}

func TestReplaceReleaseFileS3(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedWindowsVariant(t, db)

	if err := db.Model(&store.SystemSettings{}).Where("id = ?", "default").Updates(map[string]any{
		"primary_domain": "https://dist.example.com",
		"storage_type":   "s3",
		"s3_endpoint":    "https://s3.example.com",
		"s3_bucket":      "bucket",
		"s3_access_key":  "key",
		"s3_secret_key":  "secret",
		"s3_public_url":  "https://files.example.com",
	}).Error; err != nil {
		t.Fatalf("update settings: %v", err)
	}

	release := store.Release{
		ID:          "rel_replace_s3",
		VariantID:   variant.ID,
		Version:     "1.0.0",
		Build:       1,
		StoragePath: "s3://uploads/prd/prd/var/rel_replace_s3/old.exe",
		FileName:    "old.exe",
		FileExt:     ".exe",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	fake := &fakeS3Client{}
	orig := newS3StorageClient
	newS3StorageClient = func(store.SystemSettings) s3StorageClient { return fake }
	defer func() { newS3StorageClient = orig }()

	rr, c := replaceMultipartRequest(t, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]string{
		"version": "2.0.0",
		"build":   "3",
	}, "app_file", "new.exe", []byte("s3-payload"))
	AdminReplaceReleaseFile(db, config.Default())(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(fake.uploads) != 1 {
		t.Fatalf("expected 1 s3 upload, got %d", len(fake.uploads))
	}
	if len(fake.deletes) != 1 {
		t.Fatalf("expected 1 s3 delete for old object, got %d", len(fake.deletes))
	}
	var loaded store.Release
	if err := db.Where("id = ?", release.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if !isS3StoragePath(loaded.StoragePath) || loaded.StoragePath == release.StoragePath {
		t.Fatalf("unexpected s3 storage path: %q", loaded.StoragePath)
	}
	if loaded.Version != "2.0.0" || loaded.Build != 3 {
		t.Fatalf("release metadata not updated: %+v", loaded)
	}
}

func TestReplaceReleaseFileRejectsIdentifierMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedAdminProduct(t, db) // iOS variant, identifier com.example.admin
	oldDir := t.TempDir()
	oldPath := filepath.Join(oldDir, "old.ipa")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	release := store.Release{
		ID:          "rel_replace_mismatch",
		VariantID:   variant.ID,
		Version:     "1.0.0",
		Build:       1,
		StoragePath: oldPath,
		FileName:    "old.ipa",
		FileExt:     ".ipa",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	ipa := testIPA(t, "com.other.example", "new-profile-uuid")
	rr, c := replaceMultipartRequest(t, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]string{
		"version": "1.1.0",
		"build":   "2",
	}, "app_file", "new.ipa", ipa)
	AdminReplaceReleaseFile(db, config.Default())(c)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for identifier mismatch, got %d body=%s", rr.Code, rr.Body.String())
	}
	var loaded store.Release
	if err := db.Where("id = ?", release.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load release: %v", err)
	}
	if loaded.Version != "1.0.0" {
		t.Fatalf("release should remain unchanged: %+v", loaded)
	}
}

func TestReplaceReleaseFileRefreshesIosProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedAdminProduct(t, db)
	oldDir := t.TempDir()
	oldPath := filepath.Join(oldDir, "old.ipa")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	release := store.Release{
		ID:          "rel_replace_profile",
		VariantID:   variant.ID,
		Version:     "1.0.0",
		Build:       1,
		StoragePath: oldPath,
		FileName:    "old.ipa",
		FileExt:     ".ipa",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}
	oldProfile := store.ProvisioningProfile{
		ID:        "prof_old",
		ReleaseID: release.ID,
		UUID:      "old-profile-uuid",
		BundleID:  "com.example.admin",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&oldProfile).Error; err != nil {
		t.Fatalf("create old profile: %v", err)
	}

	ipa := testIPA(t, "com.example.admin", "new-profile-uuid")
	rr, c := replaceMultipartRequest(t, "/admin/api/releases/"+release.ID, gin.Params{
		{Key: "releaseID", Value: release.ID},
	}, map[string]string{
		"version": "1.1.0",
		"build":   "2",
	}, "app_file", "new.ipa", ipa)
	AdminReplaceReleaseFile(db, config.Default())(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var profiles []store.ProvisioningProfile
	if err := db.Where("release_id = ?", release.ID).Find(&profiles).Error; err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile after replacement, got %d", len(profiles))
	}
	if profiles[0].UUID != "new-profile-uuid" {
		t.Fatalf("expected refreshed profile uuid, got %s", profiles[0].UUID)
	}
}

func TestAdminDownloadReleaseServesLocalFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedWindowsVariant(t, db)
	filePath := filepath.Join(t.TempDir(), "release.exe")
	if err := os.WriteFile(filePath, []byte("download-me"), 0o644); err != nil {
		t.Fatalf("write release file: %v", err)
	}
	release := store.Release{
		ID:          "rel_admin_download",
		VariantID:   variant.ID,
		Version:     "1.0.0",
		Build:       1,
		StoragePath: filePath,
		FileName:    "release.exe",
		FileExt:     ".exe",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/api/releases/"+release.ID+"/file", nil)
	c.Params = gin.Params{{Key: "releaseID", Value: release.ID}}

	AdminDownloadRelease(db)(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "download-me" {
		t.Fatalf("unexpected file body: %q", rr.Body.String())
	}
}

func TestAdminDownloadReleaseStreamsS3File(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openAdminProductsTestDB(t)
	_, variant := seedWindowsVariant(t, db)
	if err := db.Model(&store.SystemSettings{}).Where("id = ?", "default").Updates(map[string]any{
		"storage_type":  "s3",
		"s3_endpoint":   "https://s3.example.com",
		"s3_bucket":     "bucket",
		"s3_access_key": "key",
		"s3_secret_key": "secret",
	}).Error; err != nil {
		t.Fatalf("update settings: %v", err)
	}
	release := store.Release{
		ID:          "rel_admin_s3_download",
		VariantID:   variant.ID,
		Version:     "1.0.0",
		Build:       1,
		StoragePath: "s3://uploads/release.exe",
		FileName:    "release.exe",
		FileExt:     ".exe",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&release).Error; err != nil {
		t.Fatalf("create release: %v", err)
	}

	fake := &fakeS3Client{getData: []byte("s3-download")}
	orig := newS3StorageClient
	newS3StorageClient = func(store.SystemSettings) s3StorageClient { return fake }
	defer func() { newS3StorageClient = orig }()

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/api/releases/"+release.ID+"/file", nil)
	c.Params = gin.Params{{Key: "releaseID", Value: release.ID}}

	AdminDownloadRelease(db)(c)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "s3-download" {
		t.Fatalf("unexpected s3 file body: %q", rr.Body.String())
	}
}

func seedWindowsVariant(t *testing.T, db *gorm.DB) (store.Product, store.Variant) {
	t.Helper()
	product := store.Product{
		ID:        "prd_windows",
		Slug:      "windows-suite",
		Name:      "Windows Suite",
		Published: true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	variant := store.Variant{
		ID:            "var_windows",
		ProductID:     product.ID,
		Platform:      "windows",
		Identifier:    "com.example.windows",
		DisplayName:   "Windows",
		InstallerType: "exe",
		Published:     true,
	}
	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("create variant: %v", err)
	}
	return product, variant
}

func replaceMultipartRequest(t *testing.T, path string, params gin.Params, fields map[string]string, fileField, filename string, content []byte) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	if fileField != "" {
		fw, err := mw.CreateFormFile(fileField, filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodPost, path, &buf)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = params
	return rr, c
}

func testIPA(t *testing.T, bundleID, profileUUID string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + bundleID + `</string>
<key>CFBundleShortVersionString</key><string>1.0</string>
<key>CFBundleVersion</key><string>1</string>
<key>MinimumOSVersion</key><string>14.0</string>
</dict></plist>`
	f, err := zw.Create("Payload/App.app/Info.plist")
	if err != nil {
		t.Fatalf("create info plist: %v", err)
	}
	if _, err := io.WriteString(f, plist); err != nil {
		t.Fatalf("write info plist: %v", err)
	}

	profile := `garbage<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>
<key>UUID</key><string>` + profileUUID + `</string>
<key>Name</key><string>Test Profile</string>
<key>Entitlements</key><dict><key>application-identifier</key><string>TEAMID.` + bundleID + `</string></dict>
</dict></plist>garbage`
	pf, err := zw.Create("Payload/App.app/embedded.mobileprovision")
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if _, err := io.WriteString(pf, profile); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

type fakeS3Client struct {
	uploads []string
	deletes []string
	gets    []string
	getData []byte
}

func (f *fakeS3Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	f.uploads = append(f.uploads, key)
	return nil
}

func (f *fakeS3Client) Delete(ctx context.Context, key string) error {
	f.deletes = append(f.deletes, key)
	return nil
}

func (f *fakeS3Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	f.gets = append(f.gets, key)
	if f.getData == nil {
		return io.NopCloser(bytes.NewReader([]byte("fake"))), nil
	}
	return io.NopCloser(bytes.NewReader(f.getData)), nil
}
