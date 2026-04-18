package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/s3/domain"
)

func TestFSRepositorySaveAndLoadAreDeterministic(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	state := domain.State{
		Service: "s3",
		Buckets: []domain.Bucket{
			{Name: "zeta", Region: "us-east-1", CreatedAt: time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC)},
			{Name: "alpha", Region: "us-west-2", CreatedAt: time.Date(2026, time.April, 17, 11, 0, 0, 0, time.UTC)},
		},
		Objects: []domain.Object{
			{Bucket: "zeta", Key: "b.txt", Body: []byte("zeta-body"), Size: int64(len("zeta-body")), ContentType: "text/plain"},
			{Bucket: "alpha", Key: "a.txt", Body: []byte("alpha-body"), Size: int64(len("alpha-body")), ContentType: "text/plain"},
		},
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save first pass: %v", err)
	}
	firstBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read first state file: %v", err)
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save second pass: %v", err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read second state file: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("expected deterministic state.json output across identical saves")
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got, want := loaded.Buckets[0].Name, "alpha"; got != want {
		t.Fatalf("expected sorted bucket order after round-trip: got %q want %q", got, want)
	}
	if loaded.Buckets[0].CreatedAt.IsZero() || loaded.Buckets[1].CreatedAt.IsZero() {
		t.Fatal("expected bucket creation timestamps after round-trip")
	}
	if got, want := loaded.Objects[0].Bucket, "alpha"; got != want {
		t.Fatalf("expected sorted object order after round-trip: got %q want %q", got, want)
	}
	if got, want := string(loaded.Objects[0].Body), "alpha-body"; got != want {
		t.Fatalf("expected loaded object body after round-trip: got %q want %q", got, want)
	}
	if strings.Contains(string(secondBytes), "alpha-body") || strings.Contains(string(secondBytes), "zeta-body") {
		t.Fatal("expected payload bytes to stay out of state.json")
	}
}

func TestRepositoryFSPayloadLayoutRoundTrip(t *testing.T) {
	t.Helper()
	TestFSRepositorySaveAndLoadAreDeterministic(t)
}

func TestRepositoryFSPayloadRoundTrip(t *testing.T) {
	t.Helper()
	TestFSRepositorySaveAndLoadAreDeterministic(t)
}

func TestFSRepositoryPersistsVersionHistoryRoundTrip(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	state := domain.NewState()
	bucket := state.UpsertBucket(domain.Bucket{Name: "history-bucket", Region: "us-west-2", CreatedAt: time.Date(2026, time.April, 17, 13, 0, 0, 0, time.UTC)})
	state.SetBucketVersioning(bucket.Name, domain.VersioningEnabled)

	first := state.UpsertObject(domain.Object{
		Bucket:      bucket.Name,
		Key:         "artifact.txt",
		Body:        []byte("v1"),
		Size:        2,
		ContentType: "text/plain",
	})
	state.RecordObjectVersion(first)
	second := state.UpsertObject(domain.Object{
		Bucket:      bucket.Name,
		Key:         "artifact.txt",
		Body:        []byte("v2"),
		Size:        2,
		ContentType: "text/plain",
	})
	state.RecordObjectVersion(second)
	state.DeleteObject(bucket.Name, "artifact.txt")
	state.RecordDeleteMarker(bucket.Name, "artifact.txt")

	if err := repo.Save(state); err != nil {
		t.Fatalf("save versioned state: %v", err)
	}
	firstBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read versioned state file: %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load versioned state: %v", err)
	}
	if got, want := loaded.BucketVersioningStatus(bucket.Name), domain.VersioningEnabled; got != want {
		t.Fatalf("unexpected loaded versioning status: got %q want %q", got, want)
	}
	versions := loaded.ListObjectVersions(bucket.Name)
	if got, want := len(versions), 3; got != want {
		t.Fatalf("unexpected loaded version count: got %d want %d", got, want)
	}
	if !versions[0].IsDeleteMarker {
		t.Fatal("expected loaded delete marker to remain latest")
	}
	if got, want := string(versions[1].Body), "v2"; got != want {
		t.Fatalf("unexpected loaded second version body: got %q want %q", got, want)
	}
	if got, want := string(versions[2].Body), "v1"; got != want {
		t.Fatalf("unexpected loaded first version body: got %q want %q", got, want)
	}

	if err := repo.Save(loaded); err != nil {
		t.Fatalf("resave loaded versioned state: %v", err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read resaved versioned state file: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("expected versioned state round-trip to stay deterministic")
	}
}

