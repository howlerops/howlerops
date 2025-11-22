/**
 * Pending Invitations Page
 *
 * Lists all pending organization invitations for the logged-in user.
 * Route: /invitations
 *
 * Features:
 * - List all pending invitations
 * - Show org details, inviter, role, expiration
 * - Accept/Decline actions for each invitation
 * - Empty state when no invitations
 * - Filters out expired invitations
 * - Auto-refresh after actions
 */

import {
  AlertCircle,
  Calendar,
  CheckCircle,
  Inbox,
  Loader2,
  Mail,
  Shield,
  Users,
  XCircle,
} from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useOrganizationInvitations } from '@/store/organization-store'
import type { OrganizationInvitation } from '@/types/organization'
import { formatRelativeTime,isInvitationValid } from '@/types/organization'

export function PendingInvitationsPage() {
  const navigate = useNavigate()
  const {
    pendingInvitations,
    fetchPendingInvitations,
    acceptInvitation,
    declineInvitation,
    loading,
  } = useOrganizationInvitations()

  const [actionLoading, setActionLoading] = useState<Record<string, boolean>>({})
  const [error, setError] = useState<string | null>(null)

  const loadInvitations = useCallback(async () => {
    setError(null)
    try {
      await fetchPendingInvitations()
    } catch {
      setError('Failed to load invitations. Please try again.')
    }
  }, [fetchPendingInvitations])

  // Fetch invitations on mount
  useEffect(() => {
    loadInvitations()
  }, [loadInvitations])

  const handleAccept = async (invitation: OrganizationInvitation) => {
    setActionLoading({ ...actionLoading, [invitation.id]: true })

    try {
      await acceptInvitation(invitation.id)

      toast.success('Invitation accepted!', {
        description: invitation.organization
          ? `You are now a member of ${invitation.organization.name}`
          : 'You have joined the organization',
      })

      // Refresh list
      await loadInvitations()

      // Navigate to organization if we have the ID
      if (invitation.organization_id) {
        setTimeout(() => {
          navigate(`/dashboard?org=${invitation.organization_id}`)
        }, 1000)
      }
    } catch (err) {
      toast.error('Failed to accept invitation', {
        description: err instanceof Error ? err.message : 'Please try again',
      })
    } finally {
      setActionLoading({ ...actionLoading, [invitation.id]: false })
    }
  }

  const handleDecline = async (invitation: OrganizationInvitation) => {
    setActionLoading({ ...actionLoading, [invitation.id]: true })

    try {
      await declineInvitation(invitation.id)

      toast.success('Invitation declined')

      // Refresh list
      await loadInvitations()
    } catch (err) {
      toast.error('Failed to decline invitation', {
        description: err instanceof Error ? err.message : 'Please try again',
      })
    } finally {
      setActionLoading({ ...actionLoading, [invitation.id]: false })
    }
  }

  // Filter valid (non-expired, non-accepted) invitations
  const validInvitations = pendingInvitations.filter(
    (inv) => !inv.accepted_at && isInvitationValid(inv)
  )

  const expiredInvitations = pendingInvitations.filter(
    (inv) => !inv.accepted_at && !isInvitationValid(inv)
  )

  // Loading state
  if (loading && pendingInvitations.length === 0) {
    return (
      <div className="container mx-auto max-w-4xl p-4 py-8">
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <p className="mt-4 text-sm text-muted-foreground">
              Loading invitations...
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="container mx-auto max-w-4xl p-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold">Pending Invitations</h1>
        <p className="mt-2 text-muted-foreground">
          Manage your organization invitations
        </p>
      </div>

      {/* Error Alert */}
      {error && (
        <Alert variant="destructive" className="mb-6">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription className="flex items-center justify-between">
            <span>{error}</span>
            <Button
              variant="outline"
              size="sm"
              onClick={loadInvitations}
              className="ml-4"
            >
              Retry
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Empty State */}
      {validInvitations.length === 0 && expiredInvitations.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <div className="rounded-full bg-muted p-4">
              <Inbox className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">No pending invitations</h3>
            <p className="mt-2 text-center text-sm text-muted-foreground">
              You don't have any pending organization invitations at the moment.
            </p>
            <Button
              variant="outline"
              className="mt-6"
              onClick={() => navigate('/dashboard')}
            >
              Go to Dashboard
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Valid Invitations */}
      {validInvitations.length > 0 && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">
              Active Invitations ({validInvitations.length})
            </h2>
            <Button
              variant="ghost"
              size="sm"
              onClick={loadInvitations}
              disabled={loading}
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                'Refresh'
              )}
            </Button>
          </div>

          {validInvitations.map((invitation) => (
            <Card key={invitation.id} className="overflow-hidden">
              <CardHeader className="bg-muted/50 pb-4">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                  <div className="flex-1">
                    <CardTitle className="flex items-center gap-2">
                      <Users className="h-5 w-5" />
                      {invitation.organization?.name || 'Organization'}
                    </CardTitle>
                    {invitation.organization?.description && (
                      <CardDescription className="mt-2">
                        {invitation.organization.description}
                      </CardDescription>
                    )}
                  </div>
                  <Badge variant="secondary" className="self-start">
                    <Shield className="mr-1 h-3 w-3" />
                    {invitation.role}
                  </Badge>
                </div>
              </CardHeader>

              <CardContent className="pt-6">
                <div className="mb-4 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                  {/* Inviter Email */}
                  <div className="flex items-center gap-2">
                    <Mail className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">
                      Invited by: <strong>{invitation.email}</strong>
                    </span>
                  </div>

                  {/* Member Count */}
                  {invitation.organization?.member_count !== undefined && (
                    <div className="flex items-center gap-2">
                      <Users className="h-4 w-4 text-muted-foreground" />
                      <span className="text-muted-foreground">
                        {invitation.organization.member_count} member
                        {invitation.organization.member_count !== 1 ? 's' : ''}
                      </span>
                    </div>
                  )}

                  {/* Created Date */}
                  <div className="flex items-center gap-2">
                    <Calendar className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">
                      Sent {formatRelativeTime(new Date(invitation.created_at))}
                    </span>
                  </div>

                  {/* Expiration */}
                  <div className="flex items-center gap-2">
                    <Calendar className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">
                      Expires {formatRelativeTime(new Date(invitation.expires_at))}
                    </span>
                  </div>
                </div>

                {/* Actions */}
                <div className="flex flex-col gap-2 sm:flex-row">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleDecline(invitation)}
                    disabled={actionLoading[invitation.id]}
                    className="w-full sm:w-auto"
                  >
                    {actionLoading[invitation.id] ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <XCircle className="mr-2 h-4 w-4" />
                    )}
                    Decline
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => handleAccept(invitation)}
                    disabled={actionLoading[invitation.id]}
                    className="w-full flex-1 sm:w-auto"
                  >
                    {actionLoading[invitation.id] ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <CheckCircle className="mr-2 h-4 w-4" />
                    )}
                    Accept Invitation
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Expired Invitations */}
      {expiredInvitations.length > 0 && (
        <div className="mt-8 space-y-4">
          <h2 className="text-lg font-semibold text-muted-foreground">
            Expired Invitations ({expiredInvitations.length})
          </h2>

          {expiredInvitations.map((invitation) => (
            <Card
              key={invitation.id}
              className="border-muted bg-muted/20 opacity-60"
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2 text-base">
                      <Users className="h-4 w-4" />
                      {invitation.organization?.name || 'Organization'}
                    </CardTitle>
                    {invitation.organization?.description && (
                      <CardDescription className="mt-1 text-xs">
                        {invitation.organization.description}
                      </CardDescription>
                    )}
                  </div>
                  <Badge variant="outline" className="text-muted-foreground">
                    Expired
                  </Badge>
                </div>
              </CardHeader>

              <CardContent>
                <p className="text-sm text-muted-foreground">
                  This invitation expired{' '}
                  {formatRelativeTime(new Date(invitation.expires_at))}.
                  Contact the organization admin for a new invitation.
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
