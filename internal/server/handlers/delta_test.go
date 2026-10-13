package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
)

func TestDeltaUploadAndInvalidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUploadWorkdir(t)
	db := openProductsAPITestDB(t)
	product, _, variant := seedProductFixture(t, db)
	variant.Platform = "android"
	variant.Identifier = "moe.rinn.mychu"
	variant.Arch = "arm64-v8a"
	if err := db.Save(&variant).Error; err != nil {
		t.Fatal(err)
	}
	apk := []byte("exact distributed target apk bytes")
	patch := []byte("patch")
	if err := os.WriteFile("target.apk", apk, 0600); err != nil {
		t.Fatal(err)
	}
	base := store.Release{ID: "base", VariantID: variant.ID, Build: 2012, SHA256: deltaHash([]byte("base")), Channel: "official"}
	target := store.Release{ID: "target", VariantID: variant.ID, Build: 2013, Version: "1.0.2", SHA256: deltaHash(apk), SizeBytes: int64(len(apk)), StoragePath: "target.apk", FileExt: ".apk", Channel: "official"}
	db.Create(&base)
	target.CreatedAt = time.Now().Add(2 * time.Minute)
	db.Create(&target)
	m := deltaManifest{SchemaVersion: 1, PackageName: variant.Identifier, Channel: "official", ABI: variant.Arch, Target: deltaTarget{Version: target.Version, VersionCode: target.Build, Size: target.SizeBytes, SHA256: target.SHA256}, Patches: []deltaPatch{{Algorithm: "hdiffpatch", Compression: "zlib", FromVersionCode: base.Build, FromSHA256: base.SHA256, Size: int64(len(patch)), SHA256: deltaHash(patch), URL: "https://untrusted.example/patch"}}}
	upload := func(manifest deltaManifest, data []byte) int {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		raw, _ := json.Marshal(manifest)
		w.WriteField("manifest", string(raw))
		f, _ := w.CreateFormFile(manifest.Patches[0].SHA256, "patch.hdiff")
		f.Write(data)
		w.Close()
		rr := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rr)
		c.Request = httptest.NewRequest("POST", "/releases/target/delta", &b)
		c.Request.Header.Set("Content-Type", w.FormDataContentType())
		c.Params = gin.Params{{Key: "releaseID", Value: target.ID}}
		UploadDelta(db)(c)
		return rr.Code
	}
	if status := upload(m, []byte("wrong")); status != 400 {
		t.Fatalf("bad patch accepted: %d", status)
	}
	invalid := m
	invalid.Target.SHA256 = base.SHA256
	if status := upload(invalid, patch); status != 400 {
		t.Fatalf("wrong target accepted: %d", status)
	}
	if err := os.WriteFile("target.apk", []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if status := upload(m, patch); status != 400 {
		t.Fatalf("tampered target accepted: %d", status)
	}
	os.WriteFile("target.apk", apk, 0600)
	if status := upload(m, patch); status != 200 {
		t.Fatalf("upload failed: %d", status)
	}
	if status := upload(m, patch); status != 200 {
		t.Fatalf("idempotent retry failed: %d", status)
	}
	db.First(&target, "id = ?", target.ID)
	url := deltaManifestURL(target, "https://dist.example.com")
	if url == "" || deltaManifestURL(target, "http://dist.example.com") != "" {
		t.Fatal("HTTPS URL contract")
	}
	r := gin.New()
	r.GET("/delta/:releaseID/:targetHash/:manifestHash/manifest.json", PublicDeltaManifest(db))
	r.GET("/delta/:releaseID/:targetHash/:manifestHash/:patchHash", PublicDeltaPatch(db))
	path := strings.TrimPrefix(url, "https://dist.example.com")
	get := func(path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
		return rr
	}
	rr := get(path)
	if rr.Code != 200 {
		t.Fatalf("manifest GET: %d", rr.Code)
	}
	var got deltaManifest
	json.Unmarshal(rr.Body.Bytes(), &got)
	if got.Patches[0].URL != m.Patches[0].SHA256 {
		t.Fatal("patch URL not normalized")
	}
	rr = get(strings.TrimSuffix(path, "manifest.json") + got.Patches[0].URL)
	if rr.Code != 200 || !bytes.Equal(rr.Body.Bytes(), patch) {
		t.Fatal("patch download mismatch")
	}
	status, payload := performPublicLatest(t, PublicLatestRelease(db, config.Default()), "/api/v1/products/"+product.Slug+"/latest?variant="+variant.ID, gin.Params{{Key: "slug", Value: product.Slug}})
	if status != 200 || payload["data"].(map[string]any)["release"].(map[string]any)["delta_manifest_url"] == "" {
		t.Fatalf("latest missing delta URL: status=%d payload=%v", status, payload)
	}
	// Historical logical builds still publish original APK version codes.
	db.Model(&base).Update("build", 12)
	db.Model(&target).Update("build", 13)
	if status := upload(m, patch); status != 200 {
		t.Fatalf("logical MyCHU build rejected: %d", status)
	}
	db.Model(&variant).Update("published", false)
	if get(path).Code != 404 {
		t.Fatal("unpublished patch exposed")
	}
	db.Model(&variant).Update("published", true)
	db.Model(&target).Update("sha256", base.SHA256)
	if get(path).Code != 404 {
		t.Fatal("stale manifest exposed after replacement")
	}
}

