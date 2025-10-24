# Frontend Integration Guide

## Overview

This guide shows how to integrate SQL Studio's sync system into your frontend application (React, Electron, etc.).

## TypeScript Client

### 1. Define Types

```typescript
// types/sync.ts

export type SyncItemType = 'connection' | 'saved_query' | 'query_history';
export type SyncAction = 'create' | 'update' | 'delete';
export type ConflictResolutionStrategy = 'last_write_wins' | 'keep_both' | 'user_choice';

export interface ConnectionTemplate {
  id: string;
  name: string;
  type: 'postgres' | 'mysql' | 'sqlite' | 'mongodb' | string;
  host?: string;
  port?: number;
  database: string;
  username?: string;
  use_ssh?: boolean;
  ssh_host?: string;
  ssh_port?: number;
  ssh_user?: string;
  color?: string;
  icon?: string;
  metadata?: Record<string, string>;
  created_at: string;
  updated_at: string;
  sync_version: number;
}

export interface SavedQuery {
  id: string;
  name: string;
  description?: string;
  query: string;
  connection_id?: string;
  tags?: string[];
  favorite: boolean;
  metadata?: Record<string, string>;
  created_at: string;
  updated_at: string;
  sync_version: number;
}

export interface SyncChange {
  id: string;
  item_type: SyncItemType;
  item_id: string;
  action: SyncAction;
  data: any;
  updated_at: string;
  sync_version: number;
  device_id: string;
}

export interface Conflict {
  id: string;
  item_type: SyncItemType;
  item_id: string;
  local_version: {
    data: any;
    updated_at: string;
    sync_version: number;
    device_id: string;
  };
  remote_version: {
    data: any;
    updated_at: string;
    sync_version: number;
  };
  detected_at: string;
}
```

### 2. Create Sync Client

```typescript
// services/syncClient.ts

import { v4 as uuidv4 } from 'uuid';

export class SyncClient {
  private baseURL: string;
  private deviceId: string;
  private token: string | null = null;

  constructor(baseURL: string = 'http://localhost:8500') {
    this.baseURL = baseURL;
    this.deviceId = this.getOrCreateDeviceId();
  }

  setToken(token: string) {
    this.token = token;
  }

  private getOrCreateDeviceId(): string {
    let deviceId = localStorage.getItem('device_id');
    if (!deviceId) {
      deviceId = uuidv4();
      localStorage.setItem('device_id', deviceId);
    }
    return deviceId;
  }

  private async fetch(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<Response> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Request failed');
    }

    return response;
  }

  async upload(changes: SyncChange[]) {
    const lastSyncAt = localStorage.getItem('last_sync_at') || new Date(0).toISOString();

    const response = await this.fetch('/api/sync/upload', {
      method: 'POST',
      body: JSON.stringify({
        device_id: this.deviceId,
        last_sync_at: lastSyncAt,
        changes,
      }),
    });

    const data = await response.json();

    // Update last sync time
    if (data.success) {
      localStorage.setItem('last_sync_at', data.synced_at);
    }

    return data;
  }

  async download(since?: string) {
    const sinceTime = since || localStorage.getItem('last_sync_at') ||
                      new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();

    const response = await this.fetch(
      `/api/sync/download?since=${encodeURIComponent(sinceTime)}&device_id=${this.deviceId}`
    );

    const data = await response.json();

    // Update last sync time
    localStorage.setItem('last_sync_at', data.sync_timestamp);

    return data;
  }

  async listConflicts(): Promise<Conflict[]> {
    const response = await this.fetch('/api/sync/conflicts');
    const data = await response.json();
    return data.conflicts;
  }

  async resolveConflict(
    conflictId: string,
    strategy: ConflictResolutionStrategy,
    chosenVersion?: 'local' | 'remote'
  ) {
    return this.fetch(`/api/sync/conflicts/${conflictId}/resolve`, {
      method: 'POST',
      body: JSON.stringify({
        strategy,
        chosen_version: chosenVersion,
      }),
    });
  }
}
```

### 3. Create Sync Manager

