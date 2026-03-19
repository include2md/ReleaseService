package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"releaseservice/internal/repository"
)

type AssetRequest struct {
	Environment string
	App         string
	Version     string
	Path        string
}

type AssetResponse struct {
	Body        io.ReadCloser
	ContentType string
}

type AssetService interface {
	GetAsset(ctx context.Context, in AssetRequest) (AssetResponse, error)
}

type AssetHandler struct {
	svc AssetService
}

func NewAssetHandler(svc AssetService) *AssetHandler {
	return &AssetHandler{svc: svc}
}

func (h *AssetHandler) Get(c *gin.Context) {
	env := c.Query("env")
	if env == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "env is required"})
		return
	}

	assetPath := strings.TrimPrefix(c.Param("path"), "/")
	if assetPath == "" {
		assetPath = "index.html"
	}

	res, err := h.svc.GetAsset(c.Request.Context(), AssetRequest{
		Environment: env,
		App:         c.Param("app"),
		Version:     c.Param("version"),
		Path:        assetPath,
	})
	if err != nil {
		if errors.Is(err, repository.ErrObjectNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch asset"})
		return
	}
	defer res.Body.Close()

	if assetPath == "index.html" || strings.HasSuffix(assetPath, "/index.html") {
		c.Header("Cache-Control", "no-cache")
	} else {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	}
	if res.ContentType != "" {
		c.Header("Content-Type", res.ContentType)
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, res.Body)
}
