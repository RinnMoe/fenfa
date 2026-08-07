# RinnMoe Fenfa

This repository is a maintained fork of [openprx/fenfa](https://github.com/openprx/fenfa).

## Upstream

- Upstream repository: https://github.com/openprx/fenfa
- Initial baseline commit: `e0fbb666d779274329091677e4d6218b15c6d81c`
- This fork is independently maintained and does not intend to submit PRs upstream.
- The `upstream` remote is kept for selectively cherry-picking security fixes or bugfixes.

## Differences from Upstream

### Release.force_update

`Release` model has a `force_update` boolean field (default `false`). Clients can query whether a release is marked as mandatory via the public latest API.

### Product Announcement

New `Announcement` model scoped to `Product`:

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | primary key |
| `product_id` | string | validated against existing product |
| `title` | string | required |
| `content` | text | required, plain text |
| `enabled` | bool | default `true` |
| `start_at` | datetime nullable | stored in UTC |
| `end_at` | datetime nullable | stored in UTC |
| `created_at` | datetime | |
| `updated_at` | datetime | |

### Public Client APIs

Two read-only endpoints for software clients (no admin token required):

```
GET /api/v1/products/:slug/latest?variant=<variantID>
GET /api/v1/products/:slug/announcements
```

The latest API reuses the existing `created_at DESC` ordering. The announcements API filters by `enabled=true` and current UTC time window.

### Admin APIs

Announcement CRUD and release force_update toggle:

```
GET    /admin/api/products/:productID/announcements
POST   /admin/api/products/:productID/announcements
GET    /admin/api/announcements/:announcementID
PUT    /admin/api/announcements/:announcementID
DELETE /admin/api/announcements/:announcementID
PUT    /admin/api/releases/:releaseID
```

### Admin UI

- Product detail row has an Announcements entry.
- Announcement modal: list, create, edit, enable/disable, delete.
- Upload forms include a Force Update checkbox.
- Latest release card can toggle force_update directly.

### Other Changes

- `.gitattributes` enforces LF for shell scripts.
- `Dockerfile` supports `GOPROXY` build arg.
- `.github/workflows/docker-publish.yml` publishes to GHCR on tag push.
- Frontend dependencies updated to resolve npm audit findings (vite 8.0 -> 8.2).

## Database Migration

GORM `AutoMigrate` handles everything automatically:

- Adds `releases.force_update` with default `false`
- Creates `announcements` table
- Existing data is preserved
- Idempotent across restarts

No manual SQL or database reset required.

## Docker

```bash
docker build -t ghcr.io/rinnmoe/fenfa:v0.1.0 .
```

Production deployment uses 1Panel with reverse proxy. See [DEPLOY_1PANEL.md](DEPLOY_1PANEL.md).
