# Android delta attachments

Fenfa accepts publisher-generated HDiffPatch/zlib patches. Existing full
`download_url` values and old clients keep working. Fenfa does not generate or
apply patches: clients must verify the base, patch and reconstructed APK hashes,
package name, original Android versionCode and signing certificate, and fall
back to the full APK when delta installation is unavailable.

## Publish

1. Upload the exact full target APK using `/upload` or `/smart-upload`.
2. Generate direct patches from recent exact APKs already recorded in the same
   variant/channel (initially two bases; maximum eight; no upgrade chains).
3. POST multipart to `/releases/:releaseID/delta` with upload-scoped
   `X-Auth-Token`, or `/<adminPath>/api/releases/:releaseID/delta` with admin
   authentication. Put JSON text in the `manifest` field. Each patch file field
   name must be its lowercase SHA256 (filename may be arbitrary).

```json
{
  "schema_version": 1,
  "package_name": "moe.rinn.mychu",
  "channel": "official",
  "abi": "arm64-v8a",
  "target": {
    "version": "1.0.2",
    "version_code": 2013,
    "size": 12345678,
    "sha256": "<64 lowercase hex characters>"
  },
  "patches": [{
    "algorithm": "hdiffpatch",
    "compression": "zlib",
    "from_version_code": 2012,
    "from_sha256": "<exact base APK SHA256>",
    "size": 123456,
    "sha256": "<patch SHA256>",
    "url": ""
  }]
}
```

The variant identifier and arch must equal package_name and abi. Channel and
target version/size/hash must equal the release; the stored full APK is read and
hashed before attaching. Manifest version codes are always original Android
codes. Smart upload stores that code in release.build; for historical MyCHU
records only, build 1..1999 also matches the original code build+2000. Other
packages require exact build equality. Patch bases must match a known release
hash/code in the same variant/channel. Patches must be smaller than the target.
Multipart body limit is 512 MiB, manifest limit 64 KiB. Errors use the existing
JSON envelope with `DELTA_INVALID`; a concurrently changed target returns 409.

The publisher's patch URLs are ignored and rewritten to hash-relative paths.
The response includes the normalized manifest in `data.manifest`.
Local patches live in `uploads/delta/<patchSHA256>.hdiff`; S3 patches use
`delta/<patchSHA256>.hdiff`. Content-addressed orphan files are retained for
safe concurrent uploads; automatic garbage collection is not provided.

## Client discovery and invalidation

`GET /api/v1/products/:slug/latest?variant=<variantID>` adds
`data.release.delta_manifest_url`, `size_bytes`, and `sha256`. The manifest URL
is empty when absent, invalid, or the public domain is not HTTPS. A configured
HTTPS primary domain is required for client integration. Public endpoints:

```
GET /delta/:releaseID/:targetAPKHash/:manifestHash/manifest.json
GET /delta/:releaseID/:targetAPKHash/:manifestHash/:patchHash
```

The manifest is raw JSON (no response envelope). Resolve patch.url relative to
the manifest URL. Local patch downloads support Range; S3 redirects require an
HTTPS S3 public URL. Unpublished products/variants and stale hash paths return
404. Responses request revalidation; never configure an edge to cache these
paths indefinitely. Explicit manifest discovery is authoritative; the optional
client convention `download_url + '.delta.json'` is not hosted by Fenfa.

Replacing the APK clears its manifest in the same database transaction. Target
metadata changes invalidate mismatching manifests; changing identifier/arch
also prevents public access. To disable manually, use
`DELETE /<adminPath>/api/releases/:releaseID/delta`. Old patch objects may remain
in S3, but their bytes cannot reconstruct a different target SHA256.

## Local checks and development

From the Fenfa checkout, run:

```powershell
go test ./internal/server/handlers -run 'Delta|ReplaceReleaseFileLocal' -count=1
go test ./internal/server ./internal/store
$env:FENFA_PORT = '18080'
$env:FENFA_DATA_DIR = Join-Path $env:TEMP 'fenfa-delta-dev'
go run ./cmd/server
```

Use an isolated working directory/configuration for integration data and set
credentials locally; never send them in chat. The server also writes uploads
relative to its working directory. `/healthz` is available over HTTP for local
diagnostics; an HTTPS reverse proxy is needed to test actual client manifest
discovery. No development service is started by these documentation changes.
