import { AlertTriangle, CheckCircle, Copy, GitCompare, Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useToast } from '@/hooks/use-toast'
import {
  CompareConnectionSchemas,
  CompareWithSnapshot,
  CreateSchemaSnapshot,
  DeleteSchemaSnapshot,
  GenerateMigrationSQL,
  GenerateMigrationSQLFromSnapshot,
  ListSchemaSnapshots,
} from '../../wailsjs/go/main/App'
import { schemadiff } from '../../wailsjs/go/models'
import { useConnectionStore } from '@/store/connection-store'

// Map Wails model status to our display change types
type ChangeType = 'added' | 'modified' | 'deleted'

function mapStatus(status: string): ChangeType {
  switch (status?.toLowerCase()) {
    case 'added':
    case 'created':
    case 'new':
      return 'added'
    case 'deleted':
    case 'removed':
    case 'dropped':
      return 'deleted'
    default:
      return 'modified'
  }
}

export function SchemaDiff() {
  const { toast } = useToast()
  const connections = useConnectionStore((state) => state.connections)
  const [snapshots, setSnapshots] = useState<schemadiff.SnapshotMetadata[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Create snapshot dialog
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [newSnapshotName, setNewSnapshotName] = useState('')
  const [selectedConnectionId, setSelectedConnectionId] = useState('')
  const [creating, setCreating] = useState(false)

  // Delete confirmation
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [snapshotToDelete, setSnapshotToDelete] = useState<schemadiff.SnapshotMetadata | null>(null)

  // Comparison
  const [sourceType, setSourceType] = useState<'connection' | 'snapshot'>('connection')
  const [targetType, setTargetType] = useState<'connection' | 'snapshot'>('connection')
  const [sourceId, setSourceId] = useState('')
  const [targetId, setTargetId] = useState('')
  const [comparing, setComparing] = useState(false)
  const [comparisonResult, setComparisonResult] = useState<schemadiff.SchemaDiff | null>(null)

  // Migration generation
  const [allowDestructive, setAllowDestructive] = useState(false)
  const [generatedSQL, setGeneratedSQL] = useState('')
  const [generating, setGenerating] = useState(false)
  const [expandedTables, setExpandedTables] = useState<Set<string>>(new Set())

  useEffect(() => {
    loadSnapshots()
  }, [])

  const loadSnapshots = async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await ListSchemaSnapshots()
      setSnapshots(result || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load snapshots')
    } finally {
      setLoading(false)
    }
  }

  const handleCreateSnapshot = async () => {
    if (!newSnapshotName.trim() || !selectedConnectionId) {
      toast({ title: 'Missing information', description: 'Please provide a name and select a connection', variant: 'destructive' })
      return
    }

    setCreating(true)
    try {
      await CreateSchemaSnapshot(selectedConnectionId, newSnapshotName.trim())
      toast({ title: 'Snapshot created', duration: 2000 })
      setCreateDialogOpen(false)
      setNewSnapshotName('')
      setSelectedConnectionId('')
      await loadSnapshots()
    } catch (err) {
      const errMsg = typeof err === 'string'
        ? err
        : err instanceof Error
          ? err.message
          : (err as { message?: string })?.message || JSON.stringify(err) || 'Unknown error'
      console.error('Failed to create snapshot:', err)
      toast({
        title: 'Failed to create snapshot',
        description: errMsg,
        variant: 'destructive',
      })
    } finally {
      setCreating(false)
    }
  }

  const handleDeleteSnapshot = async () => {
    if (!snapshotToDelete) return

    try {
      await DeleteSchemaSnapshot(snapshotToDelete.id)
      toast({ title: 'Snapshot deleted', variant: 'destructive', duration: 2000 })
      setDeleteDialogOpen(false)
      setSnapshotToDelete(null)
      await loadSnapshots()
    } catch (err) {
      const errMsg = typeof err === 'string'
        ? err
        : err instanceof Error
          ? err.message
          : (err as { message?: string })?.message || JSON.stringify(err) || 'Unknown error'
      console.error('Failed to delete snapshot:', err)
      toast({
        title: 'Failed to delete snapshot',
        description: errMsg,
        variant: 'destructive',
      })
    }
  }

  const handleCompare = async () => {
    if (!sourceId || !targetId) {
      toast({ title: 'Missing selection', description: 'Please select both source and target', variant: 'destructive' })
      return
    }

    setComparing(true)
    setComparisonResult(null)
    setGeneratedSQL('')
    try {
      let result
      if (sourceType === 'connection' && targetType === 'connection') {
        result = await CompareConnectionSchemas(sourceId, targetId)
      } else if (sourceType === 'connection' && targetType === 'snapshot') {
        result = await CompareWithSnapshot(sourceId, targetId)
      } else if (sourceType === 'snapshot' && targetType === 'connection') {
        result = await CompareWithSnapshot(targetId, sourceId)
      } else {
        toast({ title: 'Invalid comparison', description: 'Cannot compare two snapshots', variant: 'destructive' })
        return
      }

      setComparisonResult(result)
      toast({ title: 'Comparison complete', duration: 2000 })
    } catch (err) {
      // Wails errors can come as strings, Error objects, or objects with message property
      const errMsg = typeof err === 'string'
        ? err
        : err instanceof Error
          ? err.message
          : (err as { message?: string })?.message || JSON.stringify(err) || 'Unknown error'
      console.error('Schema comparison failed:', err)
      toast({
        title: 'Comparison failed',
        description: errMsg,
        variant: 'destructive',
      })
    } finally {
      setComparing(false)
    }
  }

  const handleGenerateMigration = async () => {
    if (!sourceId || !targetId) return

    setGenerating(true)
    try {
      let result: schemadiff.MigrationScript
      if (sourceType === 'connection' && targetType === 'connection') {
        result = await GenerateMigrationSQL(sourceId, targetId, allowDestructive)
      } else if (sourceType === 'connection' && targetType === 'snapshot') {
        result = await GenerateMigrationSQLFromSnapshot(sourceId, targetId, allowDestructive)
      } else if (sourceType === 'snapshot' && targetType === 'connection') {
        result = await GenerateMigrationSQLFromSnapshot(targetId, sourceId, allowDestructive)
      } else {
        toast({ title: 'Invalid migration', description: 'Cannot generate migration between two snapshots', variant: 'destructive' })
        return
      }

      // Extract the SQL string from the MigrationScript object
      setGeneratedSQL(result.sql || '')
      toast({ title: 'Migration SQL generated', duration: 2000 })
    } catch (err) {
      const errMsg = typeof err === 'string'
        ? err
        : err instanceof Error
          ? err.message
          : (err as { message?: string })?.message || JSON.stringify(err) || 'Unknown error'
      console.error('Migration generation failed:', err)
      toast({
        title: 'Failed to generate migration',
        description: errMsg,
        variant: 'destructive',
      })
    } finally {
      setGenerating(false)
    }
  }

  const handleCopySQL = async () => {
    try {
      await navigator.clipboard.writeText(generatedSQL)
      toast({ title: 'Copied to clipboard', duration: 2000 })
    } catch (err) {
      toast({ title: 'Failed to copy', variant: 'destructive', duration: 2000 })
    }
  }

  const toggleTableExpansion = (tableName: string) => {
    setExpandedTables((prev) => {
      const next = new Set(prev)
      if (next.has(tableName)) {
        next.delete(tableName)
      } else {
        next.add(tableName)
      }
      return next
    })
  }

  const getChangeIcon = (changeType: ChangeType) => {
    switch (changeType) {
      case 'added':
        return <Plus className="h-4 w-4 text-green-600" />
      case 'modified':
        return <AlertTriangle className="h-4 w-4 text-yellow-600" />
      case 'deleted':
        return <Trash2 className="h-4 w-4 text-red-600" />
    }
  }

  const getChangeBadgeVariant = (changeType: ChangeType) => {
    switch (changeType) {
      case 'added':
        return 'default' as const
      case 'modified':
        return 'secondary' as const
      case 'deleted':
        return 'destructive' as const
    }
  }

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="flex-1 space-y-6 overflow-y-auto p-6">
        {/* Header */}
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Schema Diff</h1>
            <p className="text-sm text-muted-foreground">
              Compare database schemas and generate migration scripts
            </p>
          </div>
          <Button onClick={() => setCreateDialogOpen(true)} disabled={loading}>
            <Plus className="mr-2 h-4 w-4" /> New Snapshot
          </Button>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertTitle>Unable to load snapshots</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="grid gap-6 lg:grid-cols-2">
          {/* Snapshots List */}
          <Card>
            <CardHeader>
              <CardTitle>Schema Snapshots</CardTitle>
              <CardDescription>
                Saved snapshots of database schemas for comparison
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading && snapshots.length === 0 ? (
                <div className="space-y-2">
                  <Skeleton className="h-16 w-full" />
                  <Skeleton className="h-16 w-full" />
                  <Skeleton className="h-16 w-full" />
                </div>
              ) : snapshots.length === 0 ? (
                <EmptyState
                  icon={GitCompare}
                  title="No snapshots yet"
                  description="Create your first schema snapshot to start comparing"
                />
              ) : (
                <div className="space-y-2 max-h-96 overflow-y-auto">
                  {snapshots.map((snapshot) => (
                    <div
                      key={snapshot.id}
                      className="flex items-center justify-between rounded-md border p-3 hover:bg-muted/50"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium truncate">{snapshot.name}</span>
                          <Badge variant="outline">{snapshot.table_count} tables</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground truncate">
                          {snapshot.database_type} • {snapshot.connection_id}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {snapshot.created_at ? new Date(snapshot.created_at).toLocaleString() : 'Unknown date'}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setSnapshotToDelete(snapshot)
                          setDeleteDialogOpen(true)
                        }}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Comparison Controls */}
          <Card>
            <CardHeader>
              <CardTitle>Compare Schemas</CardTitle>
              <CardDescription>
                Select a source and target to compare
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Source Selection */}
              <div className="space-y-2">
                <Label>Source</Label>
                <div className="flex gap-2">
                  <Select value={sourceType} onValueChange={(v) => setSourceType(v as 'connection' | 'snapshot')}>
                    <SelectTrigger className="w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="connection">Connection</SelectItem>
                      <SelectItem value="snapshot">Snapshot</SelectItem>
                    </SelectContent>
                  </Select>
                  <Select value={sourceId} onValueChange={setSourceId}>
                    <SelectTrigger className="flex-1">
                      <SelectValue placeholder={`Select ${sourceType}`} />
                    </SelectTrigger>
                    <SelectContent>
                      {sourceType === 'connection'
                        ? connections
                            .filter((conn) => conn.isConnected && conn.sessionId)
                            .map((conn) => (
                              <SelectItem key={conn.id} value={conn.sessionId!}>
                                {conn.name}
                              </SelectItem>
                            ))
                        : snapshots.map((snap) => (
                            <SelectItem key={snap.id} value={snap.id}>
                              {snap.name}
                            </SelectItem>
                          ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Target Selection */}
              <div className="space-y-2">
                <Label>Target</Label>
                <div className="flex gap-2">
                  <Select value={targetType} onValueChange={(v) => setTargetType(v as 'connection' | 'snapshot')}>
                    <SelectTrigger className="w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="connection">Connection</SelectItem>
                      <SelectItem value="snapshot">Snapshot</SelectItem>
                    </SelectContent>
                  </Select>
                  <Select value={targetId} onValueChange={setTargetId}>
                    <SelectTrigger className="flex-1">
                      <SelectValue placeholder={`Select ${targetType}`} />
                    </SelectTrigger>
                    <SelectContent>
                      {targetType === 'connection'
                        ? connections
                            .filter((conn) => conn.isConnected && conn.sessionId)
                            .map((conn) => (
                              <SelectItem key={conn.id} value={conn.sessionId!}>
                                {conn.name}
                              </SelectItem>
                            ))
                        : snapshots.map((snap) => (
                            <SelectItem key={snap.id} value={snap.id}>
                              {snap.name}
                            </SelectItem>
                          ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <Button
                onClick={handleCompare}
                disabled={!sourceId || !targetId || comparing}
                className="w-full"
              >
                <GitCompare className="mr-2 h-4 w-4" />
                {comparing ? 'Comparing...' : 'Compare'}
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* Comparison Results */}
        {comparisonResult && (
          <Card>
            <CardHeader>
              <CardTitle>Schema Differences</CardTitle>
              <CardDescription>
                Changes detected between source and target schemas
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {comparisonResult.tables.length === 0 ? (
                <div className="flex items-center gap-2 p-4 rounded-md bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800">
                  <CheckCircle className="h-5 w-5 text-green-600" />
                  <span className="text-sm text-green-800 dark:text-green-200">
                    No differences found - schemas are identical
                  </span>
                </div>
              ) : (
                <>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">
                      {comparisonResult.tables.length} table{comparisonResult.tables.length === 1 ? '' : 's'} changed
                    </span>
                    <div className="flex items-center gap-2">
                      <Switch
                        id="destructive"
                        checked={allowDestructive}
                        onCheckedChange={setAllowDestructive}
                      />
                      <Label htmlFor="destructive" className="text-sm cursor-pointer">
                        Allow destructive changes
                      </Label>
                    </div>
                  </div>

                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-12"></TableHead>
                        <TableHead>Table Name</TableHead>
                        <TableHead>Change Type</TableHead>
                        <TableHead className="w-20"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {comparisonResult.tables.map((table) => {
                        const changeType = mapStatus(table.status)
                        return (
                          <>
                            <TableRow key={table.name}>
                              <TableCell>{getChangeIcon(changeType)}</TableCell>
                              <TableCell className="font-medium">
                                {table.schema ? `${table.schema}.${table.name}` : table.name}
                              </TableCell>
                              <TableCell>
                                <Badge variant={getChangeBadgeVariant(changeType)}>
                                  {changeType}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {table.columns && table.columns.length > 0 && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => toggleTableExpansion(table.name)}
                                  >
                                    {expandedTables.has(table.name) ? 'Hide' : 'Show'} columns
                                  </Button>
                                )}
                              </TableCell>
                            </TableRow>
                            {expandedTables.has(table.name) && table.columns && (
                              <TableRow>
                                <TableCell colSpan={4} className="bg-muted/30">
                                  <div className="pl-8 space-y-2">
                                    {table.columns.map((col) => {
                                      const colChangeType = mapStatus(col.status)
                                      return (
                                        <div key={col.name} className="flex items-center gap-3 text-sm">
                                          {getChangeIcon(colChangeType)}
                                          <span className="font-medium">{col.name}</span>
                                          <Badge variant="outline" className="text-xs">
                                            {colChangeType}
                                          </Badge>
                                          {col.old_type && col.new_type && (
                                            <span className="text-muted-foreground">
                                              {col.old_type} → {col.new_type}
                                            </span>
                                          )}
                                          {col.old_type && !col.new_type && (
                                            <span className="text-muted-foreground">{col.old_type}</span>
                                          )}
                                          {!col.old_type && col.new_type && (
                                            <span className="text-muted-foreground">{col.new_type}</span>
                                          )}
                                        </div>
                                      )
                                    })}
                                  </div>
                                </TableCell>
                              </TableRow>
                            )}
                          </>
                        )
                      })}
                    </TableBody>
                  </Table>

                  <div className="flex justify-end">
                    <Button
                      onClick={handleGenerateMigration}
                      disabled={generating}
                    >
                      {generating ? 'Generating...' : 'Generate Migration SQL'}
                    </Button>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        )}

        {/* Generated SQL */}
        {generatedSQL && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Generated Migration SQL</CardTitle>
                <Button variant="outline" size="sm" onClick={handleCopySQL}>
                  <Copy className="mr-2 h-4 w-4" />
                  Copy to Clipboard
                </Button>
              </div>
              <CardDescription>
                Review and execute these SQL statements to migrate the schema
              </CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="p-4 rounded-md bg-muted text-sm overflow-x-auto">
                <code>{generatedSQL}</code>
              </pre>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Create Snapshot Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Schema Snapshot</DialogTitle>
            <DialogDescription>
              Save a snapshot of the current database schema for comparison
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="snapshot-name">Snapshot Name</Label>
              <Input
                id="snapshot-name"
                placeholder="e.g., Production Schema 2025-01-15"
                value={newSnapshotName}
                onChange={(e) => setNewSnapshotName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="snapshot-connection">Connection</Label>
              <Select value={selectedConnectionId} onValueChange={setSelectedConnectionId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select connection" />
                </SelectTrigger>
                <SelectContent>
                  {connections
                    .filter((conn) => conn.isConnected && conn.sessionId)
                    .map((conn) => (
                      <SelectItem key={conn.id} value={conn.sessionId!}>
                        {conn.name}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateSnapshot} disabled={creating}>
              {creating ? 'Creating...' : 'Create Snapshot'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        open={deleteDialogOpen}
        title="Delete Snapshot?"
        description={`"${snapshotToDelete?.name}" will be permanently deleted. This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDeleteSnapshot}
        onCancel={() => setDeleteDialogOpen(false)}
      />
    </div>
  )
}

export default SchemaDiff
