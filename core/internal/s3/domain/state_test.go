package domain

import (
	"testing"
	"time"
)

func TestStateSnapshotCopiesLiveData(t *testing.T) {
	t.Helper()

	state := NewState()
	bucket := state.UpsertBucket(Bucket{Name: "mildstack-archive", Region: "us-west-2"})
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

	bucket := state.UpsertBucket(Bucket{Name: "mildstack-logs", Region: "us-west-2"})
	if got, want := bucket.Region, "us-west-2"; got != want {
		t.Fatalf("unexpected bucket region: got %q want %q", got, want)
	}
	if !state.HasBucket("mildstack-logs") {
		t.Fatal("expected new bucket to be present")
	}

	object := state.UpsertObject(Object{
		Bucket:      bucket.Name,
		Key:         "audit.log",
		Body:        []byte("payload"),
		Size:        7,
		ContentType: "text/plain",
	})
	if got, want := object.Key, "audit.log"; got != want {
		t.Fatalf("unexpected object key: got %q want %q", got, want)
	}
	if got, want := string(object.Body), "payload"; got != want {
		t.Fatalf("unexpected object body: got %q want %q", got, want)
	}
	if !state.HasObject(bucket.Name, "audit.log") {
		t.Fatal("expected new object to be present")
	}

	object.Body[0] = 'P'
	fetched, ok := state.Object(bucket.Name, "audit.log")
	if !ok {
		t.Fatal("expected object to remain present")
	}
	if got, want := string(fetched.Body), "payload"; got != want {
		t.Fatalf("stored object body was aliased: got %q want %q", got, want)
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

func TestStateObjectReturnsCopyOfStoredBody(t *testing.T) {
	t.Helper()

	state := NewState()
	state.UpsertObject(Object{
		Bucket:      "mildstack-assets",
		Key:         "copy-safe.txt",
		Body:        []byte("copy-safe"),
		Size:        int64(len("copy-safe")),
		ContentType: "text/plain",
	})

	object, ok := state.Object("mildstack-assets", "copy-safe.txt")
	if !ok {
		t.Fatal("expected object to be present")
	}

	object.Body[0] = 'C'

	again, ok := state.Object("mildstack-assets", "copy-safe.txt")
	if !ok {
		t.Fatal("expected object to remain present")
	}
	if got, want := string(again.Body), "copy-safe"; got != want {
		t.Fatalf("returned object body was aliased: got %q want %q", got, want)
	}
}

func TestStateListObjectsReturnsCopySafeBodies(t *testing.T) {
	t.Helper()

	state := NewState()
	state.UpsertObject(Object{
		Bucket:      "mildstack-assets",
		Key:         "listed.txt",
		Body:        []byte("listed-body"),
		Size:        int64(len("listed-body")),
		ContentType: "text/plain",
	})

	objects := state.ListObjects("mildstack-assets")
	if got, want := len(objects), 2; got != want {
		t.Fatalf("unexpected object count: got %d want %d", got, want)
	}

	for i := range objects {
		if objects[i].Key == "listed.txt" {
			objects[i].Body[0] = 'L'
		}
	}

	again, ok := state.Object("mildstack-assets", "listed.txt")
	if !ok {
		t.Fatal("expected listed object to remain present")
	}
	if got, want := string(again.Body), "listed-body"; got != want {
		t.Fatalf("listed object body was aliased: got %q want %q", got, want)
	}
}

func TestStateListObjectPageUsesDeterministicDelimiterPagination(t *testing.T) {
	t.Helper()

	state := NewState()
	state.UpsertBucket(Bucket{Name: "catalog-bucket", Region: "us-east-1"})
	state.UpsertObject(Object{Bucket: "catalog-bucket", Key: "alpha.txt", Body: []byte("a"), ContentType: "text/plain"})
	state.UpsertObject(Object{Bucket: "catalog-bucket", Key: "photos/2026/01.jpg", Body: []byte("b"), ContentType: "image/jpeg"})
	state.UpsertObject(Object{Bucket: "catalog-bucket", Key: "photos/2027/02.jpg", Body: []byte("c"), ContentType: "image/jpeg"})
	state.UpsertObject(Object{Bucket: "catalog-bucket", Key: "zeta.txt", Body: []byte("d"), ContentType: "text/plain"})

	page := state.ListObjectPage("catalog-bucket", ListObjectsOptions{
		Delimiter: "/",
		MaxKeys:   2,
	})

	if got, want := len(page.Objects), 1; got != want {
		t.Fatalf("unexpected object count: got %d want %d", got, want)
	}
	if got, want := page.Objects[0].Key, "alpha.txt"; got != want {
		t.Fatalf("unexpected first object key: got %q want %q", got, want)
	}
	if got, want := len(page.CommonPrefixes), 1; got != want {
		t.Fatalf("unexpected common prefix count: got %d want %d", got, want)
	}
	if got, want := page.CommonPrefixes[0], "photos/"; got != want {
		t.Fatalf("unexpected common prefix: got %q want %q", got, want)
	}
	if !page.IsTruncated {
		t.Fatal("expected page to be truncated")
	}
	if got, want := page.NextMarker, "photos/2027/02.jpg"; got != want {
		t.Fatalf("unexpected next marker: got %q want %q", got, want)
	}

	page.Objects[0].Key = "mutated"
	page.CommonPrefixes[0] = "mutated/"

	again := state.ListObjectPage("catalog-bucket", ListObjectsOptions{
		Delimiter: "/",
		MaxKeys:   2,
	})
	if got, want := again.Objects[0].Key, "alpha.txt"; got != want {
		t.Fatalf("object page aliased live state: got %q want %q", got, want)
	}
	if got, want := again.CommonPrefixes[0], "photos/"; got != want {
		t.Fatalf("common prefixes aliased live state: got %q want %q", got, want)
	}
}

func TestStateVersionHistoryIsCopySafeAndDeterministic(t *testing.T) {
	t.Helper()

	state := NewState()
	bucket := state.UpsertBucket(Bucket{Name: "versioned-bucket", Region: "us-west-2"})
	state.SetBucketVersioning(bucket.Name, VersioningEnabled)

	first := state.UpsertObject(Object{
		Bucket:      bucket.Name,
		Key:         "release.txt",
		Body:        []byte("v1"),
		Size:        2,
		ContentType: "text/plain",
	})
	state.RecordObjectVersion(first)
	second := state.UpsertObject(Object{
		Bucket:      bucket.Name,
		Key:         "release.txt",
		Body:        []byte("v2"),
		Size:        2,
		ContentType: "text/plain",
	})
	state.RecordObjectVersion(second)
	state.DeleteObject(bucket.Name, "release.txt")
	state.RecordDeleteMarker(bucket.Name, "release.txt")

	versions := state.ListObjectVersions(bucket.Name)
	if got, want := len(versions), 3; got != want {
		t.Fatalf("unexpected version count: got %d want %d", got, want)
	}
	if !versions[0].IsDeleteMarker {
		t.Fatal("expected latest version entry to be a delete marker")
	}
	if got, want := string(versions[1].Body), "v2"; got != want {
		t.Fatalf("unexpected second version body: got %q want %q", got, want)
	}
	if got, want := string(versions[2].Body), "v1"; got != want {
		t.Fatalf("unexpected first version body: got %q want %q", got, want)
	}

	versions[1].Body[0] = 'X'
	again := state.ListObjectVersions(bucket.Name)
	if got, want := string(again[1].Body), "v2"; got != want {
		t.Fatalf("version history was aliased: got %q want %q", got, want)
	}

	if deleted := state.DeleteBucket(bucket.Name); deleted {
		t.Fatal("expected versioned bucket delete to fail while version history exists")
	}
}

func TestStateBucketDeleteClearsGovernanceState(t *testing.T) {
	t.Helper()

	state := NewState()
	bucket := state.UpsertBucket(Bucket{Name: "governed-bucket", Region: "us-west-2"})
	state.SetBucketPolicy(bucket.Name, []byte(`{"statement":"allow"}`))
	state.SetBucketEncryptionConfig(bucket.Name, []byte("<EncryptionConfiguration/>"))
	state.SetBucketLifecycleConfig(bucket.Name, []byte("<LifecycleConfiguration/>"))
	state.SetBucketCORSConfig(bucket.Name, []byte("<CORSConfiguration/>"))
	state.SetBucketACLConfig(bucket.Name, []byte("<AccessControlPolicy/>"))
	state.SetBucketTaggingConfig(bucket.Name, []byte("<Tagging/>"))
	state.SetBucketNotification(bucket.Name, []byte("<NotificationConfiguration/>"))
	state.SetBucketLoggingConfig(bucket.Name, []byte("<BucketLoggingStatus/>"))
	state.SetBucketReplicationConfig(bucket.Name, BucketReplicationConfig{
		Role: "arn:aws:iam::123456789012:role/replication",
		Rules: []BucketReplicationRule{
			{
				ID:     "rule-1",
				Status: "Enabled",
			},
		},
	})

	if deleted := state.DeleteBucket(bucket.Name); !deleted {
		t.Fatal("expected governed bucket delete to succeed")
	}
	if _, ok := state.BucketPolicy(bucket.Name); ok {
		t.Fatal("expected bucket policy to be cleared on delete")
	}
	if _, ok := state.BucketEncryptionConfig(bucket.Name); ok {
		t.Fatal("expected bucket encryption to be cleared on delete")
	}
	if _, ok := state.BucketLifecycleConfig(bucket.Name); ok {
		t.Fatal("expected bucket lifecycle to be cleared on delete")
	}
	if _, ok := state.BucketCORSConfig(bucket.Name); ok {
		t.Fatal("expected bucket CORS to be cleared on delete")
	}
	if _, ok := state.BucketACLConfig(bucket.Name); ok {
		t.Fatal("expected bucket ACL to be cleared on delete")
	}
	if _, ok := state.BucketTaggingConfig(bucket.Name); ok {
		t.Fatal("expected bucket tagging to be cleared on delete")
	}
	if _, ok := state.BucketNotification(bucket.Name); ok {
		t.Fatal("expected bucket notification to be cleared on delete")
	}
	if _, ok := state.BucketLoggingConfig(bucket.Name); ok {
		t.Fatal("expected bucket logging to be cleared on delete")
	}
	if _, ok := state.BucketReplicationConfig(bucket.Name); ok {
		t.Fatal("expected bucket replication to be cleared on delete")
	}
}

func TestStateSnapshotCopiesGovernanceData(t *testing.T) {
	t.Helper()

	state := NewState()
	bucket := state.UpsertBucket(Bucket{Name: "snapshot-governance", Region: "us-east-1"})
	state.SetBucketPolicy(bucket.Name, []byte(`{"statement":"allow"}`))
	state.SetBucketACLConfig(bucket.Name, []byte("<AccessControlPolicy/>"))
	state.SetBucketNotification(bucket.Name, []byte("<NotificationConfiguration/>"))
	state.SetBucketLoggingConfig(bucket.Name, []byte("<BucketLoggingStatus/>"))
	state.SetBucketReplicationConfig(bucket.Name, BucketReplicationConfig{
		Role: "arn:aws:iam::123456789012:role/replication",
		Rules: []BucketReplicationRule{
			{
				ID:     "rule-1",
				Status: "Enabled",
			},
		},
	})

	snapshot := state.Snapshot()
	policies := snapshot["bucket_policies"].([]any)
	policies[0].(map[string]any)["body"] = "mutated"

	again, ok := state.BucketPolicy(bucket.Name)
	if !ok {
		t.Fatal("expected bucket policy to remain present")
	}
	if got, want := string(again), `{"statement":"allow"}`; got != want {
		t.Fatalf("snapshot mutated live policy state: got %q want %q", got, want)
	}

	notifications := snapshot["bucket_notifications"].([]any)
	notifications[0].(map[string]any)["body"] = "mutated"
	if got, ok := state.BucketNotification(bucket.Name); !ok || string(got) != "<NotificationConfiguration/>" {
		t.Fatalf("snapshot mutated live notification state: ok=%v body=%q", ok, string(got))
	}

	replication := snapshot["bucket_replication"].([]any)
	replication[0].(map[string]any)["role"] = "mutated"
	if got, ok := state.BucketReplicationConfig(bucket.Name); !ok || got.Role != "arn:aws:iam::123456789012:role/replication" {
		t.Fatalf("snapshot mutated live replication state: ok=%v role=%q", ok, got.Role)
	}
}
