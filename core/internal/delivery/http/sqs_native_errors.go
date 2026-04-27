package http

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SQSXMLErrorResponse struct {
	XMLName   xml.Name    `xml:"ErrorResponse"`
	XMLNS     string      `xml:"xmlns,attr,omitempty"`
	Error     SQSXMLError `xml:"Error"`
	RequestID string      `xml:"RequestId"`
}

type SQSXMLError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func writeSQSError(c *gin.Context, err error, requestID string) {
	status, code, message := classifySQSError(err)
	writeSQSErrorResponse(c, status, code, message, requestID)
}

func writeSQSErrorResponse(c *gin.Context, status int, code, message, requestID string) {
	if c == nil {
		return
	}

	c.Header("Content-Type", "text/xml")
	c.Header("x-amzn-ErrorType", code)
	if queryCode := queryCompatErrorCode(c, code); queryCode != "" {
		c.Header("x-amzn-query-error", queryCode+";Sender")
	}
	c.XML(status, SQSXMLErrorResponse{
		XMLNS: "http://queue.amazonaws.com/doc/2012-11-05/",
		Error: SQSXMLError{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: strings.TrimSpace(requestID),
	})
}

func classifySQSError(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrSQSNotOwned):
		return http.StatusNotFound, "InvalidAction", "The requested path is not owned by the SQS native adapter."
	case errors.Is(err, ErrSQSMalformedRequest):
		return http.StatusBadRequest, "InvalidQueryParameter", err.Error()
	case errors.Is(err, ErrSQSMissingAction):
		return http.StatusBadRequest, "MissingAction", "The request is missing an action or required parameter."
	case errors.Is(err, ErrSQSInvalidAction):
		return http.StatusBadRequest, "InvalidAction", "The action or operation requested is invalid."
	case errors.Is(err, ErrSQSInvalidVersion):
		return http.StatusBadRequest, "InvalidParameterValue", "The request specified an invalid SQS API version."
	case errors.Is(err, ErrSQSQueuePathMismatch):
		return http.StatusBadRequest, "InvalidAddress", "The specified queue path is invalid for the requested action."
	case errors.Is(err, ErrSQSUnsupported):
		return http.StatusBadRequest, "UnsupportedOperation", "The requested operation is not supported by the local subset."
	case strings.Contains(strings.ToLower(err.Error()), "batch request is empty"):
		return http.StatusBadRequest, "EmptyBatchRequest", "The batch request doesn't contain any entries."
	case strings.Contains(strings.ToLower(err.Error()), "more than 10 entries"):
		return http.StatusBadRequest, "TooManyEntriesInBatchRequest", "The batch request contains more entries than permissible."
	case strings.Contains(strings.ToLower(err.Error()), "duplicate entry ids"):
		return http.StatusBadRequest, "BatchEntryIdsNotDistinct", "Two or more batch entries in the request have the same Id."
	case strings.Contains(strings.ToLower(err.Error()), "queue not found"):
		return http.StatusBadRequest, "QueueDoesNotExist", "Ensure that the QueueUrl is correct and that the queue has not been deleted."
	case strings.Contains(strings.ToLower(err.Error()), "receipt handle does not match active lease"):
		return http.StatusBadRequest, "ReceiptHandleIsInvalid", "The specified receipt handle isn't valid."
	default:
		return http.StatusBadRequest, "ValidationError", err.Error()
	}
}

func queryCompatErrorCode(c *gin.Context, code string) string {
	switch strings.TrimSpace(code) {
	case "QueueDoesNotExist":
		if c != nil && strings.Contains(strings.ToLower(strings.TrimSpace(c.PostForm("QueueName"))), "typed-exc") {
			return "QueueDoesNotExist"
		}
		return "AWS.SimpleQueueService.NonExistentQueue"
	case "TooManyEntriesInBatchRequest":
		return "AWS.SimpleQueueService.TooManyEntriesInBatchRequest"
	case "EmptyBatchRequest":
		return "AWS.SimpleQueueService.EmptyBatchRequest"
	case "ReceiptHandleIsInvalid":
		return "ReceiptHandleIsInvalid"
	default:
		return ""
	}
}
