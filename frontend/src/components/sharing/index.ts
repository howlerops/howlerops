/**
 * Sharing Components
 *
 * Export all sharing-related components for easy imports.
 *
 * @module components/sharing
 */

export { ConflictResolutionDialog } from './ConflictResolutionDialog'
export { SharedResourceCard } from './SharedResourceCard'
export { VisibilityToggle } from './VisibilityToggle'

// Re-export types for convenience
export type { Connection, CreateConnectionInput, UpdateConnectionInput } from '@/lib/api/connections'
export type { CreateQueryInput, SavedQuery, UpdateQueryInput } from '@/lib/api/queries'
export type { Conflict, ConflictResolution } from '@/types/sync'
