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
  environments?: string[] // Environment tags like "local", "dev", "prod"
}

interface ConnectionState {
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  defaultConnectionId: string | null
  autoConnectEnabled: boolean
  isConnecting: boolean
  activeEnvironmentFilter: string | null // null = "All", otherwise specific environment
  availableEnvironments: string[]

  addConnection: (connection: Omit<DatabaseConnection, 'id' | 'isConnected' | 'sessionId'>) => void
  updateConnection: (id: string, updates: Partial<DatabaseConnection>) => void
  removeConnection: (id: string) => void
  setActiveConnection: (connection: DatabaseConnection | null) => void
  setDefaultConnection: (connectionId: string | null) => void
  getDefaultConnection: () => DatabaseConnection | null
  setAutoConnect: (enabled: boolean) => void
  connectToDatabase: (connectionId: string) => Promise<void>
  disconnectFromDatabase: (connectionId: string) => Promise<void>
  setEnvironmentFilter: (env: string | null) => void
  getFilteredConnections: () => DatabaseConnection[]
  addEnvironmentToConnection: (connId: string, env: string) => void
  removeEnvironmentFromConnection: (connId: string, env: string) => void
  refreshAvailableEnvironments: () => void
}

export const useConnectionStore = create<ConnectionState>()(
  devtools(
    persist(
      (set, get) => ({
        connections: [],
        activeConnection: null,
        defaultConnectionId: null,
        autoConnectEnabled: true,
        isConnecting: false,
        activeEnvironmentFilter: null, // null = "All"
        availableEnvironments: [],

        addConnection: (connectionData) => {
          const newConnection: DatabaseConnection = {
            ...connectionData,
            id: crypto.randomUUID(),
            isConnected: false,
          }

          set((state) => ({
            connections: [...state.connections, newConnection],
          }))
          
          // Auto-connect if enabled
          if (get().autoConnectEnabled) {
            console.log(`⚡ Auto-connecting to: ${newConnection.name}`)
            
            // Delay slightly to ensure state is updated
            setTimeout(async () => {
              try {
                await get().connectToDatabase(newConnection.id)
                console.log(`✓ Auto-connected to: ${newConnection.name}`)
              } catch (error) {
                console.warn(`✗ Failed to auto-connect to ${newConnection.name}:`, error)
              }
            }, 100)
          }
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
            defaultConnectionId: state.defaultConnectionId === id ? null : state.defaultConnectionId,
          }))
        },

        setActiveConnection: (connection) => {
          set({ activeConnection: connection })
        },
        
        setDefaultConnection: (connectionId) => {
          // Validate connection exists and is connected
          const conn = get().connections.find(c => c.id === connectionId)
          if (connectionId && (!conn || !conn.isConnected)) {
            console.warn('Cannot set default: connection not found or not connected')
            return
          }
          
          set({ defaultConnectionId: connectionId })
          
          // Emit event for UI updates
          if (typeof window !== 'undefined') {
            window.dispatchEvent(new CustomEvent('connection:default-changed', {
              detail: { connectionId, connection: conn }
            }))
          }
          
          console.log(`⭐ Default connection ${connectionId ? 'set to' : 'cleared'}:`, conn?.name || '')
        },
        
        getDefaultConnection: () => {
          const state = get()
          if (!state.defaultConnectionId) return null
          return state.connections.find(c => c.id === state.defaultConnectionId) || null
        },
        
        setAutoConnect: (enabled) => {
          set({ autoConnectEnabled: enabled })
          console.log(`⚙️ Auto-connect ${enabled ? 'enabled' : 'disabled'}`)
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

        setEnvironmentFilter: async (env) => {
          set({ activeEnvironmentFilter: env })
          console.log(`🌍 Environment filter set to: ${env || 'All'}`)
          
          // Auto-connect filtered connections if not already connected
          const filtered = get().getFilteredConnections()
          const disconnected = filtered.filter(c => !c.isConnected)
          
          if (disconnected.length > 0) {
            console.log(`⚡ Auto-connecting ${disconnected.length} filtered connections...`)
            
            // Connect in parallel
            const connectPromises = disconnected.map(async (conn) => {
              try {
                await get().connectToDatabase(conn.id)
                console.log(`  ✓ Auto-connected: ${conn.name}`)
              } catch (error) {
                console.warn(`  ✗ Failed to auto-connect ${conn.name}:`, error)
              }
            })
            
            await Promise.allSettled(connectPromises)
            console.log(`⚡ Auto-connect complete`)
          }
        },

        getFilteredConnections: () => {
          const state = get()
          const { connections, activeEnvironmentFilter } = state

          // If no filter, return all connections
          if (!activeEnvironmentFilter) {
            return connections
          }

          // Filter by environment
          return connections.filter((conn) => {
            // Include connections with no environments (backward compatibility)
            if (!conn.environments || conn.environments.length === 0) {
              return true
            }
            // Include if connection has the active environment
            return conn.environments.includes(activeEnvironmentFilter)
          })
        },

        addEnvironmentToConnection: (connId, env) => {
          set((state) => ({
            connections: state.connections.map((conn) => {
              if (conn.id === connId) {
                const environments = conn.environments || []
                if (!environments.includes(env)) {
                  return { ...conn, environments: [...environments, env] }
                }
              }
              return conn
            }),
          }))
          get().refreshAvailableEnvironments()
        },

        removeEnvironmentFromConnection: (connId, env) => {
          set((state) => ({
            connections: state.connections.map((conn) => {
              if (conn.id === connId && conn.environments) {
                return {
                  ...conn,
                  environments: conn.environments.filter((e) => e !== env),
                }
              }
              return conn
            }),
          }))
          get().refreshAvailableEnvironments()
        },

        refreshAvailableEnvironments: () => {
          const state = get()
          const envSet = new Set<string>()
          
          state.connections.forEach((conn) => {
            conn.environments?.forEach((env) => envSet.add(env))
          })

          set({ availableEnvironments: Array.from(envSet).sort() })
        },
      }),
      {
        name: 'connection-store',
        partialize: (state) => ({
          connections: state.connections.map(({ sessionId, isConnected, lastUsed, ...rest }) => { // eslint-disable-line @typescript-eslint/no-unused-vars
            return { ...rest }
          }),
          defaultConnectionId: state.defaultConnectionId,
          autoConnectEnabled: state.autoConnectEnabled,
          activeEnvironmentFilter: state.activeEnvironmentFilter,
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
          // Keep defaultConnectionId, autoConnectEnabled, and activeEnvironmentFilter on rehydrate
          
          // Refresh available environments
          state.refreshAvailableEnvironments()
        },
      }
    )
  )
)

// Expose store globally for cross-store access (avoids circular imports)
if (typeof window !== 'undefined') {
  (window as any).__connectionStore = useConnectionStore
}
