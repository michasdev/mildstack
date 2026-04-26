package infrastructure

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type PublishRepository struct {
	store *SQLiteStore
}

func NewPublishRepository(store *SQLiteStore) PublishRepository {
	return PublishRepository{store: store}
}

func (r PublishRepository) SavePublishedMessage(message domain.PublishedMessage, dedupScopeKey string, deduplicated bool) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	attributesJSON, err := marshalMessageAttributes(message.MessageAttributes)
	if err != nil {
		return err
	}

	deduplicatedFlag := 0
	if deduplicated {
		deduplicatedFlag = 1
	}

	_, err = db.Exec(`
INSERT INTO published_messages (
  message_id,
  tenant_key,
  target_kind,
  target_ref,
  topic_arn,
  payload,
  subject,
  message_structure,
  message_attributes_json,
  message_group_id,
  message_deduplication_id,
  sequence_number,
  dedup_scope_key,
  deduplicated,
  published_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		message.MessageID,
		message.TenantKey,
		message.TargetKind,
		message.TargetRef,
		nullableString(message.TopicARN),
		message.Payload,
		nullableString(message.Subject),
		normalizeMessageStructureValue(message.MessageStructure),
		attributesJSON,
		strings.TrimSpace(message.MessageGroupID),
		strings.TrimSpace(message.MessageDeduplicationID),
		strings.TrimSpace(message.SequenceNumber),
		strings.TrimSpace(dedupScopeKey),
		deduplicatedFlag,
		message.PublishedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("sns: persist published message: %w", err)
	}
	return nil
}

func (r PublishRepository) FindRecentByDedupID(tenantKey, topicARN, dedupScopeKey, deduplicationID string, since time.Time) (domain.PublishedMessage, bool, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PublishedMessage{}, false, err
	}
	if strings.TrimSpace(deduplicationID) == "" {
		return domain.PublishedMessage{}, false, nil
	}

	row := db.QueryRow(`
SELECT
  message_id,
  tenant_key,
  target_kind,
  target_ref,
  COALESCE(topic_arn, ''),
  payload,
  COALESCE(subject, ''),
  COALESCE(message_structure, 'plain'),
  message_attributes_json,
  COALESCE(message_group_id, ''),
  COALESCE(message_deduplication_id, ''),
  COALESCE(sequence_number, ''),
  published_at
FROM published_messages
WHERE tenant_key = ?
  AND topic_arn = ?
  AND dedup_scope_key = ?
  AND message_deduplication_id = ?
  AND published_at >= ?
ORDER BY published_at DESC
LIMIT 1
`,
		strings.TrimSpace(tenantKey),
		strings.TrimSpace(topicARN),
		strings.TrimSpace(dedupScopeKey),
		strings.TrimSpace(deduplicationID),
		since.UTC().Format(time.RFC3339Nano),
	)

	message, err := scanPublishedMessageRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PublishedMessage{}, false, nil
		}
		return domain.PublishedMessage{}, false, err
	}
	return message, true, nil
}

func (r PublishRepository) NextSequenceNumber(tenantKey, topicARN, messageGroupID string) (string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return "", err
	}

	var maxValue sql.NullString
	err = db.QueryRow(`
SELECT COALESCE(MAX(CAST(sequence_number AS INTEGER)), 0)
FROM published_messages
WHERE tenant_key = ?
  AND topic_arn = ?
  AND message_group_id = ?
  AND sequence_number != ''
`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN), strings.TrimSpace(messageGroupID)).Scan(&maxValue)
	if err != nil {
		return "", fmt.Errorf("sns: query next sequence number: %w", err)
	}

	base := int64(0)
	if maxValue.Valid && strings.TrimSpace(maxValue.String) != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(maxValue.String), 10, 64)
		if err != nil {
			return "", fmt.Errorf("sns: parse max sequence number: %w", err)
		}
		base = parsed
	}
	return strconv.FormatInt(base+1, 10), nil
}

func (r PublishRepository) SaveDeliveryAttempt(attempt domain.DeliveryAttempt) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT INTO delivery_attempts (
  attempt_id,
  message_id,
  subscription_arn,
  endpoint_arn,
  tenant_key,
  protocol,
  status,
  failure_code,
  failure_message,
  request_snapshot_json,
  response_snapshot_json,
  attempted_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		attempt.AttemptID,
		attempt.MessageID,
		nullableString(attempt.SubscriptionARN),
		nullableString(attempt.EndpointARN),
		attempt.TenantKey,
		attempt.Protocol,
		attempt.Status,
		nullableString(attempt.FailureCode),
		nullableString(attempt.FailureMessage),
		defaultJSON(attempt.RequestSnapshotJSON),
		defaultJSON(attempt.ResponseSnapshotJSON),
		attempt.AttemptedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("sns: persist delivery attempt: %w", err)
	}
	return nil
}

func (r PublishRepository) ListDeliveryAttemptsByMessageID(tenantKey, messageID string) ([]domain.DeliveryAttempt, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT
  attempt_id,
  message_id,
  COALESCE(subscription_arn, ''),
  COALESCE(endpoint_arn, ''),
  tenant_key,
  protocol,
  status,
  COALESCE(failure_code, ''),
  COALESCE(failure_message, ''),
  COALESCE(request_snapshot_json, '{}'),
  COALESCE(response_snapshot_json, '{}'),
  attempted_at
FROM delivery_attempts
WHERE tenant_key = ? AND message_id = ?
ORDER BY attempted_at ASC
`, strings.TrimSpace(tenantKey), strings.TrimSpace(messageID))
	if err != nil {
		return nil, fmt.Errorf("sns: list delivery attempts: %w", err)
	}
	defer rows.Close()

	attempts := make([]domain.DeliveryAttempt, 0)
	for rows.Next() {
		attempt, err := scanDeliveryAttemptRow(rows)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sns: iterate delivery attempts: %w", err)
	}
	return attempts, nil
}

