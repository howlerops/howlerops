/**
 * Multi-Tab Sync Hook
 *
 * React hook providing easy access to multi-tab synchronization features.
 * Manages broadcast state, tab lifecycle, and password sharing.
 *
 * Usage:
 * ```typescript
 * function MyComponent() {
 *   const {
 *     isConnected,
 *     activeTabs,
 *     isPrimaryTab,
 *     requestPasswordShare,
 *     broadcastLogout
 *   } = useMultiTabSync()
 *
 *   if (isPrimaryTab) {
 *     return <div>This is the primary tab</div>
 *   }
 * }
 * ```
 */

import { useState, useEffect, useCallback, useMemo } from 'react'
import { create } from 'zustand'
import { getBroadcastSync } from '@/lib/sync/broadcast-sync'
import { getTabLifecycleManager, type TabInfo } from '@/lib/sync/tab-lifecycle'
import { getPasswordTransferManager, type PasswordData } from '@/lib/sync/password-transfer'
import { getStoreRegistry } from '@/lib/sync/store-registry'

/**
 * Extended window interface for password share callbacks
 */
interface WindowWithPasswordShare extends Window {
  __passwordShareApprove?: (passwords: PasswordData[]) => Promise<void>
  __passwordShareDeny?: () => void
}

/**
 * Broadcast sync state store
 */
interface BroadcastSyncState {
  isConnected: boolean
  activeTabs: Map<string, TabInfo>
  isPrimaryTab: boolean
  tabCount: number
  currentTabId: string
  primaryTabId: string | null

  // Actions
  setConnected: (connected: boolean) => void
  setTabs: (tabs: Map<string, TabInfo>) => void
  setPrimary: (isPrimary: boolean, primaryTabId: string | null) => void
}

const useBroadcastStore = create<BroadcastSyncState>((set) => ({
  isConnected: false,
  activeTabs: new Map(),
  isPrimaryTab: false,
  tabCount: 0,
  currentTabId: '',
  primaryTabId: null,

  setConnected: (connected) => set({ isConnected: connected }),
  setTabs: (tabs) => set({ activeTabs: tabs, tabCount: tabs.size }),
  setPrimary: (isPrimary, primaryTabId) => set({ isPrimaryTab: isPrimary, primaryTabId })
}))

/**
 * Password share request state
 */
interface PasswordShareRequest {
  connectionIds: string[]
  requesterId: string
  timestamp: number
}

/**
 * Multi-tab sync hook return type
 */
interface UseMultiTabSyncReturn {
  /** Whether broadcast channel is connected */
  isConnected: boolean

  /** Map of active tabs */
  activeTabs: Map<string, TabInfo>

  /** Number of active tabs */
  tabCount: number

  /** Whether this tab is the primary tab */
  isPrimaryTab: boolean

  /** Current tab ID */
  currentTabId: string

  /** Primary tab ID */
  primaryTabId: string | null

  /** Request password sharing from other tabs */
  requestPasswordShare: (connectionIds: string[]) => Promise<void>

  /** Broadcast logout to all tabs */
  broadcastLogout: () => void

  /** Broadcast connection added to all tabs */
  broadcastConnectionAdded: (connectionId: string) => void

  /** Broadcast tier changed to all tabs */
  broadcastTierChanged: (newTier: string) => void

  /** Current password share request (if any) */
  passwordShareRequest: PasswordShareRequest | null

  /** Approve password share request */
  approvePasswordShare: (passwords: PasswordData[]) => Promise<void>

  /** Deny password share request */
  denyPasswordShare: () => void

  /** Whether a password share is in progress */
  isPasswordSharePending: boolean
}

/**
 * Multi-tab sync hook
 */
