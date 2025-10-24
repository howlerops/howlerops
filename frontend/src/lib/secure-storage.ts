/**
 * Secure Storage Utility
 *
 * Provides secure storage for sensitive credentials like passwords and API keys.
 * Uses Wails keychain API for persistent storage across app restarts.
 *
 * Features:
 * - OS-level keychain integration (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux)
 * - In-memory cache for performance optimization
 * - Async operations with proper error handling
 * - Graceful fallback to in-memory only if keychain unavailable
 */

import * as App from '../../wailsjs/go/main/App'

export interface SecureCredential {
  connectionId: string
  password?: string
  sshPassword?: string
  sshPrivateKey?: string
}

/**
 * Secure credential storage using OS keychain via Wails
 * Credentials persist across app restarts securely in the OS keychain
 */
class SecureCredentialStorage {
  private credentials: Map<string, SecureCredential> = new Map()
  private keychainAvailable = true

  /**
   * Store credentials for a connection in keychain and cache
   */
  async setCredentials(
    connectionId: string,
    credentials: Omit<SecureCredential, 'connectionId'>
  ): Promise<void> {
    const fullCredentials: SecureCredential = {
      connectionId,
      ...credentials,
    }

    // Update in-memory cache immediately
    this.credentials.set(connectionId, fullCredentials)

    // Persist to keychain if available
    if (this.keychainAvailable) {
      try {
        // Store as JSON in keychain with prefixed key
        const key = `sql-studio-credentials-${connectionId}`
        const value = JSON.stringify(fullCredentials)

        // Check if App.StorePassword exists (type-safe check)
        if (typeof (App as any).StorePassword === 'function') {
          await (App as any).StorePassword(key, value)
        } else {
          console.warn(
            '[SecureStorage] Keychain API (App.StorePassword) not available yet. Credentials stored in-memory only for this session.'
          )
          this.keychainAvailable = false
        }
      } catch (error) {
        console.error('[SecureStorage] Failed to store credentials in keychain:', error)
        console.warn('[SecureStorage] Falling back to in-memory storage for this session')
        this.keychainAvailable = false
        // Don't throw - credentials are still in cache for this session
      }
    }
  }

  /**
   * Get credentials for a connection from cache or keychain
   */
  async getCredentials(connectionId: string): Promise<SecureCredential | null> {
    // Check in-memory cache first
    const cached = this.credentials.get(connectionId)
    if (cached) {
      return cached
    }

    // Try to load from keychain if available
    if (this.keychainAvailable) {
      try {
        const key = `sql-studio-credentials-${connectionId}`

        // Check if App.GetPassword exists (type-safe check)
        if (typeof (App as any).GetPassword === 'function') {
          const value = await (App as any).GetPassword(key)

          if (value) {
            const credentials = JSON.parse(value) as SecureCredential
            // Update cache
            this.credentials.set(connectionId, credentials)
            return credentials
          }
        } else {
          console.warn(
            '[SecureStorage] Keychain API (App.GetPassword) not available yet. Only in-memory credentials accessible.'
          )
          this.keychainAvailable = false
        }
      } catch (error) {
        // If error is "not found", that's expected - just return null
        const errorMsg = String(error)
        if (!errorMsg.includes('not found') && !errorMsg.includes('NotFound')) {
          console.error('[SecureStorage] Failed to retrieve credentials from keychain:', error)
        }
        // Continue - credentials might not exist, which is fine
      }
    }

    return null
  }

  /**
   * Get just the password for a connection (convenience method)
   */
  async getPassword(connectionId: string): Promise<string | undefined> {
    const credentials = await this.getCredentials(connectionId)
    return credentials?.password
  }

