/**
 * SharedResourcesPage
 *
 * Organization onboarding page. Redirects users with an org to /connections?tab=team.
 * Shows org creation/join options for users without an organization.
 *
 * @module pages/SharedResourcesPage
 */

import { AlertCircle, Ticket, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

import { OrganizationCreateModal } from '@/components/organizations'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useOrganizationStore } from '@/store/organization-store'

/**
 * SharedResourcesPage Component
 *
 * Usage:
 * ```tsx
 * <Route path="/shared" element={<SharedResourcesPage />} />
 * ```
 */
export function SharedResourcesPage() {
  const navigate = useNavigate()

  // Modal states
  const [showCreateOrgModal, setShowCreateOrgModal] = useState(false)
  const [showJoinOrgModal, setShowJoinOrgModal] = useState(false)
  const [inviteCode, setInviteCode] = useState('')
  const [joinLoading, setJoinLoading] = useState(false)

  const {
    currentOrgId,
    organizations,
    createOrganization,
    loading: orgLoading,
    error: orgError,
  } = useOrganizationStore()

  // Redirect users with an org to the connections team tab
  useEffect(() => {
    if (currentOrgId) {
      navigate('/connections?tab=team', { replace: true })
    }
  }, [currentOrgId, navigate])

  // Handle creating organization
  const handleCreateOrganization = async (data: { name: string; description?: string }) => {
    try {
      const org = await createOrganization(data)
      toast.success(`Organization "${org.name}" created!`)
      setShowCreateOrgModal(false)
      // Will redirect via useEffect when currentOrgId updates
    } catch (error) {
      console.error('Failed to create organization:', error)
      throw error
    }
  }

  // Handle joining via invite code
  const handleJoinWithInviteCode = () => {
    if (!inviteCode.trim()) {
      toast.error('Please enter an invite code')
      return
    }

    setJoinLoading(true)

    const token = inviteCode.trim()

    setShowJoinOrgModal(false)
    setInviteCode('')
    setJoinLoading(false)

    navigate(`/invite/${token}`)
  }

  // If user has an org, show loading while redirecting
  if (currentOrgId) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent" />
          <p className="mt-4 text-muted-foreground">Redirecting to team resources...</p>
        </div>
      </div>
    )
  }

  const hasOrganizations = organizations.length > 0

  return (
    <div className="container mx-auto p-6 max-w-7xl">
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-2">
          <Users className="h-6 w-6 text-primary" />
          <h1 className="text-3xl font-bold">Team Sharing</h1>
        </div>
        <p className="text-muted-foreground">
          Share database connections and queries with your team
        </p>
      </div>

      <div className="text-center py-12 border-2 border-dashed rounded-lg">
        <Users className="h-12 w-12 mx-auto text-muted-foreground mb-4" />

        {hasOrganizations ? (
          <>
            <h3 className="text-lg font-semibold mb-2">
              Select an Organization
            </h3>
            <p className="text-muted-foreground mb-4 max-w-md mx-auto">
              You have access to {organizations.length} organization{organizations.length > 1 ? 's' : ''}.
              Select one from the organization switcher to view shared resources.
            </p>
            <Alert className="max-w-md mx-auto text-left">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>How to select an organization</AlertTitle>
              <AlertDescription>
                Use the organization switcher in the header or sidebar to
                switch between your organizations.
              </AlertDescription>
            </Alert>
          </>
        ) : (
          <>
            <h3 className="text-lg font-semibold mb-2">
              Personal Account
            </h3>
            <p className="text-muted-foreground mb-4 max-w-md mx-auto">
              You're using a personal account. To share connections and queries
              with teammates, create or join an organization.
            </p>
            <div className="flex gap-3 justify-center">
              <Button variant="default" onClick={() => setShowCreateOrgModal(true)}>
                Create Organization
              </Button>
              <Button variant="outline" onClick={() => setShowJoinOrgModal(true)}>
                <Ticket className="h-4 w-4 mr-2" />
                Join with Invite Code
              </Button>
            </div>
            <p className="text-xs text-muted-foreground mt-4">
              Organizations allow secure credential sharing with end-to-end encryption
            </p>
          </>
        )}
      </div>

      {/* Create Organization Modal */}
      <OrganizationCreateModal
        open={showCreateOrgModal}
        onOpenChange={setShowCreateOrgModal}
        onCreate={handleCreateOrganization}
        loading={orgLoading.creating}
        error={orgError}
      />

      {/* Join with Invite Code Modal */}
      <Dialog open={showJoinOrgModal} onOpenChange={setShowJoinOrgModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Ticket className="h-5 w-5" />
              Join Organization
            </DialogTitle>
            <DialogDescription>
              Enter the invite code you received to join an organization.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="invite-code">Invite Code</Label>
              <Input
                id="invite-code"
                placeholder="Enter invite code or paste invite link"
                value={inviteCode}
                onChange={(e) => setInviteCode(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleJoinWithInviteCode()
                  }
                }}
              />
              <p className="text-xs text-muted-foreground">
                The invite code looks like: abc123xyz or a full URL
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowJoinOrgModal(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleJoinWithInviteCode}
              disabled={!inviteCode.trim() || joinLoading}
            >
              {joinLoading ? 'Joining...' : 'Join Organization'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
