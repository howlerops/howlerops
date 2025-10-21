/**
 * SQL Preview Component for Visual Query Builder
 * Shows generated SQL and handles manual SQL editing
 */

import { useState, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Copy, AlertTriangle, CheckCircle, Edit3, Eye } from 'lucide-react'
import { SqlPreviewProps } from './types'
import { generateSQL } from '@/lib/query-ir'

export function SqlPreview({
  queryIR,
  dialect,
  manualSQL,
  onSQLChange
}: SqlPreviewProps) {
  const [isManualMode, setIsManualMode] = useState(false)
  const generatedSQL = useMemo(() => {
    try {
      return generateSQL(queryIR, dialect)
    } catch (error) {
      console.error('Failed to generate SQL:', error)
      return '-- Error generating SQL'
    }
  }, [queryIR, dialect])

  const hasChanges = useMemo(() => {
    if (!manualSQL) {
      return false
    }
    return manualSQL.trim() !== generatedSQL.trim()
  }, [manualSQL, generatedSQL])

  // Handle manual SQL toggle
  const handleManualToggle = () => {
    if (isManualMode) {
      // Switching back to visual mode
      setIsManualMode(false)
      if (onSQLChange) {
        onSQLChange(generatedSQL)
      }
    } else {
      // Switching to manual mode
      setIsManualMode(true)
      if (onSQLChange && !manualSQL) {
        onSQLChange(generatedSQL)
      }
    }
  }

  // Handle manual SQL change
  const handleManualSQLChange = (value: string) => {
    if (onSQLChange) {
      onSQLChange(value)
    }
  }

  // Copy SQL to clipboard
  const copyToClipboard = async (sql: string) => {
    try {
      await navigator.clipboard.writeText(sql)
    } catch (error) {
      console.error('Failed to copy to clipboard:', error)
    }
  }

  // Get status badge
  const getStatusBadge = () => {
    if (isManualMode) {
      return (
        <Badge variant="outline" className="text-orange-600 border-orange-200">
          <Edit3 className="w-3 h-3 mr-1" />
          Manual
        </Badge>
      )
    } else if (hasChanges) {
      return (
        <Badge variant="outline" className="text-yellow-600 border-yellow-200">
          <AlertTriangle className="w-3 h-3 mr-1" />
          Modified
        </Badge>
      )
    } else {
      return (
        <Badge variant="outline" className="text-green-600 border-green-200">
          <CheckCircle className="w-3 h-3 mr-1" />
          Generated
        </Badge>
      )
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">SQL Preview</h3>
        <div className="flex items-center space-x-2">
          {getStatusBadge()}
          <Button
            variant="outline"
            size="sm"
            onClick={() => copyToClipboard(isManualMode ? (manualSQL || '') : generatedSQL)}
          >
            <Copy className="w-4 h-4 mr-1" />
            Copy
          </Button>
        </div>
      </div>

      {/* Manual mode warning */}
      {isManualMode && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            Manual SQL mode is active. Changes to the visual query builder will not affect this SQL.
            Switch back to visual mode to sync with the query builder.
          </AlertDescription>
        </Alert>
      )}

      {/* Modified SQL warning */}
      {hasChanges && !isManualMode && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            The SQL has been modified manually. Switching to visual mode will overwrite your changes.
          </AlertDescription>
        </Alert>
      )}

      <Tabs value={isManualMode ? 'manual' : 'generated'} onValueChange={(value) => {
        if (value === 'manual' && !isManualMode) {
          handleManualToggle()
        } else if (value === 'generated' && isManualMode) {
          handleManualToggle()
        }
      }}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="generated" className="flex items-center space-x-1">
            <Eye className="w-4 h-4" />
            <span>Generated</span>
          </TabsTrigger>
          <TabsTrigger value="manual" className="flex items-center space-x-1">
            <Edit3 className="w-4 h-4" />
            <span>Manual</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="generated" className="mt-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Generated SQL</CardTitle>
            </CardHeader>
            <CardContent>
              <pre className="text-xs bg-muted p-3 rounded-md overflow-x-auto">
                {generatedSQL}
              </pre>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="manual" className="mt-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Manual SQL</CardTitle>
            </CardHeader>
            <CardContent>
              <textarea
                value={manualSQL || ''}
                onChange={(e) => handleManualSQLChange(e.target.value)}
                placeholder="Enter your SQL query here..."
                className="w-full min-h-32 font-mono text-xs p-3 border rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Query Info */}
      <div className="text-xs text-muted-foreground space-y-1">
        <div>Dialect: {dialect.toUpperCase()}</div>
        <div>Table: {queryIR.from.schema}.{queryIR.from.table}</div>
        <div>Columns: {queryIR.select.length}</div>
        {queryIR.joins?.length && <div>Joins: {queryIR.joins.length}</div>}
        {queryIR.where && <div>Filters: Applied</div>}
        {queryIR.orderBy?.length && <div>Sort: {queryIR.orderBy.length} column(s)</div>}
        {queryIR.limit && <div>Limit: {queryIR.limit}</div>}
        {queryIR.offset && <div>Offset: {queryIR.offset}</div>}
      </div>
    </div>
  )
}
