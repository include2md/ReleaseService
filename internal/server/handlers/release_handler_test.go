package handlers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

func (f fakeReleaseService) ListAvailableVersions(ctx context.Context, appName string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.versions, nil
}

func TestReleaseHandler_ReturnsBadRequestForNonTarGzArtifact(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{})
	r := setupReleaseTestRouter(h)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("app_name", "my-app")
	_ = w.WriteField("version", "1.2.3")
	part, _ := w.CreateFormFile("artifact", "artifact.txt")
	_, _ = part.Write([]byte("not-tar-gz"))
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
	part, _ := w.CreateFormFile("artifact", "artifact.tar.gz")
	_, _ = part.Write(validTarGzBytes(t))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.Code)
	}
}

func TestReleaseHandler_ListVersions_ReturnsInternalErrorOnServiceFailure(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{err: context.DeadlineExceeded})
	r := setupReleaseTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/my-app/versions", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestReleaseHandler_ListVersions_ReturnsVersions(t *testing.T) {
	h := NewReleaseHandler(fakeReleaseService{versions: []string{"2.0.0", "1.9.0"}})
	r := setupReleaseTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/my-app/versions", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var body struct {
		Success  bool     `json:"success"`
		AppName  string   `json:"app_name"`
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success || body.AppName != "my-app" {
		t.Fatalf("unexpected response metadata: %#v", body)
	}
	if len(body.Versions) != 2 || body.Versions[0] != "2.0.0" || body.Versions[1] != "1.9.0" {
		t.Fatalf("unexpected versions: %#v", body.Versions)
	}
}

func validTarGzBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	content := []byte("ok")
	if err := tw.WriteHeader(&tar.Header{
		Name: "index.html",
		Mode: 0o644,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}
