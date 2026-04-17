//go:build windows

package runtime

import (
	"path/filepath"
	"testing"
)

func TestResolvePathsFromUsesWindowsSafeHomePath(t *testing.T) {
	t.Helper()

	paths := ResolvePathsFrom(`C:\Users\alex`, `C:\Users\alex\AppData\Roaming`)

	if got, want := paths.BaseDir, filepath.Join(`C:\Users\alex`, ".mildstack"); got != want {
		t.Fatalf("unexpected base dir: got %q want %q", got, want)
	}
	if got, want := paths.ConfigDir, filepath.Join(`C:\Users\alex`, ".mildstack", "config"); got != want {
		t.Fatalf("unexpected config dir: got %q want %q", got, want)
	}
}
