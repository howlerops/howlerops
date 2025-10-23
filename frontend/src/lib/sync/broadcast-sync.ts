/**
 * BroadcastChannel Wrapper for Multi-Tab Synchronization
 *
 * Provides type-safe BroadcastChannel communication between browser tabs.
 * Prevents infinite loops, manages tab lifecycle, and implements retry logic.
 *
 * Browser Support:
 * - Chrome 54+, Firefox 38+, Safari 15.4+, Edge 79+
 *
 * Usage:
 * ```typescript
 * const channel = new BroadcastSync('sql-studio')
 *
 * channel.on('store-update', (message) => {
 *   if (message.senderId !== channel.getTabId()) {
 *     applyUpdate(message.storeName, message.patch)
 *   }
 * })
 *
 * channel.send({ type: 'store-update', storeName: 'connections', patch: {...}, senderId: channel.getTabId() })
 * ```
 */

/**
 * Message types for cross-tab communication
 */
export type BroadcastMessage =
  | { type: 'store-update', storeName: string, patch: any, senderId: string, timestamp: number }
  | { type: 'logout', senderId: string, timestamp: number }
  | { type: 'sync-complete', timestamp: number, senderId: string }
  | { type: 'connection-added', connectionId: string, senderId: string, timestamp: number }
  | { type: 'tier-changed', newTier: string, senderId: string, timestamp: number }
  | { type: 'password-share-request', connectionId: string, requesterId: string, timestamp: number }
  | { type: 'password-share-response', connectionId: string, encryptedPassword: string, key: string, iv: string, senderId: string, timestamp: number }
  | { type: 'tab-alive', tabId: string, timestamp: number, isPrimary?: boolean }
  | { type: 'tab-closed', tabId: string, timestamp: number }
  | { type: 'request-password-share', connectionIds: string[], requesterId: string, timestamp: number }

/**
 * Message handler function type
 */
export type BroadcastMessageHandler<T extends BroadcastMessage['type']> = (
  message: Extract<BroadcastMessage, { type: T }>
) => void | Promise<void>

/**
 * Retry configuration for failed messages
 */
interface RetryConfig {
  maxRetries: number
  retryDelay: number
  backoffMultiplier: number
}

const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  retryDelay: 100, // ms
  backoffMultiplier: 2
}

/**
 * BroadcastChannel wrapper with type safety and lifecycle management
 */
export class BroadcastSync {
  private channel: BroadcastChannel | null = null
  private tabId: string
  private handlers: Map<BroadcastMessage['type'], Set<BroadcastMessageHandler<any>>> = new Map()
  private messageQueue: Array<{ message: BroadcastMessage, retries: number }> = []
  private retryConfig: RetryConfig
  private isConnected = false
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5

  constructor(
    private channelName: string,
    retryConfig: Partial<RetryConfig> = {}
  ) {
    this.tabId = this.generateTabId()
    this.retryConfig = { ...DEFAULT_RETRY_CONFIG, ...retryConfig }
    this.initialize()
  }

  /**
   * Initialize the broadcast channel
   */
  private initialize() {
    if (typeof BroadcastChannel === 'undefined') {
      console.warn('[BroadcastSync] BroadcastChannel API not available in this browser')
      return
    }

    try {
      this.channel = new BroadcastChannel(this.channelName)
      this.isConnected = true
      this.reconnectAttempts = 0

      this.channel.addEventListener('message', this.handleMessage.bind(this))
      this.channel.addEventListener('messageerror', this.handleMessageError.bind(this))

      // Listen for when the page is about to unload
      window.addEventListener('beforeunload', this.handleBeforeUnload.bind(this))

      console.log(`[BroadcastSync] Initialized channel "${this.channelName}" for tab ${this.tabId}`)
    } catch (error) {
      console.error('[BroadcastSync] Failed to initialize BroadcastChannel:', error)
      this.handleConnectionError()
    }
  }

  /**
   * Handle incoming messages from other tabs
   */
  private handleMessage(event: MessageEvent<BroadcastMessage>) {
    const message = event.data

    // Ignore messages from this tab (prevent infinite loops)
    if (message.senderId === this.tabId || (message as any).tabId === this.tabId) {
      return
    }

    // Validate message structure
    if (!message || typeof message !== 'object' || !message.type) {
      console.warn('[BroadcastSync] Received invalid message:', message)
      return
    }

    // Get handlers for this message type
    const handlers = this.handlers.get(message.type)
    if (!handlers || handlers.size === 0) {
      return
    }

    // Execute all handlers for this message type
    handlers.forEach(async (handler) => {
      try {
        await handler(message)
      } catch (error) {
        console.error(`[BroadcastSync] Handler error for message type "${message.type}":`, error)
      }
    })
  }

  /**
   * Handle message deserialization errors
   */
  private handleMessageError(event: MessageEvent) {
    console.error('[BroadcastSync] Message error:', event)
  }

