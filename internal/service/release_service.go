package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"releaseservice/internal/domain"
	"releaseservice/internal/repository"
)

var ErrReleaseAlreadyExists = errors.New("release already exists")

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
		Status:        "success",
		StoragePrefix: "/" + prefix + "/",
		CommitSHA:     in.CommitSHA,
		BuildID:       in.BuildID,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repo.Insert(ctx, release); err != nil {
		_ = s.storage.DeletePrefix(ctx, prefix)
		return err
	}

	return nil
}

func buildPrefix(environment, app, version string) string {
	return path.Join(environment, app, version)
}
