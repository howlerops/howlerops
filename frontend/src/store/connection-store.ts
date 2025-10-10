import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { wailsEndpoints } from '@/lib/wails-api'

export interface DatabaseConnection {
  id: string
  sessionId?: string
  name: string
  type: 'postgresql' | 'mysql' | 'sqlite' | 'mssql'
  host?: string
  port?: number
  database: string
  username?: string
  password?: string
  isConnected: boolean
  lastUsed?: Date
}

interface ConnectionState {
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  isConnecting: boolean

  addConnection: (connection: Omit<DatabaseConnection, 'id' | 'isConnected' | 'sessionId'>) => void
  updateConnection: (id: string, updates: Partial<DatabaseConnection>) => void
  removeConnection: (id: string) => void
  setActiveConnection: (connection: DatabaseConnection | null) => void
  connectToDatabase: (connectionId: string) => Promise<void>
  disconnectFromDatabase: (connectionId: string) => Promise<void>
}

export const useConnectionStore = create<ConnectionState>()(
  devtools(
    persist(
      (set, get) => ({
        connections: [],
        activeConnection: null,
        isConnecting: false,

        addConnection: (connectionData) => {
          const newConnection: DatabaseConnection = {
            ...connectionData,
            id: crypto.randomUUID(),
            isConnected: false,
          }

          set((state) => ({
            connections: [...state.connections, newConnection],
          }))
        },

        updateConnection: (id, updates) => {
          set((state) => ({
            connections: state.connections.map((conn) =>
              conn.id === id ? { ...conn, ...updates } : conn
            ),
            activeConnection:
              state.activeConnection?.id === id
                ? { ...state.activeConnection, ...updates }
                : state.activeConnection,
          }))
        },

        removeConnection: (id) => {
          set((state) => ({
            connections: state.connections.filter((conn) => conn.id !== id),
            activeConnection: state.activeConnection?.id === id ? null : state.activeConnection,
          }))
        },

        setActiveConnection: (connection) => {
          set({ activeConnection: connection })
        },

        connectToDatabase: async (connectionId) => {
          const state = get()
          const connection = state.connections.find((conn) => conn.id === connectionId)
          if (!connection) {
            console.error(`Connection ${connectionId} not found`)
            return
          }

          set({ isConnecting: true })
          try {
            const response = await wailsEndpoints.connections.create({
              name: connection.name,
              type: connection.type,
              host: connection.host ?? '',
              port: connection.port ?? 0,
              database: connection.database,
              username: connection.username ?? '',
              password: connection.password ?? '',
              parameters: {},
            })

            if (!response.success || !response.data?.id) {
              throw new Error(response.message || 'Failed to create connection')
            }

            const updatedConnection: DatabaseConnection = {
              ...connection,
              sessionId: response.data.id,
              isConnected: true,
              lastUsed: new Date(),
            }

            set((currentState) => ({
              connections: currentState.connections.map((conn) =>
                conn.id === connectionId ? updatedConnection : conn
              ),
              activeConnection: updatedConnection,
            }))
          } catch (error) {
            console.error('Failed to connect to database:', error)
            throw error
          } finally {
            set({ isConnecting: false })
          }
        },

        disconnectFromDatabase: async (connectionId) => {
          const state = get()
          const connection = state.connections.find((conn) => conn.id === connectionId)
          if (!connection) {
            return
          }

          if (connection.sessionId) {
            const response = await wailsEndpoints.connections.remove(connection.sessionId)
            if (!response.success) {
              console.error('Failed to remove connection:', response.message)
            }
          }

          const updatedConnection: DatabaseConnection = {
            ...connection,
            sessionId: undefined,
            isConnected: false,
          }

          set((currentState) => ({
            connections: currentState.connections.map((conn) =>
              conn.id === connectionId ? updatedConnection : conn
            ),
            activeConnection:
              currentState.activeConnection?.id === connectionId ? null : currentState.activeConnection,
          }))
        },
      }),
      {
        name: 'connection-store',
        partialize: (state) => ({
          connections: state.connections.map(({ sessionId, isConnected, lastUsed, ...rest }) => { // eslint-disable-line @typescript-eslint/no-unused-vars
            return { ...rest }
          }),
        }),
        onRehydrateStorage: () => (state, error) => {
          if (error) {
            console.error('Failed to rehydrate connection store', error)
            return
          }
          if (!state) return

          state.connections = state.connections.map((connection) => ({
            ...connection,
            sessionId: undefined,
            isConnected: false,
            lastUsed: undefined,
          }))
          state.activeConnection = null
        },
      }
    )
  )
)
