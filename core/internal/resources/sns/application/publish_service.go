package application

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
	"github.com/michasdev/mildstack/core/internal/resources/sns/infrastructure"
)

const snsDeduplicationWindow = 5 * time.Minute

func (s *Service) Publish(request domain.PublishRequest) (result domain.PublishResult, err error) {
	startedAt := time.Now().UTC()
	targetKind := ""
	deduplicated := false
	defer func() {
		if s == nil || s.observability == nil {
			return
		}
		s.observability.recordPublish(targetKind, deduplicated, time.Since(startedAt), err)
		s.syncObservabilitySnapshot()
	}()

	if err = s.ensureStore(); err != nil {
		return domain.PublishResult{}, err
	}

	tenant := s.defaultTenant()
	message, err := domain.NewPublishedMessage(tenant, request, time.Now().UTC())
	if err != nil {
		return domain.PublishResult{}, err
	}
	targetKind = message.TargetKind

	var topic domain.Topic
	hasTopic := strings.TrimSpace(message.TopicARN) != ""
	if hasTopic {
		topic, err = s.topicRepository().GetByARN(tenant.Key(), message.TopicARN)
		if err != nil {
			return domain.PublishResult{}, err
		}
	}

	dedupScopeKey := ""
	if hasTopic && topic.IsFIFO {
		message, dedupScopeKey, deduplicated, err = s.applyFIFORules(topic, message)
		if err != nil {
			return domain.PublishResult{}, err
		}
	}

	if !deduplicated {
		if err = s.publishRepository().SavePublishedMessage(message, dedupScopeKey, deduplicated); err != nil {
			return domain.PublishResult{}, err
		}
		if err = s.dispatchDelivery(message, hasTopic, topic); err != nil {
			return domain.PublishResult{}, err
		}
	}

	return domain.PublishResult{
		MessageID:      message.MessageID,
		SequenceNumber: message.SequenceNumber,
	}, nil
}

func (s *Service) PublishBatch(request domain.PublishBatchRequest) (result domain.PublishBatchResult, err error) {
	startedAt := time.Now().UTC()
	entryCount := len(request.Entries)
	result = domain.PublishBatchResult{}
	defer func() {
		if s == nil || s.observability == nil {
			return
		}
		s.observability.recordPublishBatch(entryCount, len(result.Successful), len(result.Failed), time.Since(startedAt), err)
		s.syncObservabilitySnapshot()
	}()

	if err = s.ensureStore(); err != nil {
		return domain.PublishBatchResult{}, err
	}

	request.TopicARN = strings.TrimSpace(request.TopicARN)
	if request.TopicARN == "" {
		return domain.PublishBatchResult{}, fmt.Errorf("%w: TopicArn is required", domain.ErrValidation)
	}
	if len(request.Entries) == 0 {
		return domain.PublishBatchResult{}, fmt.Errorf("%w: PublishBatchRequestEntries is required", domain.ErrValidation)
	}
	if len(request.Entries) > 10 {
		return domain.PublishBatchResult{}, fmt.Errorf("%w: too many entries in batch request", domain.ErrValidation)
	}
	if hasDuplicateBatchEntryIDs(request.Entries) {
		return domain.PublishBatchResult{}, domain.ErrBatchEntryIDsNotDistinct
	}

	if _, err = s.topicRepository().GetByARN(s.defaultTenant().Key(), request.TopicARN); err != nil {
		return domain.PublishBatchResult{}, err
	}

	result = domain.PublishBatchResult{
		Successful: make([]domain.PublishBatchResultEntry, 0, len(request.Entries)),
		Failed:     make([]domain.PublishBatchErrorEntry, 0),
	}

	seenIDs := make(map[string]struct{}, len(request.Entries))
	for _, entry := range request.Entries {
		entryID := strings.TrimSpace(entry.ID)
		if entryID == "" {
			result.Failed = append(result.Failed, domain.PublishBatchErrorEntry{
				ID:          entryID,
				Code:        "InvalidBatchEntryId",
				Message:     "Batch entry Id is required.",
				SenderFault: true,
			})
			continue
		}
		if _, exists := seenIDs[entryID]; exists {
			result.Failed = append(result.Failed, domain.PublishBatchErrorEntry{
				ID:          entryID,
				Code:        "InvalidParameterException",
				Message:     "Two or more batch entries in the request have the same Id.",
				SenderFault: true,
			})
			continue
		}
		seenIDs[entryID] = struct{}{}

		var publishResult domain.PublishResult
		publishResult, err = s.Publish(domain.PublishRequest{
			TopicARN:               request.TopicARN,
			Message:                entry.Message,
			Subject:                entry.Subject,
			MessageStructure:       entry.MessageStructure,
			MessageAttributes:      entry.MessageAttributes,
			MessageGroupID:         entry.MessageGroupID,
			MessageDeduplicationID: entry.MessageDeduplicationID,
		})
		if err != nil {
			code, message, senderFault := classifyPublishBatchEntryError(err)
			result.Failed = append(result.Failed, domain.PublishBatchErrorEntry{
				ID:          entryID,
				Code:        code,
				Message:     message,
				SenderFault: senderFault,
			})
			continue
		}

		result.Successful = append(result.Successful, domain.PublishBatchResultEntry{
			ID:             entryID,
			MessageID:      publishResult.MessageID,
			SequenceNumber: publishResult.SequenceNumber,
		})
	}

	return result, nil
}

