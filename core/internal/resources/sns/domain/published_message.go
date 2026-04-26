package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	PublishTargetKindTopic      = "Topic"
	PublishTargetKindTargetARN  = "TargetArn"
	PublishTargetKindPhone      = "PhoneNumber"
	messageStructurePlain       = "plain"
	messageStructureJSON        = "json"
	publishBatchEntryMaxEntries = 10
)

var publishBatchEntryIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// MessageAttributeValue models SNS message attribute data shape.
type MessageAttributeValue struct {
	DataType    string
	StringValue string
	BinaryValue string
}

// PublishedMessage models a persisted publish request accepted by SNS.
type PublishedMessage struct {
	MessageID              string
	TenantKey              string
	TargetKind             string
	TargetRef              string
	TopicARN               string
	Payload                string
	Subject                string
	MessageStructure       string
	MessageAttributes      map[string]MessageAttributeValue
	MessageGroupID         string
	MessageDeduplicationID string
	SequenceNumber         string
	PublishedAt            time.Time
}

// PublishRequest describes the API input for Publish action.
type PublishRequest struct {
	TopicARN               string
	TargetARN              string
	PhoneNumber            string
	Message                string
	Subject                string
	MessageStructure       string
	MessageAttributes      map[string]MessageAttributeValue
	MessageGroupID         string
	MessageDeduplicationID string
}

// PublishResult describes the API output for Publish action.
type PublishResult struct {
	MessageID      string
	SequenceNumber string
}

// PublishBatchRequestEntry describes one PublishBatch item.
type PublishBatchRequestEntry struct {
	ID                     string
	Message                string
	Subject                string
	MessageStructure       string
	MessageAttributes      map[string]MessageAttributeValue
	MessageGroupID         string
	MessageDeduplicationID string
}

// PublishBatchRequest describes PublishBatch input.
type PublishBatchRequest struct {
	TopicARN string
	Entries  []PublishBatchRequestEntry
}

// PublishBatchResultEntry describes one successful PublishBatch item.
type PublishBatchResultEntry struct {
	ID             string
	MessageID      string
	SequenceNumber string
}

// PublishBatchErrorEntry describes one failed PublishBatch item.
type PublishBatchErrorEntry struct {
	ID          string
	Code        string
	Message     string
	SenderFault bool
}

// PublishBatchResult describes PublishBatch output payload.
type PublishBatchResult struct {
	Successful []PublishBatchResultEntry
	Failed     []PublishBatchErrorEntry
}

func NewPublishedMessage(tenant Tenant, request PublishRequest, now time.Time) (PublishedMessage, error) {
	targetKind, targetRef, topicARN, err := resolvePublishTarget(request.TopicARN, request.TargetARN, request.PhoneNumber)
	if err != nil {
		return PublishedMessage{}, err
	}

	payload := strings.TrimSpace(request.Message)
	if payload == "" {
		return PublishedMessage{}, fmt.Errorf("%w: message is required", ErrValidation)
	}

	subject := strings.TrimSpace(request.Subject)
	if strings.ContainsAny(subject, "\r\n") {
		return PublishedMessage{}, fmt.Errorf("%w: subject cannot contain line breaks", ErrValidation)
	}
	if len(subject) > 100 {
		return PublishedMessage{}, fmt.Errorf("%w: subject exceeds 100 characters", ErrValidation)
	}

	messageStructure, err := normalizeMessageStructure(request.MessageStructure, payload)
	if err != nil {
		return PublishedMessage{}, err
	}

	messageAttributes, err := normalizeMessageAttributes(request.MessageAttributes)
	if err != nil {
		return PublishedMessage{}, err
	}

	now = normalizeTimestamp(now)
	return PublishedMessage{
		MessageID:              uuid.NewString(),
		TenantKey:              tenant.Key(),
		TargetKind:             targetKind,
		TargetRef:              targetRef,
		TopicARN:               topicARN,
		Payload:                payload,
		Subject:                subject,
		MessageStructure:       messageStructure,
		MessageAttributes:      messageAttributes,
		MessageGroupID:         strings.TrimSpace(request.MessageGroupID),
		MessageDeduplicationID: strings.TrimSpace(request.MessageDeduplicationID),
		PublishedAt:            now,
	}, nil
}

