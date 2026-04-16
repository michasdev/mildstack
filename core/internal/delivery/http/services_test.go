package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

func TestServicesIndexReturnsSortedSummaries(t *testing.T) {
	t.Helper()

	router := NewRouter(DefaultConfig(), snapshotStub{
		snapshot: runtime.Snapshot{
			Services: []orchestrator.Metadata{
				{
					Name:        "beta",
					Description: "second service",
					Version:     "v2",
					Tags:        []string{"beta"},
				},
				{
					Name:        "alpha",
					Description: "first service",
					Version:     "v1",
					Tags:        []string{"alpha"},
				},
			},
			Ports: []int{8080},
		},
	})

	if err := router.Registrar().Register(orchestrator.Route{
		Method: "GET",
		Path:   "/alpha/health",
		Name:   "health",
	}); err != nil {
		t.Fatalf("register alpha route: %v", err)
	}
	if err := router.Registrar().Register(orchestrator.Route{
		Method: "POST",
		Path:   "/beta/items",
		Name:   "create-item",
	}); err != nil {
		t.Fatalf("register beta route: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/services", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}

	var response servicesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal services response: %v", err)
	}
	if got, want := len(response.Services), 2; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}
	if got, want := response.Services[0].Name, "alpha"; got != want {
		t.Fatalf("unexpected first service: got %q want %q", got, want)
	}
	if got, want := response.Services[0].RouteCount, 1; got != want {
		t.Fatalf("unexpected alpha route count: got %d want %d", got, want)
	}
	if got, want := response.Services[1].Name, "beta"; got != want {
		t.Fatalf("unexpected second service: got %q want %q", got, want)
	}
	if got, want := response.Services[1].RouteCount, 1; got != want {
		t.Fatalf("unexpected beta route count: got %d want %d", got, want)
	}
}

func TestServiceEndpointReturnsMetadataAndNormalizedRoutes(t *testing.T) {
	t.Helper()

	router := NewRouter(DefaultConfig(), snapshotStub{
		snapshot: runtime.Snapshot{
			Services: []orchestrator.Metadata{
				{
					Name:        "alpha",
					Description: "first service",
					Version:     "v1",
					Tags:        []string{"core", "alpha"},
				},
			},
		},
	})

	routes := []orchestrator.Route{
		{Method: "post", Path: "/alpha/items", Name: "create-item"},
		{Method: "get", Path: "/alpha/health", Name: "health"},
	}
	for _, route := range routes {
		if err := router.Registrar().Register(route); err != nil {
			t.Fatalf("register route %+v: %v", route, err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/services/alpha", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}

	var response serviceResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal service response: %v", err)
	}
	if got, want := response.Name, "alpha"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := response.Description, "first service"; got != want {
		t.Fatalf("unexpected service description: got %q want %q", got, want)
	}
	if got, want := response.Tags, []string{"core", "alpha"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected service tags: got %v want %v", got, want)
	}
	if got, want := len(response.Routes), 2; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	if got, want := response.Routes[0].Path, "/api/v1/runtime/services/alpha/health"; got != want {
		t.Fatalf("unexpected first route path: got %q want %q", got, want)
	}
	if got, want := response.Routes[1].Path, "/api/v1/runtime/services/alpha/items"; got != want {
		t.Fatalf("unexpected second route path: got %q want %q", got, want)
	}
}

func TestServiceEndpointReturnsNotFoundForUnknownService(t *testing.T) {
	t.Helper()

	router := NewRouter(DefaultConfig(), snapshotStub{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/services/missing", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusNotFound; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}
}

func TestServicesIndexReturnsInternalServerErrorForMissingRegistrarEntry(t *testing.T) {
	t.Helper()

	router := NewRouter(DefaultConfig(), snapshotStub{
		snapshot: runtime.Snapshot{
			Services: []orchestrator.Metadata{
				{
					Name:        "alpha",
					Description: "first service",
					Version:     "v1",
					Tags:        []string{"alpha"},
				},
			},
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/services", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal services error response: %v", err)
	}
	if got, want := response["error"], errServiceRoutesNotRegistered.Error(); got != want {
		t.Fatalf("unexpected error body: got %q want %q", got, want)
	}
}

func TestServiceEndpointReturnsInternalServerErrorForMissingRegistrarEntry(t *testing.T) {
	t.Helper()

	router := NewRouter(DefaultConfig(), snapshotStub{
		snapshot: runtime.Snapshot{
			Services: []orchestrator.Metadata{
				{
					Name:        "alpha",
					Description: "first service",
					Version:     "v1",
					Tags:        []string{"core", "alpha"},
				},
			},
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/services/alpha", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal service error response: %v", err)
	}
	if got, want := response["error"], errServiceRoutesNotRegistered.Error(); got != want {
		t.Fatalf("unexpected error body: got %q want %q", got, want)
	}
}
