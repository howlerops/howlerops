import React, { useState, useEffect, useCallback, useMemo } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  ReactFlowProvider,
} from 'reactflow'
import 'reactflow/dist/style.css'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import {
  X,
  Search,
  Settings,
  Download,
  RotateCcw,
  Maximize2,
  Minimize2,
  Layers,
  Database,
} from 'lucide-react'

import { TableNode } from './table-node'
import { SchemaErrorBoundary } from './schema-error-boundary'
import { LayoutEngine } from '@/lib/schema-layout'
import { SchemaConfigBuilder } from '@/lib/schema-config'
import { SchemaNode } from '@/hooks/use-schema-introspection'
import { useDebounce } from '@/hooks/use-debounce'
import {
  SchemaConfig,
  LayoutAlgorithm,
  LayoutOptions,
  FilterOptions,
  SchemaVisualizerNode,
  SchemaVisualizerEdge,
} from '@/types/schema-visualizer'

interface SchemaVisualizerProps {
  schema: SchemaNode[]
  onClose: () => void
}

export function SchemaVisualizer({ schema, onClose }: SchemaVisualizerProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])
  const [schemaConfig, setSchemaConfig] = useState<SchemaConfig | null>(null)

  // Memoize nodeTypes to prevent unnecessary re-renders
  const nodeTypes = useMemo(() => ({
    table: TableNode,
  }), [])
  
  // UI State
  const [searchInput, setSearchInput] = useState('')
  const debouncedSearchTerm = useDebounce(searchInput, 300) // 300ms delay
  const [selectedSchemas, setSelectedSchemas] = useState<string[]>([])
  const [showForeignKeys, setShowForeignKeys] = useState(true)
  const [showPrimaryKeys, setShowPrimaryKeys] = useState(true)
  const [layoutAlgorithm, setLayoutAlgorithm] = useState<LayoutAlgorithm>('hierarchical')
  const [isFullscreen, setIsFullscreen] = useState(true)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  // Initialize schema configuration
  useEffect(() => {
    const initializeSchema = async () => {
      if (schema.length > 0) {
        try {
          const config = await SchemaConfigBuilder.fromSchemaNodes(schema)
          setSchemaConfig(config)

          console.log('Schema config created:', {
            tables: config.tables.length,
            edges: config.edges.length,
            edgeDetails: config.edges.map(e => ({
              id: e.id,
              source: e.source,
              target: e.target,
              relation: e.relation,
              label: e.label
            }))
          })

          const { nodes: flowNodes, edges: flowEdges } = SchemaConfigBuilder.toReactFlowNodes(config)
          setNodes(flowNodes)
          setEdges(flowEdges)

          console.log('ReactFlow nodes and edges:', {
            nodes: flowNodes.length,
            edges: flowEdges.length,
            edgeTypes: flowEdges.map(e => e.type)
          })

          // Smart layout selection based on table count
          const tableCount = config.tables.length
          if (tableCount < 50) {
            setLayoutAlgorithm('hierarchical')
          } else if (tableCount < 200) {
            setLayoutAlgorithm('grid')
            // Show warning for large schemas
            if (tableCount > 100) {
              console.info(`Large schema detected: ${tableCount} tables. Consider using filters to improve performance.`)
            }
          } else {
            setLayoutAlgorithm('grid')
            setSidebarCollapsed(false) // Force sidebar open for filtering
            console.warn(`Very large schema: ${tableCount} tables. Please use filters before visualizing.`)
          }

          // Extract unique schemas for filtering
          const uniqueSchemas = [...new Set(config.tables.map(table => table.schema))]
          setSelectedSchemas(uniqueSchemas)
        } catch (error) {
          console.error('Failed to initialize schema configuration:', error)
        }
      }
    }

    initializeSchema()
  }, [schema, setNodes, setEdges])

  // Filter options
  const filterOptions: FilterOptions = useMemo(() => ({ // eslint-disable-line @typescript-eslint/no-unused-vars
    searchTerm: debouncedSearchTerm,
    selectedSchemas,
    showForeignKeys,
    showPrimaryKeys,
  }), [debouncedSearchTerm, selectedSchemas, showForeignKeys, showPrimaryKeys])

  // Create node lookup map for performance
  const nodeMap = useMemo(() => {
    const map = new Map()
    nodes.forEach(node => map.set(node.id, node))
    return map
  }, [nodes])

  // Apply filters
  const filteredNodes = useMemo(() => {
    return nodes.filter((node) => {
      const tableData = node.data as { name: string; schema: string; columns: Array<{ name: string }> }

      // Search filter (using debounced term)
      if (debouncedSearchTerm) {
        const searchLower = debouncedSearchTerm.toLowerCase()
        const matchesTable = tableData.name.toLowerCase().includes(searchLower)
        const matchesColumn = tableData.columns.some((col) =>
          col.name.toLowerCase().includes(searchLower)
        )
        if (!matchesTable && !matchesColumn) return false
      }

      // Schema filter
      if (selectedSchemas.length > 0 && !selectedSchemas.includes(tableData.schema)) {
        return false
      }

      return true
    })
  }, [nodes, debouncedSearchTerm, selectedSchemas])

  // Create filtered nodes Set for O(1) lookup
  const filteredNodeIds = useMemo(() => {
    return new Set(filteredNodes.map(n => n.id))
  }, [filteredNodes])

  const filteredEdges = useMemo(() => {
    if (!showForeignKeys) return []

    // Use Set for O(1) lookup instead of O(n) includes()
    return edges.filter((edge) => {
      return filteredNodeIds.has(edge.source) && filteredNodeIds.has(edge.target)
    })
  }, [edges, filteredNodeIds, showForeignKeys])

  // Layout functions
  const applyLayout = useCallback((algorithm: LayoutAlgorithm) => {
    if (!schemaConfig) return
    
    const layoutOptions: LayoutOptions = {
      algorithm,
      spacing: { x: 300, y: 200 },
    }
    
    const { nodes: layoutedNodes } = LayoutEngine.applyLayout(
      filteredNodes as SchemaVisualizerNode[],
      filteredEdges as SchemaVisualizerEdge[],
      layoutOptions
    )
    
    setNodes(layoutedNodes)
  }, [schemaConfig, filteredNodes, filteredEdges, setNodes])

  // Export functions
  const exportConfig = useCallback(() => {
    if (!schemaConfig) return
    
    const jsonString = SchemaConfigBuilder.exportToJSON(schemaConfig)
    const blob = new Blob([jsonString], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'schema-config.json'
    a.click()
    URL.revokeObjectURL(url)
  }, [schemaConfig])

  const exportCSV = useCallback(() => {
    if (!schemaConfig) return
    
    const csvString = SchemaConfigBuilder.generateCSVExport(schemaConfig)
    const blob = new Blob([csvString], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'schema-export.csv'
    a.click()
    URL.revokeObjectURL(url)
  }, [schemaConfig])

  // Copy positions to clipboard
  const copyPositions = useCallback(() => {
    const positions = nodes.reduce((acc, node) => {
      acc[node.id] = node.position
      return acc
    }, {} as Record<string, { x: number; y: number }>)
    
    navigator.clipboard.writeText(JSON.stringify(positions, null, 2))
  }, [nodes])

  // Handle edge connections
  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge(params, eds))
    },
    [setEdges]
  )

  if (!schemaConfig) {
    return (
      <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center">
        <Card className="w-96">
          <CardContent className="p-6">
            <div className="text-center">
              <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
              <h3 className="text-lg font-semibold mb-2">No Schema Data</h3>
              <p className="text-muted-foreground mb-4">
                Connect to a database to visualize its schema
              </p>
              <Button onClick={onClose}>Close</Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Show performance warning for large schemas
  const showPerformanceWarning = schemaConfig && schemaConfig.tables.length > 100

  return (
    <div className={`fixed inset-0 bg-background z-50 ${isFullscreen ? '' : 'p-4'}`}>
      <div className="h-full flex flex-col">
        {/* Performance Warning Banner */}
        {showPerformanceWarning && filteredNodes.length > 50 && (
          <div className="bg-yellow-500/10 border-b border-yellow-500/20 px-4 py-2">
            <p className="text-sm text-yellow-700 dark:text-yellow-400">
              ⚠️ Large schema detected ({schemaConfig?.tables.length} tables).
              Use filters to reduce visible tables for better performance.
              Currently showing {filteredNodes.length} tables.
            </p>
          </div>
        )}

        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center space-x-4">
            <h2 className="text-xl font-semibold">Schema Visualizer</h2>
            <Badge variant="secondary">
              {filteredNodes.length} table{filteredNodes.length !== 1 ? 's' : ''}
            </Badge>
            {schemaConfig && schemaConfig.tables.length !== filteredNodes.length && (
              <Badge variant="outline">
                {schemaConfig.tables.length} total
              </Badge>
            )}
          </div>
          
          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            >
              <Settings className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsFullscreen(!isFullscreen)}
            >
              {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            </Button>
            <Button variant="outline" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar */}
          {!sidebarCollapsed && (
            <div className="w-80 border-r bg-muted/30 p-4 space-y-4 overflow-y-auto">
              {/* Search */}
              <div className="space-y-2">
                <Label htmlFor="search">Search Tables</Label>
                <div className="relative">
                  <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="search"
                    placeholder="Search tables or columns..."
                    value={searchInput}
                    onChange={(e) => setSearchInput(e.target.value)}
                    className="pl-10"
                  />
                </div>
              </div>

              {/* Schema Filter */}
              <div className="space-y-2">
                <Label>Schemas</Label>
                <div className="space-y-2 max-h-32 overflow-y-auto">
                  {Object.keys(schemaConfig.schemaColors).map((schemaName) => (
                    <div key={schemaName} className="flex items-center space-x-2">
                      <Switch
                        checked={selectedSchemas.includes(schemaName)}
                        onCheckedChange={(checked) => {
                          if (checked) {
                            setSelectedSchemas([...selectedSchemas, schemaName])
                          } else {
                            setSelectedSchemas(selectedSchemas.filter(s => s !== schemaName))
                          }
                        }}
                      />
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: schemaConfig.schemaColors[schemaName] }}
                      />
                      <span className="text-sm">{schemaName}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Display Options */}
              <div className="space-y-3">
                <Label>Display Options</Label>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="foreign-keys" className="text-sm">Foreign Keys</Label>
                    <Switch
                      id="foreign-keys"
                      checked={showForeignKeys}
                      onCheckedChange={setShowForeignKeys}
                    />
                  </div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="primary-keys" className="text-sm">Primary Keys</Label>
                    <Switch
                      id="primary-keys"
                      checked={showPrimaryKeys}
                      onCheckedChange={setShowPrimaryKeys}
                    />
                  </div>
                </div>
              </div>

              {/* Layout Options */}
              <div className="space-y-2">
                <Label>Layout Algorithm</Label>
                <Select value={layoutAlgorithm} onValueChange={(value: LayoutAlgorithm) => setLayoutAlgorithm(value)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="force">Force Directed</SelectItem>
                    <SelectItem value="hierarchical">Hierarchical</SelectItem>
                    <SelectItem value="grid">Grid</SelectItem>
                    <SelectItem value="circular">Circular</SelectItem>
                  </SelectContent>
                </Select>
                <Button onClick={() => applyLayout(layoutAlgorithm)} className="w-full">
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Apply Layout
                </Button>
              </div>

              {/* Export Options */}
              <div className="space-y-2">
                <Label>Export</Label>
                <div className="space-y-2">
                  <Button onClick={exportConfig} variant="outline" className="w-full">
                    <Download className="h-4 w-4 mr-2" />
                    Export Config
                  </Button>
                  <Button onClick={exportCSV} variant="outline" className="w-full">
                    <Download className="h-4 w-4 mr-2" />
                    Export CSV
                  </Button>
                  <Button onClick={copyPositions} variant="outline" className="w-full">
                    <Layers className="h-4 w-4 mr-2" />
                    Copy Positions
                  </Button>
                </div>
              </div>
            </div>
          )}

          {/* Main Visualization Area */}
          <div className="flex-1">
            <ReactFlow
              nodes={filteredNodes}
              edges={filteredEdges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              nodeTypes={nodeTypes}
              onlyRenderVisibleElements={true}
              fitView
              attributionPosition="bottom-left"
            >
              <Background />
              <Controls />
              <MiniMap />
            </ReactFlow>
          </div>
        </div>
      </div>
    </div>
  )
}

// Wrapper with ReactFlowProvider and Error Boundary
export function SchemaVisualizerWrapper(props: SchemaVisualizerProps) {
  return (
    <SchemaErrorBoundary onReset={() => window.location.reload()}>
      <ReactFlowProvider>
        <SchemaVisualizer {...props} />
      </ReactFlowProvider>
    </SchemaErrorBoundary>
  )
}
