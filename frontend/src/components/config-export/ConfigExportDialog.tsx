import {
  AlertTriangle,
  CheckCircle2,
  Database,
  Download,
  Eye,
  EyeOff,
  FileText,
  History,
  Key,
  Loader2,
  Lock,
  Share2,
  Shield,
} from 'lucide-react'
import { useMemo,useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  type ConfigExportOptions,
  defaultExportOptions,
  downloadConfigFile,
  downloadEncryptedConfigFile,
  validatePassphraseStrength,
} from '@/lib/api/config-export'

interface ConfigExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ConfigExportDialog({ open, onOpenChange }: ConfigExportDialogProps) {
  const [options, setOptions] = useState<ConfigExportOptions>(defaultExportOptions)
  const [isExporting, setIsExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Encrypted export state
  const [includePasswords, setIncludePasswords] = useState(false)
  const [passphrase, setPassphrase] = useState('')
  const [confirmPassphrase, setConfirmPassphrase] = useState('')
  const [showPassphrase, setShowPassphrase] = useState(false)

  // Validate passphrase strength
  const passphraseValidation = useMemo(() => {
    if (!includePasswords || !passphrase) {
      return { valid: true, errors: [] }
    }
    return validatePassphraseStrength(passphrase)
  }, [includePasswords, passphrase])

  // Check if passphrases match
  const passphrasesMatch = passphrase === confirmPassphrase

  // Can export with passwords?
  const canExportWithPasswords =
    includePasswords &&
    passphraseValidation.valid &&
    passphrasesMatch &&
    passphrase.length >= 12

  const handleExport = async () => {
    setIsExporting(true)
    setError(null)

    try {
      if (includePasswords) {
        if (!canExportWithPasswords) {
          setError('Please fix passphrase issues before exporting')
          setIsExporting(false)
          return
        }
        await downloadEncryptedConfigFile(options, passphrase)
      } else {
        await downloadConfigFile(options)
      }
      // Reset state on success
      setPassphrase('')
      setConfirmPassphrase('')
      setIncludePasswords(false)
      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed')
    } finally {
      setIsExporting(false)
    }
  }

  const updateOption = <K extends keyof ConfigExportOptions>(
    key: K,
    value: ConfigExportOptions[K]
  ) => {
    setOptions((prev) => ({ ...prev, [key]: value }))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Export Configuration
          </DialogTitle>
          <DialogDescription>
            Export your database connections, saved queries, and settings to a JSON file.
            {includePasswords
              ? ' Passwords will be securely encrypted with your passphrase.'
              : ' Passwords are not included by default.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* What to include */}
          <div className="space-y-3">
            <Label className="text-sm font-medium">Include in export</Label>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include_connections"
                checked={options.include_connections}
                onCheckedChange={(checked) =>
                  updateOption('include_connections', checked === true)
                }
              />
              <Label htmlFor="include_connections" className="flex items-center gap-2 cursor-pointer">
                <Database className="h-4 w-4 text-muted-foreground" />
                Database Connections
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include_saved_queries"
                checked={options.include_saved_queries}
                onCheckedChange={(checked) =>
                  updateOption('include_saved_queries', checked === true)
                }
              />
              <Label htmlFor="include_saved_queries" className="flex items-center gap-2 cursor-pointer">
                <FileText className="h-4 w-4 text-muted-foreground" />
                Saved Queries & Tags
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include_query_history"
                checked={options.include_query_history}
                onCheckedChange={(checked) =>
                  updateOption('include_query_history', checked === true)
                }
              />
              <Label htmlFor="include_query_history" className="flex items-center gap-2 cursor-pointer">
                <History className="h-4 w-4 text-muted-foreground" />
                Query History
                <Badge variant="secondary" className="text-xs">Sanitized</Badge>
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include_shared"
                checked={options.include_shared}
                onCheckedChange={(checked) =>
                  updateOption('include_shared', checked === true)
                }
              />
              <Label htmlFor="include_shared" className="flex items-center gap-2 cursor-pointer">
                <Share2 className="h-4 w-4 text-muted-foreground" />
                Include Shared Resources
              </Label>
            </div>
          </div>

          <Separator />

          {/* Privacy options */}
          <div className="space-y-3">
            <Label className="text-sm font-medium">Privacy Options</Label>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="metadata_only"
                checked={options.metadata_only}
                onCheckedChange={(checked) =>
                  updateOption('metadata_only', checked === true)
                }
              />
              <Label htmlFor="metadata_only" className="flex items-center gap-2 cursor-pointer">
                <EyeOff className="h-4 w-4 text-muted-foreground" />
                Metadata Only
                <Badge variant="outline" className="text-xs">No SQL</Badge>
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="anonymize_hosts"
                checked={options.anonymize_hosts}
                onCheckedChange={(checked) =>
                  updateOption('anonymize_hosts', checked === true)
                }
              />
              <Label htmlFor="anonymize_hosts" className="flex items-center gap-2 cursor-pointer">
                <Shield className="h-4 w-4 text-muted-foreground" />
                Anonymize Hostnames
              </Label>
            </div>
          </div>

          <Separator />

          {/* Encrypted export with passwords */}
          <div className="space-y-3">
            <Label className="text-sm font-medium">Password Export</Label>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include_passwords"
                checked={includePasswords}
                onCheckedChange={(checked) => {
                  setIncludePasswords(checked === true)
                  if (!checked) {
                    setPassphrase('')
                    setConfirmPassphrase('')
                  }
                }}
              />
              <Label htmlFor="include_passwords" className="flex items-center gap-2 cursor-pointer">
                <Key className="h-4 w-4 text-muted-foreground" />
                Include Passwords
                <Badge variant="secondary" className="text-xs">Encrypted</Badge>
              </Label>
            </div>

            {includePasswords && (
              <div className="ml-6 space-y-3 p-3 bg-muted/50 rounded-lg border">
                <Alert className="border-amber-200 bg-amber-50 dark:bg-amber-950/20">
                  <Lock className="h-4 w-4 text-amber-600" />
                  <AlertDescription className="text-amber-800 dark:text-amber-200">
                    Passwords will be encrypted with AES-256-GCM using your passphrase.
                    <strong> You must remember this passphrase</strong> to import later.
                  </AlertDescription>
                </Alert>

                <div className="space-y-2">
                  <Label htmlFor="passphrase" className="text-sm">
                    Encryption Passphrase
                  </Label>
                  <div className="relative">
                    <Input
                      id="passphrase"
                      type={showPassphrase ? 'text' : 'password'}
                      value={passphrase}
                      onChange={(e) => setPassphrase(e.target.value)}
                      placeholder="Enter a strong passphrase (12+ chars)"
                      className="pr-10"
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

                  {/* Passphrase strength indicator */}
                  {passphrase && (
                    <div className="space-y-1">
                      {passphraseValidation.valid ? (
                        <div className="flex items-center gap-1 text-xs text-green-600">
                          <CheckCircle2 className="h-3 w-3" />
                          Passphrase strength OK
                        </div>
                      ) : (
                        passphraseValidation.errors.map((err, i) => (
                          <div key={i} className="flex items-center gap-1 text-xs text-red-600">
                            <AlertTriangle className="h-3 w-3" />
                            {err}
                          </div>
                        ))
                      )}
                    </div>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="confirm_passphrase" className="text-sm">
                    Confirm Passphrase
                  </Label>
                  <Input
                    id="confirm_passphrase"
                    type={showPassphrase ? 'text' : 'password'}
                    value={confirmPassphrase}
                    onChange={(e) => setConfirmPassphrase(e.target.value)}
                    placeholder="Re-enter passphrase"
                  />
                  {confirmPassphrase && !passphrasesMatch && (
                    <div className="flex items-center gap-1 text-xs text-red-600">
                      <AlertTriangle className="h-3 w-3" />
                      Passphrases do not match
                    </div>
                  )}
                  {confirmPassphrase && passphrasesMatch && passphrase && (
                    <div className="flex items-center gap-1 text-xs text-green-600">
                      <CheckCircle2 className="h-3 w-3" />
                      Passphrases match
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Security notice */}
          {!includePasswords && (
            <Alert>
              <Eye className="h-4 w-4" />
              <AlertDescription>
                <strong>Passwords are not included.</strong> You'll need to re-enter passwords
                when importing on another device.
              </AlertDescription>
            </Alert>
          )}

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleExport}
            disabled={isExporting || (includePasswords && !canExportWithPasswords)}
          >
            {isExporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {includePasswords ? 'Encrypting...' : 'Exporting...'}
              </>
            ) : (
              <>
                {includePasswords ? (
                  <Lock className="mr-2 h-4 w-4" />
                ) : (
                  <Download className="mr-2 h-4 w-4" />
                )}
                {includePasswords ? 'Export Encrypted' : 'Export'}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
