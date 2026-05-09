package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
	sqsapplication "github.com/michasdev/mildstack/core/internal/resources/sqs/application"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

var defaultSQSAccountID = awscontext.Default().AccountID

func defaultSQSQueueURL(queueName string) string {
	aws := awscontext.Default()
	return strings.TrimRight(aws.Endpoint, "/") + "/" + aws.AccountID + "/" + queueName
}

func defaultSQSQueueARN(queueName string) string {
	return awscontext.Default().ServiceARN("sqs", queueName)
}

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
	queueRequest := httptest.NewRequest(http.MethodPost, "/"+defaultSQSAccountID+"/orders/", strings.NewReader("Action=SendMessage&Version=2012-11-05&MessageBody=hello"))
	queueRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.Engine().ServeHTTP(queueRecorder, queueRequest)
	if got, want := queueRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected queue status: got %d want %d", got, want)
	}
	if !strings.Contains(queueRecorder.Body.String(), "SendMessageResponse") {
		t.Fatalf("expected send message XML, got %q", queueRecorder.Body.String())
	}

	mismatchRecorder := httptest.NewRecorder()
	mismatchRequest := httptest.NewRequest(http.MethodGet, "/"+defaultSQSAccountID+"/orders/?Action=ListQueues&Version=2012-11-05", nil)
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

	receiptHandle := aws.ToString(receiveOutput.Messages[0].ReceiptHandle)
	if receiptHandle == "" {
		t.Fatal("expected receive message to include a receipt handle")
	}

	if _, err := client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(service.QueueURL("orders")),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: 0,
	}); err != nil {
		t.Fatalf("expected change message visibility to return successfully: %v", err)
	}

	visibleAgain, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(service.QueueURL("orders")),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		t.Fatalf("expected message to become visible again: %v", err)
	}
	if got, want := len(visibleAgain.Messages), 1; got != want {
		t.Fatalf("unexpected redelivery count: got %d want %d", got, want)
	}
	if got, want := aws.ToString(visibleAgain.Messages[0].Body), "hello"; got != want {
		t.Fatalf("unexpected redelivery body: got %q want %q", got, want)
	}

	if _, err := client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(service.QueueURL("orders")),
		ReceiptHandle: visibleAgain.Messages[0].ReceiptHandle,
	}); err != nil {
		t.Fatalf("expected delete message to return successfully: %v", err)
	}

	cleared, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(service.QueueURL("orders")),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		t.Fatalf("expected empty queue receive to succeed: %v", err)
	}
	if got, want := len(cleared.Messages), 0; got != want {
		t.Fatalf("unexpected post-delete receive count: got %d want %d", got, want)
	}
}

func TestSQSNativeQueryDeleteMessageReturnsXMLResponse(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	service := sqsapplication.New()
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	if _, err := service.CreateQueue("orders", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}
	if _, err := service.SendMessage("orders", contracts.SendMessageRequest{
		QueueUrl:    service.QueueURL("orders"),
		MessageBody: "delete-me",
	}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	messages, err := service.ReceiveMessage("orders", 1, 0)
	if err != nil {
		t.Fatalf("receive message: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected a message for delete")
	}

	receiptHandle := sqsapplication.CurrentReceiptHandle(messages[0])
	body := "Action=DeleteMessage&Version=2012-11-05&QueueUrl=" + url.QueryEscape(service.QueueURL("orders")) + "&ReceiptHandle=" + url.QueryEscape(receiptHandle)
	request := httptest.NewRequest(http.MethodPost, "/"+defaultSQSAccountID+"/orders/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete status: got %d want %d", got, want)
	}
	if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("unexpected delete content type: got %q", ct)
	}
	if !strings.Contains(recorder.Body.String(), "<DeleteMessageResponse>") {
		t.Fatalf("expected delete XML response, got %q", recorder.Body.String())
	}
}

func TestSQSNativeQueryChangeVisibilityReturnsXMLResponse(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	service := sqsapplication.New()
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	if _, err := service.CreateQueue("orders", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}
	if _, err := service.SendMessage("orders", contracts.SendMessageRequest{
		QueueUrl:    service.QueueURL("orders"),
		MessageBody: "visible",
	}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	messages, err := service.ReceiveMessage("orders", 1, 0)
	if err != nil {
		t.Fatalf("receive message: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected a message for change visibility")
	}

	receiptHandle := sqsapplication.CurrentReceiptHandle(messages[0])
	body := "Action=ChangeMessageVisibility&Version=2012-11-05&QueueUrl=" + url.QueryEscape(service.QueueURL("orders")) + "&ReceiptHandle=" + url.QueryEscape(receiptHandle) + "&VisibilityTimeout=0"
	request := httptest.NewRequest(http.MethodPost, "/"+defaultSQSAccountID+"/orders/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected visibility status: got %d want %d", got, want)
	}
	if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("unexpected visibility content type: got %q", ct)
	}
	if !strings.Contains(recorder.Body.String(), "<ChangeMessageVisibilityResponse>") {
		t.Fatalf("expected visibility XML response, got %q", recorder.Body.String())
	}
}

