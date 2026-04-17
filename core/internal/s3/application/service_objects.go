package application

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

type ListObjectsV1Request = domain.ListObjectsV1Request
type ListObjectsV1Result = domain.ListObjectsV1Result
type ListObjectsV2Request = domain.ListObjectsV2Request
type ListObjectsV2Result = domain.ListObjectsV2Result
type DeleteObjectsRequest = domain.DeleteObjectsRequest
type DeletedObject = domain.DeletedObject
type DeleteObjectsError = domain.DeleteObjectsError
type DeleteObjectsResult = domain.DeleteObjectsResult

func (s *Service) ListObjects(bucket string) ([]domain.Object, error) {
	result, err := s.ListObjectsV1(ListObjectsV1Request{Bucket: bucket})
	if err != nil {
		return nil, err
	}
	return result.Objects, nil
}

func (s *Service) ListObjectsV1(request ListObjectsV1Request) (ListObjectsV1Result, error) {
	bucket, err := s.requireBucket(request.Bucket)
	if err != nil {
		return ListObjectsV1Result{}, err
	}

	page := s.state.ListObjectPage(bucket, domain.ListObjectsOptions{
		Prefix:     strings.TrimSpace(request.Prefix),
		Delimiter:  strings.TrimSpace(request.Delimiter),
		MaxKeys:    request.MaxKeys,
		StartAfter: strings.TrimSpace(request.Marker),
	})

	result := ListObjectsV1Result{
		Bucket:         bucket,
		Prefix:         strings.TrimSpace(request.Prefix),
		Marker:         strings.TrimSpace(request.Marker),
		Delimiter:      strings.TrimSpace(request.Delimiter),
		MaxKeys:        normalizedMaxKeys(request.MaxKeys),
		IsTruncated:    page.IsTruncated,
		Objects:        cloneObjects(page.Objects),
		CommonPrefixes: append([]string(nil), page.CommonPrefixes...),
	}
	if result.IsTruncated && result.Delimiter != "" && page.NextMarker != "" {
		result.NextMarker = page.NextMarker
	}
	return result, nil
}

func (s *Service) ListObjectsV2(request ListObjectsV2Request) (ListObjectsV2Result, error) {
	bucket, err := s.requireBucket(request.Bucket)
	if err != nil {
		return ListObjectsV2Result{}, err
	}

	startAfter := strings.TrimSpace(request.StartAfter)
	continuationToken := strings.TrimSpace(request.ContinuationToken)
	if continuationToken != "" {
		decoded, decodeErr := base64.StdEncoding.DecodeString(continuationToken)
		if decodeErr == nil {
			startAfter = string(decoded)
		} else {
			startAfter = continuationToken
		}
	}

	page := s.state.ListObjectPage(bucket, domain.ListObjectsOptions{
		Prefix:     strings.TrimSpace(request.Prefix),
		Delimiter:  strings.TrimSpace(request.Delimiter),
		MaxKeys:    request.MaxKeys,
		StartAfter: startAfter,
	})

	result := ListObjectsV2Result{
		Bucket:            bucket,
		Prefix:            strings.TrimSpace(request.Prefix),
		Delimiter:         strings.TrimSpace(request.Delimiter),
		ContinuationToken: continuationToken,
		StartAfter:        strings.TrimSpace(request.StartAfter),
		MaxKeys:           normalizedMaxKeys(request.MaxKeys),
		KeyCount:          len(page.Objects) + len(page.CommonPrefixes),
		IsTruncated:       page.IsTruncated,
		Objects:           cloneObjects(page.Objects),
		CommonPrefixes:    append([]string(nil), page.CommonPrefixes...),
	}
	if result.IsTruncated && page.NextMarker != "" {
		result.NextContinuationToken = base64.StdEncoding.EncodeToString([]byte(page.NextMarker))
	}
	return result, nil
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

func (s *Service) DeleteObjects(request DeleteObjectsRequest) (DeleteObjectsResult, error) {
	bucket, err := s.requireBucket(request.Bucket)
	if err != nil {
		return DeleteObjectsResult{}, err
	}

	result := DeleteObjectsResult{
		Deleted: make([]DeletedObject, 0, len(request.Keys)),
		Errors:  make([]DeleteObjectsError, 0),
	}
	for _, key := range request.Keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		s.state.DeleteObject(bucket, trimmed)
		if !request.Quiet {
			result.Deleted = append(result.Deleted, DeletedObject{Key: trimmed})
		}
	}

	if err := s.persist(); err != nil {
		return DeleteObjectsResult{}, err
	}
	return result, nil
}

func (s *Service) requireBucket(bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("s3: bucket name is required")
	}
	if !s.state.HasBucket(bucket) {
		return "", fmt.Errorf("s3: bucket %q not found", bucket)
	}
	return bucket, nil
}

func normalizedMaxKeys(maxKeys int) int {
	if maxKeys <= 0 {
		return 1000
	}
	return maxKeys
}

func cloneObjects(objects []domain.Object) []domain.Object {
	cloned := make([]domain.Object, len(objects))
	for i := range objects {
		cloned[i] = objects[i]
		cloned[i].Body = append([]byte(nil), objects[i].Body...)
		cloned[i].Metadata = cloneObjectStringMap(objects[i].Metadata)
		cloned[i].PreservedHeaders = cloneObjectStringMap(objects[i].PreservedHeaders)
	}
	return cloned
}

func cloneObjectStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
