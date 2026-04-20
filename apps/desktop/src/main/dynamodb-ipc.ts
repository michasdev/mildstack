import { getActiveInstancePort } from './instance-state'
import { registerValidatedHandler } from './ipc-middleware'
import {
  DynamoDBClient,
  ListTablesCommand,
  DescribeTableCommand,
  CreateTableCommand,
  DeleteTableCommand,
  ScanCommand,
  QueryCommand,
  PutItemCommand,
  DeleteItemCommand,
  GetItemCommand,
  type KeySchemaElement,
  type AttributeDefinition,
  type ScanCommandInput,
  type QueryCommandInput
} from '@aws-sdk/client-dynamodb'

type DescribeTableArgs = {
  tableName: string
  region?: string
}

type CreateTableArgs = {
  tableName: string
  keySchema: KeySchemaElement[]
  attributeDefinitions: AttributeDefinition[]
  region?: string
}

type ScanArgs = {
  tableName: string
  exclusiveStartKey?: Record<string, any>
  limit?: number
  region?: string
  filterExpression?: string
  expressionAttributeNames?: Record<string, string>
  expressionAttributeValues?: Record<string, any>
}

type QueryArgs = {
  tableName: string
  indexName?: string
  keyConditionExpression: string
  filterExpression?: string
  expressionAttributeNames?: Record<string, string>
  expressionAttributeValues?: Record<string, any>
  exclusiveStartKey?: Record<string, any>
  limit?: number
  scanIndexForward?: boolean
  region?: string
}

type PutItemArgs = {
  tableName: string
  item: Record<string, any>
  region?: string
}

type DeleteItemArgs = {
  tableName: string
  key: Record<string, any>
  region?: string
}

type GetItemArgs = {
  tableName: string
  key: Record<string, any>
  region?: string
}

type DynamoDBClientCacheEntry = {
  region: string
  endpoint: string
  client: DynamoDBClient
}

const clientCache = new Map<string, DynamoDBClientCacheEntry>()

