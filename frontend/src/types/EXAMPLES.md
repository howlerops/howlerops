# Type Usage Examples

Practical examples of using the Schema Diff and Data Catalog types in React components.

## Schema Diff Examples

### Snapshot Management Component

```typescript
import React, { useState, useEffect } from 'react'
import {
  type SnapshotMetadata,
  type SnapshotListFilter,
  type SnapshotListSort,
  createSnapshotId
} from '@/types/schema-diff'
import {
  ListSchemaSnapshots,
  CreateSchemaSnapshot,
  DeleteSchemaSnapshot
} from '@/wailsjs/go/main/App'

export function SnapshotManager() {
  const [snapshots, setSnapshots] = useState<SnapshotMetadata[]>([])
  const [filter, setFilter] = useState<SnapshotListFilter>({})
  const [sort, setSort] = useState<SnapshotListSort>({
    field: 'createdAt',
    direction: 'desc'
  })

  useEffect(() => {
    loadSnapshots()
  }, [filter, sort])

  async function loadSnapshots() {
    const data = await ListSchemaSnapshots()

    // Apply filters
    let filtered = data
    if (filter.connectionId) {
      filtered = filtered.filter(s => s.connection_id === filter.connectionId)
    }
    if (filter.searchQuery) {
      const query = filter.searchQuery.toLowerCase()
      filtered = filtered.filter(s =>
        s.name.toLowerCase().includes(query)
      )
    }

    // Apply sorting
    filtered.sort((a, b) => {
      const aVal = a[sort.field]
      const bVal = b[sort.field]
      const direction = sort.direction === 'asc' ? 1 : -1
      return aVal > bVal ? direction : -direction
    })

    setSnapshots(filtered)
  }

  async function createSnapshot(connectionId: string, name: string) {
    const metadata = await CreateSchemaSnapshot(connectionId, name)
    await loadSnapshots()
    return createSnapshotId(metadata.id)
  }

  async function deleteSnapshot(id: string) {
    await DeleteSchemaSnapshot(id)
    await loadSnapshots()
  }

  return (
    <div>
      <SnapshotFilters filter={filter} onFilterChange={setFilter} />
      <SnapshotSort sort={sort} onSortChange={setSort} />
      <SnapshotList
        snapshots={snapshots}
        onDelete={deleteSnapshot}
      />
    </div>
  )
}
```

### Schema Comparison Component

