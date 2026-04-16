package http

import (
	"errors"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

func TestRegistrarNormalizesRoutesUnderServiceNamespace(t *testing.T) {
	t.Helper()

	registrar := NewRegistrar()

	if err := registrar.Register(orchestrator.Route{
		Method: "get",
		Path:   "/alpha/health",
		Name:   "health",
	}); err != nil {
		t.Fatalf("register route: %v", err)
	}

	entry, ok := registrar.Service("alpha")
	if !ok {
		t.Fatal("expected alpha service to be registered")
	}
	if got, want := len(entry.Routes), 1; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}

	route := entry.Routes[0]
	if got, want := route.Method, "GET"; got != want {
		t.Fatalf("unexpected route method: got %q want %q", got, want)
	}
	if got, want := route.Path, "/api/v1/runtime/services/alpha/health"; got != want {
		t.Fatalf("unexpected normalized path: got %q want %q", got, want)
	}
	if got, want := route.Name, "health"; got != want {
		t.Fatalf("unexpected route name: got %q want %q", got, want)
	}
}

func TestRegistrarRejectsDuplicateRoutes(t *testing.T) {
	t.Helper()

	registrar := NewRegistrar()

	route := orchestrator.Route{
		Method: "GET",
		Path:   "/alpha/health",
		Name:   "health",
	}
	if err := registrar.Register(route); err != nil {
		t.Fatalf("register route: %v", err)
	}
	if err := registrar.Register(route); !errors.Is(err, ErrDuplicateRoute) {
		t.Fatalf("expected duplicate route error, got %v", err)
	}
}

func TestRegistrarRejectsMalformedRoutes(t *testing.T) {
	t.Helper()

	registrar := NewRegistrar()

	for _, route := range []orchestrator.Route{
		{Method: "GET", Path: "alpha/health", Name: "health"},
		{Method: "GET", Path: "/alpha//health", Name: "health"},
		{Method: "GET", Path: "/", Name: "health"},
	} {
		if err := registrar.Register(route); !errors.Is(err, ErrInvalidRoute) {
			t.Fatalf("expected invalid route error for %+v, got %v", route, err)
		}
	}
}
