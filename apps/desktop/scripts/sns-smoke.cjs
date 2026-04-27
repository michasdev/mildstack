#!/usr/bin/env node
'use strict';

const { randomUUID } = require('node:crypto');
const {
  SNSClient,
  AddPermissionCommand,
  CheckIfPhoneNumberIsOptedOutCommand,
  ConfirmSubscriptionCommand,
  CreatePlatformApplicationCommand,
  CreatePlatformEndpointCommand,
  CreateSMSSandboxPhoneNumberCommand,
  CreateTopicCommand,
  DeleteEndpointCommand,
  DeletePlatformApplicationCommand,
  DeleteSMSSandboxPhoneNumberCommand,
  DeleteTopicCommand,
  GetDataProtectionPolicyCommand,
  GetEndpointAttributesCommand,
  GetPlatformApplicationAttributesCommand,
  GetSMSAttributesCommand,
  GetSMSSandboxAccountStatusCommand,
  GetSubscriptionAttributesCommand,
  GetTopicAttributesCommand,
  ListEndpointsByPlatformApplicationCommand,
  ListOriginationNumbersCommand,
  ListPhoneNumbersOptedOutCommand,
  ListPlatformApplicationsCommand,
  ListSMSSandboxPhoneNumbersCommand,
  ListSubscriptionsByTopicCommand,
  ListSubscriptionsCommand,
  ListTagsForResourceCommand,
  ListTopicsCommand,
  OptInPhoneNumberCommand,
  PublishBatchCommand,
  PublishCommand,
  PutDataProtectionPolicyCommand,
  RemovePermissionCommand,
  SetEndpointAttributesCommand,
  SetPlatformApplicationAttributesCommand,
  SetSMSAttributesCommand,
  SetSubscriptionAttributesCommand,
  SetTopicAttributesCommand,
  SubscribeCommand,
  TagResourceCommand,
  UnsubscribeCommand,
  UntagResourceCommand,
  VerifySMSSandboxPhoneNumberCommand,
} = require('@aws-sdk/client-sns');

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
  console.error('\nSNS smoke test failed');
  console.error(error instanceof Error ? error.stack || error.message : error);
  process.exitCode = 1;
});

