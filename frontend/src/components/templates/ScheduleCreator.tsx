/**
 * ScheduleCreator Component
 * Modal for creating and editing scheduled queries
 */

import React, { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Calendar, Info } from 'lucide-react'
import type { QueryTemplate, CreateScheduleInput, TemplateParameterValue } from '@/types/templates'
import { useTemplatesStore } from '@/store/templates-store'
import { CronBuilder } from './CronBuilder'
// Using simple pre/code for SQL display - can be upgraded to CodeMirror if needed

interface ScheduleCreatorProps {
  open: boolean
  onClose: () => void
  template?: QueryTemplate
}

export function ScheduleCreator({ open, onClose, template: initialTemplate }: ScheduleCreatorProps) {
  const { templates, createSchedule, loading } = useTemplatesStore()

  const [formData, setFormData] = useState<CreateScheduleInput>({
    template_id: initialTemplate?.id || '',
    name: '',
    frequency: '0 9 * * *', // Daily at 9am
    parameters: {},
    notification_email: '',
    result_storage: 'none',
  })

  const [errors, setErrors] = useState<Record<string, string>>({})

  // Find selected template
  const selectedTemplate = templates.find((t) => t.id === formData.template_id) || initialTemplate

  // Initialize parameters when template changes
  useEffect(() => {
    if (selectedTemplate) {
      const params: Record<string, any> = {}
      selectedTemplate.parameters.forEach((param) => {
        if (param.default !== undefined) {
          params[param.name] = param.default
        }
      })
      setFormData((prev) => ({ ...prev, parameters: params }))
    }
  }, [selectedTemplate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Validate
    const newErrors: Record<string, string> = {}

    if (!formData.template_id) {
      newErrors.template_id = 'Please select a template'
    }
    if (!formData.name.trim()) {
      newErrors.name = 'Schedule name is required'
    }
    if (formData.notification_email && !isValidEmail(formData.notification_email)) {
      newErrors.notification_email = 'Invalid email address'
    }

    // Validate required parameters
    if (selectedTemplate) {
      selectedTemplate.parameters.forEach((param) => {
        if (param.required && !formData.parameters[param.name]) {
          newErrors[`param_${param.name}`] = 'This parameter is required'
        }
      })
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors)
      return
    }

    try {
      await createSchedule(formData)
      onClose()
      // Reset form
      setFormData({
        template_id: '',
        name: '',
        frequency: '0 9 * * *',
        parameters: {},
        notification_email: '',
        result_storage: 'none',
      })
      setErrors({})
    } catch (error) {
      setErrors({ submit: error instanceof Error ? error.message : 'Failed to create schedule' })
    }
  }

  const updateField = <K extends keyof CreateScheduleInput>(
    field: K,
    value: CreateScheduleInput[K]
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
    setErrors((prev) => {
      const newErrors = { ...prev }
      delete newErrors[field]
      return newErrors
    })
  }

  const updateParameter = (
    name: string,
    rawValue: string,
    type: QueryTemplate['parameters'][number]['type']
  ) => {
    let parsedValue: TemplateParameterValue = rawValue

    if (type === 'number') {
      parsedValue = rawValue === '' ? null : Number(rawValue)
    } else if (type === 'boolean') {
      parsedValue = rawValue === 'true'
    } else if (type === 'date' && rawValue === '') {
      parsedValue = null
    }

    setFormData((prev) => ({
      ...prev,
      parameters: { ...prev.parameters, [name]: parsedValue },
    }))
    setErrors((prev) => {
      const newErrors = { ...prev }
      delete newErrors[`param_${name}`]
      return newErrors
    })
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Schedule Query
          </DialogTitle>
          <DialogDescription>
            Create a scheduled query that runs automatically at specified intervals
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex-1 overflow-hidden flex flex-col">
          <ScrollArea className="flex-1 pr-4">
            <div className="space-y-6">
              {/* Template Selection */}
              <div className="space-y-2">
                <Label htmlFor="template">
                  Template <span className="text-red-500">*</span>
                </Label>
                <Select
                  value={formData.template_id}
                  onValueChange={(v) => updateField('template_id', v)}
                  disabled={!!initialTemplate}
                >
                  <SelectTrigger id="template" className={errors.template_id ? 'border-red-500' : ''}>
                    <SelectValue placeholder="Select a template..." />
                  </SelectTrigger>
                  <SelectContent>
                    {templates.map((t) => (
                      <SelectItem key={t.id} value={t.id}>
                        {t.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {errors.template_id && (
                  <p className="text-xs text-red-500">{errors.template_id}</p>
                )}
              </div>

              {/* Schedule Name */}
              <div className="space-y-2">
                <Label htmlFor="name">
                  Schedule Name <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => updateField('name', e.target.value)}
                  placeholder="Daily sales report"
                  className={errors.name ? 'border-red-500' : ''}
                />
                {errors.name && <p className="text-xs text-red-500">{errors.name}</p>}
              </div>

              {/* Frequency */}
              <div className="space-y-2">
                <Label>
                  Schedule <span className="text-red-500">*</span>
                </Label>
                <CronBuilder
                  value={formData.frequency}
                  onChange={(v) => updateField('frequency', v)}
                />
              </div>

              {/* Template SQL Preview */}
              {selectedTemplate && (
                <div className="space-y-2">
                  <Label>SQL Query</Label>
                  <div className="border rounded-lg overflow-hidden">
                    <pre className="bg-muted p-4 overflow-auto font-mono text-sm max-h-[200px]">
                      <code className="language-sql">{selectedTemplate.sql_template}</code>
                    </pre>
                  </div>
                </div>
              )}

              {/* Parameters */}
              {selectedTemplate && selectedTemplate.parameters.length > 0 && (
                <div className="space-y-4">
                  <Label className="text-base font-semibold">Parameters</Label>
                  {selectedTemplate.parameters.map((param) => (
                    <div key={param.name} className="space-y-2">
                      <Label htmlFor={`param_${param.name}`}>
                        {param.name}
                        {param.required && <span className="text-red-500 ml-1">*</span>}
                      </Label>
                      {param.description && (
                        <p className="text-xs text-muted-foreground">{param.description}</p>
                      )}
                      <Input
                        id={`param_${param.name}`}
                        type={param.type === 'number' ? 'number' : param.type === 'date' ? 'date' : 'text'}
                        value={
                          formData.parameters[param.name] === null || formData.parameters[param.name] === undefined
                            ? ''
                            : String(formData.parameters[param.name])
                        }
                        onChange={(e) => updateParameter(param.name, e.target.value, param.type)}
                        placeholder={param.default !== undefined ? String(param.default) : undefined}
                        className={errors[`param_${param.name}`] ? 'border-red-500' : ''}
                      />
                      {errors[`param_${param.name}`] && (
                        <p className="text-xs text-red-500">{errors[`param_${param.name}`]}</p>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* Notification Email */}
              <div className="space-y-2">
                <Label htmlFor="email">Notification Email (Optional)</Label>
                <Input
                  id="email"
                  type="email"
                  value={formData.notification_email}
                  onChange={(e) => updateField('notification_email', e.target.value)}
                  placeholder="team@company.com"
                  className={errors.notification_email ? 'border-red-500' : ''}
                />
                <p className="text-xs text-muted-foreground">
                  Receive email notifications when the query completes or fails
                </p>
                {errors.notification_email && (
                  <p className="text-xs text-red-500">{errors.notification_email}</p>
                )}
              </div>

              {/* Result Storage */}
              <div className="space-y-2">
                <Label htmlFor="storage">Result Storage</Label>
                <Select
                  value={formData.result_storage}
                  onValueChange={(v: any) => updateField('result_storage', v)}
                >
                  <SelectTrigger id="storage">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Don't Store Results</SelectItem>
                    <SelectItem value="database">Store in Database</SelectItem>
                    <SelectItem value="s3">Store in S3 Bucket</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Choose where to store query results for later retrieval
                </p>
              </div>

              {/* Info Alert */}
              <Alert>
                <Info className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  The scheduled query will run automatically according to the specified frequency.
                  You can pause, resume, or delete the schedule at any time.
                </AlertDescription>
              </Alert>
            </div>
          </ScrollArea>

          {/* Submit Error */}
          {errors.submit && (
            <Alert variant="destructive" className="mt-4">
              <AlertDescription>{errors.submit}</AlertDescription>
            </Alert>
          )}

          <DialogFooter className="mt-6">
            <Button type="button" variant="outline" onClick={onClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? 'Creating...' : 'Create Schedule'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}
