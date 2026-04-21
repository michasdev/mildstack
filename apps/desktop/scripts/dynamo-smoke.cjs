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
  UpdateTimeToLiveCommand,
  DescribeTimeToLiveCommand,
} = require('@aws-sdk/client-dynamodb');

// Parse arguments to find port
const args = process.argv.slice(2);
let port = 4566;
let debug = false;
for (let i = 0; i < args.length; i++) {
  if (args[i] === '--port' && args[i + 1]) {
    port = parseInt(args[i + 1], 10);
  }
  if (args[i] === '--debug') {
    debug = true;
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

  const createdTables = [];
  
  async function createTable(name, options = {}) {
    const tableName = uniqueTableName(name);
    await execute(client, `CreateTable (${name})`, new CreateTableCommand({
      TableName: tableName,
      KeySchema: [
        { AttributeName: 'pk', KeyType: 'HASH' },
        { AttributeName: 'sk', KeyType: 'RANGE' },
      ],
      AttributeDefinitions: [
        { AttributeName: 'pk', AttributeType: 'S' },
        { AttributeName: 'sk', AttributeType: 'S' },
        ...(options.extraAttributes || []),
      ],
      BillingMode: 'PAY_PER_REQUEST',
      ...(options.gsi && { GlobalSecondaryIndexes: options.gsi }),
      ...(options.lsi && { LocalSecondaryIndexes: options.lsi }),
    }));
    createdTables.push(tableName);
    await waitForTableStatus(client, tableName, 'ACTIVE');
    return tableName;
  }

  try {
    // --- Setup Tables ---
    console.log('\n--- Setup Tables ---');
    const crudTable = await createTable('crud');
    const queryTable = await createTable('query');
    const batchTable = await createTable('batch');
    const txTable = await createTable('tx');
    const indexTable = await createTable('index', {
      extraAttributes: [
        { AttributeName: 'gsi1pk', AttributeType: 'S' },
        { AttributeName: 'gsi1sk', AttributeType: 'N' },
        { AttributeName: 'lsi1sk', AttributeType: 'N' },
      ],
      gsi: [{
        IndexName: 'gsi1',
        KeySchema: [
          { AttributeName: 'gsi1pk', KeyType: 'HASH' },
          { AttributeName: 'gsi1sk', KeyType: 'RANGE' },
        ],
        Projection: { ProjectionType: 'ALL' },
      }],
      lsi: [{
        IndexName: 'lsi1',
        KeySchema: [
          { AttributeName: 'pk', KeyType: 'HASH' },
          { AttributeName: 'lsi1sk', KeyType: 'RANGE' },
        ],
        Projection: { ProjectionType: 'ALL' },
      }]
    });

    // --- Validation: CRUD and Expressions ---
    console.log('\n--- Validation: CRUD and Expressions ---');
    await execute(client, 'PutItem', new PutItemCommand({
      TableName: crudTable,
      Item: { pk: { S: 'crud#1' }, sk: { S: 'meta' }, val: { N: '10' } }
    }));

    const get1 = await execute(client, 'GetItem', new GetItemCommand({
      TableName: crudTable,
      Key: { pk: { S: 'crud#1' }, sk: { S: 'meta' } },
      ConsistentRead: true
    }));
    expectEqual(attrValueString(get1.Item.val), '10', 'PutItem val');

    await expectAwsError(client, 'PutItem (Conditional Check Failed)', new PutItemCommand({
      TableName: crudTable,
      Item: { pk: { S: 'crud#1' }, sk: { S: 'meta' }, val: { N: '20' } },
      ConditionExpression: 'attribute_not_exists(pk)'
    }), 'ConditionalCheckFailedException');

    const up1 = await execute(client, 'UpdateItem', new UpdateItemCommand({
      TableName: crudTable,
      Key: { pk: { S: 'crud#1' }, sk: { S: 'meta' } },
      UpdateExpression: 'SET val = val + :inc, #nm = :name',
      ExpressionAttributeNames: { '#nm': 'name' },
      ExpressionAttributeValues: { ':inc': { N: '5' }, ':name': { S: 'tester' } },
      ReturnValues: 'ALL_NEW'
    }));
    expectEqual(attrValueString(up1.Attributes.val), '15', 'UpdateItem math (+5)');
    expectEqual(attrValueString(up1.Attributes.name), 'tester', 'UpdateItem string');

    await execute(client, 'DeleteItem', new DeleteItemCommand({
      TableName: crudTable,
      Key: { pk: { S: 'crud#1' }, sk: { S: 'meta' } }
    }));

    const get2 = await execute(client, 'GetItem (After Delete)', new GetItemCommand({
      TableName: crudTable,
      Key: { pk: { S: 'crud#1' }, sk: { S: 'meta' } }
    }));
    expectEqual(get2.Item, undefined, 'Item should be deleted');


    // --- Validation: Query, Scan and Pagination ---
    console.log('\n--- Validation: Query, Scan and Pagination ---');
    for (let i = 1; i <= 10; i++) {
      await execute(client, `PutItem (Q/S ${i})`, new PutItemCommand({
        TableName: queryTable,
        Item: {
          pk: { S: 'grp#1' },
          sk: { S: `item#${String(i).padStart(2, '0')}` },
          active: { S: i % 2 === 0 ? 'true' : 'false' }
        }
      }));
    }

    const q1 = await execute(client, 'Query (Limit)', new QueryCommand({
      TableName: queryTable,
      KeyConditionExpression: 'pk = :pk',
      ExpressionAttributeValues: { ':pk': { S: 'grp#1' } },
      Limit: 3
    }));
    expectEqual(q1.Items?.length, 3, 'Query limit 3');
    expectEqual(attrValueString(q1.LastEvaluatedKey.sk), 'item#03', 'Query LEK');

    const q2 = await execute(client, 'Query (ExclusiveStartKey)', new QueryCommand({
      TableName: queryTable,
      KeyConditionExpression: 'pk = :pk',
      ExpressionAttributeValues: { ':pk': { S: 'grp#1' } },
      ExclusiveStartKey: q1.LastEvaluatedKey
    }));
    expectEqual(q2.Items?.length, 7, 'Query pagination remaining');

    const q3 = await execute(client, 'Query (FilterExpression)', new QueryCommand({
      TableName: queryTable,
      KeyConditionExpression: 'pk = :pk',
      FilterExpression: 'active = :active',
      ExpressionAttributeValues: { ':pk': { S: 'grp#1' }, ':active': { S: 'true' } }
    }));
    expectEqual(q3.Items?.length, 5, 'Query filter active=true');

    const s1 = await execute(client, 'Scan (Limit)', new ScanCommand({
      TableName: queryTable,
      Limit: 4
    }));
    expectEqual(s1.Items?.length, 4, 'Scan limit 4');

    const s2 = await execute(client, 'Scan (ExclusiveStartKey)', new ScanCommand({
      TableName: queryTable,
      ExclusiveStartKey: s1.LastEvaluatedKey
    }));
    expectEqual(s2.Items?.length, 6, 'Scan pagination remaining');


    // --- Validation: GSI and LSI ---
    console.log('\n--- Validation: GSI and LSI ---');
    for (let i = 1; i <= 5; i++) {
      await execute(client, `PutItem (Idx ${i})`, new PutItemCommand({
        TableName: indexTable,
        Item: {
          pk: { S: 'idx#1' },
          sk: { S: `item#${i}` },
          gsi1pk: { S: 'type#A' },
          gsi1sk: { N: `${i * 10}` },
          lsi1sk: { N: `${i * 100}` }
        }
      }));
    }

    const gsiQ = await execute(client, 'Query (GSI)', new QueryCommand({
      TableName: indexTable,
      IndexName: 'gsi1',
      KeyConditionExpression: 'gsi1pk = :gsi1pk AND gsi1sk >= :gsi1sk',
      ExpressionAttributeValues: { ':gsi1pk': { S: 'type#A' }, ':gsi1sk': { N: '30' } }
    }));
    expectEqual(gsiQ.Items?.length, 3, 'GSI query count'); // 30, 40, 50

    const lsiQ = await execute(client, 'Query (LSI)', new QueryCommand({
      TableName: indexTable,
      IndexName: 'lsi1',
      KeyConditionExpression: 'pk = :pk AND lsi1sk <= :lsi1sk',
      ExpressionAttributeValues: { ':pk': { S: 'idx#1' }, ':lsi1sk': { N: '200' } }
    }));
    expectEqual(lsiQ.Items?.length, 2, 'LSI query count'); // 100, 200


    // --- Validation: Batch Operations ---
    console.log('\n--- Validation: Batch Operations ---');
    const writeReqs = [];
    for (let i = 1; i <= 25; i++) {
      writeReqs.push({ PutRequest: { Item: { pk: { S: `batch#${i}` }, sk: { S: 'meta' } } } });
    }

    await execute(client, 'BatchWriteItem (25 items)', new BatchWriteItemCommand({
      RequestItems: { [batchTable]: writeReqs }
    }));

    const getReqs = writeReqs.map(r => ({ pk: r.PutRequest.Item.pk, sk: r.PutRequest.Item.sk }));
    const bg = await execute(client, 'BatchGetItem (25 items)', new BatchGetItemCommand({
      RequestItems: { [batchTable]: { Keys: getReqs } }
    }));
    expectEqual(bg.Responses?.[batchTable]?.length, 25, 'BatchGetItem count');

    const delReqs = getReqs.map(k => ({ DeleteRequest: { Key: k } }));
    await execute(client, 'BatchWriteItem (Delete 25 items)', new BatchWriteItemCommand({
      RequestItems: { [batchTable]: delReqs }
    }));

    const bg2 = await execute(client, 'BatchGetItem (After Delete)', new BatchGetItemCommand({
      RequestItems: { [batchTable]: { Keys: [getReqs[0]] } }
    }));
    expectEqual(bg2.Responses?.[batchTable]?.length || 0, 0, 'BatchGetItem after delete');


    // --- Validation: Transactions ---
    console.log('\n--- Validation: Transactions ---');
    await execute(client, 'PutItem (Tx Initial)', new PutItemCommand({
      TableName: txTable,
      Item: { pk: { S: 'tx#1' }, sk: { S: 'meta' }, val: { N: '10' } }
    }));

    await execute(client, 'TransactWriteItems', new TransactWriteItemsCommand({
      TransactItems: [
        { Put: { TableName: txTable, Item: { pk: { S: 'tx#2' }, sk: { S: 'meta' } } } },
        { Update: {
            TableName: txTable,
            Key: { pk: { S: 'tx#1' }, sk: { S: 'meta' } },
            UpdateExpression: 'SET val = val + :inc',
            ExpressionAttributeValues: { ':inc': { N: '5' } }
        } },
        { ConditionCheck: {
            TableName: txTable,
            Key: { pk: { S: 'tx#1' }, sk: { S: 'meta' } },
            ConditionExpression: 'attribute_exists(pk)'
        } }
      ]
    }));

    const tg = await execute(client, 'TransactGetItems', new TransactGetItemsCommand({
      TransactItems: [
        { Get: { TableName: txTable, Key: { pk: { S: 'tx#1' }, sk: { S: 'meta' } } } },
        { Get: { TableName: txTable, Key: { pk: { S: 'tx#2' }, sk: { S: 'meta' } } } }
      ]
    }));
    expectEqual(tg.Responses?.length, 2, 'TransactGetItems count');
    expectEqual(attrValueString(tg.Responses[0].Item.val), '15', 'Transact updated val');
    expectEqual(attrValueString(tg.Responses[1].Item.pk), 'tx#2', 'Transact put pk');

    await expectAwsError(client, 'TransactWriteItems (Failing)', new TransactWriteItemsCommand({
      TransactItems: [
        { Put: { TableName: txTable, Item: { pk: { S: 'tx#3' }, sk: { S: 'meta' } } } },
        { ConditionCheck: {
            TableName: txTable,
            Key: { pk: { S: 'tx#999' }, sk: { S: 'meta' } },
            ConditionExpression: 'attribute_exists(pk)'
        } }
      ]
    }), 'TransactionCanceledException');

    const get3 = await execute(client, 'GetItem (Rolled Back Tx)', new GetItemCommand({
      TableName: txTable,
      Key: { pk: { S: 'tx#3' }, sk: { S: 'meta' } }
    }));
    expectEqual(get3.Item, undefined, 'Item should be rolled back');


    // --- Validation: TTL ---
    console.log('\n--- Validation: TTL ---');
    await execute(client, 'UpdateTimeToLive', new UpdateTimeToLiveCommand({
      TableName: crudTable,
      TimeToLiveSpecification: { AttributeName: 'expireAt', Enabled: true }
    }));
    
    const ttlDesc = await execute(client, 'DescribeTimeToLive', new DescribeTimeToLiveCommand({
      TableName: crudTable
    }));
    const ttlStatus = ttlDesc.TimeToLiveDescription?.TimeToLiveStatus;
    if (ttlStatus !== 'ENABLED' && ttlStatus !== 'ENABLING') {
      throw new Error(`Validation Failed [TTL Status]: got ${ttlStatus} want ENABLED or ENABLING`);
    }

    console.log('\n✓ Native AWS SDK smoke mode passed with deep behavioral validations');
  } finally {
    // --- Cleanup ---
    console.log('\n--- Cleanup ---');
    for (const tableName of createdTables) {
      try {
        await execute(client, `DeleteTable (${tableName})`, new DeleteTableCommand({ TableName: tableName }));
      } catch (err) {
        console.error(`\nFailed to delete table ${tableName}:`, err.message);
      }
    }
  }
}

async function execute(client, name, command) {
  if (debug) console.log(`\nExecuting ${name}...`);
  try {
    const response = await client.send(command);
    if (debug) {
      console.log(`✓ ${name} succeeded.`);
      console.dir(response, { depth: 4, colors: true });
    } else {
      process.stdout.write('.');
    }
    return response;
  } catch (error) {
    if (!debug) console.log(''); // newline for error
    printAwsError(name, error);
    throw error;
  }
}

async function expectAwsError(client, name, command, expectedName) {
  if (debug) console.log(`\nExecuting ${name} (expecting error)...`);
  try {
    await client.send(command);
  } catch (error) {
    if (debug) {
      console.log(`✓ ${name} failed as expected with ${error.name}.`);
    } else {
      process.stdout.write('.');
    }
    expectEqual(error?.name, expectedName, `${name} error name`);
    return;
  }
  if (!debug) console.log('');
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
      console.error('Response status:', error.$response?.statusCode);
      console.error('Response headers:', error.$response?.headers);
    }
  } else {
    console.error(error);
  }
}

function expectEqual(actual, expected, label) {
  if (actual !== expected) {
    throw new Error(`Validation Failed [${label}]: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
  }
}

function attrValueString(value) {
  if (!value) {
    throw new Error('expected attribute value to be present');
  }
  if (typeof value.S === 'string') return value.S;
  if (typeof value.N === 'string') return value.N;
  throw new Error(`unexpected attribute value shape: ${JSON.stringify(value)}`);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function uniqueTableName(prefix) {
  return `mildstack_${prefix}_${Date.now().toString(36)}_${randomUUID().slice(0, 8)}`;
}
