# Howlerops - Turso Implementation Guide

## Overview

This guide provides step-by-step instructions for implementing Turso-based cloud sync for the Individual tier.

**Prerequisites:**
- Read `turso-schema.sql` - Database schema
- Read `sync-protocol.md` - Sync algorithm and protocols
- Read `turso-cost-analysis.md` - Cost analysis and optimization

---

## Table of Contents

1. [Project Setup](#project-setup)
2. [Phase 1: Database Setup](#phase-1-database-setup)
3. [Phase 2: Sync Infrastructure](#phase-2-sync-infrastructure)
4. [Phase 3: Conflict Resolution](#phase-3-conflict-resolution)
5. [Phase 4: UI Integration](#phase-4-ui-integration)
6. [Phase 5: Testing](#phase-5-testing)
7. [Phase 6: Monitoring](#phase-6-monitoring)
8. [Phase 7: Launch](#phase-7-launch)

---

## Project Setup

### 1. Install Turso CLI

```bash
# Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# Verify installation
turso --version

# Login to Turso
turso auth login
```

### 2. Install Turso Client SDK

```bash
cd /Users/jacob_1/projects/sql-studio/frontend

# Install libSQL client
npm install @libsql/client

# Install dependencies
npm install zod  # For schema validation
```

### 3. Environment Variables

Add to `/Users/jacob_1/projects/sql-studio/frontend/.env`:

```bash
# Turso Configuration
VITE_TURSO_ORG_NAME=sql-studio
VITE_TURSO_DATABASE_URL_TEMPLATE=libsql://user-{userId}-{orgName}.turso.io
VITE_TURSO_API_BASE_URL=https://api.turso.tech/v1

# Turso API Token (for database provisioning)
TURSO_API_TOKEN=your-api-token-here

# Sync Configuration
VITE_SYNC_INTERVAL_MS=30000  # 30 seconds
VITE_SYNC_BATCH_SIZE=50
VITE_SYNC_TIMEOUT_MS=5000
VITE_SYNC_MAX_RETRIES=3
```

---

## Phase 1: Database Setup

### Step 1.1: Create Turso Organization

```bash
# Create organization
turso org create sql-studio

# List organizations
turso org list
```

### Step 1.2: Create Database Template

```bash
# Create template database (used as schema reference)
turso db create sql-studio-template --location lax

# Get database URL
turso db show sql-studio-template
```

### Step 1.3: Apply Schema

```bash
# Apply schema from turso-schema.sql
turso db shell sql-studio-template < /Users/jacob_1/projects/sql-studio/docs/turso-schema.sql

# Verify tables created
turso db shell sql-studio-template "SELECT name FROM sqlite_master WHERE type='table';"
```

Expected output:
```
user_preferences
connection_templates
query_history
saved_queries
ai_memory_sessions
ai_memory_messages
sync_metadata
device_registry
sync_conflicts_archive
user_statistics
schema_migrations
```

### Step 1.4: Create Database Provisioning API

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/turso/database-provisioner.ts`:

```typescript
/**
 * Turso Database Provisioner
 * Creates per-user databases on demand
 */

interface ProvisionDatabaseOptions {
  userId: string
  organizationName: string
  location?: string // Default: closest to user
}

interface DatabaseInfo {
  name: string
  url: string
  authToken: string
}

/**
 * Provision a new Turso database for a user
 */
export async function provisionUserDatabase(
  options: ProvisionDatabaseOptions
): Promise<DatabaseInfo> {
  const { userId, organizationName, location = 'lax' } = options

  const databaseName = `user-${userId}`
  const apiToken = import.meta.env.TURSO_API_TOKEN

  if (!apiToken) {
    throw new Error('TURSO_API_TOKEN not configured')
  }

  // Step 1: Create database
  const createResponse = await fetch(
    `https://api.turso.tech/v1/organizations/${organizationName}/databases`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        name: databaseName,
        group: 'default',
        location,
        seed: 'sql-studio-template' // Use template as seed
      })
    }
  )

  if (!createResponse.ok) {
    const error = await createResponse.json()
    throw new Error(`Failed to create database: ${error.error}`)
  }

  const database = await createResponse.json()

  // Step 2: Create auth token for user
  const tokenResponse = await fetch(
    `https://api.turso.tech/v1/organizations/${organizationName}/databases/${databaseName}/auth/tokens`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        expiration: 'never', // User-specific token
        authorization: 'full-access'
      })
    }
  )

  if (!tokenResponse.ok) {
    throw new Error('Failed to create auth token')
  }

  const token = await tokenResponse.json()

  return {
    name: databaseName,
    url: database.Hostname,
    authToken: token.jwt
  }
}

/**
 * Get existing database info for user
 */
export async function getUserDatabaseInfo(
  userId: string,
  organizationName: string
): Promise<DatabaseInfo | null> {
  const databaseName = `user-${userId}`
  const apiToken = import.meta.env.TURSO_API_TOKEN

  // Fetch database info
  const response = await fetch(
    `https://api.turso.tech/v1/organizations/${organizationName}/databases/${databaseName}`,
    {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    }
  )

  if (response.status === 404) {
    return null
  }

  if (!response.ok) {
    throw new Error('Failed to fetch database info')
  }

  const database = await response.json()

  // Get or create auth token
  // (implementation depends on token storage strategy)

  return {
    name: databaseName,
    url: database.Hostname,
    authToken: '[FETCH_FROM_STORAGE]' // TODO: Implement token management
  }
}
```

### Step 1.5: Create Turso Client Wrapper

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/turso/client.ts`:

```typescript
/**
 * Turso Client Wrapper
 * Type-safe wrapper around @libsql/client
 */

import { createClient, type Client } from '@libsql/client'

interface TursoClientConfig {
  url: string
  authToken: string
}

/**
 * Singleton Turso client
 */
class TursoClientManager {
  private client: Client | null = null
  private config: TursoClientConfig | null = null

  /**
   * Initialize Turso client
   */
  async initialize(config: TursoClientConfig): Promise<void> {
    if (this.client && this.config?.url === config.url) {
      // Already initialized with same config
      return
    }

    // Close existing client
    if (this.client) {
      this.client.close()
    }

    // Create new client
    this.client = createClient({
      url: config.url,
      authToken: config.authToken,

      // Performance options
      intMode: 'number',

      // Future: Embedded replica support
      // syncUrl: config.url,
      // syncInterval: 60
    })

    this.config = config

    console.log('[Turso] Client initialized:', config.url)
  }

  /**
   * Get client instance
   */
  getClient(): Client {
    if (!this.client) {
      throw new Error('Turso client not initialized')
    }
    return this.client
  }

  /**
   * Execute SQL query
   */
  async execute(sql: string, args?: unknown[]): Promise<any> {
    const client = this.getClient()
    return client.execute({ sql, args })
  }

  /**
   * Execute batch of queries
   */
  async batch(queries: Array<{ sql: string; args?: unknown[] }>): Promise<any[]> {
    const client = this.getClient()
    return client.batch(queries.map(q => ({ sql: q.sql, args: q.args })))
  }

  /**
   * Close client
   */
  close(): void {
    if (this.client) {
      this.client.close()
      this.client = null
      this.config = null
    }
  }
}

// Singleton instance
export const tursoClient = new TursoClientManager()
```

---

## Phase 2: Sync Infrastructure

### Step 2.1: Create Sync Manager

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/sync-manager.ts`:

```typescript
/**
 * Sync Manager
 * Orchestrates bidirectional sync between IndexedDB and Turso
 */

import { tursoClient } from '../turso/client'
import { getIndexedDBClient } from '../storage/indexeddb-client'
import { sanitizeConnection, sanitizeQuery } from './sanitizers'
import type { SyncResult, ChangeSet } from './types'

export class SyncManager {
  private syncInterval: NodeJS.Timeout | null = null
  private isSyncing = false
  private lastSyncTimestamp: Date | null = null

  /**
   * Start sync manager
   */
  async start(): Promise<void> {
    console.log('[SyncManager] Starting...')

    // Perform initial sync
    await this.performSync()

    // Set up periodic sync (every 30s)
    this.syncInterval = setInterval(() => {
      this.performSync()
    }, 30000)

    // Set up network listeners
    this.setupNetworkListeners()

    console.log('[SyncManager] Started')
  }

  /**
   * Stop sync manager
   */
  stop(): void {
    if (this.syncInterval) {
      clearInterval(this.syncInterval)
      this.syncInterval = null
    }
    console.log('[SyncManager] Stopped')
  }

  /**
   * Perform full sync cycle
   */
  private async performSync(): Promise<void> {
    if (this.isSyncing || !navigator.onLine) {
      return
    }

    this.isSyncing = true

    try {
      console.log('[SyncManager] Starting sync cycle')

      // Step 1: Upload local changes
      const uploadResult = await this.uploadChanges()
      console.log('[SyncManager] Upload complete:', uploadResult)

      // Step 2: Download remote changes
      const downloadResult = await this.downloadChanges()
      console.log('[SyncManager] Download complete:', downloadResult)

      // Step 3: Update last sync timestamp
      this.lastSyncTimestamp = new Date()

      // Step 4: Broadcast sync complete
      this.broadcastSyncComplete()

    } catch (error) {
      console.error('[SyncManager] Sync error:', error)
    } finally {
      this.isSyncing = false
    }
  }

  /**
   * Upload local changes to Turso
   */
  private async uploadChanges(): Promise<SyncResult> {
    // Implementation from sync-protocol.md
    // See "Upload Flow" section
    return { success: true, uploaded: 0 }
  }

  /**
   * Download remote changes from Turso
   */
  private async downloadChanges(): Promise<SyncResult> {
    // Implementation from sync-protocol.md
    // See "Download Flow" section
    return { success: true, downloaded: 0 }
  }

  /**
   * Force sync now (user-triggered)
   */
  async forceSyncNow(): Promise<void> {
    await this.performSync()
  }

  /**
   * Setup online/offline listeners
   */
  private setupNetworkListeners(): void {
    window.addEventListener('online', () => {
      console.log('[SyncManager] Network online')
      this.performSync()
    })

    window.addEventListener('offline', () => {
      console.log('[SyncManager] Network offline')
    })
  }

  /**
   * Broadcast sync complete to other tabs
   */
  private broadcastSyncComplete(): void {
    window.dispatchEvent(new CustomEvent('sync-complete', {
      detail: { timestamp: this.lastSyncTimestamp }
    }))
  }
}

// Singleton instance
export const syncManager = new SyncManager()
```

### Step 2.2: Create Data Sanitizers

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/sanitizers.ts`:

```typescript
/**
 * Data Sanitizers
 * Remove credentials before uploading to Turso
 */

import type { ConnectionRecord, QueryHistoryRecord } from '@/types/storage'

/**
 * Sanitize connection (remove passwords)
 */
export function sanitizeConnection(
  connection: ConnectionRecord
): Omit<ConnectionRecord, 'password'> {
  const sanitized = { ...connection }

  // Remove password fields
  delete sanitized.password

  // Remove SSH tunnel credentials
  if (sanitized.sshTunnel) {
    delete sanitized.sshTunnel.password
    delete sanitized.sshTunnel.privateKey
  }

  // Sanitize parameters
  if (sanitized.parameters) {
    const params = { ...sanitized.parameters }

    const credentialKeys = [
      'password', 'passwd', 'pwd', 'secret', 'token',
      'api_key', 'apiKey', 'private_key', 'privateKey'
    ]

    for (const key of credentialKeys) {
      delete params[key]
    }

    sanitized.parameters = params
  }

  return sanitized
}

/**
 * Sanitize query text (remove inline credentials)
 */
export function sanitizeQuery(queryText: string): string {
  let sanitized = queryText

  // Remove inline passwords
  sanitized = sanitized.replace(
    /PASSWORD\s+['"][^'"]+['"]/gi,
    "PASSWORD '[REDACTED]'"
  )

  sanitized = sanitized.replace(
    /IDENTIFIED\s+BY\s+['"][^'"]+['"]/gi,
    "IDENTIFIED BY '[REDACTED]'"
  )

  // Remove credentials from connection strings
  sanitized = sanitized.replace(
    /([a-zA-Z0-9_]+):([^@\s]+)@/g,
    '$1:[REDACTED]@'
  )

  return sanitized
}

/**
 * Validate no credentials leaked
 */
export function validateNoCredentials(data: unknown): void {
  const json = JSON.stringify(data)

  const patterns = [
    /password["\s:]+[^,}\]]+/i,
    /secret["\s:]+[^,}\]]+/i,
    /api[-_]?key["\s:]+[^,}\]]+/i,
    /bearer\s+[a-zA-Z0-9_\-\.]+/i
  ]

  for (const pattern of patterns) {
    if (pattern.test(json)) {
      throw new Error(
        `SECURITY: Credential detected in sync data (pattern: ${pattern.source})`
      )
    }
  }
}
```

### Step 2.3: Create Sync Types

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/types.ts`:

```typescript
/**
 * Sync Type Definitions
 */

export type EntityType =
  | 'connection'
  | 'query_history'
  | 'saved_query'
  | 'ai_session'
  | 'ai_message'
  | 'preference'

export type SyncOperation = 'create' | 'update' | 'delete'

export type SyncStatus = 'synced' | 'pending' | 'conflict' | 'error'

export interface ChangeSet {
  entityType: EntityType
  entityId: string
  operation: SyncOperation
  data: any
  timestamp: Date
}

export interface SyncResult {
  success: boolean
  uploaded?: number
  downloaded?: number
  conflicts?: number
  errors?: number
}

export interface Conflict {
  entityType: EntityType
  entityId: string
  localVersion: number
  remoteVersion: number
  localData: any
  remoteData: any
  detectedAt: Date
}

export interface SyncMetadata {
  entityType: EntityType
  entityId: string
  deviceId: string
  localVersion: number
  remoteVersion: number
  syncStatus: SyncStatus
  lastSynced: Date
  lastModified: Date
  conflictData?: any
}
```

---

## Phase 3: Conflict Resolution

### Step 3.1: Implement Conflict Detector

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/conflict-detector.ts`:

```typescript
/**
 * Conflict Detector
 * Detects conflicts between local and remote versions
 */

import { getIndexedDBClient } from '../storage/indexeddb-client'
import type { Conflict, EntityType } from './types'

/**
 * Detect if remote change conflicts with local state
 */
export async function detectConflict(
  entityType: EntityType,
  entityId: string,
  remoteData: any
): Promise<Conflict | null> {
  const db = getIndexedDBClient()

  // Get local record
  const storeName = getStoreName(entityType)
  const local = await db.get(storeName, entityId)

  if (!local) {
    // No local record, no conflict
    return null
  }

  // Get last sync timestamp
  const lastSync = await getLastSyncTimestamp()

  if (!lastSync || local.updated_at <= lastSync) {
    // Local not modified since last sync, no conflict
    return null
  }

  // Both local and remote modified since last sync
  // This is a conflict!
  return {
    entityType,
    entityId,
    localVersion: local.sync_version,
    remoteVersion: remoteData.sync_version,
    localData: local,
    remoteData,
    detectedAt: new Date()
  }
}

function getStoreName(entityType: EntityType): string {
  const mapping = {
    connection: 'connections',
    query_history: 'query_history',
    saved_query: 'saved_queries',
    ai_session: 'ai_sessions',
    ai_message: 'ai_messages',
    preference: 'ui_preferences'
  }
  return mapping[entityType]
}

async function getLastSyncTimestamp(): Promise<Date | null> {
  // Retrieve from localStorage or IndexedDB
  const timestamp = localStorage.getItem('last_sync_timestamp')
  return timestamp ? new Date(timestamp) : null
}
```

### Step 3.2: Implement Conflict Resolver

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/conflict-resolver.ts`:

```typescript
/**
 * Conflict Resolver
 * Resolves conflicts using various strategies
 */

import type { Conflict, EntityType } from './types'

export type ConflictStrategy =
  | 'last-write-wins'
  | 'manual'
  | 'merge'
  | 'keep-both'

/**
 * Resolve conflict
 */
export async function resolveConflict(
  conflict: Conflict
): Promise<any> {
  const strategy = getConflictStrategy(conflict.entityType)

  switch (strategy) {
    case 'last-write-wins':
      return resolveLastWriteWins(conflict)

    case 'keep-both':
      return resolveKeepBoth(conflict)

    case 'manual':
      return resolveManual(conflict)

    case 'merge':
      return resolveMerge(conflict)

    default:
      return resolveLastWriteWins(conflict)
  }
}

/**
 * Get conflict resolution strategy for entity type
 */
function getConflictStrategy(entityType: EntityType): ConflictStrategy {
  const strategies: Record<EntityType, ConflictStrategy> = {
    connection: 'keep-both',
    query_history: 'last-write-wins',
    saved_query: 'keep-both',
    ai_session: 'merge',
    ai_message: 'last-write-wins',
    preference: 'last-write-wins'
  }

  return strategies[entityType]
}

/**
 * Last-Write-Wins strategy
 */
function resolveLastWriteWins(conflict: Conflict): any {
  const { localData, remoteData } = conflict

  // Compare timestamps
  if (remoteData.updated_at > localData.updated_at) {
    return remoteData
  } else if (remoteData.updated_at < localData.updated_at) {
    return localData
  } else {
    // Same timestamp, use version as tie-breaker
    return remoteData.sync_version > localData.sync_version
      ? remoteData
      : localData
  }
}

/**
 * Keep-Both strategy (create duplicate)
 */
async function resolveKeepBoth(conflict: Conflict): Promise<any> {
  const { localData, remoteData } = conflict

  // Create duplicate with suffix
  const duplicate = {
    ...remoteData,
    id: generateId(),
    name: `${remoteData.name} (conflicted copy)`,
    created_at: new Date()
  }

  // Save duplicate locally
  await saveLocalRecord(conflict.entityType, duplicate)

  // Show notification
  showConflictNotification(conflict, 'created duplicate')

  // Keep local version
  return localData
}

/**
 * Manual resolution (prompt user)
 */
async function resolveManual(conflict: Conflict): Promise<any> {
  // Show conflict UI
  return new Promise((resolve) => {
    showConflictDialog(conflict, (resolution) => {
      resolve(resolution)
    })
  })
}

/**
 * Merge strategy
 */
function resolveMerge(conflict: Conflict): any {
  // Simple merge: combine arrays, prefer remote for scalars
  const merged = { ...conflict.remoteData }

  // Merge arrays (combine and dedupe)
  for (const key in conflict.localData) {
    if (Array.isArray(conflict.localData[key])) {
      merged[key] = [
        ...new Set([
          ...(conflict.remoteData[key] || []),
          ...(conflict.localData[key] || [])
        ])
      ]
    }
  }

  return merged
}

// Helper functions
function generateId(): string {
  return crypto.randomUUID()
}

function showConflictNotification(conflict: Conflict, action: string): void {
  console.log(`[Conflict] ${conflict.entityType} ${conflict.entityId}: ${action}`)
}

function showConflictDialog(
  conflict: Conflict,
  callback: (resolution: any) => void
): void {
  // TODO: Implement UI dialog
  callback(conflict.remoteData)
}

async function saveLocalRecord(entityType: EntityType, data: any): Promise<void> {
  // TODO: Save to IndexedDB
}
```

---

## Phase 4: UI Integration

### Step 4.1: Create Sync Status Hook

Create `/Users/jacob_1/projects/sql-studio/frontend/src/hooks/use-sync-status.ts`:

```typescript
/**
 * React Hook for Sync Status
 */

import { useState, useEffect } from 'react'
import { syncManager } from '@/lib/sync/sync-manager'

export interface SyncStatus {
  isSyncing: boolean
  lastSynced: Date | null
  pendingCount: number
  errorCount: number
  conflictCount: number
}

export function useSyncStatus(): SyncStatus & {
  forceSyncNow: () => Promise<void>
} {
  const [status, setStatus] = useState<SyncStatus>({
    isSyncing: false,
    lastSynced: null,
    pendingCount: 0,
    errorCount: 0,
    conflictCount: 0
  })

  useEffect(() => {
    // Subscribe to sync events
    const handleSyncStart = () => {
      setStatus(prev => ({ ...prev, isSyncing: true }))
    }

    const handleSyncComplete = (event: CustomEvent) => {
      setStatus(prev => ({
        ...prev,
        isSyncing: false,
        lastSynced: event.detail.timestamp
      }))
    }

    window.addEventListener('sync-start', handleSyncStart)
    window.addEventListener('sync-complete', handleSyncComplete as EventListener)

    return () => {
      window.removeEventListener('sync-start', handleSyncStart)
      window.removeEventListener('sync-complete', handleSyncComplete as EventListener)
    }
  }, [])

  return {
    ...status,
    forceSyncNow: () => syncManager.forceSyncNow()
  }
}
```

### Step 4.2: Create Sync Status Indicator Component

Create `/Users/jacob_1/projects/sql-studio/frontend/src/components/sync-status-indicator.tsx`:

```typescript
/**
 * Sync Status Indicator Component
 */

import { useSyncStatus } from '@/hooks/use-sync-status'
import { formatRelativeTime } from '@/lib/utils'

export function SyncStatusIndicator() {
  const { isSyncing, lastSynced, pendingCount, forceSyncNow } = useSyncStatus()

  return (
    <div className="sync-status">
      {isSyncing && (
        <div className="syncing">
          <Spinner size="sm" />
          <span>Syncing...</span>
        </div>
      )}

      {!isSyncing && lastSynced && (
        <div className="synced">
          <CheckIcon size={16} />
          <span>Synced {formatRelativeTime(lastSynced)}</span>
        </div>
      )}

      {pendingCount > 0 && (
        <div className="pending">
          <span>{pendingCount} pending changes</span>
        </div>
      )}

      <button onClick={forceSyncNow}>
        Sync Now
      </button>
    </div>
  )
}
```

### Step 4.3: Initial Sync Modal

Create `/Users/jacob_1/projects/sql-studio/frontend/src/components/initial-sync-modal.tsx`:

```typescript
/**
 * Initial Sync Modal
 * Shown when user first upgrades to Individual tier
 */

import { useState, useEffect } from 'react'
import { performInitialSync } from '@/lib/sync/initial-sync'

interface InitialSyncModalProps {
  isOpen: boolean
  onComplete: () => void
}

export function InitialSyncModal({ isOpen, onComplete }: InitialSyncModalProps) {
  const [progress, setProgress] = useState(0)
  const [status, setStatus] = useState('Preparing...')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isOpen) return

    performInitialSync({
      onProgress: (current, total) => {
        setProgress(Math.round((current / total) * 100))
        setStatus(`Syncing ${current} of ${total} items...`)
      }
    })
      .then(() => {
        setStatus('Sync complete!')
        setTimeout(onComplete, 1000)
      })
      .catch((err) => {
        setError(err.message)
      })
  }, [isOpen])

  if (!isOpen) return null

  return (
    <div className="modal">
      <div className="modal-content">
        <h2>Syncing Your Data</h2>

        {!error && (
          <>
            <p>{status}</p>
            <ProgressBar value={progress} max={100} />
          </>
        )}

        {error && (
          <>
            <p className="error">{error}</p>
            <button onClick={() => window.location.reload()}>
              Retry
            </button>
          </>
        )}
      </div>
    </div>
  )
}
```

---

## Phase 5: Testing

### Step 5.1: Unit Tests

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/__tests__/sanitizers.test.ts`:

