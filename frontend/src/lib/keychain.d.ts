/**
 * Type definitions for OS Keychain integration via Wails v3
 *
 * In Wails v3, we import bindings directly from generated files.
 * This file provides additional type hints for the keychain functions.
 */

// Re-export types from generated bindings
export type { CancellablePromise } from '@wailsio/runtime'

/**
 * Store a password securely in the OS keychain
 *
 * @param connectionID - Connection identifier
 * @param password - Password or sensitive value to store
 * @param masterKeyBase64 - Optional master key for encryption
 * @returns Promise that resolves when stored successfully
 * @throws Error if keychain is locked or unavailable
 *
 * @example
 * import { StorePassword } from '../../bindings/github.com/jbeck018/howlerops/app'
 * await StorePassword("conn-123", "secret123", "")
 */

/**
 * Retrieve a password from the OS keychain
 *
 * @param connectionID - Connection identifier
 * @param masterKeyBase64 - Optional master key for decryption
 * @returns Promise that resolves with the password
 * @throws Error with "not found" if password doesn't exist
 * @throws Error if keychain is locked or unavailable
 *
 * @example
 * import { GetPassword } from '../../bindings/github.com/jbeck018/howlerops/app'
 * const password = await GetPassword("conn-123", "")
 */

/**
 * Delete a password from the OS keychain
 *
 * @param connectionID - Connection identifier
 * @returns Promise that resolves when deleted successfully
 * @throws Error with "not found" if password doesn't exist
 * @throws Error if keychain is locked or unavailable
 *
 * @example
 * import { DeletePassword } from '../../bindings/github.com/jbeck018/howlerops/app'
 * await DeletePassword("conn-123")
 */

export {}
