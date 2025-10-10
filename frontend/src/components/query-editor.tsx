import { useState, useRef, useEffect, useMemo, type SyntheticEvent } from "react"
import Editor from "@monaco-editor/react"
import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useQueryStore } from "@/store/query-store"
import { useConnectionStore } from "@/store/connection-store"
import { useTheme } from "@/hooks/use-theme"
import { useAIConfig, useAIGeneration } from "@/store/ai-store"
import { Play, Square, Plus, X, Save, Brain, Wand2, AlertCircle, Lightbulb, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { useSchemaIntrospection, type SchemaNode } from "@/hooks/useSchemaIntrospection"

type TableEntry = {
  key: string
  schema: string
  table: string
  label: string
  insertText: string
  detail: string
}

type ColumnEntry = {
  key: string
  schema: string
  table: string
  column: string
  label: string
  insertText: string
  detail?: string
}

type SchemaIndex = {
  tables: TableEntry[]
  tablesByName: Record<string, TableEntry>
  tablesByFullName: Record<string, TableEntry>
  columnsByTable: Record<string, ColumnEntry[]>
  allColumns: ColumnEntry[]
}

const normalizeIdentifier = (value?: string | null) => {
  if (!value) return ""
  return value.replace(/["`]/g, "").toLowerCase()
}

const buildSchemaIndex = (schemaNodes: SchemaNode[]): SchemaIndex => {
  const tables: TableEntry[] = []
  const tablesByName: Record<string, TableEntry> = {}
  const tablesByFullName: Record<string, TableEntry> = {}
  const columnsByTable: Record<string, ColumnEntry[]> = {}
  const allColumns: ColumnEntry[] = []

  schemaNodes.forEach((schemaNode) => {
    if (schemaNode.type !== "schema" || !schemaNode.children) return
    const schemaName = schemaNode.name

    schemaNode.children.forEach((tableNode) => {
      if (tableNode.type !== "table") return
      const tableName = tableNode.name
      const tableKey = `${schemaName}.${tableName}`
      const normalizedKey = normalizeIdentifier(tableKey)

      const tableEntry: TableEntry = {
        key: normalizedKey,
        schema: schemaName,
        table: tableName,
        label: tableName,
        insertText: tableName,
        detail: schemaName ? `${schemaName}.${tableName}` : tableName,
      }

      tables.push(tableEntry)
      tablesByName[normalizeIdentifier(tableName)] = tableEntry
      tablesByFullName[normalizeIdentifier(tableKey)] = tableEntry

      const columnEntries: ColumnEntry[] = []
      tableNode.children?.forEach((columnNode) => {
        if (columnNode.type !== "column") return
        const metadata = columnNode.metadata || {}
        const rawName = metadata.name || (typeof metadata === "string" ? metadata : columnNode.name.split(" ")[0])
        const columnName = String(rawName)

        const columnEntry: ColumnEntry = {
          key: normalizedKey,
          schema: schemaName,
          table: tableName,
          column: columnName,
          label: columnName,
          insertText: columnName,
          detail: metadata.dataType ? `${tableName}.${columnName} (${metadata.dataType})` : `${tableName}.${columnName}`,
        }

        columnEntries.push(columnEntry)
        allColumns.push(columnEntry)
      })

      columnsByTable[normalizedKey] = columnEntries
    })
  })

  return {
    tables,
    tablesByName,
    tablesByFullName,
    columnsByTable,
    allColumns,
  }
}

const buildAliasMap = (query: string, index: SchemaIndex): Record<string, TableEntry> => {
  const aliasMap: Record<string, TableEntry> = {}
  const regex = /\b(?:FROM|JOIN)\s+([A-Za-z0-9_."`]+)(?:\s+(?:AS\s+)?([A-Za-z0-9_]+))?/gi
  let match: RegExpExecArray | null

  while ((match = regex.exec(query)) !== null) {
    const tableIdentifier = match[1]
    const alias = match[2]
    const normalizedIdentifier = normalizeIdentifier(tableIdentifier)

    const tableEntry =
      index.tablesByFullName[normalizedIdentifier] ||
      index.tablesByName[normalizedIdentifier]

    if (tableEntry) {
      aliasMap[normalizedIdentifier] = tableEntry
      aliasMap[normalizeIdentifier(tableEntry.table)] = tableEntry
      if (alias) {
        aliasMap[normalizeIdentifier(alias)] = tableEntry
      }
    }
  }

  return aliasMap
}

const findTableEntry = (
  identifier: string | undefined,
  index: SchemaIndex,
  aliasMap: Record<string, TableEntry>
) => {
  if (!identifier) return undefined
  const normalized = normalizeIdentifier(identifier)
  return (
    aliasMap[normalized] ||
    index.tablesByFullName[normalized] ||
    index.tablesByName[normalized]
  )
}

const getTextUntilPosition = (model: unknown, position: unknown) =>
  model.getValueInRange({
    startLineNumber: position.lineNumber,
    startColumn: 1,
    endLineNumber: position.lineNumber,
    endColumn: position.column,
  })

const getPrefixBeforeWord = (model: unknown, position: unknown, word: unknown) =>
  model.getValueInRange({
    startLineNumber: position.lineNumber,
    startColumn: 1,
    endLineNumber: position.lineNumber,
    endColumn: word.startColumn,
  })

const getTokenBeforeDot = (text: string) => {
  const dotIndex = text.lastIndexOf(".")
  if (dotIndex === -1) return undefined
  const beforeDot = text.slice(0, dotIndex)
  const match = beforeDot.match(/([A-Za-z0-9_"`]+)\s*$/)
  return match ? match[1] : undefined
}

const getPreviousToken = (text: string) => {
  const trimmed = text.trim()
  if (!trimmed) return ""
  const tokens = trimmed.split(/\s+/)
  return tokens[tokens.length - 1] || ""
}

export function QueryEditor() {
  const { theme } = useTheme()
  const { activeConnection, connections, connectToDatabase, isConnecting } = useConnectionStore()
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

  const editorRef = useRef<unknown>(null)
  const completionProviderRef = useRef<{ dispose: () => void } | null>(null)
  const schemaRef = useRef<SchemaNode[]>([])
  const schemaIndexRef = useRef<SchemaIndex>(buildSchemaIndex([]))
  const [editorContent, setEditorContent] = useState("")
  const [naturalLanguagePrompt, setNaturalLanguagePrompt] = useState("")
  const [showAIPanel, setShowAIPanel] = useState(false)
  const [lastExecutionError, setLastExecutionError] = useState<string | null>(null)
  const [lastConnectionError, setLastConnectionError] = useState<string | null>(null)

  const activeTab = tabs.find(tab => tab.id === activeTabId)

  // Remove automatic tab creation - let users create tabs manually

  const schemaIndex = useMemo(() => buildSchemaIndex(schema), [schema])

  useEffect(() => {
    schemaRef.current = schema
    schemaIndexRef.current = schemaIndex
  }, [schema, schemaIndex])

  useEffect(() => {
    return () => {
      completionProviderRef.current?.dispose()
    }
  }, [])

  const handleEditorDidMount = (editor: unknown, monaco: unknown) => {
    editorRef.current = editor
    schemaRef.current = schema
    schemaIndexRef.current = schemaIndex
    completionProviderRef.current?.dispose()

    // Configure SQL language features
    monaco.languages.setLanguageConfiguration('sql', {
      comments: {
        lineComment: '--',
        blockComment: ['/*', '*/']
      },
      brackets: [
        ['{', '}'],
        ['[', ']'],
        ['(', ')']
      ],
      autoClosingPairs: [
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '(', close: ')' },
        { open: '"', close: '"' },
        { open: "'", close: "'" },
      ],
      surroundingPairs: [
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '(', close: ')' },
        { open: '"', close: '"' },
        { open: "'", close: "'" },
      ]
    })

    const keywordList = [
      'SELECT', 'FROM', 'WHERE', 'JOIN', 'INNER', 'LEFT', 'RIGHT', 'FULL', 'OUTER',
      'GROUP', 'BY', 'ORDER', 'HAVING', 'LIMIT', 'OFFSET', 'UNION', 'ALL',
      'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE', 'CREATE', 'TABLE',
      'DROP', 'ALTER', 'INDEX', 'VIEW', 'TRIGGER', 'PROCEDURE', 'FUNCTION',
      'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'BETWEEN', 'LIKE', 'ILIKE',
      'NULL', 'IS', 'TRUE', 'FALSE', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
      'DISTINCT', 'AS', 'ON', 'USING', 'ASC', 'DESC'
    ]

    // Add SQL keywords for better syntax highlighting
    monaco.languages.setMonarchTokensProvider('sql', {
      keywords: keywordList,
      operators: [
        '=', '!=', '<>', '<', '>', '<=', '>=', '+', '-', '*', '/', '%',
        '||', '&&', '!', '~', '&', '|', '^', '<<', '>>'
      ],
      tokenizer: {
        root: [
          [/[a-zA-Z_]\w*/, {
            cases: {
              '@keywords': 'keyword',
              '@default': 'identifier'
            }
          }],
          [/'([^'\\]|\\.)*$/, 'string.invalid'],
          [/'/, 'string', '@string'],
          [/"([^"\\]|\\.)*$/, 'string.invalid'],
          [/"/, 'string', '@string'],
          [/\d*\.\d+([eE][+]?\d+)?/, 'number.float'],
          [/\d+/, 'number'],
          [/[;,.]/, 'delimiter'],
          [/[(){}[\]]/, '@brackets'],
          [/[=!<>+\-*/%&|^~]+/, 'operator'],
          [/\s+/, 'white'],
          [/--.*$/, 'comment'],
          [/\/\*/, 'comment', '@comment']
        ],
        string: [
          [/[^\\']+/, 'string'],
          [/\\./, 'string.escape'],
          [/'/, 'string', '@pop']
        ],
        comment: [
          [/[^/*]+/, 'comment'],
          [/\*\//, 'comment', '@pop'],
          [/[/*]/, 'comment']
        ]
      }
    })

    // Set editor options
    editor.updateOptions({
      fontSize: 14,
      lineNumbers: 'on',
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      automaticLayout: true,
      folding: true,
      lineDecorationsWidth: 5,
      lineNumbersMinChars: 3,
      renderLineHighlight: 'all',
      selectionHighlight: false,
      wordWrap: 'on',
    })

    // Add keyboard shortcuts
    const triggerQuery = () => {
      void handleExecuteQuery()
    }

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, triggerQuery)
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.NumpadEnter, triggerQuery)

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      void handleSaveTab()
    })

    completionProviderRef.current = monaco.languages.registerCompletionItemProvider('sql', {
      triggerCharacters: ['.', ' ', '\n'],
      provideCompletionItems: (model: unknown, position: unknown, context: unknown) => {
        const schemaIndexSnapshot = schemaIndexRef.current
        const aliasMap = buildAliasMap(model.getValue(), schemaIndexSnapshot)
        const word = model.getWordUntilPosition(position)
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        }

        const textUntilCursor = getTextUntilPosition(model, position)
        const trimmedBeforeCursor = textUntilCursor.replace(/\s+$/, '')
        const prefixBeforeWord = getPrefixBeforeWord(model, position, word)
        const previousToken = getPreviousToken(prefixBeforeWord)
        const lastChar = trimmedBeforeCursor.slice(-1)
        const triggeredByDot = context?.triggerCharacter === '.' || lastChar === '.'

        const keywordSuggestions = keywordList.map((keyword) => ({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: keyword,
          detail: 'keyword',
          range,
          sortText: `0_${keyword}`,
        }))

        const tableSuggestions = schemaIndexSnapshot.tables.map((table) => ({
          label: table.label,
          kind: monaco.languages.CompletionItemKind.Class,
          insertText: table.insertText,
          detail: table.detail,
          range,
          sortText: `1_${table.label}`,
        }))

        const columnCompletionsForTable = (entry: TableEntry | undefined) => {
          const columns = entry ? (schemaIndexSnapshot.columnsByTable[entry.key] ?? []) : schemaIndexSnapshot.allColumns
          return columns.map((column) => ({
            label: column.label,
            kind: monaco.languages.CompletionItemKind.Property,
            insertText: column.insertText,
            detail: column.detail ?? `${column.table}.${column.column}`,
            range,
            sortText: `2_${column.table}_${column.label}`,
          }))
        }

        if (triggeredByDot) {
          const tableToken = getTokenBeforeDot(trimmedBeforeCursor)
          const entry = findTableEntry(tableToken, schemaIndexSnapshot, aliasMap)
          const columnSuggestions = columnCompletionsForTable(entry)
          return { suggestions: columnSuggestions }
        }

        const prevTokenLower = previousToken.toLowerCase()
        if (["from", "join", "update", "into", "table"].includes(prevTokenLower)) {
          return { suggestions: tableSuggestions }
        }

        const defaultColumnSuggestions = columnCompletionsForTable(undefined).slice(0, 100)
        const combinedSuggestions = [
          ...keywordSuggestions,
          ...tableSuggestions,
          ...defaultColumnSuggestions,
        ]

        return { suggestions: combinedSuggestions }
      },
    })
  }

  const handleEditorChange = (value: string | undefined) => {
    if (value !== undefined && activeTab) {
      setEditorContent(value)
      updateTab(activeTab.id, {
        content: value,
        isDirty: value !== activeTab.content
      })
    }
  }

  const handleExecuteQuery = async () => {
    if (!activeTab) return

    const currentEditorValue =
      typeof editorRef.current?.getValue === 'function'
        ? editorRef.current.getValue()
        : editorContent

    let queryText = currentEditorValue ?? ''
    const editorInstance = editorRef.current
    if (
      editorInstance &&
      typeof editorInstance.getModel === 'function' &&
      typeof editorInstance.getSelection === 'function'
    ) {
      const selection = editorInstance.getSelection()
      if (selection && !selection.isEmpty()) {
        const selectedText = editorInstance.getModel()?.getValueInRange(selection) ?? ''
        if (selectedText.trim()) {
          queryText = selectedText
        }
      }
    }

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

  const handleSaveTab = () => {
    if (activeTab) {
      updateTab(activeTab.id, { isDirty: false })
      // TODO: Implement actual save functionality
    }
  }

  const handleCreateTab = () => {
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
      const schema = activeConnection?.database || undefined
      const generatedSQL = await generateSQL(naturalLanguagePrompt, schema)

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
    } catch (error) {
      console.error('Failed to generate SQL:', error)
      setLastExecutionError(error instanceof Error ? error.message : 'Failed to generate SQL')
    }
  }

  const handleFixSQL = async () => {
    if (!lastExecutionError || !editorContent.trim() || !aiEnabled) return

    try {
      const schema = activeConnection?.database || undefined
      const fixedSQL = await fixSQL(editorContent, lastExecutionError, schema)

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

  return (
    <div className="flex-1 flex h-full min-h-0 w-full flex-col">
      {/* Tab Bar */}
      <div className="flex items-center border-b bg-background">
        <div className="flex-1 flex items-center overflow-x-auto">
          <Tabs value={activeTabId || ''} className="w-full">
            <TabsList className="h-10 bg-transparent border-0 rounded-none p-0">
              {tabs.map((tab) => (
                <div key={tab.id} className="flex items-center border-r">
                  <TabsTrigger
                    value={tab.id}
                    onClick={() => handleTabClick(tab.id)}
                    className={cn(
                      "h-10 px-3 rounded-none border-0 data-[state=active]:bg-background",
                      "data-[state=active]:border-b-2 data-[state=active]:border-primary",
                      "flex items-center space-x-2 min-w-[120px] max-w-[200px]"
                    )}
                  >
                    <span className="truncate">
                      {tab.title}
                      {tab.isDirty && <span className="ml-1">•</span>}
                    </span>
                  </TabsTrigger>
                  
                  {/* Connection Selector */}
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
                </div>
              ))}
            </TabsList>
          </Tabs>
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={handleCreateTab}
          className="ml-2 mr-4"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {/* AI Natural Language Input */}
      {aiEnabled && (
        <div className="p-3 border-b bg-blue-50 dark:bg-blue-950/20">
          <div className="flex items-center gap-2 mb-2">
            <Brain className="h-4 w-4 text-blue-600 dark:text-blue-400" />
            <span className="text-sm font-medium text-blue-800 dark:text-blue-200">
              AI Assistant - Describe your query in natural language
            </span>
          </div>
          <div className="flex gap-2">
            <Input
              placeholder="e.g., 'Show me all users who signed up last month with their total orders'"
              value={naturalLanguagePrompt}
              onChange={(e) => setNaturalLanguagePrompt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleGenerateSQL()
                }
              }}
              className="flex-1"
              disabled={isGenerating}
            />
            <Button
              size="sm"
              onClick={handleGenerateSQL}
              disabled={!naturalLanguagePrompt.trim() || isGenerating}
            >
              {isGenerating ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Wand2 className="h-4 w-4 mr-2" />
              )}
              {isGenerating ? 'Generating...' : 'Generate SQL'}
            </Button>
          </div>
          {lastError && (
            <div className="mt-2 flex items-center gap-2 text-sm text-red-600 dark:text-red-400">
              <AlertCircle className="h-4 w-4" />
              {lastError}
            </div>
          )}
        </div>
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

          <Button
            variant="outline"
            size="sm"
            onClick={handleSaveTab}
            disabled={!activeTab?.isDirty}
          >
            <Save className="h-4 w-4 mr-2" />
            Save
          </Button>

          {/* AI Suggestions Toggle */}
          {aiEnabled && suggestions.length > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAIPanel(!showAIPanel)}
              className="text-blue-600 hover:text-blue-700 border-blue-200 hover:border-blue-300"
            >
              <Lightbulb className="h-4 w-4 mr-2" />
              AI Suggestions ({suggestions.length})
            </Button>
          )}
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
              <Lightbulb className="h-4 w-4 text-green-600 dark:text-green-400" />
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
        <Editor
          height="100%"
          width="100%"
          defaultLanguage="sql"
          value={editorContent}
          theme={theme === 'dark' ? 'vs-dark' : 'light'}
          onChange={handleEditorChange}
          onMount={handleEditorDidMount}
          options={{
            fontSize: 14,
            lineNumbers: 'on',
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            automaticLayout: true,
            folding: true,
            wordWrap: 'on',
            renderLineHighlight: 'all',
            selectionHighlight: false,
          }}
        />
      </div>
    </div>
  )
}
