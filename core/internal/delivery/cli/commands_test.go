package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"regexp"
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

type noopListener struct{}

func (noopListener) Accept() (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (noopListener) Close() error { return nil }

func (noopListener) Addr() net.Addr { return noopAddr("noop") }

type noopAddr string

func (a noopAddr) Network() string { return string(a) }

func (a noopAddr) String() string { return string(a) }

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

	statusOutput := stripANSI(runCommand("status"))
	if got, want := statusOutput, "Runtime Status\nState: ready\n\nServices\n  alpha v1\n  beta v2\n\nPorts\n  8080\n  9090\n"; got != want {
		t.Fatalf("unexpected status output:\n got %q\nwant %q", got, want)
	}

	portsOutput := runCommand("ports")
	if got, want := portsOutput, "8080\n9090\n"; got != want {
		t.Fatalf("unexpected ports output: got %q want %q", got, want)
	}
}

func TestCommandsServeStatusAndPortsJSON(t *testing.T) {
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

	statusOutput := runCommand("status", "--json")
	var statusPayload struct {
		State    string `json:"state"`
		Services []struct {
			Name    string   `json:"name"`
			Version string   `json:"version"`
			Tags    []string `json:"tags"`
		} `json:"services"`
		Ports []int `json:"ports"`
	}
	if err := json.Unmarshal([]byte(statusOutput), &statusPayload); err != nil {
		t.Fatalf("unmarshal status json: %v\npayload: %s", err, statusOutput)
	}
	if got, want := statusPayload.State, "ready"; got != want {
		t.Fatalf("unexpected status state: got %q want %q", got, want)
	}
	if len(statusPayload.Services) != 2 || statusPayload.Services[0].Name != "alpha" || statusPayload.Services[1].Name != "beta" {
		t.Fatalf("unexpected status services: %#v", statusPayload.Services)
	}
	if len(statusPayload.Ports) != 2 || statusPayload.Ports[0] != 8080 || statusPayload.Ports[1] != 9090 {
		t.Fatalf("unexpected status ports: %#v", statusPayload.Ports)
	}

	portsOutput := runCommand("ports", "--json")
	var portsPayload struct {
		Ports []int `json:"ports"`
	}
	if err := json.Unmarshal([]byte(portsOutput), &portsPayload); err != nil {
		t.Fatalf("unmarshal ports json: %v\npayload: %s", err, portsOutput)
	}
	if len(portsPayload.Ports) != 2 || portsPayload.Ports[0] != 8080 || portsPayload.Ports[1] != 9090 {
		t.Fatalf("unexpected ports payload: %#v", portsPayload.Ports)
	}
}

func TestServeDefaultsTo4566AndFallsBackWhenBusy(t *testing.T) {
	t.Helper()

	originalListenTCP := listenTCP
	defer func() { listenTCP = originalListenTCP }()

	listenCalls := 0
	listenTCP = func(network, address string) (net.Listener, error) {
		listenCalls++
		switch address {
		case ":4566", ":4567":
			return nil, errors.New("address already in use")
		case ":4568":
			return noopListener{}, nil
		default:
			t.Fatalf("unexpected port probe: %s %s", network, address)
			return nil, errors.New("unexpected port probe")
		}
	}

	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)

	runCommand := func(args ...string) string {
		t.Helper()

		return executeCommand(t, manager, args...)
	}

	runCommand("serve")

	if got, want := listenCalls, 3; got != want {
		t.Fatalf("unexpected probe count: got %d want %d", got, want)
	}

	ports := manager.Ports(context.Background())
	if len(ports) != 1 || ports[0] != 4568 {
		t.Fatalf("unexpected resolved port: %#v", ports)
	}
}

func TestServeExplicitPortSkipsFallback(t *testing.T) {
	t.Helper()

	originalListenTCP := listenTCP
	defer func() { listenTCP = originalListenTCP }()

	listenCalls := 0
	listenTCP = func(network, address string) (net.Listener, error) {
		listenCalls++
		return nil, errors.New("listen should not be called for explicit ports")
	}

	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)

	runCommand := func(args ...string) string {
		t.Helper()

		return executeCommand(t, manager, args...)
	}

	runCommand("serve", "--port", "4566")

	if got, want := listenCalls, 0; got != want {
		t.Fatalf("expected no port probe for explicit flag, got %d", got)
	}

	ports := manager.Ports(context.Background())
	if len(ports) != 1 || ports[0] != 4566 {
		t.Fatalf("unexpected explicit port: %#v", ports)
	}
}

func TestCommandsRenderEmptyRuntimeStatus(t *testing.T) {
	t.Helper()

	manager := runtime.New(nil)
	statusOutput := executeCommand(t, manager, "status")

	if got, want := stripANSI(statusOutput), "Runtime Status\nState: not_ready\n\nServices\n  (none)\n\nPorts\n  (none)\n"; got != want {
		t.Fatalf("unexpected empty status output:\n got %q\nwant %q", got, want)
	}
}

func TestCommandsRenderEmptyPortsState(t *testing.T) {
	t.Helper()

	manager := runtime.New(nil)
	portsOutput := stripANSI(executeCommand(t, manager, "ports"))

	if got, want := portsOutput, "No ports registered\n"; got != want {
		t.Fatalf("unexpected empty ports output:\n got %q\nwant %q", got, want)
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

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
