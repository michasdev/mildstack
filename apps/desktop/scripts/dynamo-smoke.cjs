#!/usr/bin/env node
'use strict';

const { randomUUID } = require('node:crypto');
const {
  DynamoDBClient,
  ListTablesCommand,
  CreateTableCommand,
  DescribeTableCommand,
  DeleteTableCommand,
  PutItemCommand,
  GetItemCommand,
  DeleteItemCommand,
  UpdateItemCommand,
  QueryCommand,
  ScanCommand,
  BatchWriteItemCommand,
  BatchGetItemCommand,
  TransactWriteItemsCommand,
  TransactGetItemsCommand,
} = require('@aws-sdk/client-dynamodb');

// Parse arguments to find port
const args = process.argv.slice(2);
let port = 4566;
for (let i = 0; i < args.length; i++) {
  if (args[i] === '--port' && args[i + 1]) {
    port = parseInt(args[i + 1], 10);
  }
}

main().catch((error) => {
  console.error('\nDynamoDB smoke test failed');
  console.error(error instanceof Error ? error.stack || error.message : error);
  process.exitCode = 1;
});

async function main() {
  const endpoint = process.env.MILDSTACK_DYNAMODB_ENDPOINT || process.env.AWS_DYNAMODB_ENDPOINT || `http://localhost:${port}`;

  console.log(`Running AWS SDK smoke mode against ${endpoint}`);
  const client = new DynamoDBClient({
    region: process.env.AWS_REGION || 'us-east-1',
    endpoint,
    credentials: {
      accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
      secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
    },
  });

  const smokeTable = uniqueTableName('smoke');
  const batchTable = uniqueTableName('batch');

  await execute(client, 'ListTables', new ListTablesCommand({}));

  const createOut = await execute(client, 'CreateTable', new CreateTableCommand({
    TableName: smokeTable,
    KeySchema: [
      { AttributeName: 'id', KeyType: 'HASH' },
      { AttributeName: 'sk', KeyType: 'RANGE' },
    ],
    AttributeDefinitions: [
      { AttributeName: 'id', AttributeType: 'S' },
      { AttributeName: 'sk', AttributeType: 'S' },
    ],
    BillingMode: 'PAY_PER_REQUEST',
  }));
  assertTableName(createOut, smokeTable);
  await waitForTableStatus(client, smokeTable, 'ACTIVE');

  await execute(client, 'PutItem', new PutItemCommand({
    TableName: smokeTable,
    Item: {
      id: { S: 'series#1' },
      sk: { S: '001' },
      title: { S: 'skip-one' },
    },
  }));
  await execute(client, 'PutItem', new PutItemCommand({
    TableName: smokeTable,
    Item: {
      id: { S: 'series#1' },
      sk: { S: '002' },
      title: { S: 'keep-two' },
    },
  }));
  await execute(client, 'PutItem', new PutItemCommand({
    TableName: smokeTable,
    Item: {
      id: { S: 'series#1' },
      sk: { S: '003' },
      title: { S: 'keep-three' },
    },
  }));

  const updateOut = await execute(client, 'UpdateItem', new UpdateItemCommand({
    TableName: smokeTable,
    Key: { id: { S: 'series#1' }, sk: { S: '002' } },
    UpdateExpression: 'SET title = :title ADD version :inc REMOVE archived',
    ExpressionAttributeValues: {
      ':title': { S: 'keep-two-updated' },
      ':inc': { N: '1' },
    },
    ReturnValues: 'ALL_NEW',
  }));
  expectEqual(attrValueString(updateOut.Attributes.title), 'keep-two-updated', 'updated title');
  expectEqual(attrValueString(updateOut.Attributes.version), '1', 'updated version');

  const getOut = await execute(client, 'GetItem', new GetItemCommand({
    TableName: smokeTable,
    Key: { id: { S: 'series#1' }, sk: { S: '002' } },
  }));
  expectEqual(attrValueString(getOut.Item.title), 'keep-two-updated', 'get item title');

  const queryOut = await execute(client, 'Query', new QueryCommand({
    TableName: smokeTable,
    KeyConditionExpression: 'id = :id AND sk BETWEEN :start AND :end',
    ExpressionAttributeValues: {
      ':id': { S: 'series#1' },
      ':start': { S: '001' },
      ':end': { S: '003' },
    },
    ScanIndexForward: false,
    Limit: 2,
  }));
  expectEqual(queryOut.Items.length, 2, 'query item count');
  expectEqual(attrValueString(queryOut.Items[0].sk), '003', 'query first sort key');
  expectEqual(attrValueString(queryOut.Items[1].sk), '002', 'query second sort key');
  expectEqual(attrValueString(queryOut.LastEvaluatedKey.sk), '002', 'query cursor');

  const beginsOut = await execute(client, 'Query', new QueryCommand({
    TableName: smokeTable,
    KeyConditionExpression: 'id = :id AND begins_with(sk, :prefix)',
    ExpressionAttributeValues: {
      ':id': { S: 'series#1' },
      ':prefix': { S: '00' },
    },
  }));
  expectEqual(beginsOut.Items.length, 3, 'begins_with query count');

  const scanOut = await execute(client, 'Scan', new ScanCommand({
    TableName: smokeTable,
    FilterExpression: 'begins_with(title, :prefix)',
    ExpressionAttributeValues: {
      ':prefix': { S: 'keep' },
    },
    Limit: 1,
  }));
  expectEqual(scanOut.Items.length, 0, 'first scan page count');
  expectEqual(attrValueString(scanOut.LastEvaluatedKey.sk), '001', 'scan cursor');

  const scanPage2 = await execute(client, 'Scan', new ScanCommand({
    TableName: smokeTable,
    FilterExpression: 'begins_with(title, :prefix)',
    ExpressionAttributeValues: {
      ':prefix': { S: 'keep' },
    },
    Limit: 1,
    ExclusiveStartKey: scanOut.LastEvaluatedKey,
  }));
  expectEqual(scanPage2.Items.length, 1, 'second scan page count');
  expectEqual(attrValueString(scanPage2.Items[0].title), 'keep-two-updated', 'scan page title');

  await execute(client, 'DeleteItem', new DeleteItemCommand({
    TableName: smokeTable,
    Key: { id: { S: 'series#1' }, sk: { S: '001' } },
  }));

  const batchCreateOut = await execute(client, 'CreateTable', new CreateTableCommand({
    TableName: batchTable,
    KeySchema: [
      { AttributeName: 'id', KeyType: 'HASH' },
    ],
    AttributeDefinitions: [
      { AttributeName: 'id', AttributeType: 'S' },
    ],
    BillingMode: 'PAY_PER_REQUEST',
  }));
  assertTableName(batchCreateOut, batchTable);
  await waitForTableStatus(client, batchTable, 'ACTIVE');

  const writeRequests = [];
  for (let i = 1; i <= 26; i += 1) {
    const id = `item#${String(i).padStart(2, '0')}`;
    writeRequests.push({
      PutRequest: {
        Item: {
          id: { S: id },
          title: { S: `title-${String(i).padStart(2, '0')}` },
        },
      },
    });
  }

  const batchWriteOut = await execute(client, 'BatchWriteItem', new BatchWriteItemCommand({
    RequestItems: {
      [batchTable]: writeRequests,
    },
  }));
  expectEqual(batchWriteOut.UnprocessedItems[batchTable].length, 1, 'batch write unprocessed count');
  expectEqual(attrValueString(batchWriteOut.UnprocessedItems[batchTable][0].PutRequest.Item.id), 'item#26', 'batch write unprocessed id');

  const batchGetOut = await execute(client, 'BatchGetItem', new BatchGetItemCommand({
    RequestItems: {
      [batchTable]: {
        Keys: [
          { id: { S: 'item#01' } },
          { id: { S: 'item#25' } },
          { id: { S: 'item#26' } },
        ],
      },
    },
  }));
  expectEqual(batchGetOut.Responses[batchTable].length, 2, 'batch get response count');
  expectEqual(attrValueString(batchGetOut.Responses[batchTable][0].id), 'item#01', 'batch get first id');
  expectEqual(attrValueString(batchGetOut.Responses[batchTable][1].id), 'item#25', 'batch get second id');

  const transactWriteOut = await execute(client, 'TransactWriteItems', new TransactWriteItemsCommand({
    TransactItems: [
      {
        Put: {
          TableName: batchTable,
          Item: {
            id: { S: 'item#27' },
            title: { S: 'title-27' },
          },
        },
      },
      {
        Delete: {
          TableName: batchTable,
          Key: { id: { S: 'item#01' } },
        },
      },
    ],
  }));
  if (!transactWriteOut) {
    throw new Error('expected transact write response');
  }

  const transactGetOut = await execute(client, 'TransactGetItems', new TransactGetItemsCommand({
    TransactItems: [
      {
        Get: {
          TableName: batchTable,
          Key: { id: { S: 'item#27' } },
        },
      },
      {
        Get: {
          TableName: batchTable,
          Key: { id: { S: 'item#02' } },
        },
      },
    ],
  }));
  expectEqual(transactGetOut.Responses.length, 2, 'transact get response count');
  expectEqual(attrValueString(transactGetOut.Responses[0].Item.id), 'item#27', 'transact get first id');
  expectEqual(attrValueString(transactGetOut.Responses[1].Item.id), 'item#02', 'transact get second id');

  await expectAwsError(
    client,
    'TransactWriteItems',
    new TransactWriteItemsCommand({
      TransactItems: [
        {
          Put: {
            TableName: batchTable,
            Item: {
              id: { S: 'item#28' },
              title: { S: 'title-28' },
            },
          },
        },
        {
          Delete: {
            TableName: batchTable,
            Key: { id: { S: 'item#28' } },
          },
        },
      ],
    }),
    'TransactionCanceledException',
  );

  await execute(client, 'DeleteTable', new DeleteTableCommand({ TableName: smokeTable }));
  await execute(client, 'DeleteTable', new DeleteTableCommand({ TableName: batchTable }));

  await expectAwsError(
    client,
    'DescribeTable',
    new DescribeTableCommand({ TableName: smokeTable }),
    'ResourceNotFoundException',
  );

  console.log('\n✓ Native AWS SDK smoke mode passed');
}

