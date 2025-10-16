import { useState, useRef, useEffect, useCallback, useMemo, forwardRef, useImperativeHandle, type SyntheticEvent } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useQueryStore, type QueryTab } from "@/store/query-store"
import { useConnectionStore } from "@/store/connection-store"
import { useTheme } from "@/hooks/use-theme"
import { useAIConfig, useAIGeneration } from "@/store/ai-store"
import { Play, Square, Plus, X, Wand2, AlertCircle, Loader2, Network, Database, Bug, Sparkles, Users, Pencil, Trash2 } from "lucide-react"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { AISchemaDisplay } from "@/components/ai-schema-display"
import { cn } from "@/lib/utils"
import { useAIMemoryStore } from "@/store/ai-memory-store"
import { useSchemaIntrospection, type SchemaNode } from "@/hooks/use-schema-introspection"
import { MultiDBDiagnostics } from "@/components/debug/multi-db-diagnostics"
import { CodeMirrorEditor, type CodeMirrorEditorRef } from "@/components/codemirror-editor"
import { type ColumnLoader } from "@/lib/codemirror-sql"
import { ModeSwitcher } from "@/components/mode-switcher"
import { useQueryMode } from "@/hooks/use-query-mode"
import { MultiDBConnectionSelector } from "@/components/multi-db-connection-selector"
import { AISuggestionCard } from "@/components/ai-suggestion-card"


export interface QueryEditorProps {
  mode?: 'single' | 'multi';
}

export interface QueryEditorHandle {
  openAIFix: (error: string, query: string) => void
}