```typescript
import React, { useState } from 'react'
import {
  type SchemaDiff,
  type TableDiff,
  type ColumnDiff,
  type DiffStatus,
  type DiffStatsBreakdown
} from '@/types/schema-diff'
import { CompareConnectionSchemas } from '@/wailsjs/go/main/App'

interface SchemaComparisonProps {
  sourceId: string
  targetId: string
}

export function SchemaComparison({ sourceId, targetId }: SchemaComparisonProps) {
  const [diff, setDiff] = useState<SchemaDiff | null>(null)
  const [loading, setLoading] = useState(false)
  const [selectedTable, setSelectedTable] = useState<string | null>(null)

  async function runComparison() {
    setLoading(true)
    try {
      const result = await CompareConnectionSchemas(sourceId, targetId)
      setDiff(result)
    } finally {
      setLoading(false)
    }
  }

  function calculateStats(tables: TableDiff[]): DiffStatsBreakdown {
    const total = tables.length
    const counts = {
      added: tables.filter(t => t.status === 'added').length,
      removed: tables.filter(t => t.status === 'removed').length,
      modified: tables.filter(t => t.status === 'modified').length,
      unchanged: tables.filter(t => t.status === 'unchanged').length
    }

    return {
      added: { count: counts.added, percentage: (counts.added / total) * 100 },
      removed: { count: counts.removed, percentage: (counts.removed / total) * 100 },
      modified: { count: counts.modified, percentage: (counts.modified / total) * 100 },
      unchanged: { count: counts.unchanged, percentage: (counts.unchanged / total) * 100 },
      total
    }
  }

  function getStatusIcon(status: DiffStatus): string {
    switch (status) {
      case 'added': return '+'
      case 'removed': return '-'
      case 'modified': return '~'
      case 'unchanged': return '='
    }
  }

  function getStatusColor(status: DiffStatus): string {
    switch (status) {
      case 'added': return 'text-green-600'
      case 'removed': return 'text-red-600'
      case 'modified': return 'text-yellow-600'
      case 'unchanged': return 'text-gray-400'
    }
  }

  if (!diff) {
    return (
      <button onClick={runComparison} disabled={loading}>
        {loading ? 'Comparing...' : 'Compare Schemas'}
      </button>
    )
  }

  const stats = calculateStats(diff.tables)

  return (
    <div className="space-y-4">
      {/* Summary Stats */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard
          label="Added"
          value={diff.summary.tablesAdded}
          color="green"
        />
        <StatCard
          label="Removed"
          value={diff.summary.tablesRemoved}
          color="red"
        />
        <StatCard
          label="Modified"
          value={diff.summary.tablesModified}
          color="yellow"
        />
        <StatCard
          label="Duration"
          value={`${diff.duration.toFixed(2)}ms`}
          color="blue"
        />
      </div>

      {/* Table List */}
      <div className="space-y-2">
        {diff.tables.map(table => (
          <TableDiffItem
            key={`${table.schema}.${table.name}`}
            table={table}
            isSelected={selectedTable === table.name}
            onSelect={() => setSelectedTable(table.name)}
            getStatusIcon={getStatusIcon}
            getStatusColor={getStatusColor}
          />
        ))}
      </div>

      {/* Detailed Column Diff */}
      {selectedTable && (
        <ColumnDiffPanel
          table={diff.tables.find(t => t.name === selectedTable)!}
          getStatusIcon={getStatusIcon}
          getStatusColor={getStatusColor}
        />
      )}
    </div>
  )
}

interface TableDiffItemProps {
  table: TableDiff
  isSelected: boolean
  onSelect: () => void
  getStatusIcon: (status: DiffStatus) => string
  getStatusColor: (status: DiffStatus) => string
}

function TableDiffItem({
  table,
  isSelected,
  onSelect,
  getStatusIcon,
  getStatusColor
}: TableDiffItemProps) {
  const columnChanges = table.columns?.filter(c => c.status !== 'unchanged').length ?? 0

  return (
    <div
      className={`p-3 border rounded cursor-pointer ${isSelected ? 'ring-2 ring-blue-500' : ''}`}
      onClick={onSelect}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={getStatusColor(table.status)}>
            {getStatusIcon(table.status)}
          </span>
          <span className="font-medium">
            {table.schema}.{table.name}
          </span>
        </div>
        {columnChanges > 0 && (
          <span className="text-sm text-gray-500">
            {columnChanges} column changes
          </span>
        )}
      </div>
    </div>
  )
}

interface ColumnDiffPanelProps {
  table: TableDiff
  getStatusIcon: (status: DiffStatus) => string
  getStatusColor: (status: DiffStatus) => string
}

function ColumnDiffPanel({
  table,
  getStatusIcon,
  getStatusColor
}: ColumnDiffPanelProps) {
  const changes = table.columns?.filter(c => c.status !== 'unchanged') ?? []

  return (
    <div className="border rounded p-4">
      <h3 className="font-semibold mb-3">Column Changes</h3>
      <div className="space-y-2">
        {changes.map(column => (
          <ColumnDiffItem
            key={column.name}
            column={column}
            getStatusIcon={getStatusIcon}
            getStatusColor={getStatusColor}
          />
        ))}
      </div>
    </div>
  )
}

interface ColumnDiffItemProps {
  column: ColumnDiff
  getStatusIcon: (status: DiffStatus) => string
  getStatusColor: (status: DiffStatus) => string
}

function ColumnDiffItem({
  column,
  getStatusIcon,
  getStatusColor
}: ColumnDiffItemProps) {
  return (
    <div className="flex items-start gap-2 p-2 bg-gray-50 rounded">
      <span className={getStatusColor(column.status)}>
        {getStatusIcon(column.status)}
      </span>
      <div className="flex-1">
        <div className="font-medium">{column.name}</div>
        {column.status === 'modified' && (
          <div className="text-sm text-gray-600 space-y-1">
            {column.oldType !== column.newType && (
              <div>Type: {column.oldType} → {column.newType}</div>
            )}
            {column.oldNullable !== column.newNullable && (
              <div>Nullable: {column.oldNullable ? 'YES' : 'NO'} → {column.newNullable ? 'YES' : 'NO'}</div>
            )}
            {column.oldDefault !== column.newDefault && (
              <div>Default: {column.oldDefault ?? 'NULL'} → {column.newDefault ?? 'NULL'}</div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
```

### Migration Script Generator

