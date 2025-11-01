/**
 * Invitation Accept Page
 *
 * Landing page for accepting organization invitations via magic link.
 * Route: /invite/:token
 *
 * Features:
 * - Fetches invitation details by token
 * - Shows organization preview
 * - Accept/Decline actions
 * - Handles expired/already-accepted invitations
 * - Redirects to signup if not authenticated
 */

import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuthStore } from '@/store/auth-store'
import { useOrganizationStore } from '@/store/organization-store'
import { authFetch, AuthApiError } from '@/lib/api/auth-client'
import { OrganizationRole } from '@/types/organization'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import {
  AlertCircle,
  CheckCircle,
  Users,
  Mail,
  Calendar,
  Shield,
  Loader2
} from 'lucide-react'

interface InvitationDetails {
  id: string
  organization_id: string
  email: string
  role: OrganizationRole
  invited_by: string
  token: string
  expires_at: Date
  accepted_at?: Date | null
  created_at: Date
  organization: {
    id: string
    name: string
    description?: string
    member_count?: number
  }
  inviter: {
    email: string
    username: string
  }
}

export function InviteAcceptPage() {
  const { token } = useParams<{ token: string }>()
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const acceptInvitation = useOrganizationStore((state) => state.acceptInvitation)
  const declineInvitation = useOrganizationStore((state) => state.declineInvitation)

  const [invitation, setInvitation] = useState<InvitationDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [accepting, setAccepting] = useState(false)
  const [declining, setDeclining] = useState(false)

  const fetchInvitationDetails = React.useCallback(async () => {
    if (!token) return

    setLoading(true)
    setError(null)

    try {
      const response = await authFetch<{ invitation: InvitationDetails }>(
        `/api/invitations/${token}`,
        { skipAuth: !isAuthenticated } // Allow unauthenticated access
      )

      setInvitation(response.invitation)
    } catch (err) {
      const errorMessage = err instanceof AuthApiError
        ? err.message
        : 'Failed to load invitation details'
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }, [token, isAuthenticated])

  // Fetch invitation details on mount
  useEffect(() => {
    if (!token) {
      setError('Invalid invitation link')
      setLoading(false)
      return
    }

    fetchInvitationDetails()
  }, [token, fetchInvitationDetails])

  const handleAccept = async () => {
    if (!invitation) return

    // Redirect to signup if not authenticated
    if (!isAuthenticated) {
      toast.info('Please sign in to accept this invitation')
      // Store token in session storage to redirect back after login
      sessionStorage.setItem('pendingInvitationToken', token || '')
      navigate('/signup')
      return
    }

    setAccepting(true)

    try {
      await acceptInvitation(invitation.id)

      toast.success('Invitation accepted!', {
        description: `You are now a member of ${invitation.organization.name}`,
      })

      // Redirect to organization
      navigate(`/dashboard?org=${invitation.organization_id}`)
    } catch (err) {
      const errorMessage = err instanceof AuthApiError
        ? err.message
        : 'Failed to accept invitation'
      toast.error('Failed to accept invitation', {
        description: errorMessage,
      })
    } finally {
      setAccepting(false)
    }
  }

  const handleDecline = async () => {
    if (!invitation) return

    setDeclining(true)

    try {
      await declineInvitation(invitation.id)

      toast.success('Invitation declined')

      // Redirect to dashboard or home
      navigate('/')
    } catch (err) {
      const errorMessage = err instanceof AuthApiError
        ? err.message
        : 'Failed to decline invitation'
      toast.error('Failed to decline invitation', {
        description: errorMessage,
      })
    } finally {
      setDeclining(false)
    }
  }

  // Loading state
  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-lg">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <p className="mt-4 text-sm text-muted-foreground">
              Loading invitation...
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Error state
  if (error || !invitation) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-lg">
          <CardHeader>
            <div className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-destructive" />
              <CardTitle>Invitation Not Found</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>
                {error || 'This invitation link is invalid or has expired.'}
              </AlertDescription>
            </Alert>
          </CardContent>
          <CardFooter>
            <Button onClick={() => navigate('/')} className="w-full">
              Go to Dashboard
            </Button>
          </CardFooter>
        </Card>
      </div>
    )
  }

  // Already accepted
  if (invitation.accepted_at) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-lg">
          <CardHeader>
            <div className="flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-600" />
              <CardTitle>Already Accepted</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <Alert>
              <CheckCircle className="h-4 w-4" />
              <AlertTitle>Invitation Already Accepted</AlertTitle>
              <AlertDescription>
                You have already accepted this invitation to{' '}
                <strong>{invitation.organization.name}</strong>.
              </AlertDescription>
            </Alert>
          </CardContent>
          <CardFooter>
            <Button onClick={() => navigate('/dashboard')} className="w-full">
              Go to Dashboard
            </Button>
          </CardFooter>
        </Card>
      </div>
    )
  }

  // Expired invitation (already checked accepted_at above)
  const isExpired = new Date() >= new Date(invitation.expires_at)
  if (isExpired) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-lg">
          <CardHeader>
            <div className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-amber-600" />
              <CardTitle>Invitation Expired</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>This Invitation Has Expired</AlertTitle>
              <AlertDescription>
                This invitation to <strong>{invitation.organization.name}</strong>{' '}
                expired on{' '}
                {new Date(invitation.expires_at).toLocaleDateString()}.
                <br />
                Please contact the organization admin for a new invitation.
              </AlertDescription>
            </Alert>
          </CardContent>
          <CardFooter>
            <Button onClick={() => navigate('/')} className="w-full">
              Go to Dashboard
            </Button>
          </CardFooter>
        </Card>
      </div>
    )
  }

  // Valid invitation - show accept/decline UI
  return (
    <div className="flex min-h-screen items-center justify-center p-4 bg-muted/30">
      <Card className="w-full max-w-2xl">
        <CardHeader className="text-center">
          <CardTitle className="text-3xl">You're Invited!</CardTitle>
          <CardDescription>
            You've been invited to join an organization
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Organization Details */}
          <div className="rounded-lg border bg-card p-6">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <h3 className="text-xl font-semibold">
                  {invitation.organization.name}
                </h3>
                {invitation.organization.description && (
                  <p className="mt-2 text-sm text-muted-foreground">
                    {invitation.organization.description}
                  </p>
                )}
              </div>
              <Badge variant="secondary" className="ml-4">
                <Shield className="mr-1 h-3 w-3" />
                {invitation.role}
              </Badge>
            </div>

            <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
              {/* Member Count */}
              <div className="flex items-center gap-2 text-sm">
                <Users className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">
                  {invitation.organization.member_count || 0} member
                  {invitation.organization.member_count !== 1 ? 's' : ''}
                </span>
              </div>

              {/* Inviter */}
              <div className="flex items-center gap-2 text-sm">
                <Mail className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">
                  Invited by {invitation.inviter.username}
                </span>
              </div>

              {/* Expiration */}
              <div className="flex items-center gap-2 text-sm sm:col-span-2">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">
                  Expires on {new Date(invitation.expires_at).toLocaleDateString()}
                </span>
              </div>
            </div>
          </div>

          {/* Role Description */}
          <Alert>
            <Shield className="h-4 w-4" />
            <AlertTitle>Your Role: {invitation.role}</AlertTitle>
            <AlertDescription>
              {invitation.role === 'admin' && (
                'As an admin, you can manage members, invitations, and organization settings.'
              )}
              {invitation.role === 'member' && (
                'As a member, you can view and collaborate on organization resources.'
              )}
            </AlertDescription>
          </Alert>

          {!isAuthenticated && (
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Sign In Required</AlertTitle>
              <AlertDescription>
                You need to sign in or create an account to accept this invitation.
              </AlertDescription>
            </Alert>
          )}
        </CardContent>

        <CardFooter className="flex flex-col gap-3 sm:flex-row">
          <Button
            onClick={handleDecline}
            variant="outline"
            className="w-full sm:w-auto"
            disabled={accepting || declining}
          >
            {declining && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Decline
          </Button>
          <Button
            onClick={handleAccept}
            className="w-full flex-1"
            disabled={accepting || declining}
          >
            {accepting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isAuthenticated ? 'Accept Invitation' : 'Sign In to Accept'}
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
