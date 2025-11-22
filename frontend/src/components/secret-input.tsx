import { Eye, EyeOff } from 'lucide-react'
import React, { forwardRef,useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface SecretInputProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  label?: string
  disabled?: boolean
  required?: boolean
  className?: string
  id?: string
}

export const SecretInput = forwardRef<HTMLInputElement, SecretInputProps>(
  ({ value, onChange, placeholder, label, disabled, required, className, id }, ref) => {
    const [showSecret, setShowSecret] = useState(false)

    return (
      <div className="space-y-2">
        {label && (
          <Label htmlFor={id}>
            {label}
            {required && <span className="text-destructive ml-1">*</span>}
          </Label>
        )}
        <div className="relative">
          <Input
            ref={ref}
            id={id}
            type={showSecret ? 'text' : 'password'}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            disabled={disabled}
            required={required}
            className={`pr-10 ${className || ''}`}
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
            onClick={() => setShowSecret(!showSecret)}
            disabled={disabled}
            tabIndex={-1}
          >
            {showSecret ? (
              <EyeOff className="h-4 w-4" />
            ) : (
              <Eye className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    )
  }
)

SecretInput.displayName = 'SecretInput'

interface SecretTextareaProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  label?: string
  disabled?: boolean
  required?: boolean
  className?: string
  id?: string
  rows?: number
}

export const SecretTextarea = forwardRef<HTMLTextAreaElement, SecretTextareaProps>(
  ({ value, onChange, placeholder, label, disabled, required, className, id, rows = 6 }, ref) => {
    const [showSecret, setShowSecret] = useState(false)

    return (
      <div className="space-y-2">
        {label && (
          <Label htmlFor={id}>
            {label}
            {required && <span className="text-destructive ml-1">*</span>}
          </Label>
        )}
        <div className="relative">
          <textarea
            ref={ref}
            id={id}
            value={showSecret ? value : '•'.repeat(value.length)}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            disabled={disabled}
            required={required}
            rows={rows}
            className={`w-full px-3 py-2 border border-input bg-background text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 rounded-md pr-10 ${className || ''}`}
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
            onClick={() => setShowSecret(!showSecret)}
            disabled={disabled}
            tabIndex={-1}
          >
            {showSecret ? (
              <EyeOff className="h-4 w-4" />
            ) : (
              <Eye className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    )
  }
)

SecretTextarea.displayName = 'SecretTextarea'
