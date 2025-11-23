import {
  ChevronDown,
  ChevronRight,
  Columns,
  Database,
  Filter,
  Folder,
  FolderOpen,
  Key,
  Loader2,
  Network,
  PanelLeftClose,
  PanelRightOpen,
  Plus,
  Table,
  Tag,
} from "lucide-react"
import { lazy, Suspense, useCallback, useEffect, useRef,useState } from "react"
import { createPortal } from "react-dom"
import { useNavigate } from "react-router-dom"

import { ConnectionSchemaViewer } from "@/components/connection-schema-viewer"
import { EnvironmentManager } from "@/components/environment-manager"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { toast } from "@/hooks/use-toast"
import { preloadComponent } from "@/lib/component-preload"
import { cn } from "@/lib/utils"
import { type DatabaseConnection,useConnectionStore } from "@/store/connection-store"
import { useQueryStore } from "@/store/query-store"

// Lazy-load the heavy schema visualizer (uses reactflow)
const SchemaVisualizerWrapper = lazy(() => import("@/components/schema-visualizer/schema-visualizer").then(m => ({ default: m.SchemaVisualizerWrapper })))
const preloadSchemaVisualizer = () => import("@/components/schema-visualizer/schema-visualizer").then(m => ({ default: m.SchemaVisualizerWrapper }))

interface SchemaTreeProps {
  nodes: SchemaNode[]
  level?: number
}

