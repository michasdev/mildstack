package domain

import (
	"sort"
	"strings"
	"time"
)

const StateKey = "services/dynamodb"

const (
	TableStatusCreating = "CREATING"
	TableStatusActive   = "ACTIVE"
	TableStatusDeleting = "DELETING"
)

type State struct {
	Service string
	Tables  []Table
	Items   []Item
}

type Table struct {
	Name                   string
	PartitionKey           string
	SortKey                string
	BillingMode            string
	AttributeDefinitions   []AttributeDefinition
	GlobalSecondaryIndexes []SecondaryIndex
	LocalSecondaryIndexes  []SecondaryIndex
	Status                 string
	CreatedAt              time.Time
	ActivationAt           time.Time
	DeletedAt              time.Time
}

type AttributeDefinition struct {
	Name string
	Type string
}

type KeySchemaElement struct {
	AttributeName string
	KeyType       string
}

type Projection struct {
	Type             string
	NonKeyAttributes []string
}

type SecondaryIndex struct {
	Name       string
	KeySchema  []KeySchemaElement
	Projection Projection
}

type CreateTableSpec struct {
	AttributeDefinitions   []AttributeDefinition
	GlobalSecondaryIndexes []SecondaryIndex
	LocalSecondaryIndexes  []SecondaryIndex
}

type QueryOptions struct {
	IndexName            string
	ProjectionExpression string
}

type Item struct {
	Table      string
	Key        string
	Attributes map[string]AttributeValue
}

type ReadPage struct {
	Items            []Item
	Count            int
	ScannedCount     int
	LastEvaluatedKey map[string]AttributeValue
}

type AttributeValue struct {
	S    *string
	N    *string
	BOOL *bool
	NULL bool
	M    *map[string]AttributeValue
	L    *[]AttributeValue
}

func StringValue(value string) AttributeValue {
	copied := value
	return AttributeValue{S: &copied}
}

func NumberValue(value string) AttributeValue {
	copied := value
	return AttributeValue{N: &copied}
}

func BoolValue(value bool) AttributeValue {
	copied := value
	return AttributeValue{BOOL: &copied}
}

func NullValue() AttributeValue {
	return AttributeValue{NULL: true}
}

func MapValue(values map[string]AttributeValue) AttributeValue {
	cloned := cloneAttributes(values)
	return AttributeValue{M: &cloned}
}

func ListValue(values []AttributeValue) AttributeValue {
	cloned := cloneAttributeList(values)
	return AttributeValue{L: &cloned}
}

func (v AttributeValue) Clone() AttributeValue {
	return cloneAttributeValue(v)
}

func (v AttributeValue) Any() any {
	return attributeValueToAny(v)
}

func NewEmptyState() State {
	return State{
		Service: "dynamodb",
	}
}

func NewState() State {
	return State{
		Service: "dynamodb",
		Tables: []Table{
			{
				Name:         "mildstack-records",
				PartitionKey: "id",
				SortKey:      "version",
				BillingMode:  "PAY_PER_REQUEST",
				Status:       TableStatusActive,
				CreatedAt:    time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
			},
		},
		Items: []Item{
			{
				Table: "mildstack-records",
				Key:   "example#1",
				Attributes: map[string]AttributeValue{
					"id":      StringValue("example#1"),
					"version": NumberValue("1"),
					"title":   StringValue("bootstrap item"),
				},
			},
		},
	}
}

