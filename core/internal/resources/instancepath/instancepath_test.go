package instancepath

import (
	"path/filepath"
	"testing"
)

func TestResolveBuildsDistinctInstanceScopedPaths(t *testing.T) {
	t.Helper()

	baseDir := "/tmp/mildstack-root"
	first, err := Resolve(baseDir, "instance-a", "s3")
	if err != nil {
		t.Fatalf("resolve first path: %v", err)
	}
	second, err := Resolve(baseDir, "instance-b", "s3")
	if err != nil {
		t.Fatalf("resolve second path: %v", err)
	}

	if first == second {
		t.Fatalf("expected distinct paths, got %q", first)
	}
	if got, want := first, filepath.Join(baseDir, "instances", "instance-a", "s3"); got != want {
		t.Fatalf("unexpected first path: got %q want %q", got, want)
	}
	if got, want := second, filepath.Join(baseDir, "instances", "instance-b", "s3"); got != want {
		t.Fatalf("unexpected second path: got %q want %q", got, want)
	}
}

func TestResolveRejectsEmptyInstanceID(t *testing.T) {
	t.Helper()

	_, err := Resolve("/tmp/mildstack-root", "   ", "dynamodb")
	if err == nil {
		t.Fatal("expected empty instance id to be rejected")
	}
}