```typescript
// services/syncManager.ts

import { SyncClient } from './syncClient';
import { LocalDatabase } from './localDatabase'; // Your local storage implementation

export class SyncManager {
  private client: SyncClient;
  private localDB: LocalDatabase;
  private syncInterval: NodeJS.Timeout | null = null;
  private isSyncing = false;

  constructor(client: SyncClient, localDB: LocalDatabase) {
    this.client = client;
    this.localDB = localDB;
  }

  // Start automatic sync every 5 minutes
  startAutoSync(intervalMs: number = 5 * 60 * 1000) {
    this.stopAutoSync();
    this.syncInterval = setInterval(() => {
      this.performSync();
    }, intervalMs);
  }

  stopAutoSync() {
    if (this.syncInterval) {
      clearInterval(this.syncInterval);
      this.syncInterval = null;
    }
  }

  async performSync() {
    if (this.isSyncing) {
      console.log('Sync already in progress, skipping...');
      return;
    }

    this.isSyncing = true;

    try {
      // 1. Download remote changes
      const remoteChanges = await this.client.download();
      await this.applyRemoteChanges(remoteChanges);

      // 2. Collect local changes
      const localChanges = await this.collectLocalChanges();

      // 3. Upload local changes
      if (localChanges.length > 0) {
        const uploadResult = await this.client.upload(localChanges);

        // 4. Handle conflicts
        if (uploadResult.conflicts && uploadResult.conflicts.length > 0) {
          await this.handleConflicts(uploadResult.conflicts);
        }

        // 5. Mark local changes as synced
        await this.markChangesSynced(localChanges);
      }

      console.log('Sync completed successfully');
    } catch (error) {
      console.error('Sync failed:', error);
      // Optionally: queue for retry
    } finally {
      this.isSyncing = false;
    }
  }

  private async applyRemoteChanges(remoteChanges: any) {
    // Apply connections
    for (const conn of remoteChanges.connections) {
      await this.localDB.saveConnection(conn);
    }

    // Apply saved queries
    for (const query of remoteChanges.saved_queries) {
      await this.localDB.saveQuery(query);
    }

    // Apply query history
    for (const history of remoteChanges.query_history) {
      await this.localDB.saveHistory(history);
    }
  }

  private async collectLocalChanges(): Promise<SyncChange[]> {
    // Get all unsynced changes from local database
    const unsyncedConnections = await this.localDB.getUnsyncedConnections();
    const unsyncedQueries = await this.localDB.getUnsyncedQueries();
    const unsyncedHistory = await this.localDB.getUnsyncedHistory();

    const changes: SyncChange[] = [];

    // Convert to SyncChange format
    for (const conn of unsyncedConnections) {
      changes.push({
        id: uuidv4(),
        item_type: 'connection',
        item_id: conn.id,
        action: conn.action || 'update',
        data: conn,
        updated_at: conn.updated_at,
        sync_version: conn.sync_version,
        device_id: this.client['deviceId'],
      });
    }

    // Similar for queries and history...

    return changes;
  }

  private async handleConflicts(conflicts: Conflict[]) {
    // Option 1: Auto-resolve with last_write_wins
    for (const conflict of conflicts) {
      await this.client.resolveConflict(conflict.id, 'last_write_wins');
    }

    // Option 2: Show UI to user (emit event)
    // this.emit('conflicts', conflicts);
  }

  private async markChangesSynced(changes: SyncChange[]) {
    for (const change of changes) {
      await this.localDB.markAsSynced(change.item_type, change.item_id);
    }
  }
}
```

### 4. React Hook

```typescript
// hooks/useSync.ts

import { useState, useEffect } from 'react';
import { SyncClient } from '../services/syncClient';
import { SyncManager } from '../services/syncManager';
import { LocalDatabase } from '../services/localDatabase';

export function useSync() {
  const [isSyncing, setIsSyncing] = useState(false);
  const [lastSyncAt, setLastSyncAt] = useState<string | null>(null);
  const [conflicts, setConflicts] = useState<Conflict[]>([]);
  const [syncManager] = useState(() => {
    const client = new SyncClient();
    const localDB = new LocalDatabase();
    return new SyncManager(client, localDB);
  });

  useEffect(() => {
    // Load last sync time
    const lastSync = localStorage.getItem('last_sync_at');
    setLastSyncAt(lastSync);

    // Start auto-sync
    syncManager.startAutoSync(5 * 60 * 1000); // 5 minutes

    return () => {
      syncManager.stopAutoSync();
    };
  }, [syncManager]);

  const manualSync = async () => {
    setIsSyncing(true);
    try {
      await syncManager.performSync();
      const newLastSync = localStorage.getItem('last_sync_at');
      setLastSyncAt(newLastSync);

      // Check for conflicts
      const client = syncManager['client'];
      const conflictList = await client.listConflicts();
      setConflicts(conflictList);
    } finally {
      setIsSyncing(false);
    }
  };

  return {
    isSyncing,
    lastSyncAt,
    conflicts,
    manualSync,
  };
}
```

