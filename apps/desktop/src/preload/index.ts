import { contextBridge } from 'electron'
import { ipcRenderer } from 'electron'
import { electronAPI } from '@electron-toolkit/preload'

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

// Custom APIs for renderer
const api: { s3: S3BrowserApi } = {
  s3: {
    listBuckets: (region) => ipcRenderer.invoke('s3:listBuckets', { region }),
    createBucket: (name, region) => ipcRenderer.invoke('s3:createBucket', { name, region }),
    deleteBucket: (name, region) => ipcRenderer.invoke('s3:deleteBucket', { name, region }),
    listObjects: (bucket, prefix, continuationToken, region) =>
      ipcRenderer.invoke('s3:listObjects', { bucket, prefix, continuationToken, region }),
    putObject: (bucket, key, body, contentType, region) =>
      ipcRenderer.invoke('s3:putObject', { bucket, key, body, contentType, region }),
    deleteObjects: (bucket, keys, region) =>
      ipcRenderer.invoke('s3:deleteObjects', { bucket, keys, region }),
    getObject: (bucket, key, region) => ipcRenderer.invoke('s3:getObject', { bucket, key, region })
  }
}

// Use `contextBridge` APIs to expose Electron APIs to
// renderer only if context isolation is enabled, otherwise
// just add to the DOM global.
if (process.contextIsolated) {
  try {
    contextBridge.exposeInMainWorld('electron', electronAPI)
    contextBridge.exposeInMainWorld('api', api)
  } catch (error) {
    console.error(error)
  }
} else {
  // @ts-ignore (define in dts)
  window.electron = electronAPI
  // @ts-ignore (define in dts)
  window.api = api
}