```typescript
import { describe, test, expect } from 'vitest'
import { sanitizeConnection, sanitizeQuery } from '../sanitizers'

describe('sanitizeConnection', () => {
  test('removes password field', () => {
    const connection = {
      id: '123',
      name: 'Test DB',
      password: 'secret123'
    }

    const sanitized = sanitizeConnection(connection)

    expect(sanitized.password).toBeUndefined()
  })

  test('removes SSH tunnel credentials', () => {
    const connection = {
      id: '123',
      name: 'Test DB',
      sshTunnel: {
        host: 'jump.example.com',
        password: 'ssh-secret',
        privateKey: 'ssh-rsa AAAAB...'
      }
    }

    const sanitized = sanitizeConnection(connection)

    expect(sanitized.sshTunnel.password).toBeUndefined()
    expect(sanitized.sshTunnel.privateKey).toBeUndefined()
  })
})

describe('sanitizeQuery', () => {
  test('redacts inline passwords', () => {
    const query = "ALTER USER admin PASSWORD 'secret123';"
    const sanitized = sanitizeQuery(query)

    expect(sanitized).toContain('[REDACTED]')
    expect(sanitized).not.toContain('secret123')
  })

  test('redacts connection strings', () => {
    const query = "CONNECT TO postgres://user:pass@host/db;"
    const sanitized = sanitizeQuery(query)

    expect(sanitized).toContain('user:[REDACTED]@host')
    expect(sanitized).not.toContain(':pass@')
  })
})
```

