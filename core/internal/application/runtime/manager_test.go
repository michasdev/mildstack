package runtime

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

var _ orchestrator.Service = (*serviceStub)(nil)

type serviceStub struct {
	metadata orchestrator.Metadata
}

func (s *serviceStub) Start(context.Context) error { return nil }

func (s *serviceStub) Stop(context.Context) error { return nil }

func (s *serviceStub) Metadata() orchestrator.Metadata { return s.metadata }

func (s *serviceStub) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, nil, nil, "runtime-test")
}

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
	manager := New(services)

	services[0] = second
	first.metadata.Name = "mutated"
	first.metadata.Tags[0] = "changed"

	if err := manager.Serve(context.Background(), 8080); err != nil {
		t.Fatalf("start 8080: %v", err)
	}
	if err := manager.Serve(context.Background(), 9090); err != nil {
		t.Fatalf("start 9090: %v", err)
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

	snapshot.Services[0].Name = "mutated"
	snapshot.Services[0].Tags[0] = "changed"
	snapshot.Ports[0] = 9999

	again := manager.Snapshot(context.Background())
	if got, want := again.Services[0].Name, "alpha"; got != want {
		t.Fatalf("unexpected restored first service name: got %q want %q", got, want)
	}
	if got, want := again.Services[0].Tags[0], "core"; got != want {
		t.Fatalf("unexpected restored first service tag: got %q want %q", got, want)
	}
	if got, want := again.Ports, []int{8080, 9090}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected restored ports: got %v want %v", got, want)
	}

	ports := manager.Ports(context.Background())
	if len(ports) != 2 || ports[0] != 8080 || ports[1] != 9090 {
		t.Fatalf("unexpected ports snapshot: %v", ports)
	}

	ports[0] = 7777
	againPorts := manager.Ports(context.Background())
	if len(againPorts) != 2 || againPorts[0] != 8080 || againPorts[1] != 9090 {
		t.Fatalf("unexpected restored ports snapshot: %v", againPorts)
	}
}

func TestManagerRejectsDuplicatePorts(t *testing.T) {
	t.Helper()

	manager := New(nil)

	if err := manager.Serve(context.Background(), 8080); err != nil {
		t.Fatalf("start 8080: %v", err)
	}
	if err := manager.Serve(context.Background(), 8080); err == nil {
		t.Fatal("expected duplicate port error")
	}
}

func TestNewWithPortsSeedsSnapshot(t *testing.T) {
	t.Helper()

	manager := NewWithPorts(nil, []int{9090, 8080})
	snapshot := manager.Snapshot(context.Background())

	if got, want := snapshot.Ports, []int{8080, 9090}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected seeded ports: %v", got)
	}

	snapshot.Ports[0] = 7777
	again := manager.Snapshot(context.Background())
	if got, want := again.Ports, []int{8080, 9090}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected restored seeded ports: %v", got)
	}
}

func TestManagerInstanceCarriesInstanceID(t *testing.T) {
	t.Helper()

	manager := New(nil)
	manager.SetInstanceID("test-instance-abc")
	if err := manager.Serve(context.Background(), 8080); err != nil {
		t.Fatalf("start 8080: %v", err)
	}

	snapshot := manager.Snapshot(context.Background())
	if len(snapshot.Instances) != 1 {
		t.Fatalf("expected one instance, got %d", len(snapshot.Instances))
	}
	if got, want := snapshot.Instances[0].InstanceID, "test-instance-abc"; got != want {
		t.Fatalf("unexpected instance InstanceID: got %q want %q", got, want)
	}
	if snapshot.Instances[0].Port != 8080 {
		t.Fatalf("unexpected instance port: got %d want 8080", snapshot.Instances[0].Port)
	}
}

func TestNewWithPortsSeedsInstanceIDFromRegisteredIdentity(t *testing.T) {
	t.Helper()

	manager := NewWithPorts(nil, []int{9090, 8080})
	manager.SetInstanceID("test-instance-seed")
	snapshot := manager.Snapshot(context.Background())

	if len(snapshot.Instances) != 2 {
		t.Fatalf("expected two instances, got %d", len(snapshot.Instances))
	}
	for _, instance := range snapshot.Instances {
		if instance.InstanceID == "" {
			t.Fatalf("expected instance on port %d to carry InstanceID", instance.Port)
		}
	}

	// mutation of snapshot instances must not affect the canonical identity
	snapshot.Instances[0].InstanceID = "mutated"
	again := manager.Snapshot(context.Background())
	for _, instance := range again.Instances {
		if instance.InstanceID != "test-instance-seed" {
			t.Fatalf("expected InstanceID to survive mutation, got %q on port %d", instance.InstanceID, instance.Port)
		}
	}
}