export function SchemaTree({ nodes, level = 0 }: SchemaTreeProps) {
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

        return (
          <div key={node.id}>
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-start h-8 px-2"
              style={{ paddingLeft: `${8 + level * 16}px` }}
              onClick={() => {
                if (hasChildren) {
                  toggleNode(node.id)
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

            {hasChildren && isExpanded && (
              <SchemaTree 
                nodes={node.children!} 
                level={level + 1}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}

interface SidebarProps {
  onToggle?: () => void
  isCollapsed?: boolean
}

export function Sidebar({ onToggle, isCollapsed = false }: SidebarProps) {
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
    fetchDatabases,
    switchDatabase,
  } = useConnectionStore()
  const { tabs, activeTabId, updateTab } = useQueryStore()
  const [connectingId, setConnectingId] = useState<string | null>(null)
  const [showEnvironmentManager, setShowEnvironmentManager] = useState(false)
  const [connectionDbState, setConnectionDbState] = useState<Record<string, {
    options: string[]
    loading?: boolean
    switching?: boolean
    error?: string
  }>>({})
  const dbErrorToastRef = useRef<Record<string, string | undefined>>({})
  const [dbAccordionOpen, setDbAccordionOpen] = useState<Record<string, boolean>>({})
  
  // New state for connection actions
  const [hoveredConnectionId, setHoveredConnectionId] = useState<string | null>(null)
  const [schemaViewConnectionId, setSchemaViewConnectionId] = useState<string | null>(null)
  const [diagramConnectionId, setDiagramConnectionId] = useState<string | null>(null)
  
  // Get filtered connections
  const filteredConnections = getFilteredConnections()
  const loadConnectionDatabases = useCallback(async (connectionId: string) => {
    if (!connectionId) {
      return
    }
    setConnectionDbState(prev => {
      const current = prev[connectionId]
      if (current?.loading) {
        return prev
      }
      return {
        ...prev,
        [connectionId]: {
          options: current?.options ?? [],
          loading: true,
          switching: current?.switching ?? false,
          error: undefined,
        },
      }
    })
    try {
      const dbs = await fetchDatabases(connectionId)
      setConnectionDbState(prev => ({
        ...prev,
        [connectionId]: {
          options: dbs,
          loading: false,
          switching: prev[connectionId]?.switching ?? false,
          error: dbs.length === 0 ? 'No databases available' : undefined,
        },
      }))
      delete dbErrorToastRef.current[connectionId]
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load databases'
      setConnectionDbState(prev => ({
        ...prev,
        [connectionId]: {
          options: prev[connectionId]?.options ?? [],
          loading: false,
          switching: prev[connectionId]?.switching ?? false,
          error: message,
        },
      }))
      if (dbErrorToastRef.current[connectionId] !== message) {
        toast({
          title: 'Unable to load databases',
          description: message,
          variant: 'destructive',
        })
        dbErrorToastRef.current[connectionId] = message
      }
    }
  }, [fetchDatabases])

  const handleDatabaseSelect = useCallback(async (connection: DatabaseConnection, database: string) => {
    if (!database || database === connection.database) {
      return
    }

    setConnectionDbState(prev => ({
      ...prev,
      [connection.id]: {
        options: prev[connection.id]?.options ?? [],
        loading: prev[connection.id]?.loading ?? false,
        switching: true,
        error: prev[connection.id]?.error,
      },
    }))

    try {
      await switchDatabase(connection.id, database)
      toast({
        title: 'Database switched',
        description: `${connection.name} is now using ${database}.`,
      })
    } catch (error) {
      toast({
        title: 'Failed to switch database',
        description: error instanceof Error ? error.message : 'Unable to switch database',
        variant: 'destructive',
      })
    } finally {
      setConnectionDbState(prev => ({
        ...prev,
        [connection.id]: {
          ...(prev[connection.id] ?? { options: [] }),
          switching: false,
        },
      }))
    }
  }, [switchDatabase])

  useEffect(() => {
    if (activeConnection?.id && activeConnection.isConnected) {
      void loadConnectionDatabases(activeConnection.id)
    }
  }, [activeConnection?.id, activeConnection?.isConnected, loadConnectionDatabases])

  const handleConnectionSelect = async (connection: DatabaseConnection) => {
    if (connection.sessionId) {
      setActiveConnection(connection)
      if (connection.isConnected) {
        void loadConnectionDatabases(connection.id)
      }
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

  if (isCollapsed) {
    return (
      <div className="w-10 border-r bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex-shrink-0 flex flex-col items-center py-4">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 p-0"
          onClick={onToggle}
          title="Expand sidebar"
        >
          <PanelRightOpen className="h-4 w-4" />
        </Button>
      </div>
    )
  }

  return (
    <div className="w-64 border-r bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex-shrink-0">
      <div className="flex h-full flex-col">
        {/* Connections Section */}
        <div className="p-4 border-b">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold">Connections</h2>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  navigate('/connections');
                }}
                title="Add new connection"
                className="h-7 w-7 p-0"
              >
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            {onToggle && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0"
                onClick={onToggle}
                title="Collapse sidebar"
              >
                <PanelLeftClose className="h-4 w-4" />
              </Button>
            )}
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
                );
                const dbState = connectionDbState[connection.id];

                const selectedDatabase =
                  connection.database && dbState?.options?.includes(connection.database)
                    ? connection.database
                    : undefined
                const accordionOpen = dbAccordionOpen[connection.id] ?? (connection.id === activeConnection?.id)

                return (
                  <Collapsible
                    key={connection.id}
                    open={accordionOpen}
                    onOpenChange={(open) =>
                      setDbAccordionOpen((prev) => ({
                        ...prev,
                        [connection.id]: open,
                      }))
                    }
                    className="space-y-2"
                  >
                    <div
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

                      {/* Action buttons + accordion toggle */}
                      {connection.isConnected && (
                        <>
                          {isHovered && (
                            <div className="flex items-center gap-1">
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
                                onMouseEnter={() => void preloadComponent(preloadSchemaVisualizer)}
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
                          <CollapsibleTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-6 w-6 p-0"
                              title={accordionOpen ? "Hide database selector" : "Show database selector"}
                            >
                              <ChevronDown
                                className={cn(
                                  "h-3 w-3 transition-transform",
                                  accordionOpen && "rotate-180"
                                )}
                              />
                            </Button>
                          </CollapsibleTrigger>
                        </>
                      )}
                    </div>

                    {connection.isConnected && (
                      <CollapsibleContent className="pl-6 pr-2">
                        {dbState?.options && dbState.options.length > 0 ? (
                          <Select
                            value={selectedDatabase}
                            onValueChange={(value) => handleDatabaseSelect(connection, value)}
                            disabled={dbState?.switching}
                          >
                            <SelectTrigger className="h-8 text-xs justify-between">
                              <SelectValue placeholder="Select database" />
                            </SelectTrigger>
                            <SelectContent>
                              {dbState.options.map((db) => (
                                <SelectItem key={db} value={db}>
                                  {db}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs"
                            onClick={() => loadConnectionDatabases(connection.id)}
                            disabled={dbState?.loading}
                          >
                            {dbState?.loading ? (
                              <>
                                <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                                Loading...
                              </>
                            ) : (
                              'Load databases'
                            )}
                          </Button>
                        )}
                        {dbState?.error && (
                          <p className="text-[11px] text-destructive">{dbState.error}</p>
                        )}
                      </CollapsibleContent>
                    )}
                  </Collapsible>
                );
              })
            )}
          </div>
        </div>

        <div className="flex-1" />
      </div>
      
      {/* Connection Schema Viewer Modal */}
      {schemaViewConnectionId && (
        <ConnectionSchemaViewer
          connectionId={schemaViewConnectionId}
          onClose={() => setSchemaViewConnectionId(null)}
        />
      )}
      
      {/* Connection Diagram Modal */}
      {diagramConnectionId && createPortal(
        <Suspense fallback={
          <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-50">
            <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
          </div>
        }>
          <SchemaVisualizerWrapper
            schema={[]}
            connectionId={diagramConnectionId}
            onClose={() => setDiagramConnectionId(null)}
          />
        </Suspense>,
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
