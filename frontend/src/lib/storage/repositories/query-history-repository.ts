/**
 * Query History Repository
 *
 * Manages query execution history with:
 * - Full-text search capabilities
 * - Performance analytics
 * - Connection-based filtering
 * - Pagination support
 * - Privacy mode filtering
 *
 * @module lib/storage/repositories/query-history-repository
 */

import { getIndexedDBClient } from '../indexeddb-client'
import {
  STORE_NAMES,
  type QueryHistoryRecord,
  type CreateInput,
  type UpdateInput,
  type PaginatedResult,
  type PrivacyMode,
  NotFoundError,
} from '@/types/storage'
import { useTierStore } from '@/store/tier-store'

/**
 * Search options for query history
 */
export interface QueryHistorySearchOptions {
  /** User ID filter */
  userId?: string

  /** Connection ID filter */
  connectionId?: string

  /** Privacy mode filter */
  privacyMode?: PrivacyMode

  /** Text search in query_text */
  searchText?: string

  /** Only show queries with errors */
  errorsOnly?: boolean

  /** Minimum execution time (ms) */
  minExecutionTime?: number

  /** Maximum execution time (ms) */
  maxExecutionTime?: number

  /** Start date filter */
  startDate?: Date

  /** End date filter */
  endDate?: Date

  /** Only show unsynced records */
  unsyncedOnly?: boolean

  /** Maximum number of results */
  limit?: number

  /** Number of results to skip */
  offset?: number

  /** Sort direction */
  sortDirection?: 'asc' | 'desc'
}

/**
 * Query statistics
 */
export interface QueryStatistics {
  /** Total queries executed */
  totalQueries: number

  /** Total execution time (ms) */
  totalExecutionTime: number

  /** Average execution time (ms) */
  averageExecutionTime: number

  /** Slowest query time (ms) */
  slowestQueryTime: number

  /** Number of failed queries */
  failedQueries: number

  /** Success rate (0-1) */
  successRate: number

  /** Total rows returned */
  totalRows: number
}

/**
 * Repository for managing query history
 */
export class QueryHistoryRepository {
  private client = getIndexedDBClient()
  private storeName = STORE_NAMES.QUERY_HISTORY

  /**
   * Create a new query history record
   */
  async create(
    data: CreateInput<QueryHistoryRecord>
  ): Promise<QueryHistoryRecord> {
    // Check tier limit before adding to history
    const currentCount = await this.count()
    const tierStore = useTierStore.getState()
    const limitCheck = tierStore.checkLimit('queryHistory', currentCount + 1)

    if (!limitCheck.allowed) {
      console.warn('Query history limit reached:', limitCheck)

      // Clean up oldest entries if at limit (auto-prune for local tier)
      if (tierStore.currentTier === 'local' && limitCheck.limit !== null) {
        await this.pruneOldest(1)
        console.log('Auto-pruned oldest query from history')
      } else {
        // For paid tiers with limits (shouldn't happen normally), dispatch upgrade event
        window.dispatchEvent(
          new CustomEvent('showUpgradeDialog', {
            detail: {
              limitName: 'queryHistory',
              currentTier: tierStore.currentTier,
              usage: currentCount,
              limit: limitCheck.limit,
            },
          })
        )
      }
    }

    const now = new Date()
    const record: QueryHistoryRecord = {
      id: data.id || crypto.randomUUID(),
      user_id: data.user_id,
      query_text: this.sanitizeQuery(data.query_text),
      connection_id: data.connection_id!,
      execution_time_ms: data.execution_time_ms,
      row_count: data.row_count,
      error: data.error,
      privacy_mode: data.privacy_mode,
      executed_at: data.executed_at ?? now,
      synced: data.synced ?? false,
      sync_version: data.sync_version ?? 0,
    }

    await this.client.put(this.storeName, record)
    return record
  }

  /**
   * Get a query history record by ID
   */
  async get(id: string): Promise<QueryHistoryRecord | null> {
    return this.client.get<QueryHistoryRecord>(this.storeName, id)
  }

  /**
   * Get a query history record by ID or throw error
   */
  async getOrFail(id: string): Promise<QueryHistoryRecord> {
    const record = await this.get(id)
    if (!record) {
      throw new NotFoundError(`Query history record ${id} not found`)
    }
    return record
  }

  /**
   * Update a query history record
   */
  async update(
    id: string,
    updates: UpdateInput<QueryHistoryRecord>
  ): Promise<QueryHistoryRecord> {
    const existing = await this.getOrFail(id)

    const updated: QueryHistoryRecord = {
      ...existing,
      ...updates,
      id, // Ensure ID doesn't change
    }

    await this.client.put(this.storeName, updated)
    return updated
  }

