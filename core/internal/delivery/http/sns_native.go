package http

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
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

	switch spec.Action {
	case "CreateTopic":
		h.handleCreateTopic(c, ctx)
	case "DeleteTopic":
		h.handleDeleteTopic(c, ctx)
	case "GetTopicAttributes":
		h.handleGetTopicAttributes(c, ctx)
	case "SetTopicAttributes":
		h.handleSetTopicAttributes(c, ctx)
	case "ListTopics":
		h.handleListTopics(c, ctx)
	case "Publish":
		h.handlePublish(c, ctx)
	case "PublishBatch":
		h.handlePublishBatch(c, ctx)
	case "Subscribe":
		h.handleSubscribe(c, ctx)
	case "ConfirmSubscription":
		h.handleConfirmSubscription(c, ctx)
	case "Unsubscribe":
		h.handleUnsubscribe(c, ctx)
	case "GetSubscriptionAttributes":
		h.handleGetSubscriptionAttributes(c, ctx)
	case "SetSubscriptionAttributes":
		h.handleSetSubscriptionAttributes(c, ctx)
	case "ListSubscriptions":
		h.handleListSubscriptions(c, ctx)
	case "ListSubscriptionsByTopic":
		h.handleListSubscriptionsByTopic(c, ctx)
	default:
		writeSNSError(c, ErrSNSUnsupported, snsRequestIDFromContext(c))
	}
	return true
}

func (h SNSNativeHandler) handleCreateTopic(c *gin.Context, ctx SNSRequestContext) {
	name := strings.TrimSpace(ctx.Values.Get("Name"))
	attributes := snsMapEntriesFromValues(ctx.Values, "Attributes.entry.")

	topic, err := h.service.CreateTopic(name, attributes)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSCreateTopicResponse(c, topic.ARN)
}

func (h SNSNativeHandler) handleDeleteTopic(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	if err := h.service.DeleteTopic(topicARN); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "DeleteTopic")
}

func (h SNSNativeHandler) handleGetTopicAttributes(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	attributes, err := h.service.GetTopicAttributes(topicARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSGetTopicAttributesResponse(c, attributes)
}

func (h SNSNativeHandler) handleSetTopicAttributes(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	attributeName := strings.TrimSpace(ctx.Values.Get("AttributeName"))
	attributeValue := strings.TrimSpace(ctx.Values.Get("AttributeValue"))

	if _, err := h.service.SetTopicAttributes(topicARN, attributeName, attributeValue); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "SetTopicAttributes")
}

func (h SNSNativeHandler) handleListTopics(c *gin.Context, ctx SNSRequestContext) {
	nextToken := strings.TrimSpace(ctx.Values.Get("NextToken"))
	topics, responseNextToken, err := h.service.ListTopics(nextToken)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListTopicsResponse(c, topics, responseNextToken)
}

func (h SNSNativeHandler) handlePublish(c *gin.Context, ctx SNSRequestContext) {
	request := snsPublishRequestFromValues(ctx.Values)
	result, err := h.service.Publish(request)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSPublishResponse(c, result)
}

func (h SNSNativeHandler) handlePublishBatch(c *gin.Context, ctx SNSRequestContext) {
	request := snsPublishBatchRequestFromValues(ctx.Values)
	result, err := h.service.PublishBatch(request)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSPublishBatchResponse(c, result)
}

func (h SNSNativeHandler) handleSubscribe(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	protocol := strings.TrimSpace(ctx.Values.Get("Protocol"))
	endpoint := strings.TrimSpace(ctx.Values.Get("Endpoint"))
	attributes := snsMapEntriesFromValues(ctx.Values, "Attributes.entry.")
	returnSubscriptionARN := snsBoolValue(ctx.Values, "ReturnSubscriptionArn")

	output, err := h.service.Subscribe(topicARN, protocol, endpoint, attributes, returnSubscriptionARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSSubscribeResponse(c, output.ResponseSubscription)
}

func (h SNSNativeHandler) handleConfirmSubscription(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	token := strings.TrimSpace(ctx.Values.Get("Token"))

	subscription, err := h.service.ConfirmSubscription(topicARN, token)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSConfirmSubscriptionResponse(c, subscription.ARN)
}

func (h SNSNativeHandler) handleUnsubscribe(c *gin.Context, ctx SNSRequestContext) {
	subscriptionARN := strings.TrimSpace(ctx.Values.Get("SubscriptionArn"))
	if err := h.service.Unsubscribe(subscriptionARN); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "Unsubscribe")
}

