import { act, renderHook } from '@testing-library/react'
import { createRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const executeQueryMock = vi.fn()
const editorStoreState = {
  tabs: [] as Array<Record<string, unknown>>,
  activeTabId: null as string | null,
}
const historyStoreState = {
  results: [] as Array<Record<string, unknown>>,
}

vi.mock('@/store/query-editor-store', () => ({
  useQueryEditorStore: () => editorStoreState,
}))

vi.mock('@/store/query-engine-store', () => ({
  useQueryEngineStore: () => ({
    executeQuery: executeQueryMock,
  }),
}))

vi.mock('@/store/query-history-store', () => ({
  useQueryHistoryStore: () => historyStoreState,
}))

import { useQueryExecution } from './use-query-execution'

describe('useQueryExecution', () => {
  beforeEach(() => {
    executeQueryMock.mockReset()
    editorStoreState.tabs = []
    editorStoreState.activeTabId = null
    historyStoreState.results = []
  })

  it('shows the database prompt instead of executing when no connection is selected', async () => {
    editorStoreState.tabs = [
      {
        id: 'tab-1',
        type: 'sql',
        connectionId: undefined,
        selectedConnectionIds: [],
      },
    ]
    editorStoreState.activeTabId = 'tab-1'

    const setPendingQuery = vi.fn()
    const setShowDatabasePrompt = vi.fn()
    const setLastExecutionError = vi.fn()
    const setEditorContent = vi.fn()

    const editorRef = createRef<{
      getValue: () => string
      getSelectedText: () => string
      getCursorOffset: () => number
    }>()
    editorRef.current = {
      getValue: () => 'select * from users',
      getSelectedText: () => '',
      getCursorOffset: () => 'select * from users'.length,
    }

    const { result } = renderHook(() =>
      useQueryExecution({
        editorRef,
        editorContent: 'select * from users',
        setEditorContent,
        setLastExecutionError,
        setPendingQuery,
        setShowDatabasePrompt,
        flushTabUpdate: vi.fn(),
        pendingTabUpdateRef: { current: null },
      })
    )

    await act(async () => {
      await result.current.handleExecuteQuery()
    })

    expect(setPendingQuery).toHaveBeenCalledWith('select * from users')
    expect(setShowDatabasePrompt).toHaveBeenCalledWith(true)
    expect(executeQueryMock).not.toHaveBeenCalled()
  })

  it('resets pagination to the first page when the executed query changes', async () => {
    editorStoreState.tabs = [
      {
        id: 'tab-1',
        type: 'sql',
        connectionId: 'conn-1',
        selectedConnectionIds: ['conn-1'],
      },
    ]
    editorStoreState.activeTabId = 'tab-1'
    historyStoreState.results = [
      {
        id: 'result-1',
        tabId: 'tab-1',
        query: 'select * from old_table',
      },
    ]

    const editorRef = createRef<{
      getValue: () => string
      getSelectedText: () => string
      getCursorOffset: () => number
    }>()
    editorRef.current = {
      getValue: () => 'select * from new_table',
      getSelectedText: () => '',
      getCursorOffset: () => 'select * from new_table'.length,
    }

    const { result } = renderHook(() =>
      useQueryExecution({
        editorRef,
        editorContent: 'select * from new_table',
        setEditorContent: vi.fn(),
        setLastExecutionError: vi.fn(),
        setPendingQuery: vi.fn(),
        setShowDatabasePrompt: vi.fn(),
        flushTabUpdate: vi.fn(),
        pendingTabUpdateRef: { current: null },
      })
    )

    await act(async () => {
      await result.current.handleExecuteQuery()
    })

    expect(executeQueryMock).toHaveBeenCalledWith('tab-1', 'select * from new_table', 'conn-1', 100, 0)
  })
})
