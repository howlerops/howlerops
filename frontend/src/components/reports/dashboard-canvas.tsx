import React, { useMemo, useCallback, useState, useEffect, useRef } from 'react'
import { Responsive, WidthProvider, Layout as RGLLayout } from 'react-grid-layout'
import {
  BarChart3,
  FileQuestion,
  Gauge,
  Grid3x3,
  MessageSquare,
  Play,
  Plus,
  Settings,
  Table2,
  Trash2,
  Undo,
  Redo,
} from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { EmptyState } from '@/components/ui/empty-state'
import { cn } from '@/lib/utils'
import type {
  ReportComponent,
  ReportComponentType,
  ReportLayoutSlot,
  ReportRunResult,
  ReportRunComponentResult,
} from '@/types/reports'

// Import required CSS
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './dashboard-canvas.css'

const ResponsiveGridLayout = WidthProvider(Responsive)

interface DashboardCanvasProps {
  components: ReportComponent[]
  layout: ReportLayoutSlot[]
  results?: ReportRunResult
  onLayoutChange: (newLayout: ReportLayoutSlot[]) => void
  onComponentClick?: (componentId: string) => void
  onComponentEdit?: (componentId: string) => void
  onComponentRun?: (componentId: string) => void
  onComponentDelete?: (componentId: string) => void
  onAddComponent?: (type: ReportComponentType) => void
  editMode?: boolean
}

interface LayoutHistoryEntry {
  layout: ReportLayoutSlot[]
  timestamp: number
}

const MAX_HISTORY = 10
const AUTO_SAVE_DELAY = 1000

