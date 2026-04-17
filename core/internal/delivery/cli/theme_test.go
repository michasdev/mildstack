package cli

import (
	"errors"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

func TestRenderStatusUsesStructuredTheme(t *testing.T) {
	t.Helper()

	presenter := NewPresenter(runtime.Snapshot{
		Services: []orchestrator.Metadata{
			{Name: "alpha", Version: "v1"},
			{Name: "beta", Version: "v2"},
		},
		Ports: []int{8080, 9090},
	})

	if got, want := stripANSI(RenderStatus(DefaultTheme(), presenter)), "Runtime Status\nState: ready\n\nServices\n  alpha v1\n  beta v2\n\nPorts\n  8080\n  9090\n"; got != want {
		t.Fatalf("unexpected status render:\n got %q\nwant %q", got, want)
	}
}

func TestRenderEmptyStatusAndPlainPorts(t *testing.T) {
	t.Helper()

	presenter := NewPresenter(runtime.Snapshot{})

	if got, want := stripANSI(RenderStatus(DefaultTheme(), presenter)), "Runtime Status\nState: not_ready\n\nServices\n  (none)\n\nPorts\n  (none)\n"; got != want {
		t.Fatalf("unexpected empty status render:\n got %q\nwant %q", got, want)
	}
	if got, want := stripANSI(RenderPorts(DefaultTheme(), presenter)), "No ports registered\n"; got != want {
		t.Fatalf("unexpected empty ports render:\n got %q\nwant %q", got, want)
	}
	if got, want := stripANSI(RenderReadiness(DefaultTheme(), presenter)), "State: not_ready"; got != want {
		t.Fatalf("unexpected readiness render:\n got %q\nwant %q", got, want)
	}
	if got, want := RenderError(DefaultTheme(), errors.New("boom")), "error: boom"; got != want {
		t.Fatalf("unexpected error render:\n got %q\nwant %q", got, want)
	}
	if got, want := RenderError(DefaultTheme(), errors.New("   ")), "error"; got != want {
		t.Fatalf("unexpected blank error render:\n got %q\nwant %q", got, want)
	}
}
