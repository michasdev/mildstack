package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/sns/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
	"github.com/michasdev/mildstack/core/internal/resources/sns/infrastructure"
	sqscontracts "github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
)

var _ orchestrator.Service = (*Service)(nil)

type StorageConfig struct {
	BaseDir    string
	InstanceID string
}

type Service struct {
	policy        orchestrator.EmulationPolicy
	store         *infrastructure.SQLiteStore
	stateHook     orchestrator.StateHook
	observability *snsObservability
	sqsBridge     snsSQSBridge
}

type snsSQSBridge interface {
	SendMessage(queueName string, request sqscontracts.SendMessageRequest) (sqscontracts.SendMessageResult, error)
}

func New() *Service {
	return &Service{
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityPartial,
			nil,
			contracts.ActionNames(),
			"sns",
		),
		observability: newSNSObservability(),
	}
}

func NewWithPersistence(config StorageConfig) (*Service, error) {
	statePath, err := infrastructure.ResolveStatePath(config.BaseDir, config.InstanceID)
	if err != nil {
		return nil, err
	}

	store, err := infrastructure.NewSQLiteStore(statePath)
	if err != nil {
		return nil, err
	}

	svc := New()
	svc.store = store
	return svc, nil
}

func (s *Service) Start(context.Context) error {
	return nil
}

func (s *Service) Stop(context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}
	if err := s.store.Close(); err != nil {
		return fmt.Errorf("sns: close repository: %w", err)
	}
	s.store = nil
	return nil
}

func (s *Service) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{
		Name:        "sns",
		Description: "MildStack SNS real service",
		Version:     "v1",
		Tags:        []string{"aws", "messaging", "pubsub", "real-service"},
	}
}

func (s *Service) Policy() orchestrator.EmulationPolicy {
	if s == nil {
		return orchestrator.EmulationPolicy{}
	}
	return s.policy.Clone()
}

func (s *Service) RegisterRoutes(reg orchestrator.RouteRegistrar) error {
	if reg == nil {
		return nil
	}
	if err := reg.Register(orchestrator.Route{Method: "GET", Path: "/sns", Name: "sns:status"}); err != nil {
		return err
	}
	return reg.Register(orchestrator.Route{Method: "POST", Path: "/sns", Name: "sns:noop"})
}

func (s *Service) AttachState(hook orchestrator.StateHook) error {
	if hook == nil {
		return nil
	}
	s.stateHook = hook
	hook.Set(domain.StateKey, map[string]any{
		"service":       "sns",
		"topics":        []any{},
		"observability": s.observability.snapshot(),
	})
	return nil
}

func ResolveStoragePath(config StorageConfig) (string, error) {
	return infrastructure.ResolveStatePath(strings.TrimSpace(config.BaseDir), config.InstanceID)
}

func (s *Service) SetSQSBridge(bridge snsSQSBridge) {
	if s == nil {
		return
	}
	s.sqsBridge = bridge
}
