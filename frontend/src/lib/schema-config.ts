import { MarkerType } from 'reactflow'

import { SchemaNode } from '@/hooks/use-schema-introspection'
import { ColumnConfig, EdgeConfig,SchemaConfig, TableConfig } from '@/types/schema-visualizer'

interface ForeignKeyInfo {
  name: string
  columnName: string
  referencedTableName: string
  referencedSchemaName: string
  referencedColumnName: string
  deleteRule: string
  updateRule: string
}

export class SchemaConfigBuilder {
  static async fromSchemaNodes(schemaNodes: SchemaNode[]): Promise<SchemaConfig> {
    const tables: TableConfig[] = []
    const edges: EdgeConfig[] = []
    const tablePositions: Record<string, { x: number; y: number }> = {}
    const schemaColors: Record<string, string> = {
      DEFAULT: '#6366f1',
      public: '#8b5cf6',
      information_schema: '#64748b',
      pg_catalog: '#64748b',
    }

    // Extract tables and columns
    schemaNodes.forEach((schemaNode) => {
      if (schemaNode.children) {
        schemaNode.children.forEach((tableNode) => {
          if (tableNode.type === 'table' && tableNode.children) {
            const columns: ColumnConfig[] = tableNode.children
              .filter((col) => col.type === 'column')
              .map((col) => {
                const columnMetadata = (col.metadata || {}) as Record<string, any>
                const columnName = (columnMetadata.name || col.name.split(':')[0]).trim()
                const columnType = columnMetadata.dataType || columnMetadata.data_type || 'unknown'
                const isPrimaryKey = columnMetadata.isPrimaryKey ?? columnMetadata.primaryKey ?? col.name.includes('PK')
                let isForeignKey = col.name.includes('FK')
                if (typeof columnMetadata.isForeignKey === 'boolean') {
                  isForeignKey = columnMetadata.isForeignKey
                } else if (columnMetadata.foreignKey !== undefined) {
                  isForeignKey = true
                }
                const isNullable = columnMetadata.isNullable ?? columnMetadata.nullable
                const defaultValue = columnMetadata.columnDefault ?? columnMetadata.defaultValue

                return {
                  id: col.id,
                  name: columnName,
                  type: columnType,
                  description: columnMetadata.comment,
                  isPrimaryKey,
                  isForeignKey,
                  isNullable,
                  defaultValue,
                }
              })

            const tableConfig: TableConfig = {
              id: tableNode.id,
              name: tableNode.name,
              schema: schemaNode.name,
              description: (tableNode.metadata as any)?.description,
              columns,
            }

            tables.push(tableConfig)
          }
        })
      }
    })

    // Extract foreign key relationships from API responses
    await this.extractForeignKeyRelationships(schemaNodes, tables, edges)

    // Add mock foreign key relationships for testing if no real ones exist
    if (edges.length === 0 && tables.length > 1) {
      this.addMockForeignKeyRelationships(tables, edges)
    }

    // Generate default positions using grid layout
    const cols = Math.ceil(Math.sqrt(tables.length))
    tables.forEach((table, index) => {
      tablePositions[table.id] = {
        x: (index % cols) * 300,
        y: Math.floor(index / cols) * 200,
      }
    })

    return {
      tables,
      edges,
      tablePositions,
      schemaColors,
    }
  }

