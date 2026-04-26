package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSNSContractCreateListAndGetTopicAttributesXML(t *testing.T) {
	t.Helper()

	router, _ := newSNSContractHarness(t)

	createParams := url.Values{}
	createParams.Set("Action", "CreateTopic")
	createParams.Set("Version", "2010-03-31")
	createParams.Set("Name", "orders")

	createRecorder := performSNSQuery(t, router, http.MethodGet, createParams.Encode())
	if got, want := createRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create status: got %d want %d", got, want)
	}
	if ct := createRecorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("unexpected create content type: %q", ct)
	}
	if body := createRecorder.Body.String(); !strings.Contains(body, "<CreateTopicResponse") {
		t.Fatalf("expected create response envelope, got %q", body)
	}
	if body := createRecorder.Body.String(); !strings.Contains(body, "<TopicArn>arn:aws:sns:us-east-1:00000000000:orders</TopicArn>") {
		t.Fatalf("expected topic arn in create response, got %q", body)
	}

	listParams := url.Values{}
	listParams.Set("Action", "ListTopics")
	listParams.Set("Version", "2010-03-31")

	listRecorder := performSNSQuery(t, router, http.MethodGet, listParams.Encode())
	if got, want := listRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list status: got %d want %d", got, want)
	}
	if body := listRecorder.Body.String(); !strings.Contains(body, "<ListTopicsResponse") {
		t.Fatalf("expected list response envelope, got %q", body)
	}
	if body := listRecorder.Body.String(); !strings.Contains(body, "<TopicArn>arn:aws:sns:us-east-1:00000000000:orders</TopicArn>") {
		t.Fatalf("expected topic arn in list response, got %q", body)
	}

	getParams := url.Values{}
	getParams.Set("Action", "GetTopicAttributes")
	getParams.Set("Version", "2010-03-31")
	getParams.Set("TopicArn", "arn:aws:sns:us-east-1:00000000000:orders")

	getRecorder := performSNSQuery(t, router, http.MethodGet, getParams.Encode())
	if got, want := getRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get attributes status: got %d want %d", got, want)
	}
	body := getRecorder.Body.String()
	if !strings.Contains(body, "<GetTopicAttributesResponse") {
		t.Fatalf("expected get attributes response envelope, got %q", body)
	}
	if !strings.Contains(body, "<key>Owner</key>") || !strings.Contains(body, "<value>00000000000</value>") {
		t.Fatalf("expected owner attribute in response, got %q", body)
	}
	if !strings.Contains(body, "<key>TopicArn</key>") || !strings.Contains(body, "<value>arn:aws:sns:us-east-1:00000000000:orders</value>") {
		t.Fatalf("expected topic arn attribute in response, got %q", body)
	}
}
