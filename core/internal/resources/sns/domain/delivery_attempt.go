package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DeliveryAttemptStatusPending     = "Pending"
	DeliveryAttemptStatusDelivered   = "Delivered"
	DeliveryAttemptStatusFailed      = "Failed"
	DeliveryAttemptStatusFilteredOut = "FilteredOut"
	DeliveryAttemptStatusSkipped     = "Skipped"
)

// DeliveryAttempt captures one attempted delivery decision for a published message.
type DeliveryAttempt struct {
	AttemptID            string
	MessageID            string
	SubscriptionARN      string
	EndpointARN          string
	TenantKey            string
	Protocol             string
	Status               string
	FailureCode          string
	FailureMessage       string
	RequestSnapshotJSON  string
	ResponseSnapshotJSON string
	AttemptedAt          time.Time
}

func NewDeliveryAttempt(messageID, tenantKey, subscriptionARN, endpointARN, protocol string, now time.Time) (DeliveryAttempt, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return DeliveryAttempt{}, fmt.Errorf("%w: message id is required", ErrValidation)
	}
	tenantKey = strings.TrimSpace(tenantKey)
	if tenantKey == "" {
		return DeliveryAttempt{}, fmt.Errorf("%w: tenant key is required", ErrValidation)
	}
	protocol = strings.TrimSpace(protocol)
	if protocol == "" {
		return DeliveryAttempt{}, fmt.Errorf("%w: protocol is required", ErrValidation)
	}

	return DeliveryAttempt{
		AttemptID:       uuid.NewString(),
		MessageID:       messageID,
		SubscriptionARN: strings.TrimSpace(subscriptionARN),
		EndpointARN:     strings.TrimSpace(endpointARN),
		TenantKey:       tenantKey,
		Protocol:        protocol,
		Status:          DeliveryAttemptStatusPending,
		AttemptedAt:     normalizeTimestamp(now),
	}, nil
}

func (a DeliveryAttempt) MarkDelivered(now time.Time) (DeliveryAttempt, error) {
	return a.transitionTo(DeliveryAttemptStatusDelivered, "", "", now)
}

func (a DeliveryAttempt) MarkFilteredOut(now time.Time) (DeliveryAttempt, error) {
	return a.transitionTo(DeliveryAttemptStatusFilteredOut, "", "", now)
}

func (a DeliveryAttempt) MarkSkipped(code, message string, now time.Time) (DeliveryAttempt, error) {
	return a.transitionTo(DeliveryAttemptStatusSkipped, code, message, now)
}

func (a DeliveryAttempt) MarkFailed(code, message string, now time.Time) (DeliveryAttempt, error) {
	return a.transitionTo(DeliveryAttemptStatusFailed, code, message, now)
}

func (a DeliveryAttempt) transitionTo(status, code, message string, now time.Time) (DeliveryAttempt, error) {
	if strings.TrimSpace(a.Status) != DeliveryAttemptStatusPending {
		return DeliveryAttempt{}, fmt.Errorf("%w: delivery attempt transition requires pending status", ErrValidation)
	}

	updated := a
	updated.Status = status
	updated.FailureCode = strings.TrimSpace(code)
	updated.FailureMessage = strings.TrimSpace(message)
	updated.AttemptedAt = normalizeTimestamp(now)
	return updated, nil
}
