package application

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	"github.com/michasdev/mildstack/core/internal/dynamodb/domain"
)

func TestServiceMetadataRoutesAndState(t *testing.T) {
	t.Helper()

	service := New()
	if _, ok := any(service).(orchestrator.Service); !ok {
		t.Fatal("expected service to satisfy orchestrator.Service")
	}

	metadata := service.Metadata()
	if got, want := metadata.Name, "dynamodb"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := metadata.Version, "v1"; got != want {
		t.Fatalf("unexpected service version: got %q want %q", got, want)
	}

	policy := service.Policy()
	if got, want := policy.Fidelity, orchestrator.FidelityExemplar; got != want {
		t.Fatalf("unexpected policy fidelity: got %q want %q", got, want)
	}
	if got, want := policy.ErrorPrefix, "dynamodb"; got != want {
		t.Fatalf("unexpected policy error prefix: got %q want %q", got, want)
	}
	if got, want := len(policy.Supported), 2; got != want {
		t.Fatalf("unexpected supported count: got %d want %d", got, want)
	}
	if got, want := len(policy.Unsupported), 2; got != want {
		t.Fatalf("unexpected unsupported count: got %d want %d", got, want)
	}
	policy.Supported[0] = "changed"
	policy.Unsupported[0] = "changed"
	again := service.Policy()
	if got, want := again.Supported[0], "list tables"; got != want {
		t.Fatalf("policy supported slice was not copied: got %q want %q", got, want)
	}
	if got, want := again.Unsupported[0], "global tables"; got != want {
		t.Fatalf("policy unsupported slice was not copied: got %q want %q", got, want)
	}

	expectedTags := []string{"aws", "database", "nosql", "exemplar"}
	if got, want := len(metadata.Tags), len(expectedTags); got != want {
		t.Fatalf("unexpected tag count: got %d want %d", got, want)
	}
	for i, tag := range expectedTags {
		if metadata.Tags[i] != tag {
			t.Fatalf("unexpected tag at %d: got %q want %q", i, metadata.Tags[i], tag)
		}
	}

	registrar := deliveryhttp.NewRegistrar()
	if err := service.RegisterRoutes(registrar); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	entry, ok := registrar.Service("dynamodb")
	if !ok {
		t.Fatal("expected dynamodb service to be registered")
	}
	if got, want := len(entry.Routes), 4; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	if got, want := entry.Routes[0].Path, "/api/v1/runtime/services/dynamodb/tables"; got != want {
		t.Fatalf("unexpected first route path: got %q want %q", got, want)
	}
	if got, want := entry.Routes[3].Path, "/api/v1/runtime/services/dynamodb/tables/:table/items/:item"; got != want {
		t.Fatalf("unexpected last route path: got %q want %q", got, want)
	}

	hook := runtime.NewStateHook()
	if err := service.AttachState(hook); err != nil {
		t.Fatalf("attach state: %v", err)
	}

	value, ok := hook.Get(domain.StateKey)
	if !ok {
		t.Fatalf("expected state for %q to be present", domain.StateKey)
	}
	state := value.(map[string]any)
	if got, want := state["service"], "dynamodb"; got != want {
		t.Fatalf("unexpected service state name: got %v want %v", got, want)
	}

	tables := state["tables"].([]any)
	if got, want := len(tables), 1; got != want {
		t.Fatalf("unexpected table count: got %d want %d", got, want)
	}

	items := state["items"].([]any)
	if got, want := len(items), 1; got != want {
		t.Fatalf("unexpected item count: got %d want %d", got, want)
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
