package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

const (
	sqliteFileName   = "state.db"
	schemaVersionKey = "schema_version"
	schemaVersion    = "2"
)

type SQLiteRepository struct {
	db         *sql.DB
	dbPath     string
	storageDir string
	mu         sync.Mutex
}

var errSQLiteRepositoryClosed = errors.New("sqs: repository is closed")

func NewSQLiteRepository(storagePath string) (*SQLiteRepository, error) {
	storagePath = strings.TrimSpace(storagePath)
	if storagePath == "" {
		return nil, fmt.Errorf("sqs: storage path is required")
	}

	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return nil, fmt.Errorf("sqs: create storage directory: %w", err)
	}

	dbPath := filepath.Join(storagePath, sqliteFileName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqs: open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	repo := &SQLiteRepository{
		db:         db,
		dbPath:     dbPath,
		storageDir: storagePath,
	}

	if err := repo.bootstrap(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *SQLiteRepository) Load() (domain.State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		return domain.State{}, errSQLiteRepositoryClosed
	}

	state, err := r.loadLocked()
	if err != nil {
		return domain.State{}, err
	}
	return state, nil
}

func (r *SQLiteRepository) Save(state domain.State) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		return errSQLiteRepositoryClosed
	}

	return r.saveLocked(state)
}

func (r *SQLiteRepository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		return nil
	}

	err := r.db.Close()
	r.db = nil
	return err
}

