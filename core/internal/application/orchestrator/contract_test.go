package orchestrator

import (
	"context"
	"testing"
)

var _ Service = (*fakeService)(nil)

type fakeService struct {
	metadata Metadata
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