async function main() {
  const endpoint = process.env.MILDSTACK_SNS_ENDPOINT || `http://localhost:${port}`;

  console.log(`Running AWS SDK smoke mode against ${endpoint}`);
  const client = new SNSClient({
    region: process.env.AWS_REGION || 'us-east-1',
    endpoint,
    credentials: {
      accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
      secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
    },
  });

  const topicName = uniqueName('smoke-topic');
  const platformName = uniqueName('smoke-platform');
  const sandboxPhone = '+12065550100';
  const targetPhone = '+12065550101';
  const cleanup = {
    topicArn: '',
    subscriptionArn: '',
    endpointArn: '',
    platformApplicationArn: '',
    sandboxPhoneCreated: false,
  };

  try {
    console.log('\n--- Topic and Subscription Lifecycle ---');
    const createTopic = await execute(client, 'CreateTopic', new CreateTopicCommand({
      Name: topicName,
    }));
    cleanup.topicArn = createTopic.TopicArn;
    expectDefined(cleanup.topicArn, 'CreateTopic TopicArn');

    await execute(client, 'SetTopicAttributes', new SetTopicAttributesCommand({
      TopicArn: cleanup.topicArn,
      AttributeName: 'DisplayName',
      AttributeValue: 'MildStack SNS Smoke',
    }));

    const topicAttrs = await execute(client, 'GetTopicAttributes', new GetTopicAttributesCommand({
      TopicArn: cleanup.topicArn,
    }));
    expectEqual(topicAttrs.Attributes?.DisplayName, 'MildStack SNS Smoke', 'Topic DisplayName');

    const topics = await execute(client, 'ListTopics', new ListTopicsCommand({}));
    const topicFound = topics.Topics?.some((item) => item.TopicArn === cleanup.topicArn);
    expectEqual(topicFound, true, 'ListTopics should include created topic');

    const subscribe = await execute(client, 'Subscribe', new SubscribeCommand({
      TopicArn: cleanup.topicArn,
      Protocol: 'http',
      Endpoint: 'http://127.0.0.1:7788/sns',
      ReturnSubscriptionArn: true,
    }));
    cleanup.subscriptionArn = subscribe.SubscriptionArn || '';
    expectDefined(cleanup.subscriptionArn, 'Subscribe SubscriptionArn');

    const subscriptions = await execute(client, 'ListSubscriptions', new ListSubscriptionsCommand({}));
    const subFound = subscriptions.Subscriptions?.some((item) => item.SubscriptionArn === cleanup.subscriptionArn);
    expectEqual(subFound, true, 'ListSubscriptions should include created subscription');

    const subscriptionsByTopic = await execute(client, 'ListSubscriptionsByTopic', new ListSubscriptionsByTopicCommand({
      TopicArn: cleanup.topicArn,
    }));
    const subByTopicFound = subscriptionsByTopic.Subscriptions?.some((item) => item.SubscriptionArn === cleanup.subscriptionArn);
    expectEqual(subByTopicFound, true, 'ListSubscriptionsByTopic should include created subscription');

    await execute(client, 'SetSubscriptionAttributes', new SetSubscriptionAttributesCommand({
      SubscriptionArn: cleanup.subscriptionArn,
      AttributeName: 'RawMessageDelivery',
      AttributeValue: 'true',
    }));

    const subAttrs = await execute(client, 'GetSubscriptionAttributes', new GetSubscriptionAttributesCommand({
      SubscriptionArn: cleanup.subscriptionArn,
    }));
    expectEqual(subAttrs.Attributes?.RawMessageDelivery, 'true', 'Subscription RawMessageDelivery');

    await expectAwsError(
      client,
      'ConfirmSubscription (invalid token)',
      new ConfirmSubscriptionCommand({
        TopicArn: cleanup.topicArn,
        Token: 'invalid-token',
      }),
      'NotFound'
    );

    console.log('\n--- Publish and PublishBatch ---');
    const publishTopic = await execute(client, 'Publish (TopicArn)', new PublishCommand({
      TopicArn: cleanup.topicArn,
      Message: 'sns smoke publish topic',
    }));
    expectDefined(publishTopic.MessageId, 'Publish Topic MessageId');

    const publishPhone = await execute(client, 'Publish (PhoneNumber)', new PublishCommand({
      PhoneNumber: targetPhone,
      Message: 'sns smoke publish phone',
    }));
    expectDefined(publishPhone.MessageId, 'Publish Phone MessageId');

    const batch = await execute(client, 'PublishBatch', new PublishBatchCommand({
      TopicArn: cleanup.topicArn,
      PublishBatchRequestEntries: [
        { Id: 'ok1', Message: 'batch ok' },
        { Id: 'bad1', Message: '' },
      ],
    }));
    expectEqual(batch.Successful?.length, 1, 'PublishBatch successful entries');
    expectEqual(batch.Failed?.length, 1, 'PublishBatch failed entries');
    expectEqual(batch.Failed?.[0]?.Id, 'bad1', 'PublishBatch failed id');

    console.log('\n--- Permissions, Tags and Data Protection Policy ---');
    await execute(client, 'AddPermission', new AddPermissionCommand({
      TopicArn: cleanup.topicArn,
      Label: 'smoke-label',
      AWSAccountId: ['00000000000'],
      ActionName: ['Publish'],
    }));

    await execute(client, 'RemovePermission', new RemovePermissionCommand({
      TopicArn: cleanup.topicArn,
      Label: 'smoke-label',
    }));

    await execute(client, 'TagResource', new TagResourceCommand({
      ResourceArn: cleanup.topicArn,
      Tags: [{ Key: 'env', Value: 'smoke' }, { Key: 'team', Value: 'desktop' }],
    }));

    const tags = await execute(client, 'ListTagsForResource', new ListTagsForResourceCommand({
      ResourceArn: cleanup.topicArn,
    }));
    const envTag = (tags.Tags || []).find((tag) => tag.Key === 'env');
    expectEqual(envTag?.Value, 'smoke', 'TagResource/ListTagsForResource');

    await execute(client, 'UntagResource', new UntagResourceCommand({
      ResourceArn: cleanup.topicArn,
      TagKeys: ['env'],
    }));

    const tagsAfterUntag = await execute(client, 'ListTagsForResource (after untag)', new ListTagsForResourceCommand({
      ResourceArn: cleanup.topicArn,
    }));
    const envTagAfter = (tagsAfterUntag.Tags || []).find((tag) => tag.Key === 'env');
    expectEqual(envTagAfter, undefined, 'UntagResource should remove env tag');

    const policyDocument = JSON.stringify({
      Name: 'smoke-policy',
      Statement: [{ Sid: 'AllowAll', Effect: 'Allow', Principal: '*', Action: 'sns:Publish', Resource: cleanup.topicArn }],
    });
    await execute(client, 'PutDataProtectionPolicy', new PutDataProtectionPolicyCommand({
      ResourceArn: cleanup.topicArn,
      DataProtectionPolicy: policyDocument,
    }));

    const policy = await execute(client, 'GetDataProtectionPolicy', new GetDataProtectionPolicyCommand({
      ResourceArn: cleanup.topicArn,
    }));
    expectEqual(policy.DataProtectionPolicy, policyDocument, 'DataProtectionPolicy roundtrip');

    console.log('\n--- Platform Application and Endpoint ---');
    const app = await execute(client, 'CreatePlatformApplication', new CreatePlatformApplicationCommand({
      Name: platformName,
      Platform: 'GCM',
      Attributes: { PlatformCredential: 'dummy-credential' },
    }));
    cleanup.platformApplicationArn = app.PlatformApplicationArn || '';
    expectDefined(cleanup.platformApplicationArn, 'CreatePlatformApplication Arn');

    const listApps = await execute(client, 'ListPlatformApplications', new ListPlatformApplicationsCommand({}));
    const appFound = listApps.PlatformApplications?.some((item) => item.PlatformApplicationArn === cleanup.platformApplicationArn);
    expectEqual(appFound, true, 'ListPlatformApplications should include created app');

    const appAttrs = await execute(client, 'GetPlatformApplicationAttributes', new GetPlatformApplicationAttributesCommand({
      PlatformApplicationArn: cleanup.platformApplicationArn,
    }));
    expectEqual(appAttrs.Attributes?.PlatformCredential, 'dummy-credential', 'PlatformCredential should match');

    await execute(client, 'SetPlatformApplicationAttributes', new SetPlatformApplicationAttributesCommand({
      PlatformApplicationArn: cleanup.platformApplicationArn,
      Attributes: { EventEndpointCreated: 'arn:aws:sns:us-east-1:00000000000:event-topic' },
    }));

    const endpoint = await execute(client, 'CreatePlatformEndpoint', new CreatePlatformEndpointCommand({
      PlatformApplicationArn: cleanup.platformApplicationArn,
      Token: randomUUID(),
      CustomUserData: 'smoke-user',
      Attributes: { Enabled: 'true' },
    }));
    cleanup.endpointArn = endpoint.EndpointArn || '';
    expectDefined(cleanup.endpointArn, 'CreatePlatformEndpoint Arn');

    const endpointList = await execute(client, 'ListEndpointsByPlatformApplication', new ListEndpointsByPlatformApplicationCommand({
      PlatformApplicationArn: cleanup.platformApplicationArn,
    }));
    const endpointFound = endpointList.Endpoints?.some((item) => item.EndpointArn === cleanup.endpointArn);
    expectEqual(endpointFound, true, 'ListEndpointsByPlatformApplication should include endpoint');

    const endpointAttrs = await execute(client, 'GetEndpointAttributes', new GetEndpointAttributesCommand({
      EndpointArn: cleanup.endpointArn,
    }));
    expectEqual(endpointAttrs.Attributes?.Enabled, 'true', 'Endpoint Enabled=true');

    await execute(client, 'SetEndpointAttributes', new SetEndpointAttributesCommand({
      EndpointArn: cleanup.endpointArn,
      Attributes: { Enabled: 'false', CustomUserData: 'smoke-user-updated' },
    }));

    const publishTarget = await execute(client, 'Publish (TargetArn)', new PublishCommand({
      TargetArn: cleanup.endpointArn,
      Message: 'sns smoke publish endpoint',
    }));
    expectDefined(publishTarget.MessageId, 'Publish TargetArn MessageId');

    console.log('\n--- SMS Attributes and Sandbox ---');
    await execute(client, 'SetSMSAttributes', new SetSMSAttributesCommand({
      attributes: { DefaultSMSType: 'Transactional', MonthlySpendLimit: '1' },
    }));

    const smsAttrs = await execute(client, 'GetSMSAttributes', new GetSMSAttributesCommand({
      attributes: ['DefaultSMSType', 'MonthlySpendLimit'],
    }));
    expectEqual(smsAttrs.attributes?.DefaultSMSType, 'Transactional', 'GetSMSAttributes DefaultSMSType');

    const optedOutBefore = await execute(client, 'CheckIfPhoneNumberIsOptedOut (before)', new CheckIfPhoneNumberIsOptedOutCommand({
      phoneNumber: sandboxPhone,
    }));
    expectEqual(optedOutBefore.isOptedOut, false, 'Phone should not be opted out before OptIn');

    await execute(client, 'OptInPhoneNumber', new OptInPhoneNumberCommand({
      phoneNumber: sandboxPhone,
    }));

    const optedOutList = await execute(client, 'ListPhoneNumbersOptedOut', new ListPhoneNumbersOptedOutCommand({}));
    expectEqual(Array.isArray(optedOutList.phoneNumbers), true, 'ListPhoneNumbersOptedOut returns list');

    const sandboxStatusBefore = await execute(client, 'GetSMSSandboxAccountStatus (before verify)', new GetSMSSandboxAccountStatusCommand({}));
    expectEqual(sandboxStatusBefore.isInSandbox, true, 'Account should start in sandbox');

    await execute(client, 'CreateSMSSandboxPhoneNumber', new CreateSMSSandboxPhoneNumberCommand({
      PhoneNumber: sandboxPhone,
      LanguageCode: 'en-US',
    }));
    cleanup.sandboxPhoneCreated = true;

    const sandboxList = await execute(client, 'ListSMSSandboxPhoneNumbers', new ListSMSSandboxPhoneNumbersCommand({}));
    const pendingPhoneFound = sandboxList.PhoneNumbers?.some((item) => item.PhoneNumber === sandboxPhone);
    expectEqual(pendingPhoneFound, true, 'ListSMSSandboxPhoneNumbers should include created phone');

    await execute(client, 'VerifySMSSandboxPhoneNumber', new VerifySMSSandboxPhoneNumberCommand({
      PhoneNumber: sandboxPhone,
      OneTimePassword: '123456',
    }));

    const sandboxStatusAfter = await execute(client, 'GetSMSSandboxAccountStatus (after verify)', new GetSMSSandboxAccountStatusCommand({}));
    expectEqual(sandboxStatusAfter.isInSandbox, false, 'Account should leave sandbox after verification');

    const origination = await execute(client, 'ListOriginationNumbers', new ListOriginationNumbersCommand({}));
    const originFound = origination.phoneNumbers?.includes(sandboxPhone);
    expectEqual(originFound, true, 'ListOriginationNumbers should include verified sandbox phone');

    console.log('\n--- Cleanup Operations ---');
    await execute(client, 'DeleteEndpoint', new DeleteEndpointCommand({
      EndpointArn: cleanup.endpointArn,
    }));
    cleanup.endpointArn = '';

    await execute(client, 'DeletePlatformApplication', new DeletePlatformApplicationCommand({
      PlatformApplicationArn: cleanup.platformApplicationArn,
    }));
    cleanup.platformApplicationArn = '';

    await execute(client, 'DeleteSMSSandboxPhoneNumber', new DeleteSMSSandboxPhoneNumberCommand({
      PhoneNumber: sandboxPhone,
    }));
    cleanup.sandboxPhoneCreated = false;

    await execute(client, 'Unsubscribe', new UnsubscribeCommand({
      SubscriptionArn: cleanup.subscriptionArn,
    }));
    cleanup.subscriptionArn = '';

    await execute(client, 'DeleteTopic', new DeleteTopicCommand({
      TopicArn: cleanup.topicArn,
    }));
    cleanup.topicArn = '';

    console.log('\n✓ Native AWS SDK SNS smoke mode passed with broad operation coverage');
  } finally {
    await bestEffortCleanup(client, cleanup);
  }
}