func (r *SQLiteRepository) bootstrap() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		return errSQLiteRepositoryClosed
	}

	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqs: bootstrap transaction: %w", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS sqs_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sqs_queues (
			name TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			attributes_json TEXT NOT NULL,
			ordering_hint TEXT NOT NULL DEFAULT '',
			dead_letter_queue TEXT NOT NULL DEFAULT '',
			policy_json TEXT NOT NULL,
			created_at_ns INTEGER NOT NULL DEFAULT 0,
			updated_at_ns INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS sqs_messages (
			queue_name TEXT NOT NULL,
			message_id TEXT NOT NULL,
			body TEXT NOT NULL,
			attributes_json TEXT NOT NULL,
			metadata_json TEXT NOT NULL,
			tags_json TEXT NOT NULL,
			receipt_keys_json TEXT NOT NULL,
			message_group_id TEXT NOT NULL DEFAULT '',
			sequence_number INTEGER NOT NULL DEFAULT 0,
			batch_id TEXT NOT NULL DEFAULT '',
			batch_entry_id TEXT NOT NULL DEFAULT '',
			batch_entry_index INTEGER NOT NULL DEFAULT 0,
			batch_entry_count INTEGER NOT NULL DEFAULT 0,
			dead_letter_queue TEXT NOT NULL DEFAULT '',
			dead_letter_source_queue TEXT NOT NULL DEFAULT '',
			dead_lettered_at_ns INTEGER NOT NULL DEFAULT 0,
			sent_at_ns INTEGER NOT NULL DEFAULT 0,
			available_at_ns INTEGER NOT NULL DEFAULT 0,
			received_at_ns INTEGER NOT NULL DEFAULT 0,
			recovery_attempts INTEGER NOT NULL DEFAULT 0,
			recovery_detail_json TEXT NOT NULL,
			PRIMARY KEY (queue_name, message_id),
			FOREIGN KEY (queue_name) REFERENCES sqs_queues(name) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sqs_messages_queue_name ON sqs_messages(queue_name)`,
		`CREATE TABLE IF NOT EXISTS sqs_recovery_metadata (
			key TEXT PRIMARY KEY,
			queue_name TEXT NOT NULL,
			message_id TEXT NOT NULL,
			detail_json TEXT NOT NULL
		)`,
	}

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: bootstrap schema: %w", err)
		}
	}

	if err := ensureColumn(ctx, tx, "sqs_queues", "ordering_hint TEXT NOT NULL DEFAULT ''"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure queue ordering column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "message_group_id TEXT NOT NULL DEFAULT ''"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure message group column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "sequence_number INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure message sequence column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "batch_id TEXT NOT NULL DEFAULT ''"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure batch id column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "batch_entry_id TEXT NOT NULL DEFAULT ''"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure batch entry id column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "batch_entry_index INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure batch entry index column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "batch_entry_count INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure batch entry count column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "dead_letter_queue TEXT NOT NULL DEFAULT ''"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure dead letter queue column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "dead_letter_source_queue TEXT NOT NULL DEFAULT ''"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure dead letter source column: %w", err)
	}
	if err := ensureColumn(ctx, tx, "sqs_messages", "dead_lettered_at_ns INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: ensure dead lettered at column: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sqs_meta(key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, schemaVersionKey, schemaVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: bootstrap schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqs: commit bootstrap: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) loadLocked() (domain.State, error) {
	ctx := context.Background()
	state := domain.NewState()

	queueRows, err := r.db.QueryContext(ctx, `
		SELECT name, url, attributes_json, ordering_hint, dead_letter_queue, policy_json, created_at_ns, updated_at_ns
		FROM sqs_queues
		ORDER BY name
	`)
	if err != nil {
		return domain.State{}, fmt.Errorf("sqs: query queues: %w", err)
	}
	defer queueRows.Close()

	for queueRows.Next() {
		var (
			queue       domain.Queue
			attributes  string
			ordering    string
			policy      string
			createdAtNS int64
			updatedAtNS int64
		)
		if err := queueRows.Scan(&queue.Name, &queue.URL, &attributes, &ordering, &queue.Recovery.DeadLetterQueue, &policy, &createdAtNS, &updatedAtNS); err != nil {
			return domain.State{}, fmt.Errorf("sqs: scan queue: %w", err)
		}
		queue.Attributes, err = decodeStringMap(attributes)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode queue attributes: %w", err)
		}
		queue.Recovery.Policy, err = decodeStringMap(policy)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode queue policy: %w", err)
		}
		queue.OrderingHint = ordering
		queue.CreatedAt = unixNanoToTime(createdAtNS)
		queue.UpdatedAt = unixNanoToTime(updatedAtNS)
		state.Queues = append(state.Queues, queue)
	}
	if err := queueRows.Err(); err != nil {
		return domain.State{}, fmt.Errorf("sqs: iterate queues: %w", err)
	}

	messageRows, err := r.db.QueryContext(ctx, `
		SELECT queue_name, message_id, body, attributes_json, metadata_json, tags_json, receipt_keys_json,
		       message_group_id, sequence_number, batch_id, batch_entry_id, batch_entry_index, batch_entry_count,
		       dead_letter_queue, dead_letter_source_queue, dead_lettered_at_ns,
		       sent_at_ns, available_at_ns, received_at_ns, recovery_attempts, recovery_detail_json
		FROM sqs_messages
		ORDER BY queue_name, message_id
	`)
	if err != nil {
		return domain.State{}, fmt.Errorf("sqs: query messages: %w", err)
	}
	defer messageRows.Close()

	for messageRows.Next() {
		var (
			message          domain.Message
			attributes       string
			metadata         string
			tags             string
			receiptKeys      string
			messageGroupID   string
			sequenceNumber   int64
			batchID          string
			batchEntryID     string
			batchEntryIndex  int
			batchEntryCount  int
			deadLetterQueue  string
			deadLetterSource string
			deadLetteredAtNS int64
			sentAtNS         int64
			availableAtNS    int64
			receivedAtNS     int64
			recoveryDetail   string
		)
		if err := messageRows.Scan(
			&message.Queue,
			&message.MessageID,
			&message.Body,
			&attributes,
			&metadata,
			&tags,
			&receiptKeys,
			&messageGroupID,
			&sequenceNumber,
			&batchID,
			&batchEntryID,
			&batchEntryIndex,
			&batchEntryCount,
			&deadLetterQueue,
			&deadLetterSource,
			&deadLetteredAtNS,
			&sentAtNS,
			&availableAtNS,
			&receivedAtNS,
			&message.Recovery.Attempts,
			&recoveryDetail,
		); err != nil {
			return domain.State{}, fmt.Errorf("sqs: scan message: %w", err)
		}
		message.Attributes, err = decodeStringMap(attributes)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode message attributes: %w", err)
		}
		message.Metadata, err = decodeStringMap(metadata)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode message metadata: %w", err)
		}
		message.Tags, err = decodeStringSlice(tags)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode message tags: %w", err)
		}
		message.ReceiptKeys, err = decodeStringSlice(receiptKeys)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode receipt keys: %w", err)
		}
		message.MessageGroupID = messageGroupID
		message.SequenceNumber = sequenceNumber
		message.BatchID = batchID
		message.BatchEntryID = batchEntryID
		message.BatchEntryIndex = batchEntryIndex
		message.BatchEntryCount = batchEntryCount
		message.DeadLetterQueue = deadLetterQueue
		message.DeadLetterSourceQueue = deadLetterSource
		message.DeadLetteredAt = unixNanoToTime(deadLetteredAtNS)
		message.SentAt = unixNanoToTime(sentAtNS)
		message.AvailableAt = unixNanoToTime(availableAtNS)
		message.ReceivedAt = unixNanoToTime(receivedAtNS)
		message.Recovery.Detail, err = decodeStringMap(recoveryDetail)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode message recovery detail: %w", err)
		}
		state.Messages = append(state.Messages, message)
	}
	if err := messageRows.Err(); err != nil {
		return domain.State{}, fmt.Errorf("sqs: iterate messages: %w", err)
	}

	recoveryRows, err := r.db.QueryContext(ctx, `
		SELECT key, queue_name, message_id, detail_json
		FROM sqs_recovery_metadata
		ORDER BY key
	`)
	if err != nil {
		return domain.State{}, fmt.Errorf("sqs: query recovery metadata: %w", err)
	}
	defer recoveryRows.Close()

	for recoveryRows.Next() {
		var (
			key      string
			metadata domain.RecoveryMetadata
			detail   string
		)
		if err := recoveryRows.Scan(&key, &metadata.Queue, &metadata.Message, &detail); err != nil {
			return domain.State{}, fmt.Errorf("sqs: scan recovery metadata: %w", err)
		}
		metadata.Detail, err = decodeStringMap(detail)
		if err != nil {
			return domain.State{}, fmt.Errorf("sqs: decode recovery detail: %w", err)
		}
		state.RecoveryMetadata[key] = metadata
	}
	if err := recoveryRows.Err(); err != nil {
		return domain.State{}, fmt.Errorf("sqs: iterate recovery metadata: %w", err)
	}

	return state, nil
}

func (r *SQLiteRepository) saveLocked(state domain.State) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqs: begin save transaction: %w", err)
	}

	normalized := state.Clone()
	normalized.Service = "sqs"

	if _, err := tx.ExecContext(ctx, `DELETE FROM sqs_recovery_metadata`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: clear recovery metadata: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sqs_messages`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: clear messages: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sqs_queues`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: clear queues: %w", err)
	}

	queueStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sqs_queues (
			name, url, attributes_json, ordering_hint, dead_letter_queue, policy_json, created_at_ns, updated_at_ns
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: prepare queue insert: %w", err)
	}
	defer queueStmt.Close()

	for _, queue := range normalized.ListQueues() {
		attributes, err := encodeStringMap(queue.Attributes)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode queue attributes: %w", err)
		}
		policy, err := encodeStringMap(queue.Recovery.Policy)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode queue policy: %w", err)
		}
		if _, err := queueStmt.ExecContext(ctx,
			queue.Name,
			queue.URL,
			attributes,
			queue.OrderingHint,
			queue.Recovery.DeadLetterQueue,
			policy,
			timeToUnixNano(queue.CreatedAt),
			timeToUnixNano(queue.UpdatedAt),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: insert queue %q: %w", queue.Name, err)
		}
	}

	messageStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sqs_messages (
			queue_name, message_id, body, attributes_json, metadata_json, tags_json, receipt_keys_json,
			message_group_id, sequence_number, batch_id, batch_entry_id, batch_entry_index, batch_entry_count,
			dead_letter_queue, dead_letter_source_queue, dead_lettered_at_ns,
			sent_at_ns, available_at_ns, received_at_ns, recovery_attempts, recovery_detail_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: prepare message insert: %w", err)
	}
	defer messageStmt.Close()

	for _, message := range normalized.ListMessages() {
		attributes, err := encodeStringMap(message.Attributes)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode message attributes: %w", err)
		}
		metadata, err := encodeStringMap(message.Metadata)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode message metadata: %w", err)
		}
		tags, err := encodeStringSlice(message.Tags)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode message tags: %w", err)
		}
		receiptKeys, err := encodeStringSlice(message.ReceiptKeys)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode receipt keys: %w", err)
		}
		recovery, err := encodeStringMap(message.Recovery.Detail)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode message recovery detail: %w", err)
		}
		if _, err := messageStmt.ExecContext(ctx,
			message.Queue,
			message.MessageID,
			message.Body,
			attributes,
			metadata,
			tags,
			receiptKeys,
			message.MessageGroupID,
			message.SequenceNumber,
			message.BatchID,
			message.BatchEntryID,
			message.BatchEntryIndex,
			message.BatchEntryCount,
			message.DeadLetterQueue,
			message.DeadLetterSourceQueue,
			timeToUnixNano(message.DeadLetteredAt),
			timeToUnixNano(message.SentAt),
			timeToUnixNano(message.AvailableAt),
			timeToUnixNano(message.ReceivedAt),
			message.Recovery.Attempts,
			recovery,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: insert message %q/%q: %w", message.Queue, message.MessageID, err)
		}
	}

	recoveryKeys := make([]string, 0, len(normalized.RecoveryMetadata))
	for key := range normalized.RecoveryMetadata {
		recoveryKeys = append(recoveryKeys, key)
	}
	sort.Strings(recoveryKeys)
	recoveryStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sqs_recovery_metadata (key, queue_name, message_id, detail_json)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: prepare recovery insert: %w", err)
	}
	defer recoveryStmt.Close()

	for _, key := range recoveryKeys {
		metadata := normalized.RecoveryMetadata[key]
		detail, err := encodeStringMap(metadata.Detail)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: encode recovery detail: %w", err)
		}
		if _, err := recoveryStmt.ExecContext(ctx, key, metadata.Queue, metadata.Message, detail); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqs: insert recovery metadata %q: %w", key, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sqs_meta(key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, schemaVersionKey, schemaVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqs: persist schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqs: commit save transaction: %w", err)
	}

	return nil
}

func encodeStringMap(values map[string]string) (string, error) {
	if values == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeStringMap(encoded string) (map[string]string, error) {
	if strings.TrimSpace(encoded) == "" {
		return nil, nil
	}

	var values map[string]string
	if err := json.Unmarshal([]byte(encoded), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func encodeStringSlice(values []string) (string, error) {
	if values == nil {
		return "[]", nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeStringSlice(encoded string) ([]string, error) {
	if strings.TrimSpace(encoded) == "" {
		return nil, nil
	}

	var values []string
	if err := json.Unmarshal([]byte(encoded), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func ensureColumn(ctx context.Context, tx *sql.Tx, tableName, definition string) error {
	columnName := definition
	if idx := strings.IndexAny(columnName, " \t"); idx >= 0 {
		columnName = columnName[:idx]
	}

	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid          int
			name         string
			colType      string
			notNull      int
			defaultValue sql.NullString
			pk           int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, definition))
	return err
}

func timeToUnixNano(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().UnixNano()
}

func unixNanoToTime(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value).UTC()
}
