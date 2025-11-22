/**
 * TemplateExecutor Component
 * Modal for executing query templates with parameter inputs
 */

import { AlertCircle, CheckCircle, Clock,Play } from 'lucide-react'
import React, { useMemo,useState } from 'react'

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
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { interpolateTemplate } from '@/lib/api/templates'
import { useTemplatesStore } from '@/store/templates-store'
import type { QueryResult, QueryTemplate, TemplateParameter, TemplateParameterValue } from '@/types/templates'
// Using simple pre/code for SQL display - can be upgraded to CodeMirror if needed

interface TemplateExecutorProps {
  template: QueryTemplate
  open: boolean
  onClose: () => void
}

export function TemplateExecutor({ template, open, onClose }: TemplateExecutorProps) {
  const [params, setParams] = useState<Record<string, TemplateParameterValue>>(() => {
    // Initialize with default values
    const defaults: Record<string, TemplateParameterValue> = {}
    template.parameters.forEach((param) => {
      if (param.default !== undefined) {
        defaults[param.name] = param.default
      } else if (param.type === 'boolean') {
        defaults[param.name] = false
      } else {
        defaults[param.name] = ''
      }
    })
    return defaults
  })

  const [result, setResult] = useState<QueryResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isExecuting, setIsExecuting] = useState(false)

  const { executeTemplate } = useTemplatesStore()

  // Generate SQL preview with parameters interpolated
  const sqlPreview = useMemo(() => {
    try {
      return interpolateTemplate(template.sql_template, params)
    } catch {
      return template.sql_template
    }
  }, [template.sql_template, params])

  // Validate parameters
  const validationErrors = useMemo(() => {
    const errors: Record<string, string> = {}

    template.parameters.forEach((param) => {
      const value = params[param.name]

      // Check required
      if (param.required && (value === '' || value === null || value === undefined)) {
        errors[param.name] = 'This field is required'
        return
      }

      // Skip validation if empty and not required
      if (!param.required && (value === '' || value === null || value === undefined)) {
        return
      }

      // Type validation
      if (param.type === 'number' && isNaN(Number(value))) {
        errors[param.name] = 'Must be a valid number'
      }

      // Custom validation
      if (param.validation) {
        if (param.validation.min !== undefined && Number(value) < param.validation.min) {
          errors[param.name] = `Must be at least ${param.validation.min}`
        }
        if (param.validation.max !== undefined && Number(value) > param.validation.max) {
          errors[param.name] = `Must be at most ${param.validation.max}`
        }
        if (param.validation.pattern && !new RegExp(param.validation.pattern).test(String(value))) {
          errors[param.name] = 'Invalid format'
        }
      }
    })

    return errors
  }, [template.parameters, params])

  const isValid = Object.keys(validationErrors).length === 0

  const handleParamChange = (name: string, value: TemplateParameterValue) => {
    setParams((prev) => ({ ...prev, [name]: value }))
    setError(null)
  }

  const handleExecute = async () => {
    if (!isValid) return

    setIsExecuting(true)
    setError(null)
    setResult(null)

    try {
      const result = await executeTemplate(template.id, params)
      setResult(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to execute query')
    } finally {
      setIsExecuting(false)
    }
  }

  const handleReset = () => {
    setParams({})
    setResult(null)
    setError(null)
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-5xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Play className="h-5 w-5" />
            Execute: {template.name}
          </DialogTitle>
          <DialogDescription>{template.description}</DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="params" className="flex-1 overflow-hidden flex flex-col">
          <TabsList>
            <TabsTrigger value="params">
              Parameters
              {template.parameters.length > 0 && (
                <Badge variant="secondary" className="ml-2">
                  {template.parameters.length}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="preview">SQL Preview</TabsTrigger>
            {result && <TabsTrigger value="results">Results</TabsTrigger>}
          </TabsList>

          {/* Parameters Tab */}
          <TabsContent value="params" className="flex-1 overflow-auto">
            <ScrollArea className="h-full">
              <div className="space-y-4 pr-4">
                {template.parameters.length === 0 ? (
                  <Alert>
                    <AlertDescription>
                      This template has no parameters. Click Execute to run the query.
                    </AlertDescription>
                  </Alert>
                ) : (
                  template.parameters.map((param) => (
                    <ParameterInput
                      key={param.name}
                      parameter={param}
                      value={params[param.name]}
                      error={validationErrors[param.name]}
                      onChange={(value) => handleParamChange(param.name, value)}
                    />
                  ))
                )}
              </div>
            </ScrollArea>
          </TabsContent>

          {/* SQL Preview Tab */}
          <TabsContent value="preview" className="flex-1 overflow-auto">
            <div className="relative">
              <pre className="bg-muted rounded-lg p-4 overflow-auto font-mono text-sm">
                <code className="language-sql">{sqlPreview}</code>
              </pre>
            </div>
          </TabsContent>

          {/* Results Tab */}
          {result && (
            <TabsContent value="results" className="flex-1 overflow-auto">
              <ResultsTable result={result} />
            </TabsContent>
          )}
        </Tabs>

        {/* Error Display */}
        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Success Message */}
        {result && !error && (
          <Alert>
            <CheckCircle className="h-4 w-4" />
            <AlertDescription>
              Query executed successfully in {result.executionTime}ms. {result.totalRows !== undefined ? result.totalRows : result.rowCount} rows returned.
            </AlertDescription>
          </Alert>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={handleReset} disabled={isExecuting}>
            Reset
          </Button>
          <Button onClick={handleExecute} disabled={!isValid || isExecuting}>
            {isExecuting ? (
              <>
                <Clock className="mr-2 h-4 w-4 animate-spin" />
                Executing...
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                Execute Query
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================================
// Parameter Input Component
// ============================================================================

interface ParameterInputProps {
  parameter: TemplateParameter
  value: TemplateParameterValue
  error?: string
  onChange: (value: TemplateParameterValue) => void
}

function ParameterInput({ parameter, value, error, onChange }: ParameterInputProps) {
  const renderInput = () => {
    switch (parameter.type) {
      case 'boolean':
        return (
          <div className="flex items-center space-x-2">
            <Checkbox
              id={parameter.name}
              checked={value as boolean}
              onCheckedChange={onChange}
            />
            <Label htmlFor={parameter.name} className="font-normal cursor-pointer">
              {parameter.description || parameter.name}
            </Label>
          </div>
        )

      case 'date':
        return (
          <Input
            type="date"
            value={value === null ? '' : String(value)}
            onChange={(e) => onChange(e.target.value)}
          />
        )

      case 'number':
        return (
          <Input
            type="number"
            value={value === null ? '' : String(value)}
            onChange={(e) => onChange(e.target.value)}
            min={parameter.validation?.min}
            max={parameter.validation?.max}
          />
        )

      default:
        // String or select
        if (parameter.validation?.options) {
          return (
            <select
              value={value === null ? '' : String(value)}
              onChange={(e) => onChange(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value="">Select an option...</option>
              {parameter.validation.options.map((opt) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
          )
        }

        return (
          <Input
            type="text"
            value={value === null ? '' : String(value)}
            onChange={(e) => onChange(e.target.value)}
            placeholder={parameter.default ? String(parameter.default) : undefined}
          />
        )
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Label>
          {parameter.name}
          {parameter.required && <span className="text-red-500 ml-1">*</span>}
        </Label>
        <Badge variant="outline" className="text-xs">
          {parameter.type}
        </Badge>
      </div>

      {parameter.type !== 'boolean' && parameter.description && (
        <p className="text-xs text-muted-foreground">{parameter.description}</p>
      )}

      {renderInput()}

      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  )
}

// ============================================================================
// Results Table Component
// ============================================================================

function ResultsTable({ result }: { result: QueryResult }) {
  if (result.rowCount === 0) {
    return (
      <Alert>
        <AlertDescription>Query returned no results.</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="border rounded-lg overflow-hidden">
      <div className="overflow-auto max-h-96">
        <table className="w-full text-sm">
          <thead className="bg-muted sticky top-0">
            <tr>
              {result.columns.map((col) => (
                <th key={col} className="px-4 py-2 text-left font-medium">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {result.rows.map((row, i) => (
              <tr key={i} className="border-t hover:bg-muted/50">
                {result.columns.map((col) => (
                  <td key={col} className="px-4 py-2">
                    {row[col] === null ? (
                      <span className="text-muted-foreground italic">null</span>
                    ) : (
                      String(row[col])
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
