/**
 * SQL Context Parser
 *
 * Parses SQL queries to extract context for intelligent autocomplete:
 * - Tables and their aliases
 * - Current SQL clause (WHERE, SELECT, JOIN, etc.)
 * - Multi-DB references (@connection.schema.table)
 *
 * This is a pragmatic parser using regex - handles 90% of cases without full AST parsing.
 */

export type SQLClause =
  | 'SELECT'
  | 'FROM'
  | 'WHERE'
  | 'JOIN'
  | 'ON'
  | 'ORDER_BY'
  | 'GROUP_BY'
  | 'HAVING'
  | 'INSERT'
  | 'UPDATE'
  | 'UNKNOWN'

export interface TableReference {
  /** Original table identifier (could be 'table', 'schema.table', or '@conn.schema.table') */
  identifier: string

  /** Parsed table name */
  tableName: string

  /** Schema name if specified */
  schema?: string

  /** Connection ID for multi-DB mode (@connection) */
  connectionId?: string

  /** Alias if specified (FROM accounts a, FROM users AS u) */
  alias?: string

  /** Whether this is a multi-DB reference starting with @ */
  isMultiDB: boolean
}

export interface QueryContext {
  /** All table references in the query */
  tables: TableReference[]

  /** Current SQL clause where cursor is positioned */
  currentClause: SQLClause

  /** Alias to table mapping for quick lookup */
  aliasMap: Map<string, TableReference>

  /** Full query text */
  query: string

  /** Cursor position in query */
  cursorPos: number
}

/**
 * Parse a SQL query and extract context for autocomplete
 */
export function parseQueryContext(query: string, cursorPos: number): QueryContext {
  // Get the current statement (in case of multiple statements separated by ;)
  const currentStatement = getCurrentStatement(query, cursorPos)

  // Parse table references from FROM and JOIN clauses
  const tables = parseTableReferences(currentStatement)

  // Build alias map
  const aliasMap = new Map<string, TableReference>()
  tables.forEach(table => {
    if (table.alias) {
      aliasMap.set(table.alias.toLowerCase(), table)
    }
    // Also map table name itself for non-aliased references
    aliasMap.set(table.tableName.toLowerCase(), table)
  })

  // Detect current clause
  const currentClause = detectCurrentClause(currentStatement, cursorPos)

  return {
    tables,
    currentClause,
    aliasMap,
    query: currentStatement,
    cursorPos
  }
}

/**
 * Get the current SQL statement (handles multiple statements separated by ;)
 */
function getCurrentStatement(query: string, cursorPos: number): string {
  const statements = query.split(';')
  let pos = 0

  for (const statement of statements) {
    pos += statement.length
    if (pos >= cursorPos) {
      return statement.trim()
    }
    pos += 1 // for the semicolon
  }

  return statements[statements.length - 1].trim()
}

/**
 * Parse table references from FROM and JOIN clauses
 */
function parseTableReferences(query: string): TableReference[] {
  const tables: TableReference[] = []

  // Remove comments to avoid false matches
  const cleaned = removeComments(query)

  // Pattern 1: Multi-DB format - @connection.schema.table [AS] alias
  // Matches: @db1.public.accounts, @db1.public.accounts a, @db1.public.accounts AS acc
  const multiDBPattern = /@([\w-]+)\.([\w-]+)\.([\w-]+)(?:\s+(?:AS\s+)?(\w+))?/gi
  let match: RegExpExecArray | null

  while ((match = multiDBPattern.exec(cleaned)) !== null) {
    const [, connectionId, schema, tableName, alias] = match
    tables.push({
      identifier: match[0],
      tableName,
      schema,
      connectionId,
      alias: alias || undefined,
      isMultiDB: true
    })
  }

  // Pattern 2: Schema-qualified format - schema.table [AS] alias
  // Matches: public.accounts, dbo.users AS u
  const schemaTablePattern = /(?:FROM|JOIN)\s+([\w-]+)\.([\w-]+)(?:\s+(?:AS\s+)?(\w+))?/gi

  while ((match = schemaTablePattern.exec(cleaned)) !== null) {
    const [, schema, tableName, alias] = match

    // Skip if this looks like it was part of a multi-DB reference (already captured)
    const prevChar = match.index > 0 ? cleaned[match.index - 1] : ''
    if (prevChar === '@') continue

    tables.push({
      identifier: `${schema}.${tableName}`,
      tableName,
      schema,
      alias: alias || undefined,
      isMultiDB: false
    })
  }

  // Pattern 3: Simple table format - table [AS] alias
  // Matches: accounts, users AS u, orders AS o
  const simpleTablePattern = /(?:FROM|JOIN)\s+([\w-]+)(?:\s+(?:AS\s+)?(\w+))?/gi

  while ((match = simpleTablePattern.exec(cleaned)) !== null) {
    const [, tableName, alias] = match

    // Skip if this looks like it was part of a schema.table or @conn.schema.table
    const prevChar = match.index > 0 ? cleaned[match.index - 1] : ''
    if (prevChar === '.' || prevChar === '@') continue

    // Skip if we already captured this as part of schema.table
    const alreadyCaptured = tables.some(t =>
      t.tableName === tableName &&
      match &&
      cleaned.slice(Math.max(0, match.index - 20), match.index).includes('.')
    )
    if (alreadyCaptured) continue

    tables.push({
      identifier: tableName,
      tableName,
      alias: alias || undefined,
      isMultiDB: false
    })
  }

  return tables
}

