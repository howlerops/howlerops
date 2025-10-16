/**
 * AI Schema Display Component
 *
 * Shows available databases and schemas in the AI assistant dialog
 * to help users understand what tables they can query in multi-DB mode.
 */

import { useEffect, useState } from 'react'
import { ChevronRight, ChevronDown, Database, Table, Columns, Link } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

interface SchemaNode {
  id?: string
  name: string
  type: 'schema' | 'table' | 'column'
  dataType?: string
  nullable?: boolean
  primaryKey?: boolean
  children?: SchemaNode[]
}

interface Connection {
  id: string
  name: string
  database: string
  isConnected: boolean
}

interface AISchemaDisplayProps {
  mode: 'single' | 'multi'
  connections: Connection[]
  schemasMap: Map<string, SchemaNode[]>
  onTableClick?: (connectionName: string, tableName: string, schemaName?: string) => void
  className?: string
}

export function AISchemaDisplay({
  mode,
  connections,
  schemasMap,
  onTableClick,
  className
}: AISchemaDisplayProps) {
  const [expandedDatabases, setExpandedDatabases] = useState<Set<string>>(new Set())
  const [expandedSchemas, setExpandedSchemas] = useState<Set<string>>(new Set())
  const [expandedTables, setExpandedTables] = useState<Set<string>>(new Set())

  // Auto-expand first database and schema on mount
  useEffect(() => {
    const connectedDbs = connections.filter(c => c.isConnected)
    if (connectedDbs.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setExpandedDatabases(new Set([connectedDbs[0].id]))

      const schemas = schemasMap.get(connectedDbs[0].id) || schemasMap.get(connectedDbs[0].name)
      if (schemas && schemas.length > 0) {
        setExpandedSchemas(new Set([`${connectedDbs[0].id}-${schemas[0].name}`]))
      }
    }
  }, [connections, schemasMap])

  const toggleDatabase = (dbId: string) => {
    const newExpanded = new Set(expandedDatabases)
    if (newExpanded.has(dbId)) {
      newExpanded.delete(dbId)
    } else {
      newExpanded.add(dbId)
    }
    setExpandedDatabases(newExpanded)
  }

  const toggleSchema = (schemaId: string) => {
    const newExpanded = new Set(expandedSchemas)
    if (newExpanded.has(schemaId)) {
      newExpanded.delete(schemaId)
    } else {
      newExpanded.add(schemaId)
    }
    setExpandedSchemas(newExpanded)
  }

  const toggleTable = (tableId: string) => {
    const newExpanded = new Set(expandedTables)
    if (newExpanded.has(tableId)) {
      newExpanded.delete(tableId)
    } else {
      newExpanded.add(tableId)
    }
    setExpandedTables(newExpanded)
  }

  const getTablePath = (connection: Connection, schemaName: string, tableName: string) => {
    if (mode === 'single') {
      return schemaName === 'public' ? tableName : `${schemaName}.${tableName}`
    } else {
      return schemaName === 'public'
        ? `@${connection.name}.${tableName}`
        : `@${connection.name}.${schemaName}.${tableName}`
    }
  }

  const connectedConnections = connections.filter(c => c.isConnected)

  if (connectedConnections.length === 0) {
    return (
      <div className={cn("p-4 text-center text-muted-foreground", className)}>
        <Database className="h-8 w-8 mx-auto mb-2 opacity-50" />
        <p className="text-sm">No connected databases</p>
        <p className="text-xs mt-1">Connect to a database to see available schemas</p>
      </div>
    )
  }

  return (
    <ScrollArea className={cn("h-[300px]", className)}>
      <div className="p-2 space-y-1">
        {/* Mode indicator */}
        <div className="mb-2 flex items-center gap-2">
          <Badge variant={mode === 'multi' ? 'default' : 'secondary'} className="text-xs">
            {mode === 'multi' ? 'Multi-DB Mode' : 'Single-DB Mode'}
          </Badge>
          {mode === 'multi' && (
            <span className="text-xs text-muted-foreground">
              Use @connection.table syntax
            </span>
          )}
        </div>

        {/* Database tree */}
        {connectedConnections.map(connection => {
          const schemas = schemasMap.get(connection.id) || schemasMap.get(connection.name) || []
          const isExpanded = expandedDatabases.has(connection.id)

          return (
            <Collapsible
              key={connection.id}
              open={isExpanded}
              onOpenChange={() => toggleDatabase(connection.id)}
            >
              <CollapsibleTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="w-full justify-start p-1 h-auto font-normal"
                >
                  {isExpanded ? (
                    <ChevronDown className="h-3 w-3 mr-1" />
                  ) : (
                    <ChevronRight className="h-3 w-3 mr-1" />
                  )}
                  <Database className="h-3 w-3 mr-1" />
                  <span className="text-sm font-medium">{connection.name}</span>
                  <span className="text-xs text-muted-foreground ml-auto mr-2">
                    {schemas.reduce((acc, s) => acc + (s.children?.length || 0), 0)} tables
                  </span>
                </Button>
              </CollapsibleTrigger>

              <CollapsibleContent className="pl-4">
                {schemas.map((schema, schemaIndex) => {
                  const schemaId = `${connection.id}-${schema.name}-${schemaIndex}`
                  const isSchemaExpanded = expandedSchemas.has(schemaId)

                  return (
                    <Collapsible
                      key={schemaId}
                      open={isSchemaExpanded}
                      onOpenChange={() => toggleSchema(schemaId)}
                    >
                      <CollapsibleTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="w-full justify-start p-1 h-auto font-normal"
                        >
                          {isSchemaExpanded ? (
                            <ChevronDown className="h-3 w-3 mr-1" />
                          ) : (
                            <ChevronRight className="h-3 w-3 mr-1" />
                          )}
                          <span className="text-xs">{schema.name}</span>
                          <span className="text-xs text-muted-foreground ml-auto mr-2">
                            {schema.children?.length || 0}
                          </span>
                        </Button>
                      </CollapsibleTrigger>

                      <CollapsibleContent className="pl-4">
                        {(schema.children || []).map((table, tableIndex) => {
                          const tableId = `${schemaId}-${table.name}-${tableIndex}`
                          const isTableExpanded = expandedTables.has(tableId)
                          const tablePath = getTablePath(connection, schema.name, table.name)

                          return (
                            <div key={tableId}>
                              <div className="flex items-center group">
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="flex-1 justify-start p-1 h-auto font-normal"
                                  onClick={() => {
                                    if (table.children && table.children.length > 0) {
                                      toggleTable(tableId)
                                    }
                                  }}
                                >
                                  {table.children && table.children.length > 0 ? (
                                    isTableExpanded ? (
                                      <ChevronDown className="h-3 w-3 mr-1" />
                                    ) : (
                                      <ChevronRight className="h-3 w-3 mr-1" />
                                    )
                                  ) : (
                                    <div className="w-3 mr-1" />
                                  )}
                                  <Table className="h-3 w-3 mr-1" />
                                  <span className="text-xs">{table.name}</span>
                                </Button>

                                {onTableClick && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="p-1 h-auto opacity-0 group-hover:opacity-100"
                                    onClick={() => onTableClick(connection.name, table.name, schema.name)}
                                    title={`Insert ${tablePath}`}
                                  >
                                    <Link className="h-3 w-3" />
                                  </Button>
                                )}
                              </div>

                              {/* Show columns if expanded */}
                              {isTableExpanded && table.children && (
                                <div className="pl-8 space-y-0.5">
                                  {table.children.map((column, columnIndex) => (
                                    <div
                                      key={`${tableId}-${column.name}-${columnIndex}`}
                                      className="flex items-center gap-1 py-0.5"
                                    >
                                      <Columns className="h-3 w-3 text-muted-foreground" />
                                      <span className="text-xs text-muted-foreground">
                                        {column.name}
                                      </span>
                                      <span className="text-xs text-muted-foreground">
                                        ({column.dataType || 'unknown'})
                                      </span>
                                      {column.primaryKey && (
                                        <Badge variant="outline" className="text-xs px-1 py-0 h-4">
                                          PK
                                        </Badge>
                                      )}
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>
                          )
                        })}
                      </CollapsibleContent>
                    </Collapsible>
                  )
                })}
              </CollapsibleContent>
            </Collapsible>
          )
        })}

        {/* Quick syntax help */}
        {mode === 'multi' && (
          <div className="mt-4 p-2 bg-muted/50 rounded text-xs space-y-1">
            <p className="font-medium">Multi-DB Syntax Examples:</p>
            <code className="block text-muted-foreground">@conn1.users</code>
            <code className="block text-muted-foreground">@conn2.public.orders</code>
            <code className="block text-muted-foreground">
              SELECT * FROM @prod.users u<br />
              JOIN @staging.orders o ON u.id = o.user_id
            </code>
          </div>
        )}
      </div>
    </ScrollArea>
  )
}
