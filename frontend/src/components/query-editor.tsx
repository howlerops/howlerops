import { useState, useRef, useEffect, type SyntheticEvent } from "react"
import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useQueryStore, type QueryTab } from "@/store/query-store"
import { useConnectionStore } from "@/store/connection-store"
import { useTheme } from "@/hooks/use-theme"
import { useAIConfig, useAIGeneration } from "@/store/ai-store"
import { Play, Square, Plus, X, Wand2, AlertCircle, Loader2, Network, Database, Bug, Sparkles, Users } from "lucide-react"
import { AISchemaDisplay } from "@/components/ai-schema-display"
import { cn } from "@/lib/utils"
import { useSchemaIntrospection, type SchemaNode } from "@/hooks/useSchemaIntrospection"
import { MultiDBDiagnostics } from "@/components/debug/multi-db-diagnostics"
import { CodeMirrorEditor, type CodeMirrorEditorRef } from "@/components/codemirror-editor"
import { type ColumnLoader } from "@/lib/codemirror-sql"
import { ModeSwitcher } from "@/components/mode-switcher"
import { useQueryMode } from "@/hooks/useQueryMode"
import { MultiDBConnectionSelector } from "@/components/multi-db-connection-selector"


export interface QueryEditorProps {
  mode?: 'single' | 'multi';
}

