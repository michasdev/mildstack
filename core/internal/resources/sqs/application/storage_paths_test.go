package application

import (
	"path/filepath"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

func TestResolveStoragePathUsesInstanceScopedLayout(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	path, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}

	want := filepath.Join(baseDir, "instances", "instance-a", "sqs")
	if got, want := path, want; got != want {
		t.Fatalf("unexpected storage path: got %q want %q", got, want)
	}

	otherPath, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "instance-b",
	})
	if err != nil {
		t.Fatalf("resolve other storage path: %v", err)
	}
	if path == otherPath {
		t.Fatalf("expected distinct storage paths, got %q", path)
	}
}

func TestResolveStoragePathFallsBackToRuntimeBaseDir(t *testing.T) {
	t.Helper()

	expectedBase := runtime.ResolvePaths().BaseDir
	if expectedBase == "" {
		t.Fatal("expected runtime base dir to be available")
	}

	path, err := ResolveStoragePath(StorageConfig{
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("resolve storage path with runtime base dir: %v", err)
	}

	want := filepath.Join(expectedBase, "instances", "instance-a", "sqs")
	if got, want := path, want; got != want {
		t.Fatalf("unexpected fallback storage path: got %q want %q", got, want)
	}
}

func TestResolveStoragePathRejectsEmptyInstanceID(t *testing.T) {
	t.Helper()

	if _, err := ResolveStoragePath(StorageConfig{BaseDir: t.TempDir()}); err == nil {
		t.Fatal("expected empty instance id to be rejected")
	}
}
