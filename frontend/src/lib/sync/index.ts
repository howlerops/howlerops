/**
 * Multi-Tab Synchronization System
 *
 * Comprehensive multi-tab synchronization for Howlerops using BroadcastChannel API.
 *
 * @module sync
 */

// Core broadcast functionality
export {
  BroadcastSync,
  getBroadcastSync,
  closeBroadcastSync,
  type BroadcastMessage,
  type BroadcastMessageHandler
} from './broadcast-sync'

// Zustand middleware
export {
  broadcastSync,
  broadcastAction,
  onBroadcastAction,
  type BroadcastSyncOptions
} from './zustand-broadcast-middleware'

// Tab lifecycle management
export {
  TabLifecycleManager,
  getTabLifecycleManager,
  destroyTabLifecycleManager,
  type TabInfo
} from './tab-lifecycle'

// Password transfer
export {
  PasswordTransferManager,
  getPasswordTransferManager,
  destroyPasswordTransferManager,
  type PasswordData,
  type PasswordRequestHandler,
  type PasswordReceivedHandler
} from './password-transfer'

// Store registry
export {
  getStoreRegistry,
  initializeSyncRegistry,
  getStoreConfig,
  shouldSyncStore,
  broadcastLogout,
  broadcastConnectionAdded,
  broadcastTierChanged,
  type StoreSyncConfig
} from './store-registry'
