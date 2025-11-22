import {
  AlertCircle,
  Database,
  Loader2,
  Network,
  RefreshCw,
  Table,
  X,
} from "lucide-react"
import { useCallback,useEffect, useState } from "react"
import { createPortal } from "react-dom"

import { SchemaTree } from "@/components/layout/sidebar"
import { SchemaVisualizerWrapper } from "@/components/schema-visualizer/schema-visualizer"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"
import { useConnectionStore } from "@/store/connection-store"

interface ConnectionSchemaViewerProps {
  connectionId: string | null
  onClose: () => void
}

interface SchemaNode {
  id: string
  name: string
  type: 'database' | 'schema' | 'table' | 'column'
  children?: SchemaNode[]
  expanded?: boolean
}

export function ConnectionSchemaViewer({ connectionId, onClose }: ConnectionSchemaViewerProps) {
  const { connections } = useConnectionStore()
  const [schema, setSchema] = useState<SchemaNode[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showVisualizer, setShowVisualizer] = useState(false)

  const connection = connections.find(conn => conn.id === connectionId)

  const loadSchema = useCallback(async () => {
    if (!connectionId || !connection?.sessionId) {
      setSchema([])
      return
    }

    setLoading(true)
    setError(null)

    try {
      // Import the Wails API dynamically
      const { GetSchemas, GetTables } = await import('../../wailsjs/go/main/App')
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
          id: `${connectionId}-${schemaName}-${table.name}`,
          name: table.name,
          type: 'table' as const,
          schema: table.schema,
          children: [] // Columns loaded on demand
        }))
        
        schemaNodes.push({
          id: `${connectionId}-${schemaName}`,
          name: schemaName,
          type: 'schema' as const,
          children: tablesWithColumns
        })
      }

      setSchema(schemaNodes)
    } catch (err) {
      console.error('Failed to load schema:', err)
      setError(err instanceof Error ? err.message : 'Failed to load schema')
    } finally {
      setLoading(false)
    }
  }, [connectionId, connection?.sessionId])

  useEffect(() => {
    if (connectionId) {
      loadSchema()
    }
  }, [connectionId, loadSchema])

  if (!connectionId || !connection) {
    return null
  }

  return createPortal(
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <Card className="w-full max-w-4xl h-[80vh] flex flex-col">
        <CardHeader className="flex flex-row items-center justify-between shrink-0 pb-2">
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            <CardTitle className="text-lg">
              Schema Explorer - {connection.name}
            </CardTitle>
            <Badge variant="outline" className="text-xs">
              {connection.type}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            {schema.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowVisualizer(true)}
                className="h-8 px-2"
                title="Schema Visualizer"
              >
                <Network className="h-4 w-4" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={loadSchema}
              disabled={loading}
              className="h-8 px-2"
              title="Refresh Schema"
            >
              <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
              className="h-8 w-8 p-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        
        <CardContent className="flex-1 overflow-hidden pt-0">
          <ScrollArea className="h-full">
            {error ? (
              <div className="flex items-center justify-center h-32 text-destructive">
                <div className="flex items-center gap-2">
                  <AlertCircle className="h-4 w-4" />
                  <span>{error}</span>
                </div>
              </div>
            ) : loading ? (
              <div className="flex items-center justify-center h-32 text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Loading schema...</span>
                </div>
              </div>
            ) : schema.length > 0 ? (
              <SchemaTree 
                key={connectionId || 'connection-schema-tree'}
                nodes={schema} 
              />
            ) : (
              <div className="flex items-center justify-center h-32 text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Table className="h-4 w-4" />
                  <span>No schemas found</span>
                </div>
              </div>
            )}
          </ScrollArea>
        </CardContent>
      </Card>

      {/* Schema Visualizer Modal */}
      {showVisualizer && (
        <SchemaVisualizerWrapper 
          schema={schema} 
          connectionId={connectionId}
          onClose={() => setShowVisualizer(false)} 
        />
      )}
    </div>,
    document.body
  )
}
