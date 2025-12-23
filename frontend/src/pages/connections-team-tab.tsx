/**
 * Team Connections Tab
 *
 * Displays shared connections and queries within the current organization.
 * Used as a tab within the Connections page.
 */

import { AlertCircle, Code2, Database } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { SharedResourceCard } from '@/components/sharing/SharedResourceCard'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { Connection } from '@/lib/api/connections'
import type { SavedQuery } from '@/lib/api/queries'
import { useConnectionsStore } from '@/store/connections-store'
import { useOrganizationStore } from '@/store/organization-store'
import { useQueriesStore } from '@/store/queries-store'

type ResourceTab = 'connections' | 'queries'

interface TeamConnectionsTabProps {
  hideHeader?: boolean
}

export function TeamConnectionsTab({ hideHeader = false }: TeamConnectionsTabProps) {
  const [activeTab, setActiveTab] = useState<ResourceTab>('connections')

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

      if (currentOrgId) {
        await fetchSharedQueries(currentOrgId)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : 'Failed to delete query'
      )
    }
  }

  const isLoading = connectionsLoading || queriesLoading
  const hasError = connectionsError || queriesError

  return (
    <div className={hideHeader ? "" : "p-6"}>
      {/* Header */}
      {!hideHeader && (
        <div className="mb-6">
          <p className="text-muted-foreground">
            Resources shared within{' '}
            <span className="font-medium">{currentOrg?.name}</span>
          </p>
        </div>
      )}

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

      {/* Sub-tabs for Connections vs Queries */}
      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as ResourceTab)}
      >
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
              <p className="text-sm text-muted-foreground">
                Share a connection from "My Connections" tab to make it
                available to your team.
              </p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {sharedConnections.map((connection) => (
                <SharedResourceCard
                  key={connection.id}
                  resource={connection}
                  type="connection"
                  onView={(resource) => {
                    console.log('View connection:', resource)
                  }}
                  onEdit={(resource) => {
                    console.log('Edit connection:', resource)
                  }}
                  onUnshare={(resource) =>
                    handleUnshareConnection(resource as Connection)
                  }
                  onDelete={(resource) =>
                    handleDeleteConnection(resource as Connection)
                  }
                  onUse={(resource) => {
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
              <p className="text-sm text-muted-foreground">
                Save and share queries from the Query Editor to make them
                available to your team.
              </p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {sharedQueries.map((query) => (
                <SharedResourceCard
                  key={query.id}
                  resource={query}
                  type="query"
                  onView={(resource) => {
                    console.log('View query:', resource)
                  }}
                  onEdit={(resource) => {
                    console.log('Edit query:', resource)
                  }}
                  onUnshare={(resource) =>
                    handleUnshareQuery(resource as SavedQuery)
                  }
                  onDelete={(resource) =>
                    handleDeleteQuery(resource as SavedQuery)
                  }
                  onUse={(resource) => {
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
