package application

import (
	"strconv"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

const (
	defaultVisibilityTimeoutSeconds = 30
	leaseTimeoutMetadataKey         = "visibility_timeout_seconds"
)

type DeliveryView struct {
	Queue   domain.Queue
	Message domain.Message
	Now     time.Time
}

func (v DeliveryView) Delay() bool {
	return IsDelayed(v.Message, v.Now)
}

func (v DeliveryView) Visible() bool {
	return IsVisible(v.Message, v.Queue, v.Now)
}

func (v DeliveryView) Invisible() bool {
	return IsInvisible(v.Message, v.Queue, v.Now)
}

func (v DeliveryView) Receipt() string {
	return CurrentReceiptHandle(v.Message)
}

func (v DeliveryView) Redeliver() bool {
	return CanRedeliver(v.Message, v.Queue, v.Now)
}

func IsDelayed(message domain.Message, now time.Time) bool {
	return !message.AvailableAt.IsZero() && now.Before(message.AvailableAt)
}

func IsVisible(message domain.Message, queue domain.Queue, now time.Time) bool {
	if IsDelayed(message, now) {
		return false
	}
	if message.ReceivedAt.IsZero() {
		return true
	}
	return !now.Before(leaseDeadline(message, queue))
}

func IsInvisible(message domain.Message, queue domain.Queue, now time.Time) bool {
	return !IsDelayed(message, now) && !IsVisible(message, queue, now)
}

func CurrentReceiptHandle(message domain.Message) string {
	if len(message.ReceiptKeys) == 0 {
		return ""
	}
	return message.ReceiptKeys[len(message.ReceiptKeys)-1]
}

func CanRedeliver(message domain.Message, queue domain.Queue, now time.Time) bool {
	if IsDelayed(message, now) {
		return false
	}
	deadline := leaseDeadline(message, queue)
	return !message.ReceivedAt.IsZero() && !deadline.IsZero() && !now.Before(deadline)
}

func leaseDeadline(message domain.Message, queue domain.Queue) time.Time {
	timeout := visibilityTimeout(queue, message)
	if timeout <= 0 || message.ReceivedAt.IsZero() {
		return time.Time{}
	}
	return message.ReceivedAt.Add(timeout)
}

func visibilityTimeout(queue domain.Queue, message domain.Message) time.Duration {
	if timeout := parseTimeoutSeconds(message.Metadata[leaseTimeoutMetadataKey]); timeout > 0 {
		return timeout
	}
	if timeout := parseTimeoutSeconds(queue.Attributes["VisibilityTimeout"]); timeout > 0 {
		return timeout
	}
	return defaultVisibilityTimeoutSeconds * time.Second
}

func parseTimeoutSeconds(raw string) time.Duration {
	seconds, err := strconv.Atoi(trimName(raw))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
