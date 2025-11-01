import { JsonToken } from './json-formatter'

export interface SearchMatch {
  token: JsonToken
  matchIndex: number
  matchLength: number
  isKey: boolean
  path: string
}

export interface SearchResult {
  matches: SearchMatch[]
  totalMatches: number
  currentMatchIndex: number
}

/**
 * Search through JSON tokens for key names or values
 */
export function searchJson(
  tokens: JsonToken[],
  query: string,
  options: {
    caseSensitive?: boolean
    useRegex?: boolean
    searchKeys?: boolean
    searchValues?: boolean
  } = {}
): SearchResult {
  const {
    caseSensitive = false,
    useRegex = false,
    searchKeys = true,
    searchValues = true
  } = options

  const matches: SearchMatch[] = []
  let searchPattern: RegExp

  try {
    if (useRegex) {
      // Remove leading/trailing slashes if present
      const regexPattern = query.startsWith('/') && query.endsWith('/') 
        ? query.slice(1, -1) 
        : query
      searchPattern = new RegExp(regexPattern, caseSensitive ? 'g' : 'gi')
    } else {
      const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      searchPattern = new RegExp(escapedQuery, caseSensitive ? 'g' : 'gi')
    }
  } catch {
    // Invalid regex, return empty results
    return {
      matches: [],
      totalMatches: 0,
      currentMatchIndex: -1
    }
  }

  let currentPath = ''
  let inKey = false

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]

    // Track current path for context
    if (token.type === 'punctuation') {
      if (token.value === '{') {
        currentPath += '.'
      } else if (token.value === '}') {
        currentPath = currentPath.replace(/\.?[^.]*$/, '')
      } else if (token.value === '[') {
        currentPath += '[]'
      } else if (token.value === ']') {
        currentPath = currentPath.replace(/\[\]$/, '')
      } else if (token.value === ':') {
        inKey = false
      }
    } else if (token.type === 'string' && tokens[i + 1]?.type === 'punctuation' && tokens[i + 1]?.value === ':') {
      // This is a key
      inKey = true
      currentPath = currentPath.replace(/\.?[^.]*$/, '') + '.' + token.value.slice(1, -1)
    }

    // Search in keys
    if (searchKeys && inKey && token.type === 'string') {
      const matches_in_token = findMatchesInToken(token, searchPattern, true, currentPath)
      matches.push(...matches_in_token)
    }
    // Search in values
    else if (searchValues && !inKey && (token.type === 'string' || token.type === 'number' || token.type === 'boolean' || token.type === 'null')) {
      const matches_in_token = findMatchesInToken(token, searchPattern, false, currentPath)
      matches.push(...matches_in_token)
    }
  }

  return {
    matches,
    totalMatches: matches.length,
    currentMatchIndex: matches.length > 0 ? 0 : -1
  }
}

/**
 * Find matches within a single token
 */
function findMatchesInToken(
  token: JsonToken,
  pattern: RegExp,
  isKey: boolean,
  path: string
): SearchMatch[] {
  const matches: SearchMatch[] = []
  const text = token.value
  let match: RegExpExecArray | null

  // Reset regex lastIndex to ensure we start from the beginning
  pattern.lastIndex = 0

  while ((match = pattern.exec(text)) !== null) {
    matches.push({
      token,
      matchIndex: match.index,
      matchLength: match[0].length,
      isKey,
      path
    })

    // Prevent infinite loop on zero-length matches
    if (match[0].length === 0) {
      pattern.lastIndex++
    }
  }

  return matches
}

/**
 * Navigate to next match
 */
export function getNextMatch(result: SearchResult): SearchResult {
  if (result.totalMatches === 0) return result

  const nextIndex = (result.currentMatchIndex + 1) % result.totalMatches
  return {
    ...result,
    currentMatchIndex: nextIndex
  }
}

/**
 * Navigate to previous match
 */
export function getPreviousMatch(result: SearchResult): SearchResult {
  if (result.totalMatches === 0) return result

  const prevIndex = result.currentMatchIndex === 0 
    ? result.totalMatches - 1 
    : result.currentMatchIndex - 1

  return {
    ...result,
    currentMatchIndex: prevIndex
  }
}

/**
 * Get current match
 */
export function getCurrentMatch(result: SearchResult): SearchMatch | null {
  if (result.currentMatchIndex < 0 || result.currentMatchIndex >= result.totalMatches) {
    return null
  }
  return result.matches[result.currentMatchIndex]
}

/**
 * Highlight matches in JSON string
 */
export function highlightMatches(
  json: string,
  matches: SearchMatch[],
  currentMatchIndex: number
): string {
  if (matches.length === 0) return json

  // Sort matches by position (descending) to avoid index shifting
  const sortedMatches = [...matches].sort((a, b) => b.token.start + b.matchIndex - (a.token.start + a.matchIndex))
  
  let highlighted = json

  sortedMatches.forEach((match, index) => {
    const isCurrentMatch = index === currentMatchIndex
    const start = match.token.start + match.matchIndex
    const end = start + match.matchLength
    
    const before = highlighted.slice(0, start)
    const matchText = highlighted.slice(start, end)
    const after = highlighted.slice(end)
    
    const highlightClass = isCurrentMatch ? 'bg-yellow-300' : 'bg-yellow-200'
    const highlightedMatch = `<span class="${highlightClass}">${matchText}</span>`
    
    highlighted = before + highlightedMatch + after
  })

  return highlighted
}

/**
 * Extract key path from JSON path
 */
export function getKeyPath(path: string): string {
  return path.replace(/^\./, '').replace(/\[\]/g, '')
}

/**
 * Check if query is a regex pattern
 */
export function isRegexQuery(query: string): boolean {
  return query.startsWith('/') && query.endsWith('/') && query.length > 2
}

/**
 * Validate regex pattern
 */
export function validateRegex(pattern: string): { isValid: boolean; error?: string } {
  try {
    new RegExp(pattern)
    return { isValid: true }
  } catch (error) {
    return { 
      isValid: false, 
      error: error instanceof Error ? error.message : 'Invalid regex pattern' 
    }
  }
}

/**
 * Escape special regex characters for literal search
 */
export function escapeRegex(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Get search statistics
 */
export function getSearchStats(result: SearchResult): {
  totalMatches: number
  currentMatch: number
  hasMatches: boolean
} {
  return {
    totalMatches: result.totalMatches,
    currentMatch: result.currentMatchIndex + 1,
    hasMatches: result.totalMatches > 0
  }
}
