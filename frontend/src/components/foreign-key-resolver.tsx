import React, { useState, useCallback, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, ChevronRight, ChevronDown, Database, ExternalLink } from 'lucide-react'
import { QueryEditableMetadata } from '@/store/query-store'
import { CellValue } from '@/types/table'
import { formatJson } from '@/lib/json-formatter'
import { wailsEndpoints } from '@/lib/wails-api'

interface ForeignKeyResolverProps {
  key: string
  value: CellValue
  metadata?: QueryEditableMetadata | null
  connectionId?: string
  isExpanded: boolean
  onToggle: (key: string) => void
  onLoadData: (key: string) => Promise<void>
}

interface ForeignKeyData {
  tableName: string
  columnName: string
  relatedRows: Record<string, CellValue>[]
  loading: boolean
  error?: string
}

export function ForeignKeyResolver({
  key,
  value,
  metadata,
  connectionId,
  isExpanded,
  onToggle,
  onLoadData
}: ForeignKeyResolverProps) {
  const [foreignKeyData, setForeignKeyData] = useState<ForeignKeyData | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Find foreign key metadata for this column
  const foreignKeyInfo = useMemo(() => {
    if (!metadata?.columns) return null

    const column = metadata.columns.find(col => 
      (col.name || col.resultName)?.toLowerCase() === key.toLowerCase()
    )

    if (!column?.foreignKey) return null

    return {
      tableName: column.foreignKey.table,
      columnName: column.foreignKey.column,
      schema: column.foreignKey.schema
    }
  }, [key, metadata])

  const handleToggle = useCallback(() => {
    onToggle(key)
    
    if (!isExpanded && !foreignKeyData && foreignKeyInfo) {
      loadForeignKeyData()
    }
  }, [key, isExpanded, foreignKeyData, foreignKeyInfo, onToggle])

  const loadForeignKeyData = useCallback(async () => {
    if (!foreignKeyInfo || !connectionId || isLoading) return

    setIsLoading(true)
    setError(null)

    try {
      await onLoadData(key)
      
      // Build query to fetch related records
      const tableName = foreignKeyInfo.schema 
        ? `"${foreignKeyInfo.schema}"."${foreignKeyInfo.tableName}"`
        : `"${foreignKeyInfo.tableName}"`
      
      const escapedValue = typeof value === 'string' ? `'${value.replace(/'/g, "''")}'` : String(value)
      const query = `SELECT * FROM ${tableName} WHERE "${foreignKeyInfo.columnName}" = ${escapedValue} LIMIT 10`
      
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
        relatedRows: (response.data.rows || []).map((row: any[]) => {
          const record: Record<string, CellValue> = {}
          response.data.columns.forEach((col: string, index: number) => {
            record[col] = row[index]
          })
          return record
        }),
        loading: false
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load foreign key data'
      setError(errorMessage)
      setForeignKeyData({
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        relatedRows: [],
        loading: false,
        error: errorMessage
      })
    } finally {
      setIsLoading(false)
    }
  }, [foreignKeyInfo, connectionId, isLoading, key, onLoadData, value])

  // Don't render if no foreign key info
  if (!foreignKeyInfo) {
    return null
  }

  const hasData = foreignKeyData && foreignKeyData.relatedRows.length > 0
  const showExpanded = isExpanded && (hasData || isLoading || error)

  return (
    <div className="foreign-key-resolver">
      {/* Foreign key indicator */}
      <div className="flex items-center gap-2 py-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggle}
          className="h-6 w-6 p-0 hover:bg-muted"
          disabled={isLoading}
        >
          {isExpanded ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronRight className="h-3 w-3" />
          )}
        </Button>
        
        <Badge variant="outline" className="text-xs">
          <Database className="h-3 w-3 mr-1" />
          FK
        </Badge>
        
        <span className="text-sm text-muted-foreground">
          {foreignKeyInfo.tableName}.{foreignKeyInfo.columnName}
        </span>
        
        {isLoading && <Loader2 className="h-3 w-3 animate-spin" />}
      </div>

      {/* Expanded content */}
      {showExpanded && (
        <div className="ml-6 border-l-2 border-muted pl-4 space-y-2">
          {isLoading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              Loading related data...
            </div>
          )}

          {error && (
            <div className="text-sm text-destructive bg-destructive/10 p-2 rounded">
              Error: {error}
            </div>
          )}

          {hasData && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium">
                <ExternalLink className="h-3 w-3" />
                Related Records ({foreignKeyData.relatedRows.length})
              </div>
              
              <div className="space-y-1">
                {foreignKeyData.relatedRows.map((row, index) => (
                  <div key={index} className="bg-muted/50 p-2 rounded text-xs">
                    <div className="font-mono">
                      {formatJson(row).formatted}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {!isLoading && !error && !hasData && (
            <div className="text-sm text-muted-foreground">
              No related records found
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * Hook to manage foreign key data loading and caching
 */
export function useForeignKeyResolver() {
  const [cache, setCache] = useState<Map<string, ForeignKeyData>>(new Map())

  const loadForeignKeyData = useCallback(async (
    key: string,
    connectionId: string,
    foreignKeyInfo: { tableName: string; columnName: string; schema?: string },
    value: CellValue
  ): Promise<ForeignKeyData | null> => {
    const cacheKey = `${connectionId}:${key}:${value}`
    
    // Check cache first
    if (cache.has(cacheKey)) {
      return cache.get(cacheKey)!
    }

    try {
      // Build query to fetch related records
      const tableName = foreignKeyInfo.schema 
        ? `"${foreignKeyInfo.schema}"."${foreignKeyInfo.tableName}"`
        : `"${foreignKeyInfo.tableName}"`
      
      const escapedValue = typeof value === 'string' ? `'${value.replace(/'/g, "''")}'` : String(value)
      const query = `SELECT * FROM ${tableName} WHERE "${foreignKeyInfo.columnName}" = ${escapedValue} LIMIT 10`
      
      // Execute query to get related records
      const response = await wailsEndpoints.queries.execute(connectionId, query, {
        limit: 10
      })

      if (!response.success || response.message) {
        throw new Error(response.message || 'Query execution failed')
      }

      const data: ForeignKeyData = {
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        relatedRows: (response.data.rows || []).map((row: any[]) => {
          const record: Record<string, CellValue> = {}
          response.data.columns.forEach((col: string, index: number) => {
            record[col] = row[index]
          })
          return record
        }),
        loading: false
      }

      setCache(prev => new Map(prev).set(cacheKey, data))
      return data
    } catch (error) {
      console.error('Failed to load foreign key data:', error)
      const errorData: ForeignKeyData = {
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        relatedRows: [],
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load foreign key data'
      }
      setCache(prev => new Map(prev).set(cacheKey, errorData))
      return errorData
    }
  }, [cache])

  const clearCache = useCallback(() => {
    setCache(new Map())
  }, [])

  const getCachedData = useCallback((key: string, connectionId: string): ForeignKeyData | null => {
    const cacheKey = `${connectionId}:${key}`
    return cache.get(cacheKey) || null
  }, [cache])

  return {
    loadForeignKeyData,
    clearCache,
    getCachedData
  }
}

/**
 * Component to display foreign key relationships in a table
 */
interface ForeignKeyTableProps {
  foreignKeyData: ForeignKeyData
  onRowClick?: (row: Record<string, CellValue>) => void
}

export function ForeignKeyTable({ foreignKeyData, onRowClick }: ForeignKeyTableProps) {
  if (!foreignKeyData.relatedRows.length) {
    return (
      <div className="text-sm text-muted-foreground p-2">
        No related records found
      </div>
    )
  }

  const columns = Object.keys(foreignKeyData.relatedRows[0])

  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">
        {foreignKeyData.tableName} ({foreignKeyData.relatedRows.length} records)
      </div>
      
      <div className="overflow-x-auto">
        <table className="w-full text-xs border-collapse">
          <thead>
            <tr className="border-b bg-muted/50">
              {columns.map(column => (
                <th key={column} className="text-left p-1 font-medium">
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {foreignKeyData.relatedRows.map((row, index) => (
              <tr 
                key={index} 
                className="border-b hover:bg-muted/30 cursor-pointer"
                onClick={() => onRowClick?.(row)}
              >
                {columns.map(column => (
                  <td key={column} className="p-1">
                    {String(row[column] ?? '')}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
