/**
 * Core Sync Service
 *
 * Orchestrates bidirectional sync between local IndexedDB and backend.
 * Handles offline-first, conflict detection, and data sanitization.
 *
 * @module lib/sync/sync-service
 */

import { getIndexedDBClient } from '@/lib/storage/indexeddb-client'
import { getSyncClient } from '@/lib/api/sync-client'
import {
  prepareConnectionsForSync,
  type SanitizedConnection,
} from '@/lib/sanitization/connection-sanitizer'
import { useTierStore } from '@/store/tier-store'
import { STORE_NAMES } from '@/types/storage'
import type {
  ConnectionRecord,
  SavedQueryRecord,
  QueryHistoryRecord,
} from '@/types/storage'
import type {
  SyncResult,
  SyncConfig,
  Conflict,
  ConflictResolution,
  ChangeSet,
  DeviceInfo,
  SyncProgress,
  UploadChangesRequest,
} from '@/types/sync'
import { generateDeviceId } from '@/types/sync'

/**
 * Sync service for managing cloud synchronization
 */
export class SyncService {
  private indexedDB = getIndexedDBClient()
  private syncClient = getSyncClient()
  private config: SyncConfig
  private intervalId: number | null = null
  private isSyncing = false
  private deviceInfo: DeviceInfo | null = null
  private progressCallbacks: Array<(progress: SyncProgress) => void> = []

  constructor(config: Partial<SyncConfig> = {}) {
    this.config = { ...this.getDefaultConfig(), ...config }
    this.initializeDevice()
  }

  /**
   * Get default sync configuration
   */
  private getDefaultConfig(): SyncConfig {
    return {
      autoSyncEnabled: true,
      syncIntervalMs: 5 * 60 * 1000, // 5 minutes
      syncQueryHistory: true,
      maxHistoryItems: 1000,
      enableConflictResolution: false,
      defaultConflictResolution: 'remote',
      autoRetry: true,
      maxRetries: 3,
      retryDelayMs: 1000,
      requireOnline: false,
      uploadBatchSize: 100,
      downloadBatchSize: 100,
    }
  }

  /**
   * Initialize device information
   */
  private async initializeDevice(): Promise<void> {
    const stored = localStorage.getItem('sync-device-info')

    if (stored) {
      try {
        this.deviceInfo = JSON.parse(stored)
        return
      } catch {
        // Invalid stored data, create new
      }
    }

    this.deviceInfo = {
      deviceId: generateDeviceId(),
      deviceName: `${navigator.platform} - ${new Date().toLocaleDateString()}`,
      userAgent: navigator.userAgent,
      registeredAt: new Date(),
    }

    localStorage.setItem('sync-device-info', JSON.stringify(this.deviceInfo))
  }

  /**
   * Get current device info
   */
  getDeviceInfo(): DeviceInfo {
    if (!this.deviceInfo) {
      throw new Error('Device info not initialized')
    }
    return this.deviceInfo
  }

  /**
   * Start automatic periodic sync
   */
  startSync(): void {
    if (this.intervalId !== null) {
      console.warn('Sync already started')
      return
    }

    if (!this.config.autoSyncEnabled) {
      console.warn('Auto-sync is disabled')
      return
    }

    // Initial sync
    this.syncNow().catch((error) => {
      console.error('Initial sync failed:', error)
    })

    // Set up periodic sync
    this.intervalId = window.setInterval(() => {
      if (!this.isSyncing) {
        this.syncNow().catch((error) => {
          console.error('Periodic sync failed:', error)
        })
      }
    }, this.config.syncIntervalMs)

    console.log(`Sync started: every ${this.config.syncIntervalMs / 1000}s`)
  }

