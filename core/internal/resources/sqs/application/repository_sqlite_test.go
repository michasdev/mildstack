package application

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

func TestResolveStoragePath(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	path, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}

	want := filepath.Join(baseDir, "instances", "instance-a", "sqs")
	if got, want := path, want; got != want {
		t.Fatalf("unexpected storage path: got %q want %q", got, want)
	}
}

func TestSQLiteRepositoryLoadsDefaultStateFromNewStorage(t *testing.T) {
	t.Helper()

	repo := mustOpenSQLiteRepository(t, "instance-a")
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close repo: %v", err)
		}
	}()

	state, err := repo.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if got, want := state.Service, "sqs"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := len(state.Queues), 0; got != want {
		t.Fatalf("unexpected queue count: got %d want %d", got, want)
	}
	if got, want := len(state.Messages), 0; got != want {
		t.Fatalf("unexpected message count: got %d want %d", got, want)
	}
}

func TestSQLiteRepositoryPersistsQueueAndMessageStateAcrossRestart(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	repo := mustOpenSQLiteRepositoryAt(t, baseDir, "instance-a")
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close repo: %v", err)
		}
	}()

	createdAt := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	state := domain.NewState()
	state.Queues = append(state.Queues, domain.Queue{
		Name: "queue-a",
		URL:  "https://example.invalid/queue-a",
		Attributes: map[string]string{
			"VisibilityTimeout": "30",
			"DelaySeconds":      "0",
		},
		OrderingHint: "fifo",
		Recovery: domain.QueueRecovery{
			DeadLetterQueue: "queue-dlq",
			Policy: map[string]string{
				"max_receive_count": "5",
			},
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt.Add(time.Minute),
		DeletedAt: createdAt.Add(2 * time.Minute),
		PurgedAt:  createdAt.Add(3 * time.Minute),
	})
	state.Messages = append(state.Messages, domain.Message{
		Queue:                 "queue-a",
		MessageID:             "message-1",
		Body:                  "payload",
		Attributes:            map[string]string{"foo": "bar"},
		Metadata:              map[string]string{"trace": "abc"},
		Tags:                  []string{"alpha", "beta"},
		ReceiptKeys:           []string{"r-1", "r-2"},
		MessageGroupID:        "group-a",
		SequenceNumber:        42,
		BatchID:               "batch-a",
		BatchEntryID:          "entry-a",
		BatchEntryIndex:       1,
		BatchEntryCount:       3,
		DeadLetterQueue:       "queue-dlq",
		DeadLetterSourceQueue: "queue-a",
		DeadLetteredAt:        createdAt.Add(5 * time.Minute),
		SentAt:                createdAt.Add(2 * time.Minute),
		AvailableAt:           createdAt.Add(3 * time.Minute),
		ReceivedAt:            createdAt.Add(4 * time.Minute),
		Recovery: domain.MessageRecovery{
			Attempts: 2,
			Detail:   map[string]string{"state": "pending"},
		},
	})
	state.RecoveryMetadata["queue-a/message-1"] = domain.RecoveryMetadata{
		Queue:   "queue-a",
		Message: "message-1",
		Detail:  map[string]string{"reason": "retry"},
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close repo after save: %v", err)
	}

	reopened := mustOpenSQLiteRepositoryAt(t, baseDir, "instance-a")
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Fatalf("close reopened repo: %v", err)
		}
	}()

	loaded, err := reopened.Load()
	if err != nil {
		t.Fatalf("load restarted state: %v", err)
	}
	if got, want := len(loaded.Queues), 1; got != want {
		t.Fatalf("unexpected queue count after restart: got %d want %d", got, want)
	}
	if got, want := len(loaded.Messages), 1; got != want {
		t.Fatalf("unexpected message count after restart: got %d want %d", got, want)
	}

	queue := loaded.Queues[0]
	if got, want := queue.Name, "queue-a"; got != want {
		t.Fatalf("unexpected queue name after restart: got %q want %q", got, want)
	}
	if got, want := queue.Attributes["VisibilityTimeout"], "30"; got != want {
		t.Fatalf("unexpected queue attribute after restart: got %q want %q", got, want)
	}
	if got, want := queue.OrderingHint, "fifo"; got != want {
		t.Fatalf("unexpected ordering hint after restart: got %q want %q", got, want)
	}
	if got, want := queue.Recovery.DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("unexpected dead-letter queue after restart: got %q want %q", got, want)
	}
	if got, want := queue.DeletedAt, createdAt.Add(2*time.Minute); !got.Equal(want) {
		t.Fatalf("unexpected deleted_at after restart: got %v want %v", got, want)
	}
	if got, want := queue.PurgedAt, createdAt.Add(3*time.Minute); !got.Equal(want) {
		t.Fatalf("unexpected purged_at after restart: got %v want %v", got, want)
	}

	message := loaded.Messages[0]
	if got, want := message.Body, "payload"; got != want {
		t.Fatalf("unexpected message body after restart: got %q want %q", got, want)
	}
	if got, want := message.Tags[0], "alpha"; got != want {
		t.Fatalf("unexpected message tag after restart: got %q want %q", got, want)
	}
	if got, want := message.AvailableAt, createdAt.Add(3*time.Minute); !got.Equal(want) {
		t.Fatalf("unexpected available_at after restart: got %v want %v", got, want)
	}
	if got, want := message.ReceivedAt, createdAt.Add(4*time.Minute); !got.Equal(want) {
		t.Fatalf("unexpected received_at after restart: got %v want %v", got, want)
	}
	if got, want := len(message.ReceiptKeys), 2; got != want {
		t.Fatalf("unexpected receipt key count after restart: got %d want %d", got, want)
	}
	if got, want := message.ReceiptKeys[1], "r-2"; got != want {
		t.Fatalf("unexpected latest receipt key after restart: got %q want %q", got, want)
	}
	if got, want := message.MessageGroupID, "group-a"; got != want {
		t.Fatalf("unexpected message group after restart: got %q want %q", got, want)
	}
	if got, want := message.SequenceNumber, int64(42); got != want {
		t.Fatalf("unexpected message sequence after restart: got %d want %d", got, want)
	}
	if got, want := message.BatchID, "batch-a"; got != want {
		t.Fatalf("unexpected batch id after restart: got %q want %q", got, want)
	}
	if got, want := message.DeadLetterQueue, "queue-dlq"; got != want {
		t.Fatalf("unexpected dead letter queue after restart: got %q want %q", got, want)
	}
	if got, want := message.DeadLetterSourceQueue, "queue-a"; got != want {
		t.Fatalf("unexpected dead letter source after restart: got %q want %q", got, want)
	}
	if got, want := message.DeadLetteredAt, createdAt.Add(5*time.Minute); !got.Equal(want) {
		t.Fatalf("unexpected dead letter time after restart: got %v want %v", got, want)
	}
	if got, want := message.Recovery.Detail["state"], "pending"; got != want {
		t.Fatalf("unexpected message recovery detail after restart: got %q want %q", got, want)
	}
	if got, want := loaded.RecoveryMetadata["queue-a/message-1"].Detail["reason"], "retry"; got != want {
		t.Fatalf("unexpected recovery metadata after restart: got %q want %q", got, want)
	}

	statePath := filepath.Join(baseDir, "instances", "instance-a", "sqs", sqliteFileName)
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected sqlite database to exist: %v", err)
	}
}

func TestSQLiteRepositoryCloseIsIdempotentAndRejectsFurtherUse(t *testing.T) {
	t.Helper()

	repo := mustOpenSQLiteRepository(t, "instance-a")
	if err := repo.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	if _, err := repo.Load(); err == nil {
		t.Fatal("expected load after close to fail")
	}
	if err := repo.Save(domain.NewState()); err == nil {
		t.Fatal("expected save after close to fail")
	}
}

func mustOpenSQLiteRepository(t *testing.T, instanceID string) *SQLiteRepository {
	t.Helper()
	return mustOpenSQLiteRepositoryAt(t, t.TempDir(), instanceID)
}

func mustOpenSQLiteRepositoryAt(t *testing.T, baseDir, instanceID string) *SQLiteRepository {
	t.Helper()

	storagePath, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: instanceID,
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}

	repo, err := NewSQLiteRepository(storagePath)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	return repo
}
