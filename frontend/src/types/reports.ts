/**
 * Report Builder Type Definitions
 *
 * Shared between state stores, components, and sync payloads.
 */

export type ReportComponentType = 'chart' | 'metric' | 'table' | 'combo' | 'llm'

export type ReportQueryMode = 'sql' | 'builder'

export type ReportSyncTarget = 'local' | 'remote'

export interface ReportLayoutSlot {
  componentId: string
  x: number
  y: number
  w: number
  h: number
}

export interface ReportComponentSize {
  minW?: number
  minH?: number
  maxW?: number
  maxH?: number
}

export interface ReportQueryConfig {
  mode: ReportQueryMode
  connectionId?: string
  sql?: string
  builderState?: Record<string, unknown>
  queryIr?: Record<string, unknown>
  useFederation?: boolean
  limit?: number
  cacheSeconds?: number
  topLevelFilter?: string[]
  parameters?: Record<string, unknown>
}

export type DrillDownActionType = 'detail' | 'related-report' | 'filter' | 'url'

export interface DrillDownConfig {
  enabled: boolean
  type: DrillDownActionType
  target?: string // Report ID or URL
  filterField?: string // Which field to filter by
  detailQuery?: string // SQL for detail view
  parameters?: Record<string, string> // Mapping from click to query params
}

export interface DrillDownContext {
  clickedValue: unknown
  field: string
  filters?: Record<string, unknown>
  additionalData?: Record<string, unknown>
  componentId?: string
}

export interface DrillDownAction {
  type: DrillDownActionType
  componentId: string
  context: DrillDownContext
  timestamp: Date
  config: DrillDownConfig
}

export interface ReportChartSettings {
  variant?: 'line' | 'bar' | 'area' | 'pie' | 'combo'
  xField?: string
  yField?: string
  series?: string[]
  options?: Record<string, string>
  comparison?: {
    baselineComponentId: string
    type: 'delta' | 'trend' | 'goal'
  }
  transform?: {
    kind: string
    config?: Record<string, unknown>
  }
}

export interface ReportLLMSettings {
  provider: string
  model: string
  promptTemplate: string
  contextComponents: string[]
  temperature?: number
  maxTokens?: number
  metadata?: Record<string, string>
}

export interface ReportFilterField {
  key: string
  label: string
  type: 'text' | 'number' | 'date' | 'select' | 'multi-select'
  defaultValue?: unknown
  required?: boolean
  choices?: string[]
}

export interface ReportFilterDefinition {
  fields: ReportFilterField[]
}

export interface ReportComponent {
  id: string
  title: string
  description?: string
  type: ReportComponentType
  size?: ReportComponentSize
  query?: ReportQueryConfig
  chart?: ReportChartSettings
  llm?: ReportLLMSettings
  drillDown?: DrillDownConfig
  options?: Record<string, unknown>
}

export interface ReportDefinition {
  layout: ReportLayoutSlot[]
  components: ReportComponent[]
}

export interface ReportSyncOptions {
  enabled: boolean
  cadence: string
  target: ReportSyncTarget
}

export interface ReportSummaryDTO {
  id: string
  name: string
  description?: string
  folder?: string
  tags: string[]
  starred: boolean
  starredAt?: string
  updatedAt: string
  lastRunAt?: string
  lastRunStatus?: string
}

export interface ReportSummary {
  id: string
  name: string
  description?: string
  folder?: string
  tags: string[]
  starred: boolean
  starredAt?: Date
  updatedAt: Date
  lastRunAt?: Date
  lastRunStatus?: string
}

export interface ReportRunComponentResult {
  componentId: string
  type: ReportComponentType
  columns?: string[]
  rows?: unknown[][]
  rowCount?: number
  durationMs?: number
  content?: string
  metadata?: Record<string, unknown>
  error?: string
}

export interface ReportRunResponseDTO {
  reportId: string
  startedAt: string
  completedAt: string
  results: ReportRunComponentResult[]
}

export interface ReportRunResult {
  reportId: string
  startedAt: Date
  completedAt: Date
  results: ReportRunComponentResult[]
}

export interface ReportRunOverrides {
  componentIds?: string[]
  filters?: Record<string, unknown>
}

