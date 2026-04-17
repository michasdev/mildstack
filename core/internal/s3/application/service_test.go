package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
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
	if got, want := len(policy.Supported), 6; got != want {
		t.Fatalf("unexpected supported count: got %d want %d", got, want)
	}
	if got, want := len(policy.Unsupported), 2; got != want {
		t.Fatalf("unexpected unsupported count: got %d want %d", got, want)
	}
	policy.Supported[0] = "changed"
	policy.Unsupported[0] = "changed"
	again := service.Policy()
	if got, want := again.Supported[0], "list buckets"; got != want {
		t.Fatalf("policy supported slice was not copied: got %q want %q", got, want)
	}
	if got, want := again.Unsupported[0], "bucket versioning"; got != want {
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
	if got, want := len(entry.Routes), 6; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	if got, want := entry.Routes[0].Method, "DELETE"; got != want {
		t.Fatalf("unexpected first route method: got %q want %q", got, want)
	}
	if got, want := entry.Routes[0].Path, "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object"; got != want {
		t.Fatalf("unexpected first route path: got %q want %q", got, want)
	}
	if got, want := entry.Routes[1].Path, "/api/v1/runtime/services/s3/buckets"; got != want {
		t.Fatalf("unexpected second route path: got %q want %q", got, want)
	}
	if got, want := entry.Routes[5].Method, "PUT"; got != want {
		t.Fatalf("unexpected last route method: got %q want %q", got, want)
	}
	if got, want := entry.Routes[5].Path, "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object"; got != want {
		t.Fatalf("unexpected last route path: got %q want %q", got, want)
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

	object, err := service.PutObject(bucket.Name, "archive.txt", 42, "text/plain")
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

	if err := service.DeleteObject(bucket.Name, object.Key); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if _, err := service.GetObject(bucket.Name, object.Key); err == nil {
		t.Fatal("expected deleted object lookup to fail")
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
	}
	if _, err := service.PutObject("missing", "archive.txt", 1, "text/plain"); err == nil {
		t.Fatal("expected put on missing bucket to fail")
	}
	if err := service.DeleteObject("mildstack-assets", "missing"); err == nil {
		t.Fatal("expected delete on missing object to fail")
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
	if _, err := first.PutObject(bucket.Name, "archive.txt", 42, "text/plain"); err != nil {
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
	if _, err := second.GetObject(bucket.Name, "archive.txt"); err != nil {
		t.Fatalf("expected restored object after restart: %v", err)
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
