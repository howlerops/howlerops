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
  Node,
  Edge,
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
  FolderMinus,
  FolderPlus,
} from 'lucide-react'

import { TableNode } from './table-node'
import { SchemaSummaryNode } from './schema-summary-node'
import { CustomEdge } from './custom-edge'
import { RelationshipInspector } from './relationship-inspector'
import { SchemaErrorBoundary } from './schema-error-boundary'
import { LayoutEngine } from '@/lib/schema-layout'
import { SchemaConfigBuilder } from '@/lib/schema-config'
import { SchemaNode } from '@/hooks/use-schema-introspection'
import { useDebounce } from '@/hooks/use-debounce'
import {
  SchemaConfig,
  LayoutAlgorithm,
  LayoutOptions,
  SchemaVisualizerNode,
  SchemaVisualizerEdge,
  EdgeConfig,
  TableConfig,
} from '@/types/schema-visualizer'

interface SchemaVisualizerProps {
  schema: SchemaNode[]
  onClose: () => void
  connectionId?: string
}

export function SchemaVisualizer({ schema, onClose }: SchemaVisualizerProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])
  const [schemaConfig, setSchemaConfig] = useState<SchemaConfig | null>(null)

  // Memoize nodeTypes and edgeTypes to prevent unnecessary re-renders
  const nodeTypes = useMemo(() => ({
    table: TableNode,
    schemaSummary: SchemaSummaryNode,
  }), [])

  const edgeTypes = useMemo(() => ({
    smoothstep: CustomEdge,
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
  const [collapsedSchemas, setCollapsedSchemas] = useState<Set<string>>(new Set())
  const [focusNeighborsOnly, setFocusNeighborsOnly] = useState(false)
  const [detailMode, setDetailMode] = useState<'auto' | 'full' | 'compact'>('auto')
  const [viewportZoom, setViewportZoom] = useState(1)

  // Interactive state
  const [selectedTableId, setSelectedTableId] = useState<string | null>(null)
  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<{
    edge: EdgeConfig
    sourceTable: TableConfig
    targetTable: TableConfig
    position: { x: number; y: number }
  } | null>(null)

  // Performance optimizations
  const shouldDisableAnimations = useMemo(() => {
    return schemaConfig && schemaConfig.tables.length > 50
  }, [schemaConfig])

  // Performance degradation thresholds
  const performanceLevel = useMemo(() => {
    if (!schemaConfig) return 'optimal'
    const tableCount = schemaConfig.tables.length

    if (tableCount < 50) return 'optimal'
    if (tableCount < 100) return 'good'
    if (tableCount < 200) return 'degraded'
    return 'critical'
  }, [schemaConfig])

  const showPerformanceWarning = useMemo(() => {
    return performanceLevel === 'degraded' || performanceLevel === 'critical'
  }, [performanceLevel])

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
          setNodes(flowNodes as Node[])
          setEdges(flowEdges as Edge[])

          console.log('ReactFlow nodes and edges:', {
            nodes: flowNodes.length,
            edges: flowEdges.length,
            edgeTypes: (flowEdges as Edge[]).map((e: Edge) => e.type)
          })

          // Smart layout selection based on table count
          // Performance thresholds based on ReactFlow limitations
          const tableCount = config.tables.length
          if (tableCount < 50) {
            // Optimal range: full features
            setLayoutAlgorithm('hierarchical')
          } else if (tableCount < 100) {
            // Degraded range: switch to grid, keep animations
            setLayoutAlgorithm('grid')
            console.info(`Medium schema detected: ${tableCount} tables. Using grid layout for better performance.`)
          } else if (tableCount < 200) {
            // Minimal range: grid only, no animations
            setLayoutAlgorithm('grid')
            setSidebarCollapsed(false) // Encourage filtering
            console.warn(`Large schema detected: ${tableCount} tables. Performance may be degraded. Use filters to reduce complexity.`)
          } else {
            // Critical range: warn user strongly
            setLayoutAlgorithm('grid')
            setSidebarCollapsed(false) // Force sidebar open for filtering
            console.error(`Very large schema: ${tableCount} tables. Browser visualization not recommended. Consider using a dedicated database client tool or export to documentation.`)
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

  const tableSchemaLookup = useMemo(() => {
    if (!schemaConfig) return new Map<string, string>()
    const map = new Map<string, string>()
    schemaConfig.tables.forEach((table) => {
      map.set(table.id, table.schema)
    })
    return map
  }, [schemaConfig])

  const adjacencyMap = useMemo(() => {
    if (!schemaConfig) return new Map<string, Set<string>>()
    const map = new Map<string, Set<string>>()
    schemaConfig.edges.forEach((edge) => {
      if (!map.has(edge.source)) map.set(edge.source, new Set())
      if (!map.has(edge.target)) map.set(edge.target, new Set())
      map.get(edge.source)!.add(edge.target)
      map.get(edge.target)!.add(edge.source)
    })
    return map
  }, [schemaConfig])

  const neighborWhitelist = useMemo(() => {
    if (!focusNeighborsOnly || !selectedTableId) return null
    const neighbors = new Set<string>([selectedTableId])
    adjacencyMap.get(selectedTableId)?.forEach((neighbor) => neighbors.add(neighbor))
    return neighbors
  }, [focusNeighborsOnly, selectedTableId, adjacencyMap])

  useEffect(() => {
    if (!selectedTableId) return
    const schemaName = tableSchemaLookup.get(selectedTableId)
    if (schemaName && collapsedSchemas.has(schemaName)) {
      setSelectedTableId(null)
    }
  }, [collapsedSchemas, selectedTableId, tableSchemaLookup])

  const baseFilteredNodes = useMemo(() => {
    return nodes
      .filter((node) => {
        if (node.type !== 'table') return true
        const tableData = node.data as { name: string; schema: string; columns: Array<{ name: string }> }

        if (selectedSchemas.length > 0 && !selectedSchemas.includes(tableData.schema)) {
          return false
        }

        if (neighborWhitelist && !neighborWhitelist.has(node.id)) {
          return false
        }

        if (debouncedSearchTerm) {
          const searchLower = debouncedSearchTerm.toLowerCase()
          const matchesTable = tableData.name.toLowerCase().includes(searchLower)
          const matchesColumn = tableData.columns.some((col) =>
            col.name.toLowerCase().includes(searchLower)
          )
          if (!matchesTable && !matchesColumn) return false
        }

        return true
      })
      .map((node) => {
        if (node.type !== 'table') {
          return node
        }

        const isFocused = selectedTableId === node.id
        const isDimmed =
          selectedTableId !== null &&
          selectedTableId !== node.id &&
          (focusNeighborsOnly ? !!neighborWhitelist : true)

        return {
          ...node,
          data: {
            ...node.data,
            isFocused,
            isDimmed,
          },
        }
      })
  }, [nodes, debouncedSearchTerm, selectedSchemas, selectedTableId, neighborWhitelist])

  const computedDetailLevel = useMemo<'full' | 'compact'>(() => {
    if (detailMode === 'full' || detailMode === 'compact') {
      return detailMode
    }

    const totalTables = schemaConfig?.tables.length ?? 0
    if (totalTables > 140) {
      return 'compact'
    }

    if (totalTables > 90 && viewportZoom < 1) {
      return 'compact'
    }

    return viewportZoom < 0.8 ? 'compact' : 'full'
  }, [detailMode, viewportZoom, schemaConfig])

  const expandSchema = useCallback((schemaName: string) => {
    setCollapsedSchemas((prev) => {
      if (!prev.has(schemaName)) return prev
      const next = new Set(prev)
      next.delete(schemaName)
      return next
    })
  }, [])

  const toggleSchemaCollapse = useCallback((schemaName: string) => {
    setCollapsedSchemas((prev) => {
      const next = new Set(prev)
      if (next.has(schemaName)) {
        next.delete(schemaName)
      } else {
        next.add(schemaName)
      }
      return next
    })
  }, [])

  const {
    displayNodes,
    collapsedNodeMap,
    compactNodeIds,
  } = useMemo(() => {
    const collapsedMap = new Map<string, string>()
    const nodesOut: Node[] = []
    const summaryMeta = new Map<string, { nodes: Node[] }>()
    const compactSet = new Set<string>()

    baseFilteredNodes.forEach((node) => {
      if (node.type !== 'table') {
        nodesOut.push(node)
        return
      }

      const schemaName = (node.data as TableConfig).schema
      const updatedNode: Node = {
        ...node,
        data: {
          ...node.data,
          detailLevel: computedDetailLevel,
          showPrimaryKeys,
        },
      }

      if (computedDetailLevel === 'compact') {
        compactSet.add(node.id)
      }

      if (schemaName && collapsedSchemas.has(schemaName)) {
        let summary = summaryMeta.get(schemaName)
        if (!summary) {
          summary = { nodes: [] }
          summaryMeta.set(schemaName, summary)
        }
        summary.nodes.push(updatedNode)
        collapsedMap.set(node.id, `schema-summary-${schemaName}`)
      } else {
        nodesOut.push(updatedNode)
      }
    })

    summaryMeta.forEach((summary, schemaName) => {
      if (summary.nodes.length === 0) return
      const centroid = summary.nodes.reduce(
        (acc, node) => {
          acc.x += node.position.x
          acc.y += node.position.y
          return acc
        },
        { x: 0, y: 0 }
      )
      centroid.x /= summary.nodes.length
      centroid.y /= summary.nodes.length

      nodesOut.push({
        id: `schema-summary-${schemaName}`,
        type: 'schemaSummary',
        position: centroid,
        data: {
          schema: schemaName,
          color: schemaConfig?.schemaColors[schemaName] || schemaConfig?.schemaColors.DEFAULT,
          tableCount: summary.nodes.length,
          onExpand: expandSchema,
        },
      })
    })

    return {
      displayNodes: nodesOut,
      collapsedNodeMap: collapsedMap,
      compactNodeIds: compactSet,
    }
  }, [baseFilteredNodes, collapsedSchemas, computedDetailLevel, showPrimaryKeys, schemaConfig, expandSchema])

  const visibleNodeIds = useMemo(() => {
    return new Set(displayNodes.map((node) => node.id))
  }, [displayNodes])

  const visibleTableCount = useMemo(() => {
    return displayNodes.filter((node) => node.type === 'table').length
  }, [displayNodes])

  // Debounced edge hover handler
  const debouncedHoveredEdgeId = useDebounce(hoveredEdgeId, 50)

  // Handle edge hover
  const handleEdgeHover = useCallback((edgeId: string | null) => {
    setHoveredEdgeId(edgeId)
  }, [])

  const filteredEdges = useMemo(() => {
    if (!showForeignKeys) return []

    const aggregate = new Map<string, Edge>()
    const selectedSummaryId = selectedTableId ? collapsedNodeMap.get(selectedTableId) : null

    edges.forEach((edge) => {
      const isSourceCompact = compactNodeIds.has(edge.source)
      const isTargetCompact = compactNodeIds.has(edge.target)
      const mappedSource = collapsedNodeMap.get(edge.source) ?? edge.source
      const mappedTarget = collapsedNodeMap.get(edge.target) ?? edge.target

      if (mappedSource === mappedTarget) {
        return
      }

      if (!visibleNodeIds.has(mappedSource) || !visibleNodeIds.has(mappedTarget)) {
        return
      }

      const aggregateKey =
        mappedSource.startsWith('schema-summary-') || mappedTarget.startsWith('schema-summary-')
          ? `${mappedSource}->${mappedTarget}`
          : edge.id

      const isConnectedToSelectedTable =
        selectedTableId !== null &&
        (edge.source === selectedTableId ||
          edge.target === selectedTableId ||
          (selectedSummaryId &&
            (mappedSource === selectedSummaryId || mappedTarget === selectedSummaryId)))

      const isHighlighted = debouncedHoveredEdgeId === aggregateKey || isConnectedToSelectedTable
      const isDimmed = selectedTableId !== null && !isConnectedToSelectedTable
      const shouldAnimate = !shouldDisableAnimations && edge.animated

      const baseEdge: Edge = {
        ...edge,
        id: aggregateKey,
        source: mappedSource,
        target: mappedTarget,
        sourceHandle:
          isSourceCompact || collapsedNodeMap.has(edge.source) ? undefined : edge.sourceHandle,
        targetHandle:
          isTargetCompact || collapsedNodeMap.has(edge.target) ? undefined : edge.targetHandle,
        animated: shouldAnimate,
        data: {
          ...edge.data,
          onEdgeHover: handleEdgeHover,
          isHighlighted,
          isDimmed,
        },
      }

      const isAggregateEdge = aggregateKey !== edge.id
      if (isAggregateEdge) {
        const existing = aggregate.get(aggregateKey)
        if (existing) {
          const currentCount = existing.data?.aggregateCount || 1
          aggregate.set(aggregateKey, {
            ...existing,
            label: `${currentCount + 1} relations`,
            data: {
              ...existing.data,
              aggregateCount: currentCount + 1,
            },
          })
        } else {
          aggregate.set(aggregateKey, {
            ...baseEdge,
            label: '1 relation',
            data: {
              ...baseEdge.data,
              aggregateCount: 1,
            },
          })
        }
      } else {
        aggregate.set(aggregateKey, baseEdge)
      }
    })

    return Array.from(aggregate.values())
  }, [
    edges,
    showForeignKeys,
    collapsedNodeMap,
    visibleNodeIds,
    compactNodeIds,
    selectedTableId,
    debouncedHoveredEdgeId,
    handleEdgeHover,
    shouldDisableAnimations,
  ])

  // Layout functions
  const applyLayout = useCallback((algorithm: LayoutAlgorithm) => {
    if (!schemaConfig) return
    
    const layoutOptions: LayoutOptions = {
      algorithm,
      spacing: { x: 300, y: 200 },
    }
    
    const { nodes: layoutedNodes } = LayoutEngine.applyLayout(
      displayNodes as SchemaVisualizerNode[],
      filteredEdges as SchemaVisualizerEdge[],
      layoutOptions
    )
    
    setNodes(layoutedNodes)
  }, [schemaConfig, displayNodes, filteredEdges, setNodes])

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

  // Handle node click (focus mode)
  const handleNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      event.stopPropagation()
       if (node.type !== 'table') {
         return
       }
      // Toggle focus: if already selected, deselect; otherwise select
      setSelectedTableId((prevId) => (prevId === node.id ? null : node.id))
    },
    []
  )

  // Handle edge click (show inspector)
  const handleEdgeClick = useCallback(
    (event: React.MouseEvent, edge: Edge) => {
      event.stopPropagation()

      if (!schemaConfig) return

      // Find the edge configuration data
      const edgeData = edge.data?.data as EdgeConfig | undefined
      if (!edgeData || edge.data?.aggregateCount) return

      // Find source and target tables
      const sourceTable = schemaConfig.tables.find((t) => t.id === edge.source)
      const targetTable = schemaConfig.tables.find((t) => t.id === edge.target)

      if (!sourceTable || !targetTable) return

      // Set the selected edge with position
      setSelectedEdge({
        edge: edgeData,
        sourceTable,
        targetTable,
        position: { x: event.clientX, y: event.clientY },
      })
    },
    [schemaConfig]
  )

  // Handle pane click (deselect)
  const handlePaneClick = useCallback(() => {
    setSelectedTableId(null)
    setSelectedEdge(null)
  }, [])

  const handleViewportChange = useCallback((_: any, viewport: { zoom: number }) => {
    setViewportZoom(viewport.zoom)
  }, [])

  // Keyboard support for focus mode
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setSelectedTableId(null)
        setSelectedEdge(null)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

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

  return (
    <div className={`fixed inset-0 bg-background z-50 ${isFullscreen ? '' : 'p-4'}`}>
      <div className="h-full flex flex-col">
        {/* Performance Warning Banner */}
        {showPerformanceWarning && (
          <div className={
            performanceLevel === 'critical'
              ? 'bg-red-500/10 border-b border-red-500/20 px-4 py-3'
              : 'bg-yellow-500/10 border-b border-yellow-500/20 px-4 py-2'
          }>
            <p className={
              performanceLevel === 'critical'
                ? 'text-sm text-red-700 dark:text-red-400'
                : 'text-sm text-yellow-700 dark:text-yellow-400'
            }>
              {performanceLevel === 'critical' ? (
                <>
                  ⚠️ <strong>Critical:</strong> Very large schema ({schemaConfig?.tables.length} tables).
                  Browser visualization not recommended above 200 tables.
                  Consider using a dedicated database client tool (DBeaver, DataGrip) or export to documentation.
                  Currently showing {visibleTableCount} table{visibleTableCount !== 1 ? 's' : ''}.
                </>
              ) : (
                <>
                  ⚠️ Large schema detected ({schemaConfig?.tables.length} tables).
                  Performance may be degraded. Use filters to reduce complexity.
                  Currently showing {visibleTableCount} table{visibleTableCount !== 1 ? 's' : ''}.
                  {shouldDisableAnimations && ' Edge animations disabled for better performance.'}
                </>
              )}
            </p>
          </div>
        )}

        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center space-x-4">
            <h2 className="text-xl font-semibold">Schema Visualizer</h2>
            <Badge variant="secondary">
              {visibleTableCount} table{visibleTableCount !== 1 ? 's' : ''}
            </Badge>
            {schemaConfig && schemaConfig.tables.length !== visibleTableCount && (
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
                    <div key={schemaName} className="flex items-center justify-between space-x-2">
                      <div className="flex items-center space-x-2">
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
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => toggleSchemaCollapse(schemaName)}
                        title={collapsedSchemas.has(schemaName) ? 'Expand schema' : 'Collapse schema'}
                      >
                        {collapsedSchemas.has(schemaName) ? (
                          <FolderPlus className="h-4 w-4" />
                        ) : (
                          <FolderMinus className="h-4 w-4" />
                        )}
                      </Button>
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
                  <div className="flex items-center justify-between">
                    <Label htmlFor="focus-mode" className="text-sm">Focus neighbors</Label>
                    <Switch
                      id="focus-mode"
                      disabled={!selectedTableId}
                      checked={focusNeighborsOnly}
                      onCheckedChange={setFocusNeighborsOnly}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Select a table, then enable focus mode to show only directly related tables.
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <Label>Detail Density</Label>
                <Select value={detailMode} onValueChange={(value: 'auto' | 'full' | 'compact') => setDetailMode(value)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto">Auto (zoom aware)</SelectItem>
                    <SelectItem value="full">Full detail</SelectItem>
                    <SelectItem value="compact">Compact cards</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Compact mode removes column lists for tighter layouts. Auto switches to compact when zoomed out or on very large schemas.
                </p>
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
              nodes={displayNodes}
              edges={filteredEdges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onNodeClick={handleNodeClick}
              onEdgeClick={handleEdgeClick}
              onPaneClick={handlePaneClick}
              onMove={handleViewportChange}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              onlyRenderVisibleElements={true}
              fitView
              attributionPosition="bottom-left"
            >
              <Background />
              <Controls />
              <MiniMap />
            </ReactFlow>

            {/* Relationship Inspector */}
            {selectedEdge && (
              <RelationshipInspector
                edge={selectedEdge.edge}
                sourceTable={selectedEdge.sourceTable}
                targetTable={selectedEdge.targetTable}
                position={selectedEdge.position}
                onClose={() => setSelectedEdge(null)}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// Wrapper with ReactFlowProvider and Error Boundary
export function SchemaVisualizerWrapper(props: SchemaVisualizerProps) {
  const [loadedSchema, setLoadedSchema] = useState<SchemaNode[]>(props.schema)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Load schema for specific connection if connectionId is provided
  useEffect(() => {
    if (!props.connectionId) {
      setLoadedSchema(props.schema)
      return
    }

    const loadConnectionSchema = async () => {
      setLoading(true)
      setError(null)

      try {
        // Import the Wails API dynamically
        const { GetSchemas, GetTables } = await import('../../../wailsjs/go/main/App')
        
        // Get the connection from the store
        const { useConnectionStore } = await import('@/store/connection-store')
        const connections = useConnectionStore.getState().connections
        const connection = connections.find(conn => conn.id === props.connectionId)
        
        if (!connection?.sessionId) {
          throw new Error('Connection not found or not connected')
        }

        const schemas = await GetSchemas(connection.sessionId)

        if (!schemas || !Array.isArray(schemas)) {
          throw new Error('Failed to load schemas')
        }

        // Get tables for each schema
        const schemaNames = (schemas as string[]) || []
        const allTables: Array<{ name: string; schema: string }> = []
        
        for (const schemaName of schemaNames) {
          try {
            const tables = await GetTables(connection.sessionId, schemaName)
            if (Array.isArray(tables)) {
              allTables.push(...tables.map(table => ({
                name: table.name || '',
                schema: schemaName
              })))
            }
          } catch (err) {
            console.warn(`Failed to load tables for schema ${schemaName}:`, err)
          }
        }

        // Convert to SchemaNode format
        const schemaNodes: SchemaNode[] = []

        // Process each schema
        for (const schemaName of schemaNames) {
          const schemaTables = allTables.filter(t => t.schema === schemaName)
          
          // Skip migration table and internal postgres tables
          const nonMigrationTables = schemaTables.filter(t => 
            t.name !== 'schema_migrations' && 
            t.name !== 'goose_db_version' &&
            t.name !== '_prisma_migrations' &&
            !t.name.startsWith('__drizzle') &&
            !schemaName.startsWith('pg_temp') &&
            !schemaName.startsWith('pg_toast')
          )
          
          // Skip empty schemas
          if (nonMigrationTables.length === 0) {
            continue
          }
          
          const tablesWithColumns: SchemaNode[] = nonMigrationTables.map(table => ({
            id: `${props.connectionId}-${schemaName}-${table.name}`,
            name: table.name,
            type: 'table' as const,
            schema: table.schema,
            children: [] // Columns loaded on demand
          }))
          
          schemaNodes.push({
            id: `${props.connectionId}-${schemaName}`,
            name: schemaName,
            type: 'schema' as const,
            children: tablesWithColumns
          })
        }

        setLoadedSchema(schemaNodes)
      } catch (err) {
        console.error('Failed to load schema:', err)
        setError(err instanceof Error ? err.message : 'Failed to load schema')
        setLoadedSchema([])
      } finally {
        setLoading(false)
      }
    }

    loadConnectionSchema()
  }, [props.connectionId, props.schema])

  if (loading) {
    return (
      <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center">
        <Card className="w-96 h-64 flex items-center justify-center">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
            <p className="text-muted-foreground">Loading schema...</p>
          </div>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center">
        <Card className="w-96 h-64 flex items-center justify-center">
          <div className="text-center">
            <p className="text-destructive mb-4">{error}</p>
            <Button onClick={props.onClose}>Close</Button>
          </div>
        </Card>
      </div>
    )
  }

  return (
    <SchemaErrorBoundary onReset={() => window.location.reload()}>
      <ReactFlowProvider>
        <SchemaVisualizer {...props} schema={loadedSchema} />
      </ReactFlowProvider>
    </SchemaErrorBoundary>
  )
}
