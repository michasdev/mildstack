package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SubscriptionStatusPendingConfirmation = "PendingConfirmation"
	SubscriptionStatusConfirmed           = "Confirmed"
	SubscriptionStatusDeleted             = "Deleted"
)

var supportedSubscriptionProtocols = map[string]struct{}{
	"http":        {},
	"https":       {},
	"email":       {},
	"email-json":  {},
	"sms":         {},
	"sqs":         {},
	"application": {},
	"lambda":      {},
	"firehose":    {},
}

// Subscription models an SNS subscription persisted in the local runtime.
type Subscription struct {
	ARN            string
	TopicARN       string
	TenantKey      string
	Protocol       string
	Endpoint       string
	OwnerAccountID string
	Status         string
	Token          string
	Attributes     map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ConfirmedAt    *time.Time
}

type SubscribeOutput struct {
	Subscription         Subscription
	ResponseSubscription string
}

func NewSubscription(tenant Tenant, topicARN, protocol, endpoint string, attributes map[string]string, now time.Time) (Subscription, error) {
	topicARN = strings.TrimSpace(topicARN)
	if topicARN == "" {
		return Subscription{}, fmt.Errorf("%w: topic arn is required", ErrValidation)
	}

	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" {
		return Subscription{}, fmt.Errorf("%w: protocol is required", ErrValidation)
	}
	if _, ok := supportedSubscriptionProtocols[protocol]; !ok {
		return Subscription{}, fmt.Errorf("%w: unsupported protocol %q", ErrValidation, protocol)
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return Subscription{}, fmt.Errorf("%w: endpoint is required", ErrValidation)
	}

	now = normalizeTimestamp(now)
	attributes = cloneStringMap(attributes)
	if attributes == nil {
		attributes = map[string]string{}
	}

	subscriptionARN := fmt.Sprintf("%s:%s", topicARN, uuid.NewString())
	return Subscription{
		ARN:            subscriptionARN,
		TopicARN:       topicARN,
		TenantKey:      tenant.Key(),
		Protocol:       protocol,
		Endpoint:       endpoint,
		OwnerAccountID: tenant.AccountID,
		Status:         SubscriptionStatusPendingConfirmation,
		Token:          uuid.NewString(),
		Attributes:     attributes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s Subscription) WithAttribute(attributeName, attributeValue string, now time.Time) (Subscription, error) {
	attributeName = strings.TrimSpace(attributeName)
	if attributeName == "" {
		return Subscription{}, fmt.Errorf("%w: attribute name is required", ErrValidation)
	}

	updated := s
	updated.Attributes = cloneStringMap(s.Attributes)
	if updated.Attributes == nil {
		updated.Attributes = map[string]string{}
	}
	updated.Attributes[attributeName] = strings.TrimSpace(attributeValue)
	updated.UpdatedAt = normalizeTimestamp(now)
	return updated, nil
}

func (s Subscription) Confirm(token string, now time.Time) (Subscription, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Subscription{}, fmt.Errorf("%w: token is required", ErrValidation)
	}
	if strings.TrimSpace(s.Token) == "" || s.Token != token {
		return Subscription{}, ErrInvalidToken
	}
	if s.Status != SubscriptionStatusPendingConfirmation {
		return Subscription{}, fmt.Errorf("%w: subscription is not pending confirmation", ErrValidation)
	}

	updated := s
	updated.Status = SubscriptionStatusConfirmed
	updated.Token = ""
	now = normalizeTimestamp(now)
	updated.UpdatedAt = now
	updated.ConfirmedAt = &now
	return updated, nil
}

func (s Subscription) AttributesView() map[string]string {
	view := cloneStringMap(s.Attributes)
	if view == nil {
		view = map[string]string{}
	}
	view["SubscriptionArn"] = s.ARN
	view["TopicArn"] = s.TopicARN
	view["Protocol"] = s.Protocol
	view["Endpoint"] = s.Endpoint
	view["Owner"] = s.OwnerAccountID
	view["PendingConfirmation"] = boolString(s.Status == SubscriptionStatusPendingConfirmation)
	if s.Status == SubscriptionStatusConfirmed {
		view["ConfirmationWasAuthenticated"] = "false"
	}
	return view
}

func (s Subscription) ARNForList() string {
	if s.Status == SubscriptionStatusPendingConfirmation {
		return SubscriptionStatusPendingConfirmation
	}
	return s.ARN
}

func (s Subscription) SubscribeResponseARN(returnSubscriptionARN bool) string {
	if returnSubscriptionARN {
		return s.ARN
	}
	if s.Status == SubscriptionStatusPendingConfirmation {
		return "pending confirmation"
	}
	return s.ARN
}

func (s Subscription) IsConfirmed() bool {
	return s.Status == SubscriptionStatusConfirmed
}

func (s Subscription) RawMessageDeliveryEnabled() bool {
	return parseTruthyString(s.Attributes["RawMessageDelivery"])
}

func (s Subscription) FilterPolicy() string {
	return strings.TrimSpace(s.Attributes["FilterPolicy"])
}

func (s Subscription) FilterPolicyScope() string {
	scope := strings.TrimSpace(s.Attributes["FilterPolicyScope"])
	if scope == "" {
		return FilterPolicyScopeMessageAttributes
	}
	return scope
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
