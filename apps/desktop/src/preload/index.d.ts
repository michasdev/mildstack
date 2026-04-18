import { ElectronAPI } from '@electron-toolkit/preload'

interface S3BrowserApi {
  listBuckets(region?: string): Promise<unknown>
  createBucket(name: string, region?: string): Promise<void>
  deleteBucket(name: string, region?: string): Promise<void>
  listObjects(
    bucket: string,
    prefix?: string,
    continuationToken?: string,
    region?: string
  ): Promise<unknown>
  putObject(
    bucket: string,
    key: string,
    body: ArrayBuffer,
    contentType?: string,
    region?: string
  ): Promise<void>
  deleteObjects(bucket: string, keys: string[], region?: string): Promise<void>
  getObject(bucket: string, key: string, region?: string): Promise<unknown>
}

declare global {
  interface Window {
    electron: ElectronAPI
    api: {
      s3: S3BrowserApi
    }
  }
}
