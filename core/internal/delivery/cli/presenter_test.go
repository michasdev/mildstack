package cli

import (
	"errors"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

func TestPresenterCopiesSnapshotData(t *testing.T) {
	t.Helper()

	snapshot := runtime.Snapshot{
		Services: []orchestrator.Metadata{
			{Name: "alpha", Version: "v1"},
		},
		Ports: []int{8080},
	}

	presenter := NewPresenter(snapshot)

	snapshot.Services[0].Name = "changed"
	snapshot.Services[0].Version = "v2"
	snapshot.Services[0].Tags = append(snapshot.Services[0].Tags, "mutated")
	snapshot.Ports[0] = 9090

	if got, want := presenter.PresentStatus(), "Services:\n- alpha v1\nPorts:\n- 8080\n"; got != want {
		t.Fatalf("unexpected copied status output:\n got %q\nwant %q", got, want)
	}
	if got, want := presenter.PresentPorts(), "8080\n"; got != want {
		t.Fatalf("unexpected copied ports output:\n got %q\nwant %q", got, want)
	}
	if got, want := presenter.PresentReadiness(), "ready"; got != want {
		t.Fatalf("unexpected copied readiness output:\n got %q\nwant %q", got, want)
	}
}

func TestPresenterRendersEmptyAndErrorStates(t *testing.T) {
	t.Helper()

	if got, want := PresentStatus(runtime.Snapshot{}), "Services:\n  (none)\nPorts:\n  (none)\n"; got != want {
		t.Fatalf("unexpected empty status output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentPorts(nil), "No ports registered\n"; got != want {
		t.Fatalf("unexpected empty ports output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentReadiness(runtime.Snapshot{}), "not_ready"; got != want {
		t.Fatalf("unexpected empty readiness output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentError(nil), ""; got != want {
		t.Fatalf("unexpected nil error output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentError(errors.New("boom")), "error: boom"; got != want {
		t.Fatalf("unexpected error output:\n got %q\nwant %q", got, want)
	}
}
