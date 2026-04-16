package runtime

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/composition"
)

var _ orchestrator.Service = (*serviceStub)(nil)

type serviceStub struct {
	metadata orchestrator.Metadata
}

func (s *serviceStub) Start(context.Context) error { return nil }

func (s *serviceStub) Stop(context.Context) error { return nil }

func (s *serviceStub) Metadata() orchestrator.Metadata { return s.metadata }

func (s *serviceStub) RegisterRoutes(orchestrator.RouteRegistrar) error { return nil }

func (s *serviceStub) AttachState(orchestrator.StateHook) error { return nil }

func TestManagerCopiesMetadataAndTracksMultiplePorts(t *testing.T) {
	t.Helper()

	first := &serviceStub{
		metadata: orchestrator.Metadata{
			Name:        "alpha",
			Description: "first service",
			Version:     "v1",
			Tags:        []string{"core", "alpha"},
		},
	}
	second := &serviceStub{
		metadata: orchestrator.Metadata{
			Name:        "beta",
			Description: "second service",
			Version:     "v2",
			Tags:        []string{"core", "beta"},
		},
	}

	services := []orchestrator.Service{first, second}
	manager := New(composition.Assemble(services))

	services[0] = second
	first.metadata.Name = "mutated"
	first.metadata.Tags[0] = "changed"

	if err := manager.Serve(context.Background(), 8080); err != nil {
		t.Fatalf("serve 8080: %v", err)
	}
	if err := manager.Serve(context.Background(), 9090); err != nil {
		t.Fatalf("serve 9090: %v", err)
	}

	snapshot := manager.Snapshot(context.Background())
	if got, want := len(snapshot.Services), 2; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}
	if got, want := snapshot.Services[0].Name, "alpha"; got != want {
		t.Fatalf("unexpected first service name: got %q want %q", got, want)
	}
	if got, want := snapshot.Services[0].Tags[0], "core"; got != want {
		t.Fatalf("unexpected first service tag: got %q want %q", got, want)
	}
	if got, want := snapshot.Services[1].Name, "beta"; got != want {
		t.Fatalf("unexpected second service name: got %q want %q", got, want)
	}
	if got, want := snapshot.Ports, []int{8080, 9090}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected ports: got %v want %v", got, want)
	}

	ports := manager.Ports(context.Background())
	if len(ports) != 2 || ports[0] != 8080 || ports[1] != 9090 {
		t.Fatalf("unexpected ports snapshot: %v", ports)
	}
}

func TestManagerRejectsDuplicatePorts(t *testing.T) {
	t.Helper()

	manager := New(composition.Root{})

	if err := manager.Serve(context.Background(), 8080); err != nil {
		t.Fatalf("serve 8080: %v", err)
	}
	if err := manager.Serve(context.Background(), 8080); err == nil {
		t.Fatal("expected duplicate port error")
	}
}
