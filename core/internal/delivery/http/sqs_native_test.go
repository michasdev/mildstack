package http

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	sqsapplication "github.com/michasdev/mildstack/core/internal/resources/sqs/application"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

func TestSQSNativeMiddlewareInterceptsQueryRequestsAndLeavesRuntimeRoutesUntouched(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := NewRouter(DefaultConfig(), manager)

	service := sqsapplication.New()
	RegisterSQSNativeRoutes(router.Engine(), service)

	healthRecorder := httptest.NewRecorder()
	healthRequest := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/health", nil)
	router.Engine().ServeHTTP(healthRecorder, healthRequest)
	if got, want := healthRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected health status: got %d want %d", got, want)
	}

	rootRecorder := httptest.NewRecorder()
	rootRequest := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2012-11-05", nil)
	router.Engine().ServeHTTP(rootRecorder, rootRequest)
	if got, want := rootRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected root status: got %d want %d", got, want)
	}
	if ct := rootRecorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("unexpected root content type: got %q", ct)
	}
	if !strings.Contains(rootRecorder.Body.String(), "ListQueuesResponse") {
		t.Fatalf("expected list queues XML, got %q", rootRecorder.Body.String())
	}

	if _, err := service.CreateQueue("orders", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}
	queueRecorder := httptest.NewRecorder()
	queueRequest := httptest.NewRequest(http.MethodPost, "/123456789012/orders/", strings.NewReader("Action=SendMessage&Version=2012-11-05&MessageBody=hello"))
	queueRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.Engine().ServeHTTP(queueRecorder, queueRequest)
	if got, want := queueRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected queue status: got %d want %d", got, want)
	}
	if !strings.Contains(queueRecorder.Body.String(), "SendMessageResponse") {
		t.Fatalf("expected send message XML, got %q", queueRecorder.Body.String())
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

func TestSQSNativeMessageActionsPreserveAWSRequestNames(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	service := &stubSQSNativeService{
		sendMessageResult: contracts.SendMessageResult{
			MessageId:        "message-1",
			MD5OfMessageBody: "md5-body",
		},
	}
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	payload := map[string]any{
		"QueueUrl":     service.QueueURL("orders"),
		"MessageBody":  "hello",
		"DelaySeconds": 5,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/x-amz-json-1.0")
	request.Header.Set("X-Amz-Target", "AmazonSQS.SendMessage")
	router.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected send status: got %d want %d", got, want)
	}
	if got, want := service.sendMessageQueueName, "orders"; got != want {
		t.Fatalf("unexpected send queue name: got %q want %q", got, want)
	}
	if got, want := service.sendMessageRequest.MessageBody, "hello"; got != want {
		t.Fatalf("unexpected send message body: got %q want %q", got, want)
	}
	if got, want := service.sendMessageRequest.DelaySeconds, 5; got != want {
		t.Fatalf("unexpected send delay seconds: got %d want %d", got, want)
	}
	if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected json content type, got %q", ct)
	}
	if !strings.Contains(recorder.Body.String(), "\"MessageId\"") {
		t.Fatalf("expected send message json, got %q", recorder.Body.String())
	}
}

func TestSQSSDKSmokeReceivesAWSCompatibleSuccess(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := NewRouter(DefaultConfig(), manager)
	service := sqsapplication.New()
	RegisterSQSNativeRoutes(router.Engine(), service)

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

	if _, err := service.CreateQueue("orders", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}

	_, err = client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		t.Fatalf("expected list queues to return successfully: %v", err)
	}
	if !strings.Contains(string(transport.body), "\"QueueUrls\"") {
		t.Fatalf("expected captured json body, got %q", string(transport.body))
	}

	sendOutput, err := client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(service.QueueURL("orders")),
		MessageBody: aws.String("hello"),
	})
	if err != nil {
		t.Fatalf("expected send message to return successfully: %v", err)
	}
	if sendOutput.MessageId == nil || *sendOutput.MessageId == "" {
		t.Fatal("expected send message output to include a message id")
	}

	receiveOutput, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(service.QueueURL("orders")),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		t.Fatalf("expected receive message to return successfully: %v", err)
	}
	if got, want := len(receiveOutput.Messages), 1; got != want {
		t.Fatalf("unexpected receive count: got %d want %d", got, want)
	}
	if got, want := aws.ToString(receiveOutput.Messages[0].Body), "hello"; got != want {
		t.Fatalf("unexpected receive body: got %q want %q", got, want)
	}
}

