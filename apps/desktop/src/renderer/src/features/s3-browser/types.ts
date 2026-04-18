export interface S3Bucket {
  Name?: string
  CreationDate?: string
}

export interface S3Object {
  Key?: string
  LastModified?: string
  ETag?: string
  Size?: number
  StorageClass?: string
  isFolder?: boolean
  prefix?: string
}

export interface ListObjectsResult {
  objects: S3Object[]
  hasMore: boolean
  continuationToken?: string
}

export interface S3ObjectPayload {
  contentBase64: string
  contentType?: string
}

export interface S3BrowserApi {
  listBuckets(region?: string): Promise<S3Bucket[]>
  createBucket(name: string, region?: string): Promise<void>
  deleteBucket(name: string, region?: string): Promise<void>
  listObjects(
    bucket: string,
    prefix?: string,
    continuationToken?: string,
    region?: string
  ): Promise<ListObjectsResult>
  putObject(
    bucket: string,
    key: string,
    body: ArrayBuffer,
    contentType?: string,
    region?: string
  ): Promise<void>
  deleteObjects(
    bucket: string,
    keys: string[],
    region?: string
  ): Promise<{ Deleted: { Key: string }[]; Errors: { Key: string; Code: string; Message: string }[] }>
  getObject(bucket: string, key: string, region?: string): Promise<S3ObjectPayload>
}
