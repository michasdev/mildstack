package sns_test

import (
	"context"
	"errors"
	"testing"

	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func TestSNSTopicLifecycleIdempotencyAndCascadeDelete(t *testing.T) {
	t.Helper()

	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: "integration-topic-lifecycle",
	})
	if err != nil {
		t.Fatalf("new sns service: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	firstTopic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	secondTopic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic idempotent retry: %v", err)
	}
	if got, want := secondTopic.ARN, firstTopic.ARN; got != want {
		t.Fatalf("expected idempotent topic arn: got %q want %q", got, want)
	}

	subscription, err := service.Subscribe(firstTopic.ARN, "http", "http://127.0.0.1:7777/sns", nil, true)
	if err != nil {
		t.Fatalf("subscribe endpoint: %v", err)
	}
	if subscription.Subscription.ARN == "" {
		t.Fatal("expected subscription arn")
	}

	allSubscriptions, _, err := service.ListSubscriptions("")
	if err != nil {
		t.Fatalf("list subscriptions: %v", err)
	}
	if got, want := len(allSubscriptions), 1; got != want {
		t.Fatalf("unexpected subscription count before delete: got %d want %d", got, want)
	}

	if err := service.DeleteTopic(firstTopic.ARN); err != nil {
		t.Fatalf("delete topic: %v", err)
	}

	afterDeleteSubscriptions, _, err := service.ListSubscriptions("")
	if err != nil {
		t.Fatalf("list subscriptions after topic delete: %v", err)
	}
	if got, want := len(afterDeleteSubscriptions), 0; got != want {
		t.Fatalf("expected cascade delete of subscriptions: got %d remaining", got)
	}

	if _, err := service.GetSubscriptionAttributes(subscription.Subscription.ARN); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected subscription to be removed after topic delete, got err=%v", err)
	}
}
