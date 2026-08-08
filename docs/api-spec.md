# API Specification

## Response Format

All JSON responses use a unified structure:

```json
// Success
{"ok": true, "data": {...}}

// Error
{"ok": false, "error": {"code": "BAD_REQUEST", "message": "..."}}
```

## Authentication

Protected endpoints require `X-Auth-Token` header. Two token scopes:

- **upload** — can upload builds
- **admin** — full admin access (includes upload)

## Public Endpoints

### Product Page

```
GET /products/:slug
```

HTML page with download buttons, QR code, platform detection. Supports `?r=<release_id>` to highlight a specific release.

### Direct Download

```
GET /d/:releaseID
```

Returns binary file (IPA/APK/etc). Supports HTTP Range requests. Increments download counter.

### iOS Manifest

```
GET /ios/:releaseID/manifest.plist
```

Returns `application/xml` plist for `itms-services://` installation. Contains bundle identifier, version, and download URL.

### UDID Binding (iOS)

```
GET  /udid/profile.mobileconfig?variant=<variantID>
POST /udid/callback                                    # Called by iOS automatically
GET  /udid/status?variant=<variantID>                  # Check if device is bound
```

### Health Check

```
GET /healthz
```

Returns `{"ok": true}` if the server is running.

## Upload

### Standard Upload

```
POST /upload
Content-Type: multipart/form-data
X-Auth-Token: <upload_token>
```

| Field | Required | Description |
|-------|----------|-------------|
| `variant_id` | yes | Target variant ID |
| `app_file` | yes | IPA/APK/binary file |
| `version` | no | Version string (e.g. "1.0.0") |
| `build` | no | Build number (integer) |
| `channel` | no | Channel (e.g. "internal", "beta") |
| `min_os` | no | Minimum OS version |
| `changelog` | no | Release notes |

**Response** (201):

```json
{
  "ok": true,
  "data": {
    "app": {"id": "app_xxx", "name": "MyApp", "platform": "ios", "bundle_id": "com.example.app"},
    "release": {"id": "rel_xxx", "version": "1.0.0", "build": 1},
    "urls": {
      "page": "https://dist.example.com/products/my-app",
      "download": "https://dist.example.com/d/rel_xxx",
      "ios_manifest": "https://dist.example.com/ios/rel_xxx/manifest.plist",
      "ios_install": "itms-services://..."
    }
  }
}
```

### Smart Upload

```
POST /admin/api/smart-upload
Content-Type: multipart/form-data
X-Auth-Token: <admin_token>
```

Auto-detects metadata from the package (bundle ID, version, icon, etc). Same fields as standard upload but `version` and `build` are auto-parsed.

## Admin API

All admin endpoints require `X-Auth-Token` with admin scope.

### Products

```
GET    /admin/api/products                              # List products
POST   /admin/api/products                              # Create product
GET    /admin/api/products/:productID                   # Get product with variants
PUT    /admin/api/products/:productID                   # Update product
DELETE /admin/api/products/:productID                   # Delete product (cascades)
```

### Variants

```
POST   /admin/api/products/:productID/variants          # Create variant
PUT    /admin/api/variants/:variantID                    # Update variant
DELETE /admin/api/variants/:variantID                    # Delete variant (cascades)
GET    /admin/api/variants/:variantID/stats              # Variant visit/download totals
```

### Releases

```
PUT    /admin/api/releases/:releaseID                    # Update release metadata
DELETE /admin/api/releases/:releaseID                    # Delete release
GET    /admin/api/releases/:releaseID/file               # Download installer (admin only)
```

`PUT /admin/api/releases/:releaseID` accepts a JSON body and performs a partial update. At least one field is required. `version` must be non-empty; `build` must be non-negative. Empty strings clear `channel`, `min_os`, and `changelog`.

```json
{
  "version": "1.1.0",
  "build": 10,
  "channel": "beta",
  "min_os": "iOS 15.0",
  "changelog": "Bug fixes",
  "force_update": true
}
```

`GET /admin/api/releases/:releaseID/file` returns the stored installer without requiring the product/variant to be published. Local files are streamed directly; S3 objects are fetched and streamed through the server so the admin token stays out of the browser URL.

### Replace Installer

```
POST /<admin_path>/api/releases/:releaseID/replace-file
Content-Type: multipart/form-data
X-Auth-Token: <admin_token>
```

Replaces the installer payload of an existing release while keeping the release ID and download URL unchanged. Can be sent from the configured upload domain.

| Field | Required | Description |
|-------|----------|-------------|
| `app_file` | yes | New IPA/APK/binary file |
| `version` | yes | Version string (e.g. "1.0.0") |
| `build` | no | Build number (integer) |
| `channel` | no | Channel (e.g. "internal", "beta") |
| `min_os` | no | Minimum OS version |
| `changelog` | no | Release notes |
| `force_update` | no | `true` or `false` |
| `icon_base64` | no | Data URL/base64 PNG icon to update the product icon |

APK/IPA packages are re-parsed. If the parsed package identifier does not match the variant identifier, the request is rejected with `400`. For iOS, the stored provisioning profile is refreshed from the new IPA.

### Publishing

```
PUT /admin/api/apps/:appID/publish                      # Publish app
PUT /admin/api/apps/:appID/unpublish                    # Unpublish app
```

### Devices & Stats

```
GET  /admin/api/ios_devices                             # List iOS devices
GET  /admin/api/variants/:variantID/stats               # Visit and download totals
POST /admin/api/devices/:id/register-apple              # Register device with Apple
POST /admin/api/devices/register-apple                  # Batch register devices
```

### Settings

```
GET /admin/api/settings                                 # Get system settings
PUT /admin/api/settings                                 # Update settings
GET /admin/api/upload-config                            # Get upload configuration
GET /admin/api/apple/status                             # Check Apple API status
GET /admin/api/apple/devices                            # List Apple-registered devices
```

### Export

```
GET /admin/exports/releases.csv                         # Export releases
GET /admin/exports/ios_devices.csv                      # Export iOS devices
```

## Storage

Files are stored at: `uploads/{product_id}/{variant_id}/{release_id}/filename.ext`

Each release also has a `meta.json` snapshot (local storage only).

## ID Format

IDs use prefix + random hex: `app_7f3xn`, `rel_b1cqa`, `prd_abc123`, `var_def456`.
