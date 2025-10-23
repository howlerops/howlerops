/**
 * Tab Lifecycle Manager
 *
 * Tracks active browser tabs, manages heartbeats, and implements primary tab election.
 * Provides cleanup for stale tabs and coordinated leadership across tabs.
 *
 * Features:
 * - Unique tab ID generation
 * - Heartbeat mechanism (10-second interval)
 * - Stale tab detection and cleanup (30 seconds)
 * - Primary tab election (oldest active tab)
 * - Tab close detection
 *
 * Usage:
 * ```typescript
 * const lifecycle = new TabLifecycleManager()
 *
 * lifecycle.onTabsChanged((tabs) => {
 *   console.log('Active tabs:', tabs)
 * })
 *
 * lifecycle.onPrimaryChanged((isPrimary) => {
 *   if (isPrimary) {
 *     // This tab is now the primary tab
 *   }
 * })
 *
 * lifecycle.start()
 * ```
 */

import { getBroadcastSync } from './broadcast-sync'

/**
 * Tab information
 */
export interface TabInfo {
  tabId: string
  lastHeartbeat: number
  isPrimary: boolean
}

/**
 * Callback types
 */
type TabsChangedCallback = (tabs: Map<string, TabInfo>) => void
type PrimaryChangedCallback = (isPrimary: boolean, primaryTabId: string | null) => void

/**
 * Configuration options
 */
interface TabLifecycleOptions {
  /**
   * Heartbeat interval in milliseconds
   * @default 10000 (10 seconds)
   */
  heartbeatInterval?: number

  /**
   * Stale tab timeout in milliseconds
   * Tabs are considered stale if no heartbeat received within this time
   * @default 30000 (30 seconds)
   */
  staleTimeout?: number

  /**
   * Whether to automatically start on creation
   * @default false
   */
  autoStart?: boolean
}

const DEFAULT_OPTIONS: Required<TabLifecycleOptions> = {
  heartbeatInterval: 10000, // 10 seconds
  staleTimeout: 30000, // 30 seconds
  autoStart: false
}

/**
 * Tab Lifecycle Manager
 */
export class TabLifecycleManager {
  private tabId: string
  private tabs: Map<string, TabInfo> = new Map()
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private cleanupTimer: ReturnType<typeof setInterval> | null = null
  private isPrimary = false
  private isStarted = false
  private options: Required<TabLifecycleOptions>

  // Callbacks
  private tabsChangedCallbacks: Set<TabsChangedCallback> = new Set()
  private primaryChangedCallbacks: Set<PrimaryChangedCallback> = new Set()

  constructor(options: TabLifecycleOptions = {}) {
    this.options = { ...DEFAULT_OPTIONS, ...options }
    this.tabId = getBroadcastSync().getTabId()

    // Initialize this tab
    this.tabs.set(this.tabId, {
      tabId: this.tabId,
      lastHeartbeat: Date.now(),
      isPrimary: true // Initially assume primary until we hear from others
    })

    this.setupListeners()

    if (this.options.autoStart) {
      this.start()
    }
  }

  /**
   * Set up broadcast channel listeners
   */
  private setupListeners() {
    const broadcast = getBroadcastSync()

    // Listen for heartbeats from other tabs
    broadcast.on('tab-alive', (message) => {
      if (message.tabId === this.tabId) {
        return // Ignore own heartbeats
      }

      const existingTab = this.tabs.get(message.tabId)

      this.tabs.set(message.tabId, {
        tabId: message.tabId,
        lastHeartbeat: message.timestamp,
        isPrimary: message.isPrimary || false
      })

      // If this is a new tab, check primary status
      if (!existingTab) {
        this.electPrimaryTab()
        this.notifyTabsChanged()
      }
    })

    // Listen for tab closed messages
    broadcast.on('tab-closed', (message) => {
      if (message.tabId === this.tabId) {
        return // Ignore own close messages
      }

      const wasPresent = this.tabs.has(message.tabId)

      if (wasPresent) {
        this.tabs.delete(message.tabId)
        this.electPrimaryTab()
        this.notifyTabsChanged()
      }
    })
  }

  /**
   * Start the lifecycle manager
   */
  start() {
    if (this.isStarted) {
      return
    }

    this.isStarted = true

    // Send initial heartbeat to announce presence
    this.sendHeartbeat()

    // Start heartbeat timer
    this.heartbeatTimer = setInterval(() => {
      this.sendHeartbeat()
    }, this.options.heartbeatInterval)

    // Start cleanup timer
    this.cleanupTimer = setInterval(() => {
      this.cleanupStaleTabs()
    }, this.options.heartbeatInterval)

    console.log(`[TabLifecycle] Started for tab ${this.tabId}`)
  }

