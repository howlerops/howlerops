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
 *
 * @module lib/storage/repositories/connection-repository
 */

import {
  type ConnectionRecord,
  type CreateInput,
  type DatabaseType,
  NotFoundError,
  STORE_NAMES,
  type UpdateInput,
} from '@/types/storage'

import { getIndexedDBClient } from '../indexeddb-client'

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
 */
export class ConnectionRepository {
  private client = getIndexedDBClient()
  private storeName = STORE_NAMES.CONNECTIONS

  /**
   * Create a new connection record
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

    await this.client.put(this.storeName, record)
    return record
  }

  /**
   * Get a connection by ID
   */
  async get(connectionId: string): Promise<ConnectionRecord | null> {
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

    await this.client.put(this.storeName, updated)
    return updated
  }

  /**
   * Delete a connection
   */
  async delete(connectionId: string): Promise<void> {
    await this.client.delete(this.storeName, connectionId)
  }

  /**
   * Get all connections for a user
   */
  async getAllForUser(userId: string): Promise<ConnectionRecord[]> {
    return this.client.getAll<ConnectionRecord>(this.storeName, {
      index: 'user_id',
      range: IDBKeyRange.only(userId),
    })
  }

  /**
   * Search connections with filters
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
