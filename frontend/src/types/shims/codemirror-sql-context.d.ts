declare interface QueryContext {
  currentClause?: 'SELECT' | 'FROM' | 'WHERE' | 'JOIN' | 'ON' | 'GROUP BY' | 'ORDER BY' | string
}

declare interface JoinOnContext {
  leftColumn: string
}
