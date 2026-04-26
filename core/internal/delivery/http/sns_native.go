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
	case "AddPermission":
		h.handleAddPermission(c, ctx)
	case "RemovePermission":
		h.handleRemovePermission(c, ctx)
	case "TagResource":
		h.handleTagResource(c, ctx)
	case "UntagResource":
		h.handleUntagResource(c, ctx)
	case "ListTagsForResource":
		h.handleListTagsForResource(c, ctx)
	case "GetDataProtectionPolicy":
		h.handleGetDataProtectionPolicy(c, ctx)
	case "PutDataProtectionPolicy":
		h.handlePutDataProtectionPolicy(c, ctx)
	case "CreatePlatformApplication":
		h.handleCreatePlatformApplication(c, ctx)
	case "DeletePlatformApplication":
		h.handleDeletePlatformApplication(c, ctx)
	case "GetPlatformApplicationAttributes":
		h.handleGetPlatformApplicationAttributes(c, ctx)
	case "SetPlatformApplicationAttributes":
		h.handleSetPlatformApplicationAttributes(c, ctx)
	case "ListPlatformApplications":
		h.handleListPlatformApplications(c, ctx)
	case "CreatePlatformEndpoint":
		h.handleCreatePlatformEndpoint(c, ctx)
	case "DeleteEndpoint":
		h.handleDeleteEndpoint(c, ctx)
	case "GetEndpointAttributes":
		h.handleGetEndpointAttributes(c, ctx)
	case "SetEndpointAttributes":
		h.handleSetEndpointAttributes(c, ctx)
	case "ListEndpointsByPlatformApplication":
		h.handleListEndpointsByPlatformApplication(c, ctx)
	case "SetSMSAttributes":
		h.handleSetSMSAttributes(c, ctx)
	case "GetSMSAttributes":
		h.handleGetSMSAttributes(c, ctx)
	case "CheckIfPhoneNumberIsOptedOut":
		h.handleCheckIfPhoneNumberIsOptedOut(c, ctx)
	case "OptInPhoneNumber":
		h.handleOptInPhoneNumber(c, ctx)
	case "ListPhoneNumbersOptedOut":
		h.handleListPhoneNumbersOptedOut(c, ctx)
	case "ListOriginationNumbers":
		h.handleListOriginationNumbers(c, ctx)
	case "GetSMSSandboxAccountStatus":
		h.handleGetSMSSandboxAccountStatus(c, ctx)
	case "CreateSMSSandboxPhoneNumber":
		h.handleCreateSMSSandboxPhoneNumber(c, ctx)
	case "VerifySMSSandboxPhoneNumber":
		h.handleVerifySMSSandboxPhoneNumber(c, ctx)
	case "DeleteSMSSandboxPhoneNumber":
		h.handleDeleteSMSSandboxPhoneNumber(c, ctx)
	case "ListSMSSandboxPhoneNumbers":
		h.handleListSMSSandboxPhoneNumbers(c, ctx)
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

func (h SNSNativeHandler) handleAddPermission(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	label := strings.TrimSpace(ctx.Values.Get("Label"))
	if err := h.service.AddPermission(topicARN, label, snsPermissionAWSAccountIDsFromValues(ctx.Values), snsPermissionActionNamesFromValues(ctx.Values)); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "AddPermission")
}

func (h SNSNativeHandler) handleRemovePermission(c *gin.Context, ctx SNSRequestContext) {
	topicARN := strings.TrimSpace(ctx.Values.Get("TopicArn"))
	label := strings.TrimSpace(ctx.Values.Get("Label"))
	if err := h.service.RemovePermission(topicARN, label); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "RemovePermission")
}

func (h SNSNativeHandler) handleTagResource(c *gin.Context, ctx SNSRequestContext) {
	resourceARN := strings.TrimSpace(ctx.Values.Get("ResourceArn"))
	tags := snsMapEntriesFromPrefixes(ctx.Values, "Tags.member.", "Tags.entry.")
	if err := h.service.TagResource(resourceARN, tags); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "TagResource")
}

