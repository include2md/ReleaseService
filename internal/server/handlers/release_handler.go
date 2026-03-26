package handlers

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"app-assets-service/internal/archive"
	"app-assets-service/internal/service"
)

var ErrReleaseAlreadyExists = service.ErrReleaseAlreadyExists

type ReleaseService interface {
	CreateRelease(ctx context.Context, in service.ReleaseInput) error
	ListAvailableVersions(ctx context.Context, appName string) ([]string, error)
}

type ReleaseHandler struct {
	svc ReleaseService
}

func NewReleaseHandler(svc ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{svc: svc}
}

func (h *ReleaseHandler) Upload(c *gin.Context) {
	app := c.PostForm("app_name")
	version := c.PostForm("version")
	if app == "" || version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}

	file, _, err := c.Request.FormFile("artifact")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact is required"})
		return
	}
	defer file.Close()

	artifactBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artifact"})
		return
	}
	if !archive.IsTarGz(artifactBytes) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact must be tar.gz"})
		return
	}

	dir, entries, err := archive.ExtractTarGzBytes(artifactBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tar.gz"})
		return
	}
	defer os.RemoveAll(dir)

	files := make([]service.ExtractedFile, 0, len(entries))
	for _, e := range entries {
		files = append(files, service.ExtractedFile{
			RelativePath: e.RelativePath,
			FullPath:     e.FullPath,
			ContentType:  e.ContentType,
		})
	}

	err = h.svc.CreateRelease(c.Request.Context(), service.ReleaseInput{
		AppName:      app,
		Version:      version,
		CommitSHA:    c.PostForm("commit_sha"),
		BuildID:      c.PostForm("build_id"),
		ExtractedDir: dir,
		Files:        files,
	})
	if err != nil {
		if err == service.ErrReleaseAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "release already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create release"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"app_name": app,
		"version":  version,
		"status":   "success",
	})
}

func (h *ReleaseHandler) ListVersions(c *gin.Context) {
	app := c.Param("app")
	if app == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app is required"})
		return
	}

	versions, err := h.svc.ListAvailableVersions(c.Request.Context(), app)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"app_name": app,
		"versions": versions,
	})
}
