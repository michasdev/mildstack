/* eslint-disable @typescript-eslint/explicit-function-return-type */
import {
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Database,
  Edit,
  Filter,
  Layout,
  Plus,
  Search,
  Trash2
} from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import { Spinner } from '@renderer/components/ui/spinner'
import { ScrollArea } from '@renderer/components/ui/scroll-area'
import { Input } from '@renderer/components/ui/input'
import { Checkbox } from '@renderer/components/ui/checkbox'
import { Badge } from '@renderer/components/ui/badge'
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
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle
} from '@renderer/components/ui/empty'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@renderer/components/ui/table'
import { cn } from '@renderer/lib/utils'
import { useItemsList, type SortKey } from '../hooks/use-items-list'
import { attrToValue } from '../types'
import { ItemEditor } from './item-editor'
import { QueryFilterPanel } from './query-filter-panel'

export function ItemsList() {
  const {
    tableInfo,
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
  } = useItemsList()

  const getSortIcon = (key: SortKey) => {
    if (sortKey !== key) return <ChevronsUpDown className="ml-1 h-3.5 w-3.5 opacity-50" />
    return sortOrder === 'asc' ? (
      <ChevronUp className="ml-1 h-3.5 w-3.5 text-primary" />
    ) : (
      <ChevronDown className="ml-1 h-3.5 w-3.5 text-primary" />
    )
  }

  const formatAttrValue = (av: unknown): string => {
    if (av === undefined || av === null) return '-'
    const val = attrToValue(av as any)
    if (val === null || val === undefined) return 'null'
    if (typeof val === 'object') return JSON.stringify(val)
    return String(val)
  }

  const truncate = (text: string, max = 60) => {
    if (text.length <= max) return text
    return text.slice(0, max) + '…'
  }

  const activeFilterCount = filterConditions.filter((c) => c.attribute.trim()).length
  const hasActiveSearch = activeFilterCount > 0 || (fetchMode === 'query' && queryPkValue.trim())

  return (
    <div className="relative flex h-full flex-col gap-3 p-4">
      {/* Toolbar */}
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-2 flex-1">
          <div className="relative w-full max-w-md">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Filter results locally..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          <Button
            variant={showFilterPanel ? 'default' : 'outline'}
            onClick={() => setShowFilterPanel(!showFilterPanel)}
            className="gap-1.5 shrink-0"
          >
            <Filter className="h-4 w-4" />
            Filters
            {hasActiveSearch && (
              <Badge variant="outline" className="ml-1 bg-primary/10 text-primary border-primary/30">
                {fetchMode === 'query' ? 'Query' : activeFilterCount.toString()}
              </Badge>
            )}
          </Button>
        </div>

        <div className="flex flex-wrap gap-2">
          {selectedItems.size > 0 && (
            <Button variant="destructive" onClick={handleBulkDelete} disabled={isDeleting}>
              {isDeleting ? <Spinner className="h-4 w-4" /> : (
                <Trash2 className="h-4 w-4" />
              )}
              Delete ({selectedItems.size})
            </Button>
          )}
          <Button
            variant="outline"
            onClick={() => setShowAllColumns(!showAllColumns)}
            className="flex items-center gap-2"
          >
            <Layout className="h-4 w-4" />
            {showAllColumns ? 'Show Keys Only' : 'Show All Columns'}
          </Button>
          <Button variant="outline" onClick={handleCreateItem}>
            <Plus className="h-4 w-4" />
            Create Item
          </Button>
        </div>
      </div>

      {/* Advanced filter panel */}
      <QueryFilterPanel
        fetchMode={fetchMode}
        setFetchMode={setFetchMode}
        filterConditions={filterConditions}
        setFilterConditions={setFilterConditions}
        queryPkValue={queryPkValue}
        setQueryPkValue={setQueryPkValue}
        querySkValue={querySkValue}
        setQuerySkValue={setQuerySkValue}
        querySkOperator={querySkOperator}
        setQuerySkOperator={setQuerySkOperator}
        queryIndexName={queryIndexName}
        setQueryIndexName={setQueryIndexName}
        querySortAsc={querySortAsc}
        setQuerySortAsc={setQuerySortAsc}
        activeKeySchema={activeKeySchema}
        availableIndexes={availableIndexes}
        executeSearch={executeSearch}
        clearFilters={clearFilters}
        isOpen={showFilterPanel}
        onClose={() => setShowFilterPanel(false)}
      />

      {/* Data grid */}
      <div className="flex-1 min-h-0 overflow-hidden rounded-2xl border border-border bg-card shadow-xs/5">
        <ScrollArea className="h-full">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">
                  <Checkbox
                    checked={sortedItems.length > 0 && selectedItems.size === sortedItems.length}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        const allKeys = sortedItems
                          .map((item) => getItemKeyString(item))
                          .filter(Boolean)
                        setSelectedItems(new Set(allKeys))
                      } else {
                        setSelectedItems(new Set())
                      }
                    }}
                    aria-label="Select all items"
                  />
                </TableHead>
                {displayColumns.map((col) => (
                  <TableHead key={col}>
                    <button
                      type="button"
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => toggleSort(col)}
                    >
                      {col}
                      {getSortIcon(col)}
                    </button>
                  </TableHead>
                ))}
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && !loadingMore ? (
                <TableRow>
                  <TableCell colSpan={displayColumns.length + 2} className="py-12 text-center">
                    <Spinner className="mx-auto h-6 w-6 text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : sortedItems.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={displayColumns.length + 2} className="py-12">
                    <Empty className="py-4">
                      <EmptyHeader>
                        <EmptyMedia variant="icon">
                          <Database className="h-6 w-6" />
                        </EmptyMedia>
                        <EmptyTitle>No items found</EmptyTitle>
                        <EmptyDescription>
                          {searchQuery
                            ? `No items match "${searchQuery}"`
                            : hasActiveSearch
                              ? 'No items match the current filters. Try adjusting your search criteria.'
                              : 'Create your first item to start storing data.'}
                        </EmptyDescription>
                      </EmptyHeader>
                    </Empty>
                  </TableCell>
                </TableRow>
              ) : (
                sortedItems.map((item) => {
                  const keyString = getItemKeyString(item)
                  const isSelected = selectedItems.has(keyString)

                  return (
                    <TableRow
                      key={keyString}
                      className={cn(isSelected && 'bg-primary/5')}
                      data-state={isSelected ? 'selected' : undefined}
                    >
                      <TableCell>
                        <Checkbox
                          checked={isSelected}
                          onCheckedChange={() => toggleItemSelection(keyString)}
                          aria-label={`Select item ${keyString}`}
                        />
                      </TableCell>
                      {displayColumns.map((col) => (
                        <TableCell key={col} className="max-w-[200px]">
                          <span className="block truncate font-mono text-xs text-foreground">
                            {truncate(formatAttrValue(item[col]))}
                          </span>
                        </TableCell>
                      ))}
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleEditItem(item)}
                            title="Edit item"
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleDeleteSingle(item)}
                            title="Delete item"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
              {hasMore && (
                <TableRow>
                  <TableCell colSpan={displayColumns.length + 2} className="py-0">
                    <div ref={observerTarget} className="h-10">
                      {loadingMore && (
                        <div className="flex h-full items-center justify-center">
                          <Spinner className="h-4 w-4 text-muted-foreground" />
                        </div>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </ScrollArea>
      </div>

      {isEditorOpen && tableInfo && (
        <ItemEditor
          tableInfo={tableInfo}
          item={editingItem}
          isOpen={isEditorOpen}
          onClose={() => {
            setIsEditorOpen(false)
          }}
          onSave={handleSaveItem}
        />
      )}

      <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Items</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedItems.size} item(s)? This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="ghost">Cancel</Button>
            </AlertDialogCancel>
            <Button variant="destructive" onClick={executeBulkDelete} disabled={isDeleting}>
              {isDeleting ? <Spinner className="h-4 w-4" /> : (
                <Trash2 className="h-4 w-4" />
              )}
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
