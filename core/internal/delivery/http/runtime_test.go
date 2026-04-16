package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

func TestRuntimeInfoEndpointReturnsSnapshot(t *testing.T) {
	t.Helper()

	snapshot := runtime.Snapshot{
		Services: []orchestrator.Metadata{
			{
				Name:        "alpha",
				Description: "first service",
				Version:     "v1",
				Tags:        []string{"core", "alpha"},
			},
		},
		Ports: []int{8080, 9090},
	}
	router := NewRouter(DefaultConfig(), snapshotStub{snapshot: snapshot})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/info", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}

	var response runtimeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal runtime response: %v", err)
	}
	if got, want := len(response.Services), 1; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}
	if got, want := response.Services[0].Name, "alpha"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := response.Services[0].Tags, []string{"core", "alpha"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected service tags: got %v want %v", got, want)
	}
	if got, want := response.Ports, []int{8080, 9090}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected ports: got %v want %v", got, want)
	}
}

func TestRuntimeInfoResponseCopiesSourceSlices(t *testing.T) {
	t.Helper()

	services := []orchestrator.Metadata{
		{
			Name:        "alpha",
			Description: "first service",
			Version:     "v1",
			Tags:        []string{"core"},
		},
	}
	snapshot := runtime.Snapshot{
		Services: services,
		Ports:    []int{8080},
	}

	response := runtimeResponse{
		Services: copyRuntimeServices(snapshot.Services),
		Ports:    copyRuntimePorts(snapshot.Ports),
	}

	services[0].Name = "mutated"
	services[0].Tags[0] = "changed"
	snapshot.Ports[0] = 9090

	rendered, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal runtime response: %v", err)
	}

	var copied runtimeResponse
	if err := json.Unmarshal(rendered, &copied); err != nil {
		t.Fatalf("unmarshal runtime response: %v", err)
	}

	if got, want := copied.Services[0].Name, "alpha"; got != want {
		t.Fatalf("unexpected copied service name: got %q want %q", got, want)
	}
	if got, want := copied.Services[0].Tags[0], "core"; got != want {
		t.Fatalf("unexpected copied service tag: got %q want %q", got, want)
	}
	if got, want := copied.Ports[0], 8080; got != want {
		t.Fatalf("unexpected copied port: got %d want %d", got, want)
	}
}

func TestCopyRuntimeServicesUsesIndependentTagSlices(t *testing.T) {
	t.Helper()

	services := []orchestrator.Metadata{
		{
			Name:        "alpha",
			Description: "first service",
			Version:     "v1",
			Tags:        []string{"core"},
		},
	}

	copied := copyRuntimeServices(services)
	services[0].Tags[0] = "changed"

	if got, want := copied[0].Tags[0], "core"; got != want {
		t.Fatalf("unexpected copied tag: got %q want %q", got, want)
	}
}
