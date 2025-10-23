/**
 * Secure Storage Utility
 *
 * Provides secure storage for sensitive credentials like passwords and API keys.
 * Uses sessionStorage for session-only persistence and provides encryption helpers.
 */

export interface SecureCredential {
  connectionId: string
  password?: string
  sshPassword?: string
  sshPrivateKey?: string
}

const STORAGE_KEY = 'sql-studio-secure-credentials'

/**
 * Secure credential storage using sessionStorage
 * Credentials are cleared when the browser tab is closed
 */
class SecureCredentialStorage {
  private credentials: Map<string, SecureCredential> = new Map()
  private initialized = false

  constructor() {
    this.loadFromSession()
  }

  private loadFromSession() {
    if (typeof window === 'undefined' || this.initialized) return

    try {
      const stored = sessionStorage.getItem(STORAGE_KEY)
      if (stored) {
        const parsed = JSON.parse(stored) as SecureCredential[]
        parsed.forEach(cred => {
          this.credentials.set(cred.connectionId, cred)
        })
      }
      this.initialized = true
    } catch (error) {
      console.error('[SecureStorage] Failed to load credentials:', error)
    }
  }

  private saveToSession() {
    if (typeof window === 'undefined') return

    try {
      const credArray = Array.from(this.credentials.values())
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(credArray))
    } catch (error) {
      console.error('[SecureStorage] Failed to save credentials:', error)
    }
  }

  /**
   * Store credentials for a connection
   */
  setCredentials(connectionId: string, credentials: Omit<SecureCredential, 'connectionId'>) {
    this.credentials.set(connectionId, {
      connectionId,
      ...credentials
    })
    this.saveToSession()
  }

  /**
   * Get credentials for a connection
   */
  getCredentials(connectionId: string): SecureCredential | null {
    return this.credentials.get(connectionId) || null
  }

  /**
   * Get password for a connection
   */
  getPassword(connectionId: string): string | undefined {
    return this.credentials.get(connectionId)?.password
  }

  /**
   * Remove credentials for a connection
   */
  removeCredentials(connectionId: string) {
    this.credentials.delete(connectionId)
    this.saveToSession()
  }

  /**
   * Clear all credentials
   */
  clearAll() {
    this.credentials.clear()
    if (typeof window !== 'undefined') {
      sessionStorage.removeItem(STORAGE_KEY)
    }
  }

  /**
   * Check if credentials exist for a connection
   */
  hasCredentials(connectionId: string): boolean {
    return this.credentials.has(connectionId)
  }

  /**
   * Get all connection IDs with stored credentials
   */
  getAllConnectionIds(): string[] {
    return Array.from(this.credentials.keys())
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
 * Helper to migrate passwords from localStorage to sessionStorage
 * This should be called once during app initialization
 */
export function migratePasswordsFromLocalStorage() {
  if (typeof window === 'undefined') return

  try {
    // Check if connection-store has passwords in localStorage
    const connectionStoreData = localStorage.getItem('connection-store')
    if (!connectionStoreData) return

    const parsed = JSON.parse(connectionStoreData)
    if (!parsed.state?.connections) return

    const storage = getSecureStorage()
    let migrated = 0

    // Extract and migrate passwords
    parsed.state.connections.forEach((conn: any) => {
      if (conn.password || conn.sshTunnel?.password || conn.sshTunnel?.privateKey) {
        storage.setCredentials(conn.id, {
          password: conn.password,
          sshPassword: conn.sshTunnel?.password,
          sshPrivateKey: conn.sshTunnel?.privateKey
        })
        migrated++
      }
    })

    if (migrated > 0) {
      console.log(`[SecureStorage] Migrated ${migrated} connection passwords to sessionStorage`)

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
