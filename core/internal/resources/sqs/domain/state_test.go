package domain

import (
	"testing"
	"time"
)

func TestStateSnapshotCopiesLiveData(t *testing.T) {
	t.Helper()

	state := NewState()
	state.Queues = append(state.Queues, Queue{
		Name: "queue-a",
		URL:  "https://example.invalid/queue-a",
		Attributes: map[string]string{
			"VisibilityTimeout": "30",
		},
		OrderingHint: "fifo",
		Recovery: QueueRecovery{
			DeadLetterQueue: "queue-dlq",
			Policy: map[string]string{
				"max_receive_count": "5",
			},
		},
		CreatedAt: time.Date(2026, time.April, 19, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.April, 19, 10, 1, 0, 0, time.UTC),
		DeletedAt: time.Date(2026, time.April, 19, 10, 2, 0, 0, time.UTC),
		PurgedAt:  time.Date(2026, time.April, 19, 10, 3, 0, 0, time.UTC),
	})
	state.Messages = append(state.Messages, Message{
		Queue:                 "queue-a",
		MessageID:             "message-1",
		Body:                  "payload",
		Attributes:            map[string]string{"foo": "bar"},
		Metadata:              map[string]string{"trace": "abc"},
		Tags:                  []string{"alpha", "beta"},
		ReceiptKeys:           []string{"r-1", "r-2"},
		MessageGroupID:        "group-a",
		SequenceNumber:        17,
		BatchID:               "batch-a",
		BatchEntryID:          "entry-a",
		BatchEntryIndex:       1,
		BatchEntryCount:       3,
		DeadLetterQueue:       "queue-dlq",
		DeadLetterSourceQueue: "queue-a",
		DeadLetteredAt:        time.Date(2026, time.April, 19, 10, 2, 30, 0, time.UTC),
		SentAt:                time.Date(2026, time.April, 19, 10, 2, 0, 0, time.UTC),
		Recovery: MessageRecovery{
			Attempts: 2,
			Detail:   map[string]string{"reason": "retry"},
		},
	})
	state.RecoveryMetadata["queue-a/message-1"] = RecoveryMetadata{
		Queue:   "queue-a",
		Message: "message-1",
		Detail:  map[string]string{"state": "pending"},
	}

	snapshot := state.Snapshot()

	queues := snapshot["queues"].([]any)
	queues[0].(map[string]any)["name"] = "mutated"
	queues[0].(map[string]any)["attributes"].(map[string]any)["VisibilityTimeout"] = "99"
	queues[0].(map[string]any)["recovery"].(map[string]any)["dead_letter_queue"] = "mutated-dlq"
	queues[0].(map[string]any)["ordering_hint"] = "mutated"

	messages := snapshot["messages"].([]any)
	messages[0].(map[string]any)["body"] = "mutated"
	messages[0].(map[string]any)["tags"].([]string)[0] = "mutated"
	messages[0].(map[string]any)["metadata"].(map[string]any)["trace"] = "mutated"
	messages[0].(map[string]any)["message_group_id"] = "mutated"
	messages[0].(map[string]any)["dead_letter_queue"] = "mutated"
	messages[0].(map[string]any)["receipt_keys"].([]string)[0] = "mutated"

	recovery := snapshot["recovery_metadata"].(map[string]any)
	recovery["queue-a/message-1"].(map[string]any)["queue"] = "mutated"

	if got, want := state.Queues[0].Name, "queue-a"; got != want {
		t.Fatalf("queue name was aliased: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].Attributes["VisibilityTimeout"], "30"; got != want {
		t.Fatalf("queue attributes were aliased: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].Recovery.DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("queue recovery metadata was aliased: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].OrderingHint, "fifo"; got != want {
		t.Fatalf("queue ordering hint was aliased: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].DeletedAt, time.Date(2026, time.April, 19, 10, 2, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("queue deleted at was aliased: got %v want %v", got, want)
	}
	if got, want := state.Queues[0].PurgedAt, time.Date(2026, time.April, 19, 10, 3, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("queue purged at was aliased: got %v want %v", got, want)
	}
	if got, want := state.Messages[0].Body, "payload"; got != want {
		t.Fatalf("message body was aliased: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].Tags[0], "alpha"; got != want {
		t.Fatalf("message tags were aliased: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].Metadata["trace"], "abc"; got != want {
		t.Fatalf("message metadata was aliased: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].MessageGroupID, "group-a"; got != want {
		t.Fatalf("message group id was aliased: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("dead letter queue was aliased: got %q want %q", got, want)
	}
	if got, want := state.RecoveryMetadata["queue-a/message-1"].Queue, "queue-a"; got != want {
		t.Fatalf("recovery metadata was aliased: got %q want %q", got, want)
	}
}

func TestStateCloneReturnsDeepCopy(t *testing.T) {
	t.Helper()

	state := NewState()
	state.Queues = append(state.Queues, Queue{
		Name: "queue-a",
		Attributes: map[string]string{
			"DelaySeconds": "0",
		},
		OrderingHint: "standard",
		DeletedAt:    time.Date(2026, time.April, 19, 11, 0, 0, 0, time.UTC),
		PurgedAt:     time.Date(2026, time.April, 19, 11, 1, 0, 0, time.UTC),
		Recovery: QueueRecovery{
			Policy: map[string]string{"enabled": "true"},
		},
	})
	state.Messages = append(state.Messages, Message{
		Queue:                 "queue-a",
		MessageID:             "message-1",
		Attributes:            map[string]string{"foo": "bar"},
		Metadata:              map[string]string{"baz": "qux"},
		Tags:                  []string{"alpha"},
		ReceiptKeys:           []string{"r-1"},
		MessageGroupID:        "group-a",
		SequenceNumber:        2,
		BatchID:               "batch-a",
		BatchEntryID:          "entry-a",
		BatchEntryIndex:       0,
		BatchEntryCount:       1,
		DeadLetterQueue:       "queue-dlq",
		DeadLetterSourceQueue: "queue-a",
		Recovery: MessageRecovery{
			Attempts: 1,
			Detail:   map[string]string{"reason": "initial"},
		},
	})
	state.RecoveryMetadata["queue-a/message-1"] = RecoveryMetadata{
		Queue:   "queue-a",
		Message: "message-1",
		Detail:  map[string]string{"state": "live"},
	}

	cloned := state.Clone()
	cloned.Queues[0].Attributes["DelaySeconds"] = "5"
	cloned.Queues[0].Recovery.Policy["enabled"] = "false"
	cloned.Messages[0].Tags[0] = "mutated"
	cloned.Messages[0].ReceiptKeys[0] = "mutated"
	cloned.Messages[0].MessageGroupID = "mutated"
	cloned.Messages[0].DeadLetterQueue = "mutated"
	cloned.Messages[0].Recovery.Detail["reason"] = "mutated"
	cloned.RecoveryMetadata["queue-a/message-1"] = RecoveryMetadata{
		Queue:   "changed",
		Message: "changed",
		Detail:  map[string]string{"state": "changed"},
	}

	if got, want := state.Queues[0].Attributes["DelaySeconds"], "0"; got != want {
		t.Fatalf("queue attributes were shared with clone: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].Recovery.Policy["enabled"], "true"; got != want {
		t.Fatalf("queue policy was shared with clone: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].DeletedAt, time.Date(2026, time.April, 19, 11, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("queue deleted at was shared with clone: got %v want %v", got, want)
	}
	if got, want := state.Queues[0].PurgedAt, time.Date(2026, time.April, 19, 11, 1, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("queue purged at was shared with clone: got %v want %v", got, want)
	}
	if got, want := state.Messages[0].Tags[0], "alpha"; got != want {
		t.Fatalf("message tags were shared with clone: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].ReceiptKeys[0], "r-1"; got != want {
		t.Fatalf("message receipt keys were shared with clone: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].MessageGroupID, "group-a"; got != want {
		t.Fatalf("message group id was shared with clone: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("dead letter queue was shared with clone: got %q want %q", got, want)
	}
	if got, want := state.Messages[0].Recovery.Detail["reason"], "initial"; got != want {
		t.Fatalf("message recovery detail was shared with clone: got %q want %q", got, want)
	}
	if got, want := state.RecoveryMetadata["queue-a/message-1"].Queue, "queue-a"; got != want {
		t.Fatalf("recovery metadata was shared with clone: got %q want %q", got, want)
	}
}

func TestStateKeepsRoomForRecoveryMetadataAndAttributes(t *testing.T) {
	t.Helper()

	state := NewState()
	if got, want := state.Service, "sqs"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := StateKey, "services/sqs"; got != want {
		t.Fatalf("unexpected state key: got %q want %q", got, want)
	}

	queue := Queue{
		Name: "queue-a",
		Attributes: map[string]string{
			"RedrivePolicy": "present",
		},
		OrderingHint: "fifo",
		DeletedAt:    time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC),
		PurgedAt:     time.Date(2026, time.April, 19, 12, 1, 0, 0, time.UTC),
		Recovery: QueueRecovery{
			DeadLetterQueue: "queue-dlq",
		},
	}
	state.Queues = append(state.Queues, queue)
	state.RecoveryMetadata["queue-a"] = RecoveryMetadata{
		Queue:  queue.Name,
		Detail: map[string]string{"state": "ready"},
	}

	if got, want := state.Queues[0].Recovery.DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("unexpected queue recovery value: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].OrderingHint, "fifo"; got != want {
		t.Fatalf("unexpected queue ordering hint: got %q want %q", got, want)
	}
	if got, want := state.Queues[0].DeletedAt, time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("unexpected queue deleted at: got %v want %v", got, want)
	}
	if got, want := state.Queues[0].PurgedAt, time.Date(2026, time.April, 19, 12, 1, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("unexpected queue purged at: got %v want %v", got, want)
	}
	if got, want := state.RecoveryMetadata["queue-a"].Detail["state"], "ready"; got != want {
		t.Fatalf("unexpected recovery detail: got %q want %q", got, want)
	}
}
