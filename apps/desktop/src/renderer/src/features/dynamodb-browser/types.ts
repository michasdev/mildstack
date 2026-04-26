export interface DynamoDBKeySchema {
  AttributeName: string
  KeyType: 'HASH' | 'RANGE'
}

export interface DynamoDBAttributeDefinition {
  AttributeName: string
  AttributeType: 'S' | 'N' | 'B'
}

export interface DynamoDBProjection {
  ProjectionType: string
  NonKeyAttributes: string[]
}

export interface DynamoDBGlobalSecondaryIndex {
  IndexName: string
  IndexStatus: string
  KeySchema: DynamoDBKeySchema[]
  Projection: DynamoDBProjection
  ItemCount: number
  IndexSizeBytes: number
}

export interface DynamoDBLocalSecondaryIndex {
  IndexName: string
  KeySchema: DynamoDBKeySchema[]
  Projection: DynamoDBProjection
  ItemCount: number
  IndexSizeBytes: number
}

export interface DynamoDBTableSummary {
  TableName: string
  TableStatus: string
  KeySchema: DynamoDBKeySchema[]
  AttributeDefinitions: DynamoDBAttributeDefinition[]
  ItemCount: number
  TableSizeBytes: number
  CreationDateTime?: string
  BillingMode?: string
  GlobalSecondaryIndexes?: DynamoDBGlobalSecondaryIndex[]
  LocalSecondaryIndexes?: DynamoDBLocalSecondaryIndex[]
}

export type AttributeValue =
  | { S: string }
  | { N: string }
  | { B: string }
  | { BOOL: boolean }
  | { NULL: true }
  | { L: AttributeValue[] }
  | { M: Record<string, AttributeValue> }
  | { SS: string[] }
  | { NS: string[] }
  | { BS: string[] }

export type DynamoDBItem = Record<string, AttributeValue>

export interface ScanResult {
  items: DynamoDBItem[]
  lastEvaluatedKey?: DynamoDBItem
  count: number
  scannedCount: number
}

export interface DynamoDBBrowserApi {
  listTables(region?: string): Promise<string[]>
  describeTable(tableName: string, region?: string): Promise<DynamoDBTableSummary>
  createTable(
    tableName: string,
    keySchema: DynamoDBKeySchema[],
    attributeDefinitions: DynamoDBAttributeDefinition[],
    region?: string
  ): Promise<void>
  deleteTable(tableName: string, region?: string): Promise<void>
  scan(
    tableName: string,
    exclusiveStartKey?: DynamoDBItem,
    limit?: number,
    region?: string,
    filterExpression?: string,
    expressionAttributeNames?: Record<string, string>,
    expressionAttributeValues?: DynamoDBItem
  ): Promise<ScanResult>
  query(
    tableName: string,
    keyConditionExpression: string,
    expressionAttributeNames?: Record<string, string>,
    expressionAttributeValues?: DynamoDBItem,
    indexName?: string,
    filterExpression?: string,
    exclusiveStartKey?: DynamoDBItem,
    limit?: number,
    scanIndexForward?: boolean,
    region?: string
  ): Promise<ScanResult>
  putItem(tableName: string, item: DynamoDBItem, region?: string): Promise<void>
  deleteItem(tableName: string, key: DynamoDBItem, region?: string): Promise<void>
  getItem(tableName: string, key: DynamoDBItem, region?: string): Promise<DynamoDBItem | null>
}

export const COMPARISON_OPERATORS = [
  { value: '=', label: '= (equals)' },
  { value: '<>', label: '<> (not equals)' },
  { value: '<', label: '< (less than)' },
  { value: '<=', label: '<= (less or equal)' },
  { value: '>', label: '> (greater than)' },
  { value: '>=', label: '>= (greater or equal)' },
  { value: 'begins_with', label: 'begins_with' },
  { value: 'contains', label: 'contains' },
  { value: 'attribute_exists', label: 'attribute_exists' },
  { value: 'attribute_not_exists', label: 'attribute_not_exists' }
] as const

export type ComparisonOperator = (typeof COMPARISON_OPERATORS)[number]['value']

export interface FilterCondition {
  id: string
  attribute: string
  operator: ComparisonOperator
  value: string
  valueType: 'S' | 'N' | 'BOOL'
}

export type FetchMode = 'scan' | 'query'


/**
 * Convert a single DynamoDB AttributeValue to a plain JS value.
 * { S: "foo" } → "foo", { N: "42" } → 42, etc.
 */
export function attrToValue(av: AttributeValue): unknown {
  if ('S' in av) return av.S
  if ('N' in av) return Number(av.N)
  if ('BOOL' in av) return av.BOOL
  if ('NULL' in av) return null
  if ('B' in av) return av.B
  if ('L' in av) return (av.L as AttributeValue[]).map(attrToValue)
  if ('M' in av) {
    const obj: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(av.M as Record<string, AttributeValue>)) {
      obj[k] = attrToValue(v)
    }
    return obj
  }
  if ('SS' in av) return av.SS
  if ('NS' in av) return (av.NS as string[]).map(Number)
  if ('BS' in av) return av.BS
  return av
}

/**
 * Convert a DynamoDB item (map of AttributeValues) to a plain JS object.
 */
export function marshallToFriendlyJson(item: DynamoDBItem): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, av] of Object.entries(item)) {
    result[key] = attrToValue(av)
  }
  return result
}

/**
 * Convert a plain JS value back to a DynamoDB AttributeValue.
 */
export function valueToAttr(value: unknown): AttributeValue {
  if (value === null || value === undefined) return { NULL: true }
  if (typeof value === 'string') return { S: value }
  if (typeof value === 'number') return { N: String(value) }
  if (typeof value === 'boolean') return { BOOL: value }
  if (Array.isArray(value)) {
    return { L: value.map(valueToAttr) }
  }
  if (typeof value === 'object') {
    const m: Record<string, AttributeValue> = {}
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      m[k] = valueToAttr(v)
    }
    return { M: m }
  }
  return { S: String(value) }
}

/**
 * Convert a plain JS object back to a DynamoDB item (map of AttributeValues).
 */
export function unmarshallFromFriendlyJson(json: Record<string, unknown>): DynamoDBItem {
  const result: DynamoDBItem = {}
  for (const [key, value] of Object.entries(json)) {
    result[key] = valueToAttr(value)
  }
  return result
}
