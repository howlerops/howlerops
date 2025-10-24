/**
 * Credential Migration Utility
 *
 * Migrates credentials from localStorage to OS keychain via Wails backend.
 * This migration runs once per installation and is non-blocking to app startup.
 *
 * Migration Flow:
 * 1. Check if already migrated
 * 2. Extract credentials from localStorage
 * 3. Store each credential in OS keychain via Wails
 * 4. Mark migration complete
 * 5. Clear localStorage credentials
 *
 * Error Handling:
 * - Individual credential failures don't block migration
 * - Failed migrations keep localStorage intact
 * - Non-blocking - app starts even if migration fails
 */

import { useEffect } from 'react'

// Storage keys
const STORAGE_KEY = 'sql-studio-secure-credentials'
const MIGRATION_FLAG = 'credentials-migrated'
const MIGRATION_VERSION = 'credentials-migration-version'
const CURRENT_MIGRATION_VERSION = '1.0'

/**
 * Interface for credentials stored in localStorage
 */
export interface StoredCredential {
  connectionId: string
  password?: string
  sshPassword?: string
  sshPrivateKey?: string
}

/**
 * Migration result for tracking success/failure
 */
export interface MigrationResult {
  success: boolean
  migratedCount: number
  failedCount: number
  errors: Array<{
    connectionId: string
    error: string
  }>
  skipped: boolean
  reason?: string
}

/**
 * Check if keychain API is available
 * This will be true once the Wails backend exposes keychain functions
 */
function isKeychainAvailable(): boolean {
  return typeof window !== 'undefined' &&
         typeof (window as any).go !== 'undefined' &&
         typeof (window as any).go?.main?.App?.StorePassword === 'function' &&
         typeof (window as any).go?.main?.App?.GetPassword === 'function'
}

/**
 * Store password in OS keychain via Wails backend
 * This is a placeholder for the actual Wails function once implemented
 */
async function storePasswordInKeychain(
  connectionId: string,
  credentialType: 'password' | 'ssh_password' | 'ssh_private_key',
  value: string
): Promise<void> {
  const wailsGo = (window as any).go
  if (wailsGo?.main?.App?.StorePassword) {
    const service = 'sql-studio'
    const account = `${connectionId}-${credentialType}`
    await wailsGo.main.App.StorePassword(service, account, value)
  } else {
    throw new Error('Keychain API not available')
  }
}

/**
 * Main migration function - moves credentials from localStorage to keychain
 */
