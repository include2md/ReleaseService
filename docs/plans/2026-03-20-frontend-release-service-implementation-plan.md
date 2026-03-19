# Frontend Release Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-usable v1 service in Go that supports authenticated release upload, MinIO storage, MongoDB metadata persistence, and asset proxy with cache policy.

**Architecture:** Implement a layered Gin service (`handler -> service -> repository`) with explicit interfaces for MinIO and MongoDB integration. Use synchronous upload flow with compensation cleanup to preserve immutable release behavior. Enforce TDD for each behavior slice before writing production code.

**Tech Stack:** Go, Gin, MongoDB driver, MinIO Go SDK, Testify, httptest

---

### Task 1: Bootstrap Project Skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `internal/config/config.go`
- Create: `internal/server/router.go`
- Create: `internal/server/middleware/auth.go`
- Create: `internal/server/handlers/release_handler.go`
- Create: `internal/server/handlers/asset_handler.go`

**Step 1: Write the failing test**

```go
func TestRouter_RegistersRequiredRoutes(t *testing.T) {
    r := NewRouter(testDeps())

    routes := collectRoutes(r)
    require.Contains(t, routes, "POST /api/v1/releases")
    require.Contains(t, routes, "GET /assets/:app/:version/*path")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestRouter_RegistersRequiredRoutes -v`
Expected: FAIL because router builder does not exist.

**Step 3: Write minimal implementation**

```go
func NewRouter(deps Deps) *gin.Engine {
    r := gin.New()
    r.POST("/api/v1/releases", deps.Auth, deps.ReleaseHandler.Upload)
    r.GET("/assets/:app/:version/*path", deps.AssetHandler.Get)
    return r
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestRouter_RegistersRequiredRoutes -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add go.mod cmd/server/main.go internal/config/config.go internal/server/router.go internal/server/middleware/auth.go internal/server/handlers/release_handler.go internal/server/handlers/asset_handler.go
git commit -m "chore: bootstrap release service skeleton"
```

### Task 2: Add Bearer Token Middleware

**Files:**
- Modify: `internal/server/middleware/auth.go`
- Create: `internal/server/middleware/auth_test.go`

**Step 1: Write the failing test**

```go
func TestAuthMiddleware_RejectsInvalidToken(t *testing.T) {
    mw := NewAuthMiddleware([]string{"token-a"})
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set("Authorization", "Bearer bad-token")
    c.Request = req

    mw(c)

    require.Equal(t, http.StatusUnauthorized, w.Code)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server/middleware -run TestAuthMiddleware_RejectsInvalidToken -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
func NewAuthMiddleware(tokens []string) gin.HandlerFunc {
    allowed := make(map[string]struct{}, len(tokens))
    for _, t := range tokens {
        allowed[t] = struct{}{}
    }
    return func(c *gin.Context) {
        token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        if _, ok := allowed[token]; !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }
        c.Next()
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server/middleware -v`
Expected: PASS for valid and invalid token cases.

**Step 5: Commit**

```bash
git add internal/server/middleware/auth.go internal/server/middleware/auth_test.go
git commit -m "feat: add bearer token auth middleware"
```

### Task 3: Implement Release Duplicate Detection (`409`)

**Files:**
- Create: `internal/domain/release.go`
- Create: `internal/repository/release_repository.go`
- Create: `internal/service/release_service.go`
- Create: `internal/service/release_service_test.go`

**Step 1: Write the failing test**

```go
func TestReleaseService_CreateRelease_ReturnsConflictWhenVersionExists(t *testing.T) {
    repo := &fakeReleaseRepo{exists: true}
    svc := NewReleaseService(repo, nil)

    err := svc.CreateRelease(context.Background(), ReleaseInput{AppName: "my-app", Environment: "prod", Version: "1.2.3"})

    require.ErrorIs(t, err, ErrReleaseAlreadyExists)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestReleaseService_CreateRelease_ReturnsConflictWhenVersionExists -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
if exists {
    return ErrReleaseAlreadyExists
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestReleaseService_CreateRelease_ReturnsConflictWhenVersionExists -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/domain/release.go internal/repository/release_repository.go internal/service/release_service.go internal/service/release_service_test.go
git commit -m "feat: enforce immutable release version with conflict error"
```

### Task 4: Validate Multipart Fields and Zip Artifact

**Files:**
- Modify: `internal/server/handlers/release_handler.go`
- Create: `internal/server/handlers/release_handler_test.go`
- Create: `internal/archive/zip_validator.go`
- Create: `internal/archive/zip_validator_test.go`

**Step 1: Write the failing test**

```go
func TestReleaseHandler_ReturnsBadRequestForNonZipArtifact(t *testing.T) {
    req := buildMultipartRequestWithFile("artifact", "artifact.txt", []byte("not-zip"))
    w := httptest.NewRecorder()

    router := buildTestRouterWithReleaseHandler()
    router.ServeHTTP(w, req)

    require.Equal(t, http.StatusBadRequest, w.Code)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server/handlers -run TestReleaseHandler_ReturnsBadRequestForNonZipArtifact -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
if !archive.IsZip(fileHeader) {
    c.JSON(http.StatusBadRequest, gin.H{"error": "artifact must be zip"})
    return
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server/handlers ./internal/archive -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/server/handlers/release_handler.go internal/server/handlers/release_handler_test.go internal/archive/zip_validator.go internal/archive/zip_validator_test.go
git commit -m "feat: validate release multipart fields and zip artifact"
```

### Task 5: Implement Zip Extraction With Zip-Slip Protection

**Files:**
- Create: `internal/archive/extractor.go`
- Create: `internal/archive/extractor_test.go`

**Step 1: Write the failing test**

