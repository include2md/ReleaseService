# Frontend Release & Asset Service Spec

## 1. Overview

This service is responsible for:

### Write Path (CI → Service)
- Upload frontend static assets (tar.gz)
- Extract and store in MinIO
- Record metadata in DB
- Handle versioning and rotation

### Read Path (CDN → Service → MinIO)
- Serve static assets via HTTP
- Act as a proxy between CDN and MinIO
- Provide proper caching headers

---

## 2. Architecture

[CI]
  → Release API
    → MinIO
    → Database

[Browser]
  → CDN
    → Asset API
      → MinIO

---

## 3. API Design

### POST /api/v1/releases

Upload a new frontend version.

Request (multipart/form-data):
- app_name
- version
- environment
- artifact
- commit_sha (optional)
- build_id (optional)

Response:
{
  "success": true,
  "app_name": "my-app",
  "version": "1.2.3",
  "environment": "prod",
  "status": "success"
}

---

## 4. Asset API

GET /assets/{app}/{version}/{path}

Example:
- /assets/my-app/1.2.3/index.html
- /assets/my-app/1.2.3/assets/app.js

---

## 5. MinIO Structure

/{environment}/{app}/{version}/{file}

---

## 6. DB Schema

releases:
- id
- app_name
- version
- environment
- status
- storage_prefix
- created_at

UNIQUE(app_name, version, environment)

---

## 7. Auth

Bearer Token

Authorization: Bearer <token>

---

## 8. Cache Strategy

Static assets:
Cache-Control: public, max-age=31536000, immutable

index.html:
Cache-Control: no-cache

---

## 9. Key Principles

- Immutable versioning
- CDN-first design
- Separation of write/read path
