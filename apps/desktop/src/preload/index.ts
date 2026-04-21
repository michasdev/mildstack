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
  serve(port: number): Promise<{ success: boolean; error?: string }>
  stop(port?: number, all?: boolean): Promise<{ success: boolean; error?: string }>
  delete(port?: number, all?: boolean): Promise<{ success: boolean; error?: string }>
  validateInstance(): Promise<{ valid: boolean; error?: string }>
}

// Custom APIs for renderer
const api: { s3: S3BrowserApi; dynamodb: DynamoDBBrowserApi; sqs: SQSBrowserApi; instance: InstanceApi; mildstack: MildStackApi } = {
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
  instance: {
    setSelected: (port) => ipcRenderer.invoke('instance:setSelected', port)
  },
  mildstack: {
    instances: () => ipcRenderer.invoke('mildstack:instances'),
    serve: (port) => ipcRenderer.invoke('mildstack:serve', port),
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
