package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSNSContractTagsPermissionsAndPoliciesXML(t *testing.T) {
	t.Helper()

	router, service := newSNSContractHarness(t)
	topic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic fixture: %v", err)
	}

	tagParams := url.Values{}
	tagParams.Set("Action", "TagResource")
	tagParams.Set("Version", "2010-03-31")
	tagParams.Set("ResourceArn", topic.ARN)
	tagParams.Set("Tags.member.1.Key", "env")
	tagParams.Set("Tags.member.1.Value", "dev")
	tagParams.Set("Tags.member.2.Key", "team")
	tagParams.Set("Tags.member.2.Value", "core")

	tagRecorder := performSNSQuery(t, router, http.MethodGet, tagParams.Encode())
	if got, want := tagRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected tag status: got %d want %d", got, want)
	}
	if body := tagRecorder.Body.String(); !strings.Contains(body, "<TagResourceResponse") {
		t.Fatalf("expected TagResource envelope, got %q", body)
	}

	listTagsParams := url.Values{}
	listTagsParams.Set("Action", "ListTagsForResource")
	listTagsParams.Set("Version", "2010-03-31")
	listTagsParams.Set("ResourceArn", topic.ARN)

	listTagsRecorder := performSNSQuery(t, router, http.MethodGet, listTagsParams.Encode())
	if got, want := listTagsRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list tags status: got %d want %d", got, want)
	}
	listTagsBody := listTagsRecorder.Body.String()
	if !strings.Contains(listTagsBody, "<ListTagsForResourceResponse") {
		t.Fatalf("expected ListTagsForResource envelope, got %q", listTagsBody)
	}
	if !strings.Contains(listTagsBody, "<Key>env</Key>") || !strings.Contains(listTagsBody, "<Value>dev</Value>") {
		t.Fatalf("expected env tag in response, got %q", listTagsBody)
	}

	addPermissionParams := url.Values{}
	addPermissionParams.Set("Action", "AddPermission")
	addPermissionParams.Set("Version", "2010-03-31")
	addPermissionParams.Set("TopicArn", topic.ARN)
	addPermissionParams.Set("Label", "AllowPublish")
	addPermissionParams.Set("AWSAccountId.member.1", "111111111111")
	addPermissionParams.Set("ActionName.member.1", "Publish")

	addPermissionRecorder := performSNSQuery(t, router, http.MethodGet, addPermissionParams.Encode())
	if got, want := addPermissionRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected add permission status: got %d want %d", got, want)
	}

	getTopicParams := url.Values{}
	getTopicParams.Set("Action", "GetTopicAttributes")
	getTopicParams.Set("Version", "2010-03-31")
	getTopicParams.Set("TopicArn", topic.ARN)

	getTopicRecorder := performSNSQuery(t, router, http.MethodGet, getTopicParams.Encode())
	if got, want := getTopicRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get topic attributes status: got %d want %d", got, want)
	}
	getTopicBody := getTopicRecorder.Body.String()
	if !strings.Contains(getTopicBody, "<key>Policy</key>") || !strings.Contains(getTopicBody, "AllowPublish") {
		t.Fatalf("expected policy to include label, got %q", getTopicBody)
	}

	removePermissionParams := url.Values{}
	removePermissionParams.Set("Action", "RemovePermission")
	removePermissionParams.Set("Version", "2010-03-31")
	removePermissionParams.Set("TopicArn", topic.ARN)
	removePermissionParams.Set("Label", "AllowPublish")

	removePermissionRecorder := performSNSQuery(t, router, http.MethodGet, removePermissionParams.Encode())
	if got, want := removePermissionRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected remove permission status: got %d want %d", got, want)
	}

	putPolicyParams := url.Values{}
	putPolicyParams.Set("Action", "PutDataProtectionPolicy")
	putPolicyParams.Set("Version", "2010-03-31")
	putPolicyParams.Set("ResourceArn", topic.ARN)
	putPolicyParams.Set("DataProtectionPolicy", `{"Name":"mask"}`)

	putPolicyRecorder := performSNSQuery(t, router, http.MethodGet, putPolicyParams.Encode())
	if got, want := putPolicyRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected put data protection policy status: got %d want %d", got, want)
	}

	getPolicyParams := url.Values{}
	getPolicyParams.Set("Action", "GetDataProtectionPolicy")
	getPolicyParams.Set("Version", "2010-03-31")
	getPolicyParams.Set("ResourceArn", topic.ARN)

	getPolicyRecorder := performSNSQuery(t, router, http.MethodGet, getPolicyParams.Encode())
	if got, want := getPolicyRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get data protection policy status: got %d want %d", got, want)
	}
	if body := getPolicyRecorder.Body.String(); !strings.Contains(body, "<DataProtectionPolicy>") || !strings.Contains(body, "mask") {
		t.Fatalf("expected data protection policy in response, got %q", body)
	}
}