func (r PublishRepository) ensureDB() (*sql.DB, error) {
	if r.store == nil || r.store.db == nil {
		return nil, fmt.Errorf("sns: publish repository not initialized")
	}
	return r.store.db, nil
}

func scanPublishedMessageRow(row interface{ Scan(dest ...any) error }) (domain.PublishedMessage, error) {
	var (
		messageID              string
		tenantKey              string
		targetKind             string
		targetRef              string
		topicARN               string
		payload                string
		subject                string
		messageStructure       string
		attributesJSON         string
		messageGroupID         string
		messageDeduplicationID string
		sequenceNumber         string
		publishedAtRaw         string
	)

	if err := row.Scan(
		&messageID,
		&tenantKey,
		&targetKind,
		&targetRef,
		&topicARN,
		&payload,
		&subject,
		&messageStructure,
		&attributesJSON,
		&messageGroupID,
		&messageDeduplicationID,
		&sequenceNumber,
		&publishedAtRaw,
	); err != nil {
		return domain.PublishedMessage{}, err
	}

	attributes, err := unmarshalMessageAttributes(attributesJSON)
	if err != nil {
		return domain.PublishedMessage{}, err
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, publishedAtRaw)
	if err != nil {
		return domain.PublishedMessage{}, fmt.Errorf("sns: parse published_at: %w", err)
	}

	return domain.PublishedMessage{
		MessageID:              messageID,
		TenantKey:              tenantKey,
		TargetKind:             targetKind,
		TargetRef:              targetRef,
		TopicARN:               topicARN,
		Payload:                payload,
		Subject:                subject,
		MessageStructure:       normalizeMessageStructureValue(messageStructure),
		MessageAttributes:      attributes,
		MessageGroupID:         messageGroupID,
		MessageDeduplicationID: messageDeduplicationID,
		SequenceNumber:         sequenceNumber,
		PublishedAt:            publishedAt.UTC(),
	}, nil
}

func scanDeliveryAttemptRow(row interface{ Scan(dest ...any) error }) (domain.DeliveryAttempt, error) {
	var (
		attemptID            string
		messageID            string
		subscriptionARN      string
		endpointARN          string
		tenantKey            string
		protocol             string
		status               string
		failureCode          string
		failureMessage       string
		requestSnapshotJSON  string
		responseSnapshotJSON string
		attemptedAtRaw       string
	)

	if err := row.Scan(
		&attemptID,
		&messageID,
		&subscriptionARN,
		&endpointARN,
		&tenantKey,
		&protocol,
		&status,
		&failureCode,
		&failureMessage,
		&requestSnapshotJSON,
		&responseSnapshotJSON,
		&attemptedAtRaw,
	); err != nil {
		return domain.DeliveryAttempt{}, err
	}

	attemptedAt, err := time.Parse(time.RFC3339Nano, attemptedAtRaw)
	if err != nil {
		return domain.DeliveryAttempt{}, fmt.Errorf("sns: parse delivery attempted_at: %w", err)
	}

	return domain.DeliveryAttempt{
		AttemptID:            attemptID,
		MessageID:            messageID,
		SubscriptionARN:      subscriptionARN,
		EndpointARN:          endpointARN,
		TenantKey:            tenantKey,
		Protocol:             protocol,
		Status:               status,
		FailureCode:          failureCode,
		FailureMessage:       failureMessage,
		RequestSnapshotJSON:  requestSnapshotJSON,
		ResponseSnapshotJSON: responseSnapshotJSON,
		AttemptedAt:          attemptedAt.UTC(),
	}, nil
}

func marshalMessageAttributes(attributes map[string]domain.MessageAttributeValue) (string, error) {
	if len(attributes) == 0 {
		attributes = map[string]domain.MessageAttributeValue{}
	}
	encoded, err := json.Marshal(attributes)
	if err != nil {
		return "", fmt.Errorf("sns: marshal message attributes: %w", err)
	}
	return string(encoded), nil
}

func unmarshalMessageAttributes(raw string) (map[string]domain.MessageAttributeValue, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]domain.MessageAttributeValue{}, nil
	}
	values := map[string]domain.MessageAttributeValue{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("sns: unmarshal message attributes: %w", err)
	}
	return values, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func defaultJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}

func normalizeMessageStructureValue(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "plain"
	}
	return trimmed
}
