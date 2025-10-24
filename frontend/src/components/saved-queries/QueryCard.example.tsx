/**
 * QueryCard Usage Examples
 *
 * Demonstrates how to use the QueryCard component in different scenarios
 */

import { QueryCard } from './QueryCard'
import type { SavedQueryRecord } from '@/types/storage'
import { useSavedQueriesStore } from '@/store/saved-queries-store'
import { toast } from 'sonner'

/**
 * Example 1: Basic QueryCard usage in a list
 *
 * This shows the typical use case for displaying saved queries
 * with full functionality
 */
export function SavedQueriesList() {
  const { queries, deleteQuery, toggleFavorite, duplicateQuery } =
    useSavedQueriesStore()

  const handleLoadQuery = (query: SavedQueryRecord) => {
    // Load query into the editor
    console.log('Loading query:', query.title)
    // In real app: setEditorContent(query.query_text)
    toast.success(`Loaded: ${query.title}`)
  }

  const handleEditQuery = (query: SavedQueryRecord) => {
    // Open edit dialog
    console.log('Editing query:', query.id)
    // In real app: openEditDialog(query)
  }

  const handleDeleteQuery = async (id: string) => {
    try {
      await deleteQuery(id)
      toast.success('Query deleted')
    } catch (error) {
      toast.error('Failed to delete query')
    }
  }

  const handleDuplicateQuery = async (id: string) => {
    try {
      await duplicateQuery(id)
      toast.success('Query duplicated')
    } catch (error) {
      toast.error('Failed to duplicate query')
    }
  }

  const handleToggleFavorite = async (id: string) => {
    try {
      await toggleFavorite(id)
    } catch (error) {
      toast.error('Failed to update favorite status')
    }
  }

  return (
    <div className="space-y-3">
      {queries.map((query) => (
        <QueryCard
          key={query.id}
          query={query}
          onLoad={handleLoadQuery}
          onEdit={handleEditQuery}
          onDelete={handleDeleteQuery}
          onDuplicate={handleDuplicateQuery}
          onToggleFavorite={handleToggleFavorite}
          showSyncStatus={false} // Set to true for Individual tier
        />
      ))}
    </div>
  )
}

/**
 * Example 2: QueryCard with sync status (Individual tier)
 *
 * Shows how to enable sync status indicators for Individual tier users
 */
export function SavedQueriesListWithSync() {
  const { queries } = useSavedQueriesStore()

  return (
    <div className="space-y-3">
      {queries.map((query) => (
        <QueryCard
          key={query.id}
          query={query}
          onLoad={(q) => console.log('Load:', q.title)}
          onEdit={(q) => console.log('Edit:', q.id)}
          onDelete={(id) => console.log('Delete:', id)}
          onDuplicate={(id) => console.log('Duplicate:', id)}
          onToggleFavorite={(id) => console.log('Toggle favorite:', id)}
          showSyncStatus={true} // Enable for Individual tier
        />
      ))}
    </div>
  )
}

/**
 * Example 3: Filtered view (favorites only)
 *
 * Shows how to display only favorite queries
 */
export function FavoriteQueriesList() {
  const { queries } = useSavedQueriesStore()
  const favoriteQueries = queries.filter((q) => q.is_favorite)

  if (favoriteQueries.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <p>No favorite queries yet</p>
        <p className="text-sm mt-2">
          Click the star icon on any query to add it to favorites
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {favoriteQueries.map((query) => (
        <QueryCard
          key={query.id}
          query={query}
          onLoad={(q) => console.log('Load:', q.title)}
          onEdit={(q) => console.log('Edit:', q.id)}
          onDelete={(id) => console.log('Delete:', id)}
          onDuplicate={(id) => console.log('Duplicate:', id)}
          onToggleFavorite={(id) => console.log('Toggle favorite:', id)}
        />
      ))}
    </div>
  )
}

/**
 * Example 4: Grid layout
 *
 * Shows how to display QueryCards in a responsive grid
 */
export function SavedQueriesGrid() {
  const { queries } = useSavedQueriesStore()

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {queries.map((query) => (
        <QueryCard
          key={query.id}
          query={query}
          onLoad={(q) => console.log('Load:', q.title)}
          onEdit={(q) => console.log('Edit:', q.id)}
          onDelete={(id) => console.log('Delete:', id)}
          onDuplicate={(id) => console.log('Duplicate:', id)}
          onToggleFavorite={(id) => console.log('Toggle favorite:', id)}
        />
      ))}
    </div>
  )
}

/**
 * Example 5: With search and filter
 *
 * Shows how to integrate QueryCard with search functionality
 */
export function SearchableQueriesList({ searchTerm }: { searchTerm: string }) {
  const { queries } = useSavedQueriesStore()

  const filteredQueries = queries.filter(
    (q) =>
      q.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      q.description?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      q.tags.some((tag) => tag.toLowerCase().includes(searchTerm.toLowerCase()))
  )

  if (filteredQueries.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        No queries found matching "{searchTerm}"
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {filteredQueries.map((query) => (
        <QueryCard
          key={query.id}
          query={query}
          onLoad={(q) => console.log('Load:', q.title)}
          onEdit={(q) => console.log('Edit:', q.id)}
          onDelete={(id) => console.log('Delete:', id)}
          onDuplicate={(id) => console.log('Duplicate:', id)}
          onToggleFavorite={(id) => console.log('Toggle favorite:', id)}
        />
      ))}
    </div>
  )
}

/**
 * Example 6: Grouped by folder
 *
 * Shows how to display QueryCards grouped by folders
 */
export function QueriesByFolder() {
  const { queries, folders } = useSavedQueriesStore()

  const queriesByFolder = folders.reduce(
    (acc, folder) => {
      acc[folder] = queries.filter((q) => q.folder === folder)
      return acc
    },
    {} as Record<string, SavedQueryRecord[]>
  )

  // Add uncategorized queries
  const uncategorized = queries.filter((q) => !q.folder)
  if (uncategorized.length > 0) {
    queriesByFolder['Uncategorized'] = uncategorized
  }

  return (
    <div className="space-y-6">
      {Object.entries(queriesByFolder).map(([folder, folderQueries]) => (
        <div key={folder}>
          <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
            {folder}
            <span className="text-sm text-muted-foreground font-normal">
              ({folderQueries.length})
            </span>
          </h3>
          <div className="space-y-3">
            {folderQueries.map((query) => (
              <QueryCard
                key={query.id}
                query={query}
                onLoad={(q) => console.log('Load:', q.title)}
                onEdit={(q) => console.log('Edit:', q.id)}
                onDelete={(id) => console.log('Delete:', id)}
                onDuplicate={(id) => console.log('Duplicate:', id)}
                onToggleFavorite={(id) => console.log('Toggle favorite:', id)}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
