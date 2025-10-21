interface SqlStatementSegment {
  text: string
  start: number
  end: number
}

const BLOCK_COMMENT_REGEX = /\/\*[\s\S]*?\*\//g

const COMMENT_LINE_PREFIXES = ['--', '#']

const matchDollarTag = (source: string, start: number): string | null => {
  const remaining = source.slice(start)
  const match = /^\$[A-Za-z0-9_]*\$/.exec(remaining)
  return match ? match[0] : null
}

const pushStatement = (source: string, statements: SqlStatementSegment[], segmentStart: number, segmentEnd: number) => {
  if (segmentEnd <= segmentStart) {
    return
  }

  const rawSegment = source.slice(segmentStart, segmentEnd)
  const trimmed = rawSegment.trim()
  if (!trimmed) {
    return
  }

  const leadingWhitespaceLength = rawSegment.length - rawSegment.trimStart().length
  const trailingWhitespaceLength = rawSegment.length - rawSegment.trimEnd().length

  statements.push({
    text: trimmed,
    start: segmentStart + leadingWhitespaceLength,
    end: segmentEnd - trailingWhitespaceLength,
  })
}

const sanitizeStatementText = (statement: string): string => {
  if (!statement) {
    return ''
  }

  // Remove block comments
  let sanitized = statement.replace(BLOCK_COMMENT_REGEX, ' ')

  // Remove comment-only lines and trim trailing comment markers within a line
  const lines = sanitized
    .split('\n')
    .map((line) => {
      const trimmed = line.trim()
      if (!trimmed) {
        return ''
      }

      if (COMMENT_LINE_PREFIXES.some(prefix => trimmed.startsWith(prefix))) {
        return ''
      }

      return line
    })
    .filter(Boolean)

  sanitized = lines.join('\n')

  return sanitized.trim()
}

const parseStatements = (source: string): SqlStatementSegment[] => {
  const statements: SqlStatementSegment[] = []
  const length = source.length

  let index = 0
  let segmentStart = 0

  let inSingleQuote = false
  let inDoubleQuote = false
  let inDollarTag: string | null = null
  let inLineComment = false
  let inBlockComment = false

  while (index < length) {
    const char = source[index]
    const nextChar = index + 1 < length ? source[index + 1] : ''

    if (inLineComment) {
      if (char === '\n' || char === '\r') {
        inLineComment = false
      }
      index += 1
      continue
    }

    if (inBlockComment) {
      if (char === '*' && nextChar === '/') {
        inBlockComment = false
        index += 2
      } else {
        index += 1
      }
      continue
    }

    if (inDollarTag) {
      if (source.startsWith(inDollarTag, index)) {
        index += inDollarTag.length
        inDollarTag = null
      } else {
        index += 1
      }
      continue
    }

    if (inSingleQuote) {
      if (char === "'") {
        if (nextChar === "'") {
          index += 2
          continue
        }
        inSingleQuote = false
      }
      index += 1
      continue
    }

    if (inDoubleQuote) {
      if (char === '"') {
        if (nextChar === '"') {
          index += 2
          continue
        }
        inDoubleQuote = false
      }
      index += 1
      continue
    }

    if (char === '-' && nextChar === '-') {
      inLineComment = true
      index += 2
      continue
    }

    if (char === '/' && nextChar === '*') {
      inBlockComment = true
      index += 2
      continue
    }

    if (char === "'") {
      inSingleQuote = true
      index += 1
      continue
    }

    if (char === '"') {
      inDoubleQuote = true
      index += 1
      continue
    }

    if (char === '$') {
      const tag = matchDollarTag(source, index)
      if (tag) {
        inDollarTag = tag
        index += tag.length
        continue
      }
    }

    if (char === ';') {
      pushStatement(source, statements, segmentStart, index)
      segmentStart = index + 1
    }

    index += 1
  }

  pushStatement(source, statements, segmentStart, length)

  return statements
}

const findStatementIndexForCursor = (statements: SqlStatementSegment[], cursor: number): number => {
  if (!statements.length) {
    return -1
  }

  const strictMatch = statements.findIndex((statement) => cursor >= statement.start && cursor <= statement.end)
  if (strictMatch !== -1) {
    return strictMatch
  }

  const nextMatch = statements.findIndex((statement) => cursor <= statement.start)
  if (nextMatch !== -1) {
    return nextMatch
  }

  return statements.length - 1
}

const getSanitizedStatementAtIndex = (statements: SqlStatementSegment[], index: number): string | null => {
  if (index < 0 || index >= statements.length) {
    return null
  }

  const cleaned = sanitizeStatementText(statements[index].text)
  if (cleaned) {
    return cleaned
  }

  // If the targeted statement was effectively empty after sanitisation, try to find the next meaningful statement
  for (let i = index + 1; i < statements.length; i += 1) {
    const nextCleaned = sanitizeStatementText(statements[i].text)
    if (nextCleaned) {
      return nextCleaned
    }
  }

  for (let i = index - 1; i >= 0; i -= 1) {
    const previousCleaned = sanitizeStatementText(statements[i].text)
    if (previousCleaned) {
      return previousCleaned
    }
  }

  return null
}

export const buildExecutableSql = (
  documentText: string,
  options: {
    selectionText?: string
    cursorOffset: number
  }
): string | null => {
  const { selectionText, cursorOffset } = options

  if (selectionText) {
    const cleanedSelection = sanitizeStatementText(selectionText)
    if (cleanedSelection) {
      return cleanedSelection
    }
  }

  const statements = parseStatements(documentText)
  if (!statements.length) {
    return null
  }

  const targetIndex = findStatementIndexForCursor(statements, cursorOffset)
  return getSanitizedStatementAtIndex(statements, targetIndex)
}

export const collapseWhitespace = (sql: string): string =>
  sql
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
    .join('\n')
