package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"app-assets-service/internal/domain"
	"app-assets-service/internal/repository"
)

type fakeReleaseRepo struct {
	exists         bool
	insertErr      error
	listErr        error
	updateErr      error
	inserted       bool
	activeReleases []domain.Release
	statusUpdates  []statusUpdate
}

type statusUpdate struct {
	appName string
	version string
	status  string
}

func (f *fakeReleaseRepo) Exists(ctx context.Context, appName, version string) (bool, error) {
	return f.exists, nil
}

func (f *fakeReleaseRepo) Insert(ctx context.Context, release domain.Release) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = true
	f.activeReleases = append(f.activeReleases, release)
	return nil
}

func (f *fakeReleaseRepo) ListActiveByApp(ctx context.Context, appName string) ([]domain.Release, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]domain.Release, 0, len(f.activeReleases))
	for _, r := range f.activeReleases {
		if r.AppName == appName {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReleaseRepo) UpdateStatus(ctx context.Context, appName, version, status string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.statusUpdates = append(f.statusUpdates, statusUpdate{
		appName: appName,
		version: version,
		status:  status,
	})
	for i, r := range f.activeReleases {
		if r.AppName == appName && r.Version == version {
			f.activeReleases[i].Status = status
		}
	}
	return nil
}

type fakeStorage struct {
	uploadErr     error
	deleteErr     error
	cleanupCalled bool
	uploadedKeys  []string
	deletedPrefix []string
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
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.cleanupCalled = true
	f.deletedPrefix = append(f.deletedPrefix, prefix)
	return nil
}

func TestReleaseService_CreateRelease_ReturnsConflictWhenVersionExists(t *testing.T) {
	repo := &fakeReleaseRepo{exists: true}
	storage := &fakeStorage{}
	svc := NewReleaseService(repo, storage)

	err := svc.CreateRelease(context.Background(), ReleaseInput{
		AppName: "my-app",
		Version: "1.2.3",
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

func TestReleaseService_RotatesOldReleasesOverLimit(t *testing.T) {
	now := time.Now().UTC()
	existing := make([]domain.Release, 0, 15)
	for i := 1; i <= 15; i++ {
		version := fmt.Sprintf("1.0.%d", i)
		existing = append(existing, domain.Release{
			AppName:       "my-app",
			Version:       version,
			Status:        "success",
			StoragePrefix: "/my-app/" + version + "/",
			CreatedAt:     now.Add(time.Duration(-16+i) * time.Minute),
		})
	}

	repo := &fakeReleaseRepo{activeReleases: existing}
	storage := &fakeStorage{}
	svc := NewReleaseService(repo, storage)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := svc.CreateRelease(context.Background(), ReleaseInput{
		AppName:      "my-app",
		Version:      "2.0.0",
		ExtractedDir: tmpDir,
		Files: []ExtractedFile{
			{RelativePath: "index.html", FullPath: filePath, ContentType: "text/html"},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(storage.deletedPrefix) != 1 {
		t.Fatalf("expected 1 rotated prefix delete, got %d", len(storage.deletedPrefix))
	}
	if storage.deletedPrefix[0] != "my-app/1.0.1" {
		t.Fatalf("expected oldest prefix to be deleted, got %q", storage.deletedPrefix[0])
	}
	if len(repo.statusUpdates) != 1 {
		t.Fatalf("expected 1 status update, got %d", len(repo.statusUpdates))
	}
	if repo.statusUpdates[0].status != "rotated_deleted" {
		t.Fatalf("expected status rotated_deleted, got %q", repo.statusUpdates[0].status)
	}
}

func TestReleaseService_DoesNotRotateWhenWithinLimit(t *testing.T) {
	now := time.Now().UTC()
	existing := make([]domain.Release, 0, 14)
	for i := 1; i <= 14; i++ {
		version := fmt.Sprintf("1.0.%d", i)
		existing = append(existing, domain.Release{
			AppName:       "my-app",
			Version:       version,
			Status:        "success",
			StoragePrefix: "/my-app/" + version + "/",
			CreatedAt:     now.Add(time.Duration(-15+i) * time.Minute),
		})
	}

	repo := &fakeReleaseRepo{activeReleases: existing}
	storage := &fakeStorage{}
	svc := NewReleaseService(repo, storage)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := svc.CreateRelease(context.Background(), ReleaseInput{
		AppName:      "my-app",
		Version:      "2.0.0",
		ExtractedDir: tmpDir,
		Files: []ExtractedFile{
			{RelativePath: "index.html", FullPath: filePath, ContentType: "text/html"},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(storage.deletedPrefix) != 0 {
		t.Fatalf("expected no rotation delete, got %d", len(storage.deletedPrefix))
	}
	if len(repo.statusUpdates) != 0 {
		t.Fatalf("expected no status updates, got %d", len(repo.statusUpdates))
	}
}

func TestReleaseService_ListAvailableVersions(t *testing.T) {
	repo := &fakeReleaseRepo{
		activeReleases: []domain.Release{
			{AppName: "my-app", Version: "2.0.0"},
			{AppName: "my-app", Version: "1.9.0"},
		},
	}
	svc := NewReleaseService(repo, &fakeStorage{})

	versions, err := svc.ListAvailableVersions(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0] != "2.0.0" || versions[1] != "1.9.0" {
		t.Fatalf("unexpected versions: %#v", versions)
	}
}
