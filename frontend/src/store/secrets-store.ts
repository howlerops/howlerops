import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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
        set({
          isLocked: true,
          hasUserKey: false,
          teamKeyCount: 0,
          showPassphrasePrompt: false,
          unlockError: null,
        })
      },
      
      unlock: async (passphrase: string) => {
        // TODO: Use passphrase when implementing actual UnlockKeyStore API
        console.debug('UnlockKeyStore called with passphrase length:', passphrase.length)
        const { setUnlocking, setUnlockError, setLocked, setUserKey } = get()
        
        setUnlocking(true)
        setUnlockError(null)
        
        try {
          // Call the backend to unlock the key store
          // TODO: Implement UnlockKeyStore API call when backend is ready
          // For now, simulate successful unlock
          const response = { success: true, error: null }
          
          if (response?.success) {
            setLocked(false)
            setUserKey(true)
            set({ showPassphrasePrompt: false })
          } else {
            setUnlockError(response?.error || 'Failed to unlock key store')
          }
        } catch (error) {
          setUnlockError(error instanceof Error ? error.message : 'Unknown error')
        } finally {
          setUnlocking(false)
        }
      },
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
