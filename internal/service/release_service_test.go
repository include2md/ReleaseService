package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"releaseservice/internal/domain"
	"releaseservice/internal/repository"
)

type fakeReleaseRepo struct {
	exists    bool
	insertErr error
	inserted  bool
}

func (f *fakeReleaseRepo) Exists(ctx context.Context, appName, environment, version string) (bool, error) {
	return f.exists, nil
}

func (f *fakeReleaseRepo) Insert(ctx context.Context, release domain.Release) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = true
	return nil
}

type fakeStorage struct {
	uploadErr     error
	cleanupCalled bool
	uploadedKeys  []string
}

func (f *fakeStorage) UploadFile(ctx context.Context, key, localPath, contentType string) error {
	if f.uploadErr != nil {
		return f.uploadErr
	}
	f.uploadedKeys = append(f.uploadedKeys, key)
	return nil
}

func (f *fakeStorage) GetObject(ctx context.Context, key string) (*repository.ObjectResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeStorage) DeletePrefix(ctx context.Context, prefix string) error {
	f.cleanupCalled = true
	return nil
}

func TestReleaseService_CreateRelease_ReturnsConflictWhenVersionExists(t *testing.T) {
	repo := &fakeReleaseRepo{exists: true}
	storage := &fakeStorage{}
	svc := NewReleaseService(repo, storage)

	err := svc.CreateRelease(context.Background(), ReleaseInput{
		AppName:     "my-app",
		Environment: "prod",
		Version:     "1.2.3",
	})

	if !errors.Is(err, ErrReleaseAlreadyExists) {
		t.Fatalf("expected ErrReleaseAlreadyExists, got %v", err)
	}
}

func TestReleaseService_CleansUpUploadedObjectsWhenMongoInsertFails(t *testing.T) {
	repo := &fakeReleaseRepo{insertErr: errors.New("mongo down")}
	storage := &fakeStorage{}
	svc := NewReleaseService(repo, storage)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := svc.CreateRelease(context.Background(), ReleaseInput{
		AppName:      "my-app",
		Environment:  "prod",
		Version:      "1.2.3",
		ExtractedDir: tmpDir,
		Files: []ExtractedFile{
			{RelativePath: "index.html", FullPath: filePath, ContentType: "text/html"},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !storage.cleanupCalled {
		t.Fatal("expected cleanup to be called")
	}
}
