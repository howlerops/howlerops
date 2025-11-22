/**
 * Templates & Schedules API Client
 * Handles all backend communication for query templates and scheduling
 */

import type {
  CreateScheduleInput,
  CreateTemplateInput,
  QueryResult,
  QuerySchedule,
  QueryTemplate,
  ScheduleExecution,
  TemplateFilters,
  UpdateTemplateInput} from '@/types/templates'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

class TemplatesApiError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public response?: Record<string, unknown>
  ) {
    super(message)
    this.name = 'TemplatesApiError'
  }
}

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const token = localStorage.getItem('auth_token')

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({}))
    throw new TemplatesApiError(
      error.message || `API Error: ${response.statusText}`,
      response.status,
      error
    )
  }

  return response.json()
}

// ============================================================================
// Template Operations
// ============================================================================

export async function listTemplates(
  filters?: TemplateFilters
): Promise<QueryTemplate[]> {
  const params = new URLSearchParams()

  if (filters?.category) params.append('category', filters.category)
  if (filters?.tags?.length) params.append('tags', filters.tags.join(','))
  if (filters?.search) params.append('search', filters.search)
  if (filters?.created_by) params.append('created_by', filters.created_by)
  if (filters?.is_public !== undefined) {
    params.append('is_public', String(filters.is_public))
  }

  const query = params.toString()
  const endpoint = `/api/v1/templates${query ? `?${query}` : ''}`

  return fetchApi<QueryTemplate[]>(endpoint)
}

export async function getTemplate(id: string): Promise<QueryTemplate> {
  return fetchApi<QueryTemplate>(`/api/v1/templates/${id}`)
}

export async function createTemplate(
  input: CreateTemplateInput
): Promise<QueryTemplate> {
  return fetchApi<QueryTemplate>('/api/v1/templates', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateTemplate(
  id: string,
  input: UpdateTemplateInput
): Promise<QueryTemplate> {
  return fetchApi<QueryTemplate>(`/api/v1/templates/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function deleteTemplate(id: string): Promise<void> {
  await fetchApi(`/api/v1/templates/${id}`, {
    method: 'DELETE',
  })
}

export async function executeTemplate(
  id: string,
  params: Record<string, unknown>
): Promise<QueryResult> {
  return fetchApi<QueryResult>(`/api/v1/templates/${id}/execute`, {
    method: 'POST',
    body: JSON.stringify({ parameters: params }),
  })
}

export async function duplicateTemplate(id: string): Promise<QueryTemplate> {
  return fetchApi<QueryTemplate>(`/api/v1/templates/${id}/duplicate`, {
    method: 'POST',
  })
}

export async function getTemplateStats(id: string) {
  return fetchApi(`/api/v1/templates/${id}/stats`)
}

// ============================================================================
// Schedule Operations
// ============================================================================

export async function listSchedules(): Promise<QuerySchedule[]> {
  return fetchApi<QuerySchedule[]>('/api/v1/schedules')
}

export async function getSchedule(id: string): Promise<QuerySchedule> {
  return fetchApi<QuerySchedule>(`/api/v1/schedules/${id}`)
}

export async function createSchedule(
  input: CreateScheduleInput
): Promise<QuerySchedule> {
  return fetchApi<QuerySchedule>('/api/v1/schedules', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateSchedule(
  id: string,
  input: Partial<CreateScheduleInput>
): Promise<QuerySchedule> {
  return fetchApi<QuerySchedule>(`/api/v1/schedules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function deleteSchedule(id: string): Promise<void> {
  await fetchApi(`/api/v1/schedules/${id}`, {
    method: 'DELETE',
  })
}

export async function pauseSchedule(id: string): Promise<void> {
  await fetchApi(`/api/v1/schedules/${id}/pause`, {
    method: 'POST',
  })
}

export async function resumeSchedule(id: string): Promise<void> {
  await fetchApi(`/api/v1/schedules/${id}/resume`, {
    method: 'POST',
  })
}

export async function runScheduleNow(id: string): Promise<ScheduleExecution> {
  return fetchApi<ScheduleExecution>(`/api/v1/schedules/${id}/run`, {
    method: 'POST',
  })
}

// ============================================================================
// Execution History
// ============================================================================

export async function getScheduleExecutions(
  id: string,
  limit = 50
): Promise<ScheduleExecution[]> {
  return fetchApi<ScheduleExecution[]>(
    `/api/v1/schedules/${id}/executions?limit=${limit}`
  )
}

export async function getExecution(
  scheduleId: string,
  executionId: string
): Promise<ScheduleExecution> {
  return fetchApi<ScheduleExecution>(
    `/api/v1/schedules/${scheduleId}/executions/${executionId}`
  )
}

export async function getExecutionResult(
  scheduleId: string,
  executionId: string
): Promise<QueryResult> {
  return fetchApi<QueryResult>(
    `/api/v1/schedules/${scheduleId}/executions/${executionId}/result`
  )
}

// ============================================================================
// Utility Functions
// ============================================================================

export function parseTemplateParameters(sql: string): string[] {
  // Extract {{parameter}} style placeholders
  const regex = /\{\{(\w+)\}\}/g
  const params = new Set<string>()
  let match

  while ((match = regex.exec(sql)) !== null) {
    params.add(match[1])
  }

  return Array.from(params)
}

export function interpolateTemplate(
  sql: string,
  params: Record<string, string | number | boolean | null>
): string {
  return sql.replace(/\{\{(\w+)\}\}/g, (_, paramName) => {
    const value = params[paramName]
    if (value === undefined) {
      throw new Error(`Missing required parameter: ${paramName}`)
    }
    // Basic SQL escaping - in production use proper parameterized queries
    if (typeof value === 'string') {
      return `'${value.replace(/'/g, "''")}'`
    }
    return String(value)
  })
}

export { TemplatesApiError }
