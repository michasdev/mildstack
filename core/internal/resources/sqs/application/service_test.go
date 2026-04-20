package application

import (
	"context"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
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

func TestSQSServiceAttachStateUsesNamespacedCopySafeSnapshot(t *testing.T) {
	t.Helper()

	service := newService(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-a",
				Attributes: map[string]string{
					"VisibilityTimeout": "30",
				},
				Recovery: domain.QueueRecovery{
					DeadLetterQueue: "queue-dlq",
				},
			},
		},
		Messages: []domain.Message{
			{
				Queue:       "queue-a",
				MessageID:   "message-1",
				Body:        "payload",
				Tags:        []string{"alpha"},
				Metadata:    map[string]string{"trace": "abc"},
				ReceiptKeys: []string{"r-1"},
			},
		},
	}, nil)

	hook := runtime.NewStateHook()
	if err := service.AttachState(hook); err != nil {
		t.Fatalf("attach state: %v", err)
	}

	value, ok := hook.Get(domain.StateKey)
	if !ok {
		t.Fatalf("expected state for %q to be present", domain.StateKey)
	}
	state := value.(map[string]any)
	if got, want := state["service"], "sqs"; got != want {
		t.Fatalf("unexpected service name: got %v want %v", got, want)
	}
	queues := state["queues"].([]any)
	if got, want := len(queues), 1; got != want {
		t.Fatalf("unexpected queue count: got %d want %d", got, want)
	}
	queues[0].(map[string]any)["name"] = "mutated"
	queues[0].(map[string]any)["attributes"].(map[string]any)["VisibilityTimeout"] = "99"

	messages := state["messages"].([]any)
	if got, want := len(messages), 1; got != want {
		t.Fatalf("unexpected message count: got %d want %d", got, want)
	}
	messages[0].(map[string]any)["body"] = "mutated"
	messages[0].(map[string]any)["tags"].([]string)[0] = "mutated"

	if got, want := service.state.Queues[0].Name, "queue-a"; got != want {
		t.Fatalf("service queue name was aliased: got %q want %q", got, want)
	}
	if got, want := service.state.Queues[0].Attributes["VisibilityTimeout"], "30"; got != want {
		t.Fatalf("service queue attributes were aliased: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].Body, "payload"; got != want {
		t.Fatalf("service message body was aliased: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].Tags[0], "alpha"; got != want {
		t.Fatalf("service message tags were aliased: got %q want %q", got, want)
	}
}

func TestSQSServiceNewWithPersistenceLoadsRepositoryState(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	config := StorageConfig{BaseDir: baseDir, InstanceID: "instance-a"}
	storagePath, err := ResolveStoragePath(config)
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}

	repo, err := NewSQLiteRepository(storagePath)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}

	state := domain.NewState()
	state.Queues = append(state.Queues, domain.Queue{
		Name: "queue-a",
		Attributes: map[string]string{
			"VisibilityTimeout": "45",
		},
		Recovery: domain.QueueRecovery{
			DeadLetterQueue: "queue-dlq",
		},
		CreatedAt: time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC),
	})
	state.Messages = append(state.Messages, domain.Message{
		Queue:     "queue-a",
		MessageID: "message-1",
		Body:      "payload",
		Tags:      []string{"persisted"},
	})
	if err := repo.Save(state); err != nil {
		_ = repo.Close()
		t.Fatalf("save seeded state: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close seeded repository: %v", err)
	}

	service, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("new with persistence: %v", err)
	}
	defer func() {
		if err := service.Stop(context.Background()); err != nil {
			t.Fatalf("stop service: %v", err)
		}
	}()

	if service.repo == nil {
		t.Fatal("expected persistent repository to be attached")
	}
	if got, want := len(service.state.Queues), 1; got != want {
		t.Fatalf("unexpected queue count after load: got %d want %d", got, want)
	}
	if got, want := service.state.Queues[0].Name, "queue-a"; got != want {
		t.Fatalf("unexpected queue name after load: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].MessageID, "message-1"; got != want {
		t.Fatalf("unexpected message id after load: got %q want %q", got, want)
	}
}

func TestSQSServiceStopClosesRepositoryIdempotently(t *testing.T) {
	t.Helper()

	repo := &repositoryStub{}
	service := newService(domain.NewState(), repo)

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("second stop: %v", err)
	}
	if got, want := repo.closeCount, 1; got != want {
		t.Fatalf("unexpected close count: got %d want %d", got, want)
	}
	if service.repo != nil {
		t.Fatal("expected repository handle to be cleared after stop")
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

type repositoryStub struct {
	closeCount int
}

func (r *repositoryStub) Load() (domain.State, error) {
	return domain.NewState(), nil
}

func (r *repositoryStub) Save(state domain.State) error {
	return nil
}

func (r *repositoryStub) Close() error {
	r.closeCount++
	return nil
}