func (h SNSNativeHandler) handleGetSubscriptionAttributes(c *gin.Context, ctx SNSRequestContext) {
	subscriptionARN := strings.TrimSpace(ctx.Values.Get("SubscriptionArn"))
	attributes, err := h.service.GetSubscriptionAttributes(subscriptionARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSGetSubscriptionAttributesResponse(c, attributes)
}

func (h SNSNativeHandler) handleSetSubscriptionAttributes(c *gin.Context, ctx SNSRequestContext) {
	subscriptionARN := strings.TrimSpace(ctx.Values.Get("SubscriptionArn"))
	attributeName := strings.TrimSpace(ctx.Values.Get("AttributeName"))
	attributeValue := strings.TrimSpace(ctx.Values.Get("AttributeValue"))

	if _, err := h.service.SetSubscriptionAttributes(subscriptionARN, attributeName, attributeValue); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "SetSubscriptionAttributes")
}

func (h SNSNativeHandler) handleListSubscriptions(c *gin.Context, ctx SNSRequestContext) {
	nextToken := strings.TrimSpace(ctx.Values.Get("NextToken"))
	subscriptions, responseNextToken, err := h.service.ListSubscriptions(nextToken)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListSubscriptionsResponse(c, "ListSubscriptions", subscriptions, responseNextToken)
}

