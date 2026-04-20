package application

import (
	"context"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
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

func TestSQSServiceExposesQueueLifecycleAPI(t *testing.T) {
	t.Helper()

	clock := newManualClock(time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC))
	service := newServiceWithClock(domain.NewState(), nil, clock)
	type lifecycleAPI interface {
		QueueURL(string) string
		QueueARN(string) string
		CreateQueue(string, map[string]string) (domain.Queue, error)
		DeleteQueue(string) error
		GetQueueUrl(string, string) (string, error)
		ListQueues(string, int, string, string) ([]domain.Queue, string, error)
		PurgeQueue(string) error
		GetQueueAttributes(string, []string, string) (contracts.QueueAttributesView, error)
		SetQueueAttributes(string, map[string]string) (contracts.QueueAttributesView, error)
	}

	if _, ok := any(service).(lifecycleAPI); !ok {
		t.Fatal("expected service to expose queue lifecycle API")
	}

	if got, want := service.QueueURL("orders"), "https://sqs.us-east-1.amazonaws.com/123456789012/orders"; got != want {
		t.Fatalf("unexpected queue url helper: got %q want %q", got, want)
	}
	if got, want := service.QueueARN("orders"), "arn:aws:sqs:us-east-1:123456789012:orders"; got != want {
		t.Fatalf("unexpected queue arn helper: got %q want %q", got, want)
	}

	queue, err := service.CreateQueue("orders", map[string]string{
		"VisibilityTimeout": "30",
		"RedrivePolicy":     `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:orders-dlq"}`,
	})
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	if got, want := queue.URL, service.QueueURL("orders"); got != want {
		t.Fatalf("unexpected queue url: got %q want %q", got, want)
	}
	if got, want := queue.Attributes["VisibilityTimeout"], "30"; got != want {
		t.Fatalf("unexpected queue attribute: got %q want %q", got, want)
	}

	sameQueue, err := service.CreateQueue("orders", map[string]string{
		"VisibilityTimeout": "30",
		"RedrivePolicy":     `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:orders-dlq"}`,
	})
	if err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	if got, want := sameQueue.URL, queue.URL; got != want {
		t.Fatalf("unexpected idempotent queue url: got %q want %q", got, want)
	}
	if _, err := service.CreateQueue("orders", map[string]string{"VisibilityTimeout": "45"}); err == nil {
		t.Fatal("expected create with different attributes to fail")
	}

	archiveQueue, err := service.CreateQueue("orders-archive", map[string]string{"VisibilityTimeout": "45"})
	if err != nil {
		t.Fatalf("create archive queue: %v", err)
	}
	if got, want := archiveQueue.URL, service.QueueURL("orders-archive"); got != want {
		t.Fatalf("unexpected archive queue url: got %q want %q", got, want)
	}

	list, nextToken, err := service.ListQueues("ord", 1, "", "")
	if err != nil {
		t.Fatalf("list queues: %v", err)
	}
	if got, want := len(list), 1; got != want {
		t.Fatalf("unexpected paged queue count: got %d want %d", got, want)
	}
	if got, want := list[0].Name, "orders"; got != want {
		t.Fatalf("unexpected first page queue: got %q want %q", got, want)
	}
	if got, want := nextToken, "orders"; got != want {
		t.Fatalf("unexpected next token: got %q want %q", got, want)
	}

	nextPage, nextToken, err := service.ListQueues("ord", 10, nextToken, "")
	if err != nil {
		t.Fatalf("second page list queues: %v", err)
	}
	if got, want := len(nextPage), 1; got != want {
		t.Fatalf("unexpected second page queue count: got %d want %d", got, want)
	}
	if got, want := nextPage[0].Name, "orders-archive"; got != want {
		t.Fatalf("unexpected second page queue: got %q want %q", got, want)
	}
	if got, want := nextToken, ""; got != want {
		t.Fatalf("unexpected terminal next token: got %q want %q", got, want)
	}

	queueURL, err := service.GetQueueUrl("orders", "")
	if err != nil {
		t.Fatalf("get queue url: %v", err)
	}
	if got, want := queueURL, service.QueueURL("orders"); got != want {
		t.Fatalf("unexpected get queue url result: got %q want %q", got, want)
	}

	if _, err := service.SetQueueAttributes("orders", map[string]string{
		"VisibilityTimeout":   "45",
		"RedriveAllowPolicy":  `{"redrivePermission":"byQueue"}`,
		"RedrivePolicy":       `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:orders-dlq"}`,
		"ContentBasedDeduplication": "true",
	}); err != nil {
		t.Fatalf("set queue attributes: %v", err)
	}

	attrView, err := service.GetQueueAttributes("orders", []string{"All"}, "")
	if err != nil {
		t.Fatalf("get queue attributes: %v", err)
	}
	if got, want := attrView.QueueURL, service.QueueURL("orders"); got != want {
		t.Fatalf("unexpected queue attribute url: got %q want %q", got, want)
	}
	if got, want := attrView.QueueARN, service.QueueARN("orders"); got != want {
		t.Fatalf("unexpected queue attribute arn: got %q want %q", got, want)
	}
	if got, want := attrView.Attributes["VisibilityTimeout"], "45"; got != want {
		t.Fatalf("unexpected queue attribute value: got %q want %q", got, want)
	}
	if got, want := attrView.Attributes["RedriveAllowPolicy"], `{"redrivePermission":"byQueue"}`; got != want {
		t.Fatalf("unexpected opaque attribute value: got %q want %q", got, want)
	}
	if got, want := attrView.Attributes["QueueArn"], service.QueueARN("orders"); got != want {
		t.Fatalf("unexpected queue arn attribute: got %q want %q", got, want)
	}

	if err := service.DeleteQueue("orders"); err != nil {
		t.Fatalf("delete queue: %v", err)
	}
	if _, err := service.GetQueueUrl("orders", ""); err == nil {
		t.Fatal("expected deleted queue url lookup to fail")
	}
	if _, _, err := service.ListQueues("ord", 10, "", ""); err != nil {
		t.Fatalf("list queues after delete: %v", err)
	}
	if _, err := service.CreateQueue("orders", map[string]string{"VisibilityTimeout": "30"}); err == nil {
		t.Fatal("expected recreate during delete cooldown to fail")
	}

	clock.Sleep(queueLifecycleCooldown + time.Second)
	recreated, err := service.CreateQueue("orders", map[string]string{"VisibilityTimeout": "30"})
	if err != nil {
		t.Fatalf("recreate after cooldown: %v", err)
	}
	if got, want := recreated.URL, service.QueueURL("orders"); got != want {
		t.Fatalf("unexpected recreated queue url: got %q want %q", got, want)
	}

	service.state.Messages = append(service.state.Messages, domain.Message{
		Queue:     "orders-archive",
		MessageID: "message-1",
		Body:      "payload",
	})
	if err := service.PurgeQueue("orders-archive"); err != nil {
		t.Fatalf("purge queue: %v", err)
	}
	if got, want := len(service.state.Messages), 0; got != want {
		t.Fatalf("expected purge to delete queue messages, got %d", got)
	}
	if err := service.PurgeQueue("orders-archive"); err == nil {
		t.Fatal("expected back-to-back purge to fail")
	}
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

func TestSQSServiceReceiveMessageRespectsStandardOrderAndFifoOrdering(t *testing.T) {
	t.Helper()

	clock := newManualClock(time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC))
	service := newServiceWithClock(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-standard",
				Attributes: map[string]string{
					"VisibilityTimeout": "30",
				},
			},
			{
				Name:         "queue-fifo",
				OrderingHint: "fifo",
				Attributes: map[string]string{
					"VisibilityTimeout": "30",
					"FifoQueue":         "true",
				},
			},
		},
		Messages: []domain.Message{
			{
				Queue:     "queue-standard",
				MessageID: "std-late",
				Body:      "late",
				SentAt:    clock.Now().Add(time.Second),
			},
			{
				Queue:     "queue-standard",
				MessageID: "std-early",
				Body:      "early",
				SentAt:    clock.Now(),
			},
			{
				Queue:          "queue-fifo",
				MessageID:      "fifo-second",
				Body:           "second",
				MessageGroupID: "group-a",
				SequenceNumber: 2,
				SentAt:         clock.Now().Add(2 * time.Second),
			},
			{
				Queue:          "queue-fifo",
				MessageID:      "fifo-first",
				Body:           "first",
				MessageGroupID: "group-a",
				SequenceNumber: 1,
				SentAt:         clock.Now().Add(time.Second),
			},
		},
	}, nil, clock)

	standard, err := service.ReceiveMessage("queue-standard", 2, 0)
	if err != nil {
		t.Fatalf("receive standard messages: %v", err)
	}
	if got, want := len(standard), 2; got != want {
		t.Fatalf("unexpected standard count: got %d want %d", got, want)
	}
	if got, want := standard[0].MessageID, "std-early"; got != want {
		t.Fatalf("unexpected standard first message: got %q want %q", got, want)
	}
	if got, want := standard[1].MessageID, "std-late"; got != want {
		t.Fatalf("unexpected standard second message: got %q want %q", got, want)
	}

	fifo, err := service.ReceiveMessage("queue-fifo", 2, 0)
	if err != nil {
		t.Fatalf("receive fifo messages: %v", err)
	}
	if got, want := len(fifo), 1; got != want {
		t.Fatalf("unexpected fifo count: got %d want %d", got, want)
	}
	if got, want := fifo[0].MessageID, "fifo-first"; got != want {
		t.Fatalf("unexpected fifo first message: got %q want %q", got, want)
	}
}

