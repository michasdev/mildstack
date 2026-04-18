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

  const table = uniqueTableName('native');
  const commands = [
    ['ListTables', new ListTablesCommand({})],
    ['CreateTable', new CreateTableCommand({
      TableName: table,
      KeySchema: [
        { AttributeName: 'id', KeyType: 'HASH' }
      ],
      AttributeDefinitions: [
        { AttributeName: 'id', AttributeType: 'S' }
      ],
      BillingMode: 'PAY_PER_REQUEST'
    })],
    ['DescribeTable', new DescribeTableCommand({ TableName: table })],
    ['PutItem', new PutItemCommand({
      TableName: table,
      Item: {
        id: { S: 'test-id' },
        name: { S: 'native-mode smoke payload' },
      }
    })],
    ['GetItem', new GetItemCommand({ 
      TableName: table, 
      Key: { id: { S: 'test-id' } }
    })],
    ['DeleteItem', new DeleteItemCommand({ 
      TableName: table, 
      Key: { id: { S: 'test-id' } }
    })],
    ['DeleteTable', new DeleteTableCommand({ TableName: table })],
  ];

  for (const [name, command] of commands) {
    console.log(`\nExecuting ${name}...`);
    try {
      const response = await client.send(command);
      console.log(`✓ ${name} succeeded. Response:`);
      console.dir(response, { depth: 4, colors: true });
    } catch (error) {
      console.error(`\nFailed during command: ${name}`);
      if (error.$response) {
        console.error('Response status:', error.$response.statusCode);
        console.error('Response headers:', error.$response.headers);
        if (error.$response.body) {
          const body = error.$response.body;
          if (typeof body.read === 'function') {
            const chunk = body.read();
            if (chunk) {
              console.error('Response body:', chunk.toString());
            } else {
              console.error('Response body:', body);
            }
          } else if (typeof body.toString === 'function') {
            console.error('Response body:', body.toString());
          } else {
            console.error('Response body:', body);
          }
        }
      }
      throw error;
    }
  }

  console.log('\n✓ Native AWS SDK smoke mode passed');
}

function uniqueTableName(prefix) {
  return `mildstack_${prefix}_${Date.now().toString(36)}_${randomUUID().slice(0, 8)}`;
}
