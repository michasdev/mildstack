#!/usr/bin/env node
'use strict';

const { randomUUID } = require('node:crypto');
const {
  S3Client,
  ListBucketsCommand,
  CreateBucketCommand,
  HeadBucketCommand,
  DeleteBucketCommand,
  PutObjectCommand,
  GetObjectCommand,
  DeleteObjectCommand,
} = require('@aws-sdk/client-s3');

// Parse arguments to find port
const args = process.argv.slice(2);
let port = 4566;
for (let i = 0; i < args.length; i++) {
  if (args[i] === '--port' && args[i + 1]) {
    port = parseInt(args[i + 1], 10);
  }
}

main().catch((error) => {
  console.error('\nS3 smoke test failed');
  console.error(error instanceof Error ? error.stack || error.message : error);
  process.exitCode = 1;
});

async function main() {
  const endpoint = process.env.MILDSTACK_S3_ENDPOINT || process.env.AWS_S3_ENDPOINT || `http://localhost:${port}`;

  console.log(`Running AWS SDK smoke mode against ${endpoint}`);
  const client = new S3Client({
    region: process.env.AWS_REGION || 'us-east-1',
    endpoint,
    forcePathStyle: true,
    credentials: {
      accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
      secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
    },
  });

  const bucket = uniqueBucketName('native');
  const commands = [
    ['ListBuckets', new ListBucketsCommand({})],
    ['CreateBucket', new CreateBucketCommand({ Bucket: bucket })],
    ['HeadBucket', new HeadBucketCommand({ Bucket: bucket })],
    ['PutObject', new PutObjectCommand({
      Bucket: bucket,
      Key: 'native.txt',
      Body: 'native-mode smoke payload',
      ContentType: 'text/plain',
    })],
    ['GetObject', new GetObjectCommand({ Bucket: bucket, Key: 'native.txt' })],
    ['DeleteObject', new DeleteObjectCommand({ Bucket: bucket, Key: 'native.txt' })],
    ['DeleteBucket', new DeleteBucketCommand({ Bucket: bucket })],
  ];

  for (const [name, command] of commands) {
    console.log(`\nExecuting ${name}...`);
    try {
      const response = await client.send(command);
      console.log(`✓ ${name} succeeded. Response:`);
      
      // Attempt to read stream body if it exists for GetObject to show content
      if (response.Body && typeof response.Body.transformToString === 'function') {
        const bodyText = await response.Body.transformToString();
        const clone = { ...response, Body: bodyText };
        console.dir(clone, { depth: 4, colors: true });
      } else {
        console.dir(response, { depth: 4, colors: true });
      }
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

function uniqueBucketName(prefix) {
  return `mildstack-${prefix}-${Date.now().toString(36)}-${randomUUID().slice(0, 8)}`.toLowerCase();
}
