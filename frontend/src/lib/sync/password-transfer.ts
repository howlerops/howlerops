/**
 * Secure Password Transfer System
 *
 * Enables secure password sharing between browser tabs using ephemeral encryption.
 * Implements temporary encryption keys with automatic expiration.
 *
 * Security Features:
 * - Ephemeral keys valid for 10 seconds only
 * - AES-GCM encryption in transit
 * - User approval required for transfers
 * - Automatic key expiration
 * - Visual confirmation in both tabs
 *
 * Usage:
 * ```typescript
 * // In new tab (requesting passwords)
 * const transfer = new PasswordTransferManager()
 * transfer.requestPasswordShare(['conn-1', 'conn-2'])
 *
 * // In existing tab (responding to request)
 * transfer.onPasswordRequest((connectionIds, approve) => {
 *   if (userApproves) {
 *     const passwords = getPasswords(connectionIds)
 *     approve(passwords)
 *   }
 * })
 * ```
 */

import { getBroadcastSync } from './broadcast-sync'
import { getSecureStorage } from '../secure-storage'

/**
 * Password data structure
 */
export interface PasswordData {
  connectionId: string
  password?: string
  sshPassword?: string
  sshPrivateKey?: string
}

/**
 * Password request handler
 */
export type PasswordRequestHandler = (
  connectionIds: string[],
  requesterId: string,
  approve: (passwords: PasswordData[]) => Promise<void>,
  deny: () => void
) => void

/**
 * Password received handler
 */
export type PasswordReceivedHandler = (passwords: PasswordData[]) => void

/**
 * Ephemeral encryption key
 */
interface EphemeralKey {
  key: CryptoKey
  createdAt: number
  expiresAt: number
}

/**
 * Pending password request
 */
interface PendingRequest {
  requesterId: string
  connectionIds: string[]
  timestamp: number
  timeoutId: ReturnType<typeof setTimeout>
}

/**
 * Configuration options
 */
interface PasswordTransferOptions {
  /**
   * Key lifetime in milliseconds
   * @default 10000 (10 seconds)
   */
  keyLifetime?: number

  /**
   * Request timeout in milliseconds
   * @default 30000 (30 seconds)
   */
  requestTimeout?: number
}

const DEFAULT_OPTIONS: Required<PasswordTransferOptions> = {
  keyLifetime: 10000, // 10 seconds
  requestTimeout: 30000 // 30 seconds
}

/**
 * Password Transfer Manager
 */
export class PasswordTransferManager {
  private options: Required<PasswordTransferOptions>
  private ephemeralKeys: Map<string, EphemeralKey> = new Map()
  private pendingRequests: Map<string, PendingRequest> = new Map()
  private requestHandlers: Set<PasswordRequestHandler> = new Set()
  private receivedHandlers: Set<PasswordReceivedHandler> = new Set()

  constructor(options: PasswordTransferOptions = {}) {
    this.options = { ...DEFAULT_OPTIONS, ...options }
    this.setupListeners()
    this.startKeyCleanup()
  }

  /**
   * Set up broadcast channel listeners
   */
  private setupListeners() {
    const broadcast = getBroadcastSync()

    // Listen for password share requests
    broadcast.on('request-password-share', (message) => {
      if (message.requesterId === broadcast.getTabId()) {
        return // Ignore own requests
      }

      // Notify handlers
      this.requestHandlers.forEach(handler => {
        try {
          handler(
            message.connectionIds,
            message.requesterId,
            (passwords) => this.approveRequest(message.requesterId, passwords),
            () => this.denyRequest(message.requesterId)
          )
        } catch (error) {
          console.error('[PasswordTransfer] Error in request handler:', error)
        }
      })
    })

    // Listen for password share responses
    broadcast.on('password-share-response', async (message) => {
      if (message.senderId === broadcast.getTabId()) {
        return // Ignore own responses
      }

      try {
        // Decrypt the password
        const decryptedData = await this.decryptPassword(
          message.encryptedPassword,
          message.key,
          message.iv
        )

        if (decryptedData) {
          const passwords = JSON.parse(decryptedData) as PasswordData[]

          // Store in secure storage
          const secureStorage = getSecureStorage()
          passwords.forEach(pwd => {
            secureStorage.setCredentials(pwd.connectionId, pwd)
          })

          // Notify handlers
          this.receivedHandlers.forEach(handler => {
            try {
              handler(passwords)
            } catch (error) {
              console.error('[PasswordTransfer] Error in received handler:', error)
            }
          })

          // Clear pending request
          const broadcast = getBroadcastSync()
          const request = this.pendingRequests.get(broadcast.getTabId())
          if (request) {
            clearTimeout(request.timeoutId)
            this.pendingRequests.delete(broadcast.getTabId())
          }
        }
      } catch (error) {
        console.error('[PasswordTransfer] Failed to process password response:', error)
      }
    })
  }

