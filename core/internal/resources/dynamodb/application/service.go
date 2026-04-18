package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state  domain.State
	policy orchestrator.EmulationPolicy
}

const (
	defaultPartitionKey = "id"
	defaultBillingMode  = "PAY_PER_REQUEST"
)

func New() *Service {
	return &Service{
		state: domain.NewState(),
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list tables",
				"create table",
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

	hook.Set(domain.StateKey, s.state.Snapshot())
	return nil
}

func (s *Service) ListTables() []domain.Table {
	return s.state.ListTables()
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

	return s.state.UpsertTable(name, partitionKey, sortKey, billingMode), nil
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
	if !s.state.HasTable(table) {
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
	if !s.state.HasTable(table) {
		return domain.Item{}, fmt.Errorf("dynamodb: table %q not found", table)
	}

	item := s.state.UpsertItem(domain.Item{
		Table:      table,
		Key:        key,
		Attributes: attributes,
	})
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
	if !s.state.DeleteItem(table, key) {
		return fmt.Errorf("dynamodb: item %s/%s not found", table, key)
	}
	return nil
}
