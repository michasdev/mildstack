package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSNSContractSubscribeConfirmAndUnsubscribeXML(t *testing.T) {
	t.Helper()

	router, service := newSNSContractHarness(t)
	topic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic fixture: %v", err)
	}

	subscribeParams := url.Values{}
	subscribeParams.Set("Action", "Subscribe")
	subscribeParams.Set("Version", "2010-03-31")
	subscribeParams.Set("TopicArn", topic.ARN)
	subscribeParams.Set("Protocol", "http")
	subscribeParams.Set("Endpoint", "http://127.0.0.1:7777/sns")

	subscribeRecorder := performSNSQuery(t, router, http.MethodGet, subscribeParams.Encode())
	if got, want := subscribeRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected subscribe status: got %d want %d", got, want)
	}
	subscribeBody := subscribeRecorder.Body.String()
	if !strings.Contains(subscribeBody, "<SubscribeResponse") {
		t.Fatalf("expected subscribe response envelope, got %q", subscribeBody)
	}
	if !strings.Contains(subscribeBody, "<SubscriptionArn>pending confirmation</SubscriptionArn>") {
		t.Fatalf("expected pending confirmation subscription arn, got %q", subscribeBody)
	}

	listByTopicParams := url.Values{}
	listByTopicParams.Set("Action", "ListSubscriptionsByTopic")
	listByTopicParams.Set("Version", "2010-03-31")
	listByTopicParams.Set("TopicArn", topic.ARN)

	listByTopicRecorder := performSNSQuery(t, router, http.MethodGet, listByTopicParams.Encode())
	if got, want := listByTopicRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list-by-topic status: got %d want %d", got, want)
	}
	listByTopicBody := listByTopicRecorder.Body.String()
	if !strings.Contains(listByTopicBody, "<ListSubscriptionsByTopicResult>") {
		t.Fatalf("expected list-by-topic result wrapper, got %q", listByTopicBody)
	}
	if !strings.Contains(listByTopicBody, "<Endpoint>http://127.0.0.1:7777/sns</Endpoint>") {
		t.Fatalf("expected subscribed endpoint in list-by-topic response, got %q", listByTopicBody)
	}

	subscriptions, _, err := service.ListSubscriptionsByTopic(topic.ARN, "")
	if err != nil {
		t.Fatalf("list subscriptions by topic: %v", err)
	}
	if got, want := len(subscriptions), 1; got != want {
		t.Fatalf("unexpected subscription count: got %d want %d", got, want)
	}
	token := subscriptions[0].Token
	if token == "" {
		t.Fatal("expected pending subscription token")
	}

	confirmParams := url.Values{}
	confirmParams.Set("Action", "ConfirmSubscription")
	confirmParams.Set("Version", "2010-03-31")
	confirmParams.Set("TopicArn", topic.ARN)
	confirmParams.Set("Token", token)

	confirmRecorder := performSNSQuery(t, router, http.MethodGet, confirmParams.Encode())
	if got, want := confirmRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected confirm status: got %d want %d", got, want)
	}
	confirmBody := confirmRecorder.Body.String()
	if !strings.Contains(confirmBody, "<ConfirmSubscriptionResponse") {
		t.Fatalf("expected confirm response envelope, got %q", confirmBody)
	}
	if !strings.Contains(confirmBody, "<SubscriptionArn>"+subscriptions[0].ARN+"</SubscriptionArn>") {
		t.Fatalf("expected confirmed subscription arn, got %q", confirmBody)
	}

	unsubscribeParams := url.Values{}
	unsubscribeParams.Set("Action", "Unsubscribe")
	unsubscribeParams.Set("Version", "2010-03-31")
	unsubscribeParams.Set("SubscriptionArn", subscriptions[0].ARN)

	unsubscribeRecorder := performSNSQuery(t, router, http.MethodGet, unsubscribeParams.Encode())
	if got, want := unsubscribeRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected unsubscribe status: got %d want %d", got, want)
	}
	if body := unsubscribeRecorder.Body.String(); !strings.Contains(body, "<UnsubscribeResponse") {
		t.Fatalf("expected unsubscribe response envelope, got %q", body)
	}
}
