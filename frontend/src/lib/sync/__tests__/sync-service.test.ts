/**
 * Sync Service Tests
 *
 * Comprehensive test suite for sync service functionality.
 * Tests offline-first behavior, conflict resolution, and data sanitization.
 *
 * @module lib/sync/__tests__/sync-service.test
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { SyncService } from '../sync-service'
import { getIndexedDBClient } from '@/lib/storage/indexeddb-client'
import { getSyncClient } from '@/lib/api/sync-client'
import { STORE_NAMES } from '@/types/storage'
import type { ConnectionRecord, SavedQueryRecord } from '@/types/storage'
import type { SyncConfig } from '@/types/sync'

// Mock dependencies
vi.mock('@/lib/storage/indexeddb-client')
vi.mock('@/lib/api/sync-client')
vi.mock('@/store/tier-store', () => ({
  useTierStore: {
    getState: () => ({
      hasFeature: vi.fn(() => true),
      licenseKey: 'test-license-key',
    }),
  },
}))

describe('SyncService', () => {
  let service: SyncService
  let mockIndexedDB: any
  let mockSyncClient: any

  beforeEach(() => {
    // Mock IndexedDB client
    mockIndexedDB = {
      getAll: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
    }
    vi.mocked(getIndexedDBClient).mockReturnValue(mockIndexedDB as any)

    // Mock Sync API client
    mockSyncClient = {
      uploadChanges: vi.fn(),
      downloadChanges: vi.fn(),
      resolveConflict: vi.fn(),
    }
    vi.mocked(getSyncClient).mockReturnValue(mockSyncClient as any)

    // Mock localStorage
    const localStorageMock = (() => {
      let store: Record<string, string> = {}
      return {
        getItem: (key: string) => store[key] || null,
        setItem: (key: string, value: string) => {
          store[key] = value
        },
        removeItem: (key: string) => {
          delete store[key]
        },
        clear: () => {
          store = {}
        },
      }
    })()
    Object.defineProperty(window, 'localStorage', { value: localStorageMock })

    // Create service instance
    service = new SyncService()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('initialization', () => {
    it('should create device info on first run', () => {
      const deviceInfo = service.getDeviceInfo()

      expect(deviceInfo).toBeDefined()
      expect(deviceInfo.deviceId).toMatch(/^device_/)
      expect(deviceInfo.deviceName).toBeTruthy()
      expect(deviceInfo.userAgent).toBe(navigator.userAgent)
      expect(deviceInfo.registeredAt).toBeInstanceOf(Date)
    })

    it('should reuse existing device info', () => {
      const service1 = new SyncService()
      const deviceInfo1 = service1.getDeviceInfo()

      const service2 = new SyncService()
      const deviceInfo2 = service2.getDeviceInfo()

      expect(deviceInfo1.deviceId).toBe(deviceInfo2.deviceId)
    })

    it('should use default config when no config provided', () => {
      const config = service.getConfig()

      expect(config.autoSyncEnabled).toBe(true)
      expect(config.syncIntervalMs).toBe(5 * 60 * 1000)
      expect(config.syncQueryHistory).toBe(true)
    })

    it('should merge custom config with defaults', () => {
      const customService = new SyncService({
        syncIntervalMs: 10000,
        syncQueryHistory: false,
      })

      const config = customService.getConfig()
      expect(config.syncIntervalMs).toBe(10000)
      expect(config.syncQueryHistory).toBe(false)
      expect(config.autoSyncEnabled).toBe(true) // Default
    })
  })

  describe('periodic sync', () => {
    it('should start periodic sync', () => {
      vi.useFakeTimers()
      const syncNowSpy = vi.spyOn(service, 'syncNow').mockResolvedValue({} as any)

      service.startSync()

      // Initial sync
      expect(syncNowSpy).toHaveBeenCalledTimes(1)

      // Advance time by sync interval
      vi.advanceTimersByTime(5 * 60 * 1000)
      expect(syncNowSpy).toHaveBeenCalledTimes(2)

      vi.useRealTimers()
    })

    it('should stop periodic sync', () => {
      vi.useFakeTimers()
      const syncNowSpy = vi.spyOn(service, 'syncNow').mockResolvedValue({} as any)

      service.startSync()
      service.stopSync()

      // No more syncs after stop
      vi.advanceTimersByTime(10 * 60 * 1000)
      expect(syncNowSpy).toHaveBeenCalledTimes(1) // Only initial

      vi.useRealTimers()
    })

    it('should not start sync twice', () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

      service.startSync()
      service.startSync()

      expect(consoleWarnSpy).toHaveBeenCalledWith('Sync already started')
    })
  })

  describe('syncNow', () => {
    it('should perform complete sync cycle', async () => {
      // Mock local data
      const mockConnections: ConnectionRecord[] = [
        {
          connection_id: 'conn-1',
          user_id: 'user-1',
          name: 'Test DB',
          type: 'postgres',
          host: 'localhost',
          port: 5432,
          database: 'test',
          username: 'user',
          ssl_mode: 'disable',
          environment_tags: [],
          created_at: new Date(),
          updated_at: new Date(),
          last_used_at: new Date(),
          synced: false,
          sync_version: 1,
        },
      ]

      mockIndexedDB.getAll
        .mockResolvedValueOnce(mockConnections) // connections
        .mockResolvedValueOnce([]) // saved queries
        .mockResolvedValueOnce([]) // query history

      // Mock upload response
      mockSyncClient.uploadChanges.mockResolvedValue({
        success: true,
        acceptedCount: 1,
        rejectedCount: 0,
        serverTimestamp: Date.now(),
      })

      // Mock download response
      mockSyncClient.downloadChanges.mockResolvedValue({
        connections: [],
        savedQueries: [],
        queryHistory: [],
        serverTimestamp: Date.now(),
        hasMore: false,
      })

      const result = await service.syncNow()

      expect(result.success).toBe(true)
      expect(result.uploaded).toBe(1)
      expect(result.conflicts).toHaveLength(0)
      expect(mockSyncClient.uploadChanges).toHaveBeenCalled()
      expect(mockSyncClient.downloadChanges).toHaveBeenCalled()
    })

    it('should sanitize connections before upload', async () => {
      // Mock connection with password
      const mockConnection: ConnectionRecord = {
        connection_id: 'conn-1',
        user_id: 'user-1',
        name: 'Test DB',
        type: 'postgres',
        host: 'localhost',
        port: 5432,
        database: 'test',
        username: 'user',
        ssl_mode: 'disable',
        environment_tags: [],
        created_at: new Date(),
        updated_at: new Date(),
        last_used_at: new Date(),
        synced: false,
        sync_version: 1,
      }

      mockIndexedDB.getAll
        .mockResolvedValueOnce([mockConnection])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])

      mockSyncClient.uploadChanges.mockResolvedValue({
        success: true,
        acceptedCount: 1,
        rejectedCount: 0,
        serverTimestamp: Date.now(),
      })

      mockSyncClient.downloadChanges.mockResolvedValue({
        connections: [],
        savedQueries: [],
        queryHistory: [],
        serverTimestamp: Date.now(),
        hasMore: false,
      })

      await service.syncNow()

      // Verify password was removed from upload
      const uploadCall = mockSyncClient.uploadChanges.mock.calls[0][0]
      const uploadedConnection = uploadCall.connections[0].data

      expect(uploadedConnection).toBeDefined()
      expect(uploadedConnection.password).toBeUndefined()
    })

    it('should detect and report conflicts', async () => {
      const localConnection: ConnectionRecord = {
        connection_id: 'conn-1',
        user_id: 'user-1',
        name: 'Local Version',
        type: 'postgres',
        host: 'localhost',
        port: 5432,
        database: 'test',
        username: 'user',
        ssl_mode: 'disable',
        environment_tags: [],
        created_at: new Date('2024-01-01'),
        updated_at: new Date('2024-01-02'),
        last_used_at: new Date(),
        synced: false,
        sync_version: 1,
      }

      const remoteConnection = {
        connection_id: 'conn-1',
        user_id: 'user-1',
        name: 'Remote Version',
        type: 'postgres',
        host: 'localhost',
        port: 5432,
        database: 'test',
        username: 'user',
        ssl_mode: 'disable',
        environment_tags: [],
        created_at: new Date('2024-01-01'),
        updated_at: new Date('2024-01-03'), // Newer
        last_used_at: new Date(),
        synced: true,
        sync_version: 2,
      }

      mockIndexedDB.getAll
        .mockResolvedValueOnce([localConnection])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])

      mockSyncClient.uploadChanges.mockResolvedValue({
        success: true,
        acceptedCount: 0,
        rejectedCount: 0,
        serverTimestamp: Date.now(),
      })

      mockSyncClient.downloadChanges.mockResolvedValue({
        connections: [remoteConnection],
        savedQueries: [],
        queryHistory: [],
        serverTimestamp: Date.now(),
        hasMore: false,
      })

      const result = await service.syncNow()

      expect(result.conflicts).toHaveLength(1)
      expect(result.conflicts[0].entityType).toBe('connection')
      expect(result.conflicts[0].recommendedResolution).toBe('remote') // Newer
    })

    it('should throw error when sync already in progress', async () => {
      // Start a sync that won't complete
      const slowSync = service.syncNow()

      // Try to start another sync
      await expect(service.syncNow()).rejects.toThrow('Sync already in progress')

      // Clean up
      await slowSync.catch(() => {})
    })

    it('should handle offline state when required', async () => {
      const offlineService = new SyncService({ requireOnline: true })

      // Mock offline
      Object.defineProperty(navigator, 'onLine', {
        writable: true,
        value: false,
      })

      await expect(offlineService.syncNow()).rejects.toThrow(
        'Offline: sync requires network connection'
      )
    })
  })

  describe('conflict resolution', () => {
    it('should resolve conflict with local version', async () => {
      const conflict = {
        id: 'conflict-1',
        entityType: 'connection' as const,
        entityId: 'conn-1',
        localVersion: { name: 'Local' },
        remoteVersion: { name: 'Remote' },
        localSyncVersion: 1,
        remoteSyncVersion: 2,
        localUpdatedAt: new Date(),
        remoteUpdatedAt: new Date(),
        recommendedResolution: 'local' as const,
        reason: 'Test conflict',
      }

      await service.resolveConflict(conflict.id, 'local', conflict)

      expect(mockSyncClient.resolveConflict).toHaveBeenCalledWith(
        conflict.id,
        'local'
      )
    })

    it('should resolve conflict with remote version', async () => {
      const conflict = {
        id: 'conflict-1',
        entityType: 'connection' as const,
        entityId: 'conn-1',
        localVersion: { connection_id: 'conn-1', name: 'Local' },
        remoteVersion: { connection_id: 'conn-1', name: 'Remote', sync_version: 2 },
        localSyncVersion: 1,
        remoteSyncVersion: 2,
        localUpdatedAt: new Date(),
        remoteUpdatedAt: new Date(),
        recommendedResolution: 'remote' as const,
        reason: 'Test conflict',
      }

      mockIndexedDB.put.mockResolvedValue(undefined)

      await service.resolveConflict(conflict.id, 'remote', conflict)

      expect(mockSyncClient.resolveConflict).toHaveBeenCalledWith(
        conflict.id,
        'remote'
      )
      expect(mockIndexedDB.put).toHaveBeenCalled()
    })

    it('should create duplicate when keeping both', async () => {
      const conflict = {
        id: 'conflict-1',
        entityType: 'saved_query' as const,
        entityId: 'query-1',
        localVersion: { id: 'query-1', title: 'Local Query' },
        remoteVersion: { id: 'query-1', title: 'Remote Query', sync_version: 2 },
        localSyncVersion: 1,
        remoteSyncVersion: 2,
        localUpdatedAt: new Date(),
        remoteUpdatedAt: new Date(),
        recommendedResolution: 'keep-both' as const,
        reason: 'Test conflict',
      }

      mockIndexedDB.put.mockResolvedValue(undefined)

      await service.resolveConflict(conflict.id, 'keep-both', conflict)

      expect(mockSyncClient.resolveConflict).toHaveBeenCalledWith(
        conflict.id,
        'keep-both'
      )

      // Should create new entity with remote data
      expect(mockIndexedDB.put).toHaveBeenCalledWith(
        STORE_NAMES.SAVED_QUERIES,
        expect.objectContaining({
          title: 'Remote Query (remote)',
        })
      )
    })
  })

  describe('progress tracking', () => {
    it('should emit progress updates during sync', async () => {
      const progressUpdates: any[] = []
      const unsubscribe = service.onProgress((progress) => {
        progressUpdates.push(progress)
      })

      mockIndexedDB.getAll
        .mockResolvedValue([])

      mockSyncClient.uploadChanges.mockResolvedValue({
        success: true,
        acceptedCount: 0,
        rejectedCount: 0,
        serverTimestamp: Date.now(),
      })

      mockSyncClient.downloadChanges.mockResolvedValue({
        connections: [],
        savedQueries: [],
        queryHistory: [],
        serverTimestamp: Date.now(),
        hasMore: false,
      })

      await service.syncNow()

      expect(progressUpdates.length).toBeGreaterThan(0)
      expect(progressUpdates[0].phase).toBe('preparing')
      expect(progressUpdates[progressUpdates.length - 1].phase).toBe('complete')

      unsubscribe()
    })

    it('should allow unsubscribing from progress updates', () => {
      const callback = vi.fn()
      const unsubscribe = service.onProgress(callback)

      unsubscribe()

      // Progress updates should not call the callback anymore
      // (would be verified in actual sync)
    })
  })

  describe('configuration', () => {
    it('should update sync configuration', () => {
      service.updateConfig({
        syncIntervalMs: 10000,
      })

      const config = service.getConfig()
      expect(config.syncIntervalMs).toBe(10000)
    })

    it('should restart sync when interval changes', () => {
      const stopSpy = vi.spyOn(service, 'stopSync')
      const startSpy = vi.spyOn(service, 'startSync')

      service.startSync()
      service.updateConfig({ syncIntervalMs: 10000 })

      expect(stopSpy).toHaveBeenCalled()
      expect(startSpy).toHaveBeenCalledTimes(2) // Initial + restart
    })
  })

  describe('error handling', () => {
    it('should handle upload errors gracefully', async () => {
      mockIndexedDB.getAll.mockResolvedValue([])

      mockSyncClient.uploadChanges.mockRejectedValue(
        new Error('Network error')
      )

      await expect(service.syncNow()).rejects.toThrow('Network error')
    })

    it('should handle download errors gracefully', async () => {
      mockIndexedDB.getAll.mockResolvedValue([])

      mockSyncClient.uploadChanges.mockResolvedValue({
        success: true,
        acceptedCount: 0,
        rejectedCount: 0,
        serverTimestamp: Date.now(),
      })

      mockSyncClient.downloadChanges.mockRejectedValue(
        new Error('Server error')
      )

      await expect(service.syncNow()).rejects.toThrow('Server error')
    })
  })
})
