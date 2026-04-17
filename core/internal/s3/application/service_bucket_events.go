package application

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

const s3XMLNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"

type replicationConfigXML struct {
	XMLName xml.Name             `xml:"ReplicationConfiguration"`
	Role    string               `xml:"Role"`
	Rules   []replicationRuleXML `xml:"Rule"`
}

type replicationRuleXML struct {
	ID          string                     `xml:"ID"`
	Status      string                     `xml:"Status"`
	Prefix      string                     `xml:"Prefix"`
	Destination *replicationDestinationXML `xml:"Destination"`
}

type replicationDestinationXML struct {
	Bucket       string `xml:"Bucket"`
	StorageClass string `xml:"StorageClass"`
}

func (s *Service) GetBucketNotification(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketNotification(bucket)
	if ok {
		return body, nil
	}
	return notificationDefaultBody(), nil
}

func (s *Service) PutBucketNotification(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketNotification(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) GetBucketLogging(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	body, ok := s.state.BucketLoggingConfig(bucket)
	if ok {
		return body, nil
	}
	return loggingDefaultBody(), nil
}

func (s *Service) PutBucketLogging(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	stored := s.state.SetBucketLoggingConfig(bucket, body)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *Service) GetBucketReplication(bucket string) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}

	config, ok := s.state.BucketReplicationConfig(bucket)
	if !ok {
		return nil, fmt.Errorf("s3: ReplicationConfigurationNotFoundError: The replication configuration was not found")
	}
	return renderReplicationConfig(config), nil
}

func (s *Service) PutBucketReplication(bucket string, body []byte) ([]byte, error) {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return nil, err
	}
	if s.state.BucketVersioningStatus(bucket) != domain.VersioningEnabled {
		return nil, fmt.Errorf("s3: InvalidRequest: Versioning must be 'Enabled' on the bucket to apply a replication configuration")
	}

	config, err := parseReplicationConfig(body)
	if err != nil {
		return nil, err
	}

	for i := range config.Rules {
		if config.Rules[i].ID == "" {
			config.Rules[i].ID = fmt.Sprintf("rule-%d", i+1)
		}
		if config.Rules[i].Status == "" {
			config.Rules[i].Status = "Enabled"
		}

		destBucket := strings.TrimSpace(config.Rules[i].Destination.Bucket)
		if destBucket == "" {
			continue
		}
		if s.state.HasBucket(destBucket) && s.state.BucketVersioningStatus(destBucket) != domain.VersioningEnabled {
			return nil, fmt.Errorf("s3: InvalidRequest: Destination bucket must have versioning enabled.")
		}
		config.Rules[i].Destination.Bucket = destBucket
	}

	stored := s.state.SetBucketReplicationConfig(bucket, config)
	if err := s.persist(); err != nil {
		return nil, err
	}
	return renderReplicationConfig(stored), nil
}

func (s *Service) DeleteBucketReplication(bucket string) error {
	bucket, err := s.requireBucket(bucket)
	if err != nil {
		return err
	}
	if s.state.DeleteBucketReplicationConfig(bucket) {
		return s.persist()
	}
	return nil
}

func notificationDefaultBody() []byte {
	return []byte(xml.Header + `<NotificationConfiguration xmlns="` + s3XMLNamespace + `"/>`)
}

func loggingDefaultBody() []byte {
	return []byte(xml.Header + `<BucketLoggingStatus xmlns="` + s3XMLNamespace + `"/>`)
}

func renderReplicationConfig(config domain.BucketReplicationConfig) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteByte('\n')
	buf.WriteString(`<ReplicationConfiguration xmlns="`)
	buf.WriteString(s3XMLNamespace)
	buf.WriteString(`">`)
	buf.WriteByte('\n')
	buf.WriteString("  <Role>")
	xml.EscapeText(&buf, []byte(config.Role))
	buf.WriteString("</Role>\n")
	for _, rule := range config.Rules {
		buf.WriteString("  <Rule>\n")
		buf.WriteString("    <ID>")
		xml.EscapeText(&buf, []byte(rule.ID))
		buf.WriteString("</ID>\n")
		buf.WriteString("    <Status>")
		status := rule.Status
		if status == "" {
			status = "Enabled"
		}
		xml.EscapeText(&buf, []byte(status))
		buf.WriteString("</Status>\n")
		if rule.Prefix != "" {
			buf.WriteString("    <Prefix>")
			xml.EscapeText(&buf, []byte(rule.Prefix))
			buf.WriteString("</Prefix>\n")
		}
		if rule.Destination.Bucket != "" || rule.Destination.StorageClass != "" {
			buf.WriteString("    <Destination>\n")
			if rule.Destination.Bucket != "" {
				buf.WriteString("      <Bucket>")
				xml.EscapeText(&buf, []byte(rule.Destination.Bucket))
				buf.WriteString("</Bucket>\n")
			}
			if rule.Destination.StorageClass != "" {
				buf.WriteString("      <StorageClass>")
				xml.EscapeText(&buf, []byte(rule.Destination.StorageClass))
				buf.WriteString("</StorageClass>\n")
			}
			buf.WriteString("    </Destination>\n")
		}
		buf.WriteString("  </Rule>\n")
	}
	buf.WriteString("</ReplicationConfiguration>\n")
	return buf.Bytes()
}

func parseReplicationConfig(body []byte) (domain.BucketReplicationConfig, error) {
	var payload replicationConfigXML
	if err := xml.Unmarshal(body, &payload); err != nil {
		return domain.BucketReplicationConfig{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}
	if strings.TrimSpace(payload.Role) == "" || len(payload.Rules) == 0 {
		return domain.BucketReplicationConfig{}, fmt.Errorf("s3: MalformedXML: The XML you provided was not well-formed")
	}

	config := domain.BucketReplicationConfig{
		Role:  strings.TrimSpace(payload.Role),
		Rules: make([]domain.BucketReplicationRule, len(payload.Rules)),
	}
	for i := range payload.Rules {
		rule := payload.Rules[i]
		config.Rules[i] = domain.BucketReplicationRule{
			ID:     strings.TrimSpace(rule.ID),
			Status: strings.TrimSpace(rule.Status),
			Prefix: strings.TrimSpace(rule.Prefix),
		}
		if rule.Destination != nil {
			config.Rules[i].Destination = domain.BucketReplicationDestination{
				Bucket:       strings.TrimSpace(rule.Destination.Bucket),
				StorageClass: strings.TrimSpace(rule.Destination.StorageClass),
			}
		}
	}
	return config, nil
}
