package application

import (
	"strconv"
	"strings"
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

func (v DeliveryView) DeadLetterEligible() bool {
	return IsDeadLetterEligible(v.Message, v.Queue, v.Now)
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

func IsFIFOQueue(queue domain.Queue) bool {
	if strings.EqualFold(trimName(queue.OrderingHint), "fifo") {
		return true
	}
	return strings.EqualFold(trimName(queue.Attributes["FifoQueue"]), "true")
}

func MessageGroupID(message domain.Message) string {
	return trimName(message.MessageGroupID)
}

func MessageSequenceNumber(message domain.Message) int64 {
	return message.SequenceNumber
}

func MessageBatchState(message domain.Message) (string, string, int, int) {
	return trimName(message.BatchID), trimName(message.BatchEntryID), message.BatchEntryIndex, message.BatchEntryCount
}

func IsDeadLetterEligible(message domain.Message, queue domain.Queue, now time.Time) bool {
	if trimName(message.DeadLetterQueue) != "" {
		return false
	}
	threshold := deadLetterThreshold(queue)
	if threshold <= 0 || message.Recovery.Attempts < threshold {
		return false
	}
	return CanRedeliver(message, queue, now) || IsVisible(message, queue, now)
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

func deadLetterThreshold(queue domain.Queue) int {
	if queue.Recovery.Policy == nil {
		return 0
	}
	threshold, err := strconv.Atoi(trimName(queue.Recovery.Policy["max_receive_count"]))
	if err != nil || threshold <= 0 {
		return 0
	}
	return threshold
}
