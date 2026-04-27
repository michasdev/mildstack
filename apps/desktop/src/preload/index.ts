import { contextBridge } from 'electron'
import { ipcRenderer } from 'electron'
import { electronAPI } from '@electron-toolkit/preload'

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

// Custom APIs for renderer
const api: { s3: S3BrowserApi; dynamodb: DynamoDBBrowserApi; sqs: SQSBrowserApi; sns: SNSBrowserApi; instance: InstanceApi; mildstack: MildStackApi } = {
  s3: {
    listBuckets: (region) => ipcRenderer.invoke('s3:listBuckets', { region }),
    createBucket: (name, region) => ipcRenderer.invoke('s3:createBucket', { name, region }),
    deleteBucket: (name, region) => ipcRenderer.invoke('s3:deleteBucket', { name, region }),
    listObjects: (bucket, prefix, continuationToken, region) =>
      ipcRenderer.invoke('s3:listObjects', { bucket, prefix, continuationToken, region }),
    putObject: (bucket, key, body, contentType, region) =>
      ipcRenderer.invoke('s3:putObject', { bucket, key, body, contentType, region }),
    deleteObjects: (bucket, keys, region) =>
      ipcRenderer.invoke('s3:deleteObjects', { bucket, keys, region }),
    getObject: (bucket, key, region) => ipcRenderer.invoke('s3:getObject', { bucket, key, region })
  },
  dynamodb: {
    listTables: (region) => ipcRenderer.invoke('dynamodb:listTables', { region }),
    describeTable: (tableName, region) => ipcRenderer.invoke('dynamodb:describeTable', { tableName, region }),
    createTable: (tableName, keySchema, attributeDefinitions, region) =>
      ipcRenderer.invoke('dynamodb:createTable', { tableName, keySchema, attributeDefinitions, region }),
    deleteTable: (tableName, region) => ipcRenderer.invoke('dynamodb:deleteTable', { tableName, region }),
    scan: (tableName, exclusiveStartKey, limit, region, filterExpression, expressionAttributeNames, expressionAttributeValues) =>
      ipcRenderer.invoke('dynamodb:scan', { tableName, exclusiveStartKey, limit, region, filterExpression, expressionAttributeNames, expressionAttributeValues }),
    query: (tableName, keyConditionExpression, expressionAttributeNames, expressionAttributeValues, indexName, filterExpression, exclusiveStartKey, limit, scanIndexForward, region) =>
      ipcRenderer.invoke('dynamodb:query', { tableName, keyConditionExpression, expressionAttributeNames, expressionAttributeValues, indexName, filterExpression, exclusiveStartKey, limit, scanIndexForward, region }),
    putItem: (tableName, item, region) =>
      ipcRenderer.invoke('dynamodb:putItem', { tableName, item, region }),
    deleteItem: (tableName, key, region) =>
      ipcRenderer.invoke('dynamodb:deleteItem', { tableName, key, region }),
    getItem: (tableName, key, region) =>
      ipcRenderer.invoke('dynamodb:getItem', { tableName, key, region })
  },
  sqs: {
    listQueues: (region) => ipcRenderer.invoke('sqs:listQueues', { region }),
    createQueue: (queueName, isFifo, region) => ipcRenderer.invoke('sqs:createQueue', { queueName, isFifo, region }),
    deleteQueue: (queueUrl, region) => ipcRenderer.invoke('sqs:deleteQueue', { queueUrl, region }),
    getQueueAttributes: (queueUrl, region) => ipcRenderer.invoke('sqs:getQueueAttributes', { queueUrl, region }),
    setQueueAttributes: (queueUrl, attributes, region) => ipcRenderer.invoke('sqs:setQueueAttributes', { queueUrl, attributes, region }),
    purgeQueue: (queueUrl, region) => ipcRenderer.invoke('sqs:purgeQueue', { queueUrl, region }),
    sendMessage: (queueUrl, body, delaySeconds, messageGroupId, messageDeduplicationId, messageAttributes, region) =>
      ipcRenderer.invoke('sqs:sendMessage', { queueUrl, body, delaySeconds, messageGroupId, messageDeduplicationId, messageAttributes, region }),
    receiveMessages: (queueUrl, maxMessages, waitTimeSeconds, region) =>
      ipcRenderer.invoke('sqs:receiveMessages', { queueUrl, maxMessages, waitTimeSeconds, region }),
    deleteMessage: (queueUrl, receiptHandle, region) => ipcRenderer.invoke('sqs:deleteMessage', { queueUrl, receiptHandle, region })
  },
  sns: {
    listTopics: (region) => ipcRenderer.invoke('sns:listTopics', { region }),
    createTopic: (name, attributes, region) => ipcRenderer.invoke('sns:createTopic', { name, attributes, region }),
    deleteTopic: (topicArn, region) => ipcRenderer.invoke('sns:deleteTopic', { topicArn, region }),
    getTopicAttributes: (topicArn, region) => ipcRenderer.invoke('sns:getTopicAttributes', { topicArn, region }),
    setTopicAttribute: (topicArn, attributeName, attributeValue, region) =>
      ipcRenderer.invoke('sns:setTopicAttribute', { topicArn, attributeName, attributeValue, region }),
    listSubscriptions: (region) => ipcRenderer.invoke('sns:listSubscriptions', { region }),
    listSubscriptionsByTopic: (topicArn, region) => ipcRenderer.invoke('sns:listSubscriptionsByTopic', { topicArn, region }),
    subscribe: (topicArn, protocol, endpoint, attributes, returnSubscriptionArn, region) =>
      ipcRenderer.invoke('sns:subscribe', { topicArn, protocol, endpoint, attributes, returnSubscriptionArn, region }),
    confirmSubscription: (topicArn, token, region) => ipcRenderer.invoke('sns:confirmSubscription', { topicArn, token, region }),
    unsubscribe: (subscriptionArn, region) => ipcRenderer.invoke('sns:unsubscribe', { subscriptionArn, region }),
    getSubscriptionAttributes: (subscriptionArn, region) => ipcRenderer.invoke('sns:getSubscriptionAttributes', { subscriptionArn, region }),
    setSubscriptionAttribute: (subscriptionArn, attributeName, attributeValue, region) =>
      ipcRenderer.invoke('sns:setSubscriptionAttribute', { subscriptionArn, attributeName, attributeValue, region }),
    publish: (topicArn, targetArn, phoneNumber, message, subject, messageStructure, messageAttributes, messageGroupId, messageDeduplicationId, region) =>
      ipcRenderer.invoke('sns:publish', { topicArn, targetArn, phoneNumber, message, subject, messageStructure, messageAttributes, messageGroupId, messageDeduplicationId, region }),
    publishBatch: (topicArn, entries, region) => ipcRenderer.invoke('sns:publishBatch', { topicArn, entries, region }),
    addPermission: (topicArn, label, awsAccountIDs, actionNames, region) =>
      ipcRenderer.invoke('sns:addPermission', { topicArn, label, awsAccountIDs, actionNames, region }),
    removePermission: (topicArn, label, region) => ipcRenderer.invoke('sns:removePermission', { topicArn, label, region }),
    tagResource: (resourceArn, tags, region) => ipcRenderer.invoke('sns:tagResource', { resourceArn, tags, region }),
    untagResource: (resourceArn, tagKeys, region) => ipcRenderer.invoke('sns:untagResource', { resourceArn, tagKeys, region }),
    listTagsForResource: (resourceArn, region) => ipcRenderer.invoke('sns:listTagsForResource', { resourceArn, region }),
    getDataProtectionPolicy: (resourceArn, region) => ipcRenderer.invoke('sns:getDataProtectionPolicy', { resourceArn, region }),
    putDataProtectionPolicy: (resourceArn, policyDocument, region) => ipcRenderer.invoke('sns:putDataProtectionPolicy', { resourceArn, policyDocument, region }),
    listPlatformApplications: (region) => ipcRenderer.invoke('sns:listPlatformApplications', { region }),
    createPlatformApplication: (name, platform, attributes, region) =>
      ipcRenderer.invoke('sns:createPlatformApplication', { name, platform, attributes, region }),
    deletePlatformApplication: (platformApplicationArn, region) => ipcRenderer.invoke('sns:deletePlatformApplication', { platformApplicationArn, region }),
    getPlatformApplicationAttributes: (platformApplicationArn, region) => ipcRenderer.invoke('sns:getPlatformApplicationAttributes', { platformApplicationArn, region }),
    setPlatformApplicationAttributes: (platformApplicationArn, attributes, region) =>
      ipcRenderer.invoke('sns:setPlatformApplicationAttributes', { platformApplicationArn, attributes, region }),
    createPlatformEndpoint: (platformApplicationArn, token, customUserData, attributes, region) =>
      ipcRenderer.invoke('sns:createPlatformEndpoint', { platformApplicationArn, token, customUserData, attributes, region }),
    deleteEndpoint: (endpointArn, region) => ipcRenderer.invoke('sns:deleteEndpoint', { endpointArn, region }),
    getEndpointAttributes: (endpointArn, region) => ipcRenderer.invoke('sns:getEndpointAttributes', { endpointArn, region }),
    setEndpointAttributes: (endpointArn, attributes, region) => ipcRenderer.invoke('sns:setEndpointAttributes', { endpointArn, attributes, region }),
    listEndpointsByPlatformApplication: (platformApplicationArn, region) =>
      ipcRenderer.invoke('sns:listEndpointsByPlatformApplication', { platformApplicationArn, region }),
    setSMSAttributes: (attributes, region) => ipcRenderer.invoke('sns:setSMSAttributes', { attributes, region }),
    getSMSAttributes: (attributeNames, region) => ipcRenderer.invoke('sns:getSMSAttributes', { attributeNames, region }),
    checkIfPhoneNumberIsOptedOut: (phoneNumber, region) => ipcRenderer.invoke('sns:checkIfPhoneNumberIsOptedOut', { phoneNumber, region }),
    optInPhoneNumber: (phoneNumber, region) => ipcRenderer.invoke('sns:optInPhoneNumber', { phoneNumber, region }),
    listPhoneNumbersOptedOut: (region) => ipcRenderer.invoke('sns:listPhoneNumbersOptedOut', { region }),
    listOriginationNumbers: (region) => ipcRenderer.invoke('sns:listOriginationNumbers', { region }),
    getSMSSandboxAccountStatus: (region) => ipcRenderer.invoke('sns:getSMSSandboxAccountStatus', { region }),
    createSMSSandboxPhoneNumber: (phoneNumber, languageCode, region) =>
      ipcRenderer.invoke('sns:createSMSSandboxPhoneNumber', { phoneNumber, languageCode, region }),
    verifySMSSandboxPhoneNumber: (phoneNumber, oneTimePassword, region) =>
      ipcRenderer.invoke('sns:verifySMSSandboxPhoneNumber', { phoneNumber, oneTimePassword, region }),
    deleteSMSSandboxPhoneNumber: (phoneNumber, region) => ipcRenderer.invoke('sns:deleteSMSSandboxPhoneNumber', { phoneNumber, region }),
    listSMSSandboxPhoneNumbers: (region) => ipcRenderer.invoke('sns:listSMSSandboxPhoneNumbers', { region })
  },
  instance: {
    setSelected: (port) => ipcRenderer.invoke('instance:setSelected', port)
  },
  mildstack: {
    instances: () => ipcRenderer.invoke('mildstack:instances'),
    start: (port) => ipcRenderer.invoke('mildstack:start', port),
    stop: (port?, all?) => ipcRenderer.invoke('mildstack:stop', { port, all }),
    delete: (port?, all?) => ipcRenderer.invoke('mildstack:delete', { port, all }),
    validateInstance: () => ipcRenderer.invoke('mildstack:validateInstance')
  }
}

// Use `contextBridge` APIs to expose Electron APIs to
// renderer only if context isolation is enabled, otherwise
// just add to the DOM global.
if (process.contextIsolated) {
  try {
    contextBridge.exposeInMainWorld('electron', electronAPI)
    contextBridge.exposeInMainWorld('api', api)
  } catch (error) {
    console.error(error)
  }
} else {
  // @ts-ignore (define in dts)
  window.electron = electronAPI
  // @ts-ignore (define in dts)
  window.api = api
}
