/**
 * ConflictResolutionDialog Component
 *
 * Dialog for resolving sync conflicts between local and remote versions of resources.
 * Displays side-by-side comparison and allows user to choose which version to keep.
 *
 * @module components/sharing/ConflictResolutionDialog
 */

import { formatDistanceToNow } from 'date-fns'
import { AlertTriangle, Check, Info,X } from 'lucide-react'
import { useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import type { Conflict } from '@/types/sync'

interface ConflictResolutionDialogProps {
  /** Whether dialog is open */
  open: boolean

  /** Callback to change open state */
  onOpenChange: (open: boolean) => void

  /** Conflict to resolve */
  conflict: Conflict | null

  /** Callback when conflict is resolved */
  onResolve: (resolution: 'local' | 'remote') => void

  /** Loading state during resolution */
  loading?: boolean
}

/**
 * ConflictResolutionDialog Component
 *
 * Usage:
 * ```tsx
 * <ConflictResolutionDialog
 *   open={showConflict}
 *   onOpenChange={setShowConflict}
 *   conflict={currentConflict}
 *   onResolve={handleResolve}
 * />
 * ```
 */
export function ConflictResolutionDialog({
  open,
  onOpenChange,
  conflict,
  onResolve,
  loading = false,
}: ConflictResolutionDialogProps) {
  const [selectedVersion, setSelectedVersion] = useState<'local' | 'remote'>(
    'remote'
  )

  if (!conflict) return null

  const resourceTypeName =
    conflict.entityType === 'connection'
      ? 'Connection'
      : conflict.entityType === 'saved_query'
      ? 'Query'
      : 'Resource'

  const handleResolve = () => {
    onResolve(selectedVersion)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-amber-500" />
            <DialogTitle>Resolve Sync Conflict</DialogTitle>
          </div>
          <DialogDescription>
            This {resourceTypeName.toLowerCase()} was modified by another user
            or device. Choose which version to keep.
          </DialogDescription>
        </DialogHeader>

        {/* Conflict Info Alert */}
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            <strong>Conflict reason:</strong> {conflict.reason}
          </AlertDescription>
        </Alert>

        {/* Version Comparison Grid */}
        <div className="grid grid-cols-2 gap-4 flex-1 overflow-auto">
          {/* Local Version */}
          <button
            type="button"
            onClick={() => setSelectedVersion('local')}
            className={cn(
              'flex flex-col border-2 rounded-lg p-4 transition-all text-left',
              'hover:border-primary/50 focus:outline-none focus:ring-2 focus:ring-primary',
              selectedVersion === 'local'
                ? 'border-primary bg-primary/5'
                : 'border-border'
            )}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <h4 className="font-semibold">Your Version (Local)</h4>
                {selectedVersion === 'local' && (
                  <Check className="h-4 w-4 text-primary" />
                )}
              </div>
              {conflict.recommendedResolution === 'local' && (
                <Badge variant="outline" className="text-xs">
                  Recommended
                </Badge>
              )}
            </div>

            {/* Local metadata */}
            <div className="mb-3 text-sm text-muted-foreground space-y-1">
              <div>
                Modified:{' '}
                {formatDistanceToNow(conflict.localUpdatedAt, {
                  addSuffix: true,
                })}
              </div>
              <div>
                Sync version: {conflict.localSyncVersion}
              </div>
            </div>

            {/* Local content preview */}
            <div className="flex-1 min-h-0">
              <div className="text-xs font-medium mb-2 text-muted-foreground">
                Preview:
              </div>
              <pre className="text-xs bg-muted p-3 rounded overflow-auto max-h-48 font-mono">
                {JSON.stringify(conflict.localVersion, null, 2)}
              </pre>
            </div>
          </button>

          {/* Remote Version */}
          <button
            type="button"
            onClick={() => setSelectedVersion('remote')}
            className={cn(
              'flex flex-col border-2 rounded-lg p-4 transition-all text-left',
              'hover:border-primary/50 focus:outline-none focus:ring-2 focus:ring-primary',
              selectedVersion === 'remote'
                ? 'border-primary bg-primary/5'
                : 'border-border'
            )}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <h4 className="font-semibold">Server Version</h4>
                {selectedVersion === 'remote' && (
                  <Check className="h-4 w-4 text-primary" />
                )}
              </div>
              {conflict.recommendedResolution === 'remote' && (
                <Badge variant="outline" className="text-xs">
                  Recommended
                </Badge>
              )}
            </div>

            {/* Remote metadata */}
            <div className="mb-3 text-sm text-muted-foreground space-y-1">
              <div>
                Modified:{' '}
                {formatDistanceToNow(conflict.remoteUpdatedAt, {
                  addSuffix: true,
                })}
              </div>
              <div>
                Sync version: {conflict.remoteSyncVersion}
              </div>
            </div>

            {/* Remote content preview */}
            <div className="flex-1 min-h-0">
              <div className="text-xs font-medium mb-2 text-muted-foreground">
                Preview:
              </div>
              <pre className="text-xs bg-muted p-3 rounded overflow-auto max-h-48 font-mono">
                {JSON.stringify(conflict.remoteVersion, null, 2)}
              </pre>
            </div>
          </button>
        </div>

        {/* Warning */}
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription className="text-sm">
            <strong>Warning:</strong> Choosing a version will{' '}
            <strong>permanently discard</strong> the other version. This action
            cannot be undone.
          </AlertDescription>
        </Alert>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            <X className="h-4 w-4 mr-2" />
            Cancel
          </Button>

          <Button onClick={handleResolve} disabled={loading}>
            {loading ? (
              <>
                <div className="h-4 w-4 mr-2 animate-spin rounded-full border-2 border-current border-t-transparent" />
                Resolving...
              </>
            ) : (
              <>
                <Check className="h-4 w-4 mr-2" />
                Keep {selectedVersion === 'local' ? 'My Version' : 'Server Version'}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
