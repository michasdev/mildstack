/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import type { S3BrowserApi } from '../types'

function createElectronBridgeApi(): S3BrowserApi {
  const invoke = window.electron.ipcRenderer.invoke.bind(window.electron.ipcRenderer)

  return {
    listBuckets: (region) => invoke('s3:listBuckets', { region }),
    createBucket: (name, region) => invoke('s3:createBucket', { name, region }),
    deleteBucket: (name, region) => invoke('s3:deleteBucket', { name, region }),
    listObjects: (bucket, prefix, continuationToken, region) =>
      invoke('s3:listObjects', { bucket, prefix, continuationToken, region }),
    putObject: (bucket, key, body, contentType, region) =>
      invoke('s3:putObject', { bucket, key, body, contentType, region }),
    deleteObjects: (bucket, keys, region) => invoke('s3:deleteObjects', { bucket, keys, region }),
    getObject: (bucket, key, region) => invoke('s3:getObject', { bucket, key, region })
  }
}

export function useS3Client(initialRegion: string = 'us-east-1') {
  const [region, setRegion] = useState(initialRegion)
  const api = (window.api?.s3 as S3BrowserApi | undefined) ?? createElectronBridgeApi()

  return { api, region, setRegion }
}
