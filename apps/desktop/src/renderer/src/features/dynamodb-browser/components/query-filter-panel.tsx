/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { ArrowDownAZ, ArrowUpAZ, Filter, Play, Plus, RotateCw, X } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import { Input } from '@renderer/components/ui/input'
import { Badge } from '@renderer/components/ui/badge'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue
} from '@renderer/components/ui/select'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@renderer/components/ui/tabs'
import { COMPARISON_OPERATORS, type ComparisonOperator, type FetchMode, type FilterCondition } from '../types'

interface QueryFilterPanelProps {
  fetchMode: FetchMode
  setFetchMode: (mode: FetchMode) => void
  filterConditions: FilterCondition[]
  setFilterConditions: (conditions: FilterCondition[]) => void
  queryPkValue: string
  setQueryPkValue: (val: string) => void
  querySkValue: string
  setQuerySkValue: (val: string) => void
  querySkOperator: ComparisonOperator
  setQuerySkOperator: (op: ComparisonOperator) => void
  queryIndexName: string
  setQueryIndexName: (name: string) => void
  querySortAsc: boolean
  setQuerySortAsc: (asc: boolean) => void
  activeKeySchema: { pk: string; sk: string; pkType: 'S' | 'N'; skType: 'S' | 'N' }
  availableIndexes: { name: string; type: 'Table' | 'GSI' | 'LSI' }[]
  executeSearch: () => void
  clearFilters: () => void
  isOpen: boolean
  onClose: () => void
}

const SK_OPERATORS: ComparisonOperator[] = ['=', '<', '<=', '>', '>=', 'begins_with']

