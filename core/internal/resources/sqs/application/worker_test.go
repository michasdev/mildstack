package application

import (
	"context"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

func TestWorkerPollDetectsDelayedAndLeasedState(t *testing.T) {
	t.Helper()

	now := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
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
				MessageID:   "delayed",
				AvailableAt: now.Add(2 * time.Second),
			},
			{
				Queue:      "queue-a",
				MessageID:  "leased",
				ReceivedAt: now.Add(-25 * time.Second),
				Metadata: map[string]string{
					leaseTimeoutMetadataKey: "30",
				},
			},
		},
	}, nil, newManualClock(now))

	w := newWorker(service, service.clock)
	if got, want := w.poll(now), workerPollInterval; got != want {
		t.Fatalf("unexpected poll wait: got %v want %v", got, want)
	}

	leases := w.lease(now)
	if got, want := len(leases), 1; got != want {
		t.Fatalf("unexpected lease count: got %d want %d", got, want)
	}
	if got, want := leases[0].Message, "leased"; got != want {
		t.Fatalf("unexpected leased message: got %q want %q", got, want)
	}

	redeliverable := w.redeliver(now.Add(5 * time.Second))
	if got, want := len(redeliverable), 1; got != want {
		t.Fatalf("unexpected redeliverable count: got %d want %d", got, want)
	}
	if got, want := redeliverable[0].MessageID, "leased"; got != want {
		t.Fatalf("unexpected redeliverable message: got %q want %q", got, want)
	}
}

func TestWorkerAdvanceMovesExpiredMessagesToDeadLetterQueue(t *testing.T) {
	t.Helper()

	now := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	service := newServiceWithClock(domain.State{
		Service: "sqs",
		Queues: []domain.Queue{
			{
				Name:         "queue-a",
				OrderingHint: "fifo",
				Recovery: domain.QueueRecovery{
					DeadLetterQueue: "queue-dlq",
					Policy: map[string]string{
						"max_receive_count": "2",
					},
				},
			},
			{
				Name: "queue-dlq",
			},
		},
		Messages: []domain.Message{
			{
				Queue:      "queue-a",
				MessageID:  "message-1",
				ReceivedAt: now.Add(-31 * time.Second),
				Recovery: domain.MessageRecovery{
					Attempts: 2,
				},
			},
		},
	}, nil, newManualClock(now))

	w := newWorker(service, service.clock)
	if got := w.advance(now); got != 1 {
		t.Fatalf("unexpected dead-letter mutation count: got %d want %d", got, 1)
	}
	if got, want := service.state.Messages[0].Queue, "queue-dlq"; got != want {
		t.Fatalf("unexpected dead-letter queue assignment: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("unexpected dead-letter marker: got %q want %q", got, want)
	}
	if got, want := service.state.Messages[0].DeadLetterSourceQueue, "queue-a"; got != want {
		t.Fatalf("unexpected dead-letter source: got %q want %q", got, want)
	}
	if got, want := service.state.RecoveryMetadata["queue-a/message-1"].Queue, "queue-dlq"; got != want {
		t.Fatalf("unexpected recovery metadata queue: got %q want %q", got, want)
	}
}

func TestWorkerStopTerminatesCleanly(t *testing.T) {
	t.Helper()

	service := newService(domain.NewState(), nil)
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start service: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := service.Stop(stopCtx); err != nil {
		t.Fatalf("stop service: %v", err)
	}
}
