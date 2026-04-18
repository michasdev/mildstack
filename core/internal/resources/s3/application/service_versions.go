package application

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/s3/domain"
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

	versions, err := s.hydrateVersions(s.state.ListObjectVersions(bucket))
	if err != nil {
		return domain.ListObjectVersionsResult{}, err
	}
	return domain.ListObjectVersionsResult{
		Bucket:   bucket,
		Versions: versions,
	}, nil
}

func (s *Service) storeObject(object domain.Object) (domain.Object, error) {
	stored := s.state.UpsertObject(object)
	stored.Body = nil
	if s.state.VersioningEnabled(stored.Bucket) {
		s.state.RecordObjectVersion(stored)
	}
	return stored, nil
}

func (s *Service) removeObject(bucket, key string) error {
	var payloadRef string
	if s.state.HasObject(bucket, key) {
		if err := s.objectMutationBlocked(bucket, key); err != nil {
			return err
		}
		if object, ok := s.state.Object(bucket, key); ok {
			payloadRef = object.PayloadRef
		}
	}
	if s.state.VersioningEnabled(bucket) {
		s.state.RecordDeleteMarker(bucket, key)
		payloadRef = ""
	}
	s.state.DeleteObject(bucket, key)
	if payloadRef != "" && s.payloads != nil {
		_ = s.payloads.DeletePayload(payloadRef)
	}
	return s.persist()
}
