package http

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSQSNativeErrorsWriteXMLResponse(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeSQSErrorResponse(ctx, http.StatusBadRequest, "InvalidAction", "The action or operation requested is invalid.", "req-123")

	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/xml") {
		t.Fatalf("unexpected content type: got %q", got)
	}

	var parsed SQSXMLErrorResponse
	if err := xml.Unmarshal(recorder.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if got, want := parsed.Error.Type, "Sender"; got != want {
		t.Fatalf("unexpected error type: got %q want %q", got, want)
	}
	if got, want := parsed.Error.Code, "InvalidAction"; got != want {
		t.Fatalf("unexpected error code: got %q want %q", got, want)
	}
	if got, want := parsed.Error.Message, "The action or operation requested is invalid."; got != want {
		t.Fatalf("unexpected error message: got %q want %q", got, want)
	}
	if got, want := parsed.RequestID, "req-123"; got != want {
		t.Fatalf("unexpected request id: got %q want %q", got, want)
	}
}

func TestSQSNativeErrorsClassifyExplicitErrorCodes(t *testing.T) {
	t.Helper()

	cases := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{name: "invalid action", err: ErrSQSInvalidAction, status: http.StatusBadRequest, code: "InvalidAction"},
		{name: "missing action", err: ErrSQSMissingAction, status: http.StatusBadRequest, code: "MissingAction"},
		{name: "invalid version", err: ErrSQSInvalidVersion, status: http.StatusBadRequest, code: "InvalidParameterValue"},
		{name: "queue mismatch", err: ErrSQSQueuePathMismatch, status: http.StatusBadRequest, code: "InvalidAddress"},
		{name: "unsupported", err: ErrSQSUnsupported, status: http.StatusBadRequest, code: "UnsupportedOperation"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			status, code, _ := classifySQSError(tc.err)
			if status != tc.status {
				t.Fatalf("unexpected status: got %d want %d", status, tc.status)
			}
			if code != tc.code {
				t.Fatalf("unexpected code: got %q want %q", code, tc.code)
			}
		})
	}
}

func TestSQSNativeErrorsFallbackUsesValidationError(t *testing.T) {
	t.Helper()

	status, code, _ := classifySQSError(errors.New("some local failure"))
	if status != http.StatusBadRequest {
		t.Fatalf("unexpected fallback status: got %d", status)
	}
	if code != "ValidationError" {
		t.Fatalf("unexpected fallback code: got %q", code)
	}
}
