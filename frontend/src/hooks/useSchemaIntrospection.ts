import { useEffect, useRef, useState, useCallback } from 'react'
import { useConnectionStore } from '@/store/connection-store'
import { wailsEndpoints } from '@/lib/wails-api'

export interface SchemaNode {
  id: string
  name: string
  type: 'database' | 'schema' | 'table' | 'column'
  children?: SchemaNode[]
  expanded?: boolean
  metadata?: unknown
}

// Schema cache with persistence
class SchemaCache {
  private static instance: SchemaCache
  private cache: Map<string, { data: SchemaNode[], timestamp: number }> = new Map()
  private readonly CACHE_EXPIRY = 24 * 60 * 60 * 1000 // 24 hours
  private readonly STORAGE_KEY = 'howlerops-schema-cache'

  static getInstance(): SchemaCache {
    if (!SchemaCache.instance) {
      SchemaCache.instance = new SchemaCache()
    }
    return SchemaCache.instance
  }

  constructor() {
    this.loadFromStorage()
  }

  private loadFromStorage() {
    try {
      const stored = localStorage.getItem(this.STORAGE_KEY)
      if (stored) {
        const parsed = JSON.parse(stored)
        const now = Date.now()
        
        // Only load non-expired entries
        Object.entries(parsed).forEach(([key, value]: [string, unknown]) => {
          const typedValue = value as { data: SchemaNode[]; timestamp: number };
          if (typedValue.timestamp && (now - typedValue.timestamp) < this.CACHE_EXPIRY) {
            this.cache.set(key, typedValue)
          }
        })
      }
    } catch (error) {
      console.warn('Failed to load schema cache from storage:', error)
    }
  }

  private saveToStorage() {
    try {
      const toStore: Record<string, { data: SchemaNode[]; timestamp: number }> = {}
      this.cache.forEach((value, key) => {
        toStore[key] = value
      })
      localStorage.setItem(this.STORAGE_KEY, JSON.stringify(toStore))
    } catch (error) {
      console.warn('Failed to save schema cache to storage:', error)
    }
  }

  get(key: string): SchemaNode[] | null {
    const entry = this.cache.get(key)
    if (!entry) return null
    
    const now = Date.now()
    if ((now - entry.timestamp) > this.CACHE_EXPIRY) {
      this.cache.delete(key)
      this.saveToStorage()
      return null
    }
    
    return entry.data
  }

  set(key: string, data: SchemaNode[]) {
    this.cache.set(key, {
      data,
      timestamp: Date.now()
    })
    this.saveToStorage()
  }

  clear(key?: string) {
    if (key) {
      this.cache.delete(key)
    } else {
      this.cache.clear()
    }
    this.saveToStorage()
  }

  clearExpired() {
    const now = Date.now()
    const expiredKeys: string[] = []
    
    this.cache.forEach((value, key) => {
      if ((now - value.timestamp) > this.CACHE_EXPIRY) {
        expiredKeys.push(key)
      }
    })
    
    expiredKeys.forEach(key => this.cache.delete(key))
    if (expiredKeys.length > 0) {
      this.saveToStorage()
    }
  }
}

