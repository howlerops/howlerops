export interface SchemaConfig {
  tables: TableConfig[]
  edges: EdgeConfig[]
  tablePositions: Record<string, { x: number; y: number }>
  schemaColors: Record<string, string>
}

export interface TableConfig {
  id: string
  name: string
  schema: string
  description?: string
  columns: ColumnConfig[]
  position?: { x: number; y: number }
  schemaColor?: string
  isHighlighted?: boolean
  isSelected?: boolean
  isFocused?: boolean
  isDimmed?: boolean
}

export interface ColumnConfig {
  id: string
  name: string
  type: string
  description?: string
  isPrimaryKey?: boolean
  isForeignKey?: boolean
  isNullable?: boolean
  defaultValue?: string
}

export interface EdgeConfig {
  id: string
  source: string
  sourceKey: string
  target: string
  targetKey: string
  relation: 'hasOne' | 'hasMany' | 'belongsTo' | 'manyToMany'
  label?: string
}

export interface SchemaVisualizerNode {
  id: string
  type: 'table'
  position: { x: number; y: number }
  data: TableConfig
}

export interface SchemaVisualizerEdge {
  id: string
  source: string
  target: string
  sourceHandle: string
  targetHandle: string
  type?: string
  label?: string
  data?: EdgeConfig & {
    onEdgeHover?: (edgeId: string | null) => void
    isHighlighted?: boolean
    isDimmed?: boolean
  }
}

export type LayoutAlgorithm = 'force' | 'hierarchical' | 'grid' | 'circular'

export interface LayoutOptions {
  algorithm: LayoutAlgorithm
  direction?: 'TB' | 'BT' | 'LR' | 'RL'
  spacing?: { x: number; y: number }
  maxDepth?: number
}

export interface FilterOptions {
  searchTerm: string
  selectedSchemas: string[]
  showForeignKeys: boolean
  showPrimaryKeys: boolean
}