func (h SNSNativeHandler) handleUntagResource(c *gin.Context, ctx SNSRequestContext) {
	resourceARN := strings.TrimSpace(ctx.Values.Get("ResourceArn"))
	if err := h.service.UntagResource(resourceARN, snsTagKeysFromValues(ctx.Values)); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "UntagResource")
}

func (h SNSNativeHandler) handleListTagsForResource(c *gin.Context, ctx SNSRequestContext) {
	resourceARN := strings.TrimSpace(ctx.Values.Get("ResourceArn"))
	tags, err := h.service.ListTagsForResource(resourceARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListTagsForResourceResponse(c, tags)
}

func (h SNSNativeHandler) handleGetDataProtectionPolicy(c *gin.Context, ctx SNSRequestContext) {
	resourceARN := strings.TrimSpace(ctx.Values.Get("ResourceArn"))
	policyDocument, err := h.service.GetDataProtectionPolicy(resourceARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSGetDataProtectionPolicyResponse(c, policyDocument)
}

func (h SNSNativeHandler) handlePutDataProtectionPolicy(c *gin.Context, ctx SNSRequestContext) {
	resourceARN := strings.TrimSpace(ctx.Values.Get("ResourceArn"))
	policyDocument := strings.TrimSpace(ctx.Values.Get("DataProtectionPolicy"))
	if err := h.service.PutDataProtectionPolicy(resourceARN, policyDocument); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "PutDataProtectionPolicy")
}

func (h SNSNativeHandler) handleCreatePlatformApplication(c *gin.Context, ctx SNSRequestContext) {
	name := strings.TrimSpace(ctx.Values.Get("Name"))
	platform := strings.TrimSpace(ctx.Values.Get("Platform"))
	attributes := snsMapEntriesFromAnyPrefix(ctx.Values, "Attributes")

	application, err := h.service.CreatePlatformApplication(name, platform, attributes)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSCreatePlatformApplicationResponse(c, application.ARN)
}

func (h SNSNativeHandler) handleDeletePlatformApplication(c *gin.Context, ctx SNSRequestContext) {
	platformApplicationARN := strings.TrimSpace(ctx.Values.Get("PlatformApplicationArn"))
	if err := h.service.DeletePlatformApplication(platformApplicationARN); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "DeletePlatformApplication")
}

func (h SNSNativeHandler) handleGetPlatformApplicationAttributes(c *gin.Context, ctx SNSRequestContext) {
	platformApplicationARN := strings.TrimSpace(ctx.Values.Get("PlatformApplicationArn"))
	attributes, err := h.service.GetPlatformApplicationAttributes(platformApplicationARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSAttributesActionResponse(c, "GetPlatformApplicationAttributes", "GetPlatformApplicationAttributesResult", attributes)
}

func (h SNSNativeHandler) handleSetPlatformApplicationAttributes(c *gin.Context, ctx SNSRequestContext) {
	platformApplicationARN := strings.TrimSpace(ctx.Values.Get("PlatformApplicationArn"))
	attributes := snsMapEntriesFromAnyPrefix(ctx.Values, "Attributes")
	if _, err := h.service.SetPlatformApplicationAttributes(platformApplicationARN, attributes); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "SetPlatformApplicationAttributes")
}

func (h SNSNativeHandler) handleListPlatformApplications(c *gin.Context, ctx SNSRequestContext) {
	applications, nextToken, err := h.service.ListPlatformApplications(snsNextTokenFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListPlatformApplicationsResponse(c, applications, nextToken)
}

func (h SNSNativeHandler) handleCreatePlatformEndpoint(c *gin.Context, ctx SNSRequestContext) {
	platformApplicationARN := strings.TrimSpace(ctx.Values.Get("PlatformApplicationArn"))
	token := strings.TrimSpace(ctx.Values.Get("Token"))
	customUserData := strings.TrimSpace(ctx.Values.Get("CustomUserData"))
	attributes := snsMapEntriesFromAnyPrefix(ctx.Values, "Attributes")

	endpoint, err := h.service.CreatePlatformEndpoint(platformApplicationARN, token, customUserData, attributes)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSCreatePlatformEndpointResponse(c, endpoint.ARN)
}

func (h SNSNativeHandler) handleDeleteEndpoint(c *gin.Context, ctx SNSRequestContext) {
	endpointARN := strings.TrimSpace(ctx.Values.Get("EndpointArn"))
	if err := h.service.DeleteEndpoint(endpointARN); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "DeleteEndpoint")
}

