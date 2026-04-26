package http

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const snsXMLNamespace = "https://sns.amazonaws.com/doc/2010-03-31/"

var (
	// ErrSNSUnsupported indicates the action is known but still deferred in domain implementation.
	ErrSNSUnsupported = errors.New("sns action is not implemented")
	// ErrSNSInvalidAction indicates an unrecognized or unsupported SNS action.
	ErrSNSInvalidAction = errors.New("sns action is invalid")
)

func RegisterSNSNativeRoutes(engine *gin.Engine, service SNSNativeService) {
	if engine == nil || service == nil {
		return
	}

	handler := NewSNSNativeHandler(service)
	engine.Use(func(c *gin.Context) {
		if handled := handler.dispatch(c); handled {
			c.Abort()
			return
		}
		c.Next()
	})
}

// SNSNativeHandler owns SNS Query/XML request handling.
type SNSNativeHandler struct {
	service  SNSNativeService
	registry SNSRegistry
}

func NewSNSNativeHandler(service SNSNativeService) SNSNativeHandler {
	return SNSNativeHandler{
		service:  service,
		registry: NewSNSRegistry(),
	}
}

func (h SNSNativeHandler) dispatch(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}

	ctx, err := ParseSNSRequest(c.Request, h.registry)
	if err != nil {
		if errors.Is(err, ErrSNSNotOwned) {
			return false
		}
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return true
	}

	spec, err := h.registry.Resolve(ctx)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return true
	}

	if !spec.Supported || spec.DomainDeferred {
		writeSNSError(c, ErrSNSUnsupported, snsRequestIDFromContext(c))
		return true
	}

	writeSNSSuccessEnvelope(c, spec.Action, nil)
	return true
}

func snsRequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return "mildstack-sns-request"
	}

	for _, key := range []string{"x-amzn-requestid", "X-Amzn-RequestId", "x-amz-request-id"} {
		if requestID := strings.TrimSpace(c.GetHeader(key)); requestID != "" {
			return requestID
		}
	}

	return "mildstack-sns-request"
}

type snsSuccessEnvelope struct {
	XMLName          xml.Name            `xml:""`
	XMLNS            string              `xml:"xmlns,attr"`
	Result           *snsEmptyResult     `xml:",omitempty"`
	ResponseMetadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsEmptyResult struct {
	XMLName xml.Name
}

type snsResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

func writeSNSSuccessEnvelope(c *gin.Context, action string, result any) {
	if c == nil {
		return
	}

	response := snsSuccessEnvelope{
		XMLName:          xml.Name{Local: action + "Response"},
		XMLNS:            snsXMLNamespace,
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	}
	if result != nil {
		response.Result = &snsEmptyResult{XMLName: xml.Name{Local: action + "Result"}}
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, response)
}
