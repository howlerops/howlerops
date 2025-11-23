import { AlertCircle, Code, Lock, Unlock, Wand2 } from 'lucide-react'
import React, { useEffect, useState } from 'react'

import { QueryBuilder } from '@/components/reports/query-builder'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import type { QueryBuilderState, ReportQueryMode } from '@/types/reports'

interface QueryModeSwitcherProps {
  mode: ReportQueryMode
  connectionId?: string
  sql?: string
  builderState?: QueryBuilderState
  onChange: (updates: { mode?: ReportQueryMode; sql?: string; builderState?: QueryBuilderState }) => void
  disabled?: boolean
}

export function QueryModeSwitcher({
  mode,
  connectionId,
  sql,
  builderState,
  onChange,
  disabled,
}: QueryModeSwitcherProps) {
  const [sqlLocked, setSqlLocked] = useState(true)
  const [generatedSQL, setGeneratedSQL] = useState<string>('')
  const [canSwitchToVisual, setCanSwitchToVisual] = useState(true)

  // Initialize builder state if switching from SQL to builder
  const [internalBuilderState, setInternalBuilderState] = useState<QueryBuilderState>(
    builderState || {
      dataSource: connectionId || '',
      table: '',
      columns: [],
      joins: [],
      filters: [],
      groupBy: [],
      orderBy: [],
    }
  )

  // Update internal state when props change
  useEffect(() => {
    if (builderState) {
      setInternalBuilderState(builderState)
    }
  }, [builderState])

  // Determine if we can switch to visual mode
  useEffect(() => {
    // Can switch to visual if:
    // 1. Currently have a valid builder state, OR
    // 2. SQL was generated from visual builder (tracked by having builderState)
    const hasValidBuilderState = builderState && builderState.table !== ''
    setCanSwitchToVisual(!!hasValidBuilderState)
  }, [builderState, sql])

  const handleModeChange = (newMode: ReportQueryMode) => {
    if (newMode === mode) return

    if (newMode === 'sql') {
      // Switching from visual to SQL
      // Keep the generated SQL visible
      onChange({ mode: newMode, sql: generatedSQL || sql })
      setSqlLocked(true)
    } else if (newMode === 'builder') {
      // Switching from SQL to visual
      if (!canSwitchToVisual) {
        // Show warning but don't switch
        return
      }
      onChange({ mode: newMode })
    }
  }

  const handleBuilderStateChange = (newState: QueryBuilderState) => {
    setInternalBuilderState(newState)
    onChange({ builderState: newState })
  }

  const handleGeneratedSQL = (sql: string) => {
    setGeneratedSQL(sql)
    onChange({ sql })
  }

  const handleUnlockSQL = () => {
    setSqlLocked(false)
    setCanSwitchToVisual(false) // Once manually edited, can't go back to visual
  }

  const handleSQLChange = (newSQL: string) => {
    onChange({ sql: newSQL })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Query Configuration</CardTitle>
        <CardDescription>Build your query visually or write SQL directly</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={mode} onValueChange={handleModeChange as (value: string) => void}>
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="builder" className="gap-2">
              <Wand2 className="h-4 w-4" />
              Visual Builder
            </TabsTrigger>
            <TabsTrigger value="sql" className="gap-2">
              <Code className="h-4 w-4" />
              SQL Editor
            </TabsTrigger>
          </TabsList>

          <TabsContent value="builder" className="mt-6">
            <QueryBuilder
              state={internalBuilderState}
              onChange={handleBuilderStateChange}
              onGenerateSQL={handleGeneratedSQL}
              disabled={disabled}
            />
          </TabsContent>

          <TabsContent value="sql" className="mt-6 space-y-4">
            {/* Show generated SQL info */}
            {generatedSQL && sqlLocked && (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  This SQL was generated from the visual builder. Click &quot;Edit SQL&quot; to modify it directly, but
                  note that you won&apos;t be able to return to visual mode after editing.
                </AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium">SQL Query</label>
                {sqlLocked && (
                  <Button variant="outline" size="sm" onClick={handleUnlockSQL} disabled={disabled}>
                    <Unlock className="mr-2 h-4 w-4" />
                    Edit SQL
                  </Button>
                )}
                {!sqlLocked && (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Lock className="h-4 w-4" />
                    Manual SQL editing enabled
                  </div>
                )}
              </div>

              <Textarea
                className="min-h-[300px] font-mono text-sm"
                value={generatedSQL || sql || ''}
                onChange={(e) => handleSQLChange(e.target.value)}
                disabled={disabled || sqlLocked}
                placeholder="SELECT * FROM table_name WHERE condition = 'value'"
              />

              {!canSwitchToVisual && mode === 'sql' && !sqlLocked && (
                <Alert variant="warning">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    Manual SQL edits prevent switching back to visual builder mode. To use the visual builder again,
                    you&apos;ll need to create a new query.
                  </AlertDescription>
                </Alert>
              )}
            </div>

            {/* SQL Stats */}
            {(generatedSQL || sql) && (
              <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <span>{(generatedSQL || sql || '').split('\n').length} lines</span>
                <span>{(generatedSQL || sql || '').length} characters</span>
              </div>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}
