package composition

import (
	"context"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb"
	"github.com/michasdev/mildstack/core/internal/resources/s3"
	"github.com/michasdev/mildstack/core/internal/resources/sqs"
)

type DefaultRootConfig struct {
	InstanceID             string
	S3StorageBaseDir       string
	DynamoDBStorageBaseDir string
}

func DefaultRoot(instanceID string) Root {
	return defaultRootWithHook(runtime.NewStateHook(), DefaultRootConfig{InstanceID: instanceID})
}

func defaultRootWithHook(hook orchestrator.StateHook, config DefaultRootConfig) Root {
	instanceID := strings.TrimSpace(config.InstanceID)
	// When no instance ID is provided, return a root with no services.
	// This allows read-only CLI commands (instances, status, stop, delete)
	// to run without a MILDSTACK_INSTANCE_ID env var. The serve command
	// validates the ID before starting a server.
	if instanceID == "" {
		return Assemble(nil)
	}

	s3Service, err := s3.NewWithStorage(s3.StorageConfig{
		BaseDir:    config.S3StorageBaseDir,
		InstanceID: instanceID,
	})
	if err != nil {
		panic(fmt.Sprintf("composition: init s3 service: %v", err))
	}

	dynamoService, err := dynamodb.NewWithStorage(dynamodb.StorageConfig{
		BaseDir:    config.DynamoDBStorageBaseDir,
		InstanceID: instanceID,
	})
	if err != nil {
		_ = s3Service.Stop(context.Background())
		panic(fmt.Sprintf("composition: init dynamodb service: %v", err))
	}

	sqsService := sqs.New()

	services := []orchestrator.Service{s3Service, dynamoService, sqsService}
	for _, service := range services {
		if err := service.AttachState(hook); err != nil {
			for _, candidate := range services {
				_ = candidate.Stop(context.Background())
			}
			panic(fmt.Sprintf("composition: attach %s state: %v", service.Metadata().Name, err))
		}
	}

	return Assemble(services)
}