/**
 * Detect which SQL clause the cursor is currently in
 */
function detectCurrentClause(query: string, cursorPos: number): SQLClause {
  const textBeforeCursor = query.slice(0, cursorPos).toUpperCase()

  // Check for clauses in reverse order of precedence
  // (later clauses override earlier ones)

  if (/\bINSERT\s+INTO\b/.test(textBeforeCursor)) {
    return 'INSERT'
  }

  if (/\bUPDATE\b/.test(textBeforeCursor) && /\bSET\b/.test(textBeforeCursor)) {
    return 'UPDATE'
  }

  // Check which clause keyword appears last before cursor
  const selectPos = textBeforeCursor.lastIndexOf('SELECT')
  const fromPos = textBeforeCursor.lastIndexOf('FROM')
  const wherePos = textBeforeCursor.lastIndexOf('WHERE')
  const joinPos = textBeforeCursor.lastIndexOf('JOIN')
  const onPos = textBeforeCursor.lastIndexOf(' ON ')
  const orderPos = textBeforeCursor.lastIndexOf('ORDER BY')
  const groupPos = textBeforeCursor.lastIndexOf('GROUP BY')
  const havingPos = textBeforeCursor.lastIndexOf('HAVING')

  const positions = [
    { pos: selectPos, clause: 'SELECT' as SQLClause },
    { pos: fromPos, clause: 'FROM' as SQLClause },
    { pos: wherePos, clause: 'WHERE' as SQLClause },
    { pos: joinPos, clause: 'JOIN' as SQLClause },
    { pos: onPos, clause: 'ON' as SQLClause },
    { pos: orderPos, clause: 'ORDER_BY' as SQLClause },
    { pos: groupPos, clause: 'GROUP_BY' as SQLClause },
    { pos: havingPos, clause: 'HAVING' as SQLClause },
  ]

  // Find the clause with the highest position (closest to cursor)
  const lastClause = positions
    .filter(p => p.pos >= 0)
    .sort((a, b) => b.pos - a.pos)[0]

  return lastClause?.clause || 'UNKNOWN'
}

/**
 * Remove SQL comments to avoid false positives in parsing
 */
function removeComments(query: string): string {
  // Remove single-line comments (-- comment)
  let cleaned = query.replace(/--[^\n]*/g, '')

  // Remove multi-line comments (/* comment */)
  cleaned = cleaned.replace(/\/\*[\s\S]*?\*\//g, '')

  return cleaned
}

/**
 * Check if a word is a table alias based on context
 */
export function isAlias(word: string, context: QueryContext): boolean {
  return context.aliasMap.has(word.toLowerCase())
}

/**
 * Resolve an alias to its table reference
 */
export function resolveAlias(alias: string, context: QueryContext): TableReference | undefined {
  return context.aliasMap.get(alias.toLowerCase())
}

/**
 * Get all tables that are in scope for the current position
 */
export function getTablesInScope(context: QueryContext): TableReference[] {
  // For now, return all tables in the query
  // Future: could be more sophisticated based on subquery scope
  return context.tables
}
