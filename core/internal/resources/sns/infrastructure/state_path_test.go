package infrastructure

import (
	"path/filepath"
	"testing"
)

func TestResolveStatePathIsInstanceScoped(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	first, err := ResolveStatePath(baseDir, "instance-a")
	if err != nil {
		t.Fatalf("resolve first path: %v", err)
	}
	second, err := ResolveStatePath(baseDir, "instance-b")
	if err != nil {
		t.Fatalf("resolve second path: %v", err)
	}

	if first == second {
		t.Fatalf("expected distinct paths, got %q", first)
	}
	if got, want := first, filepath.Join(baseDir, "instances", "instance-a", "sns"); got != want {
		t.Fatalf("unexpected first path: got %q want %q", got, want)
	}
}
