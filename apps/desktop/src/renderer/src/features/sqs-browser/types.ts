export interface SQSQueueSummary {
  QueueUrl: string
  QueueName: string
  IsFifo: boolean
  MessagesAvailable: number
  MessagesDelayed: number
  MessagesInvisible: number
  VisibilityTimeout: number
}

export interface SQSMessageAttribute {
  DataType: string
  StringValue?: string
  BinaryValue?: Uint8Array
}

export interface SQSMessage {
  MessageId: string
  ReceiptHandle: string
  MD5OfBody?: string
  Body?: string
  Attributes?: Record<string, string>
  MessageAttributes?: Record<string, SQSMessageAttribute>
}

export interface SQSBrowserApi {
  listQueues(region?: string): Promise<SQSQueueSummary[]>
  createQueue(queueName: string, isFifo?: boolean, region?: string): Promise<string>
  deleteQueue(queueUrl: string, region?: string): Promise<void>
  getQueueAttributes(queueUrl: string, region?: string): Promise<Record<string, string>>
  setQueueAttributes(queueUrl: string, attributes: Record<string, string>, region?: string): Promise<void>
  purgeQueue(queueUrl: string, region?: string): Promise<void>
  sendMessage(
    queueUrl: string,
    body: string,
    delaySeconds?: number,
    messageGroupId?: string,
    messageDeduplicationId?: string,
    messageAttributes?: Record<string, SQSMessageAttribute>,
    region?: string
  ): Promise<string>
  receiveMessages(
    queueUrl: string,
    maxMessages?: number,
    waitTimeSeconds?: number,
    region?: string
  ): Promise<SQSMessage[]>
  deleteMessage(queueUrl: string, receiptHandle: string, region?: string): Promise<void>
}
