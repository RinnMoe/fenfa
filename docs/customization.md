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

### User Feedback & Email Notifications

- Public product pages include an anonymous feedback form backed by `POST /api/v1/feedback`.
- Admin feedback inbox supports listing, status/category/product filtering, pagination, processing/reopening, and deletion.
- Accepted feedback is persisted first; when SMTP is enabled, an async plain-text notification is sent to the configured recipient.
- SMTP is configured in Admin > Settings > Email Notifications and supports STARTTLS, SSL/TLS, and unencrypted connections. `POST /<admin_path>/api/settings/email/test` sends a test email without saving the form.
- The SMTP password is never returned by the API; `smtp_password_set` indicates whether one is stored.

### Proxy Trust

- `FENFA_TRUSTED_PROXIES` / `trusted_proxies` restricts which proxy IPs/CIDRs may supply client IP headers.
- Leave it empty when Fenfa is directly exposed; configure it to the actual reverse proxy IP/CIDR when behind Nginx or 1Panel.

### Admin APIs

Additional admin endpoints on top of upstream include:

```
GET    /<admin_path>/api/products/:productID/announcements
POST   /<admin_path>/api/products/:productID/announcements
GET    /<admin_path>/api/announcements/:announcementID
PUT    /<admin_path>/api/announcements/:announcementID
DELETE /<admin_path>/api/announcements/:announcementID
PUT    /<admin_path>/api/releases/:releaseID
GET    /<admin_path>/api/releases/:releaseID/file
POST   /<admin_path>/api/releases/:releaseID/replace-file
GET    /<admin_path>/api/feedback
PATCH  /<admin_path>/api/feedback/:feedbackID
DELETE /<admin_path>/api/feedback/:feedbackID
GET    /<admin_path>/api/variants/:variantID/stats
GET    /<admin_path>/api/settings
PUT    /<admin_path>/api/settings
POST   /<admin_path>/api/settings/email/test
```

### Admin UI

- Product detail row has an Announcements entry.
- Announcement modal: list, create, edit, enable/disable, delete.
- Upload forms include a Force Update checkbox.
- Latest release card can toggle force_update directly.
- Release cards support post-release metadata edit, installer replacement, and admin download.
- Feedback inbox supports filters, pagination, processing/reopening, and deletion.
- Settings includes Storage and Email Notifications sections with a test-email button.

### Other Changes

- Landing page, configurable `admin_path`, and 404 page.
- Visit/download statistics simplified from raw event logs to variant totals.
- S3/R2 staging cleanup and object deletion when releases are removed.
- `.gitattributes` enforces LF for shell scripts.
- `Dockerfile` supports `GOPROXY` build arg.
- `.github/workflows/docker-publish.yml` publishes to GHCR on tag push.
- Frontend dependencies updated to resolve npm audit findings (vite 8.0 -> 8.2).

## Database Migration

GORM `AutoMigrate` handles schema changes idempotently:

- Adds `releases.force_update` with default `false`
- Creates `announcements` and `feedbacks` tables
- Adds SMTP/email columns to `system_settings`
- v0.3.3 migration backfills variant visit counts from old `events` rows and then drops the `events` table
- Existing data is otherwise preserved

Backup the database before upgrading, because the legacy events cleanup is not reversible.

## Docker

```bash
docker build -t ghcr.io/rinnmoe/fenfa:v0.4.0 .
```

Production deployment uses 1Panel with reverse proxy. See [DEPLOY_1PANEL.md](DEPLOY_1PANEL.md).