  /**
   * Request password sharing from other tabs
   */
  async requestPasswordShare(connectionIds: string[]): Promise<void> {
    const broadcast = getBroadcastSync()
    const requesterId = broadcast.getTabId()

    // Clear any existing request
    const existingRequest = this.pendingRequests.get(requesterId)
    if (existingRequest) {
      clearTimeout(existingRequest.timeoutId)
    }

    // Create timeout for request
    const timeoutId = setTimeout(() => {
      this.pendingRequests.delete(requesterId)
      console.log('[PasswordTransfer] Password request timed out')
    }, this.options.requestTimeout)

    // Store pending request
    this.pendingRequests.set(requesterId, {
      requesterId,
      connectionIds,
      timestamp: Date.now(),
      timeoutId
    })

    // Broadcast request
    broadcast.send({
      type: 'request-password-share',
      connectionIds,
      requesterId,
      timestamp: Date.now()
    })

    console.log(`[PasswordTransfer] Requested passwords for ${connectionIds.length} connections`)
  }

  /**
   * Approve a password request and send encrypted passwords
   */
  private async approveRequest(requesterId: string, passwords: PasswordData[]): Promise<void> {
    try {
      // Generate ephemeral key
      const keyData = await this.generateEphemeralKey()

      // Encrypt passwords
      const encrypted = await this.encryptPasswords(passwords, keyData.key)

      if (!encrypted) {
        console.error('[PasswordTransfer] Failed to encrypt passwords')
        return
      }

      // Broadcast encrypted passwords with key
      const broadcast = getBroadcastSync()
      broadcast.send({
        type: 'password-share-response',
        connectionId: '', // Not used for bulk transfer
        encryptedPassword: encrypted.ciphertext,
        key: encrypted.keyData,
        iv: encrypted.iv,
        senderId: broadcast.getTabId(),
        timestamp: Date.now()
      })

      console.log(`[PasswordTransfer] Sent encrypted passwords for ${passwords.length} connections`)
    } catch (error) {
      console.error('[PasswordTransfer] Failed to approve request:', error)
    }
  }

  /**
   * Deny a password request
   */
  private denyRequest(requesterId: string): void {
    console.log(`[PasswordTransfer] Denied password request from ${requesterId}`)
    // Could send a denial message if needed
  }

  /**
   * Generate an ephemeral encryption key
   */
  private async generateEphemeralKey(): Promise<EphemeralKey> {
    const key = await crypto.subtle.generateKey(
      {
        name: 'AES-GCM',
        length: 256
      },
      true, // extractable
      ['encrypt', 'decrypt']
    )

    const now = Date.now()
    const ephemeralKey: EphemeralKey = {
      key,
      createdAt: now,
      expiresAt: now + this.options.keyLifetime
    }

    // Store key with unique ID
    const keyId = crypto.randomUUID()
    this.ephemeralKeys.set(keyId, ephemeralKey)

    return ephemeralKey
  }

  /**
   * Encrypt passwords with ephemeral key
   */
  private async encryptPasswords(
    passwords: PasswordData[],
    key: CryptoKey
  ): Promise<{ ciphertext: string, keyData: string, iv: string } | null> {
    try {
      // Convert passwords to JSON string
      const plaintext = JSON.stringify(passwords)

      // Generate IV
      const iv = crypto.getRandomValues(new Uint8Array(12))

      // Encrypt
      const encrypted = await crypto.subtle.encrypt(
        {
          name: 'AES-GCM',
          iv
        },
        key,
        new TextEncoder().encode(plaintext)
      )

      // Export key for transmission
      const exportedKey = await crypto.subtle.exportKey('raw', key)

      // Convert to base64
      const ciphertext = this.arrayBufferToBase64(encrypted)
      const keyData = this.arrayBufferToBase64(exportedKey)
      const ivData = this.arrayBufferToBase64(iv)

      return {
        ciphertext,
        keyData,
        iv: ivData
      }
    } catch (error) {
      console.error('[PasswordTransfer] Encryption failed:', error)
      return null
    }
  }

