import { CheckCircle2,Play, RotateCcw } from "lucide-react"
import { useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { onboardingTracker } from "@/lib/analytics/onboarding-tracking"
import { cn } from "@/lib/utils"

interface InteractiveExampleProps {
  id: string
  title: string
  description: string
  initialQuery: string
  sampleData?: Array<Record<string, unknown>>
  editable?: boolean
  expectedResult?: string
  hint?: string
}

export function InteractiveExample({
  id,
  title,
  description,
  initialQuery,
  sampleData = [],
  editable = true,
  expectedResult,
  hint,
}: InteractiveExampleProps) {
  const [query, setQuery] = useState(initialQuery)
  const [hasRun, setHasRun] = useState(false)
  const [showHint, setShowHint] = useState(false)

  const handleRun = () => {
    setHasRun(true)
    onboardingTracker.trackInteractiveExampleRun(id)
  }

  const handleReset = () => {
    setQuery(initialQuery)
    setHasRun(false)
    setShowHint(false)
  }

  return (
    <Card className="overflow-hidden">
      {/* Header */}
      <div className="p-4 border-b bg-muted/30">
        <div className="flex items-start justify-between mb-2">
          <div className="flex-1">
            <h3 className="font-semibold">{title}</h3>
            <p className="text-sm text-muted-foreground mt-1">{description}</p>
          </div>
          {hasRun && (
            <Badge variant="outline" className="gap-1 border-green-500 text-green-600">
              <CheckCircle2 className="h-3 w-3" />
              Executed
            </Badge>
          )}
        </div>
      </div>

      {/* Query Editor */}
      <div className="border-b">
        <div className="bg-muted/50 px-4 py-2 text-xs font-medium border-b flex items-center justify-between">
          <span>SQL Query</span>
          <div className="flex items-center gap-2">
            {hint && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowHint(!showHint)}
                className="h-6 text-xs"
              >
                {showHint ? "Hide" : "Show"} Hint
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={handleReset}
              className="h-6 text-xs gap-1"
            >
              <RotateCcw className="h-3 w-3" />
              Reset
            </Button>
          </div>
        </div>

        {editable ? (
          <textarea
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full p-4 font-mono text-sm bg-slate-950 text-slate-50 min-h-[120px] resize-none focus:outline-none"
            spellCheck={false}
          />
        ) : (
          <pre className="p-4 font-mono text-sm bg-slate-950 text-slate-50 overflow-x-auto">
            {query}
          </pre>
        )}

        {showHint && hint && (
          <div className="px-4 py-3 bg-amber-50 dark:bg-amber-950 border-t border-amber-200 dark:border-amber-800">
            <p className="text-sm text-amber-900 dark:text-amber-100">
              <strong>Hint:</strong> {hint}
            </p>
          </div>
        )}
      </div>

      {/* Run Button */}
      <div className="p-4 border-b bg-muted/30">
        <Button onClick={handleRun} className="w-full gap-2">
          <Play className="h-4 w-4" />
          Run Query
        </Button>
      </div>

      {/* Results */}
      {hasRun && (
        <div>
          <div className="bg-muted/50 px-4 py-2 text-xs font-medium border-b">
            Results ({sampleData.length} {sampleData.length === 1 ? "row" : "rows"})
          </div>
          {sampleData.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/30">
                    {Object.keys(sampleData[0]).map((key) => (
                      <th
                        key={key}
                        className="px-4 py-2 text-left font-medium"
                      >
                        {key}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {sampleData.map((row, i) => (
                    <tr
                      key={i}
                      className={cn(
                        "border-b",
                        i % 2 === 0 ? "bg-muted/10" : ""
                      )}
                    >
                      {Object.values(row).map((value, j) => (
                        <td key={j} className="px-4 py-2">
                          {String(value)}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="p-8 text-center text-muted-foreground">
              No results
            </div>
          )}

          {expectedResult && (
            <div className="p-4 bg-green-50 dark:bg-green-950 border-t border-green-200 dark:border-green-800">
              <p className="text-sm text-green-900 dark:text-green-100">
                <strong>Expected:</strong> {expectedResult}
              </p>
            </div>
          )}
        </div>
      )}
    </Card>
  )
}
