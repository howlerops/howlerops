import {
  BookOpen,
  Bot,
  ChevronDown,
  ChevronRight,
  Columns,
  Database,
  FileText,
  Filter,
  Folder,
  FolderOpen,
  GitCompare,
  Key,
  Loader2,
  Network,
  PanelLeftClose,
  PanelRightOpen,
  Plus,
  Settings,
  Table,
  Tag,
  Terminal,
} from "lucide-react"
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react"
import { createPortal } from "react-dom"
import { Link, useLocation, useNavigate } from "react-router-dom"
import { useShallow } from "zustand/react/shallow"

import { ConnectionSchemaViewer } from "@/components/connection-schema-viewer"
import { EnvironmentManager } from "@/components/environment-manager"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { toast } from "@/hooks/use-toast"
import { preloadComponent } from "@/lib/component-preload"
import { cn } from "@/lib/utils"
import { type DatabaseConnection, useConnectionStore } from "@/store/connection-store"
import { useQueryStore } from "@/store/query-store"

// Lazy-load the heavy schema visualizer (uses reactflow)
const SchemaVisualizerWrapper = lazy(() => import("@/components/schema-visualizer/schema-visualizer").then(m => ({ default: m.SchemaVisualizerWrapper })))
const preloadSchemaVisualizer = () => import("@/components/schema-visualizer/schema-visualizer").then(m => ({ default: m.SchemaVisualizerWrapper as React.ComponentType<unknown> }))

