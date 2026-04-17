package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
	"github.com/michasdev/mildstack/core/internal/s3/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state  domain.State
	policy orchestrator.EmulationPolicy
}

const defaultRegion = "us-east-1"

func New() *Service {
	return &Service{
		state: domain.NewState(),
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			[]string{
				"list buckets",
				"create bucket",
				"list objects",
				"get object",
				"put object",
				"delete object",
			},
			[]string{
				"bucket versioning",
				"object locking",
			},
			"s3",
		),
	}
}

func (s *Service) Start(context.Context) error {
	return nil
}

func (s *Service) Stop(context.Context) error {
	return nil
}

func (s *Service) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{
		Name:        "s3",
		Description: "MildStack S3 real service",
		Version:     "v1",
		Tags:        []string{"aws", "storage", "real-service"},
	}
}

func (s *Service) Policy() orchestrator.EmulationPolicy {
	return s.policy.Clone()
}

func (s *Service) RegisterRoutes(registrar orchestrator.RouteRegistrar) error {
	for _, route := range infrastructure.Routes() {
		if err := registrar.Register(route); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) AttachState(hook orchestrator.StateHook) error {
	if hook == nil {
		return fmt.Errorf("s3: nil state hook")
	}

	hook.Set(domain.StateKey, s.state.Snapshot())
	return nil
}

func (s *Service) ListBuckets() []domain.Bucket {
	return s.state.ListBuckets()
}

func (s *Service) CreateBucket(name, region string) (domain.Bucket, error) {
	name = strings.TrimSpace(name)
	region = strings.TrimSpace(region)
	if name == "" {
		return domain.Bucket{}, fmt.Errorf("s3: bucket name is required")
	}
	if region == "" {
		region = defaultRegion
	}

	return s.state.UpsertBucket(name, region), nil
}

func (s *Service) ListObjects(bucket string) ([]domain.Object, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, fmt.Errorf("s3: bucket name is required")
	}
	if !s.state.HasBucket(bucket) {
		return nil, fmt.Errorf("s3: bucket %q not found", bucket)
	}

	return s.state.ListObjects(bucket), nil
}

func (s *Service) GetObject(bucket, key string) (domain.Object, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" {
		return domain.Object{}, fmt.Errorf("s3: bucket name is required")
	}
	if key == "" {
		return domain.Object{}, fmt.Errorf("s3: object key is required")
	}

	object, ok := s.state.Object(bucket, key)
	if !ok {
		return domain.Object{}, fmt.Errorf("s3: object %s/%s not found", bucket, key)
	}
	return object, nil
}

func (s *Service) PutObject(bucket, key string, size int64, contentType string) (domain.Object, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	contentType = strings.TrimSpace(contentType)
	if bucket == "" {
		return domain.Object{}, fmt.Errorf("s3: bucket name is required")
	}
	if key == "" {
		return domain.Object{}, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasBucket(bucket) {
		return domain.Object{}, fmt.Errorf("s3: bucket %q not found", bucket)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	object := s.state.UpsertObject(domain.Object{
		Bucket:      bucket,
		Key:         key,
		Size:        size,
		ContentType: contentType,
	})
	return object, nil
}

func (s *Service) DeleteObject(bucket, key string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" {
		return fmt.Errorf("s3: bucket name is required")
	}
	if key == "" {
		return fmt.Errorf("s3: object key is required")
	}
	if !s.state.DeleteObject(bucket, key) {
		return fmt.Errorf("s3: object %s/%s not found", bucket, key)
	}
	return nil
}