export function QueryEditor({ mode: propMode = 'single' }: QueryEditorProps = {}) {
  const { theme } = useTheme()
  const { mode, canToggle, toggleMode, connectionCount } = useQueryMode(propMode)
  const {
    activeConnection,
    connections,
    connectToDatabase,
    isConnecting,
    getFilteredConnections,
    activeEnvironmentFilter
  } = useConnectionStore()
  const {
    tabs,
    activeTabId,
    createTab,
    closeTab,
    updateTab,
    setActiveTab,
    executeQuery,
  } = useQueryStore()

  // AI Integration
  const { isEnabled: aiEnabled } = useAIConfig()
  const { generateSQL, fixSQL, isGenerating, lastError, suggestions } = useAIGeneration()
  const { schema } = useSchemaIntrospection()

  const editorRef = useRef<CodeMirrorEditorRef>(null)
  const [editorContent, setEditorContent] = useState("")
  const [naturalLanguagePrompt, setNaturalLanguagePrompt] = useState("")
  const [showAIPanel, setShowAIPanel] = useState(false)
  const [showAIDialog, setShowAIDialog] = useState(false)
  const [lastExecutionError, setLastExecutionError] = useState<string | null>(null)
  const [lastConnectionError, setLastConnectionError] = useState<string | null>(null)

  // Multi-DB state - schemas for all connections
  const [multiDBSchemas, setMultiDBSchemas] = useState<Map<string, SchemaNode[]>>(new Map())
  const multiDBSchemasRef = useRef<Map<string, SchemaNode[]>>(new Map())

  // Column cache for lazy loading (sessionId-schema-table -> columns)
  const columnCacheRef = useRef<Map<string, SchemaNode[]>>(new Map())

  // Diagnostics panel state
  const [showDiagnostics, setShowDiagnostics] = useState(false)
  
  // Multi-DB connection selector state
  const [showConnectionSelector, setShowConnectionSelector] = useState(false)

  const activeTab = tabs.find(tab => tab.id === activeTabId)

  // Remove automatic tab creation - let users create tabs manually

  useEffect(() => {
    multiDBSchemasRef.current = multiDBSchemas
  }, [multiDBSchemas])

  // Keyboard shortcut for diagnostics panel (Ctrl/Cmd+Shift+D)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'D') {
        e.preventDefault()
        setShowDiagnostics(prev => !prev)
        console.log('🐛 Diagnostics panel toggled')
      }
    }
    
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  const loadMultiDBSchemas = async () => {
    // Apply environment filter for multi-DB mode
    const relevantConnections = mode === 'multi' ? getFilteredConnections() : connections
    
    console.log('🔄 Loading multi-DB schemas...', { 
      mode, 
      totalConnections: connections.length,
      filteredConnections: relevantConnections.length,
      connectionIds: relevantConnections.map(c => c.id)
    })
    
    try {
      // Step 1: Ensure ALL filtered connections are connected (auto-connect)
      const disconnected = relevantConnections.filter(c => !c.isConnected)
      
      if (disconnected.length > 0) {
        console.log(`⚡ Auto-connecting ${disconnected.length} connections for multi-DB...`)
        
        const connectResults = await Promise.allSettled(
          disconnected.map(async (conn) => {
            try {
              await connectToDatabase(conn.id)
              console.log(`  ✓ Connected: ${conn.name}`)
            } catch (error) {
              console.warn(`  ✗ Failed: ${conn.name}`, error)
              throw error
            }
          })
        )
        
        const successful = connectResults.filter(r => r.status === 'fulfilled').length
        console.log(`⚡ Auto-connect complete: ${successful}/${disconnected.length} successful`)
        
        // Wait a bit for state to update after connections
        await new Promise(resolve => setTimeout(resolve, 100))
      }
      
      // Step 2: Get session IDs for backend (backend uses sessionId as map key!)
      // Filter to only connected connections (from filtered set) that have sessionIds
      const connectedWithSessions = relevantConnections.filter(c => c.isConnected && c.sessionId)
      const sessionIds = connectedWithSessions.map(c => c.sessionId!)
      
      console.log(`📊 Connected sessions for multi-DB:`, {
        total: connections.length,
        connectedWithSessions: connectedWithSessions.length,
        sessionIds,
        connectionDetails: connectedWithSessions.map(c => ({ name: c.name, id: c.id, sessionId: c.sessionId }))
      })
        
      if (sessionIds.length === 0) {
        console.warn('⚠️ No connected sessions for multi-DB - connections may still be connecting')
        setMultiDBSchemas(new Map())
        return
      }
      
      console.log(`📡 Fetching schemas for ${sessionIds.length} sessions:`, sessionIds)
      
      // Step 3: Load schemas using GetMultiConnectionSchema (uses cache!)
      try {
        const { GetMultiConnectionSchema } = await import('../../wailsjs/go/main/App')
        console.log('✓ Imported Wails functions')
        
        const combined = await GetMultiConnectionSchema(sessionIds)
        console.log('✓ GetMultiConnectionSchema returned')
        
        console.log('📦 Received combined schema:', {
          combinedType: typeof combined,
          hasConnections: !!combined?.connections,
          connectionCount: Object.keys(combined?.connections || {}).length,
          connections: Object.keys(combined?.connections || {})
        })
        
        if (!combined || !combined.connections) {
          console.error('❌ GetMultiConnectionSchema returned invalid data:', combined)
          setMultiDBSchemas(new Map())
          return
        }
      
      // Convert to SchemaNode format and load columns for each table
      const schemasMap = new Map<string, SchemaNode[]>()
      
      // Process each connection (connId here is sessionId from backend)
      for (const [sessionId, connSchema] of Object.entries(combined.connections || {})) {
        const schemaNodes: SchemaNode[] = []
        
        // Find the connection by sessionId to get its name
        const connection = connectedWithSessions.find(c => c.sessionId === sessionId)
        
        const schemaNames = (connSchema.schemas as string[]) || []
        const tables = (connSchema.tables as Array<{ name: string; schema: string }>) || []
        
        console.log(`📋 Processing session ${sessionId} (${connection?.name || 'unknown'}):`, {
          schemaCount: schemaNames.length,
          tableCount: tables.length
        })
        
        // Process each schema
        for (const schemaName of schemaNames) {
          const schemaTables = tables.filter(t => t.schema === schemaName)
          
          console.log(`  📁 Schema ${schemaName}: ${schemaTables.length} tables`)
          
          // Skip migration table and internal postgres tables
          const nonMigrationTables = schemaTables.filter(t => 
            t.name !== 'schema_migrations' && 
            t.name !== 'goose_db_version' &&
            t.name !== '_prisma_migrations' &&
            !t.name.startsWith('__drizzle') &&
            !schemaName.startsWith('pg_temp') &&
            !schemaName.startsWith('pg_toast')
          )
          
          // Skip empty schemas (like pg_temp_*, pg_toast_*)
          if (nonMigrationTables.length === 0) {
            continue
          }
          
          // ✅ DON'T load columns upfront - too slow and hits localStorage quota!
          // Columns will be loaded lazily when user accesses a table in autocomplete
          const tablesWithColumns: SchemaNode[] = nonMigrationTables.map(table => ({
            id: `${sessionId}-${schemaName}-${table.name}`,
            name: table.name,
            type: 'table' as const,
            schema: table.schema,
            sessionId,  // Store for lazy loading
            children: []  // Empty initially, loaded on-demand
          }))
          
          schemaNodes.push({
            id: `${sessionId}-${schemaName}`,
            name: schemaName,
            type: 'schema' as const,
            children: tablesWithColumns
          })
        }
        
        // Store by connection ID (not sessionId!) and name for lookup
        if (connection) {
          schemasMap.set(connection.id, schemaNodes)
          
          // Also store by connection name for @name.table lookup
          if (connection.name && connection.name !== connection.id) {
            schemasMap.set(connection.name, schemaNodes)
            console.log(`  🔗 Stored schemas for: ${connection.name} (id: ${connection.id})`)
          }
          
          // ✅ UPDATE BOTH STATE AND REF! Don't wait for useEffect - update ref immediately
          const newMap = new Map(schemasMap)
          setMultiDBSchemas(newMap)
          multiDBSchemasRef.current = newMap  // Direct ref update for immediate availability!
          
          console.log(`  ⚡ ${connection.name} schemas now available for autocomplete (${schemaNodes.length} schemas)`)
          console.log(`  📊 Ref updated with keys:`, Array.from(multiDBSchemasRef.current.keys()))
        } else {
          console.warn(`  ⚠️ Could not find connection for sessionId: ${sessionId}`)
        }
      }
      
      // Final update with complete schema
      setMultiDBSchemas(schemasMap)
      multiDBSchemasRef.current = schemasMap  // Final ref sync
      
      const totalTables = Array.from(schemasMap.values()).reduce((sum, schemas) => 
        sum + schemas.reduce((s, schema) => s + (schema.children?.length || 0), 0), 0
      )
      const totalColumns = Array.from(schemasMap.values()).reduce((sum, schemas) => 
        sum + schemas.reduce((s, schema) => 
          s + schema.children!.reduce((c, table) => c + (table.children?.length || 0), 0), 0
        ), 0
      )
      
      console.log('✅ Multi-DB schemas loaded successfully:', {
        connections: Array.from(schemasMap.keys()),
        totalTables,
        totalColumns
      })
      } catch (importError) {
        console.error('❌ Failed to import Wails functions or call GetMultiConnectionSchema:', importError)
        setMultiDBSchemas(new Map())
        return
      }
    } catch (error) {
      console.error('❌ Failed to load multi-DB schemas:', error)
      console.error('Error details:', {
        message: error instanceof Error ? error.message : String(error),
        stack: error instanceof Error ? error.stack : undefined
      })
      // Set empty map on error so autocomplete still works (without multi-DB)
      setMultiDBSchemas(new Map())
    }
  }

  // Load schemas for all connections when in multi-DB mode
  // Apply environment filter
  useEffect(() => {
    const filteredConns = mode === 'multi' ? getFilteredConnections() : connections
    
    console.log('🎯 Mode or connections changed:', { 
      mode, 
      connectionCount: connections.length,
      filteredCount: filteredConns.length,
      shouldLoadMultiDB: mode === 'multi' && filteredConns.length > 1
    })
    
    if (mode === 'multi' && filteredConns.length > 1) {
      loadMultiDBSchemas()
    } else if (mode === 'multi' && filteredConns.length <= 1) {
      console.log('⚠️ Multi-DB mode but only 1 filtered connection, not loading multi-DB schemas')
    }
  }, [mode, connections, getFilteredConnections, loadMultiDBSchemas])

  const columnLoader: ColumnLoader = async (sessionId: string, schema: string, tableName: string) => {
    try {
      console.log(`  ⏳ Loading columns for: ${schema}.${tableName}`)
      const { GetTableStructure } = await import('../../wailsjs/go/main/App')
      const structure = await GetTableStructure(sessionId, schema, tableName)

      if (!structure || !structure.columns || structure.columns.length === 0) {
        return []
      }

      // Convert to Column format
      return structure.columns.map((col: { name: string; data_type?: string; nullable?: boolean; primary_key?: boolean }) => ({
        name: col.name,
        dataType: col.data_type || 'unknown',
        nullable: col.nullable,
        primaryKey: col.primary_key
      }))
    } catch (error) {
      console.error(`Failed to load columns for ${schema}.${tableName}:`, error)
      return []
    }
  }

  const handleEditorDidMount = () => {
    // Editor mounted
    console.log('CodeMirror editor mounted')
  }

  const handleEditorChange = (value: string) => {
    if (activeTab) {
      setEditorContent(value)
      updateTab(activeTab.id, {
        content: value,
        isDirty: value !== activeTab.content
      })
    }
  }

  const handleExecuteQuery = async () => {
    if (!activeTab) return

    const currentEditorValue = editorRef.current?.getValue() ?? editorContent
    const queryText = currentEditorValue

    // TODO: Add selection support for CodeMirror
    // For now, just use the full editor content
    
    const trimmedValue = queryText.trim()
    if (!trimmedValue) return

    setLastExecutionError(null)

    if (currentEditorValue !== editorContent) {
      setEditorContent(currentEditorValue)
      updateTab(activeTab.id, {
        content: currentEditorValue,
        isDirty: currentEditorValue !== activeTab.content,
      })
    }

    await executeQuery(activeTab.id, trimmedValue)
  }

  // Removed handleSaveTab - not currently used

  const handleCreateTab = () => {
    // If no default connection and multiple connections, tab will be created without connection
    // User will see a prompt to select one via the tab connection dropdown
    createTab()
  }

  const handleTabConnectionChange = async (tabId: string, connectionId: string) => {
    // Clear any previous connection errors
    setLastConnectionError(null)
    
    // Update the tab's connection ID
    updateTab(tabId, { connectionId })
    
    // Check if the connection is already established
    const connection = connections.find(conn => conn.id === connectionId)
    if (connection && !connection.isConnected) {
      try {
        // Automatically connect to the database
        await connectToDatabase(connectionId)
      } catch (error) {
        console.error('Failed to connect to database:', error)
        const errorMessage = error instanceof Error ? error.message : 'Failed to connect to database'
        setLastConnectionError(errorMessage)
      }
    }
  }
  
  const handleMultiDBConnectionsChange = (tabId: string, connectionIds: string[]) => {
    updateTab(tabId, { selectedConnectionIds: connectionIds })
  }
  
  const getActiveConnectionsForTab = (tab: QueryTab) => {
    if (mode === 'single') {
      return tab.connectionId ? [tab.connectionId] : []
    } else {
      // Multi-DB mode: use selected connections or all filtered connections
      if (tab.selectedConnectionIds && tab.selectedConnectionIds.length > 0) {
        return tab.selectedConnectionIds
      }
      return getFilteredConnections().map(c => c.id)
    }
  }

  const handleCloseTab = (tabId: string, e: SyntheticEvent) => {
    e.stopPropagation()
    closeTab(tabId)
  }

  const handleTabClick = (tabId: string) => {
    setActiveTab(tabId)
    const tab = tabs.find(t => t.id === tabId)
    if (tab) {
      setEditorContent(tab.content)
    }
  }

  // AI Handler Functions
  const handleGenerateSQL = async () => {
    if (!naturalLanguagePrompt.trim() || !aiEnabled) return

    try {
      const schemaDatabase = activeConnection?.database || undefined

      // Prepare connections and schemas for both modes
      let connections = undefined
      let schemasMap = undefined

      if (mode === 'multi') {
        connections = getFilteredConnections()
        schemasMap = multiDBSchemas
      } else if (activeConnection && schema) {
        // In single DB mode, also provide schema context for better AI generation
        connections = [activeConnection]
        schemasMap = new Map([[activeConnection.id, schema]])
      }

      const generatedSQL = await generateSQL(naturalLanguagePrompt, schemaDatabase, mode, connections, schemasMap)

      if (activeTab) {
        setEditorContent(generatedSQL)
        updateTab(activeTab.id, {
          content: generatedSQL,
          isDirty: true
        })
        // Update the editor content directly
        if (editorRef.current) {
          editorRef.current.setValue(generatedSQL)
        }
      }

      setNaturalLanguagePrompt("")
      setShowAIPanel(true)
      setShowAIDialog(false)
    } catch (error) {
      console.error('Failed to generate SQL:', error)
      setLastExecutionError(error instanceof Error ? error.message : 'Failed to generate SQL')
    }
  }

  const handleFixSQL = async () => {
    if (!lastExecutionError || !editorContent.trim() || !aiEnabled) return

    try {
      const schema = activeConnection?.database || undefined
      const connections = mode === 'multi' ? getFilteredConnections() : undefined
      const schemasMap = mode === 'multi' ? multiDBSchemas : undefined
      const fixedSQL = await fixSQL(editorContent, lastExecutionError, schema, mode, connections, schemasMap)

      if (activeTab) {
        setEditorContent(fixedSQL)
        updateTab(activeTab.id, {
          content: fixedSQL,
          isDirty: true
        })
        // Update the editor content directly
        if (editorRef.current) {
          editorRef.current.setValue(fixedSQL)
        }
      }

      setLastExecutionError(null)
      setShowAIPanel(true)
    } catch (error) {
      console.error('Failed to fix SQL:', error)
    }
  }

  const handleApplySuggestion = (suggestion: string) => {
    if (activeTab) {
      setEditorContent(suggestion)
      updateTab(activeTab.id, {
        content: suggestion,
        isDirty: true
      })
      // Update the editor content directly
      if (editorRef.current) {
        editorRef.current.setValue(suggestion)
      }
    }
  }

  if (tabs.length === 0) {
    return (
      <div className="flex-1 flex w-full items-center justify-center">
        <div className="text-center">
          <h3 className="text-lg font-medium mb-2">No query tabs open</h3>
          <p className="text-mute mb-4">Create a new tab to start writing SQL queries</p>
          <Button onClick={handleCreateTab}>
            <Plus className="h-4 w-4 mr-2" />
            New Query
          </Button>
        </div>
      </div>
    )
  }

  // Test autocomplete programmatically
  const testAutocomplete = () => {
    console.log('🧪 === AUTOCOMPLETE TEST ===')
    console.log('Multi-DB Schemas Ref:', multiDBSchemasRef.current)
    console.log('Schema Keys:', Array.from(multiDBSchemasRef.current.keys()))
    console.log('Connections:', connections.map(c => ({ 
      id: c.id, 
      name: c.name, 
      isConnected: c.isConnected,
      sessionId: c.sessionId 
    })))
    
    // Simulate what happens when user types "@Prod-Leviosa."
    const testInput = "@Prod-Leviosa."
    const pattern = /@([\w-]+)\.(\w*)$/
    const match = testInput.match(pattern)
    
    console.log('Test Input:', testInput)
    console.log('Pattern Match:', match)
    
    if (match) {
      const connectionIdentifier = match[1]
      const partialTable = match[2]
      
      console.log('  Connection Identifier:', connectionIdentifier)
      console.log('  Partial Table:', partialTable)
      
      const schemas = multiDBSchemasRef.current.get(connectionIdentifier)
      console.log('  Found Schemas:', schemas ? `YES (${schemas.length} schemas)` : 'NO')
      
      if (schemas) {
        schemas.forEach(schema => {
          console.log(`    Schema: ${schema.name}, Tables: ${schema.children?.length || 0}`)
        })
      } else {
        console.log('  ❌ Schema lookup failed!')
        console.log('  Available keys:', Array.from(multiDBSchemasRef.current.keys()))
      }
    }
    
    console.log('========================')
  }

  return (
    <div className="flex-1 flex h-full min-h-0 w-full flex-col">
      {/* Diagnostics Panel - Toggle with Ctrl+Shift+D */}
      {showDiagnostics && (
        <MultiDBDiagnostics
          multiDBSchemas={multiDBSchemas}
          columnCache={columnCacheRef.current}
          onRefreshSchemas={() => {
            console.log('🔄 Manual schema refresh triggered')
            loadMultiDBSchemas()
          }}
          onTestAutocomplete={testAutocomplete}
        />
      )}
      
      {/* Enhanced Header with Mode Switcher */}
      <div className="border-b bg-background">
        {/* Top Header Bar with Mode Switcher and Global Actions */}
        <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/20">
          <div className="flex items-center gap-4">
            <ModeSwitcher
              mode={mode}
              canToggle={canToggle}
              toggleMode={toggleMode}
              connectionCount={connectionCount}
            />

            {/* Environment and Connection Status */}
            <div className="flex items-center gap-2">
              {activeEnvironmentFilter && (
                <div className="flex items-center gap-1.5 px-2 py-1 bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 rounded-md">
                  <span className="text-xs font-medium text-amber-700 dark:text-amber-300">
                    {activeEnvironmentFilter}
                  </span>
                </div>
              )}

              <div className="flex items-center gap-1.5 px-2 py-1 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-md">
                <Users className="h-3 w-3 text-green-600 dark:text-green-400" />
                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                  {getFilteredConnections().filter(c => c.isConnected).length}/{getFilteredConnections().length} Connected
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* AI Assistant Button */}
            {aiEnabled && (
              <Button
                variant={showAIDialog ? "default" : "ghost"}
                size="sm"
                onClick={() => setShowAIDialog(true)}
                title="AI SQL Assistant (Natural Language to SQL)"
              >
                <Sparkles className="h-4 w-4" />
                <span className="ml-1 text-xs hidden sm:inline">AI Assistant</span>
              </Button>
            )}

            {/* Diagnostics Toggle Button */}
            <Button
              variant={showDiagnostics ? "default" : "ghost"}
              size="sm"
              onClick={() => setShowDiagnostics(!showDiagnostics)}
              title="Toggle Diagnostics (Ctrl/Cmd+Shift+D)"
            >
              <Bug className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Tab Bar */}
        <div className="flex items-center">
        <div className="flex-1 flex items-center overflow-x-auto">
          <Tabs value={activeTabId || ''} className="w-full">
            <TabsList className="h-10 bg-transparent border-0 rounded-none p-0">
              {tabs.map((tab) => {
                const hasNoConnection = !tab.connectionId
                
                return (<div key={tab.id} className="flex items-center border-r">
                  <TabsTrigger
                    value={tab.id}
                    onClick={() => handleTabClick(tab.id)}
                    className={cn(
                      "h-10 px-3 rounded-none border-0 data-[state=active]:bg-background",
                      "data-[state=active]:border-b-2 data-[state=active]:border-primary",
                      "flex items-center space-x-2 min-w-[120px] max-w-[200px]",
                      mode === 'single' && hasNoConnection && "border-b-2 border-yellow-500/50"
                    )}
                  >
                    <span className={cn(
                      "truncate",
                      mode === 'single' && hasNoConnection && "text-yellow-600 dark:text-yellow-400"
                    )}>
                      {tab.title}
                      {tab.isDirty && <span className="ml-1">•</span>}
                      {mode === 'single' && hasNoConnection && <span className="ml-1" title="No connection selected">⚠️</span>}
                    </span>
                  </TabsTrigger>
                  
                  {/* Connection Selector - Conditional based on mode */}
                  {mode === 'single' ? (
                    // Single-DB Mode: Show dropdown
                    <div className="px-2">
                      <Select
                        value={tab.connectionId || ''}
                        onValueChange={(value) => handleTabConnectionChange(tab.id, value)}
                        disabled={isConnecting}
                      >
                        <SelectTrigger className="h-6 w-32 text-xs">
                          <SelectValue placeholder={isConnecting ? "Connecting..." : "Select DB"} />
                        </SelectTrigger>
                        <SelectContent>
                          {!tab.connectionId && (
                            <div className="px-2 py-1.5 text-xs text-yellow-600 dark:text-yellow-400 border-b">
                              ⚠️ Please select a connection
                            </div>
                          )}
                          {connections.map((conn) => (
                            <SelectItem key={conn.id} value={conn.id}>
                              <div className="flex items-center gap-2">
                                <span>{conn.name}</span>
                                {conn.isConnected ? (
                                  <span className="text-xs text-green-600">●</span>
                                ) : (
                                  <span className="text-xs text-gray-400">○</span>
                                )}
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {lastConnectionError && (
                        <div className="text-xs text-red-500 mt-1 max-w-32 truncate" title={lastConnectionError}>
                          {lastConnectionError}
                        </div>
                      )}
                    </div>
                  ) : (
                    // Multi-DB Mode: Show connection badge with click to open selector
                    <div className="px-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          setActiveTab(tab.id)
                          setShowConnectionSelector(true)
                        }}
                        className="h-6 px-2 text-xs bg-purple-50 dark:bg-purple-950/30 border-purple-200 dark:border-purple-800 hover:bg-purple-100 dark:hover:bg-purple-950/50"
                      >
                        <Network className="h-3 w-3 mr-1 text-purple-600 dark:text-purple-400" />
                        {(() => {
                          const activeConnections = getActiveConnectionsForTab(tab)
                          const filteredConns = getFilteredConnections()
                          return `${activeConnections.length}/${filteredConns.length} DBs`
                        })()}
                      </Button>
                    </div>
                  )}

                  {/* Close Button */}
                  {tabs.length > 1 && (
                    <span
                      role="button"
                      tabIndex={0}
                      className="px-1 inline-flex h-4 w-4 cursor-pointer items-center justify-center rounded hover:bg-destructive/10 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-destructive"
                      onClick={(e) => handleCloseTab(tab.id, e)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          handleCloseTab(tab.id, e)
                        }
                      }}
                    >
                      <X className="h-3 w-3" />
                    </span>
                  )}
                </div>)
              })}
            </TabsList>
          </Tabs>
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={handleCreateTab}
          className="ml-2"
        >
          <Plus className="h-4 w-4" />
        </Button>
        </div>
      </div>

      {/* AI Assistant Dialog */}
      {aiEnabled && (
        <Dialog open={showAIDialog} onOpenChange={setShowAIDialog}>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                AI SQL Assistant
              </DialogTitle>
              <DialogDescription>
                Describe what you want to query in natural language, and I'll generate the SQL for you.
                
                {/* Show current mode */}
                {mode === 'multi' ? (
                  <div className="mt-2 p-2 bg-purple-50 dark:bg-purple-950/30 border border-purple-200 dark:border-purple-800 rounded-md">
                    <div className="flex items-center gap-2 text-purple-700 dark:text-purple-300">
                      <Network className="h-4 w-4" />
                      <span className="text-sm font-medium">Multi-Database Mode Active</span>
                    </div>
                    <p className="text-xs text-purple-600 dark:text-purple-400 mt-1">
                      The AI can generate queries across multiple databases. Use @connectionName.table syntax in your descriptions.
                    </p>
                  </div>
                ) : (
                  <div className="mt-2 p-2 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-md">
                    <div className="flex items-center gap-2 text-blue-700 dark:text-blue-300">
                      <Database className="h-4 w-4" />
                      <span className="text-sm font-medium">Single Database Mode</span>
                    </div>
                    <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                      💡 Tip: Mention multiple databases or "compare" to trigger multi-database mode automatically.
                    </p>
                  </div>
                )}
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="space-y-2">
                <label htmlFor="ai-prompt" className="text-sm font-medium">
                  What would you like to query?
                </label>
                <textarea
                  id="ai-prompt"
                  placeholder="e.g., 'Show me all users who signed up last month with their total orders'"
                  value={naturalLanguagePrompt}
                  onChange={(e) => setNaturalLanguagePrompt(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                      e.preventDefault()
                      handleGenerateSQL()
                    }
                  }}
                  className="w-full h-32 p-3 text-sm bg-background border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent"
                  disabled={isGenerating}
                />
                <p className="text-xs text-muted-foreground">
                  Press Ctrl+Enter (or Cmd+Enter on Mac) to generate SQL
                </p>
              </div>

              {/* Show connection context */}
              {mode === 'single' && activeConnection && (
                <div className="p-2 bg-muted/50 rounded-lg">
                  <p className="text-xs font-medium text-muted-foreground mb-1">Using connection:</p>
                  <p className="text-sm font-medium">{activeConnection.name || activeConnection.database}</p>
                  <p className="text-xs text-muted-foreground">{activeConnection.type}</p>
                </div>
              )}

              {/* Show available schemas and tables */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Available Databases & Tables:</label>
                <AISchemaDisplay
                  mode={mode}
                  connections={mode === 'multi' ? getFilteredConnections() : (activeConnection ? [activeConnection] : [])}
                  schemasMap={mode === 'multi' ? multiDBSchemas : (activeConnection && schema ? new Map([[activeConnection.id, schema]]) : new Map())}
                  onTableClick={(connName, tableName, schemaName) => {
                    // Insert table reference into the prompt
                    const tablePath = mode === 'multi'
                      ? (schemaName === 'public' ? `@${connName}.${tableName}` : `@${connName}.${schemaName}.${tableName}`)
                      : (schemaName === 'public' ? tableName : `${schemaName}.${tableName}`)

                    const currentPrompt = naturalLanguagePrompt
                    const newPrompt = currentPrompt
                      ? `${currentPrompt} ${tablePath}`
                      : `Query the ${tablePath} table`
                    setNaturalLanguagePrompt(newPrompt)
                  }}
                  className="border rounded-lg"
                />
              </div>

              {/* Error display */}
              {lastError && (
                <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg">
                  <div className="flex items-center gap-2 text-red-600 dark:text-red-400">
                    <AlertCircle className="h-4 w-4" />
                    <p className="text-sm font-medium">Error</p>
                  </div>
                  <p className="text-sm text-red-700 dark:text-red-300 mt-1">{lastError}</p>
                </div>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setShowAIDialog(false)}
                disabled={isGenerating}
              >
                Cancel
              </Button>
              <Button
                onClick={handleGenerateSQL}
                disabled={!naturalLanguagePrompt.trim() || isGenerating}
              >
                {isGenerating ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Wand2 className="h-4 w-4 mr-2" />
                    Generate SQL
                  </>
                )}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {!activeConnection?.isConnected && (
        <div className="border-b bg-muted/20 p-2 text-sm text-muted-foreground">
          Connect to a database to run queries.
        </div>
      )}

      {/* Toolbar */}
      <div className="flex items-center justify-between p-2 border-b bg-muted/30">
        <div className="flex items-center space-x-2">
          <Button
            size="sm"
            onClick={handleExecuteQuery}
            disabled={
              !activeConnection?.isConnected ||
              !activeConnection.sessionId ||
              !editorContent.trim() ||
              !!activeTab?.isExecuting
            }
          >
            {activeTab?.isExecuting ? (
              <Square className="h-4 w-4 mr-2" />
            ) : (
              <Play className="h-4 w-4 mr-2" />
            )}
            {activeTab?.isExecuting ? 'Stop' : 'Run'}
          </Button>

          {/* AI Fix SQL Button */}
          {aiEnabled && lastExecutionError && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleFixSQL}
              disabled={isGenerating}
              className="text-orange-600 hover:text-orange-700 border-orange-200 hover:border-orange-300"
            >
              {isGenerating ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Wand2 className="h-4 w-4 mr-2" />
              )}
              Fix with AI
            </Button>
          )}

          {/* Save disabled for now */}
        </div>

        <div className="text-xs text-muted-foreground">
          {activeConnection ? (
            <span>Connected to {activeConnection.name}</span>
          ) : (
            <span>No database connection</span>
          )}
        </div>
      </div>

      {/* AI Suggestions Panel */}
      {aiEnabled && showAIPanel && suggestions.length > 0 && (
        <div className="border-b bg-green-50 dark:bg-green-950/20 p-3">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-green-800 dark:text-green-200">
                AI Suggestions
              </span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowAIPanel(false)}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="space-y-2">
            {suggestions.slice(0, 3).map((suggestion) => (
              <div
                key={suggestion.id}
                className="p-3 border rounded-lg bg-white dark:bg-gray-800 cursor-pointer hover:shadow-sm transition-shadow"
                onClick={() => handleApplySuggestion(suggestion.query)}
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-muted-foreground">
                      {suggestion.provider} • {suggestion.model}
                    </span>
                    <span className="text-xs px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded">
                      {Math.round(suggestion.confidence * 100)}% confidence
                    </span>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {suggestion.timestamp.toLocaleTimeString()}
                  </span>
                </div>

                <div className="text-sm mb-2 text-gray-700 dark:text-gray-300">
                  {suggestion.explanation}
                </div>

                <code className="block p-2 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
                  {suggestion.query}
                </code>

                <div className="mt-2 flex justify-end">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleApplySuggestion(suggestion.query)
                    }}
                  >
                    Apply
                  </Button>
                </div>
              </div>
            ))}
          </div>

          {suggestions.length > 3 && (
            <div className="mt-2 text-center">
              <span className="text-xs text-muted-foreground">
                {suggestions.length - 3} more suggestions available
              </span>
            </div>
          )}
        </div>
      )}

      {/* Editor */}
      <div className="flex-1 min-h-0 overflow-hidden">
        <CodeMirrorEditor
          ref={editorRef}
          value={editorContent}
          onChange={handleEditorChange}
          onMount={handleEditorDidMount}
          theme={theme === 'dark' ? 'dark' : 'light'}
          height="100%"
          connections={connections.map(conn => ({
            ...conn,
            isConnected: conn.isConnected || false,
            sessionId: conn.sessionId || undefined
          }))}
          schemas={multiDBSchemas}
          mode={mode}
          columnLoader={columnLoader}
          className="h-full"
        />
      </div>

      {/* Multi-DB Connection Selector Dialog */}
      {mode === 'multi' && activeTab && (
        <MultiDBConnectionSelector
          open={showConnectionSelector}
          onClose={() => setShowConnectionSelector(false)}
          selectedConnectionIds={getActiveConnectionsForTab(activeTab)}
          onSelectionChange={(connectionIds) => handleMultiDBConnectionsChange(activeTab.id, connectionIds)}
          filteredConnections={getFilteredConnections()}
        />
      )}
    </div>
  )
}
