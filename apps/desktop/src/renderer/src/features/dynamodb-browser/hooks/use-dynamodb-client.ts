/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import type { DynamoDBBrowserApi } from '../types'

function createElectronBridgeApi(): DynamoDBBrowserApi {
  return window.api.dynamodb
}

export function useDynamoDBClient(initialRegion: string = 'us-east-1') {
  const [region, setRegion] = useState(initialRegion)
  const api = createElectronBridgeApi()

  return { api, region, setRegion }
}