### Step 5.2: Integration Tests

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/__tests__/sync-manager.test.ts`:

```typescript
import { describe, test, expect, beforeEach } from 'vitest'
import { syncManager } from '../sync-manager'
import { setupTestDatabase } from './test-utils'

describe('SyncManager', () => {
  beforeEach(async () => {
    await setupTestDatabase()
  })

  test('uploads local changes', async () => {
    // Create local change
    await createLocalConnection({
      id: '123',
      name: 'Test Connection'
    })

    // Trigger sync
    await syncManager.forceSyncNow()

    // Verify uploaded to Turso
    const remote = await fetchRemoteConnection('123')
    expect(remote).toBeDefined()
    expect(remote.name).toBe('Test Connection')
  })

  test('downloads remote changes', async () => {
    // Create remote change
    await createRemoteConnection({
      id: '456',
      name: 'Remote Connection'
    })

    // Trigger sync
    await syncManager.forceSyncNow()

    // Verify downloaded to IndexedDB
    const local = await fetchLocalConnection('456')
    expect(local).toBeDefined()
    expect(local.name).toBe('Remote Connection')
  })

  test('detects and resolves conflicts', async () => {
    // Create conflict scenario
    // TODO: Implement conflict test
  })
})
```

### Step 5.3: E2E Tests

Create `/Users/jacob_1/projects/sql-studio/frontend/e2e/sync.spec.ts`:

```typescript
import { test, expect } from '@playwright/test'

