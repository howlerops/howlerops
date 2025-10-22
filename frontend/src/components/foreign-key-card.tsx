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
  Table,
  Key
} from 'lucide-react'
import { QueryEditableMetadata } from '@/store/query-store'
import { CellValue } from '@/types/table'
import { wailsEndpoints } from '@/lib/wails-api'

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
  key: string
  value: CellValue
  metadata: QueryEditableMetadata
  connectionId: string
  isExpanded: boolean
  onToggle: (key: string) => void
  onLoadData: (key: string) => Promise<void>
}

export function ForeignKeyCard({
  key: fieldKey,
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

  const handleToggle = useCallback(async () => {
    onToggle(fieldKey)
    
    if (!isExpanded && !foreignKeyData && foreignKeyInfo) {
      await loadForeignKeyData()
    }
  }, [fieldKey, isExpanded, foreignKeyData, foreignKeyInfo, onToggle])

  const loadForeignKeyData = useCallback(async () => {
    if (!foreignKeyInfo || !connectionId || isLoading) return

    setIsLoading(true)
    setError(null)

    try {
      await onLoadData(fieldKey)
      
      // Build query to fetch related records
      const escapedValue = typeof value === 'string' ? `'${value.replace(/'/g, "''")}'` : String(value)
      const query = `SELECT * FROM ${foreignKeyInfo.schema ? `"${foreignKeyInfo.schema}"."${foreignKeyInfo.tableName}"` : `"${foreignKeyInfo.tableName}"`} WHERE "${foreignKeyInfo.columnName}" = ${escapedValue} LIMIT 10`
      
      // Execute query to get related records
      const response = await wailsEndpoints.queries.execute(connectionId, query, {
        limit: 10
      })

      if (!response.success || response.message) {
        throw new Error(response.message || 'Query execution failed')
      }

      setForeignKeyData({
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        schema: foreignKeyInfo.schema,
        relatedRows: (response.data.rows || []).map((row: any[]) => {
          const record: ForeignKeyRecord = {}
          response.data.columns.forEach((col: string, index: number) => {
            record[col] = row[index]
          })
          return record
        }),
        loading: false,
        totalCount: response.data.rowCount
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load foreign key data'
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
  }, [foreignKeyInfo, connectionId, isLoading, fieldKey, onLoadData, value])

  // Don't render if no foreign key info
  if (!foreignKeyInfo) return null

  const hasData = foreignKeyData?.relatedRows && foreignKeyData.relatedRows.length > 0
  const showExpanded = isExpanded && (isLoading || hasData || error)

  return (
    <div className="border border-border rounded-lg bg-muted/30">
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
        <div className="border-t border-border">
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
            <div className="p-3 space-y-3">
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
                <div className="space-y-2">
                  {foreignKeyData.relatedRows.map((row, index) => (
                    <Card key={index} className="bg-background/50">
                      <CardContent className="p-3">
                        <div className="space-y-2">
                          {Object.entries(row).map(([key, value]) => (
                            <div key={key} className="flex items-center justify-between text-xs">
                              <span className="font-medium text-muted-foreground truncate flex-1 mr-2">
                                {key}
                              </span>
                              <span className="text-right font-mono truncate max-w-[200px]">
                                {formatValue(value)}
                              </span>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
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
    <div className="space-y-3">
      <div className="flex items-center gap-2 text-sm font-medium">
        <Database className="h-4 w-4" />
        Foreign Key Relationships
        <Badge variant="secondary" className="text-xs">
          {foreignKeys.length}
        </Badge>
      </div>
      
      <div className="space-y-2">
        {foreignKeys.map(({ key, value, metadata }) => (
          <ForeignKeyCard
            key={key}
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
