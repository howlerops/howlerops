/**
 * Query Templates & Scheduling Types
 * Type definitions for template management and scheduled query execution
 */

export interface QueryTemplate {
  id: string
  name: string
  description?: string
  sql_template: string
  parameters: TemplateParameter[]
  tags: string[]
  category: 'reporting' | 'analytics' | 'maintenance' | 'custom'
  organization_id?: string
  created_by: string
  created_at: string
  updated_at: string
  is_public: boolean
  usage_count: number
}

export type TemplateParameterValue = string | number | boolean | null

export interface TemplateParameter {
  name: string
  type: 'string' | 'number' | 'boolean' | 'date'
  default?: TemplateParameterValue
  required: boolean
  description?: string
  validation?: {
    min?: number
    max?: number
    pattern?: string
    options?: string[] // For enum-like parameters
  }
}

export interface QuerySchedule {
  id: string
  template_id: string
  name: string
  frequency: string // cron expression
  parameters: Record<string, TemplateParameterValue>
  last_run_at?: string
  next_run_at?: string
  status: 'active' | 'paused' | 'failed'
  created_by: string
  organization_id?: string
  notification_email?: string
  result_storage: 'none' | 's3' | 'database'
  created_at: string
  updated_at: string
}

export interface ScheduleExecution {
  id: string
  schedule_id: string
  executed_at: string
  status: 'success' | 'failed' | 'timeout'
  duration_ms: number
  rows_returned?: number
  error_message?: string
  result_preview?: Record<string, unknown>[]
}

export interface QueryResult {
  columns: string[]
  rows: Record<string, unknown>[]
  rowCount: number
  executionTime: number
  totalRows?: number // Total unpaginated rows
}

// API Input Types
export interface CreateTemplateInput {
  name: string
  description?: string
  sql_template: string
  parameters?: TemplateParameter[]
  tags?: string[]
  category: 'reporting' | 'analytics' | 'maintenance' | 'custom'
  is_public?: boolean
}

export interface UpdateTemplateInput {
  name?: string
  description?: string
  sql_template?: string
  parameters?: TemplateParameter[]
  tags?: string[]
  category?: 'reporting' | 'analytics' | 'maintenance' | 'custom'
  is_public?: boolean
}

export interface CreateScheduleInput {
  template_id: string
  name: string
  frequency: string
  parameters: Record<string, TemplateParameterValue>
  notification_email?: string
  result_storage?: 'none' | 's3' | 'database'
}

export interface TemplateFilters {
  category?: string
  tags?: string[]
  search?: string
  created_by?: string
  is_public?: boolean
}

// UI State Types
export interface TemplateExecutionState {
  isExecuting: boolean
  result: QueryResult | null
  error: string | null
}

export interface CronSchedule {
  minute: string
  hour: string
  dayOfMonth: string
  month: string
  dayOfWeek: string
}

export type TemplateSortBy = 'usage' | 'newest' | 'name' | 'updated'

export interface TemplateStats {
  totalTemplates: number
  totalSchedules: number
  activeSchedules: number
  totalExecutions: number
  successRate: number
}
