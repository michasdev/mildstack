package composition

import (
	"os"
	"path/filepath"
	"testing"

	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	dynamodbapp "github.com/michasdev/mildstack/core/internal/resources/dynamodb/application"
	dynamodbdomain "github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
	s3domain "github.com/michasdev/mildstack/core/internal/resources/s3/domain"
	sqsdomain "github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

type stateHookStub struct {
	values map[string]any
}

func (h *stateHookStub) Set(key string, value any) {
	if h.values == nil {
		h.values = make(map[string]any)
	}
	h.values[key] = value
}

func (h *stateHookStub) Get(key string) (any, bool) {
	value, ok := h.values[key]
	return value, ok
}

func TestDefaultRootIncludesS3AndDynamoDBWithDeterministicRoutes(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	hook := &stateHookStub{}
	root := defaultRootWithHook(hook, DefaultRootConfig{
		InstanceID:             "test-instance",
		S3StorageBaseDir:       baseDir,
		DynamoDBStorageBaseDir: baseDir,
		SQSStorageBaseDir:      baseDir,
	})
	if got, want := len(root.Services), 3; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}

	first := root.Services[0]
	second := root.Services[1]
	third := root.Services[2]
	if got, want := first.Metadata().Name, "s3"; got != want {
		t.Fatalf("unexpected first service name: got %q want %q", got, want)
	}
	if got, want := second.Metadata().Name, "dynamodb"; got != want {
		t.Fatalf("unexpected second service name: got %q want %q", got, want)
	}
	if got, want := third.Metadata().Name, "sqs"; got != want {
		t.Fatalf("unexpected third service name: got %q want %q", got, want)
	}

	registrar := deliveryhttp.NewRegistrar()
	for _, service := range root.Services {
		if err := service.RegisterRoutes(registrar); err != nil {
			t.Fatalf("register routes: %v", err)
		}
	}

	entries := registrar.Services()
	if got, want := len(entries), 3; got != want {
		t.Fatalf("unexpected catalog size: got %d want %d", got, want)
	}
	if got, want := entries[0].Name, "dynamodb"; got != want {
		t.Fatalf("unexpected first catalog service: got %q want %q", got, want)
	}
	if got, want := entries[1].Name, "s3"; got != want {
		t.Fatalf("unexpected second catalog service: got %q want %q", got, want)
	}
	if got, want := entries[2].Name, "sqs"; got != want {
		t.Fatalf("unexpected third catalog service: got %q want %q", got, want)
	}

	s3Entry, ok := registrar.Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(s3Entry.Routes), 62; got != want {
		t.Fatalf("unexpected s3 route count: got %d want %d", got, want)
	}
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3")
	assertRouteExists(t, s3Entry.Routes, "POST", "/api/v1/runtime/services/s3")
	assertRouteExists(t, s3Entry.Routes, "HEAD", "/api/v1/runtime/services/s3/:bucket")
	assertRouteExists(t, s3Entry.Routes, "DELETE", "/api/v1/runtime/services/s3/:bucket")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket?versioning")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/:bucket?versioning")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket?versions")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket?object-lock")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/:bucket?object-lock")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket/:object?retention")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/:bucket/:object?retention")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket/:object?legal-hold")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/:bucket/:object?legal-hold")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket?list-type=2")
	assertRouteExists(t, s3Entry.Routes, "POST", "/api/v1/runtime/services/s3/:bucket?delete")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/:bucket/:object")
	assertRouteExists(t, s3Entry.Routes, "HEAD", "/api/v1/runtime/services/s3/:bucket/:object")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/:bucket/:object")
	assertRouteExists(t, s3Entry.Routes, "DELETE", "/api/v1/runtime/services/s3/:bucket/:object")
	assertRouteExists(t, s3Entry.Routes, "POST", "/api/v1/runtime/services/s3/:bucket/:object?uploads")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/:bucket/:object?partNumber=:part&uploadId=:upload")
	assertRouteExists(t, s3Entry.Routes, "POST", "/api/v1/runtime/services/s3/:bucket/:object?uploadId=:upload")
	assertRouteExists(t, s3Entry.Routes, "DELETE", "/api/v1/runtime/services/s3/:bucket/:object?uploadId=:upload")

	dynamoEntry, ok := registrar.Service("dynamodb")
	if !ok {
		t.Fatal("expected dynamodb service to be registered")
	}
	if got, want := len(dynamoEntry.Routes), 5; got != want {
		t.Fatalf("unexpected dynamodb route count: got %d want %d", got, want)
	}
	if got, want := dynamoEntry.Routes[0].Method, "DELETE"; got != want {
		t.Fatalf("unexpected dynamodb first route method: got %q want %q", got, want)
	}
	if got, want := dynamoEntry.Routes[0].Path, "/api/v1/runtime/services/dynamodb/tables/:table/items/:item"; got != want {
		t.Fatalf("unexpected dynamodb first route path: got %q want %q", got, want)
	}
	if got, want := dynamoEntry.Routes[1].Path, "/api/v1/runtime/services/dynamodb/tables"; got != want {
		t.Fatalf("unexpected dynamodb second route path: got %q want %q", got, want)
	}
	if got, want := dynamoEntry.Routes[4].Method, "PUT"; got != want {
		t.Fatalf("unexpected dynamodb last route method: got %q want %q", got, want)
	}
	if got, want := dynamoEntry.Routes[4].Path, "/api/v1/runtime/services/dynamodb/tables/:table/items/:item"; got != want {
		t.Fatalf("unexpected dynamodb last route path: got %q want %q", got, want)
	}
	if _, ok := root.Services[1].(deliveryhttp.DynamoDBNativeService); !ok {
		t.Fatal("expected dynamodb service to expose the native http surface")
	}
	if _, ok := root.Services[2].(deliveryhttp.SQSNativeService); !ok {
		t.Fatal("expected sqs service to expose the native http surface")
	}

	if value, ok := hook.Get(dynamodbdomain.StateKey); !ok {
		t.Fatalf("expected state for %q to be present", dynamodbdomain.StateKey)
	} else {
		state := value.(map[string]any)
		if got, want := state["service"], "dynamodb"; got != want {
			t.Fatalf("unexpected dynamodb state: got %v want %v", got, want)
		}
	}

	value, ok := hook.Get(s3domain.StateKey)
	if !ok {
		t.Fatalf("expected state for %q to be present", s3domain.StateKey)
	}
	state := value.(map[string]any)
	if got, want := state["service"], "s3"; got != want {
		t.Fatalf("unexpected s3 state: got %v want %v", got, want)
	}

	if value, ok := hook.Get(sqsdomain.StateKey); !ok {
		t.Fatalf("expected state for %q to be present", sqsdomain.StateKey)
	} else {
		state := value.(map[string]any)
		if got, want := state["service"], "sqs"; got != want {
			t.Fatalf("unexpected sqs state: got %v want %v", got, want)
		}
		if got, want := len(state["queues"].([]any)), 0; got != want {
			t.Fatalf("unexpected sqs queue count: got %d want %d", got, want)
		}
	}

	dynamoDBPath := filepath.Join(baseDir, "instances", "test-instance", "dynamodb", "state.db")
	if _, err := os.Stat(dynamoDBPath); err != nil {
		t.Fatalf("expected dynamodb database to exist at %s: %v", dynamoDBPath, err)
	}
}

