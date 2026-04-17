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

	putResp, err := handlers.PutObject(infrastructure.PutObjectRequest{
		Bucket:      createResp.Bucket.Name,
		Key:         "archive.txt",
		Size:        42,
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
	if _, err := handlers.GetObject(infrastructure.GetObjectRequest{
		Bucket: createResp.Bucket.Name,
		Key:    putResp.Object.Key,
	}); err == nil {
		t.Fatal("expected deleted object lookup to fail")
	}
}

func TestHandlersSurfaceServiceErrors(t *testing.T) {
	t.Helper()

	handlers := infrastructure.NewHandlers(application.New())

	if _, err := handlers.CreateBucket(infrastructure.CreateBucketRequest{}); err == nil {
		t.Fatal("expected empty bucket creation to fail")
	}
	if _, err := handlers.ListObjects(infrastructure.ListObjectsRequest{Bucket: "missing"}); err == nil {
		t.Fatal("expected missing bucket listing to fail")
	}
	if _, err := handlers.GetObject(infrastructure.GetObjectRequest{Bucket: "missing", Key: "key"}); err == nil {
		t.Fatal("expected missing object lookup to fail")
	}
	if _, err := handlers.PutObject(infrastructure.PutObjectRequest{Bucket: "missing", Key: "key", Size: 1}); err == nil {
		t.Fatal("expected put on missing bucket to fail")
	}
	if _, err := handlers.DeleteObject(infrastructure.DeleteObjectRequest{Bucket: "missing", Key: "key"}); err == nil {
		t.Fatal("expected delete on missing object to fail")
	}
}
