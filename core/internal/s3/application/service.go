package application

import (
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state            domain.State
	policy           orchestrator.EmulationPolicy
	repo             Repository
	multipartUploads map[string]domain.MultipartUpload
}

const defaultRegion = "us-east-1"

func New() *Service {
	return newService(domain.NewState(), nil)
}

func newService(state domain.State, repo Repository) *Service {
	return &Service{
		state:            state,
		repo:             repo,
		multipartUploads: make(map[string]domain.MultipartUpload),
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list buckets",
				"create bucket",
				"head bucket",
				"delete bucket",
				"bucket policy",
				"bucket encryption",
				"bucket lifecycle",
				"bucket CORS",
				"bucket ACL",
				"bucket tagging",
				"list objects v1",
				"list objects v2",
				"get object",
				"head object",
				"put object",
				"copy object",
				"delete object",
				"delete objects",
				"bucket versioning",
				"multipart upload",
			},
			[]string{
				"object locking",
			},
			"s3",
		),
	}
}