test('syncs data across devices', async ({ context }) => {
  // Open two tabs (simulating two devices)
  const page1 = await context.newPage()
  const page2 = await context.newPage()

  // Device 1: Create connection
  await page1.goto('/')
  await page1.click('[data-testid="add-connection"]')
  await page1.fill('[data-testid="connection-name"]', 'Test DB')
  await page1.click('[data-testid="save-connection"]')

  // Wait for sync
  await page1.waitForSelector('[data-testid="sync-complete"]')

  // Device 2: Should see new connection
  await page2.goto('/')
  await page2.waitForSelector('[data-testid="sync-complete"]')

  const connections = await page2.locator('[data-testid="connection-list"] li')
  await expect(connections).toContainText('Test DB')
})
```

---

## Phase 6: Monitoring

### Step 6.1: Set Up Turso Usage Monitoring

Create `/Users/jacob_1/projects/sql-studio/frontend/src/lib/monitoring/turso-monitor.ts`:

```typescript
/**
 * Turso Usage Monitor
 * Tracks usage and costs
 */

interface TursoUsage {
  storage_gb: number
  rows_read: number
  rows_written: number
  estimated_cost: number
}

/**
 * Fetch Turso usage metrics
 */
export async function fetchTursoUsage(): Promise<TursoUsage> {
  const response = await fetch(
    `https://api.turso.tech/v1/organizations/${orgId}/usage`,
    {
      headers: {
        Authorization: `Bearer ${apiToken}`
      }
    }
  )

  const data = await response.json()

  return {
    storage_gb: data.storage.usage / 1024 / 1024 / 1024,
    rows_read: data.rows_read.usage,
    rows_written: data.rows_written.usage,
    estimated_cost: data.estimated_cost
  }
}

