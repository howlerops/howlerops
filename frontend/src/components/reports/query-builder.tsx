import { debounce } from 'lodash-es'
import {
  AlertCircle,
  ArrowDown,
  ArrowUp,
  Database,
  Filter as FilterIcon,
  Play,
  Plus,
  Table as TableIcon,
  Trash2,
  X,
} from 'lucide-react'
import React, { useEffect, useMemo, useState } from 'react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { cn } from '@/lib/utils'
import type {
  AggregationFunction,
  ColumnSelection,
  DatabaseSchema,
  FilterCombinator,
  FilterCondition,
  FilterOperator,
  JoinDefinition,
  JoinType,
  OrderByClause,
  QueryBuilderState,
  QueryPreview,
  QueryValidationError,
  SortDirection,
  TableMetadata,
} from '@/types/reports'

interface QueryBuilderProps {
  state: QueryBuilderState
  onChange: (state: QueryBuilderState) => void
  onGenerateSQL?: (sql: string) => void
  disabled?: boolean
}

export function QueryBuilder({ state, onChange, onGenerateSQL, disabled }: QueryBuilderProps) {
  const [schema, setSchema] = useState<DatabaseSchema | null>(null)
  const [loadingSchema, setLoadingSchema] = useState(false)
  const [preview, setPreview] = useState<QueryPreview | null>(null)
  const [loadingPreview, setLoadingPreview] = useState(false)
  const [validationErrors, setValidationErrors] = useState<QueryValidationError[]>([])

  // Load schema when data source changes
  useEffect(() => {
    if (!state.dataSource) {
      setSchema(null)
      return
    }

    setLoadingSchema(true)
    fetchDatabaseSchema(state.dataSource)
      .then(setSchema)
      .catch((err) => {
        console.error('Failed to load schema:', err)
        setSchema(null)
      })
      .finally(() => setLoadingSchema(false))
  }, [state.dataSource])

  // Validate query whenever state changes
  useEffect(() => {
    const errors = validateQuery(state, schema)
    setValidationErrors(errors)
  }, [state, schema])

  // Debounced SQL generation
  const debouncedGenerateSQL = useMemo(
    () =>
      debounce((builderState: QueryBuilderState) => {
        if (onGenerateSQL && validationErrors.length === 0) {
          const sql = generateSQL(builderState)
          onGenerateSQL(sql)
        }
      }, 300),
    [onGenerateSQL, validationErrors]
  )

  useEffect(() => {
    debouncedGenerateSQL(state)
  }, [state, debouncedGenerateSQL])

  // Cleanup on unmount
  useEffect(() => {
    return () => debouncedGenerateSQL.cancel()
  }, [debouncedGenerateSQL])

  const currentTable = useMemo(() => {
    if (!schema || !state.table) return null
    return schema.tables.find((t) => t.name === state.table)
  }, [schema, state.table])

  const availableColumns = useMemo(() => {
    if (!currentTable) return []

    const columns = currentTable.columns.map((col) => ({
      table: state.table,
      column: col.name,
      type: col.dataType,
      displayName: `${state.table}.${col.name}`,
    }))

    // Add columns from joined tables
    state.joins.forEach((join) => {
      const joinedTable = schema?.tables.find((t) => t.name === join.table)
      if (joinedTable) {
        joinedTable.columns.forEach((col) => {
          columns.push({
            table: join.alias || join.table,
            column: col.name,
            type: col.dataType,
            displayName: `${join.alias || join.table}.${col.name}`,
          })
        })
      }
    })

    return columns
  }, [currentTable, state.table, state.joins, schema])

  const handleTableChange = (table: string) => {
    onChange({
      ...state,
      table,
      columns: [],
      joins: [],
      filters: [],
      groupBy: [],
      orderBy: [],
    })
  }

  const handleAddColumn = () => {
    onChange({
      ...state,
      columns: [
        ...state.columns,
        {
          table: state.table,
          column: '',
          alias: '',
        },
      ],
    })
  }

  const handleUpdateColumn = (index: number, updates: Partial<ColumnSelection>) => {
    const newColumns = [...state.columns]
    newColumns[index] = { ...newColumns[index], ...updates }
    onChange({ ...state, columns: newColumns })
  }

  const handleRemoveColumn = (index: number) => {
    const newColumns = state.columns.filter((_, i) => i !== index)
    onChange({ ...state, columns: newColumns })
  }

  const handleAddJoin = () => {
    onChange({
      ...state,
      joins: [
        ...state.joins,
        {
          type: 'INNER',
          table: '',
          on: { left: '', right: '' },
        },
      ],
    })
  }

  const handleUpdateJoin = (index: number, updates: Partial<JoinDefinition>) => {
    const newJoins = [...state.joins]
    newJoins[index] = { ...newJoins[index], ...updates }
    onChange({ ...state, joins: newJoins })
  }

  const handleRemoveJoin = (index: number) => {
    const newJoins = state.joins.filter((_, i) => i !== index)
    onChange({ ...state, joins: newJoins })
  }

  const handleAddFilter = () => {
    onChange({
      ...state,
      filters: [
        ...state.filters,
        {
          id: crypto.randomUUID(),
          column: '',
          operator: '=',
          combinator: state.filters.length > 0 ? 'AND' : undefined,
        },
      ],
    })
  }

  const handleUpdateFilter = (index: number, updates: Partial<FilterCondition>) => {
    const newFilters = [...state.filters]
    newFilters[index] = { ...newFilters[index], ...updates }
    onChange({ ...state, filters: newFilters })
  }

  const handleRemoveFilter = (index: number) => {
    const newFilters = state.filters.filter((_, i) => i !== index)
    // Reset combinator for first filter if needed
    if (newFilters.length > 0 && index === 0) {
      newFilters[0] = { ...newFilters[0], combinator: undefined }
    }
    onChange({ ...state, filters: newFilters })
  }

  const handleAddOrderBy = () => {
    onChange({
      ...state,
      orderBy: [...state.orderBy, { column: '', direction: 'ASC' }],
    })
  }

  const handleUpdateOrderBy = (index: number, updates: Partial<OrderByClause>) => {
    const newOrderBy = [...state.orderBy]
    newOrderBy[index] = { ...newOrderBy[index], ...updates }
    onChange({ ...state, orderBy: newOrderBy })
  }

  const handleRemoveOrderBy = (index: number) => {
    const newOrderBy = state.orderBy.filter((_, i) => i !== index)
    onChange({ ...state, orderBy: newOrderBy })
  }

  const handleToggleGroupBy = (column: string) => {
    const isSelected = state.groupBy.includes(column)
    onChange({
      ...state,
      groupBy: isSelected ? state.groupBy.filter((c) => c !== column) : [...state.groupBy, column],
    })
  }

  const handleRunPreview = async () => {
    if (validationErrors.length > 0) return

    setLoadingPreview(true)
    try {
      const sql = generateSQL(state)
      const previewResult = await runPreviewQuery(state.dataSource, sql)
      setPreview(previewResult)
    } catch (err) {
      console.error('Preview failed:', err)
    } finally {
      setLoadingPreview(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Data Source & Table Selection */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5 text-primary" />
            <CardTitle>Data Source</CardTitle>
          </div>
          <CardDescription>Select the database and table to query</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Data Source - This would be passed from parent or selected elsewhere */}
            <div className="space-y-2">
              <Label>Database Connection</Label>
              <Badge variant="outline" className="px-3 py-2">
                {state.dataSource || 'Not selected'}
              </Badge>
            </div>

            {/* Table Selection */}
            <div className="space-y-2">
              <Label htmlFor="table-select">Table</Label>
              <Select value={state.table} onValueChange={handleTableChange} disabled={!schema || disabled}>
                <SelectTrigger id="table-select">
                  <SelectValue placeholder="Select table">
                    {state.table && (
                      <div className="flex items-center gap-2">
                        <TableIcon className="h-4 w-4" />
                        <span>{state.table}</span>
                        {currentTable && (
                          <span className="text-xs text-muted-foreground">
                            ({currentTable.rowCount.toLocaleString()} rows)
                          </span>
                        )}
                      </div>
                    )}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {loadingSchema ? (
                    <SelectItem value="loading" disabled>
                      Loading tables...
                    </SelectItem>
                  ) : schema && schema.tables.length > 0 ? (
                    schema.tables.map((table) => (
                      <SelectItem key={table.name} value={table.name}>
                        <div className="flex items-center justify-between gap-2">
                          <span>{table.name}</span>
                          <span className="text-xs text-muted-foreground">
                            {table.rowCount.toLocaleString()} rows
                          </span>
                        </div>
                      </SelectItem>
                    ))
                  ) : (
                    <SelectItem value="none" disabled>
                      No tables found
                    </SelectItem>
                  )}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Column Selection */}
      {state.table && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Columns</CardTitle>
                <CardDescription>Select columns and apply aggregations</CardDescription>
              </div>
              <Button variant="outline" size="sm" onClick={handleAddColumn} disabled={disabled}>
                <Plus className="mr-2 h-4 w-4" /> Add Column
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {state.columns.length === 0 ? (
              <p className="text-sm text-muted-foreground">No columns selected. Add at least one column.</p>
            ) : (
              state.columns.map((col, idx) => (
                <ColumnSelector
                  key={idx}
                  column={col}
                  availableColumns={availableColumns}
                  onChange={(updates) => handleUpdateColumn(idx, updates)}
                  onRemove={() => handleRemoveColumn(idx)}
                  disabled={disabled}
                />
              ))
            )}
          </CardContent>
        </Card>
      )}

      {/* Joins */}
      {state.table && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Joins</CardTitle>
                <CardDescription>Join additional tables</CardDescription>
              </div>
              <Button variant="outline" size="sm" onClick={handleAddJoin} disabled={disabled}>
                <Plus className="mr-2 h-4 w-4" /> Add Join
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {state.joins.length === 0 ? (
              <p className="text-sm text-muted-foreground">No joins configured (optional).</p>
            ) : (
              state.joins.map((join, idx) => (
                <JoinSelector
                  key={idx}
                  join={join}
                  availableTables={schema?.tables || []}
                  availableColumns={availableColumns}
                  onChange={(updates) => handleUpdateJoin(idx, updates)}
                  onRemove={() => handleRemoveJoin(idx)}
                  disabled={disabled}
                />
              ))
            )}
          </CardContent>
        </Card>
      )}

      {/* Filters */}
      {state.table && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <FilterIcon className="h-5 w-5 text-primary" />
                <div>
                  <CardTitle>Filters</CardTitle>
                  <CardDescription>Add WHERE conditions</CardDescription>
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={handleAddFilter} disabled={disabled}>
                <Plus className="mr-2 h-4 w-4" /> Add Filter
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {state.filters.length === 0 ? (
              <p className="text-sm text-muted-foreground">No filters configured (optional).</p>
            ) : (
              state.filters.map((filter, idx) => (
                <FilterSelector
                  key={filter.id}
                  filter={filter}
                  isFirst={idx === 0}
                  availableColumns={availableColumns}
                  onChange={(updates) => handleUpdateFilter(idx, updates)}
                  onRemove={() => handleRemoveFilter(idx)}
                  disabled={disabled}
                />
              ))
            )}
          </CardContent>
        </Card>
      )}

      {/* Group By & Sort */}
      {state.table && state.columns.length > 0 && (
        <div className="grid gap-6 md:grid-cols-2">
          {/* Group By */}
          <Card>
            <CardHeader>
              <CardTitle>Group By</CardTitle>
              <CardDescription>Group aggregated results</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {state.columns
                .filter((col) => col.column) // Only show selected columns
                .map((col) => {
                  const columnKey = `${col.table}.${col.column}`
                  return (
                    <label key={columnKey} className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={state.groupBy.includes(columnKey)}
                        onChange={() => handleToggleGroupBy(columnKey)}
                        disabled={disabled}
                        className="h-4 w-4"
                      />
                      <span className="text-sm">{col.alias || columnKey}</span>
                      {col.aggregation && (
                        <Badge variant="secondary" className="text-xs">
                          {col.aggregation.toUpperCase()}
                        </Badge>
                      )}
                    </label>
                  )
                })}
              {state.groupBy.length === 0 && state.columns.some((c) => c.aggregation) && (
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription className="text-xs">
                    When using aggregations, select GROUP BY columns or aggregate all rows.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          {/* Sort */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Sort</CardTitle>
                  <CardDescription>Order results</CardDescription>
                </div>
                <Button variant="outline" size="sm" onClick={handleAddOrderBy} disabled={disabled}>
                  <Plus className="mr-2 h-4 w-4" /> Add Sort
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-2">
              {state.orderBy.length === 0 ? (
                <p className="text-sm text-muted-foreground">No sorting configured (optional).</p>
              ) : (
                state.orderBy.map((order, idx) => (
                  <OrderBySelector
                    key={idx}
                    orderBy={order}
                    availableColumns={availableColumns}
                    onChange={(updates) => handleUpdateOrderBy(idx, updates)}
                    onRemove={() => handleRemoveOrderBy(idx)}
                    disabled={disabled}
                  />
                ))
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Limit */}
      {state.table && (
        <Card>
          <CardHeader>
            <CardTitle>Limit</CardTitle>
            <CardDescription>Maximum rows to return</CardDescription>
          </CardHeader>
          <CardContent>
            <Input
              type="number"
              min="1"
              max="10000"
              value={state.limit || ''}
              onChange={(e) => onChange({ ...state, limit: e.target.value ? Number(e.target.value) : undefined })}
              placeholder="No limit"
              disabled={disabled}
              className="w-40"
            />
          </CardContent>
        </Card>
      )}

      {/* Validation Errors */}
      {validationErrors.length > 0 && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Invalid Query</AlertTitle>
          <AlertDescription>
            <ul className="mt-2 list-inside list-disc space-y-1">
              {validationErrors.map((err, idx) => (
                <li key={idx} className="text-sm">
                  {err.message}
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      {/* Preview */}
      {state.table && validationErrors.length === 0 && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Preview</CardTitle>
                <CardDescription>Run query to see results</CardDescription>
              </div>
              <Button size="sm" onClick={handleRunPreview} disabled={loadingPreview || disabled}>
                <Play className="mr-2 h-4 w-4" /> {loadingPreview ? 'Running...' : 'Run Preview'}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {preview && (
              <div className="space-y-4">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">
                    Showing {preview.rows.length} of {preview.totalRows.toLocaleString()} rows
                  </span>
                  <span className="text-muted-foreground">Executed in {preview.executionTimeMs}ms</span>
                </div>
                <div className="overflow-auto rounded-md border">
                  <table className="w-full text-sm">
                    <thead className="border-b bg-muted/50">
                      <tr>
                        {preview.columns.map((col) => (
                          <th key={col} className="px-4 py-2 text-left font-medium">
                            {col}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {preview.rows.slice(0, 10).map((row, rowIdx) => (
                        <tr key={rowIdx} className="border-b last:border-none hover:bg-muted/30">
                          {row.map((cell, cellIdx) => (
                            <td key={cellIdx} className="px-4 py-2">
                              {typeof cell === 'object' ? JSON.stringify(cell) : String(cell)}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ===== Sub-components =====

interface ColumnSelectorProps {
  column: ColumnSelection
  availableColumns: Array<{ table: string; column: string; type: string; displayName: string }>
  onChange: (updates: Partial<ColumnSelection>) => void
  onRemove: () => void
  disabled?: boolean
}

function ColumnSelector({ column, availableColumns, onChange, onRemove, disabled }: ColumnSelectorProps) {
  return (
    <div className="flex gap-2">
      <Select value={column.column} onValueChange={(col) => onChange({ column: col })} disabled={disabled}>
        <SelectTrigger className="flex-1">
          <SelectValue placeholder="Select column" />
        </SelectTrigger>
        <SelectContent>
          {availableColumns.map((col) => (
            <SelectItem key={col.displayName} value={col.column}>
              <div className="flex items-center justify-between gap-2">
                <span>{col.displayName}</span>
                <Badge variant="outline" className="text-xs">
                  {col.type}
                </Badge>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={column.aggregation || 'none'}
        onValueChange={(val) => onChange({ aggregation: val === 'none' ? undefined : (val as AggregationFunction) })}
        disabled={disabled}
      >
        <SelectTrigger className="w-40">
          <SelectValue placeholder="Function" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">None</SelectItem>
          <SelectItem value="count">COUNT</SelectItem>
          <SelectItem value="sum">SUM</SelectItem>
          <SelectItem value="avg">AVG</SelectItem>
          <SelectItem value="min">MIN</SelectItem>
          <SelectItem value="max">MAX</SelectItem>
          <SelectItem value="count_distinct">COUNT DISTINCT</SelectItem>
        </SelectContent>
      </Select>

      <Input
        placeholder="Alias (optional)"
        value={column.alias || ''}
        onChange={(e) => onChange({ alias: e.target.value })}
        disabled={disabled}
        className="w-40"
      />

      <Button variant="ghost" size="icon" onClick={onRemove} disabled={disabled}>
        <Trash2 className="h-4 w-4" />
      </Button>
    </div>
  )
}

interface JoinSelectorProps {
  join: JoinDefinition
  availableTables: TableMetadata[]
  availableColumns: Array<{ table: string; column: string; type: string; displayName: string }>
  onChange: (updates: Partial<JoinDefinition>) => void
  onRemove: () => void
  disabled?: boolean
}

function JoinSelector({ join, availableTables, availableColumns, onChange, onRemove, disabled }: JoinSelectorProps) {
  return (
    <div className="space-y-2 rounded-md border p-3">
      <div className="flex gap-2">
        <Select value={join.type} onValueChange={(type) => onChange({ type: type as JoinType })} disabled={disabled}>
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="INNER">INNER</SelectItem>
            <SelectItem value="LEFT">LEFT</SelectItem>
            <SelectItem value="RIGHT">RIGHT</SelectItem>
            <SelectItem value="FULL">FULL</SelectItem>
          </SelectContent>
        </Select>

        <Select value={join.table} onValueChange={(table) => onChange({ table })} disabled={disabled}>
          <SelectTrigger className="flex-1">
            <SelectValue placeholder="Select table" />
          </SelectTrigger>
          <SelectContent>
            {availableTables.map((table) => (
              <SelectItem key={table.name} value={table.name}>
                {table.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          placeholder="Alias (optional)"
          value={join.alias || ''}
          onChange={(e) => onChange({ alias: e.target.value })}
          disabled={disabled}
          className="w-32"
        />

        <Button variant="ghost" size="icon" onClick={onRemove} disabled={disabled}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex items-center gap-2">
        <Label className="text-xs text-muted-foreground">ON</Label>
        <Select
          value={join.on.left}
          onValueChange={(left) => onChange({ on: { ...join.on, left } })}
          disabled={disabled}
        >
          <SelectTrigger className="flex-1">
            <SelectValue placeholder="Left column" />
          </SelectTrigger>
          <SelectContent>
            {availableColumns.map((col) => (
              <SelectItem key={col.displayName} value={col.displayName}>
                {col.displayName}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="text-sm">=</span>
        <Select
          value={join.on.right}
          onValueChange={(right) => onChange({ on: { ...join.on, right } })}
          disabled={disabled}
        >
          <SelectTrigger className="flex-1">
            <SelectValue placeholder="Right column" />
          </SelectTrigger>
          <SelectContent>
            {availableColumns.map((col) => (
              <SelectItem key={col.displayName} value={col.displayName}>
                {col.displayName}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

interface FilterSelectorProps {
  filter: FilterCondition
  isFirst: boolean
  availableColumns: Array<{ table: string; column: string; type: string; displayName: string }>
  onChange: (updates: Partial<FilterCondition>) => void
  onRemove: () => void
  disabled?: boolean
}

function FilterSelector({ filter, isFirst, availableColumns, onChange, onRemove, disabled }: FilterSelectorProps) {
  const needsValue = !['IS NULL', 'IS NOT NULL'].includes(filter.operator)
  const needsTwoValues = filter.operator === 'BETWEEN'

  return (
    <div className="flex flex-wrap items-center gap-2">
      {!isFirst && (
        <Select
          value={filter.combinator || 'AND'}
          onValueChange={(val) => onChange({ combinator: val as FilterCombinator })}
          disabled={disabled}
        >
          <SelectTrigger className="w-24">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="AND">AND</SelectItem>
            <SelectItem value="OR">OR</SelectItem>
          </SelectContent>
        </Select>
      )}

      <Select
        value={filter.column}
        onValueChange={(column) => onChange({ column })}
        disabled={disabled}
        className={cn(!isFirst ? 'flex-1' : '')}
      >
        <SelectTrigger className={cn('min-w-[200px]', !isFirst && 'flex-1')}>
          <SelectValue placeholder="Select column" />
        </SelectTrigger>
        <SelectContent>
          {availableColumns.map((col) => (
            <SelectItem key={col.displayName} value={col.displayName}>
              {col.displayName}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select value={filter.operator} onValueChange={(op) => onChange({ operator: op as FilterOperator })} disabled={disabled}>
        <SelectTrigger className="w-40">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="=">=</SelectItem>
          <SelectItem value="!=">!=</SelectItem>
          <SelectItem value=">">{'>'}</SelectItem>
          <SelectItem value="<">{'<'}</SelectItem>
          <SelectItem value=">=">{'>='}</SelectItem>
          <SelectItem value="<=">{'<='}</SelectItem>
          <SelectItem value="LIKE">contains</SelectItem>
          <SelectItem value="NOT LIKE">doesn't contain</SelectItem>
          <SelectItem value="IN">in list</SelectItem>
          <SelectItem value="NOT IN">not in list</SelectItem>
          <SelectItem value="IS NULL">is empty</SelectItem>
          <SelectItem value="IS NOT NULL">is not empty</SelectItem>
          <SelectItem value="BETWEEN">between</SelectItem>
        </SelectContent>
      </Select>

      {needsValue && (
        <Input
          placeholder="Value"
          value={(filter.value as string) || ''}
          onChange={(e) => onChange({ value: e.target.value })}
          disabled={disabled}
          className="flex-1 min-w-[120px]"
        />
      )}

      {needsTwoValues && (
        <>
          <span className="text-sm text-muted-foreground">and</span>
          <Input
            placeholder="To value"
            value={(filter.valueTo as string) || ''}
            onChange={(e) => onChange({ valueTo: e.target.value })}
            disabled={disabled}
            className="flex-1 min-w-[120px]"
          />
        </>
      )}

      <Button variant="ghost" size="icon" onClick={onRemove} disabled={disabled}>
        <X className="h-4 w-4" />
      </Button>
    </div>
  )
}

interface OrderBySelectorProps {
  orderBy: OrderByClause
  availableColumns: Array<{ table: string; column: string; type: string; displayName: string }>
  onChange: (updates: Partial<OrderByClause>) => void
  onRemove: () => void
  disabled?: boolean
}

function OrderBySelector({ orderBy, availableColumns, onChange, onRemove, disabled }: OrderBySelectorProps) {
  return (
    <div className="flex gap-2">
      <Select value={orderBy.column} onValueChange={(column) => onChange({ column })} disabled={disabled}>
        <SelectTrigger className="flex-1">
          <SelectValue placeholder="Select column" />
        </SelectTrigger>
        <SelectContent>
          {availableColumns.map((col) => (
            <SelectItem key={col.displayName} value={col.displayName}>
              {col.displayName}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <ToggleGroup type="single" value={orderBy.direction} onValueChange={(val) => onChange({ direction: val as SortDirection })}>
        <ToggleGroupItem value="ASC" aria-label="Ascending" disabled={disabled}>
          <ArrowUp className="h-4 w-4" />
        </ToggleGroupItem>
        <ToggleGroupItem value="DESC" aria-label="Descending" disabled={disabled}>
          <ArrowDown className="h-4 w-4" />
        </ToggleGroupItem>
      </ToggleGroup>

      <Button variant="ghost" size="icon" onClick={onRemove} disabled={disabled}>
        <Trash2 className="h-4 w-4" />
      </Button>
    </div>
  )
}

// ===== Utility Functions =====

async function fetchDatabaseSchema(connectionId: string): Promise<DatabaseSchema> {
  // TODO: Replace with actual API call
  const response = await fetch(`/api/connections/${connectionId}/schema`)
  if (!response.ok) {
    throw new Error('Failed to fetch schema')
  }
  return response.json()
}

async function runPreviewQuery(connectionId: string, sql: string): Promise<QueryPreview> {
  // TODO: Replace with actual API call
  const response = await fetch(`/api/connections/${connectionId}/preview`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sql, limit: 10 }),
  })
  if (!response.ok) {
    throw new Error('Preview query failed')
  }
  return response.json()
}

function validateQuery(state: QueryBuilderState, schema: DatabaseSchema | null): QueryValidationError[] {
  const errors: QueryValidationError[] = []

  // Must have table selected
  if (!state.table) {
    errors.push({ field: 'table', message: 'Please select a table', severity: 'error' })
    return errors
  }

  // Must have at least one column
  if (state.columns.length === 0) {
    errors.push({ field: 'columns', message: 'Please select at least one column', severity: 'error' })
  }

  // All columns must be selected
  state.columns.forEach((col, idx) => {
    if (!col.column) {
      errors.push({ field: `column-${idx}`, message: `Column ${idx + 1} not selected`, severity: 'error' })
    }
  })

  // If using aggregations, must have GROUP BY (unless aggregating everything)
  const hasAggregations = state.columns.some((c) => c.aggregation)
  const nonAggregatedColumns = state.columns.filter((c) => !c.aggregation).map((c) => `${c.table}.${c.column}`)

  if (hasAggregations && nonAggregatedColumns.length > 0 && state.groupBy.length === 0) {
    errors.push({
      field: 'groupBy',
      message: 'When using aggregations, non-aggregated columns must be in GROUP BY',
      severity: 'error',
    })
  }

  // All filters must have column and value (if needed)
  state.filters.forEach((filter, idx) => {
    if (!filter.column) {
      errors.push({ field: `filter-${idx}`, message: `Filter ${idx + 1} missing column`, severity: 'error' })
    }
    const needsValue = !['IS NULL', 'IS NOT NULL'].includes(filter.operator)
    if (needsValue && !filter.value) {
      errors.push({ field: `filter-${idx}`, message: `Filter ${idx + 1} missing value`, severity: 'error' })
    }
  })

  // All joins must have table and ON conditions
  state.joins.forEach((join, idx) => {
    if (!join.table) {
      errors.push({ field: `join-${idx}`, message: `Join ${idx + 1} missing table`, severity: 'error' })
    }
    if (!join.on.left || !join.on.right) {
      errors.push({ field: `join-${idx}`, message: `Join ${idx + 1} missing ON conditions`, severity: 'error' })
    }
  })

  // All ORDER BY must have column
  state.orderBy.forEach((order, idx) => {
    if (!order.column) {
      errors.push({ field: `orderBy-${idx}`, message: `Sort ${idx + 1} missing column`, severity: 'error' })
    }
  })

  return errors
}

function generateSQL(state: QueryBuilderState): string {
  // This is a simplified SQL generator - backend will do the real work
  const parts: string[] = []

  // SELECT clause
  const selectCols = state.columns.map((col) => {
    let expr = `${col.table}.${col.column}`
    if (col.aggregation) {
      if (col.aggregation === 'count_distinct') {
        expr = `COUNT(DISTINCT ${expr})`
      } else {
        expr = `${col.aggregation.toUpperCase()}(${expr})`
      }
    }
    return col.alias ? `${expr} AS ${col.alias}` : expr
  })
  parts.push(`SELECT ${selectCols.join(', ')}`)

  // FROM clause
  parts.push(`FROM ${state.table}`)

  // JOIN clauses
  state.joins.forEach((join) => {
    const alias = join.alias ? ` ${join.alias}` : ''
    parts.push(`${join.type} JOIN ${join.table}${alias} ON ${join.on.left} = ${join.on.right}`)
  })

  // WHERE clause
  if (state.filters.length > 0) {
    const whereClauses = state.filters.map((filter, idx) => {
      const prefix = idx === 0 ? 'WHERE' : filter.combinator || 'AND'
      let condition = `${filter.column} ${filter.operator}`

      if (!['IS NULL', 'IS NOT NULL'].includes(filter.operator)) {
        if (filter.operator === 'LIKE' || filter.operator === 'NOT LIKE') {
          condition += ` '%${filter.value}%'`
        } else if (filter.operator === 'BETWEEN') {
          condition += ` ${filter.value} AND ${filter.valueTo}`
        } else if (filter.operator === 'IN' || filter.operator === 'NOT IN') {
          condition += ` (${filter.value})`
        } else {
          condition += ` '${filter.value}'`
        }
      }

      return `${prefix} ${condition}`
    })
    parts.push(whereClauses.join(' '))
  }

  // GROUP BY clause
  if (state.groupBy.length > 0) {
    parts.push(`GROUP BY ${state.groupBy.join(', ')}`)
  }

  // ORDER BY clause
  if (state.orderBy.length > 0) {
    const orderClauses = state.orderBy.map((order) => `${order.column} ${order.direction}`)
    parts.push(`ORDER BY ${orderClauses.join(', ')}`)
  }

  // LIMIT clause
  if (state.limit) {
    parts.push(`LIMIT ${state.limit}`)
  }

  return parts.join('\n')
}
