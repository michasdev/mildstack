import { ElectronAPI } from '@electron-toolkit/preload'

interface S3BrowserApi {
  listBuckets(region?: string): Promise<any[]>
  createBucket(name: string, region?: string): Promise<void>
  deleteBucket(name: string, region?: string): Promise<void>
  listObjects(
    bucket: string,
    prefix?: string,
    continuationToken?: string,
    region?: string
  ): Promise<any>
  putObject(
    bucket: string,
    key: string,
    body: ArrayBuffer,
    contentType?: string,
    region?: string
  ): Promise<void>
  deleteObjects(
    bucket: string,
    keys: string[],
    region?: string
  ): Promise<{ Deleted: { Key: string }[]; Errors: { Key: string; Code: string; Message: string }[] }>
  getObject(bucket: string, key: string, region?: string): Promise<any>
}

interface DynamoDBBrowserApi {
  listTables(region?: string): Promise<any[]>
  describeTable(tableName: string, region?: string): Promise<any>
  createTable(
    tableName: string,
    keySchema: any[],
    attributeDefinitions: any[],
    region?: string
  ): Promise<void>
  deleteTable(tableName: string, region?: string): Promise<void>
  scan(
    tableName: string,
    exclusiveStartKey?: any,
    limit?: number,
    region?: string,
    filterExpression?: string,
    expressionAttributeNames?: Record<string, string>,
    expressionAttributeValues?: any
  ): Promise<any>
  query(
    tableName: string,
    keyConditionExpression: string,
    expressionAttributeNames?: Record<string, string>,
    expressionAttributeValues?: any,
    indexName?: string,
    filterExpression?: string,
    exclusiveStartKey?: any,
    limit?: number,
    scanIndexForward?: boolean,
    region?: string
  ): Promise<any>
  putItem(tableName: string, item: any, region?: string): Promise<void>
  deleteItem(tableName: string, key: any, region?: string): Promise<void>
  getItem(tableName: string, key: any, region?: string): Promise<any>
}

interface SQSBrowserApi {
  listQueues(region?: string): Promise<any[]>
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
    messageAttributes?: Record<string, any>,
    region?: string
  ): Promise<string>
  receiveMessages(
    queueUrl: string,
    maxMessages?: number,
    waitTimeSeconds?: number,
    region?: string
  ): Promise<any[]>
  deleteMessage(queueUrl: string, receiptHandle: string, region?: string): Promise<void>
}

interface SNSBrowserApi {
  listTopics(region?: string): Promise<any[]>
  createTopic(name: string, attributes?: Record<string, string>, region?: string): Promise<string>
  deleteTopic(topicArn: string, region?: string): Promise<void>
  getTopicAttributes(topicArn: string, region?: string): Promise<Record<string, string>>
  setTopicAttribute(topicArn: string, attributeName: string, attributeValue: string, region?: string): Promise<void>
  listSubscriptions(region?: string): Promise<any[]>
  listSubscriptionsByTopic(topicArn: string, region?: string): Promise<any[]>
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
  publishBatch(
    topicArn: string,
    entries: Array<{
      Id: string
      Message: string
      Subject?: string
      MessageStructure?: string
      MessageAttributes?: Record<string, any>
      MessageGroupId?: string
      MessageDeduplicationId?: string
    }>,
    region?: string
  ): Promise<{ Successful: any[]; Failed: any[] }>
  addPermission(topicArn: string, label: string, awsAccountIDs: string[], actionNames: string[], region?: string): Promise<void>
  removePermission(topicArn: string, label: string, region?: string): Promise<void>
  tagResource(resourceArn: string, tags: Record<string, string>, region?: string): Promise<void>
  untagResource(resourceArn: string, tagKeys: string[], region?: string): Promise<void>
  listTagsForResource(resourceArn: string, region?: string): Promise<any[]>
  getDataProtectionPolicy(resourceArn: string, region?: string): Promise<string>
  putDataProtectionPolicy(resourceArn: string, policyDocument: string, region?: string): Promise<void>
  listPlatformApplications(region?: string): Promise<any[]>
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
  listEndpointsByPlatformApplication(platformApplicationArn: string, region?: string): Promise<any[]>
  setSMSAttributes(attributes: Record<string, string>, region?: string): Promise<void>
  getSMSAttributes(attributeNames?: string[], region?: string): Promise<Record<string, string>>
  checkIfPhoneNumberIsOptedOut(phoneNumber: string, region?: string): Promise<boolean>
  optInPhoneNumber(phoneNumber: string, region?: string): Promise<void>
  listPhoneNumbersOptedOut(region?: string): Promise<any[]>
  listOriginationNumbers(region?: string): Promise<any[]>
  getSMSSandboxAccountStatus(region?: string): Promise<boolean>
  createSMSSandboxPhoneNumber(phoneNumber: string, languageCode: string, region?: string): Promise<void>
  verifySMSSandboxPhoneNumber(phoneNumber: string, oneTimePassword: string, region?: string): Promise<void>
  deleteSMSSandboxPhoneNumber(phoneNumber: string, region?: string): Promise<void>
  listSMSSandboxPhoneNumbers(region?: string): Promise<any[]>
}

interface InstanceApi {
  setSelected(port: number): Promise<void>
}

interface MildStackInstance {
  instanceId: string
  port: number
  pid?: number
  status: 'running' | 'not_started' | 'errored'
  error?: string
}

interface MildStackInstancesResponse {
  state: string
  services: Array<{
    name: string
    version: string
    tags: string[]
  }>
  instances: MildStackInstance[]
  ports: number[] | null
}

interface MildStackApi {
  instances(): Promise<MildStackInstancesResponse>
  start(port: number): Promise<{ success: boolean; error?: string }>
  stop(port?: number, all?: boolean): Promise<{ success: boolean; error?: string }>
  delete(port?: number, all?: boolean): Promise<{ success: boolean; error?: string }>
  validateInstance(): Promise<{ valid: boolean; error?: string }>
}

declare global {
  interface Window {
    electron: ElectronAPI
    api: {
      s3: S3BrowserApi
      dynamodb: DynamoDBBrowserApi
      sqs: SQSBrowserApi
      sns: SNSBrowserApi
      instance: InstanceApi
      mildstack: MildStackApi
    }
  }
}
