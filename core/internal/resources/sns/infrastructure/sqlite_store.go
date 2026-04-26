package infrastructure

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(statePath string) (*SQLiteStore, error) {
	statePath = strings.TrimSpace(statePath)
	if statePath == "" {
		return nil, fmt.Errorf("sns: storage path is required")
	}
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		return nil, fmt.Errorf("sns: create storage directory: %w", err)
	}

	dbPath := filepath.Join(statePath, sqliteFileName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sns: open sqlite database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) UpsertTopic(tenantKey, topicARN, name string) error {
	tenantKey = strings.TrimSpace(tenantKey)
	topicARN = strings.TrimSpace(topicARN)
	name = strings.TrimSpace(name)
	if tenantKey == "" || topicARN == "" || name == "" {
		return fmt.Errorf("sns: tenant, topic arn and name are required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`
INSERT INTO topics (topic_arn, name, tenant_key, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(topic_arn) DO UPDATE SET
  name = excluded.name,
  tenant_key = excluded.tenant_key,
  updated_at = excluded.updated_at
`, topicARN, name, tenantKey, now, now)
	if err != nil {
		return fmt.Errorf("sns: upsert topic: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListTopicARNsByTenant(tenantKey string) ([]string, error) {
	tenantKey = strings.TrimSpace(tenantKey)
	rows, err := s.db.Query(`SELECT topic_arn FROM topics WHERE tenant_key = ? ORDER BY topic_arn`, tenantKey)
	if err != nil {
		return nil, fmt.Errorf("sns: list topics by tenant: %w", err)
	}
	defer rows.Close()

	arns := make([]string, 0)
	for rows.Next() {
		var arn string
		if err := rows.Scan(&arn); err != nil {
			return nil, fmt.Errorf("sns: scan topic arn: %w", err)
		}
		arns = append(arns, arn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sns: iterate topic arns: %w", err)
	}
	return arns, nil
}

func (s *SQLiteStore) migrate() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sns: sqlite store is not initialized")
	}

	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("sns: create schema_migrations table: %w", err)
	}

	applied, err := s.appliedVersions()
	if err != nil {
		return err
	}

	for _, version := range migrationVersions() {
		if applied[version] {
			continue
		}
		sqlStmt := migrations[version]
		if _, err := s.db.Exec(sqlStmt); err != nil {
			return fmt.Errorf("sns: apply migration %d: %w", version, err)
		}
		if _, err := s.db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("sns: record migration %d: %w", version, err)
		}
	}

	return nil
}

func (s *SQLiteStore) appliedVersions() (map[int]bool, error) {
	rows, err := s.db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("sns: query migrations: %w", err)
	}
	defer rows.Close()

	versions := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("sns: scan migration version: %w", err)
		}
		versions[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sns: iterate migrations: %w", err)
	}
	return versions, nil
}

var migrations = map[int]string{
	1: `
CREATE TABLE IF NOT EXISTS topics (
  topic_arn TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tenant_key TEXT NOT NULL,
  attributes_json TEXT NOT NULL DEFAULT '{}',
  policy_json TEXT NOT NULL DEFAULT '{}',
  tags_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_topics_tenant_name ON topics(tenant_key, name);

CREATE TABLE IF NOT EXISTS subscriptions (
  subscription_arn TEXT PRIMARY KEY,
  topic_arn TEXT NOT NULL,
  tenant_key TEXT NOT NULL,
  protocol TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  status TEXT NOT NULL,
  token TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant ON subscriptions(tenant_key);
CREATE INDEX IF NOT EXISTS idx_subscriptions_topic ON subscriptions(topic_arn);

CREATE TABLE IF NOT EXISTS published_messages (
  message_id TEXT PRIMARY KEY,
  tenant_key TEXT NOT NULL,
  target_kind TEXT NOT NULL,
  target_ref TEXT NOT NULL,
  topic_arn TEXT,
  payload TEXT NOT NULL,
  subject TEXT,
  message_attributes_json TEXT NOT NULL DEFAULT '{}',
  published_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_published_messages_tenant ON published_messages(tenant_key);

CREATE TABLE IF NOT EXISTS delivery_attempts (
  attempt_id TEXT PRIMARY KEY,
  message_id TEXT NOT NULL,
  subscription_arn TEXT,
  endpoint_arn TEXT,
  tenant_key TEXT NOT NULL,
  protocol TEXT NOT NULL,
  status TEXT NOT NULL,
  failure_code TEXT,
  failure_message TEXT,
  attempted_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_tenant ON delivery_attempts(tenant_key);

CREATE TABLE IF NOT EXISTS platform_applications (
  platform_application_arn TEXT PRIMARY KEY,
  tenant_key TEXT NOT NULL,
  name TEXT NOT NULL,
  platform TEXT NOT NULL,
  attributes_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_platform_apps_tenant ON platform_applications(tenant_key);

CREATE TABLE IF NOT EXISTS platform_endpoints (
  endpoint_arn TEXT PRIMARY KEY,
  platform_application_arn TEXT NOT NULL,
  tenant_key TEXT NOT NULL,
  token TEXT NOT NULL,
  attributes_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_platform_endpoints_tenant ON platform_endpoints(tenant_key);

CREATE TABLE IF NOT EXISTS sms_sandbox_phone_numbers (
  phone_number TEXT PRIMARY KEY,
  tenant_key TEXT NOT NULL,
  status TEXT NOT NULL,
  language_code TEXT,
  otp_code TEXT,
  otp_expires_at TEXT,
  created_at TEXT NOT NULL,
  verified_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_sms_sandbox_tenant ON sms_sandbox_phone_numbers(tenant_key);

CREATE TABLE IF NOT EXISTS opt_out_phone_numbers (
  phone_number TEXT NOT NULL,
  tenant_key TEXT NOT NULL,
  is_opted_out INTEGER NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (phone_number, tenant_key)
);
CREATE INDEX IF NOT EXISTS idx_opt_out_tenant ON opt_out_phone_numbers(tenant_key);
`,
	2: `
ALTER TABLE subscriptions ADD COLUMN owner_account_id TEXT NOT NULL DEFAULT '';
ALTER TABLE subscriptions ADD COLUMN attributes_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE subscriptions ADD COLUMN confirmed_at TEXT;
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_topic_endpoint ON subscriptions(tenant_key, topic_arn, protocol, endpoint);
`,
	3: `
ALTER TABLE published_messages ADD COLUMN message_structure TEXT NOT NULL DEFAULT 'plain';
ALTER TABLE published_messages ADD COLUMN message_group_id TEXT NOT NULL DEFAULT '';
ALTER TABLE published_messages ADD COLUMN message_deduplication_id TEXT NOT NULL DEFAULT '';
ALTER TABLE published_messages ADD COLUMN sequence_number TEXT NOT NULL DEFAULT '';
ALTER TABLE published_messages ADD COLUMN dedup_scope_key TEXT NOT NULL DEFAULT '';
ALTER TABLE published_messages ADD COLUMN deduplicated INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_published_messages_dedup ON published_messages(tenant_key, topic_arn, dedup_scope_key, message_deduplication_id, published_at);

ALTER TABLE delivery_attempts ADD COLUMN request_snapshot_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE delivery_attempts ADD COLUMN response_snapshot_json TEXT NOT NULL DEFAULT '{}';
	`,
	4: `
ALTER TABLE platform_applications ADD COLUMN tags_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE platform_endpoints ADD COLUMN tags_json TEXT NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS sms_attributes (
  tenant_key TEXT PRIMARY KEY,
  attributes_json TEXT NOT NULL DEFAULT '{}',
  updated_at TEXT NOT NULL
);
	`,
}

func migrationVersions() []int {
	versions := make([]int, 0, len(migrations))
	for version := range migrations {
		versions = append(versions, version)
	}
	sort.Ints(versions)
	return versions
}
