package domain

import "testing"

func TestFilterPolicyMatchesMessageAttributesScope(t *testing.T) {
	message := PublishedMessage{
		Payload: "hello",
		MessageAttributes: map[string]MessageAttributeValue{
			"eventType": {DataType: "String", StringValue: "order.created"},
			"source":    {DataType: "String", StringValue: "checkout"},
		},
	}

	matched, err := EvaluateSubscriptionFilter(`{"eventType":["order.created"],"source":[{"prefix":"check"}]}`, "MessageAttributes", message)
	if err != nil {
		t.Fatalf("evaluate filter policy: %v", err)
	}
	if !matched {
		t.Fatal("expected filter policy to match message attributes")
	}
}

func TestFilterPolicyRejectsNonMatchingMessageAttributes(t *testing.T) {
	message := PublishedMessage{
		Payload: "hello",
		MessageAttributes: map[string]MessageAttributeValue{
			"eventType": {DataType: "String", StringValue: "order.cancelled"},
		},
	}

	matched, err := EvaluateSubscriptionFilter(`{"eventType":["order.created"]}`, "MessageAttributes", message)
	if err != nil {
		t.Fatalf("evaluate filter policy: %v", err)
	}
	if matched {
		t.Fatal("expected filter policy mismatch")
	}
}

func TestFilterPolicyMatchesMessageBodyScope(t *testing.T) {
	message := PublishedMessage{Payload: `{"eventType":"order.created","amount":42}`}

	matched, err := EvaluateSubscriptionFilter(`{"eventType":["order.created"],"amount":[{"numeric":[">",40]}]}`, "MessageBody", message)
	if err != nil {
		t.Fatalf("evaluate body scope filter: %v", err)
	}
	if !matched {
		t.Fatal("expected message body filter policy to match")
	}
}

func TestSubscriptionRawMessageDeliveryEnabled(t *testing.T) {
	subscription := Subscription{Attributes: map[string]string{"RawMessageDelivery": "true"}}
	if !subscription.RawMessageDeliveryEnabled() {
		t.Fatal("expected raw delivery to be enabled")
	}

	subscription = Subscription{Attributes: map[string]string{"RawMessageDelivery": "false"}}
	if subscription.RawMessageDeliveryEnabled() {
		t.Fatal("expected raw delivery to be disabled")
	}
}
