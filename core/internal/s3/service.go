package s3

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"
import "github.com/michasdev/mildstack/core/internal/s3/application"

func New() orchestrator.Service {
	return application.New()
}
