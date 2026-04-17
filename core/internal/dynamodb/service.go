package dynamodb

import (
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/dynamodb/application"
)

func New() orchestrator.Service {
	return application.New()
}
