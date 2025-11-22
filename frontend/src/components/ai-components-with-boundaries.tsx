/**
 * AI Components with Error Boundaries
 *
 * This file demonstrates how to wrap AI components with error boundaries
 * for graceful degradation. Use these wrapped versions throughout the app.
 */

import React from 'react'
import { AIErrorBoundary } from './ai-error-boundary'
import { AIQueryTabView } from './ai-query-tab'
import { NaturalLanguageInput } from './query/NaturalLanguageInput'
import { AISchemaDisplay } from './ai-schema-display'
import { AISuggestionCard } from './ai-suggestion-card'

// Wrapped AI Query Tab with error boundary
export function SafeAIQueryTab(props: React.ComponentProps<typeof AIQueryTabView>) {
  return (
    <AIErrorBoundary featureName="AI Query Agent">
      <AIQueryTabView {...props} />
    </AIErrorBoundary>
  )
}

// Wrapped Natural Language Input with error boundary
export function SafeNaturalLanguageInput(props: React.ComponentProps<typeof NaturalLanguageInput>) {
  return (
    <AIErrorBoundary featureName="Natural Language to SQL">
      <NaturalLanguageInput {...props} />
    </AIErrorBoundary>
  )
}

// Wrapped Schema Display with error boundary
export function SafeAISchemaDisplay(props: React.ComponentProps<typeof AISchemaDisplay>) {
  return (
    <AIErrorBoundary featureName="Schema Explorer">
      <AISchemaDisplay {...props} />
    </AIErrorBoundary>
  )
}

// Wrapped AI Suggestion Card with error boundary
export function SafeAISuggestionCard(props: React.ComponentProps<typeof AISuggestionCard>) {
  return (
    <AIErrorBoundary featureName="AI Suggestions">
      <AISuggestionCard {...props} />
    </AIErrorBoundary>
  )
}

/**
 * Usage Example:
 *
 * Instead of:
 * import { AIQueryTabView } from './ai-query-tab'
 *
 * Use:
 * import { SafeAIQueryTab } from './ai-components-with-boundaries'
 *
 * This ensures that if the AI feature fails, it shows a graceful error
 * instead of breaking the entire application.
 */
