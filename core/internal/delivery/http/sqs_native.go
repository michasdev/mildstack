package http

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"net/http"
	"sort"
	"strconv"
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
	service  SQSNativeService
	registry SQSRegistry
}

func newSQSNativeHandler(service SQSNativeService) sqsNativeHandler {
	return sqsNativeHandler{
		service:  service,
		registry: NewSQSRegistry(),
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

	if !spec.Supported || spec.DomainDeferred {
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
	case "ReceiveMessage":
		h.handleReceiveMessage(c, ctx)
	case "SendMessage":
		h.handleSendMessage(c, ctx)
	case "SendMessageBatch":
		h.handleSendMessageBatch(c, ctx)
	case "DeleteMessage":
		h.handleDeleteMessage(c, ctx)
	case "DeleteMessageBatch":
		h.handleDeleteMessageBatch(c, ctx)
	case "ChangeMessageVisibility":
		h.handleChangeMessageVisibility(c, ctx)
	case "ChangeMessageVisibilityBatch":
		h.handleChangeMessageVisibilityBatch(c, ctx)
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

func (h sqsNativeHandler) handleReceiveMessage(c *gin.Context, ctx SQSRequestContext) {
	request := receiveMessageRequestFromValues(ctx.Values)
	maxMessages := request.MaxNumberOfMessages
	if maxMessages <= 0 {
		maxMessages = 1
	}
	waitTime := time.Duration(request.WaitTimeSeconds) * time.Second
	messages, err := h.service.ReceiveMessage(ctx.QueueName, maxMessages, waitTime)
	if err != nil {
		h.finishMessageAction(c, err)
		return
	}
	if ctx.TargetStyle {
		writeSQSReceiveMessageJSON(c, messages)
		return
	}
	writeSQSReceiveMessageResponse(c, messages)
}

func (h sqsNativeHandler) handleSendMessage(c *gin.Context, ctx SQSRequestContext) {
	request := sendMessageRequestFromValues(ctx.Values)
	result, err := h.service.SendMessage(ctx.QueueName, request)
	if err != nil {
		h.finishMessageAction(c, err)
		return
	}
	if ctx.TargetStyle {
		writeSQSSendMessageJSON(c, result)
		return
	}
	writeSQSSendMessageResponse(c, result)
}

func (h sqsNativeHandler) handleSendMessageBatch(c *gin.Context, ctx SQSRequestContext) {
	request := sendMessageBatchRequestFromValues(ctx.Values)
	result, err := h.service.SendMessageBatch(ctx.QueueName, request)
	if err != nil {
		h.finishMessageAction(c, err)
		return
	}
	if ctx.TargetStyle {
		writeSQSSendMessageBatchJSON(c, result)
		return
	}
	writeSQSSendMessageBatchResponse(c, result)
}

func (h sqsNativeHandler) handleDeleteMessage(c *gin.Context, ctx SQSRequestContext) {
	request := deleteMessageRequestFromValues(ctx.Values)
	if err := h.service.DeleteMessage(ctx.QueueName, request.ReceiptHandle); err != nil {
		h.finishMessageAction(c, err)
		return
	}
	writeSQSDeleteMessageResponse(c)
}

func (h sqsNativeHandler) handleDeleteMessageBatch(c *gin.Context, ctx SQSRequestContext) {
	request := deleteMessageBatchRequestFromValues(ctx.Values)
	result, err := h.service.DeleteMessageBatch(ctx.QueueName, request)
	if err != nil {
		h.finishMessageAction(c, err)
		return
	}
	if ctx.TargetStyle {
		writeSQSDeleteMessageBatchJSON(c, result)
		return
	}
	writeSQSDeleteMessageBatchResponse(c, result)
}

func (h sqsNativeHandler) handleChangeMessageVisibility(c *gin.Context, ctx SQSRequestContext) {
	request := changeMessageVisibilityRequestFromValues(ctx.Values)
	visibility := time.Duration(request.VisibilityTimeout) * time.Second
	if err := h.service.ChangeMessageVisibility(ctx.QueueName, request.ReceiptHandle, visibility); err != nil {
		h.finishMessageAction(c, err)
		return
	}
	writeSQSChangeMessageVisibilityResponse(c)
}

func (h sqsNativeHandler) handleChangeMessageVisibilityBatch(c *gin.Context, ctx SQSRequestContext) {
	request := changeMessageVisibilityBatchRequestFromValues(ctx.Values)
	result, err := h.service.ChangeMessageVisibilityBatch(ctx.QueueName, request)
	if err != nil {
		h.finishMessageAction(c, err)
		return
	}
	if ctx.TargetStyle {
		writeSQSChangeMessageVisibilityBatchJSON(c, result)
		return
	}
	writeSQSChangeMessageVisibilityBatchResponse(c, result)
}

func (h sqsNativeHandler) finishQueueAction(c *gin.Context, err error) {
	if err == nil || errors.Is(err, contracts.ErrSQSOperationDeferred) {
		writeSQSError(c, ErrSQSUnsupported, requestIDFromContext(c))
		return
	}
	writeSQSError(c, err, requestIDFromContext(c))
}

func (h sqsNativeHandler) finishMessageAction(c *gin.Context, err error) {
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

type sqsSendMessageResponse struct {
	XMLName           xml.Name             `xml:"SendMessageResponse"`
	SendMessageResult sqsSendMessageResult `xml:"SendMessageResult"`
	ResponseMetadata  sqsResponseMetadata  `xml:"ResponseMetadata"`
}

type sqsSendMessageResult struct {
	MessageID                    string `xml:"MessageId,omitempty"`
	MD5OfMessageBody             string `xml:"MD5OfMessageBody,omitempty"`
	MD5OfMessageAttributes       string `xml:"MD5OfMessageAttributes,omitempty"`
	MD5OfMessageSystemAttributes string `xml:"MD5OfMessageSystemAttributes,omitempty"`
	SequenceNumber               string `xml:"SequenceNumber,omitempty"`
}

type sqsSendMessageBatchResponse struct {
	XMLName                xml.Name                  `xml:"SendMessageBatchResponse"`
	SendMessageBatchResult sqsSendMessageBatchResult `xml:"SendMessageBatchResult"`
	ResponseMetadata       sqsResponseMetadata       `xml:"ResponseMetadata"`
}

type sqsSendMessageBatchResult struct {
	Successful []contracts.SendMessageBatchResultEntry `xml:"SendMessageBatchResultEntry,omitempty"`
	Failed     []contracts.BatchResultErrorEntry       `xml:"BatchResultErrorEntry,omitempty"`
}

type sqsDeleteMessageBatchResponse struct {
	XMLName                  xml.Name                    `xml:"DeleteMessageBatchResponse"`
	DeleteMessageBatchResult sqsDeleteMessageBatchResult `xml:"DeleteMessageBatchResult"`
	ResponseMetadata         sqsResponseMetadata         `xml:"ResponseMetadata"`
}

type sqsDeleteMessageBatchResult struct {
	Successful []contracts.DeleteMessageBatchResultEntry `xml:"DeleteMessageBatchResultEntry,omitempty"`
	Failed     []contracts.BatchResultErrorEntry         `xml:"BatchResultErrorEntry,omitempty"`
}

type sqsChangeMessageVisibilityBatchResponse struct {
	XMLName                            xml.Name                              `xml:"ChangeMessageVisibilityBatchResponse"`
	ChangeMessageVisibilityBatchResult sqsChangeMessageVisibilityBatchResult `xml:"ChangeMessageVisibilityBatchResult"`
	ResponseMetadata                   sqsResponseMetadata                   `xml:"ResponseMetadata"`
}

type sqsChangeMessageVisibilityBatchResult struct {
	Successful []contracts.ChangeMessageVisibilityBatchResultEntry `xml:"ChangeMessageVisibilityBatchResultEntry,omitempty"`
	Failed     []contracts.BatchResultErrorEntry                   `xml:"BatchResultErrorEntry,omitempty"`
}

type sqsReceiveMessageResponse struct {
	XMLName              xml.Name                `xml:"ReceiveMessageResponse"`
	ReceiveMessageResult sqsReceiveMessageResult `xml:"ReceiveMessageResult"`
	ResponseMetadata     sqsResponseMetadata     `xml:"ResponseMetadata"`
}

type sqsReceiveMessageResult struct {
	Messages []sqsReceivedMessageXML `xml:"Message,omitempty"`
}

type sqsReceivedMessageXML struct {
	Body                   string                   `xml:"Body,omitempty"`
	MD5OfBody              string                   `xml:"MD5OfBody,omitempty"`
	MD5OfMessageAttributes string                   `xml:"MD5OfMessageAttributes,omitempty"`
	MessageID              string                   `xml:"MessageId,omitempty"`
	ReceiptHandle          string                   `xml:"ReceiptHandle,omitempty"`
	Attributes             []sqsMessageAttributeXML `xml:"Attribute,omitempty"`
}

type sqsMessageAttributeXML struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
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

func writeSQSSendMessageResponse(c *gin.Context, result contracts.SendMessageResult) {
	c.XML(http.StatusOK, sqsSendMessageResponse{
		SendMessageResult: sqsSendMessageResult{
			MessageID:                    result.MessageId,
			MD5OfMessageBody:             result.MD5OfMessageBody,
			MD5OfMessageAttributes:       result.MD5OfMessageAttributes,
			MD5OfMessageSystemAttributes: result.MD5OfMessageSystemAttributes,
			SequenceNumber:               result.SequenceNumber,
		},
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSSendMessageJSON(c *gin.Context, result contracts.SendMessageResult) {
	c.JSON(http.StatusOK, result)
}

func writeSQSSendMessageBatchResponse(c *gin.Context, result contracts.SendMessageBatchResult) {
	c.XML(http.StatusOK, sqsSendMessageBatchResponse{
		SendMessageBatchResult: sqsSendMessageBatchResult{
			Successful: result.Successful,
			Failed:     result.Failed,
		},
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSSendMessageBatchJSON(c *gin.Context, result contracts.SendMessageBatchResult) {
	c.JSON(http.StatusOK, result)
}

func writeSQSDeleteMessageResponse(c *gin.Context) {
	if c == nil {
		return
	}
	c.Status(http.StatusOK)
}

func writeSQSDeleteMessageBatchResponse(c *gin.Context, result contracts.DeleteMessageBatchResult) {
	c.XML(http.StatusOK, sqsDeleteMessageBatchResponse{
		DeleteMessageBatchResult: sqsDeleteMessageBatchResult{
			Successful: result.Successful,
			Failed:     result.Failed,
		},
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSDeleteMessageBatchJSON(c *gin.Context, result contracts.DeleteMessageBatchResult) {
	c.JSON(http.StatusOK, result)
}

func writeSQSChangeMessageVisibilityResponse(c *gin.Context) {
	if c == nil {
		return
	}
	c.Status(http.StatusOK)
}

func writeSQSChangeMessageVisibilityBatchResponse(c *gin.Context, result contracts.ChangeMessageVisibilityBatchResult) {
	c.XML(http.StatusOK, sqsChangeMessageVisibilityBatchResponse{
		ChangeMessageVisibilityBatchResult: sqsChangeMessageVisibilityBatchResult{
			Successful: result.Successful,
			Failed:     result.Failed,
		},
		ResponseMetadata: sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSChangeMessageVisibilityBatchJSON(c *gin.Context, result contracts.ChangeMessageVisibilityBatchResult) {
	c.JSON(http.StatusOK, result)
}

func writeSQSReceiveMessageResponse(c *gin.Context, messages []domain.Message) {
	xmlMessages := make([]sqsReceivedMessageXML, 0, len(messages))
	for _, message := range messages {
		xmlMessages = append(xmlMessages, sqsReceivedMessageXML{
			Body:                   message.Body,
			MD5OfBody:              messageMD5OfBody(message.Body),
			MessageID:              message.MessageID,
			ReceiptHandle:          applicationCurrentReceiptHandle(message),
			Attributes:             receivedMessageAttributesXML(message),
			MD5OfMessageAttributes: "",
		})
	}
	c.XML(http.StatusOK, sqsReceiveMessageResponse{
		ReceiveMessageResult: sqsReceiveMessageResult{Messages: xmlMessages},
		ResponseMetadata:     sqsResponseMetadata{RequestID: requestIDFromContext(c)},
	})
}

func writeSQSReceiveMessageJSON(c *gin.Context, messages []domain.Message) {
	received := make([]contracts.ReceivedMessage, 0, len(messages))
	for _, message := range messages {
		received = append(received, contracts.ReceivedMessage{
			Attributes:             receivedMessageAttributesMap(message),
			Body:                   message.Body,
			MD5OfBody:              messageMD5OfBody(message.Body),
			MD5OfMessageAttributes: "",
			MessageAttributes:      receivedMessageAttributesValues(message),
			MessageId:              message.MessageID,
			ReceiptHandle:          applicationCurrentReceiptHandle(message),
		})
	}
	c.JSON(http.StatusOK, contracts.ReceiveMessageResult{Messages: received})
}

func messageMD5OfBody(body string) string {
	sum := md5.Sum([]byte(body))
	return hex.EncodeToString(sum[:])
}

func applicationCurrentReceiptHandle(message domain.Message) string {
	if len(message.ReceiptKeys) == 0 {
		return ""
	}
	return message.ReceiptKeys[len(message.ReceiptKeys)-1]
}

func receivedMessageAttributesXML(message domain.Message) []sqsMessageAttributeXML {
	attrs := make([]sqsMessageAttributeXML, 0, 4)
	if message.Recovery.Attempts > 0 {
		attrs = append(attrs, sqsMessageAttributeXML{Name: "ApproximateReceiveCount", Value: strconv.Itoa(message.Recovery.Attempts)})
	}
	if !message.SentAt.IsZero() {
		attrs = append(attrs, sqsMessageAttributeXML{Name: "SentTimestamp", Value: strconv.FormatInt(message.SentAt.UnixMilli(), 10)})
	}
	if timestamp := strings.TrimSpace(message.Metadata["approximate_first_receive_timestamp"]); timestamp != "" {
		attrs = append(attrs, sqsMessageAttributeXML{Name: "ApproximateFirstReceiveTimestamp", Value: timestamp})
	}
	if groupID := strings.TrimSpace(message.MessageGroupID); groupID != "" {
		attrs = append(attrs, sqsMessageAttributeXML{Name: "MessageGroupId", Value: groupID})
	}
	if sequenceNumber := message.SequenceNumber; sequenceNumber > 0 {
		attrs = append(attrs, sqsMessageAttributeXML{Name: "SequenceNumber", Value: strconv.FormatInt(sequenceNumber, 10)})
	}
	if dedupeID := strings.TrimSpace(message.Metadata["MessageDeduplicationId"]); dedupeID != "" {
		attrs = append(attrs, sqsMessageAttributeXML{Name: "MessageDeduplicationId", Value: dedupeID})
	}
	return attrs
}

func receivedMessageAttributesMap(message domain.Message) map[string]string {
	attrs := map[string]string{}
	if message.Recovery.Attempts > 0 {
		attrs["ApproximateReceiveCount"] = strconv.Itoa(message.Recovery.Attempts)
	}
	if !message.SentAt.IsZero() {
		attrs["SentTimestamp"] = strconv.FormatInt(message.SentAt.UnixMilli(), 10)
	}
	if timestamp := strings.TrimSpace(message.Metadata["approximate_first_receive_timestamp"]); timestamp != "" {
		attrs["ApproximateFirstReceiveTimestamp"] = timestamp
	}
	if groupID := strings.TrimSpace(message.MessageGroupID); groupID != "" {
		attrs["MessageGroupId"] = groupID
	}
	if sequenceNumber := message.SequenceNumber; sequenceNumber > 0 {
		attrs["SequenceNumber"] = strconv.FormatInt(sequenceNumber, 10)
	}
	if dedupeID := strings.TrimSpace(message.Metadata["MessageDeduplicationId"]); dedupeID != "" {
		attrs["MessageDeduplicationId"] = dedupeID
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func receivedMessageAttributesValues(message domain.Message) map[string]contracts.MessageAttributeValue {
	if len(message.Attributes) == 0 {
		return nil
	}
	received := make(map[string]contracts.MessageAttributeValue, len(message.Attributes))
	for key, value := range message.Attributes {
		received[key] = contracts.MessageAttributeValue{
			DataType:    "String",
			StringValue: value,
		}
	}
	return received
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
