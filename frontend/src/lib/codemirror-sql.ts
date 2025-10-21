/**
 * CodeMirror 6 SQL Extension with Multi-Database Support
 * 
 * Provides syntax highlighting, autocomplete, and language support
 * for both single-database and multi-database queries with @connection.table syntax
 */

import { EditorView, keymap, ViewUpdate } from '@codemirror/view'
import { EditorState, Extension, StateEffect, StateField } from '@codemirror/state'
import { sql, SQLDialect } from '@codemirror/lang-sql'
import { 
  autocompletion, 
  Completion, 
  CompletionContext,
  CompletionResult,
  startCompletion,
  acceptCompletion
} from '@codemirror/autocomplete'
import { defaultKeymap, historyKeymap, history, indentMore } from '@codemirror/commands'
import { searchKeymap } from '@codemirror/search'
import { oneDark } from '@codemirror/theme-one-dark'

export interface Connection {
  id: string
  name: string
  type: string
  database: string
  sessionId?: string
  isConnected?: boolean
  alias?: string
}

export interface SchemaNode {
  name: string
  type: 'database' | 'schema' | 'table' | 'column'
  children?: SchemaNode[]
  dataType?: string
  nullable?: boolean
  primaryKey?: boolean
  sessionId?: string
}

export interface Column {
  name: string
  dataType: string
  nullable?: boolean
  primaryKey?: boolean
}

export type ColumnLoader = (sessionId: string, schema: string, table: string) => Promise<Column[]>

// State effect to update schema data
export const updateSchemaEffect = StateEffect.define<{
  connections: Connection[]
  schemas: Map<string, SchemaNode[]>
  mode: 'single' | 'multi'
  isLoading?: boolean
}>()

// State field to hold schema data
export const schemaState = StateField.define<{
  connections: Connection[]
  schemas: Map<string, SchemaNode[]>
  mode: 'single' | 'multi'
  columnCache: Map<string, Column[]>
  isLoading: boolean
}>({
  create() {
    return {
      connections: [],
      schemas: new Map(),
      mode: 'single',
      columnCache: new Map(),
      isLoading: false
    }
  },
  update(value, tr) {
    for (const effect of tr.effects) {
      if (effect.is(updateSchemaEffect)) {
        return {
          ...value,
          connections: effect.value.connections,
          schemas: effect.value.schemas,
          mode: effect.value.mode,
          isLoading: effect.value.isLoading ?? value.isLoading
        }
      }
    }
    return value
  }
})

/**
 * SQL Keywords for autocomplete
 */
const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER',
  'ON', 'AS', 'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'LIKE', 'BETWEEN',
  'ORDER', 'BY', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET', 'UNION', 'ALL',
  'DISTINCT', 'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE',
  'CREATE', 'TABLE', 'DROP', 'ALTER', 'INDEX', 'VIEW', 'WITH',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'NULL', 'IS', 'CAST',
  'COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'COALESCE', 'NULLIF'
]

/**
 * Find connection by name or ID
 */
function findConnection(connections: Connection[], identifier: string): Connection | undefined {
  return connections.find(c =>
    c.name === identifier ||
    c.id === identifier ||
    c.alias === identifier
  )
}

/**
 * Check if user is typing a SQL keyword
 */
function isTypingSQLKeyword(word: string): boolean {
  if (!word || word.length < 2) return false
  const upperWord = word.toUpperCase()
  return SQL_KEYWORDS.some(kw => kw.startsWith(upperWord))
}

/**
 * Create SQL autocomplete extension
 */