export function DashboardCanvas({
  components,
  layout,
  results,
  onLayoutChange,
  onComponentClick,
  onComponentEdit,
  onComponentRun,
  onComponentDelete,
  onAddComponent,
  editMode = true,
}: DashboardCanvasProps) {
  // Layout history for undo/redo
  const [layoutHistory, setLayoutHistory] = useState<LayoutHistoryEntry[]>([
    { layout, timestamp: Date.now() },
  ])
  const [historyIndex, setHistoryIndex] = useState(0)
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)

  // Auto-save timer
  const autoSaveTimerRef = useRef<NodeJS.Timeout | null>(null)

  // Create lookup for component results
  const resultsByComponentId = useMemo(() => {
    if (!results) return new Map<string, ReportRunComponentResult>()
    return new Map(results.results.map((result) => [result.componentId, result]))
  }, [results])

  // Convert ReportLayoutSlot to react-grid-layout Layout format
  const gridLayout = useMemo((): RGLLayout[] => {
    return layout.map((slot) => ({
      i: slot.componentId,
      x: slot.x,
      y: slot.y,
      w: slot.w,
      h: slot.h,
      minW: components.find((c) => c.id === slot.componentId)?.size?.minW,
      minH: components.find((c) => c.id === slot.componentId)?.size?.minH,
      maxW: components.find((c) => c.id === slot.componentId)?.size?.maxW,
      maxH: components.find((c) => c.id === slot.componentId)?.size?.maxH,
    }))
  }, [layout, components])

  // Handle layout changes from react-grid-layout
  const handleLayoutChange = useCallback(
    (newLayout: RGLLayout[]) => {
      const updatedLayout: ReportLayoutSlot[] = newLayout.map((item) => ({
        componentId: item.i,
        x: item.x,
        y: item.y,
        w: item.w,
        h: item.h,
      }))

      // Add to history (truncate future if we're in the middle of history)
      setLayoutHistory((prev) => {
        const newHistory = prev.slice(0, historyIndex + 1)
        newHistory.push({ layout: updatedLayout, timestamp: Date.now() })
        return newHistory.slice(-MAX_HISTORY) // Keep only last MAX_HISTORY entries
      })
      setHistoryIndex((prev) => Math.min(prev + 1, MAX_HISTORY - 1))
      setHasUnsavedChanges(true)

      // Clear existing auto-save timer
      if (autoSaveTimerRef.current) {
        clearTimeout(autoSaveTimerRef.current)
      }

      // Set new auto-save timer
      autoSaveTimerRef.current = setTimeout(() => {
        onLayoutChange(updatedLayout)
        setHasUnsavedChanges(false)
      }, AUTO_SAVE_DELAY)
    },
    [historyIndex, onLayoutChange]
  )

  // Cleanup auto-save timer on unmount
  useEffect(() => {
    return () => {
      if (autoSaveTimerRef.current) {
        clearTimeout(autoSaveTimerRef.current)
      }
    }
  }, [])

  // Undo/Redo handlers
  const handleUndo = useCallback(() => {
    if (historyIndex > 0) {
      const newIndex = historyIndex - 1
      setHistoryIndex(newIndex)
      const previousLayout = layoutHistory[newIndex].layout
      onLayoutChange(previousLayout)
      setHasUnsavedChanges(false)
    }
  }, [historyIndex, layoutHistory, onLayoutChange])

  const handleRedo = useCallback(() => {
    if (historyIndex < layoutHistory.length - 1) {
      const newIndex = historyIndex + 1
      setHistoryIndex(newIndex)
      const nextLayout = layoutHistory[newIndex].layout
      onLayoutChange(nextLayout)
      setHasUnsavedChanges(false)
    }
  }, [historyIndex, layoutHistory, onLayoutChange])

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'z' && !e.shiftKey) {
        e.preventDefault()
        handleUndo()
      } else if ((e.metaKey || e.ctrlKey) && (e.key === 'Z' || (e.shiftKey && e.key === 'z'))) {
        e.preventDefault()
        handleRedo()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleUndo, handleRedo])

  // No components - show empty state
  if (components.length === 0) {
    return (
      <div className="relative">
        <EmptyState
          icon={Grid3x3}
          title="No components on canvas"
          description="Add components to your dashboard to get started. They'll appear on an interactive grid that you can drag and resize."
          action={
            onAddComponent && (
              <ComponentPalette onAddComponent={onAddComponent} />
            )
          }
        />
      </div>
    )
  }

  return (
    <div className="relative space-y-4">
      {/* Toolbar */}
      {editMode && (
        <div className="flex items-center justify-between rounded-lg border bg-muted/50 p-3">
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="ghost"
              onClick={handleUndo}
              disabled={historyIndex <= 0}
              title="Undo (Cmd+Z)"
            >
              <Undo className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={handleRedo}
              disabled={historyIndex >= layoutHistory.length - 1}
              title="Redo (Cmd+Shift+Z)"
            >
              <Redo className="h-4 w-4" />
            </Button>
            {hasUnsavedChanges && (
              <Badge variant="secondary" className="ml-2">
                Saving...
              </Badge>
            )}
          </div>
          <div className="text-xs text-muted-foreground">
            {components.length} component{components.length !== 1 ? 's' : ''} • 12-column grid • Drag to move • Resize from corners
          </div>
        </div>
      )}

      {/* Grid Canvas */}
      <div className={cn('rounded-lg border bg-background p-4', editMode && 'bg-muted/10')}>
        <ResponsiveGridLayout
          className="layout"
          layouts={{ lg: gridLayout }}
          breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480 }}
          cols={{ lg: 12, md: 10, sm: 6, xs: 4 }}
          rowHeight={60}
          margin={[16, 16]}
          containerPadding={[0, 0]}
          isDraggable={editMode}
          isResizable={editMode}
          onLayoutChange={handleLayoutChange}
          draggableHandle=".drag-handle"
          useCSSTransforms={true}
          compactType="vertical"
        >
          {components.map((component) => (
            <div key={component.id}>
              <GridComponent
                component={component}
                result={resultsByComponentId.get(component.id)}
                editMode={editMode}
                onEdit={() => onComponentEdit?.(component.id)}
                onRun={() => onComponentRun?.(component.id)}
                onDelete={() => onComponentDelete?.(component.id)}
                onClick={() => onComponentClick?.(component.id)}
              />
            </div>
          ))}
        </ResponsiveGridLayout>
      </div>

      {/* Floating Add Button */}
      {editMode && onAddComponent && (
        <div className="fixed bottom-8 right-8 z-50">
          <ComponentPalette onAddComponent={onAddComponent} />
        </div>
      )}
    </div>
  )
}

interface GridComponentProps {
  component: ReportComponent
  result?: ReportRunComponentResult
  editMode: boolean
  onEdit: () => void
  onRun: () => void
  onDelete: () => void
  onClick: () => void
}

