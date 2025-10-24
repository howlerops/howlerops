/**
 * Organization Create Modal Component
 *
 * Modal for creating new organizations with form validation.
 * Includes name (required, 3-50 chars) and description (optional).
 */

import * as React from 'react'
import { Building2, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import type { CreateOrganizationInput } from '@/types/organization'

interface OrganizationCreateModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (data: CreateOrganizationInput) => Promise<void>
  loading?: boolean
  error?: string | null
}

export function OrganizationCreateModal({
  open,
  onOpenChange,
  onCreate,
  loading = false,
  error = null,
}: OrganizationCreateModalProps) {
  const [name, setName] = React.useState('')
  const [description, setDescription] = React.useState('')
  const [validationErrors, setValidationErrors] = React.useState<{
    name?: string
    description?: string
  }>({})

  // Reset form when modal opens/closes
  React.useEffect(() => {
    if (!open) {
      setName('')
      setDescription('')
      setValidationErrors({})
    }
  }, [open])

  const validateForm = (): boolean => {
    const errors: typeof validationErrors = {}

    // Name validation
    if (!name.trim()) {
      errors.name = 'Organization name is required'
    } else if (name.trim().length < 3) {
      errors.name = 'Organization name must be at least 3 characters'
    } else if (name.trim().length > 50) {
      errors.name = 'Organization name must be at most 50 characters'
    }

    // Description validation
    if (description.length > 500) {
      errors.description = 'Description must be at most 500 characters'
    }

    setValidationErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    const data: CreateOrganizationInput = {
      name: name.trim(),
      description: description.trim() || undefined,
    }

    try {
      await onCreate(data)
      // Modal will be closed by parent component on success
    } catch (err) {
      // Error will be displayed via error prop
      console.error('Failed to create organization:', err)
    }
  }

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value)
    if (validationErrors.name) {
      setValidationErrors({ ...validationErrors, name: undefined })
    }
  }

  const handleDescriptionChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setDescription(e.target.value)
    if (validationErrors.description) {
      setValidationErrors({ ...validationErrors, description: undefined })
    }
  }

  const isValid = name.trim().length >= 3 && name.trim().length <= 50 && description.length <= 500

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <div className="flex items-center gap-3 mb-2">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
              <Building2 className="h-5 w-5 text-white" />
            </div>
            <div>
              <DialogTitle>Create Organization</DialogTitle>
            </div>
          </div>
          <DialogDescription>
            Create a new organization to collaborate with your team. You'll be the owner.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <Label htmlFor="name">
                Organization Name <span className="text-destructive">*</span>
              </Label>
              <Input
                id="name"
                placeholder="Acme Inc."
                value={name}
                onChange={handleNameChange}
                disabled={loading}
                aria-invalid={!!validationErrors.name}
                aria-describedby={validationErrors.name ? 'name-error' : undefined}
                maxLength={50}
              />
              <div className="flex items-center justify-between">
                {validationErrors.name && (
                  <p id="name-error" className="text-sm text-destructive">
                    {validationErrors.name}
                  </p>
                )}
                <p className="text-xs text-muted-foreground ml-auto">
                  {name.length}/50
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description (optional)</Label>
              <textarea
                id="description"
                placeholder="What does your organization do?"
                value={description}
                onChange={handleDescriptionChange}
                disabled={loading}
                aria-invalid={!!validationErrors.description}
                aria-describedby={validationErrors.description ? 'description-error' : undefined}
                maxLength={500}
                rows={3}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-none"
              />
              <div className="flex items-center justify-between">
                {validationErrors.description && (
                  <p id="description-error" className="text-sm text-destructive">
                    {validationErrors.description}
                  </p>
                )}
                <p className="text-xs text-muted-foreground ml-auto">
                  {description.length}/500
                </p>
              </div>
            </div>
          </div>

          <DialogFooter>
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
                  Creating...
                </>
              ) : (
                'Create Organization'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
