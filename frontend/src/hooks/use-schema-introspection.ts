import { useCallback,useEffect, useState } from 'react'

import { useConnectionStore } from '@/store/connection-store'
import { type SchemaNode,useSchemaStore } from '@/store/schema-store'

// Re-export SchemaNode from the centralized store
export type { SchemaNode } from '@/store/schema-store'

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

    // Fetch from store (relies on cache - won't hit backend if cached)
    getSchema(sessionId, activeConnection.name || '')
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
      getSchema(activeConnection.sessionId, activeConnection.name || '', true)
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