export function useSchemaIntrospection() {
  const { activeConnection } = useConnectionStore()
  const [schema, setSchema] = useState<SchemaNode[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cacheRef = useRef<SchemaCache>(SchemaCache.getInstance())

  // Helper function for formatting column names
  const formatColumnName = useCallback((column: {
    name: string;
    dataType?: string;
    characterMaximumLength?: number;
    numericPrecision?: number;
    numericScale?: number;
    isNullable?: string;
    columnDefault?: string | null;
    isPrimaryKey?: boolean;
    isForeignKey?: boolean;
  }): string => {
    let formattedName = column.name

    if (column.dataType) {
      let typeStr = column.dataType
      if (column.characterMaximumLength) {
        typeStr += `(${column.characterMaximumLength})`
      } else if (column.numericPrecision) {
        typeStr += `(${column.numericPrecision}`
        if (column.numericScale) {
          typeStr += `,${column.numericScale}`
        }
        typeStr += ')'
      }
      formattedName += `: ${typeStr}`
    }

    const badges = []
    if (column.isPrimaryKey) badges.push('PK')
    if (column.isForeignKey) badges.push('FK')
    if (column.isNullable === 'NO') badges.push('NOT NULL')

    if (badges.length > 0) {
      formattedName += ` [${badges.join(', ')}]`
    }

    return formattedName
  }, [])

  const fetchSchema = useCallback(async (connectionId: string, cacheKey: string, force = false) => {
    if (!force) {
      const cached = cacheRef.current.get(cacheKey)
      if (cached) {
        setSchema(cached)
        setLoading(false)
        setError(null)
        return
      }
    }

    setLoading(true)
    setError(null)

    try {
      // Fetch schemas/databases
      const schemasResponse = await wailsEndpoints.schema.databases(connectionId)

      if (!schemasResponse.success || !schemasResponse.data) {
        throw new Error(schemasResponse.message || 'Failed to fetch schemas')
      }

      const schemaNodes: SchemaNode[] = []

      // For each schema, fetch tables
      for (const schemaInfo of schemasResponse.data) {
        const schemaNode: SchemaNode = {
          id: schemaInfo.name,
          name: schemaInfo.name,
          type: 'schema',
          expanded: schemaInfo.name === 'public', // Expand 'public' by default
          children: []
        }

        // Fetch tables for this schema
        const tablesResponse = await wailsEndpoints.schema.tables(
          connectionId,
          schemaInfo.name
        )

        if (tablesResponse.success && tablesResponse.data) {
          schemaNode.children = await Promise.all(
            tablesResponse.data.map(async (tableInfo, tableIndex) => {
              const tableId = `${schemaInfo.name}.${tableInfo.name}.${tableIndex}`
              const tableNode: SchemaNode = {
                id: tableId,
                name: tableInfo.name,
                type: 'table',
                children: [],
                metadata: {
                  rowCount: tableInfo.rowCount,
                  sizeBytes: tableInfo.sizeBytes,
                  comment: tableInfo.comment
                }
              }

              // Fetch columns for this table
              try {
                const columnsResponse = await wailsEndpoints.schema.columns(
                  connectionId,
                  schemaInfo.name,
                  tableInfo.name
                )

                if (columnsResponse.success && columnsResponse.data) {
                  tableNode.children = columnsResponse.data.map((columnInfo, columnIndex) => ({
                    id: `${tableId}.${columnInfo.name}.${columnIndex}`,
                    name: formatColumnName(columnInfo),
                    type: 'column' as const,
                    metadata: columnInfo
                  }))
                }
              } catch (err) {
                console.error(`Failed to fetch columns for ${tableInfo.name}:`, err)
              }

              return tableNode
            })
          )
        }

        schemaNodes.push(schemaNode)
      }

      setSchema(schemaNodes)
      cacheRef.current.set(cacheKey, schemaNodes)
    } catch (err) {
      console.error('Schema introspection error:', err)
      setError(err instanceof Error ? err.message : 'Failed to fetch schema')
      setSchema([])
    } finally {
      setLoading(false)
    }
  }, [formatColumnName])

  // Add useEffect hook here after fetchSchema is defined
  useEffect(() => {
    if (!activeConnection?.isConnected || !activeConnection.sessionId) {
      setSchema([])
      setLoading(false)
      setError(null)
      return
    }

    const cacheKey = activeConnection.sessionId ?? activeConnection.id
    if (cacheKey) {
      const cached = cacheRef.current.get(cacheKey)
      if (cached) {
        setSchema(cached)
        setLoading(false)
        setError(null)
        return
      }
    }

    fetchSchema(activeConnection.sessionId, activeConnection.sessionId ?? activeConnection.id ?? '', false)
  }, [activeConnection?.isConnected, activeConnection?.sessionId, activeConnection?.id, fetchSchema])

  const refreshSchema = useCallback(() => {
    if (activeConnection?.isConnected && activeConnection.sessionId) {
      const cacheKey = activeConnection.sessionId ?? activeConnection.id ?? ''
      cacheRef.current.clear(cacheKey)
      fetchSchema(activeConnection.sessionId, cacheKey, true)
    }
  }, [activeConnection?.isConnected, activeConnection?.sessionId, activeConnection?.id, fetchSchema])

  const clearCache = useCallback(() => {
    cacheRef.current.clear()
    setSchema([])
  }, [])

  const clearExpiredCache = useCallback(() => {
    cacheRef.current.clearExpired()
  }, [])

  return { schema, loading, error, refreshSchema, clearCache, clearExpiredCache }
}
