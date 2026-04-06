import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const {
  executeMock,
  getEditableMetadataMock,
  connectionStoreMock,
  editorStoreMock,
  historyStoreMock,
} = vi.hoisted(() => ({
  executeMock: vi.fn(),
  getEditableMetadataMock: vi.fn(),
  connectionStoreMock: {
    state: { connections: [] as unknown[] },
    getState() {
      return this.state
    },
    setState(partial: Record<string, unknown>) {
      this.state = { ...this.state, ...partial }
    },
  },
  editorStoreMock: {
    state: { tabs: [] as unknown[], activeTabId: null as string | null },
    getState() {
      return {
        ...this.state,
        updateTab: (id: string, updates: Record<string, unknown>) => {
          this.state.tabs = this.state.tabs.map((tab) =>
            (tab as { id: string }).id === id ? { ...tab, ...updates } : tab
          )
        },
      }
    },
    setState(partial: Record<string, unknown>) {
      this.state = { ...this.state, ...partial }
    },
  },
  historyStoreMock: {
    state: {
      results: [] as unknown[],
      addResult: vi.fn((result) => {
        const saved = { ...result, id: `result-${historyStoreMock.state.results.length + 1}`, timestamp: new Date() }
        historyStoreMock.state.results = [...historyStoreMock.state.results, saved]
        return saved
      }),
      updateResultRows: vi.fn(),
      updateResultEditable: vi.fn(),
      updateResultProcessing: vi.fn(),
    },
    getState() {
      return this.state
    },
    setState(partial: Record<string, unknown> | ((state: typeof historyStoreMock.state) => Record<string, unknown>)) {
      const next = typeof partial === 'function' ? partial(this.state) : partial
      this.state = { ...this.state, ...next }
    },
  },
}))

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: vi.fn(),
  },
}))

vi.mock('@/lib/api-client', () => ({
  api: {
    queries: {
      execute: executeMock,
      getEditableMetadata: getEditableMetadataMock,
    },
  },
}))

vi.mock('./connection-store', () => ({
  useConnectionStore: connectionStoreMock,
}))

vi.mock('./query-editor-store', () => ({
  useQueryEditorStore: editorStoreMock,
}))

vi.mock('./query-history-store', () => ({
  useQueryHistoryStore: historyStoreMock,
}))

import { useQueryEngineStore } from './query-engine-store'

describe('query-engine-store', () => {
  beforeEach(() => {
    executeMock.mockReset()
    getEditableMetadataMock.mockReset()

    connectionStoreMock.setState({ connections: [] })
    editorStoreMock.setState({ tabs: [], activeTabId: null })
    historyStoreMock.state.results = []
    historyStoreMock.state.addResult.mockClear()
    historyStoreMock.state.updateResultRows.mockClear()
    historyStoreMock.state.updateResultEditable.mockClear()
    historyStoreMock.state.updateResultProcessing.mockClear()
    useQueryEngineStore.setState({ executingQueries: new Map() })
  })

  afterEach(() => {
    useQueryEngineStore.setState({ executingQueries: new Map() })
  })

  it('adds an error result when no connection is selected', async () => {
    editorStoreMock.setState({
      tabs: [
        {
          id: 'tab-1',
          title: 'Test',
          type: 'sql',
          content: 'select 1',
          isDirty: false,
          isExecuting: false,
        },
      ],
      activeTabId: 'tab-1',
    })

    await useQueryEngineStore.getState().executeQuery('tab-1', 'select 1')

    expect(executeMock).not.toHaveBeenCalled()
    expect(historyStoreMock.state.results).toHaveLength(1)
    expect((historyStoreMock.state.results[0] as { error?: string }).error).toContain('No connection selected')
  })

  it('stores a normalized successful result and clears executing state', async () => {
    editorStoreMock.setState({
      tabs: [
        {
          id: 'tab-1',
          title: 'Test',
          type: 'sql',
          content: 'select 1',
          isDirty: true,
          isExecuting: false,
          connectionId: 'conn-1',
        },
      ],
      activeTabId: 'tab-1',
    })

    connectionStoreMock.setState({
      connections: [
        {
          id: 'conn-1',
          sessionId: 'session-1',
          name: 'Primary',
          type: 'postgresql',
          database: 'app',
          isConnected: true,
        },
      ],
    })

    executeMock.mockResolvedValue({
      success: true,
      data: {
        queryId: 'query-1',
        success: true,
        columns: [{ name: 'id', dataType: 'int' }],
        rows: [[1]],
        rowCount: 1,
        stats: { duration: '5ms', affectedRows: 0 },
        warnings: [],
        totalRows: 10,
        pagedRows: 1,
        hasMore: true,
        offset: 0,
      },
    })

    await useQueryEngineStore.getState().executeQuery('tab-1', 'select 1')

    expect(executeMock).toHaveBeenCalledWith('session-1', 'select 1', 5000, 0)
    expect(historyStoreMock.state.results).toHaveLength(1)
    const result = historyStoreMock.state.results[0] as { columns: string[]; totalRows?: number; hasMore?: boolean }
    expect(result.columns).toEqual(['id'])
    expect(result.totalRows).toBe(10)
    expect(result.hasMore).toBe(true)
    expect(useQueryEngineStore.getState().executingQueries.size).toBe(0)
  })
})
