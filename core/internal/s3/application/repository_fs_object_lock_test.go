package application

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

func TestFSRepositoryPersistsObjectLockRoundTrip(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	state := domain.NewState()
	bucket := state.UpsertBucket(domain.Bucket{
		Name:      "governed-bucket",
		Region:    "us-west-2",
		CreatedAt: time.Date(2026, time.April, 17, 15, 0, 0, 0, time.UTC),
	})
	state.SetBucketVersioning(bucket.Name, domain.VersioningEnabled)
	state.UpsertObject(domain.Object{
		Bucket:      bucket.Name,
		Key:         "archive.txt",
		Body:        []byte("payload"),
		Size:        int64(len("payload")),
		ContentType: "text/plain",
	})
	state.BucketObjectLock = map[string]domain.ObjectLockConfiguration{
		bucket.Name: {
			Enabled: true,
			DefaultRetention: &domain.ObjectLockRetention{
				Mode: "GOVERNANCE",
				Days: 30,
			},
		},
	}
	state.ObjectRetention = map[string]map[string]domain.ObjectRetention{
		bucket.Name: {
			"archive.txt": {
				Mode:            "GOVERNANCE",
				RetainUntilDate: time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	state.ObjectLegalHold = map[string]map[string]domain.ObjectLegalHold{
		bucket.Name: {
			"archive.txt": {
				Status: "ON",
			},
		},
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save governed state: %v", err)
	}
	firstBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read governed state file: %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load governed state: %v", err)
	}
	if got, ok := loaded.BucketObjectLockConfig(bucket.Name); !ok || !got.Enabled {
		t.Fatalf("unexpected loaded object lock config: ok=%v enabled=%v", ok, got.Enabled)
	}
	if got, ok := loaded.ObjectRetentionConfig(bucket.Name, "archive.txt"); !ok || got.Mode != "GOVERNANCE" {
		t.Fatalf("unexpected loaded retention: ok=%v retention=%+v", ok, got)
	}
	if got, ok := loaded.ObjectLegalHoldConfig(bucket.Name, "archive.txt"); !ok || got.Status != "ON" {
		t.Fatalf("unexpected loaded legal hold: ok=%v hold=%+v", ok, got)
	}

	if err := repo.Save(loaded); err != nil {
		t.Fatalf("resave governed state: %v", err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read resaved governed state file: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("expected governed state round-trip to stay deterministic")
	}
}
