/**
 * Invitation Banner Component
 *
 * Displays a banner at the top of the app when user has pending invitations.
 * Features:
 * - Shows count of pending invitations
 * - Click to navigate to /invitations page
 * - Dismissible (don't show again for 24 hours)
 * - Uses localStorage for dismissal tracking
 * - Auto-fetches invitation count on mount
 */

import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useOrganizationInvitations } from '@/store/organization-store'
import { isInvitationValid } from '@/types/organization'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Mail, X } from 'lucide-react'

const DISMISSAL_KEY = 'invitation-banner-dismissed'
const DISMISSAL_DURATION = 24 * 60 * 60 * 1000 // 24 hours in milliseconds

export function InvitationBanner() {
  const navigate = useNavigate()
  const { pendingInvitations, fetchPendingInvitations } =
    useOrganizationInvitations()

  const [isDismissed, setIsDismissed] = useState(false)
  const [loading, setLoading] = useState(true)

  // Check if banner was dismissed and fetch invitations on mount
  useEffect(() => {
    const checkDismissalStatus = () => {
      const dismissedAt = localStorage.getItem(DISMISSAL_KEY)

      if (dismissedAt) {
        const dismissedTime = parseInt(dismissedAt, 10)
        const now = Date.now()

        // Check if 24 hours have passed since dismissal
        if (now - dismissedTime < DISMISSAL_DURATION) {
          setIsDismissed(true)
        } else {
          // Dismissal expired, clear it
          localStorage.removeItem(DISMISSAL_KEY)
        }
      }
    }

    const loadInvitations = async () => {
      try {
        await fetchPendingInvitations()
      } catch (err) {
        console.error('Failed to fetch pending invitations:', err)
      } finally {
        setLoading(false)
      }
    }

    checkDismissalStatus()
    loadInvitations()
  }, [fetchPendingInvitations])

  const handleDismiss = () => {
    localStorage.setItem(DISMISSAL_KEY, Date.now().toString())
    setIsDismissed(true)
  }

  const handleView = () => {
    navigate('/invitations')
  }

  // Filter for valid (non-expired, non-accepted) invitations
  const validInvitations = pendingInvitations.filter(
    (inv) => !inv.accepted_at && isInvitationValid(inv)
  )

  const invitationCount = validInvitations.length

  // Don't show if loading, dismissed, or no invitations
  if (loading || isDismissed || invitationCount === 0) {
    return null
  }

  return (
    <Alert className="rounded-none border-x-0 border-t-0 bg-blue-50 dark:bg-blue-950/20">
      <Mail className="h-4 w-4 text-blue-600 dark:text-blue-400" />
      <AlertDescription className="flex items-center justify-between gap-4">
        <div className="flex flex-1 items-center gap-2">
          <span className="text-sm font-medium text-blue-900 dark:text-blue-100">
            You have {invitationCount} pending invitation
            {invitationCount !== 1 ? 's' : ''}
          </span>
          <Button
            size="sm"
            variant="link"
            onClick={handleView}
            className="h-auto p-0 text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
          >
            View {invitationCount === 1 ? 'invitation' : 'all'}
          </Button>
        </div>

        <Button
          size="sm"
          variant="ghost"
          onClick={handleDismiss}
          className="h-6 w-6 p-0 text-blue-600 hover:bg-blue-100 hover:text-blue-700 dark:text-blue-400 dark:hover:bg-blue-900/30 dark:hover:text-blue-300"
          aria-label="Dismiss for 24 hours"
        >
          <X className="h-4 w-4" />
        </Button>
      </AlertDescription>
    </Alert>
  )
}
