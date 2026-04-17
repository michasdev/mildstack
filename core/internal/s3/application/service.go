package application

import (
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state  domain.State
	policy orchestrator.EmulationPolicy
	repo   Repository
}

const defaultRegion = "us-east-1"

func New() *Service {
	return newService(domain.NewState(), nil)
}

func newService(state domain.State, repo Repository) *Service {
	return &Service{
		state: state,
		repo:  repo,
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list buckets",
				"create bucket",
				"head bucket",
				"delete bucket",
				"list objects",
				"get object",
				"head object",
				"put object",
				"copy object",
				"delete object",
			},
			[]string{
				"bucket versioning",
				"object locking",
			},
			"s3",
		),
	}
}
