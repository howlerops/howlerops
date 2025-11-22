/**
 * Multi-Tab Synchronization System
 *
 * Comprehensive multi-tab synchronization for Howlerops using BroadcastChannel API.
 *
 * @module sync
 */

// Core broadcast functionality
export {
  type BroadcastMessage,
  type BroadcastMessageHandler,
  BroadcastSync,
  closeBroadcastSync,
  getBroadcastSync} from './broadcast-sync'

// Zustand middleware
export {
  broadcastAction,
  broadcastSync,
  type BroadcastSyncOptions,
  onBroadcastAction} from './zustand-broadcast-middleware'

// Tab lifecycle management
export {
  destroyTabLifecycleManager,
  getTabLifecycleManager,
  type TabInfo,
  TabLifecycleManager} from './tab-lifecycle'

// Password transfer
export {
  destroyPasswordTransferManager,
  getPasswordTransferManager,
  type PasswordData,
  type PasswordReceivedHandler,
  type PasswordRequestHandler,
  PasswordTransferManager} from './password-transfer'

// Store registry
export {
  broadcastConnectionAdded,
  broadcastLogout,
  broadcastTierChanged,
  getStoreConfig,
  getStoreRegistry,
  initializeSyncRegistry,
  shouldSyncStore,
  type StoreSyncConfig
} from './store-registry'
