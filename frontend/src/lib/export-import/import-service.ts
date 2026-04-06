/**
 * Connection Import Service
 *
 * Handles importing database connections from a JSON export file.
 * See ADR-027 for architecture decisions.
 *
 * @module lib/export-import/import-service
 */

import { useConnectionStore, type DatabaseConnection } from '@/store/connection-store'
import { getSecureStorage } from '@/lib/secure-storage'

import {
  ConnectionExportFile,
  ExportedConnection,
  ImportOptions,
  ImportResult,
} from './types'
import {
  findConflictingConnections,
  validateAllConnections,
  validateExportFile,
} from './validation'

/**
 * Parse and validate an export file from JSON string
 */
export function parseExportFile(jsonString: string): ConnectionExportFile {
  const data = JSON.parse(jsonString)
  const validation = validateExportFile(data)

  if (!validation.isValid) {
    throw new Error(`Invalid export file: ${validation.errors.join(', ')}`)
  }

  return data as ConnectionExportFile
}

/**
 * Read and parse an export file from a File object
 */
export async function readExportFile(file: File): Promise<ConnectionExportFile> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()

    reader.onload = (e) => {
      try {
        const content = e.target?.result as string
        const exportFile = parseExportFile(content)
        resolve(exportFile)
      } catch (error) {
        reject(error instanceof Error ? error : new Error('Failed to parse export file'))
      }
    }

    reader.onerror = () => {
      reject(new Error('Failed to read file'))
    }

    reader.readAsText(file)
  })
}

/**
 * Get IDs of connections that conflict with existing ones
 */
export function getConflictingIds(exportFile: ConnectionExportFile): string[] {
  const state = useConnectionStore.getState()
  const existingIds = new Set(state.connections.map(c => c.id))
  return findConflictingConnections(exportFile.connections, existingIds)
}

/**
 * Preview what will happen during import (dry run)
 */
export function previewImport(
  exportFile: ConnectionExportFile,
  options: ImportOptions
): {
  toImport: number
  toSkip: number
  toOverwrite: number
  invalid: number
} {
  const state = useConnectionStore.getState()
  const existingIds = new Set(state.connections.map(c => c.id))
  const validationResults = validateAllConnections(exportFile)

  let toImport = 0
  let toSkip = 0
  let toOverwrite = 0
  let invalid = 0

  for (const { connection, validation } of validationResults) {
    if (!validation.isValid) {
      invalid++
      continue
    }

    const isConflict = existingIds.has(connection.id)

    if (!isConflict) {
      toImport++
    } else {
      switch (options.conflictResolution) {
        case 'skip':
          toSkip++
          break
        case 'overwrite':
          toOverwrite++
          break
        case 'keep-both':
          toImport++
          break
      }
    }
  }

  return { toImport, toSkip, toOverwrite, invalid }
}

/**
 * Import connections from an export file
 */
