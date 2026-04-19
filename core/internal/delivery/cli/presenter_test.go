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

	if got, want := presenter.PresentStatus(), "State: running\n\nServices:\n- alpha v1\n\nInstances:\n- 8080 running\nPorts:\n- 8080\n"; got != want {
		t.Fatalf("unexpected copied status output:\n got %q\nwant %q", got, want)
	}
	if got, want := presenter.PresentPorts(), "8080\n"; got != want {
		t.Fatalf("unexpected copied ports output:\n got %q\nwant %q", got, want)
	}
	if got, want := presenter.PresentReadiness(), "running"; got != want {
		t.Fatalf("unexpected copied readiness output:\n got %q\nwant %q", got, want)
	}
}

func TestPresenterRendersEmptyAndErrorStates(t *testing.T) {
	t.Helper()

	if got, want := PresentStatus(runtime.Snapshot{}), "State: not_started\n\nServices:\n  (none)\n\nInstances:\n  (none)\nPorts:\n  (none)\n"; got != want {
		t.Fatalf("unexpected empty status output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentPorts(nil), "No ports registered\n"; got != want {
		t.Fatalf("unexpected empty ports output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentReadiness(runtime.Snapshot{}), "not_started"; got != want {
		t.Fatalf("unexpected empty readiness output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentError(nil), ""; got != want {
		t.Fatalf("unexpected nil error output:\n got %q\nwant %q", got, want)
	}
	if got, want := PresentError(errors.New("boom")), "error: boom"; got != want {
		t.Fatalf("unexpected error output:\n got %q\nwant %q", got, want)
	}
}

func TestPresenterStatusPayloadIncludesInstanceID(t *testing.T) {
	t.Helper()

	snapshot := runtime.Snapshot{
		Instances: []runtime.Instance{
			{InstanceID: "inst-xyz", Port: 8080, PID: 1234, Status: "running"},
		},
		Ports: []int{8080},
	}

	presenter := NewPresenter(snapshot)
	payload := presenter.StatusPayload()

	if len(payload.Instances) != 1 {
		t.Fatalf("expected one instance in payload, got %d", len(payload.Instances))
	}
	if got, want := payload.Instances[0].InstanceID, "inst-xyz"; got != want {
		t.Fatalf("unexpected instanceId in payload: got %q want %q", got, want)
	}
	if got, want := payload.Instances[0].Port, 8080; got != want {
		t.Fatalf("unexpected port in payload: got %d want %d", got, want)
	}
	if got, want := payload.Instances[0].Status, "running"; got != want {
		t.Fatalf("unexpected status in payload: got %q want %q", got, want)
	}

	// mutation of the snapshot must not affect the already-built presenter
	snapshot.Instances[0].InstanceID = "mutated"
	payload2 := presenter.StatusPayload()
	if got, want := payload2.Instances[0].InstanceID, "inst-xyz"; got != want {
		t.Fatalf("payload must be copy-safe: got %q want %q", got, want)
	}
}
