import { resolveLocalEndpoint } from './local-endpoint'
import { registerValidatedHandler } from './ipc-middleware'
import {
  SQSClient,
  ListQueuesCommand,
  CreateQueueCommand,
  DeleteQueueCommand,
  GetQueueAttributesCommand,
  SetQueueAttributesCommand,
  SendMessageCommand,
  ReceiveMessageCommand,
  DeleteMessageCommand,
  PurgeQueueCommand
} from '@aws-sdk/client-sqs'

type SQSClientCacheEntry = {
  region: string
  endpoint: string
  client: SQSClient
}

const clientCache = new Map<string, SQSClientCacheEntry>()

export function registerSQSIpcHandlers(): void {
  registerValidatedHandler('sqs:listQueues', async (_event, args: { region?: string }) => {
    const client = getClient(args.region)
    const response = await client.send(new ListQueuesCommand({}))
    const queueUrls = response.QueueUrls ?? []
    
    // Fetch attributes for each queue to build the summary
    const summaries = await Promise.all(
      queueUrls.map(async (url) => {
        const attrResponse = await client.send(new GetQueueAttributesCommand({
          QueueUrl: url,
          AttributeNames: ['All']
        }))
        const attributes = attrResponse.Attributes || {}
        
        // Extract queue name from URL (last part after /)
        const queueName = url.split('/').pop() || url
        
        return {
          QueueUrl: url,
          QueueName: queueName,
          IsFifo: queueName.endsWith('.fifo'),
          MessagesAvailable: parseInt(attributes.ApproximateNumberOfMessages || '0', 10),
          MessagesDelayed: parseInt(attributes.ApproximateNumberOfMessagesDelayed || '0', 10),
          MessagesInvisible: parseInt(attributes.ApproximateNumberOfMessagesNotVisible || '0', 10),
          VisibilityTimeout: parseInt(attributes.VisibilityTimeout || '30', 10)
        }
      })
    )
    
    return summaries
  })

  registerValidatedHandler('sqs:createQueue', async (_event, args: { queueName: string; isFifo?: boolean; region?: string }) => {
    const attributes: Record<string, string> = {}
    let name = args.queueName
    
    if (args.isFifo) {
      if (!name.endsWith('.fifo')) name += '.fifo'
      attributes.FifoQueue = 'true'
    }

    const response = await getClient(args.region).send(new CreateQueueCommand({
      QueueName: name,
      Attributes: Object.keys(attributes).length > 0 ? attributes : undefined
    }))
    return response.QueueUrl
  })

  registerValidatedHandler('sqs:deleteQueue', async (_event, args: { queueUrl: string; region?: string }) => {
    await getClient(args.region).send(new DeleteQueueCommand({ QueueUrl: args.queueUrl }))
    return null
  })

  registerValidatedHandler('sqs:getQueueAttributes', async (_event, args: { queueUrl: string; region?: string }) => {
    const response = await getClient(args.region).send(new GetQueueAttributesCommand({
      QueueUrl: args.queueUrl,
      AttributeNames: ['All']
    }))
    return response.Attributes || {}
  })

  registerValidatedHandler('sqs:setQueueAttributes', async (_event, args: { queueUrl: string; attributes: Record<string, string>; region?: string }) => {
    await getClient(args.region).send(new SetQueueAttributesCommand({
      QueueUrl: args.queueUrl,
      Attributes: args.attributes
    }))
    return null
  })

  registerValidatedHandler('sqs:purgeQueue', async (_event, args: { queueUrl: string; region?: string }) => {
    await getClient(args.region).send(new PurgeQueueCommand({ QueueUrl: args.queueUrl }))
    return null
  })

  registerValidatedHandler('sqs:sendMessage', async (_event, args: {
    queueUrl: string
    body: string
    delaySeconds?: number
    messageGroupId?: string
    messageDeduplicationId?: string
    messageAttributes?: Record<string, any>
    region?: string
  }) => {
    const response = await getClient(args.region).send(new SendMessageCommand({
      QueueUrl: args.queueUrl,
      MessageBody: args.body,
      DelaySeconds: args.delaySeconds,
      MessageGroupId: args.messageGroupId,
      MessageDeduplicationId: args.messageDeduplicationId,
      MessageAttributes: args.messageAttributes
    }))
    return response.MessageId
  })

  registerValidatedHandler('sqs:receiveMessages', async (_event, args: {
    queueUrl: string
    maxMessages?: number
    waitTimeSeconds?: number
    region?: string
  }) => {
    const response = await getClient(args.region).send(new ReceiveMessageCommand({
      QueueUrl: args.queueUrl,
      MaxNumberOfMessages: args.maxMessages || 1,
      WaitTimeSeconds: args.waitTimeSeconds || 0,
      AttributeNames: ['All'],
      MessageAttributeNames: ['All']
    }))
    
    return (response.Messages || []).map(msg => ({
      MessageId: msg.MessageId,
      ReceiptHandle: msg.ReceiptHandle,
      MD5OfBody: msg.MD5OfBody,
      Body: msg.Body,
      Attributes: msg.Attributes,
      MessageAttributes: msg.MessageAttributes
    }))
  })

  registerValidatedHandler('sqs:deleteMessage', async (_event, args: { queueUrl: string; receiptHandle: string; region?: string }) => {
    await getClient(args.region).send(new DeleteMessageCommand({
      QueueUrl: args.queueUrl,
      ReceiptHandle: args.receiptHandle
    }))
    return null
  })
}

function getClient(region = 'us-east-1'): SQSClient {
  const normalizedRegion = normalizeRegion(region)
  const endpoint = resolveSQSEndpoint()
  const cacheKey = `${normalizedRegion}:${endpoint}`
  const cached = clientCache.get(cacheKey)
  if (cached) {
    return cached.client
  }

  const client = new SQSClient({
    region: normalizedRegion,
    endpoint,
    credentials: {
      accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
      secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test'
    }
  })

  clientCache.set(cacheKey, {
    region: normalizedRegion,
    endpoint,
    client
  })

  return client
}

function resolveSQSEndpoint(): string {
  return resolveLocalEndpoint('sqs')
}

function normalizeRegion(region?: string): string {
  const trimmed = region?.trim()
  return trimmed || 'us-east-1'
}
