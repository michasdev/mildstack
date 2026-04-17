package infrastructure_test

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/s3/application"
	"github.com/michasdev/mildstack/core/internal/s3/infrastructure"
)

func TestHandlersDriveRealServiceAndReturnCopies(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	buckets := handlers.ListBuckets()
	if got, want := len(buckets.Buckets), 1; got != want {
		t.Fatalf("unexpected initial bucket count: got %d want %d", got, want)
	}
	if buckets.Buckets[0].CreatedAt.IsZero() {
		t.Fatal("expected bucket payload to include created_at")
	}
	buckets.Buckets[0].Name = "mutated"
	again := handlers.ListBuckets()
	if got, want := again.Buckets[0].Name, "mildstack-assets"; got != want {
		t.Fatalf("bucket payload was not copied: got %q want %q", got, want)
	}

	createResp, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{
		Name:   "mildstack-logs",
		Region: "us-west-2",
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if got, want := createResp.Bucket.Name, "mildstack-logs"; got != want {
		t.Fatalf("unexpected bucket name: got %q want %q", got, want)
	}
	if createResp.Bucket.CreatedAt.IsZero() {
		t.Fatal("expected create bucket response to include created_at")
	}

	headResp, err := handlers.HeadBucket(infrastructure.HeadBucketRequest{Name: createResp.Bucket.Name})
	if err != nil {
		t.Fatalf("head bucket: %v", err)
	}
	if got, want := headResp.Bucket.Region, "us-west-2"; got != want {
		t.Fatalf("unexpected head bucket region: got %q want %q", got, want)
	}

	putResp, err := handlers.PutObject(infrastructure.PutObjectRequest{
		Bucket:      createResp.Bucket.Name,
		Key:         "archive.txt",
		Body:        []byte("archive payload"),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}
	if got, want := putResp.Object.Key, "archive.txt"; got != want {
		t.Fatalf("unexpected object key: got %q want %q", got, want)
	}

	listResp, err := handlers.ListObjects(infrastructure.ListObjectsRequest{Bucket: createResp.Bucket.Name})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if got, want := len(listResp.Objects), 1; got != want {
		t.Fatalf("unexpected object count: got %d want %d", got, want)
	}
	listResp.Objects[0].Key = "mutated"
	againObjects, err := handlers.GetObject(infrastructure.GetObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    putResp.Object.Key,
	})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if got, want := againObjects.Object.Key, "archive.txt"; got != want {
		t.Fatalf("object payload was not copied: got %q want %q", got, want)
	}
	if got, want := string(againObjects.Object.Body), "archive payload"; got != want {
		t.Fatalf("unexpected object body: got %q want %q", got, want)
	}

	headObjectResp, err := handlers.HeadObject(infrastructure.HeadObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    putResp.Object.Key,
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}
	if got, want := headObjectResp.Object.ETag, putResp.Object.ETag; got != want {
		t.Fatalf("unexpected head object etag: got %q want %q", got, want)
	}
	if len(headObjectResp.Object.Body) != 0 {
		t.Fatalf("expected head payload body to be empty, got %d bytes", len(headObjectResp.Object.Body))
	}

	copyResp, err := handlers.CopyObject(infrastructure.CopyObjectRequest{
		Bucket:          createResp.Bucket.Name,
		Key:             "archive-copy.txt",
		SourceBucket:    createResp.Bucket.Name,
		SourceObjectKey: putResp.Object.Key,
	})
	if err != nil {
		t.Fatalf("copy object: %v", err)
	}
	if got, want := copyResp.Object.Key, "archive-copy.txt"; got != want {
		t.Fatalf("unexpected copied key: got %q want %q", got, want)
	}
	if got, want := copyResp.Object.ETag, putResp.Object.ETag; got != want {
		t.Fatalf("unexpected copied etag: got %q want %q", got, want)
	}
	if got, want := string(copyResp.Object.Body), "archive payload"; got != want {
		t.Fatalf("unexpected copied body: got %q want %q", got, want)
	}

	deleteResp, err := handlers.DeleteObject(infrastructure.DeleteObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    putResp.Object.Key,
	})
	if err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if !deleteResp.Deleted {
		t.Fatal("expected delete response to report success")
	}
	if _, err := handlers.DeleteObject(infrastructure.DeleteObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    putResp.Object.Key,
	}); err != nil {
		t.Fatalf("expected delete response to stay idempotent: %v", err)
	}
	if _, err := handlers.GetObject(infrastructure.GetObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    putResp.Object.Key,
	}); err == nil {
		t.Fatal("expected deleted object lookup to fail")
	}
	if _, err := handlers.DeleteObject(infrastructure.DeleteObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    copyResp.Object.Key,
	}); err != nil {
		t.Fatalf("delete copied object: %v", err)
	}

	if _, err := handlers.DeleteBucket(infrastructure.DeleteBucketRequest{Name: "mildstack-assets"}); err == nil {
		t.Fatal("expected non-empty bucket delete to fail")
	}

	deleteBucketResp, err := handlers.DeleteBucket(infrastructure.DeleteBucketRequest{Name: createResp.Bucket.Name})
	if err != nil {
		t.Fatalf("delete bucket: %v", err)
	}
	if !deleteBucketResp.Deleted {
		t.Fatal("expected delete bucket response to report success")
	}
}

