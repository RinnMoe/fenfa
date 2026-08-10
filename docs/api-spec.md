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

### User Feedback

```http
POST /api/v1/feedback
Content-Type: application/json
```

The endpoint is public and accepts anonymous feedback submitted from a published product page. It does not require `X-Auth-Token`.

| Field | Required | Description |
|-------|----------|-------------|
| `product_id` | yes | Published product ID |
| `variant_id` | no | Published variant belonging to the product |
| `release_id` | no | Release belonging to the selected variant |
| `category` | yes | `bug`, `suggestion`, or `other` |
| `content` | yes | Feedback text, up to 4000 characters |
| `contact` | no | Free-form contact text, up to 255 characters |
| `website` | no | Honeypot field; should be empty |

**Response** (201):

```json
{
  "ok": true,
  "data": {
    "feedback": {
      "id": "fb_xxx",
      "status": "pending",
      "created_at": "2026-08-09T00:00:00Z"
    }
  }
}
```

Requests are limited to five accepted submissions per client IP per ten minutes. Invalid or oversized requests are separately limited to 30 requests per client IP per ten minutes. Rate-limited requests return `429` with `Retry-After`. The in-process limit is intended for a single Fenfa process; multi-instance deployments should add a shared or edge rate limiter.

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

### Feedback

All feedback endpoints below require the admin token. Replace `<admin_path>` with the configured admin path (default `admin`).

```
GET    /<admin_path>/api/feedback
PATCH  /<admin_path>/api/feedback/:feedbackID
DELETE /<admin_path>/api/feedback/:feedbackID
```

`GET` supports `page`, `page_size` (1–100, default 20), `status`, `category`, and `product_id`. Results are ordered by `created_at DESC, id DESC` and return `items`, `total`, `page`, `page_size`, and `total_pages`.

`PATCH` accepts `{"status":"pending"}` or `{"status":"processed"}`. `DELETE` permanently removes the feedback item and should only be called after an administrative confirmation.

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
POST /<admin_path>/api/settings/email/test             # Send a test email without saving
GET /admin/api/upload-config                            # Get upload configuration
GET /admin/api/apple/status                             # Check Apple API status
GET /admin/api/apple/devices                            # List Apple-registered devices
```

`GET /<admin_path>/api/settings` returns SMTP settings without the password. The
response includes `email_enabled`, `email_configured`, `smtp_host`,
`smtp_port`, `smtp_encryption` (`starttls`, `ssl`, or `none`),
`smtp_username`, `smtp_from`, `smtp_to`, and `smtp_password_set`.

`PUT /<admin_path>/api/settings` accepts the same SMTP fields plus `smtp_password`.
An empty password keeps the saved value. When `email_enabled` is true, the
SMTP host, port, encryption mode, sender, and recipient must be valid.

`POST /<admin_path>/api/settings/email/test` accepts the current SMTP form values and
sends a plain-text test message without persisting them. An empty password
reuses the saved SMTP password. SMTP errors return `502` with
`EMAIL_SEND_FAILED`.

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