```typescript
import React, { useState } from 'react'
import {
  type MigrationScript,
  type MigrationPreview,
  type MigrationWarning
} from '@/types/schema-diff'
import { GenerateMigrationSQL } from '@/wailsjs/go/main/App'

interface MigrationGeneratorProps {
  sourceId: string
  targetId: string
}

export function MigrationGenerator({ sourceId, targetId }: MigrationGeneratorProps) {
  const [script, setScript] = useState<MigrationScript | null>(null)
  const [warnings, setWarnings] = useState<MigrationWarning[]>([])
  const [includeDestructive, setIncludeDestructive] = useState(false)

  async function generateScript() {
    const result = await GenerateMigrationSQL(
      sourceId,
      targetId,
      includeDestructive
    )
    setScript(result)

    // Parse warnings from SQL comments
    const detectedWarnings = parseWarnings(result.sql)
    setWarnings(detectedWarnings)
  }

  function parseWarnings(sql: string): MigrationWarning[] {
    const warnings: MigrationWarning[] = []
    const lines = sql.split('\n')

    lines.forEach(line => {
      if (line.includes('DROP COLUMN')) {
        warnings.push({
          severity: 'error',
          objectType: 'column',
          objectName: extractObjectName(line),
          message: 'Column will be dropped - data loss will occur',
          isDestructive: true
        })
      } else if (line.includes('DROP TABLE')) {
        warnings.push({
          severity: 'error',
          objectType: 'table',
          objectName: extractObjectName(line),
          message: 'Table will be dropped - data loss will occur',
          isDestructive: true
        })
      } else if (line.includes('ALTER COLUMN') && line.includes('TYPE')) {
        warnings.push({
          severity: 'warning',
          objectType: 'column',
          objectName: extractObjectName(line),
          message: 'Column type change may require data conversion',
          isDestructive: false
        })
      }
    })

    return warnings
  }

  function extractObjectName(sql: string): string {
    // Simple extraction - improve based on SQL parsing needs
    const match = sql.match(/(?:TABLE|COLUMN)\s+(\w+\.?\w+)/)
    return match?.[1] ?? 'unknown'
  }

  function getSeverityColor(severity: MigrationWarning['severity']): string {
    switch (severity) {
      case 'error': return 'bg-red-100 text-red-800'
      case 'warning': return 'bg-yellow-100 text-yellow-800'
      case 'info': return 'bg-blue-100 text-blue-800'
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={includeDestructive}
            onChange={e => setIncludeDestructive(e.target.checked)}
          />
          Include destructive operations
        </label>
        <button
          onClick={generateScript}
          className="px-4 py-2 bg-blue-500 text-white rounded"
        >
          Generate Migration
        </button>
      </div>

      {warnings.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">Warnings</h3>
          {warnings.map((warning, i) => (
            <div
              key={i}
              className={`p-3 rounded ${getSeverityColor(warning.severity)}`}
            >
              <div className="flex items-start justify-between">
                <div>
                  <div className="font-medium">{warning.objectName}</div>
                  <div className="text-sm">{warning.message}</div>
                </div>
                {warning.isDestructive && (
                  <span className="text-xs font-semibold">DESTRUCTIVE</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {script && (
        <div className="border rounded">
          <div className="bg-gray-100 p-2 border-b">
            <div className="text-sm text-gray-600">
              Generated: {new Date(script.generated_at).toLocaleString()}
            </div>
          </div>
          <pre className="p-4 overflow-x-auto">
            <code>{script.sql}</code>
          </pre>
        </div>
      )}
    </div>
  )
}
```

## Data Catalog Examples

### Catalog Browser Component

