/**
 * Storage Migration: IndexedDB to SQLite
 *
 * This module handles migrating user data from WebKit IndexedDB to SQLite.
 * WebKit IndexedDB is origin-bound and data is lost when the origin changes
 * (e.g., on app updates). SQLite at ~/.howlerops/local.db is persistent.
 *
 * Migration Flow:
 * 1. Check SQLite migration status via Wails binding
 * 2. If SQLite has data or migration complete: skip (use SQLite)
 * 3. If SQLite empty: try to read from IndexedDB and import
 * 4. Mark migration complete in SQLite
 *
 * This runs once at app startup and is non-blocking.
 */

import { useEffect, useState } from 'react'

import { STORE_NAMES } from '@/types/storage'

import { getIndexedDBClient } from './indexeddb-client'

// Wails v3 bindings - import dynamically to avoid build issues during SSR
let AppBindings: typeof import('../../../bindings/github.com/jbeck018/howlerops/app') | null = null

interface MigrationStatus {
  sqlite_has_data: boolean
  migration_done: boolean
  connection_count: number
  query_count: number
  history_count: number
  preferences_count: number
}

interface ImportResult {
  imported: number
  skipped: number
  errors: string[]
}

interface MigrationResult {
  success: boolean
  connectionsImported: number
  queriesImported: number
  historyImported: number
  errors: string[]
  skipped: boolean
  reason?: string
}

/**
 * Load Wails v3 bindings dynamically
 */
async function loadAppBindings(): Promise<typeof import('../../../bindings/github.com/jbeck018/howlerops/app') | null> {
  if (AppBindings) return AppBindings
  try {
    AppBindings = await import('../../../bindings/github.com/jbeck018/howlerops/app')
    return AppBindings
  } catch (error) {
    console.debug('[StorageMigration] Failed to load Wails bindings:', error)
    return null
  }
}

/**
 * Check if SQLite storage API is available
 */
async function isSQLiteAvailable(): Promise<boolean> {
  if (typeof window === 'undefined') return false

  const App = await loadAppBindings()
  if (!App) return false

  return typeof App.StorageMigrationStatus === 'function' &&
         typeof App.StorageImportConnections === 'function'
}

/**
 * Get migration status from SQLite via Wails
 */
async function getMigrationStatus(): Promise<MigrationStatus | null> {
  const App = await loadAppBindings()
  if (!App || typeof App.StorageMigrationStatus !== 'function') {
    return null
  }

  try {
    return await App.StorageMigrationStatus()
  } catch (error) {
    console.error('[StorageMigration] Failed to get migration status:', error)
    return null
  }
}

/**
 * Read all connections from IndexedDB
 */
async function readIndexedDBConnections(): Promise<unknown[]> {
  try {
    const client = getIndexedDBClient()
    const connections = await client.getAll(STORE_NAMES.CONNECTIONS)
    return connections || []
  } catch (error) {
    console.warn('[StorageMigration] Failed to read connections from IndexedDB:', error)
    return []
  }
}

/**
 * Read all saved queries from IndexedDB
 */
async function readIndexedDBQueries(): Promise<unknown[]> {
  try {
    const client = getIndexedDBClient()
    const queries = await client.getAll(STORE_NAMES.SAVED_QUERIES)
    return queries || []
  } catch (error) {
    console.warn('[StorageMigration] Failed to read queries from IndexedDB:', error)
    return []
  }
}

/**
 * Read query history from IndexedDB
 */
async function readIndexedDBHistory(): Promise<unknown[]> {
  try {
    const client = getIndexedDBClient()
    const history = await client.getAll(STORE_NAMES.QUERY_HISTORY)
    return history || []
  } catch (error) {
    console.warn('[StorageMigration] Failed to read history from IndexedDB:', error)
    return []
  }
}

/**
 * Read UI preferences from IndexedDB
 */
