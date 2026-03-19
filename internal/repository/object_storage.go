package repository

import (
	"context"
	"io"
)

type ObjectResult struct {
	Body        io.ReadCloser
	ContentType string
}

type ObjectStorage interface {
	UploadFile(ctx context.Context, key, localPath, contentType string) error
	GetObject(ctx context.Context, key string) (*ObjectResult, error)
	DeletePrefix(ctx context.Context, prefix string) error
}
