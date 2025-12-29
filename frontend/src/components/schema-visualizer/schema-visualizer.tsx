/**
 * Schema Visualizer - React Flow Performance Optimizations
 *
 * This component follows React Flow best practices for performance optimization.
 * Key optimizations implemented (per https://reactflow.dev/learn/advanced-use/performance):
 *
 * 1. NODE/EDGE TYPES OUTSIDE COMPONENT
 *    - ERD_NODE_TYPES, CLASSIC_NODE_TYPES defined outside SchemaVisualizer
 *    - ERD_EDGE_TYPES, CLASSIC_EDGE_TYPES defined outside SchemaVisualizer
 *    - This prevents React from creating new objects on every render
 *
 * 2. CUSTOM NODES/EDGES WRAPPED IN React.memo
 *    - ERDTableNode, TableNode: React.memo wrapped
 *    - SchemaSummaryNode: React.memo wrapped
 *    - ERDEdge: React.memo with custom comparator
 *    - CustomEdge: React.memo with custom comparator
 *
 * 3. CALLBACKS WRAPPED IN useCallback
 *    - handleNodeClick, handleEdgeClick, handlePaneClick
 *    - handleViewportChange (throttled to prevent pan re-renders)
 *    - handleEdgeHover, applyLayout, export functions
 *
 * 4. COMPUTED VALUES MEMOIZED WITH useMemo
 *    - displayNodes, filteredEdges
 *    - adjacencyMap, neighborWhitelist
 *    - performanceLevel, computedDetailLevel
 *
 * 5. VIEWPORT OPTIMIZATION
 *    - onlyRenderVisibleElements={true} enabled
 *    - Zoom state updates throttled (5% delta threshold)
 *    - Uses ref for lastZoom to avoid state updates during pan
 *
 * 6. SELECTION STATE DECOUPLED
 *    - selectedTableId tracked separately from nodes array
 *    - hoveredEdgeId debounced (50ms) to reduce edge highlight re-renders
 *
 * 7. PERFORMANCE PROPS ENABLED
 *    - elevateEdgesOnSelect={false}
 *    - elevateNodesOnSelect={false}
 *    - selectNodesOnDrag={false}
 *    - nodesDraggable={false} by default
 *
 * 8. STATIC CONSTANTS OUTSIDE COMPONENT
 *    - DEFAULT_VIEWPORT, MIN_ZOOM, MAX_ZOOM, NODE_EXTENT
 *    - Prevents object recreation on every render
 */

import 'reactflow/dist/style.css'

import {
  Database,
  Download,
  FolderMinus,
  FolderPlus,
  Layers,
  Maximize2,
  Minimize2,
  RotateCcw,
  Search,
  Settings,
  X,
} from 'lucide-react'
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Background,
  ConnectionLineType,
  Controls,
  Edge,
  MiniMap,
  Node,
  OnMove,
  ReactFlow,
  ReactFlowProvider,
  useEdgesState,
  useNodesState,
  useReactFlow,
  Viewport,
} from 'reactflow'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { useDebounce } from '@/hooks/use-debounce'
import { SchemaNode } from '@/hooks/use-schema-introspection'
import { SchemaConfigBuilder } from '@/lib/schema-config'
import { LayoutEngine } from '@/lib/schema-layout'
import {
  EdgeConfig,
  LayoutAlgorithm,
  LayoutOptions,
  SchemaConfig,
  SchemaVisualizerEdge,
  SchemaVisualizerNode,
  TableConfig,
} from '@/types/schema-visualizer'

import { CustomEdge } from './custom-edge'
import { ERDEdge } from './erd-edge'
import { ERDTableNode } from './erd-table-node'
import { RelationshipInspector } from './relationship-inspector'
import { SchemaErrorBoundary } from './schema-error-boundary'
import { SchemaSummaryNode } from './schema-summary-node'
import { TableNode } from './table-node'

interface SchemaVisualizerProps {
  schema: SchemaNode[]
  onClose: () => void
  connectionId?: string
}

// Performance constants - defined outside component to prevent recreation
const DEFAULT_VIEWPORT: Viewport = { x: 0, y: 0, zoom: 0.5 }
const MIN_ZOOM = 0.1
const MAX_ZOOM = 2
const NODE_EXTENT: [[number, number], [number, number]] = [
  [-10000, -10000],
  [10000, 10000],
]

// Define node types OUTSIDE component to prevent recreation on every render
// These are static and should never change during component lifecycle
const ERD_NODE_TYPES = {
  table: ERDTableNode,
  schemaSummary: SchemaSummaryNode,
}

const CLASSIC_NODE_TYPES = {
  table: TableNode,
  schemaSummary: SchemaSummaryNode,
}

// Define edge types OUTSIDE component
const ERD_EDGE_TYPES = {
  smoothstep: ERDEdge,
  erd: ERDEdge,
}

const CLASSIC_EDGE_TYPES = {
  smoothstep: CustomEdge,
  erd: ERDEdge,
}

