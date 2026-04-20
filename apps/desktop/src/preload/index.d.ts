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

declare global {
  interface Window {
    electron: ElectronAPI
    api: {
      s3: S3BrowserApi
      dynamodb: DynamoDBBrowserApi
      instance: InstanceApi
      mildstack: MildStackApi
    }
  }
}
