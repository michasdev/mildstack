package sns_test

import (
	"context"
	"testing"

	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func TestSNSSubscriptionPendingTransitionsToConfirmed(t *testing.T) {
	t.Helper()

	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: "integration-subscription-confirm",
	})
	if err != nil {
		t.Fatalf("new sns service: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	topic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}

	subscribeOutput, err := service.Subscribe(topic.ARN, "http", "http://127.0.0.1:7777/sns", nil, false)
	if err != nil {
		t.Fatalf("subscribe pending: %v", err)
	}
	if got, want := subscribeOutput.ResponseSubscription, "pending confirmation"; got != want {
		t.Fatalf("unexpected subscribe response arn: got %q want %q", got, want)
	}
	if got, want := subscribeOutput.Subscription.Status, domain.SubscriptionStatusPendingConfirmation; got != want {
		t.Fatalf("unexpected initial subscription status: got %q want %q", got, want)
	}

	confirmed, err := service.ConfirmSubscription(topic.ARN, subscribeOutput.Subscription.Token)
	if err != nil {
		t.Fatalf("confirm subscription: %v", err)
	}
	if got, want := confirmed.Status, domain.SubscriptionStatusConfirmed; got != want {
		t.Fatalf("unexpected confirmed status: got %q want %q", got, want)
	}

	attrs, err := service.GetSubscriptionAttributes(confirmed.ARN)
	if err != nil {
		t.Fatalf("get subscription attributes: %v", err)
	}
	if got, want := attrs["PendingConfirmation"], "false"; got != want {
		t.Fatalf("unexpected pending confirmation attribute: got %q want %q", got, want)
	}

	byTopic, _, err := service.ListSubscriptionsByTopic(topic.ARN, "")
	if err != nil {
		t.Fatalf("list subscriptions by topic: %v", err)
	}
	if got, want := len(byTopic), 1; got != want {
		t.Fatalf("unexpected subscriptions by topic count: got %d want %d", got, want)
	}
	if got, want := byTopic[0].Status, domain.SubscriptionStatusConfirmed; got != want {
		t.Fatalf("unexpected by-topic status after confirm: got %q want %q", got, want)
	}
}
