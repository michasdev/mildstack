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
}
