import { HelpCircle, Layout, Loader2, Play, Save, Square, Wand2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"

import type { QueryMode } from "../types"

interface EditorToolbarProps {
  mode: QueryMode
  editorContent: string
  isExecuting: boolean
  isVisualMode: boolean
  isGenerating: boolean
  hasExecutionError: boolean
  aiEnabled: boolean
  connectionId: string | undefined
  onExecute: () => void
  onToggleVisualMode: () => void
  onFixWithAI: () => void
  onSaveQuery: () => void
  onOpenQueryLibrary: () => void
}

export function EditorToolbar({
  mode,
  editorContent,
  isExecuting,
  isVisualMode,
  isGenerating,
  hasExecutionError,
  aiEnabled,
  connectionId,
  onExecute,
  onToggleVisualMode,
  onFixWithAI,
  onSaveQuery,
  onOpenQueryLibrary,
}: EditorToolbarProps) {
  return (
    <div className="flex items-center justify-between p-2 border-b bg-muted/30">
      <div className="flex items-center space-x-2">
        <Button
          size="sm"
          onClick={onExecute}
          disabled={!editorContent.trim() || isExecuting}
          title={!connectionId ? "Select a database to execute this query" : undefined}
        >
          {isExecuting ? (
            <Square className="h-4 w-4 mr-2" />
          ) : (
            <Play className="h-4 w-4 mr-2" />
          )}
          {isExecuting ? 'Stop' : 'Run'}
        </Button>

        {/* Visual Mode Toggle */}
        <Button
          variant={isVisualMode ? "default" : "outline"}
          size="sm"
          onClick={onToggleVisualMode}
          className="ml-2"
        >
          <Layout className="h-4 w-4 mr-2" />
          {isVisualMode ? 'Visual' : 'SQL'}
        </Button>

        {/* AI Fix SQL Button */}
        {aiEnabled && hasExecutionError && (
          <Button
            variant="outline"
            size="sm"
            onClick={onFixWithAI}
            disabled={isGenerating}
            className="text-accent-foreground hover:text-accent-foreground/80 border-accent hover:border-accent"
          >
            {isGenerating ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <Wand2 className="h-4 w-4 mr-2" />
            )}
            Fix with AI
          </Button>
        )}

        {/* Save Query Button */}
        <Button
          variant="outline"
          size="sm"
          onClick={onSaveQuery}
          disabled={!editorContent.trim()}
          title="Save query to library (Ctrl/Cmd+Shift+S)"
        >
          <Save className="h-4 w-4 mr-2" />
          Save Query
        </Button>

        <Button
          variant="ghost"
          size="sm"
          onClick={onOpenQueryLibrary}
          title="Open Saved Queries (Ctrl/Cmd+Shift+L)"
        >
          Query Library
        </Button>
      </div>

      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        {mode === 'multi' ? (
          <>
            <span>Multi-database mode active</span>
            <Popover>
              <PopoverTrigger asChild>
                <Button variant="ghost" size="icon" className="h-5 w-5 p-0 text-muted-foreground hover:text-foreground">
                  <HelpCircle className="h-3.5 w-3.5" />
                  <span className="sr-only">Multi-database syntax help</span>
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-80" side="bottom" align="end">
                <div className="space-y-3">
                  <h4 className="font-medium text-sm">Multi-Database Query Syntax</h4>
                  <div className="space-y-2 text-sm">
                    <div>
                      <code className="px-1.5 py-0.5 rounded bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300 font-mono text-xs">
                        @connection.table
                      </code>
                      <p className="text-muted-foreground mt-1">
                        Reference a table from another database connection
                      </p>
                    </div>
                    <div>
                      <code className="px-1.5 py-0.5 rounded bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300 font-mono text-xs">
                        @connection.schema.table
                      </code>
                      <p className="text-muted-foreground mt-1">
                        Include schema name for databases that require it
                      </p>
                    </div>
                  </div>
                  <div className="pt-2 border-t">
                    <p className="text-xs text-muted-foreground">
                      <span className="inline-block w-2 h-2 rounded-sm bg-violet-500/20 border border-violet-500/50 mr-1.5" />
                      Purple highlighting indicates multi-database references
                    </p>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
          </>
        ) : (
          <span>Single database mode</span>
        )}
      </div>
    </div>
  )
}
