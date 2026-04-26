/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import { useLocation, useNavigate, Outlet } from 'react-router'
import { ChevronRight, RotateCw } from 'lucide-react'

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
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@renderer/components/ui/select'
import { Tooltip, TooltipTrigger, TooltipContent } from '@renderer/components/ui/tooltip'
import { regions } from '@renderer/constants'

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

  return (
    <Frame className="w-full h-full flex flex-col">
      <FrameHeader className="flex-none">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between w-full">
          <div className="flex items-start gap-3">
            <div className="space-y-2">
              <FrameTitle className="flex items-center gap-2">
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
                {currentPrefix &&
                  currentPrefix
                    .split('/')
                    .filter(Boolean)
                    .map((part, index, all) => {
                      const pathSoFar = all.slice(0, index + 1).join('/')
                      const isLast = index === all.length - 1

                      return (
                        <div key={pathSoFar} className="flex items-center gap-2">
                          <ChevronRight className="w-3 h-3 text-muted-foreground" />
                          {isLast ? (
                            <span className="font-medium text-foreground truncate max-w-[260px]">
                              {part}
                            </span>
                          ) : (
                            <button
                              type="button"
                              className="transition-colors hover:text-foreground"
                              onClick={() => navigate(`/resources/s3/${currentBucket}/${pathSoFar}/`)}
                            >
                              {part}
                            </button>
                          )}
                        </div>
                      )
                    })}
              </FrameDescription>
            </div>
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <span className="text-sm text-muted-foreground">Region</span>
            <div className="flex items-center gap-2">
              <Select value={region} onValueChange={(val) => setRegion(val as string)}>
                <SelectTrigger className="h-8 w-[140px] px-3 shadow-xs/5">
                  <SelectValue placeholder="Select region" />
                </SelectTrigger>
                <SelectContent>
                  {regions.map((value) => (
                    <SelectItem key={value} value={value}>
                      {value}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Tooltip>
                <TooltipTrigger>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => setRefreshKey((prev) => prev + 1)}
                  >
                    <RotateCw className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Refresh</TooltipContent>
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
