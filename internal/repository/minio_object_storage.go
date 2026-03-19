package repository

import (
	"context"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
)

type MinioObjectStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioObjectStorage(client *minio.Client, bucket string) *MinioObjectStorage {
	return &MinioObjectStorage{client: client, bucket: bucket}
}

func (m *MinioObjectStorage) UploadFile(ctx context.Context, key, localPath, contentType string) error {
	_, err := m.client.FPutObject(ctx, m.bucket, key, localPath, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (m *MinioObjectStorage) GetObject(ctx context.Context, key string) (*ObjectResult, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	stat, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	return &ObjectResult{Body: obj, ContentType: stat.ContentType}, nil
}

func (m *MinioObjectStorage) DeletePrefix(ctx context.Context, prefix string) error {
	prefix = strings.TrimPrefix(path.Clean(prefix), "/")
	for obj := range m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return obj.Err
		}
		err := m.client.RemoveObject(ctx, m.bucket, obj.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
