package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/spf13/cobra"
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
	manager  *runtime.Manager
	storage  Storage
	port     int
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (s *commandServerStub) Start(ctx context.Context) error {
	defer func() {
		if s.finished != nil {
			close(s.finished)
		}
	}()
	if err := s.manager.Serve(ctx, s.port); err != nil {
		return err
	}
	instanceID, err := s.storage.ResolveInstanceIDForPort(s.port)
	if err != nil {
		return err
	}
	if err := s.storage.SaveSavedInstanceWithID(instanceID, s.port); err != nil {
		return err
	}
	if err := s.storage.SaveActiveInstanceWithID(instanceID, s.port); err != nil {
		return err
	}
	if s.started != nil {
		close(s.started)
	}
	if s.release != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.release:
		}
	}
	return nil
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

func TestCommandsServeInstances(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "beta", Version: "v2"}},
	}).Services)

	runCommand := func(args ...string) string {
		t.Helper()

		return executeCommand(t, manager, storage, args...)
	}

	runCommand("serve", "9090")
	runCommand("serve", "8080")

	instancesOutput := stripANSI(runCommand("instances"))
	if got, want := instancesOutput, "Runtime Status\nState: running\n\nServices\n  alpha v1\n  beta v2\n\nInstances\n  8080 running\n  9090 running\n\nPorts\n  8080\n  9090\n"; got != want {
		t.Fatalf("unexpected instances output:\n got %q\nwant %q", got, want)
	}
}

func TestCommandsServeInstancesJSON(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "beta", Version: "v2"}},
	}).Services)
	manager.SetInstanceID("test-instance-json")

	runCommand := func(args ...string) string {
		t.Helper()

		return executeCommand(t, manager, storage, args...)
	}

	runCommand("serve", "9090")
	runCommand("serve", "8080")

	instancesOutput := runCommand("instances", "--json")
	var statusPayload struct {
		State    string `json:"state"`
		Services []struct {
			Name    string   `json:"name"`
			Version string   `json:"version"`
			Tags    []string `json:"tags"`
		} `json:"services"`
		Instances []struct {
			InstanceID string `json:"instanceId"`
			Port       int    `json:"port"`
			Status     string `json:"status"`
			Error      string `json:"error"`
		} `json:"instances"`
		Ports []int `json:"ports"`
	}
	if err := json.Unmarshal([]byte(instancesOutput), &statusPayload); err != nil {
		t.Fatalf("unmarshal instances json: %v\npayload: %s", err, instancesOutput)
	}
	if got, want := statusPayload.State, "running"; got != want {
		t.Fatalf("unexpected status state: got %q want %q", got, want)
	}
	if len(statusPayload.Services) != 2 || statusPayload.Services[0].Name != "alpha" || statusPayload.Services[1].Name != "beta" {
		t.Fatalf("unexpected status services: %#v", statusPayload.Services)
	}
	if len(statusPayload.Ports) != 2 || statusPayload.Ports[0] != 8080 || statusPayload.Ports[1] != 9090 {
		t.Fatalf("unexpected status ports: %#v", statusPayload.Ports)
	}
	if len(statusPayload.Instances) != 2 || statusPayload.Instances[0].Status != "running" || statusPayload.Instances[1].Status != "running" {
		t.Fatalf("unexpected status instances: %#v", statusPayload.Instances)
	}
	// instanceId must be present in the JSON payload
	for _, inst := range statusPayload.Instances {
		if inst.InstanceID == "" {
			t.Fatalf("expected instanceId to be non-empty in JSON payload for port %d", inst.Port)
		}
	}
}

