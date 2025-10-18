import { useCallback, useRef, useState } from "react"
import { QueryEditor, type QueryEditorHandle } from "@/components/query-editor"
import { ResultsPanel } from "@/components/results-panel"
import { useQueryMode } from "@/hooks/use-query-mode"

const MIN_PANEL_FRACTION = 0.02

export function Dashboard() {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const queryEditorRef = useRef<QueryEditorHandle>(null)
  const [editorFraction, setEditorFraction] = useState(0.55)
  const { mode } = useQueryMode('auto')

  // Remove automatic tab creation - let users create tabs manually

  const handleFixWithAI = useCallback((error: string, query: string) => {
    queryEditorRef.current?.openAIFix(error, query)
  }, [])

  const handleResizeStart = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault()
    const container = containerRef.current
    if (!container) return

    const { top, height } = container.getBoundingClientRect()

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const offset = moveEvent.clientY - top
      const nextFraction = offset / height
      const clamped = Math.min(1 - MIN_PANEL_FRACTION, Math.max(MIN_PANEL_FRACTION, nextFraction))
      setEditorFraction(clamped)
    }

    const handleMouseUp = () => {
      window.removeEventListener("mousemove", handleMouseMove)
      window.removeEventListener("mouseup", handleMouseUp)
    }

    window.addEventListener("mousemove", handleMouseMove)
    window.addEventListener("mouseup", handleMouseUp)
  }, [])

  return (
    <div
      ref={containerRef}
      className="flex flex-1 h-full min-h-0 w-full flex-col overflow-hidden"
    >
      <div
        className="flex min-h-[200px] flex-col border-b overflow-hidden"
        style={{ flexGrow: editorFraction, flexShrink: 1, flexBasis: 0 }}
      >
        <QueryEditor ref={queryEditorRef} mode={mode} />
      </div>

      <div
        className="relative h-2 cursor-row-resize bg-border hover:bg-primary/40"
        onMouseDown={handleResizeStart}
        role="separator"
        aria-orientation="horizontal"
        aria-label="Resize editor and results panels"
      >
        <div className="absolute inset-x-4 top-1/2 h-0.5 -translate-y-1/2 rounded bg-muted-foreground/50" />
      </div>

      <div
        className="flex min-h-[64px] flex-col overflow-hidden"
        style={{ flexGrow: 1 - editorFraction, flexShrink: 1, flexBasis: 0 }}
      >
        <ResultsPanel onFixWithAI={handleFixWithAI} />
      </div>
    </div>
  )
}
