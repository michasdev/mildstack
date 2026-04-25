/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router'
import { Database, Plus, Search, Trash2 } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import SpotlightCard from '@renderer/components/ui/spotlight-card'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@renderer/components/ui/dialog'
import { Spinner } from '@renderer/components/ui/spinner'
import { Badge } from '@renderer/components/ui/badge'
import { Input } from '@renderer/components/ui/input'
import { Checkbox } from '@renderer/components/ui/checkbox'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle
} from '@renderer/components/ui/empty'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel
} from '@renderer/components/ui/alert-dialog'
import {
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardAction
} from '@renderer/components/ui/card'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue
} from '@renderer/components/ui/select'
import { toast} from 'sonner'
import { cn } from '@renderer/lib/utils'
import type { DynamoDBTableSummary } from '../types'
import type { DynamoDBBrowserOutletContext } from '../dynamodb-layout'

export function TablesList() {
  const { api, region } = useOutletContext<DynamoDBBrowserOutletContext>()
  const navigate = useNavigate()
  const [tables, setTables] = useState<DynamoDBTableSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [newTableName, setNewTableName] = useState('')
  const [partitionKeyName, setPartitionKeyName] = useState('')
  const [partitionKeyType, setPartitionKeyType] = useState<'S' | 'N'>('S')
  const [sortKeyName, setSortKeyName] = useState('')
  const [sortKeyType, setSortKeyType] = useState<'S' | 'N'>('S')
  const [isCreating, setIsCreating] = useState(false)
  const [selectedTables, setSelectedTables] = useState<Set<string>>(new Set())
  const [isDeleting, setIsDeleting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const fetchTables = useCallback(async () => {
    try {
      setLoading(true)
      const tableNames = await api.listTables(region)
      // Describe each table to get details
      const details = await Promise.all(
        tableNames.map((name: string) =>
          api.describeTable(name, region).catch(() => ({
            TableName: name,
            TableStatus: 'UNKNOWN',
            KeySchema: [],
            AttributeDefinitions: [],
            ItemCount: 0,
            TableSizeBytes: 0
          }))
        )
      )
      setTables(details)
      setSelectedTables(new Set())
    } catch (error) {
      console.error('Failed to fetch tables:', error)
      toast.error('Failed to fetch tables', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setLoading(false)
    }
  }, [api, region])

  useEffect(() => {
    void fetchTables()
  }, [fetchTables])

  const filteredTables = useMemo(
    () =>
      tables.filter((table) =>
        table.TableName.toLowerCase().includes(searchQuery.toLowerCase())
      ),
    [tables, searchQuery]
  )

  const handleTableClick = (tableName: string) => {
    navigate(`/resources/dynamodb/${encodeURIComponent(tableName)}`)
  }

  const toggleTableSelection = (tableName: string) => {
    setSelectedTables((current) => {
      const next = new Set(current)
      if (next.has(tableName)) {
        next.delete(tableName)
      } else {
        next.add(tableName)
      }
      return next
    })
  }

  const handleCreateTable = async () => {
    const name = newTableName.trim()
    const pk = partitionKeyName.trim()
    if (!name || !pk) return

    setIsCreating(true)
    try {
      const keySchema: { AttributeName: string; KeyType: 'HASH' | 'RANGE' }[] = [
        { AttributeName: pk, KeyType: 'HASH' }
      ]
      const attributeDefinitions = [{ AttributeName: pk, AttributeType: partitionKeyType }]

      const sk = sortKeyName.trim()
      if (sk) {
        keySchema.push({ AttributeName: sk, KeyType: 'RANGE' })
        attributeDefinitions.push({ AttributeName: sk, AttributeType: sortKeyType })
      }

      await api.createTable(name, keySchema, attributeDefinitions, region)
      setIsCreateOpen(false)
      resetCreateForm()
      await fetchTables()
      toast.success('Table created successfully', {
        description: `Table "${name}" created successfully.`
      })
    } catch (err) {
      console.error('Failed to create table:', err)
      toast.error('Failed to create table', {
        description: err instanceof Error ? err.message : String(err)
      })
    } finally {
      setIsCreating(false)
    }
  }

  const resetCreateForm = () => {
    setNewTableName('')
    setPartitionKeyName('')
    setPartitionKeyType('S')
    setSortKeyName('')
    setSortKeyType('S')
  }

  const executeBulkDelete = async () => {
    setShowDeleteConfirm(false)
    setIsDeleting(true)
    try {
      for (const table of Array.from(selectedTables)) {
        try {
          await api.deleteTable(table, region)
        } catch (err) {
          console.error(`Failed to delete table ${table}:`, err)
          toast.error(`Failed to delete table ${table}`, {
            description: err instanceof Error ? err.message : String(err)
          })
        }
      }
      await fetchTables()
    } finally {
      setIsDeleting(false)
    }
  }

  const handleBulkDelete = () => {
    if (selectedTables.size === 0) return
    setShowDeleteConfirm(true)
  }

  const getPartitionKey = (table: DynamoDBTableSummary) => {
    const pk = table.KeySchema.find((k) => k.KeyType === 'HASH')
    if (!pk) return '-'
    const attr = table.AttributeDefinitions.find((a) => a.AttributeName === pk.AttributeName)
    return `${pk.AttributeName} (${attr?.AttributeType ?? '?'})`
  }

  const getSortKey = (table: DynamoDBTableSummary) => {
    const sk = table.KeySchema.find((k) => k.KeyType === 'RANGE')
    if (!sk) return null
    const attr = table.AttributeDefinitions.find((a) => a.AttributeName === sk.AttributeName)
    return `${sk.AttributeName} (${attr?.AttributeType ?? '?'})`
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

  return (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search tables..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {selectedTables.size > 0 && (
            <Button variant="destructive" onClick={handleBulkDelete} disabled={isDeleting}>
              {isDeleting && <Spinner className="h-4 w-4" />}
              {!isDeleting && <Trash2 className="h-4 w-4" />}
              Delete ({selectedTables.size})
            </Button>
          )}

          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Plus className="h-4 w-4" />
                Create Table
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Table</DialogTitle>
                <DialogDescription>
                  Create a new DynamoDB table in your local environment.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 px-6 pb-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="table-name">
                    Table name
                  </label>
                  <Input
                    id="table-name"
                    type="text"
                    placeholder="my-table"
                    value={newTableName}
                    onChange={(e) => setNewTableName(e.target.value)}
                    autoFocus
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground" htmlFor="pk-name">
                      Partition key
                    </label>
                    <Input
                      id="pk-name"
                      type="text"
                      placeholder="id"
                      value={partitionKeyName}
                      onChange={(e) => setPartitionKeyName(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground" htmlFor="pk-type">
                      Type
                    </label>
                    <Select value={partitionKeyType} onValueChange={(val) => setPartitionKeyType(val as 'S' | 'N')}>
                      <SelectTrigger className="h-9 shadow-xs/5" id="pk-type">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="S">String (S)</SelectItem>
                        <SelectItem value="N">Number (N)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground" htmlFor="sk-name">
                      Sort key <span className="text-muted-foreground">(optional)</span>
                    </label>
                    <Input
                      id="sk-name"
                      type="text"
                      placeholder="sk"
                      value={sortKeyName}
                      onChange={(e) => setSortKeyName(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground" htmlFor="sk-type">
                      Type
                    </label>
                    <Select value={sortKeyType} onValueChange={(val) => setSortKeyType(val as 'S' | 'N')}>
                      <SelectTrigger className="h-9 shadow-xs/5" id="sk-type">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="S">String (S)</SelectItem>
                        <SelectItem value="N">Number (N)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="ghost">Cancel</Button>
                </DialogClose>
                <Button
                  onClick={handleCreateTable}
                  disabled={!newTableName.trim() || !partitionKeyName.trim()}
                >
                  {isCreating && <Spinner className="h-4 w-4" />}
                  {!isCreating && <Plus className="h-4 w-4" />}
                  Create
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-hidden">
        {loading ? (
          <div className="flex h-40 items-center justify-center">
            <Spinner className="h-6 w-6 text-muted-foreground" />
          </div>
        ) : filteredTables.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Database className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No tables found</EmptyTitle>
              <EmptyDescription>
                {searchQuery
                  ? `No tables match "${searchQuery}"`
                  : 'Create your first DynamoDB table to start storing data.'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="grid grid-cols-1 gap-4 pb-6 md:grid-cols-2 xl:grid-cols-3">
            {filteredTables.map((table) => {
              const selected = selectedTables.has(table.TableName)
              const sortKey = getSortKey(table)

              return (
                <SpotlightCard
                  key={table.TableName}
                  className={cn(
                    'cursor-pointer transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md',
                    selected && 'border-primary/40 bg-primary/5'
                  )}
                  onClick={() => handleTableClick(table.TableName)}
                >
                  <CardHeader className="gap-3">
                    <CardAction onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        checked={selected}
                        onCheckedChange={() => toggleTableSelection(table.TableName)}
                        aria-label={`Select ${table.TableName}`}
                      />
                    </CardAction>
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-primary/20 bg-primary/10 text-primary">
                        <Database className="h-5 w-5" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <CardTitle className="truncate text-sm">{table.TableName}</CardTitle>
                        <CardDescription className="mt-1 truncate">
                          PK: {getPartitionKey(table)}
                          {sortKey && ` · SK: ${sortKey}`}
                        </CardDescription>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="flex items-center justify-between gap-3 pt-0">
                    <div className="flex items-center gap-2">
                      <Badge
                        variant="outline"
                        className={statusColor(table.TableStatus)}
                      >
                        {table.TableStatus}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {table.ItemCount} item{table.ItemCount !== 1 ? 's' : ''}
                      </span>
                    </div>
                    <span className="text-xs text-muted-foreground">Click to browse</span>
                  </CardContent>
                </SpotlightCard>
              )
            })}
          </div>
        )}
      </div>

      <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Tables</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedTables.size} table(s)? This action cannot
              be undone and all data in the tables will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="ghost">Cancel</Button>
            </AlertDialogCancel>
            <Button variant="destructive" onClick={executeBulkDelete} disabled={isDeleting}>
              {isDeleting && <Spinner className="h-4 w-4" />}
              {!isDeleting && <Trash2 className="h-4 w-4" />}
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
