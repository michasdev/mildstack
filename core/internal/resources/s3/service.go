package s3

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"
import "github.com/michasdev/mildstack/core/internal/resources/s3/application"

type StorageConfig = application.StorageConfig

func New() orchestrator.Service {
	return application.New()
}

func NewWithStorage(config StorageConfig) (orchestrator.Service, error) {
	return application.NewWithPersistence(config)
}
