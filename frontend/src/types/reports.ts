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
