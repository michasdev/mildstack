package composition

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb"
	"github.com/michasdev/mildstack/core/internal/resources/s3"
)

type DefaultRootConfig struct {
	InstanceID       string
	S3StorageBaseDir string
}

func DefaultRoot(instanceID string) Root {
	return defaultRootWithHook(runtime.NewStateHook(), DefaultRootConfig{InstanceID: instanceID})
}

func defaultRootWithHook(hook orchestrator.StateHook, config DefaultRootConfig) Root {
	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		panic("composition: s3 instance id is required")
	}

	s3Service, err := s3.NewWithStorage(s3.StorageConfig{
		BaseDir:    config.S3StorageBaseDir,
		InstanceID: instanceID,
	})
	if err != nil {
		panic(fmt.Sprintf("composition: init s3 service: %v", err))
	}

	services := []orchestrator.Service{
		s3Service,
		dynamodb.New(),
	}

	for _, service := range services {
		if err := service.AttachState(hook); err != nil {
			panic(fmt.Sprintf("composition: attach %s state: %v", service.Metadata().Name, err))
		}
	}

	return Assemble(services)
}
