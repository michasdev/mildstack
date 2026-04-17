package composition

import (
	"testing"

	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	dynamodbdomain "github.com/michasdev/mildstack/core/internal/dynamodb/domain"
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
	root := defaultRootWithHook(hook)
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
	if got, want := len(s3Entry.Routes), 6; got != want {
		t.Fatalf("unexpected s3 route count: got %d want %d", got, want)
	}
	if got, want := s3Entry.Routes[0].Method, "DELETE"; got != want {
		t.Fatalf("unexpected s3 first route method: got %q want %q", got, want)
	}
	if got, want := s3Entry.Routes[0].Path, "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object"; got != want {
		t.Fatalf("unexpected s3 first route path: got %q want %q", got, want)
	}
	if got, want := s3Entry.Routes[1].Path, "/api/v1/runtime/services/s3/buckets"; got != want {
		t.Fatalf("unexpected s3 second route path: got %q want %q", got, want)
	}
	if got, want := s3Entry.Routes[5].Method, "PUT"; got != want {
		t.Fatalf("unexpected s3 last route method: got %q want %q", got, want)
	}
	if got, want := s3Entry.Routes[5].Path, "/api/v1/runtime/services/s3/buckets/:bucket/objects/:object"; got != want {
		t.Fatalf("unexpected s3 last route path: got %q want %q", got, want)
	}

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
