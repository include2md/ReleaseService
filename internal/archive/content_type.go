package archive

import (
	"mime"
	"path/filepath"
)

func detectContentType(path string) string {
	ext := filepath.Ext(path)
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	if ext == ".js" {
		return "application/javascript"
	}
	if ext == ".css" {
		return "text/css"
	}
	return "application/octet-stream"
}
