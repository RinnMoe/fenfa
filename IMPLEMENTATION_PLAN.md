# Fenfa Minimal Customization Implementation Plan

## Baseline

- Repository: `https://github.com/RinnMoe/fenfa.git`
- Upstream baseline commit: `e0fbb666d779274329091677e4d6218b15c6d81c`
- Branch: `feat/minimal-release-announcement`
- Backend: Go 1.25+ / Gin / GORM / SQLite
- Frontend: Vue 3 + Vite in `web/front` (public) and `web/admin` (admin)
- Storage: existing local/S3 abstraction unchanged
- Baseline verified: `go test ./...` passed, both Vite builds passed, server `/healthz` returned 200

## Architecture Findings

- Product/Variant/Release models: `internal/store/models.go`
- Migrations: `internal/store/migrations.go` via GORM `AutoMigrate`
- Router: `internal/server/router.go`
- Admin handlers: `internal/server/handlers/*.go`, protected by `middleware.RequireToken(cfg, "admin")`
- Release creation: `POST /upload` and `POST /smart-upload`; there is no upstream release edit endpoint
- Release latest logic used by public page: releases under a variant are ordered by `created_at desc`, first item is latest
- Download URL construction: existing `resolvePublicDomains` + `releaseActions`; download remains `/d/:releaseID`
- API envelope: `{"ok": true, "data": ...}` / `{"ok": false, "error": {"code": ..., "message": ...}}`
- Admin UI has no central API client; pages call `fetch` with `X-Auth-Token`
- i18n: `web/admin/src/i18n/en.ts` and `zh.ts`, default locale `zh`

## Scope

Only these additions are implemented:

1. `Release.force_update` bool, default `false`
2. Product-scoped `Announcement` model and Admin CRUD
3. Announcement management in the existing Apps page via a product modal
4. Public read-only `GET /api/v1/products/:slug/latest`
5. Public read-only `GET /api/v1/products/:slug/announcements`

No new storage abstraction, no R2/S3 SDK, no new infrastructure, no new auth system.

## Files To Change

- `internal/store/models.go`: add `ForceUpdate` to `Release`; add `Announcement` model
- `internal/store/migrations.go`: register `Announcement` in `AutoMigrate`
- `internal/server/router.go`: register announcement admin routes, release update route, and public v1 routes
- `internal/server/handlers/products_api.go`: include `force_update` in release DTOs
- `internal/server/handlers/products_admin.go`: include `force_update` in admin release DTOs; cascade delete announcements with product; add minimal `AdminUpdateRelease`
- `internal/server/handlers/apps_api.go` and `apps_admin.go`: include `force_update` in legacy release DTOs for consistency
- `internal/server/handlers/upload.go`: accept and persist `force_update`
- `internal/server/handlers/smart_upload.go`: accept and pass through `force_update`
- `web/admin/src/components/UploadModal.vue`: add force update checkbox
- `web/admin/src/components/SmartUpload.vue`: add force update checkbox
- `web/admin/src/pages/UploadPage.vue`: add force update checkbox
- `web/admin/src/pages/AppsPage.vue`: show/edit force update on latest release; add Announcements entry
- `web/admin/src/i18n/en.ts`, `zh.ts`: add release/announcement labels

## Files To Add

- `internal/server/handlers/announcements.go`: Admin announcement CRUD handlers
- `internal/server/handlers/public_api.go`: public latest and announcements handlers
- `web/admin/src/components/AnnouncementModal.vue`: product announcement list/edit modal
- `internal/server/handlers/release_force_update_test.go`: release flag tests
- `internal/server/handlers/announcements_test.go`: announcement CRUD/public filtering tests
- `internal/server/handlers/public_api_test.go`: latest API tests
- `internal/server/router_test.go`: auth and public no-token route tests

## Database Migration

- `Release.force_update`: GORM `AutoMigrate` adds a `force_update` boolean column with `default:false`
- Existing rows are treated as `false` without manual SQL
- `Announcement`: added to `AutoMigrate`; table is created on existing databases
- `AutoMigrate` remains idempotent; restart does not reinitialize or drop data

## Admin API Changes

- `GET /admin/api/products/:productID/announcements` (admin)
- `POST /admin/api/products/:productID/announcements` (admin)
- `GET /admin/api/announcements/:announcementID` (admin)
- `PUT /admin/api/announcements/:announcementID` (admin)
- `DELETE /admin/api/announcements/:announcementID` (admin)
- `PUT /admin/api/releases/:releaseID` (admin, minimal `force_update` update)

Announcement validation: product exists, title/content non-empty, `start_at <= end_at` when both set.

## Public API Changes

`GET /api/v1/products/:slug/latest?variant=<variantID>`

- Requires the existing variant ID used by the public page
- Returns product, variant, latest release by `created_at desc`, `force_update`, download URL, release page
- 404 for missing product/variant/release, with existing error envelope

`GET /api/v1/products/:slug/announcements`

- Returns enabled announcements for the product whose time window is currently active
- Product missing: 404; no announcements: 200 with empty `items`

## Test Plan

- Go: release default/true/false create and update, announcement CRUD, time filtering, public latest, public announcements, route auth
- Frontend: `npm run build` for both apps
- Database upgrade: create a pre-change DB with old `releases` table, start new server twice, verify column and announcement table
- Docker: `docker build -t fenfa-custom:dev .` using upstream Dockerfile
- Docs: `docs/customization.md` with API examples and migration notes
