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

func TestRegistrarNormalizesTrailingSlashRoutes(t *testing.T) {
	t.Helper()

	registrar := NewRegistrar()

	if err := registrar.Register(orchestrator.Route{
		Method: "post",
		Path:   "/alpha/items/",
		Name:   "create-item",
	}); err != nil {
		t.Fatalf("register route: %v", err)
	}

	entry, ok := registrar.Service("alpha")
	if !ok {
		t.Fatal("expected alpha service to be registered")
	}
	if got, want := entry.Routes[0].Path, "/api/v1/runtime/services/alpha/items"; got != want {
		t.Fatalf("unexpected normalized path: got %q want %q", got, want)
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

func TestRegistrarRejectsDuplicateRouteNamesAcrossDifferentPaths(t *testing.T) {
	t.Helper()

	registrar := NewRegistrar()

	if err := registrar.Register(orchestrator.Route{
		Method: "GET",
		Path:   "/alpha/health",
		Name:   "health",
	}); err != nil {
		t.Fatalf("register first route: %v", err)
	}
	if err := registrar.Register(orchestrator.Route{
		Method: "POST",
		Path:   "/alpha/items",
		Name:   "health",
	}); !errors.Is(err, ErrDuplicateRoute) {
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

func TestRegistrarServicesReturnsSortedCatalog(t *testing.T) {
	t.Helper()

	registrar := NewRegistrar()

	for _, route := range []orchestrator.Route{
		{Method: "GET", Path: "/beta/health", Name: "health"},
		{Method: "GET", Path: "/alpha/health", Name: "health"},
	} {
		if err := registrar.Register(route); err != nil {
			t.Fatalf("register route %+v: %v", route, err)
		}
	}

	entries := registrar.Services()
	if got, want := len(entries), 2; got != want {
		t.Fatalf("unexpected entry count: got %d want %d", got, want)
	}
	if got, want := entries[0].Name, "alpha"; got != want {
		t.Fatalf("unexpected first entry: got %q want %q", got, want)
	}
	if got, want := entries[1].Name, "beta"; got != want {
		t.Fatalf("unexpected second entry: got %q want %q", got, want)
	}
}