```go
func TestExtractor_RejectsZipSlipPath(t *testing.T) {
    payload := zipWithEntries("../evil.txt")

    _, err := ExtractToTemp(payload)

    require.ErrorContains(t, err, "zip slip")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/archive -run TestExtractor_RejectsZipSlipPath -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
target := filepath.Join(dest, name)
if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
    return "", ErrZipSlip
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/archive -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/archive/extractor.go internal/archive/extractor_test.go
git commit -m "feat: add secure zip extraction"
```

### Task 6: Add MinIO Upload + Compensation Cleanup

**Files:**
- Create: `internal/repository/object_storage.go`
- Modify: `internal/service/release_service.go`
- Modify: `internal/service/release_service_test.go`

**Step 1: Write the failing test**

```go
func TestReleaseService_CleansUpUploadedObjectsWhenMongoInsertFails(t *testing.T) {
    storage := &fakeStorage{uploaded: []string{"a.js", "index.html"}}
    repo := &fakeReleaseRepo{insertErr: errors.New("mongo down")}
    svc := NewReleaseService(repo, storage)

    err := svc.CreateRelease(context.Background(), validInput())

    require.Error(t, err)
    require.True(t, storage.cleanupCalled)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestReleaseService_CleansUpUploadedObjectsWhenMongoInsertFails -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
if err := repo.InsertRelease(ctx, doc); err != nil {
    _ = storage.DeletePrefix(ctx, prefix)
    return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/repository/object_storage.go internal/service/release_service.go internal/service/release_service_test.go
git commit -m "feat: add minio upload workflow with compensation cleanup"
```

### Task 7: Implement Asset Proxy Cache Policy

**Files:**
- Modify: `internal/server/handlers/asset_handler.go`
- Create: `internal/server/handlers/asset_handler_test.go`
- Create: `internal/service/asset_service.go`

**Step 1: Write the failing test**

```go
func TestAssetHandler_SetsNoCacheForIndexHTML(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/assets/my-app/1.2.3/index.html?env=prod", nil)
    w := httptest.NewRecorder()

    router := buildTestRouterWithAssetHandler(returning("text/html", []byte("ok")))
    router.ServeHTTP(w, req)

    require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server/handlers -run TestAssetHandler_SetsNoCacheForIndexHTML -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
if strings.HasSuffix(path, "/index.html") || path == "index.html" {
    c.Header("Cache-Control", "no-cache")
} else {
    c.Header("Cache-Control", "public, max-age=31536000, immutable")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server/handlers -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/server/handlers/asset_handler.go internal/server/handlers/asset_handler_test.go internal/service/asset_service.go
git commit -m "feat: implement asset proxy cache strategy"
```

### Task 8: Add Mongo/MinIO Production Adapters and Wiring

**Files:**
- Create: `internal/repository/mongo_release_repository.go`
- Create: `internal/repository/minio_object_storage.go`
- Modify: `cmd/server/main.go`
- Modify: `internal/config/config.go`

**Step 1: Write the failing test**

```go
func TestConfig_LoadRequiredEnv(t *testing.T) {
    t.Setenv("MONGO_URI", "")

    _, err := LoadConfig()

    require.ErrorContains(t, err, "MONGO_URI")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestConfig_LoadRequiredEnv -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
if cfg.MongoURI == "" {
    return Config{}, errors.New("MONGO_URI is required")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -v && go test ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/repository/mongo_release_repository.go internal/repository/minio_object_storage.go cmd/server/main.go internal/config/config.go
git commit -m "feat: wire mongo and minio adapters"
```

### Task 9: Add End-to-End Integration Tests

**Files:**
- Create: `test/integration/release_flow_test.go`
- Create: `test/integration/asset_flow_test.go`
- Create: `test/integration/testenv.go`

**Step 1: Write the failing test**

```go
func TestReleaseUploadAndAssetReadFlow(t *testing.T) {
    env := newTestEnv(t)
    app := newServer(env)

    resp := uploadRelease(t, app, "my-app", "1.2.3", "prod", zipFixture())
    require.Equal(t, http.StatusOK, resp.StatusCode)

    asset := getAsset(t, app, "/assets/my-app/1.2.3/index.html?env=prod")
    require.Equal(t, http.StatusOK, asset.StatusCode)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./test/integration -run TestReleaseUploadAndAssetReadFlow -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```go
// provide test env wiring and any missing handlers/repository methods required by flow
```

**Step 4: Run test to verify it passes**

Run: `go test ./test/integration -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add test/integration/release_flow_test.go test/integration/asset_flow_test.go test/integration/testenv.go
git commit -m "test: add integration coverage for release and asset flow"
```

### Task 10: Final Verification and Runbook

**Files:**
- Create: `doc/runbook.md`
- Modify: `doc/frontend-release-service-spec.md`

**Step 1: Write the failing test**

```go
func TestReadmeSnippet_ContainsRequiredEnvVars(t *testing.T) {
    b, _ := os.ReadFile("doc/runbook.md")
    s := string(b)
    require.Contains(t, s, "MONGO_URI")
    require.Contains(t, s, "MINIO_ENDPOINT")
    require.Contains(t, s, "RELEASE_TOKENS")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestReadmeSnippet_ContainsRequiredEnvVars -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

```md
# Runbook

## Required env
- MONGO_URI
- MINIO_ENDPOINT
- MINIO_ACCESS_KEY
- MINIO_SECRET_KEY
- MINIO_BUCKET
- RELEASE_TOKENS
```

**Step 4: Run test to verify it passes**

Run: `go test ./... -v`
Expected: PASS across project tests.

**Step 5: Commit**

```bash
git add doc/runbook.md doc/frontend-release-service-spec.md
git commit -m "docs: add runbook and align spec with implementation"
```
