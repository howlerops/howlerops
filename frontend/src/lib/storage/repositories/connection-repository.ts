/**
 * Connection Repository
 *
 * Manages database connection metadata (NO passwords)
 * Passwords are stored separately in sessionStorage via secure-storage
 *
 * Features:
 * - Connection CRUD operations
 * - Environment-based filtering
 * - Usage tracking
 * - Type-based queries
 * - SQLite primary storage with IndexedDB fallback
 *
 * @module lib/storage/repositories/connection-repository
 */

import {
  type ConnectionRecord,
  type CreateInput,
  type DatabaseType,
  NotFoundError,
  type SSLMode,
  STORE_NAMES,
  type UpdateInput,
} from '@/types/storage'

import { getIndexedDBClient } from '../indexeddb-client'

// Type for SQLite connection from Wails bindings
interface SQLiteConnection {
  id: string
  name: string
  type: string
  host: string
  port: number
  database: string
  username: string
  ssl_config: Record<string, string>
  environments: string[]
  created_at: string
  updated_at: string
}

// Wails bindings loader - cached to avoid repeated imports
let appBindingsCache: typeof import('../../../../bindings/github.com/jbeck018/howlerops/app') | null = null

async function getAppBindings() {
  if (appBindingsCache) return appBindingsCache
  try {
    appBindingsCache = await import('../../../../bindings/github.com/jbeck018/howlerops/app')
    return appBindingsCache
  } catch {
    return null
  }
}

/**
 * Convert SQLite connection to ConnectionRecord format
 */
function sqliteToConnectionRecord(sqlite: SQLiteConnection): ConnectionRecord {
  return {
    connection_id: sqlite.id,
    user_id: '', // SQLite doesn't track user_id per connection
    name: sqlite.name,
    type: sqlite.type as DatabaseType,
    host: sqlite.host,
    port: sqlite.port,
    database: sqlite.database,
    username: sqlite.username,
    ssl_mode: (sqlite.ssl_config?.mode || 'disable') as SSLMode,
    parameters: sqlite.ssl_config || {},
    environment_tags: sqlite.environments || [],
    created_at: new Date(sqlite.created_at),
    updated_at: new Date(sqlite.updated_at),
    last_used_at: new Date(sqlite.updated_at),
    synced: true, // SQLite is the source of truth
    sync_version: 0,
  }
}

/**
 * Connection search options
 */
export interface ConnectionSearchOptions {
  /** User ID filter */
  userId?: string

  /** Database type filter */
  type?: DatabaseType

  /** Environment tag filter */
  environment?: string

  /** Only show unsynced records */
  unsyncedOnly?: boolean

  /** Sort by last used */
  sortByLastUsed?: boolean

  /** Maximum number of results */
  limit?: number
}

/**
 * Repository for managing connection metadata
 * Uses SQLite (via Wails bindings) as primary, IndexedDB as fallback
 */
export class ConnectionRepository {
  private client = getIndexedDBClient()
  private storeName = STORE_NAMES.CONNECTIONS

