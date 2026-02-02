import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import {
  decryptMasterKey,
  type EncryptedMasterKey,
  encryptMasterKey,
  generateMasterKey,
} from '@/lib/crypto/encryption'

import * as App from '../../bindings/github.com/jbeck018/howlerops/app'

// Service name for keychain storage
const KEYCHAIN_SERVICE = 'sql-studio-keystore'
const KEYCHAIN_MASTER_KEY = 'encrypted-master-key'

// In-memory master key (never persisted to disk)
let sessionMasterKey: CryptoKey | null = null

interface SecretsState {
  // Key store state
  isLocked: boolean
  hasUserKey: boolean
  teamKeyCount: number

  // UI state
  showPassphrasePrompt: boolean
  isUnlocking: boolean
  unlockError: string | null

  // Actions
  setLocked: (locked: boolean) => void
  setUserKey: (hasKey: boolean) => void
  setTeamKeyCount: (count: number) => void
  showUnlockPrompt: () => void
  hideUnlockPrompt: () => void
  setUnlocking: (unlocking: boolean) => void
  setUnlockError: (error: string | null) => void
  lock: () => void
  unlock: (passphrase: string) => Promise<void>

  // Master key access (read-only, key lives in memory only)
  getMasterKey: () => CryptoKey | null
  hasMasterKey: () => boolean
}

export const useSecretsStore = create<SecretsState>()(
  persist(
    (set, get) => ({
      // Initial state
      isLocked: true,
      hasUserKey: false,
      teamKeyCount: 0,
      showPassphrasePrompt: false,
      isUnlocking: false,
      unlockError: null,

      // Actions
      setLocked: (locked) => set({ isLocked: locked }),
      setUserKey: (hasKey) => set({ hasUserKey: hasKey }),
      setTeamKeyCount: (count) => set({ teamKeyCount: count }),
      
      showUnlockPrompt: () => set({ showPassphrasePrompt: true, unlockError: null }),
      hideUnlockPrompt: () => set({ showPassphrasePrompt: false, unlockError: null }),
      
      setUnlocking: (unlocking) => set({ isUnlocking: unlocking }),
      setUnlockError: (error) => set({ unlockError: error }),
      
      lock: () => {
        // Clear master key from memory
        sessionMasterKey = null

        set({
          isLocked: true,
          hasUserKey: false,
          teamKeyCount: 0,
          showPassphrasePrompt: false,
          unlockError: null,
        })
      },
      
      unlock: async (passphrase: string) => {
        const { setUnlocking, setUnlockError, setLocked, setUserKey } = get()

        setUnlocking(true)
        setUnlockError(null)

        try {
          // Try to retrieve encrypted master key from OS keychain
          let encryptedKeyJson: string | null = null

          try {
            if (typeof App.GetPassword === 'function') {
              encryptedKeyJson = await App.GetPassword(KEYCHAIN_SERVICE, KEYCHAIN_MASTER_KEY)
            }
          } catch (err) {
            // Key not found is expected for first-time setup
            const errorMsg = String(err)
            if (!errorMsg.includes('not found') && !errorMsg.includes('NotFound')) {
              console.error('[SecretsStore] Failed to retrieve encrypted key:', err)
            }
          }

          if (encryptedKeyJson) {
            // Existing key found - decrypt it with the passphrase
            try {
              const encryptedKey: EncryptedMasterKey = JSON.parse(encryptedKeyJson)
              sessionMasterKey = await decryptMasterKey(encryptedKey, passphrase)

              setLocked(false)
              setUserKey(true)
              set({ showPassphrasePrompt: false })
            } catch {
              // Decryption failed - wrong passphrase
              console.debug('[SecretsStore] Decryption failed, likely wrong passphrase')
              setUnlockError('Invalid passphrase. Please try again.')
            }
          } else {
            // No existing key - create a new master key and encrypt it
            console.debug('[SecretsStore] No existing key found, creating new master key')

            const newMasterKey = await generateMasterKey()
            const encryptedKey = await encryptMasterKey(newMasterKey, passphrase)

            // Store encrypted key in OS keychain
            try {
              if (typeof App.StorePassword === 'function') {
                await App.StorePassword(
                  KEYCHAIN_SERVICE,
                  KEYCHAIN_MASTER_KEY,
                  JSON.stringify(encryptedKey)
                )
              } else {
                console.warn('[SecretsStore] Keychain API not available, key stored in memory only')
              }
            } catch (storeError) {
              console.error('[SecretsStore] Failed to store encrypted key:', storeError)
              // Continue anyway - key is in memory for this session
            }

            sessionMasterKey = newMasterKey

            setLocked(false)
            setUserKey(true)
            set({ showPassphrasePrompt: false })
          }
        } catch (error) {
          setUnlockError(error instanceof Error ? error.message : 'Unknown error')
        } finally {
          setUnlocking(false)
        }
      },

      // Master key access
      getMasterKey: () => sessionMasterKey,
      hasMasterKey: () => sessionMasterKey !== null,
    }),
    {
      name: 'secrets-store',
      partialize: (state) => ({
        // Only persist the locked state, not sensitive data
        isLocked: state.isLocked,
        hasUserKey: state.hasUserKey,
        teamKeyCount: state.teamKeyCount,
      }),
    }
  )
)

// Hook for checking if secrets are available
export const useSecretsAvailable = () => {
  const { isLocked, hasUserKey } = useSecretsStore()
  return !isLocked && hasUserKey
}

// Hook for requiring secrets to be unlocked
export const useRequireSecrets = () => {
  const { isLocked, showUnlockPrompt } = useSecretsStore()
  
  return () => {
    if (isLocked) {
      showUnlockPrompt()
      return false
    }
    return true
  }
}
