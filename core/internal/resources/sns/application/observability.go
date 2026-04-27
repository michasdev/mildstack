package application

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type snsObservability struct {
	mu sync.Mutex

	publishCalls          int64
	publishBatchCalls     int64
	publishSuccess        int64
	publishFailure        int64
	publishDeduplicated   int64
	publishTotalDuration  time.Duration
	lastPublishError      string
	deliveryAttempts      int64
	deliveryDelivered     int64
	deliveryFailed        int64
	deliveryFilteredOut   int64
	deliverySkipped       int64
	deliveryTotalDuration time.Duration
	lastDeliveryError     string
	lastUpdatedAt         time.Time
}

func newSNSObservability() *snsObservability {
	return &snsObservability{}
}

func (o *snsObservability) recordPublish(targetKind string, deduplicated bool, duration time.Duration, err error) {
	if o == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.publishCalls++
	o.publishTotalDuration += duration
	if err != nil {
		o.publishFailure++
		o.lastPublishError = strings.TrimSpace(err.Error())
	} else {
		o.publishSuccess++
		if deduplicated {
			o.publishDeduplicated++
		}
	}
	o.lastUpdatedAt = time.Now().UTC()

	log.Printf(
		"sns observability publish target_kind=%s deduplicated=%t duration_ms=%d success=%t",
		strings.TrimSpace(targetKind),
		deduplicated,
		duration.Milliseconds(),
		err == nil,
	)
}

func (o *snsObservability) recordPublishBatch(entryCount, successCount, failedCount int, duration time.Duration, err error) {
	if o == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.publishBatchCalls++
	if err != nil {
		o.lastPublishError = strings.TrimSpace(err.Error())
	}
	o.lastUpdatedAt = time.Now().UTC()

	log.Printf(
		"sns observability publish_batch entries=%d success=%d failed=%d duration_ms=%d success=%t",
		entryCount,
		successCount,
		failedCount,
		duration.Milliseconds(),
		err == nil,
	)
}

func (o *snsObservability) recordDelivery(status, protocol, failureCode string, duration time.Duration, err error) {
	if o == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	normalizedStatus := strings.TrimSpace(status)
	o.deliveryAttempts++
	o.deliveryTotalDuration += duration
	switch normalizedStatus {
	case domain.DeliveryAttemptStatusDelivered:
		o.deliveryDelivered++
	case domain.DeliveryAttemptStatusFailed:
		o.deliveryFailed++
	case domain.DeliveryAttemptStatusFilteredOut:
		o.deliveryFilteredOut++
	case domain.DeliveryAttemptStatusSkipped:
		o.deliverySkipped++
	}

	if err != nil {
		o.lastDeliveryError = strings.TrimSpace(err.Error())
	} else if strings.TrimSpace(failureCode) != "" {
		o.lastDeliveryError = strings.TrimSpace(failureCode)
	}
	o.lastUpdatedAt = time.Now().UTC()

	log.Printf(
		"sns observability delivery status=%s protocol=%s failure_code=%s duration_ms=%d persisted=%t",
		normalizedStatus,
		strings.TrimSpace(protocol),
		strings.TrimSpace(failureCode),
		duration.Milliseconds(),
		err == nil,
	)
}

func (o *snsObservability) snapshot() map[string]any {
	if o == nil {
		return map[string]any{}
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	publishAvg := int64(0)
	if o.publishCalls > 0 {
		publishAvg = o.publishTotalDuration.Milliseconds() / o.publishCalls
	}

	deliveryAvg := int64(0)
	if o.deliveryAttempts > 0 {
		deliveryAvg = o.deliveryTotalDuration.Milliseconds() / o.deliveryAttempts
	}

	return map[string]any{
		"publishCalls":           o.publishCalls,
		"publishBatchCalls":      o.publishBatchCalls,
		"publishSuccess":         o.publishSuccess,
		"publishFailure":         o.publishFailure,
		"publishDeduplicated":    o.publishDeduplicated,
		"publishAverageMs":       publishAvg,
		"lastPublishError":       o.lastPublishError,
		"deliveryAttempts":       o.deliveryAttempts,
		"deliveryDelivered":      o.deliveryDelivered,
		"deliveryFailed":         o.deliveryFailed,
		"deliveryFilteredOut":    o.deliveryFilteredOut,
		"deliverySkipped":        o.deliverySkipped,
		"deliveryAverageMs":      deliveryAvg,
		"lastDeliveryError":      o.lastDeliveryError,
		"lastUpdatedAtRFC3339":   o.lastUpdatedAt.Format(time.RFC3339Nano),
		"publishTotalDurationMs": o.publishTotalDuration.Milliseconds(),
		"deliveryTotalMs":        o.deliveryTotalDuration.Milliseconds(),
	}
}

func (s *Service) syncObservabilitySnapshot() {
	if s == nil || s.stateHook == nil || s.observability == nil {
		return
	}

	snapshot := map[string]any{
		"service": "sns",
	}

	if current, ok := s.stateHook.Get(domain.StateKey); ok {
		if currentMap, ok := current.(map[string]any); ok {
			for key, value := range currentMap {
				snapshot[key] = value
			}
		}
	}

	snapshot["observability"] = s.observability.snapshot()
	s.stateHook.Set(domain.StateKey, snapshot)
}
