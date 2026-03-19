package archive

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIsZip(t *testing.T) {
	if IsZip([]byte("not-a-zip")) {
		t.Fatal("expected false for non zip")
	}
	if !IsZip([]byte{'P', 'K', 0x03, 0x04}) {
		t.Fatal("expected true for zip header")
	}
}

func TestExtractZipBytes_RejectsZipSlip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("../evil.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, _ = f.Write([]byte("x"))
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	_, _, err = ExtractZipBytes(buf.Bytes())
	if err == nil {
		t.Fatal("expected zip slip error")
	}
}

func TestExtractZipBytes_Success(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("assets/app.js")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, _ = f.Write([]byte("console.log('ok')"))
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	dir, files, err := ExtractZipBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("extract zip: %v", err)
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
