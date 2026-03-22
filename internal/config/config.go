package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Addr            string
	MongoURI        string
	MongoDB         string
	MongoCollection string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucket     string
	MinioUseSSL     bool
	ReleaseTokens   []string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:            envOrDefault("ADDR", ":8080"),
		MongoURI:        os.Getenv("MONGO_URI"),
		MongoDB:         envOrDefault("MONGO_DB", "app_asset_service"),
		MongoCollection: envOrDefault("MONGO_COLLECTION", "app_asset_releases"),
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:     os.Getenv("MINIO_BUCKET"),
		MinioUseSSL:     strings.EqualFold(os.Getenv("MINIO_USE_SSL"), "true"),
		ReleaseTokens:   splitCSV(os.Getenv("RELEASE_TOKENS")),
	}

	if cfg.MongoURI == "" {
		return Config{}, errors.New("MONGO_URI is required")
	}
	if cfg.MinioEndpoint == "" || cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" || cfg.MinioBucket == "" {
		return Config{}, errors.New("MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_BUCKET are required")
	}
	if len(cfg.ReleaseTokens) == 0 {
		return Config{}, errors.New("RELEASE_TOKENS is required")
	}
	return cfg, nil
}

func envOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