func (h SNSNativeHandler) handleGetEndpointAttributes(c *gin.Context, ctx SNSRequestContext) {
	endpointARN := strings.TrimSpace(ctx.Values.Get("EndpointArn"))
	attributes, err := h.service.GetEndpointAttributes(endpointARN)
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSAttributesActionResponse(c, "GetEndpointAttributes", "GetEndpointAttributesResult", attributes)
}

func (h SNSNativeHandler) handleSetEndpointAttributes(c *gin.Context, ctx SNSRequestContext) {
	endpointARN := strings.TrimSpace(ctx.Values.Get("EndpointArn"))
	attributes := snsMapEntriesFromAnyPrefix(ctx.Values, "Attributes")
	if _, err := h.service.SetEndpointAttributes(endpointARN, attributes); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "SetEndpointAttributes")
}

func (h SNSNativeHandler) handleListEndpointsByPlatformApplication(c *gin.Context, ctx SNSRequestContext) {
	platformApplicationARN := strings.TrimSpace(ctx.Values.Get("PlatformApplicationArn"))
	endpoints, nextToken, err := h.service.ListEndpointsByPlatformApplication(platformApplicationARN, snsNextTokenFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListEndpointsByPlatformApplicationResponse(c, endpoints, nextToken)
}

func (h SNSNativeHandler) handleSetSMSAttributes(c *gin.Context, ctx SNSRequestContext) {
	attributes := snsMapEntriesFromAnyPrefix(ctx.Values, "attributes", "Attributes")
	if err := h.service.SetSMSAttributes(attributes); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "SetSMSAttributes")
}

