package dynamodb

import (
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/application"
)

func New() orchestrator.Service {
	return application.New()
}
