package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestIsTarGz(t *testing.T) {
	if IsTarGz([]byte("not-a-tar-gz")) {
		t.Fatal("expected false for non tar.gz")
	}
	if !IsTarGz([]byte{0x1f, 0x8b, 0x08, 0x00}) {
		t.Fatal("expected true for gzip header")
	}
}

func TestExtractTarGzBytes_RejectsTarSlip(t *testing.T) {
	blob := buildTarGzBytes(t, "../evil.txt", []byte("x"))

	_, _, err := ExtractTarGzBytes(blob)
	if err == nil {
		t.Fatal("expected tar slip error")
	}
}

func TestExtractTarGzBytes_Success(t *testing.T) {
	blob := buildTarGzBytes(t, "assets/app.js", []byte("console.log('ok')"))

	dir, files, err := ExtractTarGzBytes(blob)
	if err != nil {
		t.Fatalf("extract tar.gz: %v", err)
	}
	defer os.RemoveAll(dir)

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].RelativePath != "assets/app.js" {
		t.Fatalf("unexpected relative path %s", files[0].RelativePath)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets/app.js")); err != nil {
		t.Fatalf("expected file exists: %v", err)
	}
}

func buildTarGzBytes(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}