func TestServeDefaultsTo4566AndFallsBackWhenBusy(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
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

		return executeCommand(t, manager, storage, args...)
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

	storage := newTestStorage(t)
	originalListenTCP := listenTCP
	defer func() { listenTCP = originalListenTCP }()

	listenCalls := 0
	listenTCP = func(network, address string) (net.Listener, error) {
		listenCalls++
		if address != ":4566" {
			t.Fatalf("unexpected fallback probe: %s %s", network, address)
		}
		return noopListener{}, nil
	}

	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)

	runCommand := func(args ...string) string {
		t.Helper()

		return executeCommand(t, manager, storage, args...)
	}

	runCommand("serve", "4566")

	if got, want := listenCalls, 1; got != want {
		t.Fatalf("expected exactly one explicit probe, got %d", got)
	}

	ports := manager.Ports(context.Background())
	if len(ports) != 1 || ports[0] != 4566 {
		t.Fatalf("unexpected explicit port: %#v", ports)
	}
}

func TestServeExplicitPortFailsWhenBusy(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	originalListenTCP := listenTCP
	defer func() { listenTCP = originalListenTCP }()

	listenTCP = func(network, address string) (net.Listener, error) {
		if address == ":4566" {
			return nil, errors.New("address already in use")
		}
		t.Fatalf("unexpected probe for %s %s", network, address)
		return nil, errors.New("unexpected probe")
	}

	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)

	factoryCalls := 0
	cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
		factoryCalls++
		t.Fatalf("factory should not be called for a busy explicit port, got %d", port)
		return nil
	})
	cmd.SetArgs([]string{"serve", "4566"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected busy explicit port error")
	}
	if got := err.Error(); !strings.Contains(got, "serve: port 4566 is already in use") {
		t.Fatalf("unexpected error: %v", err)
	}
	if factoryCalls != 0 {
		t.Fatalf("expected no server factory calls, got %d", factoryCalls)
	}
}

func TestCommandsServeDetachedLifecycle(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalDetachedServe := startDetachedServe
	defer func() { startDetachedServe = originalDetachedServe }()

	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	startDetachedServe = func(ctx context.Context, port int, start func(context.Context) error) error {
		errCh := make(chan error, 1)
		go func() {
			errCh <- start(ctx)
		}()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			if containsPort(manager.Ports(context.Background()), port) {
				return nil
			}

			select {
			case err := <-errCh:
				return err
			case <-ticker.C:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
		return &commandServerStub{
			manager:  manager,
			storage:  storage,
			port:     port,
			started:  started,
			release:  release,
			finished: finished,
		}
	})
	cmd.SetArgs([]string{"serve", "--detach", "9090"})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for detached server startup")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("detached serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("detached serve did not return after startup")
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after detached serve: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 9090 || instances[0].Status != "running" {
		t.Fatalf("unexpected detached lifecycle state: %#v", instances)
	}

	ports := manager.Ports(context.Background())
	if len(ports) != 1 || ports[0] != 9090 {
		t.Fatalf("unexpected detached manager ports: %#v", ports)
	}

	close(release)

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for detached server shutdown")
	}
}

func TestCommandsStopAndDeleteLifecycle(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalTerminateProcess := terminateProcessFn
	defer func() { terminateProcessFn = originalTerminateProcess }()
	terminateProcessFn = func(int) error { return nil }

	run := func(args ...string) error {
		t.Helper()

		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve lifecycle seed: %v", err)
	}

	if err := run("stop", "9090"); err != nil {
		t.Fatalf("stop lifecycle: %v", err)
	}

	if ports := manager.Ports(context.Background()); len(ports) != 0 {
		t.Fatalf("expected manager ports to be cleared after stop, got %#v", ports)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after stop: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 9090 || instances[0].Status != "not_started" {
		t.Fatalf("unexpected stop lifecycle state: %#v", instances)
	}

	if err := run("delete", "9090"); err != nil {
		t.Fatalf("delete lifecycle: %v", err)
	}

	instances, err = storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after delete: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("expected registry to be empty after delete, got %#v", instances)
	}

	if ports := manager.Ports(context.Background()); len(ports) != 0 {
		t.Fatalf("expected manager ports to remain empty after delete, got %#v", ports)
	}
}