func (s State) ListTables() []Table {
	tables := make([]Table, len(s.Tables))
	copy(tables, s.Tables)
	for i := range tables {
		tables[i] = normalizeTable(tables[i])
	}
	sort.SliceStable(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
	return tables
}

func (s State) VisibleTables() []Table {
	tables := s.ListTables()
	visible := make([]Table, 0, len(tables))
	for _, table := range tables {
		if table.Status == TableStatusDeleting {
			continue
		}
		visible = append(visible, table)
	}
	return visible
}

func (s State) ListItems(table string) []Item {
	items := make([]Item, 0, len(s.Items))
	for _, item := range s.Items {
		if item.Table == table {
			items = append(items, Item{
				Table:      item.Table,
				Key:        item.Key,
				Attributes: cloneAttributes(item.Attributes),
			})
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
	return items
}

func (s State) Table(name string) (Table, bool) {
	for _, table := range s.Tables {
		if table.Name == name {
			return normalizeTable(table), true
		}
	}
	return Table{}, false
}

func (s State) Item(table, key string) (Item, bool) {
	for _, item := range s.Items {
		if item.Table == table && item.Key == key {
			return Item{
				Table:      item.Table,
				Key:        item.Key,
				Attributes: cloneAttributes(item.Attributes),
			}, true
		}
	}
	return Item{}, false
}

func (s State) HasTable(name string) bool {
	_, ok := s.Table(name)
	return ok
}

func (s State) HasItem(table, key string) bool {
	_, ok := s.Item(table, key)
	return ok
}

func (s *State) UpsertTable(table Table) Table {
	table = normalizeTable(table)
	for i := range s.Tables {
		if s.Tables[i].Name == table.Name {
			s.Tables[i] = table
			return s.Tables[i]
		}
	}

	s.Tables = append(s.Tables, table)
	return table
}

func (s *State) UpsertItem(item Item) Item {
	cloned := Item{
		Table:      item.Table,
		Key:        item.Key,
		Attributes: cloneAttributes(item.Attributes),
	}

	for i := range s.Items {
		if s.Items[i].Table == cloned.Table && s.Items[i].Key == cloned.Key {
			s.Items[i] = cloned
			return Item{
				Table:      s.Items[i].Table,
				Key:        s.Items[i].Key,
				Attributes: cloneAttributes(s.Items[i].Attributes),
			}
		}
	}

	s.Items = append(s.Items, cloned)
	return Item{
		Table:      cloned.Table,
		Key:        cloned.Key,
		Attributes: cloneAttributes(cloned.Attributes),
	}
}

func (s *State) DeleteItem(table, key string) bool {
	for i := range s.Items {
		if s.Items[i].Table == table && s.Items[i].Key == key {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			return true
		}
	}
	return false
}

func (s State) Snapshot() map[string]any {
	tables := make([]any, 0, len(s.Tables))
	for _, table := range s.ListTables() {
		tables = append(tables, map[string]any{
			"name":                     table.Name,
			"partition_key":            table.PartitionKey,
			"sort_key":                 table.SortKey,
			"billing_mode":             table.BillingMode,
			"attribute_definitions":    copyAttributeDefinitions(table.AttributeDefinitions),
			"global_secondary_indexes": copySecondaryIndexes(table.GlobalSecondaryIndexes),
			"local_secondary_indexes":  copySecondaryIndexes(table.LocalSecondaryIndexes),
			"status":                   table.Status,
			"created_at":               snapshotTime(table.CreatedAt),
			"activation_at":            snapshotTime(table.ActivationAt),
			"deleted_at":               snapshotTime(table.DeletedAt),
		})
	}

	items := make([]any, 0, len(s.Items))
	for _, item := range s.sortedItems() {
		items = append(items, map[string]any{
			"table":      item.Table,
			"key":        item.Key,
			"attributes": copyAttributesAny(item.Attributes),
		})
	}

	return map[string]any{
		"service": s.Service,
		"tables":  tables,
		"items":   items,
	}
}

func (s State) Clone() State {
	cloned := State{
		Service: s.Service,
		Tables:  make([]Table, len(s.Tables)),
		Items:   make([]Item, len(s.Items)),
	}
	copy(cloned.Tables, s.Tables)
	for i, item := range s.Items {
		cloned.Items[i] = Item{
			Table:      item.Table,
			Key:        item.Key,
			Attributes: cloneAttributes(item.Attributes),
		}
	}
	for i := range cloned.Tables {
		cloned.Tables[i].AttributeDefinitions = cloneAttributeDefinitions(cloned.Tables[i].AttributeDefinitions)
		cloned.Tables[i].GlobalSecondaryIndexes = cloneSecondaryIndexes(cloned.Tables[i].GlobalSecondaryIndexes)
		cloned.Tables[i].LocalSecondaryIndexes = cloneSecondaryIndexes(cloned.Tables[i].LocalSecondaryIndexes)
	}
	return cloned
}

func normalizeTable(table Table) Table {
	table.Name = strings.TrimSpace(table.Name)
	table.PartitionKey = strings.TrimSpace(table.PartitionKey)
	table.SortKey = strings.TrimSpace(table.SortKey)
	table.BillingMode = strings.TrimSpace(table.BillingMode)
	table.AttributeDefinitions = normalizeAttributeDefinitions(table.AttributeDefinitions)
	table.GlobalSecondaryIndexes = normalizeSecondaryIndexes(table.GlobalSecondaryIndexes)
	table.LocalSecondaryIndexes = normalizeSecondaryIndexes(table.LocalSecondaryIndexes)
	table.Status = strings.ToUpper(strings.TrimSpace(table.Status))

	switch table.Status {
	case "", TableStatusActive:
		table.Status = TableStatusActive
	case TableStatusCreating, TableStatusDeleting:
		// leave as-is
	default:
		table.Status = TableStatusActive
	}

	return table
}

func snapshotTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func (s State) sortedItems() []Item {
	items := make([]Item, len(s.Items))
	for i, item := range s.Items {
		items[i] = Item{
			Table:      item.Table,
			Key:        item.Key,
			Attributes: cloneAttributes(item.Attributes),
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Table == items[j].Table {
			return items[i].Key < items[j].Key
		}
		return items[i].Table < items[j].Table
	})
	return items
}

func cloneAttributes(attributes map[string]AttributeValue) map[string]AttributeValue {
	if attributes == nil {
		return nil
	}

	cloned := make(map[string]AttributeValue, len(attributes))
	for key, value := range attributes {
		cloned[key] = cloneAttributeValue(value)
	}
	return cloned
}

func cloneAttributeList(values []AttributeValue) []AttributeValue {
	if values == nil {
		return nil
	}

	cloned := make([]AttributeValue, len(values))
	for i, value := range values {
		cloned[i] = cloneAttributeValue(value)
	}
	return cloned
}

func cloneAttributeValue(value AttributeValue) AttributeValue {
	cloned := AttributeValue{
		NULL: value.NULL,
	}
	if value.S != nil {
		copied := *value.S
		cloned.S = &copied
	}
	if value.N != nil {
		copied := *value.N
		cloned.N = &copied
	}
	if value.BOOL != nil {
		copied := *value.BOOL
		cloned.BOOL = &copied
	}
	if value.M != nil {
		clonedMap := cloneAttributes(*value.M)
		cloned.M = &clonedMap
	}
	if value.L != nil {
		clonedList := cloneAttributeList(*value.L)
		cloned.L = &clonedList
	}
	return cloned
}

func copyAttributesAny(attributes map[string]AttributeValue) map[string]any {
	if attributes == nil {
		return nil
	}

	copied := make(map[string]any, len(attributes))
	for key, value := range attributes {
		copied[key] = attributeValueToAny(value)
	}
	return copied
}

func attributeValueToAny(value AttributeValue) any {
	switch {
	case value.S != nil:
		return *value.S
	case value.N != nil:
		return *value.N
	case value.BOOL != nil:
		return *value.BOOL
	case value.NULL:
		return nil
	case value.M != nil:
		copied := make(map[string]any, len(*value.M))
		for key, child := range *value.M {
			copied[key] = attributeValueToAny(child)
		}
		return copied
	case value.L != nil:
		copied := make([]any, len(*value.L))
		for i, child := range *value.L {
			copied[i] = attributeValueToAny(child)
		}
		return copied
	default:
		return nil
	}
}

func cloneAttributeDefinitions(source []AttributeDefinition) []AttributeDefinition {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]AttributeDefinition, len(source))
	copy(cloned, source)
	return cloned
}

func cloneSecondaryIndexes(source []SecondaryIndex) []SecondaryIndex {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]SecondaryIndex, len(source))
	for i, index := range source {
		cloned[i] = cloneSecondaryIndex(index)
	}
	return cloned
}

