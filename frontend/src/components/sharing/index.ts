/**
 * Sharing Components
 *
 * Export all sharing-related components for easy imports.
 *
 * @module components/sharing
 */

export { VisibilityToggle } from './VisibilityToggle'
export { SharedResourceCard } from './SharedResourceCard'
export { ConflictResolutionDialog } from './ConflictResolutionDialog'

// Re-export types for convenience
export type { Connection, CreateConnectionInput, UpdateConnectionInput } from '@/lib/api/connections'
export type { SavedQuery, CreateQueryInput, UpdateQueryInput } from '@/lib/api/queries'
export type { Conflict, ConflictResolution } from '@/types/sync'
