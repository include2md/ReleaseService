package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func setupAssetTestRouter(h *AssetHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/assets/:app/:version/*path", h.Get)
	return r
}

func setupReleaseTestRouter(h *ReleaseHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/releases", h.Upload)
	r.GET("/api/v1/apps/:app/versions", h.ListVersions)
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}
