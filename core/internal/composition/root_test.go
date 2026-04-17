package composition

import (
	"context"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

type stubService struct {
	name string
}

func (s *stubService) Start(context.Context) error { return nil }

func (s *stubService) Stop(context.Context) error { return nil }

func (s *stubService) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{Name: s.name}
}

func (s *stubService) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, nil, nil, "composition-test")
}

func (s *stubService) RegisterRoutes(orchestrator.RouteRegistrar) error { return nil }

func (s *stubService) AttachState(orchestrator.StateHook) error { return nil }

func TestAssemblePreservesOrderAndCopiesInput(t *testing.T) {
	t.Helper()

	first := &stubService{name: "one"}
	second := &stubService{name: "two"}
	input := []orchestrator.Service{first, second}

	root := Assemble(input)

	if len(root.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(root.Services))
	}
	if root.Services[0] != first || root.Services[1] != second {
		t.Fatal("service order was not preserved")
	}

	input[0] = second
	if root.Services[0] != first {
		t.Fatal("assembled services changed after input mutation")
	}
}

func TestRootDoesNotImportDIOrReflection(t *testing.T) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "root.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse root.go: %v", err)
	}

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "reflect" || path == "go.uber.org/dig" || path == "github.com/google/wire" || path == "go.uber.org/fx" {
			t.Fatalf("forbidden import detected: %s", path)
		}
	}
}
