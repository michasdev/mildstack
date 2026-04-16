package composition

import (
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/s3"
)

func DefaultRoot() Root {
	service := s3.New()
	hook := runtime.NewStateHook()
	if err := service.AttachState(hook); err != nil {
		panic(fmt.Sprintf("composition: attach s3 state: %v", err))
	}

	return Assemble([]orchestrator.Service{service})
}
