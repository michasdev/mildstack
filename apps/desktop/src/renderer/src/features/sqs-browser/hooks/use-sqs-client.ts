/* eslint-disable @typescript-eslint/explicit-function-return-type */

import { useState } from 'react'
import type { SQSBrowserApi } from '../types'

function createElectronBridgeApi(): SQSBrowserApi {
  return window.api.sqs
}

export function useSQSClient(initialRegion: string = 'us-east-1') {
  const [region, setRegion] = useState(initialRegion)
  const api = createElectronBridgeApi()

  return { api, region, setRegion }
}
