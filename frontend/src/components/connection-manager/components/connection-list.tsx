import { Database, Plus, Tag } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardTitle } from "@/components/ui/card"

import type { ConnectionGroup, DatabaseConnection } from "../types"
import { ConnectionCard } from "./connection-card"

interface ConnectionListProps {
  connections: DatabaseConnection[]
  groupedConnections: ConnectionGroup[]
  groupByEnvironment: boolean
  isConnecting: boolean
  hasConnections: boolean
  activeEnvironmentFilter: string | null
  onAddConnection: () => void
  onEditConnection: (connection: DatabaseConnection) => void
  onDeleteConnection: (connection: DatabaseConnection) => void
  onConnectConnection: (connection: DatabaseConnection) => void
  onDiagnosticsConnection: (connection: DatabaseConnection) => void
  onClearEnvironmentFilter: () => void
}

/**
 * Component for displaying the list/grid of connections
 */
export function ConnectionList({
  connections,
  groupedConnections,
  groupByEnvironment,
  isConnecting,
  hasConnections,
  activeEnvironmentFilter,
  onAddConnection,
  onEditConnection,
  onDeleteConnection,
  onConnectConnection,
  onDiagnosticsConnection,
  onClearEnvironmentFilter,
}: ConnectionListProps) {
  const renderConnectionCard = (connection: DatabaseConnection) => (
    <ConnectionCard
      key={connection.id}
      connection={connection}
      isConnecting={isConnecting}
      onEdit={onEditConnection}
      onDelete={onDeleteConnection}
      onConnect={onConnectConnection}
      onDiagnostics={onDiagnosticsConnection}
    />
  )

  const renderEmptyState = (message: string) => (
    <Card className="col-span-full">
      <CardContent className="flex flex-col items-center justify-center py-12 text-center space-y-4">
        <Database className="h-12 w-12 text-muted-foreground" />
        <div>
          <CardTitle className="mb-2">No connections</CardTitle>
          <CardDescription>{message}</CardDescription>
        </div>
        {!hasConnections ? (
          <Button onClick={onAddConnection}>
            <Plus className="h-4 w-4 mr-2" />
            Add Connection
          </Button>
        ) : activeEnvironmentFilter ? (
          <Button variant="outline" onClick={onClearEnvironmentFilter}>
            Clear Environment Filter
          </Button>
        ) : (
          <Button onClick={onAddConnection}>
            <Plus className="h-4 w-4 mr-2" />
            Add Connection
          </Button>
        )}
      </CardContent>
    </Card>
  )

  // Grouped view
  if (groupByEnvironment) {
    if (groupedConnections.length > 0) {
      return (
        <div className="space-y-6">
          {groupedConnections.map((group) => (
            <div key={group.key} className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Tag className="h-4 w-4 text-muted-foreground" />
                  <h3 className="text-sm font-semibold">{group.label}</h3>
                </div>
                <Badge variant="outline" className="text-xs">
                  {group.connections.length} {group.connections.length === 1 ? 'connection' : 'connections'}
                </Badge>
              </div>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {group.connections.map(renderConnectionCard)}
              </div>
            </div>
          ))}
        </div>
      )
    }

    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {renderEmptyState(
          activeEnvironmentFilter
            ? 'No connections match this environment filter.'
            : 'Add your first database connection to get started.'
        )}
      </div>
    )
  }

  // Flat view
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {connections.length > 0
        ? connections.map(renderConnectionCard)
        : renderEmptyState(
            !hasConnections
              ? 'Add your first database connection to get started.'
              : 'No connections match this environment filter.'
          )}
    </div>
  )
}
