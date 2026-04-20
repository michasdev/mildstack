import { getActiveInstancePort } from './instance-state'
import { registerValidatedHandler } from './ipc-middleware'
import {
  S3Client,
  ListBucketsCommand,
  CreateBucketCommand,
  DeleteBucketCommand,
  ListObjectsV2Command,
  PutObjectCommand,
  DeleteObjectsCommand,
  GetObjectCommand,
  type BucketLocationConstraint
} from '@aws-sdk/client-s3'

type ListObjectsArgs = {
  bucket: string
  prefix?: string
  continuationToken?: string
  region?: string
}

type CreateBucketArgs = {
  name: string
  region?: string
}

type PutObjectArgs = {
  bucket: string
  key: string
  body: ArrayBuffer
  contentType?: string
  region?: string
}

type DeleteObjectsArgs = {
  bucket: string
  keys: string[]
  region?: string
}

type GetObjectArgs = {
  bucket: string
  key: string
  region?: string
}

type S3ClientCacheEntry = {
  region: string
  endpoint: string
  client: S3Client
}

const clientCache = new Map<string, S3ClientCacheEntry>()

export function registerS3IpcHandlers(): void {
  registerValidatedHandler('s3:listBuckets', async (_event, args: { region?: string }) => {
    const response = await getClient(args.region).send(new ListBucketsCommand({}))
    return (response.Buckets ?? []).map((bucket) => ({
      Name: bucket.Name,
      CreationDate: bucket.CreationDate?.toISOString()
    }))
  })

  registerValidatedHandler('s3:createBucket', async (_event, args: CreateBucketArgs) => {
    const region = normalizeRegion(args.region)
    const input =
      region === 'us-east-1'
        ? { Bucket: args.name }
        : {
            Bucket: args.name,
            CreateBucketConfiguration: {
              LocationConstraint: region as BucketLocationConstraint
            }
          }

    await getClient(region).send(new CreateBucketCommand(input))
    return null
  })

  registerValidatedHandler('s3:deleteBucket', async (_event, args: { name: string; region?: string }) => {
    await getClient(args.region).send(new DeleteBucketCommand({ Bucket: args.name }))
    return null
  })

  registerValidatedHandler('s3:listObjects', async (_event, args: ListObjectsArgs) => {
    const response = await getClient(args.region).send(
      new ListObjectsV2Command({
        Bucket: args.bucket,
        Prefix: args.prefix?.length ? args.prefix : undefined,
        Delimiter: '/',
        ContinuationToken: args.continuationToken,
        MaxKeys: 50
      })
    )

    const folders = (response.CommonPrefixes ?? []).flatMap((prefix) => {
      if (!prefix.Prefix) return []
      return [
        {
          Key: prefix.Prefix,
          prefix: prefix.Prefix,
          isFolder: true
        }
      ]
    })

    const folderKeys = new Set(folders.map((f) => f.Key))

    const files = (response.Contents ?? [])
      .filter((object) => object.Key && object.Key !== args.prefix && !folderKeys.has(object.Key))
      .map((object) => ({
        Key: object.Key!,
        LastModified: object.LastModified?.toISOString(),
        ETag: object.ETag,
        Size: object.Size,
        StorageClass: object.StorageClass,
        isFolder: false
      }))

    return {
      objects: [...folders, ...files],
      hasMore: Boolean(response.IsTruncated),
      continuationToken: response.NextContinuationToken
    }
  })

  registerValidatedHandler('s3:putObject', async (_event, args: PutObjectArgs) => {
    const body = Buffer.from(args.body)
    await getClient(args.region).send(
      new PutObjectCommand({
        Bucket: args.bucket,
        Key: args.key,
        Body: body,
        ContentType: args.contentType || 'application/octet-stream'
      })
    )
    return null
  })

  registerValidatedHandler('s3:deleteObjects', async (_event, args: DeleteObjectsArgs) => {
    console.log(`[S3-IPC] deleteObjects called for bucket: ${args.bucket}, keys: ${args.keys?.length}`)
    if (args.keys.length === 0) {
      return { Deleted: [], Errors: [] }
    }

    const response = await getClient(args.region).send(
      new DeleteObjectsCommand({
        Bucket: args.bucket,
        Delete: {
          Objects: args.keys.map((key) => ({ Key: key })),
          Quiet: false
        }
      })
    )

    return {
      Deleted: (response.Deleted ?? []).map((d) => ({ Key: d.Key })),
      Errors: (response.Errors ?? []).map((e) => ({
        Key: e.Key,
        Code: e.Code,
        Message: e.Message
      }))
    }
  })

  registerValidatedHandler('s3:getObject', async (_event, args: GetObjectArgs) => {
    const response = await getClient(args.region).send(
      new GetObjectCommand({
        Bucket: args.bucket,
        Key: args.key
      })
    )

    if (!response.Body || typeof response.Body.transformToByteArray !== 'function') {
      throw new Error('s3:getObject: empty object body')
    }

    const bytes = await response.Body.transformToByteArray()
    return {
      contentBase64: Buffer.from(bytes).toString('base64'),
      contentType: response.ContentType
    }
  })
}

function getClient(region = 'us-east-1'): S3Client {
  const normalizedRegion = normalizeRegion(region)
  const endpoint = resolveS3Endpoint()
  const cacheKey = `${normalizedRegion}:${endpoint}`
  const cached = clientCache.get(cacheKey)
  if (cached) {
    return cached.client
  }

  const client = new S3Client({
    region: normalizedRegion,
    endpoint,
    forcePathStyle: true,
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

function resolveS3Endpoint(): string {
  const port = getActiveInstancePort()
  return process.env.MILDSTACK_S3_ENDPOINT || process.env.AWS_S3_ENDPOINT || `http://127.0.0.1:${port}`
}

function normalizeRegion(region?: string): string {
  const trimmed = region?.trim()
  return trimmed || 'us-east-1'
}
