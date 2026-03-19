package config

import "testing"

func TestLoad_RequiresMongoURI(t *testing.T) {
	t.Setenv("MONGO_URI", "")
	t.Setenv("MINIO_ENDPOINT", "x")
	t.Setenv("MINIO_ACCESS_KEY", "x")
	t.Setenv("MINIO_SECRET_KEY", "x")
	t.Setenv("MINIO_BUCKET", "x")
	t.Setenv("RELEASE_TOKENS", "t1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}
