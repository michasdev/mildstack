export interface SNSTopicSummary {
  TopicArn: string
  TopicName: string
  Attributes: Record<string, string>
  Tags: Array<{ Key?: string; Value?: string }>
}

export interface SNSSubscriptionSummary {
  SubscriptionArn: string
  TopicArn: string
  TopicName: string
  Protocol: string
  Endpoint: string
  Owner?: string
  Attributes: Record<string, string>
}

export interface SNSPlatformApplicationSummary {
  PlatformApplicationArn: string
  Name: string
  Platform: string
  Attributes: Record<string, string>
}

export interface SNSPlatformEndpointSummary {
  EndpointArn: string
  Attributes: Record<string, string>
}

export interface SMSSandboxPhoneNumber {
  PhoneNumber: string
  LanguageCode?: string
  Status?: string
  CreatedAt?: string
}

export interface SNSPublishBatchEntry {
  Id: string
  Message: string
  Subject?: string
  MessageStructure?: string
  MessageAttributes?: Record<string, any>
  MessageGroupId?: string
  MessageDeduplicationId?: string
}

export interface SNSBrowserApi {
  listTopics(region?: string): Promise<SNSTopicSummary[]>
  createTopic(name: string, attributes?: Record<string, string>, region?: string): Promise<string>
  deleteTopic(topicArn: string, region?: string): Promise<void>
  getTopicAttributes(topicArn: string, region?: string): Promise<Record<string, string>>
  setTopicAttribute(topicArn: string, attributeName: string, attributeValue: string, region?: string): Promise<void>
  listSubscriptions(region?: string): Promise<SNSSubscriptionSummary[]>
  listSubscriptionsByTopic(topicArn: string, region?: string): Promise<SNSSubscriptionSummary[]>
  subscribe(
    topicArn: string,
    protocol: string,
    endpoint: string,
    attributes?: Record<string, string>,
    returnSubscriptionArn?: boolean,
    region?: string
  ): Promise<string>
  confirmSubscription(topicArn: string, token: string, region?: string): Promise<string>
  unsubscribe(subscriptionArn: string, region?: string): Promise<void>
  getSubscriptionAttributes(subscriptionArn: string, region?: string): Promise<Record<string, string>>
  setSubscriptionAttribute(subscriptionArn: string, attributeName: string, attributeValue: string, region?: string): Promise<void>
  publish(
    topicArn?: string,
    targetArn?: string,
    phoneNumber?: string,
    message?: string,
    subject?: string,
    messageStructure?: string,
    messageAttributes?: Record<string, any>,
    messageGroupId?: string,
    messageDeduplicationId?: string,
    region?: string
  ): Promise<string>
  publishBatch(topicArn: string, entries: SNSPublishBatchEntry[], region?: string): Promise<{ Successful: any[]; Failed: any[] }>
  addPermission(topicArn: string, label: string, awsAccountIDs: string[], actionNames: string[], region?: string): Promise<void>
  removePermission(topicArn: string, label: string, region?: string): Promise<void>
  tagResource(resourceArn: string, tags: Record<string, string>, region?: string): Promise<void>
  untagResource(resourceArn: string, tagKeys: string[], region?: string): Promise<void>
  listTagsForResource(resourceArn: string, region?: string): Promise<Array<{ Key?: string; Value?: string }>>
  getDataProtectionPolicy(resourceArn: string, region?: string): Promise<string>
  putDataProtectionPolicy(resourceArn: string, policyDocument: string, region?: string): Promise<void>
  listPlatformApplications(region?: string): Promise<SNSPlatformApplicationSummary[]>
  createPlatformApplication(name: string, platform: string, attributes?: Record<string, string>, region?: string): Promise<string>
  deletePlatformApplication(platformApplicationArn: string, region?: string): Promise<void>
  getPlatformApplicationAttributes(platformApplicationArn: string, region?: string): Promise<Record<string, string>>
  setPlatformApplicationAttributes(platformApplicationArn: string, attributes: Record<string, string>, region?: string): Promise<void>
  createPlatformEndpoint(
    platformApplicationArn: string,
    token: string,
    customUserData?: string,
    attributes?: Record<string, string>,
    region?: string
  ): Promise<string>
  deleteEndpoint(endpointArn: string, region?: string): Promise<void>
  getEndpointAttributes(endpointArn: string, region?: string): Promise<Record<string, string>>
  setEndpointAttributes(endpointArn: string, attributes: Record<string, string>, region?: string): Promise<void>
  listEndpointsByPlatformApplication(platformApplicationArn: string, region?: string): Promise<SNSPlatformEndpointSummary[]>
  setSMSAttributes(attributes: Record<string, string>, region?: string): Promise<void>
  getSMSAttributes(attributeNames?: string[], region?: string): Promise<Record<string, string>>
  checkIfPhoneNumberIsOptedOut(phoneNumber: string, region?: string): Promise<boolean>
  optInPhoneNumber(phoneNumber: string, region?: string): Promise<void>
  listPhoneNumbersOptedOut(region?: string): Promise<string[]>
  listOriginationNumbers(region?: string): Promise<Array<{ PhoneNumber?: string; Status?: string; CreatedAt?: string }>>
  getSMSSandboxAccountStatus(region?: string): Promise<boolean>
  createSMSSandboxPhoneNumber(phoneNumber: string, languageCode: string, region?: string): Promise<void>
  verifySMSSandboxPhoneNumber(phoneNumber: string, oneTimePassword: string, region?: string): Promise<void>
  deleteSMSSandboxPhoneNumber(phoneNumber: string, region?: string): Promise<void>
  listSMSSandboxPhoneNumbers(region?: string): Promise<SMSSandboxPhoneNumber[]>
}

export type SNSBrowserSection = 'topics' | 'subscriptions' | 'platform-applications' | 'sms'

export function topicNameFromArn(arn: string): string {
  return arn.split(':').pop() ?? arn
}

export function parseAttributesJson(input: string): Record<string, string> {
  const trimmed = input.trim()
  if (!trimmed) return {}
  const parsed = JSON.parse(trimmed) as Record<string, unknown>
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Expected a JSON object')
  }

  const result: Record<string, string> = {}
  for (const [key, value] of Object.entries(parsed)) {
    if (value === null || value === undefined) continue
    result[key] = String(value)
  }
  return result
}

export function formatAttributesJson(attributes: Record<string, string> | undefined): string {
  return JSON.stringify(attributes ?? {}, null, 2)
}

export function safeJsonParse(text: string): unknown | null {
  const trimmed = text.trim()
  if (!trimmed) return null
  return JSON.parse(trimmed) as unknown
}
