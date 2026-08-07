# Backup and Restore

## Database

Fenfa uses SQLite. The database file is at `/data/fenfa.db` inside the container.

### Backup

Stop the container or ensure no writes are in progress, then copy the file:

```bash
docker cp fenfa:/data/fenfa.db ./fenfa-backup-$(date +%Y%m%d).db
```

For a safe online backup without stopping the service:

```bash
docker exec fenfa sqlite3 /data/fenfa.db ".backup /data/fenfa-backup.db"
docker cp fenfa:/data/fenfa-backup.db ./fenfa-backup-$(date +%Y%m%d).db
```

### Restore

1. Deploy the same Fenfa image version
2. Stop the container
3. Copy the backup file back:

```bash
docker cp ./fenfa-backup-YYYYMMDD.db fenfa:/data/fenfa.db
```

4. Start the container
5. Verify with `curl http://127.0.0.1:8000/healthz`

## Object Storage (R2/S3)

Installation packages are stored in Cloudflare R2 or your configured S3-compatible storage. R2 manages its own durability. No additional backup is needed for uploaded files as long as your R2 bucket is retained.

## Full Recovery

1. Deploy the same Fenfa Docker image
2. Restore /data/fenfa.db
3. Configure the same environment variables (tokens, domain)
4. Configure the same R2/S3 storage settings
5. Start the container
6. Run health check and smoke test

## Upgrade Procedure

1. Backup the database (see above)
2. Pull the new image: `docker pull ghcr.io/rinnmoe/fenfa:0.1.1`
3. Update the image tag in docker-compose.yml
4. Recreate the container: `docker compose up -d`
5. GORM AutoMigrate runs automatically on startup
6. Verify: `curl http://127.0.0.1:8000/healthz`
7. Smoke test: create a test product, upload a release, check public API

### Rollback

If the upgrade fails:

1. Stop the container
2. Revert the image tag in docker-compose.yml to the previous version
3. Restore the database backup
4. Start the container

GORM migrations may not be reversible. Always backup before upgrading.
