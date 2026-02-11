/**
 * .env File Import Dialog Component
 *
 * Multi-step dialog for importing database connections from .env files
 * with AI-powered extraction and user confirmation.
 */

import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Clipboard,
  Edit2,
  File,
  Loader2,
  SkipForward,
  Sparkles,
  Trash2,
  Upload,
} from 'lucide-react'
import { useCallback, useReducer, useState } from 'react'
import { useDropzone } from 'react-dropzone'

import { OpenEnvFileDialog } from '../../../../bindings/github.com/jbeck018/howlerops/app'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  applyFilenameEnvironment,
  convertToExportedConnection,
  detectDuplicates,
  envImportActions,
  envImportReducer,
  envImportSelectors,
  extractConnectionsWithAI,
  filterConnectionRelatedEntries,
  isAcceptedEnvFile,
  parseEnvFile,
  readEnvFile,
  validateParsedConnection,
  type ExistingConnection,
  type ParsedEnvConnection,
} from '@/lib/export-import/env-parser'
import { INITIAL_ENV_IMPORT_STATE } from '@/lib/export-import/env-parser/types'
import { importConnections } from '@/lib/export-import'
import { useAIGeneration, useAIConfig } from '@/store/ai-store'
import { useConnectionStore } from '@/store/connection-store'

// ============================================================================
// Types
// ============================================================================

interface EnvImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImportComplete?: () => void
}

// ============================================================================
// Connection Card Component
// ============================================================================

interface ConnectionCardProps {
  connection: ParsedEnvConnection
  onEdit: () => void
  onToggleSkip: () => void
  onRemove: () => void
  validation: ReturnType<typeof validateParsedConnection>
}