func TestDeltaS3Storage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openProductsAPITestDB(t)
	_, _, v := seedProductFixture(t, db)
	v.Platform = "android"
	v.Identifier = "test.android"
	v.Arch = "arm64-v8a"
	db.Save(&v)
	setS3StorageSettings(t, db)
	fake := &fakeS3StorageClient{}
	useFakeS3StorageClient(t, fake)
	r := store.Release{ID: "s3target", VariantID: v.ID, Version: "2", Build: 2, SizeBytes: 4, SHA256: deltaHash([]byte("fake")), FileExt: ".apk", StoragePath: "s3://target.apk", Channel: "official"}
	b := store.Release{ID: "s3base", VariantID: v.ID, Build: 1, SHA256: deltaHash([]byte("base")), Channel: r.Channel}
	db.Create(&b)
	db.Create(&r)
	p := []byte("p")
	m := deltaManifest{SchemaVersion: 1, PackageName: v.Identifier, Channel: r.Channel, ABI: v.Arch, Target: deltaTarget{Version: r.Version, VersionCode: r.Build, Size: r.SizeBytes, SHA256: r.SHA256}, Patches: []deltaPatch{{Algorithm: "hdiffpatch", Compression: "zlib", FromVersionCode: 1, FromSHA256: b.SHA256, Size: 1, SHA256: deltaHash(p)}}}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	raw, _ := json.Marshal(m)
	w.WriteField("manifest", string(raw))
	f, _ := w.CreateFormFile(m.Patches[0].SHA256, "p.hdiff")
	f.Write(p)
	w.Close()
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("POST", "/releases/s3target/delta", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	c.Params = gin.Params{{Key: "releaseID", Value: r.ID}}
	UploadDelta(db)(c)
	if rr.Code != 200 || len(fake.uploadKeys) != 1 || fake.uploadKeys[0] != "delta/"+deltaHash(p)+".hdiff" {
		t.Fatalf("S3 upload: %d %s keys=%v", rr.Code, rr.Body.String(), fake.uploadKeys)
	}
	db.First(&r, "id = ?", r.ID)
	engine := gin.New()
	engine.GET("/delta/:releaseID/:targetHash/:manifestHash/:patchHash", PublicDeltaPatch(db))
	rr = httptest.NewRecorder()
	engine.ServeHTTP(rr, httptest.NewRequest("GET", deltaPrefix(r)+deltaHash(p), nil))
	if rr.Code != 302 || rr.Header().Get("Location") != "https://cdn.example.com/delta/"+deltaHash(p)+".hdiff" {
		t.Fatalf("S3 redirect: %d %s", rr.Code, rr.Header().Get("Location"))
	}
}

func TestDeltaManifestMetadataChangesInvalidate(t *testing.T) {
	r := store.Release{Version: "1", Build: 2, SHA256: deltaHash([]byte("apk")), SizeBytes: 20, Channel: "official"}
	m := deltaManifest{Target: deltaTarget{Version: r.Version, VersionCode: r.Build, SHA256: r.SHA256, Size: r.SizeBytes}, Channel: r.Channel}
	raw, _ := json.Marshal(m)
	r.DeltaManifest = string(raw)
	if _, ok := activeDelta(r); !ok {
		t.Fatal("valid manifest rejected")
	}
	r.Build = 3
	if _, ok := activeDelta(r); ok {
		t.Fatal("metadata change did not invalidate")
	}
}