func TestSQSServiceDeadLetterEligibilityUsesRecoveryPolicy(t *testing.T) {
	t.Helper()

	now := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	service := newService(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-a",
				Recovery: domain.QueueRecovery{
					DeadLetterQueue: "queue-dlq",
					Policy: map[string]string{
						"max_receive_count": "3",
					},
				},
			},
		},
		Messages: []domain.Message{
			{
				Queue:      "queue-a",
				MessageID:  "message-1",
				ReceivedAt: now.Add(-31 * time.Second),
				Recovery: domain.MessageRecovery{
					Attempts: 3,
				},
			},
		},
	}, nil)

	queue, ok := service.queueByNameLocked("queue-a")
	if !ok {
		t.Fatal("expected queue to exist")
	}
	if !service.deadLetterEligibleLocked(service.state.Messages[0], queue, now) {
		t.Fatal("expected message to be dead-letter eligible")
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
		OrderingHint: "fifo",
		Recovery: domain.QueueRecovery{
			DeadLetterQueue: "queue-dlq",
		},
		CreatedAt: time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC),
	})
	state.Messages = append(state.Messages, domain.Message{
		Queue:                 "queue-a",
		MessageID:             "message-1",
		Body:                  "payload",
		Tags:                  []string{"persisted"},
		MessageGroupID:        "group-a",
		SequenceNumber:        9,
		BatchID:               "batch-a",
		BatchEntryID:          "entry-a",
		BatchEntryIndex:       0,
		BatchEntryCount:       1,
		DeadLetterQueue:       "queue-dlq",
		DeadLetterSourceQueue: "queue-a",
		DeadLetteredAt:        time.Date(2026, time.April, 19, 12, 5, 0, 0, time.UTC),
		AvailableAt:           time.Date(2026, time.April, 19, 12, 3, 0, 0, time.UTC),
		ReceivedAt:            time.Date(2026, time.April, 19, 12, 4, 0, 0, time.UTC),
		ReceiptKeys:           []string{"r-1", "r-2"},
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
	if got, want := service.state.Queues[0].OrderingHint, "fifo"; got != want {
		t.Fatalf("unexpected queue ordering after load: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].MessageID, "message-1"; got != want {
		t.Fatalf("unexpected message id after load: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].AvailableAt, time.Date(2026, time.April, 19, 12, 3, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("unexpected available_at after load: got %v want %v", got, want)
	}
	if got, want := service.state.Messages[0].ReceivedAt, time.Date(2026, time.April, 19, 12, 4, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("unexpected received_at after load: got %v want %v", got, want)
	}
	if got, want := service.state.Messages[0].ReceiptKeys[1], "r-2"; got != want {
		t.Fatalf("unexpected receipt handle history after load: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].MessageGroupID, "group-a"; got != want {
		t.Fatalf("unexpected message group after load: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("unexpected dead letter queue after load: got %q want %q", got, want)
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

func TestSQSServiceReceiveMessageHonorsDelayAndBoundsLongPoll(t *testing.T) {
	t.Helper()

	clock := newManualClock(time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC))
	service := newServiceWithClock(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-a",
				Attributes: map[string]string{
					"VisibilityTimeout": "30",
				},
			},
		},
		Messages: []domain.Message{
			{
				Queue:       "queue-a",
				MessageID:   "message-1",
				Body:        "payload",
				SentAt:      clock.Now(),
				AvailableAt: clock.Now().Add(150 * time.Millisecond),
			},
		},
	}, nil, clock)

	messages, err := service.ReceiveMessage("queue-a", 1, 2*time.Second)
	if err != nil {
		t.Fatalf("receive message: %v", err)
	}
	if got, want := len(messages), 1; got != want {
		t.Fatalf("unexpected message count: got %d want %d", got, want)
	}
	if got, want := messages[0].MessageID, "message-1"; got != want {
		t.Fatalf("unexpected message id: got %q want %q", got, want)
	}
	if got := CurrentReceiptHandle(messages[0]); got == "" {
		t.Fatal("expected receive to issue a receipt handle")
	}
	if got := clock.SleepCount(); got == 0 {
		t.Fatal("expected long poll to sleep at least once")
	}
	if !messages[0].ReceivedAt.After(time.Time{}) {
		t.Fatal("expected receive to stamp the delivery time")
	}
}

func TestSQSServiceCapsLongPollAtTwentySeconds(t *testing.T) {
	t.Helper()

	clock := newManualClock(time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC))
	service := newServiceWithClock(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-a",
			},
		},
	}, nil, clock)

	messages, err := service.ReceiveMessage("queue-a", 1, 30*time.Second)
	if err != nil {
		t.Fatalf("receive message: %v", err)
	}
	if got, want := len(messages), 0; got != want {
		t.Fatalf("expected no messages, got %d", got)
	}
	if got, want := clock.totalSleep, 20*time.Second; got != want {
		t.Fatalf("unexpected capped sleep total: got %v want %v", got, want)
	}
}

