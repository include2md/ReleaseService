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

func TestLoad_DefaultMongoCollection(t *testing.T) {
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("MINIO_ENDPOINT", "x")
	t.Setenv("MINIO_ACCESS_KEY", "x")
	t.Setenv("MINIO_SECRET_KEY", "x")
	t.Setenv("MINIO_BUCKET", "x")
	t.Setenv("RELEASE_TOKENS", "t1")
	t.Setenv("MONGO_COLLECTION", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.MongoCollection != "app_asset_releases" {
		t.Fatalf("expected default mongo collection app_asset_releases, got %q", cfg.MongoCollection)
	}
}

func TestLoad_UsesMongoCollectionOverride(t *testing.T) {
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("MINIO_ENDPOINT", "x")
	t.Setenv("MINIO_ACCESS_KEY", "x")
	t.Setenv("MINIO_SECRET_KEY", "x")
	t.Setenv("MINIO_BUCKET", "x")
	t.Setenv("RELEASE_TOKENS", "t1")
	t.Setenv("MONGO_COLLECTION", "custom_collection")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.MongoCollection != "custom_collection" {
		t.Fatalf("expected mongo collection override custom_collection, got %q", cfg.MongoCollection)
	}
}
