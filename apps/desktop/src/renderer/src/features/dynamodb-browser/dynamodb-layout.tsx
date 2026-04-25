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
import { useDynamoDBClient } from './hooks/use-dynamodb-client'
import type { DynamoDBBrowserApi } from './types'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@renderer/components/ui/select'
import { Tooltip, TooltipTrigger, TooltipContent } from '@renderer/components/ui/tooltip'
import { regions } from '@renderer/constants'

export function DynamoDBLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { api, region, setRegion } = useDynamoDBClient()
  const [refreshKey, setRefreshKey] = useState(0)

  const pathParts = location.pathname.split('/').filter(Boolean)
  const isDynamoDBRoot = pathParts[pathParts.length - 1] === 'dynamodb'

  let currentTable = ''

  if (!isDynamoDBRoot) {
    const dynamoIndex = pathParts.indexOf('dynamodb')
    if (dynamoIndex !== -1 && pathParts.length > dynamoIndex + 1) {
      currentTable = decodeURIComponent(pathParts[dynamoIndex + 1])
    }
  }

  const navigateToDynamoDBRoot = () => navigate('/resources/dynamodb')

  return (
    <Frame className="w-full h-full flex flex-col">
      <FrameHeader className="flex-none">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between w-full">
          <div className="flex items-start gap-3">
            <div className="space-y-2">
              <FrameTitle className="flex items-center gap-2">
                DynamoDB Resource Browser
              </FrameTitle>
              <FrameDescription className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  className={`transition-colors hover:text-foreground ${isDynamoDBRoot ? 'font-medium text-foreground' : ''
                    }`}
                  onClick={navigateToDynamoDBRoot}
                >
                  Tables
                </button>
                {currentTable && (
                  <>
                    <ChevronRight className="w-3 h-3 text-muted-foreground" />
                    <span className="font-medium text-foreground truncate max-w-[260px]">
                      {currentTable}
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

export type DynamoDBBrowserOutletContext = {
  api: DynamoDBBrowserApi
  region: string
}