  /**
   * Stop the lifecycle manager
   */
  stop() {
    if (!this.isStarted) {
      return
    }

    this.isStarted = false

    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }

    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer)
      this.cleanupTimer = null
    }

    // Send tab closed message
    const broadcast = getBroadcastSync()
    broadcast.send({
      type: 'tab-closed',
      tabId: this.tabId,
      timestamp: Date.now()
    })

    console.log(`[TabLifecycle] Stopped for tab ${this.tabId}`)
  }

  /**
   * Send a heartbeat to other tabs
   */
  private sendHeartbeat() {
    const now = Date.now()

    // Update own heartbeat
    this.tabs.set(this.tabId, {
      tabId: this.tabId,
      lastHeartbeat: now,
      isPrimary: this.isPrimary
    })

    // Broadcast heartbeat
    const broadcast = getBroadcastSync()
    broadcast.send({
      type: 'tab-alive',
      tabId: this.tabId,
      timestamp: now,
      isPrimary: this.isPrimary
    })
  }

  /**
   * Clean up stale tabs that haven't sent a heartbeat
   */
  private cleanupStaleTabs() {
    const now = Date.now()
    let hasChanges = false

    for (const [tabId, info] of this.tabs.entries()) {
      // Don't clean up this tab
      if (tabId === this.tabId) {
        continue
      }

      // Check if tab is stale
      if (now - info.lastHeartbeat > this.options.staleTimeout) {
        console.log(`[TabLifecycle] Cleaning up stale tab ${tabId}`)
        this.tabs.delete(tabId)
        hasChanges = true
      }
    }

    if (hasChanges) {
      this.electPrimaryTab()
      this.notifyTabsChanged()
    }
  }

  /**
   * Elect a primary tab (oldest active tab)
   */
  private electPrimaryTab() {
    if (this.tabs.size === 0) {
      return
    }

    // Find the oldest tab (lowest timestamp)
    let oldestTabId: string | null = null
    let oldestTimestamp = Infinity

    for (const [tabId, info] of this.tabs.entries()) {
      // Use the timestamp from the tab ID (first 13 characters after 'tab-')
      const timestamp = info.lastHeartbeat

      if (timestamp < oldestTimestamp) {
        oldestTimestamp = timestamp
        oldestTabId = tabId
      }
    }

    // Check if primary status changed
    const wasPrimary = this.isPrimary
    this.isPrimary = oldestTabId === this.tabId

    // Update primary status in tabs map
    for (const [tabId, info] of this.tabs.entries()) {
      info.isPrimary = tabId === oldestTabId
    }

    if (wasPrimary !== this.isPrimary) {
      console.log(`[TabLifecycle] Primary tab changed: ${this.isPrimary ? 'This tab is now primary' : 'This tab is no longer primary'}`)
      this.notifyPrimaryChanged()

      // Send updated heartbeat with new primary status
      this.sendHeartbeat()
    }
  }

  /**
   * Get the current tab ID
   */
  getTabId(): string {
    return this.tabId
  }

  /**
   * Check if this tab is the primary tab
   */
  getIsPrimary(): boolean {
    return this.isPrimary
  }

  /**
   * Get all active tabs
   */
  getTabs(): Map<string, TabInfo> {
    return new Map(this.tabs)
  }

  /**
   * Get the number of active tabs
   */
  getTabCount(): number {
    return this.tabs.size
  }

  /**
   * Get the primary tab ID
   */
  getPrimaryTabId(): string | null {
    for (const [tabId, info] of this.tabs.entries()) {
      if (info.isPrimary) {
        return tabId
      }
    }
    return null
  }

  /**
   * Register a callback for tabs changed events
   */
  onTabsChanged(callback: TabsChangedCallback): () => void {
    this.tabsChangedCallbacks.add(callback)

    // Call immediately with current tabs
    callback(this.getTabs())

    // Return unsubscribe function
    return () => {
      this.tabsChangedCallbacks.delete(callback)
    }
  }

  /**
   * Register a callback for primary changed events
   */
  onPrimaryChanged(callback: PrimaryChangedCallback): () => void {
    this.primaryChangedCallbacks.add(callback)

    // Call immediately with current primary status
    callback(this.isPrimary, this.getPrimaryTabId())

    // Return unsubscribe function
    return () => {
      this.primaryChangedCallbacks.delete(callback)
    }
  }

  /**
   * Notify all callbacks of tabs changed
   */
  private notifyTabsChanged() {
    const tabs = this.getTabs()
    this.tabsChangedCallbacks.forEach(callback => {
      try {
        callback(tabs)
      } catch (error) {
        console.error('[TabLifecycle] Error in tabs changed callback:', error)
      }
    })
  }

  /**
   * Notify all callbacks of primary changed
   */
  private notifyPrimaryChanged() {
    const primaryTabId = this.getPrimaryTabId()
    this.primaryChangedCallbacks.forEach(callback => {
      try {
        callback(this.isPrimary, primaryTabId)
      } catch (error) {
        console.error('[TabLifecycle] Error in primary changed callback:', error)
      }
    })
  }

  /**
   * Clean up resources
   */
  destroy() {
    this.stop()
    this.tabsChangedCallbacks.clear()
    this.primaryChangedCallbacks.clear()
    this.tabs.clear()
  }
}

/**
 * Singleton instance
 */
let tabLifecycleInstance: TabLifecycleManager | null = null

/**
 * Get or create the singleton TabLifecycleManager instance
 */
export function getTabLifecycleManager(): TabLifecycleManager {
  if (!tabLifecycleInstance) {
    tabLifecycleInstance = new TabLifecycleManager({ autoStart: true })
  }
  return tabLifecycleInstance
}

/**
 * Clean up the TabLifecycleManager instance
 */
export function destroyTabLifecycleManager() {
  if (tabLifecycleInstance) {
    tabLifecycleInstance.destroy()
    tabLifecycleInstance = null
  }
}
