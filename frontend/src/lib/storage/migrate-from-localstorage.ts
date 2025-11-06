/**
 * Migration Utility - LocalStorage to IndexedDB
 *
 * Helps migrate existing data from localStorage/Zustand persist
 * to IndexedDB for better performance and storage capacity.
 *
 * @module lib/storage/migrate-from-localstorage
 */

import {
  getConnectionRepository,
  getQueryHistoryRepository,
  getPreferenceRepository,
} from './repositories'

/**
 * Migration result
 */
export interface MigrationResult {
  connections: number
  queryHistory: number
  preferences: number
  errors: string[]
}

/**
 * Migrate connection data from localStorage to IndexedDB
 */
async function migrateConnections(userId: string): Promise<number> {
  try {
    const repo = getConnectionRepository()
    const stored = localStorage.getItem('connection-store')

    if (!stored) {
      return 0
    }

    const data = JSON.parse(stored)
    const connections = data?.state?.connections || []

    let migrated = 0
    for (const conn of connections) {
      try {
        // Check if already exists
        const existing = await repo.get(conn.id)
        if (existing) {
          continue
        }

        // Create connection record
        await repo.create({
          connection_id: conn.id,
          user_id: userId,
          name: conn.name,
          type: mapDatabaseType(conn.type),
          host: conn.host || '',
          port: conn.port || 5432,
          database: conn.database,
          username: conn.username || '',
          ssl_mode: conn.sslMode || 'prefer',  // Default to 'prefer' for better security
          parameters: conn.parameters,
          environment_tags: conn.environments || [],
          last_used_at: conn.lastUsed ? new Date(conn.lastUsed) : new Date(),
        })

        migrated++
      } catch (error) {
        console.error('Failed to migrate connection:', conn.id, error)
      }
    }

    return migrated
  } catch (error) {
    console.error('Failed to migrate connections:', error)
    return 0
  }
}

/**
 * Migrate query history from localStorage to IndexedDB
 */
async function migrateQueryHistory(userId: string): Promise<number> {
  try {
    const repo = getQueryHistoryRepository()
    const stored = localStorage.getItem('query-store')

    if (!stored) {
      return 0
    }

    const data = JSON.parse(stored)
    const tabs = data?.state?.tabs || []

    let migrated = 0

    // Extract unique queries from tabs
    interface LegacyTab {
      content?: string
      connectionId?: string
      lastExecuted?: string
    }
    const uniqueQueries = new Map<string, LegacyTab>()

    tabs.forEach((tab: LegacyTab) => {
      if (tab.content && tab.lastExecuted) {
        const key = `${tab.content}-${tab.connectionId}-${tab.lastExecuted}`
        if (!uniqueQueries.has(key)) {
          uniqueQueries.set(key, tab)
        }
      }
    })

    for (const tab of uniqueQueries.values()) {
      if (!tab.content) {
        continue
      }

      try {
        await repo.create({
          user_id: userId,
          query_text: tab.content,
          connection_id: tab.connectionId || 'unknown',
          execution_time_ms: 0, // Unknown from old data
          row_count: 0, // Unknown from old data
          privacy_mode: 'normal',
          executed_at: tab.lastExecuted ? new Date(tab.lastExecuted) : new Date(),
        })

        migrated++
      } catch (error) {
        console.error('Failed to migrate query:', error)
      }
    }

    return migrated
  } catch (error) {
    console.error('Failed to migrate query history:', error)
    return 0
  }
}

/**
 * Migrate UI preferences from localStorage to IndexedDB
 */
async function migratePreferences(userId: string): Promise<number> {
  try {
    const repo = getPreferenceRepository()

    let migrated = 0

    // Common preference keys in localStorage
    const preferenceKeys = [
      'theme',
      'editor-font-size',
      'editor-tab-size',
      'auto-connect',
      'show-line-numbers',
      'word-wrap',
    ]

    for (const key of preferenceKeys) {
      try {
        const value = localStorage.getItem(key)
        if (value !== null) {
          await repo.setUserPreference(userId, key, value, 'custom')
          migrated++
        }
      } catch (error) {
        console.error('Failed to migrate preference:', key, error)
      }
    }

    return migrated
  } catch (error) {
    console.error('Failed to migrate preferences:', error)
    return 0
  }
}

/**
 * Map old database type to new type
 */
function mapDatabaseType(oldType: string): 'postgres' | 'mysql' | 'sqlite' | 'mssql' | 'oracle' | 'mongodb' {
  switch (oldType.toLowerCase()) {
    case 'postgresql':
    case 'postgres':
      return 'postgres'
    case 'mysql':
    case 'mariadb':
      return 'mysql'
    case 'sqlite':
      return 'sqlite'
    case 'mssql':
    case 'sqlserver':
      return 'mssql'
    case 'oracle':
      return 'oracle'
    case 'mongodb':
    case 'mongo':
      return 'mongodb'
    default:
      return 'postgres'
  }
}

/**
 * Run complete migration from localStorage to IndexedDB
 *
 * @param userId - Current user ID
 * @param clearLocalStorage - Whether to clear localStorage after migration
 */
export async function migrateFromLocalStorage(
  userId: string,
  clearLocalStorage = false
): Promise<MigrationResult> {
  const errors: string[] = []

  console.log('Starting migration from localStorage to IndexedDB...')

  // Migrate connections
  let connections = 0
  try {
    connections = await migrateConnections(userId)
    console.log(`Migrated ${connections} connections`)
  } catch (error) {
    const msg = `Connection migration failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    errors.push(msg)
    console.error(msg)
  }

  // Migrate query history
  let queryHistory = 0
  try {
    queryHistory = await migrateQueryHistory(userId)
    console.log(`Migrated ${queryHistory} query history records`)
  } catch (error) {
    const msg = `Query history migration failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    errors.push(msg)
    console.error(msg)
  }

  // Migrate preferences
  let preferences = 0
  try {
    preferences = await migratePreferences(userId)
    console.log(`Migrated ${preferences} preferences`)
  } catch (error) {
    const msg = `Preference migration failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    errors.push(msg)
    console.error(msg)
  }

  // Clear localStorage if requested
  if (clearLocalStorage && errors.length === 0) {
    try {
      localStorage.removeItem('connection-store')
      localStorage.removeItem('query-store')
      console.log('Cleared localStorage')
    } catch (error) {
      console.warn('Failed to clear localStorage:', error)
    }
  }

  console.log('Migration complete:', { connections, queryHistory, preferences, errors })

  return {
    connections,
    queryHistory,
    preferences,
    errors,
  }
}

/**
 * Check if migration is needed
 */
export function needsMigration(): boolean {
  return (
    localStorage.getItem('connection-store') !== null ||
    localStorage.getItem('query-store') !== null
  )
}

/**
 * Get migration status
 */
export function getMigrationStatus(): {
  needed: boolean
  hasConnections: boolean
  hasQueryHistory: boolean
} {
  return {
    needed: needsMigration(),
    hasConnections: localStorage.getItem('connection-store') !== null,
    hasQueryHistory: localStorage.getItem('query-store') !== null,
  }
}