export function useMultiTabSync(): UseMultiTabSyncReturn {
  const {
    isConnected,
    activeTabs,
    tabCount,
    isPrimaryTab,
    currentTabId,
    primaryTabId,
    setConnected,
    setTabs,
    setPrimary
  } = useBroadcastStore()

  const [passwordShareRequest, setPasswordShareRequest] = useState<PasswordShareRequest | null>(null)
  const [isPasswordSharePending, setIsPasswordSharePending] = useState(false)

  // Initialize broadcast sync on mount
  useEffect(() => {
    const broadcast = getBroadcastSync()
    const lifecycle = getTabLifecycleManager()
    const passwordTransfer = getPasswordTransferManager()
    const registry = getStoreRegistry()

    // Initialize registry if not already initialized
    if (!registry.isInitialized()) {
      registry.initialize()
    }

    // Set initial state
    setConnected(broadcast.isChannelConnected())
    useBroadcastStore.setState({ currentTabId: broadcast.getTabId() })

    // Listen for tabs changed
    const unsubscribeTabs = lifecycle.onTabsChanged((tabs) => {
      setTabs(tabs)
    })

    // Listen for primary changed
    const unsubscribePrimary = lifecycle.onPrimaryChanged((isPrimary, primaryId) => {
      setPrimary(isPrimary, primaryId)
    })

    // Listen for password share requests
    const unsubscribePasswordRequest = passwordTransfer.onPasswordRequest(
      (connectionIds, requesterId, approve, deny) => {
        setPasswordShareRequest({
          connectionIds,
          requesterId,
          timestamp: Date.now()
        })

        // Store approve/deny callbacks
        ;(window as WindowWithPasswordShare).__passwordShareApprove = approve
        ;(window as WindowWithPasswordShare).__passwordShareDeny = deny
      }
    )

    // Listen for password received
    const unsubscribePasswordReceived = passwordTransfer.onPasswordReceived((passwords) => {
      setIsPasswordSharePending(false)
      console.log('[MultiTabSync] Received passwords for', passwords.length, 'connections')

      // Dispatch event for connection store to refresh
      window.dispatchEvent(new CustomEvent('passwords-received', {
        detail: { passwords }
      }))
    })

    // Cleanup
    return () => {
      unsubscribeTabs()
      unsubscribePrimary()
      unsubscribePasswordRequest()
      unsubscribePasswordReceived()

      // Clean up stored callbacks
      delete (window as WindowWithPasswordShare).__passwordShareApprove
      delete (window as WindowWithPasswordShare).__passwordShareDeny
    }
  }, [setConnected, setTabs, setPrimary])

  /**
   * Request password sharing from other tabs
   */
  const requestPasswordShare = useCallback(async (connectionIds: string[]) => {
    setIsPasswordSharePending(true)

    const passwordTransfer = getPasswordTransferManager()
    await passwordTransfer.requestPasswordShare(connectionIds)

    // Set timeout to clear pending state
    setTimeout(() => {
      setIsPasswordSharePending(false)
    }, 30000) // 30 seconds
  }, [])

  /**
   * Approve password share request
   */
  const approvePasswordShare = useCallback(async (passwords: PasswordData[]) => {
    const approve = (window as WindowWithPasswordShare).__passwordShareApprove
    if (approve) {
      await approve(passwords)
    }

    setPasswordShareRequest(null)
    delete (window as WindowWithPasswordShare).__passwordShareApprove
    delete (window as WindowWithPasswordShare).__passwordShareDeny
  }, [])

  /**
   * Deny password share request
   */
  const denyPasswordShare = useCallback(() => {
    const deny = (window as WindowWithPasswordShare).__passwordShareDeny
    if (deny) {
      deny()
    }

    setPasswordShareRequest(null)
    delete (window as WindowWithPasswordShare).__passwordShareApprove
    delete (window as WindowWithPasswordShare).__passwordShareDeny
  }, [])

  /**
   * Broadcast logout to all tabs
   */
  const broadcastLogout = useCallback(() => {
    const registry = getStoreRegistry()
    registry.broadcastLogout()
  }, [])

  /**
   * Broadcast connection added to all tabs
   */
  const broadcastConnectionAdded = useCallback((connectionId: string) => {
    const registry = getStoreRegistry()
    registry.broadcastConnectionAdded(connectionId)
  }, [])

  /**
   * Broadcast tier changed to all tabs
   */
  const broadcastTierChanged = useCallback((newTier: string) => {
    const registry = getStoreRegistry()
    registry.broadcastTierChanged(newTier)
  }, [])

  return {
    isConnected,
    activeTabs,
    tabCount,
    isPrimaryTab,
    currentTabId,
    primaryTabId,
    requestPasswordShare,
    broadcastLogout,
    broadcastConnectionAdded,
    broadcastTierChanged,
    passwordShareRequest,
    approvePasswordShare,
    denyPasswordShare,
    isPasswordSharePending
  }
}

/**
 * Hook for listening to broadcast events
 */
export function useBroadcastEvent<T = unknown>(
  eventType: string,
  handler: (payload: T) => void
): void {
  useEffect(() => {
    const handleEvent = (event: CustomEvent<T>) => {
      handler(event.detail)
    }

    window.addEventListener(eventType, handleEvent as EventListener)

    return () => {
      window.removeEventListener(eventType, handleEvent as EventListener)
    }
  }, [eventType, handler])
}

/**
 * Hook for checking if multi-tab sync is available
 */
export function useIsMultiTabSyncAvailable(): boolean {
  return useMemo(() => {
    return typeof BroadcastChannel !== 'undefined'
  }, [])
}
