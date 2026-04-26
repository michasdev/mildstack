package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type stubSNSService struct{}

func (stubSNSService) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityPartial, nil, nil, "sns")
}

func (stubSNSService) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{Name: "sns", Version: "v1"}
}

func (stubSNSService) CreateTopic(string, map[string]string) (domain.Topic, error) {
	return domain.Topic{}, domain.ErrValidation
}

func (stubSNSService) DeleteTopic(string) error {
	return domain.ErrValidation
}

func (stubSNSService) GetTopicAttributes(string) (map[string]string, error) {
	return nil, domain.ErrValidation
}

func (stubSNSService) SetTopicAttributes(string, string, string) (map[string]string, error) {
	return nil, domain.ErrValidation
}

func (stubSNSService) ListTopics(string) ([]domain.Topic, string, error) {
	return nil, "", domain.ErrValidation
}

func (stubSNSService) Subscribe(string, string, string, map[string]string, bool) (domain.SubscribeOutput, error) {
	return domain.SubscribeOutput{}, domain.ErrValidation
}

func (stubSNSService) ConfirmSubscription(string, string) (domain.Subscription, error) {
	return domain.Subscription{}, domain.ErrValidation
}

func (stubSNSService) Unsubscribe(string) error {
	return domain.ErrValidation
}

func (stubSNSService) GetSubscriptionAttributes(string) (map[string]string, error) {
	return nil, domain.ErrValidation
}

func (stubSNSService) SetSubscriptionAttributes(string, string, string) (map[string]string, error) {
	return nil, domain.ErrValidation
}

func (stubSNSService) ListSubscriptions(string) ([]domain.Subscription, string, error) {
	return nil, "", domain.ErrValidation
}

func (stubSNSService) ListSubscriptionsByTopic(string, string) ([]domain.Subscription, string, error) {
	return nil, "", domain.ErrValidation
}

func (stubSNSService) Publish(domain.PublishRequest) (domain.PublishResult, error) {
	return domain.PublishResult{}, domain.ErrValidation
}

func (stubSNSService) PublishBatch(domain.PublishBatchRequest) (domain.PublishBatchResult, error) {
	return domain.PublishBatchResult{}, domain.ErrValidation
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
