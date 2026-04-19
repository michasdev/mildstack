/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useOutletContext, useParams } from 'react-router'
import { toastManager } from '@renderer/components/ui/toast'
import type { DynamoDBItem, DynamoDBTableSummary, FilterCondition, FetchMode, ComparisonOperator } from '../types'
import type { DynamoDBBrowserOutletContext } from '../dynamodb-layout'
import { valueToAttr } from '../types'

export type SortKey = string
export type SortOrder = 'asc' | 'desc'

/* ── Expression builder ─────────────────────────────────────────────── */

function buildExpressions(conditions: FilterCondition[]) {
  const validConditions = conditions.filter((c) => c.attribute.trim())
  if (validConditions.length === 0) return { expression: undefined, names: undefined, values: undefined }

  const parts: string[] = []
  const names: Record<string, string> = {}
  const values: Record<string, any> = {}

  for (let i = 0; i < validConditions.length; i++) {
    const c = validConditions[i]
    const nameKey = `#attr${i}`
    names[nameKey] = c.attribute

    const op = c.operator as ComparisonOperator

    if (op === 'attribute_exists') {
      parts.push(`attribute_exists(${nameKey})`)
    } else if (op === 'attribute_not_exists') {
      parts.push(`attribute_not_exists(${nameKey})`)
    } else {
      const valueKey = `:val${i}`
      values[valueKey] = resolveValue(c.value, c.valueType)

      if (op === 'begins_with') {
        parts.push(`begins_with(${nameKey}, ${valueKey})`)
      } else if (op === 'contains') {
        parts.push(`contains(${nameKey}, ${valueKey})`)
      } else {
        parts.push(`${nameKey} ${op} ${valueKey}`)
      }
    }
  }

  return {
    expression: parts.join(' AND '),
    names: Object.keys(names).length > 0 ? names : undefined,
    values: Object.keys(values).length > 0 ? values : undefined
  }
}

function resolveValue(raw: string, type: 'S' | 'N' | 'BOOL') {
  switch (type) {
    case 'N':
      return valueToAttr(Number(raw))
    case 'BOOL':
      return valueToAttr(raw.toLowerCase() === 'true')
    default:
      return valueToAttr(raw)
  }
}

/* ── Hook ───────────────────────────────────────────────────────────── */