func ValidatePublishBatch(entries []PublishBatchRequestEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("%w: PublishBatchRequestEntries is required", ErrValidation)
	}
	if len(entries) > publishBatchEntryMaxEntries {
		return fmt.Errorf("%w: too many entries in batch request", ErrValidation)
	}

	seenIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if !publishBatchEntryIDPattern.MatchString(id) {
			return fmt.Errorf("%w: invalid batch entry id", ErrValidation)
		}
		if _, exists := seenIDs[id]; exists {
			return fmt.Errorf("%w: duplicate batch entry id", ErrValidation)
		}
		seenIDs[id] = struct{}{}
	}
	return nil
}

func resolvePublishTarget(topicARN, targetARN, phoneNumber string) (targetKind, targetRef, normalizedTopicARN string, err error) {
	topicARN = strings.TrimSpace(topicARN)
	targetARN = strings.TrimSpace(targetARN)
	phoneNumber = strings.TrimSpace(phoneNumber)

	targets := 0
	if topicARN != "" {
		targets++
	}
	if targetARN != "" {
		targets++
	}
	if phoneNumber != "" {
		targets++
	}
	if targets != 1 {
		return "", "", "", fmt.Errorf("%w: exactly one target is required (TopicArn, TargetArn, or PhoneNumber)", ErrValidation)
	}

	switch {
	case topicARN != "":
		return PublishTargetKindTopic, topicARN, topicARN, nil
	case targetARN != "":
		return PublishTargetKindTargetARN, targetARN, "", nil
	default:
		return PublishTargetKindPhone, phoneNumber, "", nil
	}
}

func normalizeMessageStructure(raw string, payload string) (string, error) {
	structure := strings.ToLower(strings.TrimSpace(raw))
	if structure == "" {
		return messageStructurePlain, nil
	}
	if structure != messageStructureJSON {
		return "", fmt.Errorf("%w: MessageStructure must be json when provided", ErrValidation)
	}

	var object map[string]any
	if err := json.Unmarshal([]byte(payload), &object); err != nil {
		return "", fmt.Errorf("%w: Message must be valid JSON when MessageStructure=json", ErrValidation)
	}
	if object == nil {
		return "", fmt.Errorf("%w: Message must be a JSON object when MessageStructure=json", ErrValidation)
	}
	defaultValue, ok := object["default"]
	if !ok {
		return "", fmt.Errorf("%w: Message json must contain default key", ErrValidation)
	}
	if _, ok := defaultValue.(string); !ok {
		return "", fmt.Errorf("%w: Message json default key must be a string", ErrValidation)
	}
	return messageStructureJSON, nil
}

func normalizeMessageAttributes(attributes map[string]MessageAttributeValue) (map[string]MessageAttributeValue, error) {
	if len(attributes) == 0 {
		return map[string]MessageAttributeValue{}, nil
	}

	normalized := make(map[string]MessageAttributeValue, len(attributes))
	for key, value := range attributes {
		name := strings.TrimSpace(key)
		if name == "" {
			return nil, fmt.Errorf("%w: message attribute name cannot be empty", ErrValidation)
		}
		dataType := strings.TrimSpace(value.DataType)
		if dataType == "" {
			return nil, fmt.Errorf("%w: message attribute %q DataType is required", ErrValidation, name)
		}
		stringValue := strings.TrimSpace(value.StringValue)
		binaryValue := strings.TrimSpace(value.BinaryValue)
		if stringValue == "" && binaryValue == "" {
			return nil, fmt.Errorf("%w: message attribute %q requires StringValue or BinaryValue", ErrValidation, name)
		}
		normalized[name] = MessageAttributeValue{
			DataType:    dataType,
			StringValue: stringValue,
			BinaryValue: binaryValue,
		}
	}
	return normalized, nil
}
