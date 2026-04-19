package application

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
)

func (s *Service) GetBucketPolicy(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketPolicy(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: NoSuchBucketPolicy: The bucket policy does not exist")
	}
	return body, nil
}

func (s *Service) PutBucketPolicy(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketPolicy(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteBucketPolicy(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketPolicy(bucket) {
		return s.persist()
	}
	return nil
}

func (s *Service) GetBucketEncryption(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketEncryptionConfig(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: ServerSideEncryptionConfigurationNotFoundError: The server side encryption configuration was not found")
	}
	return body, nil
}

func (s *Service) PutBucketEncryption(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketEncryptionConfig(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteBucketEncryption(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketEncryptionConfig(bucket) {
		return s.persist()
	}
	return nil
}

func (s *Service) GetBucketLifecycle(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketLifecycleConfig(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: NoSuchLifecycleConfiguration: The lifecycle configuration does not exist")
	}
	return body, nil
}

func (s *Service) PutBucketLifecycle(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketLifecycleConfig(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteBucketLifecycle(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketLifecycleConfig(bucket) {
		return s.persist()
	}
	return nil
}

func (s *Service) GetBucketCORS(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketCORSConfig(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: NoSuchCORSConfiguration: The CORS configuration does not exist")
	}
	return body, nil
}

func (s *Service) PutBucketCORS(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketCORSConfig(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteBucketCORS(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketCORSConfig(bucket) {
		return s.persist()
	}
	return nil
}

func (s *Service) GetBucketACL(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketACLConfig(bucket)
	if ok {
		return body, nil
	}
	return defaultBucketACLBody(bucket), nil
}

func (s *Service) PutBucketACL(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return defaultBucketACLBody(bucket), nil
	}

	stored := s.state.SetBucketACLConfig(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) GetBucketTagging(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketTaggingConfig(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: NoSuchTagSet: The TagSet does not exist")
	}
	return body, nil
}

func (s *Service) PutBucketTagging(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketTaggingConfig(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteBucketTagging(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketTaggingConfig(bucket) {
		return s.persist()
	}
	return nil
}

func defaultBucketACLBody(bucket string) []byte {
	_ = bucket
	aws := awscontext.Default()
	return []byte(strings.TrimSpace(`<?xml version="1.0" encoding="UTF-8"?>
<AccessControlPolicy xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner>
    <ID>` + aws.AccountID + `</ID>
    <DisplayName>mildstack</DisplayName>
  </Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser">
        <ID>` + aws.AccountID + `</ID>
        <DisplayName>mildstack</DisplayName>
      </Grantee>
      <Permission>FULL_CONTROL</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`))
}
