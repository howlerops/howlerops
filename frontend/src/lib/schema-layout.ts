// import { Node, Edge, Position } from 'reactflow'
import { SchemaVisualizerNode, SchemaVisualizerEdge, LayoutOptions } from '@/types/schema-visualizer'

export class LayoutEngine {
  static applyLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: LayoutOptions
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    switch (options.algorithm) {
      case 'force':
        return this.forceDirectedLayout(nodes, edges, options)
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

  private static forceDirectedLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: LayoutOptions
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    const spacing = options.spacing || { x: 300, y: 200 }
    const iterations = 100
    const k = Math.sqrt((nodes.length * spacing.x * spacing.y) / nodes.length)
    
    // Initialize positions randomly
    const positionedNodes = nodes.map(node => ({
      ...node,
      position: node.position || {
        x: Math.random() * 800,
        y: Math.random() * 600
      }
    }))

    // Simple force-directed algorithm
    for (let i = 0; i < iterations; i++) {
      const forces = new Map<string, { x: number; y: number }>()
      
      // Initialize forces
      positionedNodes.forEach(node => {
        forces.set(node.id, { x: 0, y: 0 })
      })

      // Repulsive forces between all nodes
      for (let i = 0; i < positionedNodes.length; i++) {
        for (let j = i + 1; j < positionedNodes.length; j++) {
          const node1 = positionedNodes[i]
          const node2 = positionedNodes[j]
          const dx = node1.position.x - node2.position.x
          const dy = node1.position.y - node2.position.y
          const distance = Math.sqrt(dx * dx + dy * dy) || 1
          
          const force = (k * k) / distance
          const fx = (dx / distance) * force
          const fy = (dy / distance) * force
          
          forces.get(node1.id)!.x += fx
          forces.get(node1.id)!.y += fy
          forces.get(node2.id)!.x -= fx
          forces.get(node2.id)!.y -= fy
        }
      }

      // Attractive forces for connected nodes
      edges.forEach(edge => {
        const sourceNode = positionedNodes.find(n => n.id === edge.source)
        const targetNode = positionedNodes.find(n => n.id === edge.target)
        
        if (sourceNode && targetNode) {
          const dx = targetNode.position.x - sourceNode.position.x
          const dy = targetNode.position.y - sourceNode.position.y
          const distance = Math.sqrt(dx * dx + dy * dy) || 1
          
          const force = (distance * distance) / k
          const fx = (dx / distance) * force
          const fy = (dy / distance) * force
          
          forces.get(sourceNode.id)!.x += fx
          forces.get(sourceNode.id)!.y += fy
          forces.get(targetNode.id)!.x -= fx
          forces.get(targetNode.id)!.y -= fy
        }
      })

      // Apply forces
      positionedNodes.forEach(node => {
        const force = forces.get(node.id)!
        const damping = 0.1
        node.position.x += force.x * damping
        node.position.y += force.y * damping
      })
    }

    return { nodes: positionedNodes, edges }
  }

  private static hierarchicalLayout(
    nodes: SchemaVisualizerNode[],
    edges: SchemaVisualizerEdge[],
    options: LayoutOptions
  ): { nodes: SchemaVisualizerNode[]; edges: SchemaVisualizerEdge[] } {
    const spacing = options.spacing || { x: 300, y: 200 }
    const direction = options.direction || 'TB'
    
    // Build adjacency list
    const adjacencyList = new Map<string, string[]>()
    const inDegree = new Map<string, number>()
    
    nodes.forEach(node => {
      adjacencyList.set(node.id, [])
      inDegree.set(node.id, 0)
    })
    
    edges.forEach(edge => {
      adjacencyList.get(edge.source)?.push(edge.target)
      inDegree.set(edge.target, (inDegree.get(edge.target) || 0) + 1)
    })

    // Topological sort
    const queue: string[] = []
    const levels: string[][] = []
    
    inDegree.forEach((degree, nodeId) => {
      if (degree === 0) queue.push(nodeId)
    })
    
    while (queue.length > 0) {
      const levelSize = queue.length
      const currentLevel: string[] = []
      
      for (let i = 0; i < levelSize; i++) {
        const nodeId = queue.shift()!
        currentLevel.push(nodeId)
        
        adjacencyList.get(nodeId)?.forEach(neighbor => {
          inDegree.set(neighbor, (inDegree.get(neighbor) || 0) - 1)
          if (inDegree.get(neighbor) === 0) {
            queue.push(neighbor)
          }
        })
      }
      
      levels.push(currentLevel)
    }

    // Position nodes
    const positionedNodes = nodes.map(node => ({ ...node }))
    
    levels.forEach((level, levelIndex) => {
      level.forEach((nodeId, nodeIndex) => {
        const node = positionedNodes.find(n => n.id === nodeId)
        if (node) {
          if (direction === 'TB' || direction === 'BT') {
            node.position = {
              x: nodeIndex * spacing.x,
              y: levelIndex * spacing.y
            }
          } else {
            node.position = {
              x: levelIndex * spacing.x,
              y: nodeIndex * spacing.y
            }
          }
        }
      })
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
    _options: LayoutOptions // eslint-disable-line @typescript-eslint/no-unused-vars
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
