/**
 * Sync Status Indicator Component
 *
 * Visual indicator showing sync status, progress, and controls.
 * Displays in toolbar/header for real-time sync feedback.
 *
 * @module components/sync/sync-indicator
 */

import { useState, useEffect } from 'react'
import { useSyncStatus, useSyncActions } from '@/store/sync-store'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Cloud,
  CloudOff,
  RefreshCw,
  AlertCircle,
  CheckCircle2,
  Clock,
  Settings,
  WifiOff,
  Loader2,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { cn } from '@/lib/utils'

/**
 * Main sync indicator component
 */
export function SyncIndicator() {
  const {
    status,
    isSyncing,
    lastSyncAt,
    syncEnabled,
    hasConflicts,
    conflictCount,
    hasError,
    errorMessage,
    progress,
  } = useSyncStatus()

  const { syncNow } = useSyncActions()
  const [syncing, setSyncing] = useState(false)
  const [isOnline, setIsOnline] = useState(navigator.onLine)

  // Monitor online status
  useEffect(() => {
    const handleOnline = () => setIsOnline(true)
    const handleOffline = () => setIsOnline(false)

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [])

  const handleSync = async () => {
    setSyncing(true)
    try {
      await syncNow()
    } catch (error) {
      console.error('Manual sync failed:', error)
    } finally {
      setSyncing(false)
    }
  }

  const getStatusIcon = () => {
    if (!syncEnabled) {
      return <CloudOff className="h-4 w-4 text-muted-foreground" />
    }

    if (!isOnline) {
      return <WifiOff className="h-4 w-4 text-orange-500" />
    }

    if (isSyncing || syncing) {
      return <RefreshCw className="h-4 w-4 animate-spin text-blue-500" />
    }

    if (hasError) {
      return <AlertCircle className="h-4 w-4 text-red-500" />
    }

    if (hasConflicts) {
      return <AlertCircle className="h-4 w-4 text-yellow-500" />
    }

    return <Cloud className="h-4 w-4 text-green-500" />
  }

  const getStatusText = () => {
    if (!syncEnabled) {
      return 'Sync disabled'
    }

    if (!isOnline) {
      return 'Offline'
    }

    if (isSyncing) {
      if (progress) {
        return `Syncing: ${progress.phase} (${progress.percentage}%)`
      }
      return 'Syncing...'
    }

    if (hasError) {
      return `Error: ${errorMessage?.substring(0, 30)}...`
    }

    if (hasConflicts) {
      return `${conflictCount} conflict${conflictCount > 1 ? 's' : ''}`
    }

    if (lastSyncAt) {
      return `Synced ${formatDistanceToNow(lastSyncAt, { addSuffix: true })}`
    }

    return 'Not synced'
  }

  const getStatusColor = () => {
    if (!syncEnabled || !isOnline) return 'secondary'
    if (hasError) return 'destructive'
    if (hasConflicts) return 'warning'
    if (isSyncing) return 'default'
    return 'success'
  }

  return (
    <div className="flex items-center gap-2">
      {/* Status Icon */}
      <div className="flex items-center gap-2">
        {getStatusIcon()}
        <span className="text-xs text-muted-foreground">{getStatusText()}</span>
      </div>

      {/* Sync Button */}
      {syncEnabled && isOnline && !isSyncing && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleSync}
          disabled={syncing}
          className="h-7 px-2"
        >
          <RefreshCw className={`h-3 w-3 ${syncing ? 'animate-spin' : ''}`} />
        </Button>
      )}

      {/* Settings Dialog */}
      <SyncSettingsDialog />
    </div>
  )
}

/**
 * Compact sync indicator (for space-constrained areas)
 */
