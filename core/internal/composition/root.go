package composition

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

type Root struct {
	Services []orchestrator.Service
}

func Assemble(services []orchestrator.Service) Root {
	copied := make([]orchestrator.Service, len(services))
	copy(copied, services)
	return Root{Services: copied}
}

