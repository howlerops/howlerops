/**
 * Shared utility functions for query stores
 */

import type {
  NormalisedRowsResult,
  QueryEditableColumn,
  QueryEditableMetadata,
  QueryResultRow,
} from './query-types'

export function generateRowId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function transformEditableColumn(raw: unknown): QueryEditableColumn {
  if (!raw || typeof raw !== 'object') {
    return {
      name: '',
      resultName: '',
      dataType: '',
      editable: false,
      primaryKey: false,
    }
  }

  const column = raw as Record<string, unknown>
  const name = typeof column.name === 'string'
    ? column.name
    : typeof column.Name === 'string'
      ? column.Name
      : ''

  const resultName = typeof column.resultName === 'string'
    ? column.resultName
    : typeof column.result_name === 'string'
      ? column.result_name
      : name

  return {
    name,
    resultName,
    dataType: typeof column.dataType === 'string'
      ? column.dataType
      : typeof column.data_type === 'string'
        ? column.data_type
        : '',
    editable: Boolean(column.editable),
    primaryKey: Boolean(column.primaryKey ?? column.primary_key),
    hasDefault: Boolean(column.hasDefault ?? column.has_default),
    defaultValue: column.defaultValue ?? column.default_value,
    defaultExpression: typeof column.defaultExpression === 'string'
      ? column.defaultExpression
      : typeof column.default_expression === 'string'
        ? column.default_expression
        : undefined,
    autoNumber: Boolean(column.autoNumber ?? column.auto_number),
    timeZone: Boolean(column.timeZone ?? column.time_zone),
    precision: typeof column.precision === 'number' ? column.precision : undefined,
  }
}

export function transformEditableMetadata(raw: unknown): QueryEditableMetadata | null {
  if (!raw || typeof raw !== 'object') {
    return null
  }

  const metadataRaw = raw as Record<string, unknown>

  const primaryKeys = Array.isArray(metadataRaw.primaryKeys)
    ? metadataRaw.primaryKeys.filter((value): value is string => typeof value === 'string')
    : Array.isArray(metadataRaw.primary_keys)
      ? metadataRaw.primary_keys.filter((value): value is string => typeof value === 'string')
      : []

  const metadata: QueryEditableMetadata = {
    enabled: Boolean(metadataRaw.enabled),
    reason: typeof metadataRaw.reason === 'string' ? metadataRaw.reason : undefined,
    schema: typeof metadataRaw.schema === 'string' ? metadataRaw.schema : undefined,
    table: typeof metadataRaw.table === 'string' ? metadataRaw.table : undefined,
    primaryKeys,
    columns: Array.isArray(metadataRaw.columns) ? metadataRaw.columns.map(transformEditableColumn) : [],
    pending: Boolean(metadataRaw.pending),
    jobId: (metadataRaw.jobId as string | undefined) ?? (metadataRaw.job_id as string | undefined),
    job_id: (metadataRaw.job_id as string | undefined) ?? (metadataRaw.jobId as string | undefined),
  }

  if (!metadata.primaryKeys.length && Array.isArray(metadataRaw.primary_keys)) {
    metadata.primaryKeys = metadataRaw.primary_keys.filter((value): value is string => typeof value === 'string')
  }

  if (!metadata.jobId && metadata.job_id) {
    metadata.jobId = metadata.job_id
  }
  if (!metadata.job_id && metadata.jobId) {
    metadata.job_id = metadata.jobId
  }

  const rawCapabilities = metadataRaw.capabilities as Record<string, unknown> | undefined
  const capabilitySource = typeof rawCapabilities === 'object' && rawCapabilities !== null ? rawCapabilities : undefined
  if (capabilitySource) {
    const canInsertValue = (capabilitySource as Record<string, unknown>).canInsert ?? (capabilitySource as Record<string, unknown>).can_insert
    const canUpdateValue = (capabilitySource as Record<string, unknown>).canUpdate ?? (capabilitySource as Record<string, unknown>).can_update
    const canDeleteValue = (capabilitySource as Record<string, unknown>).canDelete ?? (capabilitySource as Record<string, unknown>).can_delete
    metadata.capabilities = {
      canInsert: Boolean(canInsertValue),
      canUpdate: Boolean(canUpdateValue),
      canDelete: Boolean(canDeleteValue),
      reason: typeof capabilitySource.reason === 'string' ? capabilitySource.reason : undefined,
    }
  }

  return metadata
}

