/**
 * Types for Visual Query Builder components
 */

import { FilterOperator } from '@/workers/types'
import { QueryIR, TableRef, SelectItem, Expr, OrderBy } from '@/lib/query-ir'

export type { OrderBy }
import { FieldType } from '@/lib/type-registry'

export interface ConnectionInfo {
  id: string
  name: string
  type: string
  isConnected: boolean
}

export interface SchemaInfo {
  name: string
  tables: TableInfo[]
}

export interface TableInfo {
  name: string
  schema: string
  columns: ColumnInfo[]
  rowCount?: number
  sizeBytes?: number
}

export interface ColumnInfo {
  name: string
  dataType: string
  isNullable: boolean
  isPrimaryKey: boolean
  isForeignKey: boolean
  enumValues?: string[]
  foreignKeyTable?: string
  foreignKeyColumn?: string
}

export interface FilterCondition {
  id: string
  column: string
  operator: FilterOperator
  value: unknown
  not?: boolean
}

export interface FilterGroup {
  id: string
  operator: 'AND' | 'OR'
  conditions: (FilterCondition | FilterGroup)[]
  not?: boolean
}

export interface VisualQueryState {
  connections: string[]
  from: TableRef | null
  joins: Array<{
    id: string
    type: 'inner' | 'left' | 'right' | 'full'
    table: TableRef
    on: Expr
  }>
  select: SelectItem[]
  where?: Expr
  orderBy: OrderBy[]
  limit?: number
  offset?: number
}

export interface SourcePickerProps {
  connections: ConnectionInfo[]
  schemas: Map<string, SchemaInfo[]>
  selectedConnections: string[]
  selectedTable: TableRef | null
  onConnectionsChange: (connectionIds: string[]) => void
  onTableChange: (table: TableRef | null) => void
  onSchemaLoad: (connectionId: string) => Promise<void>
}

export interface ColumnPickerProps {
  table: TableRef | null
  columns: ColumnInfo[]
  selectedColumns: SelectItem[]
  onColumnsChange: (columns: SelectItem[]) => void
}

export interface FilterEditorProps {
  columns: ColumnInfo[]
  where?: Expr
  onWhereChange: (where: Expr | undefined) => void
}

export interface JoinBuilderProps {
  availableTables: TableInfo[]
  existingJoins: Array<{
    id: string
    type: 'inner' | 'left' | 'right' | 'full'
    table: TableRef
    on: Expr
  }>
  onJoinsChange: (joins: Array<{
    id: string
    type: 'inner' | 'left' | 'right' | 'full'
    table: TableRef
    on: Expr
  }>) => void
}

export interface SortLimitProps {
  columns: ColumnInfo[]
  orderBy: OrderBy[]
  limit?: number
  offset?: number
  onOrderByChange: (orderBy: OrderBy[]) => void
  onLimitChange: (limit?: number) => void
  onOffsetChange: (offset?: number) => void
}

export interface SqlPreviewProps {
  queryIR: QueryIR
  dialect: 'postgres' | 'mysql' | 'sqlite' | 'mssql'
  manualSQL?: string
  onSQLChange?: (sql: string) => void
}

export interface VisualQueryBuilderProps {
  connections: ConnectionInfo[]
  schemas: Map<string, SchemaInfo[]>
  onQueryChange: (queryIR: QueryIR) => void
  onSQLChange?: (sql: string) => void
  initialQuery?: QueryIR
}

export interface TypeInputProps {
  fieldType: FieldType
  value: unknown
  onChange: (value: unknown) => void
  column: ColumnInfo
  operator: FilterOperator
  placeholder?: string
  disabled?: boolean
}

export interface OperatorSelectProps {
  fieldType: FieldType
  value: FilterOperator
  onChange: (operator: FilterOperator) => void
  disabled?: boolean
}

export interface ColumnSelectProps {
  columns: ColumnInfo[]
  value: string
  onChange: (column: string) => void
  placeholder?: string
  disabled?: boolean
}