func TestSQSServiceRejectsStaleReceiptHandlesAfterRedelivery(t *testing.T) {
	t.Helper()

	now := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	clock := newManualClock(now)
	service := newServiceWithClock(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-a",
				Attributes: map[string]string{
					"VisibilityTimeout": "30",
				},
			},
		},
		Messages: []domain.Message{
			{
				Queue:     "queue-a",
				MessageID: "message-1",
				Body:      "payload",
				SentAt:    now,
			},
		},
	}, nil, clock)

	first, err := service.ReceiveMessage("queue-a", 1, 0)
	if err != nil {
		t.Fatalf("first receive: %v", err)
	}
	if got, want := len(first), 1; got != want {
		t.Fatalf("unexpected first receive count: got %d want %d", got, want)
	}
	firstHandle := CurrentReceiptHandle(first[0])
	if firstHandle == "" {
		t.Fatal("expected first receive to issue a receipt handle")
	}

	clock.Sleep(31 * time.Second)
	second, err := service.ReceiveMessage("queue-a", 1, 0)
	if err != nil {
		t.Fatalf("second receive: %v", err)
	}
	if got, want := len(second), 1; got != want {
		t.Fatalf("unexpected second receive count: got %d want %d", got, want)
	}
	secondHandle := CurrentReceiptHandle(second[0])
	if secondHandle == "" || secondHandle == firstHandle {
		t.Fatalf("expected a rotated receipt handle, got %q and %q", firstHandle, secondHandle)
	}

	if err := service.DeleteMessage("queue-a", firstHandle); err == nil {
		t.Fatal("expected stale receipt handle delete to fail")
	}
	if err := service.DeleteMessage("queue-a", secondHandle); err != nil {
		t.Fatalf("delete with current handle: %v", err)
	}
}

