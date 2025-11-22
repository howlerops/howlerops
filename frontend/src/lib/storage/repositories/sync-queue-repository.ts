/**
 * Sync Queue Repository
 *
 * Manages offline change queue for server synchronization
 *
 * Features:
 * - Queue offline changes
 * - Retry failed syncs
 * - Entity-based filtering
 * - Priority management
 * - Conflict resolution
 *
 * @module lib/storage/repositories/sync-queue-repository
 */

import {
  type CreateInput,
  type EntityType,
  NotFoundError,
  STORE_NAMES,
  type SyncOperation,
  type SyncQueueRecord,
} from '@/types/storage'

import { getIndexedDBClient } from '../indexeddb-client'

/**
 * Sync queue search options
 */
export interface SyncQueueSearchOptions {
  /** Entity type filter */
  entityType?: EntityType

  /** Entity ID filter */
  entityId?: string

  /** Operation filter */
  operation?: SyncOperation

  /** Only failed items (with retry count > 0) */
  failedOnly?: boolean

  /** Maximum retry count filter */
  maxRetries?: number

  /** Sort by timestamp */
  sortByTimestamp?: boolean

  /** Maximum number of results */
  limit?: number
}

/**
 * Sync statistics
 */
export interface SyncStatistics {
  /** Total queued items */
  totalQueued: number

  /** Items by operation type */
  byOperation: Record<SyncOperation, number>

  /** Items by entity type */
  byEntity: Record<EntityType, number>

  /** Failed items (retry count > 0) */
  failedItems: number

  /** Average retry count */
  averageRetries: number
}

/**
 * Repository for managing sync queue
 */
export class SyncQueueRepository {
  private client = getIndexedDBClient()
  private storeName = STORE_NAMES.SYNC_QUEUE

  /**
   * Create a new sync queue record
   */
  async create(
    data: CreateInput<SyncQueueRecord>
  ): Promise<SyncQueueRecord> {
    const record: SyncQueueRecord = {
      id: data.id || crypto.randomUUID(),
      entity_type: data.entity_type,
      entity_id: data.entity_id,
      operation: data.operation,
      payload: data.payload,
      timestamp: data.timestamp ?? new Date(),
      retry_count: data.retry_count ?? 0,
      last_error: data.last_error,
    }

    await this.client.put(this.storeName, record)
    return record
  }

  /**
   * Get a sync queue record by ID
   */
  async get(id: string): Promise<SyncQueueRecord | null> {
    return this.client.get<SyncQueueRecord>(this.storeName, id)
  }

  /**
   * Get a sync queue record by ID or throw error
   */
  async getOrFail(id: string): Promise<SyncQueueRecord> {
    const record = await this.get(id)
    if (!record) {
      throw new NotFoundError(`Sync queue record ${id} not found`)
    }
    return record
  }

  /**
   * Queue a change for sync
   */
  async queueChange(
    entityType: EntityType,
    entityId: string,
    operation: SyncOperation,
    payload: Record<string, unknown>
  ): Promise<SyncQueueRecord> {
    // Check if there's already a queued operation for this entity
    const existing = await this.findByEntity(entityType, entityId)

    if (existing.length > 0) {
      // If there's a create followed by delete, remove both
      if (
        existing.some((e) => e.operation === 'create') &&
        operation === 'delete'
      ) {
        await Promise.all(existing.map((e) => this.delete(e.id)))
        return this.create({
          entity_type: entityType,
          entity_id: entityId,
          operation,
          payload,
        })
      }

      // If there's an update or create, update the payload
      if (operation === 'update') {
        const lastOperation = existing[existing.length - 1]
        const updated: SyncQueueRecord = {
          ...lastOperation,
          payload: { ...lastOperation.payload, ...payload },
          timestamp: new Date(),
        }
        await this.client.put(this.storeName, updated)
        return updated
      }
    }

    // Queue new change
    return this.create({
      entity_type: entityType,
      entity_id: entityId,
      operation,
      payload,
    })
  }

  /**
   * Find sync queue records by entity
   */
  async findByEntity(
    entityType: EntityType,
    entityId: string
  ): Promise<SyncQueueRecord[]> {
    return this.client.getAll<SyncQueueRecord>(this.storeName, {
      index: 'entity_lookup',
      range: IDBKeyRange.only([entityType, entityId]),
    })
  }

  /**
   * Search sync queue with filters
   */
  async search(
    options: SyncQueueSearchOptions = {}
  ): Promise<SyncQueueRecord[]> {
    const {
      entityType,
      entityId,
      operation,
      failedOnly,
      maxRetries,
      sortByTimestamp,
      limit,
    } = options

    // Determine best index to use
    let indexName: string | undefined
    let keyRange: IDBKeyRange | undefined

    if (entityType && entityId) {
      indexName = 'entity_lookup'
      keyRange = IDBKeyRange.only([entityType, entityId])
    } else if (entityType) {
      indexName = 'entity_type'
      keyRange = IDBKeyRange.only(entityType)
    } else if (operation) {
      indexName = 'operation'
      keyRange = IDBKeyRange.only(operation)
    } else if (sortByTimestamp) {
      indexName = 'timestamp'
    }

    const records = await this.client.getAll<SyncQueueRecord>(
      this.storeName,
      {
        index: indexName,
        range: keyRange,
        direction: sortByTimestamp ? 'next' : undefined,
        limit,
      }
    )

    // Apply in-memory filters
    let filtered = records

    if (operation && !indexName?.includes('operation')) {
      filtered = filtered.filter((r) => r.operation === operation)
    }

    if (failedOnly) {
      filtered = filtered.filter((r) => r.retry_count > 0)
    }

    if (maxRetries !== undefined) {
      filtered = filtered.filter((r) => r.retry_count <= maxRetries)
    }

    return filtered
  }

