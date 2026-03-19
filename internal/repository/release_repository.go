package repository

import (
	"context"
	"releaseservice/internal/domain"
)

type ReleaseRepository interface {
	Exists(ctx context.Context, appName, environment, version string) (bool, error)
	Insert(ctx context.Context, release domain.Release) error
}
