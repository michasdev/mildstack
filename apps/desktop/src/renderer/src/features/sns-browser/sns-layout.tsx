/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import { useLocation, useNavigate, Outlet } from 'react-router'
import { Bell, ChevronRight, RotateCw } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import {
  Frame,
  FrameDescription,
  FrameHeader,
  FramePanel,
  FrameTitle
} from '@renderer/components/ui/frame'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@renderer/components/ui/select'
import { Tooltip, TooltipTrigger, TooltipContent } from '@renderer/components/ui/tooltip'
import { regions } from '@renderer/constants'
import { useSNSClient } from './hooks/use-sns-client'
import type { SNSBrowserApi } from './types'

export function SNSLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { api, region, setRegion } = useSNSClient()
  const [refreshKey, setRefreshKey] = useState(0)

  const section = new URLSearchParams(location.search).get('section') ?? 'topics'
  const pathParts = location.pathname.split('/').filter(Boolean)
  const snsIndex = pathParts.indexOf('sns')
  const currentTopicName = snsIndex !== -1 && pathParts.length > snsIndex + 1
    ? decodeURIComponent(pathParts[snsIndex + 1])
    : ''

  const goToRoot = () => navigate('/resources/sns')
  const goToSection = (nextSection: string) => navigate(`/resources/sns?section=${nextSection}`)

  return (
    <Frame className="flex h-full w-full flex-col">
      <FrameHeader className="flex-none">
        <div className="flex w-full flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="space-y-2">
              <FrameTitle className="flex items-center gap-2">
                <Bell className="h-4 w-4" />
                SNS Resource Browser
              </FrameTitle>
              <FrameDescription className="flex flex-wrap items-center gap-2">
                {currentTopicName ? (
                  <>
                    <button
                      type="button"
                      className="transition-colors hover:text-foreground"
                      onClick={goToRoot}
                    >
                      Topics
                    </button>
                    <ChevronRight className="h-3 w-3 text-muted-foreground" />
                    <span className="truncate font-medium text-foreground max-w-[260px]">
                      {currentTopicName}
                    </span>
                  </>
                ) : (
                  <>
                    <button
                      type="button"
                      className={`transition-colors hover:text-foreground ${section === 'topics' ? 'font-medium text-foreground' : ''}`}
                      onClick={goToRoot}
                    >
                      Topics
                    </button>
                    <ChevronRight className="h-3 w-3 text-muted-foreground" />
                    <button
                      type="button"
                      className={`transition-colors hover:text-foreground ${section === 'subscriptions' ? 'font-medium text-foreground' : ''}`}
                      onClick={() => goToSection('subscriptions')}
                    >
                      Subscriptions
                    </button>
                    <ChevronRight className="h-3 w-3 text-muted-foreground" />
                    <button
                      type="button"
                      className={`transition-colors hover:text-foreground ${section === 'platform-applications' ? 'font-medium text-foreground' : ''}`}
                      onClick={() => goToSection('platform-applications')}
                    >
                      Platform Apps
                    </button>
                    <ChevronRight className="h-3 w-3 text-muted-foreground" />
                    <button
                      type="button"
                      className={`transition-colors hover:text-foreground ${section === 'sms' ? 'font-medium text-foreground' : ''}`}
                      onClick={() => goToSection('sms')}
                    >
                      SMS Sandbox
                    </button>
                  </>
                )}
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

      <FramePanel className="flex-1 overflow-hidden border-none bg-transparent p-0 shadow-none">
        <Outlet key={refreshKey} context={{ api, region }} />
      </FramePanel>
    </Frame>
  )
}

export type SNSBrowserOutletContext = {
  api: SNSBrowserApi
  region: string
}
