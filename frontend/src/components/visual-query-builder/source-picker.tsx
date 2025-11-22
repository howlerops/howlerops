/**
 * Source Picker Component for Visual Query Builder
 * Handles connection and table selection
 */

import { AlertCircle,Database, Loader2 } from 'lucide-react'
import { useEffect,useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableRef } from '@/lib/query-ir'

import { SourcePickerProps, TableInfo } from './types'

export function SourcePicker({
  connections,
  schemas,
  selectedConnections,
  selectedTable,
  onConnectionsChange,
  onTableChange,
  onSchemaLoad
}: SourcePickerProps) {
  const [loadingConnections, setLoadingConnections] = useState<Set<string>>(new Set())
  const [availableTables, setAvailableTables] = useState<TableInfo[]>([])
  const [selectedSchema, setSelectedSchema] = useState<string>('')

  // Load schemas for selected connections
  useEffect(() => {
    const loadSchemas = async () => {
      for (const connectionId of selectedConnections) {
        if (!schemas.has(connectionId)) {
          setLoadingConnections(prev => new Set(prev).add(connectionId))
          try {
            await onSchemaLoad(connectionId)
          } catch (error) {
            console.error(`Failed to load schema for connection ${connectionId}:`, error)
          } finally {
            setLoadingConnections(prev => {
              const next = new Set(prev)
              next.delete(connectionId)
              return next
            })
          }
        }
      }
    }

    if (selectedConnections.length > 0) {
      loadSchemas()
    }
  }, [selectedConnections, schemas, onSchemaLoad])

  // Update available tables when connections or schemas change
  useEffect(() => {
    const tables: TableInfo[] = []
    
    for (const connectionId of selectedConnections) {
      const connectionSchemas = schemas.get(connectionId) || []
      for (const schema of connectionSchemas) {
        for (const table of schema.tables) {
          tables.push({
            ...table,
            schema: schema.name
          })
        }
      }
    }
    
    setAvailableTables(tables)
  }, [selectedConnections, schemas])

  // Handle connection selection
  const handleConnectionToggle = (connectionId: string) => {
    const isSelected = selectedConnections.includes(connectionId)
    if (isSelected) {
      onConnectionsChange(selectedConnections.filter(id => id !== connectionId))
    } else {
      onConnectionsChange([...selectedConnections, connectionId])
    }
  }

  // Handle table selection
  const handleTableSelect = (tableName: string) => {
    if (!tableName || !selectedSchema) return
    
    const table: TableRef = {
      schema: selectedSchema,
      table: tableName
    }
    onTableChange(table)
  }

  // Get available schemas for selected connections
  const getAvailableSchemas = (): string[] => {
    const schemaNames = new Set<string>()
    
    for (const connectionId of selectedConnections) {
      const connectionSchemas = schemas.get(connectionId) || []
      for (const schema of connectionSchemas) {
        schemaNames.add(schema.name)
      }
    }
    
    return Array.from(schemaNames).sort()
  }

  // Get tables for selected schema
  const getTablesForSchema = (schemaName: string): TableInfo[] => {
    return availableTables.filter(table => table.schema === schemaName)
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium mb-2">Data Sources</h3>
        
        {/* Connection Selection */}
        <div className="space-y-2">
          {connections.map(connection => {
            const isSelected = selectedConnections.includes(connection.id)
            const isLoading = loadingConnections.has(connection.id)
            const isConnected = connection.isConnected
            
            return (
              <div key={connection.id} className="flex items-center space-x-2">
                <Button
                  variant={isSelected ? "default" : "outline"}
                  size="sm"
                  onClick={() => handleConnectionToggle(connection.id)}
                  disabled={!isConnected && !isLoading}
                  className="flex-1 justify-start"
                >
                  <Database className="w-4 h-4 mr-2" />
                  {connection.name}
                  {isLoading && <Loader2 className="w-4 h-4 ml-2 animate-spin" />}
                  {!isConnected && !isLoading && (
                    <AlertCircle className="w-4 h-4 ml-2 text-yellow-500" />
                  )}
                </Button>
                
                {isSelected && (
                  <Badge variant="secondary">
                    {selectedConnections.length > 1 ? 'Multi-DB' : 'Single'}
                  </Badge>
                )}
              </div>
            )
          })}
        </div>

        {/* Multi-connection warning */}
        {selectedConnections.length > 1 && (
          <Alert className="mt-2">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Multi-connection mode: queries will be executed across all selected connections and results merged.
              Cross-database joins are not supported.
            </AlertDescription>
          </Alert>
        )}
      </div>

      {/* Table Selection */}
      {selectedConnections.length > 0 && (
        <div>
          <h3 className="text-sm font-medium mb-2">Table</h3>
          
          {/* Schema Selection */}
          <div className="mb-3">
            <Select value={selectedSchema} onValueChange={setSelectedSchema}>
              <SelectTrigger>
                <SelectValue placeholder="Select schema..." />
              </SelectTrigger>
              <SelectContent>
                {getAvailableSchemas().map(schemaName => (
                  <SelectItem key={schemaName} value={schemaName}>
                    {schemaName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Table Selection */}
          {selectedSchema && (
            <Select 
              value={selectedTable?.table || ''} 
              onValueChange={handleTableSelect}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select table..." />
              </SelectTrigger>
              <SelectContent>
                {getTablesForSchema(selectedSchema).map(table => (
                  <SelectItem key={table.name} value={table.name}>
                    <div className="flex items-center justify-between w-full">
                      <span>{table.name}</span>
                      {table.rowCount !== undefined && (
                        <Badge variant="outline" className="ml-2">
                          {table.rowCount.toLocaleString()} rows
                        </Badge>
                      )}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}

          {/* Selected table info */}
          {selectedTable && (
            <div className="mt-2 p-2 bg-muted rounded-md">
              <div className="text-sm font-medium">
                {selectedTable.schema}.{selectedTable.table}
              </div>
              {selectedConnections.length > 1 && (
                <div className="text-xs text-muted-foreground mt-1">
                  Available in {selectedConnections.length} connection(s)
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* No connections selected */}
      {selectedConnections.length === 0 && (
        <div className="text-center py-8 text-muted-foreground">
          <Database className="w-8 h-8 mx-auto mb-2 opacity-50" />
          <p>Select one or more connections to begin</p>
        </div>
      )}
    </div>
  )
}
