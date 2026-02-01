// Main component
export { ConnectionManager } from "./connection-manager"

// Types
export type {
  ConnectionFormData,
  ConnectionGroup,
  ConnectionPayload,
  DatabaseConnection,
  DatabaseTypeOption,
  DatabaseTypeString,
  SSHTunnelConfig,
  VPCConfig,
} from "./types"
export {
  ALL_ENVIRONMENTS_FILTER,
  DATABASE_TYPE_OPTIONS,
  SSHAuthMethod,
  UNASSIGNED_ENVIRONMENT_LABEL,
} from "./types"

// Utilities
export {
  buildConnectionPayload,
  createDefaultFormData,
  getDatabaseLabel,
  getDefaultPort,
  isDatabaseRequired,
  isUsernameRequired,
  populateFormFromConnection,
  requiresHostPort,
  supportsSSL,
} from "./utils"

// Hooks
export { useConnectionActions, useConnectionForm, useConnectionList } from "./hooks"

// Sub-components (for potential reuse)
export { ConnectionCard, ConnectionForm, ConnectionList, EnvironmentFilter } from "./components"
