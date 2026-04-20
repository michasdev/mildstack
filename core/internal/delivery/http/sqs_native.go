package http

import (
	"encoding/xml"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

type SQSNativeService interface {
	Policy() orchestrator.EmulationPolicy
	Metadata() orchestrator.Metadata
	QueueURL(queueName string) string
	QueueARN(queueName string) string
	CreateQueue(queueName string, attributes map[string]string) (domain.Queue, error)
	DeleteQueue(queueName string) error
	GetQueueUrl(queueName, ownerAccountID string) (string, error)
	ListQueues(queueNamePrefix string, maxResults int, nextToken, ownerAccountID string) ([]domain.Queue, string, error)
	PurgeQueue(queueName string) error
	GetQueueAttributes(queueName string, attributeNames []string, ownerAccountID string) (contracts.QueueAttributesView, error)
	SetQueueAttributes(queueName string, attributes map[string]string) (contracts.QueueAttributesView, error)
	ReceiveMessage(queueName string, maxMessages int, waitTime time.Duration) ([]domain.Message, error)
	DeleteMessage(queueName string, receiptHandle string) error
	ChangeMessageVisibility(queueName string, receiptHandle string, visibility time.Duration) error
	SendMessage(queueName string, request contracts.SendMessageRequest) (contracts.SendMessageResult, error)
	SendMessageBatch(queueName string, request contracts.SendMessageBatchRequest) (contracts.SendMessageBatchResult, error)
	DeleteMessageBatch(queueName string, request contracts.DeleteMessageBatchRequest) (contracts.DeleteMessageBatchResult, error)
	ChangeMessageVisibilityBatch(queueName string, request contracts.ChangeMessageVisibilityBatchRequest) (contracts.ChangeMessageVisibilityBatchResult, error)
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

	switch spec.Action {
	case "CreateQueue":
		h.handleCreateQueue(c, ctx)
	case "DeleteQueue":
		h.handleDeleteQueue(c, ctx)
	case "GetQueueUrl":
		h.handleGetQueueUrl(c, ctx)
	case "ListQueues":
		h.handleListQueues(c, ctx)
	case "PurgeQueue":
		h.handlePurgeQueue(c, ctx)
	case "GetQueueAttributes":
		h.handleGetQueueAttributes(c, ctx)
	case "SetQueueAttributes":
		h.handleSetQueueAttributes(c, ctx)
	default:
		writeSQSError(c, ErrSQSUnsupported, requestIDFromContext(c))
	}
	return true
}

func (h sqsNativeHandler) handleCreateQueue(c *gin.Context, ctx SQSRequestContext) {
	queueName := strings.TrimSpace(ctx.Values.Get("QueueName"))
	attributes := queueAttributesFromValues(ctx.Values)
	queue, err := h.service.CreateQueue(queueName, attributes)
	if err != nil {
		h.finishQueueAction(c, err)
		return
	}
	writeSQSCreateQueueResponse(c, queue)
}

func (h sqsNativeHandler) handleDeleteQueue(c *gin.Context, ctx SQSRequestContext) {
	if err := h.service.DeleteQueue(ctx.QueueName); err != nil {
		h.finishQueueAction(c, err)
		return
	}
	writeSQSDeleteQueueResponse(c)
}

func (h sqsNativeHandler) handleGetQueueUrl(c *gin.Context, ctx SQSRequestContext) {
	queueName := strings.TrimSpace(ctx.Values.Get("QueueName"))
	ownerAccountID := queueOwnerAccountID(ctx.Values)
	queueURL, err := h.service.GetQueueUrl(queueName, ownerAccountID)
	if err != nil {
		h.finishQueueAction(c, err)
		return
	}
	writeSQSGetQueueUrlResponse(c, queueURL)
}

func (h sqsNativeHandler) handleListQueues(c *gin.Context, ctx SQSRequestContext) {
	prefix := queueNamePrefix(ctx.Values)
	maxResults := queueMaxResults(ctx.Values)
	nextToken := queueNextToken(ctx.Values)
	ownerAccountID := queueOwnerAccountID(ctx.Values)
	queues, nextPageToken, err := h.service.ListQueues(prefix, maxResults, nextToken, ownerAccountID)
	if err != nil {
		h.finishQueueAction(c, err)
		return
	}
	if ctx.TargetStyle {
		writeSQSListQueuesJSONResponse(c, queues, nextPageToken)
		return
	}
	writeSQSListQueuesResponse(c, queues, nextPageToken)
}

func (h sqsNativeHandler) handlePurgeQueue(c *gin.Context, ctx SQSRequestContext) {
	if err := h.service.PurgeQueue(ctx.QueueName); err != nil {
		h.finishQueueAction(c, err)
		return
	}
	writeSQSPurgeQueueResponse(c)
}

func (h sqsNativeHandler) handleGetQueueAttributes(c *gin.Context, ctx SQSRequestContext) {
	attributeNames := queueAttributeNames(ctx.Values)
	ownerAccountID := queueOwnerAccountID(ctx.Values)
	attributes, err := h.service.GetQueueAttributes(ctx.QueueName, attributeNames, ownerAccountID)
	if err != nil {
		h.finishQueueAction(c, err)
		return
	}
	writeSQSGetQueueAttributesResponse(c, attributes)
}

func (h sqsNativeHandler) handleSetQueueAttributes(c *gin.Context, ctx SQSRequestContext) {
	attributes := queueAttributesFromValues(ctx.Values)
	view, err := h.service.SetQueueAttributes(ctx.QueueName, attributes)
	if err != nil {
		h.finishQueueAction(c, err)
		return
	}
	writeSQSSetQueueAttributesResponse(c, view)
}

func (h sqsNativeHandler) finishQueueAction(c *gin.Context, err error) {
	if err == nil || errors.Is(err, contracts.ErrSQSOperationDeferred) {
		writeSQSError(c, ErrSQSUnsupported, requestIDFromContext(c))
		return
	}
	writeSQSError(c, err, requestIDFromContext(c))
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

type sqsQueueUrlResponse struct {
	XMLName           xml.Name            `xml:"GetQueueUrlResponse"`
	GetQueueUrlResult sqsQueueUrlResult   `xml:"GetQueueUrlResult"`
	ResponseMetadata  sqsResponseMetadata `xml:"ResponseMetadata"`
}

type sqsQueueUrlResult struct {
	QueueURL string `xml:"QueueUrl"`
}

type sqsCreateQueueResponse struct {
	XMLName           xml.Name            `xml:"CreateQueueResponse"`
	CreateQueueResult sqsQueueUrlResult   `xml:"CreateQueueResult"`
	ResponseMetadata  sqsResponseMetadata `xml:"ResponseMetadata"`
}

type sqsListQueuesResponse struct {
	XMLName          xml.Name            `xml:"ListQueuesResponse"`
	ListQueuesResult sqsListQueuesResult `xml:"ListQueuesResult"`
	ResponseMetadata sqsResponseMetadata `xml:"ResponseMetadata"`
}

type sqsListQueuesResult struct {
	QueueUrls []string `xml:"QueueUrl"`
	NextToken string   `xml:"NextToken,omitempty"`
}

type sqsListQueuesJSONResponse struct {
	QueueUrls []string `json:"QueueUrls"`
	NextToken string   `json:"NextToken,omitempty"`
}

type sqsGetQueueAttributesResponse struct {
	XMLName                  xml.Name                    `xml:"GetQueueAttributesResponse"`
	GetQueueAttributesResult sqsGetQueueAttributesResult `xml:"GetQueueAttributesResult"`
	ResponseMetadata         sqsResponseMetadata         `xml:"ResponseMetadata"`
}

type sqsGetQueueAttributesResult struct {
	Attributes []sqsQueueAttributeXML `xml:"Attribute"`
}

type sqsQueueAttributeXML struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type sqsSetQueueAttributesResponse struct {
	XMLName          xml.Name            `xml:"SetQueueAttributesResponse"`
	ResponseMetadata sqsResponseMetadata `xml:"ResponseMetadata"`
}

type sqsDeleteQueueResponse struct {
	XMLName          xml.Name            `xml:"DeleteQueueResponse"`
	ResponseMetadata sqsResponseMetadata `xml:"ResponseMetadata"`
}

type sqsPurgeQueueResponse struct {
	XMLName          xml.Name            `xml:"PurgeQueueResponse"`
	ResponseMetadata sqsResponseMetadata `xml:"ResponseMetadata"`
}

type sqsResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

func writeSQSCreateQueueResponse(c *gin.Context, queue domain.Queue) {
	queueURL := queue.URL
	if queueURL == "" {
		aws := awscontext.Default()
		queueURL = "https://sqs." + aws.Region + ".amazonaws.com/" + aws.AccountID + "/" + queue.Name
	}
	c.XML(http.StatusOK, sqsCreateQueueResponse{
		CreateQueueResult: sqsQueueUrlResult{QueueURL: queueURL},
		ResponseMetadata:  sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSGetQueueUrlResponse(c *gin.Context, queueURL string) {
	c.XML(http.StatusOK, sqsQueueUrlResponse{
		GetQueueUrlResult: sqsQueueUrlResult{QueueURL: queueURL},
		ResponseMetadata:  sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSListQueuesResponse(c *gin.Context, queues []domain.Queue, nextToken string) {
	urls := make([]string, 0, len(queues))
	for _, queue := range queues {
		queueURL := queue.URL
		if queueURL == "" {
			aws := awscontext.Default()
			queueURL = "https://sqs." + aws.Region + ".amazonaws.com/" + aws.AccountID + "/" + queue.Name
		}
		urls = append(urls, queueURL)
	}
	c.XML(http.StatusOK, sqsListQueuesResponse{
		ListQueuesResult: sqsListQueuesResult{
			QueueUrls: urls,
			NextToken: nextToken,
		},
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSListQueuesJSONResponse(c *gin.Context, queues []domain.Queue, nextToken string) {
	urls := make([]string, 0, len(queues))
	for _, queue := range queues {
		queueURL := queue.URL
		if queueURL == "" {
			aws := awscontext.Default()
			queueURL = "https://sqs." + aws.Region + ".amazonaws.com/" + aws.AccountID + "/" + queue.Name
		}
		urls = append(urls, queueURL)
	}

	c.JSON(http.StatusOK, sqsListQueuesJSONResponse{
		QueueUrls: urls,
		NextToken: nextToken,
	})
}

func writeSQSGetQueueAttributesResponse(c *gin.Context, view contracts.QueueAttributesView) {
	c.XML(http.StatusOK, sqsGetQueueAttributesResponse{
		GetQueueAttributesResult: sqsGetQueueAttributesResult{
			Attributes: sortedQueueAttributeXML(view.Attributes),
		},
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSSetQueueAttributesResponse(c *gin.Context, _ contracts.QueueAttributesView) {
	c.XML(http.StatusOK, sqsSetQueueAttributesResponse{
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSDeleteQueueResponse(c *gin.Context) {
	c.XML(http.StatusOK, sqsDeleteQueueResponse{
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSPurgeQueueResponse(c *gin.Context) {
	c.XML(http.StatusOK, sqsPurgeQueueResponse{
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func sortedQueueAttributeXML(attributes map[string]string) []sqsQueueAttributeXML {
	if len(attributes) == 0 {
		return nil
	}

	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]sqsQueueAttributeXML, 0, len(keys))
	for _, key := range keys {
		items = append(items, sqsQueueAttributeXML{Name: key, Value: attributes[key]})
	}
	return items
}
