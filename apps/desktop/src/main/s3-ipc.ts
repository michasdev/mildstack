import { resolveLocalEndpoint } from './local-endpoint'
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
    const client = getClient(args.region)

    let hasMore = true
    let continuationToken: string | undefined = undefined

    while (hasMore) {
      const listResponse = await client.send(
        new ListObjectsV2Command({
          Bucket: args.name,
          ContinuationToken: continuationToken
        })
      )

      const contents = listResponse.Contents ?? []
      if (contents.length > 0) {
        await client.send(
          new DeleteObjectsCommand({
            Bucket: args.name,
            Delete: {
              Objects: contents.map((obj) => ({ Key: obj.Key! })),
              Quiet: true
            }
          })
        )
      }

      hasMore = listResponse.IsTruncated ?? false
      continuationToken = listResponse.NextContinuationToken
    }

    await client.send(new DeleteBucketCommand({ Bucket: args.name }))
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

    const folders = new Map<string, { Key: string; prefix: string; isFolder: true }>()

    for (const prefix of response.CommonPrefixes ?? []) {
      if (!prefix.Prefix) continue
      folders.set(prefix.Prefix, {
        Key: prefix.Prefix,
        prefix: prefix.Prefix,
        isFolder: true
      })
    }

    const files = (response.Contents ?? []).flatMap((object) => {
      if (!object.Key || object.Key === args.prefix) return []

      if (object.Key.endsWith('/')) {
        if (!folders.has(object.Key)) {
          folders.set(object.Key, {
            Key: object.Key,
            prefix: object.Key,
            isFolder: true
          })
        }
        return []
      }

      return [
        {
          Key: object.Key,
          LastModified: object.LastModified?.toISOString(),
          ETag: object.ETag,
          Size: object.Size,
          StorageClass: object.StorageClass,
          isFolder: false
        }
      ]
    })

    return {
      objects: [...folders.values(), ...files],
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
  return resolveLocalEndpoint('s3')
}

function normalizeRegion(region?: string): string {
  const trimmed = region?.trim()
  return trimmed || 'us-east-1'
}
