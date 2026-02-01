/**
 * Query Editor Store
 * Manages editor state: tabs, active query, cursor position
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { useShallow } from 'zustand/react/shallow'

import { type DatabaseConnection } from './connection-store'
import type { QueryTab, QueryTabType } from './query-types'

interface QueryEditorState {
  tabs: QueryTab[]
  activeTabId: string | null

  // Actions
  createTab: (title?: string, options?: { connectionId?: string; type?: QueryTabType; aiSessionId?: string }) => string
  closeTab: (id: string) => void
  updateTab: (id: string, updates: Partial<QueryTab>) => void
  setActiveTab: (id: string) => void
  getActiveTab: () => QueryTab | undefined
}

export const useQueryEditorStore = create<QueryEditorState>()(
  devtools(
    persist(
      (set, get) => ({
        tabs: [],
        activeTabId: null,

        createTab: (title = 'New Query', options?: { connectionId?: string; type?: QueryTabType; aiSessionId?: string }) => {
          const desiredType = options?.type ?? 'sql'
          let initialConnectionId = options?.connectionId
          let environmentSnapshot: string | null = null

          // Get connection state for both connectionId and selectedConnectionIds
          const connectionState = window.__connectionStore?.getState?.()

          if (!initialConnectionId) {
            if (connectionState) {
              const { connections, activeConnection, activeEnvironmentFilter } = connectionState
              environmentSnapshot = activeEnvironmentFilter

              if (activeConnection) {
                initialConnectionId = activeConnection.id
              } else if (connections.length > 0) {
                const firstConnected = connections.find((c: DatabaseConnection) => c.isConnected)
                if (firstConnected) {
                  initialConnectionId = firstConnected.id
                }
              }
            }
          }

          const newTab: QueryTab = {
            id: crypto.randomUUID(),
            title,
            type: desiredType,
            content: '',
            isDirty: false,
            isExecuting: false,
            connectionId: initialConnectionId,
            selectedConnectionIds: initialConnectionId ? [initialConnectionId] : [],
            environmentSnapshot,
            aiSessionId: options?.aiSessionId,
          }

          set((state) => ({
            tabs: [...state.tabs, newTab],
            activeTabId: newTab.id,
          }))

          return newTab.id
        },

        closeTab: (id) => {
          set((state) => {
            const newTabs = state.tabs.filter((tab) => tab.id !== id)
            const wasActive = state.activeTabId === id

            return {
              tabs: newTabs,
              activeTabId: wasActive
                ? newTabs.length > 0
                  ? newTabs[newTabs.length - 1].id
                  : null
                : state.activeTabId,
            }
          })
        },

        updateTab: (id, updates) => {
          set((state) => ({
            tabs: state.tabs.map((tab) => {
              if (tab.id !== id) {
                return tab
              }

              return {
                ...tab,
                ...updates,
                type: tab.type,
                aiSessionId: tab.aiSessionId,
              }
            }),
          }))
        },

        setActiveTab: (id) => {
          set({ activeTabId: id })
        },

        getActiveTab: () => {
          const state = get()
          return state.tabs.find((tab) => tab.id === state.activeTabId)
        },
      }),
      {
        name: 'query-editor-store',
        partialize: (state) => ({
          tabs: state.tabs,
          activeTabId: state.activeTabId,
        }),
      }
    ),
    {
      name: 'query-editor-store',
    }
  )
)

// Selectors with useShallow for efficient subscriptions
export const useQueryEditorTabs = () =>
  useQueryEditorStore(useShallow((state) => state.tabs))

export const useActiveTabId = () =>
  useQueryEditorStore((state) => state.activeTabId)

export const useActiveTab = () =>
  useQueryEditorStore(
    useShallow((state) => state.tabs.find((tab) => tab.id === state.activeTabId))
  )

export const useQueryEditorActions = () =>
  useQueryEditorStore(
    useShallow((state) => ({
      createTab: state.createTab,
      closeTab: state.closeTab,
      updateTab: state.updateTab,
      setActiveTab: state.setActiveTab,
    }))
  )

// Re-export types
export type { QueryTab, QueryTabType } from './query-types'
