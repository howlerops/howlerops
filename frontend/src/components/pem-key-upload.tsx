import React, { useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Upload, AlertCircle, CheckCircle } from 'lucide-react'
import { SecretTextarea } from './secret-input'

interface PemKeyUploadProps {
  onUpload: (keyContent: string) => void
  onError?: (error: string) => void
  disabled?: boolean
  className?: string
}

export function PemKeyUpload({ onUpload, onError, disabled, className }: PemKeyUploadProps) {
  const [keyContent, setKeyContent] = useState('')
  const [isDragOver, setIsDragOver] = useState(false)
  const [validationError, setValidationError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const validatePemKey = (content: string): string | null => {
    if (!content.trim()) {
      return 'Please provide a private key'
    }

    // Check for common PEM key formats
    const pemPatterns = [
      /-----BEGIN RSA PRIVATE KEY-----[\s\S]*?-----END RSA PRIVATE KEY-----/,
      /-----BEGIN PRIVATE KEY-----[\s\S]*?-----END PRIVATE KEY-----/,
      /-----BEGIN OPENSSH PRIVATE KEY-----[\s\S]*?-----END OPENSSH PRIVATE KEY-----/,
      /-----BEGIN EC PRIVATE KEY-----[\s\S]*?-----END EC PRIVATE KEY-----/,
      /-----BEGIN DSA PRIVATE KEY-----[\s\S]*?-----END DSA PRIVATE KEY-----/,
    ]

    const isValidPem = pemPatterns.some(pattern => pattern.test(content))
    if (!isValidPem) {
      return 'Invalid PEM key format. Please ensure the key starts with -----BEGIN and ends with -----END'
    }

    return null
  }

  const handleFileSelect = (file: File) => {
    if (!file) return

    // Check file size (max 10KB for private keys)
    if (file.size > 10 * 1024) {
      const error = 'File too large. Private keys should be less than 10KB'
      setValidationError(error)
      onError?.(error)
      return
    }

    // Check file type
    if (!file.name.match(/\.(pem|key|rsa|dsa|ec)$/i)) {
      const error = 'Invalid file type. Please select a .pem, .key, .rsa, .dsa, or .ec file'
      setValidationError(error)
      onError?.(error)
      return
    }

    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target?.result as string
      const error = validatePemKey(content)
      
      if (error) {
        setValidationError(error)
        onError?.(error)
      } else {
        setValidationError('')
        setKeyContent(content)
        onUpload(content)
      }
    }
    reader.onerror = () => {
      const error = 'Failed to read file'
      setValidationError(error)
      onError?.(error)
    }
    reader.readAsText(file)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
    
    const files = Array.from(e.dataTransfer.files)
    if (files.length > 0) {
      handleFileSelect(files[0])
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
  }

  const handlePaste = (e: React.ClipboardEvent) => {
    const pastedText = e.clipboardData.getData('text')
    if (pastedText) {
      const error = validatePemKey(pastedText)
      if (error) {
        setValidationError(error)
        onError?.(error)
      } else {
        setValidationError('')
        setKeyContent(pastedText)
        onUpload(pastedText)
      }
    }
  }

  const getKeyFingerprint = (content: string): string => {
    // Simple fingerprint calculation (in real implementation, use proper SSH key fingerprint)
    const lines = content.split('\n').filter(line => 
      !line.startsWith('-----BEGIN') && 
      !line.startsWith('-----END') && 
      line.trim()
    )
    const keyData = lines.join('')
    return keyData.substring(0, 20) + '...'
  }

  return (
    <div className={className}>
      <Tabs defaultValue="upload" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="upload">Upload File</TabsTrigger>
          <TabsTrigger value="paste">Paste Key</TabsTrigger>
        </TabsList>
        
        <TabsContent value="upload" className="space-y-4">
          <div
            className={`border-2 border-dashed rounded-lg p-6 text-center transition-colors ${
              isDragOver 
                ? 'border-primary bg-primary/5' 
                : 'border-muted-foreground/25 hover:border-muted-foreground/50'
            }`}
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
          >
            <Upload className="mx-auto h-8 w-8 text-muted-foreground mb-2" />
            <div className="space-y-2">
              <p className="text-sm font-medium">Drop your private key file here</p>
              <p className="text-xs text-muted-foreground">
                or click to browse (.pem, .key, .rsa, .dsa, .ec files)
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              className="mt-4"
              onClick={() => fileInputRef.current?.click()}
              disabled={disabled}
            >
              Choose File
            </Button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".pem,.key,.rsa,.dsa,.ec"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) handleFileSelect(file)
              }}
              className="hidden"
              disabled={disabled}
            />
          </div>
        </TabsContent>
        
        <TabsContent value="paste" className="space-y-4">
          <SecretTextarea
            value={keyContent}
            onChange={(value) => {
              setKeyContent(value)
              const error = validatePemKey(value)
              if (error) {
                setValidationError(error)
                onError?.(error)
              } else {
                setValidationError('')
                onUpload(value)
              }
            }}
            placeholder="-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
            label="Private Key Content"
            disabled={disabled}
            onPaste={handlePaste}
          />
        </TabsContent>
      </Tabs>

      {validationError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{validationError}</AlertDescription>
        </Alert>
      )}

      {keyContent && !validationError && (
        <Alert>
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>
            <div className="space-y-1">
              <p>Private key loaded successfully</p>
              <p className="text-xs text-muted-foreground">
                Fingerprint: {getKeyFingerprint(keyContent)}
              </p>
            </div>
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
}