export function QueryFilterPanel({
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
  activeKeySchema,
  availableIndexes,
  executeSearch,
  clearFilters,
  isOpen,
  onClose
}: QueryFilterPanelProps) {
  if (!isOpen) return null

  const addFilterCondition = () => {
    setFilterConditions([
      ...filterConditions,
      {
        id: crypto.randomUUID(),
        attribute: '',
        operator: '=',
        value: '',
        valueType: 'S'
      }
    ])
  }

  const updateFilterCondition = (id: string, updates: Partial<FilterCondition>) => {
    setFilterConditions(
      filterConditions.map((c) => (c.id === id ? { ...c, ...updates } : c))
    )
  }

  const removeFilterCondition = (id: string) => {
    setFilterConditions(filterConditions.filter((c) => c.id !== id))
  }

  const needsValue = (op: ComparisonOperator) =>
    op !== 'attribute_exists' && op !== 'attribute_not_exists'

  const handleRun = () => {
    executeSearch()
  }

  const activeFilterCount = filterConditions.filter((c) => c.attribute.trim()).length

  return (
    <div className="rounded-xl border border-border bg-muted/30 overflow-hidden animate-in slide-in-from-top-2 duration-200">
      <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium">Advanced Search</span>
          {activeFilterCount > 0 && (
            <Badge variant="outline" className="bg-primary/10 text-primary border-primary/30">
              {activeFilterCount} filter{activeFilterCount !== 1 ? 's' : ''}
            </Badge>
          )}
        </div>
        <Button variant="ghost" size="icon-sm" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <Tabs
        defaultValue={fetchMode}
        onValueChange={(val) => setFetchMode(val as FetchMode)}
        className="px-4 pt-3"
      >
        <TabsList>
          <TabsTrigger value="scan">Scan</TabsTrigger>
          <TabsTrigger value="query">Query</TabsTrigger>
        </TabsList>

        {/* Scan panel */}
        <TabsContent value="scan" className="py-3 space-y-3">
          <p className="text-xs text-muted-foreground">
            Scan reads every item and applies optional filters. Use for exploring data when you don't know the key.
          </p>
        </TabsContent>

        {/* Query panel */}
        <TabsContent value="query" className="py-3 space-y-4">
          <p className="text-xs text-muted-foreground">
            Query uses the partition key (required) for efficient lookups. Optionally add a sort key condition and choose an index.
          </p>

          {/* Index selector */}
          {availableIndexes.length > 1 && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-foreground">Index</label>
              <Select value={queryIndexName} onValueChange={(val) => setQueryIndexName(val ?? '')}>
                <SelectTrigger className="h-8 shadow-xs/5">
                  <SelectValue placeholder="Table (default)" />
                </SelectTrigger>
                <SelectContent>
                  {availableIndexes.map((idx) => (
                    <SelectItem key={idx.name || '__table'} value={idx.name}>
                      {idx.name || 'Table (Primary Key)'}
                      {idx.type !== 'Table' && (
                        <span className="ml-2 text-xs text-muted-foreground">({idx.type})</span>
                      )}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Partition Key */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-foreground">
              Partition Key — <span className="font-mono text-primary">{activeKeySchema.pk}</span>
              <span className="text-muted-foreground ml-1">({activeKeySchema.pkType})</span>
            </label>
            <Input
              type="text"
              placeholder={`Enter ${activeKeySchema.pk} value...`}
              value={queryPkValue}
              onChange={(e) => setQueryPkValue(e.target.value)}
              className="h-8"
            />
          </div>

          {/* Sort Key (if exists) */}
          {activeKeySchema.sk && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-foreground">
                Sort Key — <span className="font-mono text-primary">{activeKeySchema.sk}</span>
                <span className="text-muted-foreground ml-1">({activeKeySchema.skType})</span>
                <span className="text-muted-foreground ml-1">(optional)</span>
              </label>
              <div className="flex items-center gap-2">
                <Select value={querySkOperator} onValueChange={(val) => setQuerySkOperator(val as ComparisonOperator)}>
                  <SelectTrigger className="h-8 w-[150px] shadow-xs/5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SK_OPERATORS.map((op) => (
                      <SelectItem key={op} value={op}>
                        {COMPARISON_OPERATORS.find((c) => c.value === op)?.label ?? op}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  type="text"
                  placeholder={`Enter ${activeKeySchema.sk} value...`}
                  value={querySkValue}
                  onChange={(e) => setQuerySkValue(e.target.value)}
                  className="h-8 flex-1"
                />
              </div>
            </div>
          )}

          {/* Sort direction */}
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setQuerySortAsc(!querySortAsc)}
              className="gap-1.5"
            >
              {querySortAsc ? (
                <ArrowUpAZ className="h-3.5 w-3.5" />
              ) : (
                <ArrowDownAZ className="h-3.5 w-3.5" />
              )}
              {querySortAsc ? 'Ascending' : 'Descending'}
            </Button>
          </div>
        </TabsContent>
      </Tabs>

      {/* Filter conditions — shared between Scan and Query */}
      <div className="border-t border-border px-4 py-3 space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-foreground">Filter Conditions</span>
          <Button variant="ghost" size="sm" onClick={addFilterCondition} className="gap-1.5">
            <Plus className="h-3.5 w-3.5" />
            Add Filter
          </Button>
        </div>

        {filterConditions.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            No filters applied. Results will include all items{fetchMode === 'query' ? ' matching the key condition' : ''}.
          </p>
        ) : (
          <div className="space-y-2">
            {filterConditions.map((condition, idx) => (
              <div key={condition.id} className="flex items-center gap-2">
                {idx > 0 && (
                  <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground w-8 shrink-0 text-center">
                    AND
                  </span>
                )}
                <Input
                  type="text"
                  placeholder="Attribute name"
                  value={condition.attribute}
                  onChange={(e) => updateFilterCondition(condition.id, { attribute: e.target.value })}
                  className="h-8 w-36"
                />
                <Select
                  value={condition.operator}
                  onValueChange={(val) => updateFilterCondition(condition.id, { operator: val as ComparisonOperator })}
                >
                  <SelectTrigger className="h-8 w-[140px] shadow-xs/5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {COMPARISON_OPERATORS.map((op) => (
                      <SelectItem key={op.value} value={op.value}>
                        {op.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {needsValue(condition.operator) && (
                  <>
                    <Input
                      type="text"
                      placeholder="Value"
                      value={condition.value}
                      onChange={(e) => updateFilterCondition(condition.id, { value: e.target.value })}
                      className="h-8 flex-1"
                    />
                    <Select
                      value={condition.valueType}
                      onValueChange={(val) => updateFilterCondition(condition.id, { valueType: val as 'S' | 'N' | 'BOOL' })}
                    >
                      <SelectTrigger className="h-8 w-20 shadow-xs/5">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="S">String</SelectItem>
                        <SelectItem value="N">Number</SelectItem>
                        <SelectItem value="BOOL">Bool</SelectItem>
                      </SelectContent>
                    </Select>
                  </>
                )}
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => removeFilterCondition(condition.id)}
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between border-t border-border px-4 py-2.5">
        <Button variant="ghost" size="sm" onClick={clearFilters} className="gap-1.5">
          <RotateCw className="h-3.5 w-3.5" />
          Reset
        </Button>
        <Button
          size="sm"
          onClick={handleRun}
          disabled={fetchMode === 'query' && !queryPkValue.trim()}
          className="gap-1.5"
        >
          <Play className="h-3.5 w-3.5" />
          Run {fetchMode === 'query' ? 'Query' : 'Scan'}
        </Button>
      </div>
    </div>
  )
}
