# Fenfa Customization

This document describes the minimal customization applied on top of Fenfa for self-hosted software version management, release, and distribution.

## Upstream Baseline

- Upstream repository: `https://github.com/openprx/fenfa`
- Mirror used for this clone: `https://github.com/RinnMoe/fenfa.git`
- Baseline commit: `e0fbb666d779274329091677e4d6218b15c6d81c`
- Custom branch: `feat/minimal-release-announcement`

## Added Features

1. `Release.force_update` boolean, default `false`
2. Product-scoped `Announcement` model
3. Announcement Admin CRUD
4. Public latest release API
5. Public announcements API

## Database Migration

The existing GORM `AutoMigrate` flow is reused. Starting the new server on an existing SQLite database:

- Adds `releases.force_update` with default `false`
- Creates the `announcements` table
- Keeps existing rows untouched
- Is idempotent across restarts

No manual SQL or database reset is required.

## Announcement Model

`announcements`

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | primary key |
| `product_id` | string | validated against existing product |
| `title` | string | required |
| `content` | text | required, plain text |
| `enabled` | bool | default `true` |
| `start_at` | datetime nullable | empty means no start bound |
| `end_at` | datetime nullable | empty means no end bound |
| `created_at` | datetime | |
| `updated_at` | datetime | |

## Admin API

All admin endpoints require `X-Auth-Token` with admin scope.

### Announcements

```text
GET    /admin/api/products/:productID/announcements
POST   /admin/api/products/:productID/announcements
GET    /admin/api/announcements/:announcementID
PUT    /admin/api/announcements/:announcementID
DELETE /admin/api/announcements/:announcementID
```

Create example:

```json
{
  "title": "Maintenance",
  "content": "Service will be upgraded.",
  "enabled": true,
  "start_at": "2026-08-01T00:00:00Z",
  "end_at": "2026-08-02T00:00:00Z"
}
```

Validation:

- Product must exist
- Title and content must be non-empty
- `start_at` must not be later than `end_at` when both are present

### Release Force Update

```text
PUT /admin/api/releases/:releaseID
```

```json
{
  "force_update": true
}
```

## Public API

Public endpoints are read-only and do not require admin or upload tokens.

### Latest Release

```text
GET /api/v1/products/:slug/latest?variant=<variantID>
```

For products with exactly one published variant, `variant` can be omitted. For multi-variant products, `variant` is required and must be an existing published variant ID.

The latest release uses the same ordering as the existing public product page: `created_at DESC`.

Example response:

```json
{
  "ok": true,
  "data": {
    "product": {
      "id": "prd_xxx",
      "slug": "demo-app",
      "name": "Demo App",
      "published": true
    },
    "variant": {
      "id": "var_xxx",
      "product_id": "prd_xxx",
      "platform": "windows",
      "identifier": "com.example.app",
      "display_name": "Demo Windows",
      "arch": "",
      "installer_type": "exe",
      "min_os": "",
      "published": true
    },
    "release": {
      "id": "rel_xxx",
      "version": "1.2.0",
      "build": 12,
      "changelog": "Release notes",
      "force_update": false,
      "created_at": "2026-08-07T00:00:00Z",
      "min_os": "",
      "channel": "release",
      "download_url": "https://dist.example.com/d/rel_xxx",
      "release_page": "https://dist.example.com/products/demo-app?r=rel_xxx",
      "actions": {
        "download": "https://dist.example.com/d/rel_xxx",
        "release_page": "https://dist.example.com/products/demo-app?r=rel_xxx"
      }
    }
  }
}
```

### Announcements

```text
GET /api/v1/products/:slug/announcements
```

Returns only enabled announcements for the product that are currently within their optional `start_at` / `end_at` window. Missing products return 404. Products without announcements return 200 with an empty `items` array.

Example response:

```json
{
  "ok": true,
  "data": {
    "items": [
      {
        "id": "ann_xxx",
        "title": "Maintenance",
        "content": "Service will be upgraded.",
        "start_at": "2026-08-01T00:00:00Z",
        "end_at": "2026-08-02T00:00:00Z"
      }
    ]
  }
}
```

## Admin UI

- The Apps page now has an `Announcements` entry inside each product detail row.
- Announcement management is implemented as a product-scoped modal with list, create, edit, enable/disable, and delete actions.
- Release upload forms now include a `Force Update` checkbox.
- The latest release shown in a product variant card can toggle `force_update` directly.
- UI text is covered by the existing Chinese and English i18n files.

## Docker Build

Use the upstream Dockerfile:

```bash
docker build -t fenfa-custom:dev .
```

Production port recommendation:

```yaml
services:
  fenfa:
    image: fenfa-custom:latest
    ports:
      - "127.0.0.1:8000:8000"
    volumes:
      - ./data:/data
      - ./uploads:/app/uploads
    environment:
      FENFA_ADMIN_TOKEN: "replace-with-secret"
      FENFA_UPLOAD_TOKEN: "replace-with-secret"
```

Note: when building from a Windows checkout, `scripts/entrypoint.sh` may be checked out with CRLF and fail inside Linux containers. Build from a Linux checkout, normalize the file to LF, or override the container entrypoint for local testing.
