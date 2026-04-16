package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type snapshotStub struct {
	snapshot runtime.Snapshot
}

func (s snapshotStub) Snapshot(context.Context) runtime.Snapshot {
	return s.snapshot
}

func TestHealthEndpointReturnsOk(t *testing.T) {
	t.Helper()

	router := NewRouter(DefaultConfig(), snapshotStub{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/health", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status code: got %d want %d", got, want)
	}

	var response healthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if got, want := response.Status, "ok"; got != want {
		t.Fatalf("unexpected health status: got %q want %q", got, want)
	}
}

func TestReadinessEndpointReflectsRuntimeSnapshot(t *testing.T) {
	t.Helper()

	readySnapshot := runtime.Snapshot{
		Services: []orchestrator.Metadata{
			{
				Name:        "alpha",
				Description: "first service",
				Version:     "v1",
				Tags:        []string{"core"},
			},
		},
		Ports: []int{8080},
	}
	router := NewRouter(DefaultConfig(), snapshotStub{snapshot: readySnapshot})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/ready", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected readiness status code: got %d want %d", got, want)
	}

	var response readinessResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal readiness response: %v", err)
	}
	if got, want := response.Status, "ready"; got != want {
		t.Fatalf("unexpected readiness status: got %q want %q", got, want)
	}
	if got, want := len(response.Services), 1; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}
	if got, want := response.Ports, []int{8080}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected readiness ports: got %v want %v", got, want)
	}

	notReadyRouter := NewRouter(DefaultConfig(), snapshotStub{})
	notReadyRecorder := httptest.NewRecorder()
	notReadyRequest := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/ready", nil)
	notReadyRouter.Engine().ServeHTTP(notReadyRecorder, notReadyRequest)

	if got, want := notReadyRecorder.Code, http.StatusServiceUnavailable; got != want {
		t.Fatalf("unexpected not-ready status code: got %d want %d", got, want)
	}

	var notReadyResponse readinessResponse
	if err := json.Unmarshal(notReadyRecorder.Body.Bytes(), &notReadyResponse); err != nil {
		t.Fatalf("unmarshal not-ready response: %v", err)
	}
	if got, want := notReadyResponse.Status, "not_ready"; got != want {
		t.Fatalf("unexpected not-ready status: got %q want %q", got, want)
	}
}

func TestReadinessResponseCopiesSourceSlices(t *testing.T) {
	t.Helper()

	services := []orchestrator.Metadata{
		{
			Name:        "alpha",
			Description: "first service",
			Version:     "v1",
			Tags:        []string{"core"},
		},
	}
	ports := []int{8080}

	response := readinessResponse{
		Status:   "ready",
		Services: copyRuntimeServices(services),
		Ports:    copyRuntimePorts(ports),
	}

	services[0].Name = "mutated"
	services[0].Tags[0] = "changed"
	ports[0] = 9090

	rendered, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal readiness response: %v", err)
	}

	var copied readinessResponse
	if err := json.Unmarshal(rendered, &copied); err != nil {
		t.Fatalf("unmarshal readiness response: %v", err)
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