async function execute(client, name, command) {
  console.log(`\nExecuting ${name}...`);
  try {
    const response = await client.send(command);
    console.log(`✓ ${name} succeeded.`);
    console.dir(response, { depth: 4, colors: true });
    return response;
  } catch (error) {
    printAwsError(name, error);
    throw error;
  }
}

async function expectAwsError(client, name, command, expectedName) {
  try {
    await client.send(command);
  } catch (error) {
    printAwsError(name, error);
    expectEqual(error?.name, expectedName, `${name} error name`);
    return;
  }
  throw new Error(`expected ${name} to fail with ${expectedName}`);
}

async function waitForTableStatus(client, tableName, status) {
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    try {
      const response = await client.send(new DescribeTableCommand({ TableName: tableName }));
      if (response?.Table?.TableStatus === status) {
        return;
      }
    } catch (error) {
      if (error?.name !== 'ResourceNotFoundException') {
        throw error;
      }
    }
    await sleep(50);
  }
  throw new Error(`table ${tableName} did not reach status ${status}`);
}

function printAwsError(name, error) {
  console.error(`\nFailed during command: ${name}`);
  if (error && typeof error === 'object') {
    console.error('Error name:', error.name || 'unknown');
    console.error('Error message:', error.message || String(error));
    if (error.$response) {
      console.error('Response status:', error.$response.statusCode);
      console.error('Response headers:', error.$response.headers);
    }
  } else {
    console.error(error);
  }
}

function expectEqual(actual, expected, label) {
  if (actual !== expected) {
    throw new Error(`unexpected ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
  }
}

function attrValueString(value) {
  if (!value) {
    throw new Error('expected attribute value to be present');
  }
  if (typeof value.S === 'string') {
    return value.S;
  }
  if (typeof value.N === 'string') {
    return value.N;
  }
  throw new Error(`unexpected attribute value shape: ${JSON.stringify(value)}`);
}

function assertTableName(output, tableName) {
  if (output?.TableDescription?.TableName !== tableName) {
    throw new Error(`unexpected table name in response: ${output?.TableDescription?.TableName || 'missing'} want ${tableName}`);
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function uniqueTableName(prefix) {
  return `mildstack_${prefix}_${Date.now().toString(36)}_${randomUUID().slice(0, 8)}`;
}