func TestCommandsDeleteStopsRunningInstanceBeforeRemoval(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalTerminateProcess := terminateProcessFn
	defer func() { terminateProcessFn = originalTerminateProcess }()

	var terminated []int
	terminateProcessFn = func(pid int) error {
		terminated = append(terminated, pid)
		return nil
	}

	run := func(args ...string) error {
		t.Helper()

		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve lifecycle seed: %v", err)
	}

	if err := run("delete", "9090"); err != nil {
		t.Fatalf("delete running instance: %v", err)
	}

	if got, want := len(terminated), 1; got != want {
		t.Fatalf("unexpected termination count: got %d want %d", got, want)
	}
	if got, want := terminated[0], os.Getpid(); got != want {
		t.Fatalf("unexpected terminated pid: got %d want %d", got, want)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after delete: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("expected registry to be empty after delete, got %#v", instances)
	}

	if ports := manager.Ports(context.Background()); len(ports) != 0 {
		t.Fatalf("expected manager ports to be cleared after delete, got %#v", ports)
	}
}

func TestCommandsStopAllStopsEveryRunningInstance(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalTerminateProcess := terminateProcessFn
	defer func() { terminateProcessFn = originalTerminateProcess }()

	var terminated []int
	terminateProcessFn = func(pid int) error {
		terminated = append(terminated, pid)
		return nil
	}

	run := func(args ...string) error {
		t.Helper()

		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve first instance: %v", err)
	}
	if err := run("serve", "8080"); err != nil {
		t.Fatalf("serve second instance: %v", err)
	}

	if err := run("stop", "--all"); err != nil {
		t.Fatalf("stop all instances: %v", err)
	}

	if got, want := len(terminated), 2; got != want {
		t.Fatalf("unexpected termination count: got %d want %d", got, want)
	}

	ports, err := storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports after stop all: %v", err)
	}
	if len(ports) != 0 {
		t.Fatalf("expected no active ports after stop all, got %#v", ports)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after stop all: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("expected two saved instances after stop all, got %#v", instances)
	}
	for _, instance := range instances {
		if instance.Status != "not_started" {
			t.Fatalf("expected stopped instances to be not_started, got %#v", instances)
		}
	}
}

func TestCommandsDeleteAllRemovesEveryInstanceAndResources(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalTerminateProcess := terminateProcessFn
	defer func() { terminateProcessFn = originalTerminateProcess }()
	terminateProcessFn = func(int) error { return nil }

	run := func(args ...string) error {
		t.Helper()

		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve first instance: %v", err)
	}
	if err := run("serve", "8080"); err != nil {
		t.Fatalf("serve second instance: %v", err)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after serve: %v", err)
	}
	for _, instance := range instances {
		instanceRoot := filepath.Join(storage.paths.BaseDir, "instances", instance.InstanceID)
		if err := os.MkdirAll(filepath.Join(instanceRoot, "s3"), 0o755); err != nil {
			t.Fatalf("create instance resource dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(instanceRoot, "s3", "state.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write instance resource payload: %v", err)
		}
	}

	if err := run("delete", "--all"); err != nil {
		t.Fatalf("delete all instances: %v", err)
	}

	instances, err = storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after delete all: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("expected registry to be empty after delete all, got %#v", instances)
	}

	if ports := manager.Ports(context.Background()); len(ports) != 0 {
		t.Fatalf("expected manager ports to be empty after delete all, got %#v", ports)
	}
	if entries, err := os.ReadDir(storage.paths.InstancesDir); err != nil {
		t.Fatalf("read instances dir after delete all: %v", err)
	} else {
		for _, entry := range entries {
			if strings.TrimSpace(entry.Name()) == "active" || strings.TrimSpace(entry.Name()) == "saved" {
				if nested, err := os.ReadDir(filepath.Join(storage.paths.InstancesDir, entry.Name())); err == nil && len(nested) != 0 {
					t.Fatalf("expected %s to be empty after delete all, got %#v", entry.Name(), nested)
				}
			}
		}
	}
}