async function readIndexedDBPreferences(): Promise<Record<string, unknown>> {
  try {
    const client = getIndexedDBClient()
    const prefs = await client.getAll(STORE_NAMES.UI_PREFERENCES)

    // Convert array of preference objects to a map
    const prefMap: Record<string, unknown> = {}
    for (const pref of prefs || []) {
      if (pref && typeof pref === 'object' && 'id' in pref) {
        const id = (pref as { id: string }).id
        prefMap[id] = pref
      }
    }
    return prefMap
  } catch (error) {
    console.warn('[StorageMigration] Failed to read preferences from IndexedDB:', error)
    return {}
  }
}

/**
 * Main migration function
 */
export async function migrateIndexedDBToSQLite(): Promise<MigrationResult> {
  const result: MigrationResult = {
    success: false,
    connectionsImported: 0,
    queriesImported: 0,
    historyImported: 0,
    errors: [],
    skipped: false,
  }

  try {
    // Check if SQLite API is available
    const sqliteAvailable = await isSQLiteAvailable()
    if (!sqliteAvailable) {
      result.skipped = true
      result.reason = 'SQLite API not available - Wails bindings not loaded'
      console.log('[StorageMigration] SQLite API not available, skipping migration')
      return result
    }

    // Check migration status
    const status = await getMigrationStatus()
    if (!status) {
      result.skipped = true
      result.reason = 'Could not get migration status'
      console.log('[StorageMigration] Could not get migration status, skipping')
      return result
    }

    // If migration already done, skip
    if (status.migration_done) {
      result.skipped = true
      result.reason = 'Migration already completed'
      result.success = true
      console.log('[StorageMigration] Migration already completed, using SQLite')
      return result
    }

    // NOTE: We intentionally do NOT skip migration just because SQLite has data.
    // This allows partial migration recovery - if migration failed midway,
    // we can retry and the backend will handle duplicates via ID checking.

    const App = await loadAppBindings()
    if (!App) {
      result.skipped = true
      result.reason = 'Wails bindings not available'
      return result
    }

    console.log('[StorageMigration] Starting migration from IndexedDB to SQLite...')

    // Read all data from IndexedDB
    const [connections, queries, history, preferences] = await Promise.all([
      readIndexedDBConnections(),
      readIndexedDBQueries(),
      readIndexedDBHistory(),
      readIndexedDBPreferences(),
    ])

    // Check if there's any data to migrate
    const hasData = connections.length > 0 || queries.length > 0 ||
                    history.length > 0 || Object.keys(preferences).length > 0

    if (!hasData) {
      // No data to migrate - mark as complete
      try {
        await App.StorageCompleteMigration()
      } catch (error) {
        console.warn('[StorageMigration] Failed to mark migration complete:', error)
      }
      result.success = true
      result.skipped = true
      result.reason = 'No data to migrate'
      console.log('[StorageMigration] No data in IndexedDB to migrate')
      return result
    }

    console.log(`[StorageMigration] Found data to migrate: ${connections.length} connections, ${queries.length} queries, ${history.length} history entries`)

    // Import connections
    if (connections.length > 0) {
      try {
        const connectionsJSON = JSON.stringify(connections)
        const importResult = await App.StorageImportConnections(connectionsJSON) as ImportResult
        result.connectionsImported = importResult.imported
        if (importResult.errors && importResult.errors.length > 0) {
          result.errors.push(...importResult.errors.map(e => `Connection: ${e}`))
        }
        console.log(`[StorageMigration] Imported ${importResult.imported} connections (${importResult.skipped} skipped)`)
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error)
        result.errors.push(`Failed to import connections: ${errorMsg}`)
        console.error('[StorageMigration] Failed to import connections:', error)
      }
    }

    // Import queries
    if (queries.length > 0) {
      try {
        const queriesJSON = JSON.stringify(queries)
        const importResult = await App.StorageImportQueries(queriesJSON) as ImportResult
        result.queriesImported = importResult.imported
        if (importResult.errors && importResult.errors.length > 0) {
          result.errors.push(...importResult.errors.map(e => `Query: ${e}`))
        }
        console.log(`[StorageMigration] Imported ${importResult.imported} queries (${importResult.skipped} skipped)`)
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error)
        result.errors.push(`Failed to import queries: ${errorMsg}`)
        console.error('[StorageMigration] Failed to import queries:', error)
      }
    }

    // Import history
    if (history.length > 0) {
      try {
        const historyJSON = JSON.stringify(history)
        const importResult = await App.StorageImportHistory(historyJSON) as ImportResult
        result.historyImported = importResult.imported
        if (importResult.errors && importResult.errors.length > 0) {
          result.errors.push(...importResult.errors.map(e => `History: ${e}`))
        }
        console.log(`[StorageMigration] Imported ${importResult.imported} history entries`)
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error)
        result.errors.push(`Failed to import history: ${errorMsg}`)
        console.error('[StorageMigration] Failed to import history:', error)
      }
    }

    // Import preferences
    if (Object.keys(preferences).length > 0) {
      try {
        const preferencesJSON = JSON.stringify(preferences)
        await App.StorageImportPreferences(preferencesJSON)
        console.log(`[StorageMigration] Imported ${Object.keys(preferences).length} preferences`)
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error)
        result.errors.push(`Failed to import preferences: ${errorMsg}`)
        console.error('[StorageMigration] Failed to import preferences:', error)
      }
    }

    // Mark migration complete if we imported anything
    const imported = result.connectionsImported + result.queriesImported + result.historyImported
    if (imported > 0 || result.errors.length === 0) {
      try {
        await App.StorageCompleteMigration()
        result.success = true
        console.log('[StorageMigration] Migration completed successfully')
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error)
        result.errors.push(`Failed to mark migration complete: ${errorMsg}`)
        console.error('[StorageMigration] Failed to mark migration complete:', error)
      }
    }

    return result
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error)
    result.errors.push(`Unexpected error: ${errorMsg}`)
    console.error('[StorageMigration] Migration failed:', error)
    return result
  }
}

