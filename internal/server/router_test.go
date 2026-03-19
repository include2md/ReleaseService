package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type noopReleaseHandler struct{}

type noopAssetHandler struct{}

func (h noopReleaseHandler) Upload(c *gin.Context) { c.Status(http.StatusOK) }
func (h noopAssetHandler) Get(c *gin.Context)      { c.Status(http.StatusOK) }

func TestRouter_RegistersRequiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Dependencies{
		AuthMiddleware: func(c *gin.Context) { c.Next() },
		ReleaseHandler: noopReleaseHandler{},
		AssetHandler:   noopAssetHandler{},
	})

	routes := r.Routes()
	want := map[string]bool{
		http.MethodPost + " /api/v1/releases":           false,
		http.MethodGet + " /assets/:app/:version/*path": false,
	}

	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}

	for key, found := range want {
		if !found {
			t.Fatalf("missing route %s", key)
		}
	}
}

func TestRouter_CORSPreflight_AllowsAnyOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(Dependencies{
		AuthMiddleware: func(c *gin.Context) { c.Next() },
		ReleaseHandler: noopReleaseHandler{},
		AssetHandler:   noopAssetHandler{},
	})

	req, _ := http.NewRequest(http.MethodOptions, "/api/v1/releases", nil)
	req.Header.Set("Origin", "http://localhost:5500")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected allow origin *, got %s", got)
	}
}