  /**
   * Get pending sync items (oldest first)
   */
  async getPending(limit = 50): Promise<SyncQueueRecord[]> {
    return this.search({
      sortByTimestamp: true,
      limit,
    })
  }

  /**
   * Get failed sync items
   */
  async getFailed(limit = 50): Promise<SyncQueueRecord[]> {
    return this.search({
      failedOnly: true,
      sortByTimestamp: true,
      limit,
    })
  }

  /**
   * Mark sync as failed and increment retry count
   */
  async markFailed(id: string, error: string): Promise<SyncQueueRecord> {
    const record = await this.getOrFail(id)

    const updated: SyncQueueRecord = {
      ...record,
      retry_count: record.retry_count + 1,
      last_error: error,
    }

    await this.client.put(this.storeName, updated)
    return updated
  }

  /**
   * Mark sync as successful and remove from queue
   */
  async markSuccessful(id: string): Promise<void> {
    await this.delete(id)
  }

  /**
   * Delete a sync queue record
   */
  async delete(id: string): Promise<void> {
    await this.client.delete(this.storeName, id)
  }

  /**
   * Delete all sync queue records for an entity
   */
  async deleteByEntity(
    entityType: EntityType,
    entityId: string
  ): Promise<number> {
    const records = await this.findByEntity(entityType, entityId)
    await Promise.all(records.map((r) => this.delete(r.id)))
    return records.length
  }

  /**
   * Clear items that have exceeded max retries
   */
  async clearExceededRetries(maxRetries: number): Promise<number> {
    const allRecords = await this.client.getAll<SyncQueueRecord>(
      this.storeName
    )
    const exceeded = allRecords.filter((r) => r.retry_count > maxRetries)

    await Promise.all(exceeded.map((r) => this.delete(r.id)))
    return exceeded.length
  }

  /**
   * Get sync statistics
   */
  async getStatistics(): Promise<SyncStatistics> {
    const allRecords = await this.client.getAll<SyncQueueRecord>(
      this.storeName
    )

    const totalQueued = allRecords.length

    const byOperation: Record<SyncOperation, number> = {
      create: 0,
      update: 0,
      delete: 0,
    }

    const byEntity: Record<EntityType, number> = {
      connection: 0,
      query: 0,
      preference: 0,
      ai_session: 0,
      report: 0,
    }

    let failedItems = 0
    let totalRetries = 0

    allRecords.forEach((record) => {
      byOperation[record.operation]++
      byEntity[record.entity_type]++

      if (record.retry_count > 0) {
        failedItems++
      }
      totalRetries += record.retry_count
    })

    return {
      totalQueued,
      byOperation,
      byEntity,
      failedItems,
      averageRetries: totalQueued > 0 ? totalRetries / totalQueued : 0,
    }
  }

  /**
   * Process sync queue (batch operation)
   *
   * Returns successfully synced IDs and failed records
   */
  async processBatch(
    syncFn: (record: SyncQueueRecord) => Promise<void>,
    batchSize = 10
  ): Promise<{
    successful: string[]
    failed: Array<{ id: string; error: string }>
  }> {
    const pending = await this.getPending(batchSize)
    const successful: string[] = []
    const failed: Array<{ id: string; error: string }> = []

    for (const record of pending) {
      try {
        await syncFn(record)
        await this.markSuccessful(record.id)
        successful.push(record.id)
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : 'Unknown error'
        await this.markFailed(record.id, errorMessage)
        failed.push({ id: record.id, error: errorMessage })
      }
    }

    return { successful, failed }
  }

  /**
   * Get total count of queued items
   */
  async count(options?: { entityType?: EntityType }): Promise<number> {
    if (options?.entityType) {
      return this.client.count(this.storeName, {
        index: 'entity_type',
        range: IDBKeyRange.only(options.entityType),
      })
    }

    return this.client.count(this.storeName)
  }

  /**
   * Clear all sync queue records
   */
  async clearAll(): Promise<void> {
    await this.client.clear(this.storeName)
  }

  /**
   * Retry all failed items
   */
  async retryFailed(): Promise<SyncQueueRecord[]> {
    const failed = await this.getFailed()

    const updated = await Promise.all(
      failed.map(async (record) => {
        const retried: SyncQueueRecord = {
          ...record,
          retry_count: 0,
          last_error: undefined,
          timestamp: new Date(), // Move to front of queue
        }
        await this.client.put(this.storeName, retried)
        return retried
      })
    )

    return updated
  }
}

/**
 * Singleton instance
 */
let repositoryInstance: SyncQueueRepository | null = null

/**
 * Get singleton instance of SyncQueueRepository
 */
export function getSyncQueueRepository(): SyncQueueRepository {
  if (!repositoryInstance) {
    repositoryInstance = new SyncQueueRepository()
  }
  return repositoryInstance
}