func TestSQSServiceChangeMessageVisibilityPostponesRedelivery(t *testing.T) {
	t.Helper()

	now := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	clock := newManualClock(now)
	service := newServiceWithClock(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name: "queue-a",
				Attributes: map[string]string{
					"VisibilityTimeout": "30",
				},
			},
		},
		Messages: []domain.Message{
			{
				Queue:     "queue-a",
				MessageID: "message-1",
				Body:      "payload",
				SentAt:    now,
			},
		},
	}, nil, clock)

	first, err := service.ReceiveMessage("queue-a", 1, 0)
	if err != nil {
		t.Fatalf("first receive: %v", err)
	}
	handle := CurrentReceiptHandle(first[0])
	if handle == "" {
		t.Fatal("expected first receive to issue a receipt handle")
	}

	if err := service.ChangeMessageVisibility("queue-a", handle, 2*time.Minute); err != nil {
		t.Fatalf("change visibility: %v", err)
	}

	clock.Sleep(31 * time.Second)
	messages, err := service.ReceiveMessage("queue-a", 1, 0)
	if err != nil {
		t.Fatalf("receive before extended visibility expires: %v", err)
	}
	if got, want := len(messages), 0; got != want {
		t.Fatalf("expected message to remain hidden, got %d message(s)", got)
	}

	clock.Sleep(90 * time.Second)
	messages, err = service.ReceiveMessage("queue-a", 1, 0)
	if err != nil {
		t.Fatalf("receive after extended visibility expires: %v", err)
	}
	if got, want := len(messages), 1; got != want {
		t.Fatalf("expected message to redeliver after visibility change, got %d message(s)", got)
	}
}

type manualClock struct {
	now        time.Time
	sleepCount int
	totalSleep time.Duration
}

func newManualClock(now time.Time) *manualClock {
	return &manualClock{now: now}
}

func (c *manualClock) Now() time.Time {
	return c.now
}

func (c *manualClock) Sleep(duration time.Duration) {
	if duration <= 0 {
		return
	}
	c.sleepCount++
	c.totalSleep += duration
	c.now = c.now.Add(duration)
}

func (c *manualClock) SleepCount() int {
	return c.sleepCount
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
