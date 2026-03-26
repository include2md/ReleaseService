package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"app-assets-service/internal/domain"
)

type MongoReleaseRepository struct {
	col *mongo.Collection
}

func NewMongoReleaseRepository(col *mongo.Collection) (*MongoReleaseRepository, error) {
	repo := &MongoReleaseRepository{col: col}
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "app_name", Value: 1}, {Key: "version", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := col.Indexes().CreateOne(context.Background(), idx)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (m *MongoReleaseRepository) Exists(ctx context.Context, appName, version string) (bool, error) {
	filter := bson.M{"app_name": appName, "version": version}
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
		"status":         release.Status,
		"storage_prefix": release.StoragePrefix,
		"commit_sha":     release.CommitSHA,
		"build_id":       release.BuildID,
		"created_at":     release.CreatedAt,
	})
	return err
}

func (m *MongoReleaseRepository) ListActiveByApp(ctx context.Context, appName string) ([]domain.Release, error) {
	filter := bson.M{
		"app_name": appName,
		"status":   "success",
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var releases []domain.Release
	for cur.Next(ctx) {
		var doc struct {
			AppName       string    `bson:"app_name"`
			Version       string    `bson:"version"`
			Status        string    `bson:"status"`
			StoragePrefix string    `bson:"storage_prefix"`
			CommitSHA     string    `bson:"commit_sha"`
			BuildID       string    `bson:"build_id"`
			CreatedAt     time.Time `bson:"created_at"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		releases = append(releases, domain.Release{
			AppName:       doc.AppName,
			Version:       doc.Version,
			Status:        doc.Status,
			StoragePrefix: doc.StoragePrefix,
			CommitSHA:     doc.CommitSHA,
			BuildID:       doc.BuildID,
			CreatedAt:     doc.CreatedAt,
		})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return releases, nil
}

func (m *MongoReleaseRepository) UpdateStatus(ctx context.Context, appName, version, status string) error {
	filter := bson.M{"app_name": appName, "version": version}
	update := bson.M{"$set": bson.M{"status": status}}
	_, err := m.col.UpdateOne(ctx, filter, update)
	return err
}
