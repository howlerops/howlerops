/**
 * QueryCard Component
 *
 * Displays an individual saved query in a list with metadata and actions
 *
 * Features:
 * - Query metadata display (title, description, folder, tags, dates)
 * - Visual indicators (favorite star, sync status)
 * - Action dropdown menu (Load, Edit, Duplicate, Toggle Favorite, Delete)
 * - Click card to load query
 * - Hover states and accessibility support
 *
 * @module components/saved-queries/QueryCard
 */

import { useState, useEffect } from 'react'
import { formatDistanceToNow } from 'date-fns'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  Star,
  MoreVertical,
  Play,
  Edit,
  Copy,
  Trash2,
  Folder,
  Cloud,
  CloudOff,
} from 'lucide-react'
import type { SavedQueryRecord } from '@/types/storage'
import { cn } from '@/lib/utils'

/**
 * Props for QueryCard component
 */
export interface QueryCardProps {
  /** The saved query record to display */
  query: SavedQueryRecord

  /** Callback when user wants to load the query */
  onLoad: (query: SavedQueryRecord) => void

  /** Callback when user wants to edit the query */
  onEdit: (query: SavedQueryRecord) => void

  /** Callback when user wants to delete the query */
  onDelete: (id: string) => void

  /** Callback when user wants to duplicate the query */
  onDuplicate: (id: string) => void

  /** Callback when user toggles favorite status */
  onToggleFavorite: (id: string) => void

  /** Whether to show sync status indicator (for Individual tier) */
  showSyncStatus?: boolean

  /** Internal testing overrides */
  testOverrides?: {
    defaultMenuOpen?: boolean
    forceDeleteDialogOpen?: boolean
    onDeleteRequest?: () => void
  }
}

/**
 * QueryCard component for displaying saved queries in a list
 *
 * Usage:
 * ```tsx
 * <QueryCard
 *   query={savedQuery}
 *   onLoad={(q) => loadQueryInEditor(q)}
 *   onEdit={(q) => openEditDialog(q)}
 *   onDelete={(id) => deleteQuery(id)}
 *   onDuplicate={(id) => duplicateQuery(id)}
 *   onToggleFavorite={(id) => toggleFavorite(id)}
 *   showSyncStatus={userTier === 'individual'}
 * />
 * ```
 */
export function QueryCard({
  query,
  onLoad,
  onEdit,
  onDelete,
  onDuplicate,
  onToggleFavorite,
  showSyncStatus = false,
  testOverrides,
}: QueryCardProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(
    testOverrides?.forceDeleteDialogOpen ?? false
  )

  useEffect(() => {
    if (typeof testOverrides?.forceDeleteDialogOpen === 'boolean') {
      setShowDeleteDialog(testOverrides.forceDeleteDialogOpen)
    }
  }, [testOverrides?.forceDeleteDialogOpen])

  // Format the relative time since last update
  const lastUpdated = formatDistanceToNow(new Date(query.updated_at), {
    addSuffix: true,
  })

  // Truncate description to 2 lines (approx 120 chars)
  const truncatedDescription = query.description
    ? query.description.length > 120
      ? query.description.slice(0, 120) + '...'
      : query.description
    : null

  // Handle card click to load query (except when clicking actions)
  const handleCardClick = (e: React.MouseEvent<HTMLDivElement>) => {
    // Don't trigger if clicking on dropdown or favorite star
    if ((e.target as HTMLElement).closest('[data-no-propagate]')) {
      return
    }
    onLoad(query)
  }

  // Handle favorite toggle
  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    onToggleFavorite(query.id)
  }

  // Handle delete confirmation
  const handleDeleteRequest = () => {
    setShowDeleteDialog(true)
    testOverrides?.onDeleteRequest?.()
  }

  const handleDeleteConfirm = () => {
    onDelete(query.id)
    setShowDeleteDialog(false)
  }

  return (
    <>
      <Card
        className={cn(
          'cursor-pointer transition-colors hover:bg-accent/50',
          'focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2'
        )}
        onClick={handleCardClick}
        role="button"
        tabIndex={0}
        aria-label={`Load query: ${query.title}`}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onLoad(query)
          }
        }}
      >
        <CardHeader className="pb-3">
          <div className="flex items-start justify-between gap-2">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <CardTitle className="text-base truncate">
                  {query.title}
                </CardTitle>
                {/* Favorite star indicator */}
                <button
                  onClick={handleFavoriteClick}
                  className="shrink-0 p-1 rounded hover:bg-accent transition-colors"
                  aria-label={query.is_favorite ? 'Remove from favorites' : 'Add to favorites'}
                  data-no-propagate
                >
                  <Star
                    className={cn(
                      'h-4 w-4 transition-colors',
                      query.is_favorite
                        ? 'fill-yellow-400 text-yellow-400'
                        : 'text-muted-foreground hover:text-foreground'
                    )}
                  />
                </button>
              </div>

              {truncatedDescription && (
                <CardDescription className="line-clamp-2 text-sm">
                  {truncatedDescription}
                </CardDescription>
              )}
            </div>

            {/* Actions dropdown */}
            <DropdownMenu defaultOpen={testOverrides?.defaultMenuOpen}>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 w-8 p-0 shrink-0"
                  aria-label="Query actions"
                  data-no-propagate
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onSelect={() => onLoad(query)}>
                  <Play className="h-4 w-4 mr-2" />
                  Load Query
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => onEdit(query)}>
                  <Edit className="h-4 w-4 mr-2" />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => onDuplicate(query.id)}>
                  <Copy className="h-4 w-4 mr-2" />
                  Duplicate
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onSelect={() => onToggleFavorite(query.id)}>
                  <Star
                    className={cn(
                      'h-4 w-4 mr-2',
                      query.is_favorite && 'fill-yellow-400 text-yellow-400'
                    )}
                  />
                  {query.is_favorite ? 'Remove from Favorites' : 'Add to Favorites'}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onSelect={handleDeleteRequest}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardHeader>

        <CardContent className="pt-0">
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            {/* Folder badge */}
            {query.folder && (
              <Badge variant="outline" className="gap-1">
                <Folder className="h-3 w-3" />
                {query.folder}
              </Badge>
            )}

            {/* Tags */}
            {query.tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}

            {/* Sync status (Individual tier only) */}
            {showSyncStatus && (
              <Badge
                variant="outline"
                className={cn(
                  'gap-1',
                  query.synced
                    ? 'text-green-700 border-green-300 dark:text-green-300 dark:border-green-700'
                    : 'text-orange-700 border-orange-300 dark:text-orange-300 dark:border-orange-700'
                )}
                aria-label={query.synced ? 'Synced to cloud' : 'Not synced'}
              >
                {query.synced ? (
                  <>
                    <Cloud className="h-3 w-3" />
                    Synced
                  </>
                ) : (
                  <>
                    <CloudOff className="h-3 w-3" />
                    Local
                  </>
                )}
              </Badge>
            )}

            {/* Last updated timestamp */}
            <span className="ml-auto" aria-label={`Updated ${lastUpdated}`}>
              {lastUpdated}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Delete confirmation dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Saved Query?</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{query.title}"? This action cannot be
              undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteConfirm}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