  static toReactFlowNodes(config: SchemaConfig): {
    nodes: unknown[]
    edges: unknown[]
  } {
    const nodes = config.tables.map((table) => ({
      id: table.id,
      type: 'table',
      position: config.tablePositions[table.id] || { x: 0, y: 0 },
      data: {
        ...table,
        schemaColor: config.schemaColors[table.schema] || config.schemaColors.DEFAULT,
      },
    }))

    const edges = config.edges.map((edge) => {
      let edgeStyle: {
        stroke: string;
        strokeWidth: number;
        strokeDasharray?: string;
      } = {
        stroke: '#64748b',
        strokeWidth: 2,
      }
      let markerEnd = {
        type: MarkerType.ArrowClosed,
        color: '#64748b',
      }

      if (edge.relation === 'hasMany') {
        edgeStyle = {
          stroke: '#f59e0b',
          strokeWidth: 2,
        }
        markerEnd = {
          type: MarkerType.ArrowClosed,
          color: '#f59e0b',
        }
      } else if (edge.relation === 'hasOne') {
        edgeStyle = {
          stroke: '#3b82f6',
          strokeWidth: 2,
        }
        markerEnd = {
          type: MarkerType.ArrowClosed,
          color: '#3b82f6',
        }
      } else if (edge.relation === 'belongsTo') {
        edgeStyle = {
          stroke: '#8b5cf6',
          strokeWidth: 2,
          strokeDasharray: '5,5',
        }
        markerEnd = {
          type: MarkerType.ArrowClosed,
          color: '#8b5cf6',
        }
      }

      // Find the actual column IDs from the table configs
      const sourceTable = config.tables.find(t => t.id === edge.source)
      const targetTable = config.tables.find(t => t.id === edge.target)

      let sourceHandle = undefined
      let targetHandle = undefined

      // Try to find column by ID first, then by name
      if (sourceTable) {
        const sourceColumn = sourceTable.columns.find(c =>
          c.id === edge.sourceKey || c.name === edge.sourceKey
        )
        if (sourceColumn) {
          sourceHandle = `${sourceColumn.id}-source`
        }
      }

      if (targetTable) {
        const targetColumn = targetTable.columns.find(c =>
          c.id === edge.targetKey || c.name === edge.targetKey
        )
        if (targetColumn) {
          targetHandle = `${targetColumn.id}-target`
        }
      }

      // Skip this edge if we can't find valid handles
      if (!sourceHandle || !targetHandle) {
        console.warn(`Skipping edge ${edge.id}: Cannot find handle for source "${edge.sourceKey}" or target "${edge.targetKey}"`)
        return null
      }

      return {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        sourceHandle,
        targetHandle,
        type: 'smoothstep',
        animated: edge.relation === 'hasMany',
        style: edgeStyle,
        markerEnd,
        label: edge.label,
        data: edge,
      }
    }).filter(edge => edge !== null) // Remove any edges that couldn't be mapped

    return { nodes, edges }
  }

  static exportToJSON(config: SchemaConfig): string {
    return JSON.stringify(config, null, 2)
  }

  static importFromJSON(jsonString: string): SchemaConfig {
    try {
      const parsed = JSON.parse(jsonString)
      return {
        tables: parsed.tables || [],
        edges: parsed.edges || [],
        tablePositions: parsed.tablePositions || {},
        schemaColors: parsed.schemaColors || { DEFAULT: '#6366f1' },
      }
    } catch {  
      throw new Error('Invalid JSON configuration')
    }
  }

  static generateCSVExport(config: SchemaConfig): string {
    const headers = ['table_schema', 'table_name', 'column_name', 'data_type', 'ordinal_position']
    const rows = config.tables.flatMap((table) =>
      table.columns.map((column, index) => [
        table.schema,
        table.name,
        column.name,
        column.type,
        index + 1,
      ])
    )

    return [headers, ...rows].map((row) => row.join(',')).join('\n')
  }

