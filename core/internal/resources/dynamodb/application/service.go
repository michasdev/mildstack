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

func (s *Service) CreateTable(name, partitionKey, sortKey, billingMode string) (domain.Table, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.state.Clone()
	if next.HasTable(name) {
		return domain.Table{}, fmt.Errorf("dynamodb: table %q already exists", name)
	}

	now := s.currentTime()
	table := next.UpsertTable(domain.Table{
		Name:         name,
		PartitionKey: partitionKey,
		SortKey:      sortKey,
		BillingMode:  billingMode,
		Status:       domain.TableStatusCreating,
		CreatedAt:    now,
		ActivationAt: now.Add(defaultActivationDelay),
	})
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
	if table.Status == domain.TableStatusCreating {
		return domain.Table{}, fmt.Errorf("dynamodb: table %q not found", name)
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
	if table.Status == domain.TableStatusCreating {
		return domain.Table{}, fmt.Errorf("dynamodb: table %q is still creating", name)
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

func (s *Service) PutItem(table, key string, attributes map[string]string) (domain.Item, error) {
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
