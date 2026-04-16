package http

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
)

func TestNewRouterUsesVersionedBasePath(t *testing.T) {
	t.Helper()

	manager := runtime.New(composition.Assemble(nil))
	router := NewRouter(Config{}, manager)

	if got, want := router.BasePath(), "/api/v1"; got != want {
		t.Fatalf("unexpected base path: got %q want %q", got, want)
	}
	if got, want := router.RuntimePath(), "/api/v1/runtime"; got != want {
		t.Fatalf("unexpected runtime path: got %q want %q", got, want)
	}
	if got, want := router.ServicesPath(), "/api/v1/runtime/services"; got != want {
		t.Fatalf("unexpected services path: got %q want %q", got, want)
	}
	if router.Engine() == nil {
		t.Fatal("expected gin engine to be initialized")
	}
}

func TestNewRouterNormalizesCustomBasePath(t *testing.T) {
	t.Helper()

	manager := runtime.New(composition.Assemble(nil))
	router := NewRouter(Config{BasePath: "api/v1/"}, manager)

	if got, want := router.BasePath(), "/api/v1"; got != want {
		t.Fatalf("unexpected normalized base path: got %q want %q", got, want)
	}
	if got, want := router.RuntimePath(), "/api/v1/runtime"; got != want {
		t.Fatalf("unexpected normalized runtime path: got %q want %q", got, want)
	}
	if got, want := router.ServicesPath(), "/api/v1/runtime/services"; got != want {
		t.Fatalf("unexpected normalized services path: got %q want %q", got, want)
	}
	if snapshot := router.snapshotter.Snapshot(context.Background()); len(snapshot.Ports) != 0 {
		t.Fatalf("expected empty snapshot from runtime manager, got %v", snapshot.Ports)
	}
}