```typescript
import React, { useState, useEffect } from 'react'
import {
  type TableCatalogEntry,
  type SearchFilters,
  type SearchResults,
  type CatalogStats
} from '@/types/catalog'
import {
  SearchCatalog,
  GetCatalogStats,
  ListTableCatalogEntries
} from '@/wailsjs/go/main/App'

export function CatalogBrowser({ connectionId }: { connectionId: string }) {
  const [tables, setTables] = useState<TableCatalogEntry[]>([])
  const [searchResults, setSearchResults] = useState<SearchResults | null>(null)
  const [stats, setStats] = useState<CatalogStats | null>(null)
  const [filters, setFilters] = useState<SearchFilters>({ connectionId })

  useEffect(() => {
    loadData()
  }, [connectionId])

  async function loadData() {
    const [tableData, statsData] = await Promise.all([
      ListTableCatalogEntries(connectionId),
      GetCatalogStats(connectionId)
    ])
    setTables(tableData)
    setStats(statsData)
  }

  async function handleSearch(query: string) {
    const results = await SearchCatalog(query, filters)
    setSearchResults(results)
  }

  return (
    <div className="space-y-4">
      {stats && <CatalogStatsPanel stats={stats} />}

      <SearchBar
        filters={filters}
        onFiltersChange={setFilters}
        onSearch={handleSearch}
      />

      {searchResults ? (
        <SearchResultsList results={searchResults} />
      ) : (
        <TableList tables={tables} />
      )}
    </div>
  )
}

function CatalogStatsPanel({ stats }: { stats: CatalogStats }) {
  return (
    <div className="grid grid-cols-4 gap-4">
      <StatCard
        label="Total Tables"
        value={stats.totalTables}
        subvalue={`${stats.coveragePercentage.toFixed(1)}% cataloged`}
      />
      <StatCard
        label="Total Columns"
        value={stats.totalColumns}
        subvalue={`${stats.piiColumns} with PII`}
      />
      <StatCard
        label="Tagged Tables"
        value={stats.taggedTables}
        subvalue={`${((stats.taggedTables / stats.totalTables) * 100).toFixed(1)}%`}
      />
      <StatCard
        label="Stewarded"
        value={stats.stewardedTables}
        subvalue={`${((stats.stewardedTables / stats.totalTables) * 100).toFixed(1)}%`}
      />
    </div>
  )
}
```

### PII Detection Component

```typescript
import React, { useState } from 'react'
import {
  type PIIType,
  type PIIConfidence,
  type ColumnCatalogEntry,
  type PIIDetectionResult
} from '@/types/catalog'
import { MarkColumnAsPII, ListColumnCatalogEntries } from '@/wailsjs/go/main/App'

const PII_TYPES: { value: PIIType; label: string }[] = [
  { value: 'email', label: 'Email Address' },
  { value: 'phone', label: 'Phone Number' },
  { value: 'ssn', label: 'Social Security Number' },
  { value: 'credit_card', label: 'Credit Card' },
  { value: 'ip_address', label: 'IP Address' },
  { value: 'postal_address', label: 'Postal Address' },
  { value: 'name', label: 'Personal Name' },
  { value: 'date_of_birth', label: 'Date of Birth' },
  { value: 'financial_account', label: 'Financial Account' },
  { value: 'medical_record', label: 'Medical Record' },
  { value: 'biometric', label: 'Biometric Data' },
  { value: 'government_id', label: 'Government ID' },
  { value: 'custom', label: 'Custom PII Type' }
]

interface PIIManagerProps {
  tableCatalogId: string
}

export function PIIManager({ tableCatalogId }: PIIManagerProps) {
  const [columns, setColumns] = useState<ColumnCatalogEntry[]>([])

  useEffect(() => {
    loadColumns()
  }, [tableCatalogId])

  async function loadColumns() {
    const data = await ListColumnCatalogEntries(tableCatalogId)
    setColumns(data)
  }

  async function markAsPII(
    columnName: string,
    piiType: PIIType,
    confidence: number
  ) {
    await MarkColumnAsPII(tableCatalogId, columnName, piiType, confidence)
    await loadColumns()
  }

  function getConfidenceLevel(score?: number): PIIConfidence {
    if (!score) return 'low'
    if (score >= 0.9) return 'high'
    if (score >= 0.7) return 'medium'
    return 'low'
  }

  function getConfidenceColor(level: PIIConfidence): string {
    switch (level) {
      case 'confirmed': return 'bg-purple-100 text-purple-800'
      case 'high': return 'bg-red-100 text-red-800'
      case 'medium': return 'bg-yellow-100 text-yellow-800'
      case 'low': return 'bg-blue-100 text-blue-800'
    }
  }

  const piiColumns = columns.filter(c => c.piiType)
  const nonPiiColumns = columns.filter(c => !c.piiType)

  return (
    <div className="space-y-6">
      {piiColumns.length > 0 && (
        <div>
          <h3 className="font-semibold mb-3">PII Columns ({piiColumns.length})</h3>
          <div className="space-y-2">
            {piiColumns.map(column => {
              const level = getConfidenceLevel(column.piiConfidence)
              return (
                <div key={column.id} className="border rounded p-3">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="font-medium">{column.columnName}</div>
                      {column.description && (
                        <div className="text-sm text-gray-600">
                          {column.description}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{column.piiType}</span>
                      <span className={`text-xs px-2 py-1 rounded ${getConfidenceColor(level)}`}>
                        {level} ({(column.piiConfidence! * 100).toFixed(0)}%)
                      </span>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {nonPiiColumns.length > 0 && (
        <div>
          <h3 className="font-semibold mb-3">Non-PII Columns ({nonPiiColumns.length})</h3>
          <div className="space-y-2">
            {nonPiiColumns.map(column => (
              <PIIClassificationForm
                key={column.id}
                column={column}
                onClassify={markAsPII}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

interface PIIClassificationFormProps {
  column: ColumnCatalogEntry
  onClassify: (columnName: string, piiType: PIIType, confidence: number) => Promise<void>
}

function PIIClassificationForm({ column, onClassify }: PIIClassificationFormProps) {
  const [selectedType, setSelectedType] = useState<PIIType>('email')
  const [confidence, setConfidence] = useState(0.5)
  const [expanded, setExpanded] = useState(false)

  async function handleSubmit() {
    await onClassify(column.columnName, selectedType, confidence)
    setExpanded(false)
  }

  return (
    <div className="border rounded p-3">
      <div className="flex items-center justify-between">
        <div className="font-medium">{column.columnName}</div>
        {!expanded && (
          <button
            onClick={() => setExpanded(true)}
            className="text-sm text-blue-600 hover:underline"
          >
            Mark as PII
          </button>
        )}
      </div>

      {expanded && (
        <div className="mt-3 space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1">PII Type</label>
            <select
              value={selectedType}
              onChange={e => setSelectedType(e.target.value as PIIType)}
              className="w-full border rounded px-3 py-2"
            >
              {PII_TYPES.map(type => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              Confidence: {(confidence * 100).toFixed(0)}%
            </label>
            <input
              type="range"
              min="0"
              max="1"
              step="0.01"
              value={confidence}
              onChange={e => setConfidence(parseFloat(e.target.value))}
              className="w-full"
            />
          </div>

          <div className="flex gap-2">
            <button
              onClick={handleSubmit}
              className="px-4 py-2 bg-blue-500 text-white rounded"
            >
              Confirm
            </button>
            <button
              onClick={() => setExpanded(false)}
              className="px-4 py-2 border rounded"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
```