func TestSQSNativeMiddlewareRoutesLifecycleActionThroughService(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	service := &stubSQSNativeService{}
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2012-11-05&QueueNamePrefix=ord&MaxResults=2&NextToken=token-1&QueueOwnerAWSAccountId=123456789012", nil)
	router.ServeHTTP(recorder, request)

	if !service.listQueuesCalled {
		t.Fatal("expected lifecycle request to reach the service")
	}
	if got, want := service.queueNamePrefix, "ord"; got != want {
		t.Fatalf("unexpected queue name prefix: got %q want %q", got, want)
	}
	if got, want := service.maxResults, 2; got != want {
		t.Fatalf("unexpected max results: got %d want %d", got, want)
	}
	if got, want := service.nextToken, "token-1"; got != want {
		t.Fatalf("unexpected next token: got %q want %q", got, want)
	}
	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("unexpected lifecycle response status: got %d want %d", got, want)
	}
	if !strings.Contains(recorder.Body.String(), "UnsupportedOperation") {
		t.Fatalf("expected deferred lifecycle response, got %q", recorder.Body.String())
	}
}

type stubSQSNativeService struct {
	listQueuesCalled     bool
	queueNamePrefix      string
	maxResults           int
	nextToken            string
	sendMessageQueueName string
	sendMessageRequest   contracts.SendMessageRequest
	sendMessageResult    contracts.SendMessageResult
}

func (s *stubSQSNativeService) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, contracts.ActionNames(), nil, "sqs")
}

func (s *stubSQSNativeService) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{Name: "sqs"}
}

func (s *stubSQSNativeService) QueueURL(queueName string) string {
	return "https://sqs.us-east-1.amazonaws.com/123456789012/" + queueName
}

func (s *stubSQSNativeService) QueueARN(queueName string) string {
	return "arn:aws:sqs:us-east-1:123456789012:" + queueName
}

func (s *stubSQSNativeService) CreateQueue(queueName string, attributes map[string]string) (domain.Queue, error) {
	return domain.Queue{Name: queueName, URL: s.QueueURL(queueName)}, contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) DeleteQueue(queueName string) error {
	return contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) GetQueueUrl(queueName, ownerAccountID string) (string, error) {
	return s.QueueURL(queueName), contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) ListQueues(queueNamePrefix string, maxResults int, nextToken, ownerAccountID string) ([]domain.Queue, string, error) {
	s.listQueuesCalled = true
	s.queueNamePrefix = queueNamePrefix
	s.maxResults = maxResults
	s.nextToken = nextToken
	return nil, "", contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) PurgeQueue(queueName string) error {
	return contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) GetQueueAttributes(queueName string, attributeNames []string, ownerAccountID string) (contracts.QueueAttributesView, error) {
	return contracts.QueueAttributesView{
		QueueName: queueName,
		QueueURL:  s.QueueURL(queueName),
		QueueARN:  s.QueueARN(queueName),
	}, contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) SetQueueAttributes(queueName string, attributes map[string]string) (contracts.QueueAttributesView, error) {
	return contracts.QueueAttributesView{
		QueueName:  queueName,
		QueueURL:   s.QueueURL(queueName),
		QueueARN:   s.QueueARN(queueName),
		Attributes: attributes,
	}, contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) ReceiveMessage(queueName string, maxMessages int, waitTime time.Duration) ([]domain.Message, error) {
	return []domain.Message{
		{
			Queue:       queueName,
			MessageID:   "message-1",
			Body:        "hello",
			SentAt:      time.Now(),
			ReceiptKeys: []string{"receipt-1"},
		},
	}, nil
}

func (s *stubSQSNativeService) DeleteMessage(queueName string, receiptHandle string) error {
	return contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) ChangeMessageVisibility(queueName string, receiptHandle string, visibility time.Duration) error {
	return contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) SendMessage(queueName string, request contracts.SendMessageRequest) (contracts.SendMessageResult, error) {
	s.sendMessageQueueName = queueName
	s.sendMessageRequest = request
	if s.sendMessageResult.MessageId == "" {
		s.sendMessageResult = contracts.SendMessageResult{
			MessageId:        "message-1",
			MD5OfMessageBody: "md5-body",
		}
	}
	return s.sendMessageResult, nil
}

func (s *stubSQSNativeService) SendMessageBatch(queueName string, request contracts.SendMessageBatchRequest) (contracts.SendMessageBatchResult, error) {
	return contracts.SendMessageBatchResult{}, contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) DeleteMessageBatch(queueName string, request contracts.DeleteMessageBatchRequest) (contracts.DeleteMessageBatchResult, error) {
	return contracts.DeleteMessageBatchResult{}, contracts.ErrSQSOperationDeferred
}

func (s *stubSQSNativeService) ChangeMessageVisibilityBatch(queueName string, request contracts.ChangeMessageVisibilityBatchRequest) (contracts.ChangeMessageVisibilityBatchResult, error) {
	return contracts.ChangeMessageVisibilityBatchResult{}, contracts.ErrSQSOperationDeferred
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
