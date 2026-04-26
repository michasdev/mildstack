package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

type stubSNSService struct{}

func (stubSNSService) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityPartial, nil, nil, "sns")
}

func (stubSNSService) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{Name: "sns", Version: "v1"}
}

func TestSNSContractMissingActionReturnsMissingActionError(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	deliveryhttp.RegisterSNSNativeRoutes(engine, stubSNSService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/?Version=2010-03-31", nil)
	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "MissingAction") {
		t.Fatalf("expected MissingAction in body, got %q", body)
	}
}

func TestSNSContractMissingVersionReturnsMissingParameterError(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	deliveryhttp.RegisterSNSNativeRoutes(engine, stubSNSService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/?Action=CreateTopic", nil)
	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "MissingParameter") {
		t.Fatalf("expected MissingParameter in body, got %q", body)
	}
}

func TestSNSContractInvalidVersionReturnsInvalidParameterValue(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	deliveryhttp.RegisterSNSNativeRoutes(engine, stubSNSService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/?Action=CreateTopic&Version=2012-11-05", nil)
	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "InvalidParameterValue") {
		t.Fatalf("expected InvalidParameterValue in body, got %q", body)
	}
}
