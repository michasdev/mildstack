package infrastructure

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type SubscriptionRepository struct {
	store *SQLiteStore
}

func NewSubscriptionRepository(store *SQLiteStore) SubscriptionRepository {
	return SubscriptionRepository{store: store}
}

func (r SubscriptionRepository) Create(subscription domain.Subscription) (domain.Subscription, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Subscription{}, err
	}

	if existing, err := r.GetByEndpoint(subscription.TenantKey, subscription.TopicARN, subscription.Protocol, subscription.Endpoint); err == nil {
		if existing.Status != domain.SubscriptionStatusDeleted {
			return existing, nil
		}
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Subscription{}, err
	}

	attributesJSON, err := marshalStringMap(subscription.Attributes)
	if err != nil {
		return domain.Subscription{}, err
	}

	confirmedAt := ""
	if subscription.ConfirmedAt != nil {
		confirmedAt = subscription.ConfirmedAt.UTC().Format(time.RFC3339Nano)
	}

	_, err = db.Exec(`
INSERT INTO subscriptions (
  subscription_arn,
  topic_arn,
  tenant_key,
  protocol,
  endpoint,
  owner_account_id,
  status,
  token,
  attributes_json,
  confirmed_at,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		subscription.ARN,
		subscription.TopicARN,
		subscription.TenantKey,
		subscription.Protocol,
		subscription.Endpoint,
		subscription.OwnerAccountID,
		subscription.Status,
		subscription.Token,
		attributesJSON,
		confirmedAt,
		subscription.CreatedAt.UTC().Format(time.RFC3339Nano),
		subscription.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("sns: create subscription: %w", err)
	}
	return r.GetByARN(subscription.TenantKey, subscription.ARN)
}

func (r SubscriptionRepository) GetByEndpoint(tenantKey, topicARN, protocol, endpoint string) (domain.Subscription, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Subscription{}, err
	}

	row := db.QueryRow(`
SELECT
  subscription_arn,
  topic_arn,
  tenant_key,
  protocol,
  endpoint,
  owner_account_id,
  status,
  token,
  attributes_json,
  confirmed_at,
  created_at,
  updated_at
FROM subscriptions
WHERE tenant_key = ? AND topic_arn = ? AND protocol = ? AND endpoint = ?
ORDER BY created_at ASC
LIMIT 1
`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN), strings.ToLower(strings.TrimSpace(protocol)), strings.TrimSpace(endpoint))

	subscription, err := scanSubscriptionRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Subscription{}, domain.ErrNotFound
		}
		return domain.Subscription{}, err
	}
	return subscription, nil
}

func (r SubscriptionRepository) GetByARN(tenantKey, subscriptionARN string) (domain.Subscription, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Subscription{}, err
	}

	row := db.QueryRow(`
SELECT
  subscription_arn,
  topic_arn,
  tenant_key,
  protocol,
  endpoint,
  owner_account_id,
  status,
  token,
  attributes_json,
  confirmed_at,
  created_at,
  updated_at
FROM subscriptions
WHERE tenant_key = ? AND subscription_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(subscriptionARN))

	subscription, err := scanSubscriptionRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Subscription{}, domain.ErrNotFound
		}
		return domain.Subscription{}, err
	}
	return subscription, nil
}

