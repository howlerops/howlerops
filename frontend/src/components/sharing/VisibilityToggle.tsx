/**
 * VisibilityToggle Component
 *
 * Dropdown component for toggling resource visibility between personal and shared (organization).
 * Supports permission-aware UI with disabled states and tooltips.
 *
 * @module components/sharing/VisibilityToggle
 */

import { Globe, Loader2,Lock } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { usePermissions } from '@/hooks/usePermissions'
import { useOrganizationStore } from '@/store/organization-store'

interface VisibilityToggleProps {
  /** Resource ID */
  resourceId: string

  /** Resource type (connection or query) */
  resourceType: 'connection' | 'query'

  /** Current visibility state */
  currentVisibility: 'personal' | 'shared'

  /** Current organization ID (if shared) */
  currentOrgId?: string | null

  /** Resource owner user ID */
  ownerId?: string

  /** Callback after successful toggle */
  onUpdate: () => void

  /** Callback for share action */
  onShare: (id: string, orgId: string) => Promise<void>

  /** Callback for unshare action */
  onUnshare: (id: string) => Promise<void>

  /** Compact mode (icon only) */
  compact?: boolean

  /** Disabled state */
  disabled?: boolean
}

/**
 * VisibilityToggle Component
 *
 * Usage:
 * ```tsx
 * <VisibilityToggle
 *   resourceId={connection.id}
 *   resourceType="connection"
 *   currentVisibility={connection.visibility}
 *   currentOrgId={connection.organization_id}
 *   onShare={shareConnection}
 *   onUnshare={unshareConnection}
 *   onUpdate={() => refetch()}
 * />
 * ```
 */
export function VisibilityToggle({
  resourceId,
  resourceType,
  currentVisibility,
  currentOrgId,
  ownerId,
  onUpdate,
  onShare,
  onUnshare,
  compact = false,
  disabled = false,
}: VisibilityToggleProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { organizations, currentOrgId: selectedOrgId } = useOrganizationStore()
  const { canUpdateResource } = usePermissions(currentOrgId || selectedOrgId)

  // Check if user can update this resource
  const canUpdate = currentOrgId
    ? canUpdateResource(ownerId || '', '')
    : true

  const isDisabled = disabled || isLoading || !canUpdate

  // Get current organization
  const currentOrg = organizations.find(
    (o) => o.id === (currentOrgId || selectedOrgId)
  )

  const handleToggle = async (value: string) => {
    if (isLoading) return

    setIsLoading(true)

    try {
      if (value === 'shared' && selectedOrgId) {
        await onShare(resourceId, selectedOrgId)
        toast.success(
          `${resourceType === 'connection' ? 'Connection' : 'Query'} shared with ${currentOrg?.name || 'organization'}`
        )
      } else if (value === 'personal') {
        await onUnshare(resourceId)
        toast.success(
          `${resourceType === 'connection' ? 'Connection' : 'Query'} is now personal`
        )
      }

      onUpdate()
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : `Failed to ${value === 'shared' ? 'share' : 'unshare'} ${resourceType}`

      toast.error(message)
    } finally {
      setIsLoading(false)
    }
  }

  // Compact mode - badge only
  if (compact) {
    return (
      <Badge variant={currentVisibility === 'shared' ? 'default' : 'outline'}>
        {isLoading ? (
          <Loader2 className="h-3 w-3 animate-spin" />
        ) : currentVisibility === 'shared' ? (
          <>
            <Globe className="h-3 w-3 mr-1" />
            Shared
          </>
        ) : (
          <>
            <Lock className="h-3 w-3 mr-1" />
            Personal
          </>
        )}
      </Badge>
    )
  }

  // Full mode - dropdown select
  return (
    <div className="flex items-center gap-2">
      <label className="text-sm font-medium text-muted-foreground">
        Visibility:
      </label>

      <Select
        value={currentVisibility}
        onValueChange={handleToggle}
        disabled={isDisabled}
      >
        <SelectTrigger className="w-[200px]">
          <SelectValue>
            {isLoading ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Updating...
              </span>
            ) : currentVisibility === 'shared' ? (
              <span className="flex items-center gap-2">
                <Globe className="h-4 w-4" />
                Shared with {currentOrg?.name || 'Organization'}
              </span>
            ) : (
              <span className="flex items-center gap-2">
                <Lock className="h-4 w-4" />
                Personal
              </span>
            )}
          </SelectValue>
        </SelectTrigger>

        <SelectContent>
          <SelectItem value="personal">
            <div className="flex items-center gap-2">
              <Lock className="h-4 w-4" />
              <div>
                <div className="font-medium">Personal</div>
                <div className="text-xs text-muted-foreground">
                  Only you can access
                </div>
              </div>
            </div>
          </SelectItem>

          {selectedOrgId && currentOrg && (
            <SelectItem value="shared">
              <div className="flex items-center gap-2">
                <Globe className="h-4 w-4" />
                <div>
                  <div className="font-medium">
                    Shared with {currentOrg.name}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {currentOrg.member_count || 0} members can access
                  </div>
                </div>
              </div>
            </SelectItem>
          )}

          {!selectedOrgId && (
            <SelectItem value="shared" disabled>
              <div className="flex items-center gap-2">
                <Globe className="h-4 w-4 opacity-50" />
                <div>
                  <div className="font-medium text-muted-foreground">
                    No organization selected
                  </div>
                  <div className="text-xs text-muted-foreground">
                    Select an organization first
                  </div>
                </div>
              </div>
            </SelectItem>
          )}
        </SelectContent>
      </Select>

      {!canUpdate && (
        <span className="text-xs text-muted-foreground">
          (You cannot change visibility)
        </span>
      )}
    </div>
  )
}