### 5. React Component

```tsx
// components/SyncStatus.tsx

import React from 'react';
import { useSync } from '../hooks/useSync';
import { formatDistanceToNow } from 'date-fns';

export function SyncStatus() {
  const { isSyncing, lastSyncAt, conflicts, manualSync } = useSync();

  return (
    <div className="sync-status">
      <button
        onClick={manualSync}
        disabled={isSyncing}
        className="sync-button"
      >
        {isSyncing ? (
          <>
            <Spinner />
            Syncing...
          </>
        ) : (
          <>
            <RefreshIcon />
            Sync Now
          </>
        )}
      </button>

      {lastSyncAt && (
        <span className="last-sync">
          Last synced {formatDistanceToNow(new Date(lastSyncAt))} ago
        </span>
      )}

      {conflicts.length > 0 && (
        <div className="conflicts-badge">
          {conflicts.length} conflicts
        </div>
      )}
    </div>
  );
}
```

### 6. Conflict Resolution UI

```tsx
// components/ConflictResolver.tsx

import React, { useState } from 'react';
import { Conflict } from '../types/sync';

interface Props {
  conflict: Conflict;
  onResolve: (strategy: string, chosenVersion?: string) => void;
}

export function ConflictResolver({ conflict, onResolve }: Props) {
  const [selectedVersion, setSelectedVersion] = useState<'local' | 'remote'>('local');

  return (
    <div className="conflict-resolver">
      <h3>Conflict Detected</h3>
      <p>Item: {conflict.item_type} - {conflict.item_id}</p>

      <div className="versions">
        <div className="version">
          <h4>Local Version</h4>
          <input
            type="radio"
            checked={selectedVersion === 'local'}
            onChange={() => setSelectedVersion('local')}
          />
          <pre>{JSON.stringify(conflict.local_version.data, null, 2)}</pre>
          <small>Updated: {conflict.local_version.updated_at}</small>
        </div>

        <div className="version">
          <h4>Remote Version</h4>
          <input
            type="radio"
            checked={selectedVersion === 'remote'}
            onChange={() => setSelectedVersion('remote')}
          />
          <pre>{JSON.stringify(conflict.remote_version.data, null, 2)}</pre>
          <small>Updated: {conflict.remote_version.updated_at}</small>
        </div>
      </div>

      <div className="actions">
        <button onClick={() => onResolve('user_choice', selectedVersion)}>
          Use Selected Version
        </button>
        <button onClick={() => onResolve('keep_both')}>
          Keep Both
        </button>
        <button onClick={() => onResolve('last_write_wins')}>
          Auto-Resolve (Latest)
        </button>
      </div>
    </div>
  );
}
```

## Local Storage Strategy

### IndexedDB Schema

```typescript
// services/localDatabase.ts

import Dexie, { Table } from 'dexie';

export interface LocalConnection extends ConnectionTemplate {
  synced: boolean;
  action?: 'create' | 'update' | 'delete';
}

export interface LocalQuery extends SavedQuery {
  synced: boolean;
  action?: 'create' | 'update' | 'delete';
}

class LocalDatabase extends Dexie {
  connections!: Table<LocalConnection>;
  queries!: Table<LocalQuery>;
  history!: Table<QueryHistory>;

  constructor() {
    super('SQLStudioDB');

    this.version(1).stores({
      connections: 'id, user_id, updated_at, synced',
      queries: 'id, user_id, updated_at, synced',
      history: 'id, user_id, executed_at',
    });
  }

  async saveConnection(conn: ConnectionTemplate) {
    return this.connections.put({
      ...conn,
      synced: true,
      action: undefined,
    });
  }

  async updateConnection(id: string, updates: Partial<ConnectionTemplate>) {
    const conn = await this.connections.get(id);
    if (!conn) throw new Error('Connection not found');

    return this.connections.put({
      ...conn,
      ...updates,
      updated_at: new Date().toISOString(),
      sync_version: conn.sync_version + 1,
      synced: false,
      action: 'update',
    });
  }

  async getUnsyncedConnections() {
    return this.connections.where('synced').equals(0).toArray();
  }

  async markAsSynced(type: string, id: string) {
    if (type === 'connection') {
      await this.connections.update(id, { synced: true, action: undefined });
    }
    // Similar for other types...
  }
}

export const localDB = new LocalDatabase();
```

