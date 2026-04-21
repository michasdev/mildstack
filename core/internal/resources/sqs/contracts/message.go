package contracts

// MessageAttributeValue mirrors the AWS message attribute payload shape.
type MessageAttributeValue struct {
	BinaryListValues [][]byte `json:"BinaryListValues,omitempty"`
	BinaryValue      []byte   `json:"BinaryValue,omitempty"`
	DataType         string   `json:"DataType,omitempty"`
	StringListValues []string `json:"StringListValues,omitempty"`
	StringValue      string   `json:"StringValue,omitempty"`
}

// SendMessageRequest preserves the AWS field names required by the message
// write path.
type SendMessageRequest struct {
	DelaySeconds            int                              `json:"DelaySeconds,omitempty"`
	MessageAttributes       map[string]MessageAttributeValue `json:"MessageAttributes,omitempty"`
	MessageBody             string                           `json:"MessageBody"`
	MessageDeduplicationId  string                           `json:"MessageDeduplicationId,omitempty"`
	MessageGroupId          string                           `json:"MessageGroupId,omitempty"`
	MessageSystemAttributes map[string]MessageAttributeValue `json:"MessageSystemAttributes,omitempty"`
	QueueUrl                string                           `json:"QueueUrl"`
}

// SendMessageResult preserves the AWS response field names for SendMessage.
type SendMessageResult struct {
	MD5OfMessageAttributes       string `json:"MD5OfMessageAttributes,omitempty"`
	MD5OfMessageBody             string `json:"MD5OfMessageBody,omitempty"`
	MD5OfMessageSystemAttributes string `json:"MD5OfMessageSystemAttributes,omitempty"`
	MessageId                    string `json:"MessageId,omitempty"`
	SequenceNumber               string `json:"SequenceNumber,omitempty"`
}

// SendMessageBatchRequestEntry mirrors the AWS batch message entry payload.
type SendMessageBatchRequestEntry struct {
	DelaySeconds            int                              `json:"DelaySeconds,omitempty"`
	Id                      string                           `json:"Id"`
	MessageAttributes       map[string]MessageAttributeValue `json:"MessageAttributes,omitempty"`
	MessageBody             string                           `json:"MessageBody"`
	MessageDeduplicationId  string                           `json:"MessageDeduplicationId,omitempty"`
	MessageGroupId          string                           `json:"MessageGroupId,omitempty"`
	MessageSystemAttributes map[string]MessageAttributeValue `json:"MessageSystemAttributes,omitempty"`
}

// SendMessageBatchRequest preserves the AWS field names for batched sends.
type SendMessageBatchRequest struct {
	Entries  []SendMessageBatchRequestEntry `json:"Entries"`
	QueueUrl string                         `json:"QueueUrl"`
}

// SendMessageBatchResultEntry mirrors the AWS success payload for a batch send.
type SendMessageBatchResultEntry struct {
	Id                           string `json:"Id,omitempty"`
	MD5OfMessageAttributes       string `json:"MD5OfMessageAttributes,omitempty"`
	MD5OfMessageBody             string `json:"MD5OfMessageBody,omitempty"`
	MD5OfMessageSystemAttributes string `json:"MD5OfMessageSystemAttributes,omitempty"`
	MessageId                    string `json:"MessageId,omitempty"`
	SequenceNumber               string `json:"SequenceNumber,omitempty"`
}

// BatchResultErrorEntry preserves the AWS batch failure payload shape.
type BatchResultErrorEntry struct {
	Code        string `json:"Code,omitempty"`
	Id          string `json:"Id,omitempty"`
	Message     string `json:"Message,omitempty"`
	SenderFault bool   `json:"SenderFault,omitempty"`
}

// SendMessageBatchResult preserves the AWS batch response field names.
type SendMessageBatchResult struct {
	Failed     []BatchResultErrorEntry       `json:"Failed,omitempty"`
	Successful []SendMessageBatchResultEntry `json:"Successful,omitempty"`
}

// DeleteMessageRequest preserves the AWS delete payload.
type DeleteMessageRequest struct {
	QueueUrl      string `json:"QueueUrl"`
	ReceiptHandle string `json:"ReceiptHandle"`
}

