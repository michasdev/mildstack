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
  ListObjectsV2Command,
  CreateMultipartUploadCommand,
  UploadPartCommand,
  CompleteMultipartUploadCommand,
  CopyObjectCommand,
  HeadObjectCommand,
  DeleteObjectsCommand,
} = require('@aws-sdk/client-s3');
const { getSignedUrl } = require('@aws-sdk/s3-request-presigner');

// Parse arguments to find port
const args = process.argv.slice(2);
let port = 4566;
for (let i = 0; i < args.length; i++) {
  if (args[i] === '--port' && args[i + 1]) {
    port = parseInt(args[i + 1], 10);
  }
}

function expectEqual(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(`Assertion failed: ${message}\nExpected: ${expected}\nActual: ${actual}`);
  }
}

function expectDefined(actual, message) {
  if (actual === undefined || actual === null) {
    throw new Error(`Assertion failed: ${message}\nValue was undefined or null.`);
  }
}

async function main() {
  const endpoint = process.env.MILDSTACK_S3_ENDPOINT || process.env.AWS_S3_ENDPOINT || `http://localhost:${port}`;
  
  console.log(`Running S3 behavioral validation against ${endpoint}`);
  const client = new S3Client({
    region: process.env.AWS_REGION || 'us-east-1',
    endpoint,
    forcePathStyle: true,
    credentials: {
      accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
      secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
    },
  });

  const bucket = uniqueBucketName('behavioral');
  const bucketsToCleanup = [];

  try {
    console.log('\n--- 1. Bucket Operations ---');
    console.log(`Creating bucket ${bucket}...`);
    await client.send(new CreateBucketCommand({ Bucket: bucket }));
    bucketsToCleanup.push(bucket);

    console.log('Validating bucket creation via HeadBucket...');
    await client.send(new HeadBucketCommand({ Bucket: bucket }));
    
    console.log('Validating bucket appears in ListBuckets...');
    const listBucketsRes = await client.send(new ListBucketsCommand({}));
    const bucketExists = listBucketsRes.Buckets?.some(b => b.Name === bucket);
    expectEqual(bucketExists, true, 'Created bucket should appear in ListBuckets');

    console.log('\n--- 2. Basic Object Operations ---');
    const objectKey = 'test-folder/hello.txt';
    const objectBody = 'Hello, MildStack S3!';
    const metadata = { 'custom-author': 'bot', 'status': 'draft' };
    
    console.log(`Putting object ${objectKey}...`);
    await client.send(new PutObjectCommand({
      Bucket: bucket,
      Key: objectKey,
      Body: objectBody,
      ContentType: 'text/plain',
      Metadata: metadata,
    }));

    console.log(`Getting object ${objectKey}...`);
    const getObjRes = await client.send(new GetObjectCommand({
      Bucket: bucket,
      Key: objectKey,
    }));
    const bodyText = await getObjRes.Body.transformToString();
    
    expectEqual(bodyText, objectBody, 'GetObject body should match PutObject body');
    expectEqual(getObjRes.ContentType, 'text/plain', 'GetObject ContentType should match');
    expectEqual(getObjRes.Metadata?.['custom-author'], 'bot', 'GetObject Metadata custom-author should match');
    expectEqual(getObjRes.Metadata?.status, 'draft', 'GetObject Metadata status should match');

    console.log('Validating HeadObject...');
    const headObjRes = await client.send(new HeadObjectCommand({
      Bucket: bucket,
      Key: objectKey,
    }));
    expectEqual(headObjRes.ContentType, 'text/plain', 'HeadObject ContentType should match');
    expectEqual(headObjRes.Metadata?.['custom-author'], 'bot', 'HeadObject Metadata should match');

    console.log('\n--- 3. Pagination and Listing (ListObjectsV2) ---');
    console.log('Creating multiple objects for listing...');
    for (let i = 1; i <= 5; i++) {
      await client.send(new PutObjectCommand({
        Bucket: bucket,
        Key: `listing/item-${i}.txt`,
        Body: `Content ${i}`,
      }));
    }
    
    let listRes = await client.send(new ListObjectsV2Command({
      Bucket: bucket,
      Prefix: 'listing/',
      MaxKeys: 3,
    }));
    
    expectEqual(listRes.Contents?.length, 3, 'ListObjectsV2 should return exactly MaxKeys items');
    expectEqual(listRes.IsTruncated, true, 'ListObjectsV2 should be truncated with more items remaining');
    expectDefined(listRes.NextContinuationToken, 'NextContinuationToken should be defined');
    
    const secondListRes = await client.send(new ListObjectsV2Command({
      Bucket: bucket,
      Prefix: 'listing/',
      MaxKeys: 3,
      ContinuationToken: listRes.NextContinuationToken,
    }));
    expectEqual(secondListRes.Contents?.length, 2, 'Second ListObjectsV2 should return remaining 2 items');
    expectEqual(secondListRes.IsTruncated, false, 'Second ListObjectsV2 should not be truncated');

    console.log('\n--- 4. Copy Object ---');
    const sourceKey = 'listing/item-1.txt';
    const destKey = 'copied/item-1-copy.txt';
    console.log(`Copying ${sourceKey} to ${destKey}...`);
    
    await client.send(new CopyObjectCommand({
      Bucket: bucket,
      CopySource: `${bucket}/${sourceKey}`,
      Key: destKey,
    }));
    
    const copiedObj = await client.send(new GetObjectCommand({
      Bucket: bucket,
      Key: destKey,
    }));
    const copiedBody = await copiedObj.Body.transformToString();
    expectEqual(copiedBody, 'Content 1', 'Copied object content should match original');

    console.log('\n--- 5. Multipart Upload ---');
    const mpKey = 'multipart/large-file.bin';
    console.log(`Creating Multipart Upload for ${mpKey}...`);
    const createMpRes = await client.send(new CreateMultipartUploadCommand({
      Bucket: bucket,
      Key: mpKey,
      Metadata: { 'mp-meta': 'yes' }
    }));
    const uploadId = createMpRes.UploadId;
    expectDefined(uploadId, 'UploadId must be returned');

    console.log(`Uploading parts...`);
    const part1Body = 'Part 1 data. '.repeat(100);
    const part2Body = 'Part 2 data. '.repeat(100);
    
    const p1Res = await client.send(new UploadPartCommand({
      Bucket: bucket,
      Key: mpKey,
      UploadId: uploadId,
      PartNumber: 1,
      Body: part1Body,
    }));
    
    const p2Res = await client.send(new UploadPartCommand({
      Bucket: bucket,
      Key: mpKey,
      UploadId: uploadId,
      PartNumber: 2,
      Body: part2Body,
    }));

    console.log(`Completing Multipart Upload...`);
    await client.send(new CompleteMultipartUploadCommand({
      Bucket: bucket,
      Key: mpKey,
      UploadId: uploadId,
      MultipartUpload: {
        Parts: [
          { PartNumber: 1, ETag: p1Res.ETag },
          { PartNumber: 2, ETag: p2Res.ETag },
        ],
      },
    }));

    const mpGetRes = await client.send(new GetObjectCommand({
      Bucket: bucket,
      Key: mpKey,
    }));
    const mpText = await mpGetRes.Body.transformToString();
    expectEqual(mpText, part1Body + part2Body, 'Multipart concatenated body should match parts');
    expectEqual(mpGetRes.Metadata?.['mp-meta'], 'yes', 'Multipart Metadata should be preserved');

    console.log('\n--- 6. Presigned URLs ---');
    const presignedKey = 'presigned/upload.txt';
    console.log('Generating presigned PUT URL...');
    const putCommand = new PutObjectCommand({ Bucket: bucket, Key: presignedKey });
    const putUrl = await getSignedUrl(client, putCommand, { expiresIn: 60 });
    expectDefined(putUrl, 'Presigned PUT URL should be generated');
    expectEqual(putUrl.includes(presignedKey), true, 'Presigned URL should contain the key');

    console.log('Using presigned PUT URL via native fetch...');
    const presignedPutBody = 'Hello via presigned url!';
    const putResponse = await fetch(putUrl, {
      method: 'PUT',
      body: presignedPutBody,
    });
    expectEqual(putResponse.ok, true, 'Fetch via presigned PUT should succeed');

    console.log('Generating presigned GET URL...');
    const getCommand = new GetObjectCommand({ Bucket: bucket, Key: presignedKey });
    const getUrl = await getSignedUrl(client, getCommand, { expiresIn: 60 });
    
    console.log('Using presigned GET URL via native fetch...');
    const getResponse = await fetch(getUrl);
    expectEqual(getResponse.ok, true, 'Fetch via presigned GET should succeed');
    const presignedGetBody = await getResponse.text();
    expectEqual(presignedGetBody, presignedPutBody, 'Content fetched via presigned URL should match uploaded content');

    console.log('\n--- 7. Deletion and Not Found Behaviors ---');
    console.log(`Deleting object ${objectKey}...`);
    await client.send(new DeleteObjectCommand({ Bucket: bucket, Key: objectKey }));
    
    try {
      await client.send(new GetObjectCommand({ Bucket: bucket, Key: objectKey }));
      throw new Error('GetObject on deleted object should have thrown an error');
    } catch (err) {
      expectEqual(err.name === 'NoSuchKey' || err.name === 'NotFound', true, 'Error name should be NoSuchKey or NotFound');
    }

    console.log('\nAll behavioral validations passed successfully! 🚀');

  } finally {
    console.log('\n--- 8. Cleanup ---');
    for (const b of bucketsToCleanup) {
      console.log(`Cleaning up bucket: ${b}`);
      try {
        let hasMore = true;
        let token = undefined;
        let objectsToDelete = [];
        
        while (hasMore) {
          const listParams = { Bucket: b, ContinuationToken: token };
          const listedObjects = await client.send(new ListObjectsV2Command(listParams));
          if (listedObjects.Contents && listedObjects.Contents.length > 0) {
             objectsToDelete = objectsToDelete.concat(listedObjects.Contents.map(obj => ({ Key: obj.Key })));
          }
          token = listedObjects.NextContinuationToken;
          hasMore = !!token;
        }

        if (objectsToDelete.length > 0) {
          console.log(`Deleting ${objectsToDelete.length} objects from ${b}...`);
          for (let i = 0; i < objectsToDelete.length; i += 1000) {
            const chunk = objectsToDelete.slice(i, i + 1000);
            await client.send(new DeleteObjectsCommand({
              Bucket: b,
              Delete: { Objects: chunk, Quiet: true },
            }));
          }
        }

        console.log(`Deleting bucket ${b}...`);
        await client.send(new DeleteBucketCommand({ Bucket: b }));
        console.log(`✓ Cleanup complete for ${b}.`);
      } catch (err) {
        console.error(`Failed to cleanup bucket ${b}:`, err.message);
      }
    }
  }
}

function uniqueBucketName(prefix) {
  return `mildstack-${prefix}-${Date.now().toString(36)}-${randomUUID().slice(0, 8)}`.toLowerCase();
}

main().catch((error) => {
  console.error('\n❌ S3 smoke test failed');
  console.error(error instanceof Error ? error.stack || error.message : error);
  process.exitCode = 1;
});
