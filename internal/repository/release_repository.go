package repository

import (
	"app-assets-service/internal/domain"
	"context"
)

type ReleaseRepository interface {
	Exists(ctx context.Context, appName, environment, version string) (bool, error)
	Insert(ctx context.Context, release domain.Release) error
	ListActiveByAppEnvironment(ctx context.Context, appName, environment string) ([]domain.Release, error)
	UpdateStatus(ctx context.Context, appName, environment, version, status string) error
}
