package infrastructure

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

const snsPageSize = 100

type TopicRepository struct {
	store *SQLiteStore
}

func NewTopicRepository(store *SQLiteStore) TopicRepository {
	return TopicRepository{store: store}
}

func (r TopicRepository) Create(topic domain.Topic) (domain.Topic, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Topic{}, err
	}

	if existing, err := r.GetByName(topic.TenantKey, topic.Name); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Topic{}, err
	}

	attributesJSON, err := marshalStringMap(topic.Attributes)
	if err != nil {
		return domain.Topic{}, err
	}
	tagsJSON, err := marshalStringMap(topic.Tags)
	if err != nil {
		return domain.Topic{}, err
	}

	policyJSON := strings.TrimSpace(topic.PolicyJSON)
	if policyJSON == "" {
		policyJSON = "{}"
	}
	createdAt := topic.CreatedAt.UTC().Format(time.RFC3339Nano)
	updatedAt := topic.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if strings.TrimSpace(createdAt) == "" {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		createdAt = now
		updatedAt = now
	}

	_, err = db.Exec(`
INSERT INTO topics (topic_arn, name, tenant_key, attributes_json, policy_json, tags_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, topic.ARN, topic.Name, topic.TenantKey, attributesJSON, policyJSON, tagsJSON, createdAt, updatedAt)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("sns: create topic: %w", err)
	}
	return r.GetByARN(topic.TenantKey, topic.ARN)
}

func (r TopicRepository) GetByName(tenantKey, name string) (domain.Topic, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Topic{}, err
	}

	row := db.QueryRow(`
SELECT topic_arn, name, tenant_key, attributes_json, policy_json, tags_json, created_at, updated_at
FROM topics
WHERE tenant_key = ? AND name = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(name))

	topic, err := scanTopicRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Topic{}, domain.ErrNotFound
		}
		return domain.Topic{}, err
	}
	return topic, nil
}

func (r TopicRepository) GetByARN(tenantKey, topicARN string) (domain.Topic, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Topic{}, err
	}

	row := db.QueryRow(`
SELECT topic_arn, name, tenant_key, attributes_json, policy_json, tags_json, created_at, updated_at
FROM topics
WHERE tenant_key = ? AND topic_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN))

	topic, err := scanTopicRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Topic{}, domain.ErrNotFound
		}
		return domain.Topic{}, err
	}
	return topic, nil
}

func (r TopicRepository) ListByTenant(tenantKey, nextToken string, limit int) ([]domain.Topic, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}

	limit = normalizeSNSPageLimit(limit)
	nextToken = strings.TrimSpace(nextToken)

	rows, err := db.Query(`
SELECT topic_arn, name, tenant_key, attributes_json, policy_json, tags_json, created_at, updated_at
FROM topics
WHERE tenant_key = ? AND topic_arn > ?
ORDER BY topic_arn ASC
LIMIT ?
`, strings.TrimSpace(tenantKey), nextToken, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list topics: %w", err)
	}
	defer rows.Close()

	topics := make([]domain.Topic, 0, limit+1)
	for rows.Next() {
		topic, err := scanTopicRows(rows)
		if err != nil {
			return nil, "", err
		}
		topics = append(topics, topic)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate topics: %w", err)
	}

	if len(topics) <= limit {
		return topics, "", nil
	}
	page := topics[:limit]
	return page, page[len(page)-1].ARN, nil
}

func (r TopicRepository) Update(topic domain.Topic) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	attributesJSON, err := marshalStringMap(topic.Attributes)
	if err != nil {
		return err
	}
	tagsJSON, err := marshalStringMap(topic.Tags)
	if err != nil {
		return err
	}

	policyJSON := strings.TrimSpace(topic.PolicyJSON)
	if policyJSON == "" {
		policyJSON = "{}"
	}
	updatedAt := topic.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if strings.TrimSpace(updatedAt) == "" {
		updatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	result, err := db.Exec(`
UPDATE topics
SET attributes_json = ?, policy_json = ?, tags_json = ?, updated_at = ?
WHERE tenant_key = ? AND topic_arn = ?
`, attributesJSON, policyJSON, tagsJSON, updatedAt, strings.TrimSpace(topic.TenantKey), strings.TrimSpace(topic.ARN))
	if err != nil {
		return fmt.Errorf("sns: update topic: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: update topic affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r TopicRepository) DeleteByARN(tenantKey, topicARN string) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("sns: begin topic delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM subscriptions WHERE tenant_key = ? AND topic_arn = ?`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN)); err != nil {
		return fmt.Errorf("sns: cascade delete subscriptions: %w", err)
	}

	result, err := tx.Exec(`DELETE FROM topics WHERE tenant_key = ? AND topic_arn = ?`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN))
	if err != nil {
		return fmt.Errorf("sns: delete topic: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: delete topic affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sns: commit topic delete: %w", err)
	}
	return nil
}

func (r TopicRepository) ensureDB() (*sql.DB, error) {
	if r.store == nil || r.store.db == nil {
		return nil, fmt.Errorf("sns: topic repository not initialized")
	}
	return r.store.db, nil
}

func scanTopicRow(row interface{ Scan(dest ...any) error }) (domain.Topic, error) {
	var (
		topicARN       string
		name           string
		tenantKey      string
		attributesJSON string
		policyJSON     string
		tagsJSON       string
		createdAtRaw   string
		updatedAtRaw   string
	)
	if err := row.Scan(&topicARN, &name, &tenantKey, &attributesJSON, &policyJSON, &tagsJSON, &createdAtRaw, &updatedAtRaw); err != nil {
		return domain.Topic{}, err
	}

	attributes, err := unmarshalStringMap(attributesJSON)
	if err != nil {
		return domain.Topic{}, err
	}
	tags, err := unmarshalStringMap(tagsJSON)
	if err != nil {
		return domain.Topic{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("sns: parse topic created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("sns: parse topic updated_at: %w", err)
	}

	return domain.Topic{
		ARN:        topicARN,
		Name:       name,
		TenantKey:  tenantKey,
		Attributes: attributes,
		PolicyJSON: strings.TrimSpace(policyJSON),
		Tags:       tags,
		IsFIFO:     strings.HasSuffix(name, ".fifo"),
		CreatedAt:  createdAt.UTC(),
		UpdatedAt:  updatedAt.UTC(),
	}, nil
}

func scanTopicRows(rows *sql.Rows) (domain.Topic, error) {
	return scanTopicRow(rows)
}

func marshalStringMap(values map[string]string) (string, error) {
	if values == nil {
		values = map[string]string{}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("sns: marshal map: %w", err)
	}
	return string(encoded), nil
}

func unmarshalStringMap(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	values := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("sns: unmarshal map: %w", err)
	}
	return values, nil
}

func normalizeSNSPageLimit(limit int) int {
	if limit <= 0 || limit > snsPageSize {
		return snsPageSize
	}
	return limit
}
