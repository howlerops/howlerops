/**
 * Type definitions for OS Keychain integration via Wails
 *
 * These functions will be added to the Wails backend to support
 * secure credential storage in the OS keychain.
 *
 * Implementation: github.com/zalando/go-keyring
 */

declare module '../../wailsjs/go/main/App' {
  /**
   * Store a password securely in the OS keychain
   *
   * @param service - Service name (e.g., "sql-studio")
   * @param account - Account/key name (e.g., "conn-123-password")
   * @param password - Password or sensitive value to store
   * @returns Promise that resolves when stored successfully
   * @throws Error if keychain is locked or unavailable
   *
   * @example
   * await StorePassword("sql-studio", "conn-123-password", "secret123")
   */
  export function StorePassword(
    service: string,
    account: string,
    password: string
  ): Promise<void>

  /**
   * Retrieve a password from the OS keychain
   *
   * @param service - Service name (e.g., "sql-studio")
   * @param account - Account/key name (e.g., "conn-123-password")
   * @returns Promise that resolves with the password
   * @throws Error with "not found" if password doesn't exist
   * @throws Error if keychain is locked or unavailable
   *
   * @example
   * const password = await GetPassword("sql-studio", "conn-123-password")
   */
  export function GetPassword(
    service: string,
    account: string
  ): Promise<string>

  /**
   * Delete a password from the OS keychain
   *
   * @param service - Service name (e.g., "sql-studio")
   * @param account - Account/key name (e.g., "conn-123-password")
   * @returns Promise that resolves when deleted successfully
   * @throws Error with "not found" if password doesn't exist
   * @throws Error if keychain is locked or unavailable
   *
   * @example
   * await DeletePassword("sql-studio", "conn-123-password")
   */
  export function DeletePassword(
    service: string,
    account: string
  ): Promise<void>
}

/**
 * Global type augmentation for window.go runtime
 * This allows TypeScript to recognize the Wails runtime at development time
 */
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          StorePassword?: (service: string, account: string, password: string) => Promise<void>
          GetPassword?: (service: string, account: string) => Promise<string>
          DeletePassword?: (service: string, account: string) => Promise<void>
        }
      }
    }
  }
}

export {}
