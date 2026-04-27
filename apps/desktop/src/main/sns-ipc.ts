import { resolveLocalEndpoint } from './local-endpoint'
import { registerValidatedHandler } from './ipc-middleware'
import {
  SNSClient,
  AddPermissionCommand,
  CheckIfPhoneNumberIsOptedOutCommand,
  ConfirmSubscriptionCommand,
  CreatePlatformApplicationCommand,
  CreatePlatformEndpointCommand,
  CreateSMSSandboxPhoneNumberCommand,
  CreateTopicCommand,
  DeleteEndpointCommand,
  DeletePlatformApplicationCommand,
  DeleteSMSSandboxPhoneNumberCommand,
  DeleteTopicCommand,
  GetDataProtectionPolicyCommand,
  GetEndpointAttributesCommand,
  GetPlatformApplicationAttributesCommand,
  GetSMSAttributesCommand,
  GetSMSSandboxAccountStatusCommand,
  GetSubscriptionAttributesCommand,
  GetTopicAttributesCommand,
  ListEndpointsByPlatformApplicationCommand,
  ListOriginationNumbersCommand,
  ListPhoneNumbersOptedOutCommand,
  ListPlatformApplicationsCommand,
  ListSMSSandboxPhoneNumbersCommand,
  ListSubscriptionsByTopicCommand,
  ListSubscriptionsCommand,
  ListTagsForResourceCommand,
  ListTopicsCommand,
  OptInPhoneNumberCommand,
  PublishBatchCommand,
  PublishCommand,
  PutDataProtectionPolicyCommand,
  RemovePermissionCommand,
  SetEndpointAttributesCommand,
  SetPlatformApplicationAttributesCommand,
  SetSMSAttributesCommand,
  SetSubscriptionAttributesCommand,
  SetTopicAttributesCommand,
  SubscribeCommand,
  TagResourceCommand,
  UntagResourceCommand,
  UnsubscribeCommand,
  VerifySMSSandboxPhoneNumberCommand
} from '@aws-sdk/client-sns'

type SNSClientCacheEntry = {
  region: string
  endpoint: string
  client: SNSClient
}

const clientCache = new Map<string, SNSClientCacheEntry>()

