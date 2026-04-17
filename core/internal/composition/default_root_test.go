package composition

import (
	"os"
	"path/filepath"
	"testing"

	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	dynamodbdomain "github.com/michasdev/mildstack/core/internal/dynamodb/domain"
	"github.com/michasdev/mildstack/core/internal/s3/application"
	s3domain "github.com/michasdev/mildstack/core/internal/s3/domain"
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

	hook := &stateHookStub{}
	root := defaultRootWithHook(hook, DefaultRootConfig{InstanceID: "test-instance"})
	if got, want := len(root.Services), 2; got != want {
		t.Fatalf("unexpected service count: got %d want %d", got, want)
	}

	first := root.Services[0]
	second := root.Services[1]
	if got, want := first.Metadata().Name, "s3"; got != want {
		t.Fatalf("unexpected first service name: got %q want %q", got, want)
	}
	if got, want := second.Metadata().Name, "dynamodb"; got != want {
		t.Fatalf("unexpected second service name: got %q want %q", got, want)
	}

	registrar := deliveryhttp.NewRegistrar()
	for _, service := range root.Services {
		if err := service.RegisterRoutes(registrar); err != nil {
			t.Fatalf("register routes: %v", err)
		}
	}

	entries := registrar.Services()
	if got, want := len(entries), 2; got != want {
		t.Fatalf("unexpected catalog size: got %d want %d", got, want)
	}
	if got, want := entries[0].Name, "dynamodb"; got != want {
		t.Fatalf("unexpected first catalog service: got %q want %q", got, want)
	}
	if got, want := entries[1].Name, "s3"; got != want {
		t.Fatalf("unexpected second catalog service: got %q want %q", got, want)
	}

	s3Entry, ok := registrar.Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(s3Entry.Routes), 11; got != want {
		t.Fatalf("unexpected s3 route count: got %d want %d", got, want)
	}
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/buckets")
	assertRouteExists(t, s3Entry.Routes, "POST", "/api/v1/runtime/services/s3/buckets")
	assertRouteExists(t, s3Entry.Routes, "HEAD", "/api/v1/runtime/services/s3/buckets/:bucket")
	assertRouteExists(t, s3Entry.Routes, "DELETE", "/api/v1/runtime/services/s3/buckets/:bucket")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects/v2")
	assertRouteExists(t, s3Entry.Routes, "POST", "/api/v1/runtime/services/s3/buckets/:bucket/objects/delete")
	assertRouteExists(t, s3Entry.Routes, "GET", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object")
	assertRouteExists(t, s3Entry.Routes, "HEAD", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object")
	assertRouteExists(t, s3Entry.Routes, "PUT", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object")
	assertRouteExists(t, s3Entry.Routes, "DELETE", "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object")

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

func TestDefaultRootFailsFastWhenPersistedS3StateIsCorrupt(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	storagePath, err := application.ResolveStoragePath(application.StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "broken-instance",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatalf("mkdir storage path: %v", err)
	}
	statePath := filepath.Join(storagePath, "state.json")
	if err := os.WriteFile(statePath, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected corrupt state to panic during default root bootstrap")
		}
	}()

	_ = defaultRootWithHook(&stateHookStub{}, DefaultRootConfig{
		InstanceID:       "broken-instance",
		S3StorageBaseDir: baseDir,
	})
}