// ===== Query Builder Types =====

export type AggregationFunction = 'count' | 'sum' | 'avg' | 'min' | 'max' | 'count_distinct'

export type FilterOperator =
  | '='
  | '!='
  | '>'
  | '<'
  | '>='
  | '<='
  | 'LIKE'
  | 'NOT LIKE'
  | 'IN'
  | 'NOT IN'
  | 'IS NULL'
  | 'IS NOT NULL'
  | 'BETWEEN'

export type JoinType = 'INNER' | 'LEFT' | 'RIGHT' | 'FULL'

export type SortDirection = 'ASC' | 'DESC'

export type FilterCombinator = 'AND' | 'OR'

export interface ColumnSelection {
  table: string
  column: string
  alias?: string
  aggregation?: AggregationFunction
}

export interface JoinDefinition {
  type: JoinType
  table: string
  alias?: string
  on: {
    left: string // format: "table.column"
    right: string // format: "table.column"
  }
}

export interface FilterCondition {
  id: string // unique ID for React keys
  column: string // format: "table.column"
  operator: FilterOperator
  value?: unknown
  valueTo?: unknown // for BETWEEN operator
  combinator?: FilterCombinator // only used for filters after the first
}

export interface OrderByClause {
  column: string // format: "table.column"
  direction: SortDirection
}

export interface QueryBuilderState {
  dataSource: string // connectionId
  table: string
  columns: ColumnSelection[]
  joins: JoinDefinition[]
  filters: FilterCondition[]
  groupBy: string[] // format: ["table.column"]
  orderBy: OrderByClause[]
  limit?: number
  offset?: number
}

// Schema introspection types
export interface DatabaseSchema {
  tables: TableMetadata[]
}

export interface TableMetadata {
  schema: string
  name: string
  type: string
  comment?: string
  rowCount: number
  columns: ColumnMetadata[]
  foreignKeys: ForeignKeyMetadata[]
}

export interface ColumnMetadata {
  name: string
  dataType: string
  nullable: boolean
  defaultValue?: string
  primaryKey: boolean
  unique: boolean
  indexed: boolean
  comment?: string
  ordinalPosition: number
  characterMaxLength?: number
  numericPrecision?: number
  numericScale?: number
}

export interface ForeignKeyMetadata {
  name: string
  columns: string[]
  referencedTable: string
  referencedSchema: string
  referencedColumns: string[]
  onDelete: string
  onUpdate: string
}

// Query validation
export interface QueryValidationError {
  field: string
  message: string
  severity: 'error' | 'warning'
}

export interface QueryValidationResult {
  valid: boolean
  errors: QueryValidationError[]
  warnings: QueryValidationError[]
}

// Query preview
export interface QueryPreview {
  sql: string
  estimatedRows: number
  columns: string[]
  rows: unknown[][]
  totalRows: number
  executionTimeMs: number
}

// ===== Folder, Tag, and Template Types =====

export interface ReportFolder {
  id: string
  name: string
  parentId?: string
  color?: string
  icon?: string
  createdAt: Date
  updatedAt: Date
}

export interface ReportFolderDTO {
  id: string
  name: string
  parentId?: string
  color?: string
  icon?: string
  createdAt: string
  updatedAt: string
}

export interface FolderNode extends ReportFolder {
  children: FolderNode[]
  reportCount: number
  expanded?: boolean
}

export interface Tag {
  name: string
  color?: string
  reportCount: number
}

export interface ReportTemplate {
  id: string
  name: string
  description: string
  category: 'analytics' | 'operations' | 'sales' | 'custom'
  thumbnail?: string
  icon: string
  tags: string[]
  definition: ReportDefinition
  filter?: ReportFilterDefinition
  featured: boolean
  usageCount: number
  createdAt: Date
  updatedAt: Date
}

export interface ReportTemplateDTO {
  id: string
  name: string
  description: string
  category: string
  thumbnail?: string
  icon: string
  tags: string[]
  definition: ReportDefinition
  filter?: ReportFilterDefinition
  featured: boolean
  usageCount: number
  createdAt: string
  updatedAt: string
}
