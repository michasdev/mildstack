package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
)

func defaultSQSQueueURLForContract(queueName string) string {
	aws := awscontext.Default()
	return strings.TrimRight(aws.Endpoint, "/") + "/" + aws.AccountID + "/" + queueName
}

func TestSQSNativeContractParsesQueryAndFormValues(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/"+awscontext.Default().AccountID+"/orders/", strings.NewReader(
		"Action=SendMessage&Version=2012-11-05&QueueUrl="+url.QueryEscape(defaultSQSQueueURLForContract("orders"))+"&QueueNamePrefix=ord&QueueOwnerAWSAccountId="+awscontext.Default().AccountID+"&Attribute.1.Name=DelaySeconds&Attribute.1.Value.StringValue=5",
	))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx, err := ParseSQSRequest(req)
	if err != nil {
		t.Fatalf("parse request: %v", err)
	}
	if got, want := ctx.Kind, SQSRequestKindQueue; got != want {
		t.Fatalf("unexpected kind: got %q want %q", got, want)
	}
	if got, want := ctx.AccountID, awscontext.Default().AccountID; got != want {
		t.Fatalf("unexpected account id: got %q want %q", got, want)
	}
	if got, want := ctx.QueueName, "orders"; got != want {
		t.Fatalf("unexpected queue name: got %q want %q", got, want)
	}
	if got, want := ctx.NormalizedPath, "/"+awscontext.Default().AccountID+"/orders"; got != want {
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
	if got, want := ctx.Values.Get("QueueUrl"), defaultSQSQueueURLForContract("orders"); got != want {
		t.Fatalf("unexpected queue url: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("QueueNamePrefix"), "ord"; got != want {
		t.Fatalf("unexpected queue name prefix: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("QueueOwnerAWSAccountId"), awscontext.Default().AccountID; got != want {
		t.Fatalf("unexpected queue owner account id: got %q want %q", got, want)
	}
}

func TestSQSNativeContractParsesTargetStyleJsonRequests(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"QueueNamePrefix":"ord","MaxResults":2,"NextToken":"token-1"}`))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonSQS.ListQueues")

	ctx, err := ParseSQSRequest(req)
	if err != nil {
		t.Fatalf("parse target-style request: %v", err)
	}
	if !ctx.TargetStyle {
		t.Fatal("expected target-style request to be marked as such")
	}
	if got, want := ctx.Action, "ListQueues"; got != want {
		t.Fatalf("unexpected action: got %q want %q", got, want)
	}
	if got, want := ctx.Version, sqsQueryVersion; got != want {
		t.Fatalf("unexpected version: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("QueueNamePrefix"), "ord"; got != want {
		t.Fatalf("unexpected queue name prefix: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("MaxResults"), "2"; got != want {
		t.Fatalf("unexpected max results: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("NextToken"), "token-1"; got != want {
		t.Fatalf("unexpected next token: got %q want %q", got, want)
	}
}

func TestSQSNativeContractInfersQueueContextFromTargetStyleQueueURL(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"QueueUrl":"`+defaultSQSQueueURLForContract("orders")+`","Attributes":{"DelaySeconds":"0"}}`))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonSQS.SetQueueAttributes")

	ctx, err := ParseSQSRequest(req)
	if err != nil {
		t.Fatalf("parse target-style queue request: %v", err)
	}
	if got, want := ctx.Kind, SQSRequestKindQueue; got != want {
		t.Fatalf("unexpected kind: got %q want %q", got, want)
	}
	if got, want := ctx.QueueName, "orders"; got != want {
		t.Fatalf("unexpected queue name: got %q want %q", got, want)
	}
	if got, want := ctx.AccountID, awscontext.Default().AccountID; got != want {
		t.Fatalf("unexpected account id: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("Attribute.1.Name"), "DelaySeconds"; got != want {
		t.Fatalf("unexpected attribute name: got %q want %q", got, want)
	}
	if got, want := ctx.Values.Get("Attribute.1.Value.StringValue"), "0"; got != want {
		t.Fatalf("unexpected attribute value: got %q want %q", got, want)
	}
}

func TestSQSNativeContractParsesTargetStyleMessageAttributes(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
		"QueueUrl":"`+defaultSQSQueueURLForContract("orders")+`",
		"MessageBody":"hello",
		"MessageAttributes":{
			"Author":{"DataType":"String","StringValue":"MildStack"}
		}
	}`))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonSQS.SendMessage")

	ctx, err := ParseSQSRequest(req)
	if err != nil {
		t.Fatalf("parse target-style message attributes request: %v", err)
	}
	attrs := messageAttributesFromValues(ctx.Values, "MessageAttribute")
	if got, want := attrs["Author"].DataType, "String"; got != want {
		t.Fatalf("unexpected attribute data type: got %q want %q", got, want)
	}
	if got, want := attrs["Author"].StringValue, "MildStack"; got != want {
		t.Fatalf("unexpected attribute string value: got %q want %q", got, want)
	}
}

func TestSQSNativeContractParsesTargetStyleBatchEntries(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
		"QueueUrl":"`+defaultSQSQueueURLForContract("orders")+`",
		"Entries":[
			{"Id":"msg1","MessageBody":"one"},
			{"Id":"msg2","MessageBody":"two"}
		]
	}`))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonSQS.SendMessageBatch")

	ctx, err := ParseSQSRequest(req)
	if err != nil {
		t.Fatalf("parse target-style batch request: %v", err)
	}
	entries := sendMessageBatchEntriesFromValues(ctx.Values)
	if got, want := len(entries), 2; got != want {
		t.Fatalf("unexpected batch entry count: got %d want %d", got, want)
	}
	if got, want := entries[0].Id, "msg1"; got != want {
		t.Fatalf("unexpected first entry id: got %q want %q", got, want)
	}
	if got, want := entries[1].MessageBody, "two"; got != want {
		t.Fatalf("unexpected second entry body: got %q want %q", got, want)
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

	queueReq := httptest.NewRequest(http.MethodGet, "/"+awscontext.Default().AccountID+"/orders/?Action=GetQueueAttributes&Version=2012-11-05", nil)
	queueCtx, err := ParseSQSRequest(queueReq)
	if err != nil {
		t.Fatalf("parse queue request: %v", err)
	}
	if queueCtx.Kind != SQSRequestKindQueue {
		t.Fatalf("unexpected queue kind: got %q", queueCtx.Kind)
	}
	if queueCtx.NormalizedPath != "/"+awscontext.Default().AccountID+"/orders" {
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

func TestSQSNativeContractDoesNotOwnSNSRequests(t *testing.T) {
	t.Helper()

	snsRequest := httptest.NewRequest(http.MethodGet, "/?Action=CreateTopic&Version=2010-03-31", nil)
	if _, err := ParseSQSRequest(snsRequest); err != ErrSQSNotOwned {
		t.Fatalf("unexpected sns routing result: got %v want %v", err, ErrSQSNotOwned)
	}

	snsOverlappingActionRequest := httptest.NewRequest(http.MethodGet, "/?Action=AddPermission&Version=2010-03-31", nil)
	if _, err := ParseSQSRequest(snsOverlappingActionRequest); err != ErrSQSNotOwned {
		t.Fatalf("unexpected sns overlapping action result: got %v want %v", err, ErrSQSNotOwned)
	}
}
