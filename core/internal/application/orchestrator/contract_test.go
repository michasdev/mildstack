package orchestrator

import (
	"context"
	"strings"
	"testing"
)

var _ Service = (*fakeService)(nil)

type fakeService struct {
	metadata Metadata
	policy   EmulationPolicy
}

func (f *fakeService) Start(context.Context) error {
	return nil
}

func (f *fakeService) Stop(context.Context) error {
	return nil
}

func (f *fakeService) Metadata() Metadata {
	return f.metadata
}

func (f *fakeService) Policy() EmulationPolicy {
	return f.policy.Clone()
}

func (f *fakeService) RegisterRoutes(reg RouteRegistrar) error {
	return reg.Register(Route{Method: "GET", Path: "/health", Name: "health"})
}

func (f *fakeService) AttachState(hook StateHook) error {
	hook.Set("service", f.metadata.Name)
	return nil
}

type fakeRegistrar struct {
	routes []Route
}

func (r *fakeRegistrar) Register(route Route) error {
	r.routes = append(r.routes, route)
	return nil
}

type fakeStateHook struct {
	values map[string]any
}

func (h *fakeStateHook) Set(key string, value any) {
	if h.values == nil {
		h.values = make(map[string]any)
	}
	h.values[key] = value
}

func (h *fakeStateHook) Get(key string) (any, bool) {
	value, ok := h.values[key]
	return value, ok
}

func TestFakeServiceImplementsContract(t *testing.T) {
	t.Helper()

	service := &fakeService{
		metadata: Metadata{
			Name:        "fake",
			Description: "test service",
			Version:     "v1",
			Tags:        []string{"core"},
		},
		policy: NewEmulationPolicy(
			FidelityExemplar,
			[]string{"list health"},
			[]string{"write health"},
			"fake",
		),
	}

	policy := service.Policy()
	if got, want := policy.Fidelity, FidelityExemplar; got != want {
		t.Fatalf("unexpected fidelity: got %q want %q", got, want)
	}
	if got, want := policy.ErrorPrefix, "fake"; got != want {
		t.Fatalf("unexpected error prefix: got %q want %q", got, want)
	}
	if got, want := len(policy.Supported), 1; got != want {
		t.Fatalf("unexpected supported count: got %d want %d", got, want)
	}
	if got, want := len(policy.Unsupported), 1; got != want {
		t.Fatalf("unexpected unsupported count: got %d want %d", got, want)
	}

	policy.Supported[0] = "changed"
	policy.Unsupported[0] = "changed"

	again := service.Policy()
	if got, want := again.Supported[0], "list health"; got != want {
		t.Fatalf("policy supported slice was not copied: got %q want %q", got, want)
	}
	if got, want := again.Unsupported[0], "write health"; got != want {
		t.Fatalf("policy unsupported slice was not copied: got %q want %q", got, want)
	}

	err := UnsupportedError(again, "DeleteHealth")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if got, want := err.Error(), "fake: unsupported operation DeleteHealth"; got != want {
		t.Fatalf("unexpected unsupported error: got %q want %q", got, want)
	}

	registrar := &fakeRegistrar{}
	if err := service.RegisterRoutes(registrar); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	if len(registrar.routes) != 1 {
		t.Fatalf("expected one route, got %d", len(registrar.routes))
	}
	if registrar.routes[0].Name != "health" {
		t.Fatalf("unexpected route name: %s", registrar.routes[0].Name)
	}

	state := &fakeStateHook{}
	if err := service.AttachState(state); err != nil {
		t.Fatalf("attach state: %v", err)
	}
	value, ok := state.Get("service")
	if !ok {
		t.Fatal("expected state value to be set")
	}
	if value != "fake" {
		t.Fatalf("unexpected state value: %v", value)
	}
}

func TestNewEmulationPolicyCopiesInputSlices(t *testing.T) {
	t.Helper()

	supported := []string{"read"}
	unsupported := []string{"write"}

	policy := NewEmulationPolicy(FidelityPartial, supported, unsupported, "fake")
	supported[0] = "changed"
	unsupported[0] = "changed"

	if got, want := policy.Supported[0], "read"; got != want {
		t.Fatalf("supported slice was not copied: got %q want %q", got, want)
	}
	if got, want := policy.Unsupported[0], "write"; got != want {
		t.Fatalf("unsupported slice was not copied: got %q want %q", got, want)
	}
}

func TestUnsupportedErrorUsesPolicyPrefix(t *testing.T) {
	t.Helper()

	err := UnsupportedError(EmulationPolicy{ErrorPrefix: "fake"}, "DeleteHealth")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if !strings.HasPrefix(err.Error(), "fake:") {
		t.Fatalf("unexpected unsupported error prefix: %q", err.Error())
	}
}