// DeleteMessageBatchRequestEntry preserves the AWS batch delete entry shape.
type DeleteMessageBatchRequestEntry struct {
	Id            string `json:"Id"`
	ReceiptHandle string `json:"ReceiptHandle"`
}

// DeleteMessageBatchRequest preserves the AWS batch delete payload.
type DeleteMessageBatchRequest struct {
	Entries  []DeleteMessageBatchRequestEntry `json:"Entries"`
	QueueUrl string                           `json:"QueueUrl"`
}

// DeleteMessageBatchResultEntry mirrors the AWS delete batch success entry.
type DeleteMessageBatchResultEntry struct {
	Id string `json:"Id,omitempty"`
}

// DeleteMessageBatchResult preserves the AWS batch delete response shape.
type DeleteMessageBatchResult struct {
	Failed     []BatchResultErrorEntry         `json:"Failed,omitempty"`
	Successful []DeleteMessageBatchResultEntry `json:"Successful,omitempty"`
}

// ChangeMessageVisibilityRequest preserves the AWS visibility payload.
type ChangeMessageVisibilityRequest struct {
	QueueUrl          string `json:"QueueUrl"`
	ReceiptHandle     string `json:"ReceiptHandle"`
	VisibilityTimeout int    `json:"VisibilityTimeout"`
}

// ChangeMessageVisibilityBatchRequestEntry mirrors the AWS batch visibility
// entry payload.
type ChangeMessageVisibilityBatchRequestEntry struct {
	Id                string `json:"Id"`
	ReceiptHandle     string `json:"ReceiptHandle"`
	VisibilityTimeout int    `json:"VisibilityTimeout"`
}

// ChangeMessageVisibilityBatchRequest preserves the AWS batch visibility payload.
type ChangeMessageVisibilityBatchRequest struct {
	Entries  []ChangeMessageVisibilityBatchRequestEntry `json:"Entries"`
	QueueUrl string                                     `json:"QueueUrl"`
}

// ChangeMessageVisibilityBatchResultEntry mirrors the AWS batch visibility
// success entry.
type ChangeMessageVisibilityBatchResultEntry struct {
	Id string `json:"Id,omitempty"`
}

// ChangeMessageVisibilityBatchResult preserves the AWS batch visibility
// response shape.
type ChangeMessageVisibilityBatchResult struct {
	Failed     []BatchResultErrorEntry                   `json:"Failed,omitempty"`
	Successful []ChangeMessageVisibilityBatchResultEntry `json:"Successful,omitempty"`
}

// ReceiveMessageRequest preserves the AWS receive payload.
type ReceiveMessageRequest struct {
	AttributeNames              []string `json:"AttributeNames,omitempty"`
	MaxNumberOfMessages         int      `json:"MaxNumberOfMessages,omitempty"`
	MessageAttributeNames       []string `json:"MessageAttributeNames,omitempty"`
	MessageSystemAttributeNames []string `json:"MessageSystemAttributeNames,omitempty"`
	QueueUrl                    string   `json:"QueueUrl"`
	ReceiveRequestAttemptId     string   `json:"ReceiveRequestAttemptId,omitempty"`
	VisibilityTimeout           int      `json:"VisibilityTimeout,omitempty"`
	WaitTimeSeconds             int      `json:"WaitTimeSeconds,omitempty"`
}

// ReceivedMessage mirrors the AWS receive response message shape.
type ReceivedMessage struct {
	Attributes             map[string]string                `json:"Attributes,omitempty"`
	Body                   string                           `json:"Body,omitempty"`
	MD5OfBody              string                           `json:"MD5OfBody,omitempty"`
	MD5OfMessageAttributes string                           `json:"MD5OfMessageAttributes,omitempty"`
	MessageAttributes      map[string]MessageAttributeValue `json:"MessageAttributes,omitempty"`
	MessageId              string                           `json:"MessageId,omitempty"`
	ReceiptHandle          string                           `json:"ReceiptHandle,omitempty"`
}

// ReceiveMessageResult preserves the AWS receive response wrapper.
type ReceiveMessageResult struct {
	Messages []ReceivedMessage `json:"Messages,omitempty"`
}
