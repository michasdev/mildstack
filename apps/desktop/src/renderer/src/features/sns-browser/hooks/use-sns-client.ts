/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import type { SNSBrowserApi } from '../types'

function createElectronBridgeApi(): SNSBrowserApi {
  return window.api.sns
}

export function useSNSClient(initialRegion: string = 'us-east-1') {
  const [region, setRegion] = useState(initialRegion)
  const api = createElectronBridgeApi()

  return { api, region, setRegion }
}
