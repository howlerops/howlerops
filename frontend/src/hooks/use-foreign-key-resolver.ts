import { useCallback,useState } from 'react'

import { executeQueryByConnectionId } from '@/lib/query-engine/runtime'
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

      const result = await executeQueryByConnectionId(connectionId, query, { limit: 10 })

      const data: ForeignKeyData = {
        tableName: foreignKeyInfo.tableName,
        columnName: foreignKeyInfo.columnName,
        relatedRows: (result.rows || []).map((row) => {
          const record: Record<string, CellValue> = {}
          result.columns.forEach((col) => {
            record[col] = row[col] as CellValue
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