func TestFSRepositoryPersistsGovernanceRoundTrip(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	state := domain.NewState()
	bucket := state.UpsertBucket(domain.Bucket{Name: "governed-bucket", Region: "us-west-2", CreatedAt: time.Date(2026, time.April, 17, 14, 0, 0, 0, time.UTC)})
	state.SetBucketPolicy(bucket.Name, []byte(`{"statement":"allow"}`))
	state.SetBucketEncryptionConfig(bucket.Name, []byte("<EncryptionConfiguration/>"))
	state.SetBucketLifecycleConfig(bucket.Name, []byte("<LifecycleConfiguration/>"))
	state.SetBucketCORSConfig(bucket.Name, []byte("<CORSConfiguration/>"))
	state.SetBucketACLConfig(bucket.Name, []byte("<AccessControlPolicy/>"))
	state.SetBucketTaggingConfig(bucket.Name, []byte("<Tagging/>"))
	state.SetBucketNotification(bucket.Name, []byte("<NotificationConfiguration/>"))
	state.SetBucketLoggingConfig(bucket.Name, []byte("<BucketLoggingStatus/>"))
	state.SetBucketReplicationConfig(bucket.Name, domain.BucketReplicationConfig{
		Role: "arn:aws:iam::123456789012:role/replication",
		Rules: []domain.BucketReplicationRule{
			{
				ID:     "rule-1",
				Status: "Enabled",
			},
		},
	})

	if err := repo.Save(state); err != nil {
		t.Fatalf("save governed state: %v", err)
	}
	firstBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read governed state file: %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load governed state: %v", err)
	}
	if got, ok := loaded.BucketPolicy(bucket.Name); !ok || string(got) != `{"statement":"allow"}` {
		t.Fatalf("unexpected loaded policy: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketEncryptionConfig(bucket.Name); !ok || string(got) != "<EncryptionConfiguration/>" {
		t.Fatalf("unexpected loaded encryption: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketLifecycleConfig(bucket.Name); !ok || string(got) != "<LifecycleConfiguration/>" {
		t.Fatalf("unexpected loaded lifecycle: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketCORSConfig(bucket.Name); !ok || string(got) != "<CORSConfiguration/>" {
		t.Fatalf("unexpected loaded cors: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketACLConfig(bucket.Name); !ok || string(got) != "<AccessControlPolicy/>" {
		t.Fatalf("unexpected loaded acl: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketTaggingConfig(bucket.Name); !ok || string(got) != "<Tagging/>" {
		t.Fatalf("unexpected loaded tagging: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketNotification(bucket.Name); !ok || string(got) != "<NotificationConfiguration/>" {
		t.Fatalf("unexpected loaded notification: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketLoggingConfig(bucket.Name); !ok || string(got) != "<BucketLoggingStatus/>" {
		t.Fatalf("unexpected loaded logging: ok=%v body=%q", ok, string(got))
	}
	if got, ok := loaded.BucketReplicationConfig(bucket.Name); !ok || got.Role != "arn:aws:iam::123456789012:role/replication" {
		t.Fatalf("unexpected loaded replication: ok=%v role=%q", ok, got.Role)
	}
	if got, ok := loaded.BucketReplicationConfig(bucket.Name); !ok || len(got.Rules) != 1 || got.Rules[0].ID != "rule-1" {
		t.Fatalf("unexpected loaded replication rules: ok=%v rules=%+v", ok, got.Rules)
	}

	if err := repo.Save(loaded); err != nil {
		t.Fatalf("resave governed state: %v", err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(repo.storagePath, stateFileName))
	if err != nil {
		t.Fatalf("read resaved governed state file: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("expected governed state round-trip to stay deterministic")
	}
}

func TestFSRepositoryLoadTreatsMissingFileAsNewState(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load missing state: %v", err)
	}
	if got, want := loaded.Service, "s3"; got != want {
		t.Fatalf("unexpected default service: got %q want %q", got, want)
	}
}

func TestFSRepositoryLoadRejectsInvalidState(t *testing.T) {
	t.Helper()

	repo := NewFSRepository(t.TempDir())
	statePath := filepath.Join(repo.storagePath, stateFileName)
	if err := os.MkdirAll(repo.storagePath, 0o755); err != nil {
		t.Fatalf("mkdir storage path: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("{oops"), 0o644); err != nil {
		t.Fatalf("write invalid state: %v", err)
	}

	_, err := repo.Load()
	if err == nil {
		t.Fatal("expected invalid state file to return an error")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("unexpected error for invalid state: %v", err)
	}
}

func TestResolveStoragePathUsesMildstackInstanceLayout(t *testing.T) {
	t.Helper()

	path, err := ResolveStoragePath(StorageConfig{
		BaseDir:    "/tmp/mildstack-root",
		InstanceID: "instance-42",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	if got, want := path, filepath.Join("/tmp/mildstack-root", "instances", "instance-42", "s3"); got != want {
		t.Fatalf("unexpected path: got %q want %q", got, want)
	}
}
