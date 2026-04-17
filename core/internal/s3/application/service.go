package application

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
	"github.com/michasdev/mildstack/core/internal/s3/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state  domain.State
	policy orchestrator.EmulationPolicy
}

func New() *Service {
	return &Service{
		state: domain.NewState(),
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list buckets",
				"read objects",
			},
			[]string{
				"bucket versioning",
				"object locking",
			},
			"s3",
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
		Name:        "s3",
		Description: "MildStack S3 exemplar service",
		Version:     "v1",
		Tags:        []string{"aws", "storage", "exemplar"},
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
		return fmt.Errorf("s3: nil state hook")
	}

	hook.Set(domain.StateKey, s.state.Snapshot())
	return nil
}
