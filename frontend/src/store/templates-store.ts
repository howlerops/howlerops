/**
 * Templates Store
 * Zustand store for managing query templates and schedules
 */

import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

import * as api from '@/lib/api/templates'
import type {
  CreateScheduleInput,
  CreateTemplateInput,
  QueryResult,
  QuerySchedule,
  QueryTemplate,
  ScheduleExecution,
  TemplateFilters,
  TemplateSortBy,
  UpdateTemplateInput} from '@/types/templates'

interface TemplatesStore {
  // State
  templates: QueryTemplate[]
  schedules: QuerySchedule[]
  executions: Map<string, ScheduleExecution[]>
  loading: boolean
  error: string | null
  filters: TemplateFilters
  sortBy: TemplateSortBy

  // Templates Actions
  fetchTemplates: () => Promise<void>
  getTemplate: (id: string) => Promise<QueryTemplate>
  createTemplate: (input: CreateTemplateInput) => Promise<QueryTemplate>
  updateTemplate: (id: string, input: UpdateTemplateInput) => Promise<void>
  deleteTemplate: (id: string) => Promise<void>
  duplicateTemplate: (id: string) => Promise<QueryTemplate>
  executeTemplate: (id: string, params: Record<string, unknown>) => Promise<QueryResult>

  // Schedules Actions
  fetchSchedules: () => Promise<void>
  getSchedule: (id: string) => Promise<QuerySchedule>
  createSchedule: (input: CreateScheduleInput) => Promise<QuerySchedule>
  updateSchedule: (id: string, input: Partial<CreateScheduleInput>) => Promise<void>
  deleteSchedule: (id: string) => Promise<void>
  pauseSchedule: (id: string) => Promise<void>
  resumeSchedule: (id: string) => Promise<void>
  runScheduleNow: (id: string) => Promise<void>

  // Execution History
  fetchExecutions: (scheduleId: string) => Promise<void>
  getExecution: (scheduleId: string, executionId: string) => Promise<ScheduleExecution>

  // Filtering & Search
  setFilters: (filters: Partial<TemplateFilters>) => void
  setSortBy: (sortBy: TemplateSortBy) => void
  searchTemplates: (query: string) => QueryTemplate[]
  filterByCategory: (category: string) => QueryTemplate[]
  filterByTags: (tags: string[]) => QueryTemplate[]
  getFilteredTemplates: () => QueryTemplate[]

  // Utility
  clearError: () => void
  reset: () => void
}

const initialState = {
  templates: [],
  schedules: [],
  executions: new Map(),
  loading: false,
  error: null,
  filters: {},
  sortBy: 'usage' as TemplateSortBy,
}