export async function migrateCredentialsToKeychain(): Promise<MigrationResult> {
  const result: MigrationResult = {
    success: false,
    migratedCount: 0,
    failedCount: 0,
    errors: [],
    skipped: false,
  }

  try {
    // Check if running in browser environment
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') {
      result.skipped = true
      result.reason = 'Not running in browser environment'
      return result
    }

    // Check if keychain API is available
    if (!isKeychainAvailable()) {
      result.skipped = true
      result.reason = 'Keychain API not yet available - keeping localStorage credentials'
      console.log('[CredentialMigration] Keychain API not available, skipping migration')
      return result
    }

    // Check if already migrated
    const migrated = localStorage.getItem(MIGRATION_FLAG)
    const migrationVersion = localStorage.getItem(MIGRATION_VERSION)

    if (migrated === 'true' && migrationVersion === CURRENT_MIGRATION_VERSION) {
      result.skipped = true
      result.reason = 'Already migrated'
      console.log('[CredentialMigration] Already migrated, skipping')
      return result
    }

    // Get credentials from localStorage
    const credentialsJson = localStorage.getItem(STORAGE_KEY)

    if (!credentialsJson) {
      // No credentials to migrate - mark as complete
      localStorage.setItem(MIGRATION_FLAG, 'true')
      localStorage.setItem(MIGRATION_VERSION, CURRENT_MIGRATION_VERSION)
      result.success = true
      result.skipped = true
      result.reason = 'No credentials found'
      console.log('[CredentialMigration] No credentials to migrate')
      return result
    }

    // Parse credentials
    let credentials: StoredCredential[]
    try {
      credentials = JSON.parse(credentialsJson) as StoredCredential[]
    } catch (parseError) {
      result.success = false
      result.errors.push({
        connectionId: 'parse-error',
        error: `Failed to parse credentials: ${parseError}`,
      })
      console.error('[CredentialMigration] Failed to parse credentials:', parseError)
      return result
    }

    console.log(`[CredentialMigration] Starting migration of ${credentials.length} credential(s)`)

    // Handle empty array case
    if (credentials.length === 0) {
      localStorage.setItem(MIGRATION_FLAG, 'true')
      localStorage.setItem(MIGRATION_VERSION, CURRENT_MIGRATION_VERSION)
      localStorage.removeItem(STORAGE_KEY)
      result.success = true
      console.log('[CredentialMigration] No credentials to migrate - marked as complete')
      return result
    }

    // Migrate each credential
    for (const credential of credentials) {
      const { connectionId, password, sshPassword, sshPrivateKey } = credential

      try {
        // Store database password
        if (password) {
          await storePasswordInKeychain(connectionId, 'password', password)
          console.log(`[CredentialMigration] Migrated password for connection: ${connectionId}`)
        }

        // Store SSH password
        if (sshPassword) {
          await storePasswordInKeychain(connectionId, 'ssh_password', sshPassword)
          console.log(`[CredentialMigration] Migrated SSH password for connection: ${connectionId}`)
        }

        // Store SSH private key
        if (sshPrivateKey) {
          await storePasswordInKeychain(connectionId, 'ssh_private_key', sshPrivateKey)
          console.log(`[CredentialMigration] Migrated SSH private key for connection: ${connectionId}`)
        }

        result.migratedCount++
      } catch (error) {
        result.failedCount++
        result.errors.push({
          connectionId,
          error: error instanceof Error ? error.message : String(error),
        })
        console.error(`[CredentialMigration] Failed to migrate credentials for ${connectionId}:`, error)
      }
    }

    // If all migrations succeeded, mark complete and clear localStorage
    if (result.failedCount === 0 && result.migratedCount > 0) {
      localStorage.setItem(MIGRATION_FLAG, 'true')
      localStorage.setItem(MIGRATION_VERSION, CURRENT_MIGRATION_VERSION)
      localStorage.removeItem(STORAGE_KEY)
      result.success = true
      console.log(`[CredentialMigration] Successfully migrated ${result.migratedCount} credential(s) to keychain`)
    } else if (result.migratedCount > 0) {
      // Partial success - keep localStorage until all succeed
      result.success = false
      console.warn(
        `[CredentialMigration] Partial migration: ${result.migratedCount} succeeded, ${result.failedCount} failed. ` +
        'Keeping localStorage credentials until all migrate successfully.'
      )
    } else {
      result.success = false
      console.error('[CredentialMigration] Migration failed - no credentials migrated')
    }

    return result
  } catch (error) {
    result.success = false
    result.errors.push({
      connectionId: 'migration-error',
      error: error instanceof Error ? error.message : String(error),
    })
    console.error('[CredentialMigration] Migration failed with error:', error)
    return result
  }
}

/**
 * Force retry migration - useful for debugging or manual retry
 * Clears migration flag and attempts migration again
 */
export async function retryMigration(): Promise<MigrationResult> {
  if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
    localStorage.removeItem(MIGRATION_FLAG)
    localStorage.removeItem(MIGRATION_VERSION)
  }
  return migrateCredentialsToKeychain()
}

/**
 * Check migration status without performing migration
 */
export function getMigrationStatus(): {
  migrated: boolean
  version: string | null
  hasCredentials: boolean
  keychainAvailable: boolean
} {
  if (typeof window === 'undefined' || typeof localStorage === 'undefined') {
    return {
      migrated: false,
      version: null,
      hasCredentials: false,
      keychainAvailable: false,
    }
  }

  return {
    migrated: localStorage.getItem(MIGRATION_FLAG) === 'true',
    version: localStorage.getItem(MIGRATION_VERSION),
    hasCredentials: localStorage.getItem(STORAGE_KEY) !== null,
    keychainAvailable: isKeychainAvailable(),
  }
}

/**
 * React hook for credential migration
 * Runs migration once on mount - non-blocking to app startup
 */
export function useMigrateCredentials() {
  useEffect(() => {
    // Run migration asynchronously without blocking render
    migrateCredentialsToKeychain()
      .then((result) => {
        if (result.success) {
          console.log('[CredentialMigration] Migration completed successfully')
        } else if (result.skipped) {
          console.log(`[CredentialMigration] Migration skipped: ${result.reason}`)
        } else {
          console.warn('[CredentialMigration] Migration completed with errors:', result)
        }
      })
      .catch((error) => {
        // This shouldn't happen since migrateCredentialsToKeychain catches all errors
        console.error('[CredentialMigration] Unexpected migration error:', error)
      })
  }, [])
}

/**
 * Clear migration flag - useful for testing
 * WARNING: Only use in development/testing
 */
export function clearMigrationFlag(): void {
  if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
    localStorage.removeItem(MIGRATION_FLAG)
    localStorage.removeItem(MIGRATION_VERSION)
    console.log('[CredentialMigration] Migration flag cleared')
  }
}
