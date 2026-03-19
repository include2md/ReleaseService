# Local Runbook

## Start all services

```bash
docker compose up --build
```

Services:
- Release Service: `http://localhost:8080`
- MongoDB: `mongodb://localhost:27017`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001` (`minioadmin/minioadmin`)

## Required runtime env vars
- `MONGO_URI`
- `MONGO_DB`
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `MINIO_BUCKET`
- `MINIO_USE_SSL`
- `RELEASE_TOKENS`

## Quick upload test

```bash
zip -r sample.zip index.html assets

curl -X POST 'http://localhost:8080/api/v1/releases' \
  -H 'Authorization: Bearer local-dev-token' \
  -F 'app_name=my-app' \
  -F 'version=1.2.3' \
  -F 'environment=prod' \
  -F 'artifact=@sample.zip'
```

## Quick asset read test

```bash
curl -i 'http://localhost:8080/assets/my-app/1.2.3/index.html?env=prod'
```
