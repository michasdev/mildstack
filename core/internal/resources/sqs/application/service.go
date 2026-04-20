package application

import (
	"context"
	"fmt"
	"sync"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state     domain.State
	policy    orchestrator.EmulationPolicy
	repo      Repository
	stateHook orchestrator.StateHook
	mu        sync.Mutex
}

func New() *Service {
	return newService(domain.NewState(), nil)
}

func newService(state domain.State, repo Repository) *Service {
	return &Service{
		state: state.Clone(),
		repo:  repo,
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			contracts.ActionNames(),
			nil,
			"sqs",
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
		return fmt.Errorf("sqs: close repository: %w", err)
	}
	s.repo = nil
	return nil
}

func (s *Service) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{
		Name:        "sqs",
		Description: "MildStack SQS real service",
		Version:     "v1",
		Tags:        []string{"aws", "messaging", "queue", "real-service"},
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
		return fmt.Errorf("sqs: nil state hook")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateHook = hook
	s.publishSnapshotLocked()
	return nil
}

func (s *Service) publishSnapshotLocked() {
	if s.stateHook == nil {
		return
	}

	s.stateHook.Set(domain.StateKey, s.state.Snapshot())
}