// Navigation items configuration
const NAV_ITEMS = [
  { path: '/dashboard', label: 'Queries', icon: Terminal },
  { path: '/connections', label: 'Connections', icon: Database },
  { path: '/reports', label: 'Reports', icon: FileText },
  { path: '/schema-diff', label: 'Schema Diff', icon: GitCompare },
  { path: '/data-catalog', label: 'Data Catalog', icon: BookOpen },
  { path: '/agents', label: 'Agents', icon: Bot },
  { path: '/settings', label: 'Settings', icon: Settings },
] as const

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
  const location = useLocation()
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
  } = useConnectionStore(useShallow((state) => ({
    connections: state.connections,
    activeConnection: state.activeConnection,
    setActiveConnection: state.setActiveConnection,
    connectToDatabase: state.connectToDatabase,
    isConnecting: state.isConnecting,
    activeEnvironmentFilter: state.activeEnvironmentFilter,
    availableEnvironments: state.availableEnvironments,
    setEnvironmentFilter: state.setEnvironmentFilter,
    getFilteredConnections: state.getFilteredConnections,
    fetchDatabases: state.fetchDatabases,
    switchDatabase: state.switchDatabase,
  })))
  const { tabs, activeTabId, updateTab } = useQueryStore(useShallow((state) => ({
    tabs: state.tabs,
    activeTabId: state.activeTabId,
    updateTab: state.updateTab,
  })))
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
  const [connectionsExpanded, setConnectionsExpanded] = useState(true)

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

  // Collapsed sidebar view
  if (isCollapsed) {
    return (
      <div className="w-12 border-r bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex-shrink-0 flex flex-col items-center py-3 gap-1">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 p-0 mb-2"
          onClick={onToggle}
          title="Expand sidebar"
        >
          <PanelRightOpen className="h-4 w-4" />
        </Button>

        <Separator className="w-6 my-2" />

        {/* Collapsed nav icons */}
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon
          const isActive = location.pathname === item.path
          return (
            <Button
              key={item.path}
              variant={isActive ? "secondary" : "ghost"}
              size="icon"
              className="h-8 w-8 p-0"
              onClick={() => navigate(item.path)}
              title={item.label}
            >
              <Icon className="h-4 w-4" />
            </Button>
          )
        })}
      </div>
    )
  }

  return (
    <div className="w-56 border-r bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex-shrink-0 flex flex-col">
      <ScrollArea className="flex-1">
        <div className="flex flex-col h-full">
          {/* Header with collapse button */}
          <div className="p-3 flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Navigation</span>
            {onToggle && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
                onClick={onToggle}
                title="Collapse sidebar"
              >
                <PanelLeftClose className="h-4 w-4" />
              </Button>
            )}
          </div>

          {/* Navigation Links */}
          <nav className="px-2 space-y-1">
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon
              const isActive = location.pathname === item.path
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                    isActive
                      ? "bg-secondary text-secondary-foreground"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  )}
                >
                  <Icon className="h-4 w-4 flex-shrink-0" />
                  <span className="truncate">{item.label}</span>
                </Link>
              )
            })}
          </nav>

          <Separator className="my-3" />

          {/* Connections Section */}
          <Collapsible
            open={connectionsExpanded}
            onOpenChange={setConnectionsExpanded}
            className="px-2"
          >
            <CollapsibleTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="w-full justify-between px-3 py-2 h-auto"
              >
                <div className="flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  <span className="text-xs font-semibold uppercase tracking-wider">Active Connections</span>
                </div>
                <ChevronDown className={cn("h-4 w-4 transition-transform", !connectionsExpanded && "-rotate-90")} />
              </Button>
            </CollapsibleTrigger>

            <CollapsibleContent className="space-y-1 mt-1">
              {/* Environment Filter */}
              {availableEnvironments.length > 0 && (
                <div className="px-1 mb-2 flex gap-1">
                  <Select
                    value={activeEnvironmentFilter || "__all__"}
                    onValueChange={(value) => setEnvironmentFilter(value === "__all__" ? null : value)}
                  >
                    <SelectTrigger className="h-7 text-xs flex-1">
                      <div className="flex items-center gap-1">
                        <Filter className="h-3 w-3" />
                        <SelectValue placeholder="All Envs" />
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
                    className="h-7 w-7 p-0"
                    onClick={() => setShowEnvironmentManager(true)}
                    title="Manage environments"
                  >
                    <Tag className="h-3 w-3" />
                  </Button>
                </div>
              )}

              {/* Manage Environments button when no environments exist */}
              {availableEnvironments.length === 0 && connections.length > 0 && (
                <div className="px-1 mb-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full h-7 text-xs"
                    onClick={() => setShowEnvironmentManager(true)}
                  >
                    <Tag className="h-3 w-3 mr-1" />
                    Add Environments
                  </Button>
                </div>
              )}

              {/* Connection List */}
              <div className="space-y-1 px-1">
                {filteredConnections.length === 0 && connections.length > 0 ? (
                  <div className="text-xs text-muted-foreground text-center py-3">
                    No connections for this environment
                  </div>
                ) : filteredConnections.length === 0 ? (
                  <div className="text-xs text-muted-foreground text-center py-3">
                    <p>No connections</p>
                    <Button
                      variant="link"
                      size="sm"
                      className="h-auto p-0 text-xs"
                      onClick={() => navigate('/connections')}
                    >
                      Add one
                    </Button>
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
                        className="space-y-1"
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
                            className="h-7 flex-1 justify-start overflow-hidden text-xs"
                            disabled={isConnecting}
                            onClick={() => {
                              void handleConnectionSelect(connection)
                            }}
                          >
                            <span className="truncate flex-1 text-left">{connection.name}</span>

                            <span className="ml-1 inline-flex items-center flex-shrink-0">
                              {isPending ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : connection.isConnected ? (
                                <span className="h-1.5 w-1.5 rounded-full bg-green-500" />
                              ) : null}
                            </span>
                          </Button>

                          {/* Action buttons + accordion toggle */}
                          {connection.isConnected && (
                            <>
                              {isHovered && (
                                <div className="flex items-center">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-5 w-5 p-0"
                                    onClick={() => handleViewSchema(connection.id)}
                                    title="View Tables"
                                  >
                                    <Table className="h-3 w-3" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-5 w-5 p-0"
                                    onClick={() => handleViewDiagram(connection.id)}
                                    onMouseEnter={() => void preloadComponent(preloadSchemaVisualizer)}
                                    title="View Schema Diagram"
                                  >
                                    <Network className="h-3 w-3" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-5 w-5 p-0"
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
                                  className="h-5 w-5 p-0"
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
                          <CollapsibleContent className="pl-4 pr-1">
                            {dbState?.options && dbState.options.length > 0 ? (
                              <Select
                                value={selectedDatabase}
                                onValueChange={(value) => handleDatabaseSelect(connection, value)}
                                disabled={dbState?.switching}
                              >
                                <SelectTrigger className="h-7 text-xs justify-between">
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
                                className="h-6 px-2 text-xs"
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
                              <p className="text-[10px] text-destructive mt-1">{dbState.error}</p>
                            )}
                          </CollapsibleContent>
                        )}
                      </Collapsible>
                    );
                  })
                )}
              </div>
            </CollapsibleContent>
          </Collapsible>

          <div className="flex-1" />
        </div>
      </ScrollArea>

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
