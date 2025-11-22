/**
 * Tests for credential migration utility
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  clearMigrationFlag,
  getMigrationStatus,
  migrateCredentialsToKeychain,
  retryMigration,
  type StoredCredential,
} from './migrate-credentials'

// Mock localStorage
const localStorageMock: Record<string, string> = {}

const mockLocalStorage = {
  getItem: (key: string) => localStorageMock[key] ?? null,
  setItem: (key: string, value: string) => {
    localStorageMock[key] = value
  },
  removeItem: (key: string) => {
    delete localStorageMock[key]
  },
  clear: () => {
    Object.keys(localStorageMock).forEach((key) => delete localStorageMock[key])
  },
}

// Mock window.go Wails API
const mockWailsAPI = {
  StorePassword: vi.fn(),
  GetPassword: vi.fn(),
}

describe('migrate-credentials', () => {
  beforeEach(() => {
    // Clear localStorage mock
    mockLocalStorage.clear()

    // Reset Wails API mocks
    mockWailsAPI.StorePassword.mockReset()
    mockWailsAPI.GetPassword.mockReset()

    // Setup window mocks
    Object.defineProperty(global, 'window', {
      value: {
        go: {
          main: {
            App: mockWailsAPI,
          },
        },
      },
      writable: true,
    })

    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    })

    // Mock console methods
    vi.spyOn(console, 'log').mockImplementation(() => {})
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('getMigrationStatus', () => {
    it('should return not migrated by default', () => {
      const status = getMigrationStatus()
      expect(status.migrated).toBe(false)
      expect(status.version).toBe(null)
      expect(status.hasCredentials).toBe(false)
      expect(status.keychainAvailable).toBe(true)
    })

    it('should detect migrated state', () => {
      localStorage.setItem('credentials-migrated', 'true')
      localStorage.setItem('credentials-migration-version', '1.0')

      const status = getMigrationStatus()
      expect(status.migrated).toBe(true)
      expect(status.version).toBe('1.0')
    })

    it('should detect credentials presence', () => {
      const credentials: StoredCredential[] = [
        { connectionId: 'conn-1', password: 'test123' },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))

      const status = getMigrationStatus()
      expect(status.hasCredentials).toBe(true)
    })

    it('should detect keychain availability', () => {
      const status = getMigrationStatus()
      expect(status.keychainAvailable).toBe(true)
    })
  })

  describe('migrateCredentialsToKeychain', () => {
    it('should skip if already migrated', async () => {
      localStorage.setItem('credentials-migrated', 'true')
      localStorage.setItem('credentials-migration-version', '1.0')

      const result = await migrateCredentialsToKeychain()

      expect(result.skipped).toBe(true)
      expect(result.reason).toBe('Already migrated')
      expect(mockWailsAPI.StorePassword).not.toHaveBeenCalled()
    })

    it('should skip if no credentials found', async () => {
      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(result.skipped).toBe(true)
      expect(result.reason).toBe('No credentials found')
      expect(localStorage.getItem('credentials-migrated')).toBe('true')
    })

    it('should migrate single credential successfully', async () => {
      const credentials: StoredCredential[] = [
        {
          connectionId: 'conn-1',
          password: 'password123',
        },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(result.migratedCount).toBe(1)
      expect(result.failedCount).toBe(0)
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledWith(
        'sql-studio',
        'conn-1-password',
        'password123'
      )
      expect(localStorage.getItem('credentials-migrated')).toBe('true')
      expect(localStorage.getItem('sql-studio-secure-credentials')).toBe(null)
    })

    it('should migrate multiple credentials successfully', async () => {
      const credentials: StoredCredential[] = [
        {
          connectionId: 'conn-1',
          password: 'password123',
        },
        {
          connectionId: 'conn-2',
          password: 'password456',
          sshPassword: 'sshpass789',
        },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(result.migratedCount).toBe(2)
      expect(result.failedCount).toBe(0)
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledTimes(3)
      expect(localStorage.getItem('sql-studio-secure-credentials')).toBe(null)
    })

    it('should migrate all credential types', async () => {
      const credentials: StoredCredential[] = [
        {
          connectionId: 'conn-1',
          password: 'dbpass',
          sshPassword: 'sshpass',
          sshPrivateKey: '-----BEGIN RSA PRIVATE KEY-----',
        },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledWith(
        'sql-studio',
        'conn-1-password',
        'dbpass'
      )
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledWith(
        'sql-studio',
        'conn-1-ssh_password',
        'sshpass'
      )
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledWith(
        'sql-studio',
        'conn-1-ssh_private_key',
        '-----BEGIN RSA PRIVATE KEY-----'
      )
    })

    it('should handle individual credential failures gracefully', async () => {
      const credentials: StoredCredential[] = [
        { connectionId: 'conn-1', password: 'password123' },
        { connectionId: 'conn-2', password: 'password456' },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))

      // First call succeeds, second fails
      mockWailsAPI.StorePassword
        .mockResolvedValueOnce(undefined)
        .mockRejectedValueOnce(new Error('Keychain access denied'))

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(false)
      expect(result.migratedCount).toBe(1)
      expect(result.failedCount).toBe(1)
      expect(result.errors).toHaveLength(1)
      expect(result.errors[0].connectionId).toBe('conn-2')
      expect(result.errors[0].error).toBe('Keychain access denied')
      // localStorage should NOT be cleared on partial failure
      expect(localStorage.getItem('sql-studio-secure-credentials')).not.toBe(null)
    })

    it('should handle parse errors gracefully', async () => {
      localStorage.setItem('sql-studio-secure-credentials', 'invalid-json')

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(false)
      expect(result.errors).toHaveLength(1)
      expect(result.errors[0].connectionId).toBe('parse-error')
      expect(mockWailsAPI.StorePassword).not.toHaveBeenCalled()
    })

    it('should handle keychain API not available', async () => {
      // Remove Wails API
      Object.defineProperty(global, 'window', {
        value: {},
        writable: true,
      })

      const credentials: StoredCredential[] = [
        { connectionId: 'conn-1', password: 'password123' },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))

      const result = await migrateCredentialsToKeychain()

      expect(result.skipped).toBe(true)
      expect(result.reason).toContain('Keychain API not yet available')
      // Credentials should remain in localStorage
      expect(localStorage.getItem('sql-studio-secure-credentials')).not.toBe(null)
    })

    it('should handle missing credential fields', async () => {
      const credentials: StoredCredential[] = [
        { connectionId: 'conn-1' }, // No password fields
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      // Should succeed but not call StorePassword since no fields to migrate
      expect(result.success).toBe(true)
      expect(result.migratedCount).toBe(1)
      expect(mockWailsAPI.StorePassword).not.toHaveBeenCalled()
    })
  })

  describe('retryMigration', () => {
    it('should clear migration flag and retry', async () => {
      localStorage.setItem('credentials-migrated', 'true')
      localStorage.setItem('credentials-migration-version', '1.0')

      const credentials: StoredCredential[] = [
        { connectionId: 'conn-1', password: 'password123' },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await retryMigration()

      expect(result.success).toBe(true)
      expect(result.migratedCount).toBe(1)
      expect(mockWailsAPI.StorePassword).toHaveBeenCalled()
    })
  })

  describe('clearMigrationFlag', () => {
    it('should clear migration flags', () => {
      localStorage.setItem('credentials-migrated', 'true')
      localStorage.setItem('credentials-migration-version', '1.0')

      clearMigrationFlag()

      expect(localStorage.getItem('credentials-migrated')).toBe(null)
      expect(localStorage.getItem('credentials-migration-version')).toBe(null)
    })
  })

  describe('edge cases', () => {
    it('should handle empty credentials array', async () => {
      localStorage.setItem('sql-studio-secure-credentials', '[]')
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(result.migratedCount).toBe(0)
      expect(mockWailsAPI.StorePassword).not.toHaveBeenCalled()
    })

    it('should handle very long credential values', async () => {
      const longPrivateKey = 'A'.repeat(10000)
      const credentials: StoredCredential[] = [
        {
          connectionId: 'conn-1',
          sshPrivateKey: longPrivateKey,
        },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledWith(
        'sql-studio',
        'conn-1-ssh_private_key',
        longPrivateKey
      )
    })

    it('should handle special characters in connection IDs', async () => {
      const credentials: StoredCredential[] = [
        {
          connectionId: 'conn-with-special-chars-!@#$%',
          password: 'password123',
        },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(mockWailsAPI.StorePassword).toHaveBeenCalledWith(
        'sql-studio',
        'conn-with-special-chars-!@#$%-password',
        'password123'
      )
    })
  })

  describe('migration version handling', () => {
    it('should upgrade from old version', async () => {
      localStorage.setItem('credentials-migrated', 'true')
      localStorage.setItem('credentials-migration-version', '0.9') // Old version

      const credentials: StoredCredential[] = [
        { connectionId: 'conn-1', password: 'password123' },
      ]
      localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))
      mockWailsAPI.StorePassword.mockResolvedValue(undefined)

      const result = await migrateCredentialsToKeychain()

      expect(result.success).toBe(true)
      expect(localStorage.getItem('credentials-migration-version')).toBe('1.0')
    })
  })
})
