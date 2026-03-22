package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"app-assets-service/internal/domain"
	"app-assets-service/internal/repository"
)

var ErrReleaseAlreadyExists = errors.New("release already exists")

const (
	releaseStatusSuccess        = "success"
	releaseStatusRotatedDeleted = "rotated_deleted"
	maxActiveVersionsPerAppEnv  = 15
)

type ExtractedFile struct {
	RelativePath string
	FullPath     string
	ContentType  string
}

type ReleaseInput struct {
	AppName      string
	Version      string
	Environment  string
	CommitSHA    string
	BuildID      string
	ExtractedDir string
	Files        []ExtractedFile
}

type ReleaseService struct {
	repo    repository.ReleaseRepository
	storage repository.ObjectStorage
}

func NewReleaseService(repo repository.ReleaseRepository, storage repository.ObjectStorage) *ReleaseService {
	return &ReleaseService{repo: repo, storage: storage}
}

func (s *ReleaseService) CreateRelease(ctx context.Context, in ReleaseInput) error {
	exists, err := s.repo.Exists(ctx, in.AppName, in.Environment, in.Version)
	if err != nil {
		return err
	}
	if exists {
		return ErrReleaseAlreadyExists
	}

	prefix := buildPrefix(in.Environment, in.AppName, in.Version)
	for _, file := range in.Files {
		key := path.Join(prefix, file.RelativePath)
		if err := s.storage.UploadFile(ctx, key, file.FullPath, file.ContentType); err != nil {
			_ = s.storage.DeletePrefix(ctx, prefix)
			return fmt.Errorf("upload %s: %w", key, err)
		}
	}

	release := domain.Release{
		AppName:       in.AppName,
		Version:       in.Version,
		Environment:   in.Environment,
		Status:        releaseStatusSuccess,
		StoragePrefix: "/" + prefix + "/",
		CommitSHA:     in.CommitSHA,
		BuildID:       in.BuildID,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repo.Insert(ctx, release); err != nil {
		_ = s.storage.DeletePrefix(ctx, prefix)
		return err
	}

	if err := s.rotateOldReleases(ctx, in.AppName, in.Environment); err != nil {
		return err
	}

	return nil
}

func buildPrefix(environment, app, version string) string {
	return path.Join(environment, app, version)
}

func (s *ReleaseService) rotateOldReleases(ctx context.Context, appName, environment string) error {
	activeReleases, err := s.repo.ListActiveByAppEnvironment(ctx, appName, environment)
	if err != nil {
		return err
	}
	if len(activeReleases) <= maxActiveVersionsPerAppEnv {
		return nil
	}

	sort.Slice(activeReleases, func(i, j int) bool {
		return activeReleases[i].CreatedAt.After(activeReleases[j].CreatedAt)
	})

	for _, stale := range activeReleases[maxActiveVersionsPerAppEnv:] {
		prefix := strings.Trim(stale.StoragePrefix, "/")
		if prefix == "" {
			prefix = buildPrefix(stale.Environment, stale.AppName, stale.Version)
		}
		if err := s.storage.DeletePrefix(ctx, prefix); err != nil {
			return fmt.Errorf("rotate delete prefix %s: %w", prefix, err)
		}
		if err := s.repo.UpdateStatus(ctx, stale.AppName, stale.Environment, stale.Version, releaseStatusRotatedDeleted); err != nil {
			return err
		}
	}
	return nil
}
