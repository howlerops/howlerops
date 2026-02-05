/**
 * Connection Export/Import Dialog Components
 *
 * Provides UI for exporting and importing database connections.
 */

import {
  AlertCircle,
  CheckCircle2,
  Download,
  Loader2,
  Upload,
} from 'lucide-react'
import { useCallback, useRef, useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  type ConflictResolution,
  type ConnectionExportFile,
  type ImportResult,
  exportConnections,
  getConflictingIds,
  getExportableConnections,
  importConnections,
  previewImport,
  readExportFile,
} from '@/lib/export-import'

// ============================================================================
// Export Dialog
// ============================================================================

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ExportDialog({ open, onOpenChange }: ExportDialogProps) {
  const [includePasswords, setIncludePasswords] = useState(false)
  const [passwordWarningAcknowledged, setPasswordWarningAcknowledged] = useState(false)
  const [isExporting, setIsExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const connections = getExportableConnections()

  const handleExport = async () => {
    if (includePasswords && !passwordWarningAcknowledged) {
      return
    }

    setIsExporting(true)
    setError(null)

    try {
      await exportConnections({ includePasswords })
      onOpenChange(false)
      // Reset state
      setIncludePasswords(false)
      setPasswordWarningAcknowledged(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed')
    } finally {
      setIsExporting(false)
    }
  }

  const handleClose = () => {
    if (!isExporting) {
      onOpenChange(false)
      setIncludePasswords(false)
      setPasswordWarningAcknowledged(false)
      setError(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Export Connections
          </DialogTitle>
          <DialogDescription>
            Export your database connections to a JSON file for backup or transfer to another device.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Connection count */}
          <div className="text-sm text-muted-foreground">
            {connections.length === 0 ? (
              'No connections to export.'
            ) : (
              `${connections.length} connection${connections.length === 1 ? '' : 's'} will be exported.`
            )}
          </div>

          {/* Include passwords option */}
          {connections.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="include-passwords"
                  checked={includePasswords}
                  onCheckedChange={(checked) => {
                    setIncludePasswords(checked === true)
                    if (!checked) {
                      setPasswordWarningAcknowledged(false)
                    }
                  }}
                />
                <Label htmlFor="include-passwords" className="text-sm font-medium">
                  Include database passwords
                </Label>
              </div>

              {/* Password warning */}
              {includePasswords && (
                <Alert variant="destructive" className="bg-destructive/10">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription className="text-sm">
                    <strong>Security Warning:</strong> Passwords will be stored in plain text in the export file.
                    Only export with passwords if you need to transfer credentials and will delete the file after import.
                    <div className="mt-2 flex items-center space-x-2">
                      <Checkbox
                        id="acknowledge-warning"
                        checked={passwordWarningAcknowledged}
                        onCheckedChange={(checked) => setPasswordWarningAcknowledged(checked === true)}
                      />
                      <Label htmlFor="acknowledge-warning" className="text-sm">
                        I understand the risks
                      </Label>
                    </div>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}

          {/* Error display */}
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isExporting}>
            Cancel
          </Button>
          <Button
            onClick={handleExport}
            disabled={
              isExporting ||
              connections.length === 0 ||
              (includePasswords && !passwordWarningAcknowledged)
            }
          >
            {isExporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Exporting...
              </>
            ) : (
              <>
                <Download className="mr-2 h-4 w-4" />
                Export
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================================
// Import Dialog
// ============================================================================

interface ImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImportComplete?: () => void
}

export function ImportDialog({ open, onOpenChange, onImportComplete }: ImportDialogProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [parsedFile, setParsedFile] = useState<ConnectionExportFile | null>(null)
  const [conflictingIds, setConflictingIds] = useState<string[]>([])
  const [conflictResolution, setConflictResolution] = useState<ConflictResolution>('skip')
  const [isImporting, setIsImporting] = useState(false)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isDragOver, setIsDragOver] = useState(false)

  const handleFileSelect = useCallback(async (file: File) => {
    setError(null)
    setResult(null)

    try {
      const exportFile = await readExportFile(file)
      setParsedFile(exportFile)

      const conflicts = getConflictingIds(exportFile)
      setConflictingIds(conflicts)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to read file')
      setParsedFile(null)
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)

    const file = e.dataTransfer.files[0]
    if (file && file.name.endsWith('.json')) {
      handleFileSelect(file)
    } else {
      setError('Please drop a JSON file')
    }
  }, [handleFileSelect])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
  }, [])

  const handleImport = async () => {
    if (!parsedFile) return

    setIsImporting(true)
    setError(null)

    try {
      const importResult = await importConnections(parsedFile, { conflictResolution })
      setResult(importResult)

      if (importResult.imported > 0 || importResult.overwritten > 0) {
        onImportComplete?.()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed')
    } finally {
      setIsImporting(false)
    }
  }

  const handleClose = () => {
    if (!isImporting) {
      onOpenChange(false)
      // Reset state after animation
      setTimeout(() => {
        setParsedFile(null)
        setConflictingIds([])
        setConflictResolution('skip')
        setResult(null)
        setError(null)
      }, 200)
    }
  }

  const preview = parsedFile ? previewImport(parsedFile, { conflictResolution }) : null

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Upload className="h-5 w-5" />
            Import Connections
          </DialogTitle>
          <DialogDescription>
            Import database connections from a previously exported JSON file.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Result display */}
          {result ? (
            <div className="space-y-3">
              <Alert variant="default" className="bg-green-500/10 border-green-500/20">
                <CheckCircle2 className="h-4 w-4 text-green-500" />
                <AlertDescription>
                  <div className="font-medium">Import Complete</div>
                  <ul className="mt-1 text-sm space-y-0.5">
                    {result.imported > 0 && <li>{result.imported} connection(s) imported</li>}
                    {result.overwritten > 0 && <li>{result.overwritten} connection(s) updated</li>}
                    {result.skipped > 0 && <li>{result.skipped} connection(s) skipped (duplicates)</li>}
                    {result.failed.length > 0 && (
                      <li className="text-destructive">{result.failed.length} connection(s) failed</li>
                    )}
                  </ul>
                </AlertDescription>
              </Alert>

              {/* Failed imports detail */}
              {result.failed.length > 0 && (
                <Alert variant="destructive" className="bg-destructive/10">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    <div className="font-medium">Failed Imports:</div>
                    <ul className="mt-1 text-sm space-y-0.5">
                      {result.failed.map((f, i) => (
                        <li key={i}>{f.connectionName}: {f.reason}</li>
                      ))}
                    </ul>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          ) : (
            <>
              {/* File drop zone */}
              {!parsedFile && (
                <div
                  className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
                    isDragOver
                      ? 'border-primary bg-primary/5'
                      : 'border-muted-foreground/25 hover:border-muted-foreground/50'
                  }`}
                  onDrop={handleDrop}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onClick={() => fileInputRef.current?.click()}
                >
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".json"
                    onChange={(e) => {
                      const file = e.target.files?.[0]
                      if (file) handleFileSelect(file)
                    }}
                    className="hidden"
                  />
                  <Upload className="mx-auto h-8 w-8 text-muted-foreground mb-2" />
                  <p className="text-sm font-medium">Drop JSON file here or click to browse</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    HowlerOps connection export files only
                  </p>
                </div>
              )}

              {/* Parsed file info */}
              {parsedFile && (
                <div className="space-y-4">
                  <Alert>
                    <AlertDescription>
                      <div className="font-medium">
                        {parsedFile.connections.length} connection(s) found
                      </div>
                      <div className="text-xs text-muted-foreground mt-1">
                        Exported on {new Date(parsedFile.metadata.exportedAt).toLocaleString()}
                        {parsedFile.metadata.includesPasswords && (
                          <span className="ml-2 text-amber-500">(includes passwords)</span>
                        )}
                      </div>
                    </AlertDescription>
                  </Alert>

                  {/* Conflict handling */}
                  {conflictingIds.length > 0 && (
                    <div className="space-y-2">
                      <Label className="text-sm font-medium">
                        {conflictingIds.length} connection(s) already exist. How to handle?
                      </Label>
                      <Select
                        value={conflictResolution}
                        onValueChange={(v) => setConflictResolution(v as ConflictResolution)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="skip">Skip duplicates (keep existing)</SelectItem>
                          <SelectItem value="overwrite">Overwrite existing connections</SelectItem>
                          <SelectItem value="keep-both">Import as new (create duplicates)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  {/* Preview */}
                  {preview && (
                    <div className="text-sm text-muted-foreground">
                      <div className="font-medium mb-1">Preview:</div>
                      <ul className="space-y-0.5">
                        {preview.toImport > 0 && <li>{preview.toImport} will be imported</li>}
                        {preview.toOverwrite > 0 && <li>{preview.toOverwrite} will be updated</li>}
                        {preview.toSkip > 0 && <li>{preview.toSkip} will be skipped</li>}
                        {preview.invalid > 0 && <li className="text-destructive">{preview.invalid} invalid (will fail)</li>}
                      </ul>
                    </div>
                  )}

                  {/* Change file button */}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setParsedFile(null)
                      setConflictingIds([])
                      setError(null)
                    }}
                  >
                    Choose Different File
                  </Button>
                </div>
              )}
            </>
          )}

          {/* Error display */}
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isImporting}>
            {result ? 'Close' : 'Cancel'}
          </Button>
          {!result && (
            <Button
              onClick={handleImport}
              disabled={isImporting || !parsedFile}
            >
              {isImporting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Importing...
                </>
              ) : (
                <>
                  <Upload className="mr-2 h-4 w-4" />
                  Import
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