const GridComponent = React.memo(
  ({ component, result, editMode, onEdit, onRun, onDelete, onClick }: GridComponentProps) => {
    const [isHovered, setIsHovered] = useState(false)

    const getComponentIcon = (type: ReportComponentType) => {
      switch (type) {
        case 'chart':
        case 'combo':
          return BarChart3
        case 'metric':
          return Gauge
        case 'table':
          return Table2
        case 'llm':
          return MessageSquare
        default:
          return Grid3x3
      }
    }

    const Icon = getComponentIcon(component.type)

    return (
      <Card
        className={cn(
          'group h-full flex flex-col overflow-hidden transition-all',
          editMode ? 'cursor-move hover:shadow-lg' : 'cursor-pointer hover:shadow-md',
          isHovered && editMode && 'shadow-lg ring-2 ring-primary/20'
        )}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        onClick={!editMode ? onClick : undefined}
      >
        {/* Header with drag handle */}
        <CardHeader
          className={cn(
            'drag-handle pb-3',
            editMode && 'cursor-move',
            'flex flex-row items-center justify-between space-y-0'
          )}
        >
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <Icon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <CardTitle className="text-sm truncate">{component.title}</CardTitle>
            <Badge variant="outline" className="text-xs flex-shrink-0">
              {component.type}
            </Badge>
          </div>

          {/* Toolbar - visible on hover in edit mode */}
          {editMode && (
            <div
              className={cn(
                'flex gap-1 transition-opacity',
                isHovered ? 'opacity-100' : 'opacity-0'
              )}
              onClick={(e) => e.stopPropagation()}
            >
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={(e) => {
                  e.stopPropagation()
                  onEdit()
                }}
                title="Edit component"
              >
                <Settings className="h-3.5 w-3.5" />
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={(e) => {
                  e.stopPropagation()
                  onRun()
                }}
                title="Run component"
              >
                <Play className="h-3.5 w-3.5" />
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7 text-destructive hover:text-destructive"
                onClick={(e) => {
                  e.stopPropagation()
                  onDelete()
                }}
                title="Delete component"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          )}
        </CardHeader>

        {/* Component content */}
        <CardContent className="flex-1 overflow-auto">
          {result ? (
            result.error ? (
              <Alert variant="destructive">
                <AlertTitle>Error</AlertTitle>
                <AlertDescription className="text-xs">{result.error}</AlertDescription>
              </Alert>
            ) : (
              <ComponentPreview type={component.type} result={result} />
            )
          ) : (
            <EmptyState
              icon={FileQuestion}
              title="No data"
              description="Run this component to see results"
              compact
            />
          )}
        </CardContent>
      </Card>
    )
  }
)

GridComponent.displayName = 'GridComponent'

interface ComponentPreviewProps {
  type: ReportComponentType
  result: ReportRunComponentResult
}

