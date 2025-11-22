/**
 * CronBuilder Component
 * Visual cron expression builder with presets and custom options
 */

import { Clock, Info } from 'lucide-react'
import React, { useEffect,useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  buildCronExpression,
  CRON_PRESETS,
  type CronSchedule,
  cronToHumanReadable,
  isValidCronExpression,
  parseCronExpression,
} from '@/lib/utils/cron'

interface CronBuilderProps {
  value: string
  onChange: (expression: string) => void
  className?: string
}

const HOURS = Array.from({ length: 24 }, (_, i) => i)
const MINUTES = Array.from({ length: 12 }, (_, i) => i * 5) // Every 5 minutes
const DAYS_OF_MONTH = Array.from({ length: 31 }, (_, i) => i + 1)
const DAYS_OF_WEEK = [
  { value: '0', label: 'Sunday' },
  { value: '1', label: 'Monday' },
  { value: '2', label: 'Tuesday' },
  { value: '3', label: 'Wednesday' },
  { value: '4', label: 'Thursday' },
  { value: '5', label: 'Friday' },
  { value: '6', label: 'Saturday' },
]

export function CronBuilder({ value, onChange, className }: CronBuilderProps) {
  const [mode, setMode] = useState<'preset' | 'custom' | 'advanced'>('preset')
  const [schedule, setSchedule] = useState<CronSchedule>(() => {
    try {
      return parseCronExpression(value)
    } catch {
      return {
        minute: '0',
        hour: '9',
        dayOfMonth: '*',
        month: '*',
        dayOfWeek: '*',
      }
    }
  })
  const [customExpression, setCustomExpression] = useState(value)
  const [isValid, setIsValid] = useState(true)

  // Update parent when schedule changes
  useEffect(() => {
    if (mode === 'custom' || mode === 'preset') {
      const expression = buildCronExpression(schedule)
      onChange(expression)
    }
  }, [schedule, mode, onChange])

  // Validate custom expression
  useEffect(() => {
    if (mode === 'advanced') {
      const valid = isValidCronExpression(customExpression)
      setIsValid(valid)
      if (valid) {
        onChange(customExpression)
      }
    }
  }, [customExpression, mode, onChange])

  const handlePresetSelect = (expression: string) => {
    try {
      const newSchedule = parseCronExpression(expression)
      setSchedule(newSchedule)
      setCustomExpression(expression)
    } catch (error) {
      console.error('Failed to parse preset:', error)
    }
  }

  const updateSchedulePart = (part: keyof CronSchedule, value: string) => {
    setSchedule((prev) => ({ ...prev, [part]: value }))
  }

  const description = cronToHumanReadable(
    mode === 'advanced' ? customExpression : buildCronExpression(schedule)
  )

  return (
    <div className={className}>
      <Tabs value={mode} onValueChange={(v) => setMode(v as typeof mode)}>
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="preset">Presets</TabsTrigger>
          <TabsTrigger value="custom">Custom</TabsTrigger>
          <TabsTrigger value="advanced">Advanced</TabsTrigger>
        </TabsList>

        {/* Presets Tab */}
        <TabsContent value="preset" className="space-y-4">
          <div className="grid gap-2">
            {CRON_PRESETS.map((preset) => (
              <Button
                key={preset.expression}
                variant={
                  buildCronExpression(schedule) === preset.expression
                    ? 'default'
                    : 'outline'
                }
                className="justify-start h-auto py-3 px-4"
                onClick={() => handlePresetSelect(preset.expression)}
              >
                <div className="text-left">
                  <div className="font-medium">{preset.label}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    {preset.description}
                  </div>
                </div>
              </Button>
            ))}
          </div>
        </TabsContent>

        {/* Custom Builder Tab */}
        <TabsContent value="custom" className="space-y-4">
          <div className="grid gap-4">
            {/* Time */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Hour</Label>
                <Select
                  value={schedule.hour}
                  onValueChange={(v) => updateSchedulePart('hour', v)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="*">Every hour</SelectItem>
                    <SelectItem value="*/2">Every 2 hours</SelectItem>
                    <SelectItem value="*/6">Every 6 hours</SelectItem>
                    {HOURS.map((h) => (
                      <SelectItem key={h} value={String(h)}>
                        {h.toString().padStart(2, '0')}:00
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>Minute</Label>
                <Select
                  value={schedule.minute}
                  onValueChange={(v) => updateSchedulePart('minute', v)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {MINUTES.map((m) => (
                      <SelectItem key={m} value={String(m)}>
                        :{m.toString().padStart(2, '0')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Day of Week */}
            <div className="space-y-2">
              <Label>Day of Week</Label>
              <Select
                value={schedule.dayOfWeek}
                onValueChange={(v) => updateSchedulePart('dayOfWeek', v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="*">Every day</SelectItem>
                  <SelectItem value="1-5">Weekdays (Mon-Fri)</SelectItem>
                  <SelectItem value="0,6">Weekends (Sat-Sun)</SelectItem>
                  {DAYS_OF_WEEK.map((day) => (
                    <SelectItem key={day.value} value={day.value}>
                      {day.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Day of Month */}
            <div className="space-y-2">
              <Label>Day of Month</Label>
              <Select
                value={schedule.dayOfMonth}
                onValueChange={(v) => updateSchedulePart('dayOfMonth', v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="*">Every day</SelectItem>
                  <SelectItem value="1">1st of month</SelectItem>
                  <SelectItem value="15">15th of month</SelectItem>
                  <SelectItem value="L">Last day of month</SelectItem>
                  {DAYS_OF_MONTH.map((d) => (
                    <SelectItem key={d} value={String(d)}>
                      {d}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </TabsContent>

        {/* Advanced Tab */}
        <TabsContent value="advanced" className="space-y-4">
          <div className="space-y-2">
            <Label>Cron Expression</Label>
            <Input
              value={customExpression}
              onChange={(e) => setCustomExpression(e.target.value)}
              placeholder="0 9 * * *"
              className={!isValid ? 'border-red-500' : ''}
            />
            {!isValid && (
              <p className="text-xs text-red-500">Invalid cron expression</p>
            )}
          </div>

          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription className="text-xs">
              Format: <code className="font-mono">minute hour day month weekday</code>
              <br />
              Use <code className="font-mono">*</code> for any value,{' '}
              <code className="font-mono">*/n</code> for every n units
            </AlertDescription>
          </Alert>
        </TabsContent>
      </Tabs>

      {/* Human-readable description */}
      <Alert className="mt-4">
        <Clock className="h-4 w-4" />
        <AlertDescription className="font-medium">{description}</AlertDescription>
      </Alert>

      {/* Show expression */}
      <div className="mt-2 text-xs text-muted-foreground font-mono">
        Expression: {mode === 'advanced' ? customExpression : buildCronExpression(schedule)}
      </div>
    </div>
  )
}
