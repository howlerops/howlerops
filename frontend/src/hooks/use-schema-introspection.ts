import { useEffect, useState, useCallback } from 'react'
import { useConnectionStore } from '@/store/connection-store'
import { useSchemaStore, type SchemaNode } from '@/store/schema-store'

interface SyntheticViewColumn {
  name: string
  type?: string
  [key: string]: unknown
}

interface SyntheticViewDefinition {
  name: string
  columns?: SyntheticViewColumn[]
  [key: string]: unknown
}

const normaliseSyntheticColumn = (column: unknown): SyntheticViewColumn | null => {
  if (!column || typeof column !== 'object') {
    return null
  }

  const candidate = column as Record<string, unknown>
  const name = typeof candidate.name === 'string' ? candidate.name : undefined
  if (!name) {
    return null
  }

  const type = typeof candidate.type === 'string' ? candidate.type : undefined

  return {
    ...candidate,
    name,
    ...(type ? { type } : {}),
  }
}

const _normaliseSyntheticView = (view: unknown): SyntheticViewDefinition | null => {
  if (!view || typeof view !== 'object') {
    return null
  }

  const candidate = view as Record<string, unknown>
  const name = typeof candidate.name === 'string' ? candidate.name : undefined
  if (!name) {
    return null
  }

  let columns: SyntheticViewColumn[] | undefined
  if (Array.isArray(candidate.columns)) {
    columns = candidate.columns
      .map(normaliseSyntheticColumn)
      .filter((column): column is SyntheticViewColumn => column !== null)
  }

  return {
    ...candidate,
    name,
    ...(columns ? { columns } : {}),
  }
}

// Re-export SchemaNode from the centralized store
export type { SchemaNode } from '@/store/schema-store'

// Deprecated: Old SchemaCache - replaced by centralized useSchemaStore
// Keeping for backward compatibility during migration
class _SchemaCache {
  private static instance: SchemaCache

  static getInstance(): SchemaCache {
    if (!SchemaCache.instance) {
      SchemaCache.instance = new SchemaCache()
    }
    return SchemaCache.instance
  }

  get(_key: string): SchemaNode[] | null {
    return null // Deprecated - use useSchemaStore instead
  }

  set(_key: string, _data: SchemaNode[]) {
    // Deprecated - use useSchemaStore instead
  }

  clear(_key?: string) {
    // Deprecated - use useSchemaStore instead
  }

  clearExpired() {
    // Deprecated - use useSchemaStore instead
  }
}

/**
 * Simplified hook for schema introspection using centralized store
 *
 * @deprecated Consider using useSchemaStore directly for more control
 */
export function useSchemaIntrospection() {
  const { activeConnection } = useConnectionStore()
  const getSchema = useSchemaStore((state) => state.getSchema)
  const isLoading = useSchemaStore((state) => state.isLoading)
  const getError = useSchemaStore((state) => state.getError)
  const invalidate = useSchemaStore((state) => state.invalidate)
  const _invalidateAll = useSchemaStore((state) => state._invalidateAll)

  const [schema, setSchema] = useState<SchemaNode[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Fetch schema when connection changes
  useEffect(() => {
    if (!activeConnection?.isConnected || !activeConnection.sessionId) {
      setSchema([])
      setLoading(false)
      setError(null)
      return
    }

    const sessionId = activeConnection.sessionId

    // Update loading state
    setLoading(isLoading(sessionId))
    setError(getError(sessionId) || null)

    // Fetch from store
    getSchema(sessionId, activeConnection.name)
      .then((schemas) => {
        setSchema(schemas)
        setLoading(false)
        setError(null)
      })
      .catch((err) => {
        console.error('Failed to load schema:', err)
        setSchema([])
        setLoading(false)
        setError(err instanceof Error ? err.message : 'Failed to load schema')
      })
  }, [activeConnection?.isConnected, activeConnection?.sessionId, activeConnection?.name, getSchema, isLoading, getError])

  const refreshSchema = useCallback(() => {
    if (activeConnection?.isConnected && activeConnection.sessionId) {
      invalidate(activeConnection.sessionId)
      getSchema(activeConnection.sessionId, activeConnection.name, true)
        .then((schemas) => {
          setSchema(schemas)
          setLoading(false)
          setError(null)
        })
        .catch((err) => {
          console.error('Failed to refresh schema:', err)
          setError(err instanceof Error ? err.message : 'Failed to refresh schema')
        })
    }
  }, [activeConnection?.isConnected, activeConnection?.sessionId, activeConnection?.name, getSchema, invalidate])

  const clearCache = useCallback(() => {
    if (activeConnection?.sessionId) {
      invalidate(activeConnection.sessionId)
    }
    setSchema([])
  }, [activeConnection?.sessionId, invalidate])

  const clearExpiredCache = useCallback(() => {
    // No-op - store handles expiration automatically
  }, [])

  return { schema, loading, error, refreshSchema, clearCache, clearExpiredCache }
}
