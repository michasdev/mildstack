package domain

import (
	"testing"
	"time"
)

func TestStateSnapshotCopiesLiveData(t *testing.T) {
	t.Helper()

	state := NewState()
	bucket := state.UpsertBucket("mildstack-archive", "us-west-2")
	state.UpsertObject(Object{
		Bucket:      bucket.Name,
		Key:         "manifest.txt",
		Size:        64,
		ContentType: "text/plain",
	})

	snapshot := state.Snapshot()

	buckets := snapshot["buckets"].([]any)
	buckets[0].(map[string]any)["name"] = "mutated"
	objects := snapshot["objects"].([]any)
	objects[0].(map[string]any)["content_type"] = "application/json"

	originalBucket, ok := state.Bucket("mildstack-assets")
	if !ok {
		t.Fatal("expected bootstrap bucket to remain present")
	}
	if got, want := originalBucket.Name, "mildstack-assets"; got != want {
		t.Fatalf("unexpected bucket name: got %q want %q", got, want)
	}
	originalObject, ok := state.Object(bucket.Name, "manifest.txt")
	if !ok {
		t.Fatal("expected bootstrap object to remain present")
	}
	if got, want := originalObject.ContentType, "text/plain"; got != want {
		t.Fatalf("unexpected object content type: got %q want %q", got, want)
	}
}

func TestStateMutationHelpersReturnCopiesAndUpdateState(t *testing.T) {
	t.Helper()

	state := NewState()

	buckets := state.ListBuckets()
	buckets[0].Name = "mutated"
	if got, want := state.Buckets[0].Name, "mildstack-assets"; got != want {
		t.Fatalf("bucket slice aliased live state: got %q want %q", got, want)
	}

	objects := state.ListObjects("mildstack-assets")
	objects[0].Key = "mutated"
	if got, want := state.Objects[0].Key, "bootstrap.txt"; got != want {
		t.Fatalf("object slice aliased live state: got %q want %q", got, want)
	}

	bucket := state.UpsertBucket("mildstack-logs", "us-west-2")
	if got, want := bucket.Region, "us-west-2"; got != want {
		t.Fatalf("unexpected bucket region: got %q want %q", got, want)
	}
	if !state.HasBucket("mildstack-logs") {
		t.Fatal("expected new bucket to be present")
	}

	object := state.UpsertObject(Object{
		Bucket:      bucket.Name,
		Key:         "audit.log",
		Size:        7,
		ContentType: "text/plain",
	})
	if got, want := object.Key, "audit.log"; got != want {
		t.Fatalf("unexpected object key: got %q want %q", got, want)
	}
	if !state.HasObject(bucket.Name, "audit.log") {
		t.Fatal("expected new object to be present")
	}

	if deleted := state.DeleteObject(bucket.Name, "audit.log"); !deleted {
		t.Fatal("expected object delete to report success")
	}
	if state.HasObject(bucket.Name, "audit.log") {
		t.Fatal("expected deleted object to be removed")
	}
}

func TestStateListBucketsReturnsSortedCopiesWithCreationTimestamps(t *testing.T) {
	t.Helper()

	state := NewState()
	createdAt := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)
	state.UpsertBucket(Bucket{
		Name:      "zeta-assets",
		Region:    "us-west-2",
		CreatedAt: createdAt,
	})
	state.UpsertBucket(Bucket{
		Name:      "alpha-assets",
		Region:    "sa-east-1",
		CreatedAt: createdAt.Add(time.Minute),
	})

	buckets := state.ListBuckets()
	if got, want := len(buckets), 3; got != want {
		t.Fatalf("unexpected bucket count: got %d want %d", got, want)
	}
	if got, want := buckets[0].Name, "alpha-assets"; got != want {
		t.Fatalf("unexpected first bucket: got %q want %q", got, want)
	}
	if got, want := buckets[2].Name, "zeta-assets"; got != want {
		t.Fatalf("unexpected last bucket: got %q want %q", got, want)
	}
	if buckets[0].CreatedAt.IsZero() {
		t.Fatal("expected listed buckets to include creation timestamp")
	}

	buckets[0].Name = "mutated"
	buckets[0].Region = "mutated"
	if got, want := state.Buckets[2].Name, "alpha-assets"; got != want {
		t.Fatalf("bucket list aliased live state: got %q want %q", got, want)
	}
}

func TestStateDeleteBucketRequiresEmptyBucket(t *testing.T) {
	t.Helper()

	state := NewState()

	if deleted := state.DeleteBucket("mildstack-assets"); deleted {
		t.Fatal("expected non-empty bootstrap bucket delete to fail")
	}

	state.UpsertBucket(Bucket{Name: "empty-bucket", Region: "us-east-1", CreatedAt: time.Now().UTC()})
	if deleted := state.DeleteBucket("empty-bucket"); !deleted {
		t.Fatal("expected empty bucket delete to succeed")
	}
	if state.HasBucket("empty-bucket") {
		t.Fatal("expected deleted bucket to be removed")
	}
}
