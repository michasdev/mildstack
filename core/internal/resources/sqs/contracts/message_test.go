package contracts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
)

func defaultSQSQueueURL(queueName string) string {
	aws := awscontext.Default()
	return strings.TrimRight(aws.Endpoint, "/") + "/" + aws.AccountID + "/" + queueName
}

func TestMessageContractsPreserveAWSFieldNames(t *testing.T) {
	t.Helper()

	request := SendMessageRequest{
		DelaySeconds: 5,
		MessageAttributes: map[string]MessageAttributeValue{
			"trace": {
				DataType:    "String",
				StringValue: "abc",
			},
		},
		MessageBody:            "payload",
		MessageDeduplicationId: "dedupe-1",
		MessageGroupId:         "group-1",
		MessageSystemAttributes: map[string]MessageAttributeValue{
			"AWSTraceHeader": {
				DataType:    "String",
				StringValue: "Root=1-12345678-1234567890abcdef12345678",
			},
		},
		QueueUrl: defaultSQSQueueURL("orders"),
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal send request: %v", err)
	}
	assertJSONContains(t, string(data), []string{
		`"DelaySeconds":5`,
		`"MessageAttributes"`,
		`"MessageBody":"payload"`,
		`"MessageDeduplicationId":"dedupe-1"`,
		`"MessageGroupId":"group-1"`,
		`"MessageSystemAttributes"`,
		`"QueueUrl":"` + defaultSQSQueueURL("orders") + `"`,
	})

	batch := SendMessageBatchRequest{
		Entries: []SendMessageBatchRequestEntry{
			{
				Id:          "entry-1",
				MessageBody: "payload",
			},
		},
		QueueUrl: request.QueueUrl,
	}
	data, err = json.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal send batch request: %v", err)
	}
	assertJSONContains(t, string(data), []string{
		`"Entries"`,
		`"Id":"entry-1"`,
		`"MessageBody":"payload"`,
		`"QueueUrl":"` + defaultSQSQueueURL("orders") + `"`,
	})

	sendResult := SendMessageResult{
		MD5OfMessageAttributes:       "md5-attrs",
		MD5OfMessageBody:             "md5-body",
		MD5OfMessageSystemAttributes: "md5-system",
		MessageId:                    "message-1",
		SequenceNumber:               "42",
	}
	data, err = json.Marshal(sendResult)
	if err != nil {
		t.Fatalf("marshal send result: %v", err)
	}
	assertJSONContains(t, string(data), []string{
		`"MD5OfMessageAttributes":"md5-attrs"`,
		`"MD5OfMessageBody":"md5-body"`,
		`"MD5OfMessageSystemAttributes":"md5-system"`,
		`"MessageId":"message-1"`,
		`"SequenceNumber":"42"`,
	})

	receiveResult := ReceiveMessageResult{
		Messages: []ReceivedMessage{
			{
				Attributes: map[string]string{
					"ApproximateReceiveCount": "1",
				},
				Body:                   "payload",
				MD5OfBody:              "md5-body",
				MD5OfMessageAttributes: "md5-attrs",
				MessageId:              "message-1",
				ReceiptHandle:          "receipt-1",
			},
		},
	}
	data, err = json.Marshal(receiveResult)
	if err != nil {
		t.Fatalf("marshal receive result: %v", err)
	}
	assertJSONContains(t, string(data), []string{
		`"Messages"`,
		`"Body":"payload"`,
		`"MD5OfBody":"md5-body"`,
		`"MessageId":"message-1"`,
		`"ReceiptHandle":"receipt-1"`,
	})
}

func assertJSONContains(t *testing.T, json string, expected []string) {
	t.Helper()

	for _, token := range expected {
		if !strings.Contains(json, token) {
			t.Fatalf("expected json to contain %q, got %s", token, json)
		}
	}
}
