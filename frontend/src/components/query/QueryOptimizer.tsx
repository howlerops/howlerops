import {
  AlertCircle,
  AlertTriangle,
  CheckCircle,
  ChevronDown,
  ChevronUp,
  Info,
  Lightbulb,
  TrendingUp,
  XCircle,
} from 'lucide-react'
import React, { useCallback,useEffect, useState } from 'react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent,CardHeader, CardTitle } from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { analyzeQuery } from '@/lib/api/query-optimizer'

interface QueryOptimizerProps {
  sql: string
  connectionId?: string
  isEnabled?: boolean
}

interface AnalysisResult {
  suggestions: Suggestion[]
  score: number
  warnings: Warning[]
  complexity: string
  estimated_cost: number
}

interface Suggestion {
  type: string
  severity: 'info' | 'warning' | 'critical'
  message: string
  original_sql?: string
  improved_sql?: string
  impact?: string
}

interface Warning {
  message: string
  severity: 'low' | 'medium' | 'high'
}

export function QueryOptimizer({ sql, connectionId, isEnabled = true }: QueryOptimizerProps) {
  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [expandedSuggestions, setExpandedSuggestions] = useState<Set<number>>(new Set())

  const performAnalysis = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const result = await analyzeQuery(sql, connectionId)
      setAnalysis(result)
    } catch (err) {
      console.error('Query analysis failed:', err)
      setError('Failed to analyze query')
    } finally {
      setIsLoading(false)
    }
  }, [sql, connectionId])

  useEffect(() => {
    if (!isEnabled || !sql || sql.trim().length < 10) {
      setAnalysis(null)
      return
    }

    // Debounce analysis (run 1 second after user stops typing)
    const timer = setTimeout(() => {
      performAnalysis()
    }, 1000)

    return () => clearTimeout(timer)
  }, [sql, connectionId, isEnabled, performAnalysis])

  const toggleSuggestion = (index: number) => {
    const newExpanded = new Set(expandedSuggestions)
    if (newExpanded.has(index)) {
      newExpanded.delete(index)
    } else {
      newExpanded.add(index)
    }
    setExpandedSuggestions(newExpanded)
  }

  const getScoreBadge = (score: number) => {
    if (score >= 80) {
      return (
        <Badge variant="default" className="bg-green-500">
          <CheckCircle className="w-3 h-3 mr-1" />
          Excellent ({score}/100)
        </Badge>
      )
    } else if (score >= 60) {
      return (
        <Badge variant="default" className="bg-yellow-500">
          <AlertCircle className="w-3 h-3 mr-1" />
          Good ({score}/100)
        </Badge>
      )
    } else {
      return (
        <Badge variant="destructive">
          <XCircle className="w-3 h-3 mr-1" />
          Needs Improvement ({score}/100)
        </Badge>
      )
    }
  }

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'info':
        return <Info className="w-4 h-4 text-blue-500" />
      case 'warning':
        return <AlertTriangle className="w-4 h-4 text-yellow-500" />
      case 'critical':
        return <XCircle className="w-4 h-4 text-red-500" />
      default:
        return <Lightbulb className="w-4 h-4 text-gray-500" />
    }
  }

  const getComplexityBadge = (complexity: string) => {
    const colors = {
      simple: 'bg-green-100 text-green-800',
      moderate: 'bg-yellow-100 text-yellow-800',
      complex: 'bg-red-100 text-red-800',
    }

    return (
      <Badge className={colors[complexity as keyof typeof colors] || 'bg-gray-100'}>
        {complexity}
      </Badge>
    )
  }

  if (!isEnabled || !sql || sql.trim().length < 10) {
    return null
  }

  if (isLoading) {
    return (
      <Card className="mt-4 animate-pulse">
        <CardHeader>
          <CardTitle className="text-sm">Analyzing query...</CardTitle>
        </CardHeader>
      </Card>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive" className="mt-4">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Analysis Error</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    )
  }

  if (!analysis) {
    return null
  }

  return (
    <Card className="mt-4">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <TrendingUp className="w-5 h-5" />
            Query Analysis
          </CardTitle>
          <div className="flex items-center gap-2">
            {getScoreBadge(analysis.score)}
            {getComplexityBadge(analysis.complexity)}
            {analysis.estimated_cost > 0 && (
              <Badge variant="outline">Cost: {analysis.estimated_cost}</Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Warnings */}
        {analysis.warnings.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-red-600">Warnings</h3>
            {analysis.warnings.map((warning, i) => (
              <Alert
                key={i}
                variant="destructive"
                className={`${
                  warning.severity === 'high'
                    ? 'border-red-500'
                    : warning.severity === 'medium'
                    ? 'border-yellow-500'
                    : 'border-orange-400'
                }`}
              >
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{warning.message}</AlertDescription>
              </Alert>
            ))}
          </div>
        )}

        {/* Suggestions */}
        {analysis.suggestions.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium">Optimization Suggestions</h3>
            {analysis.suggestions.map((suggestion, i) => (
              <Collapsible
                key={i}
                open={expandedSuggestions.has(i)}
                onOpenChange={() => toggleSuggestion(i)}
              >
                <div className="border rounded-lg p-3 hover:bg-gray-50">
                  <CollapsibleTrigger className="flex items-start justify-between w-full text-left">
                    <div className="flex items-start gap-2 flex-1">
                      {getSeverityIcon(suggestion.severity)}
                      <div className="flex-1">
                        <p className="text-sm font-medium">{suggestion.message}</p>
                        {suggestion.impact && (
                          <p className="text-xs text-gray-600 mt-1">{suggestion.impact}</p>
                        )}
                      </div>
                    </div>
                    <Button variant="ghost" size="sm" className="ml-2">
                      {expandedSuggestions.has(i) ? (
                        <ChevronUp className="h-4 w-4" />
                      ) : (
                        <ChevronDown className="h-4 w-4" />
                      )}
                    </Button>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="mt-3">
                    {suggestion.improved_sql && (
                      <div className="space-y-2">
                        <div>
                          <p className="text-xs font-medium text-gray-600 mb-1">
                            Suggested improvement:
                          </p>
                          <pre className="text-xs bg-gray-100 p-2 rounded overflow-x-auto">
                            <code>{suggestion.improved_sql}</code>
                          </pre>
                        </div>
                      </div>
                    )}
                    <div className="mt-2 flex gap-2">
                      <Badge variant="outline" className="text-xs">
                        {suggestion.type}
                      </Badge>
                      <Badge
                        variant="outline"
                        className={`text-xs ${
                          suggestion.severity === 'critical'
                            ? 'text-red-600'
                            : suggestion.severity === 'warning'
                            ? 'text-yellow-600'
                            : 'text-blue-600'
                        }`}
                      >
                        {suggestion.severity}
                      </Badge>
                    </div>
                  </CollapsibleContent>
                </div>
              </Collapsible>
            ))}
          </div>
        )}

        {/* Perfect query message */}
        {analysis.score === 100 && analysis.suggestions.length === 0 && (
          <Alert className="border-green-200 bg-green-50">
            <CheckCircle className="h-4 w-4 text-green-600" />
            <AlertTitle className="text-green-800">Excellent!</AlertTitle>
            <AlertDescription className="text-green-700">
              Your query is well-optimized. No improvements suggested.
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  )
}
