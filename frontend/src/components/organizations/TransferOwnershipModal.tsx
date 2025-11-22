/**
 * TransferOwnershipModal Component
 *
 * Modal for transferring organization ownership to another admin.
 * Only visible to current owners. Requires password confirmation.
 * Shows clear warnings about consequences.
 */

import { AlertTriangle, ArrowRight,Crown, Key, Loader2, User } from 'lucide-react'
import * as React from 'react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
import { cn } from '@/lib/utils'
import type { OrganizationMember } from '@/types/organization'
import { OrganizationRole } from '@/types/organization'

interface TransferOwnershipModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  organizationName: string
  members: OrganizationMember[]
  onTransfer: (newOwnerId: string, password: string) => Promise<void>
  onSuccess?: () => void
  loading?: boolean
  error?: string | null
  className?: string
}

/**
 * TransferOwnershipModal
 *
 * Workflow:
 * 1. Select new owner from admin list
 * 2. Read and accept warning about consequences
 * 3. Enter password for confirmation
 * 4. Confirm transfer
 * 5. Redirect to organization list after success
 */
export function TransferOwnershipModal({
  open,
  onOpenChange,
  organizationName,
  members,
  onTransfer,
  onSuccess,
  loading = false,
  error = null,
  className,
}: TransferOwnershipModalProps) {
  const [selectedMemberId, setSelectedMemberId] = React.useState<string>('')
  const [password, setPassword] = React.useState('')
  const [acknowledged, setAcknowledged] = React.useState(false)
  const [validationErrors, setValidationErrors] = React.useState<{
    member?: string
    password?: string
  }>({})

  // Reset form when modal opens/closes
  React.useEffect(() => {
    if (!open) {
      setSelectedMemberId('')
      setPassword('')
      setAcknowledged(false)
      setValidationErrors({})
    }
  }, [open])

  // Filter to only show admins (only admins can become owners)
  const eligibleMembers = React.useMemo(() => {
    return members.filter((m) => m.role === OrganizationRole.Admin)
  }, [members])

  const selectedMember = eligibleMembers.find((m) => m.id === selectedMemberId)

  const validateForm = (): boolean => {
    const errors: typeof validationErrors = {}

    if (!selectedMemberId) {
      errors.member = 'Please select a new owner'
    }

    if (!password.trim()) {
      errors.password = 'Password is required to confirm transfer'
    } else if (password.trim().length < 6) {
      errors.password = 'Password must be at least 6 characters'
    }

    setValidationErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleTransfer = async () => {
    if (!validateForm() || !acknowledged) {
      return
    }

    const member = eligibleMembers.find((m) => m.id === selectedMemberId)
    if (!member) return

    try {
      await onTransfer(member.user_id, password)

      toast.success('Ownership transferred', {
        description: `${member.user?.display_name || member.user?.username} is now the owner`,
      })

      onSuccess?.()
      onOpenChange(false)
    } catch (err) {
      console.error('Failed to transfer ownership:', err)
      // Error is shown in the modal via error prop
    }
  }

  const isValid = selectedMemberId && password.length >= 6 && acknowledged

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn('sm:max-w-[550px]', className)}>
        <DialogHeader>
          <div className="flex items-center gap-3 mb-2">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-yellow-500 to-orange-500 flex items-center justify-center">
              <Crown className="h-5 w-5 text-white" />
            </div>
            <div>
              <DialogTitle>Transfer Ownership</DialogTitle>
            </div>
          </div>
          <DialogDescription>
            Transfer ownership of <strong>{organizationName}</strong> to another admin
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Critical Warning */}
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              <strong>This action is permanent and cannot be undone.</strong>
              <br />
              You will become an Admin after this transfer.
            </AlertDescription>
          </Alert>

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {/* No Eligible Members Warning */}
          {eligibleMembers.length === 0 && (
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                No eligible members found. You must promote at least one member to
                Admin before you can transfer ownership.
              </AlertDescription>
            </Alert>
          )}

          {eligibleMembers.length > 0 && (
            <>
              {/* Select New Owner */}
              <div className="space-y-2">
                <Label htmlFor="new-owner">
                  New Owner <span className="text-destructive">*</span>
                </Label>
                <Select
                  value={selectedMemberId}
                  onValueChange={setSelectedMemberId}
                  disabled={loading}
                >
                  <SelectTrigger
                    id="new-owner"
                    aria-invalid={!!validationErrors.member}
                  >
                    <SelectValue placeholder="Select an admin" />
                  </SelectTrigger>
                  <SelectContent>
                    {eligibleMembers.map((member) => (
                      <SelectItem key={member.id} value={member.id}>
                        <div className="flex items-center gap-3">
                          <User className="h-4 w-4 text-muted-foreground" />
                          <div className="flex flex-col items-start">
                            <span className="font-medium">
                              {member.user?.display_name || member.user?.username}
                            </span>
                            <span className="text-xs text-muted-foreground">
                              {member.user?.email}
                            </span>
                          </div>
                          <Badge variant="secondary" className="ml-auto">
                            Admin
                          </Badge>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {validationErrors.member && (
                  <p className="text-sm text-destructive">
                    {validationErrors.member}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  Only admins can become owners. Members must be promoted to Admin
                  first.
                </p>
              </div>

              {/* Transfer Preview */}
              {selectedMember && (
                <div className="p-4 rounded-lg border bg-muted/50 space-y-3">
                  <h4 className="text-sm font-semibold">Transfer Summary</h4>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-sm">You</span>
                      <Badge variant="default">Owner</Badge>
                    </div>
                    <ArrowRight className="h-4 w-4 text-muted-foreground" />
                    <Badge variant="secondary">Admin</Badge>
                  </div>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-sm truncate max-w-[200px]">
                        {selectedMember.user?.display_name ||
                          selectedMember.user?.username}
                      </span>
                      <Badge variant="secondary">Admin</Badge>
                    </div>
                    <ArrowRight className="h-4 w-4 text-muted-foreground" />
                    <Badge variant="default">Owner</Badge>
                  </div>
                </div>
              )}

              {/* Password Confirmation */}
              <div className="space-y-2">
                <Label htmlFor="password-confirm">
                  Confirm Your Password <span className="text-destructive">*</span>
                </Label>
                <div className="relative">
                  <Key className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="password-confirm"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    disabled={loading}
                    className="pl-10"
                    placeholder="Enter your password"
                    aria-invalid={!!validationErrors.password}
                    autoComplete="current-password"
                  />
                </div>
                {validationErrors.password && (
                  <p className="text-sm text-destructive">
                    {validationErrors.password}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  Enter your password to confirm this critical action
                </p>
              </div>

              {/* Acknowledgement Checkbox */}
              <div className="flex items-start gap-3 p-3 rounded-lg border bg-destructive/5">
                <Checkbox
                  id="acknowledge"
                  checked={acknowledged}
                  onCheckedChange={(checked) =>
                    setAcknowledged(checked === true)
                  }
                  disabled={loading}
                  className="mt-1"
                />
                <div className="flex-1">
                  <Label
                    htmlFor="acknowledge"
                    className="text-sm font-medium cursor-pointer"
                  >
                    I understand the consequences
                  </Label>
                  <p className="text-xs text-muted-foreground mt-1">
                    I understand that this action is permanent, I will lose owner
                    privileges, and the new owner will have full control of the
                    organization.
                  </p>
                </div>
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          {eligibleMembers.length > 0 && (
            <Button
              variant="destructive"
              onClick={handleTransfer}
              disabled={!isValid || loading}
            >
              {loading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Transferring...
                </>
              ) : (
                <>
                  <Crown className="h-4 w-4 mr-2" />
                  Transfer Ownership
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