func cloneSecondaryIndex(index SecondaryIndex) SecondaryIndex {
	index.KeySchema = cloneKeySchema(index.KeySchema)
	index.Projection = cloneProjection(index.Projection)
	return index
}

func cloneKeySchema(source []KeySchemaElement) []KeySchemaElement {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]KeySchemaElement, len(source))
	copy(cloned, source)
	return cloned
}

func cloneProjection(projection Projection) Projection {
	projection.NonKeyAttributes = cloneStrings(projection.NonKeyAttributes)
	return projection
}

func copyAttributeDefinitions(source []AttributeDefinition) []any {
	if len(source) == 0 {
		return nil
	}
	copied := make([]any, len(source))
	for i, definition := range source {
		copied[i] = map[string]any{
			"name": definition.Name,
			"type": definition.Type,
		}
	}
	return copied
}

func copySecondaryIndexes(source []SecondaryIndex) []any {
	if len(source) == 0 {
		return nil
	}
	copied := make([]any, len(source))
	for i, index := range source {
		copied[i] = map[string]any{
			"name":       index.Name,
			"key_schema": copyKeySchema(index.KeySchema),
			"projection": map[string]any{
				"type":               index.Projection.Type,
				"non_key_attributes": cloneStrings(index.Projection.NonKeyAttributes),
			},
		}
	}
	return copied
}

func copyKeySchema(source []KeySchemaElement) []any {
	if len(source) == 0 {
		return nil
	}
	copied := make([]any, len(source))
	for i, element := range source {
		copied[i] = map[string]any{
			"attribute_name": element.AttributeName,
			"key_type":       element.KeyType,
		}
	}
	return copied
}

func normalizeAttributeDefinitions(source []AttributeDefinition) []AttributeDefinition {
	if len(source) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(source))
	normalized := make([]AttributeDefinition, 0, len(source))
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

func normalizeSecondaryIndexes(source []SecondaryIndex) []SecondaryIndex {
	if len(source) == 0 {
		return nil
	}
	normalized := make([]SecondaryIndex, 0, len(source))
	for _, index := range source {
		index.Name = strings.TrimSpace(index.Name)
		index.KeySchema = normalizeKeySchema(index.KeySchema)
		index.Projection = normalizeProjection(index.Projection)
		if index.Name == "" {
			continue
		}
		normalized = append(normalized, index)
	}
	return normalized
}

func normalizeKeySchema(source []KeySchemaElement) []KeySchemaElement {
	if len(source) == 0 {
		return nil
	}
	normalized := make([]KeySchemaElement, 0, len(source))
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

func normalizeProjection(projection Projection) Projection {
	projection.Type = strings.ToUpper(strings.TrimSpace(projection.Type))
	switch projection.Type {
	case "", "ALL":
		projection.Type = "ALL"
		projection.NonKeyAttributes = nil
	case "KEYS_ONLY":
		projection.NonKeyAttributes = nil
	case "INCLUDE":
		projection.NonKeyAttributes = uniqueStrings(projection.NonKeyAttributes)
	default:
		projection.Type = "ALL"
		projection.NonKeyAttributes = nil
	}
	return projection
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func uniqueStrings(values []string) []string {
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
