/**
 * SharedResourcesPage
 *
 * Main page for viewing and managing shared resources (connections and queries)
 * within an organization. Provides tabs, filtering, and action menus.
 *
 * @module pages/SharedResourcesPage
 */

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Database, Code2, AlertCircle, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { SharedResourceCard } from '@/components/sharing/SharedResourceCard'
import { useOrganizationStore } from '@/store/organization-store'
import { useConnectionsStore } from '@/store/connections-store'
import { useQueriesStore } from '@/store/queries-store'
import type { Connection } from '@/lib/api/connections'
import type { SavedQuery } from '@/lib/api/queries'

/**
 * SharedResourcesPage Component
 *
 * Usage:
 * ```tsx
 * <Route path="/shared" element={<SharedResourcesPage />} />
 * ```
 */
export function SharedResourcesPage() {
  const [activeTab, setActiveTab] = useState<'connections' | 'queries'>(
    'connections'
  )

  const { currentOrgId, organizations } = useOrganizationStore()
  const {
    sharedConnections,
    fetchSharedConnections,
    unshareConnection,
    deleteConnection,
    loading: connectionsLoading,
    error: connectionsError,
  } = useConnectionsStore()

  const {
    sharedQueries,
    fetchSharedQueries,
    unshareQuery,
    deleteQuery,
    loading: queriesLoading,
    error: queriesError,
  } = useQueriesStore()

  // Get current organization details
  const currentOrg = organizations.find((o) => o.id === currentOrgId)

  // Fetch shared resources when organization changes
  useEffect(() => {
    if (currentOrgId) {
      fetchSharedConnections(currentOrgId).catch((error) => {
        console.error('Failed to fetch shared connections:', error)
      })

      fetchSharedQueries(currentOrgId).catch((error) => {
        console.error('Failed to fetch shared queries:', error)
      })
    }
  }, [currentOrgId, fetchSharedConnections, fetchSharedQueries])

  // Handle unsharing connection
  const handleUnshareConnection = async (connection: Connection) => {
    try {
      await unshareConnection(connection.id)
      toast.success('Connection is now personal')

      // Refresh list
      if (currentOrgId) {
        await fetchSharedConnections(currentOrgId)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : 'Failed to unshare connection'
      )
    }
  }

  // Handle deleting connection
  const handleDeleteConnection = async (connection: Connection) => {
    if (
      !confirm(
        `Are you sure you want to delete "${connection.name}"? This action cannot be undone.`
      )
    ) {
      return
    }

    try {
      await deleteConnection(connection.id)
      toast.success('Connection deleted')

      // Refresh list
      if (currentOrgId) {
        await fetchSharedConnections(currentOrgId)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : 'Failed to delete connection'
      )
    }
  }

  // Handle unsharing query
  const handleUnshareQuery = async (query: SavedQuery) => {
    try {
      await unshareQuery(query.id)
      toast.success('Query is now personal')

      // Refresh list
      if (currentOrgId) {
        await fetchSharedQueries(currentOrgId)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : 'Failed to unshare query'
      )
    }
  }

  // Handle deleting query
  const handleDeleteQuery = async (query: SavedQuery) => {
    if (
      !confirm(
        `Are you sure you want to delete "${query.title}"? This action cannot be undone.`
      )
    ) {
      return
    }

    try {
      await deleteQuery(query.id)
      toast.success('Query deleted')

      // Refresh list
      if (currentOrgId) {
        await fetchSharedQueries(currentOrgId)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : 'Failed to delete query'
      )
    }
  }

  // No organization selected
  if (!currentOrgId) {
    return (
      <div className="container mx-auto p-6 max-w-7xl">
        <div className="mb-6">
          <h1 className="text-3xl font-bold mb-2">Shared Resources</h1>
          <p className="text-muted-foreground">
            View and manage resources shared within your organization
          </p>
        </div>

        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>No Organization Selected</AlertTitle>
          <AlertDescription>
            Please select an organization to view shared resources. You can
            select an organization from the organization switcher in the
            sidebar.
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  const isLoading = connectionsLoading || queriesLoading
  const hasError = connectionsError || queriesError

  return (
    <div className="container mx-auto p-6 max-w-7xl">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-2">
          <Users className="h-6 w-6 text-primary" />
          <h1 className="text-3xl font-bold">Shared Resources</h1>
        </div>
        <p className="text-muted-foreground">
          Resources shared within{' '}
          <span className="font-medium">{currentOrg?.name}</span>
        </p>
      </div>

      {/* Error Alert */}
      {hasError && (
        <Alert variant="destructive" className="mb-6">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error Loading Resources</AlertTitle>
          <AlertDescription>
            {connectionsError || queriesError}
          </AlertDescription>
        </Alert>
      )}

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as any)}>
        <TabsList className="mb-6">
          <TabsTrigger value="connections" className="gap-2">
            <Database className="h-4 w-4" />
            Connections
            {sharedConnections.length > 0 && (
              <span className="ml-1 text-xs bg-primary/20 px-1.5 py-0.5 rounded-full">
                {sharedConnections.length}
              </span>
            )}
          </TabsTrigger>

          <TabsTrigger value="queries" className="gap-2">
            <Code2 className="h-4 w-4" />
            Queries
            {sharedQueries.length > 0 && (
              <span className="ml-1 text-xs bg-primary/20 px-1.5 py-0.5 rounded-full">
                {sharedQueries.length}
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        {/* Connections Tab */}
        <TabsContent value="connections" className="space-y-4">
          {isLoading && sharedConnections.length === 0 ? (
            <div className="text-center py-12">
              <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent" />
              <p className="mt-4 text-muted-foreground">
                Loading connections...
              </p>
            </div>
          ) : sharedConnections.length === 0 ? (
            <div className="text-center py-12 border-2 border-dashed rounded-lg">
              <Database className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">
                No Shared Connections
              </h3>
              <p className="text-muted-foreground mb-4">
                No database connections have been shared in this organization
                yet.
              </p>
              <Button variant="outline">Create Connection</Button>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {sharedConnections.map((connection) => (
                <SharedResourceCard
                  key={connection.id}
                  resource={connection}
                  type="connection"
                  onView={(resource) => {
                    // TODO: Open connection details modal
                    console.log('View connection:', resource)
                  }}
                  onEdit={(resource) => {
                    // TODO: Open connection edit form
                    console.log('Edit connection:', resource)
                  }}
                  onUnshare={(resource) =>
                    handleUnshareConnection(resource as Connection)
                  }
                  onDelete={(resource) =>
                    handleDeleteConnection(resource as Connection)
                  }
                  onUse={(resource) => {
                    // TODO: Connect to database
                    console.log('Use connection:', resource)
                  }}
                />
              ))}
            </div>
          )}
        </TabsContent>

        {/* Queries Tab */}
        <TabsContent value="queries" className="space-y-4">
          {isLoading && sharedQueries.length === 0 ? (
            <div className="text-center py-12">
              <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent" />
              <p className="mt-4 text-muted-foreground">Loading queries...</p>
            </div>
          ) : sharedQueries.length === 0 ? (
            <div className="text-center py-12 border-2 border-dashed rounded-lg">
              <Code2 className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">No Shared Queries</h3>
              <p className="text-muted-foreground mb-4">
                No queries have been shared in this organization yet.
              </p>
              <Button variant="outline">Create Query</Button>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {sharedQueries.map((query) => (
                <SharedResourceCard
                  key={query.id}
                  resource={query}
                  type="query"
                  onView={(resource) => {
                    // TODO: Open query details modal
                    console.log('View query:', resource)
                  }}
                  onEdit={(resource) => {
                    // TODO: Open query editor
                    console.log('Edit query:', resource)
                  }}
                  onUnshare={(resource) =>
                    handleUnshareQuery(resource as SavedQuery)
                  }
                  onDelete={(resource) =>
                    handleDeleteQuery(resource as SavedQuery)
                  }
                  onUse={(resource) => {
                    // TODO: Execute query
                    console.log('Run query:', resource)
                  }}
                />
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
