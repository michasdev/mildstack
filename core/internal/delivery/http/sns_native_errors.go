package http

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type SNSXMLErrorResponse struct {
	XMLName   xml.Name    `xml:"ErrorResponse"`
	XMLNS     string      `xml:"xmlns,attr,omitempty"`
	Error     SNSXMLError `xml:"Error"`
	RequestID string      `xml:"RequestId"`
}

type SNSXMLError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func writeSNSError(c *gin.Context, err error, requestID string) {
	status, code, message := classifySNSError(err)
	writeSNSErrorResponse(c, status, code, message, requestID)
}

func writeSNSErrorResponse(c *gin.Context, status int, code, message, requestID string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(status, SNSXMLErrorResponse{
		XMLNS: snsXMLNamespace,
		Error: SNSXMLError{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: strings.TrimSpace(requestID),
	})
}

func classifySNSError(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrSNSNotOwned):
		return http.StatusNotFound, "InvalidAction", "The requested path is not owned by the SNS native adapter."
	case errors.Is(err, ErrSNSMalformedRequest):
		return http.StatusBadRequest, "InvalidQueryParameter", err.Error()
	case errors.Is(err, ErrSNSMissingAction):
		return http.StatusBadRequest, "MissingAction", "The request is missing an action or required parameter."
	case errors.Is(err, ErrSNSMissingVersion):
		return http.StatusBadRequest, "MissingParameter", "The request must contain the parameter Version."
	case errors.Is(err, ErrSNSInvalidVersion):
		return http.StatusBadRequest, "InvalidParameterValue", "Value  for parameter Version is invalid. Reason: Invalid API Version"
	case errors.Is(err, ErrSNSInvalidAction):
		return http.StatusBadRequest, "InvalidAction", "The action or operation requested is invalid."
	case errors.Is(err, ErrSNSUnsupported):
		return http.StatusBadRequest, "InvalidAction", "The action is valid for SNS but not implemented in the local runtime yet."
	case errors.Is(err, domain.ErrBatchEntryIDsNotDistinct):
		return http.StatusBadRequest, "BatchEntryIdsNotDistinct", "Two or more batch entries in the request have the same Id."
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrInvalidToken):
		code := "InvalidParameter"
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "fifo") ||
			strings.Contains(lower, "messagegroupid") ||
			strings.Contains(lower, "messagededuplicationid") ||
			strings.Contains(lower, "contentbaseddeduplication") ||
			strings.Contains(lower, "topic name suffix") ||
			strings.Contains(lower, "sqs endpoint arn") {
			code = "InvalidParameterException"
		}
		return http.StatusBadRequest, code, err.Error()
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "NotFound", "The requested resource does not exist."
	case strings.Contains(strings.ToLower(err.Error()), "not initialized"), strings.Contains(strings.ToLower(err.Error()), "not configured"):
		return http.StatusInternalServerError, "InternalError", "SNS persistence is not available."
	default:
		return http.StatusBadRequest, "ValidationError", err.Error()
	}
}
