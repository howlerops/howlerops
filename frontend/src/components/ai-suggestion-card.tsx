import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { CheckCircle2 } from "lucide-react"
import type { SQLSuggestion } from "@/store/ai-store"

interface AISuggestionCardProps {
  suggestion: SQLSuggestion
  onApply: (query: string) => void
  isApplied?: boolean
}

export function AISuggestionCard({ suggestion, onApply, isApplied }: AISuggestionCardProps) {
  return (
    <div className="p-4 border rounded-lg bg-card hover:shadow-sm transition-shadow">
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-medium text-muted-foreground">
            {suggestion.provider} • {suggestion.model}
          </span>
          <Badge variant="secondary" className="text-xs">
            {Math.round(suggestion.confidence * 100)}% confidence
          </Badge>
          {isApplied && (
            <Badge variant="default" className="text-xs gap-1">
              <CheckCircle2 className="h-3 w-3" />
              Applied
            </Badge>
          )}
        </div>
        <span className="text-xs text-muted-foreground whitespace-nowrap ml-2">
          {suggestion.timestamp.toLocaleTimeString()}
        </span>
      </div>

      {suggestion.explanation && (
        <div className="text-sm mb-3 text-foreground">
          {suggestion.explanation}
        </div>
      )}

      <pre className="block p-3 bg-muted rounded text-xs font-mono text-foreground whitespace-pre-wrap overflow-x-auto max-h-48 overflow-y-auto">
        {suggestion.query}
      </pre>

      <div className="mt-3 flex justify-end">
        <Button
          size="sm"
          variant={isApplied ? "outline" : "default"}
          onClick={() => onApply(suggestion.query)}
          disabled={isApplied}
        >
          {isApplied ? "Applied" : "Apply"}
        </Button>
      </div>
    </div>
  )
}