func (h SNSNativeHandler) handleGetSMSAttributes(c *gin.Context, ctx SNSRequestContext) {
	attributes, err := h.service.GetSMSAttributes(snsSMSAttributeNamesFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSAttributesActionResponse(c, "GetSMSAttributes", "GetSMSAttributesResult", attributes)
}

func (h SNSNativeHandler) handleCheckIfPhoneNumberIsOptedOut(c *gin.Context, ctx SNSRequestContext) {
	isOptedOut, err := h.service.CheckIfPhoneNumberIsOptedOut(snsPhoneNumberFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSCheckIfPhoneNumberIsOptedOutResponse(c, isOptedOut)
}

func (h SNSNativeHandler) handleOptInPhoneNumber(c *gin.Context, ctx SNSRequestContext) {
	if err := h.service.OptInPhoneNumber(snsPhoneNumberFromValues(ctx.Values)); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "OptInPhoneNumber")
}

func (h SNSNativeHandler) handleListPhoneNumbersOptedOut(c *gin.Context, ctx SNSRequestContext) {
	phoneNumbers, nextToken, err := h.service.ListPhoneNumbersOptedOut(snsNextTokenFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSPhoneNumberListResponse(c, "ListPhoneNumbersOptedOut", phoneNumbers, nextToken)
}

func (h SNSNativeHandler) handleListOriginationNumbers(c *gin.Context, ctx SNSRequestContext) {
	phoneNumbers, nextToken, err := h.service.ListOriginationNumbers(snsNextTokenFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSPhoneNumberListResponse(c, "ListOriginationNumbers", phoneNumbers, nextToken)
}

func (h SNSNativeHandler) handleGetSMSSandboxAccountStatus(c *gin.Context, _ SNSRequestContext) {
	isInSandbox, err := h.service.GetSMSSandboxAccountStatus()
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSGetSMSSandboxAccountStatusResponse(c, isInSandbox)
}

func (h SNSNativeHandler) handleCreateSMSSandboxPhoneNumber(c *gin.Context, ctx SNSRequestContext) {
	phoneNumber := snsPhoneNumberFromValues(ctx.Values)
	languageCode := strings.TrimSpace(ctx.Values.Get("LanguageCode"))
	if err := h.service.CreateSMSSandboxPhoneNumber(phoneNumber, languageCode); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "CreateSMSSandboxPhoneNumber")
}

func (h SNSNativeHandler) handleVerifySMSSandboxPhoneNumber(c *gin.Context, ctx SNSRequestContext) {
	phoneNumber := snsPhoneNumberFromValues(ctx.Values)
	oneTimePassword := strings.TrimSpace(ctx.Values.Get("OneTimePassword"))
	if err := h.service.VerifySMSSandboxPhoneNumber(phoneNumber, oneTimePassword); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "VerifySMSSandboxPhoneNumber")
}

func (h SNSNativeHandler) handleDeleteSMSSandboxPhoneNumber(c *gin.Context, ctx SNSRequestContext) {
	if err := h.service.DeleteSMSSandboxPhoneNumber(snsPhoneNumberFromValues(ctx.Values)); err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSNoResultActionResponse(c, "DeleteSMSSandboxPhoneNumber")
}

func (h SNSNativeHandler) handleListSMSSandboxPhoneNumbers(c *gin.Context, ctx SNSRequestContext) {
	phoneNumbers, nextToken, err := h.service.ListSMSSandboxPhoneNumbers(snsNextTokenFromValues(ctx.Values))
	if err != nil {
		writeSNSError(c, err, snsRequestIDFromContext(c))
		return
	}
	writeSNSListSMSSandboxPhoneNumbersResponse(c, phoneNumbers, nextToken)
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

type snsListTagsForResourceResponse struct {
	XMLName                   xml.Name                     `xml:"ListTagsForResourceResponse"`
	XMLNS                     string                       `xml:"xmlns,attr"`
	ListTagsForResourceResult snsListTagsForResourceResult `xml:"ListTagsForResourceResult"`
	ResponseMetadata          snsResponseMetadata          `xml:"ResponseMetadata"`
}

type snsListTagsForResourceResult struct {
	Tags snsTagMembers `xml:"Tags"`
}

type snsTagMembers struct {
	Members []snsTagMember `xml:"member"`
}

type snsTagMember struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type snsGetDataProtectionPolicyResponse struct {
	XMLName                       xml.Name                         `xml:"GetDataProtectionPolicyResponse"`
	XMLNS                         string                           `xml:"xmlns,attr"`
	GetDataProtectionPolicyResult snsGetDataProtectionPolicyResult `xml:"GetDataProtectionPolicyResult"`
	ResponseMetadata              snsResponseMetadata              `xml:"ResponseMetadata"`
}

type snsGetDataProtectionPolicyResult struct {
	DataProtectionPolicy string `xml:"DataProtectionPolicy"`
}

type snsCreatePlatformApplicationResponse struct {
	XMLName                         xml.Name                           `xml:"CreatePlatformApplicationResponse"`
	XMLNS                           string                             `xml:"xmlns,attr"`
	CreatePlatformApplicationResult snsCreatePlatformApplicationResult `xml:"CreatePlatformApplicationResult"`
	ResponseMetadata                snsResponseMetadata                `xml:"ResponseMetadata"`
}

type snsCreatePlatformApplicationResult struct {
	PlatformApplicationARN string `xml:"PlatformApplicationArn"`
}

type snsListPlatformApplicationsResponse struct {
	XMLName                        xml.Name                          `xml:"ListPlatformApplicationsResponse"`
	XMLNS                          string                            `xml:"xmlns,attr"`
	ListPlatformApplicationsResult snsListPlatformApplicationsResult `xml:"ListPlatformApplicationsResult"`
	ResponseMetadata               snsResponseMetadata               `xml:"ResponseMetadata"`
}

type snsListPlatformApplicationsResult struct {
	PlatformApplications snsPlatformApplicationMembers `xml:"PlatformApplications"`
	NextToken            string                        `xml:"NextToken,omitempty"`
}

type snsPlatformApplicationMembers struct {
	Members []snsPlatformApplicationMember `xml:"member"`
}

type snsPlatformApplicationMember struct {
	PlatformApplicationARN string `xml:"PlatformApplicationArn"`
}

type snsCreatePlatformEndpointResponse struct {
	XMLName                      xml.Name                        `xml:"CreatePlatformEndpointResponse"`
	XMLNS                        string                          `xml:"xmlns,attr"`
	CreatePlatformEndpointResult snsCreatePlatformEndpointResult `xml:"CreatePlatformEndpointResult"`
	ResponseMetadata             snsResponseMetadata             `xml:"ResponseMetadata"`
}

type snsCreatePlatformEndpointResult struct {
	EndpointARN string `xml:"EndpointArn"`
}

type snsListEndpointsByPlatformApplicationResponse struct {
	XMLName                                  xml.Name                                    `xml:"ListEndpointsByPlatformApplicationResponse"`
	XMLNS                                    string                                      `xml:"xmlns,attr"`
	ListEndpointsByPlatformApplicationResult snsListEndpointsByPlatformApplicationResult `xml:"ListEndpointsByPlatformApplicationResult"`
	ResponseMetadata                         snsResponseMetadata                         `xml:"ResponseMetadata"`
}

type snsListEndpointsByPlatformApplicationResult struct {
	Endpoints snsPlatformEndpointMembers `xml:"Endpoints"`
	NextToken string                     `xml:"NextToken,omitempty"`
}

type snsPlatformEndpointMembers struct {
	Members []snsPlatformEndpointMember `xml:"member"`
}

type snsPlatformEndpointMember struct {
	EndpointARN string `xml:"EndpointArn"`
}

type snsAttributesActionResponse struct {
	XMLName          xml.Name            `xml:""`
	XMLNS            string              `xml:"xmlns,attr"`
	Result           snsAttributesResult `xml:""`
	ResponseMetadata snsResponseMetadata `xml:"ResponseMetadata"`
}

type snsAttributesResult struct {
	XMLName    xml.Name            `xml:""`
	Attributes snsAttributeEntries `xml:"Attributes"`
}

type snsCheckIfPhoneNumberIsOptedOutResponse struct {
	XMLName                            xml.Name                              `xml:"CheckIfPhoneNumberIsOptedOutResponse"`
	XMLNS                              string                                `xml:"xmlns,attr"`
	CheckIfPhoneNumberIsOptedOutResult snsCheckIfPhoneNumberIsOptedOutResult `xml:"CheckIfPhoneNumberIsOptedOutResult"`
	ResponseMetadata                   snsResponseMetadata                   `xml:"ResponseMetadata"`
}

type snsCheckIfPhoneNumberIsOptedOutResult struct {
	IsOptedOut bool `xml:"isOptedOut"`
}

type snsPhoneNumberListResponse struct {
	XMLName          xml.Name                 `xml:""`
	XMLNS            string                   `xml:"xmlns,attr"`
	Result           snsPhoneNumberListResult `xml:""`
	ResponseMetadata snsResponseMetadata      `xml:"ResponseMetadata"`
}

type snsPhoneNumberListResult struct {
	XMLName      xml.Name              `xml:""`
	PhoneNumbers snsPhoneNumberMembers `xml:"phoneNumbers"`
	NextToken    string                `xml:"nextToken,omitempty"`
}

type snsPhoneNumberMembers struct {
	Members []snsPhoneNumberMember `xml:"member"`
}

type snsPhoneNumberMember struct {
	PhoneNumber string `xml:",chardata"`
}

type snsGetSMSSandboxAccountStatusResponse struct {
	XMLName                          xml.Name                            `xml:"GetSMSSandboxAccountStatusResponse"`
	XMLNS                            string                              `xml:"xmlns,attr"`
	GetSMSSandboxAccountStatusResult snsGetSMSSandboxAccountStatusResult `xml:"GetSMSSandboxAccountStatusResult"`
	ResponseMetadata                 snsResponseMetadata                 `xml:"ResponseMetadata"`
}

type snsGetSMSSandboxAccountStatusResult struct {
	IsInSandbox bool `xml:"IsInSandbox"`
}

type snsListSMSSandboxPhoneNumbersResponse struct {
	XMLName                          xml.Name                            `xml:"ListSMSSandboxPhoneNumbersResponse"`
	XMLNS                            string                              `xml:"xmlns,attr"`
	ListSMSSandboxPhoneNumbersResult snsListSMSSandboxPhoneNumbersResult `xml:"ListSMSSandboxPhoneNumbersResult"`
	ResponseMetadata                 snsResponseMetadata                 `xml:"ResponseMetadata"`
}

type snsListSMSSandboxPhoneNumbersResult struct {
	PhoneNumbers snsSMSSandboxPhoneNumberMembers `xml:"PhoneNumbers"`
	NextToken    string                          `xml:"NextToken,omitempty"`
}

type snsSMSSandboxPhoneNumberMembers struct {
	Members []snsSMSSandboxPhoneNumberMember `xml:"member"`
}

type snsSMSSandboxPhoneNumberMember struct {
	PhoneNumber string `xml:"PhoneNumber"`
	Status      string `xml:"Status"`
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

func writeSNSListTagsForResourceResponse(c *gin.Context, tags map[string]string) {
	if c == nil {
		return
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	members := make([]snsTagMember, 0, len(keys))
	for _, key := range keys {
		members = append(members, snsTagMember{Key: key, Value: tags[key]})
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsListTagsForResourceResponse{
		XMLNS: snsXMLNamespace,
		ListTagsForResourceResult: snsListTagsForResourceResult{
			Tags: snsTagMembers{Members: members},
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSGetDataProtectionPolicyResponse(c *gin.Context, policyDocument string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsGetDataProtectionPolicyResponse{
		XMLNS: snsXMLNamespace,
		GetDataProtectionPolicyResult: snsGetDataProtectionPolicyResult{
			DataProtectionPolicy: strings.TrimSpace(policyDocument),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSCreatePlatformApplicationResponse(c *gin.Context, platformApplicationARN string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsCreatePlatformApplicationResponse{
		XMLNS: snsXMLNamespace,
		CreatePlatformApplicationResult: snsCreatePlatformApplicationResult{
			PlatformApplicationARN: strings.TrimSpace(platformApplicationARN),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSListPlatformApplicationsResponse(c *gin.Context, applications []domain.PlatformApplication, nextToken string) {
	if c == nil {
		return
	}
	members := make([]snsPlatformApplicationMember, 0, len(applications))
	for _, application := range applications {
		members = append(members, snsPlatformApplicationMember{
			PlatformApplicationARN: application.ARN,
		})
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsListPlatformApplicationsResponse{
		XMLNS: snsXMLNamespace,
		ListPlatformApplicationsResult: snsListPlatformApplicationsResult{
			PlatformApplications: snsPlatformApplicationMembers{Members: members},
			NextToken:            strings.TrimSpace(nextToken),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSCreatePlatformEndpointResponse(c *gin.Context, endpointARN string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsCreatePlatformEndpointResponse{
		XMLNS: snsXMLNamespace,
		CreatePlatformEndpointResult: snsCreatePlatformEndpointResult{
			EndpointARN: strings.TrimSpace(endpointARN),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSListEndpointsByPlatformApplicationResponse(c *gin.Context, endpoints []domain.PlatformEndpoint, nextToken string) {
	if c == nil {
		return
	}
	members := make([]snsPlatformEndpointMember, 0, len(endpoints))
	for _, endpoint := range endpoints {
		members = append(members, snsPlatformEndpointMember{EndpointARN: endpoint.ARN})
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsListEndpointsByPlatformApplicationResponse{
		XMLNS: snsXMLNamespace,
		ListEndpointsByPlatformApplicationResult: snsListEndpointsByPlatformApplicationResult{
			Endpoints: snsPlatformEndpointMembers{Members: members},
			NextToken: strings.TrimSpace(nextToken),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSAttributesActionResponse(c *gin.Context, action, resultElement string, attributes map[string]string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsAttributesActionResponse{
		XMLName: xml.Name{Local: action + "Response"},
		XMLNS:   snsXMLNamespace,
		Result: snsAttributesResult{
			XMLName:    xml.Name{Local: resultElement},
			Attributes: snsAttributeEntries{Entries: snsAttributeEntriesFromMap(attributes)},
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSCheckIfPhoneNumberIsOptedOutResponse(c *gin.Context, isOptedOut bool) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsCheckIfPhoneNumberIsOptedOutResponse{
		XMLNS: snsXMLNamespace,
		CheckIfPhoneNumberIsOptedOutResult: snsCheckIfPhoneNumberIsOptedOutResult{
			IsOptedOut: isOptedOut,
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSPhoneNumberListResponse(c *gin.Context, action string, phoneNumbers []string, nextToken string) {
	if c == nil {
		return
	}
	members := make([]snsPhoneNumberMember, 0, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		members = append(members, snsPhoneNumberMember{PhoneNumber: phoneNumber})
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsPhoneNumberListResponse{
		XMLName: xml.Name{Local: action + "Response"},
		XMLNS:   snsXMLNamespace,
		Result: snsPhoneNumberListResult{
			XMLName:      xml.Name{Local: action + "Result"},
			PhoneNumbers: snsPhoneNumberMembers{Members: members},
			NextToken:    strings.TrimSpace(nextToken),
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSGetSMSSandboxAccountStatusResponse(c *gin.Context, isInSandbox bool) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsGetSMSSandboxAccountStatusResponse{
		XMLNS: snsXMLNamespace,
		GetSMSSandboxAccountStatusResult: snsGetSMSSandboxAccountStatusResult{
			IsInSandbox: isInSandbox,
		},
		ResponseMetadata: snsResponseMetadata{RequestID: snsRequestIDFromContext(c)},
	})
}

func writeSNSListSMSSandboxPhoneNumbersResponse(c *gin.Context, phoneNumbers []domain.SMSSandboxPhoneNumber, nextToken string) {
	if c == nil {
		return
	}
	members := make([]snsSMSSandboxPhoneNumberMember, 0, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		members = append(members, snsSMSSandboxPhoneNumberMember{
			PhoneNumber: phoneNumber.PhoneNumber,
			Status:      phoneNumber.Status,
		})
	}
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, snsListSMSSandboxPhoneNumbersResponse{
		XMLNS: snsXMLNamespace,
		ListSMSSandboxPhoneNumbersResult: snsListSMSSandboxPhoneNumbersResult{
			PhoneNumbers: snsSMSSandboxPhoneNumberMembers{Members: members},
			NextToken:    strings.TrimSpace(nextToken),
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
	return snsMapEntriesFromPrefixes(values, prefix)
}

func snsMapEntriesFromAnyPrefix(values url.Values, roots ...string) map[string]string {
	if len(roots) == 0 {
		return map[string]string{}
	}
	prefixes := make([]string, 0, len(roots)*2)
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		prefixes = append(prefixes, trimmed+".entry.", trimmed+".member.")
	}
	return snsMapEntriesFromPrefixes(values, prefixes...)
}

func snsMapEntriesFromPrefixes(values url.Values, prefixes ...string) map[string]string {
	if len(prefixes) == 0 {
		return map[string]string{}
	}

	indexed := map[int]snsAttributeEntry{}
	for rawKey, bucket := range values {
		if len(bucket) == 0 {
			continue
		}
		matchedPrefix := ""
		for _, prefix := range prefixes {
			if strings.HasPrefix(rawKey, prefix) {
				matchedPrefix = prefix
				break
			}
		}
		if matchedPrefix == "" {
			continue
		}
		suffix := strings.TrimPrefix(rawKey, matchedPrefix)
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
		switch strings.ToLower(parts[1]) {
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