export async function importConnections(
  exportFile: ConnectionExportFile,
  options: ImportOptions
): Promise<ImportResult> {
  const result: ImportResult = {
    imported: 0,
    skipped: 0,
    overwritten: 0,
    failed: [],
  }

  const state = useConnectionStore.getState()
  const existingIds = new Set(state.connections.map(c => c.id))
  const validationResults = validateAllConnections(exportFile)

  for (const { connection, validation } of validationResults) {
    // Skip invalid connections
    if (!validation.isValid) {
      result.failed.push({
        connectionName: connection.name || 'Unknown',
        originalId: connection.id || 'no-id',
        reason: validation.errors.join('; '),
      })
      continue
    }

    const isConflict = existingIds.has(connection.id)

    try {
      if (!isConflict) {
        // No conflict - import directly
        await importSingleConnection(connection)
        result.imported++
      } else {
        // Handle conflict based on resolution strategy
        switch (options.conflictResolution) {
          case 'skip':
            result.skipped++
            break

          case 'overwrite':
            await overwriteConnection(connection)
            result.overwritten++
            break

          case 'keep-both': {
            // Import with new UUID
            const newConnection = { ...connection, id: crypto.randomUUID() }
            await importSingleConnection(newConnection)
            result.imported++
            break
          }
        }
      }
    } catch (error) {
      result.failed.push({
        connectionName: connection.name,
        originalId: connection.id,
        reason: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }

  return result
}

/**
 * Import a single connection into the store
 */
async function importSingleConnection(exported: ExportedConnection): Promise<void> {
  // Build connection object
  const connection: Omit<DatabaseConnection, 'isConnected' | 'sessionId'> = {
    id: exported.id,
    name: exported.name,
    type: exported.type,
    host: exported.host,
    port: exported.port,
    database: exported.database,
    username: exported.username,
    sslMode: exported.sslMode,
    environments: exported.environments,
    useTunnel: exported.useTunnel,
    useVpc: exported.useVpc,
    parameters: exported.parameters,
  }

  // Reconstruct SSH tunnel config
  if (exported.sshTunnel) {
    connection.sshTunnel = {
      host: exported.sshTunnel.host,
      port: exported.sshTunnel.port,
      user: exported.sshTunnel.user,
      authMethod: exported.sshTunnel.authMethod,
      privateKeyPath: exported.sshTunnel.privateKeyPath,
      knownHostsPath: exported.sshTunnel.knownHostsPath,
      strictHostKeyChecking: exported.sshTunnel.strictHostKeyChecking,
      timeoutSeconds: exported.sshTunnel.timeoutSeconds,
      keepAliveIntervalSeconds: exported.sshTunnel.keepAliveIntervalSeconds,
    }
  }

  // Reconstruct VPC config
  if (exported.vpcConfig) {
    connection.vpcConfig = {
      vpcId: exported.vpcConfig.vpcId,
      subnetId: exported.vpcConfig.subnetId,
      securityGroupIds: exported.vpcConfig.securityGroupIds,
      privateLinkService: exported.vpcConfig.privateLinkService,
      endpointServiceName: exported.vpcConfig.endpointServiceName,
    }
  }

  // Store password in secure storage if provided
  if (exported.password) {
    try {
      const secureStorage = getSecureStorage()
      await secureStorage.setCredentials(exported.id, {
        password: exported.password,
      })
    } catch {
      // Secure storage unavailable - password won't be stored
    }
  }

  const fullConnection = { ...connection, isConnected: false } as DatabaseConnection

  // Add to store (bypass the addConnection method to preserve ID)
  useConnectionStore.setState((state) => ({
    connections: [...state.connections, fullConnection],
  }))

  // Sync to SQLite for v3 persistence (fire-and-forget)
  syncToSQLite(fullConnection)
}

/**
 * Sync a connection to SQLite storage (v3 compatibility)
 * This is a fire-and-forget operation - errors are logged but don't fail the import
 */
async function syncToSQLite(connection: DatabaseConnection): Promise<void> {
  try {
    const App = await import('../../../bindings/github.com/jbeck018/howlerops/app')
    if (typeof App.SQLiteSaveConnection !== 'function') return

    await App.SQLiteSaveConnection(JSON.stringify({
      id: connection.id,
      name: connection.name,
      type: connection.type,
      host: connection.host || '',
      port: connection.port || 0,
      database: connection.database,
      username: connection.username || '',
      ssl_config: connection.sslMode ? { mode: connection.sslMode } : {},
      environments: connection.environments || [],
    }))
  } catch {
    // SQLite sync is best-effort - Wails bindings may not be available (web mode)
  }
}

/**
 * Overwrite an existing connection with imported data
 */
async function overwriteConnection(exported: ExportedConnection): Promise<void> {
  const store = useConnectionStore.getState()

  // Build updates
  const updates: Partial<DatabaseConnection> = {
    name: exported.name,
    type: exported.type,
    host: exported.host,
    port: exported.port,
    database: exported.database,
    username: exported.username,
    sslMode: exported.sslMode,
    environments: exported.environments,
    useTunnel: exported.useTunnel,
    useVpc: exported.useVpc,
    parameters: exported.parameters,
  }

  // Reconstruct SSH tunnel config
  if (exported.sshTunnel) {
    updates.sshTunnel = {
      host: exported.sshTunnel.host,
      port: exported.sshTunnel.port,
      user: exported.sshTunnel.user,
      authMethod: exported.sshTunnel.authMethod,
      privateKeyPath: exported.sshTunnel.privateKeyPath,
      knownHostsPath: exported.sshTunnel.knownHostsPath,
      strictHostKeyChecking: exported.sshTunnel.strictHostKeyChecking,
      timeoutSeconds: exported.sshTunnel.timeoutSeconds,
      keepAliveIntervalSeconds: exported.sshTunnel.keepAliveIntervalSeconds,
    }
  }

  // Reconstruct VPC config
  if (exported.vpcConfig) {
    updates.vpcConfig = {
      vpcId: exported.vpcConfig.vpcId,
      subnetId: exported.vpcConfig.subnetId,
      securityGroupIds: exported.vpcConfig.securityGroupIds,
      privateLinkService: exported.vpcConfig.privateLinkService,
      endpointServiceName: exported.vpcConfig.endpointServiceName,
    }
  }

  // Store password in secure storage if provided
  if (exported.password) {
    try {
      const secureStorage = getSecureStorage()
      await secureStorage.setCredentials(exported.id, {
        password: exported.password,
      })
    } catch {
      // Secure storage unavailable
    }
  }

  // Update via store (this will trigger SQLite sync via the store's updateConnection)
  await store.updateConnection(exported.id, updates)

  // Also explicitly sync to SQLite in case updateConnection doesn't have the sync
  const updatedConn = useConnectionStore.getState().connections.find(c => c.id === exported.id)
  if (updatedConn) {
    syncToSQLite(updatedConn)
  }
}
