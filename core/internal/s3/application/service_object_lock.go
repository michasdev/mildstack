package application

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

const objectLockXMLNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"

type objectLockConfigurationXML struct {
	XMLName           xml.Name                  `xml:"ObjectLockConfiguration"`
	ObjectLockEnabled string                    `xml:"ObjectLockEnabled"`
	Rule              *objectLockRuleXML        `xml:"Rule,omitempty"`
}

type objectLockRuleXML struct {
	DefaultRetention *objectLockRetentionXML `xml:"DefaultRetention,omitempty"`
}

type objectLockRetentionXML struct {
	Mode  string `xml:"Mode"`
	Days  int    `xml:"Days,omitempty"`
	Years int    `xml:"Years,omitempty"`
}

type objectRetentionXML struct {
	XMLName         xml.Name `xml:"Retention"`
	Mode            string   `xml:"Mode"`
	RetainUntilDate string   `xml:"RetainUntilDate"`
}

type objectLegalHoldXML struct {
	XMLName xml.Name `xml:"LegalHold"`
	Status  string   `xml:"Status"`
}

type objectLockConfigurationEnvelope struct {
	XMLName           xml.Name               `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ObjectLockConfiguration"`
	ObjectLockEnabled string                 `xml:"ObjectLockEnabled"`
	Rule              *objectLockEnvelopeRule `xml:"Rule,omitempty"`
}

type objectLockEnvelopeRule struct {
	DefaultRetention *objectLockEnvelopeRetention `xml:"DefaultRetention,omitempty"`
}

type objectLockEnvelopeRetention struct {
	Mode  string `xml:"Mode"`
	Days  int    `xml:"Days,omitempty"`
	Years int    `xml:"Years,omitempty"`
}

type objectRetentionEnvelope struct {
	XMLName         xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ Retention"`
	Mode            string   `xml:"Mode"`
	RetainUntilDate string   `xml:"RetainUntilDate"`
}

type objectLegalHoldEnvelope struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ LegalHold"`
	Status  string   `xml:"Status"`
}

func (s *Service) GetObjectLockConfiguration(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	config, ok := s.state.BucketObjectLockConfig(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: ObjectLockConfigurationNotFoundError: Object Lock configuration does not exist for this bucket")
	}
	return marshalObjectLockConfiguration(config)
}

func (s *Service) PutObjectLockConfiguration(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	if s.state.BucketVersioningStatus(bucket) != domain.VersioningEnabled {
		return nil, fmt.Errorf("s3: InvalidBucketState: Versioning must be 'Enabled' on the bucket to apply a Object Lock configuration")
	}

	config, err := parseObjectLockConfiguration(body)
	if err != nil {
		return nil, err
	}
	stored := s.state.SetBucketObjectLockConfig(bucket, config)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return marshalObjectLockConfiguration(stored)
}

func (s *Service) GetObjectRetention(bucket, key string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, fmt.Errorf("s3: NoSuchKey: The specified key does not exist")
	}
	if _, ok := s.state.BucketObjectLockConfig(bucket); !ok {
		return nil, fmt.Errorf("s3: InvalidRequest: Bucket is missing Object Lock Configuration")
	}

	retention, ok := s.state.ObjectRetentionConfig(bucket, key)
	if !ok {
		return nil, fmt.Errorf("s3: NoSuchObjectLockConfiguration: The specified object does not have a ObjectLock configuration")
	}
	return marshalObjectRetention(retention)
}

func (s *Service) PutObjectRetention(bucket, key string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, fmt.Errorf("s3: NoSuchKey: The specified key does not exist")
	}
	if _, ok := s.state.BucketObjectLockConfig(bucket); !ok {
		return nil, fmt.Errorf("s3: InvalidRequest: Bucket is missing Object Lock Configuration")
	}

	retention, err := parseObjectRetention(body)
	if err != nil {
		return nil, err
	}
	if existing, ok := s.state.ObjectRetentionConfig(bucket, key); ok && objectRetentionReduction(existing, retention) {
		return nil, fmt.Errorf("s3: AccessDenied: Access Denied because object protected by object lock.")
	}

	stored := s.state.SetObjectRetention(bucket, key, retention)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return marshalObjectRetention(stored)
}

func (s *Service) GetObjectLegalHold(bucket, key string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, fmt.Errorf("s3: NoSuchKey: The specified key does not exist")
	}
	if _, ok := s.state.BucketObjectLockConfig(bucket); !ok {
		return nil, fmt.Errorf("s3: InvalidRequest: Bucket is missing Object Lock Configuration")
	}

	hold, ok := s.state.ObjectLegalHoldConfig(bucket, key)
	if !ok {
		return nil, fmt.Errorf("s3: NoSuchObjectLockConfiguration: The specified object does not have a ObjectLock configuration")
	}
	return marshalObjectLegalHold(hold)
}

func (s *Service) PutObjectLegalHold(bucket, key string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasObject(bucket, key) {
		return nil, fmt.Errorf("s3: NoSuchKey: The specified key does not exist")
	}
	if _, ok := s.state.BucketObjectLockConfig(bucket); !ok {
		return nil, fmt.Errorf("s3: InvalidRequest: Bucket is missing Object Lock Configuration")
	}

	hold, err := parseObjectLegalHold(body)
	if err != nil {
		return nil, err
	}
	stored := s.state.SetObjectLegalHold(bucket, key, hold)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return marshalObjectLegalHold(stored)
}