func (h SNSNativeHandler) handleListSubscriptionsByTopic(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	nextToken := strings.TrimSpace(ctx.Values.Get("NextToken"))

	subscriptions, responseNextToken, err := h.service.ListSubscriptionsByTopic(topicARN, nextToken)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListSubscriptionsResponse(c, "ListSubscriptionsByTopic", subscriptions, responseNextToken)
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

type snsResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type snsNoResultActionResponse struct {
	XMLName          xml.Name            `xml:""`
	XMLNS            string              `xml:"xmlns,attr"`
	ResponseMetadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsCreateTopicResponse struct {
	XMLName           xml.Name             `xml:"CreateTopicResponse"`
	XMLNS             string               `xml:"xmlns,attr"`
	CreateTopicResult snsCreateTopicResult `xml:"CreateTopicResult"`
	ResponseMetadata  snsResponseMetadata  `xml:"ResponseMetadata"`
}

type snsCreateTopicResult struct {
	TopicARN string `xml:"TopicArn"`
}

type snsListTopicsResponse struct {
	XMLName          xml.Name            `xml:"ListTopicsResponse"`
	XMLNS            string              `xml:"xmlns,attr"`
	ListTopicsResult snsListTopicsResult `xml:"ListTopicsResult"`
	ResponseMetadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsListTopicsResult struct {
	Topics    snsTopicMembers `xml:"Topics"`
	NextToken string          `xml:"NextToken,omitempty"`
}

type snsTopicMembers struct {
	Members []snsTopicMember `xml:"member"`
}

type snsTopicMember struct {
	TopicARN string `xml:"TopicArn"`
}

type snsGetTopicAttributesResponse struct {
	XMLName                  xml.Name                    `xml:"GetTopicAttributesResponse"`
	XMLNS                    string                      `xml:"xmlns,attr"`
	GetTopicAttributesResult snsGetTopicAttributesResult `xml:"GetTopicAttributesResult"`
	ResponseMetadata         snsResponseMetadata         `xml:"ResponseMetadata"`
}

type snsGetTopicAttributesResult struct {
	Attributes snsAttributeEntries `xml:"Attributes"`
}

type snsPublishResponse struct {
	XMLName          xml.Name            `xml:"PublishResponse"`
	XMLNS            string              `xml:"xmlns,attr"`
	PublishResult    snsPublishResult    `xml:"PublishResult"`
	ResponseMetadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsPublishResult struct {
	MessageID      string `xml:"MessageId"`
	SequenceNumber string `xml:"SequenceNumber,omitempty"`
}

type snsPublishBatchResponse struct {
	XMLName            xml.Name              `xml:"PublishBatchResponse"`
	XMLNS              string                `xml:"xmlns,attr"`
	PublishBatchResult snsPublishBatchResult `xml:"PublishBatchResult"`
	ResponseMetadata   snsResponseMetadata   `xml:"ResponseMetadata"`
}

type snsPublishBatchResult struct {
	Successful snsPublishBatchSuccessfulMembers `xml:"Successful"`
	Failed     snsPublishBatchFailedMembers     `xml:"Failed"`
}

type snsPublishBatchSuccessfulMembers struct {
	Members []snsPublishBatchSuccessfulMember `xml:"member,omitempty"`
}

type snsPublishBatchSuccessfulMember struct {
	ID             string `xml:"Id"`
	MessageID      string `xml:"MessageId"`
	SequenceNumber string `xml:"SequenceNumber,omitempty"`
}

type snsPublishBatchFailedMembers struct {
	Members []snsPublishBatchFailedMember `xml:"member,omitempty"`
}

type snsPublishBatchFailedMember struct {
	Code        string `xml:"Code"`
	ID          string `xml:"Id"`
	SenderFault bool   `xml:"SenderFault"`
	Message     string `xml:"Message,omitempty"`
}

type snsGetSubscriptionAttributesResponse struct {
	XMLName                         xml.Name                           `xml:"GetSubscriptionAttributesResponse"`
	XMLNS                           string                             `xml:"xmlns,attr"`
	GetSubscriptionAttributesResult snsGetSubscriptionAttributesResult `xml:"GetSubscriptionAttributesResult"`
	ResponseMetadata                snsResponseMetadata                `xml:"ResponseMetadata"`
}

type snsGetSubscriptionAttributesResult struct {
	Attributes snsAttributeEntries `xml:"Attributes"`
}

type snsAttributeEntries struct {
	Entries []snsAttributeEntry `xml:"entry"`
}

type snsAttributeEntry struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type snsSubscribeResponse struct {
	XMLName          xml.Name            `xml:"SubscribeResponse"`
	XMLNS            string              `xml:"xmlns,attr"`
	SubscribeResult  snsSubscribeResult  `xml:"SubscribeResult"`
	ResponseMetadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsSubscribeResult struct {
	SubscriptionARN string `xml:"SubscriptionArn"`
}

type snsConfirmSubscriptionResponse struct {
	XMLName                   xml.Name                     `xml:"ConfirmSubscriptionResponse"`
	XMLNS                     string                       `xml:"xmlns,attr"`
	ConfirmSubscriptionResult snsConfirmSubscriptionResult `xml:"ConfirmSubscriptionResult"`
	ResponseMetadata          snsResponseMetadata          `xml:"ResponseMetadata"`
}

type snsConfirmSubscriptionResult struct {
	SubscriptionARN string `xml:"SubscriptionArn"`
}

type snsListSubscriptionsResponse struct {
	XMLName                 xml.Name                   `xml:""`
	XMLNS                   string                     `xml:"xmlns,attr"`
	ListSubscriptionsResult snsListSubscriptionsResult `xml:"ListSubscriptionsResult"`
	ResponseMetadata        snsResponseMetadata        `xml:"ResponseMetadata"`
}

type snsListSubscriptionsResult struct {
	Subscriptions snsSubscriptionMembers `xml:"Subscriptions"`
	NextToken     string                 `xml:"NextToken,omitempty"`
}

type snsSubscriptionMembers struct {
	Members []snsSubscriptionMember `xml:"member"`
}

type snsSubscriptionMember struct {
	TopicARN        string `xml:"TopicArn"`
	Protocol        string `xml:"Protocol"`
	SubscriptionARN string `xml:"SubscriptionArn"`
	Owner           string `xml:"Owner"`
	Endpoint        string `xml:"Endpoint"`
}

func writeSNSNoResultActionResponse(c *gin.Context, action string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsNoResultActionResponse{
		XMLName:          xml.Name{Local: action + "Response"},
		XMLNS:            snsXMLNamespace,
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSCreateTopicResponse(c *gin.Context, topicARN string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsCreateTopicResponse{
		XMLNS: snsXMLNamespace,
		CreateTopicResult: snsCreateTopicResult{
			TopicARN: strings.TrimSpace(topicARN),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSListTopicsResponse(c *gin.Context, topics []domain.Topic, nextToken string) {
	if c == nil {
		return
	}
	members := make([]snsTopicMember, 0, len(topics))
	for _, topic := range topics {
		members = append(members, snsTopicMember{TopicARN: topic.ARN})
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsListTopicsResponse{
		XMLNS: snsXMLNamespace,
		ListTopicsResult: snsListTopicsResult{
			Topics:    snsTopicMembers{Members: members},
			NextToken: strings.TrimSpace(nextToken),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSPublishResponse(c *gin.Context, result domain.PublishResult) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsPublishResponse{
		XMLNS: snsXMLNamespace,
		PublishResult: snsPublishResult{
			MessageID:      strings.TrimSpace(result.MessageID),
			SequenceNumber: strings.TrimSpace(result.SequenceNumber),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSPublishBatchResponse(c *gin.Context, result domain.PublishBatchResult) {
	if c == nil {
		return
	}

	successfulMembers := make([]snsPublishBatchSuccessfulMember, 0, len(result.Successful))
	for _, successful := range result.Successful {
		successfulMembers = append(successfulMembers, snsPublishBatchSuccessfulMember{
			ID:             successful.ID,
			MessageID:      successful.MessageID,
			SequenceNumber: successful.SequenceNumber,
		})
	}

	failedMembers := make([]snsPublishBatchFailedMember, 0, len(result.Failed))
	for _, failed := range result.Failed {
		failedMembers = append(failedMembers, snsPublishBatchFailedMember{
			Code:        failed.Code,
			ID:          failed.ID,
			SenderFault: failed.SenderFault,
			Message:     failed.Message,
		})
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsPublishBatchResponse{
		XMLNS: snsXMLNamespace,
		PublishBatchResult: snsPublishBatchResult{
			Successful: snsPublishBatchSuccessfulMembers{Members: successfulMembers},
			Failed:     snsPublishBatchFailedMembers{Members: failedMembers},
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSGetTopicAttributesResponse(c *gin.Context, attributes map[string]string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsGetTopicAttributesResponse{
		XMLNS: snsXMLNamespace,
		GetTopicAttributesResult: snsGetTopicAttributesResult{
			Attributes: snsAttributeEntries{Entries: snsAttributeEntriesFromMap(attributes)},
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSGetSubscriptionAttributesResponse(c *gin.Context, attributes map[string]string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsGetSubscriptionAttributesResponse{
		XMLNS: snsXMLNamespace,
		GetSubscriptionAttributesResult: snsGetSubscriptionAttributesResult{
			Attributes: snsAttributeEntries{Entries: snsAttributeEntriesFromMap(attributes)},
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSSubscribeResponse(c *gin.Context, subscriptionARN string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsSubscribeResponse{
		XMLNS: snsXMLNamespace,
		SubscribeResult: snsSubscribeResult{
			SubscriptionARN: strings.TrimSpace(subscriptionARN),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSConfirmSubscriptionResponse(c *gin.Context, subscriptionARN string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsConfirmSubscriptionResponse{
		XMLNS: snsXMLNamespace,
		ConfirmSubscriptionResult: snsConfirmSubscriptionResult{
			SubscriptionARN: strings.TrimSpace(subscriptionARN),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSListSubscriptionsResponse(c *gin.Context, action string, subscriptions []domain.Subscription, nextToken string) {
	if c == nil {
		return
	}
	members := make([]snsSubscriptionMember, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		members = append(members, snsSubscriptionMember{
			TopicARN:        subscription.TopicARN,
			Protocol:        subscription.Protocol,
			SubscriptionARN: subscription.ARNForList(),
			Owner:           subscription.OwnerAccountID,
			Endpoint:        subscription.Endpoint,
		})
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsListSubscriptionsResponse{
		XMLName: xml.Name{Local: action + "Response"},
		XMLNS:   snsXMLNamespace,
		ListSubscriptionsResult: snsListSubscriptionsResult{
			Subscriptions: snsSubscriptionMembers{Members: members},
			NextToken:     strings.TrimSpace(nextToken),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func snsAttributeEntriesFromMap(values map[string]string) []snsAttributeEntry {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]snsAttributeEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, snsAttributeEntry{Key: key, Value: values[key]})
	}
	return entries
}

func snsMapEntriesFromValues(values url.Values, prefix string) map[string]string {
	indexed := map[int]snsAttributeEntry{}
	for rawKey, bucket := range values {
		if !strings.HasPrefix(rawKey, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(rawKey, prefix)
		parts := strings.Split(suffix, ".")
		if len(parts) != 2 {
			continue
		}
		index, err := strconv.Atoi(parts[0])
		if err != nil || index <= 0 {
			continue
		}
		value := ""
		if len(bucket) > 0 {
			value = strings.TrimSpace(bucket[0])
		}
		entry := indexed[index]
		switch parts[1] {
		case "key":
			entry.Key = value
		case "value":
			entry.Value = value
		default:
			continue
		}
		indexed[index] = entry
	}

	if len(indexed) == 0 {
		return map[string]string{}
	}
	indexes := make([]int, 0, len(indexed))
	for index := range indexed {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	result := make(map[string]string, len(indexed))
	for _, index := range indexes {
		entry := indexed[index]
		if strings.TrimSpace(entry.Key) == "" {
			continue
		}
		result[entry.Key] = entry.Value
	}
	return result
}

func snsBoolValue(values url.Values, key string) bool {
	value := strings.TrimSpace(strings.ToLower(values.Get(key)))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