func TestSQSSDKSmokeCoversGovernanceAndRedriveActions(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	service := &stubSQSNativeService{
		listQueueTagsResult: map[string]string{
			"env": "dev",
		},
		listDeadLetterSourceQueuesResult: []string{"orders-source"},
		startMessageMoveTaskResult:       defaultSQSQueueARN("orders-dlq") + "|task-1",
		cancelMessageMoveTaskResult:      7,
		listMessageMoveTasksResult: []domain.MessageMoveTask{
			{
				TaskHandle:                       defaultSQSQueueARN("orders-dlq") + "|task-1",
				SourceArn:                        defaultSQSQueueARN("orders-dlq"),
				DestinationArn:                   defaultSQSQueueARN("orders"),
				MaxNumberOfMessagesPerSecond:     12,
				ApproximateNumberOfMessagesMoved: 7,
				Status:                           "RUNNING",
			},
		},
	}
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "test")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(server.URL)
	})

	queueURL := service.QueueURL("orders")
	if _, err := client.TagQueue(ctx, &sqs.TagQueueInput{
		QueueUrl: aws.String(queueURL),
		Tags: map[string]string{
			"env":  "dev",
			"team": "platform",
		},
	}); err != nil {
		t.Fatalf("tag queue: %v", err)
	}
	if got, want := service.tagQueueQueueName, "orders"; got != want {
		t.Fatalf("unexpected tag queue name: got %q want %q", got, want)
	}
	if got, want := service.tagQueueTags["team"], "platform"; got != want {
		t.Fatalf("unexpected tag queue payload: got %q want %q", got, want)
	}

	tagOutput, err := client.ListQueueTags(ctx, &sqs.ListQueueTagsInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		t.Fatalf("list queue tags: %v", err)
	}
	if got, want := tagOutput.Tags["env"], "dev"; got != want {
		t.Fatalf("unexpected tag list value: got %q want %q", got, want)
	}

	if _, err := client.AddPermission(ctx, &sqs.AddPermissionInput{
		QueueUrl:      aws.String(queueURL),
		Label:         aws.String("label-a"),
		AWSAccountIds: []string{defaultSQSAccountID},
		Actions:       []string{"SendMessage"},
	}); err != nil {
		t.Fatalf("add permission: %v", err)
	}
	if got, want := service.addPermissionLabel, "label-a"; got != want {
		t.Fatalf("unexpected add permission label: got %q want %q", got, want)
	}
	if got, want := service.addPermissionAWSAccountIDs[0], defaultSQSAccountID; got != want {
		t.Fatalf("unexpected add permission account: got %q want %q", got, want)
	}

	if _, err := client.RemovePermission(ctx, &sqs.RemovePermissionInput{
		QueueUrl: aws.String(queueURL),
		Label:    aws.String("label-a"),
	}); err != nil {
		t.Fatalf("remove permission: %v", err)
	}
	if got, want := service.removePermissionLabel, "label-a"; got != want {
		t.Fatalf("unexpected remove permission label: got %q want %q", got, want)
	}

	if _, err := client.UntagQueue(ctx, &sqs.UntagQueueInput{
		QueueUrl: aws.String(queueURL),
		TagKeys:  []string{"env", "team"},
	}); err != nil {
		t.Fatalf("untag queue: %v", err)
	}
	if got, want := service.untagQueueQueueName, "orders"; got != want {
		t.Fatalf("unexpected untag queue name: got %q want %q", got, want)
	}
	if got, want := len(service.untagQueueTagKeys), 2; got != want {
		t.Fatalf("unexpected untag key count: got %d want %d", got, want)
	}

	dlqURL := service.QueueURL("orders-dlq")
	dlqOutput, err := client.ListDeadLetterSourceQueues(ctx, &sqs.ListDeadLetterSourceQueuesInput{
		QueueUrl:   aws.String(dlqURL),
		MaxResults: aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("list dead letter source queues: %v", err)
	}
	if got, want := service.listDeadLetterSourceQueuesQueueName, "orders-dlq"; got != want {
		t.Fatalf("unexpected dead letter source queue name: got %q want %q", got, want)
	}
	if got, want := len(dlqOutput.QueueUrls), 1; got != want {
		t.Fatalf("unexpected dead letter source count: got %d want %d", got, want)
	}
	if got, want := dlqOutput.QueueUrls[0], service.QueueURL("orders-source"); got != want {
		t.Fatalf("unexpected dead letter source URL: got %q want %q", got, want)
	}

	startOutput, err := client.StartMessageMoveTask(ctx, &sqs.StartMessageMoveTaskInput{
		SourceArn:                    aws.String(service.QueueARN("orders-dlq")),
		DestinationArn:               aws.String(service.QueueARN("orders")),
		MaxNumberOfMessagesPerSecond: aws.Int32(12),
	})
	if err != nil {
		t.Fatalf("start message move task: %v", err)
	}
	if got, want := service.startMessageMoveTaskSourceArn, service.QueueARN("orders-dlq"); got != want {
		t.Fatalf("unexpected start source arn: got %q want %q", got, want)
	}
	if got, want := service.startMessageMoveTaskDestinationArn, service.QueueARN("orders"); got != want {
		t.Fatalf("unexpected start destination arn: got %q want %q", got, want)
	}
	if got, want := service.startMessageMoveTaskMaxPerSecond, 12; got != want {
		t.Fatalf("unexpected start rate: got %d want %d", got, want)
	}
	if got, want := aws.ToString(startOutput.TaskHandle), defaultSQSQueueARN("orders-dlq")+"|task-1"; got != want {
		t.Fatalf("unexpected task handle: got %q want %q", got, want)
	}

	cancelOutput, err := client.CancelMessageMoveTask(ctx, &sqs.CancelMessageMoveTaskInput{
		TaskHandle: startOutput.TaskHandle,
	})
	if err != nil {
		t.Fatalf("cancel message move task: %v", err)
	}
	if got, want := service.cancelMessageMoveTaskTaskHandle, defaultSQSQueueARN("orders-dlq")+"|task-1"; got != want {
		t.Fatalf("unexpected cancel task handle: got %q want %q", got, want)
	}
	if got, want := cancelOutput.ApproximateNumberOfMessagesMoved, int64(7); got != want {
		t.Fatalf("unexpected moved count: got %d want %d", got, want)
	}

	tasksOutput, err := client.ListMessageMoveTasks(ctx, &sqs.ListMessageMoveTasksInput{
		SourceArn:  aws.String(service.QueueARN("orders-dlq")),
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("list message move tasks: %v", err)
	}
	if got, want := service.listMessageMoveTasksQueueName, "orders-dlq"; got != want {
		t.Fatalf("unexpected move tasks queue name: got %q want %q", got, want)
	}
	if got, want := len(tasksOutput.Results), 1; got != want {
		t.Fatalf("unexpected task result count: got %d want %d", got, want)
	}
	if got, want := aws.ToString(tasksOutput.Results[0].TaskHandle), defaultSQSQueueARN("orders-dlq")+"|task-1"; got != want {
		t.Fatalf("unexpected task result handle: got %q want %q", got, want)
	}
	if got, want := aws.ToString(tasksOutput.Results[0].DestinationArn), service.QueueARN("orders"); got != want {
		t.Fatalf("unexpected task destination arn: got %q want %q", got, want)
	}
}

func TestSQSSDKSmokePreservesBatchEntrySemantics(t *testing.T) {
	t.Helper()

	service := sqsapplication.New()

	if _, err := service.CreateQueue("orders", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}

	result, err := service.SendMessageBatch("orders", contracts.SendMessageBatchRequest{
		QueueUrl: service.QueueURL("orders"),
		Entries: []contracts.SendMessageBatchRequestEntry{
			{Id: "entry-1", MessageBody: "one"},
			{Id: "entry-2", MessageBody: "two"},
			{Id: "entry-3", MessageBody: ""},
		},
	})
	if err != nil {
		t.Fatalf("send message batch: %v", err)
	}
	if got, want := len(result.Successful), 2; got != want {
		t.Fatalf("unexpected batch send success count: got %d want %d", got, want)
	}
	if got, want := len(result.Failed), 1; got != want {
		t.Fatalf("unexpected batch send failure count: got %d want %d", got, want)
	}
	if got, want := result.Successful[0].Id, "entry-1"; got != want {
		t.Fatalf("unexpected first batch success id: got %q want %q", got, want)
	}
	if got, want := result.Failed[0].Id, "entry-3"; got != want {
		t.Fatalf("unexpected batch failure id: got %q want %q", got, want)
	}

	messages, err := service.ReceiveMessage("orders", 2, 0)
	if err != nil {
		t.Fatalf("receive queued batch messages: %v", err)
	}
	if got, want := len(messages), 2; got != want {
		t.Fatalf("unexpected queued message count after batch send: got %d want %d", got, want)
	}
	if got, want := messages[0].Body, "one"; got != want {
		t.Fatalf("unexpected first queued batch body: got %q want %q", got, want)
	}
	if got, want := messages[1].Body, "two"; got != want {
		t.Fatalf("unexpected second queued batch body: got %q want %q", got, want)
	}
}

func TestSQSNativeMiddlewareRoutesLifecycleActionThroughService(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	service := &stubSQSNativeService{}
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2012-11-05&QueueNamePrefix=ord&MaxResults=2&NextToken=token-1&QueueOwnerAWSAccountId="+defaultSQSAccountID, nil)
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

	tagQueueQueueName                   string
	tagQueueTags                        map[string]string
	untagQueueQueueName                 string
	untagQueueTagKeys                   []string
	addPermissionQueueName              string
	addPermissionLabel                  string
	addPermissionAWSAccountIDs          []string
	addPermissionActions                []string
	removePermissionQueueName           string
	removePermissionLabel               string
	listQueueTagsQueueName              string
	listQueueTagsResult                 map[string]string
	listDeadLetterSourceQueuesQueueName string
	listDeadLetterSourceQueuesResult    []string
	startMessageMoveTaskSourceArn       string
	startMessageMoveTaskDestinationArn  string
	startMessageMoveTaskMaxPerSecond    int
	startMessageMoveTaskResult          string
	cancelMessageMoveTaskTaskHandle     string
	cancelMessageMoveTaskResult         int64
	listMessageMoveTasksQueueName       string
	listMessageMoveTasksResult          []domain.MessageMoveTask
}

func (s *stubSQSNativeService) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, contracts.ActionNames(), nil, "sqs")
}

