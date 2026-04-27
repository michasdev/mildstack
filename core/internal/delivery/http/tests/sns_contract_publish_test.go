package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSNSContractPublishAndPublishBatchXML(t *testing.T) {
	t.Helper()

	router, service := newSNSContractHarness(t)
	topic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic fixture: %v", err)
	}

	subscribeOutput, err := service.Subscribe(topic.ARN, "http", "http://127.0.0.1:7777/sns", nil, true)
	if err != nil {
		t.Fatalf("subscribe fixture: %v", err)
	}
	if _, err := service.ConfirmSubscription(topic.ARN, subscribeOutput.Subscription.Token); err != nil {
		t.Fatalf("confirm fixture subscription: %v", err)
	}

	publishParams := url.Values{}
	publishParams.Set("Action", "Publish")
	publishParams.Set("Version", "2010-03-31")
	publishParams.Set("TopicArn", topic.ARN)
	publishParams.Set("Message", "hello world")

	publishRecorder := performSNSQuery(t, router, http.MethodGet, publishParams.Encode())
	if got, want := publishRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected publish status: got %d want %d", got, want)
	}
	publishBody := publishRecorder.Body.String()
	if !strings.Contains(publishBody, "<PublishResponse") {
		t.Fatalf("expected publish response envelope, got %q", publishBody)
	}
	if !strings.Contains(publishBody, "<MessageId>") {
		t.Fatalf("expected message id in publish response, got %q", publishBody)
	}

	batchParams := url.Values{}
	batchParams.Set("Action", "PublishBatch")
	batchParams.Set("Version", "2010-03-31")
	batchParams.Set("TopicArn", topic.ARN)
	batchParams.Set("PublishBatchRequestEntries.member.1.Id", "ok1")
	batchParams.Set("PublishBatchRequestEntries.member.1.Message", "first")
	batchParams.Set("PublishBatchRequestEntries.member.2.Id", "bad1")
	batchParams.Set("PublishBatchRequestEntries.member.2.Message", "")

	batchRecorder := performSNSQuery(t, router, http.MethodGet, batchParams.Encode())
	if got, want := batchRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected publish batch status: got %d want %d", got, want)
	}
	batchBody := batchRecorder.Body.String()
	if !strings.Contains(batchBody, "<PublishBatchResponse") {
		t.Fatalf("expected publish batch response envelope, got %q", batchBody)
	}
	if !strings.Contains(batchBody, "<Successful>") || !strings.Contains(batchBody, "<Id>ok1</Id>") {
		t.Fatalf("expected successful batch entry in response, got %q", batchBody)
	}
	if !strings.Contains(batchBody, "<Failed>") || !strings.Contains(batchBody, "<Id>bad1</Id>") {
		t.Fatalf("expected failed batch entry in response, got %q", batchBody)
	}
	if !strings.Contains(batchBody, "<SenderFault>true</SenderFault>") {
		t.Fatalf("expected sender fault marker in failed batch entry, got %q", batchBody)
	}
}

func TestSNSContractPublishFIFOSequenceNumberXML(t *testing.T) {
	t.Helper()

	router, _ := newSNSContractHarness(t)

	createFIFOParams := url.Values{}
	createFIFOParams.Set("Action", "CreateTopic")
	createFIFOParams.Set("Version", "2010-03-31")
	createFIFOParams.Set("Name", "orders.fifo")
	createFIFOParams.Set("Attributes.entry.1.key", "FifoTopic")
	createFIFOParams.Set("Attributes.entry.1.value", "true")

	createRecorder := performSNSQuery(t, router, http.MethodGet, createFIFOParams.Encode())
	if got, want := createRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create fifo status: got %d want %d", got, want)
	}

	publishParams := url.Values{}
	publishParams.Set("Action", "Publish")
	publishParams.Set("Version", "2010-03-31")
	publishParams.Set("TopicArn", "arn:aws:sns:us-east-1:00000000000:orders.fifo")
	publishParams.Set("Message", "hello fifo")
	publishParams.Set("MessageGroupId", "group-1")
	publishParams.Set("MessageDeduplicationId", "dedup-1")

	publishRecorder := performSNSQuery(t, router, http.MethodGet, publishParams.Encode())
	if got, want := publishRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected fifo publish status: got %d want %d", got, want)
	}
	body := publishRecorder.Body.String()
	if !strings.Contains(body, "<SequenceNumber>") {
		t.Fatalf("expected sequence number in fifo publish response, got %q", body)
	}
}