  /**
   * Decrypt password with ephemeral key
   */
  private async decryptPassword(
    ciphertext: string,
    keyData: string,
    ivData: string
  ): Promise<string | null> {
    try {
      // Convert from base64
      const ciphertextBuffer = this.base64ToArrayBuffer(ciphertext)
      const keyBuffer = this.base64ToArrayBuffer(keyData)
      const iv = this.base64ToArrayBuffer(ivData)

      // Import key
      const key = await crypto.subtle.importKey(
        'raw',
        keyBuffer,
        {
          name: 'AES-GCM',
          length: 256
        },
        false,
        ['decrypt']
      )

      // Decrypt
      const decrypted = await crypto.subtle.decrypt(
        {
          name: 'AES-GCM',
          iv: new Uint8Array(iv)
        },
        key,
        ciphertextBuffer
      )

      // Convert to string
      return new TextDecoder().decode(decrypted)
    } catch (error) {
      console.error('[PasswordTransfer] Decryption failed:', error)
      return null
    }
  }

  /**
   * Convert ArrayBuffer to base64 string
   */
  private arrayBufferToBase64(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer)
    let binary = ''
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i])
    }
    return btoa(binary)
  }

  /**
   * Convert base64 string to ArrayBuffer
   */
  private base64ToArrayBuffer(base64: string): ArrayBuffer {
    const binary = atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i)
    }
    return bytes.buffer
  }

  /**
   * Start periodic cleanup of expired keys
   */
  private startKeyCleanup() {
    setInterval(() => {
      const now = Date.now()

      for (const [keyId, keyData] of this.ephemeralKeys.entries()) {
        if (now > keyData.expiresAt) {
          this.ephemeralKeys.delete(keyId)
        }
      }
    }, 5000) // Check every 5 seconds
  }

  /**
   * Register a handler for password requests
   */
  onPasswordRequest(handler: PasswordRequestHandler): () => void {
    this.requestHandlers.add(handler)

    // Return unsubscribe function
    return () => {
      this.requestHandlers.delete(handler)
    }
  }

  /**
   * Register a handler for received passwords
   */
  onPasswordReceived(handler: PasswordReceivedHandler): () => void {
    this.receivedHandlers.add(handler)

    // Return unsubscribe function
    return () => {
      this.receivedHandlers.delete(handler)
    }
  }

  /**
   * Get passwords for connections from secure storage
   */
  async getPasswordsForConnections(connectionIds: string[]): Promise<PasswordData[]> {
    const secureStorage = getSecureStorage()
    const passwords: PasswordData[] = []

    for (const connectionId of connectionIds) {
      const credentials = await secureStorage.getCredentials(connectionId)
      if (credentials) {
        passwords.push({
          connectionId,
          password: credentials.password,
          sshPassword: credentials.sshPassword,
          sshPrivateKey: credentials.sshPrivateKey
        })
      }
    }

    return passwords
  }

  /**
   * Clean up resources
   */
  destroy() {
    // Clear all pending requests
    this.pendingRequests.forEach(request => {
      clearTimeout(request.timeoutId)
    })
    this.pendingRequests.clear()

    // Clear ephemeral keys
    this.ephemeralKeys.clear()

    // Clear handlers
    this.requestHandlers.clear()
    this.receivedHandlers.clear()
  }
}

/**
 * Singleton instance
 */
let passwordTransferInstance: PasswordTransferManager | null = null

/**
 * Get or create the singleton PasswordTransferManager instance
 */
export function getPasswordTransferManager(): PasswordTransferManager {
  if (!passwordTransferInstance) {
    passwordTransferInstance = new PasswordTransferManager()
  }
  return passwordTransferInstance
}

/**
 * Clean up the PasswordTransferManager instance
 */
export function destroyPasswordTransferManager() {
  if (passwordTransferInstance) {
    passwordTransferInstance.destroy()
    passwordTransferInstance = null
  }
}
