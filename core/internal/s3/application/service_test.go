package application

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func TestServiceMetadataRoutesAndState(t *testing.T) {
	t.Helper()

	service := New()
	if _, ok := any(service).(orchestrator.Service); !ok {
		t.Fatal("expected service to satisfy orchestrator.Service")
	}

	metadata := service.Metadata()
	if got, want := metadata.Name, "s3"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := metadata.Version, "v1"; got != want {
		t.Fatalf("unexpected service version: got %q want %q", got, want)
	}

	registrar := deliveryhttp.NewRegistrar()
	if err := service.RegisterRoutes(registrar); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	entry, ok := registrar.Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(entry.Routes), 4; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	if got, want := entry.Routes[0].Path, "/api/v1/runtime/services/s3/buckets"; got != want {
		t.Fatalf("unexpected first route path: got %q want %q", got, want)
	}
	if got, want := entry.Routes[3].Path, "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object"; got != want {
		t.Fatalf("unexpected last route path: got %q want %q", got, want)
	}

	hook := runtime.NewStateHook()
	if err := service.AttachState(hook); err != nil {
		t.Fatalf("attach state: %v", err)
	}

	value, ok := hook.Get("services/s3")
	if !ok {
		t.Fatal("expected s3 state to be present")
	}
	state := value.(map[string]any)
	if got, want := state["service"], "s3"; got != want {
		t.Fatalf("unexpected service state name: got %v want %v", got, want)
	}
}

func TestServiceStartAndStopAreNoops(t *testing.T) {
	t.Helper()

	service := New()

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
