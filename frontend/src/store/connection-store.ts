import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { wailsEndpoints } from '@/lib/wails-api'
import { SSHAuthMethod } from '@/generated/database'
import { getSecureStorage, migratePasswordsFromLocalStorage } from '@/lib/secure-storage'
import { useTierStore } from './tier-store'

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
  lastActiveConnectionId: string | null // Track last active connection for auto-reconnect
  autoConnectEnabled: boolean
  isConnecting: boolean
  activeEnvironmentFilter: string | null // null = "All", otherwise specific environment
  availableEnvironments: string[]

  addConnection: (connection: Omit<DatabaseConnection, 'id' | 'isConnected' | 'sessionId'>) => Promise<void>
  updateConnection: (id: string, updates: Partial<DatabaseConnection>) => Promise<void>
  removeConnection: (id: string) => Promise<void>
  setActiveConnection: (connection: DatabaseConnection | null) => void
  setAutoConnect: (enabled: boolean) => void
  connectToDatabase: (connectionId: string) => Promise<void>
  disconnectFromDatabase: (connectionId: string) => Promise<void>
  fetchDatabases: (connectionId: string) => Promise<string[]>
  switchDatabase: (connectionId: string, database: string) => Promise<void>
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
        lastActiveConnectionId: null,
        autoConnectEnabled: true,
        isConnecting: false,
        activeEnvironmentFilter: null, // null = "All"
        availableEnvironments: [],

        addConnection: async (connectionData) => {
          // Check tier limit before adding
          const currentConnections = get().connections.length
          const tierStore = useTierStore.getState()
          const limitCheck = tierStore.checkLimit('connections', currentConnections + 1)

          if (!limitCheck.allowed) {
            console.warn('Connection limit reached:', limitCheck)
            // Dispatch event for upgrade prompt
            window.dispatchEvent(
              new CustomEvent('showUpgradeDialog', {
                detail: {
                  limitName: 'connections',
                  currentTier: tierStore.currentTier,
                  usage: currentConnections,
                  limit: limitCheck.limit,
                },
              })
            )
            throw new Error(
              `Connection limit reached. Current tier allows ${limitCheck.limit} connections.`
            )
          }

          const newConnection: DatabaseConnection = {
            ...connectionData,
            id: crypto.randomUUID(),
            isConnected: false,
          }

          // Store sensitive credentials in secure storage
          const secureStorage = getSecureStorage()
          await secureStorage.setCredentials(newConnection.id, {
            password: connectionData.password,
            sshPassword: connectionData.sshTunnel?.password,
            sshPrivateKey: connectionData.sshTunnel?.privateKey
          })

          // Remove passwords from the connection object before storing
          const safeConnection = {
            ...newConnection,
            password: undefined,
            sshTunnel: newConnection.sshTunnel ? {
              ...newConnection.sshTunnel,
              password: undefined,
              privateKey: undefined
            } : undefined
          }

          set((state) => ({
            connections: [...state.connections, safeConnection],
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

        updateConnection: async (id, updates) => {
          // If updating password or SSH credentials, store in secure storage
          if (updates.password || updates.sshTunnel?.password || updates.sshTunnel?.privateKey) {
            const secureStorage = getSecureStorage()
            const existing = (await secureStorage.getCredentials(id)) || { connectionId: id }
            await secureStorage.setCredentials(id, {
              password: updates.password ?? existing.password,
              sshPassword: updates.sshTunnel?.password ?? existing.sshPassword,
              sshPrivateKey: updates.sshTunnel?.privateKey ?? existing.sshPrivateKey
            })

            // Remove passwords from updates
            const safeUpdates = {
              ...updates,
              password: undefined,
              sshTunnel: updates.sshTunnel ? {
                ...updates.sshTunnel,
                password: undefined,
                privateKey: undefined
              } : updates.sshTunnel
            }

            set((state) => ({
              connections: state.connections.map((conn) =>
                conn.id === id ? { ...conn, ...safeUpdates } : conn
              ),
              activeConnection:
                state.activeConnection?.id === id
                  ? { ...state.activeConnection, ...safeUpdates }
                  : state.activeConnection,
            }))
          } else {
            set((state) => ({
              connections: state.connections.map((conn) =>
                conn.id === id ? { ...conn, ...updates } : conn
              ),
              activeConnection:
                state.activeConnection?.id === id
                  ? { ...state.activeConnection, ...updates }
                  : state.activeConnection,
            }))
          }
        },

        removeConnection: async (id) => {
          // Remove from secure storage
          const secureStorage = getSecureStorage()
          await secureStorage.removeCredentials(id)

          set((state) => ({
            connections: state.connections.filter((conn) => conn.id !== id),
            activeConnection: state.activeConnection?.id === id ? null : state.activeConnection,
          }))
        },

        setActiveConnection: (connection) => {
          set({
            activeConnection: connection,
            lastActiveConnectionId: connection?.id ?? null
          })
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
            // Retrieve password from secure storage
            const secureStorage = getSecureStorage()
            const credentials = await secureStorage.getCredentials(connectionId)

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
              id: connectionId, // Pass stored connection ID for reconnecting
              name: connection.name,
              type: connection.type,
              host: connection.host ?? '',
              port: connection.port ?? 0,
              database: connection.database,
              username: connection.username ?? '',
              password: credentials?.password ?? '',
              ssl_mode: connection.sslMode ?? 'prefer',  // Pass SSL mode
              parameters: aliasParameters,
            })

            if (!response.success || !response.data?.id) {
              throw new Error(response.message || 'Failed to create connection')
            }

            // Save connection metadata to backend storage for RAG indexing
            try {
              await wailsEndpoints.connections.save({
                id: connectionId,
                name: connection.name,
                type: connection.type,
                host: connection.host ?? '',
                port: connection.port ?? 0,
                database: connection.database,
                username: connection.username ?? '',
                password: credentials?.password ?? '',
                ssl_mode: connection.sslMode ?? 'prefer',  // Pass SSL mode
                parameters: aliasParameters,
              })
            } catch (saveError) {
              console.warn('Failed to save connection metadata:', saveError)
              // Don't fail the connection if saving metadata fails
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
              lastActiveConnectionId: connectionId,
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

        fetchDatabases: async (connectionId) => {
          const state = get()
          const connection = state.connections.find((conn) => conn.id === connectionId || conn.sessionId === connectionId)

          if (!connection) {
            // If we don't know about this connection but it might be a live session ID, try once.
            const response = await wailsEndpoints.connections.listDatabases(connectionId)
            if (!response.success) {
              throw new Error(response.message || 'Unable to fetch databases for this connection.')
            }
            return response.databases ?? []
          }

          if (!connection.sessionId) {
            // Not connected yet; nothing to return.
            return []
          }

          const response = await wailsEndpoints.connections.listDatabases(connection.sessionId)
          if (!response.success) {
            throw new Error(response.message || 'Unable to fetch databases for this connection.')
          }
          return response.databases ?? []
        },

        switchDatabase: async (connectionId, database) => {
          const state = get()
          const connection = state.connections.find((conn) => conn.id === connectionId || conn.sessionId === connectionId)
          if (!connection) {
            throw new Error('Connection not found')
          }

          const managerId = connection.sessionId
          if (!managerId) {
            throw new Error('Connection is not active')
          }

          const response = await wailsEndpoints.connections.switchDatabase(managerId, database)
          if (!response.success) {
            throw new Error(response.message || 'Failed to switch database.')
          }

          set((currentState) => ({
            connections: currentState.connections.map((conn) =>
              conn.id === connection.id ? { ...conn, database } : conn
            ),
            activeConnection:
              currentState.activeConnection?.id === connection.id
                ? { ...currentState.activeConnection, database }
                : currentState.activeConnection,
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
          connections: state.connections.map(({ sessionId: _sessionId, isConnected: _isConnected, lastUsed: _lastUsed, password: _password, ...rest }) => {
            // Strip sensitive credentials and connection state
            const { sshTunnel, ...safeRest } = rest
            return {
              ...safeRest,
              // Strip passwords from SSH tunnel config
              sshTunnel: sshTunnel ? {
                ...sshTunnel,
                password: undefined,
                privateKey: undefined
              } : undefined
            }
          }),
          lastActiveConnectionId: state.lastActiveConnectionId,
          autoConnectEnabled: state.autoConnectEnabled,
          activeEnvironmentFilter: state.activeEnvironmentFilter,
        }),
        onRehydrateStorage: () => (state, error) => {
          if (error) {
            console.error('Failed to rehydrate connection store', error)
            return
          }
          if (!state) return

          // Migrate any passwords from localStorage to sessionStorage
          migratePasswordsFromLocalStorage()

          state.connections = state.connections.map((connection) => ({
            ...connection,
            sessionId: undefined,
            isConnected: false,
            lastUsed: undefined,
            password: undefined, // Ensure no passwords in rehydrated state
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

/**
 * Initialize connection store and auto-connect to last active connection
 * Call this on app startup after store hydration
 */
export async function initializeConnectionStore() {
  const state = useConnectionStore.getState()

  // Check if auto-connect is enabled
  if (!state.autoConnectEnabled) {
    console.debug('Auto-connect is disabled')
    return
  }

  // Get last active connection
  const lastConnectionId = state.lastActiveConnectionId
  if (!lastConnectionId) {
    console.debug('No last active connection found')
    return
  }

  // Find the connection
  const connection = state.connections.find(c => c.id === lastConnectionId)
  if (!connection) {
    console.debug('Last active connection no longer exists:', lastConnectionId)
    return
  }

  // Check if already connected (shouldn't happen, but safety check)
  if (connection.isConnected) {
    console.debug('Connection already active:', connection.name)
    return
  }

  // Auto-connect in background
  console.debug('Auto-connecting to:', connection.name)
  try {
    await state.connectToDatabase(lastConnectionId)
    console.debug('Auto-connect successful:', connection.name)
  } catch (error) {
    console.warn('Auto-connect failed:', connection.name, error)
    // Fail silently - don't block app startup
  }
}
