package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/openprx/fenfa/internal/store"
	"gorm.io/gorm"
)

type deltaTarget struct {
	Version     string `json:"version"`
	VersionCode int64  `json:"version_code"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}
type deltaPatch struct {
	Algorithm       string `json:"algorithm"`
	Compression     string `json:"compression"`
	FromVersionCode int64  `json:"from_version_code"`
	FromSHA256      string `json:"from_sha256"`
	Size            int64  `json:"size"`
	SHA256          string `json:"sha256"`
	URL             string `json:"url"`
}
type deltaManifest struct {
	SchemaVersion int          `json:"schema_version"`
	PackageName   string       `json:"package_name"`
	Channel       string       `json:"channel"`
	ABI           string       `json:"abi"`
	Target        deltaTarget  `json:"target"`
	Patches       []deltaPatch `json:"patches"`
}

func validDeltaHash(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && s == strings.ToLower(s)
}
func deltaHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// MyCHU's official Android releases also historically used a logical build
// number in Fenfa; the installed APK adds the documented 2000 offset.
func deltaBuildMatches(packageName string, raw, build int64) bool {
	return raw == build || (packageName == "moe.rinn.mychu" && build > 0 && build < 2000 && raw == build+2000)
}
func deltaPrefix(r store.Release) string {
	return "/delta/" + r.ID + "/" + r.SHA256 + "/" + deltaHash([]byte(r.DeltaManifest)) + "/"
}
func activeDelta(r store.Release) (deltaManifest, bool) {
	var m deltaManifest
	err := json.Unmarshal([]byte(r.DeltaManifest), &m)
	if r.ExternalURL != "" {
		return m, false
	}
	return m, err == nil && m.Target.SHA256 == r.SHA256 && m.Target.Size == r.SizeBytes && m.Target.Version == r.Version && deltaBuildMatches(m.PackageName, m.Target.VersionCode, r.Build) && m.Channel == r.Channel
}
func deltaManifestURL(r store.Release, base string) string {
	if _, ok := activeDelta(r); !ok || !strings.HasPrefix(base, "https://") {
		return ""
	}
	return strings.TrimRight(base, "/") + deltaPrefix(r) + "manifest.json"
}
func deltaError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"ok": false, "error": gin.H{"code": "DELTA_INVALID", "message": message}})
}

// UploadDelta accepts a manifest form field and one file field per patch SHA256.
// Files are staged and verified before a compare-and-swap attaches the manifest.
func UploadDelta(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512<<20)
		if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
			deltaError(c, 400, "invalid or oversized multipart body")
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		var r store.Release
		if err := db.First(&r, "id = ?", c.Param("releaseID")).Error; err != nil {
			deltaError(c, 404, "release not found")
			return
		}
		ctx, err := loadReleaseVariantContext(db, r)
		if r.ExternalURL != "" {
			deltaError(c, 400, "delta attachments require a Fenfa-hosted APK")
			return
		}
		if err != nil || ctx.Variant.Platform != "android" || r.FileExt != ".apk" {
			deltaError(c, 400, "delta requires an Android APK release")
			return
		}
		var m deltaManifest
		raw := c.PostForm("manifest")
		if len(raw) > 65536 || json.Unmarshal([]byte(raw), &m) != nil {
			deltaError(c, 400, "invalid manifest JSON")
			return
		}
		if m.SchemaVersion != 1 || m.PackageName != ctx.Variant.Identifier || m.PackageName == "" || m.Channel != r.Channel || m.ABI != ctx.Variant.Arch || m.ABI == "" || m.Target.Version != r.Version || !deltaBuildMatches(m.PackageName, m.Target.VersionCode, r.Build) || m.Target.Size != r.SizeBytes || m.Target.SHA256 != r.SHA256 || !validDeltaHash(m.Target.SHA256) || len(m.Patches) < 1 || len(m.Patches) > 8 {
			deltaError(c, 400, "manifest does not match release metadata or supported schema")
			return
		}
		seen := map[string]bool{}
		seenFiles := map[string]bool{}
		for i := range m.Patches {
			p := &m.Patches[i]
			key := fmt.Sprintf("%d:%s", p.FromVersionCode, p.FromSHA256)
			if p.Algorithm != "hdiffpatch" || p.Compression != "zlib" || p.FromVersionCode <= 0 || p.FromVersionCode >= m.Target.VersionCode || !validDeltaHash(p.FromSHA256) || !validDeltaHash(p.SHA256) || p.Size <= 0 || p.Size >= r.SizeBytes || seen[key] || seenFiles[p.SHA256] {
				deltaError(c, 400, "invalid or duplicate patch")
				return
			}
			seen[key] = true
			seenFiles[p.SHA256] = true
			var base store.Release
			if db.Where("variant_id = ? AND sha256 = ? AND channel = ?", r.VariantID, p.FromSHA256, r.Channel).First(&base).Error != nil || !deltaBuildMatches(m.PackageName, p.FromVersionCode, base.Build) {
				deltaError(c, 400, "patch base must be a known exact release in the same variant and channel")
				return
			}
			p.URL = p.SHA256 // Relative to immutable manifest URL; caller URLs are never trusted.
		}
		canonical, _ := json.Marshal(m)
		// Verify the currently stored full APK, including S3 objects, rather than relying only on metadata.
		var settings store.SystemSettings
		if err := db.First(&settings, "id = ?", "default").Error; err != nil {
			deltaError(c, 500, "storage settings unavailable")
			return
		}
		var full io.ReadCloser
		if isS3StoragePath(r.StoragePath) {
			full, err = newS3StorageClient(settings).Get(c.Request.Context(), s3KeyFromStoragePath(r.StoragePath))
		} else {
			full, err = os.Open(r.StoragePath)
		}
		if err != nil {
			deltaError(c, 400, "target APK unavailable")
			return
		}
		h := sha256.New()
		n, copyErr := io.Copy(h, full)
		full.Close()
		if copyErr != nil || n != r.SizeBytes || hex.EncodeToString(h.Sum(nil)) != r.SHA256 {
			deltaError(c, 400, "stored target APK does not match release hash")
			return
		}
		staged := make([]string, 0, len(m.Patches))
		defer func() {
			for _, p := range staged {
				os.Remove(p)
			}
		}()
		for _, p := range m.Patches {
			files := c.Request.MultipartForm.File[p.SHA256]
			if len(files) != 1 || files[0].Size != p.Size {
				deltaError(c, 400, "missing patch file or size mismatch: "+p.SHA256)
				return
			}
			src, e := files[0].Open()
			if e != nil {
				deltaError(c, 400, "cannot read patch")
				return
			}
			tmp, e := os.CreateTemp("", "fenfa-delta-*")
			if e != nil {
				src.Close()
				deltaError(c, 500, "cannot stage patch")
				return
			}
			staged = append(staged, tmp.Name())
			h = sha256.New()
			n, e = io.Copy(io.MultiWriter(tmp, h), src)
			src.Close()
			closeErr := tmp.Close()
			if e != nil || closeErr != nil || n != p.Size || hex.EncodeToString(h.Sum(nil)) != p.SHA256 {
				deltaError(c, 400, "patch hash mismatch")
				return
			}
		}
		// Content-addressed storage is shared by identical patch bytes. Unreferenced
		// objects are retained rather than deleting files used by concurrent uploads.
		for i, p := range m.Patches {
			if isS3StoragePath(r.StoragePath) {
				f, e := os.Open(staged[i])
				if e != nil {
					deltaError(c, 500, "cannot read staged patch")
					return
				}
				e = newS3StorageClient(settings).Upload(c.Request.Context(), "delta/"+p.SHA256+".hdiff", f, "application/octet-stream")
				f.Close()
				if e != nil {
					deltaError(c, 500, "cannot store patch")
					return
				}
			} else {
				dir := filepath.Join("uploads", "delta")
				if os.MkdirAll(dir, 0755) != nil {
					deltaError(c, 500, "cannot create patch directory")
					return
				}
				// Windows rename cannot replace an existing destination. Identical
				// content is reusable; verify it before accepting a retry.
				dest := filepath.Join(dir, p.SHA256+".hdiff")
				if existing, err := os.Open(dest); err == nil {
					h := sha256.New()
					n, err := io.Copy(h, existing)
					existing.Close()
					if err == nil && n == p.Size && hex.EncodeToString(h.Sum(nil)) == p.SHA256 {
						continue
					}
					deltaError(c, 500, "existing content-addressed patch is corrupted")
					return
				}
				f, e := os.Open(staged[i])
				if e != nil {
					deltaError(c, 500, "cannot read staged patch")
					return
				}
				out, e := os.CreateTemp(dir, ".delta-*")
				if e != nil {
					f.Close()
					deltaError(c, 500, "cannot store patch")
					return
				}
				_, e = io.Copy(out, f)
				f.Close()
				ce := out.Close()
				if e == nil {
					e = ce
				}
				if e == nil {
					e = os.Rename(out.Name(), dest)
				}
				os.Remove(out.Name())
				if e != nil {
					deltaError(c, 500, "cannot store patch")
					return
				}
			}
		}
		result := db.Model(&store.Release{}).Where("id = ? AND sha256 = ? AND storage_path = ? AND version = ? AND build = ? AND channel = ? AND size_bytes = ?", r.ID, r.SHA256, r.StoragePath, r.Version, r.Build, r.Channel, r.SizeBytes).Update("delta_manifest", string(canonical))
		if result.Error != nil {
			deltaError(c, 500, "cannot attach manifest")
			return
		}
		if result.RowsAffected != 1 {
			deltaError(c, 409, "release changed during upload; retry with current APK")
			return
		}
		c.JSON(200, gin.H{"ok": true, "data": gin.H{"manifest": m}})
	}
}

func DeleteDelta(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := db.Model(&store.Release{}).Where("id = ?", c.Param("releaseID")).Update("delta_manifest", "")
		if result.Error != nil {
			deltaError(c, 500, "cannot detach manifest")
			return
		}
		if result.RowsAffected != 1 {
			deltaError(c, 404, "release not found")
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func publicDeltaRelease(c *gin.Context, db *gorm.DB) (store.Release, deltaManifest, bool) {
	var r store.Release
	if db.First(&r, "id = ?", c.Param("releaseID")).Error != nil {
		deltaError(c, 404, "release not found")
		return r, deltaManifest{}, false
	}
	ctx, err := loadReleaseVariantContext(db, r)
	m, ok := activeDelta(r)
	if err != nil || !ctx.Product.Published || !ctx.Variant.Published || !ok || m.PackageName != ctx.Variant.Identifier || m.ABI != ctx.Variant.Arch || c.Param("targetHash") != r.SHA256 || c.Param("manifestHash") != deltaHash([]byte(r.DeltaManifest)) {
		deltaError(c, 404, "delta unavailable")
		return r, m, false
	}
	return r, m, true
}
func PublicDeltaManifest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, _, ok := publicDeltaRelease(c, db)
		if !ok {
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(200, "application/json", []byte(r.DeltaManifest))
	}
}
func PublicDeltaPatch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, m, ok := publicDeltaRelease(c, db)
		if !ok {
			return
		}
		hash := c.Param("patchHash")
		found := false
		for _, p := range m.Patches {
			if p.SHA256 == hash {
				found = true
				break
			}
		}
		if !found {
			deltaError(c, 404, "patch not found")
			return
		}
		c.Header("Cache-Control", "no-cache, no-transform")
		c.Header("Content-Type", "application/octet-stream")
		if isS3StoragePath(r.StoragePath) {
			var s store.SystemSettings
			if db.First(&s, "id = ?", "default").Error != nil || !strings.HasPrefix(s.S3PublicURL, "https://") {
				deltaError(c, 404, "HTTPS object storage URL unavailable")
				return
			}
			c.Redirect(http.StatusFound, strings.TrimRight(s.S3PublicURL, "/")+"/delta/"+hash+".hdiff")
			return
		}
		c.File(filepath.Join("uploads", "delta", hash+".hdiff"))
	}
}
