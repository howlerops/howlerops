import { CheckCircle2 } from 'lucide-react'
import { memo } from 'react'

import { ConfidenceIndicator } from '@/components/ConfidenceIndicator'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { SQLSuggestion } from '@/store/ai-store'

interface AISuggestionCardProps {
  suggestion: SQLSuggestion
  onApply: (query: string) => void
  isApplied?: boolean
}

export const AISuggestionCard = memo(function AISuggestionCard({ suggestion, onApply, isApplied }: AISuggestionCardProps) {
  return (
    <div className="p-4 border rounded-lg bg-card hover:shadow-sm transition-shadow flex flex-col gap-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">
              {suggestion.provider} • {suggestion.model}
            </span>
            <ConfidenceIndicator confidence={suggestion.confidence} size="sm" />
          </div>
          {isApplied && (
            <Badge variant="default" className="text-xs gap-1">
              <CheckCircle2 className="h-3 w-3" />
              Applied
            </Badge>
          )}
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            {suggestion.timestamp.toLocaleTimeString()}
          </span>
        </div>
        <Button
          size="sm"
          variant={isApplied ? 'outline' : 'default'}
          onClick={() => onApply(suggestion.query)}
          disabled={isApplied}
        >
          {isApplied ? 'Applied' : 'Apply Changes'}
        </Button>
      </div>

      {suggestion.explanation && (
        <div className="text-sm mb-3 text-foreground">
          {suggestion.explanation}
        </div>
      )}

      <pre className="block p-3 bg-muted rounded text-xs font-mono text-foreground whitespace-pre-wrap overflow-x-auto max-h-48 overflow-y-auto">
        {suggestion.query}
      </pre>
    </div>
  )
})
