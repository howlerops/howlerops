import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { wailsEndpoints } from '@/lib/wails-api'
import { DatabaseType, SSHAuthMethod } from '@/generated/database'

export type DatabaseTypeString =
  | 'postgresql'
  | 'mysql'
  | 'sqlite'
  | 'mssql'
  | 'mariadb'
  | 'elasticsearch'
  | 'opensearch'
  | 'clickhouse'
  | 'mongodb'
  | 'tidb'

export interface SSHTunnelConfig {
  host: string
  port: number
  user: string
  authMethod: SSHAuthMethod
  password?: string
  privateKey?: string
  privateKeyPath?: string
  knownHostsPath?: string
  strictHostKeyChecking: boolean
  timeoutSeconds: number
  keepAliveIntervalSeconds: number
}

export interface VPCConfig {
  vpcId: string
  subnetId: string
  securityGroupIds: string[]
  privateLinkService?: string
  endpointServiceName?: string
  customConfig?: Record<string, string>
}

export interface DatabaseConnection {
  id: string
  sessionId?: string
  name: string
  type: DatabaseTypeString
  host?: string
  port?: number
  database: string
  username?: string
  password?: string
  sslMode?: string

  // SSH Tunnel
  useTunnel?: boolean
  sshTunnel?: SSHTunnelConfig

  // VPC
  useVpc?: boolean
  vpcConfig?: VPCConfig

  // Database-specific parameters
  parameters?: Record<string, string>

  isConnected: boolean
  lastUsed?: Date
  environments?: string[] // Environment tags like "local", "dev", "prod"
}

interface ConnectionState {
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  autoConnectEnabled: boolean
  isConnecting: boolean
  activeEnvironmentFilter: string | null // null = "All", otherwise specific environment
  availableEnvironments: string[]

  addConnection: (connection: Omit<DatabaseConnection, 'id' | 'isConnected' | 'sessionId'>) => void
  updateConnection: (id: string, updates: Partial<DatabaseConnection>) => void
  removeConnection: (id: string) => void
  setActiveConnection: (connection: DatabaseConnection | null) => void
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
            // Delay slightly to ensure state is updated
            setTimeout(async () => {
              try {
                await get().connectToDatabase(newConnection.id)
              } catch {
                // Auto-connect failed
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
          }))
        },

        setActiveConnection: (connection) => {
          set({ activeConnection: connection })
        },
        
        setAutoConnect: (enabled) => {
          set({ autoConnectEnabled: enabled })
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
            const alias = connection.name?.trim()
            const aliasParameters: Record<string, string> = {}

            if (alias) {
              aliasParameters.alias = alias

              const slug = alias.replace(/[^\w-]/g, '-')
              if (slug && slug !== alias) {
                aliasParameters.alias_slug = slug
              }

              const lower = alias.toLowerCase()
              if (lower !== alias) {
                aliasParameters.alias_lower = lower
              }
            }

            const response = await wailsEndpoints.connections.create({
              name: connection.name,
              type: connection.type,
              host: connection.host ?? '',
              port: connection.port ?? 0,
              database: connection.database,
              username: connection.username ?? '',
              password: connection.password ?? '',
              parameters: aliasParameters,
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
          
          // Auto-connect filtered connections if not already connected
          const filtered = get().getFilteredConnections()
          const disconnected = filtered.filter(c => !c.isConnected)
          
          if (disconnected.length > 0) {
            // Connect in parallel
            const connectPromises = disconnected.map(async (conn) => {
              try {
                await get().connectToDatabase(conn.id)
              } catch {
                // Auto-connect failed
              }
            })
            
            await Promise.allSettled(connectPromises)
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
          // Keep autoConnectEnabled and activeEnvironmentFilter on rehydrate
          
          // Refresh available environments
          state.refreshAvailableEnvironments()
        },
      }
    )
  )
)

// Expose store globally for cross-store access (avoids circular imports)
declare global {
  interface Window {
    __connectionStore?: typeof useConnectionStore;
  }
}

if (typeof window !== 'undefined') {
  window.__connectionStore = useConnectionStore
}