func (r SubscriptionRepository) GetByToken(tenantKey, topicARN, token string) (domain.Subscription, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.Subscription{}, err
	}

	row := db.QueryRow(`
SELECT
  subscription_arn,
  topic_arn,
  tenant_key,
  protocol,
  endpoint,
  owner_account_id,
  status,
  token,
  attributes_json,
  confirmed_at,
  created_at,
  updated_at
FROM subscriptions
WHERE tenant_key = ? AND topic_arn = ? AND token = ? AND status = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN), strings.TrimSpace(token), domain.SubscriptionStatusPendingConfirmation)

	subscription, err := scanSubscriptionRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Subscription{}, domain.ErrNotFound
		}
		return domain.Subscription{}, err
	}
	return subscription, nil
}

func (r SubscriptionRepository) ListByTenant(tenantKey, nextToken string, limit int) ([]domain.Subscription, string, error) {
	return r.listByQuery(`
SELECT
  subscription_arn,
  topic_arn,
  tenant_key,
  protocol,
  endpoint,
  owner_account_id,
  status,
  token,
  attributes_json,
  confirmed_at,
  created_at,
  updated_at
FROM subscriptions
WHERE tenant_key = ? AND subscription_arn > ?
ORDER BY subscription_arn ASC
LIMIT ?
`, []any{strings.TrimSpace(tenantKey), strings.TrimSpace(nextToken), normalizeSNSPageLimit(limit) + 1}, normalizeSNSPageLimit(limit))
}

func (r SubscriptionRepository) ListByTopic(tenantKey, topicARN, nextToken string, limit int) ([]domain.Subscription, string, error) {
	return r.listByQuery(`
SELECT
  subscription_arn,
  topic_arn,
  tenant_key,
  protocol,
  endpoint,
  owner_account_id,
  status,
  token,
  attributes_json,
  confirmed_at,
  created_at,
  updated_at
FROM subscriptions
WHERE tenant_key = ? AND topic_arn = ? AND subscription_arn > ?
ORDER BY subscription_arn ASC
LIMIT ?
`, []any{strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN), strings.TrimSpace(nextToken), normalizeSNSPageLimit(limit) + 1}, normalizeSNSPageLimit(limit))
}

func (r SubscriptionRepository) CountByTopicAndStatus(tenantKey, topicARN string) (confirmed, pending, deleted int, err error) {
	db, err := r.ensureDB()
	if err != nil {
		return 0, 0, 0, err
	}

	rows, err := db.Query(`
SELECT status, COUNT(*)
FROM subscriptions
WHERE tenant_key = ? AND topic_arn = ?
GROUP BY status
`, strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sns: count subscriptions by status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return 0, 0, 0, fmt.Errorf("sns: scan subscription status count: %w", err)
		}
		switch status {
		case domain.SubscriptionStatusConfirmed:
			confirmed = count
		case domain.SubscriptionStatusPendingConfirmation:
			pending = count
		case domain.SubscriptionStatusDeleted:
			deleted = count
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("sns: iterate subscription status count: %w", err)
	}
	return confirmed, pending, deleted, nil
}

func (r SubscriptionRepository) Update(subscription domain.Subscription) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	attributesJSON, err := marshalStringMap(subscription.Attributes)
	if err != nil {
		return err
	}
	confirmedAt := ""
	if subscription.ConfirmedAt != nil {
		confirmedAt = subscription.ConfirmedAt.UTC().Format(time.RFC3339Nano)
	}

	result, err := db.Exec(`
UPDATE subscriptions
SET
  protocol = ?,
  endpoint = ?,
  owner_account_id = ?,
  status = ?,
  token = ?,
  attributes_json = ?,
  confirmed_at = ?,
  updated_at = ?
WHERE tenant_key = ? AND subscription_arn = ?
`,
		subscription.Protocol,
		subscription.Endpoint,
		subscription.OwnerAccountID,
		subscription.Status,
		subscription.Token,
		attributesJSON,
		confirmedAt,
		subscription.UpdatedAt.UTC().Format(time.RFC3339Nano),
		subscription.TenantKey,
		subscription.ARN,
	)
	if err != nil {
		return fmt.Errorf("sns: update subscription: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: update subscription affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r SubscriptionRepository) DeleteByARN(tenantKey, subscriptionARN string) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM subscriptions
WHERE tenant_key = ? AND subscription_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(subscriptionARN))
	if err != nil {
		return fmt.Errorf("sns: delete subscription: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: delete subscription affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r SubscriptionRepository) listByQuery(query string, args []any, limit int) ([]domain.Subscription, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]domain.Subscription, 0, limit+1)
	for rows.Next() {
		subscription, err := scanSubscriptionRows(rows)
		if err != nil {
			return nil, "", err
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate subscriptions: %w", err)
	}

	if len(subscriptions) <= limit {
		return subscriptions, "", nil
	}
	page := subscriptions[:limit]
	return page, page[len(page)-1].ARN, nil
}

func (r SubscriptionRepository) ensureDB() (*sql.DB, error) {
	if r.store == nil || r.store.db == nil {
		return nil, fmt.Errorf("sns: subscription repository not initialized")
	}
	return r.store.db, nil
}

func scanSubscriptionRow(row interface{ Scan(dest ...any) error }) (domain.Subscription, error) {
	var (
		subscriptionARN string
		topicARN        string
		tenantKey       string
		protocol        string
		endpoint        string
		ownerAccountID  string
		status          string
		token           string
		attributesJSON  string
		confirmedAtRaw  string
		createdAtRaw    string
		updatedAtRaw    string
	)
	if err := row.Scan(
		&subscriptionARN,
		&topicARN,
		&tenantKey,
		&protocol,
		&endpoint,
		&ownerAccountID,
		&status,
		&token,
		&attributesJSON,
		&confirmedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return domain.Subscription{}, err
	}

	attributes, err := unmarshalStringMap(attributesJSON)
	if err != nil {
		return domain.Subscription{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("sns: parse subscription created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("sns: parse subscription updated_at: %w", err)
	}

	var confirmedAt *time.Time
	if trimmed := strings.TrimSpace(confirmedAtRaw); trimmed != "" {
		parsed, err := time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return domain.Subscription{}, fmt.Errorf("sns: parse subscription confirmed_at: %w", err)
		}
		parsed = parsed.UTC()
		confirmedAt = &parsed
	}

	return domain.Subscription{
		ARN:            subscriptionARN,
		TopicARN:       topicARN,
		TenantKey:      tenantKey,
		Protocol:       protocol,
		Endpoint:       endpoint,
		OwnerAccountID: ownerAccountID,
		Status:         status,
		Token:          token,
		Attributes:     attributes,
		ConfirmedAt:    confirmedAt,
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
	}, nil
}

func scanSubscriptionRows(rows *sql.Rows) (domain.Subscription, error) {
	return scanSubscriptionRow(rows)
}
