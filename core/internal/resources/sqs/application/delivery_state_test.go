package application

import (
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

func TestDeliveryHelpersCoverDelayVisibilityLeaseAndReceiptRotation(t *testing.T) {
	t.Helper()

	now := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	queue := domain.Queue{
		Name: "queue-a",
		Attributes: map[string]string{
			"VisibilityTimeout": "30",
		},
	}

	delayed := domain.Message{
		Queue:       "queue-a",
		MessageID:   "message-delayed",
		AvailableAt: now.Add(2 * time.Minute),
	}
	if !IsDelayed(delayed, now) {
		t.Fatal("expected delayed message to be marked delayed")
	}
	if (DeliveryView{Queue: queue, Message: delayed, Now: now}).Visible() {
		t.Fatal("expected delayed message to be hidden")
	}
	if (DeliveryView{Queue: queue, Message: delayed, Now: now}).Redeliver() {
		t.Fatal("expected delayed message to not redeliver yet")
	}

	visible := domain.Message{
		Queue:     "queue-a",
		MessageID: "message-visible",
	}
	if !IsVisible(visible, queue, now) {
		t.Fatal("expected new message to be visible")
	}
	if (DeliveryView{Queue: queue, Message: visible, Now: now}).Invisible() {
		t.Fatal("expected new message to not be invisible")
	}

	inflight := domain.Message{
		Queue:       "queue-a",
		MessageID:   "message-inflight",
		ReceivedAt:  now.Add(-20 * time.Second),
		ReceiptKeys: []string{"r-1", "r-2"},
		Metadata: map[string]string{
			leaseTimeoutMetadataKey: "30",
		},
	}
	if !(DeliveryView{Queue: queue, Message: inflight, Now: now}).Invisible() {
		t.Fatal("expected in-flight message to be invisible")
	}
	if got, want := CurrentReceiptHandle(inflight), "r-2"; got != want {
		t.Fatalf("unexpected receipt handle: got %q want %q", got, want)
	}
	if (DeliveryView{Queue: queue, Message: inflight, Now: now.Add(9 * time.Second)}).Redeliver() {
		t.Fatal("expected lease to remain active before timeout")
	}
	if !CanRedeliver(inflight, queue, now.Add(10*time.Second)) {
		t.Fatal("expected lease to redeliver once timeout expires")
	}
}
