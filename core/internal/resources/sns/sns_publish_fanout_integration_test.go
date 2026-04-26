package sns_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func TestSNSPublishFanOutRespectsFilterPolicies(t *testing.T) {
	t.Helper()

	deliveryBodies := make(chan string, 1)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		payload, _ := io.ReadAll(r.Body)
		deliveryBodies <- string(payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer endpoint.Close()

	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: "integration-publish-fanout",
	})
	if err != nil {
		t.Fatalf("new sns service: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	topic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}

	matching, err := service.Subscribe(topic.ARN, "http", endpoint.URL, map[string]string{
		"FilterPolicy":       `{"eventType":["order.created"]}`,
		"FilterPolicyScope":  "MessageAttributes",
		"RawMessageDelivery": "true",
	}, true)
	if err != nil {
		t.Fatalf("subscribe matching endpoint: %v", err)
	}
	if _, err := service.ConfirmSubscription(topic.ARN, matching.Subscription.Token); err != nil {
		t.Fatalf("confirm matching subscription: %v", err)
	}

	nonMatching, err := service.Subscribe(topic.ARN, "http", "http://127.0.0.1:8899/ignored", map[string]string{
		"FilterPolicy":      `{"eventType":["order.cancelled"]}`,
		"FilterPolicyScope": "MessageAttributes",
	}, true)
	if err != nil {
		t.Fatalf("subscribe non-matching endpoint: %v", err)
	}
	if _, err := service.ConfirmSubscription(topic.ARN, nonMatching.Subscription.Token); err != nil {
		t.Fatalf("confirm non-matching subscription: %v", err)
	}

	publishResult, err := service.Publish(domain.PublishRequest{
		TopicARN: topic.ARN,
		Message:  "hello from sns",
		MessageAttributes: map[string]domain.MessageAttributeValue{
			"eventType": {DataType: "String", StringValue: "order.created"},
		},
	})
	if err != nil {
		t.Fatalf("publish topic message: %v", err)
	}
	if publishResult.MessageID == "" {
		t.Fatal("expected publish result message id")
	}

	select {
	case body := <-deliveryBodies:
		if body != "hello from sns" {
			t.Fatalf("expected raw delivery payload, got %q", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for matching delivery attempt")
	}

	attempts, err := service.ListDeliveryAttemptsByMessageID(publishResult.MessageID)
	if err != nil {
		t.Fatalf("list delivery attempts: %v", err)
	}
	if got, want := len(attempts), 2; got != want {
		t.Fatalf("unexpected attempt count: got %d want %d", got, want)
	}

	statuses := map[string]int{}
	for _, attempt := range attempts {
		statuses[attempt.Status]++
	}
	if got, want := statuses[domain.DeliveryAttemptStatusDelivered], 1; got != want {
		t.Fatalf("unexpected delivered count: got %d want %d", got, want)
	}
	if got, want := statuses[domain.DeliveryAttemptStatusFilteredOut], 1; got != want {
		t.Fatalf("unexpected filtered count: got %d want %d", got, want)
	}
}
