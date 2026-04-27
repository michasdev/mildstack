package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSNSContractSMSOptOutAndSandboxXML(t *testing.T) {
	t.Helper()

	router, _ := newSNSContractHarness(t)

	setAttributesParams := url.Values{}
	setAttributesParams.Set("Action", "SetSMSAttributes")
	setAttributesParams.Set("Version", "2010-03-31")
	setAttributesParams.Set("attributes.entry.1.key", "DefaultSenderID")
	setAttributesParams.Set("attributes.entry.1.value", "MILD")

	setAttributesRecorder := performSNSQuery(t, router, http.MethodGet, setAttributesParams.Encode())
	if got, want := setAttributesRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected set sms attributes status: got %d want %d", got, want)
	}
	if body := setAttributesRecorder.Body.String(); !strings.Contains(body, "<SetSMSAttributesResponse") {
		t.Fatalf("expected SetSMSAttributes envelope, got %q", body)
	}

	getAttributesParams := url.Values{}
	getAttributesParams.Set("Action", "GetSMSAttributes")
	getAttributesParams.Set("Version", "2010-03-31")
	getAttributesParams.Set("attributes.member.1", "DefaultSenderID")

	getAttributesRecorder := performSNSQuery(t, router, http.MethodGet, getAttributesParams.Encode())
	if got, want := getAttributesRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get sms attributes status: got %d want %d", got, want)
	}
	if body := getAttributesRecorder.Body.String(); !strings.Contains(body, "<attributes>") || !strings.Contains(body, "DefaultSenderID") || !strings.Contains(body, "MILD") {
		t.Fatalf("expected sms attributes in response, got %q", body)
	}

	checkOptOutParams := url.Values{}
	checkOptOutParams.Set("Action", "CheckIfPhoneNumberIsOptedOut")
	checkOptOutParams.Set("Version", "2010-03-31")
	checkOptOutParams.Set("phoneNumber", "+12065550100")

	checkOptOutRecorder := performSNSQuery(t, router, http.MethodGet, checkOptOutParams.Encode())
	if got, want := checkOptOutRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected check opt-out status: got %d want %d", got, want)
	}
	if body := checkOptOutRecorder.Body.String(); !strings.Contains(body, "<isOptedOut>false</isOptedOut>") {
		t.Fatalf("expected false opt-out status, got %q", body)
	}

	optInParams := url.Values{}
	optInParams.Set("Action", "OptInPhoneNumber")
	optInParams.Set("Version", "2010-03-31")
	optInParams.Set("phoneNumber", "+12065550100")

	optInRecorder := performSNSQuery(t, router, http.MethodGet, optInParams.Encode())
	if got, want := optInRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected opt-in status: got %d want %d", got, want)
	}

	listOptedOutParams := url.Values{}
	listOptedOutParams.Set("Action", "ListPhoneNumbersOptedOut")
	listOptedOutParams.Set("Version", "2010-03-31")

	listOptedOutRecorder := performSNSQuery(t, router, http.MethodGet, listOptedOutParams.Encode())
	if got, want := listOptedOutRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list opted-out phones status: got %d want %d", got, want)
	}
	if body := listOptedOutRecorder.Body.String(); !strings.Contains(body, "<ListPhoneNumbersOptedOutResponse") {
		t.Fatalf("expected list opted-out envelope, got %q", body)
	}

	createSandboxParams := url.Values{}
	createSandboxParams.Set("Action", "CreateSMSSandboxPhoneNumber")
	createSandboxParams.Set("Version", "2010-03-31")
	createSandboxParams.Set("PhoneNumber", "+12065550100")
	createSandboxParams.Set("LanguageCode", "en-US")

	createSandboxRecorder := performSNSQuery(t, router, http.MethodGet, createSandboxParams.Encode())
	if got, want := createSandboxRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create sms sandbox phone status: got %d want %d", got, want)
	}

	listSandboxParams := url.Values{}
	listSandboxParams.Set("Action", "ListSMSSandboxPhoneNumbers")
	listSandboxParams.Set("Version", "2010-03-31")

	listSandboxRecorder := performSNSQuery(t, router, http.MethodGet, listSandboxParams.Encode())
	if got, want := listSandboxRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list sms sandbox phone numbers status: got %d want %d", got, want)
	}
	if body := listSandboxRecorder.Body.String(); !strings.Contains(body, "<PhoneNumber>+12065550100</PhoneNumber>") || !strings.Contains(body, "<Status>Pending</Status>") {
		t.Fatalf("expected pending sandbox phone in response, got %q", body)
	}

	verifySandboxParams := url.Values{}
	verifySandboxParams.Set("Action", "VerifySMSSandboxPhoneNumber")
	verifySandboxParams.Set("Version", "2010-03-31")
	verifySandboxParams.Set("PhoneNumber", "+12065550100")
	verifySandboxParams.Set("OneTimePassword", "123456")

	verifySandboxRecorder := performSNSQuery(t, router, http.MethodGet, verifySandboxParams.Encode())
	if got, want := verifySandboxRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected verify sms sandbox phone status: got %d want %d", got, want)
	}

	statusParams := url.Values{}
	statusParams.Set("Action", "GetSMSSandboxAccountStatus")
	statusParams.Set("Version", "2010-03-31")

	statusRecorder := performSNSQuery(t, router, http.MethodGet, statusParams.Encode())
	if got, want := statusRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected sms sandbox account status response: got %d want %d", got, want)
	}
	if body := statusRecorder.Body.String(); !strings.Contains(body, "<IsInSandbox>") {
		t.Fatalf("expected IsInSandbox in response, got %q", body)
	}

	listOriginationParams := url.Values{}
	listOriginationParams.Set("Action", "ListOriginationNumbers")
	listOriginationParams.Set("Version", "2010-03-31")

	listOriginationRecorder := performSNSQuery(t, router, http.MethodGet, listOriginationParams.Encode())
	if got, want := listOriginationRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list origination numbers status: got %d want %d", got, want)
	}
	if body := listOriginationRecorder.Body.String(); !strings.Contains(body, "+12065550100") {
		t.Fatalf("expected verified number in origination list, got %q", body)
	}

	deleteSandboxParams := url.Values{}
	deleteSandboxParams.Set("Action", "DeleteSMSSandboxPhoneNumber")
	deleteSandboxParams.Set("Version", "2010-03-31")
	deleteSandboxParams.Set("PhoneNumber", "+12065550100")

	deleteSandboxRecorder := performSNSQuery(t, router, http.MethodGet, deleteSandboxParams.Encode())
	if got, want := deleteSandboxRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete sms sandbox phone status: got %d want %d", got, want)
	}
}
