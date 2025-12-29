import dagre from '@dagrejs/dagre'

import {
  ERDLayoutDirection,
  SchemaVisualizerEdge,
  SchemaVisualizerNode,
  TableConfig,
} from '@/types/schema-visualizer'

interface ERDLayoutOptions {
  direction: ERDLayoutDirection
  nodeSpacing: number
  rankSpacing: number
  centerFocusedTable?: string
}

const DEFAULT_OPTIONS: ERDLayoutOptions = {
  direction: 'LR',
  nodeSpacing: 60,
  rankSpacing: 200,
}

/**
 * ERD-specific layout engine
 * Positions tables with foreign keys on the left, referenced tables on the right
 * Optimized for showing relationship hierarchies clearly
 */
export class ERDLayoutEngine {
  /**
   * Apply ERD-optimized hierarchical layout
   * Tables with more outgoing FKs are placed on the left
   * Tables that are frequently referenced are placed on the right
   */
  static applyLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: Partial<ERDLayoutOptions> = {}
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    if (!nodes.length) {
      return { nodes: [], edges: [] }
    }

    const opts = { ...DEFAULT_OPTIONS, ...options }
    const dagreGraph = new dagre.graphlib.Graph()

    dagreGraph.setDefaultEdgeLabel(() => ({}))
    dagreGraph.setGraph({
      rankdir: opts.direction,
      align: 'UL',
      nodesep: opts.nodeSpacing,
      edgesep: 30,
      ranksep: opts.rankSpacing,
      marginx: 40,
      marginy: 40,
    })

    // Calculate node dimensions based on column count
    nodes.forEach(node => {
      const data = node.data as TableConfig
      const columnCount = data.columns?.length || 1
      const nodeHeight = 44 + (columnCount * 28) // header + rows
      const nodeWidth = 240

      dagreGraph.setNode(node.id, {
        width: nodeWidth,
        height: nodeHeight,
      })
    })

    // Add edges - direction matters for hierarchy
    edges.forEach(edge => {
      dagreGraph.setEdge(edge.source, edge.target)
    })

    // Calculate layout
    dagre.layout(dagreGraph)

    // Apply calculated positions
    const positionedNodes = nodes.map(node => {
      const dagreNode = dagreGraph.node(node.id)
      if (!dagreNode) return node

      return {
        ...node,
        position: {
          x: dagreNode.x - dagreNode.width / 2,
          y: dagreNode.y - dagreNode.height / 2,
        },
      }
    })

