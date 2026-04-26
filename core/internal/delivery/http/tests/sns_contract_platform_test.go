package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSNSContractPlatformApplicationAndEndpointXML(t *testing.T) {
	t.Helper()

	router, _ := newSNSContractHarness(t)

	createAppParams := url.Values{}
	createAppParams.Set("Action", "CreatePlatformApplication")
	createAppParams.Set("Version", "2010-03-31")
	createAppParams.Set("Name", "newsapp")
	createAppParams.Set("Platform", "APNS")
	createAppParams.Set("Attributes.entry.1.key", "PlatformCredential")
	createAppParams.Set("Attributes.entry.1.value", "dev-credential")

	createAppRecorder := performSNSQuery(t, router, http.MethodGet, createAppParams.Encode())
	if got, want := createAppRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create platform application status: got %d want %d", got, want)
	}
	createAppBody := createAppRecorder.Body.String()
	if !strings.Contains(createAppBody, "<CreatePlatformApplicationResponse") {
		t.Fatalf("expected create platform application envelope, got %q", createAppBody)
	}
	platformApplicationARN := "arn:aws:sns:us-east-1:00000000000:app/APNS/newsapp"
	if !strings.Contains(createAppBody, "<PlatformApplicationArn>"+platformApplicationARN+"</PlatformApplicationArn>") {
		t.Fatalf("expected platform application arn in response, got %q", createAppBody)
	}

	listAppsParams := url.Values{}
	listAppsParams.Set("Action", "ListPlatformApplications")
	listAppsParams.Set("Version", "2010-03-31")

	listAppsRecorder := performSNSQuery(t, router, http.MethodGet, listAppsParams.Encode())
	if got, want := listAppsRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list platform applications status: got %d want %d", got, want)
	}
	if body := listAppsRecorder.Body.String(); !strings.Contains(body, platformApplicationARN) {
		t.Fatalf("expected platform application in list response, got %q", body)
	}

	getAppAttributesParams := url.Values{}
	getAppAttributesParams.Set("Action", "GetPlatformApplicationAttributes")
	getAppAttributesParams.Set("Version", "2010-03-31")
	getAppAttributesParams.Set("PlatformApplicationArn", platformApplicationARN)

	getAppAttributesRecorder := performSNSQuery(t, router, http.MethodGet, getAppAttributesParams.Encode())
	if got, want := getAppAttributesRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get platform application attributes status: got %d want %d", got, want)
	}
	if body := getAppAttributesRecorder.Body.String(); !strings.Contains(body, "PlatformCredential") {
		t.Fatalf("expected platform credential attribute, got %q", body)
	}

	setAppAttributesParams := url.Values{}
	setAppAttributesParams.Set("Action", "SetPlatformApplicationAttributes")
	setAppAttributesParams.Set("Version", "2010-03-31")
	setAppAttributesParams.Set("PlatformApplicationArn", platformApplicationARN)
	setAppAttributesParams.Set("Attributes.entry.1.key", "EventEndpointCreated")
	setAppAttributesParams.Set("Attributes.entry.1.value", "arn:aws:sns:us-east-1:00000000000:events")

	setAppAttributesRecorder := performSNSQuery(t, router, http.MethodGet, setAppAttributesParams.Encode())
	if got, want := setAppAttributesRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected set platform application attributes status: got %d want %d", got, want)
	}

	createEndpointParams := url.Values{}
	createEndpointParams.Set("Action", "CreatePlatformEndpoint")
	createEndpointParams.Set("Version", "2010-03-31")
	createEndpointParams.Set("PlatformApplicationArn", platformApplicationARN)
	createEndpointParams.Set("Token", "device-token-1")
	createEndpointParams.Set("CustomUserData", "customer-a")

	createEndpointRecorder := performSNSQuery(t, router, http.MethodGet, createEndpointParams.Encode())
	if got, want := createEndpointRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create platform endpoint status: got %d want %d", got, want)
	}
	createEndpointBody := createEndpointRecorder.Body.String()
	if !strings.Contains(createEndpointBody, "<CreatePlatformEndpointResponse") {
		t.Fatalf("expected create platform endpoint envelope, got %q", createEndpointBody)
	}
	endpointARN := extractXMLValue(createEndpointBody, "EndpointArn")
	if endpointARN == "" {
		t.Fatalf("expected endpoint arn in response, got %q", createEndpointBody)
	}

	getEndpointAttributesParams := url.Values{}
	getEndpointAttributesParams.Set("Action", "GetEndpointAttributes")
	getEndpointAttributesParams.Set("Version", "2010-03-31")
	getEndpointAttributesParams.Set("EndpointArn", endpointARN)

	getEndpointAttributesRecorder := performSNSQuery(t, router, http.MethodGet, getEndpointAttributesParams.Encode())
	if got, want := getEndpointAttributesRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get endpoint attributes status: got %d want %d", got, want)
	}
	if body := getEndpointAttributesRecorder.Body.String(); !strings.Contains(body, "device-token-1") {
		t.Fatalf("expected endpoint token in response, got %q", body)
	}

	setEndpointAttributesParams := url.Values{}
	setEndpointAttributesParams.Set("Action", "SetEndpointAttributes")
	setEndpointAttributesParams.Set("Version", "2010-03-31")
	setEndpointAttributesParams.Set("EndpointArn", endpointARN)
	setEndpointAttributesParams.Set("Attributes.entry.1.key", "Enabled")
	setEndpointAttributesParams.Set("Attributes.entry.1.value", "false")

	setEndpointAttributesRecorder := performSNSQuery(t, router, http.MethodGet, setEndpointAttributesParams.Encode())
	if got, want := setEndpointAttributesRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected set endpoint attributes status: got %d want %d", got, want)
	}

	listEndpointsParams := url.Values{}
	listEndpointsParams.Set("Action", "ListEndpointsByPlatformApplication")
	listEndpointsParams.Set("Version", "2010-03-31")
	listEndpointsParams.Set("PlatformApplicationArn", platformApplicationARN)

	listEndpointsRecorder := performSNSQuery(t, router, http.MethodGet, listEndpointsParams.Encode())
	if got, want := listEndpointsRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list endpoints status: got %d want %d", got, want)
	}
	if body := listEndpointsRecorder.Body.String(); !strings.Contains(body, endpointARN) {
		t.Fatalf("expected endpoint arn in list response, got %q", body)
	}

	deleteEndpointParams := url.Values{}
	deleteEndpointParams.Set("Action", "DeleteEndpoint")
	deleteEndpointParams.Set("Version", "2010-03-31")
	deleteEndpointParams.Set("EndpointArn", endpointARN)

	deleteEndpointRecorder := performSNSQuery(t, router, http.MethodGet, deleteEndpointParams.Encode())
	if got, want := deleteEndpointRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete endpoint status: got %d want %d", got, want)
	}

	deleteAppParams := url.Values{}
	deleteAppParams.Set("Action", "DeletePlatformApplication")
	deleteAppParams.Set("Version", "2010-03-31")
	deleteAppParams.Set("PlatformApplicationArn", platformApplicationARN)

	deleteAppRecorder := performSNSQuery(t, router, http.MethodGet, deleteAppParams.Encode())
	if got, want := deleteAppRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete platform application status: got %d want %d", got, want)
	}
}

func extractXMLValue(body, tag string) string {
	start := "<" + tag + ">"
	end := "</" + tag + ">"
	startIndex := strings.Index(body, start)
	if startIndex < 0 {
		return ""
	}
	startIndex += len(start)
	endIndex := strings.Index(body[startIndex:], end)
	if endIndex < 0 {
		return ""
	}
	return body[startIndex : startIndex+endIndex]
}
