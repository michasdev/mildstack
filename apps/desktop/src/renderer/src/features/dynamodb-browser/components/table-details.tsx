/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext, useParams } from 'react-router'
import { Database, Key, Rows3 } from 'lucide-react'

import { Badge } from '@renderer/components/ui/badge'
import { Tabs, TabsList, TabsTab, TabsPanel } from '@renderer/components/ui/tabs'
import { ItemsList } from './items-list'
import { TableSchema } from './table-schema'
import { TableIndexes } from './table-indexes'
import type { DynamoDBTableSummary } from '../types'
import type { DynamoDBBrowserOutletContext } from '../dynamodb-layout'

export function TableDetails() {
  const { tableName } = useParams()
  const { api, region } = useOutletContext<DynamoDBBrowserOutletContext>()
  const [tableInfo, setTableInfo] = useState<DynamoDBTableSummary | null>(null)
  const [loadingInfo, setLoadingInfo] = useState(true)

  const fetchTableInfo = useCallback(async () => {
    if (!tableName) return
    setLoadingInfo(true)
    try {
      const info = await api.describeTable(tableName, region)
      setTableInfo(info)
    } catch (err) {
      console.error('Failed to describe table:', err)
    } finally {
      setLoadingInfo(false)
    }
  }, [api, tableName, region])

  useEffect(() => {
    void fetchTableInfo()
  }, [fetchTableInfo])

  return (
    <div className="flex h-full flex-col rounded-2xl border border-border bg-card shadow-xs/5">
      <Tabs defaultValue="items" className="">
        <div className="flex flex-col gap-3 border-b border-border px-4 py-3 md:flex-row md:items-center md:justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold">{tableName}</h2>
              <Badge variant="outline" size="sm">
                Table details
              </Badge>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              Browse items, schema, and indexes.
            </p>
          </div>
          <TabsList className="mx-4 mt-2">
            <TabsTab value="items">
              <Rows3 className="h-4 w-4" />
              Items
            </TabsTab>
            <TabsTab value="schema">
              <Key className="h-4 w-4" />
              Schema
            </TabsTab>
            <TabsTab value="indexes">
              <Database className="h-4 w-4" />
              Indexes
            </TabsTab>
          </TabsList>
        </div>

        <TabsPanel value="items" className="flex-1 min-h-0">
          <ItemsList />
        </TabsPanel>

        <TabsPanel value="schema" className="flex-1 min-h-0 overflow-y-auto">
          <TableSchema tableInfo={tableInfo} loading={loadingInfo} />
        </TabsPanel>

        <TabsPanel value="indexes" className="flex-1 min-h-0 overflow-y-auto">
          <TableIndexes tableInfo={tableInfo} loading={loadingInfo} />
        </TabsPanel>
      </Tabs>
    </div>
  )
}
