package application

import (
	"sync"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state            domain.State
	policy           orchestrator.EmulationPolicy
	repo             Repository
	payloads         PayloadStore
	multipartUploads map[string]domain.MultipartUpload
	mu               sync.Mutex
}

const defaultRegion = "us-east-1"

func New() *Service {
	return newService(domain.NewState(), nil)
}

func newService(state domain.State, repo Repository) *Service {
	service := &Service{
		state:            state,
		repo:             repo,
		multipartUploads: make(map[string]domain.MultipartUpload),
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list buckets",
				"create bucket",
				"head bucket",
				"get bucket location",
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
				"list multipart uploads",
				"list parts",
				"bucket notification",
				"bucket logging",
				"bucket replication",
				"object locking",
				"bucket ownership controls",
				"public access block",
				"object acl",
				"object tagging",
			},
			[]string{
				"directory buckets / S3 Express",
				"reporting and admin surfaces",
				"lower-value management features",
				"specialized data-plane / Object Lambda actions",
			},
			"s3",
		),
	}
	if payloads, ok := repo.(PayloadStore); ok {
		service.payloads = payloads
	} else {
		service.payloads = newMemoryPayloadStore()
	}
	return service
}
