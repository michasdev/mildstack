package application

import (
	"context"
	"fmt"
	"sync"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	policy orchestrator.EmulationPolicy
	mu     sync.Mutex
}

func New() *Service {
	return &Service{
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

	hook.Set("sqs", map[string]any{
		"service":  "sqs",
		"metadata": s.Metadata(),
		"policy":   s.Policy(),
		"routes":   infrastructure.Routes(),
		"catalog":  contracts.Catalog(),
		"phase":    34,
	})
	return nil
}
