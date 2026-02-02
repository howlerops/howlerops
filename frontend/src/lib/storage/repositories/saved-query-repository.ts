/**
 * Saved Query Repository
 *
 * Manages user's saved query library with:
 * - Full-text search capabilities
 * - Folder and tag organization
 * - Tier-aware limit enforcement
 * - Auto-pruning for local tier
 * - Favorites protection
 * - SQLite primary storage with IndexedDB fallback
 *
 * @module lib/storage/repositories/saved-query-repository
 */

import { useTierStore } from '@/store/tier-store'
import {
  type CreateInput,
  NotFoundError,
  type PaginatedResult,
  type SavedQueryRecord,
  STORE_NAMES,
  type UpdateInput,
} from '@/types/storage'

import { getIndexedDBClient } from '../indexeddb-client'

// Type for SQLite saved query from Wails bindings
interface SQLiteSavedQuery {
  id: string
  title: string
  description: string
  query: string
  connection_id: string
  folder: string
  tags: string[]
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
 * Convert SQLite saved query to SavedQueryRecord format
 */
function sqliteToSavedQueryRecord(sqlite: SQLiteSavedQuery, userId: string = ''): SavedQueryRecord {
  return {
    id: sqlite.id,
    user_id: userId,
    title: sqlite.title,
    description: sqlite.description,
    query_text: sqlite.query,
    tags: sqlite.tags || [],
    folder: sqlite.folder,
    is_favorite: false, // SQLite doesn't track favorites yet
    created_at: new Date(sqlite.created_at),
    updated_at: new Date(sqlite.updated_at),
    synced: true, // SQLite is the source of truth
    sync_version: 0,
  }
}

/**
 * Search options for saved queries
 */
export interface SavedQuerySearchOptions {
  /** User ID filter */
  userId?: string

  /** Text search in title, description, and query_text */
  searchText?: string

  /** Folder filter */
  folder?: string

  /** Tag filter (matches any of the provided tags) */
  tags?: string[]

  /** Only show favorites */
  favoritesOnly?: boolean

  /** Only show unsynced records */
  unsyncedOnly?: boolean

  /** Start date filter */
  startDate?: Date

  /** End date filter */
  endDate?: Date

  /** Sort field */
  sortBy?: 'created_at' | 'updated_at' | 'title'

  /** Sort direction */
  sortDirection?: 'asc' | 'desc'

  /** Maximum number of results */
  limit?: number

  /** Number of results to skip */
  offset?: number
}

/**
 * Repository for managing saved queries
 * Uses SQLite (via Wails bindings) as primary, IndexedDB as fallback
 */
export class SavedQueryRepository {
  private client = getIndexedDBClient()
  private storeName = STORE_NAMES.SAVED_QUERIES

