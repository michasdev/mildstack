package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

type SQSNativeService interface {
	Policy() orchestrator.EmulationPolicy
	Metadata() orchestrator.Metadata
}

func RegisterSQSNativeRoutes(engine *gin.Engine, service SQSNativeService) {
	if engine == nil || service == nil {
		return
	}

	handler := newSQSNativeHandler(service)
	engine.Use(func(c *gin.Context) {
		if handled := handler.dispatch(c); handled {
			c.Abort()
			return
		}
		c.Next()
	})
}

type sqsNativeHandler struct {
	service   SQSNativeService
	registry  SQSRegistry
	supported map[string]struct{}
}

func newSQSNativeHandler(service SQSNativeService) sqsNativeHandler {
	supported := make(map[string]struct{})
	if service != nil {
		for _, action := range service.Policy().Supported {
			supported[action] = struct{}{}
		}
	}

	return sqsNativeHandler{
		service:   service,
		registry:  NewSQSRegistry(),
		supported: supported,
	}
}

func (h sqsNativeHandler) dispatch(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}

	path := strings.TrimSpace(c.Request.URL.Path)
	if path == "" || strings.HasPrefix(path, "/api/") {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet, http.MethodPost:
	default:
		return false
	}

	ctx, err := ParseSQSRequest(c.Request)
	if err != nil {
		if errors.Is(err, ErrSQSNotOwned) {
			return false
		}
		writeSQSError(c, err, requestIDFromContext(c))
		return true
	}

	spec, err := h.registry.Resolve(ctx)
	if err != nil {
		writeSQSError(c, err, requestIDFromContext(c))
		return true
	}

	if _, ok := h.supported[spec.Action]; !ok || spec.DomainDeferred {
		writeSQSError(c, ErrSQSUnsupported, requestIDFromContext(c))
		return true
	}

	writeSQSError(c, ErrSQSUnsupported, requestIDFromContext(c))
	return true
}

func requestIDFromContext(c *gin.Context) string {
	if c == nil {
		return "mildstack-sqs-request"
	}

	for _, key := range []string{"x-amzn-requestid", "X-Amzn-RequestId", "x-amz-request-id"} {
		if requestID := strings.TrimSpace(c.GetHeader(key)); requestID != "" {
			return requestID
		}
	}

	return "mildstack-sqs-request"
}
