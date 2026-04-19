package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceScopeGuardrailRequiresSharedInstancePathHelper(t *testing.T) {
	t.Helper()

	matches, err := filepath.Glob("../resources/*/application/storage_paths.go")
	if err != nil {
		t.Fatalf("glob storage paths: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one storage path resolver to validate")
	}

	for _, path := range matches {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(content), "instancepath.Resolve(") {
			t.Fatalf("%s must use the shared instancepath helper", path)
		}
	}
}
