import { AlertCircle, BarChart3, Clock, Database, RotateCcw, Wand2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { DataProcessingIndicator } from '@/components/data-processing-indicator'
import { QueryLoadingIndicator } from '@/components/query-loading-indicator'
import { QueryResultsTable } from '@/components/query-results-table'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAIConfig } from '@/store/ai-store'
import { useConnectionStore } from '@/store/connection-store'
import { useQueryStore } from '@/store/query-store'

export interface ResultsPanelProps {
  onFixWithAI?: (error: string, query: string) => void
  onPageChange?: (tabId: string, limit: number, offset: number) => Promise<void>
}

export function ResultsPanel({ onFixWithAI, onPageChange }: ResultsPanelProps = {}) {
  const tabs = useQueryStore((state) => state.tabs)
  const activeTabId = useQueryStore((state) => state.activeTabId)
  const results = useQueryStore((state) => state.results)
  const connections = useConnectionStore((state) => state.connections)

  const activeTab = tabs.find((tab) => tab.id === activeTabId)
  const tabResults = results.filter((result) => result.tabId === activeTabId)
  const latestResult = tabResults.length > 0 ? tabResults[tabResults.length - 1] : null
  const hasHistory = tabResults.length > 1

  const { isEnabled: aiEnabled } = useAIConfig()
  const [showHistory, setShowHistory] = useState(false)

  useEffect(() => {
    if (!hasHistory && showHistory) {

      setShowHistory(false)
    }
  }, [hasHistory, showHistory])

   
  const numericColumns = useMemo(() => {
    if (!latestResult) return []
    return latestResult.columns.filter((column) => {
      return latestResult.rows.some((row) => {
        const value = row[column]
        if (value === null || value === undefined) return false
        const numeric = typeof value === 'number' ? value : Number(value)
        return !Number.isNaN(numeric)
      })
    })
  }, [latestResult])
   

  const [selectedMetric, setSelectedMetric] = useState<string | null>(null)

  useEffect(() => {
    if (numericColumns.length === 0) {

      setSelectedMetric(null)
    } else if (!selectedMetric || !numericColumns.includes(selectedMetric)) {
      setSelectedMetric(numericColumns[0])
    }
  }, [numericColumns, selectedMetric])

  const metricValues = useMemo(() => {
    if (!latestResult || !selectedMetric) return []
    return latestResult.rows
      .map((row) => {
        const raw = row[selectedMetric]
        if (raw === null || raw === undefined) return null
        const numeric = typeof raw === 'number' ? raw : Number(raw)
        return Number.isNaN(numeric) ? null : numeric
      })
      .filter((value): value is number => value !== null)
  }, [latestResult, selectedMetric])  

  const metricStats = useMemo(() => {
    if (metricValues.length === 0) {
      return { min: 0, max: 0, avg: 0 }
    }
    const min = Math.min(...metricValues)
    const max = Math.max(...metricValues)
    const sum = metricValues.reduce((acc, value) => acc + value, 0)
    return { min, max, avg: sum / metricValues.length }
  }, [metricValues])

  const chartValues = useMemo(() => {
    const maxBars = 40
    if (metricValues.length <= maxBars) return metricValues
    const step = Math.ceil(metricValues.length / maxBars)
    const sampled: number[] = []
    for (let i = 0; i < metricValues.length; i += step) {
      sampled.push(metricValues[i])
    }
    return sampled
  }, [metricValues])

  const formatNumber = (value: number) => {
    if (Math.abs(value) >= 1000) {
      return value.toLocaleString()
    }
    return Number.isInteger(value) ? value.toString() : value.toFixed(2)
  }

  if (!activeTab || !latestResult) {
    return (
      <div className="flex-1 flex w-full items-center justify-center">
        <div className="text-center text-muted-foreground">
          <RotateCcw className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>No query results yet</p>
          <p className="text-sm">Execute a query to see results here</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 flex min-h-0 w-full flex-col overflow-hidden">
      <Tabs defaultValue="results" className="flex flex-1 min-h-0 flex-col">
        <TabsList className="flex h-10 shrink-0 items-stretch border-b bg-background px-1">
          <TabsTrigger
            value="results"
            className="flex-1 select-none whitespace-nowrap px-3 text-sm font-medium leading-none text-muted-foreground transition-colors data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=inactive]:border-b-2 data-[state=inactive]:border-transparent"
          >
            Results
          </TabsTrigger>
          <TabsTrigger
            value="visualizations"
            disabled={!latestResult || !!latestResult.error || numericColumns.length === 0}
            className="flex-1 select-none whitespace-nowrap px-3 text-sm font-medium leading-none text-muted-foreground transition-colors data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=inactive]:border-b-2 data-[state=inactive]:border-transparent disabled:opacity-50"
          >
            Visualizations
          </TabsTrigger>
          <TabsTrigger
            value="messages"
            className="flex-1 select-none whitespace-nowrap px-3 text-sm font-medium leading-none text-muted-foreground transition-colors data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=inactive]:border-b-2 data-[state=inactive]:border-transparent"
          >
            Messages
          </TabsTrigger>
          <TabsTrigger
            value="execution"
            className="flex-1 select-none whitespace-nowrap px-3 text-sm font-medium leading-none text-muted-foreground transition-colors data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=inactive]:border-b-2 data-[state=inactive]:border-transparent"
          >
            Execution
          </TabsTrigger>
        </TabsList>

        <TabsContent
          value="results"
          className="flex flex-1 min-h-0 flex-col mt-0 overflow-hidden data-[state=inactive]:hidden"
        >
          {activeTab?.isExecuting && activeTab?.executionStartTime ? (
            <QueryLoadingIndicator
              startTime={activeTab.executionStartTime}
              query={activeTab.content}
            />
          ) : latestResult.isProcessing ? (
            <DataProcessingIndicator
              rowCount={latestResult.rowCount}
              progress={latestResult.processingProgress}
            />
          ) : latestResult.error ? (
            <div className="flex h-full items-center justify-center p-6">
              <Alert variant="destructive" className="max-w-lg text-left">
                <div className="flex items-start gap-3">
                  <AlertCircle className="h-5 w-5 mt-0.5 flex-shrink-0" />
                  <div className="flex-1">
                    <AlertTitle>Query failed</AlertTitle>
                    <AlertDescription className="mt-1 whitespace-pre-wrap">
                      {latestResult.error}
                    </AlertDescription>
                    {aiEnabled && onFixWithAI && latestResult.query && (
                      <div className="mt-3">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => onFixWithAI(latestResult.error!, latestResult.query)}
                          className="gap-2"
                        >
                          <Wand2 className="h-4 w-4" />
                          Fix with AI
                        </Button>
                      </div>
                    )}
                  </div>
                </div>
              </Alert>
            </div>
          ) : (
            <QueryResultsTable
              resultId={latestResult.id}
              columns={latestResult.columns}
              rows={latestResult.rows}
              originalRows={latestResult.originalRows}
              metadata={latestResult.editable}
              query={latestResult.query}
              connectionId={(() => {
                // Use the tab's connection if it exists and is connected
                if (activeTab?.connectionId) {
                  const tabConnection = connections.find(
                    (c) => c.id === activeTab.connectionId && c.isConnected
                  )
                  if (tabConnection) {
                    // Use sessionId if available (for backend connection manager), fallback to id
                    return tabConnection.sessionId || tabConnection.id
                  }
                }
                // Fallback to the result's connection ID - try to find sessionId for it
                if (latestResult.connectionId) {
                  const resultConnection = connections.find(
                    (c) => c.id === latestResult.connectionId && c.isConnected
                  )
                  if (resultConnection) {
                    return resultConnection.sessionId || resultConnection.id
                  }
                }
                return latestResult.connectionId
              })()}
              executionTimeMs={latestResult.executionTime}
              rowCount={latestResult.rowCount}
              executedAt={latestResult.timestamp}
              affectedRows={latestResult.affectedRows}
              isLarge={latestResult.isLarge}
              chunkingEnabled={latestResult.chunkingEnabled}
              displayMode={latestResult.displayMode}
              totalRows={latestResult.totalRows}
              hasMore={latestResult.hasMore}
              offset={latestResult.offset}
              onPageChange={onPageChange && activeTabId ? (limit, offset) => onPageChange(activeTabId, limit, offset) : undefined}
            />
          )}
        </TabsContent>

        <TabsContent
          value="visualizations"
          className="flex flex-1 min-h-0 flex-col overflow-auto px-3 py-2 data-[state=inactive]:hidden"
        >
          {!latestResult.error && numericColumns.length > 0 && selectedMetric && (
            <Card>
              <CardHeader className="pb-2">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex items-center gap-2">
                    <BarChart3 className="h-4 w-4" />
                    <CardTitle className="text-sm">Visualization</CardTitle>
                  </div>
                  <Select value={selectedMetric} onValueChange={setSelectedMetric}>
                    <SelectTrigger className="w-[200px]">
                      <SelectValue placeholder="Select column" />
                    </SelectTrigger>
                    <SelectContent>
                      {numericColumns.map((column) => (
                        <SelectItem key={column} value={column}>
                          {column}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-wrap gap-4 text-sm">
                  <div className="rounded border px-3 py-2">
                    <span className="text-xs text-muted-foreground">Min</span>
                    <div className="font-medium">{formatNumber(metricStats.min)}</div>
                  </div>
                  <div className="rounded border px-3 py-2">
                    <span className="text-xs text-muted-foreground">Avg</span>
                    <div className="font-medium">{formatNumber(metricStats.avg)}</div>
                  </div>
                  <div className="rounded border px-3 py-2">
                    <span className="text-xs text-muted-foreground">Max</span>
                    <div className="font-medium">{formatNumber(metricStats.max)}</div>
                  </div>
                </div>

                <div className="mt-2">
                  <div className="relative h-32 w-full overflow-hidden rounded border bg-muted/30">
                    <div className="absolute inset-0 flex items-end justify-start gap-1 px-3 pb-3">
                      {chartValues.map((value, index) => {
                        const max = metricStats.max || 1
                        const height = Math.max(4, (value / max) * 100)
                        return (
                          <div
                            key={`${selectedMetric}-bar-${index}`}
                            className="w-2 rounded-t bg-primary/70"
                            style={{ height: `${height}%` }}
                            title={`${formatNumber(value)}`}
                          />
                        )
                      })}
                    </div>
                  </div>
                  <p className="mt-2 text-xs text-muted-foreground">
                    Showing {chartValues.length} of {metricValues.length} samples for "{selectedMetric}".
                  </p>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent
          value="messages"
          className="flex flex-1 min-h-0 flex-col overflow-auto px-3 py-2 data-[state=inactive]:hidden"
        >
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Messages</CardTitle>
            </CardHeader>
            <CardContent>
              {tabResults.length === 0 ? (
                <div className="text-sm text-muted-foreground">
                  Run a query to see execution messages here.
                </div>
              ) : (
                <div className="space-y-2 text-sm">
                  {tabResults.map((result) => (
                    <div key={result.id} className="flex items-center space-x-2">
                      <span className="text-muted-foreground">
                        [{new Date(result.timestamp).toLocaleTimeString()}]
                      </span>
                      {result.error ? (
                        <span className="text-destructive">Error: {result.error}</span>
                      ) : (
                        <span className="text-primary">
                          {(() => {
                            const hasTabularData = result.columns.length > 0
                            const count = hasTabularData
                              ? (result.totalRows !== undefined ? result.totalRows : result.rowCount)
                              : result.affectedRows
                            const formattedCount = count.toLocaleString()
                            const noun = count === 1 ? 'row' : 'rows'
                            const verb = hasTabularData ? 'returned' : 'affected'
                            return `Query executed successfully. ${formattedCount} ${noun} ${verb}.`
                          })()}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent
          value="execution"
          className="flex flex-1 min-h-0 flex-col overflow-auto px-3 py-2 data-[state=inactive]:hidden"
        >
          <div className="space-y-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">Execution Summary</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 text-sm">
                <div>
                  <span className="text-xs text-muted-foreground">Run at</span>
                  <div className="font-medium">
                    {new Date(latestResult.timestamp).toLocaleString()}
                  </div>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">Duration</span>
                  <div className="font-medium">{latestResult.executionTime.toFixed(2)}ms</div>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    {latestResult.columns.length > 0 ? 'Total Rows' : 'Rows Affected'}
                  </span>
                  <div className="font-medium">
                    {latestResult.columns.length > 0
                      ? (latestResult.totalRows !== undefined ? latestResult.totalRows : latestResult.rowCount)
                      : latestResult.affectedRows}
                  </div>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">Status</span>
                  <div className={`font-medium ${latestResult.error ? 'text-destructive' : ''}`}>
                    {latestResult.error ? 'Failed' : 'Succeeded'}
                  </div>
                </div>
              </CardContent>
            </Card>

            {hasHistory && (
              <Card>
                <CardHeader className="pb-2 flex flex-row items-center justify-between">
                  <CardTitle className="text-sm">History</CardTitle>
                  <Button variant="ghost" size="sm" onClick={() => setShowHistory(!showHistory)}>
                    {showHistory ? 'Hide History' : 'Show History'}
                  </Button>
                </CardHeader>
                {showHistory && (
                  <CardContent className="space-y-3 max-h-60 overflow-auto">
                    {tabResults
                      .slice(0, -1)
                      .reverse()
                      .map((result, index) => (
                        <div key={result.id} className="rounded border p-3 text-xs space-y-2">
                          <div className="flex items-center justify-between text-muted-foreground">
                            <span>Run #{tabResults.length - index - 1}</span>
                            <span>{new Date(result.timestamp).toLocaleTimeString()}</span>
                          </div>
                          {result.error ? (
                            <div className="flex items-start gap-2 text-destructive">
                              <AlertCircle className="h-4 w-4 mt-0.5" />
                              <span>{result.error}</span>
                            </div>
                          ) : (
                            <div className="flex items-center gap-4 text-muted-foreground">
                              <div className="flex items-center gap-1">
                                <Database className="h-3 w-3" />
                                <span>
                                  {result.totalRows !== undefined ? result.totalRows : result.rowCount} rows
                                  {result.totalRows !== undefined && result.rowCount < result.totalRows && (
                                    <span className="text-xs ml-1">({result.rowCount} shown)</span>
                                  )}
                                </span>
                              </div>
                              <div className="flex items-center gap-1">
                                <Clock className="h-3 w-3" />
                                <span>{result.executionTime.toFixed(2)}ms</span>
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                  </CardContent>
                )}
              </Card>
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
