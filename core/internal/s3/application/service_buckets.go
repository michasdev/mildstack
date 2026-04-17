package application

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

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

	bucket := s.state.UpsertBucket(name, region)
	if err := s.persist(); err != nil {
		return domain.Bucket{}, err
	}
	return bucket, nil
}
