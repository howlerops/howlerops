/**
 * Hook for managing AI schema context in multi-database environments
 *
 * Provides schema information and context building for AI SQL generation
 */

import { useCallback,useMemo } from 'react'

import { AISchemaContextBuilder, MultiDatabaseContext } from '@/lib/ai-schema-context'
import { useConnectionStore } from '@/store/connection-store'

import { type SchemaNode,useSchemaIntrospection } from './use-schema-introspection'

export function useAISchemaContext(
  mode: 'single' | 'multi',
  multiDBSchemas?: Map<string, SchemaNode[]>
) {
  const {
    activeConnection,
    getFilteredConnections
  } = useConnectionStore()

  const { schema: singleDBSchema } = useSchemaIntrospection()

  /**
   * Build the AI context based on current mode and available schemas
   */
  const buildAIContext = useCallback((): MultiDatabaseContext | null => {
    if (mode === 'multi') {
      const filteredConns = getFilteredConnections()
      if (!multiDBSchemas || filteredConns.length === 0) {
        return null
      }

      return AISchemaContextBuilder.buildMultiDatabaseContext(
        filteredConns,
        multiDBSchemas,
        activeConnection?.id
      )
    } else {
      // Single database mode
      if (!activeConnection || !singleDBSchema) {
        return null
      }

      return AISchemaContextBuilder.buildSingleDatabaseContext(
        activeConnection,
        singleDBSchema
      )
    }
  }, [mode, activeConnection, singleDBSchema, multiDBSchemas, getFilteredConnections])

  /**
   * Get a compact schema summary for token-efficient prompts
   */
  const getCompactSchemaContext = useCallback((): string => {
    const context = buildAIContext()
    if (!context) {
      return ''
    }

    return AISchemaContextBuilder.generateCompactSchemaContext(context)
  }, [buildAIContext])

  /**
   * Generate a full AI prompt with schema context
   */
  const generateAIPrompt = useCallback((userPrompt: string): string => {
    const context = buildAIContext()
    if (!context) {
      return userPrompt
    }

    return AISchemaContextBuilder.generateAIPrompt(userPrompt, context)
  }, [buildAIContext])

  /**
   * Get syntax examples for the current mode
   */
  const getSyntaxExamples = useMemo((): string[] => {
    const context = buildAIContext()
    return context?.syntaxExamples || []
  }, [buildAIContext])

  /**
   * Check if schema context is ready
   */
  const isContextReady = useMemo((): boolean => {
    if (mode === 'multi') {
      return !!multiDBSchemas && multiDBSchemas.size > 0
    } else {
      return !!activeConnection && !!singleDBSchema && singleDBSchema.length > 0
    }
  }, [mode, activeConnection, singleDBSchema, multiDBSchemas])

  /**
   * Get table count across all databases
   */
  const getTableCount = useMemo((): number => {
    const context = buildAIContext()
    if (!context) return 0

    return context.databases.reduce((total, db) =>
      total + db.schemas.reduce((sum, schema) =>
        sum + schema.tables.length, 0
      ), 0
    )
  }, [buildAIContext])

  /**
   * Get connected database count
   */
  const getDatabaseCount = useMemo((): number => {
    const context = buildAIContext()
    return context?.databases.length || 0
  }, [buildAIContext])

  return {
    buildAIContext,
    getCompactSchemaContext,
    generateAIPrompt,
    getSyntaxExamples,
    isContextReady,
    getTableCount,
    getDatabaseCount,
    mode,
    activeConnection
  }
}