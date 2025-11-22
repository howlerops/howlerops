/**
 * SavedQueriesPanel Component
 *
 * Comprehensive sidebar panel for browsing and managing saved queries
 *
 * Features:
 * - Sheet component (drawer) that slides in from right
 * - Search with debounced input (300ms)
 * - Filter controls (folder dropdown, tags multi-select, favorites toggle)
 * - Sort controls (by title/created/updated with direction toggle)
 * - Stats display with tier limit indicators
 * - Scrollable query list
 * - Empty states for various scenarios
 * - Integration with SaveQueryDialog and useSavedQueriesStore
 *
 * @module components/saved-queries/SavedQueriesPanel
 */

import {
  AlertCircle,
  ArrowDown,
  ArrowUp,
  Filter,
  Folder,
  Inbox,
  Loader2,
  Search,
  Star,
  Tag,
  X,
} from 'lucide-react'
import { useEffect, useMemo,useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import { useSavedQueriesStore } from '@/store/saved-queries-store'
import { useTierStore } from '@/store/tier-store'
import type { SavedQueryRecord } from '@/types/storage'

import { QueryCard } from './QueryCard'
import { SaveQueryDialog } from './SaveQueryDialog'

interface SavedQueriesPanelProps {
  /** Whether the panel is open */
  open: boolean

  /** Callback when panel should close */
  onClose: () => void

  /** User ID for filtering queries */
  userId: string

  /** Callback when user loads a query */
  onLoadQuery: (query: SavedQueryRecord) => void
}

/**
 * SavedQueriesPanel - Main component for saved queries management
 */
export function SavedQueriesPanel({
  open,
  onClose,
  userId,
  onLoadQuery,
}: SavedQueriesPanelProps) {
  const {
    queries,
    isLoading,
    error,
    folders,
    tags,
    searchText,
    selectedFolder,
    selectedTags,
    showFavoritesOnly,
    sortBy,
    sortDirection,
    setSearchText,
    setSelectedFolder,
    toggleTag,
    setShowFavoritesOnly,
    setSortBy,
    setSortDirection,
    clearFilters,
    loadQueries,
    deleteQuery,
    duplicateQuery,
    toggleFavorite,
  } = useSavedQueriesStore()

  const tierStore = useTierStore()

  // Edit dialog state
  const [editingQuery, setEditingQuery] = useState<SavedQueryRecord | null>(null)
  const [showEditDialog, setShowEditDialog] = useState(false)

  // Debounced search input
  const [searchInput, setSearchInput] = useState(searchText)

  // Debounce search text updates (300ms)
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearchText(searchInput)
    }, 300)

    return () => clearTimeout(timer)
  }, [searchInput, setSearchText])

  // Load queries when panel opens or filters change
  useEffect(() => {
    if (open && userId) {
      loadQueries(userId)
    }
  }, [
    open,
    userId,
    searchText,
    selectedFolder,
    selectedTags,
    showFavoritesOnly,
    sortBy,
    sortDirection,
    loadQueries,
  ])

  // Calculate tier limits
  const tierLimitCheck = tierStore.checkLimit('savedQueries', queries.length)
  const isNearLimit = tierLimitCheck.isNearLimit && !tierLimitCheck.isUnlimited
  const isAtLimit = tierLimitCheck.isAtLimit && !tierLimitCheck.isUnlimited

  // Active filters count
  const activeFiltersCount = useMemo(() => {
    let count = 0
    if (selectedFolder) count++
    if (selectedTags.length > 0) count++
    if (showFavoritesOnly) count++
    if (searchInput.trim()) count++
    return count
  }, [selectedFolder, selectedTags, showFavoritesOnly, searchInput])

  // Handlers
  const handleEdit = (query: SavedQueryRecord) => {
    setEditingQuery(query)
    setShowEditDialog(true)
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteQuery(id)
    } catch (err) {
      console.error('Failed to delete query:', err)
    }
  }

  const handleDuplicate = async (id: string) => {
    try {
      await duplicateQuery(id)
    } catch (err) {
      console.error('Failed to duplicate query:', err)
    }
  }

  const handleToggleFavorite = async (id: string) => {
    try {
      await toggleFavorite(id)
    } catch (err) {
      console.error('Failed to toggle favorite:', err)
    }
  }

  const handleLoad = (query: SavedQueryRecord) => {
    onLoadQuery(query)
    onClose()
  }

  const handleToggleSortDirection = () => {
    setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
  }

  const handleClearFilters = () => {
    setSearchInput('')
    clearFilters()
  }

  // Empty states
  const hasNoQueries = queries.length === 0 && !isLoading
  const hasNoResults = queries.length === 0 && !isLoading && activeFiltersCount > 0
  const hasNoFavorites = queries.length === 0 && !isLoading && showFavoritesOnly

  return (
    <>
      <Sheet open={open} onOpenChange={onClose}>
        <SheetContent
          side="right"
          className="w-full sm:max-w-2xl flex flex-col p-0"
        >
          {/* Header */}
          <SheetHeader className="px-6 pt-6 pb-4 border-b">
            <div className="flex items-center justify-between">
              <div>
                <SheetTitle className="text-xl">Saved Queries</SheetTitle>
                <SheetDescription className="mt-1">
                  Browse and manage your query library
                </SheetDescription>
              </div>
            </div>

            {/* Stats Display */}
            <div className="mt-4 space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">
                  {queries.length} {queries.length === 1 ? 'query' : 'queries'}
                </span>
                {!tierLimitCheck.isUnlimited && (
                  <span
                    className={cn(
                      'font-medium',
                      isAtLimit && 'text-destructive',
                      isNearLimit && !isAtLimit && 'text-orange-600 dark:text-orange-400'
                    )}
                  >
                    {tierLimitCheck.remaining}/{tierLimitCheck.limit} remaining
                  </span>
                )}
              </div>

              {/* Progress bar for Local tier */}
              {!tierLimitCheck.isUnlimited && (
                <div className="space-y-1">
                  <Progress
                    value={tierLimitCheck.percentage}
                    className={cn(
                      'h-2',
                      isAtLimit && '[&>*]:bg-destructive',
                      isNearLimit && !isAtLimit && '[&>*]:bg-orange-500'
                    )}
                  />
                  {isNearLimit && (
                    <div className="flex items-center gap-1.5 text-xs text-orange-600 dark:text-orange-400">
                      <AlertCircle className="h-3 w-3" />
                      <span>
                        {isAtLimit
                          ? 'Limit reached. Upgrade for unlimited queries.'
                          : 'Approaching limit. Consider upgrading.'}
                      </span>
                    </div>
                  )}
                </div>
              )}
            </div>
          </SheetHeader>

          {/* Search and Filters */}
          <div className="px-6 py-4 space-y-3 border-b bg-muted/30">
            {/* Search Bar */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search queries..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="pl-9 pr-9"
              />
              {searchInput && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7 p-0"
                  onClick={() => setSearchInput('')}
                >
                  <X className="h-4 w-4" />
                </Button>
              )}
            </div>

            {/* Filter Controls */}
            <div className="flex flex-wrap gap-2">
              {/* Folder Filter */}
              <Select
                value={selectedFolder || 'all'}
                onValueChange={(value) =>
                  setSelectedFolder(value === 'all' ? null : value)
                }
              >
                <SelectTrigger className="w-[160px] h-9">
                  <div className="flex items-center gap-2">
                    <Folder className="h-4 w-4" />
                    <SelectValue />
                  </div>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Folders</SelectItem>
                  <Separator className="my-1" />
                  {folders.map((folder) => (
                    <SelectItem key={folder} value={folder}>
                      {folder}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {/* Tags Filter (shows active tags as badges) */}
              <div className="flex-1 flex items-center gap-1 flex-wrap min-w-[200px]">
                {selectedTags.length > 0 ? (
                  <>
                    {selectedTags.map((tag) => (
                      <Badge
                        key={tag}
                        variant="secondary"
                        className="gap-1 cursor-pointer"
                        onClick={() => toggleTag(tag)}
                      >
                        {tag}
                        <X className="h-3 w-3" />
                      </Badge>
                    ))}
                  </>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-9 gap-2"
                    onClick={() => {
                      // Simple inline tag selector (could be improved with a popover)
                      const tag = prompt('Enter a tag to filter by:')
                      if (tag && tags.includes(tag)) {
                        toggleTag(tag)
                      }
                    }}
                  >
                    <Tag className="h-4 w-4" />
                    Filter by tags
                  </Button>
                )}
              </div>

              {/* Favorites Toggle */}
              <Button
                variant={showFavoritesOnly ? 'default' : 'outline'}
                size="sm"
                className="h-9 gap-2"
                onClick={() => setShowFavoritesOnly(!showFavoritesOnly)}
              >
                <Star
                  className={cn(
                    'h-4 w-4',
                    showFavoritesOnly && 'fill-current'
                  )}
                />
                Favorites
              </Button>
            </div>

            {/* Sort Controls */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">Sort by:</span>
                <Select value={sortBy} onValueChange={(value: string) => setSortBy(value as 'title' | 'created_at' | 'updated_at')}>
                  <SelectTrigger className="w-[140px] h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="title">Title</SelectItem>
                    <SelectItem value="created_at">Created</SelectItem>
                    <SelectItem value="updated_at">Updated</SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-8 p-0"
                  onClick={handleToggleSortDirection}
                  title={`Sort ${sortDirection === 'asc' ? 'ascending' : 'descending'}`}
                >
                  {sortDirection === 'asc' ? (
                    <ArrowUp className="h-4 w-4" />
                  ) : (
                    <ArrowDown className="h-4 w-4" />
                  )}
                </Button>
              </div>

              {/* Clear Filters */}
              {activeFiltersCount > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 gap-2"
                  onClick={handleClearFilters}
                >
                  <X className="h-4 w-4" />
                  Clear filters ({activeFiltersCount})
                </Button>
              )}
            </div>
          </div>

          {/* Query List */}
          <ScrollArea className="flex-1 px-6">
            <div className="py-4 space-y-3">
              {/* Loading State */}
              {isLoading && (
                <div className="flex flex-col items-center justify-center py-12 gap-3">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                  <p className="text-sm text-muted-foreground">Loading queries...</p>
                </div>
              )}

              {/* Error State */}
              {error && !isLoading && (
                <div className="flex flex-col items-center justify-center py-12 gap-3">
                  <AlertCircle className="h-8 w-8 text-destructive" />
                  <div className="text-center">
                    <p className="text-sm font-medium">Failed to load queries</p>
                    <p className="text-xs text-muted-foreground mt-1">{error}</p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => loadQueries(userId)}
                  >
                    Try again
                  </Button>
                </div>
              )}

              {/* Empty State - No Queries */}
              {hasNoQueries && !error && (
                <div className="flex flex-col items-center justify-center py-12 gap-3">
                  <Inbox className="h-12 w-12 text-muted-foreground" />
                  <div className="text-center">
                    <p className="text-sm font-medium">No saved queries yet</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Save your first query to get started
                    </p>
                  </div>
                </div>
              )}

              {/* Empty State - No Favorites */}
              {hasNoFavorites && !error && (
                <div className="flex flex-col items-center justify-center py-12 gap-3">
                  <Star className="h-12 w-12 text-muted-foreground" />
                  <div className="text-center">
                    <p className="text-sm font-medium">No favorite queries</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Mark queries as favorites to see them here
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowFavoritesOnly(false)}
                  >
                    Show all queries
                  </Button>
                </div>
              )}

              {/* Empty State - No Results */}
              {hasNoResults && !hasNoFavorites && !error && (
                <div className="flex flex-col items-center justify-center py-12 gap-3">
                  <Filter className="h-12 w-12 text-muted-foreground" />
                  <div className="text-center">
                    <p className="text-sm font-medium">No queries found</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Try adjusting your filters
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleClearFilters}
                  >
                    Clear filters
                  </Button>
                </div>
              )}

              {/* Query Cards */}
              {!isLoading && !error && queries.length > 0 && (
                <>
                  {queries.map((query) => (
                    <QueryCard
                      key={query.id}
                      query={query}
                      onLoad={handleLoad}
                      onEdit={handleEdit}
                      onDelete={handleDelete}
                      onDuplicate={handleDuplicate}
                      onToggleFavorite={handleToggleFavorite}
                      showSyncStatus={tierStore.currentTier === 'individual'}
                    />
                  ))}
                </>
              )}
            </div>
          </ScrollArea>
        </SheetContent>
      </Sheet>

      {/* Edit Dialog */}
      {editingQuery && (
        <SaveQueryDialog
          open={showEditDialog}
          onClose={() => {
            setShowEditDialog(false)
            setEditingQuery(null)
          }}
          userId={userId}
          existingQuery={editingQuery}
          onSaved={() => {
            setShowEditDialog(false)
            setEditingQuery(null)
            loadQueries(userId)
          }}
        />
      )}
    </>
  )
}