async function bestEffortCleanup(client, cleanup) {
  if (!cleanup || !client) {
    return;
  }

  const tasks = [];
  if (cleanup.endpointArn) {
    tasks.push(execute(client, 'Cleanup DeleteEndpoint', new DeleteEndpointCommand({
      EndpointArn: cleanup.endpointArn,
    })).catch(() => undefined));
  }
  if (cleanup.platformApplicationArn) {
    tasks.push(execute(client, 'Cleanup DeletePlatformApplication', new DeletePlatformApplicationCommand({
      PlatformApplicationArn: cleanup.platformApplicationArn,
    })).catch(() => undefined));
  }
  if (cleanup.sandboxPhoneCreated) {
    tasks.push(execute(client, 'Cleanup DeleteSMSSandboxPhoneNumber', new DeleteSMSSandboxPhoneNumberCommand({
      PhoneNumber: '+12065550100',
    })).catch(() => undefined));
  }
  if (cleanup.subscriptionArn) {
    tasks.push(execute(client, 'Cleanup Unsubscribe', new UnsubscribeCommand({
      SubscriptionArn: cleanup.subscriptionArn,
    })).catch(() => undefined));
  }
  if (cleanup.topicArn) {
    tasks.push(execute(client, 'Cleanup DeleteTopic', new DeleteTopicCommand({
      TopicArn: cleanup.topicArn,
    })).catch(() => undefined));
  }

  if (tasks.length > 0) {
    await Promise.all(tasks);
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
    if (!debug) console.log('');
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

function expectDefined(actual, label) {
  if (actual === undefined || actual === null || actual === '') {
    throw new Error(`Validation Failed [${label}]: expected a defined value, got ${JSON.stringify(actual)}`);
  }
}

function uniqueName(prefix) {
  return `mildstack-${prefix}-${Date.now().toString(36)}-${randomUUID().slice(0, 8)}`;
}
