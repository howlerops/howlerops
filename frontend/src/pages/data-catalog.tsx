import {
  AlertCircle,
  BookOpen,
  ChevronRight,
  Database,
  Edit2,
  Eye,
  EyeOff,
  Filter,
  Plus,
  RefreshCw,
  Search,
  Shield,
  Tag,
  Tags,
  Trash2,
  User,
  X,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
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
import { ScrollArea } from '@/components/ui/scroll-area'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'
import { useConnectionStore } from '@/store/connection-store'
import {
  AssignTableSteward,
  CreateCatalogTag,
  CreateColumnCatalogEntry,
  CreateTableCatalogEntry,
  DeleteCatalogTag,
  GetCatalogStats,
  GetColumnCatalogEntry,
  GetTableCatalogEntry,
  ListCatalogTags,
  ListColumnCatalogEntries,
  ListTableCatalogEntries,
  MarkColumnAsPII,
  SearchCatalog,
  SyncCatalogFromConnection,
  UpdateColumnCatalogEntry,
  UpdateTableCatalogEntry,
} from '../../wailsjs/go/main/App'
import { catalog, main } from '../../wailsjs/go/models'

type ViewMode = 'tree' | 'table'

interface ExpandedState {
  [tableId: string]: boolean
}

const PII_TYPES = [
  { value: 'email', label: 'Email' },
  { value: 'phone', label: 'Phone Number' },
  { value: 'ssn', label: 'SSN' },
  { value: 'credit_card', label: 'Credit Card' },
  { value: 'address', label: 'Address' },
  { value: 'name', label: 'Full Name' },
  { value: 'dob', label: 'Date of Birth' },
  { value: 'ip_address', label: 'IP Address' },
  { value: 'other', label: 'Other' },
]

export function DataCatalog() {
  const { toast } = useToast()
  const { connections, connectToDatabase } = useConnectionStore()

  // State
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [viewMode, setViewMode] = useState<ViewMode>('tree')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedConnectionId, setSelectedConnectionId] = useState<string>('') // stored connection id
  const [schemaFilter, setSchemaFilter] = useState('__all__')
  const [tagFilter, setTagFilter] = useState<string[]>([])
  const [showPIIOnly, setShowPIIOnly] = useState(false)
  const [showStewardedOnly, setShowStewardedOnly] = useState(false)

  // Data
  const [tables, setTables] = useState<catalog.TableCatalogEntry[]>([])
  const [selectedTable, setSelectedTable] = useState<catalog.TableCatalogEntry | null>(null)
  const [selectedColumn, setSelectedColumn] = useState<catalog.ColumnCatalogEntry | null>(null)
  const [columns, setColumns] = useState<catalog.ColumnCatalogEntry[]>([])
  const [tags, setTags] = useState<catalog.CatalogTag[]>([])
  const [stats, setStats] = useState<main.CatalogStats | null>(null)
  const [expandedTables, setExpandedTables] = useState<ExpandedState>({})

  // Dialogs
  const [editTableDialogOpen, setEditTableDialogOpen] = useState(false)
  const [editColumnDialogOpen, setEditColumnDialogOpen] = useState(false)
  const [createTagDialogOpen, setCreateTagDialogOpen] = useState(false)
  const [deleteTagDialogOpen, setDeleteTagDialogOpen] = useState(false)
  const [tagToDelete, setTagToDelete] = useState<catalog.CatalogTag | null>(null)

  // Form state
  const [editingTableDescription, setEditingTableDescription] = useState('')
  const [editingTableTags, setEditingTableTags] = useState<string[]>([])
  const [editingTableSteward, setEditingTableSteward] = useState('')
  const [editingColumnDescription, setEditingColumnDescription] = useState('')
  const [editingColumnTags, setEditingColumnTags] = useState<string[]>([])
  const [editingColumnPIIType, setEditingColumnPIIType] = useState('__none__')
  const [editingColumnPIIConfidence, setEditingColumnPIIConfidence] = useState(0)
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState('#3b82f6')
  const [newTagDescription, setNewTagDescription] = useState('')

  // Get the selected connection object and its sessionId
  const selectedConnection = connections.find((c) => c.id === selectedConnectionId)
  const activeSessionId = selectedConnection?.sessionId

  // Handle connection selection - auto-connect if needed
  const handleConnectionSelect = async (connectionId: string) => {
    if (connectionId === '__none__') {
      setSelectedConnectionId('')
      return
    }

    setSelectedConnectionId(connectionId)
    const conn = connections.find((c) => c.id === connectionId)

    if (conn && !conn.isConnected) {
      setConnecting(true)
      try {
        await connectToDatabase(connectionId)
        toast({ title: 'Connected', description: `Connected to ${conn.name}` })
      } catch (error) {
        toast({
          title: 'Connection failed',
          description: error instanceof Error ? error.message : 'Failed to connect',
          variant: 'destructive',
        })
      } finally {
        setConnecting(false)
      }
    }
  }

  // Load initial data when sessionId becomes available
  useEffect(() => {
    loadData()
  }, [activeSessionId])

  const loadData = async () => {
    setLoading(true)
    try {
      const [tagsData, tablesData] = await Promise.all([
        ListCatalogTags(null),
        activeSessionId ? ListTableCatalogEntries(activeSessionId) : Promise.resolve([]),
      ])

      setTags(tagsData || [])
      setTables(tablesData || [])

      if (activeSessionId) {
        const statsData = await GetCatalogStats(activeSessionId)
        setStats(statsData)
      } else {
        setStats(null)
      }
    } catch (error) {
      console.error('Failed to load catalog data:', error)
      toast({
        title: 'Error loading catalog',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleSync = async () => {
    if (!activeSessionId) {
      toast({
        title: 'No connection selected',
        description: 'Please select and connect to a database first',
        variant: 'destructive',
      })
      return
    }

    setSyncing(true)
    try {
      const result = await SyncCatalogFromConnection(activeSessionId)
      toast({
        title: 'Sync complete',
        description: `Added ${result.tables_added} tables, updated ${result.tables_updated} tables, added ${result.columns_added} columns`,
      })
      await loadData()
    } catch (error) {
      toast({
        title: 'Sync failed',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setSyncing(false)
    }
  }

  const handleSelectTable = async (table: catalog.TableCatalogEntry) => {
    setSelectedTable(table)
    setSelectedColumn(null)
    setLoading(true)
    try {
      const columnsData = await ListColumnCatalogEntries(table.id)
      setColumns(columnsData || [])
    } catch (error) {
      console.error('Failed to load columns:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleEditTable = () => {
    if (!selectedTable) return
    setEditingTableDescription(selectedTable.description || '')
    setEditingTableTags(selectedTable.tags || [])
    setEditingTableSteward(selectedTable.steward_user_id || '')
    setEditTableDialogOpen(true)
  }

  const handleSaveTable = async () => {
    if (!selectedTable) return
    setLoading(true)
    try {
      // Use createFrom to preserve existing fields while updating specific ones
      const updated = catalog.TableCatalogEntry.createFrom({
        ...selectedTable,
        description: editingTableDescription,
        tags: editingTableTags,
      })
      await UpdateTableCatalogEntry(updated)

      if (editingTableSteward !== selectedTable.steward_user_id) {
        await AssignTableSteward(selectedTable.id, editingTableSteward)
      }

      toast({ title: 'Table updated' })
      setEditTableDialogOpen(false)
      await loadData()
    } catch (error) {
      toast({
        title: 'Failed to update table',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleEditColumn = (column: catalog.ColumnCatalogEntry) => {
    setSelectedColumn(column)
    setEditingColumnDescription(column.description || '')
    setEditingColumnTags(column.tags || [])
    setEditingColumnPIIType(column.pii_type || '')
    setEditingColumnPIIConfidence(column.pii_confidence || 0)
    setEditColumnDialogOpen(true)
  }

  const handleSaveColumn = async () => {
    if (!selectedColumn || !selectedTable) return
    setLoading(true)
    try {
      // Use createFrom to preserve existing fields while updating specific ones
      const updated = catalog.ColumnCatalogEntry.createFrom({
        ...selectedColumn,
        description: editingColumnDescription,
        tags: editingColumnTags,
      })
      await UpdateColumnCatalogEntry(updated)

      if (editingColumnPIIType && editingColumnPIIType !== '__none__') {
        await MarkColumnAsPII(selectedTable.id, selectedColumn.column_name, editingColumnPIIType, editingColumnPIIConfidence)
      }

      toast({ title: 'Column updated' })
      setEditColumnDialogOpen(false)
      await handleSelectTable(selectedTable)
    } catch (error) {
      toast({
        title: 'Failed to update column',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleCreateTag = async () => {
    if (!newTagName.trim()) return
    setLoading(true)
    try {
      // Create tag object - backend will set id and created_at
      const tag = catalog.CatalogTag.createFrom({
        name: newTagName,
        color: newTagColor,
        description: newTagDescription,
        is_system: false,
      })
      await CreateCatalogTag(tag)
      toast({ title: 'Tag created' })
      setCreateTagDialogOpen(false)
      setNewTagName('')
      setNewTagColor('#3b82f6')
      setNewTagDescription('')
      await loadData()
    } catch (error) {
      toast({
        title: 'Failed to create tag',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteTag = async () => {
    if (!tagToDelete) return
    setLoading(true)
    try {
      await DeleteCatalogTag(tagToDelete.id)
      toast({ title: 'Tag deleted' })
      setDeleteTagDialogOpen(false)
      setTagToDelete(null)
      await loadData()
    } catch (error) {
      toast({
        title: 'Failed to delete tag',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const toggleTableExpansion = async (tableId: string, table: catalog.TableCatalogEntry) => {
    const isCurrentlyExpanded = expandedTables[tableId]
    setExpandedTables((prev) => ({
      ...prev,
      [tableId]: !prev[tableId],
    }))
    // If expanding (not collapsing), load the columns
    if (!isCurrentlyExpanded) {
      await handleSelectTable(table)
    }
  }

  // Filter tables
  const filteredTables = useMemo(() => {
    let filtered = tables

    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (t) =>
          t.table_name.toLowerCase().includes(query) ||
          t.schema_name.toLowerCase().includes(query) ||
          (t.description && t.description.toLowerCase().includes(query))
      )
    }

    if (schemaFilter && schemaFilter !== '__all__') {
      filtered = filtered.filter((t) => t.schema_name === schemaFilter)
    }

    if (tagFilter.length > 0) {
      filtered = filtered.filter((t) => t.tags && tagFilter.some((tag) => t.tags?.includes(tag)))
    }

    if (showStewardedOnly) {
      filtered = filtered.filter((t) => t.steward_user_id)
    }

    return filtered
  }, [tables, searchQuery, schemaFilter, tagFilter, showStewardedOnly])

  const schemas = useMemo(() => {
    const schemaSet = new Set(tables.map((t) => t.schema_name))
    return Array.from(schemaSet).sort()
  }, [tables])

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="flex-1 space-y-6 overflow-y-auto p-6">
        {/* Header */}
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Data Catalog</h1>
            <p className="text-sm text-muted-foreground">
              Document and manage your database tables, columns, and metadata
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={loadData} disabled={loading || syncing}>
              <RefreshCw className={cn('mr-2 h-4 w-4', (loading || syncing) && 'animate-spin')} />
              Refresh
            </Button>
            <Button onClick={handleSync} disabled={!activeSessionId || syncing || loading || connecting}>
              <Database className="mr-2 h-4 w-4" />
              {connecting ? 'Connecting...' : 'Sync Connection'}
            </Button>
          </div>
        </div>

        {/* Stats */}
        {stats && (
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Total Tables</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.total_tables}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Total Columns</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.total_columns}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Tagged Tables</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.tagged_tables}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">PII Columns</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-orange-600">{stats.pii_columns}</div>
              </CardContent>
            </Card>
          </div>
        )}

        <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
          {/* Filters Sidebar */}
          <aside className="space-y-4">
            <Card>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-medium">Filters</CardTitle>
                  <Filter className="h-4 w-4 text-muted-foreground" />
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="search">Search</Label>
                  <div className="relative">
                    <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                      id="search"
                      placeholder="Search tables..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="pl-8"
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="connection">Connection</Label>
                  <Select
                    value={selectedConnectionId || '__none__'}
                    onValueChange={handleConnectionSelect}
                    disabled={connecting}
                  >
                    <SelectTrigger id="connection">
                      <SelectValue placeholder="Select a connection" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__none__">Select a connection...</SelectItem>
                      {connections.map((conn) => (
                        <SelectItem key={conn.id} value={conn.id}>
                          <span className="flex items-center gap-2">
                            <span
                              className={cn(
                                'h-2 w-2 rounded-full',
                                conn.isConnected ? 'bg-green-500' : 'bg-gray-300'
                              )}
                            />
                            {conn.name}
                            {!conn.isConnected && (
                              <span className="text-xs text-muted-foreground">(click to connect)</span>
                            )}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {connecting && (
                    <p className="text-xs text-muted-foreground">Connecting...</p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="schema">Schema</Label>
                  <Select value={schemaFilter} onValueChange={setSchemaFilter}>
                    <SelectTrigger id="schema">
                      <SelectValue placeholder="All schemas" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__all__">All schemas</SelectItem>
                      {schemas.map((schema) => (
                        <SelectItem key={schema} value={schema}>
                          {schema}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label>Tags</Label>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setCreateTagDialogOpen(true)}
                      className="h-6 px-2 text-xs"
                    >
                      <Plus className="h-3 w-3" />
                    </Button>
                  </div>
                  <ScrollArea className="h-32 rounded-md border p-2">
                    <div className="space-y-2">
                      {tags.map((tag) => (
                        <div key={tag.id} className="flex items-center space-x-2">
                          <Checkbox
                            id={`tag-${tag.id}`}
                            checked={tagFilter.includes(tag.name)}
                            onCheckedChange={(checked) => {
                              if (checked) {
                                setTagFilter([...tagFilter, tag.name])
                              } else {
                                setTagFilter(tagFilter.filter((t) => t !== tag.name))
                              }
                            }}
                          />
                          <label
                            htmlFor={`tag-${tag.id}`}
                            className="flex flex-1 items-center gap-2 text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                          >
                            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: tag.color }} />
                            {tag.name}
                          </label>
                        </div>
                      ))}
                    </div>
                  </ScrollArea>
                </div>

                <Separator />

                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="pii-only" className="text-sm">
                      PII Only
                    </Label>
                    <Switch id="pii-only" checked={showPIIOnly} onCheckedChange={setShowPIIOnly} />
                  </div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="stewarded-only" className="text-sm">
                      Stewarded Only
                    </Label>
                    <Switch id="stewarded-only" checked={showStewardedOnly} onCheckedChange={setShowStewardedOnly} />
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Tag Management */}
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Manage Tags</CardTitle>
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-40">
                  <div className="space-y-2">
                    {tags.map((tag) => (
                      <div key={tag.id} className="flex items-center justify-between rounded-md border p-2">
                        <div className="flex items-center gap-2">
                          <span className="h-3 w-3 rounded-full" style={{ backgroundColor: tag.color }} />
                          <span className="text-sm">{tag.name}</span>
                        </div>
                        {!tag.is_system && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setTagToDelete(tag)
                              setDeleteTagDialogOpen(true)
                            }}
                            className="h-6 w-6 p-0"
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        )}
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </CardContent>
            </Card>
          </aside>

          {/* Main Content */}
          <div className="space-y-4">
            {!selectedConnectionId ? (
              <EmptyState
                icon={Database}
                title="Select a connection"
                description="Choose a connection to view and manage its catalog"
              />
            ) : connecting ? (
              <EmptyState
                icon={Database}
                title="Connecting..."
                description={`Connecting to ${selectedConnection?.name || 'database'}...`}
              />
            ) : !activeSessionId ? (
              <EmptyState
                icon={Database}
                title="Connection not active"
                description="The selected connection is not active. Try selecting it again."
              />
            ) : loading && tables.length === 0 ? (
              <div className="space-y-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : filteredTables.length === 0 ? (
              <EmptyState
                icon={BookOpen}
                title="No tables found"
                description="Try adjusting your filters or sync the connection"
                action={
                  <Button onClick={handleSync} disabled={syncing}>
                    <Database className="mr-2 h-4 w-4" />
                    Sync Connection
                  </Button>
                }
              />
            ) : (
              <Tabs value={viewMode} onValueChange={(v) => setViewMode(v as ViewMode)}>
                <div className="flex items-center justify-between">
                  <TabsList>
                    <TabsTrigger value="tree">Tree View</TabsTrigger>
                    <TabsTrigger value="table">Table View</TabsTrigger>
                  </TabsList>
                  <div className="text-sm text-muted-foreground">
                    {filteredTables.length} table{filteredTables.length !== 1 ? 's' : ''}
                  </div>
                </div>

                <TabsContent value="tree" className="space-y-2">
                  {filteredTables.map((table) => (
                    <Card key={table.id} className={cn(selectedTable?.id === table.id && 'border-primary')}>
                      <CardHeader className="pb-3">
                        <div className="flex items-start justify-between">
                          <div className="flex items-start gap-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => toggleTableExpansion(table.id, table)}
                              className="h-6 w-6 p-0"
                            >
                              <ChevronRight
                                className={cn(
                                  'h-4 w-4 transition-transform duration-200',
                                  expandedTables[table.id] && 'rotate-90'
                                )}
                              />
                            </Button>
                            <div>
                              <div className="flex items-center gap-2">
                                <h3 className="font-semibold">
                                  {table.schema_name}.{table.table_name}
                                </h3>
                                {table.tags && table.tags.length > 0 && (
                                  <div className="flex gap-1">
                                    {table.tags.slice(0, 3).map((tagName) => {
                                      const tag = tags.find((t) => t.name === tagName)
                                      return (
                                        <Badge key={tagName} variant="outline" style={{ borderColor: tag?.color }}>
                                          {tagName}
                                        </Badge>
                                      )
                                    })}
                                    {table.tags.length > 3 && (
                                      <Badge variant="outline">+{table.tags.length - 3}</Badge>
                                    )}
                                  </div>
                                )}
                                {table.steward_user_id && (
                                  <Badge variant="secondary" className="gap-1">
                                    <User className="h-3 w-3" />
                                    Stewarded
                                  </Badge>
                                )}
                              </div>
                              {table.description && (
                                <p className="mt-1 text-sm text-muted-foreground">{table.description}</p>
                              )}
                            </div>
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              handleSelectTable(table)
                              handleEditTable()
                            }}
                            className="h-8"
                          >
                            <Edit2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </CardHeader>

                      <div
                        className={cn(
                          'grid transition-all duration-200 ease-in-out',
                          expandedTables[table.id] ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'
                        )}
                      >
                        <div className="overflow-hidden">
                          <CardContent className="pt-0 pb-4">
                            <div className="space-y-1 rounded-md border p-2">
                              {loading && selectedTable?.id === table.id ? (
                                <div className="space-y-1">
                                  <Skeleton className="h-8 w-full animate-pulse" />
                                  <Skeleton className="h-8 w-full animate-pulse" />
                                </div>
                              ) : selectedTable?.id === table.id && columns.length > 0 ? (
                                columns.map((column, index) => (
                                  <div
                                    key={column.id}
                                    className="flex items-center justify-between rounded-md p-2 hover:bg-muted/50 animate-in fade-in slide-in-from-top-1 duration-200"
                                    style={{ animationDelay: `${index * 30}ms` }}
                                  >
                                    <div className="flex items-center gap-2">
                                      <span className="text-sm font-medium">{column.column_name}</span>
                                      {column.pii_type && (
                                        <Badge variant="destructive" className="gap-1">
                                          <Shield className="h-3 w-3" />
                                          {column.pii_type}
                                        </Badge>
                                      )}
                                      {column.tags && column.tags.length > 0 && (
                                        <div className="flex gap-1">
                                          {column.tags.map((tagName) => {
                                            const tag = tags.find((t) => t.name === tagName)
                                            return (
                                              <Badge key={tagName} variant="outline" style={{ borderColor: tag?.color }}>
                                                {tagName}
                                              </Badge>
                                            )
                                          })}
                                        </div>
                                      )}
                                    </div>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      onClick={() => handleEditColumn(column)}
                                      className="h-6"
                                    >
                                      <Edit2 className="h-3 w-3" />
                                    </Button>
                                  </div>
                                ))
                              ) : (
                                <button
                                  onClick={() => handleSelectTable(table)}
                                  className="text-sm text-muted-foreground hover:text-foreground cursor-pointer transition-colors"
                                >
                                  Click to load columns
                                </button>
                              )}
                            </div>
                          </CardContent>
                        </div>
                      </div>
                    </Card>
                  ))}
                </TabsContent>

                <TabsContent value="table" className="space-y-2">
                  <div className="rounded-md border">
                    <div className="grid grid-cols-[2fr_3fr_1fr_auto] gap-2 border-b bg-muted/50 p-2 text-sm font-medium">
                      <div>Table</div>
                      <div>Description</div>
                      <div>Tags</div>
                      <div>Actions</div>
                    </div>
                    <div className="divide-y">
                      {filteredTables.map((table) => (
                        <div key={table.id} className="grid grid-cols-[2fr_3fr_1fr_auto] gap-2 p-2 text-sm">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">
                              {table.schema_name}.{table.table_name}
                            </span>
                            {table.steward_user_id && (
                              <User className="h-3 w-3 text-muted-foreground" />
                            )}
                          </div>
                          <div className="text-muted-foreground">{table.description || '-'}</div>
                          <div className="flex flex-wrap gap-1">
                            {table.tags?.slice(0, 2).map((tagName) => {
                              const tag = tags.find((t) => t.name === tagName)
                              return (
                                <Badge key={tagName} variant="outline" style={{ borderColor: tag?.color }}>
                                  {tagName}
                                </Badge>
                              )
                            })}
                            {table.tags && table.tags.length > 2 && (
                              <Badge variant="outline">+{table.tags.length - 2}</Badge>
                            )}
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              handleSelectTable(table)
                              handleEditTable()
                            }}
                            className="h-6"
                          >
                            <Edit2 className="h-3 w-3" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
            )}
          </div>
        </div>
      </div>

      {/* Edit Table Dialog */}
      <Dialog open={editTableDialogOpen} onOpenChange={setEditTableDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Edit Table Metadata</DialogTitle>
            <DialogDescription>
              {selectedTable && `${selectedTable.schema_name}.${selectedTable.table_name}`}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="table-description">Description</Label>
              <Textarea
                id="table-description"
                value={editingTableDescription}
                onChange={(e) => setEditingTableDescription(e.target.value)}
                rows={3}
                placeholder="Describe this table..."
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="table-steward">Steward</Label>
              <Input
                id="table-steward"
                value={editingTableSteward}
                onChange={(e) => setEditingTableSteward(e.target.value)}
                placeholder="User ID"
              />
            </div>
            <div className="space-y-2">
              <Label>Tags</Label>
              <div className="flex flex-wrap gap-2">
                {tags.map((tag) => (
                  <Badge
                    key={tag.id}
                    variant={editingTableTags.includes(tag.name) ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => {
                      if (editingTableTags.includes(tag.name)) {
                        setEditingTableTags(editingTableTags.filter((t) => t !== tag.name))
                      } else {
                        setEditingTableTags([...editingTableTags, tag.name])
                      }
                    }}
                    style={editingTableTags.includes(tag.name) ? { backgroundColor: tag.color } : undefined}
                  >
                    {tag.name}
                  </Badge>
                ))}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTableDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleSaveTable} disabled={loading}>
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Column Dialog */}
      <Dialog open={editColumnDialogOpen} onOpenChange={setEditColumnDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Edit Column Metadata</DialogTitle>
            <DialogDescription>{selectedColumn?.column_name}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="column-description">Description</Label>
              <Textarea
                id="column-description"
                value={editingColumnDescription}
                onChange={(e) => setEditingColumnDescription(e.target.value)}
                rows={3}
                placeholder="Describe this column..."
              />
            </div>
            <div className="space-y-2">
              <Label>Tags</Label>
              <div className="flex flex-wrap gap-2">
                {tags.map((tag) => (
                  <Badge
                    key={tag.id}
                    variant={editingColumnTags.includes(tag.name) ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => {
                      if (editingColumnTags.includes(tag.name)) {
                        setEditingColumnTags(editingColumnTags.filter((t) => t !== tag.name))
                      } else {
                        setEditingColumnTags([...editingColumnTags, tag.name])
                      }
                    }}
                    style={editingColumnTags.includes(tag.name) ? { backgroundColor: tag.color } : undefined}
                  >
                    {tag.name}
                  </Badge>
                ))}
              </div>
            </div>
            <Separator />
            <div className="space-y-4">
              <h4 className="text-sm font-semibold">PII Classification</h4>
              <div className="space-y-2">
                <Label htmlFor="pii-type">PII Type</Label>
                <Select value={editingColumnPIIType} onValueChange={setEditingColumnPIIType}>
                  <SelectTrigger id="pii-type">
                    <SelectValue placeholder="Select PII type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__none__">None</SelectItem>
                    {PII_TYPES.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        {type.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {editingColumnPIIType && editingColumnPIIType !== '__none__' && (
                <div className="space-y-2">
                  <Label htmlFor="pii-confidence">
                    Confidence: {editingColumnPIIConfidence}%
                  </Label>
                  <Slider
                    min={0}
                    max={100}
                    step={10}
                    value={[editingColumnPIIConfidence]}
                    onValueChange={(value) => setEditingColumnPIIConfidence(value[0])}
                  />
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditColumnDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleSaveColumn} disabled={loading}>
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create Tag Dialog */}
      <Dialog open={createTagDialogOpen} onOpenChange={setCreateTagDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Tag</DialogTitle>
            <DialogDescription>Add a new tag for categorizing tables and columns</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tag-name">Name</Label>
              <Input
                id="tag-name"
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                placeholder="Tag name"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tag-color">Color</Label>
              <Input
                id="tag-color"
                type="color"
                value={newTagColor}
                onChange={(e) => setNewTagColor(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tag-description">Description</Label>
              <Textarea
                id="tag-description"
                value={newTagDescription}
                onChange={(e) => setNewTagDescription(e.target.value)}
                rows={2}
                placeholder="Optional description"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateTagDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateTag} disabled={loading || !newTagName.trim()}>
              Create Tag
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Tag Dialog */}
      <ConfirmDialog
        open={deleteTagDialogOpen}
        title="Delete Tag?"
        description={`Are you sure you want to delete the tag "${tagToDelete?.name}"? This will remove it from all tables and columns.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDeleteTag}
        onCancel={() => {
          setDeleteTagDialogOpen(false)
          setTagToDelete(null)
        }}
      />
    </div>
  )
}
