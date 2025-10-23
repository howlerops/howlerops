import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { createPortal } from "react-dom"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useConnectionStore, type DatabaseConnection } from "@/store/connection-store"
import { useQueryStore } from "@/store/query-store"
import { useSchemaIntrospection, SchemaNode } from "@/hooks/use-schema-introspection"
import { SchemaVisualizerWrapper } from "@/components/schema-visualizer/schema-visualizer"
import { ConnectionSchemaViewer } from "@/components/connection-schema-viewer"
import { EnvironmentManager } from "@/components/environment-manager"
import {
  Database,
  Table,
  Plus,
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  Columns,
  Key,
  RefreshCw,
  AlertCircle,
  Loader2,
  Network,
  Filter,
  Tag,
} from "lucide-react"
import { cn } from "@/lib/utils"

interface SchemaTreeProps {
  nodes: SchemaNode[]
  level?: number
  collapsedSchemas?: Set<string>
  onToggleSchema?: (schemaId: string) => void
}

export function SchemaTree({ nodes, level = 0, collapsedSchemas = new Set(), onToggleSchema }: SchemaTreeProps) {
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(
    new Set(nodes.filter(node => node.expanded).map(node => node.id))
  )

  const toggleNode = (nodeId: string) => {
    setExpandedNodes(prev => {
      const newSet = new Set(prev)
      if (newSet.has(nodeId)) {
        newSet.delete(nodeId)
      } else {
        newSet.add(nodeId)
      }
      return newSet
    })
  }

  const getIcon = (node: SchemaNode, isExpanded: boolean) => {
    switch (node.type) {
      case 'database':
      case 'schema':
        return isExpanded ? <FolderOpen className="h-4 w-4" /> : <Folder className="h-4 w-4" />
      case 'table':
        return <Table className="h-4 w-4" />
      case 'column':
        return node.name.includes('PK') ? <Key className="h-4 w-4" /> : <Columns className="h-4 w-4" />
      default:
        return <div className="h-4 w-4" />
    }
  }

  return (
    <div className="space-y-1">
      {nodes.map((node) => {
        const isExpanded = expandedNodes.has(node.id)
        const hasChildren = node.children && node.children.length > 0
        const isSchemaCollapsed = node.type === 'schema' && collapsedSchemas.has(node.id)

        return (
          <div key={node.id}>
            <Button
              variant="ghost"
              size="sm"
              className={cn(
                "w-full justify-start h-8 px-2",
                `pl-${2 + level * 4}`
              )}
              onClick={() => {
                // Always toggle expansion for nodes with children
                if (hasChildren) {
                  toggleNode(node.id)
                }
                // Additionally call schema toggle callback if provided
                if (node.type === 'schema' && onToggleSchema) {
                  onToggleSchema(node.id)
                }
              }}
            >
              {hasChildren && (
                <div className="mr-1">
                  {isExpanded ? (
                    <ChevronDown className="h-3 w-3" />
                  ) : (
                    <ChevronRight className="h-3 w-3" />
                  )}
                </div>
              )}
              {!hasChildren && <div className="w-4" />}
              <div className="mr-2">
                {getIcon(node, isExpanded)}
              </div>
              <span className="text-sm truncate">{node.name}</span>
              {node.type === 'schema' && node.children && (
                <Badge variant="secondary" className="ml-auto text-xs">
                  {node.children.length}
                </Badge>
              )}
            </Button>

            {hasChildren && isExpanded && !isSchemaCollapsed && (
              <SchemaTree 
                nodes={node.children!} 
                level={level + 1}
                collapsedSchemas={collapsedSchemas}
                onToggleSchema={onToggleSchema}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}

export function Sidebar() {
  const navigate = useNavigate()
  const {
    connections,
    activeConnection,
    setActiveConnection,
    connectToDatabase,
    isConnecting,
    activeEnvironmentFilter,
    availableEnvironments,
    setEnvironmentFilter,
    getFilteredConnections,
  } = useConnectionStore()
  const { tabs, activeTabId, updateTab } = useQueryStore()
  const { schema, loading, error, refreshSchema } = useSchemaIntrospection()
  const [connectingId, setConnectingId] = useState<string | null>(null)
  const [showVisualizer, setShowVisualizer] = useState(false)
  const [collapsedSchemas, setCollapsedSchemas] = useState<Set<string>>(new Set())
  const [showEnvironmentManager, setShowEnvironmentManager] = useState(false)
  
  // New state for connection actions
  const [hoveredConnectionId, setHoveredConnectionId] = useState<string | null>(null)
  const [schemaViewConnectionId, setSchemaViewConnectionId] = useState<string | null>(null)
  const [diagramConnectionId, setDiagramConnectionId] = useState<string | null>(null)
  
  // Get filtered connections
  const filteredConnections = getFilteredConnections()

  const handleConnectionSelect = async (connection: DatabaseConnection) => {
    if (connection.sessionId) {
      setActiveConnection(connection)
      return
    }

    setConnectingId(connection.id)
    try {
      await connectToDatabase(connection.id)
    } catch (error) {
      console.error('Failed to activate connection:', error)
    } finally {
      setConnectingId(null)
    }
  }

  const toggleSchema = (schemaId: string) => {
    setCollapsedSchemas(prev => {
      const newSet = new Set(prev)
      if (newSet.has(schemaId)) {
        newSet.delete(schemaId)
      } else {
        newSet.add(schemaId)
      }
      return newSet
    })
  }

  const handleAddToQueryTab = (connectionId: string) => {
    const activeTab = tabs.find(tab => tab.id === activeTabId)
    if (!activeTab) {
      // No active tab, could show a toast notification
      return
    }

    // Check if connection is already in the tab
    const isAlreadyInTab = activeTab.connectionId === connectionId || 
      (activeTab.selectedConnectionIds && activeTab.selectedConnectionIds.includes(connectionId))
    
    if (isAlreadyInTab) {
      return
    }

    // Add connection to the active tab
    if (activeTab.selectedConnectionIds) {
      // Multi-DB mode: add to selectedConnectionIds
      updateTab(activeTab.id, {
        selectedConnectionIds: [...(activeTab.selectedConnectionIds || []), connectionId]
      })
    } else {
      // Single-DB mode: set connectionId
      updateTab(activeTab.id, {
        connectionId: connectionId,
        selectedConnectionIds: [connectionId]
      })
    }
  }

  const handleViewSchema = (connectionId: string) => {
    setSchemaViewConnectionId(connectionId)
  }

  const handleViewDiagram = (connectionId: string) => {
    setDiagramConnectionId(connectionId)
  }

  return (
    <div className="w-64 border-r bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-full flex-col">
        {/* Connections Section */}
        <div className="p-4 border-b">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold">Connections</h2>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                navigate('/connections');
              }}
              title="Add new connection"
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>

          {/* Environment Filter */}
          {availableEnvironments.length > 0 && (
            <div className="mb-3 flex gap-2">
              <Select
                value={activeEnvironmentFilter || "__all__"}
                onValueChange={(value) => setEnvironmentFilter(value === "__all__" ? null : value)}
              >
                <SelectTrigger className="h-8 text-xs flex-1">
                  <div className="flex items-center gap-2">
                    <Filter className="h-3 w-3" />
                    <SelectValue placeholder="All Environments" />
                  </div>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All Environments</SelectItem>
                  {availableEnvironments.map((env) => (
                    <SelectItem key={env} value={env}>
                      {env}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 px-2"
                onClick={() => setShowEnvironmentManager(true)}
                title="Manage environments"
              >
                <Tag className="h-3 w-3" />
              </Button>
            </div>
          )}

          {/* Manage Environments button when no environments exist */}
          {availableEnvironments.length === 0 && connections.length > 0 && (
            <div className="mb-3">
              <Button
                variant="outline"
                size="sm"
                className="w-full h-8 text-xs"
                onClick={() => setShowEnvironmentManager(true)}
              >
                <Tag className="h-3 w-3 mr-2" />
                Add Environments
              </Button>
            </div>
          )}

          <div className="space-y-2">
            {filteredConnections.length === 0 && connections.length > 0 ? (
              <div className="text-xs text-muted-foreground text-center py-4">
                No connections for this environment
              </div>
            ) : filteredConnections.length === 0 ? (
              <div className="text-xs text-muted-foreground text-center py-4">
                No connections configured
              </div>
            ) : (
              filteredConnections.map((connection) => {
                const isActive = activeConnection?.id === connection.id
                const isPending = connectingId === connection.id
                const isHovered = hoveredConnectionId === connection.id
                const activeTab = tabs.find(tab => tab.id === activeTabId)
                const isInActiveTab = activeTab && (
                  activeTab.connectionId === connection.id || 
                  (activeTab.selectedConnectionIds && activeTab.selectedConnectionIds.includes(connection.id))
                )

                return (
                  <div 
                    key={connection.id} 
                    className="flex items-center gap-1 group"
                    onMouseEnter={() => setHoveredConnectionId(connection.id)}
                    onMouseLeave={() => setHoveredConnectionId(null)}
                  >
                    {/* Connection button */}
                    <Button
                      variant={isActive || isPending ? "secondary" : "ghost"}
                      size="sm"
                      className="h-8 flex-1 justify-start overflow-hidden"
                      disabled={isConnecting}
                      onClick={() => {
                        void handleConnectionSelect(connection)
                      }}
                    >
                      <Database className="mr-2 h-4 w-4 flex-shrink-0" />
                      <span className="truncate flex-1 text-left">{connection.name}</span>
                      
                      {/* Show environment chips when "All" is selected */}
                      {!activeEnvironmentFilter && connection.environments && connection.environments.length > 0 && (
                        <div className="flex gap-1 ml-2 flex-shrink-0">
                          {connection.environments.slice(0, 2).map((env) => (
                            <Badge key={env} variant="outline" className="text-[10px] px-1 py-0 h-4">
                              {env}
                            </Badge>
                          ))}
                          {connection.environments.length > 2 && (
                            <Badge variant="outline" className="text-[10px] px-1 py-0 h-4">
                              +{connection.environments.length - 2}
                            </Badge>
                          )}
                        </div>
                      )}
                      
                      <span className="ml-2 inline-flex items-center flex-shrink-0">
                        {isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : connection.isConnected ? (
                          <span className="h-2 w-2 rounded-full bg-primary" />
                        ) : null}
                      </span>
                    </Button>

                    {/* Action buttons - show on hover */}
                    {isHovered && connection.isConnected && (
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0"
                          onClick={() => handleViewSchema(connection.id)}
                          title="View Tables"
                        >
                          <Table className="h-3 w-3" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0"
                          onClick={() => handleViewDiagram(connection.id)}
                          title="View Schema Diagram"
                        >
                          <Network className="h-3 w-3" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0"
                          onClick={() => handleAddToQueryTab(connection.id)}
                          disabled={!activeTab || isInActiveTab}
                          title={!activeTab ? "No active query tab" : isInActiveTab ? "Already in query tab" : "Add to Query Tab"}
                        >
                          <Plus className="h-3 w-3" />
                        </Button>
                      </div>
                    )}
                  </div>
                )
              })
            )}
          </div>
        </div>

        {/* Schema Explorer */}
        <div className="flex-1 p-4 overflow-hidden">
          <Card className="h-full flex flex-col">
            <CardHeader className="pb-2 flex flex-row items-center justify-between shrink-0">
              <CardTitle className="text-sm">Schema Explorer</CardTitle>
              <div className="flex items-center gap-1">
                {activeConnection && schema.length > 0 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowVisualizer(true)}
                    className="h-6 w-6 p-0"
                    title="Schema Visualizer"
                  >
                    <Network className="h-3 w-3" />
                  </Button>
                )}
                {activeConnection && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={refreshSchema}
                    disabled={loading}
                    className="h-6 w-6 p-0"
                    title="Refresh Schema"
                  >
                    <RefreshCw className={cn("h-3 w-3", loading && "animate-spin")} />
                  </Button>
                )}
              </div>
            </CardHeader>
            <CardContent className="pt-0 flex-1 overflow-hidden">
              <div className="h-full overflow-y-auto">
                {error ? (
                  <div className="text-xs text-destructive text-center py-4 flex items-center justify-center">
                    <AlertCircle className="h-3 w-3 mr-1" />
                    {error}
                  </div>
                ) : loading ? (
                  <div className="text-xs text-muted-foreground text-center py-4">
                    Loading schema...
                  </div>
                ) : activeConnection ? (
                  schema.length > 0 ? (
                    <SchemaTree 
                      nodes={schema} 
                      collapsedSchemas={collapsedSchemas}
                      onToggleSchema={toggleSchema}
                    />
                  ) : (
                    <div className="text-xs text-muted-foreground text-center py-4">
                      No schemas found
                    </div>
                  )
                ) : (
                  <div className="text-xs text-muted-foreground text-center py-4">
                    Choose a database to explore schema
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
      
      {/* Schema Visualizer Modal */}
      {showVisualizer && createPortal(
        <SchemaVisualizerWrapper 
          schema={schema} 
          onClose={() => setShowVisualizer(false)} 
        />,
        document.body
      )}
      
      {/* Connection Schema Viewer Modal */}
      {schemaViewConnectionId && (
        <ConnectionSchemaViewer
          connectionId={schemaViewConnectionId}
          onClose={() => setSchemaViewConnectionId(null)}
        />
      )}
      
      {/* Connection Diagram Modal */}
      {diagramConnectionId && createPortal(
        <SchemaVisualizerWrapper 
          schema={[]} 
          connectionId={diagramConnectionId}
          onClose={() => setDiagramConnectionId(null)} 
        />,
        document.body
      )}
      
      {/* Environment Manager Modal */}
      {showEnvironmentManager && (
        <EnvironmentManager
          open={showEnvironmentManager}
          onClose={() => setShowEnvironmentManager(false)}
        />
      )}
    </div>
  )
}
