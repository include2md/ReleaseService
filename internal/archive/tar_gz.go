package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrTarSlip = errors.New("tar slip detected")

func IsTarGz(header []byte) bool {
	if len(header) < 3 {
		return false
	}
	return header[0] == 0x1f && header[1] == 0x8b && header[2] == 0x08
}

func ExtractTarGzBytes(data []byte) (string, []FileEntry, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	tmpDir, err := os.MkdirTemp("", "release-artifact-*")
	if err != nil {
		return "", nil, err
	}

	entries := make([]FileEntry, 0, 16)
	base := filepath.Clean(tmpDir) + string(os.PathSeparator)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return "", nil, err
		}

		if hdr == nil || hdr.FileInfo().IsDir() {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}

		cleanName := filepath.Clean(hdr.Name)
		target := filepath.Join(tmpDir, cleanName)
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, base) {
			_ = os.RemoveAll(tmpDir)
			return "", nil, ErrTarSlip
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			_ = os.RemoveAll(tmpDir)
			return "", nil, err
		}

		out, err := os.Create(cleanTarget)
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return "", nil, err
		}
		_, copyErr := io.Copy(out, tr)
		_ = out.Close()
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
