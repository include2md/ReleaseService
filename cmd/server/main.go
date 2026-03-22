package main

import (
	"context"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"app-assets-service/internal/config"
	"app-assets-service/internal/repository"
	"app-assets-service/internal/server"
	"app-assets-service/internal/server/handlers"
	"app-assets-service/internal/server/middleware"
	"app-assets-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}

	minioClient, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("init minio client: %v", err)
	}

	releaseRepo, err := repository.NewMongoReleaseRepository(mongoClient.Database(cfg.MongoDB).Collection(cfg.MongoCollection))
	if err != nil {
		log.Fatalf("init release repo: %v", err)
	}
	storage := repository.NewMinioObjectStorage(minioClient, cfg.MinioBucket)
	releaseSvc := service.NewReleaseService(releaseRepo, storage)
	assetSvc := service.NewAssetService(storage)

	releaseHandler := handlers.NewReleaseHandler(releaseSvc)
	assetHandler := handlers.NewAssetHandler(handlers.AssetServiceAdapter{Svc: assetSvc})
	auth := middleware.NewAuthMiddleware(cfg.ReleaseTokens)

	r := server.NewRouter(server.Dependencies{
		AuthMiddleware: auth,
		ReleaseHandler: releaseHandler,
		AssetHandler:   assetHandler,
	})

	if err := r.Run(cfg.Addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
