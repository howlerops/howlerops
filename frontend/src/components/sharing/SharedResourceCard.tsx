/**
 * SharedResourceCard Component
 *
 * Card component for displaying shared connections or queries in an organization.
 * Shows metadata, ownership, timestamps, and provides action menu with permissions.
 *
 * @module components/sharing/SharedResourceCard
 */

import { formatDistanceToNow } from 'date-fns'
import {
  Code2,
  Database,
  Edit,
  ExternalLink,
  Eye,
  Lock,
  MoreVertical,
  Trash2,
  Users,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { usePermissions } from '@/hooks/usePermissions'
import type { Connection } from '@/lib/api/connections'
import type { SavedQuery } from '@/lib/api/queries'

type Resource = Connection | SavedQuery

interface SharedResourceCardProps {
  /** The resource (connection or query) */
  resource: Resource

  /** Resource type */
  type: 'connection' | 'query'

  /** Callback for viewing resource details */
  onView?: (resource: Resource) => void

  /** Callback for editing resource */
  onEdit?: (resource: Resource) => void

  /** Callback for making resource private */
  onUnshare?: (resource: Resource) => void

  /** Callback for deleting resource */
  onDelete?: (resource: Resource) => void

  /** Callback for using/executing resource */
  onUse?: (resource: Resource) => void
}

/**
 * Check if resource is a SavedQuery
 */
function isQuery(resource: Resource): resource is SavedQuery {
  return 'sql_content' in resource
}

/**
 * Check if resource is a Connection
 */
function isConnection(resource: Resource): resource is Connection {
  return 'database_type' in resource && !('sql_content' in resource)
}

/**
 * SharedResourceCard Component
 *
 * Usage:
 * ```tsx
 * <SharedResourceCard
 *   resource={connection}
 *   type="connection"
 *   onView={handleView}
 *   onEdit={handleEdit}
 *   onUnshare={handleUnshare}
 *   onDelete={handleDelete}
 * />
 * ```
 */
export function SharedResourceCard({
  resource,
  type,
  onView,
  onEdit,
  onUnshare,
  onDelete,
  onUse,
}: SharedResourceCardProps) {
  const { canUpdateResource, canDeleteResource } = usePermissions(
    resource.organization_id
  )

  // Permission checks
  const canUpdate = canUpdateResource(resource.user_id, resource.user_id)
  const canRemove = canDeleteResource(resource.user_id, resource.user_id)

  // Get resource-specific details
  const title = isQuery(resource) ? resource.title : resource.name
  const description = resource.description || 'No description provided'
  const icon = type === 'connection' ? Database : Code2

  const Icon = icon

  return (
    <Card className="hover:shadow-lg transition-shadow duration-200">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-3 flex-1">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Icon className="h-5 w-5 text-primary" />
            </div>

            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <CardTitle className="text-lg truncate">{title}</CardTitle>

                <Badge variant="secondary" className="shrink-0">
                  <Users className="h-3 w-3 mr-1" />
                  Shared
                </Badge>
              </div>

              <CardDescription className="line-clamp-2">
                {description}
              </CardDescription>
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                <MoreVertical className="h-4 w-4" />
                <span className="sr-only">Open menu</span>
              </Button>
            </DropdownMenuTrigger>

            <DropdownMenuContent align="end">
              {onView && (
                <DropdownMenuItem onClick={() => onView(resource)}>
                  <Eye className="h-4 w-4 mr-2" />
                  View Details
                </DropdownMenuItem>
              )}

              {onUse && (
                <DropdownMenuItem onClick={() => onUse(resource)}>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  {type === 'connection' ? 'Connect' : 'Run Query'}
                </DropdownMenuItem>
              )}

              {canUpdate && (
                <>
                  <DropdownMenuSeparator />

                  {onEdit && (
                    <DropdownMenuItem onClick={() => onEdit(resource)}>
                      <Edit className="h-4 w-4 mr-2" />
                      Edit
                    </DropdownMenuItem>
                  )}

                  {onUnshare && (
                    <DropdownMenuItem onClick={() => onUnshare(resource)}>
                      <Lock className="h-4 w-4 mr-2" />
                      Make Private
                    </DropdownMenuItem>
                  )}
                </>
              )}

              {canRemove && onDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive"
                    onClick={() => onDelete(resource)}
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
          {/* Database type / Query language */}
          {isConnection(resource) && (
            <div className="flex items-center gap-1">
              <Database className="h-3.5 w-3.5" />
              <span>{resource.database_type.toUpperCase()}</span>
            </div>
          )}

          {isQuery(resource) && (
            <>
              <div className="flex items-center gap-1">
                <Code2 className="h-3.5 w-3.5" />
                <span>{resource.database_type.toUpperCase()}</span>
              </div>

              {resource.tags && resource.tags.length > 0 && (
                <div className="flex items-center gap-1">
                  {resource.tags.slice(0, 2).map((tag) => (
                    <Badge key={tag} variant="outline" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                  {resource.tags.length > 2 && (
                    <Badge variant="outline" className="text-xs">
                      +{resource.tags.length - 2}
                    </Badge>
                  )}
                </div>
              )}
            </>
          )}

          {/* Owner */}
          {resource.created_by_email && (
            <span className="text-xs">
              Created by{' '}
              <span className="font-medium">{resource.created_by_email}</span>
            </span>
          )}

          {/* Last modified */}
          <span className="text-xs">
            Modified{' '}
            {formatDistanceToNow(new Date(resource.updated_at), {
              addSuffix: true,
            })}
          </span>

          {/* Last used (for connections) */}
          {isConnection(resource) && resource.last_used && (
            <span className="text-xs">
              Last used{' '}
              {formatDistanceToNow(new Date(resource.last_used), {
                addSuffix: true,
              })}
            </span>
          )}

          {/* Last executed (for queries) */}
          {isQuery(resource) && resource.last_executed && (
            <span className="text-xs">
              Last run{' '}
              {formatDistanceToNow(new Date(resource.last_executed), {
                addSuffix: true,
              })}
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