func assertRouteExists(t *testing.T, routes []deliveryhttp.RegisteredRoute, method, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	t.Fatalf("expected route %s %s to be registered", method, path)
}

func TestDefaultRootUsesInstanceScopedDynamoDBStorage(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	storagePath, err := dynamodbapp.ResolveStoragePath(dynamodbapp.StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	otherStoragePath, err := dynamodbapp.ResolveStoragePath(dynamodbapp.StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "instance-b",
	})
	if err != nil {
		t.Fatalf("resolve other storage path: %v", err)
	}
	if storagePath == otherStoragePath {
		t.Fatalf("expected distinct storage paths, got %q", storagePath)
	}

	root := defaultRootWithHook(&stateHookStub{}, DefaultRootConfig{
		InstanceID:             "instance-a",
		S3StorageBaseDir:       baseDir,
		DynamoDBStorageBaseDir: baseDir,
		SQSStorageBaseDir:      baseDir,
	})
	if got, want := len(root.Services), 3; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}

	if _, err := os.Stat(filepath.Join(storagePath, "state.db")); err != nil {
		t.Fatalf("expected storage file at %s: %v", storagePath, err)
	}
}

func TestDefaultRootWithEmptyInstanceIDReturnsNoServices(t *testing.T) {
	root := defaultRootWithHook(&stateHookStub{}, DefaultRootConfig{})
	if len(root.Services) != 0 {
		t.Fatalf("expected empty root when instance id is missing, got %d services", len(root.Services))
	}
}