func TestHandlersSurfaceServiceErrors(t *testing.T) {
	t.Helper()

	handlers := infrastructure.NewHandlers(application.New())

	if _, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{}); err == nil {
		t.Fatal("expected empty bucket creation to fail")
	}
	if _, err := handlers.HeadBucket(infrastructure.HeadBucketRequest{Name: "missing"}); err == nil {
		t.Fatal("expected missing bucket head to fail")
	}
	if _, err := handlers.DeleteBucket(infrastructure.DeleteBucketRequest{Name: "missing"}); err == nil {
		t.Fatal("expected missing bucket delete to fail")
	}
	if _, err := handlers.GetBucketNotification(infrastructure.GetBucketNotificationRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket notification lookup to fail")
	}
	if _, err := handlers.PutBucketNotification(infrastructure.PutBucketNotificationRequest{Bucket: "missing", Body: []byte("<NotificationConfiguration/>")}); err == nil {
		t.Fatal("expected missing bucket notification update to fail")
	}
	if _, err := handlers.GetBucketLogging(infrastructure.GetBucketLoggingRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket logging lookup to fail")
	}
	if _, err := handlers.PutBucketLogging(infrastructure.PutBucketLoggingRequest{Bucket: "missing", Body: []byte("<BucketLoggingStatus/>")}); err == nil {
		t.Fatal("expected missing bucket logging update to fail")
	}
	if _, err := handlers.GetBucketReplication(infrastructure.GetBucketReplicationRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket replication lookup to fail")
	}
	if _, err := handlers.PutBucketReplication(infrastructure.PutBucketReplicationRequest{Bucket: "missing", Body: []byte("<ReplicationConfiguration/>")}); err == nil {
		t.Fatal("expected missing bucket replication update to fail")
	}
	if _, err := handlers.DeleteBucketReplication(infrastructure.DeleteBucketReplicationRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket replication delete to fail")
	}
	if _, err := handlers.GetBucketVersioning(infrastructure.GetBucketVersioningRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket versioning lookup to fail")
	}
	if _, err := handlers.PutBucketVersioning(infrastructure.PutBucketVersioningRequest{Bucket: "missing", Status: "Enabled"}); err == nil {
		t.Fatal("expected missing bucket versioning update to fail")
	}
	if _, err := handlers.ListObjects(infrastructure.ListObjectsRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket listing to fail")
	}
	if _, err := handlers.GetObject(infrastructure.GetObjectRequest{Bucket: "missing", Key: "key"}); err == nil {
		t.Fatal("expected missing object lookup to fail")
	}
	if _, err := handlers.HeadObject(infrastructure.HeadObjectRequest{Bucket: "missing", Key: "key"}); err == nil {
		t.Fatal("expected missing object head to fail")
	}
	if _, err := handlers.CopyObject(infrastructure.CopyObjectRequest{
		Bucket: "missing", Key: "copy", SourceBucket: "missing", SourceObjectKey: "key",
	}); err == nil {
		t.Fatal("expected missing object copy to fail")
	}
	if _, err := handlers.PutObject(infrastructure.PutObjectRequest{Bucket: "missing", Key: "key", Body: []byte("x")}); err == nil {
		t.Fatal("expected put on missing bucket to fail")
	}
	if _, err := handlers.DeleteObject(infrastructure.DeleteObjectRequest{Bucket: "mildstack-assets", Key: "missing"}); err != nil {
		t.Fatalf("expected delete on missing key to succeed: %v", err)
	}
	if _, err := handlers.CreateMultipartUpload(infrastructure.CreateMultipartUploadRequest{Bucket: "missing", Key: "key"}); err == nil {
		t.Fatal("expected multipart create on missing bucket to fail")
	}
	if _, err := handlers.UploadPart(infrastructure.UploadPartRequest{UploadID: "missing", PartNumber: 1, Body: []byte("x")}); err == nil {
		t.Fatal("expected upload part on missing upload to fail")
	}
	if _, err := handlers.CompleteMultipartUpload(infrastructure.CompleteMultipartUploadRequest{UploadID: "missing"}); err == nil {
		t.Fatal("expected multipart complete on missing upload to fail")
	}
	if _, err := handlers.AbortMultipartUpload(infrastructure.AbortMultipartUploadRequest{UploadID: "missing"}); err == nil {
		t.Fatal("expected multipart abort on missing upload to fail")
	}
}

