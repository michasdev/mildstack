package application

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

func (s *Service) GetBucketVersioning(bucket string) (domain.BucketVersioning, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return domain.BucketVersioning{}, fmt.Errorf("s3: bucket name is required")
	}
	if !s.state.HasBucket(bucket) {
		return domain.BucketVersioning{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}

	return domain.BucketVersioning{
		Bucket: bucket,
		Status: s.state.BucketVersioningStatus(bucket),
	}, nil
}

func (s *Service) PutBucketVersioning(bucket, status string) (domain.BucketVersioning, error) {
	bucket = strings.TrimSpace(bucket)
	status = strings.TrimSpace(status)
	if bucket == "" {
		return domain.BucketVersioning{}, fmt.Errorf("s3: bucket name is required")
	}
	if !s.state.HasBucket(bucket) {
		return domain.BucketVersioning{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}
	switch status {
	case domain.VersioningEnabled, domain.VersioningSuspended:
	default:
		return domain.BucketVersioning{}, fmt.Errorf("s3: invalid bucket versioning status %q", status)
	}

	setting := s.state.SetBucketVersioning(bucket, status)
	if err := s.persist(); err != nil {
		return domain.BucketVersioning{}, err
	}
	return setting, nil
}

func (s *Service) ListObjectVersions(bucket string) (domain.ListObjectVersionsResult, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return domain.ListObjectVersionsResult{}, fmt.Errorf("s3: bucket name is required")
	}
	if !s.state.HasBucket(bucket) {
		return domain.ListObjectVersionsResult{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}

	return domain.ListObjectVersionsResult{
		Bucket:   bucket,
		Versions: s.state.ListObjectVersions(bucket),
	}, nil
}

func (s *Service) storeObject(object domain.Object) (domain.Object, error) {
	stored := s.state.UpsertObject(object)
	if s.state.VersioningEnabled(stored.Bucket) {
		s.state.RecordObjectVersion(stored)
	}
	return stored, nil
}

func (s *Service) removeObject(bucket, key string) error {
	if s.state.HasObject(bucket, key) {
		if err := s.objectMutationBlocked(bucket, key); err != nil {
			return err
		}
	}
	if s.state.VersioningEnabled(bucket) {
		s.state.RecordDeleteMarker(bucket, key)
	}
	s.state.DeleteObject(bucket, key)
	return s.persist()
}