  /**
   * Create a new saved query
   *
   * Uses a transaction to prevent race conditions between count check and insert
   */
  async create(
    data: CreateInput<SavedQueryRecord>
  ): Promise<SavedQueryRecord> {
    const tierStore = useTierStore.getState()

    // Atomic check-and-insert within transaction
    return this.client.transaction(this.storeName, 'readwrite', async (storeOrStores) => {
      // Handle both single store and array of stores
      const store = Array.isArray(storeOrStores) ? storeOrStores[0] : storeOrStores

      // Count existing queries for this user within the transaction
      const index = store.index('user_id')
      const countRequest = index.count(IDBKeyRange.only(data.user_id))

      const currentCount = await new Promise<number>((resolve, reject) => {
        countRequest.onsuccess = () => resolve(countRequest.result)
        countRequest.onerror = () => reject(countRequest.error)
      })

      const limitCheck = tierStore.checkLimit('savedQueries', currentCount + 1)

      if (!limitCheck.allowed) {
        console.warn('Saved queries limit reached:', limitCheck)

        // Auto-prune for local tier (outside transaction to avoid deadlock)
        if (tierStore.currentTier === 'local' && limitCheck.limit !== null) {
          // Note: Prune will be handled outside transaction
          // For now, throw error and let caller retry after prune
          throw new Error('PRUNE_REQUIRED')
        } else {
          // For paid tiers (shouldn't happen with unlimited)
          window.dispatchEvent(
            new CustomEvent('showUpgradeDialog', {
              detail: {
                limitName: 'savedQueries',
                currentTier: tierStore.currentTier,
                usage: currentCount,
                limit: limitCheck.limit,
              },
            })
          )
          throw new Error('Saved queries limit reached')
        }
      }

      const now = new Date()
      const record: SavedQueryRecord = {
        id: data.id || crypto.randomUUID(),
        user_id: data.user_id,
        title: data.title.trim(),
        description: data.description?.trim(),
        query_text: this.sanitizeQuery(data.query_text),
        tags: data.tags || [],
        folder: data.folder?.trim(),
        is_favorite: data.is_favorite || false,
        created_at: data.created_at ?? now,
        updated_at: data.updated_at ?? now,
        synced: data.synced ?? false,
        sync_version: data.sync_version ?? 0,
      }

      // Insert within transaction (IndexedDB)
      const putRequest = store.put(record)
      await new Promise<void>((resolve, reject) => {
        putRequest.onsuccess = () => resolve()
        putRequest.onerror = () => reject(putRequest.error)
      })

      // Dual-write: also save to SQLite (primary storage)
      try {
        const App = await getAppBindings()
        if (App?.SQLiteSaveQuery) {
          const sqliteRecord = {
            id: record.id,
            title: record.title,
            description: record.description || '',
            query: record.query_text,
            connection_id: '', // Not tracked in frontend
            folder: record.folder || '',
            tags: record.tags,
            created_at: record.created_at.toISOString(),
            updated_at: record.updated_at.toISOString(),
          }
          await App.SQLiteSaveQuery(JSON.stringify(sqliteRecord))
        }
      } catch (sqliteError) {
        console.warn('[SavedQueryRepository] SQLite write failed, IndexedDB write succeeded:', sqliteError)
      }

      return record
    }).catch(async (error) => {
      // Handle prune requirement outside transaction
      if (error instanceof Error && error.message === 'PRUNE_REQUIRED') {
        await this.pruneOldest(1, data.user_id)
        console.log('Auto-pruned oldest non-favorite query, retrying...')
        // Retry the create operation
        return this.create(data)
      }
      throw error
    })
  }

  /**
   * Get a saved query by ID
   * Tries SQLite first, falls back to IndexedDB
   */
  async get(id: string): Promise<SavedQueryRecord | null> {
    // Try SQLite first (primary)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteGetQuery) {
        const sqliteQuery = await App.SQLiteGetQuery(id)
        if (sqliteQuery) {
          return sqliteToSavedQueryRecord(sqliteQuery as SQLiteSavedQuery)
        }
      }
    } catch (error) {
      console.debug('[SavedQueryRepository] SQLite read failed, falling back to IndexedDB:', error)
    }

