package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"app-assets-service/internal/service"
)

type fakeReleaseService struct {
	err      error
	versions []string
}

func (f fakeReleaseService) CreateRelease(ctx context.Context, in service.ReleaseInput) error {
	return f.err
}

func (f fakeReleaseService) ListAvailableVersions(ctx context.Context, appName, environment string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.versions, nil
}

func TestReleaseHandler_ReturnsBadRequestForNonZipArtifact(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{})
	r := setupReleaseTestRouter(h)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("app_name", "my-app")
	_ = w.WriteField("version", "1.2.3")
	_ = w.WriteField("environment", "prod")
	part, _ := w.CreateFormFile("artifact", "artifact.txt")
	_, _ = part.Write([]byte("not-zip"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestReleaseHandler_ReturnsConflictWhenServiceSaysDuplicate(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{err: ErrReleaseAlreadyExists})
	r := setupReleaseTestRouter(h)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("app_name", "my-app")
	_ = w.WriteField("version", "1.2.3")
	_ = w.WriteField("environment", "prod")
	part, _ := w.CreateFormFile("artifact", "artifact.zip")
	_, _ = part.Write(validZipBytes(t))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.Code)
	}
}

func TestReleaseHandler_ListVersions_ReturnsBadRequestWhenEnvironmentMissing(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{})
	r := setupReleaseTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/my-app/versions", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestReleaseHandler_ListVersions_ReturnsVersions(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{versions: []string{"2.0.0", "1.9.0"}})
	r := setupReleaseTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/my-app/versions?environment=prod", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var body struct {
		Success     bool     `json:"success"`
		AppName     string   `json:"app_name"`
		Environment string   `json:"environment"`
		Versions    []string `json:"versions"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success || body.AppName != "my-app" || body.Environment != "prod" {
		t.Fatalf("unexpected response metadata: %#v", body)
	}
	if len(body.Versions) != 2 || body.Versions[0] != "2.0.0" || body.Versions[1] != "1.9.0" {
		t.Fatalf("unexpected versions: %#v", body.Versions)
	}
}

func validZipBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("index.html")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, _ = f.Write([]byte("ok"))
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
