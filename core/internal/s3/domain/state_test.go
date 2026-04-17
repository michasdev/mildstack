package domain

import "testing"

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
