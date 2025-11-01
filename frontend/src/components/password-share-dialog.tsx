/**
 * Password Share Dialog Component
 *
 * Dialog for approving/denying password sharing between browser tabs.
 * Shows connection names, provides approve/deny buttons, and displays
 * progress and feedback.
 *
 * Features:
 * - Connection list with names
 * - Approve/Deny actions
 * - Progress indicator
 * - Success/failure feedback
 * - Auto-dismiss after success
 *
 * Usage:
 * ```tsx
 * <PasswordShareDialog
 *   request={passwordShareRequest}
 *   onApprove={approvePasswordShare}
 *   onDeny={denyPasswordShare}
 *   connections={connections}
 * />
 * ```
 */

import React, { useState, useEffect, useMemo, useCallback } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import {
  Key,
  Shield,
  Check,
  X,
  AlertTriangle,
  Clock,
  Lock
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { getPasswordTransferManager, type PasswordData } from '@/lib/sync/password-transfer'
import type { DatabaseConnection } from '@/store/connection-store'

export interface PasswordShareRequest {
  connectionIds: string[]
  requesterId: string
  timestamp: number
}

export interface PasswordShareDialogProps {
  /**
   * Password share request (if any)
   */
  request: PasswordShareRequest | null

  /**
   * Callback to approve the request
   */
  onApprove: (passwords: PasswordData[]) => Promise<void>

  /**
   * Callback to deny the request
   */
  onDeny: () => void

  /**
   * Available connections to show names
   */
  connections?: DatabaseConnection[]

  /**
   * Custom className
   */
  className?: string
}

type DialogState = 'pending' | 'approving' | 'success' | 'error'

/**
 * Password Share Dialog Component
 */
export function PasswordShareDialog({
  request,
  onApprove,
  onDeny,
  connections = [],
  className
}: PasswordShareDialogProps) {
  const [dialogState, setDialogState] = useState<DialogState>('pending')
  const [error, setError] = useState<string | null>(null)
  const [progress, setProgress] = useState(0)

  // Reset state when request changes
  useEffect(() => {
    if (request) {
      setDialogState('pending')
      setError(null)
      setProgress(0)
    }
  }, [request])

  const handleClose = useCallback(() => {
    if (dialogState === 'approving') {
      return
    }

    onDeny()
  }, [dialogState, onDeny])

  // Auto-dismiss after success
  useEffect(() => {
    if (dialogState === 'success') {
      const timer = setTimeout(() => {
        handleClose()
      }, 2000)

      return () => clearTimeout(timer)
    }
    return undefined
  }, [dialogState, handleClose])

  // Get connection names
  const requestedConnections = useMemo(() => {
    if (!request) return []

    return request.connectionIds.map(id => {
      const connection = connections.find(c => c.id === id)
      return {
        id,
        name: connection?.name || 'Unknown Connection',
        type: connection?.type || 'unknown'
      }
    })
  }, [request, connections])

  // Calculate time since request
  const timeSinceRequest = useMemo(() => {
    if (!request) return ''

    const seconds = Math.floor((Date.now() - request.timestamp) / 1000)
    if (seconds < 60) return `${seconds}s ago`
    const minutes = Math.floor(seconds / 60)
    return `${minutes}m ago`
  }, [request])

  /**
   * Handle approve action
   */
  const handleApprove = async () => {
    if (!request) return

    setDialogState('approving')
    setProgress(10)

    try {
      // Get passwords for requested connections
      const passwordTransfer = getPasswordTransferManager()
      const passwords = await passwordTransfer.getPasswordsForConnections(request.connectionIds)

      setProgress(50)

      // Send passwords to requesting tab
      await onApprove(passwords)

      setProgress(100)
      setDialogState('success')
    } catch (err) {
      console.error('[PasswordShareDialog] Failed to approve request:', err)
      setError(err instanceof Error ? err.message : 'Failed to share passwords')
      setDialogState('error')
    }
  }

  /**
   * Handle deny action
   */
  const handleDeny = () => {
    onDeny()
    handleClose()
  }

  // Don't render if no request
  if (!request) {
    return null
  }

  const isOpen = Boolean(request)

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className={cn('sm:max-w-md', className)}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="h-5 w-5 text-primary" />
            Password Share Request
          </DialogTitle>
          <DialogDescription>
            Another tab is requesting access to connection passwords
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Security notice */}
          <Alert>
            <Shield className="h-4 w-4" />
            <AlertDescription className="text-xs">
              Passwords are encrypted in transit using ephemeral AES-256 keys that expire after 10 seconds.
            </AlertDescription>
          </Alert>

          {/* Request info */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Requested:</span>
            <div className="flex items-center gap-1 text-muted-foreground">
              <Clock className="h-3 w-3" />
              {timeSinceRequest}
            </div>
          </div>

          {/* Connection list */}
          <div className="space-y-2">
            <div className="text-sm font-medium">Connections ({requestedConnections.length})</div>
            <div className="max-h-48 overflow-y-auto space-y-2 rounded-md border p-3">
              {requestedConnections.map((conn) => (
                <div
                  key={conn.id}
                  className="flex items-center justify-between p-2 rounded bg-muted/50"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Lock className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <span className="text-sm font-medium truncate">{conn.name}</span>
                  </div>
                  <Badge variant="outline" className="text-xs flex-shrink-0">
                    {conn.type}
                  </Badge>
                </div>
              ))}
            </div>
          </div>

          {/* Progress indicator */}
          {dialogState === 'approving' && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Clock className="h-4 w-4 animate-spin" />
                Encrypting and sharing passwords...
              </div>
              <Progress value={progress} className="h-2" />
            </div>
          )}

          {/* Success message */}
          {dialogState === 'success' && (
            <Alert className="bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-800">
              <Check className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-800 dark:text-green-200">
                Passwords shared successfully!
              </AlertDescription>
            </Alert>
          )}

          {/* Error message */}
          {dialogState === 'error' && error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {/* Warning */}
          {dialogState === 'pending' && (
            <div className="flex items-start gap-2 p-3 rounded-md bg-yellow-50 dark:bg-yellow-950/20 border border-yellow-200 dark:border-yellow-800">
              <AlertTriangle className="h-4 w-4 text-yellow-600 mt-0.5 flex-shrink-0" />
              <p className="text-xs text-yellow-800 dark:text-yellow-200">
                Only approve if you recognize the requesting tab. Passwords will be encrypted
                but visible in the other tab's memory.
              </p>
            </div>
          )}
        </div>

        <DialogFooter className="flex-col sm:flex-row gap-2">
          {dialogState === 'pending' && (
            <>
              <Button
                variant="outline"
                onClick={handleDeny}
                className="w-full sm:w-auto"
              >
                <X className="h-4 w-4 mr-2" />
                Deny
              </Button>
              <Button
                onClick={handleApprove}
                className="w-full sm:w-auto"
              >
                <Check className="h-4 w-4 mr-2" />
                Approve & Share
              </Button>
            </>
          )}

          {dialogState === 'error' && (
            <Button
              variant="outline"
              onClick={handleClose}
              className="w-full sm:w-auto"
            >
              Close
            </Button>
          )}

          {dialogState === 'success' && (
            <div className="flex items-center justify-center w-full text-sm text-muted-foreground">
              Closing automatically...
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Hook to automatically show password share dialog
 */
export function usePasswordShareDialog(connections: DatabaseConnection[]) {
  const [isOpen, setIsOpen] = useState(false)

  // This would be connected to the multi-tab sync hook
  // For now, it's a placeholder for the integration

  return {
    isOpen,
    setIsOpen
  }
}
