import type { ColumnInfo, SchemaInfo, TableInfo } from "@/components/visual-query-builder/types"
import type { SchemaNode } from "@/hooks/use-schema-introspection"

import type { SqlDialect } from "./types"

/**
 * Map connection database type to SQL dialect for query generation
 */
export function getDialectFromConnectionType(connectionType: string | undefined): SqlDialect {
  if (!connectionType) return 'postgres'
  switch (connectionType.toLowerCase()) {
    case 'postgresql':
    case 'postgres':
      return 'postgres'
    case 'mysql':
    case 'mariadb':
    case 'tidb':
      return 'mysql'
    case 'sqlite':
      return 'sqlite'
    case 'mssql':
    case 'sqlserver':
      return 'mssql'
    default:
      return 'postgres'
  }
}

/**
 * Convert SchemaNode[] (hierarchical tree) to SchemaInfo[] (flat list for visual query builder)
 */
export function convertSchemaNodes(schemaNodes: SchemaNode[]): SchemaInfo[] {
  const result: SchemaInfo[] = []

  for (const schemaOrDb of schemaNodes) {
    // Handle both database-level nodes and schema-level nodes
    if (schemaOrDb.type === 'schema') {
      const tables: TableInfo[] = []
      for (const tableNode of schemaOrDb.children || []) {
        if (tableNode.type === 'table') {
          const columns: ColumnInfo[] = []
          for (const colNode of tableNode.children || []) {
            if (colNode.type === 'column') {
              const meta = colNode.metadata as Record<string, unknown> | undefined
              columns.push({
                name: colNode.name,
                dataType: (meta?.dataType as string) || (meta?.data_type as string) || 'unknown',
                isNullable: meta?.isNullable === true || meta?.nullable === true || meta?.isNullable === 'YES',
                isPrimaryKey: meta?.isPrimaryKey === true || meta?.primaryKey === true,
                isForeignKey: meta?.isForeignKey === true || meta?.foreignKey === true,
              })
            }
          }
          const tableMeta = tableNode.metadata as Record<string, unknown> | undefined
          tables.push({
            name: tableNode.name,
            schema: schemaOrDb.name,
            columns,
            rowCount: tableMeta?.rowCount as number | undefined,
            sizeBytes: tableMeta?.sizeBytes as number | undefined,
          })
        }
      }
      result.push({ name: schemaOrDb.name, tables })
    } else if (schemaOrDb.type === 'database') {
      // Recurse into database children (which should be schemas)
      for (const childSchema of schemaOrDb.children || []) {
        if (childSchema.type === 'schema') {
          result.push(...convertSchemaNodes([childSchema]))
        }
      }
    }
  }

  return result
}

/**
 * Check if a target is a typing-related input element
 */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }
  const tag = target.tagName.toLowerCase()
  return tag === 'input' || tag === 'textarea' || target.isContentEditable
}

/**
 * Filter tables that should be excluded (migrations, internal tables)
 */
export function shouldExcludeTable(tableName: string, schemaName: string): boolean {
  return (
    tableName === 'schema_migrations' ||
    tableName === 'goose_db_version' ||
    tableName === '_prisma_migrations' ||
    tableName.startsWith('__drizzle') ||
    schemaName.startsWith('pg_temp') ||
    schemaName.startsWith('pg_toast')
  )
}

/**
 * Generate connection keys for schema lookup (id, name, and slug variants)
 */
export function getConnectionKeys(connectionId: string, connectionName?: string): Set<string> {
  const keys = new Set<string>([connectionId])

  if (connectionName) {
    keys.add(connectionName)

    const slug = connectionName.replace(/[^\w-]/g, '-')
    if (slug && slug !== connectionName) {
      keys.add(slug)
    }
  }

  return keys
}

/**
 * Format a date for display in memory sessions
 */
export function formatSessionDate(timestamp: number): string {
  return new Date(timestamp).toLocaleString()
}

/**
 * Debounce utility for scheduling updates
 */
export function createDebouncer(delay: number): {
  schedule: (callback: () => void) => void
  cancel: () => void
  flush: (callback: () => void) => void
} {
  let timeoutId: number | null = null

  return {
    schedule(callback: () => void) {
      if (typeof window === 'undefined') {
        callback()
        return
      }

      if (timeoutId !== null) {
        window.clearTimeout(timeoutId)
      }

      timeoutId = window.setTimeout(() => {
        callback()
        timeoutId = null
      }, delay)
    },
    cancel() {
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId)
        timeoutId = null
      }
    },
    flush(callback: () => void) {
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId)
        timeoutId = null
        callback()
      }
    },
  }
}
