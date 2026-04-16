package composition

import (
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/dynamodb"
	"github.com/michasdev/mildstack/core/internal/s3"
)

func DefaultRoot() Root {
	return defaultRootWithHook(runtime.NewStateHook())
}

func defaultRootWithHook(hook orchestrator.StateHook) Root {
	services := []orchestrator.Service{
		s3.New(),
		dynamodb.New(),
	}

	for _, service := range services {
		if err := service.AttachState(hook); err != nil {
			panic(fmt.Sprintf("composition: attach %s state: %v", service.Metadata().Name, err))
		}
	}

	return Assemble(services)
}