  /**
   * Remove credentials for a connection from keychain and cache
   */
  async removeCredentials(connectionId: string): Promise<void> {
    // Remove from in-memory cache immediately
    this.credentials.delete(connectionId)

    // Remove from keychain if available
    if (this.keychainAvailable) {
      try {
        const key = `sql-studio-credentials-${connectionId}`

        // Check if App.DeletePassword exists (type-safe check)
        if (typeof (App as any).DeletePassword === 'function') {
          await (App as any).DeletePassword(key)
        } else {
          console.warn(
            '[SecureStorage] Keychain API (App.DeletePassword) not available yet. Credentials removed from memory only.'
          )
          this.keychainAvailable = false
        }
      } catch (error) {
        // If error is "not found", that's fine - credential was already gone
        const errorMsg = String(error)
        if (!errorMsg.includes('not found') && !errorMsg.includes('NotFound')) {
          console.error('[SecureStorage] Failed to delete credentials from keychain:', error)
        }
        // Don't throw - credentials are removed from cache anyway
      }
    }
  }

  /**
   * Clear all credentials from cache
   * Note: Keychain items must be cleared individually via removeCredentials()
   */
  async clearAll(): Promise<void> {
    // Get all connection IDs before clearing
    const connectionIds = Array.from(this.credentials.keys())

    // Clear in-memory cache
    this.credentials.clear()

    // Remove each from keychain
    if (this.keychainAvailable && connectionIds.length > 0) {
      await Promise.allSettled(
        connectionIds.map(id => this.removeCredentials(id))
      )
    }
  }

  /**
   * Check if credentials exist for a connection (cache only)
   * For definitive check, use getCredentials() which checks keychain too
   */
  hasCredentials(connectionId: string): boolean {
    return this.credentials.has(connectionId)
  }

  /**
   * Get all connection IDs with stored credentials (cache only)
   * Note: This only returns IDs from in-memory cache
   */
  getAllConnectionIds(): string[] {
    return Array.from(this.credentials.keys())
  }

  /**
   * Preload credentials from keychain into cache
   * Useful for warming up the cache at app startup if connection IDs are known
   */
  async preloadCredentials(connectionIds: string[]): Promise<void> {
    await Promise.allSettled(
      connectionIds.map(id => this.getCredentials(id))
    )
  }
}

// Singleton instance
let storageInstance: SecureCredentialStorage | null = null

export function getSecureStorage(): SecureCredentialStorage {
  if (!storageInstance) {
    storageInstance = new SecureCredentialStorage()
  }
  return storageInstance
}

/**
 * Hook for accessing secure storage in React components
 */
export function useSecureStorage() {
  return getSecureStorage()
}

/**
 * Helper to migrate passwords from localStorage to keychain
 * This should be called once during app initialization
 *
 * NOTE: Migration implementation handled by separate migration utility
 * This is a placeholder for the migration hook
 */
export async function migratePasswordsFromLocalStorage(): Promise<void> {
  if (typeof window === 'undefined') return

  try {
    // Check if connection-store has passwords in localStorage
    const connectionStoreData = localStorage.getItem('connection-store')
    if (!connectionStoreData) return

    const parsed = JSON.parse(connectionStoreData)
    if (!parsed.state?.connections) return

    const storage = getSecureStorage()
    let migrated = 0

    // Extract and migrate passwords to keychain
    const migrations = parsed.state.connections.map(async (conn: any) => {
      if (conn.password || conn.sshTunnel?.password || conn.sshTunnel?.privateKey) {
        await storage.setCredentials(conn.id, {
          password: conn.password,
          sshPassword: conn.sshTunnel?.password,
          sshPrivateKey: conn.sshTunnel?.privateKey,
        })
        migrated++
      }
    })

    await Promise.all(migrations)

    if (migrated > 0) {
      console.log(`[SecureStorage] Migrated ${migrated} connection passwords to keychain`)

      // Remove passwords from the localStorage data
      parsed.state.connections = parsed.state.connections.map((conn: any) => {
        const { password, sshTunnel, ...rest } = conn
        if (sshTunnel) {
          const { password: sshPassword, privateKey, ...sshRest } = sshTunnel // eslint-disable-line @typescript-eslint/no-unused-vars
          return { ...rest, sshTunnel: sshRest }
        }
        return rest
      })

      // Save cleaned data back to localStorage
      localStorage.setItem('connection-store', JSON.stringify(parsed))
    }
  } catch (error) {
    console.error('[SecureStorage] Failed to migrate passwords:', error)
  }
}