export function sqlAutocompletion(columnLoader?: ColumnLoader): Extension {
  return autocompletion({
    override: [
      async (context: CompletionContext): Promise<CompletionResult | null> => {
        const state = context.state.field(schemaState, false)
        if (!state) return null

        const { connections, schemas, mode, columnCache } = state
        const word = context.matchBefore(/[@\w.-]*/)
        if (!word) return null

        const textBeforeCursor = context.state.doc.sliceString(
          Math.max(0, context.pos - 200),
          context.pos
        )

        const options: Completion[] = []

        // Multi-DB Mode Patterns
        if (mode === 'multi') {
          // Pattern 1: User typed '@' - show connections
          if (textBeforeCursor.endsWith('@')) {
            connections
              .filter(conn => conn.isConnected)
              .forEach((conn, idx) => {
                options.push({
                  label: `@${conn.name}`,
                  type: 'namespace',
                  detail: `${conn.type} - ${conn.database}`,
                  info: `Connection: ${conn.name}`,
                  boost: 100 - idx
                })
              })

            return {
              from: context.pos - 1,
              options,
              validFor: /@\w*/
            }
          }

          // Pattern 2: @connection.schema.table. - show columns
          const columnPattern = /@([\w-]+)\.([\w-]+)\.([\w-]+)\.(\w*)$/
          const columnMatch = textBeforeCursor.match(columnPattern)

          if (columnMatch && columnLoader) {
            const [, connId, schemaName, tableName, partialColumn] = columnMatch
            const connection = findConnection(connections, connId)

            if (connection?.sessionId) {
              const cacheKey = `${connection.sessionId}-${schemaName}-${tableName}`
              let columns: Column[]

              if (columnCache.has(cacheKey)) {
                columns = columnCache.get(cacheKey)!
              } else {
                try {
                  columns = await columnLoader(connection.sessionId, schemaName, tableName)
                  columnCache.set(cacheKey, columns)
                } catch {
                  columns = []
                }
              }

              columns
                .filter(col => !partialColumn || col.name.toLowerCase().startsWith(partialColumn.toLowerCase()))
                .forEach(col => {
                  options.push({
                    label: col.name,
                    type: 'property',
                    detail: col.dataType,
                    info: col.nullable ? 'Nullable' : 'Not null',
                    boost: 90
                  })
                })

              return {
                from: context.pos - partialColumn.length,
                options
              }
            }
          }

          // Pattern 3: @connection. - show schemas/tables
          const tablePattern = /@([\w-]+)\.(\w*)$/
          const tableMatch = textBeforeCursor.match(tablePattern)

          if (tableMatch) {
            const [, connId, partialTable] = tableMatch
            const connection = findConnection(connections, connId)

            // Use connId (matched identifier) instead of connection.id for schema lookup
            if (connection && schemas.has(connId)) {
              const connSchemas = schemas.get(connId)!

              connSchemas.forEach(schema => {
                if (schema.type === 'schema' && schema.children) {
                  schema.children
                    .filter(t => t.type === 'table')
                    .filter(t => !partialTable || t.name.toLowerCase().startsWith(partialTable.toLowerCase()))
                    .forEach(table => {
                      const fullName = schema.name === 'public' ? table.name : `${schema.name}.${table.name}`
                      options.push({
                        label: table.name,
                        type: 'class',
                        detail: `@${connection.name}.${schema.name}.${table.name}`,
                        apply: fullName,
                        info: `Table from ${connection.name}`,
                        boost: 80
                      })
                    })
                }
              })

              return {
                from: context.pos - partialTable.length,
                options
              }
            }
            
            // If pattern matched but no schemas found, return empty to prevent fallthrough
            return {
              from: context.pos - partialTable.length,
              options: []
            }
          }
        }

        // Single-DB Mode or fallback: table.column pattern
        if (mode === 'single' || !textBeforeCursor.includes('@')) {
          const tableColPattern = /(\w+)\.(\w*)$/
          const match = textBeforeCursor.match(tableColPattern)

          if (match && columnLoader && schemas.size > 0) {
            const [, tableName, partialColumn] = match
            const firstSchemaSet = Array.from(schemas.values())[0]

            if (firstSchemaSet && connections.length > 0 && connections[0].sessionId) {
              for (const schemaNode of firstSchemaSet) {
                if (schemaNode.type === 'schema' && schemaNode.children) {
                  const table = schemaNode.children.find(
                    t => t.type === 'table' && t.name.toLowerCase() === tableName.toLowerCase()
                  )

                  if (table) {
                    const cacheKey = `${connections[0].sessionId}-${schemaNode.name}-${tableName}`
                    let columns: Column[]

                    if (columnCache.has(cacheKey)) {
                      columns = columnCache.get(cacheKey)!
                    } else {
                      try {
                        columns = await columnLoader(connections[0].sessionId, schemaNode.name, tableName)
                        columnCache.set(cacheKey, columns)
                      } catch {
                        columns = []
                      }
                    }

                    columns
                      .filter(col => !partialColumn || col.name.toLowerCase().startsWith(partialColumn.toLowerCase()))
                      .forEach(col => {
                        options.push({
                          label: col.name,
                          type: 'property',
                          detail: col.dataType,
                          info: col.nullable ? 'Nullable' : 'Not null',
                          boost: 90
                        })
                      })

                    if (options.length > 0) {
                      return {
                        from: context.pos - partialColumn.length,
                        options
                      }
                    }
                  }
                }
              }
            }
          }
        }

        // Default: SQL keywords and tables
        const currentWord = word.text.toUpperCase()
        const isKeyword = isTypingSQLKeyword(currentWord)

        // Add SQL keywords
        if (!textBeforeCursor.includes('@') || isKeyword) {
          SQL_KEYWORDS.forEach((keyword, idx) => {
            if (keyword.startsWith(currentWord) || currentWord.length < 2) {
              options.push({
                label: keyword,
                type: 'keyword',
                detail: 'SQL keyword',
                boost: 50 - idx
              })
            }
          })
        }

        // Add tables from all schemas
        const schemasToProcess = mode === 'single'
          ? [Array.from(schemas.values())[0]]
          : Array.from(schemas.values())

        schemasToProcess.forEach(schemaSet => {
          if (schemaSet) {
            schemaSet.forEach(schemaNode => {
              if (schemaNode.type === 'schema' && schemaNode.children) {
                schemaNode.children.forEach(table => {
                  if (table.type === 'table') {
                    options.push({
                      label: table.name,
                      type: 'class',
                      detail: `Table in ${schemaNode.name}`,
                      info: mode === 'multi' && connections.length > 1
                        ? `Use @connection.${table.name} for multi-DB queries`
                        : undefined,
                      boost: 70
                    })
                  }
                })
              }
            })
          }
        })

        if (options.length === 0) return null

        return {
          from: word.from,
          options
        }
      }
    ]
  })
}

