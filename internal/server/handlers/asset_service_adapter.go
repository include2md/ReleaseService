package handlers

import (
	"context"

	"app-assets-service/internal/service"
)

type AssetGetter interface {
	GetAsset(ctx context.Context, req service.AssetRequest) (*service.AssetResult, error)
}

type AssetServiceAdapter struct {
	Svc AssetGetter
}

func (a AssetServiceAdapter) GetAsset(ctx context.Context, in AssetRequest) (AssetResponse, error) {
	res, err := a.Svc.GetAsset(ctx, service.AssetRequest{
		Environment: in.Environment,
		App:         in.App,
		Version:     in.Version,
		AssetPath:   in.Path,
	})
	if err != nil {
		return AssetResponse{}, err
	}
	return AssetResponse{Body: res.Body, ContentType: res.ContentType}, nil
}
