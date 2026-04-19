/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { Database, Hash, Layers } from 'lucide-react'

import { Badge } from '@renderer/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@renderer/components/ui/table'
import { Spinner } from '@renderer/components/ui/spinner'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle
} from '@renderer/components/ui/empty'
import type { DynamoDBTableSummary, DynamoDBGlobalSecondaryIndex, DynamoDBLocalSecondaryIndex } from '../types'

interface TableIndexesProps {
  tableInfo: DynamoDBTableSummary | null
  loading: boolean
}

export function TableIndexes({ tableInfo, loading }: TableIndexesProps) {
  if (loading || !tableInfo) {
    return (
      <div className="flex h-40 items-center justify-center">
        <Spinner className="h-6 w-6 text-muted-foreground" />
      </div>
    )
  }

  const gsiList = tableInfo.GlobalSecondaryIndexes ?? []
  const lsiList = tableInfo.LocalSecondaryIndexes ?? []
  const hasIndexes = gsiList.length > 0 || lsiList.length > 0

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const index = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${Number((bytes / 1024 ** index).toFixed(2))} ${sizes[index]}`
  }

  const statusColor = (status: string) => {
    switch (status) {
      case 'ACTIVE':
        return 'bg-green-500/15 text-green-400 border-green-500/30'
      case 'CREATING':
        return 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30'
      case 'DELETING':
        return 'bg-red-500/15 text-red-400 border-red-500/30'
      default:
        return ''
    }
  }

  if (!hasIndexes) {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Database className="h-6 w-6" />
            </EmptyMedia>
            <EmptyTitle>No indexes</EmptyTitle>
            <EmptyDescription>
              This table has no Global or Local Secondary Indexes configured.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 p-4">
      {/* Global Secondary Indexes */}
      {gsiList.length > 0 && (
        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
            <Database className="h-4 w-4 text-primary" />
            Global Secondary Indexes
            <Badge variant="outline" size="sm">{gsiList.length}</Badge>
          </h3>
          <div className="space-y-4">
            {gsiList.map((gsi) => (
              <IndexCard key={gsi.IndexName} type="GSI" gsi={gsi} statusColor={statusColor} formatBytes={formatBytes} />
            ))}
          </div>
        </section>
      )}

      {/* Local Secondary Indexes */}
      {lsiList.length > 0 && (
        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
            <Layers className="h-4 w-4 text-primary" />
            Local Secondary Indexes
            <Badge variant="outline" size="sm">{lsiList.length}</Badge>
          </h3>
          <div className="space-y-4">
            {lsiList.map((lsi) => (
              <IndexCard key={lsi.IndexName} type="LSI" lsi={lsi} formatBytes={formatBytes} />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function IndexCard({
  type,
  gsi,
  lsi,
  statusColor,
  formatBytes
}: {
  type: 'GSI' | 'LSI'
  gsi?: DynamoDBGlobalSecondaryIndex
  lsi?: DynamoDBLocalSecondaryIndex
  statusColor?: (status: string) => string
  formatBytes: (bytes: number) => string
}) {
  const index = gsi ?? lsi
  if (!index) return null

  return (
    <div className="rounded-xl border border-border overflow-hidden">
      {/* Index header */}
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted/30 px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm font-semibold text-foreground">
            {index.IndexName}
          </span>
          <Badge variant="outline" size="sm">
            {type}
          </Badge>
          {gsi && statusColor && (
            <Badge variant="outline" size="sm" className={statusColor(gsi.IndexStatus)}>
              {gsi.IndexStatus}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span>{index.ItemCount.toLocaleString()} items</span>
          <span>{formatBytes(index.IndexSizeBytes)}</span>
        </div>
      </div>

      {/* Key schema + projection */}
      <div className="p-4 space-y-3">
        <Table variant="card">
          <TableHeader>
            <TableRow>
              <TableHead>Attribute</TableHead>
              <TableHead>Key Type</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {index.KeySchema.map((ks) => (
              <TableRow key={ks.AttributeName}>
                <TableCell>
                  <div className="flex items-center gap-2">
                    {ks.KeyType === 'HASH' ? (
                      <Hash className="h-3.5 w-3.5 text-primary" />
                    ) : (
                      <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                    )}
                    <span className="font-mono text-sm">{ks.AttributeName}</span>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline" size="sm">
                    {ks.KeyType === 'HASH' ? 'Partition Key' : 'Sort Key'}
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">Projection:</span>
          <Badge variant="outline" size="sm">
            {index.Projection.ProjectionType}
          </Badge>
          {index.Projection.NonKeyAttributes.length > 0 && (
            <span className="text-xs text-muted-foreground">
              ({index.Projection.NonKeyAttributes.join(', ')})
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
