package application

import (
	"fmt"
	"strings"
)

func (s *Service) GetObjectAcl(bucket, key string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, noSuchObjectKeyError()
	}

	body, ok := s.state.ObjectACL(bucket, key)
	if ok {
		return body, nil
	}
	return defaultAccessControlPolicyBody(), nil
}

func (s *Service) PutObjectAcl(bucket, key string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, noSuchObjectKeyError()
	}

	stored := s.state.SetObjectACL(bucket, key, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) GetObjectTagging(bucket, key string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, noSuchObjectKeyError()
	}

	body, ok := s.state.ObjectTagging(bucket, key)
	if ok {
		return body, nil
	}
	return defaultObjectTaggingBody(), nil
}

func (s *Service) PutObjectTagging(bucket, key string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, noSuchObjectKeyError()
	}

	stored := s.state.SetObjectTagging(bucket, key, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteObjectTagging(bucket, key string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return noSuchObjectKeyError()
	}
	if s.state.DeleteObjectTagging(bucket, key) {
		return s.persist()
	}
	return nil
}