export function normaliseRows(
  columns: string[],
  rows: unknown[],
  metadata?: QueryEditableMetadata | null
): NormalisedRowsResult {
  if (!Array.isArray(rows)) {
    return {
      rows: [],
      originalRows: {},
    }
  }

  const processedRows: QueryResultRow[] = []
  const originalRows: Record<string, QueryResultRow> = {}

  const columnLookup: Record<string, string> = {}
  columns.forEach((name) => {
    columnLookup[name.toLowerCase()] = name
  })

  const primaryKeyColumns = (metadata?.primaryKeys || []).map((pk) => {
    return columnLookup[pk.toLowerCase()] ?? pk
  })

  const assignValue = (target: Record<string, unknown>, columnName: string, value: unknown) => {
    if (value && typeof value === 'object' && 'String' in value && 'Valid' in value) {
      const sqlValue = value as { String: unknown; Valid: boolean }
      target[columnName] = sqlValue.Valid ? sqlValue.String : null
    } else {
      target[columnName] = value
    }
  }

  rows.forEach((row, rowIndex) => {
    const record: Record<string, unknown> = {}

    if (Array.isArray(row)) {
      row.forEach((value, index) => {
        const columnName = columns[index] ?? `col_${index}`
        assignValue(record, columnName, value)
      })
    } else if (row && typeof row === 'object') {
      const rowObject = row as Record<string | number, unknown>
      columns.forEach((columnName, index) => {
        if (columnName in rowObject) {
          assignValue(record, columnName, rowObject[columnName])
        } else if (index in rowObject) {
          assignValue(record, columnName, rowObject[index])
        } else if (String(index) in rowObject) {
          assignValue(record, columnName, rowObject[String(index)])
        } else {
          assignValue(record, columnName, undefined)
        }
      })
    } else {
      columns.forEach((columnName) => assignValue(record, columnName, undefined))
    }

    let rowId = ''
    if (primaryKeyColumns.length > 0) {
      const parts: string[] = []
      let allPresent = true
      primaryKeyColumns.forEach((pkColumn) => {
        const value = record[pkColumn]
        if (value === undefined) {
          allPresent = false
        } else {
          const serialised =
            value === null || value === undefined ? 'NULL' : String(value)
          parts.push(`${pkColumn}:${serialised}`)
        }
      })
      if (allPresent && parts.length > 0) {
        rowId = parts.join('|')
      }
    }

    if (!rowId) {
      rowId = `${generateRowId()}-${rowIndex}`
    }

    const completeRow: QueryResultRow = {
      ...record,
      __rowId: rowId,
    }

    processedRows.push(completeRow)
    originalRows[rowId] = { ...completeRow }
  })

  return {
    rows: processedRows,
    originalRows,
  }
}

export function parseDurationMs(duration?: string): number {
  if (!duration) return 0
  const value = duration.toLowerCase()

  if (value.endsWith('ms')) {
    return parseFloat(value.replace('ms', ''))
  }
  if (value.endsWith('s')) {
    return parseFloat(value.replace('s', '')) * 1000
  }
  if (value.endsWith('µs') || value.endsWith('us')) {
    return parseFloat(value.replace('µs', '').replace('us', '')) / 1000
  }
  if (value.endsWith('ns')) {
    return parseFloat(value.replace('ns', '')) / 1e6
  }

  const parsed = parseFloat(value)
  return Number.isNaN(parsed) ? 0 : parsed
}