func TestCommandsServeReusesInstanceIDAfterStop(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalTerminateProcess := terminateProcessFn
	defer func() { terminateProcessFn = originalTerminateProcess }()
	terminateProcessFn = func(int) error { return nil }

	run := func(args ...string) error {
		t.Helper()

		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve lifecycle seed: %v", err)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after first serve: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance after first serve, got %#v", instances)
	}
	firstID := instances[0].InstanceID

	if err := run("stop", "9090"); err != nil {
		t.Fatalf("stop lifecycle: %v", err)
	}
	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve lifecycle restart: %v", err)
	}

	instances, err = storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after restart: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance after restart, got %#v", instances)
	}
	if got, want := instances[0].InstanceID, firstID; got != want {
		t.Fatalf("expected serve to reuse instance id: got %q want %q", got, want)
	}
}

func TestCommandsDeleteRemovesInstanceResources(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	originalTerminateProcess := terminateProcessFn
	defer func() { terminateProcessFn = originalTerminateProcess }()
	terminateProcessFn = func(int) error { return nil }

	run := func(args ...string) error {
		t.Helper()

		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("serve", "9090"); err != nil {
		t.Fatalf("serve lifecycle seed: %v", err)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after serve: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}

	instanceRoot := filepath.Join(storage.paths.BaseDir, "instances", instances[0].InstanceID)
	if err := os.MkdirAll(filepath.Join(instanceRoot, "s3"), 0o755); err != nil {
		t.Fatalf("create instance resource dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(instanceRoot, "s3", "state.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write instance resource payload: %v", err)
	}

	if err := run("delete", "9090"); err != nil {
		t.Fatalf("delete instance with resources: %v", err)
	}

	if _, err := os.Stat(instanceRoot); !os.IsNotExist(err) {
		t.Fatalf("expected instance resources to be deleted, got err=%v", err)
	}
}

func TestCommandsRejectMissingInstanceLifecycleTargets(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(nil)

	for _, args := range [][]string{{"stop"}, {"delete"}} {
		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("expected error for %v", args)
		}
		if got := err.Error(); !strings.Contains(got, "requires a port or --all") {
			t.Fatalf("unexpected missing target error for %v: %v", args, err)
		}
	}
}

func TestCommandsRejectAllWithPortTargets(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(nil)

	for _, args := range [][]string{{"stop", "9090", "--all"}, {"delete", "9090", "--all"}} {
		cmd, _, _ := newTestCommand(t, manager, storage, func(port int) HTTPServer {
			return &commandServerStub{manager: manager, storage: storage, port: port}
		})
		cmd.SetArgs(args)

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("expected error for %v", args)
		}
		if got := err.Error(); !strings.Contains(got, "--all cannot be combined with a port") {
			t.Fatalf("unexpected all-with-port error for %v: %v", args, err)
		}
	}
}

func TestCommandsRenderEmptyRuntimeStatus(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(nil)
	statusOutput := executeCommand(t, manager, storage, "instances")

	if got, want := stripANSI(statusOutput), "Runtime Status\nState: not_started\n\nServices\n  (none)\n\nInstances\n  (none)\n\nPorts\n  (none)\n"; got != want {
		t.Fatalf("unexpected empty status output:\n got %q\nwant %q", got, want)
	}
}

func TestCommandsRenderErroredInstanceStatus(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)

	if err := storage.SaveSavedInstance(8080); err != nil {
		t.Fatalf("save saved instance: %v", err)
	}
	if err := storage.SaveErroredInstance(8080, errors.New("failed to start")); err != nil {
		t.Fatalf("save errored instance: %v", err)
	}

	statusOutput := stripANSI(executeCommand(t, manager, storage, "instances"))
	if got, want := statusOutput, "Runtime Status\nState: errored\n\nServices\n  alpha v1\n\nInstances\n  8080 errored: failed to start\n\nPorts\n  (none)\n"; got != want {
		t.Fatalf("unexpected errored status output:\n got %q\nwant %q", got, want)
	}
}

