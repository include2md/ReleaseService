package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"releaseservice/internal/domain"
)

type MongoReleaseRepository struct {
	col *mongo.Collection
}

func NewMongoReleaseRepository(col *mongo.Collection) (*MongoReleaseRepository, error) {
	repo := &MongoReleaseRepository{col: col}
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "app_name", Value: 1}, {Key: "environment", Value: 1}, {Key: "version", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := col.Indexes().CreateOne(context.Background(), idx)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (m *MongoReleaseRepository) Exists(ctx context.Context, appName, environment, version string) (bool, error) {
	filter := bson.M{"app_name": appName, "environment": environment, "version": version}
	count, err := m.col.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *MongoReleaseRepository) Insert(ctx context.Context, release domain.Release) error {
	_, err := m.col.InsertOne(ctx, bson.M{
		"app_name":       release.AppName,
		"version":        release.Version,
		"environment":    release.Environment,
		"status":         release.Status,
		"storage_prefix": release.StoragePrefix,
		"commit_sha":     release.CommitSHA,
		"build_id":       release.BuildID,
		"created_at":     release.CreatedAt,
	})
	return err
}