func (s *Service) objectMutationBlocked(bucket, key string) error {
	if hold, ok := s.state.ObjectLegalHoldConfig(bucket, key); ok && strings.EqualFold(hold.Status, "ON") {
		return fmt.Errorf("s3: AccessDenied: Access Denied because object protected by object lock.")
	}

	retention, ok := s.state.ObjectRetentionConfig(bucket, key)
	if !ok {
		return nil
	}
	if retention.RetainUntilDate.IsZero() || !retention.RetainUntilDate.After(time.Now().UTC()) {
		return nil
	}
	if retention.Mode == "COMPLIANCE" || retention.Mode == "GOVERNANCE" {
		return fmt.Errorf("s3: AccessDenied: Access Denied because object protected by object lock.")
	}
	return nil
}

func (s *Service) clearObjectProtection(bucket, key string) {
	s.state.DeleteObjectRetention(bucket, key)
	s.state.DeleteObjectLegalHold(bucket, key)
}

func (s *Service) copyObjectProtection(destBucket, destKey, sourceBucket, sourceKey string) {
	if retention, ok := s.state.ObjectRetentionConfig(sourceBucket, sourceKey); ok {
		s.state.SetObjectRetention(destBucket, destKey, retention)
	} else {
		s.state.DeleteObjectRetention(destBucket, destKey)
	}
	if hold, ok := s.state.ObjectLegalHoldConfig(sourceBucket, sourceKey); ok {
		s.state.SetObjectLegalHold(destBucket, destKey, hold)
	} else {
		s.state.DeleteObjectLegalHold(destBucket, destKey)
	}
}

func parseObjectLockConfiguration(body []byte) (domain.ObjectLockConfiguration, error) {
	var payload objectLockConfigurationXML
	if err := xml.Unmarshal(body, &payload); err != nil {
		return domain.ObjectLockConfiguration{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	if strings.TrimSpace(payload.ObjectLockEnabled) != "Enabled" {
		return domain.ObjectLockConfiguration{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}

	config := domain.ObjectLockConfiguration{Enabled: true}
	if payload.Rule != nil && payload.Rule.DefaultRetention != nil {
		retention := payload.Rule.DefaultRetention
		if retention.Mode != "GOVERNANCE" && retention.Mode != "COMPLIANCE" {
			return domain.ObjectLockConfiguration{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
		}
		hasDays := retention.Days > 0
		hasYears := retention.Years > 0
		if hasDays == hasYears {
			return domain.ObjectLockConfiguration{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
		}

		stored := &domain.ObjectLockRetention{Mode: retention.Mode}
		if hasDays {
			stored.Days = retention.Days
		}
		if hasYears {
			stored.Years = retention.Years
		}
		config.DefaultRetention = stored
	}
	return config, nil
}

func parseObjectRetention(body []byte) (domain.ObjectRetention, error) {
	var payload objectRetentionXML
	if err := xml.Unmarshal(body, &payload); err != nil {
		return domain.ObjectRetention{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	if payload.Mode != "GOVERNANCE" && payload.Mode != "COMPLIANCE" {
		return domain.ObjectRetention{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	retainUntil, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.RetainUntilDate))
	if err != nil {
		return domain.ObjectRetention{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	return domain.ObjectRetention{
		Mode:            payload.Mode,
		RetainUntilDate: retainUntil.UTC(),
	}, nil
}

func parseObjectLegalHold(body []byte) (domain.ObjectLegalHold, error) {
	var payload objectLegalHoldXML
	if err := xml.Unmarshal(body, &payload); err != nil {
		return domain.ObjectLegalHold{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	if payload.Status != "ON" && payload.Status != "OFF" {
		return domain.ObjectLegalHold{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	return domain.ObjectLegalHold{Status: payload.Status}, nil
}

func objectRetentionReduction(existing, next domain.ObjectRetention) bool {
	if next.RetainUntilDate.Before(existing.RetainUntilDate) {
		return true
	}
	return existing.Mode == "COMPLIANCE" && next.Mode == "GOVERNANCE"
}

func marshalObjectLockConfiguration(config domain.ObjectLockConfiguration) ([]byte, error) {
	payload := objectLockConfigurationEnvelope{
		ObjectLockEnabled: "Enabled",
	}
	if config.DefaultRetention != nil {
		retention := &objectLockEnvelopeRetention{
			Mode: config.DefaultRetention.Mode,
		}
		if config.DefaultRetention.Days > 0 {
			retention.Days = config.DefaultRetention.Days
		}
		if config.DefaultRetention.Years > 0 {
			retention.Years = config.DefaultRetention.Years
		}
		payload.Rule = &objectLockEnvelopeRule{DefaultRetention: retention}
	}
	return marshalXML(payload)
}

func marshalObjectRetention(retention domain.ObjectRetention) ([]byte, error) {
	return marshalXML(objectRetentionEnvelope{
		Mode:            retention.Mode,
		RetainUntilDate: retention.RetainUntilDate.UTC().Format(time.RFC3339),
	})
}

func marshalObjectLegalHold(hold domain.ObjectLegalHold) ([]byte, error) {
	return marshalXML(objectLegalHoldEnvelope{Status: hold.Status})
}

func marshalXML(value any) ([]byte, error) {
	output, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(output, '\n'), nil
}
