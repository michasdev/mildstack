package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
)

var _ orchestrator.Service = (*commandServiceStub)(nil)

type commandServiceStub struct {
	metadata orchestrator.Metadata
}

func (s *commandServiceStub) Start(context.Context) error { return nil }

func (s *commandServiceStub) Stop(context.Context) error { return nil }

func (s *commandServiceStub) Metadata() orchestrator.Metadata { return s.metadata }

func (s *commandServiceStub) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, nil, nil, "cli-test")
}

func (s *commandServiceStub) RegisterRoutes(orchestrator.RouteRegistrar) error { return nil }

func (s *commandServiceStub) AttachState(orchestrator.StateHook) error { return nil }

type commandServerStub struct {
	manager *runtime.Manager
	port    int
}

func (s *commandServerStub) Start(ctx context.Context) error {
	return s.manager.Serve(ctx, s.port)
}

func TestCommandsServeStatusAndPorts(t *testing.T) {
	t.Helper()

	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "beta", Version: "v2"}},
	}).Services)

	runCommand := func(args ...string) string {
		t.Helper()

		return executeCommand(t, manager, args...)
	}

	runCommand("serve", "--port", "9090")
	runCommand("serve", "--port", "8080")

	statusOutput := runCommand("status")
	if got, want := statusOutput, "Runtime Status\nState: ready\n\nServices\n  alpha v1\n  beta v2\n\nPorts\n  8080\n  9090\n"; got != want {
		t.Fatalf("unexpected status output:\n got %q\nwant %q", got, want)
	}

	portsOutput := runCommand("ports")
	if got, want := portsOutput, "8080\n9090\n"; got != want {
		t.Fatalf("unexpected ports output: got %q want %q", got, want)
	}
}

func TestCommandsRenderEmptyRuntimeStatus(t *testing.T) {
	t.Helper()

	manager := runtime.New(nil)
	statusOutput := executeCommand(t, manager, "status")

	if got, want := statusOutput, "Runtime Status\nState: not_ready\n\nServices\n  (none)\n\nPorts\n  (none)\n"; got != want {
		t.Fatalf("unexpected empty status output:\n got %q\nwant %q", got, want)
	}
}

func executeCommand(t *testing.T, manager *runtime.Manager, args ...string) string {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := NewRootCommand(stdout, stderr, Commands{
		Serve: NewServeCommand(manager, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, port: port}
		}),
		Status: NewStatusCommand(manager),
		Ports:  NewPortsCommand(manager),
	})
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v\nstderr: %s", args, err, stderr.String())
	}

	return stdout.String()
}
