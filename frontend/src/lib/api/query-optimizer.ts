import { apiClient } from './client'

interface AnalysisResult {
  suggestions: Suggestion[]
  score: number
  warnings: Warning[]
  complexity: string
  estimated_cost: number
}

interface Suggestion {
  type: string
  severity: 'info' | 'warning' | 'critical'
  message: string
  original_sql?: string
  improved_sql?: string
  impact?: string
}

interface Warning {
  message: string
  severity: 'low' | 'medium' | 'high'
}

interface ConversionResult {
  sql: string
  confidence: number
  template: string
  suggestions?: string[]
}

interface AutocompleteSuggestion {
  text: string
  type: 'table' | 'column' | 'keyword' | 'function' | 'snippet'
  description?: string
  insert_text?: string
  detail?: string
}

interface ExplanationResult {
  explanation?: string
  summary?: string
  type?: string
  tables?: string[]
  complexity?: string
  operations?: string[]
  warnings?: string[]
  suggestions?: string[]
}

/**
 * Analyze a SQL query for optimization opportunities
 */
export async function analyzeQuery(
  sql: string,
  connectionId?: string
): Promise<AnalysisResult> {
  const response = await apiClient.post('/api/query/analyze', {
    sql,
    connection_id: connectionId,
  })

  return response.data
}

/**
 * Convert natural language to SQL
 */
export async function nl2sql(
  query: string,
  connectionId?: string
): Promise<ConversionResult> {
  const response = await apiClient.post('/api/query/nl2sql', {
    query,
    connection_id: connectionId,
  })

  return response.data
}

/**
 * Get autocomplete suggestions for SQL
 */
export async function getAutocompleteSuggestions(
  sql: string,
  cursorPos: number,
  connectionId?: string
): Promise<AutocompleteSuggestion[]> {
  const response = await apiClient.post('/api/query/autocomplete', {
    sql,
    cursor: cursorPos,
    connection_id: connectionId,
  })

  return response.data.suggestions || []
}

/**
 * Explain what a SQL query does in plain English
 */
export async function explainQuery(
  sql: string,
  verbose: boolean = false
): Promise<ExplanationResult> {
  const response = await apiClient.post('/api/query/explain', {
    sql,
    verbose,
  })

  return response.data
}

/**
 * Get supported natural language patterns
 */
export async function getSupportedPatterns(): Promise<{
  patterns: Array<{
    description: string
    examples: string[]
    pattern: string
  }>
  total: number
}> {
  const response = await apiClient.get('/api/query/patterns')
  return response.data
}

/**
 * Helper to format SQL with proper indentation
 */
export function formatSQL(sql: string): string {
  // Basic SQL formatting
  const keywords = [
    'SELECT',
    'FROM',
    'WHERE',
    'JOIN',
    'LEFT JOIN',
    'RIGHT JOIN',
    'INNER JOIN',
    'GROUP BY',
    'ORDER BY',
    'HAVING',
    'LIMIT',
    'OFFSET',
    'UNION',
    'INSERT INTO',
    'VALUES',
    'UPDATE',
    'SET',
    'DELETE FROM',
  ]

  let formatted = sql
  keywords.forEach((keyword) => {
    const regex = new RegExp(`\\b${keyword}\\b`, 'gi')
    formatted = formatted.replace(regex, `\n${keyword}`)
  })

  // Clean up extra newlines and trim
  formatted = formatted
    .split('\n')
    .filter((line) => line.trim())
    .map((line) => line.trim())
    .join('\n')

  return formatted
}

/**
 * Check if a query is likely to be expensive
 */
export function isExpensiveQuery(sql: string): boolean {
  const upper = sql.toUpperCase()

  // Check for patterns that indicate expensive queries
  const expensivePatterns = [
    /SELECT\s+\*/, // SELECT *
    /WHERE\s+\w+\s+LIKE\s+'%/, // Leading wildcard
    /NOT\s+IN\s*\(.*SELECT/, // NOT IN with subquery
    /CROSS\s+JOIN/, // Cross join
    /ORDER\s+BY\s+RAND/, // Random ordering
    /DISTINCT/, // DISTINCT can be expensive
  ]

  return expensivePatterns.some((pattern) => pattern.test(upper))
}

/**
 * Get query type from SQL
 */
export function getQueryType(sql: string): string {
  const upper = sql.trim().toUpperCase()

  if (upper.startsWith('SELECT')) return 'SELECT'
  if (upper.startsWith('INSERT')) return 'INSERT'
  if (upper.startsWith('UPDATE')) return 'UPDATE'
  if (upper.startsWith('DELETE')) return 'DELETE'
  if (upper.startsWith('CREATE')) return 'CREATE'
  if (upper.startsWith('ALTER')) return 'ALTER'
  if (upper.startsWith('DROP')) return 'DROP'
  if (upper.startsWith('TRUNCATE')) return 'TRUNCATE'

  return 'UNKNOWN'
}

/**
 * Estimate query execution time (rough estimate)
 */
export function estimateExecutionTime(
  sql: string,
  rowCount?: number
): { min: number; max: number; unit: string } {
  const type = getQueryType(sql)
  const isExpensive = isExpensiveQuery(sql)
  const hasJoins = /JOIN/i.test(sql)
  const hasSubquery = /SELECT.*\(.*SELECT/i.test(sql)

  // Base estimates in milliseconds
  let minTime = 1
  let maxTime = 10

  if (type === 'SELECT') {
    if (isExpensive) {
      minTime = 100
      maxTime = 5000
    } else if (hasJoins) {
      minTime = 10
      maxTime = 500
    } else if (hasSubquery) {
      minTime = 20
      maxTime = 1000
    } else {
      minTime = 1
      maxTime = 50
    }
  } else if (type === 'INSERT' || type === 'UPDATE' || type === 'DELETE') {
    minTime = 5
    maxTime = 100
  }

  // Adjust for row count if provided
  if (rowCount) {
    if (rowCount > 1000000) {
      minTime *= 10
      maxTime *= 20
    } else if (rowCount > 100000) {
      minTime *= 5
      maxTime *= 10
    } else if (rowCount > 10000) {
      minTime *= 2
      maxTime *= 5
    }
  }

  // Determine unit
  let unit = 'ms'
  if (maxTime >= 60000) {
    minTime /= 60000
    maxTime /= 60000
    unit = 'min'
  } else if (maxTime >= 1000) {
    minTime /= 1000
    maxTime /= 1000
    unit = 's'
  }

  return {
    min: Math.round(minTime),
    max: Math.round(maxTime),
    unit,
  }
}