package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	sqsresource "github.com/michasdev/mildstack/core/internal/resources/sqs"
)

func TestSQSNativeMiddlewareInterceptsQueryRequestsAndLeavesRuntimeRoutesUntouched(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := NewRouter(DefaultConfig(), manager)

	RegisterSQSNativeRoutes(router.Engine(), sqsresource.New())

	healthRecorder := httptest.NewRecorder()
	healthRequest := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/health", nil)
	router.Engine().ServeHTTP(healthRecorder, healthRequest)
	if got, want := healthRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected health status: got %d want %d", got, want)
	}

	rootRecorder := httptest.NewRecorder()
	rootRequest := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2012-11-05", nil)
	router.Engine().ServeHTTP(rootRecorder, rootRequest)
	if got, want := rootRecorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected root status: got %d want %d", got, want)
	}
	if ct := rootRecorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("unexpected root content type: got %q", ct)
	}
	if !strings.Contains(rootRecorder.Body.String(), "UnsupportedOperation") {
		t.Fatalf("expected unsupported operation XML, got %q", rootRecorder.Body.String())
	}

	queueRecorder := httptest.NewRecorder()
	queueRequest := httptest.NewRequest(http.MethodPost, "/123456789012/orders/", strings.NewReader("Action=SendMessage&Version=2012-11-05&MessageBody=hello"))
	queueRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.Engine().ServeHTTP(queueRecorder, queueRequest)
	if got, want := queueRecorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected queue status: got %d want %d", got, want)
	}
	if !strings.Contains(queueRecorder.Body.String(), "UnsupportedOperation") {
		t.Fatalf("expected unsupported operation XML, got %q", queueRecorder.Body.String())
	}

	mismatchRecorder := httptest.NewRecorder()
	mismatchRequest := httptest.NewRequest(http.MethodGet, "/123456789012/orders/?Action=ListQueues&Version=2012-11-05", nil)
	router.Engine().ServeHTTP(mismatchRecorder, mismatchRequest)
	if got, want := mismatchRecorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected mismatch status: got %d want %d", got, want)
	}
	if !strings.Contains(mismatchRecorder.Body.String(), "InvalidAddress") {
		t.Fatalf("expected invalid address XML, got %q", mismatchRecorder.Body.String())
	}
}

func TestSQSSDKSmokeReceivesAWSCompatibleXMLError(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := NewRouter(DefaultConfig(), manager)
	RegisterSQSNativeRoutes(router.Engine(), sqsresource.New())

	server := httptest.NewServer(router.Engine())
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	transport := &captureTransport{base: http.DefaultTransport}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "test")),
		awsconfig.WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(server.URL)
	})

	_, err = client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err == nil {
		t.Fatal("expected list queues to return an AWS-compatible error")
	}
	if !strings.Contains(string(transport.body), "<ErrorResponse>") {
		t.Fatalf("expected captured xml error body, got %q", string(transport.body))
	}
	if !strings.Contains(string(transport.body), "UnsupportedOperation") && !strings.Contains(string(transport.body), "InvalidQueryParameter") {
		t.Fatalf("expected captured xml error body to contain an SQS error code, got %q", string(transport.body))
	}
}

type captureTransport struct {
	base http.RoundTripper
	body []byte
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return resp, nil
	}

	data, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	t.body = append([]byte(nil), data...)
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, nil
}