  /**
   * Handle connection errors and attempt reconnection
   */
  private handleConnectionError() {
    this.isConnected = false

    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      const delay = Math.min(5000, 1000 * Math.pow(2, this.reconnectAttempts - 1))

      console.log(`[BroadcastSync] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

      setTimeout(() => {
        this.initialize()
      }, delay)
    } else {
      console.error('[BroadcastSync] Max reconnection attempts reached')
    }
  }

  /**
   * Send message before tab closes
   */
  private handleBeforeUnload() {
    this.send({
      type: 'tab-closed',
      tabId: this.tabId,
      timestamp: Date.now()
    })
  }

  /**
   * Register a handler for a specific message type
   */
  on<T extends BroadcastMessage['type']>(
    type: T,
    handler: BroadcastMessageHandler<T>
  ): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set())
    }

    this.handlers.get(type)!.add(handler)

    // Return unsubscribe function
    return () => {
      const handlers = this.handlers.get(type)
      if (handlers) {
        handlers.delete(handler)
        if (handlers.size === 0) {
          this.handlers.delete(type)
        }
      }
    }
  }

  /**
   * Remove a handler for a specific message type
   */
  off<T extends BroadcastMessage['type']>(
    type: T,
    handler: BroadcastMessageHandler<T>
  ): void {
    const handlers = this.handlers.get(type)
    if (handlers) {
      handlers.delete(handler)
      if (handlers.size === 0) {
        this.handlers.delete(type)
      }
    }
  }

  /**
   * Send a message to all other tabs
   */
  send(message: BroadcastMessage, retry = true): boolean {
    if (!this.channel || !this.isConnected) {
      if (retry) {
        this.queueMessage(message)
      }
      return false
    }

    try {
      // Add timestamp if not present
      const messageWithTimestamp = {
        ...message,
        timestamp: message.timestamp || Date.now(),
        senderId: message.senderId || this.tabId
      }

      this.channel.postMessage(messageWithTimestamp)
      return true
    } catch (error) {
      console.error('[BroadcastSync] Failed to send message:', error)

      if (retry) {
        this.queueMessage(message)
      }

      return false
    }
  }

  /**
   * Queue a message for retry
   */
  private queueMessage(message: BroadcastMessage) {
    this.messageQueue.push({ message, retries: 0 })
    this.processQueue()
  }

  /**
   * Process queued messages with retry logic
   */
  private async processQueue() {
    if (this.messageQueue.length === 0) {
      return
    }

    const pending = [...this.messageQueue]
    this.messageQueue = []

    for (const item of pending) {
      const success = this.send(item.message, false)

      if (!success && item.retries < this.retryConfig.maxRetries) {
        item.retries++
        const delay = this.retryConfig.retryDelay * Math.pow(this.retryConfig.backoffMultiplier, item.retries - 1)

        await new Promise(resolve => setTimeout(resolve, delay))
        this.messageQueue.push(item)
      }
    }

    // Process remaining messages
    if (this.messageQueue.length > 0) {
      setTimeout(() => this.processQueue(), this.retryConfig.retryDelay)
    }
  }

  /**
   * Get the unique tab ID
   */
  getTabId(): string {
    return this.tabId
  }

  /**
   * Check if the channel is connected
   */
  isChannelConnected(): boolean {
    return this.isConnected
  }

  /**
   * Get the number of pending messages
   */
  getPendingMessageCount(): number {
    return this.messageQueue.length
  }

  /**
   * Close the broadcast channel
   */
  close() {
    if (this.channel) {
      // Send tab closed message
      this.send({
        type: 'tab-closed',
        tabId: this.tabId,
        timestamp: Date.now()
      })

      this.channel.close()
      this.channel = null
      this.isConnected = false
    }

    window.removeEventListener('beforeunload', this.handleBeforeUnload.bind(this))
    this.handlers.clear()
    this.messageQueue = []

    console.log(`[BroadcastSync] Channel "${this.channelName}" closed for tab ${this.tabId}`)
  }

  /**
   * Generate a unique tab ID
   */
  private generateTabId(): string {
    // Use crypto.randomUUID if available, otherwise fallback
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return crypto.randomUUID()
    }

    // Fallback for older browsers
    return `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }
}

/**
 * Singleton instance for the application
 */
let broadcastSyncInstance: BroadcastSync | null = null

/**
 * Get or create the singleton BroadcastSync instance
 */
export function getBroadcastSync(): BroadcastSync {
  if (!broadcastSyncInstance) {
    broadcastSyncInstance = new BroadcastSync('sql-studio-sync')
  }
  return broadcastSyncInstance
}

/**
 * Clean up the BroadcastSync instance
 */
export function closeBroadcastSync() {
  if (broadcastSyncInstance) {
    broadcastSyncInstance.close()
    broadcastSyncInstance = null
  }
}
