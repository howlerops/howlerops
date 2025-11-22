/**
 * Invite Member Modal Component
 *
 * Modal for inviting new members to an organization.
 * Supports email validation, role selection, bulk invites (comma-separated emails),
 * and displays pending invitations.
 */

import { AlertCircle, Loader2, Mail, Send,UserPlus, X } from 'lucide-react'
import * as React from 'react'

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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type {
  CreateInvitationInput,
  OrganizationInvitation,
} from '@/types/organization'
import {
  formatRelativeTime,
  getRoleDisplayName,
  isValidEmail,
  OrganizationRole,
  OrganizationRole as OrgRole,
} from '@/types/organization'

interface InviteMemberModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onInvite: (invitations: CreateInvitationInput[]) => Promise<void>
  pendingInvitations?: OrganizationInvitation[]
  onRevokeInvitation?: (invitationId: string) => Promise<void>
  loading?: boolean
  error?: string | null
  /** Allow inviting admins (owner only) */
  canInviteAdmin?: boolean
  /** Current user's role - used to determine available role options */
  currentUserRole?: OrganizationRole
}

export function InviteMemberModal({
  open,
  onOpenChange,
  onInvite,
  pendingInvitations = [],
  onRevokeInvitation,
  loading = false,
  error = null,
  canInviteAdmin = false,
  currentUserRole,
}: InviteMemberModalProps) {
  // Only owners can invite admins
  const isOwner = currentUserRole === OrganizationRole.Owner
  const effectiveCanInviteAdmin = canInviteAdmin && isOwner
  const [emails, setEmails] = React.useState('')
  const [role, setRole] = React.useState<OrganizationRole>(OrgRole.Member)
  const [validationErrors, setValidationErrors] = React.useState<{
    emails?: string
  }>({})
  const [parsedEmails, setParsedEmails] = React.useState<string[]>([])

  // Reset form when modal opens/closes
  React.useEffect(() => {
    if (!open) {
      setEmails('')
      setRole(OrgRole.Member)
      setValidationErrors({})
      setParsedEmails([])
    }
  }, [open])

  // Parse and validate emails as user types
  React.useEffect(() => {
    if (!emails.trim()) {
      setParsedEmails([])
      setValidationErrors({})
      return
    }

    const emailList = emails
      .split(',')
      .map((e) => e.trim())
      .filter((e) => e.length > 0)

    setParsedEmails(emailList)

    // Validate emails
    const invalidEmails = emailList.filter((email) => !isValidEmail(email))
    if (invalidEmails.length > 0) {
      setValidationErrors({
        emails: `Invalid email${invalidEmails.length > 1 ? 's' : ''}: ${invalidEmails.join(', ')}`,
      })
    } else {
      setValidationErrors({})
    }
  }, [emails])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (parsedEmails.length === 0) {
      setValidationErrors({ emails: 'Please enter at least one email address' })
      return
    }

    if (validationErrors.emails) {
      return
    }

    const invitations: CreateInvitationInput[] = parsedEmails.map((email) => ({
      email,
      role: role as OrgRole.Admin | OrgRole.Member,
    }))

    try {
      await onInvite(invitations)
      // Modal will be closed by parent component on success
    } catch (err) {
      console.error('Failed to send invitations:', err)
    }
  }

  const handleRevokeInvitation = async (invitationId: string) => {
    if (!onRevokeInvitation) return

    try {
      await onRevokeInvitation(invitationId)
    } catch (err) {
      console.error('Failed to revoke invitation:', err)
    }
  }

  const isValid = parsedEmails.length > 0 && !validationErrors.emails

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-3 mb-2">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
              <UserPlus className="h-5 w-5 text-white" />
            </div>
            <div>
              <DialogTitle>Invite Members</DialogTitle>
            </div>
          </div>
          <DialogDescription>
            Invite team members to join your organization via email
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col flex-1 overflow-hidden">
          <div className="space-y-4 py-4 flex-shrink-0">
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <Label htmlFor="emails">
                Email Addresses <span className="text-destructive">*</span>
              </Label>
              <Input
                id="emails"
                type="text"
                placeholder="john@example.com, jane@example.com"
                value={emails}
                onChange={(e) => setEmails(e.target.value)}
                disabled={loading}
                aria-invalid={!!validationErrors.emails}
                aria-describedby={validationErrors.emails ? 'emails-error' : 'emails-help'}
              />
              {validationErrors.emails ? (
                <p id="emails-error" className="text-sm text-destructive">
                  {validationErrors.emails}
                </p>
              ) : (
                <p id="emails-help" className="text-xs text-muted-foreground">
                  Enter one or more email addresses separated by commas
                </p>
              )}

              {parsedEmails.length > 0 && !validationErrors.emails && (
                <div className="flex flex-wrap gap-2 mt-2">
                  {parsedEmails.map((email, index) => (
                    <Badge key={index} variant="secondary" className="gap-1">
                      <Mail className="h-3 w-3" />
                      {email}
                    </Badge>
                  ))}
                </div>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="role">
                Role <span className="text-destructive">*</span>
              </Label>
              <Select
                value={role}
                onValueChange={(value) => setRole(value as OrganizationRole)}
                disabled={loading}
              >
                <SelectTrigger id="role">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {effectiveCanInviteAdmin && (
                    <SelectItem value={OrgRole.Admin}>
                      <div className="flex flex-col items-start">
                        <span className="font-medium">{getRoleDisplayName(OrgRole.Admin)}</span>
                        <span className="text-xs text-muted-foreground">
                          Can manage members and settings
                        </span>
                      </div>
                    </SelectItem>
                  )}
                  <SelectItem value={OrgRole.Member}>
                    <div className="flex flex-col items-start">
                      <span className="font-medium">{getRoleDisplayName(OrgRole.Member)}</span>
                      <span className="text-xs text-muted-foreground">
                        Basic access to resources
                      </span>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Pending Invitations */}
          {pendingInvitations.length > 0 && (
            <div className="border-t pt-4 mt-4 flex-1 overflow-hidden flex flex-col">
              <h3 className="font-semibold mb-3 flex-shrink-0">
                Pending Invitations ({pendingInvitations.length})
              </h3>
              <div className="space-y-2 overflow-auto flex-1">
                {pendingInvitations.map((invitation) => (
                  <div
                    key={invitation.id}
                    className="flex items-center justify-between gap-3 p-3 rounded-lg border bg-muted/50"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <Mail className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                        <span className="font-medium text-sm truncate">
                          {invitation.email}
                        </span>
                        <Badge variant="outline" className="text-xs flex-shrink-0">
                          {getRoleDisplayName(invitation.role)}
                        </Badge>
                      </div>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <p className="text-xs text-muted-foreground cursor-default">
                              Sent {formatRelativeTime(new Date(invitation.created_at))}
                            </p>
                          </TooltipTrigger>
                          <TooltipContent>
                            Expires:{' '}
                            {new Date(invitation.expires_at).toLocaleDateString('en-US', {
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                            })}
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    {onRevokeInvitation && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRevokeInvitation(invitation.id)}
                        aria-label={`Revoke invitation for ${invitation.email}`}
                      >
                        <X className="h-4 w-4 text-muted-foreground" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          <DialogFooter className="mt-4 flex-shrink-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!isValid || loading}>
              {loading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Sending...
                </>
              ) : (
                <>
                  <Send className="h-4 w-4 mr-2" />
                  Send {parsedEmails.length > 1 ? `${parsedEmails.length} Invitations` : 'Invitation'}
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
