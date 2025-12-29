// import { Node, Edge, Position } from 'reactflow'
import dagre from '@dagrejs/dagre'

import {
  LayoutOptions,
  SchemaSummaryNodeData,
  SchemaVisualizerEdge,
  SchemaVisualizerNode,
  TableConfig,
} from '@/types/schema-visualizer'

const hasColumns = (data: TableConfig | SchemaSummaryNodeData): data is TableConfig => {
  return Array.isArray((data as TableConfig).columns)
}

export class LayoutEngine {
  static applyLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: LayoutOptions
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    switch (options.algorithm) {
      case 'force':
        // Force-directed layout removed for performance - use hierarchical instead
        return this.hierarchicalLayout(nodes, edges, options)
      case 'hierarchical':
        return this.hierarchicalLayout(nodes, edges, options)
      case 'grid':
        return this.gridLayout(nodes, edges, options)
      case 'circular':
        return this.circularLayout(nodes, edges, options)
      default:
        return { nodes, edges }
    }
  }

  // Force-directed layout removed for performance reasons
  // The O(n²) complexity with 100 iterations was causing performance issues
  // Use hierarchical, grid, or circular layouts instead

  private static hierarchicalLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: LayoutOptions
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    // Handle empty nodes case
    if (!nodes || nodes.length === 0) {
      return { nodes: [], edges: [] }
    }

    const spacing = options.spacing || { x: 300, y: 200 }
    const direction = options.direction || 'TB'

    // Create a new Dagre graph
    const dagreGraph = new dagre.graphlib.Graph()
    dagreGraph.setDefaultEdgeLabel(() => ({}))

    // Configure the layout with optimized settings for FK relationships
    dagreGraph.setGraph({
      rankdir: direction, // TB, BT, LR, RL
      align: 'UL', // Align nodes to upper-left within ranks
      nodesep: spacing.x / 2, // Horizontal spacing between nodes at same level
      edgesep: 25, // Spacing between edges
      ranksep: spacing.y, // Vertical spacing between hierarchical levels
      marginx: 50,
      marginy: 50,
      ranker: 'network-simplex', // Best algorithm for hierarchical layouts
    })

    // Add nodes to the graph with size information
    nodes.forEach(node => {
      const nodeData = node.data as TableConfig | SchemaSummaryNodeData
      const isTableNode = hasColumns(nodeData)
      const columnCount = isTableNode ? nodeData.columns.length : 1
      const nodeHeight = isTableNode ? 40 + (columnCount * 24) + 30 : 140
      const nodeWidth = isTableNode ? 250 : 220

      dagreGraph.setNode(node.id, {
        width: nodeWidth,
        height: nodeHeight,
        node: node
      })
    })

    // Add edges to the graph
    // For FK relationships, the edge direction determines hierarchy:
    // - Source table (with FK column) points to target table (referenced)
    // - Target tables should be at higher level (top) in hierarchy
    edges.forEach(edge => {
      // Reverse the edge direction so that referenced tables (targets) appear at top
      // and tables with FKs (sources) appear below their dependencies
      dagreGraph.setEdge(edge.target, edge.source, {
        weight: 1, // All FK relationships have equal weight
      })
    })

    // Calculate the layout
    dagre.layout(dagreGraph)

    // Update node positions based on Dagre layout
    const positionedNodes = nodes.map(node => {
      const dagreNode = dagreGraph.node(node.id)
      if (!dagreNode) return node

      // Dagre centers nodes, we need to adjust to top-left positioning for ReactFlow
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

  private static gridLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: LayoutOptions
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    const spacing = options.spacing || { x: 300, y: 200 }
    const cols = Math.ceil(Math.sqrt(nodes.length))
    
    const positionedNodes = nodes.map((node, index) => ({
      ...node,
      position: {
        x: (index % cols) * spacing.x,
        y: Math.floor(index / cols) * spacing.y
      }
    }))

    return { nodes: positionedNodes, edges }
  }

  private static circularLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    _options: LayoutOptions  
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    const radius = Math.max(200, nodes.length * 20)
    const centerX = 400
    const centerY = 300
    
    const positionedNodes = nodes.map((node, index) => {
      const angle = (2 * Math.PI * index) / nodes.length
      return {
        ...node,
        position: {
          x: centerX + radius * Math.cos(angle),
          y: centerY + radius * Math.sin(angle)
        }
      }
    })

    return { nodes: positionedNodes, edges }
  }
}
