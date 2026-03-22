package server

import (
	"app-assets-service/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type ReleaseHandler interface {
	Upload(c *gin.Context)
	ListVersions(c *gin.Context)
}

type AssetHandler interface {
	Get(c *gin.Context)
}

type Dependencies struct {
	AuthMiddleware gin.HandlerFunc
	ReleaseHandler ReleaseHandler
	AssetHandler   AssetHandler
}

func NewRouter(deps Dependencies) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.NewCORSMiddleware())

	r.POST("/api/v1/releases", deps.AuthMiddleware, deps.ReleaseHandler.Upload)
	r.GET("/api/v1/apps/:app/versions", deps.ReleaseHandler.ListVersions)
	r.GET("/assets/:app/:version/*path", deps.AssetHandler.Get)
	return r
}