export function SchemaVisualizer({ schema, onClose }: SchemaVisualizerProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])
  const [schemaConfig, setSchemaConfig] = useState<SchemaConfig | null>(null)
  const [initialFitDone, setInitialFitDone] = useState(false)
  const reactFlowInstance = useReactFlow()

  // ERD mode toggle - 'erd' for new dark design, 'classic' for original
  const [visualizationMode, setVisualizationMode] = useState<'erd' | 'classic'>('erd')

  // Use pre-defined node/edge types based on mode - no useMemo needed since they're static
  const nodeTypes = visualizationMode === 'erd' ? ERD_NODE_TYPES : CLASSIC_NODE_TYPES
  const edgeTypes = visualizationMode === 'erd' ? ERD_EDGE_TYPES : CLASSIC_EDGE_TYPES
  
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
  const [nodesDraggable, setNodesDraggable] = useState(false) // Disabled by default for performance
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

          const { nodes: flowNodes, edges: flowEdges } = SchemaConfigBuilder.toReactFlowNodes(config)

          // Smart layout selection based on table count
          // Performance thresholds based on ReactFlow limitations
          const tableCount = config.tables.length
          let selectedLayout: LayoutAlgorithm = 'hierarchical'

          if (tableCount < 50) {
            // Optimal range: full features with hierarchical layout
            selectedLayout = 'hierarchical'
          } else if (tableCount < 100) {
            // Degraded range: switch to grid, keep animations
            selectedLayout = 'grid'
            console.info(`Medium schema detected: ${tableCount} tables. Using grid layout for better performance.`)
          } else if (tableCount < 200) {
            // Minimal range: grid only, no animations
            selectedLayout = 'grid'
            setSidebarCollapsed(false) // Encourage filtering
            console.warn(`Large schema detected: ${tableCount} tables. Performance may be degraded. Use filters to reduce complexity.`)
          } else {
            // Critical range: warn user strongly
            selectedLayout = 'grid'
            setSidebarCollapsed(false) // Force sidebar open for filtering
            console.error(`Very large schema: ${tableCount} tables. Browser visualization not recommended. Consider using a dedicated database client tool or export to documentation.`)
          }

          setLayoutAlgorithm(selectedLayout)

          // Apply initial layout if we have edges (FK relationships)
          if (flowEdges.length > 0) {
            const layoutOptions: LayoutOptions = {
              algorithm: selectedLayout,
              spacing: { x: 300, y: 200 },
            }

            const { nodes: layoutedNodes } = LayoutEngine.applyLayout(
              flowNodes as SchemaVisualizerNode[],
              flowEdges as SchemaVisualizerEdge[],
              layoutOptions
            )

            setNodes(layoutedNodes as Node[])
            setEdges(flowEdges as Edge[])
          } else {
            // No edges, just set nodes and edges without layout
            setNodes(flowNodes as Node[])
            setEdges(flowEdges as Edge[])
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

  // Perform fitView only once after initial nodes are loaded
  useEffect(() => {
    if (!initialFitDone && nodes.length > 0 && reactFlowInstance) {
      // Use requestAnimationFrame to ensure DOM is ready
      requestAnimationFrame(() => {
        reactFlowInstance.fitView({ padding: 0.1, maxZoom: 1 })
        setInitialFitDone(true)
      })
    }
  }, [nodes.length, initialFitDone, reactFlowInstance])

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
        const isRelated = selectedTableId !== null &&
          selectedTableId !== node.id &&
          neighborWhitelist?.has(node.id)
        const isDimmed =
          selectedTableId !== null &&
          selectedTableId !== node.id &&
          (focusNeighborsOnly ? !neighborWhitelist?.has(node.id) : false)

        return {
          ...node,
          data: {
            ...node.data,
            isFocused,
            isRelated,
            isDimmed,
          },
        }
      })
  }, [nodes, debouncedSearchTerm, selectedSchemas, selectedTableId, neighborWhitelist, focusNeighborsOnly])

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

      // Determine handles - use column-specific handles when available,
      // fall back to table-level handles for collapsed/compact nodes
      let sourceHandle = edge.sourceHandle
      let targetHandle = edge.targetHandle

      // For collapsed schemas, use table-level handles
      if (collapsedNodeMap.has(edge.source)) {
        sourceHandle = 'table-source'
      }
      if (collapsedNodeMap.has(edge.target)) {
        targetHandle = 'table-target'
      }

      const baseEdge: Edge = {
        ...edge,
        id: aggregateKey,
        source: mappedSource,
        target: mappedTarget,
        sourceHandle,
        targetHandle,
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

  // Memoized close handler for RelationshipInspector
  const handleInspectorClose = useCallback(() => {
    setSelectedEdge(null)
  }, [])

  // Throttled viewport change handler - only update zoom state when zoom changes significantly
  // This prevents constant re-renders during pan operations
  const lastZoomRef = useRef(1)
  const handleViewportChange = useCallback<OnMove>((_event, viewport) => {
    // Only update state if zoom changed by more than 5% - prevents re-renders during panning
    const zoomDelta = Math.abs(viewport.zoom - lastZoomRef.current)
    if (zoomDelta > 0.05) {
      lastZoomRef.current = viewport.zoom
      setViewportZoom(viewport.zoom)
    }
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
            {/* ERD/Classic Mode Toggle */}
            <Button
              variant={visualizationMode === 'erd' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setVisualizationMode(visualizationMode === 'erd' ? 'classic' : 'erd')}
              title={visualizationMode === 'erd' ? 'Switch to Classic mode' : 'Switch to ERD mode'}
            >
              <Database className="h-4 w-4 mr-1" />
              {visualizationMode === 'erd' ? 'ERD' : 'Classic'}
            </Button>
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
                  <div className="flex items-center justify-between">
                    <Label htmlFor="draggable-nodes" className="text-sm">Draggable nodes</Label>
                    <Switch
                      id="draggable-nodes"
                      checked={nodesDraggable}
                      onCheckedChange={setNodesDraggable}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Enable dragging to rearrange tables manually. Disabled by default for better performance.
                  </p>
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
          <div className="flex-1 bg-background">
            <ReactFlow
              nodes={displayNodes}
              edges={filteredEdges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onNodeClick={handleNodeClick}
              onEdgeClick={handleEdgeClick}
              onPaneClick={handlePaneClick}
              onMove={handleViewportChange}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              // Performance optimizations - critical for smooth scrolling
              onlyRenderVisibleElements={true}
              nodesDraggable={nodesDraggable}
              nodesConnectable={false}
              elementsSelectable={false}
              minZoom={MIN_ZOOM}
              maxZoom={MAX_ZOOM}
              defaultViewport={DEFAULT_VIEWPORT}
              nodeExtent={NODE_EXTENT}
              panOnScroll={true}
              panOnDrag={true}
              zoomOnScroll={true}
              zoomOnPinch={true}
              zoomOnDoubleClick={false}
              selectNodesOnDrag={false}
              // Additional performance props
              elevateEdgesOnSelect={false}
              elevateNodesOnSelect={false}
              connectionLineType={ConnectionLineType.SmoothStep}
              deleteKeyCode={null}
              multiSelectionKeyCode={null}
              selectionKeyCode={null}
              // fitView removed - handled by useEffect for single execution
              attributionPosition="bottom-left"
              className="[&_.react-flow__pane]:bg-background"
            >
              <Background
                className="!stroke-border"
                gap={visualizationMode === 'erd' ? 20 : 16}
              />
              <Controls
                className="[&>button]:bg-card [&>button]:border-border [&>button]:text-foreground [&>button:hover]:bg-accent"
              />
              <MiniMap
                nodeColor="hsl(var(--muted))"
                maskColor="hsl(var(--background) / 0.7)"
                className="bg-card border-border"
              />
            </ReactFlow>

            {/* Relationship Inspector */}
            {selectedEdge && (
              <RelationshipInspector
                edge={selectedEdge.edge}
                sourceTable={selectedEdge.sourceTable}
                targetTable={selectedEdge.targetTable}
                position={selectedEdge.position}
                onClose={handleInspectorClose}
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

  // Update loaded schema when props.schema changes
  // If schema is already provided with columns, use it directly
  useEffect(() => {
    // Check if schema has columns (not just empty children arrays)
    const hasColumns = props.schema?.some(schemaNode =>
      schemaNode.children?.some(tableNode =>
        tableNode.children && tableNode.children.length > 0
      )
    )

    if (props.schema && props.schema.length > 0 && hasColumns) {
      console.log('[SchemaVisualizer] Using provided schema with columns')
      setLoadedSchema(props.schema)
      setLoading(false)
      return
    }

    // Load schema with columns from schema store if we have a connectionId
    if (!props.connectionId) {
      // If no connectionId and schema has no columns, just use what we have
      if (props.schema && props.schema.length > 0) {
        setLoadedSchema(props.schema)
        setLoading(false)
      }
      return
    }

    const loadConnectionSchema = async () => {
      setLoading(true)
      setError(null)

      try {
        // Get the connection from the store
        const { useConnectionStore } = await import('@/store/connection-store')
        const connections = useConnectionStore.getState().connections
        const connection = connections.find(conn => conn.id === props.connectionId)

        if (!connection?.sessionId) {
          throw new Error('Connection not found or not connected')
        }

        // Use the schema store to get fully populated schema with columns and FKs
        const { useSchemaStore } = await import('@/store/schema-store')
        const schemaData = await useSchemaStore.getState().getSchema(
          connection.sessionId,
          connection.name,
          false // don't force - use cache if available
        )

        if (!schemaData || schemaData.length === 0) {
          throw new Error('No schema data available')
        }

        console.log('[SchemaVisualizer] Loaded schema from store:', {
          schemaCount: schemaData.length,
          tableCount: schemaData.reduce((acc, s) => acc + (s.children?.length || 0), 0),
          firstSchemaColumns: schemaData[0]?.children?.[0]?.children?.length || 0
        })

        setLoadedSchema(schemaData)
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
