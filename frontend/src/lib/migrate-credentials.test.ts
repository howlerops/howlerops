import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { StoredCredential } from './migrate-credentials'

type MigrationModule = typeof import('./migrate-credentials')

let mockStorePassword: ReturnType<typeof vi.fn> | undefined
let mockGetPassword: ReturnType<typeof vi.fn> | undefined
let mockDeletePassword: ReturnType<typeof vi.fn> | undefined

const createStorageMock = (): Storage => {
  const store = new Map<string, string>()
  return {
    get length() {
      return store.size
    },
    clear: () => {
      store.clear()
    },
    getItem: (key: string) => store.get(key) ?? null,
    key: (index: number) => Array.from(store.keys())[index] ?? null,
    removeItem: (key: string) => {
      store.delete(key)
    },
    setItem: (key: string, value: string) => {
      store.set(key, value)
    },
  }
}

vi.mock('../../bindings/github.com/jbeck018/howlerops/app', () => ({
  get StorePassword() {
    return mockStorePassword
  },
  get GetPassword() {
    return mockGetPassword
  },
  get DeletePassword() {
    return mockDeletePassword
  },
}))

const loadModule = async (): Promise<MigrationModule> => import('./migrate-credentials')

describe('migrate-credentials', () => {
  beforeEach(() => {
    vi.resetModules()
    const storage = createStorageMock()
    Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true })
    Object.defineProperty(globalThis, 'window', {
      value: { localStorage: storage },
      configurable: true,
      writable: true,
    })
    storage.clear()
    mockStorePassword = vi.fn().mockResolvedValue(undefined)
    mockGetPassword = vi.fn().mockResolvedValue('stored-secret')
    mockDeletePassword = vi.fn().mockResolvedValue(undefined)
    vi.spyOn(console, 'debug').mockImplementation(() => {})
    vi.spyOn(console, 'log').mockImplementation(() => {})
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reports default unmigrated status', async () => {
    const { getMigrationStatus } = await loadModule()

    await expect(getMigrationStatus()).resolves.toEqual({
      migrated: false,
      version: null,
      hasCredentials: false,
      keychainAvailable: true,
    })
  })

  it('migrates stored credentials into keychain bindings', async () => {
    const { migrateCredentialsToKeychain } = await loadModule()
    const credentials: StoredCredential[] = [{ connectionId: 'conn-1', password: 'password123' }]
    localStorage.setItem('sql-studio-secure-credentials', JSON.stringify(credentials))

    const result = await migrateCredentialsToKeychain()

    expect(result).toMatchObject({ success: true, migratedCount: 1, failedCount: 0 })
    expect(mockStorePassword).toHaveBeenCalledWith('conn-1-password', 'password123', '')
    expect(localStorage.getItem('credentials-migrated')).toBe('true')
    expect(localStorage.getItem('sql-studio-secure-credentials')).toBe(null)
  })

  it('migrates all supported credential fields', async () => {
    const { migrateCredentialsToKeychain } = await loadModule()
    localStorage.setItem(
      'sql-studio-secure-credentials',
      JSON.stringify([
        {
          connectionId: 'conn-1',
          password: 'dbpass',
          sshPassword: 'sshpass',
          sshPrivateKey: 'private-key',
        },
      ] satisfies StoredCredential[])
    )

    const result = await migrateCredentialsToKeychain()

    expect(result.success).toBe(true)
    expect(mockStorePassword).toHaveBeenNthCalledWith(1, 'conn-1-password', 'dbpass', '')
    expect(mockStorePassword).toHaveBeenNthCalledWith(2, 'conn-1-ssh_password', 'sshpass', '')
    expect(mockStorePassword).toHaveBeenNthCalledWith(3, 'conn-1-ssh_private_key', 'private-key', '')
  })

  it('keeps localStorage intact on partial migration failure', async () => {
    const { migrateCredentialsToKeychain } = await loadModule()
    localStorage.setItem(
      'sql-studio-secure-credentials',
      JSON.stringify([
        { connectionId: 'conn-1', password: 'password123' },
        { connectionId: 'conn-2', password: 'password456' },
      ] satisfies StoredCredential[])
    )
    mockStorePassword = vi
      .fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error('Keychain access denied'))

    const result = await migrateCredentialsToKeychain()

    expect(result).toMatchObject({ success: false, migratedCount: 1, failedCount: 1 })
    expect(result.errors[0]).toEqual({
      connectionId: 'conn-2',
      error: 'Keychain access denied',
    })
    expect(localStorage.getItem('sql-studio-secure-credentials')).not.toBe(null)
  })

  it('skips when bindings are unavailable', async () => {
    mockStorePassword = undefined
    mockGetPassword = undefined
    const { migrateCredentialsToKeychain } = await loadModule()
    localStorage.setItem(
      'sql-studio-secure-credentials',
      JSON.stringify([{ connectionId: 'conn-1', password: 'password123' }] satisfies StoredCredential[])
    )

    const result = await migrateCredentialsToKeychain()

    expect(result.skipped).toBe(true)
    expect(result.reason).toContain('Keychain API not yet available')
  })

  it('returns parse errors without touching keychain', async () => {
    const { migrateCredentialsToKeychain } = await loadModule()
    localStorage.setItem('sql-studio-secure-credentials', 'invalid-json')

    const result = await migrateCredentialsToKeychain()

    expect(result.success).toBe(false)
    expect(result.errors[0]?.connectionId).toBe('parse-error')
    expect(mockStorePassword).not.toHaveBeenCalled()
  })

  it('retryMigration clears prior flags before migrating again', async () => {
    const { retryMigration } = await loadModule()
    localStorage.setItem('credentials-migrated', 'true')
    localStorage.setItem('credentials-migration-version', '0.9')
    localStorage.setItem(
      'sql-studio-secure-credentials',
      JSON.stringify([{ connectionId: 'conn-1', password: 'password123' }] satisfies StoredCredential[])
    )

    const result = await retryMigration()

    expect(result.success).toBe(true)
    expect(localStorage.getItem('credentials-migration-version')).toBe('1.0')
    expect(mockStorePassword).toHaveBeenCalledTimes(1)
  })

  it('clearMigrationFlag removes migration markers only', async () => {
    const { clearMigrationFlag } = await loadModule()
    localStorage.setItem('credentials-migrated', 'true')
    localStorage.setItem('credentials-migration-version', '1.0')
    localStorage.setItem('sql-studio-secure-credentials', '[]')

    clearMigrationFlag()

    expect(localStorage.getItem('credentials-migrated')).toBe(null)
    expect(localStorage.getItem('credentials-migration-version')).toBe(null)
    expect(localStorage.getItem('sql-studio-secure-credentials')).toBe('[]')
  })
})
