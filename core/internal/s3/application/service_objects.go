package application

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

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

	if !s.state.HasBucket(bucket) {
		return domain.Object{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}

	object, ok := s.state.Object(bucket, key)
	if !ok {
		return domain.Object{}, fmt.Errorf("s3: NoSuchKey: The specified key does not exist")
	}
	return object, nil
}

func (s *Service) HeadObject(bucket, key string) (domain.Object, error) {
	object, err := s.GetObject(bucket, key)
	if err != nil {
		return domain.Object{}, err
	}
	object.Body = nil
	return object, nil
}

func (s *Service) PutObject(bucket, key string, body []byte, contentType string) (domain.Object, error) {
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
		return domain.Object{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	object := s.state.UpsertObject(domain.Object{
		Bucket:      bucket,
		Key:         key,
		Body:        append([]byte(nil), body...),
		Size:        int64(len(body)),
		ContentType: contentType,
	})
	if err := s.persist(); err != nil {
		return domain.Object{}, err
	}
	return object, nil
}

func (s *Service) CopyObject(bucket, key, sourceBucket, sourceKey string) (domain.Object, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	sourceBucket = strings.TrimSpace(sourceBucket)
	sourceKey = strings.TrimSpace(sourceKey)
	if bucket == "" {
		return domain.Object{}, fmt.Errorf("s3: bucket name is required")
	}
	if key == "" {
		return domain.Object{}, fmt.Errorf("s3: object key is required")
	}
	if sourceBucket == "" {
		return domain.Object{}, fmt.Errorf("s3: source bucket name is required")
	}
	if sourceKey == "" {
		return domain.Object{}, fmt.Errorf("s3: source object key is required")
	}
	if !s.state.HasBucket(bucket) {
		return domain.Object{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}

	source, err := s.GetObject(sourceBucket, sourceKey)
	if err != nil {
		return domain.Object{}, err
	}

	object := s.state.UpsertObject(domain.Object{
		Bucket:           bucket,
		Key:              key,
		Body:             append([]byte(nil), source.Body...),
		Size:             source.Size,
		ContentType:      source.ContentType,
		ETag:             source.ETag,
		Metadata:         source.Metadata,
		PreservedHeaders: source.PreservedHeaders,
	})
	if err := s.persist(); err != nil {
		return domain.Object{}, err
	}
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
	if !s.state.HasBucket(bucket) {
		return fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}
	s.state.DeleteObject(bucket, key)
	return s.persist()
}
