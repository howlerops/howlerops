/**
 * Organization Settings Component
 *
 * Settings page for managing organization details.
 * Includes edit name/description, member count, creation date, and delete organization (owner only).
 */

import * as React from 'react'
import { Building2, Calendar, Users, Loader2, Trash2, Save, ArrowRight } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { Organization, UpdateOrganizationInput, OrganizationRole } from '@/types/organization'
import { canUpdateSettings, canDeleteOrganization } from '@/types/organization'
import { cn } from '@/lib/utils'

interface OrganizationSettingsProps {
  organization: Organization
  currentUserRole: OrganizationRole
  onUpdate: (data: UpdateOrganizationInput) => Promise<void>
  onDelete: () => Promise<void>
  onNavigateToMembers: () => void
  loading?: boolean
  error?: string | null
  className?: string
}

export function OrganizationSettings({
  organization,
  currentUserRole,
  onUpdate,
  onDelete,
  onNavigateToMembers,
  loading = false,
  error = null,
  className,
}: OrganizationSettingsProps) {
  const [name, setName] = React.useState(organization.name)
  const [description, setDescription] = React.useState(organization.description || '')
  const [showDeleteDialog, setShowDeleteDialog] = React.useState(false)
  const [validationErrors, setValidationErrors] = React.useState<{
    name?: string
    description?: string
  }>({})
  const [hasChanges, setHasChanges] = React.useState(false)

  const canEdit = canUpdateSettings(currentUserRole)
  const canDelete = canDeleteOrganization(currentUserRole)

  // Track changes
  React.useEffect(() => {
    const nameChanged = name.trim() !== organization.name
    const descriptionChanged = description.trim() !== (organization.description || '')
    setHasChanges(nameChanged || descriptionChanged)
  }, [name, description, organization.name, organization.description])

  const validateForm = (): boolean => {
    const errors: typeof validationErrors = {}

    if (!name.trim()) {
      errors.name = 'Organization name is required'
    } else if (name.trim().length < 3) {
      errors.name = 'Organization name must be at least 3 characters'
    } else if (name.trim().length > 50) {
      errors.name = 'Organization name must be at most 50 characters'
    }

    if (description.length > 500) {
      errors.description = 'Description must be at most 500 characters'
    }

    setValidationErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleSave = async () => {
    if (!validateForm() || !canEdit) {
      return
    }

    const data: UpdateOrganizationInput = {}

    if (name.trim() !== organization.name) {
      data.name = name.trim()
    }

    if (description.trim() !== (organization.description || '')) {
      data.description = description.trim()
    }

    try {
      await onUpdate(data)
      setHasChanges(false)
    } catch (err) {
      console.error('Failed to update organization:', err)
    }
  }

  const handleDelete = async () => {
    try {
      await onDelete()
      setShowDeleteDialog(false)
    } catch (err) {
      console.error('Failed to delete organization:', err)
    }
  }

  const handleReset = () => {
    setName(organization.name)
    setDescription(organization.description || '')
    setValidationErrors({})
    setHasChanges(false)
  }

  const createdDate = new Date(organization.created_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })

  return (
    <div className={cn('space-y-6', className)}>
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* General Settings Card */}
      <Card>
        <CardHeader>
          <CardTitle>General Settings</CardTitle>
          <CardDescription>
            {canEdit
              ? 'Update your organization details'
              : 'View organization details (read-only)'}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="org-name">
              Organization Name {canEdit && <span className="text-destructive">*</span>}
            </Label>
            <Input
              id="org-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={!canEdit || loading}
              aria-invalid={!!validationErrors.name}
              aria-describedby={validationErrors.name ? 'org-name-error' : undefined}
              maxLength={50}
            />
            <div className="flex items-center justify-between">
              {validationErrors.name && (
                <p id="org-name-error" className="text-sm text-destructive">
                  {validationErrors.name}
                </p>
              )}
              <p className="text-xs text-muted-foreground ml-auto">{name.length}/50</p>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="org-description">Description (optional)</Label>
            <textarea
              id="org-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={!canEdit || loading}
              aria-invalid={!!validationErrors.description}
              aria-describedby={validationErrors.description ? 'org-description-error' : undefined}
              maxLength={500}
              rows={3}
              className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-none"
              placeholder="What does your organization do?"
            />
            <div className="flex items-center justify-between">
              {validationErrors.description && (
                <p id="org-description-error" className="text-sm text-destructive">
                  {validationErrors.description}
                </p>
              )}
              <p className="text-xs text-muted-foreground ml-auto">{description.length}/500</p>
            </div>
          </div>

          {canEdit && hasChanges && (
            <div className="flex items-center gap-2 pt-2">
              <Button onClick={handleSave} disabled={loading || !!Object.keys(validationErrors).length}>
                {loading ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="h-4 w-4 mr-2" />
                    Save Changes
                  </>
                )}
              </Button>
              <Button variant="outline" onClick={handleReset} disabled={loading}>
                Cancel
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Organization Info Card */}
      <Card>
        <CardHeader>
          <CardTitle>Organization Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3 text-sm">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="font-medium">Created</p>
              <p className="text-muted-foreground">{createdDate}</p>
            </div>
          </div>

          <div className="flex items-center gap-3 text-sm">
            <Users className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="font-medium">Members</p>
              <p className="text-muted-foreground">
                {organization.member_count || 0} / {organization.max_members} members
              </p>
            </div>
          </div>

          <Button variant="outline" onClick={onNavigateToMembers} className="w-full">
            Manage Members
            <ArrowRight className="h-4 w-4 ml-2" />
          </Button>
        </CardContent>
      </Card>

      {/* Danger Zone Card */}
      {canDelete && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">Danger Zone</CardTitle>
            <CardDescription>
              Irreversible actions that affect your organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-start justify-between gap-4 p-4 rounded-lg border border-destructive/50 bg-destructive/5">
              <div>
                <h4 className="font-semibold mb-1">Delete Organization</h4>
                <p className="text-sm text-muted-foreground">
                  Permanently delete this organization and all associated data. This action cannot be
                  undone.
                </p>
              </div>
              <Button
                variant="destructive"
                onClick={() => setShowDeleteDialog(true)}
                disabled={loading}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Organization</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <strong>{organization.name}</strong>? This action cannot
              be undone and will remove all members and associated data.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)} disabled={loading}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Deleting...
                </>
              ) : (
                <>
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete Organization
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
