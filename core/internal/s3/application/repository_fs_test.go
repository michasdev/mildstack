package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

func TestFSRepositorySaveAndLoadAreDeterministic(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	state := domain.State{
		Service: "s3",
		Buckets: []domain.Bucket{
			{Name: "zeta", Region: "us-east-1", CreatedAt: time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC)},
			{Name: "alpha", Region: "us-west-2", CreatedAt: time.Date(2026, time.April, 17, 11, 0, 0, 0, time.UTC)},
		},
		Objects: []domain.Object{
			{Bucket: "zeta", Key: "b.txt", Size: 2, ContentType: "text/plain"},
			{Bucket: "alpha", Key: "a.txt", Size: 1, ContentType: "text/plain"},
		},
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save first pass: %v", err)
	}
	firstBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read first state file: %v", err)
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save second pass: %v", err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read second state file: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("expected deterministic state.json output across identical saves")
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got, want := loaded.Buckets[0].Name, "alpha"; got != want {
		t.Fatalf("expected sorted bucket order after round-trip: got %q want %q", got, want)
	}
	if loaded.Buckets[0].CreatedAt.IsZero() || loaded.Buckets[1].CreatedAt.IsZero() {
		t.Fatal("expected bucket creation timestamps after round-trip")
	}
	if got, want := loaded.Objects[0].Bucket, "alpha"; got != want {
		t.Fatalf("expected sorted object order after round-trip: got %q want %q", got, want)
	}
}

func TestFSRepositoryLoadTreatsMissingFileAsNewState(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load missing state: %v", err)
	}
	if got, want := loaded.Service, "s3"; got != want {
		t.Fatalf("unexpected default service: got %q want %q", got, want)
	}
}

func TestFSRepositoryLoadRejectsInvalidState(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	statePath := filepath.Join(repo.storagePath, stateFileName)
	if err := os.MkdirAll(repo.storagePath, 0o755); err != nil {
		t.Fatalf("mkdir storage path: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("{oops"), 0o644); err != nil {
		t.Fatalf("write invalid state: %v", err)
	}

	_, err := repo.Load()
	if err == nil {
		t.Fatal("expected invalid state file to return an error")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("unexpected error for invalid state: %v", err)
	}
}

func TestResolveStoragePathUsesMildstackInstanceLayout(t *testing.T) {
	t.Helper()

	path, err := ResolveStoragePath(StorageConfig{
		BaseDir:    "/tmp/mildstack-root",
		InstanceID: "instance-42",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	if got, want := path, filepath.Join("/tmp/mildstack-root", "instances", "instance-42", "s3"); got != want {
		t.Fatalf("unexpected path: got %q want %q", got, want)
	}
}
