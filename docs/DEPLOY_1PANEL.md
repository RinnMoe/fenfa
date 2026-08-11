# 1Panel Deployment Guide

## Prerequisites

- 1Panel installed on your server
- Docker available through 1Panel
- A domain with DNS pointing to your server

## 1. Generate Production Tokens

```bash
openssl rand -hex 32   # FENFA_ADMIN_TOKEN
openssl rand -hex 32   # FENFA_UPLOAD_TOKEN
```

## 2. Create Compose Project

Copy deploy/1panel/docker-compose.yml.example and deploy/1panel/.env.example to your server:

```bash
mkdir -p /opt/fenfa
cp deploy/1panel/docker-compose.yml.example /opt/fenfa/docker-compose.yml
cp deploy/1panel/.env.example /opt/fenfa/.env
```

Edit /opt/fenfa/.env with your generated tokens and domain. If Fenfa will run behind a reverse proxy, also set `FENFA_TRUSTED_PROXIES` to the proxy's actual IP/CIDR.

In 1Panel: create a new Compose project pointing to /opt/fenfa/docker-compose.yml.

## 3. Start the Container

The container listens on 127.0.0.1:8000. Traffic goes through 1Panel reverse proxy.

## 4. Configure Reverse Proxy in 1Panel

1. In 1Panel, go to Websites > Create Website > Reverse Proxy
2. Domain: release.yourdomain.com
3. Proxy target: http://127.0.0.1:8000
4. Enable HTTPS with Let's Encrypt

If you use the feedback rate limiter or other client-IP-derived behavior, set `FENFA_TRUSTED_PROXIES` in `/opt/fenfa/.env` to the 1Panel reverse proxy IP/CIDR, then recreate the container. Do not use `0.0.0.0/0` or `::/0`.

## 5. Configure R2/S3 Storage

After the container is running:

1. Visit https://release.yourdomain.com/admin
2. Login with your FENFA_ADMIN_TOKEN
3. Go to Settings > Storage
4. Set Storage Mode to S3
5. Fill in R2 endpoint (https://ACCOUNT_ID.r2.cloudflarestorage.com), bucket, access key, secret key, and public URL

The S3 endpoint should NOT include the bucket name.

## 6. Health Check

```bash
curl http://127.0.0.1:8000/healthz
```

Expected response with ok: true.

## 7. Verify

- Admin panel accessible at /admin
- Public product page at /products/:slug
- Public API at /api/v1/products/:slug/latest