export const QueryEditor = forwardRef<QueryEditorHandle, QueryEditorProps>(({ mode: propMode = 'single' }, ref) => {
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
  const { config: aiConfig, isEnabled: aiEnabled } = useAIConfig()
  const {
    generateSQL,
    fixSQL,
    isGenerating,
    lastError,
    suggestions,
    clearSuggestions,
    resetSession,
    hydrateMemoriesFromBackend,
    deleteMemorySession,
    persistMemoriesIfEnabled,
  } = useAIGeneration()
  const { schema } = useSchemaIntrospection()

  const editorRef = useRef<CodeMirrorEditorRef>(null)
  const [editorContent, setEditorContent] = useState("")
  const [naturalLanguagePrompt, setNaturalLanguagePrompt] = useState("")
  const [showAIDialog, setShowAIDialog] = useState(false)
  const [lastExecutionError, setLastExecutionError] = useState<string | null>(null)
  const [lastConnectionError, setLastConnectionError] = useState<string | null>(null)
  const [appliedSuggestionId, setAppliedSuggestionId] = useState<string | null>(null)
  const [isFixMode, setIsFixMode] = useState(false)
  const [aiSheetTab, setAISheetTab] = useState<'assistant' | 'memories'>('assistant')
  const [renameSessionId, setRenameSessionId] = useState<string | null>(null)
  const [renameTitle, setRenameTitle] = useState('')

  // Multi-DB state - schemas for all connections
  const [multiDBSchemas, setMultiDBSchemas] = useState<Map<string, SchemaNode[]>>(new Map())
  const multiDBSchemasRef = useRef<Map<string, SchemaNode[]>>(new Map())

  // Column cache for lazy loading (sessionId-schema-table -> columns)
  const columnCacheRef = useRef<Map<string, SchemaNode[]>>(new Map())

  const memorySessionsMap = useAIMemoryStore(state => state.sessions)
  const activeMemorySessionId = useAIMemoryStore(state => state.activeSessionId)
  const setActiveMemorySession = useAIMemoryStore(state => state.setActiveSession)
  const startMemorySession = useAIMemoryStore(state => state.startNewSession)
  const renameMemorySession = useAIMemoryStore(state => state.renameSession)
  const clearAllMemorySessions = useAIMemoryStore(state => state.clearAll)
  const memorySessions = useMemo(() =>
    Object.values(memorySessionsMap).sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0)),
    [memorySessionsMap]
  )

  // Diagnostics panel state
  const [showDiagnostics, setShowDiagnostics] = useState(false)
  
  // Multi-DB connection selector state
  const [showConnectionSelector, setShowConnectionSelector] = useState(false)

  // Expose methods to parent components
  useImperativeHandle(ref, () => ({
    openAIFix: (error: string, query: string) => {
      setIsFixMode(true)
      setLastExecutionError(error)
      if (activeTab) {
        setEditorContent(query)
        updateTab(activeTab.id, { content: query })
      }
      setShowAIDialog(true)
      // Automatically trigger fix
      handleFixQueryError(error, query)
    }
  }))

  const filteredConnections = useMemo(
    () => getFilteredConnections(),
    [getFilteredConnections]
  )

  const editorConnections = useMemo(() => {
    if (mode === 'multi') {
      return filteredConnections.filter(conn => conn.isConnected)
    }

    if (activeConnection?.isConnected) {
      return [activeConnection]
    }

    return []
  }, [mode, filteredConnections, activeConnection])

  const singleConnectionSchemas = useMemo(() => {
    if (!activeConnection?.id || schema.length === 0) {
      return new Map<string, SchemaNode[]>()
    }

    const map = new Map<string, SchemaNode[]>()

    map.set(activeConnection.id, schema)
    if (activeConnection.name && activeConnection.name !== activeConnection.id) {
      map.set(activeConnection.name, schema)
    }

    if (activeConnection.name) {
      const slug = activeConnection.name.replace(/[^\w-]/g, '-')
      if (slug && slug !== activeConnection.name) {
        map.set(slug, schema)
      }
    }

    return map
  }, [activeConnection?.id, activeConnection?.name, schema])

  const codeMirrorConnections = useMemo(
    () => editorConnections.map(conn => ({
      id: conn.id,
      name: conn.name,
      type: conn.type,
      database: conn.database,
      sessionId: conn.sessionId,
      isConnected: conn.isConnected,
    })),
    [editorConnections]
  )

  const activeTab = tabs.find(tab => tab.id === activeTabId)

  // Remove automatic tab creation - let users create tabs manually

  useEffect(() => {
    multiDBSchemasRef.current = multiDBSchemas
  }, [multiDBSchemas])

  useEffect(() => {
    if (!aiEnabled || !aiConfig.syncMemories) {
      return
    }

    hydrateMemoriesFromBackend().catch(error => {
      console.error('Failed to hydrate AI memories:', error)
    })
  }, [aiEnabled, aiConfig.syncMemories, hydrateMemoriesFromBackend])

  // Keyboard shortcut for diagnostics panel (Ctrl/Cmd+Shift+D)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'D') {
        e.preventDefault()
        setShowDiagnostics(prev => !prev)
      }
    }
    
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  const loadMultiDBSchemas = useCallback(async () => {
    // Apply environment filter for multi-DB mode
    const relevantConnections = mode === 'multi' ? getFilteredConnections() : connections
    
    try {
      // Step 1: Ensure ALL filtered connections are connected (auto-connect)
      const disconnected = relevantConnections.filter(c => !c.isConnected)
      
      if (disconnected.length > 0) {
        await Promise.allSettled(
          disconnected.map(async (conn) => {
            await connectToDatabase(conn.id)
          })
        )
        
        // Wait a bit for state to update after connections
        await new Promise(resolve => setTimeout(resolve, 100))
      }
      
      // Step 2: Get session IDs for backend (backend uses sessionId as map key!)
      // Filter to only connected connections (from filtered set) that have sessionIds
      const connectedWithSessions = relevantConnections.filter(c => c.isConnected && c.sessionId)
      const sessionIds = connectedWithSessions.map(c => c.sessionId!)
        
      if (sessionIds.length === 0) {
        setMultiDBSchemas(new Map())
        return
      }
      
      // Step 3: Load schemas using GetMultiConnectionSchema (uses cache!)
      try {
        const { GetMultiConnectionSchema } = await import('../../wailsjs/go/main/App')
        const combined = await GetMultiConnectionSchema(sessionIds)
        
          if (!combined || !combined.connections) {
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
        
        // Process each schema
        for (const schemaName of schemaNames) {
          const schemaTables = tables.filter(t => t.schema === schemaName)
          
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
          const keys = new Set<string>([connection.id])

          if (connection.name) {
            keys.add(connection.name)

            const slug = connection.name.replace(/[^\w-]/g, '-')
            if (slug && slug !== connection.name) {
              keys.add(slug)
            }
          }

          keys.forEach(key => {
            schemasMap.set(key, schemaNodes)
          })

          // ✅ UPDATE BOTH STATE AND REF! Don't wait for useEffect - update ref immediately
          const newMap = new Map(schemasMap)
          setMultiDBSchemas(newMap)
          multiDBSchemasRef.current = newMap  // Direct ref update for immediate availability!
        }
      }
      
      // Final update with complete schema
      setMultiDBSchemas(schemasMap)
      multiDBSchemasRef.current = schemasMap  // Final ref sync
      } catch {
      setMultiDBSchemas(new Map())
      return
      }
    } catch {
      // Set empty map on error so autocomplete still works (without multi-DB)
      setMultiDBSchemas(new Map())
    }
  }, [mode, getFilteredConnections, connections, connectToDatabase])

  // Load schemas for all connections when in multi-DB mode
  // Apply environment filter
  useEffect(() => {
    if (mode !== 'multi') {
      return
    }

    if (filteredConnections.length === 0) {
      const emptyMap = new Map<string, SchemaNode[]>()
      setMultiDBSchemas(emptyMap)
      multiDBSchemasRef.current = emptyMap
      return
    }

    loadMultiDBSchemas()
  }, [mode, filteredConnections, loadMultiDBSchemas])

  const columnLoader: ColumnLoader = useCallback(async (sessionId: string, schema: string, tableName: string) => {
    try {
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
    } catch {
      return []
    }
  }, [])

  const handleEditorDidMount = () => {
    // Editor mounted
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

  const handleExecuteQuery = useCallback(async () => {
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
  }, [activeTab, editorContent, executeQuery, updateTab])

  // Keyboard shortcut for executing query (Ctrl/Cmd+Enter)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        if (activeConnection?.isConnected && editorContent.trim() && !activeTab?.isExecuting) {
          handleExecuteQuery()
        }
      }
    }
    
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [activeConnection, editorContent, activeTab?.isExecuting, handleExecuteQuery])

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

      await generateSQL(naturalLanguagePrompt, schemaDatabase, mode, connections, schemasMap)

      // Don't auto-apply, let user apply from suggestions in sidebar
      // The suggestion is already added to the store by generateSQL
      setNaturalLanguagePrompt("")
    } catch (error) {
      console.error('Failed to generate SQL:', error)
      setLastExecutionError(error instanceof Error ? error.message : 'Failed to generate SQL')
    }
  }

  const handleFixSQL = async () => {
    if (!lastExecutionError || !editorContent.trim() || !aiEnabled) return

    // Open the AI sidebar in fix mode
    setIsFixMode(true)
    setShowAIDialog(true)
    
    // Call the fix handler
    await handleFixQueryError(lastExecutionError, editorContent)
  }

  const handleFixQueryError = async (error: string, query: string) => {
    if (!aiEnabled) return

    try {
      const schemaDatabase = activeConnection?.database || undefined

      // Prepare connections and schemas for RAG context
      let connections = undefined
      let schemasMap = undefined

      if (mode === 'multi') {
        connections = getFilteredConnections()
        schemasMap = multiDBSchemas
      } else if (activeConnection && schema) {
        connections = [activeConnection]
        schemasMap = new Map([[activeConnection.id, schema]])
      }

      // Call fixSQL with full context including schemas for RAG
      await fixSQL(query, error, schemaDatabase, mode, connections, schemasMap)

      // Don't auto-apply, let user apply from suggestions in sidebar
      // The suggestion is already added to the store by fixSQL
    } catch (error) {
      console.error('Failed to fix SQL:', error)
    }
  }

  const handleApplySuggestion = (suggestionQuery: string, suggestionId: string) => {
    if (activeTab) {
      setEditorContent(suggestionQuery)
      updateTab(activeTab.id, {
        content: suggestionQuery,
        isDirty: true
      })
      // Update the editor content directly
      if (editorRef.current) {
        editorRef.current.setValue(suggestionQuery)
      }
      // Track applied suggestion and close dialog
      setAppliedSuggestionId(suggestionId)
      setShowAIDialog(false)
      setIsFixMode(false)
    }
  }

  const handleResetAISession = () => {
    resetSession()
    clearSuggestions()
    setNaturalLanguagePrompt('')
    setAppliedSuggestionId(null)
    setIsFixMode(false)
  }

  const handleCreateMemorySession = () => {
    const sessionId = startMemorySession({ title: `Session ${new Date().toLocaleString()}` })
    setActiveMemorySession(sessionId)
    setAISheetTab('assistant')
    void persistMemoriesIfEnabled()
  }

  const handleDeleteMemorySession = async (sessionId: string) => {
    if (!sessionId) return
    if (!window.confirm('Delete this memory session? This cannot be undone.')) {
      return
    }
    await deleteMemorySession(sessionId)
  }

  const handleClearAllMemories = () => {
    if (memorySessions.length === 0) return
    if (!window.confirm('Clear all AI memory sessions? This cannot be undone.')) {
      return
    }
    clearAllMemorySessions()
    void persistMemoriesIfEnabled()
  }

  const handleResumeMemorySession = (sessionId: string) => {
    setActiveMemorySession(sessionId)
    setAISheetTab('assistant')
  }

  const openRenameDialog = (sessionId: string, currentTitle: string) => {
    setRenameSessionId(sessionId)
    setRenameTitle(currentTitle)
  }

  const closeRenameDialog = () => {
    setRenameSessionId(null)
    setRenameTitle('')
  }

  const handleConfirmRename = () => {
    if (!renameSessionId) return
    const title = renameTitle.trim()
    if (!title) {
      return
    }
    renameMemorySession(renameSessionId, title)
    void persistMemoriesIfEnabled()
    closeRenameDialog()
  }

  useEffect(() => {
    if (!aiEnabled || !aiConfig.syncMemories) {
      return
    }

    hydrateMemoriesFromBackend().catch((error) => {
      console.error('Failed to hydrate AI memories:', error)
    })
  }, [aiEnabled, aiConfig.syncMemories, hydrateMemoriesFromBackend])

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
    // Test autocomplete functionality
    const testInput = "@Prod-Leviosa."
    const pattern = /@([\w-]+)\.(\w*)$/
    const match = testInput.match(pattern)
    
    if (match) {
      const connectionIdentifier = match[1]
      multiDBSchemasRef.current.get(connectionIdentifier)
    }
  }

  const editorSchemas = mode === 'multi' ? multiDBSchemas : singleConnectionSchemas

  return (
    <div className="flex-1 flex h-full min-h-0 w-full flex-col">
      {/* Diagnostics Panel - Toggle with Ctrl+Shift+D */}
      {showDiagnostics && (
        <MultiDBDiagnostics
          multiDBSchemas={multiDBSchemas}
          columnCache={columnCacheRef.current}
          onRefreshSchemas={() => {
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
                <Badge variant="secondary" className="gap-1.5 font-medium">
                  {activeEnvironmentFilter}
                </Badge>
              )}

              <Badge variant="secondary" className="gap-1.5 font-medium">
                <Users className="h-3 w-3" />
                {getFilteredConnections().filter(c => c.isConnected).length}/{getFilteredConnections().length} Connected
              </Badge>
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
                      mode === 'single' && hasNoConnection && "border-b-2 border-accent/50"
                    )}
                  >
                    <span className={cn(
                      "truncate",
                      mode === 'single' && hasNoConnection && "text-accent-foreground"
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
                            <div className="px-2 py-1.5 text-xs text-accent-foreground border-b">
                              ⚠️ Please select a connection
                            </div>
                          )}
                          {connections.map((conn) => (
                            <SelectItem key={conn.id} value={conn.id}>
                              <div className="flex items-center gap-2">
                                <span>{conn.name}</span>
                                {conn.isConnected ? (
                                  <span className="text-xs text-primary">●</span>
                                ) : (
                                  <span className="text-xs text-muted-foreground">○</span>
                                )}
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {lastConnectionError && (
                        <div className="text-xs text-destructive mt-1 max-w-32 truncate" title={lastConnectionError}>
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
                        className="h-6 px-2 text-xs bg-accent/10 border-accent hover:bg-accent/20"
                      >
                        <Network className="h-3 w-3 mr-1 text-accent-foreground" />
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

      {/* AI Assistant Drawer */}
      {aiEnabled && (
        <Sheet open={showAIDialog} onOpenChange={setShowAIDialog}>
          <SheetContent
            side="right"
            className="w-[600px] sm:max-w-[600px] m-4 h-[calc(100vh-2rem)] rounded-xl shadow-2xl border overflow-y-auto flex flex-col"
          >
            <SheetHeader className="space-y-4 border-b pb-4">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <SheetTitle className="flex items-center gap-2">
                    <Sparkles className="h-5 w-5 text-primary" />
                    {isFixMode ? 'AI Query Fixer' : 'AI SQL Assistant'}
                  </SheetTitle>
                  <SheetDescription className="mt-2 text-left">
                    {isFixMode ? (
                      <>
                        The AI will analyze the error and suggest fixes for your query.
                        {lastExecutionError && (
                          <Alert variant="destructive" className="mt-2">
                            <AlertCircle className="h-4 w-4" />
                            <AlertTitle>Query Error</AlertTitle>
                            <AlertDescription className="text-xs whitespace-pre-wrap">
                              {lastExecutionError}
                            </AlertDescription>
                          </Alert>
                        )}
                      </>
                    ) : (
                      <>
                        Describe what you want to query in natural language, and I'll generate the SQL for you.

                        {mode === 'multi' ? (
                          <Alert className="mt-2">
                            <Network className="h-4 w-4" />
                            <AlertTitle>Multi-Database Mode Active</AlertTitle>
                            <AlertDescription>
                              The AI can generate queries across multiple databases. Use @connectionName.table syntax in your descriptions.
                            </AlertDescription>
                          </Alert>
                        ) : (
                          <Alert className="mt-2">
                            <Database className="h-4 w-4" />
                            <AlertTitle>Single Database Mode</AlertTitle>
                            <AlertDescription>
                              💡 Tip: Mention multiple databases or "compare" to trigger multi-database mode automatically.
                            </AlertDescription>
                          </Alert>
                        )}
                      </>
                    )}
                  </SheetDescription>
                </div>
                {!isFixMode && (
                  <Button size="sm" onClick={handleCreateMemorySession}>
                    <Plus className="h-4 w-4 mr-2" />
                    New Session
                  </Button>
                )}
              </div>
              {!isFixMode && (
                <Tabs
                  value={aiSheetTab}
                  onValueChange={(value) => setAISheetTab(value as 'assistant' | 'memories')}
                  className="w-full"
                >
                  <TabsList className="grid w-full grid-cols-2 bg-muted/40 p-1 rounded-lg">
                    <TabsTrigger value="assistant" className="h-8 text-sm">
                      Assistant
                    </TabsTrigger>
                    <TabsTrigger value="memories" className="h-8 text-sm">
                      Memories
                    </TabsTrigger>
                  </TabsList>
                </Tabs>
              )}
            </SheetHeader>

            <div className="flex-1 overflow-hidden">
              {isFixMode || aiSheetTab === 'assistant' ? (
                <div className="grid h-full gap-4 py-4 pr-2 overflow-y-auto">
                  {!isFixMode && (
                    <>
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

                      {mode === 'single' && activeConnection && (
                        <div className="p-2 bg-muted/50 rounded-lg">
                          <p className="text-xs font-medium text-muted-foreground mb-1">Using connection:</p>
                          <p className="text-sm font-medium">{activeConnection.name || activeConnection.database}</p>
                          <p className="text-xs text-muted-foreground">{activeConnection.type}</p>
                        </div>
                      )}

                      <div className="space-y-2">
                        <label className="text-sm font-medium">Available Databases & Tables:</label>
                        <AISchemaDisplay
                          mode={mode}
                          connections={mode === 'multi' ? getFilteredConnections() : (activeConnection ? [activeConnection] : [])}
                          schemasMap={mode === 'multi' ? multiDBSchemas : (activeConnection && schema ? new Map([[activeConnection.id, schema]]) : new Map())}
                          onTableClick={(connName, tableName, schemaName) => {
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
                    </>
                  )}

                  {lastError && (
                    <Alert variant="destructive">
                      <AlertCircle className="h-4 w-4" />
                      <AlertTitle>Error</AlertTitle>
                      <AlertDescription>{lastError}</AlertDescription>
                    </Alert>
                  )}

                  {suggestions.length > 0 && (
                    <div className="border-t pt-4 space-y-3">
                      <div className="flex items-center justify-between">
                        <h3 className="text-sm font-semibold">
                          {isFixMode ? 'Suggested Fixes' : 'Generated Queries'}
                        </h3>
                        <span className="text-xs text-muted-foreground">
                          {suggestions.length} suggestion{suggestions.length !== 1 ? 's' : ''}
                        </span>
                      </div>
                      <div className="space-y-3 max-h-[400px] overflow-y-auto">
                        {suggestions.map((suggestion) => (
                          <AISuggestionCard
                            key={suggestion.id}
                            suggestion={suggestion}
                            onApply={(query) => handleApplySuggestion(query, suggestion.id)}
                            isApplied={appliedSuggestionId === suggestion.id}
                          />
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex h-full flex-col py-4">
                  <div className="flex flex-col gap-2 border-b pb-4 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <h3 className="text-sm font-semibold">Memory Sessions</h3>
                      <p className="text-xs text-muted-foreground">
                        Switch between saved assistant context or start fresh sessions.
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleClearAllMemories}
                        disabled={memorySessions.length === 0}
                      >
                        Clear All
                      </Button>
                      <Button size="sm" onClick={handleCreateMemorySession}>
                        <Plus className="h-4 w-4 mr-2" />
                        New Session
                      </Button>
                    </div>
                  </div>

                  <ScrollArea className="flex-1 pr-4">
                    <div className="space-y-2 py-4">
                      {memorySessions.length === 0 ? (
                        <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
                          No memory sessions yet. Create one to let the assistant remember context.
                        </div>
                      ) : (
                        memorySessions.map((session) => (
                          <div
                            key={session.id}
                            className={cn(
                              "rounded-lg border p-3 transition-colors",
                              session.id === activeMemorySessionId
                                ? "border-primary bg-primary/5"
                                : "hover:bg-muted/50"
                            )}
                          >
                            <div className="flex items-start justify-between gap-2">
                              <div className="min-w-0">
                                <p className="text-sm font-semibold flex items-center gap-2">
                                  <span className="truncate">{session.title}</span>
                                  {session.id === activeMemorySessionId && (
                                    <Badge variant="secondary" className="text-[10px] uppercase tracking-wide">
                                      Active
                                    </Badge>
                                  )}
                                </p>
                                <p className="text-xs text-muted-foreground mt-1">
                                  Updated {new Date((session.updatedAt || session.createdAt)).toLocaleString()}
                                </p>
                                <p className="text-xs text-muted-foreground mt-1">
                                  {session.messages.length} message{session.messages.length === 1 ? '' : 's'}
                                </p>
                                {session.summary && (
                                  <p className="text-xs text-muted-foreground mt-2 whitespace-pre-wrap">
                                    {session.summary}
                                  </p>
                                )}
                              </div>
                              <div className="flex items-center gap-1">
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7"
                                  onClick={() => openRenameDialog(session.id, session.title)}
                                >
                                  <Pencil className="h-3.5 w-3.5" />
                                  <span className="sr-only">Rename session</span>
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7 text-destructive"
                                  onClick={() => { void handleDeleteMemorySession(session.id) }}
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                  <span className="sr-only">Delete session</span>
                                </Button>
                              </div>
                            </div>
                            <div className="mt-3 flex flex-wrap gap-2">
                              <Button
                                size="sm"
                                variant={session.id === activeMemorySessionId ? "default" : "outline"}
                                onClick={() => handleResumeMemorySession(session.id)}
                              >
                                {session.id === activeMemorySessionId ? 'Continue in Assistant' : 'Resume in Assistant'}
                              </Button>
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                  </ScrollArea>
                </div>
              )}
            </div>

            <div className="flex flex-wrap justify-end gap-2 border-t pt-4">
              {(isFixMode || aiSheetTab === 'assistant') && (
                <Button
                  variant="outline"
                  onClick={handleResetAISession}
                  disabled={isGenerating}
                >
                  Reset Session
                </Button>
              )}
              <Button
                variant="outline"
                onClick={() => {
                  setShowAIDialog(false)
                  setIsFixMode(false)
                }}
                disabled={isGenerating}
              >
                Close
              </Button>
              {!isFixMode && aiSheetTab === 'assistant' && (
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
              )}
            </div>
          </SheetContent>
        </Sheet>
      )}

      <Dialog
        open={!!renameSessionId}
        onOpenChange={(open) => {
          if (!open) {
            closeRenameDialog()
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename Memory Session</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <Input
              autoFocus
              value={renameTitle}
              onChange={(e) => setRenameTitle(e.target.value)}
              placeholder="Session title"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeRenameDialog}>
              Cancel
            </Button>
            <Button onClick={handleConfirmRename} disabled={!renameTitle.trim()}>
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
              className="text-accent-foreground hover:text-accent-foreground/80 border-accent hover:border-accent"
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

      {/* Editor */}
      <div className="flex-1 min-h-0 overflow-hidden">
        <CodeMirrorEditor
          ref={editorRef}
          value={editorContent}
            onChange={handleEditorChange}
            onMount={handleEditorDidMount}
            theme={theme === 'dark' ? 'dark' : 'light'}
          height="100%"
          connections={codeMirrorConnections}
          schemas={editorSchemas}
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
})

QueryEditor.displayName = 'QueryEditor'