function ComponentPreview({ type, result }: ComponentPreviewProps) {
  // LLM content
  if (result.content) {
    return (
      <div className="whitespace-pre-wrap rounded-md bg-muted p-3 text-xs">
        {result.content}
      </div>
    )
  }

  // Table/Chart data
  if (result.rows && result.columns) {
    // Metric component - show single value large
    if (type === 'metric' && result.rows.length > 0 && result.columns.length > 0) {
      const value = result.rows[0][0]
      return (
        <div className="flex h-full items-center justify-center">
          <div className="text-center">
            <div className="text-4xl font-bold">
              {typeof value === 'number' ? value.toLocaleString() : String(value)}
            </div>
            {result.columns[0] && (
              <div className="text-sm text-muted-foreground mt-2">{result.columns[0]}</div>
            )}
          </div>
        </div>
      )
    }

    // Table component - show data table
    return (
      <div className="overflow-auto rounded-md border">
        <table className="w-full min-w-[300px] text-xs">
          <thead>
            <tr className="border-b bg-muted/50 text-left">
              {result.columns.map((column) => (
                <th key={column} className="px-3 py-2 font-medium">
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {result.rows.slice(0, 10).map((row, rowIndex) => (
              <tr key={rowIndex} className="border-b last:border-none hover:bg-muted/30">
                {row.map((value, colIndex) => (
                  <td key={`${rowIndex}-${colIndex}`} className="px-3 py-2">
                    {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {result.rows.length > 10 && (
          <div className="border-t bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
            Showing first 10 of {result.rows.length} rows
          </div>
        )}
      </div>
    )
  }

  return <p className="text-xs text-muted-foreground">No preview data available</p>
}

interface ComponentPaletteProps {
  onAddComponent: (type: ReportComponentType) => void
}

function ComponentPalette({ onAddComponent }: ComponentPaletteProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size="lg" className="rounded-full shadow-lg">
          <Plus className="h-5 w-5 mr-2" /> Add Component
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuItem onClick={() => onAddComponent('chart')}>
          <BarChart3 className="mr-2 h-4 w-4" />
          <div>
            <p className="font-medium">Chart</p>
            <p className="text-xs text-muted-foreground">Line, bar, area, pie</p>
          </div>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onAddComponent('metric')}>
          <Gauge className="mr-2 h-4 w-4" />
          <div>
            <p className="font-medium">Metric</p>
            <p className="text-xs text-muted-foreground">Single KPI value</p>
          </div>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onAddComponent('table')}>
          <Table2 className="mr-2 h-4 w-4" />
          <div>
            <p className="font-medium">Table</p>
            <p className="text-xs text-muted-foreground">Tabular data display</p>
          </div>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => onAddComponent('combo')}>
          <BarChart3 className="mr-2 h-4 w-4" />
          <div>
            <p className="font-medium">Combo Chart</p>
            <p className="text-xs text-muted-foreground">Multiple series</p>
          </div>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onAddComponent('llm')}>
          <MessageSquare className="mr-2 h-4 w-4" />
          <div>
            <p className="font-medium">LLM Summary</p>
            <p className="text-xs text-muted-foreground">AI-generated insights</p>
          </div>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

// Layout Templates
export const LAYOUT_TEMPLATES = {
  'single-kpi': {
    name: 'Single KPI',
    description: '1 large metric centered',
    getLayout: (componentIds: string[]): ReportLayoutSlot[] => [
      { componentId: componentIds[0], x: 3, y: 0, w: 6, h: 4 },
    ],
  },
  'dual-kpi': {
    name: 'Dual KPI',
    description: '2 metrics side-by-side',
    getLayout: (componentIds: string[]): ReportLayoutSlot[] => [
      { componentId: componentIds[0], x: 0, y: 0, w: 6, h: 4 },
      { componentId: componentIds[1], x: 6, y: 0, w: 6, h: 4 },
    ],
  },
  'dashboard': {
    name: 'Dashboard',
    description: '4 KPIs top, 2 charts below',
    getLayout: (componentIds: string[]): ReportLayoutSlot[] => [
      { componentId: componentIds[0], x: 0, y: 0, w: 3, h: 3 },
      { componentId: componentIds[1], x: 3, y: 0, w: 3, h: 3 },
      { componentId: componentIds[2], x: 6, y: 0, w: 3, h: 3 },
      { componentId: componentIds[3], x: 9, y: 0, w: 3, h: 3 },
      { componentId: componentIds[4], x: 0, y: 3, w: 6, h: 5 },
      { componentId: componentIds[5], x: 6, y: 3, w: 6, h: 5 },
    ],
  },
  'report': {
    name: 'Report',
    description: 'Filters top, table full-width',
    getLayout: (componentIds: string[]): ReportLayoutSlot[] => [
      { componentId: componentIds[0], x: 0, y: 0, w: 12, h: 8 },
    ],
  },
  'analytics': {
    name: 'Analytics',
    description: '3 KPIs, 1 large chart, 2 small charts',
    getLayout: (componentIds: string[]): ReportLayoutSlot[] => [
      { componentId: componentIds[0], x: 0, y: 0, w: 4, h: 3 },
      { componentId: componentIds[1], x: 4, y: 0, w: 4, h: 3 },
      { componentId: componentIds[2], x: 8, y: 0, w: 4, h: 3 },
      { componentId: componentIds[3], x: 0, y: 3, w: 8, h: 6 },
      { componentId: componentIds[4], x: 8, y: 3, w: 4, h: 3 },
      { componentId: componentIds[5], x: 8, y: 6, w: 4, h: 3 },
    ],
  },
}

export type LayoutTemplate = keyof typeof LAYOUT_TEMPLATES