export const useTemplatesStore = create<TemplatesStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      // ========================================================================
      // Templates Actions
      // ========================================================================

      fetchTemplates: async () => {
        set({ loading: true, error: null })
        try {
          const templates = await api.listTemplates(get().filters)
          set({ templates, loading: false })
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to fetch templates',
            loading: false,
          })
        }
      },

      getTemplate: async (id: string) => {
        set({ loading: true, error: null })
        try {
          const template = await api.getTemplate(id)
          // Update in store if exists
          set((state) => ({
            templates: state.templates.map((t) => (t.id === id ? template : t)),
            loading: false,
          }))
          return template
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to get template',
            loading: false,
          })
          throw error
        }
      },

      createTemplate: async (input: CreateTemplateInput) => {
        set({ loading: true, error: null })
        try {
          const template = await api.createTemplate(input)
          set((state) => ({
            templates: [template, ...state.templates],
            loading: false,
          }))
          return template
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to create template',
            loading: false,
          })
          throw error
        }
      },

      updateTemplate: async (id: string, input: UpdateTemplateInput) => {
        set({ loading: true, error: null })
        try {
          const updated = await api.updateTemplate(id, input)
          set((state) => ({
            templates: state.templates.map((t) => (t.id === id ? updated : t)),
            loading: false,
          }))
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to update template',
            loading: false,
          })
          throw error
        }
      },

      deleteTemplate: async (id: string) => {
        set({ loading: true, error: null })
        try {
          await api.deleteTemplate(id)
          set((state) => ({
            templates: state.templates.filter((t) => t.id !== id),
            loading: false,
          }))
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to delete template',
            loading: false,
          })
          throw error
        }
      },

      duplicateTemplate: async (id: string) => {
        set({ loading: true, error: null })
        try {
          const duplicated = await api.duplicateTemplate(id)
          set((state) => ({
            templates: [duplicated, ...state.templates],
            loading: false,
          }))
          return duplicated
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to duplicate template',
            loading: false,
          })
          throw error
        }
      },

      executeTemplate: async (id: string, params: Record<string, unknown>) => {
        set({ loading: true, error: null })
        try {
          const result = await api.executeTemplate(id, params)
          // Increment usage count optimistically
          set((state) => ({
            templates: state.templates.map((t) =>
              t.id === id ? { ...t, usage_count: t.usage_count + 1 } : t
            ),
            loading: false,
          }))
          return result
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to execute template',
            loading: false,
          })
          throw error
        }
      },

      // ========================================================================
      // Schedules Actions
      // ========================================================================

      fetchSchedules: async () => {
        set({ loading: true, error: null })
        try {
          const schedules = await api.listSchedules()
          set({ schedules, loading: false })
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to fetch schedules',
            loading: false,
          })
        }
      },

      getSchedule: async (id: string) => {
        set({ loading: true, error: null })
        try {
          const schedule = await api.getSchedule(id)
          set((state) => ({
            schedules: state.schedules.map((s) => (s.id === id ? schedule : s)),
            loading: false,
          }))
          return schedule
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to get schedule',
            loading: false,
          })
          throw error
        }
      },

      createSchedule: async (input: CreateScheduleInput) => {
        set({ loading: true, error: null })
        try {
          const schedule = await api.createSchedule(input)
          set((state) => ({
            schedules: [schedule, ...state.schedules],
            loading: false,
          }))
          return schedule
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to create schedule',
            loading: false,
          })
          throw error
        }
      },

      updateSchedule: async (id: string, input: Partial<CreateScheduleInput>) => {
        set({ loading: true, error: null })
        try {
          const updated = await api.updateSchedule(id, input)
          set((state) => ({
            schedules: state.schedules.map((s) => (s.id === id ? updated : s)),
            loading: false,
          }))
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to update schedule',
            loading: false,
          })
          throw error
        }
      },

      deleteSchedule: async (id: string) => {
        set({ loading: true, error: null })
        try {
          await api.deleteSchedule(id)
          set((state) => ({
            schedules: state.schedules.filter((s) => s.id !== id),
            loading: false,
          }))
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to delete schedule',
            loading: false,
          })
          throw error
        }
      },

      pauseSchedule: async (id: string) => {
        try {
          await api.pauseSchedule(id)
          set((state) => ({
            schedules: state.schedules.map((s) =>
              s.id === id ? { ...s, status: 'paused' as const } : s
            ),
          }))
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to pause schedule',
          })
          throw error
        }
      },

      resumeSchedule: async (id: string) => {
        try {
          await api.resumeSchedule(id)
          set((state) => ({
            schedules: state.schedules.map((s) =>
              s.id === id ? { ...s, status: 'active' as const } : s
            ),
          }))
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to resume schedule',
          })
          throw error
        }
      },

      runScheduleNow: async (id: string) => {
        try {
          const execution = await api.runScheduleNow(id)
          // Add to executions map
          set((state) => {
            const newExecutions = new Map(state.executions)
            const scheduleExecs = newExecutions.get(id) || []
            newExecutions.set(id, [execution, ...scheduleExecs])
            return { executions: newExecutions }
          })
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to run schedule',
          })
          throw error
        }
      },

      // ========================================================================
      // Execution History
      // ========================================================================

      fetchExecutions: async (scheduleId: string) => {
        set({ loading: true, error: null })
        try {
          const executions = await api.getScheduleExecutions(scheduleId)
          set((state) => {
            const newExecutions = new Map(state.executions)
            newExecutions.set(scheduleId, executions)
            return { executions: newExecutions, loading: false }
          })
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to fetch executions',
            loading: false,
          })
        }
      },

      getExecution: async (scheduleId: string, executionId: string) => {
        set({ loading: true, error: null })
        try {
          const execution = await api.getExecution(scheduleId, executionId)
          set({ loading: false })
          return execution
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Failed to get execution',
            loading: false,
          })
          throw error
        }
      },

      // ========================================================================
      // Filtering & Search
      // ========================================================================

      setFilters: (filters: Partial<TemplateFilters>) => {
        set((state) => ({
          filters: { ...state.filters, ...filters },
        }))
      },

      setSortBy: (sortBy: TemplateSortBy) => {
        set({ sortBy })
      },

      searchTemplates: (query: string) => {
        const { templates } = get()
        const lowerQuery = query.toLowerCase()

        return templates.filter(
          (t) =>
            t.name.toLowerCase().includes(lowerQuery) ||
            t.description?.toLowerCase().includes(lowerQuery) ||
            t.sql_template.toLowerCase().includes(lowerQuery) ||
            t.tags.some((tag) => tag.toLowerCase().includes(lowerQuery))
        )
      },

      filterByCategory: (category: string) => {
        const { templates } = get()
        return templates.filter((t) => t.category === category)
      },

      filterByTags: (tags: string[]) => {
        const { templates } = get()
        return templates.filter((t) => tags.some((tag) => t.tags.includes(tag)))
      },

      getFilteredTemplates: () => {
        const { templates, filters, sortBy } = get()
        let filtered = [...templates]

        // Apply category filter
        if (filters.category && filters.category !== 'all') {
          filtered = filtered.filter((t) => t.category === filters.category)
        }

        // Apply tags filter
        if (filters.tags?.length) {
          filtered = filtered.filter((t) =>
            filters.tags!.some((tag) => t.tags.includes(tag))
          )
        }

        // Apply search filter
        if (filters.search) {
          const lowerQuery = filters.search.toLowerCase()
          filtered = filtered.filter(
            (t) =>
              t.name.toLowerCase().includes(lowerQuery) ||
              t.description?.toLowerCase().includes(lowerQuery) ||
              t.sql_template.toLowerCase().includes(lowerQuery) ||
              t.tags.some((tag) => tag.toLowerCase().includes(lowerQuery))
          )
        }

        // Apply is_public filter
        if (filters.is_public !== undefined) {
          filtered = filtered.filter((t) => t.is_public === filters.is_public)
        }

        // Sort
        filtered.sort((a, b) => {
          switch (sortBy) {
            case 'usage':
              return b.usage_count - a.usage_count
            case 'newest':
              return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
            case 'name':
              return a.name.localeCompare(b.name)
            case 'updated':
              return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
            default:
              return 0
          }
        })

        return filtered
      },

      // ========================================================================
      // Utility
      // ========================================================================

      clearError: () => set({ error: null }),

      reset: () => set(initialState),
    }),
    { name: 'TemplatesStore' }
  )
)