/**
 * Check if approaching free tier limits
 */
export async function checkFreeTierStatus(): Promise<{
  storagePercent: number
  readsPercent: number
  writesPercent: number
  warnings: string[]
}> {
  const usage = await fetchTursoUsage()

  const FREE_TIER = {
    storage_gb: 9,
    rows_read: 25_000_000_000,
    rows_written: 25_000_000
  }

  const storagePercent = (usage.storage_gb / FREE_TIER.storage_gb) * 100
  const readsPercent = (usage.rows_read / FREE_TIER.rows_read) * 100
  const writesPercent = (usage.rows_written / FREE_TIER.rows_written) * 100

  const warnings: string[] = []

  if (storagePercent > 80) {
    warnings.push(`Storage at ${storagePercent.toFixed(0)}% of free tier`)
  }

  if (writesPercent > 80) {
    warnings.push(`Writes at ${writesPercent.toFixed(0)}% of free tier`)
  }

  return {
    storagePercent,
    readsPercent,
    writesPercent,
    warnings
  }
}
```

### Step 6.2: Set Up Alerts

Add to monitoring system:

```typescript
// Alert when costs exceed threshold
if (costPerUser > 0.10) {
  sendAlert({
    severity: 'warning',
    message: `Turso cost per user exceeded $0.10: $${costPerUser}`
  })
}

