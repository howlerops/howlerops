import { useState, useCallback } from 'react'
import { useDropzone } from 'react-dropzone'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Upload,
  FileJson,
  Database,
  FileText,
  AlertTriangle,
  CheckCircle2,
  Loader2,
  Key,
  Tag,
  FolderOpen,
  Lock,
  Eye,
  EyeOff,
  ShieldCheck,
} from 'lucide-react'
import {
  type ExportedConfig,
  type EncryptedConfigExport,
  type ConflictStrategy,
  type ImportResult,
  readConfigFile,
  readEncryptedConfigFile,
  isEncryptedExport,
  validateConfig,
  previewImport,
  importConfig,
  importEncryptedConfig,
} from '@/lib/api/config-export'

interface ConfigImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImportComplete?: (result: ImportResult) => void
}

type Step = 'upload' | 'passphrase' | 'preview' | 'importing' | 'complete'

export function ConfigImportDialog({
  open,
  onOpenChange,
  onImportComplete,
}: ConfigImportDialogProps) {
  const [step, setStep] = useState<Step>('upload')
  const [config, setConfig] = useState<ExportedConfig | null>(null)
  const [preview, setPreview] = useState<ImportResult | null>(null)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  // Encrypted import state
  const [isEncrypted, setIsEncrypted] = useState(false)
  const [encryptedConfig, setEncryptedConfig] = useState<EncryptedConfigExport | null>(null)
  const [passphrase, setPassphrase] = useState('')
  const [showPassphrase, setShowPassphrase] = useState(false)
  const [rawFileContent, setRawFileContent] = useState<string>('')

  // Import options
  const [importConnections, setImportConnections] = useState(true)
  const [importQueries, setImportQueries] = useState(true)
  const [conflictStrategy, setConflictStrategy] = useState<ConflictStrategy>('skip')

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    if (acceptedFiles.length === 0) return

    setIsLoading(true)
    setError(null)

    try {
      const file = acceptedFiles[0]

      // Read file content to check if encrypted
      const fileContent = await file.text()
      setRawFileContent(fileContent)

      if (isEncryptedExport(fileContent)) {
        // Handle encrypted file - need passphrase
        const encrypted = await readEncryptedConfigFile(file)
        setEncryptedConfig(encrypted)
        setIsEncrypted(true)
        setStep('passphrase')
        return
      }

      // Handle regular (non-encrypted) file
      const parsedConfig = await readConfigFile(file)

      // Validate the config
      const validation = await validateConfig(parsedConfig)
      if (!validation.valid) {
        setError(`Invalid config: ${validation.issues.join(', ')}`)
        return
      }

      setConfig(parsedConfig)
      setIsEncrypted(false)

      // Get preview
      const previewResult = await previewImport(parsedConfig, {
        import_connections: importConnections,
        import_saved_queries: importQueries,
        conflict_strategy: conflictStrategy,
      })
      setPreview(previewResult)
      setStep('preview')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to parse config file')
    } finally {
      setIsLoading(false)
    }
  }, [importConnections, importQueries, conflictStrategy])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/json': ['.json'],
    },
    maxFiles: 1,
  })

  const handleImport = async () => {
    setIsLoading(true)
    setStep('importing')
    setError(null)

    try {
      let importResult: ImportResult

      if (isEncrypted && encryptedConfig) {
        // Import encrypted config with passphrase
        importResult = await importEncryptedConfig(encryptedConfig, passphrase, {
          import_connections: importConnections,
          import_saved_queries: importQueries,
          conflict_strategy: conflictStrategy,
        })
      } else if (config) {
        // Import regular config
        importResult = await importConfig(config, {
          import_connections: importConnections,
          import_saved_queries: importQueries,
          conflict_strategy: conflictStrategy,
        })
      } else {
        throw new Error('No config to import')
      }

      setResult(importResult)
      setStep('complete')
      onImportComplete?.(importResult)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed')
      setStep('preview')
    } finally {
      setIsLoading(false)
    }
  }

  // Handle passphrase submission for encrypted imports
  const handleDecryptAndPreview = async () => {
    if (!encryptedConfig || !passphrase) return

    setIsLoading(true)
    setError(null)

    try {
      // For encrypted imports, we skip the standard preview flow
      // The backend will validate during import
      // Just move to preview step showing the hint data
      setStep('preview')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to process encrypted config')
    } finally {
      setIsLoading(false)
    }
  }

  const handleClose = () => {
    setStep('upload')
    setConfig(null)
    setPreview(null)
    setResult(null)
    setError(null)
    setIsEncrypted(false)
    setEncryptedConfig(null)
    setPassphrase('')
    setRawFileContent('')
    onOpenChange(false)
  }

  const renderUploadStep = () => (
    <>
      <div
        {...getRootProps()}
        className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
          isDragActive
            ? 'border-primary bg-primary/5'
            : 'border-muted-foreground/25 hover:border-primary/50'
        }`}
      >
        <input {...getInputProps()} />
        <FileJson className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
        {isDragActive ? (
          <p className="text-sm text-muted-foreground">Drop the config file here...</p>
        ) : (
          <>
            <p className="text-sm font-medium mb-1">
              Drag & drop a config file here, or click to select
            </p>
            <p className="text-xs text-muted-foreground">
              Only .json files exported from HowlerOps are supported
            </p>
          </>
        )}
      </div>

      <Separator />

      <div className="space-y-3">
        <Label className="text-sm font-medium">Import Options</Label>

        <div className="flex items-center space-x-2">
          <Checkbox
            id="import_connections"
            checked={importConnections}
            onCheckedChange={(checked) => setImportConnections(checked === true)}
          />
          <Label htmlFor="import_connections" className="flex items-center gap-2 cursor-pointer">
            <Database className="h-4 w-4 text-muted-foreground" />
            Import Connections
          </Label>
        </div>

        <div className="flex items-center space-x-2">
          <Checkbox
            id="import_queries"
            checked={importQueries}
            onCheckedChange={(checked) => setImportQueries(checked === true)}
          />
          <Label htmlFor="import_queries" className="flex items-center gap-2 cursor-pointer">
            <FileText className="h-4 w-4 text-muted-foreground" />
            Import Saved Queries
          </Label>
        </div>

        <div className="space-y-2">
          <Label className="text-sm">Conflict Handling</Label>
          <Select value={conflictStrategy} onValueChange={(v) => setConflictStrategy(v as ConflictStrategy)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="skip">Skip existing items</SelectItem>
              <SelectItem value="overwrite">Overwrite existing items</SelectItem>
              <SelectItem value="rename">Rename duplicates</SelectItem>
              <SelectItem value="merge">Merge metadata & tags</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </>
  )

  const renderPassphraseStep = () => (
    <>
      {encryptedConfig && (
        <div className="space-y-4">
          <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/20">
            <ShieldCheck className="h-4 w-4 text-blue-600" />
            <AlertTitle className="text-blue-800 dark:text-blue-200">Encrypted Export Detected</AlertTitle>
            <AlertDescription className="text-blue-700 dark:text-blue-300">
              This export contains encrypted passwords. Enter the passphrase used during export to decrypt.
            </AlertDescription>
          </Alert>

          {/* Show hint data */}
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div className="p-3 bg-muted rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <Database className="h-4 w-4" />
                <span className="font-medium">Connections</span>
              </div>
              <div className="text-2xl font-bold">{encryptedConfig.hint.connection_count}</div>
            </div>
            <div className="p-3 bg-muted rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <FileText className="h-4 w-4" />
                <span className="font-medium">Queries</span>
              </div>
              <div className="text-2xl font-bold">{encryptedConfig.hint.query_count}</div>
            </div>
          </div>

          {encryptedConfig.hint.database_types && encryptedConfig.hint.database_types.length > 0 && (
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm text-muted-foreground">Database types:</span>
              {encryptedConfig.hint.database_types.map((type) => (
                <Badge key={type} variant="secondary">{type}</Badge>
              ))}
            </div>
          )}

          <Separator />

          <div className="space-y-2">
            <Label htmlFor="import_passphrase" className="text-sm font-medium">
              Decryption Passphrase
            </Label>
            <div className="relative">
              <Input
                id="import_passphrase"
                type={showPassphrase ? 'text' : 'password'}
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                placeholder="Enter the passphrase used during export"
                className="pr-10"
                autoFocus
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                onClick={() => setShowPassphrase(!showPassphrase)}
              >
                {showPassphrase ? (
                  <EyeOff className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <Eye className="h-4 w-4 text-muted-foreground" />
                )}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Exported by {encryptedConfig.hint.exported_by || 'unknown'} on{' '}
              {new Date(encryptedConfig.created_at).toLocaleDateString()}
            </p>
          </div>
        </div>
      )}
    </>
  )

  const renderPreviewStep = () => {
    // For encrypted imports, show different preview based on hint data
    if (isEncrypted && encryptedConfig) {
      return (
        <div className="space-y-4">
          <Alert className="border-green-200 bg-green-50 dark:bg-green-950/20">
            <Lock className="h-4 w-4 text-green-600" />
            <AlertTitle className="text-green-800 dark:text-green-200">Encrypted Import</AlertTitle>
            <AlertDescription className="text-green-700 dark:text-green-300">
              Passwords will be decrypted and securely stored in your account.
            </AlertDescription>
          </Alert>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div className="p-3 bg-muted rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <Database className="h-4 w-4" />
                <span className="font-medium">Connections</span>
              </div>
              <div className="text-2xl font-bold">{encryptedConfig.hint.connection_count}</div>
            </div>
            <div className="p-3 bg-muted rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <FileText className="h-4 w-4" />
                <span className="font-medium">Queries</span>
              </div>
              <div className="text-2xl font-bold">{encryptedConfig.hint.query_count}</div>
            </div>
          </div>

          {encryptedConfig.hint.database_types && encryptedConfig.hint.database_types.length > 0 && (
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm text-muted-foreground">Database types:</span>
              {encryptedConfig.hint.database_types.map((type) => (
                <Badge key={type} variant="secondary">{type}</Badge>
              ))}
            </div>
          )}

          <Separator />

          <div className="space-y-3">
            <Label className="text-sm font-medium">Import Options</Label>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="import_connections_preview"
                checked={importConnections}
                onCheckedChange={(checked) => setImportConnections(checked === true)}
              />
              <Label htmlFor="import_connections_preview" className="flex items-center gap-2 cursor-pointer">
                <Database className="h-4 w-4 text-muted-foreground" />
                Import Connections (with passwords)
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="import_queries_preview"
                checked={importQueries}
                onCheckedChange={(checked) => setImportQueries(checked === true)}
              />
              <Label htmlFor="import_queries_preview" className="flex items-center gap-2 cursor-pointer">
                <FileText className="h-4 w-4 text-muted-foreground" />
                Import Saved Queries
              </Label>
            </div>

            <div className="space-y-2">
              <Label className="text-sm">Conflict Handling</Label>
              <Select value={conflictStrategy} onValueChange={(v) => setConflictStrategy(v as ConflictStrategy)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="skip">Skip existing items</SelectItem>
                  <SelectItem value="overwrite">Overwrite existing items</SelectItem>
                  <SelectItem value="rename">Rename duplicates</SelectItem>
                  <SelectItem value="merge">Merge metadata & tags</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <p className="text-xs text-muted-foreground">
            Exported by {encryptedConfig.hint.exported_by || 'unknown'} on{' '}
            {new Date(encryptedConfig.created_at).toLocaleDateString()}
          </p>
        </div>
      )
    }

    // Regular (non-encrypted) preview
    return (
      <>
        {config && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div className="p-3 bg-muted rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <Database className="h-4 w-4" />
                  <span className="font-medium">Connections</span>
                </div>
                <div className="text-2xl font-bold">{config.connections?.length || 0}</div>
                {preview && preview.connections_skipped > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {preview.connections_skipped} will be skipped
                  </p>
                )}
              </div>
              <div className="p-3 bg-muted rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <FileText className="h-4 w-4" />
                  <span className="font-medium">Queries</span>
                </div>
                <div className="text-2xl font-bold">{config.saved_queries?.length || 0}</div>
                {preview && preview.queries_skipped > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {preview.queries_skipped} will be skipped
                  </p>
                )}
              </div>
            </div>

            {config.tags && config.tags.length > 0 && (
              <div className="flex items-center gap-2 flex-wrap">
                <Tag className="h-4 w-4 text-muted-foreground" />
                {config.tags.map((tag) => (
                  <Badge key={tag} variant="secondary">{tag}</Badge>
                ))}
              </div>
            )}

            {config.folders && config.folders.length > 0 && (
              <div className="flex items-center gap-2 flex-wrap">
                <FolderOpen className="h-4 w-4 text-muted-foreground" />
                {config.folders.map((folder) => (
                  <Badge key={folder} variant="outline">{folder}</Badge>
                ))}
              </div>
            )}

            <Separator />

            {/* Connections needing passwords */}
            {preview && preview.connections_needing_passwords.length > 0 && (
              <Alert>
                <Key className="h-4 w-4" />
                <AlertTitle>Passwords Required</AlertTitle>
                <AlertDescription>
                  {preview.connections_needing_passwords.length} connection(s) will need passwords
                  set after import.
                </AlertDescription>
              </Alert>
            )}

            {/* Warnings */}
            {preview?.warnings && preview.warnings.length > 0 && (
              <Alert variant="default">
                <AlertTriangle className="h-4 w-4" />
                <AlertTitle>Warnings</AlertTitle>
                <AlertDescription>
                  <ScrollArea className="h-20">
                    <ul className="list-disc list-inside text-sm">
                      {preview.warnings.map((warning, i) => (
                        <li key={i}>{warning}</li>
                      ))}
                    </ul>
                  </ScrollArea>
                </AlertDescription>
              </Alert>
            )}

            <p className="text-xs text-muted-foreground">
              Exported by {config.exported_by || 'unknown'} on{' '}
              {new Date(config.exported_at).toLocaleDateString()}
            </p>
          </div>
        )}
      </>
    )
  }

  const renderCompleteStep = () => (
    <div className="text-center space-y-4">
      <CheckCircle2 className="h-16 w-16 mx-auto text-green-500" />
      <div>
        <h3 className="text-lg font-medium">Import Complete!</h3>
        <p className="text-sm text-muted-foreground mt-1">
          Imported {result?.connections_imported || 0} connections and{' '}
          {result?.queries_imported || 0} queries.
        </p>
      </div>

      {result?.connections_needing_passwords && result.connections_needing_passwords.length > 0 && (
        <Alert>
          <Key className="h-4 w-4" />
          <AlertTitle>Set Passwords</AlertTitle>
          <AlertDescription>
            <p className="mb-2">The following connections need passwords:</p>
            <ul className="list-disc list-inside text-sm">
              {result.connections_needing_passwords.map((conn) => (
                <li key={conn.new_connection_id}>
                  {conn.name} ({conn.host})
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}
    </div>
  )

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {isEncrypted ? <Lock className="h-5 w-5" /> : <Upload className="h-5 w-5" />}
            {step === 'upload' && 'Import Configuration'}
            {step === 'passphrase' && 'Enter Passphrase'}
            {step === 'preview' && (isEncrypted ? 'Review Encrypted Import' : 'Review Import')}
            {step === 'importing' && (isEncrypted ? 'Decrypting & Importing...' : 'Importing...')}
            {step === 'complete' && 'Import Complete'}
          </DialogTitle>
          <DialogDescription>
            {step === 'upload' && 'Upload a previously exported configuration file.'}
            {step === 'passphrase' && 'Enter the passphrase used when creating this encrypted export.'}
            {step === 'preview' && (isEncrypted
              ? 'Review the encrypted configuration before importing with passwords.'
              : 'Review what will be imported before confirming.'
            )}
            {step === 'importing' && (isEncrypted
              ? 'Decrypting passwords and importing your configuration...'
              : 'Please wait while your configuration is imported.'
            )}
            {step === 'complete' && 'Your configuration has been imported successfully.'}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4 space-y-4">
          {isLoading && step !== 'complete' && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          )}

          {!isLoading && step === 'upload' && renderUploadStep()}
          {!isLoading && step === 'passphrase' && renderPassphraseStep()}
          {!isLoading && step === 'preview' && renderPreviewStep()}
          {step === 'complete' && renderCompleteStep()}

          {error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          {step === 'upload' && (
            <Button variant="outline" onClick={handleClose}>
              Cancel
            </Button>
          )}

          {step === 'passphrase' && (
            <>
              <Button variant="outline" onClick={() => {
                setStep('upload')
                setIsEncrypted(false)
                setEncryptedConfig(null)
                setPassphrase('')
              }}>
                Back
              </Button>
              <Button onClick={handleDecryptAndPreview} disabled={!passphrase || isLoading}>
                <Key className="mr-2 h-4 w-4" />
                Continue
              </Button>
            </>
          )}

          {step === 'preview' && (
            <>
              <Button variant="outline" onClick={() => {
                if (isEncrypted) {
                  setStep('passphrase')
                } else {
                  setStep('upload')
                }
              }}>
                Back
              </Button>
              <Button onClick={handleImport} disabled={isLoading}>
                {isEncrypted ? (
                  <Lock className="mr-2 h-4 w-4" />
                ) : (
                  <Upload className="mr-2 h-4 w-4" />
                )}
                {isEncrypted ? 'Import with Passwords' : 'Import'}
              </Button>
            </>
          )}

          {step === 'complete' && (
            <Button onClick={handleClose}>
              Done
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
