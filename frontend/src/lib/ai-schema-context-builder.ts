/**
 * AI Schema Context Builder
 *
 * Shared utilities for building schema context for AI operations.
 * Extracted from ai-store to reduce code duplication and improve maintainability.
 */

import type { SchemaNode } from '@/hooks/use-schema-introspection'
import type { DatabaseConnection } from '@/store/connection-store'

/**
 * Build context for multi-database or single-database queries
 */
export function buildSchemaContext(options: {
  mode: 'single' | 'multi'
  schema?: string
  connections?: DatabaseConnection[]
  schemasMap?: Map<string, SchemaNode[]>
  activeConnectionId?: string
}): string {
  const { mode, schema, connections, schemasMap } = options

  if (mode === 'multi' && connections && schemasMap && connections.length > 1) {
    // Multi-database mode: build comprehensive context
    const connectedDbs = connections.filter(c => c.isConnected)

    if (connectedDbs.length === 0) {
      return schema || ''
    }

    const contextParts: string[] = []

    for (const conn of connectedDbs) {
      const schemaNodes = schemasMap.get(conn.id)
      if (!schemaNodes || schemaNodes.length === 0) {
        continue
      }

      contextParts.push(`Database: @${conn.name}`)
      contextParts.push(`Connection: ${conn.type}`)

      // Add table information
      const tables = schemaNodes.filter(node => node.type === 'table')
      if (tables.length > 0) {
        contextParts.push('Tables:')
        for (const table of tables) {
          contextParts.push(`  @${conn.name}.${table.name}`)

          // If table has children (columns), list them
          if (table.children && table.children.length > 0) {
            const columns = table.children
              .filter(col => col.type === 'column')
              .map(col => `    - ${col.name}`)
              .join('\n')
            if (columns) {
              contextParts.push(columns)
            }
          }
        }
      }
      contextParts.push('') // Empty line between databases
    }

    return contextParts.join('\n')
  }

  // Single database mode: use simple schema context
  return schema || ''
}

/**
 * Enhance prompt with multi-database syntax instructions if needed
 */
export function enhancePromptForMode(
  prompt: string,
  mode: 'single' | 'multi',
  connections?: DatabaseConnection[]
): string {
  if (mode === 'multi' && connections && connections.length > 1) {
    const connectedDbs = connections.filter(c => c.isConnected)
    const dbNames = connectedDbs.map(c => c.name).join(', ')

    return `${prompt}\n\nIMPORTANT: This is a multi-database query. Use @connection_name.table syntax to reference tables from different databases. Available connections: ${dbNames}`
  }

  return prompt
}

/**
 * Detect if user wants multi-DB query based on prompt
 */
export function detectsMultiDB(prompt: string): boolean {
  const multiDBKeywords = [
    /join.*from.*and.*from/i,
    /across.*database/i,
    /between.*database/i,
    /from.*database.*and.*database/i,
    /@\w+\./,  // Already using @connection syntax
    /compare.*from.*and/i,
    /merge.*from.*and/i,
    /combine.*from.*and/i,
    /different.*database/i,
    /multiple.*database/i,
  ]

  return multiDBKeywords.some(pattern => pattern.test(prompt))
}

/**
 * Add memory context to schema context
 */
export function addMemoryContext(
  schemaContext: string,
  memoryContext: string | undefined
): string {
  if (!memoryContext) {
    return schemaContext
  }

  return schemaContext
    ? `${schemaContext}\n\n---\n\nConversation Memory:\n${memoryContext}`
    : `Conversation Memory:\n${memoryContext}`
}

/**
 * Add recall context to schema context
 */
export function addRecallContext(
  schemaContext: string,
  recallContext: string | undefined
): string {
  if (!recallContext) {
    return schemaContext
  }

  return schemaContext
    ? `${schemaContext}\n\n---\n\nRelated Sessions:\n${recallContext}`
    : `Related Sessions:\n${recallContext}`
}
