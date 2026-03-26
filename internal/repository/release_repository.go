package repository

import (
	"app-assets-service/internal/domain"
	"context"
)

type ReleaseRepository interface {
	Exists(ctx context.Context, appName, version string) (bool, error)
	Insert(ctx context.Context, release domain.Release) error
	ListActiveByApp(ctx context.Context, appName string) ([]domain.Release, error)
	UpdateStatus(ctx context.Context, appName, version, status string) error
}
