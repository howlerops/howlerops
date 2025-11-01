/**
 * Conflict Resolver Component
 *
 * Modal UI for resolving sync conflicts between local and remote versions.
 * Provides side-by-side comparison and resolution options.
 *
 * @module components/sync/conflict-resolver
 */

import { useState } from 'react'
import { useSyncStore, useSyncActions } from '@/store/sync-store'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { AlertTriangle, CheckCircle, Clock, Database, FileText } from 'lucide-react'
import type { Conflict, ConflictResolution } from '@/types/sync'
import type { Connection } from '@/lib/api/connections'
import type { SavedQuery } from '@/lib/api/queries'

interface ConflictResolverProps {
  /** Whether to show the modal */
  open?: boolean
  /** Callback when all conflicts are resolved */
  onAllResolved?: () => void
}

/**
 * Conflict Resolver Modal
 */
export function ConflictResolver({ open, onAllResolved }: ConflictResolverProps) {
  const pendingConflicts = useSyncStore((state) => state.pendingConflicts)
  const { resolveConflict, clearConflicts } = useSyncActions()
  const [currentIndex, setCurrentIndex] = useState(0)
  const [resolving, setResolving] = useState(false)

  const isOpen = open !== undefined ? open : pendingConflicts.length > 0
  const currentConflict = pendingConflicts[currentIndex]

  if (!isOpen || !currentConflict) {
    return null
  }

  const handleResolve = async (resolution: ConflictResolution) => {
    setResolving(true)
    try {
      await resolveConflict(currentConflict.id, resolution)

      // Move to next conflict or close
      if (currentIndex >= pendingConflicts.length - 1) {
        setCurrentIndex(0)
        onAllResolved?.()
      } else {
        setCurrentIndex(currentIndex + 1)
      }
    } catch (error) {
      console.error('Failed to resolve conflict:', error)
      alert('Failed to resolve conflict. Please try again.')
    } finally {
      setResolving(false)
    }
  }

  const handleSkip = () => {
    if (currentIndex < pendingConflicts.length - 1) {
      setCurrentIndex(currentIndex + 1)
    } else {
      setCurrentIndex(0)
    }
  }

  const handleResolveAll = async (resolution: ConflictResolution) => {
    setResolving(true)
    try {
      for (const conflict of pendingConflicts) {
        await resolveConflict(conflict.id, resolution)
      }
      setCurrentIndex(0)
      onAllResolved?.()
    } catch (error) {
      console.error('Failed to resolve all conflicts:', error)
      alert('Failed to resolve some conflicts. Please try again.')
    } finally {
      setResolving(false)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && clearConflicts()}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-yellow-500" />
                Sync Conflicts Detected
              </DialogTitle>
              <DialogDescription>
                Changes were made on multiple devices. Please choose how to resolve.
              </DialogDescription>
            </div>
            <Badge variant="secondary">
              {currentIndex + 1} of {pendingConflicts.length}
            </Badge>
          </div>
        </DialogHeader>

        <div className="space-y-4 mt-4">
          <ConflictItem
            conflict={currentConflict}
            onResolve={handleResolve}
            resolving={resolving}
          />

          <div className="flex justify-between items-center pt-4 border-t">
            <Button
              variant="outline"
              onClick={handleSkip}
              disabled={resolving || pendingConflicts.length === 1}
            >
              Skip to Next
            </Button>

            <div className="flex gap-2">
              {pendingConflicts.length > 1 && (
                <>
                  <Button
                    variant="outline"
                    onClick={() => handleResolveAll('local')}
                    disabled={resolving}
                  >
                    Keep All Local
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => handleResolveAll('remote')}
                    disabled={resolving}
                  >
                    Keep All Remote
                  </Button>
                </>
              )}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

interface ConflictItemProps {
  conflict: Conflict
  onResolve: (resolution: ConflictResolution) => Promise<void>
  resolving: boolean
}

function ConflictItem({ conflict, onResolve, resolving }: ConflictItemProps) {
  const getEntityIcon = () => {
    switch (conflict.entityType) {
      case 'connection':
        return <Database className="h-4 w-4" />
      case 'saved_query':
        return <FileText className="h-4 w-4" />
      default:
        return null
    }
  }

  const getEntityName = () => {
    if (conflict.entityType === 'connection') {
      return (conflict.localVersion as Connection)?.name || 'Unknown Connection'
    }
    if (conflict.entityType === 'saved_query') {
      return (conflict.localVersion as SavedQuery)?.title || 'Unknown Query'
    }
    return conflict.entityId
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            {getEntityIcon()}
            {getEntityName()}
          </CardTitle>
          <CardDescription>{conflict.reason}</CardDescription>
        </CardHeader>
      </Card>

      <div className="grid grid-cols-2 gap-4">
        {/* Local Version */}
        <VersionCard
          title="Local Version"
          version={conflict.localVersion}
          updatedAt={conflict.localUpdatedAt}
          syncVersion={conflict.localSyncVersion}
          isRecommended={conflict.recommendedResolution === 'local'}
        />

        {/* Remote Version */}
        <VersionCard
          title="Remote Version"
          version={conflict.remoteVersion}
          updatedAt={conflict.remoteUpdatedAt}
          syncVersion={conflict.remoteSyncVersion}
          isRecommended={conflict.recommendedResolution === 'remote'}
        />
      </div>

      {/* Resolution Buttons */}
      <div className="flex gap-2 justify-center pt-2">
        <Button
          onClick={() => onResolve('local')}
          disabled={resolving}
          variant={conflict.recommendedResolution === 'local' ? 'default' : 'outline'}
          className="flex items-center gap-2"
        >
          <CheckCircle className="h-4 w-4" />
          Keep Local
        </Button>

        <Button
          onClick={() => onResolve('remote')}
          disabled={resolving}
          variant={conflict.recommendedResolution === 'remote' ? 'default' : 'outline'}
          className="flex items-center gap-2"
        >
          <CheckCircle className="h-4 w-4" />
          Keep Remote
        </Button>

        <Button
          onClick={() => onResolve('keep-both')}
          disabled={resolving}
          variant="outline"
          className="flex items-center gap-2"
        >
          <FileText className="h-4 w-4" />
          Keep Both
        </Button>
      </div>
    </div>
  )
}

interface VersionCardProps {
  title: string
  version: unknown
  updatedAt: Date
  syncVersion: number
  isRecommended: boolean
}

function VersionCard({ title, version, updatedAt, syncVersion, isRecommended }: VersionCardProps) {
  const formatDate = (date: Date) => {
    return new Intl.DateTimeFormat('en-US', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(date)
  }

  return (
    <Card className={isRecommended ? 'border-primary' : ''}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {isRecommended && (
            <Badge variant="default" className="text-xs">
              Recommended
            </Badge>
          )}
        </div>
        <CardDescription className="flex items-center gap-1 text-xs">
          <Clock className="h-3 w-3" />
          {formatDate(updatedAt)}
          <span className="ml-2 text-muted-foreground">v{syncVersion}</span>
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="bg-muted p-3 rounded-md max-h-64 overflow-y-auto">
          <pre className="text-xs font-mono whitespace-pre-wrap break-words">
            {JSON.stringify(version, null, 2)}
          </pre>
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Conflict Count Badge
 * Small badge showing number of conflicts
 */
export function ConflictBadge() {
  const conflicts = useSyncStore((state) => state.pendingConflicts)

  if (conflicts.length === 0) {
    return null
  }

  return (
    <Badge variant="destructive" className="ml-2">
      <AlertTriangle className="h-3 w-3 mr-1" />
      {conflicts.length} {conflicts.length === 1 ? 'Conflict' : 'Conflicts'}
    </Badge>
  )
}
