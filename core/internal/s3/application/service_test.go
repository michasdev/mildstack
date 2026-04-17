package application

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

func TestServiceMetadataRoutesAndState(t *testing.T) {
	t.Helper()

	service := New()
	if _, ok := any(service).(orchestrator.Service); !ok {
		t.Fatal("expected service to satisfy orchestrator.Service")
	}

	metadata := service.Metadata()
	if got, want := metadata.Name, "s3"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := metadata.Version, "v1"; got != want {
		t.Fatalf("unexpected service version: got %q want %q", got, want)
	}
	if got, want := metadata.Description, "MildStack S3 real service"; got != want {
		t.Fatalf("unexpected service description: got %q want %q", got, want)
	}

	policy := service.Policy()
	if got, want := policy.Fidelity, orchestrator.FidelityExemplar; got != want {
		t.Fatalf("unexpected policy fidelity: got %q want %q", got, want)
	}
	if got, want := policy.ErrorPrefix, "s3"; got != want {
		t.Fatalf("unexpected policy error prefix: got %q want %q", got, want)
	}
	if got, want := len(policy.Supported), 20; got != want {
		t.Fatalf("unexpected supported count: got %d want %d", got, want)
	}
	if got, want := len(policy.Unsupported), 1; got != want {
		t.Fatalf("unexpected unsupported count: got %d want %d", got, want)
	}
	policy.Supported[0] = "changed"
	policy.Unsupported[0] = "changed"
	again := service.Policy()
	if got, want := again.Supported[0], "list buckets"; got != want {
		t.Fatalf("policy supported slice was not copied: got %q want %q", got, want)
	}
	if got, want := again.Supported[4], "bucket policy"; got != want {
		t.Fatalf("expected policy to move into supported capabilities: got %q want %q", got, want)
	}
	if got, want := again.Supported[9], "bucket tagging"; got != want {
		t.Fatalf("expected tagging to move into supported capabilities: got %q want %q", got, want)
	}
	if got, want := again.Supported[18], "bucket versioning"; got != want {
		t.Fatalf("expected versioning to move into supported capabilities: got %q want %q", got, want)
	}
	if got, want := again.Supported[19], "multipart upload"; got != want {
		t.Fatalf("expected multipart to move into supported capabilities: got %q want %q", got, want)
	}
	if got, want := again.Unsupported[0], "object locking"; got != want {
		t.Fatalf("policy unsupported slice was not copied: got %q want %q", got, want)
	}

	registrar := deliveryhttp.NewRegistrar()
	if err := service.RegisterRoutes(registrar); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	entry, ok := registrar.Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(entry.Routes), 35; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	expectedRoutes := []struct {
		method string
		path   string
		name   string
	}{
		{"GET", "/api/v1/runtime/services/s3/buckets", "s3.buckets.index"},
		{"POST", "/api/v1/runtime/services/s3/buckets", "s3.buckets.create"},
		{"HEAD", "/api/v1/runtime/services/s3/buckets/:bucket", "s3.buckets.head"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket", "s3.buckets.delete"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/policy", "s3.buckets.policy.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/policy", "s3.buckets.policy.update"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/policy", "s3.buckets.policy.delete"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/encryption", "s3.buckets.encryption.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/encryption", "s3.buckets.encryption.update"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/encryption", "s3.buckets.encryption.delete"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/lifecycle", "s3.buckets.lifecycle.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/lifecycle", "s3.buckets.lifecycle.update"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/lifecycle", "s3.buckets.lifecycle.delete"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/cors", "s3.buckets.cors.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/cors", "s3.buckets.cors.update"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/cors", "s3.buckets.cors.delete"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/acl", "s3.buckets.acl.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/acl", "s3.buckets.acl.update"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/tagging", "s3.buckets.tagging.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/tagging", "s3.buckets.tagging.update"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/tagging", "s3.buckets.tagging.delete"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/versioning", "s3.buckets.versioning.show"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/versioning", "s3.buckets.versioning.update"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects/versions", "s3.objects.versions"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects", "s3.objects.list-v1"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects/v2", "s3.objects.list-v2"},
		{"POST", "/api/v1/runtime/services/s3/buckets/:bucket/objects/delete", "s3.objects.delete-batch"},
		{"GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object", "s3.objects.show"},
		{"HEAD", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object", "s3.objects.head"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object", "s3.objects.update"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object", "s3.objects.delete"},
		{"POST", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object/uploads", "s3.multipart.uploads.create"},
		{"PUT", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object/uploads/:upload/parts/:part", "s3.multipart.uploads.part"},
		{"POST", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object/uploads/:upload/complete", "s3.multipart.uploads.complete"},
		{"DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object/uploads/:upload", "s3.multipart.uploads.abort"},
	}
	sort.SliceStable(expectedRoutes, func(i, j int) bool {
		if expectedRoutes[i].method != expectedRoutes[j].method {
			return expectedRoutes[i].method < expectedRoutes[j].method
		}
		if expectedRoutes[i].path != expectedRoutes[j].path {
			return expectedRoutes[i].path < expectedRoutes[j].path
		}
		return expectedRoutes[i].name < expectedRoutes[j].name
	})
	for i, expected := range expectedRoutes {
		if got, want := entry.Routes[i].Method, expected.method; got != want {
			t.Fatalf("unexpected route method at %d: got %q want %q", i, got, want)
		}
		if got, want := entry.Routes[i].Path, expected.path; got != want {
			t.Fatalf("unexpected route path at %d: got %q want %q", i, got, want)
		}
		if got, want := entry.Routes[i].Name, expected.name; got != want {
			t.Fatalf("unexpected route name at %d: got %q want %q", i, got, want)
		}
	}

	hook := runtime.NewStateHook()
	if err := service.AttachState(hook); err != nil {
		t.Fatalf("attach state: %v", err)
	}

	value, ok := hook.Get("services/s3")
	if !ok {
		t.Fatal("expected s3 state to be present")
	}
	state := value.(map[string]any)
	if got, want := state["service"], "s3"; got != want {
		t.Fatalf("unexpected service state name: got %v want %v", got, want)
	}
}

func TestServiceRealOperationsMutateState(t *testing.T) {
	t.Helper()

	service := New()

	bucket, err := service.CreateBucket("mildstack-logs", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if got, want := bucket.Name, "mildstack-logs"; got != want {
		t.Fatalf("unexpected bucket name: got %q want %q", got, want)
	}
	if got, want := bucket.Region, "us-west-2"; got != want {
		t.Fatalf("unexpected bucket region: got %q want %q", got, want)
	}

	buckets := service.ListBuckets()
	if got, want := len(buckets), 2; got != want {
		t.Fatalf("unexpected bucket count: got %d want %d", got, want)
	}
	if buckets[0].CreatedAt.IsZero() || buckets[1].CreatedAt.IsZero() {
		t.Fatal("expected listed buckets to include creation timestamps")
	}

	object, err := service.PutObject(bucket.Name, "archive.txt", []byte("archive payload"), "text/plain")
	if err != nil {
		t.Fatalf("put object: %v", err)
	}
	if got, want := object.Key, "archive.txt"; got != want {
		t.Fatalf("unexpected object key: got %q want %q", got, want)
	}

	objects, err := service.ListObjects(bucket.Name)
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if got, want := len(objects), 1; got != want {
		t.Fatalf("unexpected object count: got %d want %d", got, want)
	}

	fetched, err := service.GetObject(bucket.Name, object.Key)
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if got, want := fetched.ContentType, "text/plain"; got != want {
		t.Fatalf("unexpected object content type: got %q want %q", got, want)
	}
	if got, want := string(fetched.Body), "archive payload"; got != want {
		t.Fatalf("unexpected object body: got %q want %q", got, want)
	}

	head, err := service.HeadObject(bucket.Name, object.Key)
	if err != nil {
		t.Fatalf("head object: %v", err)
	}
	if got, want := head.ETag, object.ETag; got != want {
		t.Fatalf("unexpected head etag: got %q want %q", got, want)
	}
	if len(head.Body) != 0 {
		t.Fatalf("expected head object body to be empty, got %d bytes", len(head.Body))
	}

	copied, err := service.CopyObject(bucket.Name, "archive-copy.txt", bucket.Name, object.Key)
	if err != nil {
		t.Fatalf("copy object: %v", err)
	}
	if got, want := string(copied.Body), "archive payload"; got != want {
		t.Fatalf("unexpected copied body: got %q want %q", got, want)
	}
	if got, want := copied.ETag, object.ETag; got != want {
		t.Fatalf("unexpected copied etag: got %q want %q", got, want)
	}

	copied.Body[0] = 'A'
	restoredCopy, err := service.GetObject(bucket.Name, "archive-copy.txt")
	if err != nil {
		t.Fatalf("get copied object: %v", err)
	}
	if got, want := string(restoredCopy.Body), "archive payload"; got != want {
		t.Fatalf("copied object body was aliased: got %q want %q", got, want)
	}

	if err := service.DeleteObject(bucket.Name, object.Key); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if err := service.DeleteObject(bucket.Name, object.Key); err != nil {
		t.Fatalf("delete missing object should stay idempotent: %v", err)
	}
	if _, err := service.GetObject(bucket.Name, object.Key); err == nil {
		t.Fatal("expected deleted object lookup to fail")
	} else if !strings.Contains(err.Error(), "NoSuchKey") {
		t.Fatalf("expected NoSuchKey error, got %v", err)
	}
}

func TestServiceBucketGovernanceSubresourcesRoundTripAndCleanup(t *testing.T) {
	t.Helper()

	service := New()

	bucket, err := service.CreateBucket("mildstack-governed", "us-east-1")
	if err != nil {
		t.Fatalf("create governed bucket: %v", err)
	}

	assertRoundTrip := func(name string, put func([]byte) ([]byte, error), get func() ([]byte, error), want string) {
		t.Helper()

		stored, err := put([]byte(want))
		if err != nil {
			t.Fatalf("put %s: %v", name, err)
		}
		if len(stored) > 0 {
			stored[0] = 'X'
		}

		fetched, err := get()
		if err != nil {
			t.Fatalf("get %s: %v", name, err)
		}
		if got, want := string(fetched), want; got != want {
			t.Fatalf("unexpected %s body: got %q want %q", name, got, want)
		}
	}

	assertRoundTrip("policy",
		func(body []byte) ([]byte, error) { return service.PutBucketPolicy(bucket.Name, body) },
		func() ([]byte, error) { return service.GetBucketPolicy(bucket.Name) },
		`{"Version":"2012-10-17"}`,
	)
	assertRoundTrip("encryption",
		func(body []byte) ([]byte, error) { return service.PutBucketEncryption(bucket.Name, body) },
		func() ([]byte, error) { return service.GetBucketEncryption(bucket.Name) },
		"<ServerSideEncryptionConfiguration/>",
	)
	assertRoundTrip("lifecycle",
		func(body []byte) ([]byte, error) { return service.PutBucketLifecycle(bucket.Name, body) },
		func() ([]byte, error) { return service.GetBucketLifecycle(bucket.Name) },
		"<LifecycleConfiguration/>",
	)
	assertRoundTrip("cors",
		func(body []byte) ([]byte, error) { return service.PutBucketCORS(bucket.Name, body) },
		func() ([]byte, error) { return service.GetBucketCORS(bucket.Name) },
		"<CORSConfiguration/>",
	)
	assertRoundTrip("tagging",
		func(body []byte) ([]byte, error) { return service.PutBucketTagging(bucket.Name, body) },
		func() ([]byte, error) { return service.GetBucketTagging(bucket.Name) },
		"<Tagging><TagSet><Tag><Key>env</Key><Value>dev</Value></Tag></TagSet></Tagging>",
	)

	aclDefault, err := service.GetBucketACL(bucket.Name)
	if err != nil {
		t.Fatalf("get default acl: %v", err)
	}
	if got, want := string(aclDefault), defaultBucketACLBody(bucket.Name); got != string(want) {
		t.Fatalf("unexpected default ACL body: got %q want %q", got, string(want))
	}

	aclBody := []byte("<AccessControlPolicy><Owner><ID>owner</ID></Owner></AccessControlPolicy>")
	storedACL, err := service.PutBucketACL(bucket.Name, aclBody)
	if err != nil {
		t.Fatalf("put acl: %v", err)
	}
	storedACL[0] = 'X'
	againACL, err := service.GetBucketACL(bucket.Name)
	if err != nil {
		t.Fatalf("get stored acl: %v", err)
	}
	if got, want := string(againACL), string(aclBody); got != want {
		t.Fatalf("unexpected stored ACL body: got %q want %q", got, want)
	}

	if err := service.DeleteBucket(bucket.Name); err != nil {
		t.Fatalf("delete governed bucket: %v", err)
	}

	recreated, err := service.CreateBucket(bucket.Name, "us-east-1")
	if err != nil {
		t.Fatalf("recreate governed bucket: %v", err)
	}
	if recreated.Name != bucket.Name {
		t.Fatalf("unexpected recreated bucket name: got %q want %q", recreated.Name, bucket.Name)
	}
	if _, err := service.GetBucketPolicy(bucket.Name); err == nil {
		t.Fatal("expected cleared policy lookup to fail after bucket deletion")
	}

	rebuiltACL, err := service.GetBucketACL(bucket.Name)
	if err != nil {
		t.Fatalf("get recreated bucket acl: %v", err)
	}
	if got, want := string(rebuiltACL), string(defaultBucketACLBody(bucket.Name)); got != want {
		t.Fatalf("expected ACL cleanup to restore default body: got %q want %q", got, want)
	}
}

func TestServiceVersioningTracksHistoryAndDeleteMarkers(t *testing.T) {
	t.Helper()

	service := New()

	versioned, err := service.CreateBucket("mildstack-versioned", "us-east-1")
	if err != nil {
		t.Fatalf("create versioned bucket: %v", err)
	}
	if _, err := service.PutBucketVersioning(versioned.Name, domain.VersioningEnabled); err != nil {
		t.Fatalf("enable bucket versioning: %v", err)
	}

	if _, err := service.PutObject(versioned.Name, "release.txt", []byte("v1"), "text/plain"); err != nil {
		t.Fatalf("put first version: %v", err)
	}
	if _, err := service.PutObject(versioned.Name, "release.txt", []byte("v2"), "text/plain"); err != nil {
		t.Fatalf("put second version: %v", err)
	}
	if err := service.DeleteObject(versioned.Name, "release.txt"); err != nil {
		t.Fatalf("delete versioned object: %v", err)
	}

	if _, err := service.GetObject(versioned.Name, "release.txt"); err == nil {
		t.Fatal("expected deleted versioned object lookup to fail")
	}

	versions, err := service.ListObjectVersions(versioned.Name)
	if err != nil {
		t.Fatalf("list object versions: %v", err)
	}
	if got, want := len(versions.Versions), 3; got != want {
		t.Fatalf("unexpected version count: got %d want %d", got, want)
	}
	if !versions.Versions[0].IsDeleteMarker {
		t.Fatal("expected latest entry to be a delete marker")
	}
	if got, want := versions.Versions[1].ContentType, "text/plain"; got != want {
		t.Fatalf("unexpected second version content type: got %q want %q", got, want)
	}
	if got, want := string(versions.Versions[1].Body), "v2"; got != want {
		t.Fatalf("unexpected second version body: got %q want %q", got, want)
	}
	if got, want := string(versions.Versions[2].Body), "v1"; got != want {
		t.Fatalf("unexpected first version body: got %q want %q", got, want)
	}

	plain, err := service.CreateBucket("mildstack-plain", "us-east-1")
	if err != nil {
		t.Fatalf("create plain bucket: %v", err)
	}
	if _, err := service.PutObject(plain.Name, "plain.txt", []byte("plain"), "text/plain"); err != nil {
		t.Fatalf("put plain object: %v", err)
	}
	plainVersions, err := service.ListObjectVersions(plain.Name)
	if err != nil {
		t.Fatalf("list plain versions: %v", err)
	}
	if got, want := len(plainVersions.Versions), 1; got != want {
		t.Fatalf("unexpected plain version count: got %d want %d", got, want)
	}
	if got, want := plainVersions.Versions[0].VersionID, domain.VersioningNull; got != want {
		t.Fatalf("unexpected plain version id: got %q want %q", got, want)
	}

	if err := service.DeleteBucket(versioned.Name); err == nil {
		t.Fatal("expected versioned bucket delete to fail while history exists")
	} else if !strings.Contains(err.Error(), "BucketNotEmpty") {
		t.Fatalf("expected BucketNotEmpty for versioned bucket delete, got %v", err)
	}
}

func TestServiceMultipartLifecycleAssemblesAndAbortsCopySafely(t *testing.T) {
	t.Helper()

	service := New()

	bucket, err := service.CreateBucket("mildstack-multipart", "us-east-1")
	if err != nil {
		t.Fatalf("create multipart bucket: %v", err)
	}

	upload, err := service.CreateMultipartUpload(bucket.Name, "archive.bin", "application/octet-stream", map[string]string{"owner": "ops"}, map[string]string{"cache-control": "no-cache"})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	if got, want := len(service.multipartUploads), 1; got != want {
		t.Fatalf("unexpected multipart registry size: got %d want %d", got, want)
	}

	firstBody := []byte("one")
	secondBody := []byte("two")
	partTwo, err := service.UploadPart(upload.UploadID, 2, secondBody)
	if err != nil {
		t.Fatalf("upload second part: %v", err)
	}
	if got, want := partTwo.PartNumber, 2; got != want {
		t.Fatalf("unexpected second part number: got %d want %d", got, want)
	}
	secondBody[0] = 'X'
	partOne, err := service.UploadPart(upload.UploadID, 1, firstBody)
	if err != nil {
		t.Fatalf("upload first part: %v", err)
	}
	if got, want := partOne.PartNumber, 1; got != want {
		t.Fatalf("unexpected first part number: got %d want %d", got, want)
	}
	firstBody[0] = 'Y'

	storedUpload := service.multipartUploads[upload.UploadID]
	if got, want := string(storedUpload.Parts[0].Body), "two"; got != want {
		t.Fatalf("stored second part body was aliased: got %q want %q", got, want)
	}
	if got, want := string(storedUpload.Parts[1].Body), "one"; got != want {
		t.Fatalf("stored first part body was aliased: got %q want %q", got, want)
	}

	completed, err := service.CompleteMultipartUpload(upload.UploadID)
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}
	if got, want := string(completed.Body), "onetwo"; got != want {
		t.Fatalf("unexpected assembled body: got %q want %q", got, want)
	}
	if got, want := completed.Size, int64(len("onetwo")); got != want {
		t.Fatalf("unexpected assembled size: got %d want %d", got, want)
	}
	if got, want := completed.ETag, expectedMultipartETag("one", "two"); got != want {
		t.Fatalf("unexpected assembled etag: got %q want %q", got, want)
	}
	if got, want := completed.ContentType, "application/octet-stream"; got != want {
		t.Fatalf("unexpected assembled content type: got %q want %q", got, want)
	}
	if got, want := len(service.multipartUploads), 0; got != want {
		t.Fatalf("expected multipart registry to be cleared after completion, got %d entries", got)
	}

	completed.Body[0] = 'O'
	fetched, err := service.GetObject(bucket.Name, "archive.bin")
	if err != nil {
		t.Fatalf("get completed object: %v", err)
	}
	if got, want := string(fetched.Body), "onetwo"; got != want {
		t.Fatalf("completed object body was aliased: got %q want %q", got, want)
	}

	abortedUpload, err := service.CreateMultipartUpload(bucket.Name, "aborted.bin", "text/plain", nil, nil)
	if err != nil {
		t.Fatalf("create aborted upload: %v", err)
	}
	if _, err := service.UploadPart(abortedUpload.UploadID, 1, []byte("abort")); err != nil {
		t.Fatalf("upload aborted part: %v", err)
	}
	if err := service.AbortMultipartUpload(abortedUpload.UploadID); err != nil {
		t.Fatalf("abort multipart upload: %v", err)
	}
	if got, want := len(service.multipartUploads), 0; got != want {
		t.Fatalf("expected multipart registry to be empty after abort, got %d entries", got)
	}
	if _, err := service.CompleteMultipartUpload(abortedUpload.UploadID); err == nil {
		t.Fatal("expected completing an aborted upload to fail")
	}
}

func TestServiceBucketCatalogFollowsAWSSemantics(t *testing.T) {
	t.Helper()

	service := New()

	owned, err := service.CreateBucket("mildstack-assets", "")
	if err != nil {
		t.Fatalf("idempotent create bucket: %v", err)
	}
	if got, want := owned.Region, "us-east-1"; got != want {
		t.Fatalf("unexpected owned bucket region: got %q want %q", got, want)
	}

	created, err := service.CreateBucket("mildstack-logs", "")
	if err != nil {
		t.Fatalf("create bucket with default region: %v", err)
	}
	if got, want := created.Region, "us-east-1"; got != want {
		t.Fatalf("unexpected default region: got %q want %q", got, want)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected created bucket timestamp")
	}

	head, err := service.HeadBucket("mildstack-logs")
	if err != nil {
		t.Fatalf("head bucket: %v", err)
	}
	if got, want := head.Region, "us-east-1"; got != want {
		t.Fatalf("unexpected head bucket region: got %q want %q", got, want)
	}

	if err := service.DeleteBucket("mildstack-assets"); err == nil {
		t.Fatal("expected non-empty bootstrap bucket delete to fail")
	} else if !strings.Contains(err.Error(), "BucketNotEmpty") {
		t.Fatalf("expected BucketNotEmpty error, got %v", err)
	}

	if err := service.DeleteBucket("mildstack-logs"); err != nil {
		t.Fatalf("delete empty bucket: %v", err)
	}
	if _, err := service.HeadBucket("mildstack-logs"); err == nil {
		t.Fatal("expected deleted bucket head to fail")
	}
}

func expectedMultipartETag(parts ...string) string {
	digests := make([]byte, 0, len(parts)*md5.Size)
	for _, part := range parts {
		sum := md5.Sum([]byte(part))
		digests = append(digests, sum[:]...)
	}
	final := md5.Sum(digests)
	return `"` + hex.EncodeToString(final[:]) + `-` + fmt.Sprint(len(parts)) + `"`
}

func TestServiceRejectsInvalidAndMissingRequests(t *testing.T) {
	t.Helper()

	service := New()

	if _, err := service.CreateBucket("", ""); err == nil {
		t.Fatal("expected empty bucket name to fail")
	}
	if _, err := service.CreateBucket("Invalid_Bucket", ""); err == nil {
		t.Fatal("expected invalid bucket name to fail")
	}
	if _, err := service.HeadBucket("missing"); err == nil {
		t.Fatal("expected missing bucket head to fail")
	}
	if err := service.DeleteBucket("missing"); err == nil {
		t.Fatal("expected missing bucket delete to fail")
	}
	if _, err := service.ListObjects("missing"); err == nil {
		t.Fatal("expected missing bucket listing to fail")
	}
	if _, err := service.GetObject("mildstack-assets", "missing"); err == nil {
		t.Fatal("expected missing object lookup to fail")
	} else if !strings.Contains(err.Error(), "NoSuchKey") {
		t.Fatalf("expected NoSuchKey lookup error, got %v", err)
	}
	if _, err := service.PutObject("missing", "archive.txt", []byte("x"), "text/plain"); err == nil {
		t.Fatal("expected put on missing bucket to fail")
	}
	if _, err := service.HeadObject("mildstack-assets", "missing"); err == nil {
		t.Fatal("expected missing object head to fail")
	} else if !strings.Contains(err.Error(), "NoSuchKey") {
		t.Fatalf("expected NoSuchKey head error, got %v", err)
	}
	if _, err := service.CopyObject("mildstack-assets", "copy.txt", "mildstack-assets", "missing"); err == nil {
		t.Fatal("expected copy from missing object to fail")
	} else if !strings.Contains(err.Error(), "NoSuchKey") {
		t.Fatalf("expected NoSuchKey copy error, got %v", err)
	}
	if err := service.DeleteObject("mildstack-assets", "missing"); err != nil {
		t.Fatalf("expected delete on missing object to succeed: %v", err)
	}
}

func TestServiceListObjectsV1UsesMarkerPaginationDeterministically(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("catalog-bucket", "us-east-1")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	for _, key := range []string{"charlie.txt", "alpha.txt", "bravo.txt"} {
		if _, err := service.PutObject(bucket.Name, key, []byte(key), "text/plain"); err != nil {
			t.Fatalf("put object %q: %v", key, err)
		}
	}

	first, err := service.ListObjectsV1(ListObjectsV1Request{
		Bucket:  bucket.Name,
		MaxKeys: 2,
	})
	if err != nil {
		t.Fatalf("list objects v1 first page: %v", err)
	}

	if got, want := len(first.Objects), 2; got != want {
		t.Fatalf("unexpected first page object count: got %d want %d", got, want)
	}
	if got, want := first.Objects[0].Key, "alpha.txt"; got != want {
		t.Fatalf("unexpected first object key: got %q want %q", got, want)
	}
	if got, want := first.Objects[1].Key, "bravo.txt"; got != want {
		t.Fatalf("unexpected second object key: got %q want %q", got, want)
	}
	if !first.IsTruncated {
		t.Fatal("expected first page to be truncated")
	}
	if got := first.NextMarker; got != "" {
		t.Fatalf("expected v1 next marker to stay empty without delimiter, got %q", got)
	}

	second, err := service.ListObjectsV1(ListObjectsV1Request{
		Bucket: bucket.Name,
		Marker: first.Objects[len(first.Objects)-1].Key,
	})
	if err != nil {
		t.Fatalf("list objects v1 second page: %v", err)
	}
	if got, want := len(second.Objects), 1; got != want {
		t.Fatalf("unexpected second page object count: got %d want %d", got, want)
	}
	if got, want := second.Objects[0].Key, "charlie.txt"; got != want {
		t.Fatalf("unexpected trailing object key: got %q want %q", got, want)
	}
}

func TestServiceListObjectsV2UsesContinuationTokensAndStartAfter(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("catalog-v2-bucket", "us-east-1")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	for _, key := range []string{"charlie.txt", "alpha.txt", "bravo.txt"} {
		if _, err := service.PutObject(bucket.Name, key, []byte(key), "text/plain"); err != nil {
			t.Fatalf("put object %q: %v", key, err)
		}
	}

	first, err := service.ListObjectsV2(ListObjectsV2Request{
		Bucket:  bucket.Name,
		MaxKeys: 2,
	})
	if err != nil {
		t.Fatalf("list objects v2 first page: %v", err)
	}
	if got, want := len(first.Objects), 2; got != want {
		t.Fatalf("unexpected first page object count: got %d want %d", got, want)
	}
	if got, want := first.KeyCount, 2; got != want {
		t.Fatalf("unexpected key count: got %d want %d", got, want)
	}
	if !first.IsTruncated {
		t.Fatal("expected first page to be truncated")
	}
	if got, want := first.NextContinuationToken, base64.StdEncoding.EncodeToString([]byte("bravo.txt")); got != want {
		t.Fatalf("unexpected continuation token: got %q want %q", got, want)
	}

	second, err := service.ListObjectsV2(ListObjectsV2Request{
		Bucket:            bucket.Name,
		ContinuationToken: first.NextContinuationToken,
	})
	if err != nil {
		t.Fatalf("list objects v2 second page: %v", err)
	}
	if got, want := len(second.Objects), 1; got != want {
		t.Fatalf("unexpected second page object count: got %d want %d", got, want)
	}
	if got, want := second.Objects[0].Key, "charlie.txt"; got != want {
		t.Fatalf("unexpected second page object key: got %q want %q", got, want)
	}

	startAfter, err := service.ListObjectsV2(ListObjectsV2Request{
		Bucket:     bucket.Name,
		StartAfter: "alpha.txt",
	})
	if err != nil {
		t.Fatalf("list objects v2 with start-after: %v", err)
	}
	if got, want := len(startAfter.Objects), 2; got != want {
		t.Fatalf("unexpected start-after object count: got %d want %d", got, want)
	}
	if got, want := startAfter.Objects[0].Key, "bravo.txt"; got != want {
		t.Fatalf("unexpected start-after first key: got %q want %q", got, want)
	}
	if got, want := startAfter.Objects[1].Key, "charlie.txt"; got != want {
		t.Fatalf("unexpected start-after second key: got %q want %q", got, want)
	}
}

func TestServiceDeleteObjectsPreservesOrderAndTreatsMissingKeysAsDeleted(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("catalog-delete-bucket", "us-east-1")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	for _, key := range []string{"charlie.txt", "alpha.txt", "bravo.txt"} {
		if _, err := service.PutObject(bucket.Name, key, []byte(key), "text/plain"); err != nil {
			t.Fatalf("put object %q: %v", key, err)
		}
	}

	result, err := service.DeleteObjects(DeleteObjectsRequest{
		Bucket: bucket.Name,
		Keys:   []string{"missing.txt", "bravo.txt", "alpha.txt"},
	})
	if err != nil {
		t.Fatalf("delete objects: %v", err)
	}
	if got, want := len(result.Deleted), 3; got != want {
		t.Fatalf("unexpected deleted count: got %d want %d", got, want)
	}
	if got, want := result.Deleted[0].Key, "missing.txt"; got != want {
		t.Fatalf("unexpected first deleted key: got %q want %q", got, want)
	}
	if got, want := result.Deleted[1].Key, "bravo.txt"; got != want {
		t.Fatalf("unexpected second deleted key: got %q want %q", got, want)
	}
	if got, want := result.Deleted[2].Key, "alpha.txt"; got != want {
		t.Fatalf("unexpected third deleted key: got %q want %q", got, want)
	}

	remaining, err := service.ListObjectsV1(ListObjectsV1Request{Bucket: bucket.Name})
	if err != nil {
		t.Fatalf("list remaining objects: %v", err)
	}
	if got, want := len(remaining.Objects), 1; got != want {
		t.Fatalf("unexpected remaining count: got %d want %d", got, want)
	}
	if got, want := remaining.Objects[0].Key, "charlie.txt"; got != want {
		t.Fatalf("unexpected remaining key: got %q want %q", got, want)
	}

	quiet, err := service.DeleteObjects(DeleteObjectsRequest{
		Bucket: bucket.Name,
		Keys:   []string{"charlie.txt"},
		Quiet:  true,
	})
	if err != nil {
		t.Fatalf("quiet delete objects: %v", err)
	}
	if got := len(quiet.Deleted); got != 0 {
		t.Fatalf("expected quiet delete to omit deleted payload, got %d entries", got)
	}
}

func TestServiceStartAndStopAreNoops(t *testing.T) {
	t.Helper()

	service := New()

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestServicePersistenceRoundTripAcrossRestart(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	config := StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "phase-12-instance",
	}

	first, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("new with persistence: %v", err)
	}

	bucket, err := first.CreateBucket("mildstack-logs", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if _, err := first.PutObject(bucket.Name, "archive.txt", []byte("persistent archive payload"), "text/plain"); err != nil {
		t.Fatalf("put object: %v", err)
	}

	second, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("new with persistence after restart: %v", err)
	}

	buckets := second.ListBuckets()
	if got, want := len(buckets), 2; got != want {
		t.Fatalf("unexpected bucket count after restart: got %d want %d", got, want)
	}
	object, err := second.GetObject(bucket.Name, "archive.txt")
	if err != nil {
		t.Fatalf("expected restored object after restart: %v", err)
	}
	if got, want := string(object.Body), "persistent archive payload"; got != want {
		t.Fatalf("unexpected restored object body: got %q want %q", got, want)
	}

	storagePath, err := ResolveStoragePath(config)
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	statePath := filepath.Join(storagePath, stateFileName)
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected persisted state file: %v", err)
	}
}

func TestServicePersistenceRejectsCorruptStateOnBootstrap(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	config := StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "broken-instance",
	}

	storagePath, err := ResolveStoragePath(config)
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatalf("mkdir storage path: %v", err)
	}
	statePath := filepath.Join(storagePath, stateFileName)
	if err := os.WriteFile(statePath, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}

	if _, err := NewWithPersistence(config); err == nil {
		t.Fatal("expected corrupt persisted state to fail bootstrap")
	}
}

func TestServiceCopyAndDeleteBehaviorSurviveRestart(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	config := StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "phase-13-object-core",
	}

	first, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("new with persistence: %v", err)
	}

	bucket, err := first.CreateBucket("mildstack-logs", "us-west-2")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if _, err := first.PutObject(bucket.Name, "archive.txt", []byte("archive payload"), "text/plain"); err != nil {
		t.Fatalf("put object: %v", err)
	}
	if _, err := first.CopyObject(bucket.Name, "archive-copy.txt", bucket.Name, "archive.txt"); err != nil {
		t.Fatalf("copy object: %v", err)
	}
	if err := first.DeleteObject(bucket.Name, "already-missing.txt"); err != nil {
		t.Fatalf("delete missing key before restart: %v", err)
	}

	second, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("new with persistence after restart: %v", err)
	}

	copied, err := second.GetObject(bucket.Name, "archive-copy.txt")
	if err != nil {
		t.Fatalf("get copied object after restart: %v", err)
	}
	if got, want := string(copied.Body), "archive payload"; got != want {
		t.Fatalf("unexpected copied body after restart: got %q want %q", got, want)
	}

	copied.Body[0] = 'A'
	again, err := second.GetObject(bucket.Name, "archive-copy.txt")
	if err != nil {
		t.Fatalf("get copied object again: %v", err)
	}
	if got, want := string(again.Body), "archive payload"; got != want {
		t.Fatalf("copied body was aliased after restart: got %q want %q", got, want)
	}

	if err := second.DeleteObject(bucket.Name, "already-missing.txt"); err != nil {
		t.Fatalf("delete missing key after restart: %v", err)
	}
}
