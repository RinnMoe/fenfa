# Fenfa Frontend Redesign - Phase 0 Baseline Audit

Branch: `feat/frontend-redesign`

Base commit: `c391cc9`

Date: 2026-08-07

## 1. Scope

This audit covers only the two frontend applications:

- `web/admin` - Vue 3 + Vite admin SPA
- `web/front` - Vue 3 + Vite public release SPA

Backend, API, database, storage, auth, UDID, and deployment behavior are out of
scope and were not modified.

## 2. Frontend architecture

### Admin

- Runtime: Vue 3.5, Vue Router 4.6, TypeScript, native CSS, inline SVG icon component.
- Entry: `web/admin/src/main.ts`
- Shell: `web/admin/src/App.vue`
- Router: hash history; `/apps`, `/upload`, `/stats/:variantId`, `/events/:variantId`, `/udid`, `/export`, `/settings`.
- Global CSS: `web/admin/src/style.css` (tokens/theme) + `web/admin/src/styles/common.css` (shared patterns).
- State: local composables only (`useAuth`, `useProducts`, `useToast`, `useConfirmDialog`, `useProductVariantSelector`).
- Pages: `AppsPage`, `UploadPage`, `StatsPage`, `EventsPage`, `UdidPage`, `ExportPage`, `SettingsPage`, `LoginPage`.
- Overlays: `AnnouncementModal`, `DistributionModal`, `UploadModal`, `SmartUpload`, `ConfirmDialog`, `Toast`, `ProfileModal`, `Pagination`.

### Public front

- Runtime: Vue 3.5, TypeScript, native CSS, qrcode, ua-parser-js.
- Entry: `web/front/src/main.ts` contains all types, API calls, state, tracking, UDID logic, and the full template in one file.
- There are no Vue SFC components and no router; the app is mounted directly from `main.ts`.

## 3. API usage map

All frontend API calls are listed below. The redesign must preserve these exact
URLs, methods, headers, request fields, and response assumptions.

### Admin

- `GET /admin/api/products?q=` with `X-Auth-Token` (login verification and list)
- `GET /admin/api/products?q=<query>`
- `GET /admin/api/products/:productID`
- `POST /admin/api/products`
- `PUT /admin/api/products/:productID`
- `DELETE /admin/api/products/:productID`
- `POST /admin/api/products/:productID/variants`
- `PUT /admin/api/variants/:variantID`
- `DELETE /admin/api/variants/:variantID`
- `GET /admin/api/variants/:variantID/stats`
- `PUT /admin/api/releases/:releaseID`
- `GET /admin/api/products/:productID/announcements`
- `POST /admin/api/products/:productID/announcements`
- `PUT /admin/api/announcements/:announcementID`
- `DELETE /admin/api/announcements/:announcementID`
- `GET /admin/api/events`
- `GET /admin/api/ios_variants`
- `GET /admin/api/ios_devices`
- `POST /admin/api/devices/:deviceID/register-apple`
- `POST /admin/api/devices/register-apple`
- `GET /admin/api/settings`
- `PUT /admin/api/settings`
- `GET /admin/api/upload-config`
- `GET /admin/api/apple/status`
- `GET /admin/api/apps/:appID` (ProfileModal)
- `GET /admin/exports/releases.csv`
- `GET /admin/exports/events.csv`
- `GET /admin/exports/ios_devices.csv`
- `POST /admin/api/parse-app` (SmartUpload)
- `POST /upload` (standard and smart upload; may target configured upload domain)

### Public

- `GET /api/products/:resourceID` with optional `?r=<releaseID>`
- `GET /udid/status?variant=<variantID>&nonce=<nonce>`
- `GET /udid/profile.mobileconfig?variant=<variantID>`
- `POST /events` via `navigator.sendBeacon`
- Download/install links from release actions: `/d/:releaseID`, `/ios/:releaseID/manifest.plist`, `itms-services://...`

## 4. Visual pattern inventory

Current admin UI is dominated by:

