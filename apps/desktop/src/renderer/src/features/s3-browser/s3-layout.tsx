/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import { useLocation, useNavigate, Outlet } from 'react-router'
import { ArrowLeft, ChevronRight, Database, RotateCw } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import {
  Frame,
  FrameDescription,
  FrameHeader,
  FramePanel,
  FrameTitle
} from '@renderer/components/ui/frame'
import { useS3Client } from './hooks/use-s3-client'
import type { S3BrowserApi } from './types'
import { Select, SelectTrigger, SelectPopup, SelectItem, SelectValue } from '@renderer/components/ui/select'
import { Tooltip, TooltipTrigger, TooltipPopup } from '@renderer/components/ui/tooltip'

export function S3Layout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { api, region, setRegion } = useS3Client()
  const [refreshKey, setRefreshKey] = useState(0)

  const pathParts = location.pathname.split('/').filter(Boolean)
  const isS3Root = pathParts[pathParts.length - 1] === 's3'

  let currentBucket = ''
  let currentPrefix = ''

  if (!isS3Root) {
    const s3Index = pathParts.indexOf('s3')
    if (s3Index !== -1 && pathParts.length > s3Index + 1) {
      currentBucket = pathParts[s3Index + 1]
      currentPrefix = pathParts.slice(s3Index + 2).join('/')
    }
  }

  const navigateToS3Root = () => navigate('/resources/s3')
  const navigateToBucket = () => navigate(`/resources/s3/${currentBucket}`)

  const regions = [
    'us-east-1',
    'us-east-2',
    'us-west-1',
    'us-west-2',
    'ca-central-1',
    'ca-west-1',
    'eu-central-1',
    'eu-central-2',
    'eu-west-1',
    'eu-west-2',
    'eu-west-3',
    'eu-north-1',
    'eu-south-1',
    'eu-south-2',
    'ap-northeast-1',
    'ap-northeast-2',
    'ap-northeast-3',
    'ap-southeast-1',
    'ap-southeast-2',
    'ap-southeast-3',
    'ap-southeast-4',
    'ap-southeast-5',
    'ap-southeast-7',
    'ap-south-1',
    'ap-south-2',
    'ap-east-1',
    'ap-east-2',
    'me-south-1',
    'me-central-1',
    'af-south-1',
    'sa-east-1',
    'cn-north-1',
    'cn-northwest-1',
    'us-gov-east-1',
    'us-gov-west-1'
  ]

  return (
    <Frame className="w-full h-full flex flex-col">
      <FrameHeader className="flex-none">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between w-full">
          <div className="flex items-start gap-3">
            <Button variant="ghost" size="icon" onClick={() => navigate('/resources')}>
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <div className="space-y-2">
              <FrameTitle className="flex items-center gap-2">
                <Database className="w-5 h-5 text-primary" />
                S3 Resource Browser
              </FrameTitle>
              <FrameDescription className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  className={`transition-colors hover:text-foreground ${isS3Root ? 'font-medium text-foreground' : ''
                    }`}
                  onClick={navigateToS3Root}
                >
                  Buckets
                </button>
                {currentBucket && (
                  <>
                    <ChevronRight className="w-3 h-3 text-muted-foreground" />
                    <button
                      type="button"
                      className={`transition-colors hover:text-foreground ${!currentPrefix ? 'font-medium text-foreground' : ''
                        }`}
                      onClick={navigateToBucket}
                    >
                      {currentBucket}
                    </button>
                  </>
                )}
                {currentPrefix && (
                  <>
                    <ChevronRight className="w-3 h-3 text-muted-foreground" />
                    <span className="font-medium text-foreground truncate max-w-[260px]">
                      {currentPrefix}
                    </span>
                  </>
                )}
              </FrameDescription>
            </div>
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <span className="text-sm text-muted-foreground">Region</span>
            <div className="flex items-center gap-2">
              <Select value={region} onValueChange={(val) => setRegion(val as string)}>
                <SelectTrigger className="h-9 w-[140px] px-3 shadow-xs/5">
                  <SelectValue placeholder="Select region" />
                </SelectTrigger>
                <SelectPopup>
                  {regions.map((value) => (
                    <SelectItem key={value} value={value}>
                      {value}
                    </SelectItem>
                  ))}
                </SelectPopup>
              </Select>

              <Tooltip delayDuration={1500}>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => setRefreshKey((prev) => prev + 1)}
                    />
                  }
                >
                  <RotateCw className="h-4 w-4" />
                </TooltipTrigger>
                <TooltipPopup>Refresh</TooltipPopup>
              </Tooltip>
            </div>
          </div>
        </div>
      </FrameHeader>

      <FramePanel className="flex-1 overflow-hidden border-none bg-transparent shadow-none p-0 flex flex-col">
        <Outlet key={refreshKey} context={{ api, region }} />
      </FramePanel>
    </Frame>
  )
}

export type S3BrowserOutletContext = {
  api: S3BrowserApi
  region: string
}
