package application

import (
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

func TestServiceObjectLockConfigurationAndMutationGuards(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("governed-bucket", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if _, err := service.PutObjectLockConfiguration(bucket.Name, []byte(`
<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>30</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>`)); err == nil {
		t.Fatal("expected object lock configuration to require versioning")
	}

	if _, err := service.PutBucketVersioning(bucket.Name, domain.VersioningSuspended); err != nil {
		t.Fatalf("set suspended versioning: %v", err)
	}
	if _, err := service.PutObjectLockConfiguration(bucket.Name, []byte(`
<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>30</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>`)); err == nil {
		t.Fatal("expected suspended versioning to reject object lock configuration")
	}

	if _, err := service.PutBucketVersioning(bucket.Name, domain.VersioningEnabled); err != nil {
		t.Fatalf("enable versioning: %v", err)
	}

	config, err := service.PutObjectLockConfiguration(bucket.Name, []byte(`
<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>30</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>`))
	if err != nil {
		t.Fatalf("put object lock configuration: %v", err)
	}
	if !strings.Contains(string(config), "ObjectLockEnabled") {
		t.Fatalf("unexpected object lock config body: %s", config)
	}

	if _, err := service.PutObjectRetention(bucket.Name, "archive.txt", []byte(`
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>2026-04-18T00:00:00Z</RetainUntilDate>
</Retention>`)); err == nil {
		t.Fatal("expected retention write to require an existing object")
	}

	archive, err := service.PutObject(bucket.Name, "archive.txt", []byte("archive payload"), "text/plain")
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	if _, err := service.PutObjectRetention(bucket.Name, "archive.txt", []byte(`
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>2026-04-18T00:00:00Z</RetainUntilDate>
</Retention>`)); err != nil {
		t.Fatalf("put object retention: %v", err)
	}
	if _, err := service.PutObjectLegalHold(bucket.Name, "archive.txt", []byte(`
<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>ON</Status>
</LegalHold>`)); err != nil {
		t.Fatalf("put object legal hold: %v", err)
	}

	if body, err := service.GetObjectRetention(bucket.Name, archive.Key); err != nil || !strings.Contains(string(body), "RetainUntilDate") {
		t.Fatalf("unexpected object retention lookup: err=%v body=%s", err, body)
	}
	if body, err := service.GetObjectLegalHold(bucket.Name, archive.Key); err != nil || !strings.Contains(string(body), "Status") {
		t.Fatalf("unexpected object legal hold lookup: err=%v body=%s", err, body)
	}

	if _, err := service.PutObject(bucket.Name, "archive.txt", []byte("updated payload"), "text/plain"); err == nil {
		t.Fatal("expected protected overwrite to fail")
	}
	if _, err := service.CopyObject(bucket.Name, "archive.txt", bucket.Name, archive.Key); err == nil {
		t.Fatal("expected protected copy destination to fail")
	}
	if err := service.DeleteObject(bucket.Name, "archive.txt"); err == nil {
		t.Fatal("expected protected delete to fail")
	}

	upload, err := service.CreateMultipartUpload(bucket.Name, "archive.txt", "text/plain", nil, nil)
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	if _, err := service.UploadPart(upload.UploadID, 1, []byte("part-1")); err != nil {
		t.Fatalf("upload part: %v", err)
	}
	if _, err := service.CompleteMultipartUpload(upload.UploadID); err == nil {
		t.Fatal("expected protected multipart completion to fail")
	}
}
