package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeAssetService struct {
	contentType string
	body        []byte
	err         error
}

func (f fakeAssetService) GetAsset(ctx context.Context, in AssetRequest) (AssetResponse, error) {
	if f.err != nil {
		return AssetResponse{}, f.err
	}
	return AssetResponse{
		ContentType: f.contentType,
		Body:        io.NopCloser(bytes.NewReader(f.body)),
	}, nil
}

func TestAssetHandler_SetsNoCacheForIndexHTML(t *testing.T) {
	h := NewAssetHandler(fakeAssetService{contentType: "text/html", body: []byte("ok")})
	r := setupAssetTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/assets/my-app/1.2.3/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected no-cache, got %s", got)
	}
}

func TestAssetHandler_SetsImmutableCacheForStaticAsset(t *testing.T) {
	h := NewAssetHandler(fakeAssetService{contentType: "application/javascript", body: []byte("ok")})
	r := setupAssetTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/assets/my-app/1.2.3/assets/app.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected cache-control: %s", got)
	}
}
