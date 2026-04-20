package sqs

import (
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/application"
)

func New() orchestrator.Service {
	return application.New()
}
