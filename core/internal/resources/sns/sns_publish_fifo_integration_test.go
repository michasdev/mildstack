package sns_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func TestSNSPublishFIFOAppliesDedupAndSequenceNumbering(t *testing.T) {
	t.Helper()

	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer endpoint.Close()

	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: "integration-publish-fifo",
	})
	if err != nil {
		t.Fatalf("new sns service: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	topic, err := service.CreateTopic("orders.fifo", map[string]string{
		"FifoTopic":                 "true",
		"ContentBasedDeduplication": "false",
	})
	if err != nil {
		t.Fatalf("create fifo topic: %v", err)
	}

	subscription, err := service.Subscribe(topic.ARN, "http", endpoint.URL, nil, true)
	if err != nil {
		t.Fatalf("subscribe endpoint: %v", err)
	}
	if _, err := service.ConfirmSubscription(topic.ARN, subscription.Subscription.Token); err != nil {
		t.Fatalf("confirm subscription: %v", err)
	}

	first, err := service.Publish(domain.PublishRequest{
		TopicARN:               topic.ARN,
		Message:                "first",
		MessageGroupID:         "group-a",
		MessageDeduplicationID: "dedup-1",
	})
	if err != nil {
		t.Fatalf("publish first fifo message: %v", err)
	}
	if first.SequenceNumber == "" {
		t.Fatal("expected first sequence number")
	}

	duplicate, err := service.Publish(domain.PublishRequest{
		TopicARN:               topic.ARN,
		Message:                "duplicate",
		MessageGroupID:         "group-a",
		MessageDeduplicationID: "dedup-1",
	})
	if err != nil {
		t.Fatalf("publish duplicate fifo message: %v", err)
	}

	third, err := service.Publish(domain.PublishRequest{
		TopicARN:               topic.ARN,
		Message:                "third",
		MessageGroupID:         "group-a",
		MessageDeduplicationID: "dedup-2",
	})
	if err != nil {
		t.Fatalf("publish third fifo message: %v", err)
	}
	if third.SequenceNumber == "" {
		t.Fatal("expected third sequence number")
	}
	if third.SequenceNumber <= first.SequenceNumber {
		t.Fatalf("expected sequence to increase: first=%q third=%q", first.SequenceNumber, third.SequenceNumber)
	}

	firstAttempts, err := service.ListDeliveryAttemptsByMessageID(first.MessageID)
	if err != nil {
		t.Fatalf("list first attempts: %v", err)
	}
	if got, want := len(firstAttempts), 1; got != want {
		t.Fatalf("unexpected first attempt count: got %d want %d", got, want)
	}

	duplicateAttempts, err := service.ListDeliveryAttemptsByMessageID(duplicate.MessageID)
	if err != nil {
		t.Fatalf("list duplicate attempts: %v", err)
	}
	if got, want := len(duplicateAttempts), 1; got != want {
		t.Fatalf("expected deduped publish to return first message id with a single delivery attempt: got %d", got)
	}

	thirdAttempts, err := service.ListDeliveryAttemptsByMessageID(third.MessageID)
	if err != nil {
		t.Fatalf("list third attempts: %v", err)
	}
	if got, want := len(thirdAttempts), 1; got != want {
		t.Fatalf("unexpected third attempt count: got %d want %d", got, want)
	}
}
