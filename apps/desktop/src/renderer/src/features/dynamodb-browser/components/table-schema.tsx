/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { Database, Hash, Key, Layers } from 'lucide-react'

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
import type { DynamoDBTableSummary } from '../types'

interface TableSchemaProps {
  tableInfo: DynamoDBTableSummary | null
  loading: boolean
}

export function TableSchema({ tableInfo, loading }: TableSchemaProps) {
  if (loading || !tableInfo) {
    return (
      <div className="flex h-40 items-center justify-center">
        <Spinner className="h-6 w-6 text-muted-foreground" />
      </div>
    )
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const index = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${Number((bytes / 1024 ** index).toFixed(2))} ${sizes[index]}`
  }

  const formatDate = (date?: string) => {
    if (!date) return '-'
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(new Date(date))
  }

  const billingLabel = tableInfo.BillingMode === 'PROVISIONED' ? 'Provisioned' : 'On-demand'

  return (
    <div className="flex flex-col gap-6 p-4">
      {/* Table Overview */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
          <Database className="h-4 w-4 text-primary" />
          Table Overview
        </h3>
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          <InfoCard label="Status">
            <Badge
              variant="outline"
              size="sm"
              className={
                tableInfo.TableStatus === 'ACTIVE'
                  ? 'bg-green-500/15 text-green-400 border-green-500/30'
                  : ''
              }
            >
              {tableInfo.TableStatus}
            </Badge>
          </InfoCard>
          <InfoCard label="Item Count">
            <span className="text-sm font-medium text-foreground">
              {tableInfo.ItemCount.toLocaleString()}
            </span>
          </InfoCard>
          <InfoCard label="Table Size">
            <span className="text-sm font-medium text-foreground">
              {formatBytes(tableInfo.TableSizeBytes)}
            </span>
          </InfoCard>
          <InfoCard label="Billing Mode">
            <Badge variant="outline" size="sm">
              {billingLabel}
            </Badge>
          </InfoCard>
          <InfoCard label="Created">
            <span className="text-sm text-foreground">
              {formatDate(tableInfo.CreationDateTime)}
            </span>
          </InfoCard>
        </div>
      </section>

      {/* Primary Key Schema */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
          <Key className="h-4 w-4 text-primary" />
          Primary Key Schema
        </h3>
        <div className="rounded-xl border border-border overflow-hidden">
          <Table variant="card">
            <TableHeader>
              <TableRow>
                <TableHead>Attribute Name</TableHead>
                <TableHead>Key Type</TableHead>
                <TableHead>Data Type</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tableInfo.KeySchema.map((ks) => {
                const attr = tableInfo.AttributeDefinitions.find(
                  (a) => a.AttributeName === ks.AttributeName
                )
                return (
                  <TableRow key={ks.AttributeName}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {ks.KeyType === 'HASH' ? (
                          <Hash className="h-3.5 w-3.5 text-primary" />
                        ) : (
                          <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                        )}
                        <span className="font-mono text-sm font-medium">{ks.AttributeName}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" size="sm">
                        {ks.KeyType === 'HASH' ? 'Partition Key' : 'Sort Key'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-sm text-muted-foreground">
                        {formatAttrType(attr?.AttributeType)}
                      </span>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      </section>

      {/* All Attribute Definitions */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
          <Layers className="h-4 w-4 text-primary" />
          Attribute Definitions
        </h3>
        {tableInfo.AttributeDefinitions.length === 0 ? (
          <Empty className="py-4">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Layers className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No attribute definitions</EmptyTitle>
              <EmptyDescription>
                No explicit attribute definitions are set for this table.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="rounded-xl border border-border overflow-hidden">
            <Table variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>Attribute Name</TableHead>
                  <TableHead>Data Type</TableHead>
                  <TableHead>Used In</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tableInfo.AttributeDefinitions.map((attr) => {
                  const usages: string[] = []
                  const inPK = tableInfo.KeySchema.find(
                    (ks) => ks.AttributeName === attr.AttributeName
                  )
                  if (inPK) usages.push(inPK.KeyType === 'HASH' ? 'Table PK' : 'Table SK')

                  for (const gsi of tableInfo.GlobalSecondaryIndexes ?? []) {
                    const inGSI = gsi.KeySchema.find(
                      (ks) => ks.AttributeName === attr.AttributeName
                    )
                    if (inGSI) usages.push(`GSI: ${gsi.IndexName}`)
                  }

                  for (const lsi of tableInfo.LocalSecondaryIndexes ?? []) {
                    const inLSI = lsi.KeySchema.find(
                      (ks) => ks.AttributeName === attr.AttributeName
                    )
                    if (inLSI) usages.push(`LSI: ${lsi.IndexName}`)
                  }

                  return (
                    <TableRow key={attr.AttributeName}>
                      <TableCell>
                        <span className="font-mono text-sm font-medium">
                          {attr.AttributeName}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-sm text-muted-foreground">
                          {formatAttrType(attr.AttributeType)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {usages.length > 0 ? (
                            usages.map((u) => (
                              <Badge key={u} variant="outline" size="sm">
                                {u}
                              </Badge>
                            ))
                          ) : (
                            <span className="text-xs text-muted-foreground">-</span>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </section>
    </div>
  )
}

function InfoCard({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-border bg-muted/30 px-4 py-3 space-y-1">
      <p className="text-xs text-muted-foreground">{label}</p>
      <div>{children}</div>
    </div>
  )
}

function formatAttrType(type?: string) {
  switch (type) {
    case 'S':
      return 'String (S)'
    case 'N':
      return 'Number (N)'
    case 'B':
      return 'Binary (B)'
    default:
      return type ?? '-'
  }
}