// Alert when approaching free tier
const status = await checkFreeTierStatus()
if (status.warnings.length > 0) {
  sendAlert({
    severity: 'info',
    message: status.warnings.join(', ')
  })
}
```

---

## Phase 7: Launch

### Launch Checklist

#### Pre-Launch

- [ ] Schema deployed to Turso template database
- [ ] Database provisioning API tested
- [ ] Sync manager implemented and tested
- [ ] Conflict resolution tested
- [ ] Security audit completed (no credential leaks)
- [ ] Cost monitoring set up
- [ ] E2E tests passing
- [ ] Documentation complete

#### Launch Day

- [ ] Deploy frontend with Turso integration
- [ ] Enable Individual tier in tier-store
- [ ] Monitor initial signups
- [ ] Monitor sync performance
- [ ] Monitor costs
- [ ] Be ready to scale up Turso plan if needed

#### Post-Launch (Week 1)

- [ ] Collect user feedback
- [ ] Monitor error rates
- [ ] Monitor conflict resolution rates
- [ ] Optimize sync performance
- [ ] Review cost metrics

---

## Troubleshooting

### Common Issues

**Issue: Sync is slow**
```typescript
// Solution: Increase batch size
const BATCH_SIZE = 100 // Increase from 50

// Solution: Use parallel uploads
await Promise.all([
  uploadConnections(),
  uploadQueries(),
  uploadPreferences()
])
```

**Issue: Conflicts happening too often**
```typescript
// Solution: Increase sync frequency
const SYNC_INTERVAL = 15000 // Decrease from 30s to 15s
```

**Issue: Credential leaked in sync data**
```typescript
// Solution: Enhanced validation
import { validateNoCredentials } from '@/lib/sync/sanitizers'

// Before every upload
validateNoCredentials(data)
```

**Issue: Turso quota exceeded**
```typescript
// Solution: Implement aggressive retention
await cleanupOldRecords({
  query_history_days: 30, // Reduce from 90
  ai_sessions_days: 90    // Reduce from 180
})
```

---

## Next Steps

1. **Implement Phase 1** - Set up Turso organization and schema
2. **Implement Phase 2** - Build sync infrastructure
3. **Test thoroughly** - Unit, integration, and E2E tests
4. **Launch to beta users** - Monitor closely
5. **Iterate based on feedback** - Optimize and improve

---

## Resources

- [Turso Documentation](https://docs.turso.tech/)
- [LibSQL Client API](https://github.com/libsql/libsql-client-ts)
- [Howlerops Schema](./turso-schema.sql)
- [Sync Protocol](./sync-protocol.md)
- [Cost Analysis](./turso-cost-analysis.md)
