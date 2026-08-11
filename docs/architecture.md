# Architecture

## Overview

Fenfa is a self-hosted app distribution platform. It follows a simple monolithic architecture: a single Go binary serves the API, admin panel, and public download pages.

```
                    ┌─────────────────────────┐
                    │     Nginx (reverse       │
                    │     proxy + TLS)         │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │     Fenfa Server         │
                    │     (Go + Gin)           │
                    │                          │
                    │  ┌─────────────────────┐ │
                    │  │  Embedded Frontend   │ │
                    │  │  (go:embed)          │ │
                    │  │  ├─ front/ (Vue 3)   │ │
                    │  │  └─ admin/ (Vue 3)   │ │
                    │  └─────────────────────┘ │
                    │                          │
                    │  ┌─────────────────────┐ │
                    │  │  SQLite (GORM)       │ │
                    │  └─────────────────────┘ │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │  File Storage            │
                    │  (local or S3/R2)        │
                    └─────────────────────────┘
```

## Data Model

### Product → Variant → Release

```
Product (e.g. "MyApp")
├── Variant: iOS (com.example.myapp)
│   ├── Release: v1.0.0 (build 100)
│   ├── Release: v1.1.0 (build 110)
│   └── Release: v1.2.0 (build 120)
├── Variant: Android (com.example.myapp)
│   ├── Release: v1.0.0 (build 100)
│   └── Release: v1.1.0 (build 110)
└── Variant: macOS (dmg, arm64)
    └── Release: v1.0.0
```

- **Product**: A logical app with a name, slug, icon, and description
- **Variant**: A platform-specific build target (iOS, Android, macOS, etc.) with its own identifier, architecture, and installer type
- **Release**: A specific uploaded build with version, changelog, and binary file

### Core Tables

| Table | Purpose |
|-------|---------|
| `products` | Multi-platform product pages |
| `variants` | Platform-specific build targets |
| `releases` | Uploaded builds with metadata |
| `apps` | Legacy single-platform apps (backward compatible) |
| `ios_devices` | UDID bindings for iOS devices |
| `provisioning_profiles` | Extracted iOS signing profiles |
| `system_settings` | Global configuration (domains, storage, Apple API, SMTP email) |
| `feedbacks` | Anonymous feedback and product context snapshots |

See `internal/store/models.go` for full schema.

## Request Flow

```
Request → Gin Router → Auth Middleware → Handler → GORM → SQLite
```

### Public routes (no auth)
- `GET /products/:slug` — download page
- `GET /d/:releaseID` — file download
- `GET /ios/:releaseID/manifest.plist` — iOS manifest
- `GET/POST /udid/*` — device binding
- `POST /api/v1/feedback` — anonymous product feedback submission

### Protected routes (token auth)
- `POST /upload` — upload builds (upload token)
- `* /<admin_path>/api/*` — admin API, including feedback inbox and email settings (admin token)
- `* /<admin_path>/exports/*` — CSV exports (admin token)

## Feedback Email Notifications

Accepted feedback is persisted in `feedbacks` first. If SMTP is configured and `email_enabled` is true, `internal/email` asynchronously sends a plain-text notification to `smtp_to`. The notification queue is in-process (capacity 64, up to three attempts), so pending messages are not persisted across process restarts. SMTP settings live in `system_settings`; passwords are never returned by the admin API.

## iOS Distribution Flow

```
User visits page → Binds UDID → Admin registers device
→ Re-signs IPA with new profile → Uploads to Fenfa → User installs
```

See [ios-signing.md](ios-signing.md) for the complete re-signing guide.

## File Storage

```
uploads/
└── {product_id}/
    └── {variant_id}/
        └── {release_id}/
            ├── app.ipa (or .apk, .dmg, etc.)
            └── meta.json
```

Local storage is the default. S3-compatible storage can be configured in admin settings.

## Directory Structure

```
cmd/server/              Entry point (main.go)
internal/
├── config/              Configuration loading
├── email/               SMTP sending and feedback notifier
├── server/
│   ├── router.go        Route registration
│   ├── handlers/        HTTP handlers
│   └── middleware/       Auth middleware
├── store/               GORM models + migrations
├── apple/               Apple Developer API client
├── s3/                  S3 storage client
└── web/                 Embedded frontend (go:embed)
web/
├── front/               Public download page (Vue 3 + Vite)
└── admin/               Admin panel (Vue 3 + Vite)
scripts/                 Docker build/run/upload helpers
deploy/                  Nginx config
docs/                    Documentation
```
