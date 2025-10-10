import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { createPortal } from "react-dom"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { useConnectionStore, type DatabaseConnection } from "@/store/connection-store"
import { useSchemaIntrospection, SchemaNode } from "@/hooks/useSchemaIntrospection"
import { SchemaVisualizerWrapper } from "@/components/schema-visualizer/SchemaVisualizer"
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
} from "lucide-react"
import { cn } from "@/lib/utils"

interface SchemaTreeProps {
  nodes: SchemaNode[]
  level?: number
  collapsedSchemas?: Set<string>
  onToggleSchema?: (schemaId: string) => void
}

function SchemaTree({ nodes, level = 0, collapsedSchemas = new Set(), onToggleSchema }: SchemaTreeProps) {
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
                if (node.type === 'schema' && onToggleSchema) {
                  onToggleSchema(node.id)
                } else if (hasChildren) {
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
  } = useConnectionStore()
  const { schema, loading, error, refreshSchema } = useSchemaIntrospection()
  const [connectingId, setConnectingId] = useState<string | null>(null)
  const [showVisualizer, setShowVisualizer] = useState(false)
  const [collapsedSchemas, setCollapsedSchemas] = useState<Set<string>>(new Set())

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

          <div className="space-y-2">
            {connections.length === 0 ? (
              <div className="text-xs text-muted-foreground text-center py-4">
                No connections configured
              </div>
            ) : (
              connections.map((connection) => {
                const isActive = activeConnection?.id === connection.id
                const isPending = connectingId === connection.id

                return (
                  <Button
                    key={connection.id}
                    variant={isActive || isPending ? "secondary" : "ghost"}
                    size="sm"
                    className="h-8 w-full justify-start"
                    disabled={isConnecting}
                    onClick={() => {
                      void handleConnectionSelect(connection)
                    }}
                  >
                    <Database className="mr-2 h-4 w-4" />
                    <span className="truncate">{connection.name}</span>
                    <span className="ml-auto inline-flex items-center">
                      {isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : connection.isConnected ? (
                        <span className="h-2 w-2 rounded-full bg-green-500" />
                      ) : null}
                    </span>
                  </Button>
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
                    Connect to a database to explore schema
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
    </div>
  )
}
