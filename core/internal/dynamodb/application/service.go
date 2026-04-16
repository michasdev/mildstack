package application

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/dynamodb/domain"
	"github.com/michasdev/mildstack/core/internal/dynamodb/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state domain.State
}

func New() *Service {
	return &Service{
		state: domain.NewState(),
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
		Description: "MildStack DynamoDB exemplar service",
		Version:     "v1",
		Tags:        []string{"aws", "database", "nosql", "exemplar"},
	}
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