  /**
   * Stop automatic sync
   */
  stopSync(): void {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId)
      this.intervalId = null
      console.log('Sync stopped')
    }
  }

  /**
   * Register progress callback
   */
  onProgress(callback: (progress: SyncProgress) => void): () => void {
    this.progressCallbacks.push(callback)
    return () => {
      this.progressCallbacks = this.progressCallbacks.filter((cb) => cb !== callback)
    }
  }

  /**
   * Emit progress update
   */
  private emitProgress(progress: SyncProgress): void {
    this.progressCallbacks.forEach((callback) => {
      try {
        callback(progress)
      } catch (error) {
        console.error('Progress callback error:', error)
      }
    })
  }

  /**
   * Perform a complete sync cycle
   */
  async syncNow(): Promise<SyncResult> {
    // Check if sync is already in progress
    if (this.isSyncing) {
      throw new Error('Sync already in progress')
    }

    const startTime = Date.now()
    this.isSyncing = true

    const result: SyncResult = {
      success: false,
      uploaded: 0,
      downloaded: 0,
      conflicts: [],
      completedAt: new Date(),
      durationMs: 0,
      stats: {
        connectionsUploaded: 0,
        connectionsDownloaded: 0,
        queriesUploaded: 0,
        queriesDownloaded: 0,
        historyUploaded: 0,
        historyDownloaded: 0,
      },
    }

    try {
      // 1. Check if user is authenticated and has sync feature
      this.emitProgress({ phase: 'preparing', percentage: 0, totalItems: 0, processedItems: 0 })

      if (!this.canSync()) {
        throw new Error('Sync not available for current tier or user not authenticated')
      }

      // 2. Check if online (if required)
      if (this.config.requireOnline && !this.isOnline()) {
        throw new Error('Offline: sync requires network connection')
      }

      // 3. Get local changes since last sync
      this.emitProgress({ phase: 'preparing', percentage: 10, totalItems: 0, processedItems: 0 })
      const localChanges = await this.getLocalChanges()

      // 4. Sanitize and upload changes
      this.emitProgress({
        phase: 'uploading',
        percentage: 20,
        totalItems: localChanges.total,
        processedItems: 0,
      })

      const uploadResult = await this.uploadChanges(localChanges.changes)
      result.uploaded = uploadResult?.acceptedCount ?? 0
      result.stats.connectionsUploaded = localChanges.changes.connections.length
      result.stats.queriesUploaded = localChanges.changes.savedQueries.length
      result.stats.historyUploaded = localChanges.changes.queryHistory?.length || 0

      // 5. Download remote changes
      this.emitProgress({
        phase: 'downloading',
        percentage: 50,
        totalItems: localChanges.total,
        processedItems: localChanges.total,
      })

      const rawDownload = await this.downloadChanges()
      const downloadResult = {
        connections: rawDownload?.connections ?? [],
        savedQueries: rawDownload?.savedQueries ?? [],
        queryHistory: rawDownload?.queryHistory ?? [],
        serverTimestamp: rawDownload?.serverTimestamp ?? Date.now(),
        hasMore: rawDownload?.hasMore ?? false,
      }

      result.downloaded =
        downloadResult.connections.length +
        downloadResult.savedQueries.length +
        (downloadResult.queryHistory?.length || 0)

      // 6. Detect conflicts
      this.emitProgress({ phase: 'resolving', percentage: 70, totalItems: result.downloaded, processedItems: 0 })

      const conflicts = await this.detectConflicts(localChanges.local, downloadResult)
      result.conflicts = conflicts

      // 7. Auto-resolve or prompt user
      if (conflicts.length > 0) {
        if (this.config.enableConflictResolution) {
          const autoResolved = await this.autoResolveConflicts(conflicts)
          result.conflicts = conflicts.filter((c) => !autoResolved.includes(c.id))
        }
      }

      // 8. Merge remote changes into local storage
      this.emitProgress({ phase: 'merging', percentage: 85, totalItems: result.downloaded, processedItems: 0 })

      await this.mergeRemoteChanges(downloadResult)
      result.stats.connectionsDownloaded = downloadResult.connections.length
      result.stats.queriesDownloaded = downloadResult.savedQueries.length
      result.stats.historyDownloaded = downloadResult.queryHistory?.length || 0

      // 9. Update last sync timestamp
      this.emitProgress({ phase: 'complete', percentage: 95, totalItems: result.downloaded, processedItems: result.downloaded })

      await this.updateLastSyncTimestamp(downloadResult.serverTimestamp)

      result.success = true
      result.durationMs = Date.now() - startTime

      this.emitProgress({ phase: 'complete', percentage: 100, totalItems: result.downloaded, processedItems: result.downloaded })

      return result
    } catch (error) {
      result.success = false
      result.error = error instanceof Error ? error.message : 'Unknown error'
      result.durationMs = Date.now() - startTime
      throw error
    } finally {
      this.isSyncing = false
    }
  }

  /**
   * Detect the current online status in a test-friendly way
   */
  private isOnline(): boolean {
    if (typeof navigator === 'undefined') {
      return true
    }

    if (typeof navigator.onLine === 'boolean') {
      return navigator.onLine
    }

    return true
  }

  /**
   * Normalize various date inputs to a unix timestamp
   */
  private toTimestamp(value: unknown): number {
    if (value instanceof Date) {
      return value.getTime()
    }

    if (typeof value === 'number') {
      return value
    }

    if (typeof value === 'string') {
      const parsed = Date.parse(value)
      if (!Number.isNaN(parsed)) {
        return parsed
      }
    }

    return Date.now()
  }

  /**
   * Check if sync is available
   */
  private canSync(): boolean {
    const tierStore = useTierStore.getState()
    return tierStore.hasFeature('sync') && !!tierStore.licenseKey
  }

  /**
   * Get local changes since last sync
   */
  private async getLocalChanges(): Promise<{
    changes: UploadChangesRequest
    local: {
      connections: ConnectionRecord[]
      savedQueries: SavedQueryRecord[]
      queryHistory: QueryHistoryRecord[]
    }
    total: number
  }> {
    const lastSyncAt = await this.getLastSyncTimestamp()

    // Get all connections
    const connections = (await this.indexedDB.getAll<ConnectionRecord>(
      STORE_NAMES.CONNECTIONS
    )) ?? []

    // Get saved queries
    const savedQueries = (await this.indexedDB.getAll<SavedQueryRecord>(
      STORE_NAMES.SAVED_QUERIES
    )) ?? []

    // Get query history (if enabled)
    let queryHistory: QueryHistoryRecord[] = []
    if (this.config.syncQueryHistory) {
      queryHistory = (await this.indexedDB.getAll<QueryHistoryRecord>(
        STORE_NAMES.QUERY_HISTORY,
        {
          limit: this.config.maxHistoryItems,
        }
      )) ?? []
    }

    // Filter changes since last sync
    const changedConnections = connections.filter((c) => {
      if (!c.synced) return true
      if (!c.updated_at) return false
      return this.toTimestamp(c.updated_at) > lastSyncAt
    })

    const changedQueries = savedQueries.filter((q) => {
      if (!q.synced) return true
      if (!q.updated_at) return false
      return this.toTimestamp(q.updated_at) > lastSyncAt
    })

    const changedHistory = queryHistory.filter((h) => {
      if (!h.synced) return true
      if (!h.executed_at) return false
      return this.toTimestamp(h.executed_at) > lastSyncAt
    })

    // Sanitize connections
    const { safeConnections, unsafeConnections } = prepareConnectionsForSync(
      changedConnections as any[] // Type casting needed due to ConnectionRecord vs DatabaseConnection difference
    )

    if (unsafeConnections.length > 0) {
      console.warn('Unsafe connections detected, not syncing:', unsafeConnections)
    }

    const deviceInfo = this.getDeviceInfo()

    // Build change sets - map sanitized connections back to records for change tracking
    const connectionChangeSets: ChangeSet<SanitizedConnection>[] = []
    for (const conn of changedConnections) {
      const sanitized = safeConnections.find((sc: any) => {
        const sanitizedId = sc.id ?? sc.connection_id
        return sanitizedId === conn.connection_id
      })

      if (!sanitized) {
        continue
      }

      connectionChangeSets.push({
        id: crypto.randomUUID(),
        entityType: 'connection',
        entityId: conn.connection_id,
        action: 'update',
        data: sanitized,
        timestamp: this.toTimestamp(conn.updated_at),
        syncVersion: conn.sync_version,
        deviceId: deviceInfo.deviceId,
      })
    }

    const changes: UploadChangesRequest = {
      deviceId: deviceInfo.deviceId,
      lastSyncAt,
      connections: connectionChangeSets,
      savedQueries: changedQueries.map((query) => ({
        id: crypto.randomUUID(),
        entityType: 'saved_query',
        entityId: query.id,
        action: 'update',
        data: {
          id: query.id,
          user_id: query.user_id,
          name: query.title,
          description: query.description,
          query: query.query_text,
          tags: query.tags,
          folder: query.folder,
          favorite: query.is_favorite,
          created_at: query.created_at,
          updated_at: query.updated_at,
          sync_version: query.sync_version,
        } as unknown as any,
        timestamp: this.toTimestamp(query.updated_at),
        syncVersion: query.sync_version,
        deviceId: deviceInfo.deviceId,
      })),
      queryHistory: this.config.syncQueryHistory
        ? changedHistory.map((history) => ({
            id: crypto.randomUUID(),
            entityType: 'query_history',
            entityId: history.id,
            action: 'update',
            data: history,
            timestamp: this.toTimestamp(history.executed_at),
            syncVersion: history.sync_version,
            deviceId: deviceInfo.deviceId,
          }))
        : undefined,
    }

    return {
      changes,
      local: {
        connections,
        savedQueries,
        queryHistory,
      },
      total: changedConnections.length + changedQueries.length + changedHistory.length,
    }
  }

  /**
   * Upload changes to server
   */
  private async uploadChanges(changes: UploadChangesRequest) {
    return this.syncClient.uploadChanges(changes)
  }

  /**
   * Download changes from server
   */
  private async downloadChanges() {
    const lastSyncAt = await this.getLastSyncTimestamp()
    return this.syncClient.downloadChanges({
      since: new Date(lastSyncAt),
      limit: this.config.downloadBatchSize,
    })
  }

  /**
   * Detect conflicts between local and remote data
   */
  private async detectConflicts(
    local: {
      connections: ConnectionRecord[]
      savedQueries: SavedQueryRecord[]
      queryHistory: QueryHistoryRecord[]
    },
    remote: {
      connections: SanitizedConnection[]
      savedQueries: SavedQueryRecord[]
      queryHistory?: QueryHistoryRecord[]
    }
  ): Promise<Conflict[]> {
    const conflicts: Conflict[] = []
    const debug = typeof process !== 'undefined' && process.env.DEBUG_SYNC === '1'

    // Check connection conflicts
    for (const remoteConn of remote.connections) {
      // Remote connections are sanitized, so we need to access properties carefully
      const remoteId = (remoteConn as any).id || (remoteConn as any).connection_id
      const localConn = local.connections.find((c) => c.connection_id === remoteId)

      if (localConn) {
        const remoteSyncVersion = (remoteConn as any).sync_version || 0
        const remoteUpdatedAt = new Date((remoteConn as any).updated_at || (remoteConn as any).sanitizedAt)

        // Compare sync versions and timestamps
        if (
          localConn.sync_version !== remoteSyncVersion &&
          localConn.updated_at.getTime() !== remoteUpdatedAt.getTime()
        ) {
          if (debug) {
            console.debug('[sync] conflict detected for connection', remoteId, {
              localSyncVersion: localConn.sync_version,
              remoteSyncVersion,
              localUpdatedAt: localConn.updated_at,
              remoteUpdatedAt,
            })
          }

          conflicts.push({
            id: crypto.randomUUID(),
            entityType: 'connection',
            entityId: localConn.connection_id,
            localVersion: localConn,
            remoteVersion: remoteConn,
            localSyncVersion: localConn.sync_version,
            remoteSyncVersion: remoteSyncVersion,
            localUpdatedAt: localConn.updated_at,
            remoteUpdatedAt: remoteUpdatedAt,
            recommendedResolution:
              localConn.updated_at > remoteUpdatedAt ? 'local' : 'remote',
            reason: 'Both local and remote versions were modified',
          })
        }
      } else if (debug) {
        console.debug('[sync] no local match for remote connection', remoteId)
      }
    }

    // Check saved query conflicts
    for (const remoteQuery of remote.savedQueries) {
      const localQuery = local.savedQueries.find((q) => q.id === remoteQuery.id)

      if (localQuery) {
        if (
          localQuery.sync_version !== remoteQuery.sync_version &&
          localQuery.updated_at.getTime() !== remoteQuery.updated_at.getTime()
        ) {
          conflicts.push({
            id: crypto.randomUUID(),
            entityType: 'saved_query',
            entityId: localQuery.id,
            localVersion: localQuery,
            remoteVersion: remoteQuery,
            localSyncVersion: localQuery.sync_version,
            remoteSyncVersion: remoteQuery.sync_version,
            localUpdatedAt: localQuery.updated_at,
            remoteUpdatedAt: remoteQuery.updated_at,
            recommendedResolution:
              localQuery.updated_at > remoteQuery.updated_at ? 'local' : 'remote',
            reason: 'Both local and remote versions were modified',
          })
        }
      }
    }

    return conflicts
  }

  /**
   * Auto-resolve conflicts based on strategy
   */
  private async autoResolveConflicts(conflicts: Conflict[]): Promise<string[]> {
    const resolved: string[] = []

    for (const conflict of conflicts) {
      const resolution = conflict.recommendedResolution

      if (resolution === 'manual') {
        // Can't auto-resolve, user must decide
        continue
      }

      try {
        await this.resolveConflict(conflict.id, resolution)
        resolved.push(conflict.id)
      } catch (error) {
        console.error(`Failed to auto-resolve conflict ${conflict.id}:`, error)
      }
    }

    return resolved
  }

  /**
   * Resolve a specific conflict
   */
  async resolveConflict(
    conflictId: string,
    resolution: ConflictResolution,
    conflict?: Conflict
  ): Promise<void> {
    // Notify server of resolution
    await this.syncClient.resolveConflict(conflictId, resolution)

    // Apply resolution locally if we have the conflict data
    if (conflict) {
      if (resolution === 'local') {
        // Keep local version, no action needed
      } else if (resolution === 'remote') {
        // Overwrite local with remote
        await this.applyRemoteVersion(conflict)
      } else if (resolution === 'keep-both') {
        // Create a new entity with remote data
        await this.createDuplicateFromRemote(conflict)
      }
    }
  }

  /**
   * Apply remote version in a conflict
   */
  private async applyRemoteVersion(conflict: Conflict): Promise<void> {
    if (conflict.entityType === 'connection') {
      const remoteData = conflict.remoteVersion as any
      await this.indexedDB.put(STORE_NAMES.CONNECTIONS, {
        ...remoteData,
        synced: true,
        sync_version: conflict.remoteSyncVersion,
      })
    } else if (conflict.entityType === 'saved_query') {
      await this.indexedDB.put(STORE_NAMES.SAVED_QUERIES, {
        ...(conflict.remoteVersion as any),
        synced: true,
        sync_version: conflict.remoteSyncVersion,
      })
    }
  }

  /**
   * Create a duplicate entity from remote data (keep-both resolution)
   */
  private async createDuplicateFromRemote(conflict: Conflict): Promise<void> {
    const newId = crypto.randomUUID()

    if (conflict.entityType === 'connection') {
      const remoteData = conflict.remoteVersion as any
      await this.indexedDB.put(STORE_NAMES.CONNECTIONS, {
        ...remoteData,
        connection_id: newId,
        name: `${remoteData.name} (remote)`,
        synced: true,
        sync_version: conflict.remoteSyncVersion,
      })
    } else if (conflict.entityType === 'saved_query') {
      const remoteData = conflict.remoteVersion as any
      await this.indexedDB.put(STORE_NAMES.SAVED_QUERIES, {
        ...remoteData,
        id: newId,
        title: `${remoteData.title} (remote)`,
        synced: true,
        sync_version: conflict.remoteSyncVersion,
      })
    }
  }

  /**
   * Merge remote changes into local storage
   */
  private async mergeRemoteChanges(remote: {
    connections: SanitizedConnection[]
    savedQueries: SavedQueryRecord[]
    queryHistory?: QueryHistoryRecord[]
  }): Promise<void> {
    // Merge connections (sanitized from server)
    for (const remoteConn of remote.connections) {
      const rc: any = remoteConn
      const syncVersion = (rc.sync_version ?? 0) as number
      await this.indexedDB.put(STORE_NAMES.CONNECTIONS, {
        ...rc,
        synced: true,
        sync_version: syncVersion + 1,
      })
    }

    // Map server saved query shape -> local SavedQueryRecord
    for (const remoteQuery of remote.savedQueries) {
      const localShape: SavedQueryRecord = {
        id: (remoteQuery as any).id,
        user_id: (remoteQuery as any).user_id,
        title: (remoteQuery as any).name ?? (remoteQuery as any).title,
        description: (remoteQuery as any).description,
        query_text: (remoteQuery as any).query ?? (remoteQuery as any).query_text,
        tags: (remoteQuery as any).tags ?? [],
        folder: (remoteQuery as any).folder,
        is_favorite: (remoteQuery as any).favorite ?? (remoteQuery as any).is_favorite ?? false,
        created_at: new Date((remoteQuery as any).created_at ?? new Date()),
        updated_at: new Date((remoteQuery as any).updated_at ?? new Date()),
        synced: true,
        sync_version: (remoteQuery as any).sync_version + 1,
      }
      await this.indexedDB.put(STORE_NAMES.SAVED_QUERIES, localShape)
    }

    // Merge query history (if enabled)
    if (remote.queryHistory && this.config.syncQueryHistory) {
      for (const rh of remote.queryHistory) {
        const local = {
          ...rh,
          synced: true,
          sync_version: rh.sync_version + 1,
        }
        await this.indexedDB.put(STORE_NAMES.QUERY_HISTORY, local)
      }
    }
  }

  /**
   * Get last sync timestamp from local storage
   */
  private async getLastSyncTimestamp(): Promise<number> {
    const stored = localStorage.getItem('sync-last-timestamp')
    return stored ? parseInt(stored, 10) : 0
  }

  /**
   * Update last sync timestamp
   */
  private async updateLastSyncTimestamp(timestamp: number): Promise<void> {
    localStorage.setItem('sync-last-timestamp', timestamp.toString())

    if (this.deviceInfo) {
      this.deviceInfo.lastSyncAt = new Date(timestamp)
      localStorage.setItem('sync-device-info', JSON.stringify(this.deviceInfo))
    }
  }

  /**
   * Get current sync configuration
   */
  getConfig(): SyncConfig {
    return { ...this.config }
  }

  /**
   * Update sync configuration
   */
  updateConfig(updates: Partial<SyncConfig>): void {
    this.config = { ...this.config, ...updates }

    // Restart sync if interval changed
    if (updates.syncIntervalMs && this.intervalId !== null) {
      this.stopSync()
      this.startSync()
    }
  }

  /**
   * Check if sync is currently in progress
   */
  isSyncInProgress(): boolean {
    return this.isSyncing
  }
}