func (s *stubSQSNativeService) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{Name: "sqs"}
}

func (s *stubSQSNativeService) QueueURL(queueName string) string {
	return defaultSQSQueueURL(queueName)
}

func (s *stubSQSNativeService) QueueARN(queueName string) string {
	return defaultSQSQueueARN(queueName)
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

func (s *stubSQSNativeService) TagQueue(queueName string, tags map[string]string) error {
	s.tagQueueQueueName = queueName
	s.tagQueueTags = tags
	return nil
}

func (s *stubSQSNativeService) UntagQueue(queueName string, tagKeys []string) error {
	s.untagQueueQueueName = queueName
	s.untagQueueTagKeys = tagKeys
	return nil
}

func (s *stubSQSNativeService) AddPermission(queueName, label string, awsAccountIDs, actions []string) error {
	s.addPermissionQueueName = queueName
	s.addPermissionLabel = label
	s.addPermissionAWSAccountIDs = awsAccountIDs
	s.addPermissionActions = actions
	return nil
}

func (s *stubSQSNativeService) RemovePermission(queueName, label string) error {
	s.removePermissionQueueName = queueName
	s.removePermissionLabel = label
	return nil
}

func (s *stubSQSNativeService) ListQueueTags(queueName string) (map[string]string, error) {
	s.listQueueTagsQueueName = queueName
	if s.listQueueTagsResult == nil {
		return map[string]string{}, nil
	}
	return s.listQueueTagsResult, nil
}

func (s *stubSQSNativeService) ListDeadLetterSourceQueues(queueName string) ([]string, error) {
	s.listDeadLetterSourceQueuesQueueName = queueName
	if s.listDeadLetterSourceQueuesResult == nil {
		return []string{}, nil
	}
	return s.listDeadLetterSourceQueuesResult, nil
}

func (s *stubSQSNativeService) StartMessageMoveTask(sourceArn, destinationArn string, maxNumberOfMessagesPerSecond int) (string, error) {
	s.startMessageMoveTaskSourceArn = sourceArn
	s.startMessageMoveTaskDestinationArn = destinationArn
	s.startMessageMoveTaskMaxPerSecond = maxNumberOfMessagesPerSecond
	if s.startMessageMoveTaskResult == "" {
		s.startMessageMoveTaskResult = "task-1"
	}
	return s.startMessageMoveTaskResult, nil
}

func (s *stubSQSNativeService) CancelMessageMoveTask(taskHandle string) (int64, error) {
	s.cancelMessageMoveTaskTaskHandle = taskHandle
	return s.cancelMessageMoveTaskResult, nil
}

func (s *stubSQSNativeService) ListMessageMoveTasks(queueName string) ([]domain.MessageMoveTask, error) {
	s.listMessageMoveTasksQueueName = queueName
	if s.listMessageMoveTasksResult == nil {
		return []domain.MessageMoveTask{}, nil
	}
	return s.listMessageMoveTasksResult, nil
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