## Offline Support

### 1. Detect Online Status

```typescript
// hooks/useOnlineStatus.ts

import { useState, useEffect } from 'react';

export function useOnlineStatus() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return isOnline;
}
```

### 2. Queue Failed Syncs

```typescript
// services/syncQueue.ts

export class SyncQueue {
  private queue: SyncChange[] = [];

  async add(changes: SyncChange[]) {
    this.queue.push(...changes);
    await this.save();
  }

  async process(client: SyncClient) {
    if (this.queue.length === 0) return;

    try {
      await client.upload(this.queue);
      this.queue = [];
      await this.save();
    } catch (error) {
      console.error('Failed to process queue:', error);
      throw error;
    }
  }

  private async save() {
    localStorage.setItem('sync_queue', JSON.stringify(this.queue));
  }

  static async load(): Promise<SyncQueue> {
    const queue = new SyncQueue();
    const saved = localStorage.getItem('sync_queue');
    if (saved) {
      queue.queue = JSON.parse(saved);
    }
    return queue;
  }
}
```

## Best Practices

### 1. Optimistic Updates

```typescript
async function updateConnection(id: string, updates: Partial<ConnectionTemplate>) {
  // 1. Update UI immediately
  setConnections(prev =>
    prev.map(c => c.id === id ? { ...c, ...updates } : c)
  );

  // 2. Save to local DB
  await localDB.updateConnection(id, updates);

  // 3. Queue for sync (happens in background)
  // Sync manager will handle this automatically
}
```

### 2. Error Handling

```typescript
try {
  await syncManager.performSync();
} catch (error) {
  if (error.message.includes('rate limit')) {
    // Wait and retry
    setTimeout(() => syncManager.performSync(), 60000);
  } else if (error.message.includes('unauthorized')) {
    // Refresh token
    await refreshAuthToken();
    await syncManager.performSync();
  } else {
    // Show error to user
    showErrorNotification('Sync failed. Will retry automatically.');
  }
}
```

### 3. Conflict Prevention

```typescript
// Always increment sync_version when updating
function updateItem(item: any) {
  return {
    ...item,
    updated_at: new Date().toISOString(),
    sync_version: item.sync_version + 1,
  };
}
```

## Testing

```typescript
// __tests__/syncClient.test.ts

import { SyncClient } from '../services/syncClient';

describe('SyncClient', () => {
  let client: SyncClient;

  beforeEach(() => {
    client = new SyncClient('http://localhost:8500');
    client.setToken('test-token');
  });

  test('uploads changes successfully', async () => {
    const changes = [
      {
        id: '1',
        item_type: 'connection',
        item_id: 'conn-1',
        action: 'create',
        data: { name: 'Test DB' },
        updated_at: new Date().toISOString(),
        sync_version: 1,
        device_id: 'test-device',
      },
    ];

    const result = await client.upload(changes);
    expect(result.success).toBe(true);
  });

  test('handles conflicts', async () => {
    // Test conflict resolution...
  });
});
```

## Example: Complete Integration

```typescript
// App.tsx

import React, { useEffect } from 'react';
import { SyncClient } from './services/syncClient';
import { SyncManager } from './services/syncManager';
import { localDB } from './services/localDatabase';
import { SyncStatus } from './components/SyncStatus';

export function App() {
  useEffect(() => {
    // Initialize sync
    const client = new SyncClient();
    const manager = new SyncManager(client, localDB);

    // Set token from auth
    const token = localStorage.getItem('auth_token');
    if (token) {
      client.setToken(token);
      manager.startAutoSync();
    }

    // Initial sync
    manager.performSync();

    return () => {
      manager.stopAutoSync();
    };
  }, []);

  return (
    <div className="app">
      <Header>
        <SyncStatus />
      </Header>
      <Main />
    </div>
  );
}
```

This integration guide provides everything needed to implement sync in your frontend application with proper offline support, conflict resolution, and error handling.
