package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state     domain.State
	policy    orchestrator.EmulationPolicy
	repo      Repository
	stateHook orchestrator.StateHook
	now       func() time.Time
	mu        sync.Mutex
}

const (
	defaultPartitionKey    = "id"
	defaultBillingMode     = "PAY_PER_REQUEST"
	defaultActivationDelay = 200 * time.Millisecond
)

func New() *Service {
	return newService(domain.NewState(), nil)
}

func newService(state domain.State, repo Repository) *Service {
	return &Service{
		state: state,
		repo:  repo,
		now:   func() time.Time { return time.Now().UTC() },
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list tables",
				"create table",
				"describe table",
				"delete table",
				"get item",
				"put item",
				"update item",
				"delete item",
			},
			[]string{
				"global tables",
				"transactions",
			},
			"dynamodb",
		),
	}
}

func (s *Service) Start(context.Context) error {
	return nil
}

func (s *Service) Stop(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repo == nil {
		return nil
	}

	if err := s.repo.Close(); err != nil {
		return fmt.Errorf("dynamodb: close repository: %w", err)
	}
	s.repo = nil
	return nil
}

func (s *Service) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{
		Name:        "dynamodb",
		Description: "MildStack DynamoDB real service",
		Version:     "v1",
		Tags:        []string{"aws", "database", "nosql", "real-service"},
	}
}

func (s *Service) Policy() orchestrator.EmulationPolicy {
	return s.policy.Clone()
}

