/**
 * Organization List Component
 *
 * Displays a grid/list view of user's organizations with ability to create new ones.
 * Shows organization details including name, description, member count, and user's role.
 */

import * as React from 'react'
import { Users, Building2, Plus, ChevronRight, Loader2 } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import type { OrganizationWithMembership } from '@/types/organization'
import { getRoleDisplayName, getRoleBadgeVariant } from '@/types/organization'
import { cn } from '@/lib/utils'

interface OrganizationListProps {
  organizations: OrganizationWithMembership[]
  loading?: boolean
  error?: string | null
  onCreateClick: () => void
  onOrganizationClick: (org: OrganizationWithMembership) => void
  className?: string
}

export function OrganizationList({
  organizations,
  loading = false,
  error = null,
  onCreateClick,
  onOrganizationClick,
  className,
}: OrganizationListProps) {
  if (loading) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Loading organizations...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={className}>
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </div>
    )
  }

  if (organizations.length === 0) {
    return (
      <div className={cn('flex flex-col items-center justify-center py-12 px-4', className)}>
        <div className="max-w-md text-center space-y-4">
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center">
              <Building2 className="h-8 w-8 text-muted-foreground" />
            </div>
          </div>
          <div className="space-y-2">
            <h3 className="text-lg font-semibold">No organizations yet</h3>
            <p className="text-sm text-muted-foreground">
              Create your first organization to collaborate with your team
            </p>
          </div>
          <Button onClick={onCreateClick} className="mt-4">
            <Plus className="h-4 w-4 mr-2" />
            Create Organization
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-semibold">Organizations</h2>
          <p className="text-sm text-muted-foreground mt-1">
            Manage your team workspaces and collaborations
          </p>
        </div>
        <Button onClick={onCreateClick}>
          <Plus className="h-4 w-4 mr-2" />
          Create Organization
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {organizations.map((org) => (
          <OrganizationCard
            key={org.id}
            organization={org}
            onClick={() => onOrganizationClick(org)}
          />
        ))}
      </div>
    </div>
  )
}

interface OrganizationCardProps {
  organization: OrganizationWithMembership
  onClick: () => void
}

function OrganizationCard({ organization, onClick }: OrganizationCardProps) {
  const memberCountText = organization.member_count === 1
    ? '1 member'
    : `${organization.member_count || 0} members`

  return (
    <Card
      className="cursor-pointer transition-all hover:shadow-md hover:border-primary/50 group"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
      aria-label={`Open ${organization.name} organization`}
    >
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-3 min-w-0 flex-1">
            <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
              <Building2 className="h-5 w-5 text-white" />
            </div>
            <div className="min-w-0 flex-1">
              <CardTitle className="text-lg truncate">{organization.name}</CardTitle>
              <Badge
                variant={getRoleBadgeVariant(organization.current_user_role)}
                className="mt-1 text-xs"
              >
                {getRoleDisplayName(organization.current_user_role)}
              </Badge>
            </div>
          </div>
          <ChevronRight className="h-5 w-5 text-muted-foreground shrink-0 transition-transform group-hover:translate-x-1" />
        </div>
      </CardHeader>
      <CardContent>
        {organization.description && (
          <CardDescription className="mb-3 line-clamp-2">
            {organization.description}
          </CardDescription>
        )}
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Users className="h-4 w-4" />
          <span>{memberCountText}</span>
        </div>
      </CardContent>
    </Card>
  )
}

// Mobile-friendly list view variant
export function OrganizationListMobile({
  organizations,
  loading = false,
  error = null,
  onCreateClick,
  onOrganizationClick,
  className,
}: OrganizationListProps) {
  if (loading) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return (
      <div className={className}>
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </div>
    )
  }

  if (organizations.length === 0) {
    return (
      <div className={cn('text-center py-8 px-4', className)}>
        <Building2 className="h-12 w-12 mx-auto text-muted-foreground mb-3" />
        <h3 className="font-semibold mb-2">No organizations</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Create your first organization
        </p>
        <Button onClick={onCreateClick} size="sm">
          <Plus className="h-4 w-4 mr-2" />
          Create Organization
        </Button>
      </div>
    )
  }

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold">Organizations</h2>
        <Button onClick={onCreateClick} size="sm">
          <Plus className="h-4 w-4 mr-2" />
          Create
        </Button>
      </div>

      <div className="space-y-2">
        {organizations.map((org) => (
          <button
            key={org.id}
            onClick={() => onOrganizationClick(org)}
            className="w-full text-left p-3 rounded-lg border bg-card hover:bg-accent transition-colors"
          >
            <div className="flex items-center gap-3">
              <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
                <Building2 className="h-5 w-5 text-white" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="font-semibold text-sm truncate">{org.name}</h3>
                  <Badge
                    variant={getRoleBadgeVariant(org.current_user_role)}
                    className="text-xs flex-shrink-0"
                  >
                    {getRoleDisplayName(org.current_user_role)}
                  </Badge>
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Users className="h-3 w-3" />
                  <span>{org.member_count || 0} members</span>
                </div>
              </div>
              <ChevronRight className="h-5 w-5 text-muted-foreground flex-shrink-0" />
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}
