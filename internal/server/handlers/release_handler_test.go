package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"app-assets-service/internal/service"
)

type fakeReleaseService struct {
	err error
}

func (f fakeReleaseService) CreateRelease(ctx context.Context, in service.ReleaseInput) error {
	return f.err
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
