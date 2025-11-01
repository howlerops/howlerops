// import { Node, Edge, Position } from 'reactflow'
import dagre from '@dagrejs/dagre'
import { SchemaVisualizerNode, SchemaVisualizerEdge, LayoutOptions } from '@/types/schema-visualizer'

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

    // Configure the layout
    dagreGraph.setGraph({
      rankdir: direction, // TB, BT, LR, RL
      align: 'UL',
      nodesep: spacing.x / 2,
      edgesep: 10,
      ranksep: spacing.y,
      marginx: 0,
      marginy: 0,
    })

    // Add nodes to the graph with size information
    nodes.forEach(node => {
      const tableData = node.data
      // Calculate node dimensions based on content
      const columnCount = tableData.columns?.length || 1
      const nodeHeight = 40 + (columnCount * 24) + 30 // header + columns + footer
      const nodeWidth = 250 // fixed width for consistency

      dagreGraph.setNode(node.id, {
        width: nodeWidth,
        height: nodeHeight,
        node: node
      })
    })

    // Add edges to the graph
    edges.forEach(edge => {
      dagreGraph.setEdge(edge.source, edge.target)
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