## Advanced Patterns

### Type Guards

```typescript
import type { SearchResult } from '@/types/catalog'

function isTableResult(result: SearchResult): result is SearchResult & { type: 'table' } {
  return result.type === 'table'
}

function isColumnResult(result: SearchResult): result is SearchResult & { type: 'column' } {
  return result.type === 'column'
}

// Usage
results.forEach(result => {
  if (isTableResult(result)) {
    // TypeScript knows result is a table
    console.log(result.tableName)
  } else if (isColumnResult(result)) {
    // TypeScript knows result has columnName
    console.log(`${result.tableName}.${result.columnName}`)
  }
})
```

### Utility Functions

```typescript
import type { DiffStatus, ColumnDiff } from '@/types/schema-diff'

export function filterByStatus(
  columns: ColumnDiff[],
  status: DiffStatus
): ColumnDiff[] {
  return columns.filter(c => c.status === status)
}

export function hasDestructiveChanges(columns: ColumnDiff[]): boolean {
  return columns.some(c => c.status === 'removed')
}

export function summarizeColumnChanges(column: ColumnDiff): string {
  const changes: string[] = []

  if (column.oldType !== column.newType) {
    changes.push(`type: ${column.oldType} → ${column.newType}`)
  }
  if (column.oldNullable !== column.newNullable) {
    changes.push(`nullable: ${column.oldNullable} → ${column.newNullable}`)
  }
  if (column.oldDefault !== column.newDefault) {
    changes.push(`default: ${column.oldDefault} → ${column.newDefault}`)
  }

  return changes.join(', ')
}
```

### Custom Hooks

```typescript
import { useState, useEffect } from 'react'
import type { CatalogStats } from '@/types/catalog'
import { GetCatalogStats } from '@/wailsjs/go/main/App'

export function useCatalogStats(connectionId: string) {
  const [stats, setStats] = useState<CatalogStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    let mounted = true

    async function load() {
      try {
        setLoading(true)
        const data = await GetCatalogStats(connectionId)
        if (mounted) {
          setStats(data)
          setError(null)
        }
      } catch (err) {
        if (mounted) {
          setError(err as Error)
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }

    load()

    return () => {
      mounted = false
    }
  }, [connectionId])

  return { stats, loading, error }
}

// Usage in component
function CatalogDashboard({ connectionId }: { connectionId: string }) {
  const { stats, loading, error } = useCatalogStats(connectionId)

  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!stats) return null

  return <CatalogStatsPanel stats={stats} />
}
```