func TestHandlersCopyResponsesDoNotAliasStoredBodies(t *testing.T) {
	t.Helper()

	handlers := infrastructure.NewHandlers(application.New())

	createResp, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{
		Name:   "mildstack-logs",
		Region: "us-west-2",
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	if _, err := handlers.PutObject(infrastructure.PutObjectRequest{
		Bucket:      createResp.Bucket.Name,
		Key:         "archive.txt",
		Body:        []byte("archive payload"),
		ContentType: "text/plain",
	}); err != nil {
		t.Fatalf("put object: %v", err)
	}

	copyResp, err := handlers.CopyObject(infrastructure.CopyObjectRequest{
		Bucket:          createResp.Bucket.Name,
		Key:             "archive-copy.txt",
		SourceBucket:    createResp.Bucket.Name,
		SourceObjectKey: "archive.txt",
	})
	if err != nil {
		t.Fatalf("copy object: %v", err)
	}

	copyResp.Object.Body[0] = 'A'

	again, err := handlers.GetObject(infrastructure.GetObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    "archive-copy.txt",
	})
	if err != nil {
		t.Fatalf("get copied object: %v", err)
	}
	if got, want := string(again.Object.Body), "archive payload"; got != want {
		t.Fatalf("copied response body was aliased: got %q want %q", got, want)
	}
}