  private static async extractForeignKeyRelationships(
    schemaNodes: SchemaNode[],
    tables: TableConfig[],
    edges: EdgeConfig[]
  ): Promise<void> {
    // Create a map of table IDs to table configs for quick lookup
    const tableMap = new Map<string, TableConfig>()
    tables.forEach(table => {
      tableMap.set(table.id, table)
    })

    // Process each schema and table to extract foreign key relationships
    for (const schemaNode of schemaNodes) {
      if (!schemaNode.children) continue

      for (const tableNode of schemaNode.children) {
        if (tableNode.type !== 'table' || !tableNode.children) continue

        // Look for foreign key information in table metadata
        const tableMetadata = tableNode.metadata as { foreignKeys?: ForeignKeyInfo[] }
        if (tableMetadata?.foreignKeys) {
          for (const fk of tableMetadata.foreignKeys) {
            // Find the source table
            const sourceTable = tableMap.get(tableNode.id)
            if (!sourceTable) continue

            // Find the target table by name and schema
            const targetTable = tables.find(t => 
              t.name === fk.referencedTableName && t.schema === fk.referencedSchemaName
            )
            if (!targetTable) continue

            // Determine relationship type based on cardinality
            const sourceColumn = sourceTable.columns.find(c => c.name === fk.columnName)
            const targetColumn = targetTable.columns.find(c => c.name === fk.referencedColumnName)
            
            let relationType: 'hasOne' | 'hasMany' | 'belongsTo' = 'belongsTo'
            if (sourceColumn?.isPrimaryKey && targetColumn?.isPrimaryKey) {
              relationType = 'hasOne'
            } else if (targetColumn?.isPrimaryKey) {
              relationType = 'hasMany'
            }

            // Create edge configuration
            const edgeConfig: EdgeConfig = {
              id: `${tableNode.id}_${fk.name}`,
              source: tableNode.id,
              sourceKey: fk.columnName,
              target: targetTable.id,
              targetKey: fk.referencedColumnName,
              relation: relationType,
              label: fk.name,
            }

            edges.push(edgeConfig)

            // Mark the source column as foreign key
            if (sourceColumn) {
              sourceColumn.isForeignKey = true
            }
          }
        }

        // Also check individual column metadata for foreign key information
        for (const col of tableNode.children) {
          if (col.type !== 'column') continue

          const columnMetadata = (col.metadata || {}) as Record<string, any> & {
            foreignKey?: {
              name: string
              referencedTable: string
              referencedSchema?: string
              referencedColumns: string[]
            }
          }

          if (columnMetadata.foreignKey) {
            const fk = columnMetadata.foreignKey
            const sourceTable = tableMap.get(tableNode.id)
            if (!sourceTable) continue

            const targetSchema = fk.referencedSchema || schemaNode.name
            const targetTable = tables.find(t => 
              t.name === fk.referencedTable && t.schema === targetSchema
            )
            if (!targetTable) continue

            const sourceColumn = sourceTable.columns.find(c => c.id === col.id || c.name === (columnMetadata.name || col.name.split(':')[0]))
            const targetColumnName = fk.referencedColumns[0] || 'id'
            const targetColumn = targetTable.columns.find(c => c.name === targetColumnName)
            
            let relationType: 'hasOne' | 'hasMany' | 'belongsTo' = 'belongsTo'
            if (sourceColumn?.isPrimaryKey && targetColumn?.isPrimaryKey) {
              relationType = 'hasOne'
            } else if (targetColumn?.isPrimaryKey) {
              relationType = 'hasMany'
            }

            const edgeConfig: EdgeConfig = {
              id: `${tableNode.id}_${col.id}_${fk.name}`,
              source: tableNode.id,
              sourceKey: sourceColumn?.name || columnMetadata.name || col.name.split(':')[0],
              target: targetTable.id,
              targetKey: targetColumnName,
              relation: relationType,
              label: fk.name,
            }

            edges.push(edgeConfig)

            if (sourceColumn) {
              sourceColumn.isForeignKey = true
            }
          }
        }
      }
    }
  }

  private static addMockForeignKeyRelationships(
    tables: TableConfig[],
    edges: EdgeConfig[]
  ): void {
    // Create mock foreign key relationships for demonstration
    const tableMap = new Map<string, TableConfig>()
    tables.forEach(table => {
      tableMap.set(table.id, table)
    })

    // Look for common foreign key patterns
    for (const table of tables) {
      const columns = table.columns
      
      // Look for columns that might be foreign keys (ending with _id)
      for (const column of columns) {
        if (column.name.endsWith('_id') && !column.isPrimaryKey) {
          const referencedTableName = column.name.replace('_id', '')
          const referencedTable = tables.find(t => 
            t.name.toLowerCase() === referencedTableName.toLowerCase() && 
            t.schema === table.schema
          )
          
          if (referencedTable) {
            // Find the primary key column in the referenced table
            const pkColumn = referencedTable.columns.find(c => c.isPrimaryKey)
            if (pkColumn) {
              const edgeConfig: EdgeConfig = {
                id: `${table.id}_${column.name}_fk`,
                source: table.id,
                sourceKey: column.name,
                target: referencedTable.id,
                targetKey: pkColumn.name,
                relation: 'belongsTo',
                label: `${column.name} → ${pkColumn.name}`,
              }
              
              edges.push(edgeConfig)
              column.isForeignKey = true
            }
          }
        }
      }
    }

    // Add some additional mock relationships for demonstration
    if (tables.length >= 2) {
      const firstTable = tables[0]
      const secondTable = tables[1]
      
      const firstPk = firstTable.columns.find(c => c.isPrimaryKey)
      const secondFk = secondTable.columns.find(c => !c.isPrimaryKey)
      
      if (firstPk && secondFk) {
        const edgeConfig: EdgeConfig = {
          id: `${secondTable.id}_${firstTable.name}_fk`,
          source: secondTable.id,
          sourceKey: secondFk.name,
          target: firstTable.id,
          targetKey: firstPk.name,
          relation: 'belongsTo',
          label: `${secondFk.name} → ${firstPk.name}`,
        }
        
        edges.push(edgeConfig)
        secondFk.isForeignKey = true
      }
    }
  }
}
