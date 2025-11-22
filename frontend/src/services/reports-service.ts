import type {
  ReportRunOverrides,
  ReportRunResult,
  ReportSummary,
} from '@/types/reports'
import type { ReportRecord } from '@/types/storage'

import {
  DeleteReport as deleteReportRPC,
  GetReport as getReportRPC,
  ListReports as listReportsRPC,
  RunReport as runReportRPC,
  SaveReport as saveReportRPC,
} from '../../wailsjs/go/main/App'

const DEFAULT_SYNC_OPTIONS = { enabled: false, cadence: '@every 1h', target: 'local' } as const

const ensureDate = (value?: string | Date): Date | undefined => {
  if (!value) return undefined
  return value instanceof Date ? value : new Date(value)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- Wails RPC returns untyped backend payloads
const fromBackendReport = (data: any): ReportRecord => ({
  id: data?.id ?? crypto.randomUUID(),
  name: data?.name ?? 'Untitled Report',
  description: data?.description ?? '',
  folder: data?.folder ?? '',
  tags: data?.tags ?? [],
  definition: data?.definition ?? { layout: [], components: [] },
  filter: data?.filter ?? { fields: [] },
  sync_options: data?.syncOptions ?? { ...DEFAULT_SYNC_OPTIONS },
  last_run_at: ensureDate(data?.lastRunAt),
  last_run_status: data?.lastRunStatus ?? 'idle',
  metadata: data?.metadata ?? {},
  created_at: ensureDate(data?.createdAt) ?? new Date(),
  updated_at: ensureDate(data?.updatedAt) ?? new Date(),
  synced: true,
  sync_version: data?.syncVersion ?? 0,
})

const toBackendPayload = (record: ReportRecord) => ({
  id: record.id,
  name: record.name,
  description: record.description,
  folder: record.folder,
  tags: record.tags,
  definition: record.definition,
  filter: record.filter,
  syncOptions: record.sync_options ?? DEFAULT_SYNC_OPTIONS,
  lastRunAt: record.last_run_at?.toISOString(),
  lastRunStatus: record.last_run_status,
  metadata: record.metadata ?? {},
  createdAt: record.created_at?.toISOString(),
  updatedAt: record.updated_at?.toISOString(),
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- Wails RPC returns untyped backend payloads
const fromBackendSummary = (summary: any): ReportSummary => ({
  id: summary?.id,
  name: summary?.name,
  description: summary?.description ?? '',
  folder: summary?.folder ?? '',
  tags: summary?.tags ?? [],
  updatedAt: ensureDate(summary?.updatedAt) ?? new Date(),
  lastRunAt: ensureDate(summary?.lastRunAt),
  lastRunStatus: summary?.lastRunStatus ?? 'idle',
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- Wails RPC returns untyped backend payloads
const fromBackendRun = (payload: any): ReportRunResult => ({
  reportId: payload?.reportId,
  startedAt: ensureDate(payload?.startedAt) ?? new Date(),
  completedAt: ensureDate(payload?.completedAt) ?? new Date(),
  results: payload?.results ?? [],
})

export const reportService = {
  async listSummaries(): Promise<ReportSummary[]> {
    const summaries = await listReportsRPC()
    return (summaries ?? []).map(fromBackendSummary)
  },

  async getReport(id: string): Promise<ReportRecord> {
    const data = await getReportRPC(id)
    return fromBackendReport(data)
  },

  async saveReport(record: ReportRecord): Promise<ReportRecord> {
    const payload = toBackendPayload(record)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Wails RPC requires untyped payload
    const saved = await saveReportRPC(payload as any)
    return fromBackendReport(saved)
  },

  async deleteReport(id: string): Promise<void> {
    await deleteReportRPC(id)
  },

  async runReport(reportId: string, overrides: ReportRunOverrides = {}): Promise<ReportRunResult> {
    const response = await runReportRPC({
      reportId,
      componentIds: overrides.componentIds ?? [],
      filterValues: overrides.filters ?? {},
      force: true,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Wails RPC requires untyped payload
    } as any)
    return fromBackendRun(response)
  },
}
