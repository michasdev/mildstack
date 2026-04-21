#!/usr/bin/env node
'use strict';

const { randomUUID } = require('node:crypto');
const { setTimeout } = require('node:timers/promises');
const {
  SQSClient,
  ListQueuesCommand,
  CreateQueueCommand,
  GetQueueUrlCommand,
  GetQueueAttributesCommand,
  SetQueueAttributesCommand,
  SendMessageCommand,
  ReceiveMessageCommand,
  ChangeMessageVisibilityCommand,
  DeleteMessageCommand,
  SendMessageBatchCommand,
  ChangeMessageVisibilityBatchCommand,
  DeleteMessageBatchCommand,
  PurgeQueueCommand,
  DeleteQueueCommand,
  ListQueueTagsCommand,
  TagQueueCommand,
  UntagQueueCommand,
  ListDeadLetterSourceQueuesCommand,
} = require('@aws-sdk/client-sqs');

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
  console.error('\nSQS smoke test failed');
  console.error(error instanceof Error ? error.stack || error.message : error);
  process.exitCode = 1;
});

async function main() {
  const endpoint = process.env.MILDSTACK_SQS_ENDPOINT || `http://localhost:${port}`;

  console.log(`Running AWS SDK smoke mode against ${endpoint}`);
  const client = new SQSClient({
    region: process.env.AWS_REGION || 'us-east-1',
    endpoint,
    credentials: {
      accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
      secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
    },
  });

  const stdQueueName = uniqueQueueName('smoke-std');
  const dlqQueueName = uniqueQueueName('smoke-dlq');
  const fifoQueueName = uniqueQueueName('smoke-fifo') + '.fifo';

  // --- Setup Queues ---
  console.log('\n--- Setup Queues ---');
  const dlqUrl = (await execute(client, 'CreateQueue (DLQ)', new CreateQueueCommand({
    QueueName: dlqQueueName,
  }))).QueueUrl;

  const dlqArn = (await execute(client, 'GetQueueAttributes (DLQ)', new GetQueueAttributesCommand({
    QueueUrl: dlqUrl, AttributeNames: ['QueueArn'],
  }))).Attributes?.QueueArn;

  const stdUrl = (await execute(client, 'CreateQueue (Std)', new CreateQueueCommand({
    QueueName: stdQueueName,
    Attributes: { VisibilityTimeout: '2', MessageRetentionPeriod: '345600' }, // short visibility timeout for tests
  }))).QueueUrl;

  const fifoUrl = (await execute(client, 'CreateQueue (FIFO)', new CreateQueueCommand({
    QueueName: fifoQueueName,
    Attributes: { FifoQueue: 'true', ContentBasedDeduplication: 'true', VisibilityTimeout: '10' },
  }))).QueueUrl;

  // Verify GetQueueUrl
  const getUrlOut = await execute(client, 'GetQueueUrl (Std)', new GetQueueUrlCommand({ QueueName: stdQueueName }));
  expectEqual(getUrlOut.QueueUrl, stdUrl, 'GetQueueUrl QueueUrl');

  // Verify ListQueues
  const listQueuesOut = await execute(client, 'ListQueues', new ListQueuesCommand({ QueueNamePrefix: 'mildstack-smoke' }));
  if (!listQueuesOut.QueueUrls || listQueuesOut.QueueUrls.length < 3) {
    throw new Error('ListQueues did not return created queues');
  }

  // --- Validation: Visibility Timeout and DLQ ---
  console.log('\n--- Validation: Visibility Timeout and DLQ ---');
  await execute(client, 'SetQueueAttributes (Std -> DLQ)', new SetQueueAttributesCommand({
    QueueUrl: stdUrl,
    Attributes: {
      RedrivePolicy: JSON.stringify({ deadLetterTargetArn: dlqArn, maxReceiveCount: 2 }),
    },
  }));

  const dlqSources = await execute(client, 'ListDeadLetterSourceQueues', new ListDeadLetterSourceQueuesCommand({ QueueUrl: dlqUrl }));
  if (!dlqSources.queueUrls || !dlqSources.queueUrls.includes(stdUrl)) {
    console.warn('\nWarning: ListDeadLetterSourceQueues did not include the source queue. This might be unsupported by the emulator.');
  }

  await execute(client, 'SendMessage (for DLQ test)', new SendMessageCommand({
    QueueUrl: stdUrl, MessageBody: 'dlq test message',
  }));

  const r1 = await execute(client, 'Receive 1 (DLQ test)', new ReceiveMessageCommand({ QueueUrl: stdUrl, MaxNumberOfMessages: 1 }));
  expectEqual(r1.Messages?.length, 1, 'Should receive message first time');

  const r2 = await execute(client, 'Receive 2 immediately (hidden)', new ReceiveMessageCommand({ QueueUrl: stdUrl, WaitTimeSeconds: 0 }));
  expectEqual(r2.Messages?.length || 0, 0, 'Message should be invisible');

  console.log('\nWaiting for visibility timeout (3s)...');
  await setTimeout(3000);

  const r3 = await execute(client, 'Receive 3 (DLQ test)', new ReceiveMessageCommand({ QueueUrl: stdUrl, MaxNumberOfMessages: 1 }));
  expectEqual(r3.Messages?.length, 1, 'Should receive message second time');

  console.log('\nWaiting for visibility timeout again (3s)...');
  await setTimeout(3000);

  const r4 = await execute(client, 'Receive 4 from Std (Should be empty)', new ReceiveMessageCommand({ QueueUrl: stdUrl, MaxNumberOfMessages: 1, WaitTimeSeconds: 0 }));
  expectEqual(r4.Messages?.length || 0, 0, 'Message should have been moved to DLQ');

  const r5 = await execute(client, 'Receive from DLQ', new ReceiveMessageCommand({ QueueUrl: dlqUrl, MaxNumberOfMessages: 1 }));
  expectEqual(r5.Messages?.length, 1, 'Message should be in DLQ');
  await execute(client, 'Delete from DLQ', new DeleteMessageCommand({ QueueUrl: dlqUrl, ReceiptHandle: r5.Messages[0].ReceiptHandle }));


  // --- Validation: DelaySeconds ---
  console.log('\n--- Validation: DelaySeconds ---');
  await execute(client, 'SetQueueAttributes (DelaySeconds)', new SetQueueAttributesCommand({
    QueueUrl: stdUrl, Attributes: { DelaySeconds: '2' },
  }));

  await execute(client, 'SendMessage (Queue Delay)', new SendMessageCommand({
    QueueUrl: stdUrl, MessageBody: 'queue delay message',
  }));
  
  await execute(client, 'SendMessage (Message Delay Override)', new SendMessageCommand({
    QueueUrl: stdUrl, MessageBody: 'message delay message', DelaySeconds: 4,
  }));

  const rd1 = await execute(client, 'Receive immediately (both hidden)', new ReceiveMessageCommand({ QueueUrl: stdUrl, WaitTimeSeconds: 0 }));
  expectEqual(rd1.Messages?.length || 0, 0, 'Messages should be delayed');

  console.log('\nWaiting for queue delay (3s)...');
  await setTimeout(3000);

  const rd2 = await execute(client, 'Receive after 3s', new ReceiveMessageCommand({ QueueUrl: stdUrl, WaitTimeSeconds: 0 }));
  expectEqual(rd2.Messages?.length, 1, 'Should receive only the queue-delayed message');
  expectEqual(rd2.Messages[0].Body, 'queue delay message', 'Check correct message body');
  await execute(client, 'Delete message', new DeleteMessageCommand({ QueueUrl: stdUrl, ReceiptHandle: rd2.Messages[0].ReceiptHandle }));

  console.log('\nWaiting for message delay (2s more)...');
  await setTimeout(2000);
  const rd3 = await execute(client, 'Receive after 5s total', new ReceiveMessageCommand({ QueueUrl: stdUrl, WaitTimeSeconds: 0 }));
  expectEqual(rd3.Messages?.length, 1, 'Should receive the message-delayed message');
  expectEqual(rd3.Messages[0].Body, 'message delay message', 'Check correct message body');
  await execute(client, 'Delete message', new DeleteMessageCommand({ QueueUrl: stdUrl, ReceiptHandle: rd3.Messages[0].ReceiptHandle }));

  await execute(client, 'SetQueueAttributes (Reset Delay)', new SetQueueAttributesCommand({
    QueueUrl: stdUrl, Attributes: { DelaySeconds: '0' },
  }));


  // --- Validation: Batch Send, Receive, and Visibility ---
  console.log('\n--- Validation: Batch Send and Visibility ---');
  const batchSendOut = await execute(client, 'SendMessageBatch', new SendMessageBatchCommand({
    QueueUrl: stdUrl,
    Entries: [
      { Id: 'm1', MessageBody: 'batch 1' },
      { Id: 'm2', MessageBody: 'batch 2' },
      { Id: 'm3', MessageBody: 'batch 3', DelaySeconds: 3 }, // delayed message
    ],
  }));
  expectEqual(batchSendOut.Successful?.length, 3, 'All 3 messages should be sent successfully');

  const rb1 = await execute(client, 'Receive Batch', new ReceiveMessageCommand({ QueueUrl: stdUrl, MaxNumberOfMessages: 10, WaitTimeSeconds: 0 }));
  expectEqual(rb1.Messages?.length, 2, 'Should receive 2 batch messages immediately');

  const m1 = rb1.Messages.find(m => m.Body === 'batch 1');
  const m2 = rb1.Messages.find(m => m.Body === 'batch 2');

  const visOut = await execute(client, 'ChangeMessageVisibilityBatch', new ChangeMessageVisibilityBatchCommand({
    QueueUrl: stdUrl,
    Entries: [
      { Id: 'v1', ReceiptHandle: m1.ReceiptHandle, VisibilityTimeout: 0 },
    ],
  }));
  expectEqual(visOut.Successful?.length, 1, 'Visibility change should succeed');

  const rb2 = await execute(client, 'Receive after visibility change', new ReceiveMessageCommand({ QueueUrl: stdUrl, WaitTimeSeconds: 0 }));
  expectEqual(rb2.Messages?.length, 1, 'Should receive only m1 since m2 is still invisible');
  expectEqual(rb2.Messages[0].Body, 'batch 1', 'Check body');

  const delOut = await execute(client, 'DeleteMessageBatch', new DeleteMessageBatchCommand({
    QueueUrl: stdUrl,
    Entries: [
      { Id: 'd1', ReceiptHandle: rb2.Messages[0].ReceiptHandle },
      { Id: 'd2', ReceiptHandle: m2.ReceiptHandle },
    ],
  }));
  expectEqual(delOut.Successful?.length, 2, 'Deletion should succeed');

  console.log('\nWaiting for delayed batch message (4s)...');
  await setTimeout(4000);
  const rb3 = await execute(client, 'Receive delayed batch message', new ReceiveMessageCommand({ QueueUrl: stdUrl, WaitTimeSeconds: 0 }));
  expectEqual(rb3.Messages?.length, 1, 'Should receive delayed batch message');
  await execute(client, 'Delete delayed batch message', new DeleteMessageCommand({ QueueUrl: stdUrl, ReceiptHandle: rb3.Messages[0].ReceiptHandle }));


  // --- Validation: FIFO Deduplication and Ordering ---
  console.log('\n--- Validation: FIFO Deduplication ---');
  const f1 = await execute(client, 'SendMessage (FIFO 1)', new SendMessageCommand({
    QueueUrl: fifoUrl, MessageBody: 'dup test', MessageGroupId: 'g1', MessageDeduplicationId: 'dup1',
  }));
  const f2 = await execute(client, 'SendMessage (FIFO 2 - Duplicate)', new SendMessageCommand({
    QueueUrl: fifoUrl, MessageBody: 'dup test', MessageGroupId: 'g1', MessageDeduplicationId: 'dup1',
  }));

  const rf1 = await execute(client, 'Receive FIFO', new ReceiveMessageCommand({ QueueUrl: fifoUrl, MaxNumberOfMessages: 10, WaitTimeSeconds: 0 }));
  expectEqual(rf1.Messages?.length, 1, 'Should only receive 1 message due to deduplication');
  expectEqual(rf1.Messages[0].MessageId, f1.MessageId, 'MessageId should match the first one');
  
  await execute(client, 'Delete FIFO message', new DeleteMessageCommand({
    QueueUrl: fifoUrl, ReceiptHandle: rf1.Messages[0].ReceiptHandle,
  }));


  // --- Validation: Tags ---
  console.log('\n--- Validation: Tags ---');
  await execute(client, 'TagQueue', new TagQueueCommand({
    QueueUrl: stdUrl, Tags: { Env: 'Test', App: 'MildStack' },
  }));
  const tagsOut = await execute(client, 'ListQueueTags', new ListQueueTagsCommand({ QueueUrl: stdUrl }));
  expectEqual(tagsOut.Tags?.Env, 'Test', 'Tag Env should match');
  
  await execute(client, 'UntagQueue', new UntagQueueCommand({
    QueueUrl: stdUrl, TagKeys: ['Env'],
  }));
  const tagsOut2 = await execute(client, 'ListQueueTags (After untag)', new ListQueueTagsCommand({ QueueUrl: stdUrl }));
  expectEqual(tagsOut2.Tags?.Env, undefined, 'Tag Env should be removed');
  expectEqual(tagsOut2.Tags?.App, 'MildStack', 'Tag App should remain');


  // --- Cleanup ---
  console.log('\n--- Cleanup ---');
  await execute(client, 'PurgeQueue', new PurgeQueueCommand({ QueueUrl: stdUrl }));
  await execute(client, 'DeleteQueue (Std)', new DeleteQueueCommand({ QueueUrl: stdUrl }));
  await execute(client, 'DeleteQueue (DLQ)', new DeleteQueueCommand({ QueueUrl: dlqUrl }));
  await execute(client, 'DeleteQueue (FIFO)', new DeleteQueueCommand({ QueueUrl: fifoUrl }));

  console.log('\n✓ Native AWS SDK smoke mode passed with deep behavioral validations');
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
    throw new Error(`Validation Failed [${label}]: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
  }
}

function uniqueQueueName(prefix) {
  return `mildstack-${prefix}-${Date.now().toString(36)}-${randomUUID().slice(0, 8)}`;
}
