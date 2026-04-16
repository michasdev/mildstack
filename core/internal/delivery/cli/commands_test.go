package cli

import (
	"bytes"
	"context"
	"strings"
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
	}))

	runCommand := func(args ...string) string {
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

	runCommand("serve", "--port", "9090")
	runCommand("serve", "--port", "8080")

	statusOutput := runCommand("status")
	if !strings.Contains(statusOutput, "alpha") || !strings.Contains(statusOutput, "beta") {
		t.Fatalf("status output missing service metadata: %q", statusOutput)
	}
	if !strings.Contains(statusOutput, "9090") || !strings.Contains(statusOutput, "8080") {
		t.Fatalf("status output missing ports: %q", statusOutput)
	}
	if strings.Index(statusOutput, "8080") > strings.Index(statusOutput, "9090") {
		t.Fatalf("status ports not ordered ascending: %q", statusOutput)
	}

	portsOutput := runCommand("ports")
	if got, want := strings.TrimSpace(portsOutput), "8080\n9090"; got != want {
		t.Fatalf("unexpected ports output: got %q want %q", got, want)
	}
}