/**
 * Create base SQL extensions
 */
export function createSQLExtensions(
  theme: 'light' | 'dark',
  columnLoader?: ColumnLoader,
  onChange?: (value: string) => void
): Extension[] {
  const extensions: Extension[] = [
    history(),
    EditorView.lineWrapping,
    EditorState.tabSize.of(2),
    sql({
      dialect: SQLDialect.define({
        // Support @ in identifiers for multi-DB syntax
        charSetCasts: true,
        // Add support for @connection syntax
      })
    }),
    schemaState,
    sqlAutocompletion(columnLoader),
    keymap.of([
      // Tab key accepts completion if available, otherwise indents
      {
        key: 'Tab',
        run: (view) => {
          // Try to accept completion first
          if (acceptCompletion(view)) {
            return true
          }
          // Otherwise, indent
          return indentMore(view)
        }
      },
      ...defaultKeymap,
      ...historyKeymap,
      ...searchKeymap,
      {
        key: 'Ctrl-Space',
        mac: 'Cmd-Space',
        run: startCompletion
      }
    ]),
    EditorView.updateListener.of((update: ViewUpdate) => {
      if (update.docChanged && onChange) {
        onChange(update.state.doc.toString())
      }
    })
  ]

  // Add theme
  if (theme === 'dark') {
    extensions.push(oneDark)
  }

  return extensions
}

/**
 * Update editor schema state
 */
export function updateEditorSchema(
  view: EditorView,
  connections: Connection[],
  schemas: Map<string, SchemaNode[]>,
  mode: 'single' | 'multi',
  isLoading?: boolean
) {
  view.dispatch({
    effects: updateSchemaEffect.of({ connections, schemas, mode, isLoading })
  })
}