export function SyncIndicatorCompact() {
  const { status, isSyncing, syncEnabled, hasConflicts, hasError } = useSyncStatus()
  const [isOnline] = useState(navigator.onLine)

  const getIcon = () => {
    if (!syncEnabled) return <CloudOff className="h-4 w-4 text-muted-foreground" />
    if (!isOnline) return <WifiOff className="h-4 w-4 text-orange-500" />
    if (isSyncing) return <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
    if (hasError) return <AlertCircle className="h-4 w-4 text-red-500" />
    if (hasConflicts) return <AlertCircle className="h-4 w-4 text-yellow-500" />
    return <Cloud className="h-4 w-4 text-green-500" />
  }

  return (
    <button className="p-1 hover:bg-muted rounded" title="Sync status">
      {getIcon()}
    </button>
  )
}

/**
 * Sync settings dialog
 */
function SyncSettingsDialog() {
  const { syncEnabled, lastSyncAt, progress } = useSyncStatus()
  const { enableSync, disableSync, updateConfig } = useSyncActions()
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" className="h-7 px-2">
          <Settings className="h-3 w-3" />
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Sync Settings</DialogTitle>
          <DialogDescription>Configure cloud synchronization</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Enable/Disable */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Enable Sync</p>
              <p className="text-xs text-muted-foreground">
                Sync data across devices
              </p>
            </div>
            <Button
              variant={syncEnabled ? 'default' : 'outline'}
              size="sm"
              onClick={() => (syncEnabled ? disableSync() : enableSync())}
            >
              {syncEnabled ? 'Enabled' : 'Disabled'}
            </Button>
          </div>

          {/* Last Sync */}
          {lastSyncAt && (
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">Last Sync</p>
              <p className="text-sm text-muted-foreground">
                {formatDistanceToNow(lastSyncAt, { addSuffix: true })}
              </p>
            </div>
          )}

          {/* Progress */}
          {progress && (
            <div className="space-y-2">
              <p className="text-sm font-medium">Sync Progress</p>
              <div className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground capitalize">
                    {progress.phase}
                  </span>
                  <span>{progress.percentage}%</span>
                </div>
                <div className="w-full bg-muted rounded-full h-2">
                  <div
                    className="bg-primary h-2 rounded-full transition-all"
                    style={{ width: `${progress.percentage}%` }}
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  {progress.processedItems} / {progress.totalItems} items
                </p>
              </div>
            </div>
          )}

          {/* Sync Stats */}
          <div className="border-t pt-4">
            <p className="text-sm font-medium mb-2">Statistics</p>
            <div className="space-y-1 text-xs">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Status</span>
                <Badge
                  variant={syncEnabled ? 'secondary' : 'secondary'}
                  className={cn('text-xs', syncEnabled ? 'bg-green-100 text-green-700' : undefined)}
                >
                  {syncEnabled ? 'Active' : 'Inactive'}
                </Badge>
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Sync progress bar (standalone)
 */
export function SyncProgressBar() {
  const { progress, isSyncing } = useSyncStatus()

  if (!isSyncing || !progress) {
    return null
  }

  return (
    <div className="fixed bottom-4 right-4 w-80 bg-background border rounded-lg shadow-lg p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
          <span className="text-sm font-medium">Syncing</span>
        </div>
        <span className="text-xs text-muted-foreground">{progress.percentage}%</span>
      </div>

      <div className="w-full bg-muted rounded-full h-2">
        <div
          className="bg-blue-500 h-2 rounded-full transition-all duration-300"
          style={{ width: `${progress.percentage}%` }}
        />
      </div>

      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span className="capitalize">{progress.phase}</span>
        <span>
          {progress.processedItems} / {progress.totalItems} items
        </span>
      </div>

      {progress.currentItem && (
        <p className="text-xs text-muted-foreground truncate">
          {progress.currentItem}
        </p>
      )}
    </div>
  )
}

/**
 * Format relative time
 */
function formatRelativeTime(date: Date): string {
  try {
    return formatDistanceToNow(date, { addSuffix: true })
  } catch {
    return 'recently'
  }
}
