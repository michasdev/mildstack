package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
)

func newSNSContractHarness(t *testing.T) (*gin.Engine, *snsapplication.Service) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: "contract-instance",
	})
	if err != nil {
		t.Fatalf("new sns service: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	router := gin.New()
	deliveryhttp.RegisterSNSNativeRoutes(router, service)
	return router, service
}

func performSNSQuery(t *testing.T, engine *gin.Engine, method, query string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, "/?"+query, nil)
	if strings.EqualFold(method, http.MethodPost) {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}
