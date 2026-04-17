package infrastructure_test

import (
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/s3/application"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
	"github.com/michasdev/mildstack/core/internal/s3/infrastructure"
)

func TestHandlersObjectLockRoundTripAndCopySafety(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	bucket, err := service.CreateBucket("governed-bucket", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if _, err := service.PutBucketVersioning(bucket.Name, domain.VersioningEnabled); err != nil {
		t.Fatalf("enable versioning: %v", err)
	}

	if _, err := handlers.PutObjectLockConfiguration(infrastructure.PutObjectLockConfigurationRequest{
		Bucket: bucket.Name,
		Body: []byte(`
<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>30</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>`),
	}); err != nil {
		t.Fatalf("put object lock config: %v", err)
	}

	if _, err := service.PutObject(bucket.Name, "archive.txt", []byte("payload"), "text/plain"); err != nil {
		t.Fatalf("put object: %v", err)
	}

	if _, err := handlers.PutObjectRetention(infrastructure.PutObjectRetentionRequest{
		Bucket: bucket.Name,
		Key:    "archive.txt",
		Body: []byte(`
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>2026-04-18T00:00:00Z</RetainUntilDate>
</Retention>`),
	}); err != nil {
		t.Fatalf("put object retention: %v", err)
	}
	if _, err := handlers.PutObjectLegalHold(infrastructure.PutObjectLegalHoldRequest{
		Bucket: bucket.Name,
		Key:    "archive.txt",
		Body: []byte(`
<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>ON</Status>
</LegalHold>`),
	}); err != nil {
		t.Fatalf("put object legal hold: %v", err)
	}

	lockResp, err := handlers.GetObjectLockConfiguration(infrastructure.GetObjectLockConfigurationRequest{Bucket: bucket.Name})
	if err != nil {
		t.Fatalf("get object lock config: %v", err)
	}
	lockResp.ObjectLock.Body[0] = 'x'

	again, err := handlers.GetObjectLockConfiguration(infrastructure.GetObjectLockConfigurationRequest{Bucket: bucket.Name})
	if err != nil {
		t.Fatalf("get object lock config again: %v", err)
	}
	if !strings.Contains(string(again.ObjectLock.Body), "ObjectLockEnabled") {
		t.Fatalf("unexpected lock config body: %s", again.ObjectLock.Body)
	}

	retentionResp, err := handlers.GetObjectRetention(infrastructure.GetObjectRetentionRequest{
		Bucket: bucket.Name,
		Key:    "archive.txt",
	})
	if err != nil {
		t.Fatalf("get object retention: %v", err)
	}
	retentionResp.Retention.Body[0] = 'x'

	againRetention, err := handlers.GetObjectRetention(infrastructure.GetObjectRetentionRequest{
		Bucket: bucket.Name,
		Key:    "archive.txt",
	})
	if err != nil {
		t.Fatalf("get object retention again: %v", err)
	}
	if !strings.Contains(string(againRetention.Retention.Body), "RetainUntilDate") {
		t.Fatalf("unexpected retention body: %s", againRetention.Retention.Body)
	}

	holdResp, err := handlers.GetObjectLegalHold(infrastructure.GetObjectLegalHoldRequest{
		Bucket: bucket.Name,
		Key:    "archive.txt",
	})
	if err != nil {
		t.Fatalf("get object legal hold: %v", err)
	}
	holdResp.LegalHold.Body[0] = 'x'

	againHold, err := handlers.GetObjectLegalHold(infrastructure.GetObjectLegalHoldRequest{
		Bucket: bucket.Name,
		Key:    "archive.txt",
	})
	if err != nil {
		t.Fatalf("get object legal hold again: %v", err)
	}
	if !strings.Contains(string(againHold.LegalHold.Body), "Status") {
		t.Fatalf("unexpected legal hold body: %s", againHold.LegalHold.Body)
	}
}
