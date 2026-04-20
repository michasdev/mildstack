package application

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func TestSQSServiceMetadataRoutesAndPolicy(t *testing.T) {
	t.Helper()

	service := New()
	if _, ok := any(service).(orchestrator.Service); !ok {
		t.Fatal("expected service to satisfy orchestrator.Service")
	}

	metadata := service.Metadata()
	if got, want := metadata.Name, "sqs"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := metadata.Version, "v1"; got != want {
		t.Fatalf("unexpected service version: got %q want %q", got, want)
	}
	if got, want := metadata.Description, "MildStack SQS real service"; got != want {
		t.Fatalf("unexpected service description: got %q want %q", got, want)
	}

	expectedTags := []string{"aws", "messaging", "queue", "real-service"}
	if got, want := len(metadata.Tags), len(expectedTags); got != want {
		t.Fatalf("unexpected tag count: got %d want %d", got, want)
	}
	for i, tag := range expectedTags {
		if metadata.Tags[i] != tag {
			t.Fatalf("unexpected tag at %d: got %q want %q", i, metadata.Tags[i], tag)
		}
	}

	policy := service.Policy()
	if got, want := policy.Fidelity, orchestrator.FidelityExemplar; got != want {
		t.Fatalf("unexpected policy fidelity: got %q want %q", got, want)
	}
	if got, want := policy.ErrorPrefix, "sqs"; got != want {
		t.Fatalf("unexpected policy error prefix: got %q want %q", got, want)
	}
	if got, want := len(policy.Supported), 23; got != want {
		t.Fatalf("unexpected supported count: got %d want %d", got, want)
	}
	if got, want := len(policy.Unsupported), 0; got != want {
		t.Fatalf("unexpected unsupported count: got %d want %d", got, want)
	}

	policy.Supported[0] = "changed"
	again := service.Policy()
	if got, want := again.Supported[0], "AddPermission"; got != want {
		t.Fatalf("policy supported slice was not copied: got %q want %q", got, want)
	}

	registrar := deliveryhttp.NewRegistrar()
	if err := service.RegisterRoutes(registrar); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	entry, ok := registrar.Service("sqs")
	if !ok {
		t.Fatal("expected sqs service to be registered")
	}
	if got, want := len(entry.Routes), 7; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	assertRouteExists(t, entry.Routes, "GET", "/api/v1/runtime/services/sqs/queues")
	assertRouteExists(t, entry.Routes, "POST", "/api/v1/runtime/services/sqs/queues")
	assertRouteExists(t, entry.Routes, "GET", "/api/v1/runtime/services/sqs/queues/:queue")
	assertRouteExists(t, entry.Routes, "DELETE", "/api/v1/runtime/services/sqs/queues/:queue")
	assertRouteExists(t, entry.Routes, "GET", "/api/v1/runtime/services/sqs/queues/:queue/messages")
	assertRouteExists(t, entry.Routes, "POST", "/api/v1/runtime/services/sqs/queues/:queue/messages")
	assertRouteExists(t, entry.Routes, "DELETE", "/api/v1/runtime/services/sqs/queues/:queue/messages/:receiptHandle")
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
