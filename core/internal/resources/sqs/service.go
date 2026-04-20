package sqs

import (
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/application"
)

type StorageConfig = application.StorageConfig

func New() orchestrator.Service {
	return application.New()
}

func NewWithStorage(config StorageConfig) (orchestrator.Service, error) {
	return application.NewWithPersistence(config)
}