export function useItemsList() {
  const { api, region } = useOutletContext<DynamoDBBrowserOutletContext>()
  const { tableName } = useParams()

  const [tableInfo, setTableInfo] = useState<DynamoDBTableSummary | null>(null)
  const [items, setItems] = useState<DynamoDBItem[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [lastEvaluatedKey, setLastEvaluatedKey] = useState<DynamoDBItem | undefined>()
  const [hasMore, setHasMore] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set())
  const [isDeleting, setIsDeleting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [editingItem, setEditingItem] = useState<DynamoDBItem | null>(null)
  const [isEditorOpen, setIsEditorOpen] = useState(false)

  const [sortKey, setSortKey] = useState<SortKey>('')
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc')

  // Advanced search state
  const [fetchMode, setFetchMode] = useState<FetchMode>('scan')
  const [filterConditions, setFilterConditions] = useState<FilterCondition[]>([])
  const [queryPkValue, setQueryPkValue] = useState('')
  const [querySkValue, setQuerySkValue] = useState('')
  const [querySkOperator, setQuerySkOperator] = useState<ComparisonOperator>('=')
  const [queryIndexName, setQueryIndexName] = useState<string>('')
  const [querySortAsc, setQuerySortAsc] = useState(true)
  const [showFilterPanel, setShowFilterPanel] = useState(false)

  const observerTarget = useRef<HTMLDivElement>(null)

  // Fetch table schema
  const fetchTableInfo = useCallback(async () => {
    if (!tableName) return
    try {
      const info = await api.describeTable(tableName, region)
      setTableInfo(info)
    } catch (err) {
      console.error('Failed to describe table:', err)
    }
  }, [api, tableName, region])

  // Build a stable string key from an item's primary key for selection
  const getItemKeyString = useCallback(
    (item: DynamoDBItem): string => {
      if (!tableInfo) return JSON.stringify(item)
      const parts: string[] = []
      for (const ks of tableInfo.KeySchema) {
        const attr = item[ks.AttributeName]
        if (attr) parts.push(`${ks.AttributeName}=${JSON.stringify(attr)}`)
      }
      return parts.join('|')
    },
    [tableInfo]
  )

  // Extract the primary key from an item based on table schema
  const extractKey = useCallback(
    (item: DynamoDBItem): DynamoDBItem => {
      if (!tableInfo) return item
      const key: DynamoDBItem = {}
      for (const ks of tableInfo.KeySchema) {
        if (item[ks.AttributeName]) {
          key[ks.AttributeName] = item[ks.AttributeName]
        }
      }
      return key
    },
    [tableInfo]
  )

  // Resolve PK/SK attribute names for the active index (or table)
  const activeKeySchema = useMemo(() => {
    if (!tableInfo) return { pk: '', sk: '', pkType: 'S' as const, skType: 'S' as const }

    if (queryIndexName) {
      const gsi = (tableInfo.GlobalSecondaryIndexes ?? []).find((i) => i.IndexName === queryIndexName)
      const lsi = (tableInfo.LocalSecondaryIndexes ?? []).find((i) => i.IndexName === queryIndexName)
      const idx = gsi ?? lsi
      if (idx) {
        const pkKS = idx.KeySchema.find((k) => k.KeyType === 'HASH')
        const skKS = idx.KeySchema.find((k) => k.KeyType === 'RANGE')
        const pkAttr = tableInfo.AttributeDefinitions.find((a) => a.AttributeName === pkKS?.AttributeName)
        const skAttr = tableInfo.AttributeDefinitions.find((a) => a.AttributeName === skKS?.AttributeName)
        return {
          pk: pkKS?.AttributeName ?? '',
          sk: skKS?.AttributeName ?? '',
          pkType: (pkAttr?.AttributeType ?? 'S') as 'S' | 'N',
          skType: (skAttr?.AttributeType ?? 'S') as 'S' | 'N'
        }
      }
    }

    const pkKS = tableInfo.KeySchema.find((k) => k.KeyType === 'HASH')
    const skKS = tableInfo.KeySchema.find((k) => k.KeyType === 'RANGE')
    const pkAttr = tableInfo.AttributeDefinitions.find((a) => a.AttributeName === pkKS?.AttributeName)
    const skAttr = tableInfo.AttributeDefinitions.find((a) => a.AttributeName === skKS?.AttributeName)
    return {
      pk: pkKS?.AttributeName ?? '',
      sk: skKS?.AttributeName ?? '',
      pkType: (pkAttr?.AttributeType ?? 'S') as 'S' | 'N',
      skType: (skAttr?.AttributeType ?? 'S') as 'S' | 'N'
    }
  }, [tableInfo, queryIndexName])

  // Available indexes for query
  const availableIndexes = useMemo(() => {
    if (!tableInfo) return []
    const indexes: { name: string; type: 'Table' | 'GSI' | 'LSI' }[] = [
      { name: '', type: 'Table' }
    ]
    for (const gsi of tableInfo.GlobalSecondaryIndexes ?? []) {
      indexes.push({ name: gsi.IndexName, type: 'GSI' })
    }
    for (const lsi of tableInfo.LocalSecondaryIndexes ?? []) {
      indexes.push({ name: lsi.IndexName, type: 'LSI' })
    }
    return indexes
  }, [tableInfo])

  const fetchItems = useCallback(
    async (startKey?: DynamoDBItem, isLoadMore = false) => {
      if (!tableName) return

      try {
        if (isLoadMore) {
          setLoadingMore(true)
        } else {
          setLoading(true)
        }

        // Build filter expressions from conditions
        const filterExpr = buildExpressions(filterConditions)

        let response

        if (fetchMode === 'query' && queryPkValue.trim()) {
          // Build key condition expression
          const keyNames: Record<string, string> = { ...filterExpr.names }
          const keyValues: Record<string, any> = { ...filterExpr.values }

          keyNames['#pk'] = activeKeySchema.pk
          keyValues[':pkval'] = resolveValue(queryPkValue, activeKeySchema.pkType)

          let keyCondition = '#pk = :pkval'

          if (activeKeySchema.sk && querySkValue.trim()) {
            keyNames['#sk'] = activeKeySchema.sk
            keyValues[':skval'] = resolveValue(querySkValue, activeKeySchema.skType)

            const skOp = querySkOperator
            if (skOp === 'begins_with') {
              keyCondition += ' AND begins_with(#sk, :skval)'
            } else {
              keyCondition += ` AND #sk ${skOp} :skval`
            }
          }

          response = await api.query(
            tableName,
            keyCondition,
            Object.keys(keyNames).length > 0 ? keyNames : undefined,
            Object.keys(keyValues).length > 0 ? keyValues : undefined,
            queryIndexName || undefined,
            filterExpr.expression,
            startKey,
            50,
            querySortAsc,
            region
          )
        } else {
          // Scan mode
          response = await api.scan(
            tableName,
            startKey,
            50,
            region,
            filterExpr.expression,
            filterExpr.names,
            filterExpr.values
          )
        }

        if (isLoadMore) {
          setItems((prev) => [...prev, ...response.items])
        } else {
          setItems(response.items)
          setSelectedItems(new Set())
        }
        setHasMore(!!response.lastEvaluatedKey)
        setLastEvaluatedKey(response.lastEvaluatedKey)
      } catch (error) {
        console.error('Failed to fetch items:', error)
        toastManager.add({
          title: fetchMode === 'query' ? 'Query failed' : 'Scan failed',
          type: 'error',
          description: error instanceof Error ? error.message : String(error)
        })
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [api, tableName, region, fetchMode, filterConditions, queryPkValue, querySkValue, querySkOperator, queryIndexName, querySortAsc, activeKeySchema]
  )

  const [showAllColumns, setShowAllColumns] = useState(false)

  // Discover all unique attribute names across items
  const allAttributeNames = useMemo(() => {
    const names = new Set<string>()
    for (const item of items) {
      for (const key of Object.keys(item)) {
        names.add(key)
      }
    }
    // Sort names to keep consistent order (keys first is handled by displayColumns)
    return Array.from(names).sort()
  }, [items])

  // Get the key attribute names
  const keyAttributeNames = useMemo(() => {
    if (!tableInfo) return []
    return tableInfo.KeySchema.map((ks) => ks.AttributeName)
  }, [tableInfo])

  // Columns to display: key columns first, then other attributes if requested
  const displayColumns = useMemo(() => {
    if (!showAllColumns) {
      return keyAttributeNames
    }
    const keyNames = new Set(keyAttributeNames)
    const others = allAttributeNames.filter((n) => !keyNames.has(n))
    return [...keyAttributeNames, ...others]
  }, [keyAttributeNames, allAttributeNames, showAllColumns])

  // Filter items by search query (matches attribute values as text)
  const filteredItems = useMemo(() => {
    if (!searchQuery.trim()) return items
    const query = searchQuery.toLowerCase()
    return items.filter((item) =>
      Object.values(item).some((av) => JSON.stringify(av).toLowerCase().includes(query))
    )
  }, [items, searchQuery])

  // Sort items
  const sortedItems = useMemo(() => {
    if (!sortKey) return filteredItems
    const sorted = [...filteredItems]
    sorted.sort((a, b) => {
      const aVal = JSON.stringify(a[sortKey] ?? '')
      const bVal = JSON.stringify(b[sortKey] ?? '')
      if (aVal < bVal) return sortOrder === 'asc' ? -1 : 1
      if (aVal > bVal) return sortOrder === 'asc' ? 1 : -1
      return 0
    })
    return sorted
  }, [filteredItems, sortKey, sortOrder])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortOrder('asc')
    }
  }

  useEffect(() => {
    void fetchTableInfo()
  }, [fetchTableInfo])

  useEffect(() => {
    if (tableInfo) {
      void fetchItems()
    }
  }, [tableInfo]) // eslint-disable-line react-hooks/exhaustive-deps -- only refetch on tableInfo change, not on every fetchItems rebuild

  // Infinite scroll observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore && !loading) {
          void fetchItems(lastEvaluatedKey, true)
        }
      },
      { threshold: 0.1 }
    )

    if (observerTarget.current) {
      observer.observe(observerTarget.current)
    }

    return () => observer.disconnect()
  }, [hasMore, loadingMore, loading, lastEvaluatedKey, fetchItems])

  const toggleItemSelection = (keyString: string) => {
    setSelectedItems((current) => {
      const next = new Set(current)
      if (next.has(keyString)) {
        next.delete(keyString)
      } else {
        next.add(keyString)
      }
      return next
    })
  }

  const handleBulkDelete = () => {
    if (selectedItems.size === 0 || !tableName) return
    setShowDeleteConfirm(true)
  }

  const executeBulkDelete = async () => {
    if (!tableName || selectedItems.size === 0) return

    setShowDeleteConfirm(false)
    setIsDeleting(true)

    try {
      let successCount = 0
      let errorCount = 0

      for (const keyString of Array.from(selectedItems)) {
        // Find the item by its key string
        const item = items.find((it) => getItemKeyString(it) === keyString)
        if (!item) continue

        try {
          await api.deleteItem(tableName, extractKey(item), region)
          successCount++
        } catch (err) {
          errorCount++
          console.error('Failed to delete item:', err)
        }
      }

      if (successCount > 0) {
        toastManager.add({
          title: 'Items deleted',
          description: `Successfully deleted ${successCount} item(s).`,
          type: 'success'
        })
      }

      if (errorCount > 0) {
        toastManager.add({
          title: 'Partial deletion failure',
          description: `Failed to delete ${errorCount} item(s).`,
          type: 'error'
        })
      }

      await fetchItems()
      setSelectedItems(new Set())
    } catch (err) {
      console.error('Bulk delete failed:', err)
      toastManager.add({
        title: 'Deletion failed',
        type: 'error',
        description: err instanceof Error ? err.message : 'A network or system error occurred.'
      })
    } finally {
      setIsDeleting(false)
    }
  }

  const handleCreateItem = () => {
    setEditingItem(null)
    setIsEditorOpen(true)
  }

  const handleEditItem = (item: DynamoDBItem) => {
    setEditingItem(item)
    setIsEditorOpen(true)
  }

  const handleSaveItem = async (item: DynamoDBItem) => {
    if (!tableName) return
    try {
      await api.putItem(tableName, item, region)
      toastManager.add({
        title: editingItem ? 'Item updated' : 'Item created',
        type: 'success'
      })
      setIsEditorOpen(false)
      setEditingItem(null)
      await fetchItems()
    } catch (err) {
      console.error('Failed to save item:', err)
      toastManager.add({
        title: 'Failed to save item',
        type: 'error',
        description: err instanceof Error ? err.message : String(err)
      })
    }
  }

  const handleDeleteSingle = async (item: DynamoDBItem) => {
    if (!tableName) return
    try {
      await api.deleteItem(tableName, extractKey(item), region)
      toastManager.add({ title: 'Item deleted', type: 'success' })
      await fetchItems()
    } catch (err) {
      console.error('Failed to delete item:', err)
      toastManager.add({
        title: 'Failed to delete item',
        type: 'error',
        description: err instanceof Error ? err.message : String(err)
      })
    }
  }

  // Execute search with current filters
  const executeSearch = () => {
    void fetchItems()
  }

  // Clear all filters and reset to basic scan
  const clearFilters = () => {
    setFilterConditions([])
    setQueryPkValue('')
    setQuerySkValue('')
    setQuerySkOperator('=')
    setQueryIndexName('')
    setQuerySortAsc(true)
    setFetchMode('scan')
    void fetchItems()
  }

  return {
    api,
    region,
    tableName,
    tableInfo,
    items,
    loading,
    loadingMore,
    hasMore,
    searchQuery,
    setSearchQuery,
    selectedItems,
    setSelectedItems,
    isDeleting,
    showDeleteConfirm,
    setShowDeleteConfirm,
    editingItem,
    isEditorOpen,
    setIsEditorOpen,
    sortKey,
    sortOrder,
    observerTarget,
    displayColumns,
    keyAttributeNames,
    sortedItems,
    toggleSort,
    toggleItemSelection,
    handleBulkDelete,
    executeBulkDelete,
    handleCreateItem,
    handleEditItem,
    handleSaveItem,
    handleDeleteSingle,
    getItemKeyString,
    fetchItems,
    showAllColumns,
    setShowAllColumns,
    // Advanced search
    fetchMode,
    setFetchMode,
    filterConditions,
    setFilterConditions,
    queryPkValue,
    setQueryPkValue,
    querySkValue,
    setQuerySkValue,
    querySkOperator,
    setQuerySkOperator,
    queryIndexName,
    setQueryIndexName,
    querySortAsc,
    setQuerySortAsc,
    showFilterPanel,
    setShowFilterPanel,
    activeKeySchema,
    availableIndexes,
    executeSearch,
    clearFilters
  }
}