func executeCommand(t *testing.T, manager *runtime.Manager, storage Storage, args ...string) string {
	t.Helper()

	cmd, stdout, stderr := newTestCommand(t, manager, storage, func(port int) HTTPServer {
		return &commandServerStub{manager: manager, storage: storage, port: port}
	})
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v\nstderr: %s", args, err, stderr.String())
	}

	return stdout.String()
}

func newTestCommand(t *testing.T, manager *runtime.Manager, storage Storage, factory HTTPServerFactory) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := NewRootCommand(stdout, stderr, Commands{
		Serve:     NewServeCommand(manager, factory),
		Instances: NewInstancesCommand(manager, storage),
		Status:    NewStatusCommand(manager, storage),
		Stop:      NewStopCommand(manager, storage),
		Delete:    NewDeleteCommand(manager, storage),
	})
	return cmd, stdout, stderr
}

func newTestStorage(t *testing.T) Storage {
	t.Helper()

	homeDir := t.TempDir()
	configDir := filepath.Join(t.TempDir(), "config")
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	return NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func containsPort(ports []int, port int) bool {
	for _, existing := range ports {
		if existing == port {
			return true
		}
	}
	return false
}

func TestStatusAliasMatchesInstancesOutput(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	manager.SetInstanceID("test-alias-instance")

	runCommand := func(args ...string) string {
		t.Helper()
		return executeCommand(t, manager, storage, args...)
	}

	runCommand("serve", "9090")

	instancesOut := stripANSI(runCommand("instances"))
	statusOut := stripANSI(runCommand("status"))

	if instancesOut != statusOut {
		t.Fatalf("status alias output does not match instances output:\ninstances: %q\nstatus:    %q", instancesOut, statusOut)
	}
}

func TestStatusAliasJSONMatchesInstancesJSON(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)
	manager.SetInstanceID("test-alias-json-instance")

	runCommand := func(args ...string) string {
		t.Helper()
		return executeCommand(t, manager, storage, args...)
	}

	runCommand("serve", "8080")

	instancesJSON := runCommand("instances", "--json")
	statusJSON := runCommand("status", "--json")

	if instancesJSON != statusJSON {
		t.Fatalf("status alias JSON does not match instances JSON:\ninstances: %s\nstatus:    %s", instancesJSON, statusJSON)
	}
}

func TestCommandsServeInstancesJSONIncludesInstanceID(t *testing.T) {
	t.Helper()

	storage := newTestStorage(t)
	manager := runtime.New(composition.Assemble([]orchestrator.Service{
		&commandServiceStub{metadata: orchestrator.Metadata{Name: "alpha", Version: "v1"}},
	}).Services)

	executeCommand(t, manager, storage, "serve", "7777")

	out := executeCommand(t, manager, storage, "instances", "--json")

	var payload struct {
		Instances []struct {
			InstanceID string `json:"instanceId"`
			Port       int    `json:"port"`
			Status     string `json:"status"`
		} `json:"instances"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal: %v\npayload: %s", err, out)
	}
	if len(payload.Instances) != 1 {
		t.Fatalf("expected one instance, got %d", len(payload.Instances))
	}
	if payload.Instances[0].InstanceID == "" {
		t.Fatal("expected non-empty instanceId")
	}
	if strings.HasPrefix(payload.Instances[0].InstanceID, "mildstack-") {
		t.Fatalf("expected non-legacy instanceId, got %q", payload.Instances[0].InstanceID)
	}
	if got, want := payload.Instances[0].Port, 7777; got != want {
		t.Fatalf("unexpected port: got %d want %d", got, want)
	}
	if got, want := payload.Instances[0].Status, "running"; got != want {
		t.Fatalf("unexpected status: got %q want %q", got, want)
	}
}
