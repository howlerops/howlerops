import React, { useState, useEffect } from 'react'
import { useConnectionStore } from '@/store/connection-store'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { CheckCircle, XCircle, RefreshCw, Bug, ChevronDown, ChevronRight } from 'lucide-react'

interface SchemaNode {
  name: string;
  type: string;
  children?: SchemaNode[];
}

interface Column {
  name: string;
  type: string;
}

interface DiagnosticProps {
  multiDBSchemas: Map<string, SchemaNode[]>
  columnCache: Map<string, Column[]>
  onRefreshSchemas: () => void
  onTestAutocomplete: () => void
}

export function MultiDBDiagnostics({ 
  multiDBSchemas, 
  columnCache, 
  onRefreshSchemas,
  onTestAutocomplete 
}: DiagnosticProps) {
  const { connections, activeConnection } = useConnectionStore()
  const [expanded, setExpanded] = useState(false)
  const [schemaDetails, setSchemaDetails] = useState<Record<string, boolean>>({})

  const connectedCount = connections.filter(c => c.isConnected).length
  const totalSchemas = multiDBSchemas.size
  const totalCachedColumns = columnCache.size

  // Auto-expand if there are issues
  useEffect(() => {
    if (connections.length > 0 && connectedCount === 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setExpanded(true)
    }
    if (connections.length > 1 && totalSchemas === 0) {
      setExpanded(true)
    }
  }, [connections.length, connectedCount, totalSchemas])

  const toggleSchemaDetail = (key: string) => {
    setSchemaDetails(prev => ({ ...prev, [key]: !prev[key] }))
  }

  return (
    <Card className="mb-4 border-accent">
      <CardHeader 
        className="cursor-pointer hover:bg-accent/50 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bug className="h-5 w-5 text-accent-foreground" />
            <CardTitle className="text-sm">Multi-DB Diagnostics</CardTitle>
            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={connectedCount === connections.length ? "default" : "destructive"}>
              {connectedCount}/{connections.length} Connected
            </Badge>
            <Badge variant={totalSchemas > 0 ? "default" : "secondary"}>
              {totalSchemas} Schemas Loaded
            </Badge>
            <Badge variant="outline">
              {totalCachedColumns} Cached Tables
            </Badge>
          </div>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-4">
          {/* Connection Status */}
          <div>
            <h4 className="text-sm font-semibold mb-2">Connections</h4>
            <div className="space-y-2">
              {connections.length === 0 ? (
                <p className="text-sm text-muted-foreground">No connections configured</p>
              ) : (
                connections.map(conn => (
                  <div key={conn.id} className="flex items-center justify-between p-2 border rounded text-sm">
                    <div className="flex items-center gap-2">
                      {conn.isConnected ? (
                        <CheckCircle className="h-4 w-4 text-primary" />
                      ) : (
                        <XCircle className="h-4 w-4 text-destructive" />
                      )}
                      <span className="font-medium">{conn.name || 'Unnamed'}</span>
                      <Badge variant="outline" className="text-xs">
                        {conn.type}
                      </Badge>
                      {conn.id === activeConnection?.id && (
                        <Badge variant="default" className="text-xs">Active</Badge>
                      )}
                    </div>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>ID: {conn.id.slice(0, 8)}...</span>
                      {conn.sessionId && (
                        <span>Session: {conn.sessionId.slice(0, 8)}...</span>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Schema Loading Status */}
          <div>
            <h4 className="text-sm font-semibold mb-2">Multi-DB Schemas</h4>
            {totalSchemas === 0 ? (
              <div className="p-3 border border-accent bg-yellow-50 dark:bg-accent/10/20 rounded">
                <p className="text-sm text-accent-foreground dark:text-accent-foreground">
                  ⚠️ No schemas loaded. Multi-DB autocomplete will not work.
                </p>
                <p className="text-xs text-accent-foreground dark:text-accent-foreground mt-1">
                  Expected: Map with connection names/IDs as keys
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                <p className="text-xs text-muted-foreground mb-2">
                  Schema map keys (used for @connection lookup):
                </p>
                {Array.from(multiDBSchemas.entries()).map(([key, schemas]) => (
                  <div key={key} className="border rounded">
                    <div 
                      className="flex items-center justify-between p-2 cursor-pointer hover:bg-accent/50"
                      onClick={() => toggleSchemaDetail(key)}
                    >
                      <div className="flex items-center gap-2">
                        {schemaDetails[key] ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                        <span className="font-mono text-sm">{key}</span>
                      </div>
                      <Badge variant="outline" className="text-xs">
                        {schemas.length} schema(s)
                      </Badge>
                    </div>
                    
                    {schemaDetails[key] && (
                      <div className="px-4 pb-2 space-y-1 text-xs">
                        {schemas.map((schema: SchemaNode, idx: number) => {
                          const tableCount = schema.children?.length || 0
                          const columnCount = schema.children?.reduce((sum: number, table: SchemaNode) => 
                            sum + (table.children?.length || 0), 0) || 0
                          
                          return (
                            <div key={idx} className="flex items-center justify-between py-1 px-2 bg-accent/30 rounded">
                              <span className="font-medium">{schema.name}</span>
                              <div className="flex gap-2 text-muted-foreground">
                                <span>{tableCount} tables</span>
                                <span>{columnCount} columns</span>
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Column Cache */}
          <div>
            <h4 className="text-sm font-semibold mb-2">Column Cache (Lazy Loading)</h4>
            {totalCachedColumns === 0 ? (
              <p className="text-sm text-muted-foreground">No columns cached yet. They load on first access.</p>
            ) : (
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground mb-2">
                  Cached table columns ({totalCachedColumns} total):
                </p>
                <div className="max-h-48 overflow-y-auto space-y-1">
                  {Array.from(columnCache.entries()).slice(0, 50).map(([key, columns]) => (
                    <div key={key} className="flex items-center justify-between text-xs p-1 border rounded">
                      <span className="font-mono">{key}</span>
                      <Badge variant="outline" className="text-xs">
                        {columns.length} columns
                      </Badge>
                    </div>
                  ))}
                  {totalCachedColumns > 50 && (
                    <p className="text-xs text-muted-foreground italic">
                      ... and {totalCachedColumns - 50} more
                    </p>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Diagnostic Actions */}
          <div className="flex gap-2 pt-2 border-t">
            <Button 
              size="sm" 
              variant="outline"
              onClick={onRefreshSchemas}
              className="flex items-center gap-2"
            >
              <RefreshCw className="h-3 w-3" />
              Refresh Schemas
            </Button>
            <Button 
              size="sm" 
              variant="outline"
              onClick={onTestAutocomplete}
              className="flex items-center gap-2"
            >
              <Bug className="h-3 w-3" />
              Test Autocomplete
            </Button>
            <Button 
              size="sm" 
              variant="outline"
              onClick={() => {
                console.log('=== DIAGNOSTIC SNAPSHOT ===')
                console.log('Connections:', connections)
                console.log('Connected:', connectedCount)
                console.log('MultiDB Schemas:', multiDBSchemas)
                console.log('Schema Keys:', Array.from(multiDBSchemas.keys()))
                console.log('Column Cache:', columnCache)
                console.log('Column Cache Keys:', Array.from(columnCache.keys()))
                console.log('========================')
              }}
            >
              📋 Log State
            </Button>
          </div>

          {/* Health Check Summary */}
          <div className="pt-2 border-t">
            <h4 className="text-sm font-semibold mb-2">Health Check</h4>
            <div className="space-y-1 text-xs">
              <HealthCheckItem 
                label="Connections configured"
                status={connections.length > 0}
                detail={`${connections.length} total`}
              />
              <HealthCheckItem 
                label="At least one connected"
                status={connectedCount > 0}
                detail={`${connectedCount} connected`}
              />
              <HealthCheckItem 
                label="Multi-DB schemas loaded"
                status={totalSchemas > 0}
                detail={`${totalSchemas} schema map entries`}
              />
              <HealthCheckItem 
                label="Schema keys match connections"
                status={totalSchemas > 0 && connections.some(c => 
                  multiDBSchemas.has(c.id) || multiDBSchemas.has(c.name!)
                )}
                detail={totalSchemas > 0 ? 'Keys found in map' : 'No schemas to check'}
              />
            </div>
          </div>
        </CardContent>
      )}
    </Card>
  )
}

function HealthCheckItem({ 
  label, 
  status, 
  detail 
}: { 
  label: string
  status: boolean
  detail?: string 
}) {
  return (
    <div className="flex items-center justify-between p-2 bg-accent/20 rounded">
      <div className="flex items-center gap-2">
        {status ? (
          <CheckCircle className="h-3 w-3 text-primary" />
        ) : (
          <XCircle className="h-3 w-3 text-destructive" />
        )}
        <span>{label}</span>
      </div>
      {detail && (
        <span className="text-muted-foreground">{detail}</span>
      )}
    </div>
  )
}