export function registerDynamoDBIpcHandlers(): void {
  registerValidatedHandler('dynamodb:listTables', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListTablesCommand({}))
    return response.TableNames ?? []
  })

  registerValidatedHandler('dynamodb:describeTable', async (_event, args: DescribeTableArgs) => {
    const response = await getClient(args.region).send(
      new DescribeTableCommand({ TableName: args.tableName })
    )
    const table = response.Table
    if (!table) throw new Error(`Table ${args.tableName} not found`)

    return {
      TableName: table.TableName ?? args.tableName,
      TableStatus: table.TableStatus ?? 'UNKNOWN',
      KeySchema: (table.KeySchema ?? []).map((k) => ({
        AttributeName: k.AttributeName ?? '',
        KeyType: k.KeyType ?? 'HASH'
      })),
      AttributeDefinitions: (table.AttributeDefinitions ?? []).map((a) => ({
        AttributeName: a.AttributeName ?? '',
        AttributeType: a.AttributeType ?? 'S'
      })),
      ItemCount: table.ItemCount ?? 0,
      TableSizeBytes: table.TableSizeBytes ?? 0,
      CreationDateTime: table.CreationDateTime?.toISOString(),
      BillingMode: table.BillingModeSummary?.BillingMode ?? 'PAY_PER_REQUEST',
      GlobalSecondaryIndexes: (table.GlobalSecondaryIndexes ?? []).map((gsi) => ({
        IndexName: gsi.IndexName ?? '',
        IndexStatus: gsi.IndexStatus ?? 'UNKNOWN',
        KeySchema: (gsi.KeySchema ?? []).map((k) => ({
          AttributeName: k.AttributeName ?? '',
          KeyType: k.KeyType ?? 'HASH'
        })),
        Projection: {
          ProjectionType: gsi.Projection?.ProjectionType ?? 'ALL',
          NonKeyAttributes: gsi.Projection?.NonKeyAttributes ?? []
        },
        ItemCount: gsi.ItemCount ?? 0,
        IndexSizeBytes: gsi.IndexSizeBytes ?? 0
      })),
      LocalSecondaryIndexes: (table.LocalSecondaryIndexes ?? []).map((lsi) => ({
        IndexName: lsi.IndexName ?? '',
        KeySchema: (lsi.KeySchema ?? []).map((k) => ({
          AttributeName: k.AttributeName ?? '',
          KeyType: k.KeyType ?? 'HASH'
        })),
        Projection: {
          ProjectionType: lsi.Projection?.ProjectionType ?? 'ALL',
          NonKeyAttributes: lsi.Projection?.NonKeyAttributes ?? []
        },
        ItemCount: lsi.ItemCount ?? 0,
        IndexSizeBytes: lsi.IndexSizeBytes ?? 0
      }))
    }
  })

  registerValidatedHandler('dynamodb:createTable', async (_event, args: CreateTableArgs) => {
    await getClient(args.region).send(
      new CreateTableCommand({
        TableName: args.tableName,
        KeySchema: args.keySchema,
        AttributeDefinitions: args.attributeDefinitions,
        BillingMode: 'PAY_PER_REQUEST'
      })
    )
    return null
  })

  registerValidatedHandler('dynamodb:deleteTable', async (_event, args: { tableName: string; region?: string }) => {
    await getClient(args.region).send(
      new DeleteTableCommand({ TableName: args.tableName })
    )
    return null
  })

  registerValidatedHandler('dynamodb:scan', async (_event, args: ScanArgs) => {
    const params: ScanCommandInput = {
      TableName: args.tableName,
      ExclusiveStartKey: args.exclusiveStartKey,
      Limit: args.limit ?? 50
    }
    if (args.filterExpression) {
      params.FilterExpression = args.filterExpression
    }
    if (args.expressionAttributeNames && Object.keys(args.expressionAttributeNames).length > 0) {
      params.ExpressionAttributeNames = args.expressionAttributeNames
    }
    if (args.expressionAttributeValues && Object.keys(args.expressionAttributeValues).length > 0) {
      params.ExpressionAttributeValues = args.expressionAttributeValues
    }
    const response = await getClient(args.region).send(new ScanCommand(params))

    return {
      items: response.Items ?? [],
      lastEvaluatedKey: response.LastEvaluatedKey ?? undefined,
      count: response.Count ?? 0,
      scannedCount: response.ScannedCount ?? 0
    }
  })

  registerValidatedHandler('dynamodb:query', async (_event, args: QueryArgs) => {
    const params: QueryCommandInput = {
      TableName: args.tableName,
      KeyConditionExpression: args.keyConditionExpression,
      ExclusiveStartKey: args.exclusiveStartKey,
      Limit: args.limit ?? 50,
      ScanIndexForward: args.scanIndexForward ?? true
    }
    if (args.indexName) {
      params.IndexName = args.indexName
    }
    if (args.filterExpression) {
      params.FilterExpression = args.filterExpression
    }
    if (args.expressionAttributeNames && Object.keys(args.expressionAttributeNames).length > 0) {
      params.ExpressionAttributeNames = args.expressionAttributeNames
    }
    if (args.expressionAttributeValues && Object.keys(args.expressionAttributeValues).length > 0) {
      params.ExpressionAttributeValues = args.expressionAttributeValues
    }
    const response = await getClient(args.region).send(new QueryCommand(params))

    return {
      items: response.Items ?? [],
      lastEvaluatedKey: response.LastEvaluatedKey ?? undefined,
      count: response.Count ?? 0,
      scannedCount: response.ScannedCount ?? 0
    }
  })

  registerValidatedHandler('dynamodb:putItem', async (_event, args: PutItemArgs) => {
    await getClient(args.region).send(
      new PutItemCommand({
        TableName: args.tableName,
        Item: args.item
      })
    )
    return null
  })

  registerValidatedHandler('dynamodb:deleteItem', async (_event, args: DeleteItemArgs) => {
    await getClient(args.region).send(
      new DeleteItemCommand({
        TableName: args.tableName,
        Key: args.key
      })
    )
    return null
  })

  registerValidatedHandler('dynamodb:getItem', async (_event, args: GetItemArgs) => {
    const response = await getClient(args.region).send(
      new GetItemCommand({
        TableName: args.tableName,
        Key: args.key
      })
    )
    return response.Item ?? null
  })
}

function getClient(region = 'us-east-1'): DynamoDBClient {
  const normalizedRegion = normalizeRegion(region)
  const endpoint = resolveDynamoDBEndpoint()
  const cacheKey = `${normalizedRegion}:${endpoint}`
  const cached = clientCache.get(cacheKey)
  if (cached) {
    return cached.client
  }

  const client = new DynamoDBClient({
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

function resolveDynamoDBEndpoint(): string {
  const port = getActiveInstancePort()
  return process.env.MILDSTACK_DYNAMODB_ENDPOINT || process.env.AWS_DYNAMODB_ENDPOINT || `http://127.0.0.1:${port}`
}

function normalizeRegion(region?: string): string {
  const trimmed = region?.trim()
  return trimmed || 'us-east-1'
}
