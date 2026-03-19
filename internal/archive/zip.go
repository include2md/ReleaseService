package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

var ErrZipSlip = errors.New("zip slip detected")

type FileEntry struct {
	RelativePath string
	FullPath     string
	ContentType  string
}

func IsZip(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	return header[0] == 'P' && header[1] == 'K' && header[2] == 0x03 && header[3] == 0x04
}

func ExtractZipBytes(data []byte) (string, []FileEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", nil, err
	}

	tmpDir, err := os.MkdirTemp("", "release-artifact-*")
	if err != nil {
		return "", nil, err
	}

	entries := make([]FileEntry, 0, len(zr.File))
	base := filepath.Clean(tmpDir) + string(os.PathSeparator)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		cleanName := filepath.Clean(f.Name)
		target := filepath.Join(tmpDir, cleanName)
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, base) {
			_ = os.RemoveAll(tmpDir)
			return "", nil, ErrZipSlip
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			_ = os.RemoveAll(tmpDir)
			return "", nil, err
		}

		rc, err := f.Open()
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return "", nil, err
		}
		out, err := os.Create(cleanTarget)
		if err != nil {
			_ = rc.Close()
			_ = os.RemoveAll(tmpDir)
			return "", nil, err
		}
		_, copyErr := io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			_ = os.RemoveAll(tmpDir)
			return "", nil, copyErr
		}

		entries = append(entries, FileEntry{
			RelativePath: filepath.ToSlash(cleanName),
			FullPath:     cleanTarget,
			ContentType:  detectContentType(cleanTarget),
		})
	}

	return tmpDir, entries, nil
}

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