func TestHandlersExposeListVariantsAndBatchDeleteDeterministically(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	createResp, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{
		Name:   "catalog-bucket",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	for _, key := range []string{"charlie.txt", "alpha.txt", "bravo.txt"} {
		if _, err := handlers.PutObject(infrastructure.PutObjectRequest{
			Bucket:      createResp.Bucket.Name,
			Key:         key,
			Body:        []byte(key),
			ContentType: "text/plain",
		}); err != nil {
			t.Fatalf("put object %q: %v", key, err)
		}
	}

	v1, err := handlers.ListObjectsV1(infrastructure.ListObjectsV1Request{
		Bucket:  createResp.Bucket.Name,
		MaxKeys: 2,
	})
	if err != nil {
		t.Fatalf("list objects v1: %v", err)
	}
	if got, want := len(v1.Objects), 2; got != want {
		t.Fatalf("unexpected v1 object count: got %d want %d", got, want)
	}
	if got, want := v1.Objects[0].Key, "alpha.txt"; got != want {
		t.Fatalf("unexpected v1 first key: got %q want %q", got, want)
	}
	v1.Objects[0].Key = "mutated"

	v1Again, err := handlers.ListObjectsV1(infrastructure.ListObjectsV1Request{
		Bucket: createResp.Bucket.Name,
	})
	if err != nil {
		t.Fatalf("list objects v1 again: %v", err)
	}
	if got, want := v1Again.Objects[0].Key, "alpha.txt"; got != want {
		t.Fatalf("v1 payload was aliased: got %q want %q", got, want)
	}

	v2, err := handlers.ListObjectsV2(infrastructure.ListObjectsV2Request{
		Bucket:  createResp.Bucket.Name,
		MaxKeys: 2,
	})
	if err != nil {
		t.Fatalf("list objects v2: %v", err)
	}
	if !v2.IsTruncated {
		t.Fatal("expected v2 response to be truncated")
	}
	if v2.NextContinuationToken == "" {
		t.Fatal("expected continuation token")
	}

	deleteResp, err := handlers.DeleteObjects(infrastructure.DeleteObjectsRequest{
		Bucket: createResp.Bucket.Name,
		Keys:   []string{"missing.txt", "bravo.txt", "alpha.txt"},
	})
	if err != nil {
		t.Fatalf("delete objects: %v", err)
	}
	if got, want := len(deleteResp.Deleted), 3; got != want {
		t.Fatalf("unexpected delete count: got %d want %d", got, want)
	}
	if got, want := deleteResp.Deleted[0].Key, "missing.txt"; got != want {
		t.Fatalf("unexpected first deleted key: got %q want %q", got, want)
	}
	if got, want := deleteResp.Deleted[1].Key, "bravo.txt"; got != want {
		t.Fatalf("unexpected second deleted key: got %q want %q", got, want)
	}
	if got, want := deleteResp.Deleted[2].Key, "alpha.txt"; got != want {
		t.Fatalf("unexpected third deleted key: got %q want %q", got, want)
	}

	remaining, err := handlers.ListObjectsV1(infrastructure.ListObjectsV1Request{
		Bucket: createResp.Bucket.Name,
	})
	if err != nil {
		t.Fatalf("list remaining objects: %v", err)
	}
	if got, want := len(remaining.Objects), 1; got != want {
		t.Fatalf("unexpected remaining object count: got %d want %d", got, want)
	}
	if got, want := remaining.Objects[0].Key, "charlie.txt"; got != want {
		t.Fatalf("unexpected remaining key: got %q want %q", got, want)
	}
}

func TestHandlersVersioningLifecycleIsCopySafe(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	createResp, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{
		Name:   "mildstack-versioned",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	if _, err := handlers.PutBucketVersioning(infrastructure.PutBucketVersioningRequest{
		Bucket: createResp.Bucket.Name,
		Status: "Enabled",
	}); err != nil {
		t.Fatalf("enable versioning: %v", err)
	}

	versioningResp, err := handlers.GetBucketVersioning(infrastructure.GetBucketVersioningRequest{
		Bucket: createResp.Bucket.Name,
	})
	if err != nil {
		t.Fatalf("get versioning: %v", err)
	}
	if got, want := versioningResp.Versioning.Status, "Enabled"; got != want {
		t.Fatalf("unexpected versioning status: got %q want %q", got, want)
	}

	if _, err := handlers.PutObject(infrastructure.PutObjectRequest{
		Bucket:      createResp.Bucket.Name,
		Key:         "release.txt",
		Body:        []byte("v1"),
		ContentType: "text/plain",
	}); err != nil {
		t.Fatalf("put first version: %v", err)
	}
	if _, err := handlers.PutObject(infrastructure.PutObjectRequest{
		Bucket:      createResp.Bucket.Name,
		Key:         "release.txt",
		Body:        []byte("v2"),
		ContentType: "text/plain",
	}); err != nil {
		t.Fatalf("put second version: %v", err)
	}
	if _, err := handlers.DeleteObject(infrastructure.DeleteObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    "release.txt",
	}); err != nil {
		t.Fatalf("delete versioned object: %v", err)
	}

	versions, err := handlers.ListObjectVersions(infrastructure.ListObjectVersionsRequest{
		Bucket: createResp.Bucket.Name,
	})
	if err != nil {
		t.Fatalf("list object versions: %v", err)
	}
	if got, want := len(versions.Versions), 3; got != want {
		t.Fatalf("unexpected version count: got %d want %d", got, want)
	}
	if !versions.Versions[0].IsDeleteMarker || !versions.Versions[0].IsLatest {
		t.Fatal("expected latest version entry to be a delete marker")
	}
	if got, want := versions.Versions[1].ContentType, "text/plain"; got != want {
		t.Fatalf("unexpected second version content type: got %q want %q", got, want)
	}
	versions.Versions[1].ContentType = "mutated"
	again, err := handlers.ListObjectVersions(infrastructure.ListObjectVersionsRequest{
		Bucket: createResp.Bucket.Name,
	})
	if err != nil {
		t.Fatalf("list object versions again: %v", err)
	}
	if got, want := again.Versions[1].ContentType, "text/plain"; got != want {
		t.Fatalf("version payload was not copied: got %q want %q", got, want)
	}
}

func TestHandlersMultipartLifecycleIsCopySafe(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	createResp, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{
		Name:   "mildstack-multipart",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	uploadResp, err := handlers.CreateMultipartUpload(infrastructure.CreateMultipartUploadRequest{
		Bucket:      createResp.Bucket.Name,
		Key:         "archive.bin",
		ContentType: "application/octet-stream",
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	if got, want := uploadResp.Upload.ContentType, "application/octet-stream"; got != want {
		t.Fatalf("unexpected multipart content type: got %q want %q", got, want)
	}

	secondPart, err := handlers.UploadPart(infrastructure.UploadPartRequest{
		UploadID:   uploadResp.Upload.UploadID,
		PartNumber: 2,
		Body:       []byte("two"),
	})
	if err != nil {
		t.Fatalf("upload second part: %v", err)
	}
	if got, want := secondPart.Part.PartNumber, 2; got != want {
		t.Fatalf("unexpected second part number: got %d want %d", got, want)
	}

	firstPartBody := []byte("one")
	if _, err := handlers.UploadPart(infrastructure.UploadPartRequest{
		UploadID:   uploadResp.Upload.UploadID,
		PartNumber: 1,
		Body:       firstPartBody,
	}); err != nil {
		t.Fatalf("upload first part: %v", err)
	}
	firstPartBody[0] = 'X'

	completeResp, err := handlers.CompleteMultipartUpload(infrastructure.CompleteMultipartUploadRequest{
		UploadID: uploadResp.Upload.UploadID,
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}
	if got, want := string(completeResp.Object.Body), "onetwo"; got != want {
		t.Fatalf("unexpected assembled multipart body: got %q want %q", got, want)
	}
	if got, want := completeResp.Object.Size, int64(len("onetwo")); got != want {
		t.Fatalf("unexpected assembled multipart size: got %d want %d", got, want)
	}
	completeResp.Object.Body[0] = 'O'
	againObject, err := handlers.GetObject(infrastructure.GetObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    "archive.bin",
	})
	if err != nil {
		t.Fatalf("get completed object: %v", err)
	}
	if got, want := string(againObject.Object.Body), "onetwo"; got != want {
		t.Fatalf("multipart completion response was aliased: got %q want %q", got, want)
	}

	abortedResp, err := handlers.CreateMultipartUpload(infrastructure.CreateMultipartUploadRequest{
		Bucket: createResp.Bucket.Name,
		Key:    "aborted.bin",
	})
	if err != nil {
		t.Fatalf("create multipart upload for abort: %v", err)
	}
	if _, err := handlers.UploadPart(infrastructure.UploadPartRequest{
		UploadID:   abortedResp.Upload.UploadID,
		PartNumber: 1,
		Body:       []byte("abort"),
	}); err != nil {
		t.Fatalf("upload aborted part: %v", err)
	}
	if _, err := handlers.AbortMultipartUpload(infrastructure.AbortMultipartUploadRequest{
		UploadID: abortedResp.Upload.UploadID,
	}); err != nil {
		t.Fatalf("abort multipart upload: %v", err)
	}
	if _, err := handlers.CompleteMultipartUpload(infrastructure.CompleteMultipartUploadRequest{
		UploadID: abortedResp.Upload.UploadID,
	}); err == nil {
		t.Fatal("expected completing an aborted upload to fail")
	}
}

func TestHandlersBucketGovernanceLifecycleIsCopySafe(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	createResp, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{
		Name:   "mildstack-governed",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
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
		func(body []byte) ([]byte, error) {
			resp, err := handlers.PutBucketPolicy(infrastructure.PutBucketPolicyRequest{Bucket: createResp.Bucket.Name, Body: body})
			if err != nil {
				return nil, err
			}
			return resp.Policy.Body, nil
		},
		func() ([]byte, error) {
			resp, err := handlers.GetBucketPolicy(infrastructure.GetBucketPolicyRequest{Bucket: createResp.Bucket.Name})
			if err != nil {
				return nil, err
			}
			return resp.Policy.Body, nil
		},
		`{"Version":"2012-10-17"}`,
	)
	assertRoundTrip("encryption",
		func(body []byte) ([]byte, error) {
			resp, err := handlers.PutBucketEncryption(infrastructure.PutBucketEncryptionRequest{Bucket: createResp.Bucket.Name, Body: body})
			if err != nil {
				return nil, err
			}
			return resp.Encryption.Body, nil
		},
		func() ([]byte, error) {
			resp, err := handlers.GetBucketEncryption(infrastructure.GetBucketEncryptionRequest{Bucket: createResp.Bucket.Name})
			if err != nil {
				return nil, err
			}
			return resp.Encryption.Body, nil
		},
		"<ServerSideEncryptionConfiguration/>",
	)
	assertRoundTrip("lifecycle",
		func(body []byte) ([]byte, error) {
			resp, err := handlers.PutBucketLifecycle(infrastructure.PutBucketLifecycleRequest{Bucket: createResp.Bucket.Name, Body: body})
			if err != nil {
				return nil, err
			}
			return resp.Lifecycle.Body, nil
		},
		func() ([]byte, error) {
			resp, err := handlers.GetBucketLifecycle(infrastructure.GetBucketLifecycleRequest{Bucket: createResp.Bucket.Name})
			if err != nil {
				return nil, err
			}
			return resp.Lifecycle.Body, nil
		},
		"<LifecycleConfiguration/>",
	)
	assertRoundTrip("cors",
		func(body []byte) ([]byte, error) {
			resp, err := handlers.PutBucketCORS(infrastructure.PutBucketCORSRequest{Bucket: createResp.Bucket.Name, Body: body})
			if err != nil {
				return nil, err
			}
			return resp.CORS.Body, nil
		},
		func() ([]byte, error) {
			resp, err := handlers.GetBucketCORS(infrastructure.GetBucketCORSRequest{Bucket: createResp.Bucket.Name})
			if err != nil {
				return nil, err
			}
			return resp.CORS.Body, nil
		},
		"<CORSConfiguration/>",
	)
	assertRoundTrip("tagging",
		func(body []byte) ([]byte, error) {
			resp, err := handlers.PutBucketTagging(infrastructure.PutBucketTaggingRequest{Bucket: createResp.Bucket.Name, Body: body})
			if err != nil {
				return nil, err
			}
			return resp.Tagging.Body, nil
		},
		func() ([]byte, error) {
			resp, err := handlers.GetBucketTagging(infrastructure.GetBucketTaggingRequest{Bucket: createResp.Bucket.Name})
			if err != nil {
				return nil, err
			}
			return resp.Tagging.Body, nil
		},
		"<Tagging><TagSet><Tag><Key>env</Key><Value>dev</Value></Tag></TagSet></Tagging>",
	)

	aclDefault, err := handlers.GetBucketACL(infrastructure.GetBucketACLRequest{Bucket: createResp.Bucket.Name})
	if err != nil {
		t.Fatalf("get default acl: %v", err)
	}
	const expectedDefaultACL = `<?xml version="1.0" encoding="UTF-8"?>
<AccessControlPolicy xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner>
    <ID>owner-id</ID>
    <DisplayName>mildstack</DisplayName>
  </Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser">
        <ID>owner-id</ID>
        <DisplayName>mildstack</DisplayName>
      </Grantee>
      <Permission>FULL_CONTROL</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`
	if got, want := string(aclDefault.ACL.Body), expectedDefaultACL; got != want {
		t.Fatalf("unexpected default ACL body: got %q want %q", got, want)
	}

	aclResp, err := handlers.PutBucketACL(infrastructure.PutBucketACLRequest{
		Bucket: createResp.Bucket.Name,
		Body:   []byte("<AccessControlPolicy><Owner><ID>owner</ID></Owner></AccessControlPolicy>"),
	})
	if err != nil {
		t.Fatalf("put acl: %v", err)
	}
	aclResp.ACL.Body[0] = 'X'

	againACL, err := handlers.GetBucketACL(infrastructure.GetBucketACLRequest{Bucket: createResp.Bucket.Name})
	if err != nil {
		t.Fatalf("get stored acl: %v", err)
	}
	if got, want := string(againACL.ACL.Body), "<AccessControlPolicy><Owner><ID>owner</ID></Owner></AccessControlPolicy>"; got != want {
		t.Fatalf("unexpected stored ACL body: got %q want %q", got, want)
	}

	if _, err := handlers.DeleteBucketPolicy(infrastructure.DeleteBucketPolicyRequest{Bucket: createResp.Bucket.Name}); err != nil {
		t.Fatalf("delete policy: %v", err)
	}
	if _, err := handlers.DeleteBucketEncryption(infrastructure.DeleteBucketEncryptionRequest{Bucket: createResp.Bucket.Name}); err != nil {
		t.Fatalf("delete encryption: %v", err)
	}
	if _, err := handlers.DeleteBucketLifecycle(infrastructure.DeleteBucketLifecycleRequest{Bucket: createResp.Bucket.Name}); err != nil {
		t.Fatalf("delete lifecycle: %v", err)
	}
	if _, err := handlers.DeleteBucketCORS(infrastructure.DeleteBucketCORSRequest{Bucket: createResp.Bucket.Name}); err != nil {
		t.Fatalf("delete cors: %v", err)
	}
	if _, err := handlers.DeleteBucketTagging(infrastructure.DeleteBucketTaggingRequest{Bucket: createResp.Bucket.Name}); err != nil {
		t.Fatalf("delete tagging: %v", err)
	}

	if _, err := handlers.DeleteBucket(infrastructure.DeleteBucketRequest{Name: createResp.Bucket.Name}); err != nil {
		t.Fatalf("delete governed bucket: %v", err)
	}
}