/**
 * React hook for storage migration
 * Runs migration once on mount - non-blocking to app startup
 */
export function useMigrateToSQLite() {
  const [migrationResult, setMigrationResult] = useState<MigrationResult | null>(null)
  const [isMigrating, setIsMigrating] = useState(false)

  useEffect(() => {
    setIsMigrating(true)

    migrateIndexedDBToSQLite()
      .then((result) => {
        setMigrationResult(result)
        if (result.success) {
          if (result.skipped) {
            console.log(`[StorageMigration] ${result.reason}`)
          } else {
            console.log('[StorageMigration] Migration completed:', {
              connections: result.connectionsImported,
              queries: result.queriesImported,
              history: result.historyImported,
            })
          }
        } else {
          console.warn('[StorageMigration] Migration completed with errors:', result.errors)
        }
      })
      .catch((error) => {
        console.error('[StorageMigration] Unexpected migration error:', error)
        setMigrationResult({
          success: false,
          connectionsImported: 0,
          queriesImported: 0,
          historyImported: 0,
          errors: [error instanceof Error ? error.message : String(error)],
          skipped: false,
        })
      })
      .finally(() => {
        setIsMigrating(false)
      })
  }, [])

  return { migrationResult, isMigrating }
}

/**
 * Force retry migration - useful for debugging or manual retry
 */
export async function retryMigration(): Promise<MigrationResult> {
  // Clear migration flag in SQLite
  const App = await loadAppBindings()
  if (App && typeof App.SQLiteSetSetting === 'function') {
    try {
      await App.SQLiteSetSetting('migration_from_indexeddb_complete', 'false')
    } catch (error) {
      console.warn('[StorageMigration] Failed to clear migration flag:', error)
    }
  }

  return migrateIndexedDBToSQLite()
}
