# Public Distribution Page Second-Pass Review

## Scope

This pass refines the Public Distribution Page inside `web/front` only. The goal is to move the page from a developer release console to a consumer app install page while keeping the existing API contract, UDID flow, iOS install flow, download behavior, event tracking, QR generation, and Admin untouched.

## Before

- Release-detail oriented: package identifier, minimum OS, build, relative update time, internal platform display name, and full version table were public.
- Card-heavy: the page was assembled from a header card, platform toolbar card, release details card, action card, QR card, and history card.
- Developer-facing: it read like a public read-only release management dashboard instead of a download page.

## After

- Download oriented: the visible flow is App -> Platform -> Current Version -> Install/Download.
- Minimal public metadata: only app icon, app name, platform, current version, weak update date, primary action, release notes when present, desktop QR, and a collapsed simple version list are shown.
- Single primary action: the full-width download/install/bind button is the visual core of the page.
- Mobile-first: mobile keeps icon, name, platform, version, and install action in the first viewport and hides QR.
- Consumer-facing: no package identifier, minimum OS, build field, internal variant display name, public URL text, or release table headers are rendered.

## Information Changes

Removed from public rendering:

- Product/system introduction header
- Platform label enumeration in the header
- Variant display name duplication
- Package identifier
- Minimum OS
- Build as a standalone metadata field
- Internal platform description fields
- Relative update time
- Public URL text under QR
- Full version table with Build / Updated / Action columns

Kept in the data model and Admin:

- Package identifier
- Minimum OS
- Build
- Release ID / variant ID
- Full release metadata
- Version history and release selection behavior

## Behavior Preserved

- Platform detection and recommended variant selection
- Selected variant per platform
- Release selection by latest release or `?r=` release id
- Android download
- iOS UDID status, profile installation, `udid_bound`, and manifest install
- Event tracking through `POST /events` via `sendBeacon`
- QR generation
- Light/dark theme tokens
- i18n

## Verification

- `web/front` Vite build: PASS
- `go test ./...`: PASS
- `docker build -t fenfa:public-page-redesign-test .`: PASS
- Public screenshots: not captured in this pass; visual inspection was intentionally skipped.
