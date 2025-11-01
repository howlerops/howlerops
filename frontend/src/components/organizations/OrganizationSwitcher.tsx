/**
 * Organization Switcher Component
 *
 * Dropdown in top navigation for switching between organizations.
 * Shows current organization and allows quick switching to others or personal workspace.
 */

import * as React from 'react'
import { Check, ChevronsUpDown, Building2, User, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectSeparator,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import type { OrganizationWithMembership } from '@/types/organization'

interface OrganizationSwitcherProps {
  organizations: OrganizationWithMembership[]
  currentOrganizationId: string | null
  onOrganizationChange: (organizationId: string | null) => void
  onCreateClick?: () => void
  loading?: boolean
  className?: string
}

export function OrganizationSwitcher({
  organizations,
  currentOrganizationId,
  onOrganizationChange,
  onCreateClick,
  loading = false,
  className,
}: OrganizationSwitcherProps) {
  const currentOrg = organizations.find((org) => org.id === currentOrganizationId)

  const displayName = currentOrg?.name || 'Personal Workspace'
  const displayIcon = currentOrg ? (
    <Building2 className="h-4 w-4" />
  ) : (
    <User className="h-4 w-4" />
  )

  return (
    <Select
      value={currentOrganizationId || 'personal'}
      onValueChange={(value) => {
        if (value === 'personal') {
          onOrganizationChange(null)
        } else if (value === 'create') {
          onCreateClick?.()
        } else {
          onOrganizationChange(value)
        }
      }}
      disabled={loading}
    >
      <SelectTrigger
        className={cn('w-[200px]', className)}
        aria-label="Select organization"
      >
        <div className="flex items-center gap-2 min-w-0">
          {displayIcon}
          <span className="truncate">{displayName}</span>
        </div>
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="personal">
          <div className="flex items-center gap-2">
            <User className="h-4 w-4" />
            <span>Personal Workspace</span>
          </div>
        </SelectItem>

        {organizations.length > 0 && <SelectSeparator />}

        {organizations.map((org) => (
          <SelectItem key={org.id} value={org.id}>
            <div className="flex items-center gap-2">
              <Building2 className="h-4 w-4" />
              <span className="truncate">{org.name}</span>
            </div>
          </SelectItem>
        ))}

        {onCreateClick && (
          <>
            <SelectSeparator />
            <SelectItem value="create">
              <div className="flex items-center gap-2 text-primary">
                <Plus className="h-4 w-4" />
                <span>Create Organization</span>
              </div>
            </SelectItem>
          </>
        )}
      </SelectContent>
    </Select>
  )
}

// Compact version for mobile
export function OrganizationSwitcherCompact({
  organizations,
  currentOrganizationId,
  onOrganizationChange,
  onCreateClick,
  loading = false,
  className,
}: OrganizationSwitcherProps) {
  const [open, setOpen] = React.useState(false)
  const currentOrg = organizations.find((org) => org.id === currentOrganizationId)

  const displayIcon = currentOrg ? (
    <Building2 className="h-5 w-5" />
  ) : (
    <User className="h-5 w-5" />
  )

  const handleSelect = (organizationId: string | null) => {
    onOrganizationChange(organizationId)
    setOpen(false)
  }

  return (
    <div className={cn('relative', className)}>
      <Button
        variant="outline"
        role="combobox"
        aria-expanded={open}
        aria-label="Select organization"
        className="w-full justify-between"
        onClick={() => setOpen(!open)}
        disabled={loading}
      >
        <div className="flex items-center gap-2 min-w-0">
          {displayIcon}
          <span className="truncate">{currentOrg?.name || 'Personal'}</span>
        </div>
        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </Button>

      {open && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setOpen(false)}
            aria-hidden="true"
          />

          {/* Dropdown */}
          <div className="absolute top-full left-0 right-0 mt-2 z-50 rounded-md border bg-popover shadow-md">
            <div className="p-1">
              <button
                onClick={() => handleSelect(null)}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-2 text-sm rounded-sm hover:bg-accent transition-colors',
                  !currentOrganizationId && 'bg-accent'
                )}
              >
                <User className="h-4 w-4" />
                <span className="flex-1 text-left">Personal Workspace</span>
                {!currentOrganizationId && <Check className="h-4 w-4" />}
              </button>

              {organizations.length > 0 && (
                <div className="h-px bg-border my-1" />
              )}

              {organizations.map((org) => (
                <button
                  key={org.id}
                  onClick={() => handleSelect(org.id)}
                  className={cn(
                    'w-full flex items-center gap-3 px-3 py-2 text-sm rounded-sm hover:bg-accent transition-colors',
                    currentOrganizationId === org.id && 'bg-accent'
                  )}
                >
                  <Building2 className="h-4 w-4" />
                  <span className="flex-1 text-left truncate">{org.name}</span>
                  {currentOrganizationId === org.id && <Check className="h-4 w-4" />}
                </button>
              ))}

              {onCreateClick && (
                <>
                  <div className="h-px bg-border my-1" />
                  <button
                    onClick={() => {
                      setOpen(false)
                      onCreateClick()
                    }}
                    className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-sm hover:bg-accent transition-colors text-primary"
                  >
                    <Plus className="h-4 w-4" />
                    <span className="flex-1 text-left">Create Organization</span>
                  </button>
                </>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