function ConnectionCard({
  connection,
  onEdit,
  onToggleSkip,
  onRemove,
  validation,
}: ConnectionCardProps) {
  const confidenceColors = {
    high: 'bg-green-500/10 text-green-600 border-green-500/20',
    medium: 'bg-amber-500/10 text-amber-600 border-amber-500/20',
    low: 'bg-red-500/10 text-red-600 border-red-500/20',
  }

  const environmentColors: Record<string, string> = {
    development: 'bg-blue-500/10 text-blue-600 border-blue-500/20',
    staging: 'bg-purple-500/10 text-purple-600 border-purple-500/20',
    production: 'bg-red-500/10 text-red-600 border-red-500/20',
    test: 'bg-gray-500/10 text-gray-600 border-gray-500/20',
    local: 'bg-green-500/10 text-green-600 border-green-500/20',
  }

  return (
    <div
      className={`rounded-lg border p-4 ${
        connection.isSkipped ? 'opacity-50 bg-muted/50' : ''
      } ${connection.duplicateOfId ? 'border-amber-500/50' : ''}`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium truncate">{connection.suggestedName}</span>
            <Badge variant="outline" className="text-xs">
              {connection.type}
            </Badge>
            {connection.detectedEnvironment && (
              <Badge
                variant="outline"
                className={`text-xs ${environmentColors[connection.detectedEnvironment] || 'bg-gray-500/10 text-gray-600'}`}
              >
                {connection.detectedEnvironment}
              </Badge>
            )}
            <Badge
              variant="outline"
              className={`text-xs ${confidenceColors[connection.overallConfidence]}`}
            >
              {connection.overallConfidence} confidence
            </Badge>
          </div>

          <div className="text-sm text-muted-foreground mt-1">
            {connection.host && `${connection.host}`}
            {connection.port && `:${connection.port}`}
            {connection.database && ` / ${connection.database}`}
          </div>

          {/* Extraction notes */}
          {connection.extractionNotes && (
            <div className="text-xs text-muted-foreground mt-1 italic">
              {connection.extractionNotes}
            </div>
          )}

          {/* Duplicate warning */}
          {connection.duplicateOfId && !connection.isSkipped && (
            <div className="mt-2 text-xs text-amber-600 flex items-center gap-1">
              <AlertTriangle className="h-3 w-3" />
              Possible duplicate of "{connection.duplicateOfName}"
            </div>
          )}

          {/* Validation errors */}
          {!connection.isSkipped && validation.errors.length > 0 && (
            <div className="mt-2 text-xs text-destructive">
              {validation.errors.map((err, i) => (
                <div key={i} className="flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />
                  {err}
                </div>
              ))}
            </div>
          )}

          {/* Validation warnings */}
          {!connection.isSkipped && validation.warnings.length > 0 && (
            <div className="mt-2 text-xs text-amber-600">
              {validation.warnings.map((warn, i) => (
                <div key={i} className="flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3" />
                  {warn}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onEdit}
            disabled={connection.isSkipped}
            title="Edit connection"
          >
            <Edit2 className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onToggleSkip}
            title={connection.isSkipped ? 'Include connection' : 'Skip connection'}
          >
            {connection.isSkipped ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <SkipForward className="h-4 w-4" />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
            onClick={onRemove}
            title="Remove from list"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// Connection Edit Form
// ============================================================================

interface ConnectionEditFormProps {
  connection: ParsedEnvConnection
  onSave: (updates: Partial<ParsedEnvConnection>) => void
  onCancel: () => void
}

function ConnectionEditForm({
  connection,
  onSave,
  onCancel,
}: ConnectionEditFormProps) {
  const [name, setName] = useState(connection.suggestedName)
  const [host, setHost] = useState(connection.host || '')
  const [port, setPort] = useState(connection.port?.toString() || '')
  const [database, setDatabase] = useState(connection.database || '')
  const [username, setUsername] = useState(connection.username || '')
  const [password, setPassword] = useState(connection.password || '')
  const [type, setType] = useState(connection.type)
  const [environment, setEnvironment] = useState(connection.detectedEnvironment || 'development')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSave({
      suggestedName: name,
      host,
      port: port ? parseInt(port, 10) : undefined,
      database,
      username,
      password,
      type,
      detectedEnvironment: environment,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="col-span-2">
          <Label htmlFor="name">Connection Name</Label>
          <Input
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Database"
          />
        </div>

        <div>
          <Label htmlFor="type">Database Type</Label>
          <Select value={type} onValueChange={(v) => setType(v as typeof type)}>
            <SelectTrigger id="type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="postgresql">PostgreSQL</SelectItem>
              <SelectItem value="mysql">MySQL</SelectItem>
              <SelectItem value="mariadb">MariaDB</SelectItem>
              <SelectItem value="mongodb">MongoDB</SelectItem>
              <SelectItem value="redis">Redis</SelectItem>
              <SelectItem value="elasticsearch">Elasticsearch</SelectItem>
              <SelectItem value="opensearch">OpenSearch</SelectItem>
              <SelectItem value="clickhouse">ClickHouse</SelectItem>
              <SelectItem value="mssql">SQL Server</SelectItem>
              <SelectItem value="sqlite">SQLite</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div>
          <Label htmlFor="environment">Environment</Label>
          <Select value={environment} onValueChange={setEnvironment}>
            <SelectTrigger id="environment">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="development">Development</SelectItem>
              <SelectItem value="staging">Staging</SelectItem>
              <SelectItem value="production">Production</SelectItem>
              <SelectItem value="test">Test</SelectItem>
              <SelectItem value="local">Local</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div>
          <Label htmlFor="host">Host</Label>
          <Input
            id="host"
            value={host}
            onChange={(e) => setHost(e.target.value)}
            placeholder="localhost"
          />
        </div>

        <div>
          <Label htmlFor="port">Port</Label>
          <Input
            id="port"
            type="number"
            value={port}
            onChange={(e) => setPort(e.target.value)}
            placeholder="5432"
          />
        </div>

        <div>
          <Label htmlFor="database">Database</Label>
          <Input
            id="database"
            value={database}
            onChange={(e) => setDatabase(e.target.value)}
            placeholder="mydb"
          />
        </div>

        <div>
          <Label htmlFor="username">Username</Label>
          <Input
            id="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="user"
          />
        </div>

        <div>
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
          />
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Changes</Button>
      </div>
    </form>
  )
}

// ============================================================================
// Main Dialog Component
// ============================================================================

export function EnvImportDialog({
  open,
  onOpenChange,
  onImportComplete,
}: EnvImportDialogProps) {
  const [state, dispatch] = useReducer(envImportReducer, INITIAL_ENV_IMPORT_STATE)

  // State for paste content option
  const [pasteContent, setPasteContent] = useState('')
  const [pasteFilename, setPasteFilename] = useState('.env')

  // Debug: Log render
  console.log('[EnvImportDialog] Render - open:', open, 'step:', state.step)

  // AI integration
  const { sendGenericMessage, isGenerating, lastError: aiLastError } = useAIGeneration()
  const { isEnabled: isAIEnabled } = useAIConfig()

  // Get existing connections for duplicate detection
  const existingConnections = useConnectionStore((state) => state.connections)

  // Process file content with filename
  const processFileContent = useCallback(
    async (content: string, filename: string) => {
      // Create a minimal file-like object for state
      dispatch(envImportActions.setFile({ name: filename } as File))

      try {
        // Parse .env file
        const parseResult = parseEnvFile(content)
        dispatch(envImportActions.setParseResult(parseResult))

        // Filter connection-related entries
        const connectionEntries = filterConnectionRelatedEntries(parseResult.entries)

        if (connectionEntries.length === 0) {
          dispatch(
            envImportActions.setError(
              'No database connection variables found in this file'
            )
          )
          return
        }

        // Check if AI is enabled
        if (!isAIEnabled) {
          dispatch(
            envImportActions.setError(
              'AI is not enabled. Please enable AI in Settings > AI to use .env import.'
            )
          )
          return
        }

        // Extract connections using AI
        const extractionResult = await extractConnectionsWithAI(connectionEntries, {
          sendMessage: async (prompt, options) => {
            const response = await sendGenericMessage(prompt, {
              systemPrompt: options?.systemPrompt,
              context: options?.context,
            })
            return response?.content || ''
          },
        })

        // Apply environment from filename if not detected from variables
        let connections = applyFilenameEnvironment(
          extractionResult.connections,
          filename
        )

        // Check for duplicates against existing connections
        const existingForCheck: ExistingConnection[] = existingConnections.map((c) => ({
          id: c.id,
          name: c.name,
          type: c.type,
          host: c.host,
          port: c.port,
          database: c.database,
          environments: c.environments,
        }))

        const duplicateResult = detectDuplicates(connections, existingForCheck)
        connections = duplicateResult.connections

        // Update the extraction result with enhanced connections
        const enhancedResult = {
          ...extractionResult,
          connections,
        }

        dispatch(envImportActions.setExtractionResult(enhancedResult))
      } catch (err) {
        console.error('[EnvImport] Error in processFileContent:', err)
        let message = err instanceof Error ? err.message : 'Failed to process file'

        // Check for AI-specific errors
        if (message.includes('AI') || message.includes('extraction')) {
          message = `AI extraction failed: ${message}. Make sure AI is configured in Settings > AI.`
        }

        dispatch(envImportActions.setError(message))
      }
    },
    [sendGenericMessage, existingConnections, isAIEnabled]
  )

  // Log AI errors for debugging
  if (aiLastError) {
    console.error('[EnvImport] AI last error:', aiLastError)
  }

  // Handle dropped files via react-dropzone
  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    console.log('[EnvImport] onDrop called, files:', acceptedFiles.length)
    if (acceptedFiles.length === 0) return

    const file = acceptedFiles[0]
    console.log('[EnvImport] Processing file:', file.name)

    if (!isAcceptedEnvFile(file.name)) {
      dispatch(envImportActions.setError('Please select a .env file'))
      return
    }

    dispatch(envImportActions.setStep('parsing'))

    try {
      const content = await readEnvFile(file)
      console.log('[EnvImport] File content length:', content?.length)
      await processFileContent(content, file.name)
    } catch (err) {
      console.error('[EnvImport] Error reading file:', err)
      const message = err instanceof Error ? err.message : 'Failed to read file'
      dispatch(envImportActions.setError(message))
    }
  }, [processFileContent])

  // React dropzone hook
  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    noClick: true, // We'll handle click separately for native dialog
    noKeyboard: true,
  })

  // Open native file dialog via Wails
  const handleBrowseFiles = useCallback(async () => {
    console.log('[EnvImport] handleBrowseFiles called')
    try {
      const result = await OpenEnvFileDialog()
      console.log('[EnvImport] OpenEnvFileDialog result:', result)

      if (!result || !result.path) {
        console.log('[EnvImport] No result or no path, user cancelled')
        return
      }

      const { path: filePath, content } = result
      const filename = filePath.split(/[/\\]/).pop() || 'unknown.env'

      dispatch(envImportActions.setStep('parsing'))
      await processFileContent(content, filename)
    } catch (err) {
      console.error('[EnvImport] Error:', err)
      const message = err instanceof Error ? err.message : 'Failed to open file'
      dispatch(envImportActions.setError(message))
    }
  }, [processFileContent])

  // Handle paste content submission
  const handlePasteSubmit = useCallback(async () => {
    if (!pasteContent.trim()) {
      dispatch(envImportActions.setError('Please paste .env content first'))
      return
    }

    dispatch(envImportActions.setStep('parsing'))

    try {
      await processFileContent(pasteContent, pasteFilename || '.env')
    } catch (err) {
      console.error('[EnvImport] Error processing pasted content:', err)
      const message = err instanceof Error ? err.message : 'Failed to process content'
      dispatch(envImportActions.setError(message))
    }
  }, [pasteContent, pasteFilename, processFileContent])

  // Import connections
  const handleImport = useCallback(async () => {
    const toImport = envImportSelectors.getConnectionsToImport(state)
    if (toImport.length === 0) return

    dispatch(envImportActions.setStep('importing'))

    try {
      // Convert to export format and import
      const exportedConnections = toImport.map(convertToExportedConnection)

      const result = await importConnections(
        {
          version: '1.0',
          exportedAt: new Date().toISOString(),
          connections: exportedConnections,
          metadata: {
            source: 'env-import',
            exportedAt: new Date().toISOString(),
            includesPasswords: true,
          },
        },
        { conflictResolution: state.importOptions.conflictResolution }
      )

      dispatch(
        envImportActions.setImportResult({
          imported: result.imported,
          skipped: result.skipped,
          failed: result.failed.map((f) => ({
            tempId: '',
            suggestedName: f.connectionName,
            reason: f.reason,
          })),
          importedConnectionIds: [],
        })
      )

      if (result.imported > 0 || result.overwritten > 0) {
        onImportComplete?.()
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Import failed'
      dispatch(envImportActions.setError(message))
    }
  }, [state, onImportComplete])

  // Close dialog
  const handleClose = useCallback(() => {
    if (state.step !== 'parsing' && state.step !== 'importing') {
      onOpenChange(false)
      // Reset state after animation
      setTimeout(() => {
        dispatch(envImportActions.reset())
        setPasteContent('')
        setPasteFilename('.env')
      }, 200)
    }
  }, [state.step, onOpenChange])

  // Get editing connection
  const editingConnection = envImportSelectors.getEditingConnection(state)
  const counts = envImportSelectors.getConnectionCounts(state)
  const canImport = envImportSelectors.canProceedToImport(state)

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {state.step === 'parsing' ? (
              <Loader2 className="h-5 w-5 animate-spin" />
            ) : (
              <Sparkles className="h-5 w-5" />
            )}
            Import from .env File
          </DialogTitle>
          <DialogDescription>
            {state.step === 'file-select' &&
              'Drop a .env file to automatically detect and import database connections.'}
            {state.step === 'parsing' &&
              'Analyzing file for database connections...'}
            {state.step === 'preview' &&
              `Found ${counts.total} connection(s). Review and edit before importing.`}
            {state.step === 'editing' && 'Edit connection details'}
            {state.step === 'importing' && 'Importing connections...'}
            {state.step === 'complete' && 'Import completed'}
            {state.step === 'error' && 'An error occurred'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-hidden py-4">
          {/* File Select Step */}
          {state.step === 'file-select' && (
            <Tabs defaultValue="upload" className="w-full">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="upload" className="flex items-center gap-2">
                  <Upload className="h-4 w-4" />
                  Upload File
                </TabsTrigger>
                <TabsTrigger value="paste" className="flex items-center gap-2">
                  <Clipboard className="h-4 w-4" />
                  Paste Content
                </TabsTrigger>
              </TabsList>

              {/* Upload Tab */}
              <TabsContent value="upload" className="mt-4">
                <div
                  {...getRootProps()}
                  onClick={handleBrowseFiles}
                  className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
                    isDragActive
                      ? 'border-primary bg-primary/5'
                      : 'border-muted-foreground/25 hover:border-muted-foreground/50 hover:bg-muted/30'
                  }`}
                >
                  <input {...getInputProps()} />
                  <File className="mx-auto h-10 w-10 text-muted-foreground mb-3" />
                  <p className="text-sm font-medium">Drop .env file here or click to browse</p>
                  <p className="text-xs text-muted-foreground mt-2">
                    Supports .env, .env.local, .env.development, .env.production, etc.
                  </p>
                  <div className="mt-4 flex items-center justify-center gap-2 text-xs text-muted-foreground">
                    <Sparkles className="h-3 w-3" />
                    AI will automatically detect database connections
                  </div>
                </div>
              </TabsContent>

              {/* Paste Tab */}
              <TabsContent value="paste" className="mt-4 space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="env-content">Environment Variables</Label>
                  <Textarea
                    id="env-content"
                    placeholder={'Paste your .env file content here...\n\nDATABASE_URL=postgresql://user:pass@localhost:5432/mydb\nREDIS_HOST=localhost\nREDIS_PORT=6379\n...'}
                    value={pasteContent}
                    onChange={(e) => setPasteContent(e.target.value)}
                    className="min-h-[200px] font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Paste your .env file content. The AI will automatically detect database connections.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="env-filename">Filename (optional)</Label>
                  <Input
                    id="env-filename"
                    placeholder=".env"
                    value={pasteFilename}
                    onChange={(e) => setPasteFilename(e.target.value)}
                    className="font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Used for environment detection (e.g., .env.production)
                  </p>
                </div>

                <div className="flex items-center gap-2 text-xs text-muted-foreground pt-2 border-t">
                  <Sparkles className="h-3 w-3" />
                  AI will analyze the content and extract database connections
                </div>

                <Button
                  onClick={handlePasteSubmit}
                  disabled={!pasteContent.trim()}
                  className="w-full"
                >
                  <Sparkles className="mr-2 h-4 w-4" />
                  Process Content
                </Button>
              </TabsContent>
            </Tabs>
          )}

          {/* Parsing Step */}
          {state.step === 'parsing' && (
            <div className="flex flex-col items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary mb-4" />
              <p className="text-sm font-medium">Analyzing .env file...</p>
              <p className="text-xs text-muted-foreground mt-1">
                {isGenerating
                  ? 'AI is extracting database connections'
                  : 'Parsing environment variables'}
              </p>
            </div>
          )}

          {/* Preview Step */}
          {state.step === 'preview' && state.extractionResult && (
            <ScrollArea className="h-[400px] pr-4">
              <div className="space-y-3">
                {/* Duplicate warning summary */}
                {state.extractionResult.connections.some((c) => c.duplicateOfId) && (
                  <Alert className="bg-amber-500/10 border-amber-500/20">
                    <AlertTriangle className="h-4 w-4 text-amber-600" />
                    <AlertDescription className="text-amber-600">
                      {state.extractionResult.connections.filter((c) => c.duplicateOfId).length} connection(s)
                      may already exist. Consider skipping duplicates or they will be imported as new connections.
                    </AlertDescription>
                  </Alert>
                )}

                {state.extractionResult.connections.map((conn) => (
                  <ConnectionCard
                    key={conn.tempId}
                    connection={conn}
                    validation={validateParsedConnection(conn)}
                    onEdit={() => dispatch(envImportActions.startEditing(conn.tempId))}
                    onToggleSkip={() =>
                      dispatch(envImportActions.toggleSkip(conn.tempId))
                    }
                    onRemove={() =>
                      dispatch(envImportActions.removeConnection(conn.tempId))
                    }
                  />
                ))}

                {/* Unused entries */}
                {state.extractionResult.unusedEntries.length > 0 && (
                  <>
                    <Separator className="my-4" />
                    <div className="text-sm text-muted-foreground">
                      <span className="font-medium">
                        {state.extractionResult.unusedEntries.length} environment
                        variable(s) not matched:
                      </span>
                      <div className="mt-1 text-xs font-mono bg-muted/50 rounded p-2 max-h-24 overflow-y-auto">
                        {state.extractionResult.unusedEntries.map((e) => (
                          <div key={e.lineNumber}>{e.key}</div>
                        ))}
                      </div>
                    </div>
                  </>
                )}
              </div>
            </ScrollArea>
          )}

          {/* Editing Step */}
          {state.step === 'editing' && editingConnection && (
            <ConnectionEditForm
              connection={editingConnection}
              onSave={(updates) => {
                dispatch(
                  envImportActions.updateConnection(editingConnection.tempId, updates)
                )
                dispatch(envImportActions.stopEditing())
              }}
              onCancel={() => dispatch(envImportActions.stopEditing())}
            />
          )}

          {/* Importing Step */}
          {state.step === 'importing' && (
            <div className="flex flex-col items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary mb-4" />
              <p className="text-sm font-medium">Importing connections...</p>
            </div>
          )}

          {/* Complete Step */}
          {state.step === 'complete' && state.importResult && (
            <div className="space-y-4">
              <Alert variant="default" className="bg-green-500/10 border-green-500/20">
                <CheckCircle2 className="h-4 w-4 text-green-500" />
                <AlertDescription>
                  <div className="font-medium">Import Complete</div>
                  <ul className="mt-1 text-sm space-y-0.5">
                    {state.importResult.imported > 0 && (
                      <li>{state.importResult.imported} connection(s) imported</li>
                    )}
                    {state.importResult.skipped > 0 && (
                      <li>{state.importResult.skipped} connection(s) skipped</li>
                    )}
                    {state.importResult.failed.length > 0 && (
                      <li className="text-destructive">
                        {state.importResult.failed.length} connection(s) failed
                      </li>
                    )}
                  </ul>
                </AlertDescription>
              </Alert>

              {state.importResult.failed.length > 0 && (
                <Alert variant="destructive" className="bg-destructive/10">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    <div className="font-medium">Failed Imports:</div>
                    <ul className="mt-1 text-sm space-y-0.5">
                      {state.importResult.failed.map((f, i) => (
                        <li key={i}>
                          {f.suggestedName}: {f.reason}
                        </li>
                      ))}
                    </ul>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}

          {/* Error Step */}
          {state.step === 'error' && (state.error || aiLastError) && (
            <div className="space-y-3">
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <div className="font-medium">{state.error || 'An error occurred'}</div>
                  {aiLastError && aiLastError !== state.error && (
                    <div className="mt-1 text-sm opacity-80">
                      AI Error: {aiLastError}
                    </div>
                  )}
                </AlertDescription>
              </Alert>
              <p className="text-sm text-muted-foreground">
                Make sure AI is configured in Settings {'>'} AI before importing .env files.
                You can also try the "Paste Content" tab to manually enter your variables.
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          {state.step === 'error' && (
            <Button
              variant="outline"
              onClick={() => dispatch(envImportActions.reset())}
            >
              Try Again
            </Button>
          )}

          <Button
            variant="outline"
            onClick={handleClose}
            disabled={state.step === 'parsing' || state.step === 'importing'}
          >
            {state.step === 'complete' ? 'Close' : 'Cancel'}
          </Button>

          {state.step === 'preview' && (
            <Button onClick={handleImport} disabled={!canImport}>
              <Upload className="mr-2 h-4 w-4" />
              Import {counts.toImport} Connection{counts.toImport !== 1 ? 's' : ''}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
