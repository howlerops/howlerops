import { CellValue } from '../types/table'

export interface JsonToken {
  type: 'key' | 'string' | 'number' | 'boolean' | 'null' | 'punctuation' | 'whitespace'
  value: string
  start: number
  end: number
}

export interface FormattedJson {
  tokens: JsonToken[]
  formatted: string
  hasCircularRefs: boolean
}

/**
 * Format JSON with proper indentation and syntax highlighting
 */
export function formatJson(data: unknown, indent = 2): FormattedJson {
  const tokens: JsonToken[] = []
  let formatted = ''
  let hasCircularRefs = false
  
  try {
    // Check for circular references
    const seen = new WeakSet()
    const checkCircular = (obj: unknown): boolean => {
      if (obj && typeof obj === 'object') {
        if (seen.has(obj)) return true
        seen.add(obj)
        for (const value of Object.values(obj)) {
          if (checkCircular(value)) return true
        }
        seen.delete(obj)
      }
      return false
    }
    
    hasCircularRefs = checkCircular(data)
    
    // Format with proper indentation
    formatted = JSON.stringify(data, null, indent)
    
    // Tokenize the formatted JSON
    tokens.push(...tokenizeJson(formatted))
    
  } catch (error) {
    // Fallback for circular references or other issues
    formatted = '[Circular Reference]'
    tokens.push({
      type: 'string',
      value: formatted,
      start: 0,
      end: formatted.length
    })
  }
  
  return { tokens, formatted, hasCircularRefs }
}

/**
 * Tokenize JSON string for syntax highlighting
 */
function tokenizeJson(json: string): JsonToken[] {
  const tokens: JsonToken[] = []
  let i = 0
  
  while (i < json.length) {
    const char = json[i]
    
    if (/\s/.test(char)) {
      // Whitespace
      let whitespace = ''
      while (i < json.length && /\s/.test(json[i])) {
        whitespace += json[i]
        i++
      }
      tokens.push({
        type: 'whitespace',
        value: whitespace,
        start: i - whitespace.length,
        end: i
      })
    } else if (char === '"') {
      // String
      let string = '"'
      i++
      while (i < json.length && json[i] !== '"') {
        if (json[i] === '\\' && i + 1 < json.length) {
          string += json[i] + json[i + 1]
          i += 2
        } else {
          string += json[i]
          i++
        }
      }
      if (i < json.length) {
        string += '"'
        i++
      }
      tokens.push({
        type: 'string',
        value: string,
        start: i - string.length,
        end: i
      })
    } else if (char === ':') {
      // Colon (key-value separator)
      tokens.push({
        type: 'punctuation',
        value: ':',
        start: i,
        end: i + 1
      })
      i++
    } else if (char === ',' || char === '{' || char === '}' || char === '[' || char === ']') {
      // Punctuation
      tokens.push({
        type: 'punctuation',
        value: char,
        start: i,
        end: i + 1
      })
      i++
    } else if (char === 't' || char === 'f') {
      // Boolean (true/false)
      const keyword = char === 't' ? 'true' : 'false'
      if (json.slice(i, i + keyword.length) === keyword) {
        tokens.push({
          type: 'boolean',
          value: keyword,
          start: i,
          end: i + keyword.length
        })
        i += keyword.length
      } else {
        tokens.push({
          type: 'string',
          value: char,
          start: i,
          end: i + 1
        })
        i++
      }
    } else if (char === 'n') {
      // Null
      if (json.slice(i, i + 4) === 'null') {
        tokens.push({
          type: 'null',
          value: 'null',
          start: i,
          end: i + 4
        })
        i += 4
      } else {
        tokens.push({
          type: 'string',
          value: char,
          start: i,
          end: i + 1
        })
        i++
      }
    } else if (/\d/.test(char) || char === '-') {
      // Number
      let number = ''
      while (i < json.length && (/\d/.test(json[i]) || json[i] === '.' || json[i] === 'e' || json[i] === 'E' || json[i] === '+' || json[i] === '-')) {
        number += json[i]
        i++
      }
      tokens.push({
        type: 'number',
        value: number,
        start: i - number.length,
        end: i
      })
    } else {
      // Unknown character
      tokens.push({
        type: 'string',
        value: char,
        start: i,
        end: i + 1
      })
      i++
    }
  }
  
  return tokens
}

/**
 * Convert table row data to JSON with special handling for database types
 */
export function rowToJson(row: Record<string, CellValue>, columns?: string[]): Record<string, unknown> {
  const json: Record<string, unknown> = {}
  
  const keys = columns || Object.keys(row)
  
  for (const key of keys) {
    if (key === '__rowId') continue
    
    const value = row[key]
    json[key] = formatCellValue(value)
  }
  
  return json
}

/**
 * Format cell value for JSON display
 */
function formatCellValue(value: CellValue): unknown {
  if (value === null) return null
  if (value === undefined) return undefined
  
  // Handle Date objects
  if (value instanceof Date) {
    return {
      _type: 'date',
      _value: value.toISOString(),
      _display: value.toLocaleString()
    }
  }
  
  // Handle Buffer objects (if they exist in the environment)
  if (value && typeof value === 'object' && 'buffer' in value) {
    return {
      _type: 'buffer',
      _value: '[Buffer]',
      _display: `Buffer(${value.length || 0} bytes)`
    }
  }
  
  // Handle objects that might be JSON strings
  if (typeof value === 'string' && (value.startsWith('{') || value.startsWith('['))) {
    try {
      const parsed = JSON.parse(value)
      return {
        _type: 'json',
        _value: parsed,
        _display: value
      }
    } catch {
      // Not valid JSON, treat as regular string
    }
  }
  
  return value
}

/**
 * Get CSS class for token type
 */
export function getTokenClass(token: JsonToken): string {
  switch (token.type) {
    case 'key':
      return 'text-blue-600 font-medium'
    case 'string':
      return 'text-green-600'
    case 'number':
      return 'text-purple-600'
    case 'boolean':
      return 'text-orange-600'
    case 'null':
      return 'text-gray-500'
    case 'punctuation':
      return 'text-gray-700'
    case 'whitespace':
      return ''
    default:
      return 'text-gray-900'
  }
}

/**
 * Collapse JSON to single line for compact display
 */
export function collapseJson(data: unknown): string {
  try {
    return JSON.stringify(data)
  } catch {
    return '[Circular Reference]'
  }
}

/**
 * Expand JSON with proper formatting
 */
export function expandJson(data: unknown, indent = 2): string {
  try {
    return JSON.stringify(data, null, indent)
  } catch {
    return '[Circular Reference]'
  }
}
