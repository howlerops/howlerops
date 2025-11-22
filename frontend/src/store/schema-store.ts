/**
 * Centralized Schema Store
 *
 * Single source of truth for all database schemas across the application.
 * Provides request deduplication, intelligent caching, and invalidation strategies.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { wailsEndpoints } from '@/lib/wails-api'

export interface SchemaNode {
  id: string
  name: string
  type: 'database' | 'schema' | 'table' | 'column'
  children?: SchemaNode[]
  expanded?: boolean
  metadata?: unknown
}

interface SchemaData {
  schemas: SchemaNode[]
  timestamp: number
  connectionName?: string
}

interface RawTableInfo {
  name: string
  rowCount?: number
  sizeBytes?: number
  comment?: string
}

interface RawColumnInfo {
  name: string
  dataType?: string
  data_type?: string
  characterMaximumLength?: number
  character_maximum_length?: number
  numericPrecision?: number
  numeric_precision?: number
  numericScale?: number
  numeric_scale?: number
  isNullable?: string
  nullable?: boolean
  columnDefault?: string | null
  defaultValue?: string | null
  default_value?: string | null
  isPrimaryKey?: boolean
  primaryKey?: boolean
  primary_key?: boolean
  isForeignKey?: boolean
  foreignKey?: boolean
  foreign_key?: boolean
  metadata?: Record<string, unknown>
  [key: string]: unknown
}

interface NormalizedColumnInfo {
  name: string
  dataType?: string
  characterMaximumLength?: number
  numericPrecision?: number
  numericScale?: number
  isNullable?: string
  columnDefault?: string | null
  isPrimaryKey?: boolean
  isForeignKey?: boolean
  metadata?: Record<string, unknown>
}

interface RawForeignKeyInfo {
  name?: string
  columns?: string[]
  columnName?: string
  referencedTable?: string
  referenced_table?: string
  referencedTableName?: string
  referencedSchema?: string
  referenced_schema?: string
  referencedSchemaName?: string
  referencedColumns?: string[]
  referenced_columns?: string[]
  referencedColumnName?: string
  onDelete?: string
  on_delete?: string
  deleteRule?: string
  onUpdate?: string
  on_update?: string
  updateRule?: string
}

interface NormalizedForeignKeyInfo {
  name: string
  columnName: string
  referencedTableName: string
  referencedSchemaName: string
  referencedColumnName: string
  deleteRule?: string
  updateRule?: string
}

interface SchemaStoreState {
  // Cache: sessionId -> SchemaData
  cache: Map<string, SchemaData>

  // In-flight requests: sessionId -> Promise
  pendingRequests: Map<string, Promise<SchemaNode[]>>

  // Loading states
  loading: Set<string>

  // Errors
  errors: Map<string, string>

  // Actions
  getSchema: (sessionId: string, connectionName?: string, force?: boolean) => Promise<SchemaNode[]>
  invalidate: (sessionId?: string) => void
  invalidateAll: () => void
  isLoading: (sessionId: string) => boolean
  getError: (sessionId: string) => string | undefined
  clearError: (sessionId: string) => void
}

const CACHE_DURATION_MS = 24 * 60 * 60 * 1000 // 24 hours

const normalizeColumnInfo = (
  column: RawColumnInfo,
  options?: {
    foreignKey?: NormalizedForeignKeyInfo
  }
): NormalizedColumnInfo => {
  const dataType = column.dataType ?? column.data_type
  const characterMaximumLength = column.characterMaximumLength ?? column.character_maximum_length
  const numericPrecision = column.numericPrecision ?? column.numeric_precision
  const numericScale = column.numericScale ?? column.numeric_scale

  let isNullable: string | undefined = column.isNullable
  if (!isNullable && typeof column.nullable === 'boolean') {
    isNullable = column.nullable ? 'YES' : 'NO'
  }

  const columnDefault = column.columnDefault ?? column.defaultValue ?? column.default_value ?? null
  const isPrimaryKey = column.isPrimaryKey ?? column.primaryKey ?? column.primary_key ?? false
  const isForeignKey =
    options?.foreignKey !== undefined
      ? true
      : column.isForeignKey ?? column.foreignKey ?? column.foreign_key ?? false

  const metadata: Record<string, unknown> = {
    ...column,
    dataType,
    characterMaximumLength,
    numericPrecision,
    numericScale,
    isNullable,
    columnDefault,
    isPrimaryKey,
    isForeignKey,
  }

  if (options?.foreignKey) {
    metadata.foreignKey = {
      name: options.foreignKey.name,
      referencedTable: options.foreignKey.referencedTableName,
      referencedSchema: options.foreignKey.referencedSchemaName,
      referencedColumns: [options.foreignKey.referencedColumnName],
    }
  }

  return {
    name: column.name,
    dataType,
    characterMaximumLength,
    numericPrecision,
    numericScale,
    isNullable,
    columnDefault,
    isPrimaryKey,
    isForeignKey,
    metadata,
  }
}

const normalizeForeignKeys = (
  foreignKeys: RawForeignKeyInfo[] | undefined,
  fallbackSchema: string,
  tableName: string
): NormalizedForeignKeyInfo[] => {
  if (!foreignKeys || foreignKeys.length === 0) {
    return []
  }

  const normalized: NormalizedForeignKeyInfo[] = []

  for (const fk of foreignKeys) {
    const columnName = fk.columnName || fk.columns?.[0]
    const referencedTableName = fk.referencedTableName || fk.referencedTable || fk.referenced_table
    if (!columnName || !referencedTableName) {
      continue
    }

    normalized.push({
      name: fk.name || `${tableName}_${columnName}_fk`,
      columnName,
      referencedTableName,
      referencedSchemaName: fk.referencedSchemaName || fk.referencedSchema || fk.referenced_schema || fallbackSchema,
      referencedColumnName: fk.referencedColumnName || fk.referencedColumns?.[0] || fk.referenced_columns?.[0] || 'id',
      deleteRule: fk.deleteRule || fk.onDelete || fk.on_delete || 'NO ACTION',
      updateRule: fk.updateRule || fk.onUpdate || fk.on_update || 'NO ACTION',
    })
  }

  return normalized
}

// Helper to format column names
const formatColumnName = (column: NormalizedColumnInfo): string => {
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
}

export const useSchemaStore = create<SchemaStoreState>()(
  persist(
    (set, get) => ({
      cache: new Map(),
      pendingRequests: new Map(),
      loading: new Set(),
      errors: new Map(),

      getSchema: async (sessionId: string, connectionName?: string, force = false) => {
        // Check cache first (unless force refresh)
        if (!force) {
          const cached = get().cache.get(sessionId)
          if (cached) {
            const age = Date.now() - cached.timestamp
            if (age < CACHE_DURATION_MS) {
              console.log(`[SchemaStore] ✓ Cache hit for ${sessionId} (age: ${Math.round(age / 1000)}s) - using cached schema`)
              return cached.schemas
            } else {
              console.log(`[SchemaStore] ⚠ Cache expired for ${sessionId} (age: ${Math.round(age / 1000)}s) - refreshing`)
            }
          }
        }

        // Check if request is already in flight (deduplication)
        const pending = get().pendingRequests.get(sessionId)
        if (pending) {
          console.log(`[SchemaStore] ⏳ Deduplicating request for ${sessionId} - joining existing request`)
          return pending
        }

        // Start new request
        console.log(`[SchemaStore] 🔄 Fetching schema from backend for ${sessionId}${force ? ' (force refresh)' : ''}`)
        const requestPromise = (async () => {
          try {
            set((state) => ({
              loading: new Set(state.loading).add(sessionId),
              errors: (() => {
                const newErrors = new Map(state.errors)
                newErrors.delete(sessionId)
                return newErrors
              })()
            }))

            // Fetch schemas/databases
            const schemasResponse = await wailsEndpoints.schema.databases(sessionId)

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
                sessionId,
                schemaInfo.name
              )

              if (tablesResponse.success && tablesResponse.data) {
                schemaNode.children = await Promise.all(
                  tablesResponse.data.map(async (tableInfo: RawTableInfo, tableIndex: number) => {
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
                        sessionId,
                        schemaInfo.name,
                        tableInfo.name
                      )

                      if (columnsResponse.success && columnsResponse.data) {
                        const normalizedForeignKeys = normalizeForeignKeys(
                          columnsResponse.foreignKeys as RawForeignKeyInfo[] | undefined,
                          schemaInfo.name,
                          tableInfo.name
                        )

                        const foreignKeyByColumn = new Map<string, NormalizedForeignKeyInfo>()
                        normalizedForeignKeys.forEach(fk => {
                          if (!foreignKeyByColumn.has(fk.columnName)) {
                            foreignKeyByColumn.set(fk.columnName, fk)
                          }
                        })

                        tableNode.children = columnsResponse.data.map((columnInfo: RawColumnInfo, columnIndex: number) => {
                          const normalizedColumn = normalizeColumnInfo(columnInfo, {
                            foreignKey: foreignKeyByColumn.get(columnInfo.name)
                          })

                          return {
                            id: `${tableId}.${columnInfo.name}.${columnIndex}`,
                            name: formatColumnName(normalizedColumn),
                            type: 'column' as const,
                            metadata: normalizedColumn.metadata ?? normalizedColumn
                          }
                        })

                        if (normalizedForeignKeys.length > 0) {
                          tableNode.metadata = {
                            ...(tableNode.metadata || {}),
                            foreignKeys: normalizedForeignKeys
                          }
                        }
                      }
                    } catch (err) {
                      console.error(`Failed to fetch columns for ${tableInfo.name}:`, err)
                    }

                    return tableNode
                  })
                )
              }

              // Skip empty schemas
              if (schemaNode.children && schemaNode.children.length > 0) {
                schemaNodes.push(schemaNode)
              }
            }

            // Update cache
            set((state) => {
              const newCache = new Map(state.cache)
              newCache.set(sessionId, {
                schemas: schemaNodes,
                timestamp: Date.now(),
                connectionName
              })
              return { cache: newCache }
            })

            return schemaNodes
          } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Failed to load schema'
            console.error(`[SchemaStore] Error loading schema for ${sessionId}:`, error)

            set((state) => {
              const newErrors = new Map(state.errors)
              newErrors.set(sessionId, errorMessage)
              return { errors: newErrors }
            })

            throw error
          } finally {
            // Clean up
            set((state) => {
              const newLoading = new Set(state.loading)
              newLoading.delete(sessionId)

              const newPending = new Map(state.pendingRequests)
              newPending.delete(sessionId)

              return {
                loading: newLoading,
                pendingRequests: newPending
              }
            })
          }
        })()

        // Store pending request for deduplication
        set((state) => {
          const newPending = new Map(state.pendingRequests)
          newPending.set(sessionId, requestPromise)
          return { pendingRequests: newPending }
        })

        return requestPromise
      },

      invalidate: (sessionId?: string) => {
        if (sessionId) {
          console.log(`[SchemaStore] Invalidating cache for ${sessionId}`)
          set((state) => {
            const newCache = new Map(state.cache)
            newCache.delete(sessionId)

            const newErrors = new Map(state.errors)
            newErrors.delete(sessionId)

            return {
              cache: newCache,
              errors: newErrors
            }
          })
        }
      },

      invalidateAll: () => {
        console.log('[SchemaStore] Invalidating all caches')
        set({
          cache: new Map(),
          errors: new Map()
        })
      },

      isLoading: (sessionId: string) => {
        return get().loading.has(sessionId)
      },

      getError: (sessionId: string) => {
        return get().errors.get(sessionId)
      },

      clearError: (sessionId: string) => {
        set((state) => {
          const newErrors = new Map(state.errors)
          newErrors.delete(sessionId)
          return { errors: newErrors }
        })
      }
    }),
    {
      name: 'schema-store',
      // Don't persist - keep session-only due to size
      // Backend provides persistence via schema_cache.go
      skipHydration: true,
      partialize: () => ({}) // Don't persist anything
    }
  )
)

// Helper hook to detect DDL statements that should invalidate cache
export function useSchemaInvalidation() {
  const invalidate = useSchemaStore((state) => state.invalidate)

  const shouldInvalidate = (query: string): boolean => {
    const upperQuery = query.trim().toUpperCase()
    const ddlKeywords = [
      'CREATE TABLE',
      'DROP TABLE',
      'ALTER TABLE',
      'CREATE SCHEMA',
      'DROP SCHEMA',
      'CREATE DATABASE',
      'DROP DATABASE',
      'RENAME TABLE',
      'TRUNCATE TABLE'
    ]

    return ddlKeywords.some(keyword => upperQuery.includes(keyword))
  }

  return { invalidate, shouldInvalidate }
}
