import React, { useState, useCallback, useMemo } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Database,
  Loader2,
  AlertCircle,
  Eye,
  Key
} from 'lucide-react'
import { QueryEditableMetadata } from '@/store/query-store'
import { CellValue } from '@/types/table'
import { wailsEndpoints } from '@/lib/wails-api'
import { toast } from '@/hooks/use-toast'
import { useConnectionStore } from '@/store/connection-store'

interface ForeignKeyRecord {
  [key: string]: CellValue
}

interface ForeignKeyData {
  tableName: string
  columnName: string
  schema?: string
  relatedRows: ForeignKeyRecord[]
  loading: boolean
  error?: string
  totalCount?: number
}

interface ForeignKeyCardProps {
  fieldKey: string
  value: CellValue
  metadata: QueryEditableMetadata
  connectionId: string
  isExpanded: boolean
  onToggle: (key: string) => void
  onLoadData: (key: string) => Promise<void>
}

export function ForeignKeyCard({
  fieldKey,
  value,
  metadata,
  connectionId,
  isExpanded,
  onToggle,
  onLoadData
}: ForeignKeyCardProps) {
  const [foreignKeyData, setForeignKeyData] = useState<ForeignKeyData | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Get current connections to find an active one
  const { connections } = useConnectionStore()

  // Find foreign key metadata for this column
  const foreignKeyInfo = useMemo(() => {
    if (!metadata?.columns) return null

    const column = metadata.columns.find(col => 
      (col.name || col.resultName)?.toLowerCase() === fieldKey.toLowerCase()
    )

    if (!column?.foreignKey) return null

    return {
      tableName: column.foreignKey.table,
      columnName: column.foreignKey.column,
      schema: column.foreignKey.schema
    }
  }, [fieldKey, metadata])

  const loadForeignKeyData = useCallback(async () => {
    if (!foreignKeyInfo || isLoading) return

    // Get the first connected connection directly from the store
    const activeConnection = connections.find(c => c.isConnected)

    if (!activeConnection || !activeConnection.sessionId) {
      console.error('[FK] No active connection found. Connections:', connections)
      toast({
        title: 'No Active Connection',
        description: 'Please connect to a database to explore foreign key relationships.',
        variant: 'destructive'
      })
      return
    }

    const actualConnectionId = activeConnection.sessionId
    console.log('[FK] Using active connection:', {
      connectionName: activeConnection.name,
      sessionId: actualConnectionId,
      propConnectionId: connectionId,
      fieldKey,
      foreignKeyInfo,
      value
    })

    setIsLoading(true)
    setError(null)

    try {
      await onLoadData(fieldKey)

      // Build query to fetch related records
      const escapedValue = typeof value === 'string' ? `'${value.replace(/'/g, "''")}'` : String(value)
      const query = `SELECT * FROM ${foreignKeyInfo.schema ? `"${foreignKeyInfo.schema}"."${foreignKeyInfo.tableName}"` : `"${foreignKeyInfo.tableName}"`} WHERE "${foreignKeyInfo.columnName}" = ${escapedValue} LIMIT 10`

      console.log('[FK] Executing query:', query)

      // Execute query with the ACTUAL active connection, not the prop
      const response = await wailsEndpoints.queries.execute(actualConnectionId, query, {
        limit: 10
      })

      console.log('[FK] Query response:', response)
      console.log('[FK] Response data structure:', {
        hasData: !!response.data,
        columns: response.data?.columns,
        rowsLength: response.data?.rows?.length,
        firstRow: response.data?.rows?.[0],
        rowCount: response.data?.rowCount
      })

      if (!response.success || response.message) {
        throw new Error(response.message || 'Query execution failed')
      }

      const relatedRows = (response.data.rows || []).map((row: unknown): ForeignKeyRecord => {
        const cells = Array.isArray(row) ? row : ([] as unknown[])
        const record: ForeignKeyRecord = {}
        response.data.columns.forEach((col: string, index: number) => {
          record[col] = cells[index] as CellValue
        })
        return record
      })

      console.log('[FK] Mapped relatedRows:', relatedRows)

      const newFKData = {
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        schema: foreignKeyInfo.schema,
        relatedRows,
        loading: false,
        totalCount: response.data.rowCount
      }

      console.log('[FK] Setting foreignKeyData state to:', newFKData)
      setForeignKeyData(newFKData)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load foreign key data'
      console.error('[FK] Error loading foreign key data:', err)

      // Check if it's a connection error
      if (errorMessage.includes('connection not found')) {
        toast({
          title: 'Connection Error',
          description: 'The connection used for this query is no longer available. Please re-run your query to use the current active connection.',
          variant: 'destructive'
        })
      }

      setError(errorMessage)
      setForeignKeyData({
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        schema: foreignKeyInfo.schema,
        relatedRows: [],
        loading: false,
        error: errorMessage
      })
    } finally {
      setIsLoading(false)
    }
  }, [foreignKeyInfo, connections, connectionId, isLoading, fieldKey, onLoadData, value])

  const handleToggle = useCallback(async () => {
    onToggle(fieldKey)

    if (!isExpanded && !foreignKeyData && foreignKeyInfo) {
      await loadForeignKeyData()
    }
  }, [fieldKey, foreignKeyData, foreignKeyInfo, isExpanded, loadForeignKeyData, onToggle])

  // Don't render if no foreign key info
  if (!foreignKeyInfo) return null

  const hasData = foreignKeyData?.relatedRows && foreignKeyData.relatedRows.length > 0
  const showExpanded = isExpanded && (isLoading || hasData || error)

  return (
    <div className="border border-border rounded-lg bg-muted/30 overflow-hidden w-full">
      {/* Header */}
      <div 
        className="flex items-center justify-between p-3 cursor-pointer hover:bg-muted/50 transition-colors"
        onClick={handleToggle}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <div className="flex items-center gap-1">
            {isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
            <Key className="h-3 w-3 text-blue-500" />
          </div>
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-sm truncate">{fieldKey}</span>
              <Badge variant="outline" className="text-xs">
                FK
              </Badge>
            </div>
            <div className="text-xs text-muted-foreground truncate">
              → {foreignKeyInfo.schema ? `${foreignKeyInfo.schema}.` : ''}{foreignKeyInfo.tableName}.{foreignKeyInfo.columnName}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          {isLoading && <Loader2 className="h-3 w-3 animate-spin" />}
          {foreignKeyData && (
            <span>
              {foreignKeyData.totalCount || foreignKeyData.relatedRows.length} record{(foreignKeyData.totalCount || foreignKeyData.relatedRows.length) !== 1 ? 's' : ''}
            </span>
          )}
        </div>
      </div>

      {/* Expanded content */}
      {showExpanded && (
        <div className="border-t border-border overflow-hidden">
          {isLoading && (
            <div className="flex items-center gap-2 p-3 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading related records...
            </div>
          )}

          {error && (
            <div className="p-3">
              <div className="flex items-start gap-2 text-sm text-destructive bg-destructive/10 p-2 rounded">
                <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                <div>
                  <div className="font-medium">Error loading related data</div>
                  <div className="text-xs mt-1">{error}</div>
                </div>
              </div>
            </div>
          )}

          {hasData && (
            <div className="p-3 flex flex-col gap-3 overflow-hidden">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm font-medium">
                  <ExternalLink className="h-4 w-4" />
                  Related Records
                  {foreignKeyData.totalCount && foreignKeyData.totalCount > foreignKeyData.relatedRows.length && (
                    <Badge variant="secondary" className="text-xs">
                      Showing {foreignKeyData.relatedRows.length} of {foreignKeyData.totalCount}
                    </Badge>
                  )}
                </div>
                <Button variant="ghost" size="sm" className="h-6 px-2 text-xs">
                  <Eye className="h-3 w-3 mr-1" />
                  View All
                </Button>
              </div>

              <ScrollArea className="max-h-64">
                <div className="flex flex-col gap-2 pr-4 overflow-hidden">
                  {foreignKeyData.relatedRows.map((row, index) => {
                    console.log('[FK] Rendering row:', row)
                    return (
                      <Card key={index} className="bg-background/50 overflow-hidden">
                        <CardContent className="p-3">
                          <div className="flex flex-col gap-2">
                            {Object.entries(row).map(([key, value]) => {
                              console.log('[FK] Rendering field:', key, '=', value)
                              return (
                              <div key={key} className="flex items-start gap-2 text-xs overflow-hidden">
                                <span className="font-medium text-muted-foreground flex-shrink-0 max-w-[40%] truncate">
                                  {key}:
                                </span>
                                <span className="font-mono text-foreground break-words overflow-hidden flex-1">
                                  {formatValue(value)}
                                </span>
                              </div>
                              )
                            })}
                          </div>
                        </CardContent>
                      </Card>
                    )
                  })}
                </div>
              </ScrollArea>
            </div>
          )}

          {!isLoading && !error && !hasData && (
            <div className="p-3 text-sm text-muted-foreground text-center">
              No related records found
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function formatValue(value: CellValue): string {
  if (value === null) return 'NULL'
  if (value === undefined) return 'undefined'
  if (typeof value === 'string') {
    // Truncate long strings
    return value.length > 50 ? value.substring(0, 50) + '...' : value
  }
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

/**
 * Component to display multiple foreign key relationships
 */
interface ForeignKeySectionProps {
  foreignKeys: Array<{
    key: string
    value: CellValue
    metadata: QueryEditableMetadata
  }>
  connectionId: string
  expandedKeys: Set<string>
  onToggleKey: (key: string) => void
  onLoadData: (key: string) => Promise<void>
}

export function ForeignKeySection({
  foreignKeys,
  connectionId,
  expandedKeys,
  onToggleKey,
  onLoadData
}: ForeignKeySectionProps) {
  if (foreignKeys.length === 0) return null

  return (
    <div className="flex flex-col gap-3 w-full">
      <div className="flex items-center gap-2 text-sm font-medium">
        <Database className="h-4 w-4" />
        Foreign Key Relationships
        <Badge variant="secondary" className="text-xs">
          {foreignKeys.length}
        </Badge>
      </div>

      <div className="flex flex-col gap-2">
        {foreignKeys.map(({ key, value, metadata }) => (
          <ForeignKeyCard
            key={key}
            fieldKey={key}
            value={value}
            metadata={metadata}
            connectionId={connectionId}
            isExpanded={expandedKeys.has(key)}
            onToggle={onToggleKey}
            onLoadData={onLoadData}
          />
        ))}
      </div>
    </div>
  )
}
