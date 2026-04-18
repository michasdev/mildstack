package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var forbiddenImports = []string{
	"github.com/gin-gonic/gin",
	"github.com/spf13/cobra",
	"github.com/charmbracelet/bubbletea",
	"github.com/charmbracelet/lipgloss",
}

func TestInwardLayersStayFrameworkFree(t *testing.T) {
	t.Helper()

	mustExist(t, "../domain")
	mustExist(t, "../application")
	mustExist(t, "../application/orchestrator")
	mustExist(t, "../application/runtime")
	mustExist(t, "../composition")
	mustExist(t, "../delivery")
	mustExist(t, "../delivery/cli")
	mustExist(t, "../delivery/cli/ui")
	mustExist(t, "../delivery/http")
	mustExist(t, "../infrastructure")
	mustExist(t, "../resources/s3/domain")
	mustExist(t, "../resources/s3/application")
	mustExist(t, "../resources/s3/infrastructure")
	mustExist(t, "../resources/dynamodb/domain")
	mustExist(t, "../resources/dynamodb/application")
	mustExist(t, "../resources/dynamodb/infrastructure")
	mustExist(t, "layout.md")

	scan := []string{
		"../domain",
		"../application",
		"../application/orchestrator",
		"../application/runtime",
		"../resources/s3/domain",
		"../resources/s3/application",
		"../resources/s3/infrastructure",
		"../resources/dynamodb/domain",
		"../resources/dynamodb/application",
		"../resources/dynamodb/infrastructure",
	}
	for _, root := range scan {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			assertNoForbiddenImports(t, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

func assertNoForbiddenImports(t *testing.T, path string) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		for _, forbidden := range forbiddenImports {
			if importPath == forbidden {
				t.Fatalf("%s imports forbidden package %s", path, forbidden)
			}
		}
		if strings.HasPrefix(importPath, "github.com/gin-gonic/gin") ||
			strings.HasPrefix(importPath, "github.com/spf13/cobra") ||
			strings.HasPrefix(importPath, "github.com/charmbracelet/bubbletea") ||
			strings.HasPrefix(importPath, "github.com/charmbracelet/lipgloss") {
			t.Fatalf("%s imports forbidden framework package %s", path, importPath)
		}
		if strings.Contains(importPath, "core/internal/infrastructure/") || importPath == "core/internal/infrastructure" {
			t.Fatalf("%s imports forbidden infrastructure package %s", path, importPath)
		}
	}
}

func mustExist(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
