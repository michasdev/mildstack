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
