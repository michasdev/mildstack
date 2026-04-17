package application

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

var bucketNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

func (s *Service) ListBuckets() []domain.Bucket {
	return s.state.ListBuckets()
}

func (s *Service) CreateBucket(name, region string) (domain.Bucket, error) {
	name = strings.TrimSpace(name)
	region = strings.TrimSpace(region)
	if name == "" {
		return domain.Bucket{}, fmt.Errorf("s3: bucket name is required")
	}
	if !isValidBucketName(name) {
		return domain.Bucket{}, fmt.Errorf("s3: InvalidBucketName: The specified bucket is not valid")
	}
	if region == "" {
		region = defaultRegion
	}

	if bucket, ok := s.state.Bucket(name); ok {
		if bucket.Region == "" {
			bucket.Region = defaultRegion
		}
		return bucket, nil
	}

	bucket := s.state.UpsertBucket(domain.Bucket{
		Name:   name,
		Region: region,
	})
	if err := s.persist(); err != nil {
		return domain.Bucket{}, err
	}
	return bucket, nil
}

func (s *Service) HeadBucket(name string) (domain.Bucket, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Bucket{}, fmt.Errorf("s3: bucket name is required")
	}

	bucket, ok := s.state.Bucket(name)
	if !ok {
		return domain.Bucket{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", name)
	}
	if bucket.Region == "" {
		bucket.Region = defaultRegion
	}
	return bucket, nil
}

func (s *Service) DeleteBucket(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("s3: bucket name is required")
	}
	if !s.state.HasBucket(name) {
		return fmt.Errorf("s3: NoSuchBucket: bucket %q not found", name)
	}
	if !s.state.DeleteBucket(name) {
		return fmt.Errorf("s3: BucketNotEmpty: The bucket you tried to delete is not empty")
	}
	s.removeMultipartUploads(name)
	return s.persist()
}

func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	if !bucketNamePattern.MatchString(name) {
		return false
	}
	if strings.Contains(name, "..") || strings.Contains(name, ".-") || strings.Contains(name, "-.") {
		return false
	}

	hasLetter := false
	for _, r := range name {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return false
	}

	parts := strings.Split(name, ".")
	if len(parts) == 4 {
		allNumeric := true
		for _, part := range parts {
			if part == "" {
				return false
			}
			for _, r := range part {
				if r < '0' || r > '9' {
					allNumeric = false
					break
				}
			}
			if !allNumeric {
				break
			}
		}
		if allNumeric {
			return false
		}
	}

	return true
}