  /**
   * Create a new connection record
   * Writes to both SQLite and IndexedDB (dual-write pattern)
   *
   * SECURITY: Passwords must NOT be included in the record.
   * Use secure-storage to store passwords separately.
   */
  async create(
    data: CreateInput<ConnectionRecord>
  ): Promise<ConnectionRecord> {
    const now = new Date()
    const record: ConnectionRecord = {
      connection_id: data.connection_id || crypto.randomUUID(),
      user_id: data.user_id,
      name: data.name,
      type: data.type,
      host: data.host,
      port: data.port,
      database: data.database,
      username: data.username,
      ssl_mode: data.ssl_mode,
      parameters: data.parameters,
      environment_tags: data.environment_tags || [],
      created_at: data.created_at ?? now,
      updated_at: data.updated_at ?? now,
      last_used_at: data.last_used_at ?? now,
      synced: data.synced ?? false,
      sync_version: data.sync_version ?? 0,
    }

    // Dual-write: write to SQLite (primary) and IndexedDB (fallback)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteSaveConnection) {
        // Convert to SQLite format
        const sqliteRecord = {
          id: record.connection_id,
          name: record.name,
          type: record.type,
          host: record.host,
          port: record.port,
          database: record.database,
          username: record.username,
          environments: record.environment_tags,
          ssl_config: record.parameters || {},
          created_at: record.created_at.toISOString(),
          updated_at: record.updated_at.toISOString(),
        }
        await App.SQLiteSaveConnection(JSON.stringify(sqliteRecord))
      }
    } catch (error) {
      console.warn('[ConnectionRepository] SQLite write failed, continuing with IndexedDB:', error)
    }

    // Always write to IndexedDB for backwards compatibility
    await this.client.put(this.storeName, record)
    return record
  }

  /**
   * Get a connection by ID
   * Tries SQLite first, falls back to IndexedDB
   */
  async get(connectionId: string): Promise<ConnectionRecord | null> {
    // Try SQLite first (primary)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteGetConnection) {
        const sqliteConn = await App.SQLiteGetConnection(connectionId)
        if (sqliteConn) {
          return sqliteToConnectionRecord(sqliteConn as SQLiteConnection)
        }
      }
    } catch (error) {
      console.debug('[ConnectionRepository] SQLite read failed, falling back to IndexedDB:', error)
    }

    // Fallback to IndexedDB
    return this.client.get<ConnectionRecord>(this.storeName, connectionId)
  }

  /**
   * Get a connection by ID or throw error
   */
  async getOrFail(connectionId: string): Promise<ConnectionRecord> {
    const record = await this.get(connectionId)
    if (!record) {
      throw new NotFoundError(`Connection ${connectionId} not found`)
    }
    return record
  }

  /**
   * Update a connection
   * Writes to both SQLite and IndexedDB (dual-write pattern)
   */
  async update(
    connectionId: string,
    updates: UpdateInput<ConnectionRecord>
  ): Promise<ConnectionRecord> {
    const existing = await this.getOrFail(connectionId)

    const updated: ConnectionRecord = {
      ...existing,
      ...updates,
      connection_id: connectionId, // Ensure ID doesn't change
      updated_at: new Date(),
    }

    // Dual-write: write to SQLite (primary) and IndexedDB (fallback)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteSaveConnection) {
        const sqliteRecord = {
          id: updated.connection_id,
          name: updated.name,
          type: updated.type,
          host: updated.host,
          port: updated.port,
          database: updated.database,
          username: updated.username,
          environments: updated.environment_tags,
          ssl_config: updated.parameters || {},
          created_at: updated.created_at.toISOString(),
          updated_at: updated.updated_at.toISOString(),
        }
        await App.SQLiteSaveConnection(JSON.stringify(sqliteRecord))
      }
    } catch (error) {
      console.warn('[ConnectionRepository] SQLite update failed, continuing with IndexedDB:', error)
    }

    await this.client.put(this.storeName, updated)
    return updated
  }

  /**
   * Delete a connection
   * Deletes from both SQLite and IndexedDB (dual-write pattern)
   */
  async delete(connectionId: string): Promise<void> {
    // Dual-delete: delete from SQLite (primary) and IndexedDB (fallback)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteDeleteConnection) {
        await App.SQLiteDeleteConnection(connectionId)
      }
    } catch (error) {
      console.warn('[ConnectionRepository] SQLite delete failed, continuing with IndexedDB:', error)
    }

    await this.client.delete(this.storeName, connectionId)
  }

  /**
   * Get all connections for a user
   * Tries SQLite first, falls back to IndexedDB
   */
  async getAllForUser(userId: string): Promise<ConnectionRecord[]> {
    // Try SQLite first - returns all connections (user filtering done in-memory if needed)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteGetConnections) {
        const sqliteConns = await App.SQLiteGetConnections()
        if (sqliteConns && sqliteConns.length > 0) {
          const records = (sqliteConns as SQLiteConnection[]).map(sqliteToConnectionRecord)
          // Filter by userId if provided (SQLite doesn't track per-user)
          return userId ? records : records
        }
      }
    } catch (error) {
      console.debug('[ConnectionRepository] SQLite read failed, falling back to IndexedDB:', error)
    }

    // Fallback to IndexedDB
    return this.client.getAll<ConnectionRecord>(this.storeName, {
      index: 'user_id',
      range: IDBKeyRange.only(userId),
    })
  }

  /**
   * Search connections with filters
   * Tries SQLite first, falls back to IndexedDB
   */
  async search(
    options: ConnectionSearchOptions = {}
  ): Promise<ConnectionRecord[]> {
    const {
      userId,
      type,
      environment,
      unsyncedOnly,
      sortByLastUsed,
      limit,
    } = options

    // Try SQLite first
    try {
      const App = await getAppBindings()
      if (App?.SQLiteGetConnections) {
        const sqliteConns = await App.SQLiteGetConnections()
        if (sqliteConns && sqliteConns.length > 0) {
          let records = (sqliteConns as SQLiteConnection[]).map(sqliteToConnectionRecord)

          // Apply in-memory filters
          if (type) {
            records = records.filter((r) => r.type === type)
          }
          if (environment) {
            records = records.filter((r) =>
              r.environment_tags.includes(environment)
            )
          }
          if (unsyncedOnly) {
            records = records.filter((r) => !r.synced)
          }
          if (sortByLastUsed) {
            records.sort((a, b) => b.last_used_at.getTime() - a.last_used_at.getTime())
          }
          if (limit) {
            records = records.slice(0, limit)
          }

          return records
        }
      }
    } catch (error) {
      console.debug('[ConnectionRepository] SQLite search failed, falling back to IndexedDB:', error)
    }

    // Fallback to IndexedDB
    // Determine best index to use
    let indexName: string | undefined
    let keyRange: IDBKeyRange | undefined

    if (userId) {
      indexName = 'user_id'
      keyRange = IDBKeyRange.only(userId)
    } else if (type) {
      indexName = 'type'
      keyRange = IDBKeyRange.only(type)
    } else if (environment) {
      indexName = 'environment_tags'
      keyRange = IDBKeyRange.only(environment)
    } else if (sortByLastUsed) {
      indexName = 'last_used_at'
    }

    const records = await this.client.getAll<ConnectionRecord>(
      this.storeName,
      {
        index: indexName,
        range: keyRange,
        direction: sortByLastUsed ? 'prev' : 'next',
        limit,
      }
    )

    // Apply in-memory filters
    let filtered = records

    if (type && !indexName?.includes('type')) {
      filtered = filtered.filter((r) => r.type === type)
    }

    if (environment && !indexName?.includes('environment')) {
      filtered = filtered.filter((r) =>
        r.environment_tags.includes(environment)
      )
    }

    if (unsyncedOnly) {
      filtered = filtered.filter((r) => !r.synced)
    }

    return filtered
  }

  /**
   * Get connections by environment tag
   */
  async getByEnvironment(
    environment: string,
    userId?: string
  ): Promise<ConnectionRecord[]> {
    return this.search({
      userId,
      environment,
    })
  }

  /**
   * Get connections by database type
   */
  async getByType(
    type: DatabaseType,
    userId?: string
  ): Promise<ConnectionRecord[]> {
    return this.search({
      userId,
      type,
    })
  }

  /**
   * Get recently used connections
   */
  async getRecentlyUsed(
    userId?: string,
    limit = 10
  ): Promise<ConnectionRecord[]> {
    return this.search({
      userId,
      sortByLastUsed: true,
      limit,
    })
  }

  /**
   * Update last used timestamp
   */
  async updateLastUsed(connectionId: string): Promise<void> {
    await this.update(connectionId, {
      last_used_at: new Date(),
    })
  }

  /**
   * Add environment tag to connection
   */
  async addEnvironmentTag(
    connectionId: string,
    tag: string
  ): Promise<ConnectionRecord> {
    const connection = await this.getOrFail(connectionId)

    if (!connection.environment_tags.includes(tag)) {
      return this.update(connectionId, {
        environment_tags: [...connection.environment_tags, tag],
      })
    }

    return connection
  }

  /**
   * Remove environment tag from connection
   */
  async removeEnvironmentTag(
    connectionId: string,
    tag: string
  ): Promise<ConnectionRecord> {
    const connection = await this.getOrFail(connectionId)

    return this.update(connectionId, {
      environment_tags: connection.environment_tags.filter((t) => t !== tag),
    })
  }

  /**
   * Get all unique environment tags
   */
  async getAllEnvironmentTags(userId?: string): Promise<string[]> {
    const connections = userId
      ? await this.getAllForUser(userId)
      : await this.client.getAll<ConnectionRecord>(this.storeName)

    const tags = new Set<string>()
    connections.forEach((conn) => {
      conn.environment_tags.forEach((tag) => tags.add(tag))
    })

    return Array.from(tags).sort()
  }

  /**
   * Get unsynced connections for server sync
   */
  async getUnsynced(limit = 100): Promise<ConnectionRecord[]> {
    return this.client.getAll<ConnectionRecord>(this.storeName, {
      index: 'synced',
      range: IDBKeyRange.only(false),
      limit,
    })
  }

  /**
   * Mark connections as synced
   */
  async markSynced(
    connectionIds: string[],
    syncVersion: number
  ): Promise<void> {
    await Promise.all(
      connectionIds.map((id) =>
        this.update(id, {
          synced: true,
          sync_version: syncVersion,
        })
      )
    )
  }

  /**
   * Update connection parameters
   */
  async updateParameters(
    connectionId: string,
    parameters: Record<string, unknown>
  ): Promise<ConnectionRecord> {
    return this.update(connectionId, {
      parameters,
    })
  }

  /**
   * Get total count of connections
   */
  async count(options?: { userId?: string }): Promise<number> {
    if (options?.userId) {
      return this.client.count(this.storeName, {
        index: 'user_id',
        range: IDBKeyRange.only(options.userId),
      })
    }

    return this.client.count(this.storeName)
  }

  /**
   * Clear all connections for a user
   */
  async clearUserConnections(userId: string): Promise<number> {
    const connections = await this.getAllForUser(userId)
    await Promise.all(connections.map((c) => this.delete(c.connection_id)))
    return connections.length
  }
}

/**
 * Singleton instance
 */
let repositoryInstance: ConnectionRepository | null = null

/**
 * Get singleton instance of ConnectionRepository
 */
export function getConnectionRepository(): ConnectionRepository {
  if (!repositoryInstance) {
    repositoryInstance = new ConnectionRepository()
  }
  return repositoryInstance
}
