/**
 * Store Registry for Multi-Tab Synchronization
 *
 * Central registry for managing which Zustand stores should be synchronized
 * across browser tabs. Defines sync policies and exclusion rules for each store.
 *
 * Synchronized Stores:
 * - connection-store: Connection list and metadata (excludes passwords)
 * - query-store: Query tabs and recent queries
 * - tier-store: Tier information and license status
 *
 * Excluded from Sync:
 * - Passwords and credentials (sessionStorage only)
 * - Large data (query results, full AI transcripts)
 * - Transient UI state (hover, focus, scroll position)
 *
 * Usage:
 * ```typescript
 * // Register stores on app initialization
 * initializeSyncRegistry()
 *
 * // Get sync configuration for a store
 * const config = getStoreConfig('connection-store')
 * ```
 */

import { getBroadcastSync } from './broadcast-sync'
import { getPasswordTransferManager } from './password-transfer'
import { getTabLifecycleManager } from './tab-lifecycle'

/**
 * Store synchronization configuration
 */
export interface StoreSyncConfig {
  /**
   * Store name (must match Zustand store name)
   */
  name: string

  /**
   * Whether sync is enabled for this store
   */
  enabled: boolean

  /**
   * Fields to exclude from synchronization
   */
  excludeFields: string[]

  /**
   * Debounce time in milliseconds
   */
  debounceMs: number

  /**
   * Description of what this store contains
   */
  description: string

  /**
   * Priority level (higher = more important)
   */
  priority: 'low' | 'medium' | 'high'
}

/**
 * Registry of all syncable stores
 */
const STORE_CONFIGS: Record<string, StoreSyncConfig> = {
  'connection-store': {
    name: 'connection-store',
    enabled: true,
    excludeFields: [
      'password',
      'sessionId',
      'isConnecting',
      'sshTunnel.password',
      'sshTunnel.privateKey',
      'activeConnection' // Don't sync active connection across tabs
    ],
    debounceMs: 100,
    description: 'Database connections (excludes passwords and session data)',
    priority: 'high'
  },

  'query-store': {
    name: 'query-store',
    enabled: true,
    excludeFields: [
      'results', // Don't sync large query results
      'isExecuting', // Transient state
      'activeTabId' // Each tab has its own active tab
    ],
    debounceMs: 200,
    description: 'Query tabs and history (excludes results and execution state)',
    priority: 'medium'
  },

  'tier-store': {
    name: 'tier-store',
    enabled: true,
    excludeFields: [
      'isInitialized', // Transient state
      '__broadcastSync' // Internal metadata
    ],
    debounceMs: 500,
    description: 'Tier information and license status',
    priority: 'high'
  },

  'ai-memory-store': {
    name: 'ai-memory-store',
    enabled: true,
    excludeFields: [
      'fullTranscripts', // Large data
      'isLoading', // Transient state
      '__broadcastSync'
    ],
    debounceMs: 500,
    description: 'AI session metadata (excludes full transcripts)',
    priority: 'low'
  }
}

/**
 * Stores that should never be synchronized
 */
const EXCLUDED_STORES = [
  'secrets-store', // Contains sensitive data
  'schema-store', // Large data, database-specific
  'ai-query-agent-store', // Session-specific
  'ai-store', // Session-specific with large data
  'upgrade-prompt-store' // UI state
]

/**
 * Store registry class
 */
class StoreRegistry {
  private configs: Map<string, StoreSyncConfig> = new Map()
  private initialized = false

  constructor() {
    // Load default configurations
    Object.values(STORE_CONFIGS).forEach(config => {
      this.configs.set(config.name, config)
    })
  }

  /**
   * Initialize the registry and start synchronization
   */
  initialize() {
    if (this.initialized) {
      return
    }

    // Initialize broadcast channel
    getBroadcastSync()

    // Initialize tab lifecycle
    getTabLifecycleManager()

    // Initialize password transfer
    getPasswordTransferManager()

    // Set up logout broadcast handler
    this.setupLogoutHandler()

    // Set up connection added handler
    this.setupConnectionHandler()

    // Set up tier changed handler
    this.setupTierHandler()

    this.initialized = true

    console.log('[StoreRegistry] Initialized with', this.configs.size, 'stores')
  }

  /**
   * Set up logout broadcast handler
   */
  private setupLogoutHandler() {
    const broadcast = getBroadcastSync()

    broadcast.on('logout', (message) => {
      console.log('[StoreRegistry] Logout broadcast received from', message.senderId)

      // Trigger logout in this tab
      // This will be handled by the auth system
      window.dispatchEvent(new CustomEvent('multi-tab-logout', {
        detail: { senderId: message.senderId }
      }))
    })
  }

