package runtime

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

var _ orchestrator.Service = (*statefulServiceStub)(nil)

type statefulServiceStub struct {
	metadata orchestrator.Metadata
}

func (s *statefulServiceStub) Start(context.Context) error { return nil }

func (s *statefulServiceStub) Stop(context.Context) error { return nil }

func (s *statefulServiceStub) Metadata() orchestrator.Metadata { return s.metadata }

func (s *statefulServiceStub) RegisterRoutes(orchestrator.RouteRegistrar) error { return nil }

func (s *statefulServiceStub) AttachState(hook orchestrator.StateHook) error {
	hook.Set("services/"+s.metadata.Name, map[string]any{
		"name":    s.metadata.Name,
		"version": s.metadata.Version,
	})
	return nil
}

func TestMemoryStateHookSatisfiesServiceAttachStateContract(t *testing.T) {
	t.Helper()

	service := &statefulServiceStub{
		metadata: orchestrator.Metadata{
			Name:    "s3",
			Version: "v1",
		},
	}

	hook := NewStateHook()
	if err := service.AttachState(hook); err != nil {
		t.Fatalf("attach state: %v", err)
	}

	value, ok := hook.Get("services/s3")
	if !ok {
		t.Fatal("expected namespaced service state to be present")
	}

	state := value.(map[string]any)
	if got, want := state["name"], "s3"; got != want {
		t.Fatalf("unexpected service name: got %v want %v", got, want)
	}
	if got, want := state["version"], "v1"; got != want {
		t.Fatalf("unexpected service version: got %v want %v", got, want)
	}

	state["name"] = "mutated"
	again, ok := hook.Get("services/s3")
	if !ok {
		t.Fatal("expected service state to remain present")
	}
	restored := again.(map[string]any)
	if got, want := restored["name"], "s3"; got != want {
		t.Fatalf("unexpected restored service name: got %v want %v", got, want)
	}
}