    // Fallback to IndexedDB
    return this.client.get<SavedQueryRecord>(this.storeName, id)
  }

  /**
   * Get a saved query by ID or throw error
   */
  async getOrFail(id: string): Promise<SavedQueryRecord> {
    const record = await this.get(id)
    if (!record) {
      throw new NotFoundError(`Saved query ${id} not found`)
    }
    return record
  }

  /**
   * Update a saved query
   */
  async update(
    id: string,
    updates: UpdateInput<SavedQueryRecord>
  ): Promise<SavedQueryRecord> {
    const existing = await this.getOrFail(id)

    const updated: SavedQueryRecord = {
      ...existing,
      ...updates,
      id, // Ensure ID doesn't change
      updated_at: new Date(),
      synced: false, // Mark as unsynced after update
    }

    // Sanitize query_text if updated
    if (updates.query_text !== undefined) {
      updated.query_text = this.sanitizeQuery(updates.query_text)
    }

    // Trim text fields if updated
    if (updates.title !== undefined) {
      updated.title = updates.title.trim()
    }
    if (updates.description !== undefined) {
      updated.description = updates.description?.trim()
    }
    if (updates.folder !== undefined) {
      updated.folder = updates.folder?.trim()
    }

    // Dual-write: save to SQLite (primary) and IndexedDB (fallback)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteSaveQuery) {
        const sqliteRecord = {
          id: updated.id,
          title: updated.title,
          description: updated.description || '',
          query: updated.query_text,
          connection_id: '',
          folder: updated.folder || '',
          tags: updated.tags,
          created_at: updated.created_at.toISOString(),
          updated_at: updated.updated_at.toISOString(),
        }
        await App.SQLiteSaveQuery(JSON.stringify(sqliteRecord))
      }
    } catch (error) {
      console.warn('[SavedQueryRepository] SQLite update failed, continuing with IndexedDB:', error)
    }

    await this.client.put(this.storeName, updated)
    return updated
  }

  /**
   * Delete a saved query
   * Deletes from both SQLite and IndexedDB (dual-write pattern)
   */
  async delete(id: string): Promise<void> {
    // Dual-delete: delete from SQLite (primary) and IndexedDB (fallback)
    try {
      const App = await getAppBindings()
      if (App?.SQLiteDeleteQuery) {
        await App.SQLiteDeleteQuery(id)
      }
    } catch (error) {
      console.warn('[SavedQueryRepository] SQLite delete failed, continuing with IndexedDB:', error)
    }

    await this.client.delete(this.storeName, id)
  }

  /**
   * Search saved queries with filters
   * Tries SQLite first, falls back to IndexedDB
   */
  async search(
    options: SavedQuerySearchOptions = {}
  ): Promise<PaginatedResult<SavedQueryRecord>> {
    const {
      userId,
      searchText,
      folder,
      tags,
      favoritesOnly,
      unsyncedOnly,
      startDate,
      endDate,
      sortBy = 'updated_at',
      sortDirection = 'desc',
      limit = 50,
      offset = 0,
    } = options

    // Try SQLite first
    try {
      const App = await getAppBindings()
      if (App?.SQLiteGetQueries) {
        const sqliteQueries = await App.SQLiteGetQueries()
        if (sqliteQueries && sqliteQueries.length > 0) {
          let records = (sqliteQueries as SQLiteSavedQuery[]).map(q => sqliteToSavedQueryRecord(q, userId || ''))

          // Apply in-memory filters
          if (folder) {
            records = records.filter((r) => r.folder === folder)
          }
          if (tags && tags.length > 0) {
            records = records.filter((r) =>
              tags.some((tag) => r.tags.includes(tag))
            )
          }
          if (favoritesOnly) {
            records = records.filter((r) => r.is_favorite)
          }
          if (unsyncedOnly) {
            records = records.filter((r) => !r.synced)
          }
          if (startDate) {
            records = records.filter((r) => r.created_at >= startDate)
          }
          if (endDate) {
            records = records.filter((r) => r.created_at <= endDate)
          }
          if (searchText) {
            const searchLower = searchText.toLowerCase()
            records = records.filter(
              (r) =>
                r.title.toLowerCase().includes(searchLower) ||
                r.description?.toLowerCase().includes(searchLower) ||
                r.query_text.toLowerCase().includes(searchLower) ||
                r.tags.some((tag) => tag.toLowerCase().includes(searchLower))
            )
          }

          // Sort
          records.sort((a, b) => {
            let aVal: string | Date
            let bVal: string | Date

            switch (sortBy) {
              case 'title':
                aVal = a.title.toLowerCase()
                bVal = b.title.toLowerCase()
                break
              case 'created_at':
                aVal = a.created_at
                bVal = b.created_at
                break
              case 'updated_at':
              default:
                aVal = a.updated_at
                bVal = b.updated_at
            }

            if (sortDirection === 'asc') {
              return aVal < bVal ? -1 : aVal > bVal ? 1 : 0
            } else {
              return aVal > bVal ? -1 : aVal < bVal ? 1 : 0
            }
          })

          // Apply pagination
          const total = records.length
          const items = records.slice(offset, offset + limit)
          const hasMore = offset + limit < total

          return {
            items,
            total,
            hasMore,
          }
        }
      }
    } catch (error) {
      console.debug('[SavedQueryRepository] SQLite search failed, falling back to IndexedDB:', error)
    }

    // Fallback to IndexedDB
    // Determine best index to use
    let indexName: string | undefined
    let keyRange: IDBKeyRange | undefined

    if (userId && favoritesOnly) {
      // Use compound index for user + favorites
      indexName = 'user_favorites'
      keyRange = IDBKeyRange.only([userId, true])
    } else if (userId) {
      indexName = 'user_id'
      keyRange = IDBKeyRange.only(userId)
    } else if (favoritesOnly) {
      indexName = 'is_favorite'
      keyRange = IDBKeyRange.only(true)
    } else if (folder) {
      indexName = 'folder'
      keyRange = IDBKeyRange.only(folder)
    } else if (tags && tags.length === 1) {
      // Can use index for single tag
      indexName = 'tags'
      keyRange = IDBKeyRange.only(tags[0])
    } else if (sortBy === 'created_at') {
      indexName = 'created_at'
    } else if (sortBy === 'updated_at') {
      indexName = 'updated_at'
    }

    // Get all matching records
    const allRecords = await this.client.getAll<SavedQueryRecord>(
      this.storeName,
      {
        index: indexName,
        range: keyRange,
        direction: sortDirection === 'desc' ? 'prev' : 'next',
      }
    )

    // Apply in-memory filters
    let filtered = allRecords

    if (folder && !indexName?.includes('folder')) {
      filtered = filtered.filter((r) => r.folder === folder)
    }

    if (tags && tags.length > 0) {
      filtered = filtered.filter((r) =>
        tags.some((tag) => r.tags.includes(tag))
      )
    }

    if (favoritesOnly && !indexName?.includes('favorite')) {
      filtered = filtered.filter((r) => r.is_favorite)
    }

    if (unsyncedOnly) {
      filtered = filtered.filter((r) => !r.synced)
    }

    if (startDate) {
      filtered = filtered.filter((r) => r.created_at >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter((r) => r.created_at <= endDate)
    }

    if (searchText) {
      const searchLower = searchText.toLowerCase()
      filtered = filtered.filter(
        (r) =>
          r.title.toLowerCase().includes(searchLower) ||
          r.description?.toLowerCase().includes(searchLower) ||
          r.query_text.toLowerCase().includes(searchLower) ||
          r.tags.some((tag) => tag.toLowerCase().includes(searchLower))
      )
    }

    // Sort if not using index
    if (!indexName || searchText || tags) {
      filtered.sort((a, b) => {
        let aVal: string | Date
        let bVal: string | Date

        switch (sortBy) {
          case 'title':
            aVal = a.title.toLowerCase()
            bVal = b.title.toLowerCase()
            break
          case 'created_at':
            aVal = a.created_at
            bVal = b.created_at
            break
          case 'updated_at':
          default:
            aVal = a.updated_at
            bVal = b.updated_at
        }

        if (sortDirection === 'asc') {
          return aVal < bVal ? -1 : aVal > bVal ? 1 : 0
        } else {
          return aVal > bVal ? -1 : aVal < bVal ? 1 : 0
        }
      })
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
   * Get all saved queries for a user
   * Tries SQLite first, falls back to IndexedDB
   */
  async getAllForUser(userId: string): Promise<SavedQueryRecord[]> {
    // Try SQLite first
    try {
      const App = await getAppBindings()
      if (App?.SQLiteGetQueries) {
        const sqliteQueries = await App.SQLiteGetQueries()
        if (sqliteQueries && sqliteQueries.length > 0) {
          return (sqliteQueries as SQLiteSavedQuery[]).map(q => sqliteToSavedQueryRecord(q, userId))
        }
      }
    } catch (error) {
      console.debug('[SavedQueryRepository] SQLite read failed, falling back to IndexedDB:', error)
    }

    // Fallback to IndexedDB
    return this.client.getAll<SavedQueryRecord>(this.storeName, {
      index: 'user_id',
      range: IDBKeyRange.only(userId),
    })
  }

  /**
   * Get favorite queries for a user
   */
  async getFavorites(
    userId: string,
    limit = 50
  ): Promise<SavedQueryRecord[]> {
    const result = await this.search({
      userId,
      favoritesOnly: true,
      limit,
      sortDirection: 'desc',
    })
    return result.items
  }

  /**
   * Get queries by folder
   */
  async getByFolder(
    folder: string,
    userId?: string
  ): Promise<SavedQueryRecord[]> {
    const result = await this.search({
      userId,
      folder,
    })
    return result.items
  }

  /**
   * Get queries by tag
   */
  async getByTag(
    tag: string,
    userId?: string
  ): Promise<SavedQueryRecord[]> {
    const result = await this.search({
      userId,
      tags: [tag],
    })
    return result.items
  }

  /**
   * Get recently updated queries
   */
  async getRecent(
    userId: string,
    limit = 20
  ): Promise<SavedQueryRecord[]> {
    const result = await this.search({
      userId,
      limit,
      sortBy: 'updated_at',
      sortDirection: 'desc',
    })
    return result.items
  }

  /**
   * Toggle favorite status
   */
  async toggleFavorite(id: string): Promise<SavedQueryRecord> {
    const query = await this.getOrFail(id)
    return this.update(id, {
      is_favorite: !query.is_favorite,
    })
  }

  /**
   * Get all unique folders for a user
   */
  async getAllFolders(userId: string): Promise<string[]> {
    const queries = await this.getAllForUser(userId)
    const folders = new Set<string>()

    queries.forEach((query) => {
      if (query.folder) {
        folders.add(query.folder)
      }
    })

    return Array.from(folders).sort()
  }

  /**
   * Get all unique tags for a user
   */
  async getAllTags(userId: string): Promise<string[]> {
    const queries = await this.getAllForUser(userId)
    const tags = new Set<string>()

    queries.forEach((query) => {
      query.tags.forEach((tag) => tags.add(tag))
    })

    return Array.from(tags).sort()
  }

  /**
   * Get unsynced queries for server sync
   */
  async getUnsynced(limit = 100): Promise<SavedQueryRecord[]> {
    return this.client.getAll<SavedQueryRecord>(this.storeName, {
      index: 'synced',
      range: IDBKeyRange.only(false),
      limit,
    })
  }

  /**
   * Mark queries as synced
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
   * Get total count of saved queries
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
   * Prune oldest saved queries
   * Used for automatic cleanup when tier limits are reached
   * Never deletes favorite queries
   *
   * @param count - Number of oldest entries to remove
   * @param userId - Optional user ID filter
   */
  async pruneOldest(count: number, userId?: string): Promise<number> {
    // Get all queries, sorted oldest first (non-favorites only)
    const result = await this.search({
      userId,
      favoritesOnly: false,
      sortBy: 'updated_at',
      sortDirection: 'asc',
      limit: count,
    })

    const toDelete = result.items.filter((q) => !q.is_favorite)

    if (toDelete.length === 0) {
      console.warn('Cannot prune: all queries are favorites')
      return 0
    }

    // Delete the oldest non-favorite queries
    await Promise.all(toDelete.map((q) => this.delete(q.id)))
    return toDelete.length
  }

  /**
   * Clear all saved queries for a user
   */
  async clearUserQueries(userId: string): Promise<number> {
    const queries = await this.getAllForUser(userId)
    await Promise.all(queries.map((q) => this.delete(q.id)))
    return queries.length
  }

  /**
   * Duplicate a saved query
   */
  async duplicate(id: string): Promise<SavedQueryRecord> {
    const original = await this.getOrFail(id)

    // Create a copy with modified title
    const copy = await this.create({
      user_id: original.user_id,
      title: `${original.title} (Copy)`,
      description: original.description,
      query_text: original.query_text,
      tags: [...original.tags],
      folder: original.folder,
      is_favorite: false, // Copies are not favorites by default
    })

    return copy
  }

  /**
   * Sanitize query text (remove sensitive data)
   *
   * Removes credentials, API keys, tokens, and other sensitive information
   * that might accidentally be included in SQL queries
   */
  private sanitizeQuery(query: string): string {
    let sanitized = query

    // Password patterns (quoted and unquoted)
    sanitized = sanitized.replace(
      /(password|pwd|pass|passwd)\s*[=:]\s*['"][^'"]*['"]/gi,
      "$1='***'"
    )
    sanitized = sanitized.replace(
      /(password|pwd|pass|passwd)\s*[=:]\s*([^;,\s)]+)/gi,
      '$1=***'
    )

    // API keys and tokens
    sanitized = sanitized.replace(
      /(api[_-]?key|apikey|access[_-]?key|secret[_-]?key|auth[_-]?token|bearer|token)\s*[=:]\s*['"]?[\w\-/+=]+['"]?/gi,
      '$1=***'
    )

    // AWS credentials
    sanitized = sanitized.replace(
      /(aws_access_key_id|aws_secret_access_key)\s*[=:]\s*[\w/+=]+/gi,
      '$1=***'
    )

    // Database connection strings with credentials
    sanitized = sanitized.replace(
      /(postgres|postgresql|mysql|mongodb|mssql|oracle):\/\/[^:]+:([^@]+)@/gi,
      '$1://user:***@'
    )

    // Generic credential patterns in URLs
    sanitized = sanitized.replace(
      /([?&])(key|token|secret|password|pwd)=([^&\s]+)/gi,
      '$1$2=***'
    )

    return sanitized
  }
}

/**
 * Singleton instance
 */
let repositoryInstance: SavedQueryRepository | null = null

/**
 * Get singleton instance of SavedQueryRepository
 */
export function getSavedQueryRepository(): SavedQueryRepository {
  if (!repositoryInstance) {
    repositoryInstance = new SavedQueryRepository()
  }
  return repositoryInstance
}
