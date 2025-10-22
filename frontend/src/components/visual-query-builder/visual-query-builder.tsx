/**
 * Main Visual Query Builder Component
 * Orchestrates all sub-components for building queries visually
 */

import { useState, useEffect, useCallback } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Database, Table, Filter, ArrowUpDown, Code, AlertCircle, Link } from 'lucide-react'
import { VisualQueryBuilderProps, VisualQueryState, TableInfo, ColumnInfo } from './types'
import { QueryIR, TableRef, SelectItem, OrderBy } from '@/lib/query-ir'
import { createMultiConnectionExecutor, MergedResult } from '@/lib/multi-connection-executor'
import { SourcePicker } from './source-picker'
import { ColumnPicker } from './column-picker'
import { FilterEditor } from './filter-editor'
import { JoinBuilder } from './join-builder'
import { SortLimit } from './sort-limit'
import { SqlPreview } from './sql-preview'

export function VisualQueryBuilder({
  connections,
  schemas,
  onQueryChange,
  onSQLChange,
  initialQuery
}: VisualQueryBuilderProps) {
  const [queryState, setQueryState] = useState<VisualQueryState>({
    connections: [],
    from: null,
    joins: [],
    select: [],
    where: undefined,
    orderBy: [],
    limit: undefined,
    offset: undefined
  })

  const [tableColumns, setTableColumns] = useState<ColumnInfo[]>([])
  const [activeTab, setActiveTab] = useState('source')
  const [manualSQL, setManualSQL] = useState('')
  const [isExecuting, setIsExecuting] = useState(false)
  const [executionResult, setExecutionResult] = useState<MergedResult | null>(null)

  // Initialize from initial query
  useEffect(() => {
    if (initialQuery) {
      setQueryState({
        connections: [], // Will be set by connection selection
        from: initialQuery.from,
        joins: (initialQuery.joins || []).map((join, index) => ({
          ...join,
          id: `join-${index}`
        })),
        select: initialQuery.select,
        where: initialQuery.where,
        orderBy: initialQuery.orderBy || [],
        limit: initialQuery.limit,
        offset: initialQuery.offset
      })
    }
  }, [initialQuery])

  // Load table columns from schema
  const loadTableColumns = useCallback(async (table: TableRef) => {
    // Find table in schemas
    let foundTable: TableInfo | null = null
    
    for (const connectionSchemas of schemas.values()) {
      for (const schema of connectionSchemas) {
        if (schema.name === table.schema) {
          const tableInfo = schema.tables.find(t => t.name === table.table)
          if (tableInfo) {
            foundTable = tableInfo
            break
          }
        }
      }
      if (foundTable) break
    }

    if (foundTable) {
      setTableColumns(foundTable.columns)
    } else {
      setTableColumns([])
    }
  }, [schemas])

  // Load table columns when table changes
  useEffect(() => {
    if (queryState.from) {
      loadTableColumns(queryState.from)
    } else {
      setTableColumns([])
    }
  }, [queryState.from, loadTableColumns])

  // Handle connection changes
  const handleConnectionsChange = (connectionIds: string[]) => {
    setQueryState(prev => ({
      ...prev,
      connections: connectionIds
    }))
  }

  // Handle table changes
  const handleTableChange = (table: TableRef | null) => {
    setQueryState(prev => ({
      ...prev,
      from: table,
      select: [], // Clear selections when table changes
      where: undefined,
      orderBy: [],
      limit: undefined,
      offset: undefined
    }))
  }

  // Handle column changes
  const handleColumnsChange = (columns: SelectItem[]) => {
    setQueryState(prev => ({
      ...prev,
      select: columns
    }))
  }

  // Handle where clause changes
  const handleWhereChange = (where: QueryIR['where']) => {
    setQueryState(prev => ({
      ...prev,
      where
    }))
  }

  // Handle order by changes
  const handleOrderByChange = (orderBy: OrderBy[]) => {
    setQueryState(prev => ({
      ...prev,
      orderBy
    }))
  }

  // Handle limit changes
  const handleLimitChange = (limit?: number) => {
    setQueryState(prev => ({
      ...prev,
      limit
    }))
  }

  // Handle offset changes
  const handleOffsetChange = (offset?: number) => {
    setQueryState(prev => ({
      ...prev,
      offset
    }))
  }

  // Handle manual SQL changes
  const handleSQLChange = (sql: string) => {
    setManualSQL(sql)
    if (onSQLChange) {
      onSQLChange(sql)
    }
  }

  // Handle query execution
  const handleExecuteQuery = async () => {
    if (!queryState.from || queryState.select.length === 0 || queryState.connections.length === 0) {
      return
    }

    setIsExecuting(true)
    setExecutionResult(null)

    try {
      const queryIR = generateQueryIR()
      if (!queryIR) {
        throw new Error('Invalid query configuration')
      }

      const executor = createMultiConnectionExecutor(connections)
      const result = await executor.executeQuery(queryIR, queryState.connections, {
        dialect: 'postgres', // TODO: Get from connection
        addProvenance: queryState.connections.length > 1
      })

      setExecutionResult(result)
    } catch (error) {
      console.error('Query execution failed:', error)
      // TODO: Show error to user
    } finally {
      setIsExecuting(false)
    }
  }

  // Generate QueryIR from current state
  const generateQueryIR = useCallback((): QueryIR | null => {
    if (!queryState.from || queryState.select.length === 0) {
      return null
    }

    return {
      from: queryState.from,
      joins: queryState.joins.map(join => ({
        type: join.type,
        table: join.table,
        on: join.on
      })),
      select: queryState.select,
      where: queryState.where,
      orderBy: queryState.orderBy,
      limit: queryState.limit,
      offset: queryState.offset
    }
  }, [queryState])

  // Notify parent of query changes
  useEffect(() => {
    const queryIR = generateQueryIR()
    if (queryIR) {
      onQueryChange(queryIR)
    }
  }, [queryState, generateQueryIR, onQueryChange])

  // Check if query is valid
  const isQueryValid = queryState.from && queryState.select.length > 0

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Visual Query Builder</h2>
        <div className="flex items-center space-x-2">
          {isQueryValid && (
            <Button
              onClick={handleExecuteQuery}
              disabled={isExecuting}
              className="flex items-center space-x-2"
            >
              {isExecuting ? (
                <>
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  <span>Executing...</span>
                </>
              ) : (
                <>
                  <Database className="w-4 h-4" />
                  <span>Execute Query</span>
                </>
              )}
            </Button>
          )}
          {!isQueryValid && (
            <Alert className="w-auto">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                Select a table and columns to build your query
              </AlertDescription>
            </Alert>
          )}
        </div>
      </div>

      {/* Main Content */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="source" className="flex items-center space-x-1">
            <Database className="w-4 h-4" />
            <span>Source</span>
          </TabsTrigger>
          <TabsTrigger value="columns" className="flex items-center space-x-1">
            <Table className="w-4 h-4" />
            <span>Columns</span>
          </TabsTrigger>
          <TabsTrigger value="joins" className="flex items-center space-x-1">
            <Link className="w-4 h-4" />
            <span>Joins</span>
          </TabsTrigger>
          <TabsTrigger value="filters" className="flex items-center space-x-1">
            <Filter className="w-4 h-4" />
            <span>Filters</span>
          </TabsTrigger>
          <TabsTrigger value="sort" className="flex items-center space-x-1">
            <ArrowUpDown className="w-4 h-4" />
            <span>Sort & Limit</span>
          </TabsTrigger>
          <TabsTrigger value="sql" className="flex items-center space-x-1">
            <Code className="w-4 h-4" />
            <span>SQL</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="source" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Data Source</CardTitle>
            </CardHeader>
            <CardContent>
              <SourcePicker
                connections={connections}
                schemas={schemas}
                selectedConnections={queryState.connections}
                selectedTable={queryState.from}
                onConnectionsChange={handleConnectionsChange}
                onTableChange={handleTableChange}
                onSchemaLoad={async (connectionId) => {
                  // This would typically load schema for the connection
                  console.log('Loading schema for connection:', connectionId)
                }}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="columns" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Column Selection</CardTitle>
            </CardHeader>
            <CardContent>
              <ColumnPicker
                table={queryState.from}
                columns={tableColumns}
                selectedColumns={queryState.select}
                onColumnsChange={handleColumnsChange}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="joins" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Table Joins</CardTitle>
            </CardHeader>
            <CardContent>
              <JoinBuilder
                availableTables={[]} // TODO: Get available tables from schema
                existingJoins={queryState.joins}
                onJoinsChange={(joins) => setQueryState(prev => ({ ...prev, joins }))}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="filters" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Filter Conditions</CardTitle>
            </CardHeader>
            <CardContent>
              <FilterEditor
                columns={tableColumns}
                where={queryState.where}
                onWhereChange={handleWhereChange}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="sort" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Sorting & Limits</CardTitle>
            </CardHeader>
            <CardContent>
              <SortLimit
                columns={tableColumns}
                orderBy={queryState.orderBy}
                limit={queryState.limit}
                offset={queryState.offset}
                onOrderByChange={handleOrderByChange}
                onLimitChange={handleLimitChange}
                onOffsetChange={handleOffsetChange}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="sql" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">SQL Preview</CardTitle>
            </CardHeader>
            <CardContent>
              {isQueryValid ? (
                <SqlPreview
                  queryIR={generateQueryIR()!}
                  dialect="postgres" // TODO: Get from connection
                  manualSQL={manualSQL}
                  onSQLChange={handleSQLChange}
                />
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Code className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p>Complete the query setup to see generated SQL</p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Query Summary */}
      {isQueryValid && (
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">Query Summary</div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-xs text-muted-foreground">
              <div>
                <div className="font-medium">Table</div>
                <div>{queryState.from?.schema}.{queryState.from?.table}</div>
              </div>
              <div>
                <div className="font-medium">Columns</div>
                <div>{queryState.select.length} selected</div>
              </div>
              <div>
                <div className="font-medium">Filters</div>
                <div>{queryState.where ? 'Applied' : 'None'}</div>
              </div>
              <div>
                <div className="font-medium">Sort</div>
                <div>{queryState.orderBy.length} column(s)</div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Execution Results */}
      {executionResult && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Query Results</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-4 text-sm">
                <div>
                  <div className="font-medium">Total Rows</div>
                  <div>{executionResult.rowCount.toLocaleString()}</div>
                </div>
                <div>
                  <div className="font-medium">Execution Time</div>
                  <div>{executionResult.totalExecutionTime}ms</div>
                </div>
                <div>
                  <div className="font-medium">Connections</div>
                  <div>{executionResult.connectionResults.length}</div>
                </div>
              </div>

              {/* Connection Results */}
              <div>
                <div className="text-sm font-medium mb-2">Connection Results</div>
                <div className="space-y-2">
                  {executionResult.connectionResults.map((result, index) => (
                    <div key={index} className="flex items-center justify-between p-2 bg-muted rounded">
                      <div className="flex items-center space-x-2">
                        <div className={`w-2 h-2 rounded-full ${result.success ? 'bg-green-500' : 'bg-red-500'}`} />
                        <span className="text-sm font-medium">{result.connectionName}</span>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {result.success ? (
                          `${result.data?.rowCount || 0} rows`
                        ) : (
                          result.error
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Sample Data Preview */}
              {executionResult.rows.length > 0 && (
                <div>
                  <div className="text-sm font-medium mb-2">Sample Data (first 5 rows)</div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-xs border-collapse">
                      <thead>
                        <tr className="border-b">
                          {executionResult.columns.map((col, index) => (
                            <th key={index} className="text-left p-2 font-medium">
                              {col}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {executionResult.rows.slice(0, 5).map((row, rowIndex) => (
                          <tr key={rowIndex} className="border-b">
                            {row.map((cell, cellIndex) => (
                              <td key={cellIndex} className="p-2">
                                {String(cell || '')}
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
