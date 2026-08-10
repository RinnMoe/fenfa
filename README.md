<p align="center">
  <strong>RinnMoe Fenfa</strong> &nbsp;·&nbsp; Self-hosted app distribution platform
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/openprx/fenfa"><img src="https://img.shields.io/badge/fork-openprx%2Ffenfa-8b5cf6" alt="Fork of openprx/fenfa"></a>
</p>

<p align="center">
  <a href="README.zh-CN.md">中文文档</a>
</p>

---

RinnMoe Fenfa is a maintained fork of [openprx/fenfa](https://github.com/openprx/fenfa) for self-hosted software version management, release, and distribution. It keeps the upstream Product -> Variant -> Release model and adds release force-update flags, product announcements, and client-facing public APIs.

## Features

- **Multi-platform** — iOS (IPA), Android (APK), macOS, Windows, Linux
- **Smart upload** — auto-detect app metadata (bundle ID, version, icon) from packages
- **iOS UDID binding** — device registration flow for ad-hoc distribution
- **Product pages** — public download pages with QR codes and platform detection
- **Admin panel** — manage products, variants, releases, announcements, devices, and settings
- **Release force update** — mark a release as mandatory through the admin UI or API
- **Product announcements** — create, schedule, enable, and disable product-scoped notices
- **Public client APIs** — latest release and announcements endpoints for software clients
- **S3/R2 storage** — Cloudflare R2, AWS S3, and other S3-compatible object storage
- **Feedback email reminders** — configure SMTP in the admin panel and receive new feedback alerts
- **Single binary** — Go binary with embedded frontend
- **SQLite** — zero external dependencies, data in a single file
- **i18n** — Chinese and English UI

## Quick Start

### Docker

```bash
docker pull ghcr.io/rinnmoe/fenfa:0.1.0
docker run -d --name fenfa -p 127.0.0.1:8000:8000 \
  -e FENFA_ADMIN_TOKEN=your-admin-token \
  -e FENFA_UPLOAD_TOKEN=your-upload-token \
  -e FENFA_PRIMARY_DOMAIN=https://dist.example.com \
  -v ./data:/data \
  -v ./uploads:/app/uploads \
  ghcr.io/rinnmoe/fenfa:0.1.0
```

Visit `http://localhost:8000/admin`.

Production deployment through 1Panel: see [docs/DEPLOY_1PANEL.md](docs/DEPLOY_1PANEL.md).

### Build from source

Requirements: Go 1.25+, Node.js 20+

```bash
make build   # builds frontend + backend
make run     # starts the server
```

Or manually:

```bash
cd web/front && npm ci && npm run build && cd ../..
cd web/admin && npm ci && npm run build && cd ../..
go build -o fenfa ./cmd/server
./fenfa
```

## Configuration

See [docs/config.example.json](docs/config.example.json) for all options.

| Environment Variable | Description |
|----------------------|-------------|
| `FENFA_PORT` | HTTP port |
| `FENFA_DATA_DIR` | Database directory |
| `FENFA_PRIMARY_DOMAIN` | Public domain URL |
| `FENFA_ADMIN_PATH` | Admin panel URL suffix (default `admin`) |
| `FENFA_TRUSTED_PROXIES` | Comma-separated proxy IPs/CIDRs allowed to provide client IP headers; empty disables forwarded IP headers |
| `FENFA_ADMIN_TOKEN` | Admin token |
| `FENFA_UPLOAD_TOKEN` | Upload token |

When running behind Nginx or another reverse proxy, set `FENFA_TRUSTED_PROXIES` to that proxy's actual IP or network. Do not use `0.0.0.0/0` or `::/0`.

Configure feedback email reminders in Admin > Settings > Email Notifications. STARTTLS, SSL/TLS, and unencrypted SMTP are supported. Passwords are never shown, and the test email button can validate the current form values before saving.

### Storage

Files are stored locally by default. S3-compatible storage (Cloudflare R2, AWS S3, MinIO) can be configured in Admin > Settings > Storage. For Cloudflare R2, use the account-level S3 endpoint without the bucket name appended.

## Public Client API

These endpoints are read-only and do not require an admin or upload token.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/products/:slug/latest?variant=<variantID>` | Latest release for a product variant |
| GET | `/api/v1/products/:slug/announcements` | Currently active announcements |

## Architecture

```
Request → Gin Router → Auth Middleware → Handler → GORM → SQLite
```

```
cmd/server/          Entry point
internal/server/     HTTP handlers & routing
internal/store/      Database models (GORM + SQLite)
internal/web/        Embedded frontend (go:embed)
web/front/           Public download page (Vue 3 + Vite)
web/admin/           Admin panel (Vue 3 + Vite)
```

## Upstream & Maintenance

- Initial upstream baseline: `e0fbb666d779274329091677e4d6218b15c6d81c`
- This repository is independently maintained and is not intended for upstream PRs.
- The `upstream` remote points to [openprx/fenfa](https://github.com/openprx/fenfa) and is used only to selectively absorb security fixes or bugfixes.

## Documentation

- [Customization](docs/customization.md)
- [1Panel Deployment](docs/DEPLOY_1PANEL.md)
- [Backup & Upgrade](docs/BACKUP.md)
- [Known Issues](docs/KNOWN_ISSUES.md)

## License

[MIT](LICENSE). Upstream copyright remains with the original Fenfa authors.