  /**
   * Delete a query history record
   */
  async delete(id: string): Promise<void> {
    await this.client.delete(this.storeName, id)
  }

  /**
   * Search query history with filters
   */
  async search(
    options: QueryHistorySearchOptions = {}
  ): Promise<PaginatedResult<QueryHistoryRecord>> {
    const {
      userId,
      connectionId,
      privacyMode,
      searchText,
      errorsOnly,
      minExecutionTime,
      maxExecutionTime,
      startDate,
      endDate,
      unsyncedOnly,
      limit = 50,
      offset = 0,
      sortDirection = 'desc',
    } = options

    // Determine best index to use
    let indexName: string | undefined
    let keyRange: IDBKeyRange | undefined

    if (userId && startDate) {
      // Use compound index for user + time
      indexName = 'user_executed'
      const lowerBound = [userId, startDate]
      const upperBound = endDate ? [userId, endDate] : [userId, new Date()]
      keyRange = IDBKeyRange.bound(lowerBound, upperBound)
    } else if (connectionId && startDate) {
      // Use compound index for connection + time
      indexName = 'connection_executed'
      const lowerBound = [connectionId, startDate]
      const upperBound = endDate ? [connectionId, endDate] : [connectionId, new Date()]
      keyRange = IDBKeyRange.bound(lowerBound, upperBound)
    } else if (userId) {
      indexName = 'user_id'
      keyRange = IDBKeyRange.only(userId)
    } else if (connectionId) {
      indexName = 'connection_id'
      keyRange = IDBKeyRange.only(connectionId)
    } else if (startDate) {
      indexName = 'executed_at'
      keyRange = endDate
        ? IDBKeyRange.bound(startDate, endDate)
        : IDBKeyRange.lowerBound(startDate)
    } else {
      // Use executed_at index for sorting
      indexName = 'executed_at'
    }

    // Get all matching records (we'll filter in memory)
    const allRecords = await this.client.getAll<QueryHistoryRecord>(
      this.storeName,
      {
        index: indexName,
        range: keyRange,
        direction: sortDirection === 'desc' ? 'prev' : 'next',
      }
    )

    // Apply in-memory filters
    let filtered = allRecords

    if (privacyMode) {
      filtered = filtered.filter((r) => r.privacy_mode === privacyMode)
    }

    if (errorsOnly) {
      filtered = filtered.filter((r) => r.error !== undefined)
    }

    if (minExecutionTime !== undefined) {
      filtered = filtered.filter((r) => r.execution_time_ms >= minExecutionTime)
    }

    if (maxExecutionTime !== undefined) {
      filtered = filtered.filter((r) => r.execution_time_ms <= maxExecutionTime)
    }

    if (unsyncedOnly) {
      filtered = filtered.filter((r) => !r.synced)
    }

    if (searchText) {
      const searchLower = searchText.toLowerCase()
      filtered = filtered.filter((r) =>
        r.query_text.toLowerCase().includes(searchLower)
      )
    }

    // Apply pagination
    const total = filtered.length
    const items = filtered.slice(offset, offset + limit)
    const hasMore = offset + limit < total

    return {
      items,
      total,
      hasMore,
    }
  }

  /**
   * Get recent queries for a user
   */
  async getRecent(
    userId: string,
    limit = 20
  ): Promise<QueryHistoryRecord[]> {
    const result = await this.search({
      userId,
      limit,
      sortDirection: 'desc',
    })
    return result.items
  }

  /**
   * Get recent queries for a connection
   */
  async getRecentForConnection(
    connectionId: string,
    limit = 20
  ): Promise<QueryHistoryRecord[]> {
    const result = await this.search({
      connectionId,
      limit,
      sortDirection: 'desc',
    })
    return result.items
  }

  /**
   * Get failed queries
   */
  async getFailedQueries(
    userId?: string,
    limit = 50
  ): Promise<QueryHistoryRecord[]> {
    const result = await this.search({
      userId,
      errorsOnly: true,
      limit,
      sortDirection: 'desc',
    })
    return result.items
  }

  /**
   * Get slow queries (above threshold)
   */
  async getSlowQueries(
    thresholdMs: number,
    userId?: string,
    limit = 50
  ): Promise<QueryHistoryRecord[]> {
    const result = await this.search({
      userId,
      minExecutionTime: thresholdMs,
      limit,
      sortDirection: 'desc',
    })
    return result.items
  }

