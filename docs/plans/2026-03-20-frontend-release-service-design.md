# Frontend Release Service Design

**Date:** 2026-03-20  
**Status:** Approved

## Goal
Build a Go service that accepts frontend release artifacts from CI, stores extracted assets in MinIO, saves release metadata in MongoDB, and serves assets through HTTP with correct cache behavior.

## Scope
- `POST /api/v1/releases` for authenticated artifact uploads
- `GET /assets/{app}/{version}/{path}` for asset delivery
- MinIO object storage for extracted files
- MongoDB release metadata with immutable version constraint
- Bearer token auth using static token list from environment variables

## Non-Goals
- Async background processing
- JWT or external auth service integration
- Multi-format archive support (zip only in v1)

## Architecture
Single Go service with Gin as HTTP framework. The service handles write and read paths:

- Write path: CI uploads zip artifact -> service validates and extracts -> uploads files to MinIO -> writes release record to MongoDB.
- Read path: client/CDN requests asset -> service fetches from MinIO and streams to client with proper cache headers.

### Modules
1. `HTTP Layer (Gin)`
- Routes
- Request parsing and response encoding
- Auth middleware

2. `Release Service`
- Input validation
- Duplicate release check
- Zip extraction and secure path checks
- Upload orchestration and compensation logic

3. `Storage Repository (MinIO SDK)`
- Object put/get/delete
- Prefix cleanup helpers

4. `Release Repository (MongoDB)`
- Release insert/query
- Index bootstrap

5. `Asset Service`
- MinIO key resolution
- Streaming response
- Cache-Control policy

## API Design

### POST /api/v1/releases
Content-Type: `multipart/form-data`

Required fields:
- `app_name`
- `version`
- `environment`
- `artifact` (zip)

Optional fields:
- `commit_sha`
- `build_id`

Success response:
```json
{
  "success": true,
  "app_name": "my-app",
  "version": "1.2.3",
  "environment": "prod",
  "status": "success"
}
```

Error codes:
- `400` invalid parameters or invalid zip
- `401` invalid bearer token
- `409` release already exists (`app_name + environment + version`)
- `500` internal error

### GET /assets/:app/:version/*path
Query parameter:
- `env` (required in v1)

Behavior:
- Stream object from MinIO
- `index.html` -> `Cache-Control: no-cache`
- Others -> `Cache-Control: public, max-age=31536000, immutable`

Error codes:
- `404` object not found
- `500/502` dependency or service errors

## Data Model (MongoDB)
Collection: `releases`

Fields:
- `app_name` string
- `version` string
- `environment` string
- `status` string (`success`)
- `storage_prefix` string
- `commit_sha` string (optional)
- `build_id` string (optional)
- `created_at` datetime

Indexes:
- Unique: `{app_name: 1, environment: 1, version: 1}`
- Optional: `{created_at: -1}`

## MinIO Key Structure
`/{environment}/{app}/{version}/{file}`

Example:
- `/prod/my-app/1.2.3/index.html`
- `/prod/my-app/1.2.3/assets/app.js`

## Data Flow and Error Handling

### Upload flow (`POST /api/v1/releases`)
1. Authenticate bearer token.
2. Validate form fields and artifact format.
3. Verify zip content (including zip-slip prevention).
4. Check duplicate release in MongoDB.
5. Extract archive to temporary directory.
6. Upload extracted files to MinIO.
7. Insert release metadata in MongoDB.
8. Return success response.

### Compensation rules
- If upload fails midway: delete uploaded objects for this release prefix.
- If MongoDB insert fails after upload success: delete entire release prefix from MinIO.
- If compensation fails: return error and log cleanup failure with request ID.

### Read flow (`GET /assets/...`)
1. Resolve key from `env`, `app`, `version`, `path`.
2. Fetch object stream from MinIO.
3. Apply cache header policy by path.
4. Stream bytes to caller.

## Security
- Static bearer token whitelist from environment.
- Reject path traversal in archive entries.
- Validate filename/path normalization before upload.

## Testing Strategy

### Unit tests
- Form validation
- Token middleware behavior
- Zip validation and zip-slip guard
- Cache header mapping
- Duplicate release detection mapping to `409`

### Integration tests
- Successful upload (Mongo + MinIO)
- MinIO upload failure triggers cleanup
- Mongo insert failure triggers cleanup
- Asset fetch success and not found handling

### Contract tests
- API status codes and response schema stability

## Operational Notes
- Use structured logs with request ID.
- Configure bucket name, endpoint, credentials, DB URI, and token list via env vars.
- Keep v1 synchronous for simplicity and correctness.
