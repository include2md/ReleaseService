package service

import (
	"context"
	"io"
	"path"

	"app-assets-service/internal/repository"
)

type AssetRequest struct {
	App       string
	Version   string
	AssetPath string
}

type AssetResult struct {
	Body        io.ReadCloser
	ContentType string
}

type AssetService struct {
	storage repository.ObjectStorage
}

func NewAssetService(storage repository.ObjectStorage) *AssetService {
	return &AssetService{storage: storage}
}

func (s *AssetService) GetAsset(ctx context.Context, req AssetRequest) (*AssetResult, error) {
	key := path.Join(req.App, req.Version, req.AssetPath)
	obj, err := s.storage.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	return &AssetResult{Body: obj.Body, ContentType: obj.ContentType}, nil
}
