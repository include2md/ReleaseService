# App Asset Service Project Overview (English)

## 1. Main Purpose

`App Asset Service` is a backend service focused on frontend static asset release and delivery. Its core goals are:

- Allow CI to upload frontend artifacts (zip) and create immutable versions.
- Store versioned assets in MinIO using `environment/app/version` paths.
- Provide asset read APIs for CDN / frontend clients.
- Keep at most 15 active versions (`success`) per `app + environment`.
- Run automatic rotation when over limit: remove old data from MinIO and soft-delete records in MongoDB (`status = rotated_deleted`).

---

## 2. Backend Flow

### 2.1 Upload Flow (Write Path)

1. Client/CI calls `POST /api/v1/releases` (Bearer token required).
2. Handler validates required fields and artifact format (zip only).
3. Service extracts files and uploads each file to MinIO under `/{environment}/{app}/{version}/{file}`.
4. Service inserts a release document into MongoDB with `status = success`.
5. Service performs rotation:
   - Query releases with same `app + environment` and `status=success` (newest first).
   - Keep latest 15; mark the rest as stale.
   - Delete stale MinIO prefixes.
   - Update stale MongoDB records to `status = rotated_deleted`.

### 2.2 Asset Read Flow (Read Path)

1. CDN/browser calls `GET /assets/:app/:version/*path?env=:environment`.
2. Service builds MinIO key and streams object content.
3. Cache-Control policy:
   - `index.html` -> `no-cache`
   - Other static assets -> `public, max-age=31536000, immutable`

### 2.3 Version Listing Flow

1. Frontend calls `GET /api/v1/apps/:app/versions?environment=:environment` (no token required).
2. Service reads only `status=success` releases.
3. Service returns available versions (newest first).

---

## 3. API Reference

### 3.1 Upload Release

- Method: `POST`
- Path: `/api/v1/releases`
- Auth: `Authorization: Bearer <token>`
- Content-Type: `multipart/form-data`

Request fields:
- `app_name` (required)
- `version` (required)
- `environment` (required)
- `artifact` (required, zip)
- `commit_sha` (optional)
- `build_id` (optional)

Success response (`200`):

```json
{
  "success": true,
  "app_name": "my-app",
  "version": "1.2.3",
  "environment": "prod",
  "status": "success"
}
```

Common errors:
- `400` missing fields / non-zip / invalid artifact
- `401` invalid token
- `409` duplicate `(app_name, environment, version)`
- `500` internal error

### 3.2 Get Asset

- Method: `GET`
- Path: `/assets/:app/:version/*path`
- Query: `env` (required)
- Auth: none

Example:
- `/assets/my-app/1.2.3/index.html?env=prod`

Success response:
- `200` with binary/text content
- Includes appropriate `Content-Type` and `Cache-Control`

Common errors:
- `400` missing `env`
- `404` object not found
- `500` internal read error

### 3.3 List Available Versions

- Method: `GET`
- Path: `/api/v1/apps/:app/versions`
- Query: `environment` (required)
- Auth: none

Success response (`200`):

```json
{
  "success": true,
  "app_name": "my-app",
  "environment": "prod",
  "versions": ["2.0.0", "1.9.0", "1.8.5"]
}
```

Common errors:
- `400` missing `app` or `environment`
- `500` query failure

---

## 4. Schema

### 4.1 MongoDB

- DB: default `app_asset_service` (overridable by `MONGO_DB`)
- Collection: default `app_asset_releases` (overridable by `MONGO_COLLECTION`)

Document fields:
- `app_name` (string)
- `version` (string)
- `environment` (string)
- `status` (string) - current values: `success`, `rotated_deleted`
- `storage_prefix` (string) - e.g. `/prod/my-app/1.2.3/`
- `commit_sha` (string, optional)
- `build_id` (string, optional)
- `created_at` (datetime, UTC)

Index:
- Unique: `(app_name, environment, version)`

### 4.2 MinIO Layout

- Prefix pattern: `/{environment}/{app}/{version}/{file}`
- Example: `/prod/my-app/1.2.3/index.html`

---

## 5. Future Enhancements

- Add pagination/limit/prefix filter for version listing.
- Add release detail APIs (status, timestamps, commit SHA).
- Add robust rotation compensation (retry queue/background jobs).
- Add audit fields (`rotated_at`, `rotated_by`, `delete_reason`).
- Add admin operations (manual deprecate/rollback/recover).
- Support more artifact formats (e.g., tar.gz) and checksum validation.
- Strengthen security (token lifecycle, scoped permissions, audit logs).
- Improve observability (Prometheus metrics, tracing, structured logs).
- Add more integration/e2e tests (Mongo + MinIO + router).
