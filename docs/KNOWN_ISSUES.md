# Known Issues

## Inherited Frontend Dependencies (Resolved)

The upstream Fenfa frontend dependencies (web/front, web/admin) previously had 3 high severity npm audit findings:

- vite 8.0.0 (GHSA-fx2h-pf6j-xcff and related): path traversal in dev server
- postcss <=8.5.22 (GHSA-r28c-9q8g-f849 and related): source map file disclosure
- picomatch 4.0.0-4.0.3 (GHSA-c2c7-rcm5-vvqj): ReDoS via extglob quantifiers

All three were build-time only (devDependencies of vite). No runtime browser dependencies were affected.

**Resolution**: Applied `npm audit fix` (non-breaking minor/patch updates) in both web/front and web/admin. Vite upgraded 8.0.0 to 8.2.1, postcss and picomatch updated accordingly. Both frontend builds pass. `npm audit` now reports 0 vulnerabilities.

## S3 Endpoint Configuration

When configuring Cloudflare R2 in Fenfa admin settings, the S3 endpoint must be the account-level API URL without the bucket name appended:

- Correct: `https://ACCOUNT_ID.r2.cloudflarestorage.com`
- Incorrect: `https://ACCOUNT_ID.r2.cloudflarestorage.com/bucket-name`

Fenfa uses path-style S3 addressing. The bucket name is provided separately in the Bucket field. Including the bucket in the endpoint causes objects to be stored at incorrect keys, resulting in 404 on download.

## SQLite Timezone Handling

SQLite stores datetime values as TEXT and performs string comparison for time-range queries. Fenfa stores and queries announcement start_at/end_at in UTC to ensure consistent comparison. The public announcements API uses `time.Now().UTC()` for the current time filter. Admin-created announcements are normalized to UTC before storage.