  /**
   * Set up connection added handler
   */
  private setupConnectionHandler() {
    const broadcast = getBroadcastSync()

    broadcast.on('connection-added', (message) => {
      console.log('[StoreRegistry] Connection added:', message.connectionId)

      // Notify connection store to refresh
      window.dispatchEvent(new CustomEvent('connection-added-broadcast', {
        detail: {
          connectionId: message.connectionId,
          senderId: message.senderId
        }
      }))
    })
  }

  /**
   * Set up tier changed handler
   */
  private setupTierHandler() {
    const broadcast = getBroadcastSync()

    broadcast.on('tier-changed', (message) => {
      console.log('[StoreRegistry] Tier changed to:', message.newTier)

      // Notify tier store to refresh
      window.dispatchEvent(new CustomEvent('tier-changed-broadcast', {
        detail: {
          newTier: message.newTier,
          senderId: message.senderId
        }
      }))
    })
  }

  /**
   * Get configuration for a store
   */
  getConfig(storeName: string): StoreSyncConfig | null {
    return this.configs.get(storeName) || null
  }

  /**
   * Check if a store should be synchronized
   */
  shouldSync(storeName: string): boolean {
    // Check if explicitly excluded
    if (EXCLUDED_STORES.includes(storeName)) {
      return false
    }

    // Check if has configuration and is enabled
    const config = this.configs.get(storeName)
    return config?.enabled || false
  }

  /**
   * Get all store configurations
   */
  getAllConfigs(): StoreSyncConfig[] {
    return Array.from(this.configs.values())
  }

  /**
   * Register a new store configuration
   */
  registerStore(config: StoreSyncConfig): void {
    this.configs.set(config.name, config)
    console.log('[StoreRegistry] Registered store:', config.name)
  }

  /**
   * Unregister a store
   */
  unregisterStore(storeName: string): void {
    this.configs.delete(storeName)
    console.log('[StoreRegistry] Unregistered store:', storeName)
  }

  /**
   * Update store configuration
   */
  updateConfig(storeName: string, updates: Partial<StoreSyncConfig>): void {
    const existing = this.configs.get(storeName)
    if (existing) {
      this.configs.set(storeName, { ...existing, ...updates })
      console.log('[StoreRegistry] Updated config for store:', storeName)
    }
  }

  /**
   * Broadcast a logout event to all tabs
   */
  broadcastLogout(): void {
    const broadcast = getBroadcastSync()
    broadcast.send({
      type: 'logout',
      senderId: broadcast.getTabId(),
      timestamp: Date.now()
    })
  }

  /**
   * Broadcast a connection added event
   */
  broadcastConnectionAdded(connectionId: string): void {
    const broadcast = getBroadcastSync()
    broadcast.send({
      type: 'connection-added',
      connectionId,
      senderId: broadcast.getTabId(),
      timestamp: Date.now()
    })
  }

  /**
   * Broadcast a tier changed event
   */
  broadcastTierChanged(newTier: string): void {
    const broadcast = getBroadcastSync()
    broadcast.send({
      type: 'tier-changed',
      newTier,
      senderId: broadcast.getTabId(),
      timestamp: Date.now()
    })
  }

  /**
   * Broadcast sync complete event
   */
  broadcastSyncComplete(): void {
    const broadcast = getBroadcastSync()
    broadcast.send({
      type: 'sync-complete',
      timestamp: Date.now(),
      senderId: broadcast.getTabId()
    })
  }

  /**
   * Check if registry is initialized
   */
  isInitialized(): boolean {
    return this.initialized
  }
}

/**
 * Singleton instance
 */
let storeRegistryInstance: StoreRegistry | null = null

/**
 * Get or create the singleton StoreRegistry instance
 */
export function getStoreRegistry(): StoreRegistry {
  if (!storeRegistryInstance) {
    storeRegistryInstance = new StoreRegistry()
  }
  return storeRegistryInstance
}

/**
 * Initialize the store registry
 * Call this in your main App component
 */
export function initializeSyncRegistry(): void {
  const registry = getStoreRegistry()
  registry.initialize()
}

/**
 * Get store configuration
 */
export function getStoreConfig(storeName: string): StoreSyncConfig | null {
  return getStoreRegistry().getConfig(storeName)
}

/**
 * Check if a store should be synchronized
 */
export function shouldSyncStore(storeName: string): boolean {
  return getStoreRegistry().shouldSync(storeName)
}

/**
 * Broadcast logout to all tabs
 */
export function broadcastLogout(): void {
  getStoreRegistry().broadcastLogout()
}

/**
 * Broadcast connection added to all tabs
 */
export function broadcastConnectionAdded(connectionId: string): void {
  getStoreRegistry().broadcastConnectionAdded(connectionId)
}

/**
 * Broadcast tier changed to all tabs
 */
export function broadcastTierChanged(newTier: string): void {
  getStoreRegistry().broadcastTierChanged(newTier)
}