func (s *Service) ListDeliveryAttemptsByMessageID(messageID string) ([]domain.DeliveryAttempt, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}
	return s.publishRepository().ListDeliveryAttemptsByMessageID(s.defaultTenant().Key(), messageID)
}

func (s *Service) applyFIFORules(topic domain.Topic, message domain.PublishedMessage) (domain.PublishedMessage, string, bool, error) {
	if strings.TrimSpace(message.MessageGroupID) == "" {
		return domain.PublishedMessage{}, "", false, fmt.Errorf("%w: MessageGroupId is required for FIFO topics", domain.ErrValidation)
	}

	deduplicationID := strings.TrimSpace(message.MessageDeduplicationID)
	if deduplicationID == "" {
		if !isTruthy(topic.Attributes["ContentBasedDeduplication"]) {
			return domain.PublishedMessage{}, "", false, fmt.Errorf("%w: MessageDeduplicationId is required for FIFO topics when ContentBasedDeduplication is disabled", domain.ErrValidation)
		}
		hash := sha256.Sum256([]byte(message.Payload))
		deduplicationID = hex.EncodeToString(hash[:])
	}

	dedupScopeKey := topic.ARN
	if strings.EqualFold(strings.TrimSpace(topic.Attributes["FifoThroughputScope"]), "MessageGroup") {
		dedupScopeKey = topic.ARN + "#" + strings.TrimSpace(message.MessageGroupID)
	}

	now := time.Now().UTC()
	existing, found, err := s.publishRepository().FindRecentByDedupID(
		topic.TenantKey,
		topic.ARN,
		dedupScopeKey,
		deduplicationID,
		now.Add(-snsDeduplicationWindow),
	)
	if err != nil {
		return domain.PublishedMessage{}, "", false, err
	}

	message.MessageDeduplicationID = deduplicationID
	if found {
		message.MessageID = existing.MessageID
		message.SequenceNumber = existing.SequenceNumber
		return message, dedupScopeKey, true, nil
	}

	sequenceNumber, err := s.publishRepository().NextSequenceNumber(topic.TenantKey, topic.ARN, message.MessageGroupID)
	if err != nil {
		return domain.PublishedMessage{}, "", false, err
	}
	message.SequenceNumber = sequenceNumber
	return message, dedupScopeKey, false, nil
}

func (s *Service) publishRepository() infrastructure.PublishRepository {
	return infrastructure.NewPublishRepository(s.store)
}

func classifyPublishBatchEntryError(err error) (string, string, bool) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return "InvalidParameterException", err.Error(), true
	case errors.Is(err, domain.ErrNotFound):
		return "NotFound", "The requested resource does not exist.", true
	default:
		return "InternalError", "Internal service error.", false
	}
}

func hasDuplicateBatchEntryIDs(entries []domain.PublishBatchRequestEntry) bool {
	seenIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			continue
		}
		if _, exists := seenIDs[id]; exists {
			return true
		}
		seenIDs[id] = struct{}{}
	}
	return false
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
