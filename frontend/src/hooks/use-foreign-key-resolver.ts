import { useState, useCallback } from 'react'
import { wailsEndpoints } from '@/lib/wails-api'
import type { CellValue } from '@/types/table'

interface ForeignKeyData {
  tableName: string
  columnName: string
  relatedRows: Record<string, CellValue>[]
  loading: boolean
  error?: string
}

export function useForeignKeyResolver() {
  const [cache, setCache] = useState<Map<string, ForeignKeyData>>(new Map())

  const loadForeignKeyData = useCallback(async (
    key: string,
    connectionId: string,
    foreignKeyInfo: { tableName: string; columnName: string; schema?: string },
    value: CellValue
  ): Promise<ForeignKeyData | null> => {
    const cacheKey = `${connectionId}:${key}:${value}`

    if (cache.has(cacheKey)) {
      return cache.get(cacheKey)!
    }

    try {
      const tableName = foreignKeyInfo.schema
        ? `"${foreignKeyInfo.schema}"."${foreignKeyInfo.tableName}"`
        : `"${foreignKeyInfo.tableName}"`

      const escapedValue = typeof value === 'string' ? `'${value.replace(/'/g, "''")}'` : String(value)
      const query = `SELECT * FROM ${tableName} WHERE "${foreignKeyInfo.columnName}" = ${escapedValue} LIMIT 10`

      const response = await wailsEndpoints.queries.execute(connectionId, query, {
        limit: 10,
      })

      if (!response.success || response.message) {
        throw new Error(response.message || 'Query execution failed')
      }

      const data: ForeignKeyData = {
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        relatedRows: (response.data.rows || []).map((row: CellValue[]) => {
          const record: Record<string, CellValue> = {}
          response.data.columns.forEach((col: string, index: number) => {
            record[col] = row[index]
          })
          return record
        }),
        loading: false,
      }

      setCache((prev) => new Map(prev).set(cacheKey, data))
      return data
    } catch (error) {
      console.error('Failed to load foreign key data:', error)
      const errorData: ForeignKeyData = {
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        relatedRows: [],
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load foreign key data',
      }
      setCache((prev) => new Map(prev).set(cacheKey, errorData))
      return errorData
    }
  }, [cache])

  const clearCache = useCallback(() => setCache(new Map()), [])

  return { loadForeignKeyData, clearCache }
}


