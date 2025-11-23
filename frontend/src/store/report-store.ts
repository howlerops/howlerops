import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

import { reportService } from '@/services/reports-service'
import type { ReportRunOverrides, ReportRunResult, ReportSummary } from '@/types/reports'
import type { ReportRecord } from '@/types/storage'

interface ReportStoreState {
  summaries: ReportSummary[]
  activeReport?: ReportRecord
  loading: boolean
  error?: string
  lastRun?: ReportRunResult
  filterText: string
  topLevelFilters: Record<string, unknown>

  fetchReports: () => Promise<void>
  selectReport: (id: string) => Promise<void>
  createReport: (name?: string) => Promise<ReportRecord>
  updateActive: (update: Partial<ReportRecord>) => void
  saveActive: () => Promise<void>
  runActive: (overrides?: ReportRunOverrides) => Promise<void>
  deleteReport: (id: string) => Promise<void>
  setFilterText: (value: string) => void
  setTopLevelFilters: (filters: Record<string, unknown>) => void
  clearError: () => void
}

const createBlankReport = (name = 'Untitled Report'): ReportRecord => ({
  id: crypto.randomUUID(),
  name,
  description: '',
  folder: '',
  tags: [],
  definition: { layout: [], components: [] },
  filter: { fields: [] },
  sync_options: { enabled: false, cadence: '@every 1h', target: 'local' },
  last_run_at: undefined,
  last_run_status: 'idle',
  metadata: {},
  created_at: new Date(),
  updated_at: new Date(),
  synced: false,
  sync_version: 0,
})

const initialState: Omit<ReportStoreState, 'fetchReports' | 'selectReport' | 'createReport' | 'updateActive' | 'saveActive' | 'runActive' | 'deleteReport' | 'setFilterText' | 'setTopLevelFilters' | 'clearError'> = {
  summaries: [],
  loading: false,
  error: undefined,
  filterText: '',
  topLevelFilters: {},
}

export const useReportStore = create<ReportStoreState>()(
  devtools((set, get) => ({
    ...initialState,

    fetchReports: async () => {
      set({ loading: true, error: undefined })
      try {
        const summaries = await reportService.listSummaries()
        set({ summaries, loading: false })
      } catch (error) {
        set({
          error: error instanceof Error ? error.message : 'Failed to load reports',
          loading: false,
        })
      }
    },

    selectReport: async (id: string) => {
      set({ loading: true, error: undefined })
      try {
        const report = await reportService.getReport(id)
        set({ activeReport: report, loading: false })
      } catch (error) {
        set({
          error: error instanceof Error ? error.message : 'Failed to load report',
          loading: false,
        })
      }
    },

    createReport: async (name?: string) => {
      set({ loading: true, error: undefined })
      try {
        const draft = createBlankReport(name)
        const saved = await reportService.saveReport(draft)
        set((state) => ({
          summaries: [
            {
              id: saved.id,
              name: saved.name,
              description: saved.description,
              folder: saved.folder,
              tags: saved.tags,
              starred: false,
              starredAt: undefined,
              updatedAt: saved.updated_at,
              lastRunAt: saved.last_run_at,
              lastRunStatus: saved.last_run_status,
            },
            ...state.summaries,
          ],
          activeReport: saved,
          loading: false,
        }))
        return saved
      } catch (error) {
        set({
          error: error instanceof Error ? error.message : 'Failed to create report',
          loading: false,
        })
        throw error
      }
    },

    updateActive: (update: Partial<ReportRecord>) => {
      set((state) => {
        if (!state.activeReport) return state

        // Only update if actually changed (granular comparison)
        const hasChanges = Object.keys(update).some((key) => {
          const oldValue = state.activeReport![key as keyof ReportRecord]
          const newValue = update[key as keyof Partial<ReportRecord>]

          // Deep comparison for objects
          if (typeof oldValue === 'object' && typeof newValue === 'object') {
            return JSON.stringify(oldValue) !== JSON.stringify(newValue)
          }

          return oldValue !== newValue
        })

        if (!hasChanges) return state

        const updated: ReportRecord = {
          ...state.activeReport,
          ...update,
          updated_at: new Date(),
          synced: false,
        }
        return { activeReport: updated }
      })
    },

    saveActive: async () => {
      const current = get().activeReport
      if (!current) return
      set({ loading: true, error: undefined })
      try {
        const saved = await reportService.saveReport(current)
        set((state) => ({
          summaries: state.summaries.map((summary) =>
            summary.id === saved.id
              ? {
                  ...summary,
                  name: saved.name,
                  description: saved.description,
                  tags: saved.tags,
                  folder: saved.folder,
                  updatedAt: saved.updated_at,
                  lastRunAt: saved.last_run_at,
                  lastRunStatus: saved.last_run_status,
                }
              : summary
          ),
          activeReport: saved,
          loading: false,
        }))
      } catch (error) {
        set({
          error: error instanceof Error ? error.message : 'Failed to save report',
          loading: false,
        })
        throw error
      }
    },

  runActive: async (overrides?: ReportRunOverrides) => {
      const current = get().activeReport
      if (!current) return
      set({ loading: true, error: undefined })
      try {
        const result = await reportService.runReport(current.id, {
          ...(overrides ?? {}),
          filters: get().topLevelFilters,
        })
        set((state) => ({
          lastRun: result,
          loading: false,
          summaries: state.summaries.map((summary) =>
            summary.id === current.id
              ? {
                  ...summary,
                  lastRunAt: result.completedAt,
                  lastRunStatus: result.results.every((r) => !r.error) ? 'ok' : 'error',
                }
              : summary
          ),
        }))
      } catch (error) {
        set({
          error: error instanceof Error ? error.message : 'Failed to run report',
          loading: false,
        })
        throw error
      }
    },

    deleteReport: async (id: string) => {
      set({ loading: true, error: undefined })
      try {
        await reportService.deleteReport(id)
        set((state) => ({
          summaries: state.summaries.filter((summary) => summary.id !== id),
          activeReport: state.activeReport?.id === id ? undefined : state.activeReport,
          loading: false,
        }))
      } catch (error) {
        set({
          error: error instanceof Error ? error.message : 'Failed to delete report',
          loading: false,
        })
        throw error
      }
    },

    setFilterText: (value: string) => set({ filterText: value }),

    setTopLevelFilters: (filters: Record<string, unknown>) =>
      set((state) => ({ topLevelFilters: { ...state.topLevelFilters, ...filters } })),

    clearError: () => set({ error: undefined }),
  }))
)