- Dark-first indigo/purple token palette with light theme as an override.
- Large page titles (`32px`, weight 700).
- `12px/16px` panel and control radius, `999px` pill badges.
- Cards used for nearly every section, including selectors and settings sections.
- Large shadows (`0 20px 60px` modal, `0 10px 40px` toast, `0 1px 3px` card).
- Gradients on login, upload accents, progress bars, and public fallback icons.
- `backdrop-filter: blur(...)` on modal overlay and toast.
- Hover transforms (`translateY`, `scale`) and long transitions.
- Large buttons (`44-50px` class) and large dropzones.

Current public UI is dominated by:

- Marketing-style hero with gradient background, glass card, `clamp(32px, 5vw, 52px)` title, large shadow.
- Variant card grid instead of a data-dense release table.
- Pill tags for platform/version metadata.
- Gradient CTA buttons with hover `translateY`.
- Large QR card as part of the hero.
- Release history as stacked buttons inside cards.

## 5. Duplicate style inventory

The following common control styles are repeated across page/component scoped
CSS instead of living in a single stylesheet:

- `.btn-*`, `.btn-sm`, `.btn-ghost-sm`, `.btn-text`, `.btn-link`
- `.input`, `.textarea`, `.label`, `.form-group`, `.checkbox-label`
- `.table`, `.status-dot`, `.status-on`, `.status-off`, `.badge`
- `.modal-*` / `.dialog-*` overlay, container, header, body, footer
- `.dropzone`, `.progress-*`, `.result-*`, `.success-banner`, `.error-banner`
- `.empty-state`, `.loading-state`
- Scoped `@media` rules for the same breakpoints

## 6. Responsive inventory

Admin has partial mobile handling:

- `AppsPage` tables use horizontal overflow but variants render as cards.
- `UploadPage`, `StatsPage`, `EventsPage`, `ExportPage`, and `UdidPage` collapse filter grids to one column.
- Sidebar is fixed `260px` and has a `collapsed` prop, but no drawer/mobile navigation is implemented.

Public front has responsive rules at `900px` and `640px`:

- Hero grid collapses to one column.
- QR is hidden on mobile.
- Variant cards stack.

Neither app has verified support for the required viewport matrix
(`1440`, `1280`, `1024`, `768`, `390`).

## 7. Theme and i18n

- Admin theme key: `fenfa_admin_theme`; supported values are `dark` (default) and `light`.
- Admin locale key: `fenfa_admin_locale`; supported values are `zh` and `en`.
- Public i18n is in `web/front/src/i18n.ts`; language is detected from `navigator.language`.
- New UI copy must be added to both admin `zh.ts`/`en.ts` and public `i18n.ts`.

## 8. Baseline build and tests

Run before any redesign edits:

```text
Admin npm ci:          PASS (0 vulnerabilities)
Front npm ci:          PASS (0 vulnerabilities)
Go tests:              PASS (go test ./...)
Admin build:           PASS
Front build:           PASS
```

Baseline bundle output:

```text
Admin:
  index.html 0.38 kB (gzip 0.26 kB)
  style.css 37.59 kB (gzip 7.11 kB)
  app.js 342.62 kB (gzip 113.92 kB)

Front:
  index.html 0.38 kB (gzip 0.26 kB)
  style.css 7.09 kB (gzip 2.10 kB)
  app.js 205.23 kB (gzip 75.12 kB)
  ua-parser chunk 19.81 kB (gzip 8.84 kB)
```

## 9. Compatibility constraints

The redesign must:

- Preserve all API URLs, HTTP methods, headers, request/response shapes.
- Preserve `X-Auth-Token`, `fenfa_admin_token`, `fenfa_admin_theme`, and `fenfa_admin_locale`.
- Preserve public UDID status, mobileconfig, nonce cookie, install flow, and event tracking.
- Not add fake dashboard/analytics/notification/team/workspace features.
- Not modify Go backend, storage, database, models, routes, or deployment files.
- Not create tags, releases, or production deployments.

## 10. Redesign direction

Target visual language:

- Neutral-first surfaces with blue accent and semantic colors only when meaningful.
- Compact engineering console with `32px` controls, `36-40px` table rows, `13px` body text.
- Table-first admin data views; nested variant table under product rows.
- Flat panels without decorative shadows; overlay shadows only for modal/dropdown/toast.
- Public release page that leads immediately from product identity to platform, version, and download.
