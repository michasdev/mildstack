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
	archive, err := service.PutObject(bucket.Name, "archive.txt", []byte("archive payload"), "text/plain")
	if err != nil {
		t.Fatalf("put object: %v", err)
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

	if _, err := service.PutObjectRetention(bucket.Name, "missing.txt", []byte(`
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>2026-04-18T00:00:00Z</RetainUntilDate>
</Retention>`)); err == nil {
		t.Fatal("expected retention write to require an existing object")
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

func TestServiceObjectLockAppliesDefaultRetentionToNewWrites(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("retained-bucket", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	if _, err := service.PutObject(bucket.Name, "source.txt", []byte("source payload"), "text/plain"); err != nil {
		t.Fatalf("put source object: %v", err)
	}
	if _, err := service.PutBucketVersioning(bucket.Name, domain.VersioningEnabled); err != nil {
		t.Fatalf("enable versioning: %v", err)
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
</ObjectLockConfiguration>`)); err != nil {
		t.Fatalf("put object lock configuration: %v", err)
	}

	if _, err := service.CopyObject(bucket.Name, "copy.txt", bucket.Name, "source.txt"); err != nil {
		t.Fatalf("copy object: %v", err)
	}
	if _, err := service.PutObject(bucket.Name, "new.txt", []byte("new payload"), "text/plain"); err != nil {
		t.Fatalf("put retained object: %v", err)
	}

	upload, err := service.CreateMultipartUpload(bucket.Name, "multipart.txt", "text/plain", nil, nil)
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	if _, err := service.UploadPart(upload.UploadID, 1, []byte("part-1")); err != nil {
		t.Fatalf("upload multipart part: %v", err)
	}
	if _, err := service.CompleteMultipartUpload(upload.UploadID); err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	for _, key := range []string{"copy.txt", "new.txt", "multipart.txt"} {
		body, err := service.GetObjectRetention(bucket.Name, key)
		if err != nil {
			t.Fatalf("get retention for %s: %v", key, err)
		}
		if !strings.Contains(string(body), "RetainUntilDate") {
			t.Fatalf("expected retention body for %s, got %s", key, body)
		}
	}
}

func TestServiceDeleteObjectsContinuesAfterProtectedErrors(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("batch-delete-bucket", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	if _, err := service.PutObject(bucket.Name, "free.txt", []byte("free payload"), "text/plain"); err != nil {
		t.Fatalf("put free object: %v", err)
	}
	if _, err := service.PutBucketVersioning(bucket.Name, domain.VersioningEnabled); err != nil {
		t.Fatalf("enable versioning: %v", err)
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
</ObjectLockConfiguration>`)); err != nil {
		t.Fatalf("put object lock configuration: %v", err)
	}
	if _, err := service.PutObject(bucket.Name, "protected.txt", []byte("protected payload"), "text/plain"); err != nil {
		t.Fatalf("put protected object: %v", err)
	}

	result, err := service.DeleteObjects(DeleteObjectsRequest{
		Bucket: bucket.Name,
		Keys:   []string{"protected.txt", "free.txt"},
	})
	if err != nil {
		t.Fatalf("delete objects: %v", err)
	}
	if got, want := len(result.Deleted), 1; got != want {
		t.Fatalf("unexpected deleted count: got %d want %d", got, want)
	}
	if got, want := result.Deleted[0].Key, "free.txt"; got != want {
		t.Fatalf("unexpected deleted key: got %q want %q", got, want)
	}
	if got, want := len(result.Errors), 1; got != want {
		t.Fatalf("unexpected error count: got %d want %d", got, want)
	}
	if got, want := result.Errors[0].Key, "protected.txt"; got != want {
		t.Fatalf("unexpected error key: got %q want %q", got, want)
	}
	if got, want := result.Errors[0].Code, "AccessDenied"; got != want {
		t.Fatalf("unexpected error code: got %q want %q", got, want)
	}
	if remaining, err := service.ListObjectsV1(ListObjectsV1Request{Bucket: bucket.Name}); err != nil {
		t.Fatalf("list remaining objects: %v", err)
	} else if got, want := len(remaining.Objects), 1; got != want {
		t.Fatalf("unexpected remaining count: got %d want %d", got, want)
	} else if got, want := remaining.Objects[0].Key, "protected.txt"; got != want {
		t.Fatalf("unexpected remaining key: got %q want %q", got, want)
	}
}

func TestServiceDeleteBucketClearsMultipartUploads(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("multipart-cleanup-bucket", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	upload, err := service.CreateMultipartUpload(bucket.Name, "stale.txt", "text/plain", nil, nil)
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	if got, want := len(service.multipartUploads), 1; got != want {
		t.Fatalf("unexpected multipart upload count before delete: got %d want %d", got, want)
	}

	if err := service.DeleteBucket(bucket.Name); err != nil {
		t.Fatalf("delete bucket: %v", err)
	}
	if got, want := len(service.multipartUploads), 0; got != want {
		t.Fatalf("unexpected multipart upload count after delete: got %d want %d", got, want)
	}
	if _, err := service.CompleteMultipartUpload(upload.UploadID); err == nil {
		t.Fatal("expected stale multipart completion to fail after bucket delete")
	}
}
