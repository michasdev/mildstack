package runtime

import (
	"path/filepath"
	"testing"
)

func TestResolvePathsFromPrefersHomeDirectory(t *testing.T) {
	t.Helper()

	paths := ResolvePathsFrom("/Users/alex", "/Users/alex/Library/Application Support")

	if got, want := paths.BaseDir, filepath.Join("/Users/alex", ".mildstack"); got != want {
		t.Fatalf("unexpected base dir: got %q want %q", got, want)
	}
	if got, want := paths.ConfigDir, filepath.Join("/Users/alex", ".mildstack", "config"); got != want {
		t.Fatalf("unexpected config dir: got %q want %q", got, want)
	}
	if got, want := paths.InstancesDir, filepath.Join("/Users/alex", ".mildstack", "instances"); got != want {
		t.Fatalf("unexpected instances dir: got %q want %q", got, want)
	}
	if got, want := paths.LogsDir, filepath.Join("/Users/alex", ".mildstack", "logs"); got != want {
		t.Fatalf("unexpected logs dir: got %q want %q", got, want)
	}
	if got, want := paths.CacheDir, filepath.Join("/Users/alex", ".mildstack", "cache"); got != want {
		t.Fatalf("unexpected cache dir: got %q want %q", got, want)
	}
}

func TestResolvePathsFromFallsBackToConfigDirectory(t *testing.T) {
	t.Helper()

	paths := ResolvePathsFrom("", "/Users/alex/Library/Application Support")

	if got, want := paths.BaseDir, filepath.Join("/Users/alex/Library/Application Support", ".mildstack"); got != want {
		t.Fatalf("unexpected fallback base dir: got %q want %q", got, want)
	}
	if got, want := LegacyBaseDirFrom("", "/Users/alex/Library/Application Support"), filepath.Join("/Users/alex/Library/Application Support", "mildstack"); got != want {
		t.Fatalf("unexpected legacy base dir: got %q want %q", got, want)
	}
}