  /**
   * Get query statistics for a user
   */
  async getStatistics(
    userId: string,
    options?: {
      connectionId?: string
      startDate?: Date
      endDate?: Date
    }
  ): Promise<QueryStatistics> {
    const records = await this.search({
      userId,
      connectionId: options?.connectionId,
      startDate: options?.startDate,
      endDate: options?.endDate,
      limit: 10000, // Get all for statistics
    })

    const queries = records.items
    const totalQueries = queries.length

    if (totalQueries === 0) {
      return {
        totalQueries: 0,
        totalExecutionTime: 0,
        averageExecutionTime: 0,
        slowestQueryTime: 0,
        failedQueries: 0,
        successRate: 0,
        totalRows: 0,
      }
    }

    const totalExecutionTime = queries.reduce(
      (sum, q) => sum + q.execution_time_ms,
      0
    )
    const failedQueries = queries.filter((q) => q.error !== undefined).length
    const totalRows = queries.reduce((sum, q) => sum + q.row_count, 0)
    const slowestQueryTime = Math.max(
      ...queries.map((q) => q.execution_time_ms)
    )

    return {
      totalQueries,
      totalExecutionTime,
      averageExecutionTime: totalExecutionTime / totalQueries,
      slowestQueryTime,
      failedQueries,
      successRate: (totalQueries - failedQueries) / totalQueries,
      totalRows,
    }
  }

  /**
   * Get unsynced records for server sync
   */
  async getUnsynced(limit = 100): Promise<QueryHistoryRecord[]> {
    return this.client.getAll<QueryHistoryRecord>(this.storeName, {
      index: 'synced',
      range: IDBKeyRange.only(false),
      limit,
    })
  }

  /**
   * Mark records as synced
   */
  async markSynced(
    ids: string[],
    syncVersion: number
  ): Promise<void> {
    await Promise.all(
      ids.map((id) =>
        this.update(id, {
          synced: true,
          sync_version: syncVersion,
        })
      )
    )
  }

  /**
   * Delete old query history records
   */
  async deleteOlderThan(date: Date): Promise<number> {
    const range = IDBKeyRange.upperBound(date)
    return this.client.deleteRange(this.storeName, range)
  }

  /**
   * Clear all query history for a user
   */
  async clearUserHistory(userId: string): Promise<number> {
    const records = await this.client.getAll<QueryHistoryRecord>(
      this.storeName,
      {
        index: 'user_id',
        range: IDBKeyRange.only(userId),
      }
    )

    await Promise.all(records.map((r) => this.delete(r.id)))
    return records.length
  }

  /**
   * Get total count of query history records
   */
  async count(options?: {
    userId?: string
    connectionId?: string
  }): Promise<number> {
    if (options?.userId) {
      return this.client.count(this.storeName, {
        index: 'user_id',
        range: IDBKeyRange.only(options.userId),
      })
    }

    if (options?.connectionId) {
      return this.client.count(this.storeName, {
        index: 'connection_id',
        range: IDBKeyRange.only(options.connectionId),
      })
    }

    return this.client.count(this.storeName)
  }

  /**
   * Prune oldest query history entries
   * Used for automatic cleanup when tier limits are reached
   *
   * @param count - Number of oldest entries to remove
   */
  async pruneOldest(count: number): Promise<void> {
    // Get oldest entries
    const oldest = await this.search({
      limit: count,
      sortDirection: 'asc', // Oldest first
    })

    // Delete them
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const list: QueryHistoryRecord[] = (oldest as any).data ?? []
    await Promise.all(list.map((record) => this.delete(record.id)))
  }

  /**
   * Sanitize query text (remove sensitive data)
   */
  private sanitizeQuery(query: string): string {
    // Remove potential passwords from CREATE USER, ALTER USER, etc.
    let sanitized = query.replace(
      /(password|pwd)\s*=\s*['"][^'"]*['"]/gi,
      "$1='***'"
    )

    // Remove connection strings with passwords
    sanitized = sanitized.replace(
      /(password|pwd)=([^;,\s]+)/gi,
      '$1=***'
    )

    return sanitized
  }
}

/**
 * Singleton instance
 */
let repositoryInstance: QueryHistoryRepository | null = null

/**
 * Get singleton instance of QueryHistoryRepository
 */
export function getQueryHistoryRepository(): QueryHistoryRepository {
  if (!repositoryInstance) {
    repositoryInstance = new QueryHistoryRepository()
  }
  return repositoryInstance
}
