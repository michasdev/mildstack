package application

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/s3/domain"
	"github.com/michasdev/mildstack/core/internal/resources/s3/infrastructure"
)

func (s *Service) Start(context.Context) error {
	return nil
}

func (s *Service) Stop(context.Context) error {
	return nil
}

func (s *Service) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{
		Name:        "s3",
		Description: "MildStack S3 real service",
		Version:     "v1",
		Tags:        []string{"aws", "storage", "real-service"},
	}
}

func (s *Service) Policy() orchestrator.EmulationPolicy {
	return s.policy.Clone()
}

func (s *Service) RegisterRoutes(registrar orchestrator.RouteRegistrar) error {
	for _, route := range infrastructure.Routes() {
		route.Path = "/s3" + route.Path
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
