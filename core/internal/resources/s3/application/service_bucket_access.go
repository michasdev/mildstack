package application

import (
	"encoding/xml"
	"fmt"
)

const s3BucketAccessNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"

func (s *Service) GetBucketOwnershipControls(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketOwnershipControls(bucket)
	if ok {
		return body, nil
	}
	return defaultBucketOwnershipControlsBody(), nil
}

func (s *Service) PutBucketOwnershipControls(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketOwnershipControls(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeleteBucketOwnershipControls(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketOwnershipControls(bucket) {
		return s.persist()
	}
	return nil
}

func (s *Service) GetPublicAccessBlock(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketPublicAccessBlock(bucket)
	if ok {
		return body, nil
	}
	return defaultPublicAccessBlockBody(), nil
}

func (s *Service) PutPublicAccessBlock(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketPublicAccessBlock(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) DeletePublicAccessBlock(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketPublicAccessBlock(bucket) {
		return s.persist()
	}
	return nil
}

func defaultBucketOwnershipControlsBody() []byte {
	return []byte(xml.Header + `<OwnershipControls xmlns="` + s3BucketAccessNamespace + `"><Rule><ObjectOwnership>BucketOwnerEnforced</ObjectOwnership></Rule></OwnershipControls>`)
}

func defaultPublicAccessBlockBody() []byte {
	return []byte(xml.Header + `<PublicAccessBlockConfiguration xmlns="` + s3BucketAccessNamespace + `"><BlockPublicAcls>true</BlockPublicAcls><IgnorePublicAcls>true</IgnorePublicAcls><BlockPublicPolicy>true</BlockPublicPolicy><RestrictPublicBuckets>true</RestrictPublicBuckets></PublicAccessBlockConfiguration>`)
}

func defaultAccessControlPolicyBody() []byte {
	return []byte(xml.Header + `<AccessControlPolicy xmlns="` + s3BucketAccessNamespace + `"><Owner><ID>owner-id</ID><DisplayName>mildstack</DisplayName></Owner><AccessControlList><Grant><Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser"><ID>owner-id</ID><DisplayName>mildstack</DisplayName></Grantee><Permission>FULL_CONTROL</Permission></Grant></AccessControlList></AccessControlPolicy>`)
}

func defaultObjectTaggingBody() []byte {
	return []byte(xml.Header + `<Tagging xmlns="` + s3BucketAccessNamespace + `"><TagSet></TagSet></Tagging>`)
}

func noSuchObjectKeyError() error {
	return fmt.Errorf("s3: NoSuchKey: The specified key does not exist")
}
