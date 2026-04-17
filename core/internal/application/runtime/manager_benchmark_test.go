package runtime

import (
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

var (
	benchmarkSnapshotSink Snapshot
	benchmarkPortsSink    []int
)

func BenchmarkManager(b *testing.B) {
	manager := newBenchmarkManager()
	ctx := context.Background()

	b.Run("Snapshot", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkSnapshotSink = manager.Snapshot(ctx)
		}
	})

	b.Run("Ports", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkPortsSink = manager.Ports(ctx)
		}
	})
}

func newBenchmarkManager() *Manager {
	services := []orchestrator.Service{
		&benchmarkServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
		&benchmarkServiceStub{metadata: orchestrator.Metadata{Name: "beta", Version: "v2"}},
	}

	manager := New(services)
	_ = manager.Serve(context.Background(), 9090)
	_ = manager.Serve(context.Background(), 8080)
	return manager
}

type benchmarkServiceStub struct {
	metadata orchestrator.Metadata
}

func (s *benchmarkServiceStub) Start(context.Context) error { return nil }

func (s *benchmarkServiceStub) Stop(context.Context) error { return nil }

func (s *benchmarkServiceStub) Metadata() orchestrator.Metadata { return s.metadata }

func (s *benchmarkServiceStub) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, nil, nil, "runtime-benchmark")
}

func (s *benchmarkServiceStub) RegisterRoutes(orchestrator.RouteRegistrar) error { return nil }

func (s *benchmarkServiceStub) AttachState(orchestrator.StateHook) error { return nil }
