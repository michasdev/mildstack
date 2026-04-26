package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	topicNameMaxLength = 256
)

var topicBasePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Topic models an SNS topic persisted in the local runtime.
type Topic struct {
	ARN        string
	Name       string
	TenantKey  string
	Attributes map[string]string
	PolicyJSON string
	Tags       map[string]string
	IsFIFO     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewTopic(tenant Tenant, name string, attributes map[string]string, now time.Time) (Topic, error) {
	normalizedName, isFIFO, err := normalizeTopicName(name)
	if err != nil {
		return Topic{}, err
	}

	attrs, err := normalizeTopicAttributes(attributes, isFIFO)
	if err != nil {
		return Topic{}, err
	}

	now = normalizeTimestamp(now)
	return Topic{
		ARN:        tenant.TopicARN(normalizedName),
		Name:       normalizedName,
		TenantKey:  tenant.Key(),
		Attributes: attrs,
		PolicyJSON: "{}",
		Tags:       map[string]string{},
		IsFIFO:     isFIFO,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (t Topic) WithAttribute(attributeName, attributeValue string, now time.Time) (Topic, error) {
	attributeName = strings.TrimSpace(attributeName)
	if attributeName == "" {
		return Topic{}, fmt.Errorf("%w: attribute name is required", ErrValidation)
	}

	updated := t
	updated.Attributes = cloneStringMap(t.Attributes)
	if updated.Attributes == nil {
		updated.Attributes = map[string]string{}
	}
	updated.Attributes[attributeName] = strings.TrimSpace(attributeValue)

	attrs, err := normalizeTopicAttributes(updated.Attributes, updated.IsFIFO)
	if err != nil {
		return Topic{}, err
	}
	updated.Attributes = attrs
	updated.UpdatedAt = normalizeTimestamp(now)
	return updated, nil
}

func (t Topic) AttributesView(ownerAccountID string, subscriptionsConfirmed, subscriptionsPending, subscriptionsDeleted int) map[string]string {
	view := cloneStringMap(t.Attributes)
	if view == nil {
		view = map[string]string{}
	}
	view["TopicArn"] = t.ARN
	view["Owner"] = strings.TrimSpace(ownerAccountID)
	if policy := strings.TrimSpace(t.PolicyJSON); policy != "" && policy != "{}" {
		view["Policy"] = policy
	}
	if t.IsFIFO {
		view["FifoTopic"] = "true"
	} else if _, ok := view["FifoTopic"]; !ok {
		view["FifoTopic"] = "false"
	}
	view["SubscriptionsConfirmed"] = fmt.Sprintf("%d", subscriptionsConfirmed)
	view["SubscriptionsPending"] = fmt.Sprintf("%d", subscriptionsPending)
	view["SubscriptionsDeleted"] = fmt.Sprintf("%d", subscriptionsDeleted)
	return view
}

func normalizeTopicName(name string) (string, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, fmt.Errorf("%w: topic name is required", ErrValidation)
	}
	if len(name) > topicNameMaxLength {
		return "", false, fmt.Errorf("%w: topic name exceeds %d characters", ErrValidation, topicNameMaxLength)
	}

	isFIFO := strings.HasSuffix(name, ".fifo")
	base := name
	if isFIFO {
		base = strings.TrimSuffix(name, ".fifo")
	}

	if base == "" || !topicBasePattern.MatchString(base) {
		return "", false, fmt.Errorf("%w: topic name contains unsupported characters", ErrValidation)
	}

	if strings.Contains(name, ".") && !isFIFO {
		return "", false, fmt.Errorf("%w: standard topic names cannot contain periods", ErrValidation)
	}

	return name, isFIFO, nil
}

func normalizeTopicAttributes(attributes map[string]string, isFIFO bool) (map[string]string, error) {
	normalized := cloneStringMap(attributes)
	if normalized == nil {
		normalized = map[string]string{}
	}

	if fifoValue, ok := normalized["FifoTopic"]; ok {
		isFIFORequested := parseTruthyString(fifoValue)
		if isFIFORequested != isFIFO {
			return nil, fmt.Errorf("%w: FifoTopic must match topic name suffix", ErrValidation)
		}
	}

	if !isFIFO {
		if dedupValue, ok := normalized["ContentBasedDeduplication"]; ok && parseTruthyString(dedupValue) {
			return nil, fmt.Errorf("%w: ContentBasedDeduplication is only valid for FIFO topics", ErrValidation)
		}
		normalized["FifoTopic"] = "false"
		return normalized, nil
	}

	normalized["FifoTopic"] = "true"
	if value, ok := normalized["ContentBasedDeduplication"]; ok {
		if parseTruthyString(value) {
			normalized["ContentBasedDeduplication"] = "true"
		} else {
			normalized["ContentBasedDeduplication"] = "false"
		}
	}
	return normalized, nil
}

func parseTruthyString(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func normalizeTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	copied := make(map[string]string, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}
