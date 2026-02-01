import type { QueryEditableColumn, QueryEditableMetadata, QueryResultRow } from '../../store/query-store'
import type { TableColumn } from '../../types/table'
import type { ColumnDisplayTraits } from './types'

type EditableColumnMeta = QueryEditableMetadata['columns'] extends Array<infer C> ? C : never

export const inferColumnType = (dataType?: string): TableColumn['type'] => {
  if (!dataType) return 'text'
  const normalized = dataType.toLowerCase()

  if (normalized.includes('int') || normalized.includes('numeric') || normalized.includes('decimal') || normalized.includes('double') || normalized.includes('real')) {
    return 'number'
  }
  if (normalized.includes('bool')) {
    return 'boolean'
  }
  if (normalized.includes('timestamp') || normalized.includes('time')) {
    return 'datetime'
  }
  if (normalized.includes('date')) {
    return 'date'
  }

  return 'text'
}

export const deriveColumnDisplayTraits = (
  columnName: string,
  metaColumn: EditableColumnMeta | undefined,
  columnType: TableColumn['type']
): ColumnDisplayTraits => {
  const normalizedName = columnName.toLowerCase()
  const dataType = metaColumn?.dataType?.toLowerCase() ?? ''
  const precision = typeof metaColumn?.precision === 'number' ? metaColumn.precision : undefined

  const isUUIDLike =
    dataType.includes('uuid') ||
    normalizedName.endsWith('_uuid') ||
    normalizedName.endsWith('_guid') ||
    ((normalizedName === 'id' || normalizedName.endsWith('_id')) && (precision ?? 0) >= 24)

  const isJsonLike = dataType.includes('json')
  const isTextLike = dataType.includes('text') || dataType.includes('clob') || dataType.includes('xml')
  const isBinaryLike = dataType.includes('blob') || dataType.includes('binary')
  const isLongCharacter = typeof precision === 'number' && precision >= 512
  const isNumeric = columnType === 'number'
  const isTemporal = columnType === 'datetime' || columnType === 'date'
  const isBoolean = columnType === 'boolean'

  const longText = isJsonLike || isTextLike || isBinaryLike || isLongCharacter
  const wrapContent = isUUIDLike
  const monospace = isUUIDLike || isTemporal || isNumeric

  const minWidth = longText
    ? 220
    : isUUIDLike
      ? 240
      : isTemporal
        ? 200
        : isNumeric
          ? 150
          : isBoolean
            ? 110
            : 120

  const maxWidth = longText
    ? 620
    : isUUIDLike
      ? 460
      : undefined

  const preferredWidth = longText
    ? 520
    : isUUIDLike
      ? 320
      : isTemporal
        ? 280
        : undefined

  return {
    minWidth,
    maxWidth,
    preferredWidth,
    longText,
    wrapContent,
    clipContent: !wrapContent,
    monospace,
  }
}

export const formatTimestamp = (value: Date): string => value.toLocaleString()

export const createRowId = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export const serialiseCsvValue = (value: unknown): string => {
  if (value === null || value === undefined) {
    return ''
  }

  const stringValue = String(value)
  if (stringValue.includes('"') || stringValue.includes(',') || stringValue.includes('\n')) {
    return `"${stringValue.replace(/"/g, '""')}"`
  }
  return stringValue
}

export const buildPrimaryKeyMap = (
  row: QueryResultRow,
  metadata?: QueryEditableMetadata | null,
  columnsLookup?: Record<string, string>
): Record<string, unknown> | null => {
  if (!metadata?.primaryKeys?.length) {
    return null
  }

  const primaryKey: Record<string, unknown> = {}
  let allPresent = true

  metadata.primaryKeys.forEach((pk) => {
    const resultColumnName =
      columnsLookup?.[pk.toLowerCase()] ??
      metadata.columns?.find((col) => (col.name ?? col.resultName)?.toLowerCase() === pk.toLowerCase())?.resultName ??
      pk
    const value = row[resultColumnName]
    if (value === undefined) {
      allPresent = false
    } else {
      primaryKey[pk] = value
    }
  })

  return allPresent ? primaryKey : null
}

export const buildColumnsLookup = (metadata?: QueryEditableMetadata | null): Record<string, string> => {
  const lookup: Record<string, string> = {}
  metadata?.columns?.forEach((column) => {
    const baseName = column.name ?? column.resultName
    if (!baseName) return
    const key = baseName.toLowerCase()
    const resultName = column.resultName ?? column.name ?? baseName
    lookup[key] = resultName
  })
  return lookup
}

export const buildMetadataLookup = (metadata?: QueryEditableMetadata | null): Map<string, QueryEditableColumn> => {
  const map = new Map<string, QueryEditableColumn>()
  metadata?.columns?.forEach((column) => {
    const key = (column.resultName ?? column.name ?? '').toLowerCase()
    if (!key) {
      return
    }
    map.set(key, column)
  })
  return map
}
