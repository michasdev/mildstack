package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSQSNativeContractParsesQueryAndFormValues(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/123456789012/orders/", strings.NewReader(
		"Action=SendMessage&Version=2012-11-05&QueueUrl=https%3A%2F%2Flocalhost%2F123456789012%2Forders&Attribute.1.Name=DelaySeconds&Attribute.1.Value.StringValue=5",
	))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx, err := ParseSQSRequest(req)
	if err != nil {
		t.Fatalf("parse request: %v", err)
	}
	if got, want := ctx.Kind, SQSRequestKindQueue; got != want {
		t.Fatalf("unexpected kind: got %q want %q", got, want)
	}
	if got, want := ctx.AccountID, "123456789012"; got != want {
		t.Fatalf("unexpected account id: got %q want %q", got, want)
	}
	if got, want := ctx.QueueName, "orders"; got != want {
		t.Fatalf("unexpected queue name: got %q want %q", got, want)
	}
	if got, want := ctx.NormalizedPath, "/123456789012/orders"; got != want {
		t.Fatalf("unexpected normalized path: got %q want %q", got, want)
	}
	if got, want := ctx.Action, "SendMessage"; got != want {
		t.Fatalf("unexpected action: got %q want %q", got, want)
	}
	if got, want := ctx.Version, sqsQueryVersion; got != want {
		t.Fatalf("unexpected version: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("Attribute.1.Name"), "DelaySeconds"; got != want {
		t.Fatalf("unexpected numbered attribute name: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("Attribute.1.Value.StringValue"), "5"; got != want {
		t.Fatalf("unexpected numbered attribute value: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("QueueUrl"), "https://localhost/123456789012/orders"; got != want {
		t.Fatalf("unexpected queue url: got %q want %q", got, want)
	}
}

func TestSQSNativeContractClassifiesRootAndQueuePaths(t *testing.T) {
	t.Helper()

	rootReq := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2012-11-05", nil)
	rootCtx, err := ParseSQSRequest(rootReq)
	if err != nil {
		t.Fatalf("parse root request: %v", err)
	}
	if rootCtx.Kind != SQSRequestKindRoot {
		t.Fatalf("unexpected root kind: got %q", rootCtx.Kind)
	}
	if rootCtx.NormalizedPath != "/" {
		t.Fatalf("unexpected root normalized path: got %q", rootCtx.NormalizedPath)
	}

	queueReq := httptest.NewRequest(http.MethodGet, "/123456789012/orders/?Action=GetQueueAttributes&Version=2012-11-05", nil)
	queueCtx, err := ParseSQSRequest(queueReq)
	if err != nil {
		t.Fatalf("parse queue request: %v", err)
	}
	if queueCtx.Kind != SQSRequestKindQueue {
		t.Fatalf("unexpected queue kind: got %q", queueCtx.Kind)
	}
	if queueCtx.NormalizedPath != "/123456789012/orders" {
		t.Fatalf("unexpected queue normalized path: got %q", queueCtx.NormalizedPath)
	}

	if _, err := ParseSQSRequest(httptest.NewRequest(http.MethodGet, "/api/v1/runtime/services", nil)); err != ErrSQSNotOwned {
		t.Fatalf("unexpected api path result: got %v want %v", err, ErrSQSNotOwned)
	}
}

func TestSQSNativeContractRejectsMissingActionAndBadVersion(t *testing.T) {
	t.Helper()

	missingAction := httptest.NewRequest(http.MethodGet, "/?Version=2012-11-05", nil)
	if _, err := ParseSQSRequest(missingAction); err != ErrSQSMissingAction {
		t.Fatalf("unexpected missing action result: got %v want %v", err, ErrSQSMissingAction)
	}

	badVersion := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2014-01-01", nil)
	if _, err := ParseSQSRequest(badVersion); err != ErrSQSInvalidVersion {
		t.Fatalf("unexpected bad version result: got %v want %v", err, ErrSQSInvalidVersion)
	}
}