func (s *Service) RegisterRoutes(registrar orchestrator.RouteRegistrar) error {
	for _, route := range infrastructure.Routes() {
		if err := registrar.Register(route); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) AttachState(hook orchestrator.StateHook) error {
	if hook == nil {
		return fmt.Errorf("dynamodb: nil state hook")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateHook = hook
	s.publishSnapshotLocked()
	return nil
}

func (s *Service) ListTables() []domain.Table {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.VisibleTables()
}

func (s *Service) CreateTable(name, partitionKey, sortKey, billingMode string, specs ...domain.CreateTableSpec) (domain.Table, error) {
	name = strings.TrimSpace(name)
	partitionKey = strings.TrimSpace(partitionKey)
	sortKey = strings.TrimSpace(sortKey)
	billingMode = strings.TrimSpace(billingMode)
	if name == "" {
		return domain.Table{}, fmt.Errorf("dynamodb: table name is required")
	}
	if partitionKey == "" {
		partitionKey = defaultPartitionKey
	}
	if billingMode == "" {
		billingMode = defaultBillingMode
	}
	if len(specs) > 1 {
		return domain.Table{}, fmt.Errorf("dynamodb: multiple create table specifications are not supported")
	}

	spec := domain.CreateTableSpec{}
	if len(specs) == 1 {
		spec = normalizeCreateTableSpec(specs[0])
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.state.Clone()
	if next.HasTable(name) {
		return domain.Table{}, fmt.Errorf("dynamodb: table %q already exists", name)
	}

	now := s.currentTime()
	table := domain.Table{
		Name:                   name,
		PartitionKey:           partitionKey,
		SortKey:                sortKey,
		BillingMode:            billingMode,
		AttributeDefinitions:   cloneCreateTableAttributeDefinitions(spec.AttributeDefinitions),
		GlobalSecondaryIndexes: cloneCreateTableSecondaryIndexes(spec.GlobalSecondaryIndexes),
		LocalSecondaryIndexes:  cloneCreateTableSecondaryIndexes(spec.LocalSecondaryIndexes),
		Status:                 domain.TableStatusCreating,
		CreatedAt:              now,
		ActivationAt:           now.Add(defaultActivationDelay),
	}
	if err := validateCreateTableDefinition(table); err != nil {
		return domain.Table{}, err
	}
	table = next.UpsertTable(table)
	if err := s.commitStateLocked(next); err != nil {
		return domain.Table{}, err
	}
	return table, nil
}

func (s *Service) DescribeTable(name string) (domain.Table, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Table{}, fmt.Errorf("dynamodb: table name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.state.Clone()
	if s.materializeTableLocked(&next, name) {
		if err := s.commitStateLocked(next); err != nil {
			return domain.Table{}, err
		}
	}

	table, ok := s.state.Table(name)
	if !ok || table.Status == domain.TableStatusDeleting {
		return domain.Table{}, fmt.Errorf("dynamodb: table %q not found", name)
	}
	if table.Status == domain.TableStatusCreating && !s.currentTime().Before(table.ActivationAt) {
		next = s.state.Clone()
		for i := range next.Tables {
			if next.Tables[i].Name == name {
				next.Tables[i].Status = domain.TableStatusActive
				next.Tables[i].ActivationAt = time.Time{}
				if err := s.commitStateLocked(next); err != nil {
					return domain.Table{}, err
				}
				return next.Tables[i], nil
			}
		}
	}
	return table, nil
}

func (s *Service) DeleteTable(name string) (domain.Table, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Table{}, fmt.Errorf("dynamodb: table name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.state.Clone()
	if s.materializeTableLocked(&next, name) {
		if err := s.commitStateLocked(next); err != nil {
			return domain.Table{}, err
		}
	}

	table, ok := s.state.Table(name)
	if !ok || table.Status == domain.TableStatusDeleting {
		if ok && table.Status == domain.TableStatusDeleting {
			return table, nil
		}
		return domain.Table{}, fmt.Errorf("dynamodb: table %q not found", name)
	}

	next = s.state.Clone()
	for i := range next.Tables {
		if next.Tables[i].Name == name {
			next.Tables[i].Status = domain.TableStatusDeleting
			next.Tables[i].ActivationAt = time.Time{}
			next.Tables[i].DeletedAt = s.currentTime()
			if err := s.commitStateLocked(next); err != nil {
				return domain.Table{}, err
			}
			return next.Tables[i], nil
		}
	}

	return domain.Table{}, fmt.Errorf("dynamodb: table %q not found", name)
}

func (s *Service) GetItem(table, key string) (domain.Item, error) {
	table = strings.TrimSpace(table)
	key = strings.TrimSpace(key)
	if table == "" {
		return domain.Item{}, fmt.Errorf("dynamodb: table name is required")
	}
	if key == "" {
		return domain.Item{}, fmt.Errorf("dynamodb: item key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.state.Table(table)
	if !ok {
		return domain.Item{}, fmt.Errorf("dynamodb: table %q not found", table)
	}
	item, ok := s.state.Item(table, key)
	if !ok {
		return domain.Item{}, fmt.Errorf("dynamodb: item %s/%s not found", table, key)
	}
	return item, nil
}

func (s *Service) PutItem(table, key string, attributes map[string]domain.AttributeValue) (domain.Item, error) {
	table = strings.TrimSpace(table)
	key = strings.TrimSpace(key)
	if table == "" {
		return domain.Item{}, fmt.Errorf("dynamodb: table name is required")
	}
	if key == "" {
		return domain.Item{}, fmt.Errorf("dynamodb: item key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.state.Table(table)
	if !ok {
		return domain.Item{}, fmt.Errorf("dynamodb: table %q not found", table)
	}

	next := s.state.Clone()
	item := next.UpsertItem(domain.Item{
		Table:      table,
		Key:        key,
		Attributes: attributes,
	})
	if err := s.commitStateLocked(next); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func (s *Service) DeleteItem(table, key string) error {
	table = strings.TrimSpace(table)
	key = strings.TrimSpace(key)
	if table == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}
	if key == "" {
		return fmt.Errorf("dynamodb: item key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.state.Table(table)
	if !ok {
		return fmt.Errorf("dynamodb: table %q not found", table)
	}
	if !s.state.HasItem(table, key) {
		return fmt.Errorf("dynamodb: item %s/%s not found", table, key)
	}

	next := s.state.Clone()
	if !next.DeleteItem(table, key) {
		return fmt.Errorf("dynamodb: item %s/%s not found", table, key)
	}
	return s.commitStateLocked(next)
}

func (s *Service) Query(table, keyConditionExpression, filterExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue, limit *int, exclusiveStartKey map[string]domain.AttributeValue, scanIndexForward *bool, options ...domain.QueryOptions) (domain.ReadPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table = strings.TrimSpace(table)
	if table == "" {
		return domain.ReadPage{}, fmt.Errorf("dynamodb: table name is required")
	}

	tableInfo, ok := s.state.Table(table)
	if !ok {
		return domain.ReadPage{}, fmt.Errorf("dynamodb: table %q not found", table)
	}

	option := domain.QueryOptions{}
	if len(options) > 0 {
		option = options[0]
	}

	target, err := resolveQueryTarget(tableInfo, option.IndexName)
	if err != nil {
		return domain.ReadPage{}, err
	}

	plan, err := buildQueryPlan(target, keyConditionExpression, expressionAttributeNames, expressionAttributeValues)
	if err != nil {
		return domain.ReadPage{}, err
	}

	items := s.state.ListItems(table)
	candidates := make([]domain.Item, 0, len(items))
	for _, item := range items {
		matches, err := plan.matches(item, target)
		if err != nil {
			return domain.ReadPage{}, err
		}
		if matches {
			candidates = append(candidates, item)
		}
	}

	ordered := orderQueryItems(candidates, target, scanIndexForward)
	startIndex, err := locateExclusiveStartKey(ordered, target, exclusiveStartKey)
	if err != nil {
		return domain.ReadPage{}, err
	}

	filter, err := buildExpressionFilter(filterExpression, expressionAttributeNames, expressionAttributeValues)
	if err != nil {
		return domain.ReadPage{}, err
	}

	projection, err := buildProjection(option.ProjectionExpression, expressionAttributeNames, target)
	if err != nil {
		return domain.ReadPage{}, err
	}

	return pageReadItems(ordered, target, startIndex, limit, filter, projection)
}

func (s *Service) Scan(table, filterExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue, limit *int, exclusiveStartKey map[string]domain.AttributeValue) (domain.ReadPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table = strings.TrimSpace(table)
	if table == "" {
		return domain.ReadPage{}, fmt.Errorf("dynamodb: table name is required")
	}

	tableInfo, ok := s.state.Table(table)
	if !ok {
		return domain.ReadPage{}, fmt.Errorf("dynamodb: table %q not found", table)
	}

	items := s.state.ListItems(table)
	startIndex, err := locateExclusiveStartKey(items, queryTarget{Table: tableInfo, PartitionKey: tableInfo.PartitionKey, SortKey: tableInfo.SortKey}, exclusiveStartKey)
	if err != nil {
		return domain.ReadPage{}, err
	}

	filter, err := buildExpressionFilter(filterExpression, expressionAttributeNames, expressionAttributeValues)
	if err != nil {
		return domain.ReadPage{}, err
	}

	return pageReadItems(items, queryTarget{Table: tableInfo, PartitionKey: tableInfo.PartitionKey, SortKey: tableInfo.SortKey}, startIndex, limit, filter, nil)
}

func (s *Service) commitStateLocked(next domain.State) error {
	if s.repo != nil {
		if err := s.repo.Save(next); err != nil {
			return fmt.Errorf("dynamodb: persist state: %w", err)
		}
	}

	s.state = next
	s.publishSnapshotLocked()
	return nil
}

func (s *Service) publishSnapshotLocked() {
	if s.stateHook == nil {
		return
	}

	s.stateHook.Set(domain.StateKey, s.state.Snapshot())
}

func (s *Service) currentTime() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) materializeTableLocked(state *domain.State, name string) bool {
	if state == nil {
		return false
	}

	changed := false
	now := s.currentTime()
	for i := range state.Tables {
		table := &state.Tables[i]
		if table.Name != name {
			continue
		}
		switch table.Status {
		case domain.TableStatusCreating:
			if !table.ActivationAt.IsZero() && !now.Before(table.ActivationAt) {
				table.Status = domain.TableStatusActive
				table.ActivationAt = time.Time{}
				changed = true
			}
		case "":
			table.Status = domain.TableStatusActive
			changed = true
		}
		break
	}
	return changed
}

func normalizeCreateTableSpec(spec domain.CreateTableSpec) domain.CreateTableSpec {
	spec.AttributeDefinitions = normalizeCreateTableAttributeDefinitions(spec.AttributeDefinitions)
	spec.GlobalSecondaryIndexes = normalizeCreateTableSecondaryIndexes(spec.GlobalSecondaryIndexes)
	spec.LocalSecondaryIndexes = normalizeCreateTableSecondaryIndexes(spec.LocalSecondaryIndexes)
	return spec
}

func validateCreateTableDefinition(table domain.Table) error {
	if table.Name == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}
	if table.PartitionKey == "" {
		return fmt.Errorf("dynamodb: table %q partition key is required", table.Name)
	}
	if table.BillingMode == "" {
		return fmt.Errorf("dynamodb: table %q billing mode is required", table.Name)
	}
	if err := validateAttributeDefinitions(table); err != nil {
		return err
	}
	if err := validateCreateTableIndexes(table); err != nil {
		return err
	}
	return nil
}

func validateAttributeDefinitions(table domain.Table) error {
	if len(table.AttributeDefinitions) == 0 {
		return nil
	}

	definitions := make(map[string]string, len(table.AttributeDefinitions))
	for _, definition := range table.AttributeDefinitions {
		if definition.Name == "" {
			return fmt.Errorf("dynamodb: table %q has an empty attribute definition name", table.Name)
		}
		if definition.Type == "" {
			return fmt.Errorf("dynamodb: table %q attribute %q is missing a type", table.Name, definition.Name)
		}
		if _, ok := definitions[definition.Name]; ok {
			return fmt.Errorf("dynamodb: table %q has duplicate attribute definition %q", table.Name, definition.Name)
		}
		definitions[definition.Name] = definition.Type
	}

	needed := []string{table.PartitionKey}
	if table.SortKey != "" {
		needed = append(needed, table.SortKey)
	}
	for _, index := range append(table.GlobalSecondaryIndexes, table.LocalSecondaryIndexes...) {
		for _, element := range index.KeySchema {
			if name := strings.TrimSpace(element.AttributeName); name != "" {
				needed = append(needed, name)
			}
		}
	}

	for _, name := range uniqueStrings(needed) {
		if _, ok := definitions[name]; !ok {
			return fmt.Errorf("dynamodb: table %q is missing attribute definition for %q", table.Name, name)
		}
	}

	return nil
}

func validateCreateTableIndexes(table domain.Table) error {
	indexNames := make(map[string]struct{})
	for _, index := range table.GlobalSecondaryIndexes {
		if err := validateCreateTableIndex(table, index, false); err != nil {
			return err
		}
		if _, ok := indexNames[strings.ToLower(index.Name)]; ok {
			return fmt.Errorf("dynamodb: duplicate index %q", index.Name)
		}
		indexNames[strings.ToLower(index.Name)] = struct{}{}
	}
	for _, index := range table.LocalSecondaryIndexes {
		if err := validateCreateTableIndex(table, index, true); err != nil {
			return err
		}
		if _, ok := indexNames[strings.ToLower(index.Name)]; ok {
			return fmt.Errorf("dynamodb: duplicate index %q", index.Name)
		}
		indexNames[strings.ToLower(index.Name)] = struct{}{}
	}
	return nil
}

func validateCreateTableIndex(table domain.Table, index domain.SecondaryIndex, local bool) error {
	if strings.TrimSpace(index.Name) == "" {
		return fmt.Errorf("dynamodb: index name is required")
	}
	partitionKey, sortKey, err := validateSecondaryIndexKeySchema(index.KeySchema)
	if err != nil {
		return fmt.Errorf("dynamodb: index %q: %w", index.Name, err)
	}
	if local {
		if partitionKey != table.PartitionKey {
			return fmt.Errorf("dynamodb: index %q must reuse table partition key %q", index.Name, table.PartitionKey)
		}
	} else if partitionKey == table.PartitionKey {
		return fmt.Errorf("dynamodb: index %q must not reuse table partition key %q", index.Name, table.PartitionKey)
	}
	if sortKey == "" && local {
		return fmt.Errorf("dynamodb: index %q must define a RANGE key", index.Name)
	}
	if err := validateProjection(index.Projection); err != nil {
		return fmt.Errorf("dynamodb: index %q: %w", index.Name, err)
	}
	return nil
}

func validateSecondaryIndexKeySchema(keySchema []domain.KeySchemaElement) (string, string, error) {
	var (
		hashCount  int
		rangeCount int
		hashKey    string
		rangeKey   string
	)
	for _, element := range keySchema {
		switch strings.ToUpper(strings.TrimSpace(element.KeyType)) {
		case "HASH":
			hashCount++
			hashKey = strings.TrimSpace(element.AttributeName)
		case "RANGE":
			rangeCount++
			rangeKey = strings.TrimSpace(element.AttributeName)
		}
	}
	if hashCount != 1 {
		return "", "", fmt.Errorf("must define exactly one HASH key")
	}
	if rangeCount > 1 {
		return "", "", fmt.Errorf("must define at most one RANGE key")
	}
	if hashKey == "" {
		return "", "", fmt.Errorf("HASH key attribute name is required")
	}
	return hashKey, rangeKey, nil
}

func validateProjection(projection domain.Projection) error {
	switch strings.ToUpper(strings.TrimSpace(projection.Type)) {
	case "", "ALL", "KEYS_ONLY":
		return nil
	case "INCLUDE":
		if len(projection.NonKeyAttributes) == 0 {
			return fmt.Errorf("INCLUDE projection requires non-key attributes")
		}
		return nil
	default:
		return fmt.Errorf("unsupported projection type %q", projection.Type)
	}
}

func normalizeCreateTableAttributeDefinitions(source []domain.AttributeDefinition) []domain.AttributeDefinition {
	if len(source) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]domain.AttributeDefinition, 0, len(source))
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

func normalizeCreateTableSecondaryIndexes(source []domain.SecondaryIndex) []domain.SecondaryIndex {
	if len(source) == 0 {
		return nil
	}
	normalized := make([]domain.SecondaryIndex, 0, len(source))
	for _, index := range source {
		index.Name = strings.TrimSpace(index.Name)
		index.KeySchema = normalizeCreateTableKeySchema(index.KeySchema)
		index.Projection = normalizeCreateTableProjection(index.Projection)
		if index.Name == "" {
			continue
		}
		normalized = append(normalized, index)
	}
	return normalized
}

func normalizeCreateTableKeySchema(source []domain.KeySchemaElement) []domain.KeySchemaElement {
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

func normalizeCreateTableProjection(projection domain.Projection) domain.Projection {
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

func cloneCreateTableAttributeDefinitions(source []domain.AttributeDefinition) []domain.AttributeDefinition {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]domain.AttributeDefinition, len(source))
	copy(cloned, source)
	return cloned
}

func cloneCreateTableSecondaryIndexes(source []domain.SecondaryIndex) []domain.SecondaryIndex {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]domain.SecondaryIndex, len(source))
	for i, index := range source {
		cloned[i] = domain.SecondaryIndex{
			Name:      index.Name,
			KeySchema: cloneCreateTableKeySchema(index.KeySchema),
			Projection: domain.Projection{
				Type:             index.Projection.Type,
				NonKeyAttributes: append([]string(nil), index.Projection.NonKeyAttributes...),
			},
		}
	}
	return cloned
}

func cloneCreateTableKeySchema(source []domain.KeySchemaElement) []domain.KeySchemaElement {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]domain.KeySchemaElement, len(source))
	copy(cloned, source)
	return cloned
}