    return { nodes: positionedNodes, edges }
  }

  /**
   * Apply focused layout - centers a specific table with related tables around it
   */
  static applyFocusedLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    focusedTableId: string,
    options: Partial<ERDLayoutOptions> = {}
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    if (!nodes.length) {
      return { nodes: [], edges: [] }
    }

    // Find related tables
    const relatedTableIds = new Set<string>()
    edges.forEach(edge => {
      if (edge.source === focusedTableId) {
        relatedTableIds.add(edge.target)
      }
      if (edge.target === focusedTableId) {
        relatedTableIds.add(edge.source)
      }
    })

    // Filter to only focused table and related tables
    const relevantNodes = nodes.filter(
      node => node.id === focusedTableId || relatedTableIds.has(node.id)
    )
    const relevantEdges = edges.filter(
      edge =>
        (edge.source === focusedTableId || edge.target === focusedTableId) &&
        relatedTableIds.has(edge.source === focusedTableId ? edge.target : edge.source)
    )

    const opts = { ...DEFAULT_OPTIONS, ...options }
    const dagreGraph = new dagre.graphlib.Graph()

    dagreGraph.setDefaultEdgeLabel(() => ({}))
    dagreGraph.setGraph({
      rankdir: opts.direction,
      align: 'UL',
      nodesep: opts.nodeSpacing,
      edgesep: 30,
      ranksep: opts.rankSpacing,
      marginx: 40,
      marginy: 40,
    })

    // Add nodes
    relevantNodes.forEach(node => {
      const data = node.data as TableConfig
      const columnCount = data.columns?.length || 1
      const nodeHeight = 44 + (columnCount * 28)
      const nodeWidth = 240

      dagreGraph.setNode(node.id, {
        width: nodeWidth,
        height: nodeHeight,
      })
    })

    // Add edges
    relevantEdges.forEach(edge => {
      dagreGraph.setEdge(edge.source, edge.target)
    })

    // Calculate layout
    dagre.layout(dagreGraph)

    // Apply positions with focus table marking
    const positionedNodes = relevantNodes.map(node => {
      const dagreNode = dagreGraph.node(node.id)
      if (!dagreNode) return node

      const isFocused = node.id === focusedTableId
      const isRelated = relatedTableIds.has(node.id)

      return {
        ...node,
        position: {
          x: dagreNode.x - dagreNode.width / 2,
          y: dagreNode.y - dagreNode.height / 2,
        },
        data: {
          ...node.data,
          isFocused,
          isRelated: !isFocused && isRelated,
        },
      }
    })

    return { nodes: positionedNodes, edges: relevantEdges }
  }

  /**
   * Calculate optimal layout direction based on relationships
   * Returns 'LR' if most relationships flow left-to-right, 'TB' otherwise
   */
  static calculateOptimalDirection(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[]
  ): ERDLayoutDirection {
    // Count incoming edges per node
    const incomingCount = new Map<string, number>()
    const outgoingCount = new Map<string, number>()

    nodes.forEach(node => {
      incomingCount.set(node.id, 0)
      outgoingCount.set(node.id, 0)
    })

    edges.forEach(edge => {
      incomingCount.set(edge.target, (incomingCount.get(edge.target) || 0) + 1)
      outgoingCount.set(edge.source, (outgoingCount.get(edge.source) || 0) + 1)
    })

    // If there's a clear hierarchy (some tables have many incoming but few outgoing),
    // use LR layout. Otherwise use TB for a more compact view.
    let hierarchyScore = 0
    nodes.forEach(node => {
      const incoming = incomingCount.get(node.id) || 0
      const outgoing = outgoingCount.get(node.id) || 0
      hierarchyScore += Math.abs(incoming - outgoing)
    })

    // Normalize by node count
    const normalizedScore = nodes.length > 0 ? hierarchyScore / nodes.length : 0

    return normalizedScore > 1.5 ? 'LR' : 'TB'
  }

  /**
   * Group tables by their relationship level (distance from root tables)
   * Root tables = tables with no outgoing FKs (they are only referenced)
   */
  static groupByRelationshipLevel(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[]
  ): Map<number, string[]> {
    const levels = new Map<number, string[]>()
    const nodeLevel = new Map<string, number>()

    // Find root tables (tables with no outgoing edges - only referenced)
    const hasOutgoing = new Set<string>()
    edges.forEach(edge => hasOutgoing.add(edge.source))

    const rootTables = nodes
      .filter(node => !hasOutgoing.has(node.id))
      .map(node => node.id)

    // BFS from root tables to assign levels
    const visited = new Set<string>()
    const queue: Array<{ id: string; level: number }> = rootTables.map(id => ({
      id,
      level: 0,
    }))

    while (queue.length > 0) {
      const { id, level } = queue.shift()!

      if (visited.has(id)) continue
      visited.add(id)

      nodeLevel.set(id, level)

      if (!levels.has(level)) {
        levels.set(level, [])
      }
      levels.get(level)!.push(id)

      // Find tables that reference this table
      edges.forEach(edge => {
        if (edge.target === id && !visited.has(edge.source)) {
          queue.push({ id: edge.source, level: level + 1 })
        }
      })
    }

    // Handle orphan nodes (not connected to root tables)
    nodes.forEach(node => {
      if (!visited.has(node.id)) {
        const maxLevel = Math.max(...Array.from(levels.keys()), 0)
        const level = maxLevel + 1

        if (!levels.has(level)) {
          levels.set(level, [])
        }
        levels.get(level)!.push(node.id)
      }
    })

    return levels
  }
}
