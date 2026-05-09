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
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

const (
	sqliteFileName   = "state.db"
	schemaVersionKey = "schema_version"
	schemaVersion    = "3"
)

type SQLiteRepository struct {
	db         *sql.DB
	dbPath     string
	storageDir string
	mu         sync.Mutex
}

var errSQLiteRepositoryClosed = errors.New("dynamodb: repository is closed")

func NewSQLiteRepository(storagePath string) (*SQLiteRepository, error) {
	storagePath = strings.TrimSpace(storagePath)
	if storagePath == "" {
		return nil, fmt.Errorf("dynamodb: storage path is required")
	}

	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return nil, fmt.Errorf("dynamodb: create storage directory: %w", err)
	}

	dbPath := filepath.Join(storagePath, sqliteFileName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("dynamodb: open sqlite database: %w", err)
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
	if len(state.Tables) == 0 {
		state = domain.NewEmptyState()
		if err := r.saveLocked(state); err != nil {
			return domain.State{}, err
		}
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
		return fmt.Errorf("dynamodb: bootstrap transaction: %w", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS dynamodb_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS dynamodb_tables (
			name TEXT PRIMARY KEY,
			partition_key TEXT NOT NULL,
			sort_key TEXT NOT NULL,
			billing_mode TEXT NOT NULL,
			attribute_definitions_json TEXT NOT NULL DEFAULT '[]',
			global_secondary_indexes_json TEXT NOT NULL DEFAULT '[]',
			local_secondary_indexes_json TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'ACTIVE',
			created_at_ns INTEGER NOT NULL DEFAULT 0,
			activation_at_ns INTEGER NOT NULL DEFAULT 0,
			deleted_at_ns INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS dynamodb_items (
			table_name TEXT NOT NULL,
			item_key TEXT NOT NULL,
			attributes_json TEXT NOT NULL,
			PRIMARY KEY (table_name, item_key),
			FOREIGN KEY (table_name) REFERENCES dynamodb_tables(name) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dynamodb_items_table_name ON dynamodb_items(table_name)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("dynamodb: bootstrap schema: %w", err)
		}
	}

	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "status", "TEXT NOT NULL DEFAULT 'ACTIVE'"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "attribute_definitions_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "global_secondary_indexes_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "local_secondary_indexes_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "created_at_ns", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "activation_at_ns", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ensureTableColumn(ctx, tx, "dynamodb_tables", "deleted_at_ns", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dynamodb_meta(key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, schemaVersionKey, schemaVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("dynamodb: bootstrap schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("dynamodb: commit bootstrap: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) loadLocked() (domain.State, error) {
	ctx := context.Background()
	state := domain.State{Service: "dynamodb"}

	tableRows, err := r.db.QueryContext(ctx, `
		SELECT name, partition_key, sort_key, billing_mode, attribute_definitions_json, global_secondary_indexes_json, local_secondary_indexes_json, status, created_at_ns, activation_at_ns, deleted_at_ns
		FROM dynamodb_tables
		ORDER BY name
	`)
	if err != nil {
		return domain.State{}, fmt.Errorf("dynamodb: query tables: %w", err)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var table domain.Table
		var (
			attributeDefinitionsJSON   string
			globalSecondaryIndexesJSON string
			localSecondaryIndexesJSON  string
			createdAtNS                int64
			activationAtNS             int64
			deletedAtNS                int64
		)
		if err := tableRows.Scan(&table.Name, &table.PartitionKey, &table.SortKey, &table.BillingMode, &attributeDefinitionsJSON, &globalSecondaryIndexesJSON, &localSecondaryIndexesJSON, &table.Status, &createdAtNS, &activationAtNS, &deletedAtNS); err != nil {
			return domain.State{}, fmt.Errorf("dynamodb: scan table: %w", err)
		}
		if err := json.Unmarshal([]byte(attributeDefinitionsJSON), &table.AttributeDefinitions); err != nil {
			return domain.State{}, fmt.Errorf("dynamodb: decode table attribute definitions %q: %w", table.Name, err)
		}
		if err := json.Unmarshal([]byte(globalSecondaryIndexesJSON), &table.GlobalSecondaryIndexes); err != nil {
			return domain.State{}, fmt.Errorf("dynamodb: decode table global secondary indexes %q: %w", table.Name, err)
		}
		if err := json.Unmarshal([]byte(localSecondaryIndexesJSON), &table.LocalSecondaryIndexes); err != nil {
			return domain.State{}, fmt.Errorf("dynamodb: decode table local secondary indexes %q: %w", table.Name, err)
		}
		table.CreatedAt = unixNanoToTime(createdAtNS)
		table.ActivationAt = unixNanoToTime(activationAtNS)
		table.DeletedAt = unixNanoToTime(deletedAtNS)
		state.Tables = append(state.Tables, table)
	}
	if err := tableRows.Err(); err != nil {
		return domain.State{}, fmt.Errorf("dynamodb: iterate tables: %w", err)
	}

	itemRows, err := r.db.QueryContext(ctx, `
		SELECT table_name, item_key, attributes_json
		FROM dynamodb_items
		ORDER BY table_name, item_key
	`)
	if err != nil {
		return domain.State{}, fmt.Errorf("dynamodb: query items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var (
			item       domain.Item
			attributes string
		)
		if err := itemRows.Scan(&item.Table, &item.Key, &attributes); err != nil {
			return domain.State{}, fmt.Errorf("dynamodb: scan item: %w", err)
		}
		decoded, err := decodeAttributes(attributes)
		if err != nil {
			return domain.State{}, err
		}
		item.Attributes = decoded
		state.Items = append(state.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return domain.State{}, fmt.Errorf("dynamodb: iterate items: %w", err)
	}

	return state, nil
}

func (r *SQLiteRepository) saveLocked(state domain.State) error {
	normalized := state.Clone()
	if normalized.Service == "" {
		normalized.Service = "dynamodb"
	}
	if err := validatePersistedState(normalized); err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("dynamodb: save transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM dynamodb_items`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("dynamodb: clear items: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dynamodb_tables`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("dynamodb: clear tables: %w", err)
	}

	tableStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO dynamodb_tables(name, partition_key, sort_key, billing_mode, attribute_definitions_json, global_secondary_indexes_json, local_secondary_indexes_json, status, created_at_ns, activation_at_ns, deleted_at_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("dynamodb: prepare table insert: %w", err)
	}
	defer tableStmt.Close()

	for _, table := range normalized.ListTables() {
		if _, err := tableStmt.ExecContext(ctx,
			table.Name,
			table.PartitionKey,
			table.SortKey,
			table.BillingMode,
			string(marshalJSONOrPanic(table.AttributeDefinitions)),
			string(marshalJSONOrPanic(table.GlobalSecondaryIndexes)),
			string(marshalJSONOrPanic(table.LocalSecondaryIndexes)),
			table.Status,
			timeToUnixNano(table.CreatedAt),
			timeToUnixNano(table.ActivationAt),
			timeToUnixNano(table.DeletedAt),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("dynamodb: insert table %q: %w", table.Name, err)
		}
	}

	itemStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO dynamodb_items(table_name, item_key, attributes_json)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("dynamodb: prepare item insert: %w", err)
	}
	defer itemStmt.Close()

	for _, table := range normalized.ListTables() {
		for _, item := range normalized.ListItems(table.Name) {
			attributesJSON, err := marshalAttributesStable(item.Attributes)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			if _, err := itemStmt.ExecContext(ctx, item.Table, item.Key, string(attributesJSON)); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("dynamodb: insert item %s/%s: %w", item.Table, item.Key, err)
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dynamodb_meta(key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, schemaVersionKey, schemaVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("dynamodb: update schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("dynamodb: commit save: %w", err)
	}

	return nil
}

func validatePersistedState(state domain.State) error {
	if state.Service != "dynamodb" {
		return fmt.Errorf("dynamodb: invalid service %q", state.Service)
	}

	tables := make(map[string]struct{}, len(state.Tables))
	for _, table := range state.Tables {
		table = normalizePersistedTable(table)
		if table.Name == "" {
			return fmt.Errorf("dynamodb: invalid table: empty name")
		}
		if table.PartitionKey == "" {
			return fmt.Errorf("dynamodb: invalid table %q: empty partition key", table.Name)
		}
		if table.BillingMode == "" {
			return fmt.Errorf("dynamodb: invalid table %q: empty billing mode", table.Name)
		}
		if err := validatePersistedIndexDefinitions(table); err != nil {
			return err
		}
		if _, ok := tables[table.Name]; ok {
			return fmt.Errorf("dynamodb: duplicate table %q", table.Name)
		}
		tables[table.Name] = struct{}{}
	}

	for _, item := range state.Items {
		item.Table = strings.TrimSpace(item.Table)
		item.Key = strings.TrimSpace(item.Key)
		if item.Table == "" {
			return fmt.Errorf("dynamodb: invalid item: empty table name")
		}
		if item.Key == "" {
			return fmt.Errorf("dynamodb: invalid item %q: empty key", item.Table)
		}
		if _, ok := tables[item.Table]; !ok {
			return fmt.Errorf("dynamodb: invalid item %s/%s: table not found", item.Table, item.Key)
		}
	}

	return nil
}

func normalizePersistedTable(table domain.Table) domain.Table {
	table.Name = strings.TrimSpace(table.Name)
	table.PartitionKey = strings.TrimSpace(table.PartitionKey)
	table.SortKey = strings.TrimSpace(table.SortKey)
	table.BillingMode = strings.TrimSpace(table.BillingMode)
	table.AttributeDefinitions = normalizePersistedAttributeDefinitions(table.AttributeDefinitions)
	table.GlobalSecondaryIndexes = normalizePersistedSecondaryIndexes(table.GlobalSecondaryIndexes)
	table.LocalSecondaryIndexes = normalizePersistedSecondaryIndexes(table.LocalSecondaryIndexes)
	table.Status = strings.ToUpper(strings.TrimSpace(table.Status))

	switch table.Status {
	case "", domain.TableStatusActive:
		table.Status = domain.TableStatusActive
	case domain.TableStatusCreating, domain.TableStatusDeleting:
	default:
		table.Status = domain.TableStatusActive
	}

	return table
}

func validatePersistedIndexDefinitions(table domain.Table) error {
	indexNames := map[string]struct{}{}
	for _, index := range table.GlobalSecondaryIndexes {
		if err := validatePersistedSecondaryIndex(table, index, false); err != nil {
			return fmt.Errorf("dynamodb: invalid table %q global secondary index: %w", table.Name, err)
		}
		if _, ok := indexNames[index.Name]; ok {
			return fmt.Errorf("dynamodb: invalid table %q: duplicate index %q", table.Name, index.Name)
		}
		indexNames[index.Name] = struct{}{}
	}
	for _, index := range table.LocalSecondaryIndexes {
		if err := validatePersistedSecondaryIndex(table, index, true); err != nil {
			return fmt.Errorf("dynamodb: invalid table %q local secondary index: %w", table.Name, err)
		}
		if _, ok := indexNames[index.Name]; ok {
			return fmt.Errorf("dynamodb: invalid table %q: duplicate index %q", table.Name, index.Name)
		}
		indexNames[index.Name] = struct{}{}
	}
	return nil
}

func validatePersistedSecondaryIndex(table domain.Table, index domain.SecondaryIndex, local bool) error {
	if strings.TrimSpace(index.Name) == "" {
		return fmt.Errorf("empty name")
	}
	if len(index.KeySchema) == 0 {
		return fmt.Errorf("index %q has no key schema", index.Name)
	}
	var hashCount, rangeCount int
	var partitionKey, sortKey string
	for _, element := range index.KeySchema {
		switch strings.ToUpper(strings.TrimSpace(element.KeyType)) {
		case "HASH":
			hashCount++
			partitionKey = strings.TrimSpace(element.AttributeName)
		case "RANGE":
			rangeCount++
			sortKey = strings.TrimSpace(element.AttributeName)
		}
	}
	if hashCount != 1 {
		return fmt.Errorf("index %q must have exactly one HASH key", index.Name)
	}
	if local {
		if partitionKey != table.PartitionKey {
			return fmt.Errorf("index %q must reuse table partition key %q", index.Name, table.PartitionKey)
		}
	}
	if rangeCount > 1 {
		return fmt.Errorf("index %q has duplicate RANGE keys", index.Name)
	}
	if strings.EqualFold(index.Projection.Type, "INCLUDE") && len(index.Projection.NonKeyAttributes) == 0 {
		return fmt.Errorf("index %q INCLUDE projection requires non-key attributes", index.Name)
	}
	_ = sortKey
	return nil
}

func normalizePersistedAttributeDefinitions(source []domain.AttributeDefinition) []domain.AttributeDefinition {
	if len(source) == 0 {
		return nil
	}
	normalized := make([]domain.AttributeDefinition, 0, len(source))
	seen := make(map[string]struct{}, len(source))
	for _, definition := range source {
		definition.Name = strings.TrimSpace(definition.Name)
		definition.Type = strings.ToUpper(strings.TrimSpace(definition.Type))
		if definition.Name == "" {
			continue
		}
		if _, ok := seen[definition.Name]; ok {
			continue
		}
		seen[definition.Name] = struct{}{}
		normalized = append(normalized, definition)
	}
	return normalized
}

func normalizePersistedSecondaryIndexes(source []domain.SecondaryIndex) []domain.SecondaryIndex {
	if len(source) == 0 {
		return nil
	}
	normalized := make([]domain.SecondaryIndex, 0, len(source))
	for _, index := range source {
		index.Name = strings.TrimSpace(index.Name)
		index.KeySchema = normalizePersistedKeySchema(index.KeySchema)
		index.Projection = normalizePersistedProjection(index.Projection)
		if index.Name == "" {
			continue
		}
		normalized = append(normalized, index)
	}
	return normalized
}

func normalizePersistedKeySchema(source []domain.KeySchemaElement) []domain.KeySchemaElement {
	if len(source) == 0 {
		return nil
	}
	normalized := make([]domain.KeySchemaElement, 0, len(source))
	seen := map[string]struct{}{}
	for _, element := range source {
		element.AttributeName = strings.TrimSpace(element.AttributeName)
		element.KeyType = strings.ToUpper(strings.TrimSpace(element.KeyType))
		if element.AttributeName == "" || element.KeyType == "" {
			continue
		}
		key := element.KeyType + ":" + element.AttributeName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, element)
	}
	return normalized
}

func normalizePersistedProjection(projection domain.Projection) domain.Projection {
	projection.Type = strings.ToUpper(strings.TrimSpace(projection.Type))
	switch projection.Type {
	case "", "ALL":
		projection.Type = "ALL"
		projection.NonKeyAttributes = nil
	case "KEYS_ONLY":
		projection.NonKeyAttributes = nil
	case "INCLUDE":
		projection.NonKeyAttributes = uniqueStringsLocal(projection.NonKeyAttributes)
	default:
		projection.Type = "ALL"
		projection.NonKeyAttributes = nil
	}
	return projection
}

func marshalJSONOrPanic(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func uniqueStringsLocal(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func ensureTableColumn(ctx context.Context, tx *sql.Tx, tableName, columnName, definition string) error {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return fmt.Errorf("dynamodb: inspect %s columns: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue sql.NullString
			pk           int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("dynamodb: scan %s columns: %w", tableName, err)
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("dynamodb: iterate %s columns: %w", tableName, err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, tableName, columnName, definition)); err != nil {
		return fmt.Errorf("dynamodb: add %s.%s column: %w", tableName, columnName, err)
	}
	return nil
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

func marshalAttributesStable(attributes map[string]domain.AttributeValue) ([]byte, error) {
	if attributes == nil {
		return []byte("null"), nil
	}

	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.Grow(len(attributes) * 8)
	builder.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.Quote(key))
		builder.WriteByte(':')
		valueJSON, err := marshalAttributeValueStable(attributes[key])
		if err != nil {
			return nil, err
		}
		builder.Write(valueJSON)
	}
	builder.WriteByte('}')
	return []byte(builder.String()), nil
}

func marshalAttributeValueStable(value domain.AttributeValue) ([]byte, error) {
	parts := make([]string, 0, 6)
	if value.S != nil {
		parts = append(parts, `"S":`+strconv.Quote(*value.S))
	}
	if value.N != nil {
		parts = append(parts, `"N":`+strconv.Quote(*value.N))
	}
	if value.BOOL != nil {
		if *value.BOOL {
			parts = append(parts, `"BOOL":true`)
		} else {
			parts = append(parts, `"BOOL":false`)
		}
	}
	if value.NULL {
		parts = append(parts, `"NULL":true`)
	}
	if value.M != nil {
		nested, err := marshalAttributesStable(*value.M)
		if err != nil {
			return nil, err
		}
		parts = append(parts, `"M":`+string(nested))
	}
	if value.L != nil {
		nested, err := marshalAttributeListStable(*value.L)
		if err != nil {
			return nil, err
		}
		parts = append(parts, `"L":`+string(nested))
	}
	if len(parts) == 0 {
		return []byte("null"), nil
	}
	return []byte("{" + strings.Join(parts, ",") + "}"), nil
}

func marshalAttributeListStable(values []domain.AttributeValue) ([]byte, error) {
	if values == nil {
		return []byte("null"), nil
	}

	items := make([]string, len(values))
	for i, value := range values {
		nested, err := marshalAttributeValueStable(value)
		if err != nil {
			return nil, err
		}
		items[i] = string(nested)
	}
	return []byte("[" + strings.Join(items, ",") + "]"), nil
}

func decodeAttributes(raw string) (map[string]domain.AttributeValue, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}

	attributes := make(map[string]json.RawMessage)
	if err := json.Unmarshal([]byte(raw), &attributes); err != nil {
		return nil, fmt.Errorf("dynamodb: decode item attributes: %w", err)
	}

	decoded := make(map[string]domain.AttributeValue, len(attributes))
	for key, value := range attributes {
		decodedValue, err := decodeAttributeValue(value)
		if err != nil {
			return nil, fmt.Errorf("dynamodb: decode item attributes %q: %w", key, err)
		}
		decoded[key] = decodedValue
	}
	return decoded, nil
}

func decodeAttributeValue(raw json.RawMessage) (domain.AttributeValue, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return domain.NullValue(), nil
	}

	switch raw[0] {
	case '"':
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode string attribute: %w", err)
		}
		return domain.StringValue(value), nil
	case '{':
		typed, ok, err := decodeStoredAttributeValue(raw)
		if err != nil {
			return domain.AttributeValue{}, err
		}
		if ok {
			return typed, nil
		}

		legacy := make(map[string]json.RawMessage)
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode legacy map attribute: %w", err)
		}
		values := make(map[string]domain.AttributeValue, len(legacy))
		for key, child := range legacy {
			decoded, err := decodeAttributeValue(child)
			if err != nil {
				return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode legacy map attribute %q: %w", key, err)
			}
			values[key] = decoded
		}
		return domain.MapValue(values), nil
	case '[':
		var legacy []json.RawMessage
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode list attribute: %w", err)
		}
		values := make([]domain.AttributeValue, len(legacy))
		for i, child := range legacy {
			decoded, err := decodeAttributeValue(child)
			if err != nil {
				return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode list attribute[%d]: %w", i, err)
			}
			values[i] = decoded
		}
		return domain.ListValue(values), nil
	case 't', 'f':
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode bool attribute: %w", err)
		}
		return domain.BoolValue(value), nil
	default:
		var value json.Number
		if err := json.Unmarshal(raw, &value); err != nil {
			return domain.AttributeValue{}, fmt.Errorf("dynamodb: decode number attribute: %w", err)
		}
		return domain.NumberValue(value.String()), nil
	}
}

type storedAttributeValue struct {
	S    *string                    `json:"S,omitempty"`
	N    *string                    `json:"N,omitempty"`
	BOOL *bool                      `json:"BOOL,omitempty"`
	NULL bool                       `json:"NULL,omitempty"`
	M    map[string]json.RawMessage `json:"M,omitempty"`
	L    []json.RawMessage          `json:"L,omitempty"`
}

func decodeStoredAttributeValue(raw json.RawMessage) (domain.AttributeValue, bool, error) {
	var stored storedAttributeValue
	if err := json.Unmarshal(raw, &stored); err != nil {
		return domain.AttributeValue{}, false, fmt.Errorf("dynamodb: decode stored attribute value: %w", err)
	}

	if stored.S != nil {
		return domain.StringValue(*stored.S), true, nil
	}
	if stored.N != nil {
		return domain.NumberValue(*stored.N), true, nil
	}
	if stored.BOOL != nil {
		return domain.BoolValue(*stored.BOOL), true, nil
	}
	if stored.NULL {
		return domain.NullValue(), true, nil
	}
	if stored.M != nil {
		values := make(map[string]domain.AttributeValue, len(stored.M))
		for key, child := range stored.M {
			decoded, err := decodeAttributeValue(child)
			if err != nil {
				return domain.AttributeValue{}, false, fmt.Errorf("dynamodb: decode stored map attribute %q: %w", key, err)
			}
			values[key] = decoded
		}
		return domain.MapValue(values), true, nil
	}
	if stored.L != nil {
		values := make([]domain.AttributeValue, len(stored.L))
		for i, child := range stored.L {
			decoded, err := decodeAttributeValue(child)
			if err != nil {
				return domain.AttributeValue{}, false, fmt.Errorf("dynamodb: decode stored list attribute[%d]: %w", i, err)
			}
			values[i] = decoded
		}
		return domain.ListValue(values), true, nil
	}

	return domain.AttributeValue{}, false, nil
}