export function registerSNSIpcHandlers(): void {
  registerValidatedHandler('sns:listTopics', async (_event, args: { region?: string }) => {
    const client = getClient(args.region)
    const response = await client.send(new ListTopicsCommand({}))
    const topics = response.Topics ?? []

    return Promise.all(
      topics
        .map(async (topic) => {
          const topicArn = topic.TopicArn ?? ''
          if (!topicArn) return null

          const [attributesResponse, tagsResponse] = await Promise.all([
            client.send(new GetTopicAttributesCommand({ TopicArn: topicArn })).catch(() => ({ Attributes: {} })),
            client.send(new ListTagsForResourceCommand({ ResourceArn: topicArn })).catch(() => ({ Tags: [] }))
          ])

          return {
            TopicArn: topicArn,
            TopicName: topicNameFromArn(topicArn),
            Attributes: attributesResponse.Attributes ?? {},
            Tags: tagsResponse.Tags ?? []
          }
        })
        .filter(Boolean)
    )
  })

  registerValidatedHandler('sns:createTopic', async (_event, args: { name: string; attributes?: Record<string, string>; region?: string }) => {
    const response = await getClient(args.region).send(
      new CreateTopicCommand({
        Name: args.name,
        Attributes: args.attributes
      })
    )
    return response.TopicArn ?? ''
  })

  registerValidatedHandler('sns:deleteTopic', async (_event, args: { topicArn: string; region?: string }) => {
    await getClient(args.region).send(new DeleteTopicCommand({ TopicArn: args.topicArn }))
    return null
  })

  registerValidatedHandler('sns:getTopicAttributes', async (_event, args: { topicArn: string; region?: string }) => {
    const response = await getClient(args.region).send(new GetTopicAttributesCommand({ TopicArn: args.topicArn }))
    return response.Attributes ?? {}
  })

  registerValidatedHandler('sns:setTopicAttribute', async (_event, args: { topicArn: string; attributeName: string; attributeValue: string; region?: string }) => {
    await getClient(args.region).send(
      new SetTopicAttributesCommand({
        TopicArn: args.topicArn,
        AttributeName: args.attributeName,
        AttributeValue: args.attributeValue
      })
    )
    return null
  })

  registerValidatedHandler('sns:listSubscriptions', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListSubscriptionsCommand({}))
    return (response.Subscriptions ?? []).map((subscription) => ({
      SubscriptionArn: subscription.SubscriptionArn ?? '',
      TopicArn: subscription.TopicArn ?? '',
      TopicName: topicNameFromArn(subscription.TopicArn ?? ''),
      Protocol: subscription.Protocol ?? '',
      Endpoint: subscription.Endpoint ?? '',
      Owner: subscription.Owner ?? '',
    }))
  })

  registerValidatedHandler('sns:listSubscriptionsByTopic', async (_event, args: { topicArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new ListSubscriptionsByTopicCommand({ TopicArn: args.topicArn })
    )
    return (response.Subscriptions ?? []).map((subscription) => ({
      SubscriptionArn: subscription.SubscriptionArn ?? '',
      TopicArn: subscription.TopicArn ?? '',
      TopicName: topicNameFromArn(subscription.TopicArn ?? ''),
      Protocol: subscription.Protocol ?? '',
      Endpoint: subscription.Endpoint ?? '',
      Owner: subscription.Owner ?? '',
    }))
  })

  registerValidatedHandler('sns:subscribe', async (_event, args: {
    topicArn: string
    protocol: string
    endpoint: string
    attributes?: Record<string, string>
    returnSubscriptionArn?: boolean
    region?: string
  }) => {
    const response = await getClient(args.region).send(
      new SubscribeCommand({
        TopicArn: args.topicArn,
        Protocol: args.protocol,
        Endpoint: args.endpoint,
        Attributes: args.attributes,
        ReturnSubscriptionArn: args.returnSubscriptionArn
      })
    )
    return response.SubscriptionArn ?? ''
  })

  registerValidatedHandler('sns:confirmSubscription', async (_event, args: { topicArn: string; token: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new ConfirmSubscriptionCommand({
        TopicArn: args.topicArn,
        Token: args.token
      })
    )
    return response.SubscriptionArn ?? ''
  })

  registerValidatedHandler('sns:unsubscribe', async (_event, args: { subscriptionArn: string; region?: string }) => {
    await getClient(args.region).send(new UnsubscribeCommand({ SubscriptionArn: args.subscriptionArn }))
    return null
  })

  registerValidatedHandler('sns:getSubscriptionAttributes', async (_event, args: { subscriptionArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new GetSubscriptionAttributesCommand({ SubscriptionArn: args.subscriptionArn })
    )
    return response.Attributes ?? {}
  })

  registerValidatedHandler('sns:setSubscriptionAttribute', async (_event, args: { subscriptionArn: string; attributeName: string; attributeValue: string; region?: string }) => {
    await getClient(args.region).send(
      new SetSubscriptionAttributesCommand({
        SubscriptionArn: args.subscriptionArn,
        AttributeName: args.attributeName,
        AttributeValue: args.attributeValue
      })
    )
    return null
  })

  registerValidatedHandler('sns:publish', async (_event, args: {
    topicArn?: string
    targetArn?: string
    phoneNumber?: string
    message: string
    subject?: string
    messageStructure?: string
    messageAttributes?: Record<string, any>
    messageGroupId?: string
    messageDeduplicationId?: string
    region?: string
  }) => {
    const response = await getClient(args.region).send(
      new PublishCommand({
        TopicArn: args.topicArn,
        TargetArn: args.targetArn,
        PhoneNumber: args.phoneNumber,
        Message: args.message,
        Subject: args.subject,
        MessageStructure: args.messageStructure,
        MessageAttributes: args.messageAttributes,
        MessageGroupId: args.messageGroupId,
        MessageDeduplicationId: args.messageDeduplicationId
      })
    )
    return response.MessageId ?? ''
  })

  registerValidatedHandler('sns:publishBatch', async (_event, args: {
    topicArn: string
    entries: Array<{
      Id: string
      Message: string
      Subject?: string
      MessageStructure?: string
      MessageAttributes?: Record<string, any>
      MessageGroupId?: string
      MessageDeduplicationId?: string
    }>
    region?: string
  }) => {
    const response = await getClient(args.region).send(
      new PublishBatchCommand({
        TopicArn: args.topicArn,
        PublishBatchRequestEntries: args.entries
      })
    )
    return {
      Successful: response.Successful ?? [],
      Failed: response.Failed ?? []
    }
  })

  registerValidatedHandler('sns:addPermission', async (_event, args: { topicArn: string; label: string; awsAccountIDs: string[]; actionNames: string[]; region?: string }) => {
    await getClient(args.region).send(
      new AddPermissionCommand({
        TopicArn: args.topicArn,
        Label: args.label,
        AWSAccountId: args.awsAccountIDs,
        ActionName: args.actionNames
      })
    )
    return null
  })

  registerValidatedHandler('sns:removePermission', async (_event, args: { topicArn: string; label: string; region?: string }) => {
    await getClient(args.region).send(
      new RemovePermissionCommand({
        TopicArn: args.topicArn,
        Label: args.label
      })
    )
    return null
  })

  registerValidatedHandler('sns:tagResource', async (_event, args: { resourceArn: string; tags: Record<string, string>; region?: string }) => {
    await getClient(args.region).send(
      new TagResourceCommand({
        ResourceArn: args.resourceArn,
        Tags: Object.entries(args.tags).map(([Key, Value]) => ({ Key, Value }))
      })
    )
    return null
  })

  registerValidatedHandler('sns:untagResource', async (_event, args: { resourceArn: string; tagKeys: string[]; region?: string }) => {
    await getClient(args.region).send(
      new UntagResourceCommand({
        ResourceArn: args.resourceArn,
        TagKeys: args.tagKeys
      })
    )
    return null
  })

  registerValidatedHandler('sns:listTagsForResource', async (_event, args: { resourceArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new ListTagsForResourceCommand({ ResourceArn: args.resourceArn })
    )
    return response.Tags ?? []
  })

  registerValidatedHandler('sns:getDataProtectionPolicy', async (_event, args: { resourceArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new GetDataProtectionPolicyCommand({ ResourceArn: args.resourceArn })
    )
    return response.DataProtectionPolicy ?? ''
  })

  registerValidatedHandler('sns:putDataProtectionPolicy', async (_event, args: { resourceArn: string; policyDocument: string; region?: string }) => {
    await getClient(args.region).send(
      new PutDataProtectionPolicyCommand({
        ResourceArn: args.resourceArn,
        DataProtectionPolicy: args.policyDocument
      })
    )
    return null
  })

  registerValidatedHandler('sns:listPlatformApplications', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListPlatformApplicationsCommand({}))
    return (response.PlatformApplications ?? []).map((application) => ({
      PlatformApplicationArn: application.PlatformApplicationArn ?? '',
      Name: platformApplicationNameFromArn(application.PlatformApplicationArn ?? ''),
      Platform: platformFromArn(application.PlatformApplicationArn ?? ''),
      Attributes: application.Attributes ?? {}
    }))
  })

  registerValidatedHandler('sns:createPlatformApplication', async (_event, args: { name: string; platform: string; attributes?: Record<string, string>; region?: string }) => {
    const response = await getClient(args.region).send(
      new CreatePlatformApplicationCommand({
        Name: args.name,
        Platform: args.platform,
        Attributes: args.attributes
      })
    )
    return response.PlatformApplicationArn ?? ''
  })

  registerValidatedHandler('sns:deletePlatformApplication', async (_event, args: { platformApplicationArn: string; region?: string }) => {
    await getClient(args.region).send(
      new DeletePlatformApplicationCommand({ PlatformApplicationArn: args.platformApplicationArn })
    )
    return null
  })

  registerValidatedHandler('sns:getPlatformApplicationAttributes', async (_event, args: { platformApplicationArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new GetPlatformApplicationAttributesCommand({ PlatformApplicationArn: args.platformApplicationArn })
    )
    return response.Attributes ?? {}
  })

  registerValidatedHandler('sns:setPlatformApplicationAttributes', async (_event, args: { platformApplicationArn: string; attributes: Record<string, string>; region?: string }) => {
    await getClient(args.region).send(
      new SetPlatformApplicationAttributesCommand({
        PlatformApplicationArn: args.platformApplicationArn,
        Attributes: args.attributes
      })
    )
    return null
  })

  registerValidatedHandler('sns:createPlatformEndpoint', async (_event, args: {
    platformApplicationArn: string
    token: string
    customUserData?: string
    attributes?: Record<string, string>
    region?: string
  }) => {
    const response = await getClient(args.region).send(
      new CreatePlatformEndpointCommand({
        PlatformApplicationArn: args.platformApplicationArn,
        Token: args.token,
        CustomUserData: args.customUserData,
        Attributes: args.attributes
      })
    )
    return response.EndpointArn ?? ''
  })

  registerValidatedHandler('sns:deleteEndpoint', async (_event, args: { endpointArn: string; region?: string }) => {
    await getClient(args.region).send(new DeleteEndpointCommand({ EndpointArn: args.endpointArn }))
    return null
  })

  registerValidatedHandler('sns:getEndpointAttributes', async (_event, args: { endpointArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new GetEndpointAttributesCommand({ EndpointArn: args.endpointArn })
    )
    return response.Attributes ?? {}
  })

  registerValidatedHandler('sns:setEndpointAttributes', async (_event, args: { endpointArn: string; attributes: Record<string, string>; region?: string }) => {
    await getClient(args.region).send(
      new SetEndpointAttributesCommand({
        EndpointArn: args.endpointArn,
        Attributes: args.attributes
      })
    )
    return null
  })

  registerValidatedHandler('sns:listEndpointsByPlatformApplication', async (_event, args: { platformApplicationArn: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new ListEndpointsByPlatformApplicationCommand({ PlatformApplicationArn: args.platformApplicationArn })
    )
    return (response.Endpoints ?? []).map((endpoint) => ({
      EndpointArn: endpoint.EndpointArn ?? '',
      Attributes: endpoint.Attributes ?? {}
    }))
  })

  registerValidatedHandler('sns:setSMSAttributes', async (_event, args: { attributes: Record<string, string>; region?: string }) => {
    await getClient(args.region).send(
      new SetSMSAttributesCommand({
        attributes: args.attributes
      })
    )
    return null
  })

  registerValidatedHandler('sns:getSMSAttributes', async (_event, args: { attributeNames?: string[]; region?: string }) => {
    const response = await getClient(args.region).send(
      new GetSMSAttributesCommand({
        attributes: args.attributeNames
      })
    )
    return response.attributes ?? {}
  })

  registerValidatedHandler('sns:checkIfPhoneNumberIsOptedOut', async (_event, args: { phoneNumber: string; region?: string }) => {
    const response = await getClient(args.region).send(
      new CheckIfPhoneNumberIsOptedOutCommand({ phoneNumber: args.phoneNumber })
    )
    return Boolean(response.isOptedOut)
  })

  registerValidatedHandler('sns:optInPhoneNumber', async (_event, args: { phoneNumber: string; region?: string }) => {
    await getClient(args.region).send(new OptInPhoneNumberCommand({ phoneNumber: args.phoneNumber }))
    return null
  })

  registerValidatedHandler('sns:listPhoneNumbersOptedOut', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListPhoneNumbersOptedOutCommand({}))
    return response.phoneNumbers ?? []
  })

  registerValidatedHandler('sns:listOriginationNumbers', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListOriginationNumbersCommand({}))
    return (response.PhoneNumbers ?? []).map((entry) => ({
      PhoneNumber: entry.PhoneNumber ?? '',
      Status: entry.Status ?? '',
      CreatedAt: entry.CreatedAt?.toISOString?.() ?? ''
    }))
  })

  registerValidatedHandler('sns:getSMSSandboxAccountStatus', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new GetSMSSandboxAccountStatusCommand({}))
    return Boolean(response.IsInSandbox)
  })

  registerValidatedHandler('sns:createSMSSandboxPhoneNumber', async (_event, args: { phoneNumber: string; languageCode: string; region?: string }) => {
    await getClient(args.region).send(
      new CreateSMSSandboxPhoneNumberCommand({
        PhoneNumber: args.phoneNumber,
        LanguageCode: args.languageCode as any
      })
    )
    return null
  })

  registerValidatedHandler('sns:verifySMSSandboxPhoneNumber', async (_event, args: { phoneNumber: string; oneTimePassword: string; region?: string }) => {
    await getClient(args.region).send(
      new VerifySMSSandboxPhoneNumberCommand({
        PhoneNumber: args.phoneNumber,
        OneTimePassword: args.oneTimePassword
      })
    )
    return null
  })

  registerValidatedHandler('sns:deleteSMSSandboxPhoneNumber', async (_event, args: { phoneNumber: string; region?: string }) => {
    await getClient(args.region).send(
      new DeleteSMSSandboxPhoneNumberCommand({
        PhoneNumber: args.phoneNumber
      })
    )
    return null
  })

  registerValidatedHandler('sns:listSMSSandboxPhoneNumbers', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListSMSSandboxPhoneNumbersCommand({}))
    return (response.PhoneNumbers ?? []).map((entry) => ({
      PhoneNumber: entry.PhoneNumber ?? '',
      Status: entry.Status ?? '',
    }))
  })
}

function getClient(region = 'us-east-1'): SNSClient {
  const normalizedRegion = normalizeRegion(region)
  const endpoint = resolveSNSEndpoint()
  const cacheKey = `${normalizedRegion}:${endpoint}`
  const cached = clientCache.get(cacheKey)
  if (cached) return cached.client

  const client = new SNSClient({
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

function resolveSNSEndpoint(): string {
  return resolveLocalEndpoint('sns')
}

function normalizeRegion(region?: string): string {
  const trimmed = region?.trim()
  return trimmed || 'us-east-1'
}

function topicNameFromArn(arn: string): string {
  const trimmed = arn.trim()
  if (!trimmed) return ''
  return trimmed.split(':').pop() ?? trimmed
}

function platformApplicationNameFromArn(arn: string): string {
  const trimmed = arn.trim()
  if (!trimmed) return ''
  return trimmed.split('/').pop() ?? trimmed.split(':').pop() ?? trimmed
}

function platformFromArn(arn: string): string {
  const trimmed = arn.trim()
  if (!trimmed) return ''
  const parts = trimmed.split('/')
  if (parts.length >= 2) {
    return parts[parts.length - 2] ?? ''
  }
  return ''
}
