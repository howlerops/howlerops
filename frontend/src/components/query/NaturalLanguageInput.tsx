import {
  Check,
  ChevronRight,
  Copy,
  HelpCircle,
  Loader2,
  Send,
  Sparkles,
} from 'lucide-react'
import React, { startTransition, useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { nl2sql } from '@/lib/api/query-optimizer'

interface NaturalLanguageInputProps {
  onSQLGenerated?: (sql: string) => void
  connectionId?: string
}

interface ConversionResult {
  sql: string
  confidence: number
  template: string
  suggestions?: string[]
}

export function NaturalLanguageInput({
  onSQLGenerated,
  connectionId
}: NaturalLanguageInputProps) {
  const [nlQuery, setNlQuery] = useState('')
  const [result, setResult] = useState<ConversionResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [showExamples, setShowExamples] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const examples = useMemo(() => [
    { text: 'show all users', sql: 'SELECT * FROM users LIMIT 100' },
    { text: 'count orders from today', sql: 'SELECT COUNT(*) FROM orders WHERE DATE(created_at) = CURDATE()' },
    { text: 'find products where price > 100', sql: 'SELECT * FROM products WHERE price > 100' },
    { text: 'top 10 customers ordered by total_spent', sql: 'SELECT * FROM customers ORDER BY total_spent DESC LIMIT 10' },
    { text: 'average price from products', sql: 'SELECT AVG(price) AS average FROM products' },
    { text: 'users with email contains gmail', sql: "SELECT * FROM users WHERE email LIKE '%gmail%'" },
  ], [])

  const quickSuggestions = useMemo(() => [
    'show users',
    'count products',
    'find orders where total > 100',
    'top 5 customers',
  ], [])

  useEffect(() => {
    // Focus input on mount
    inputRef.current?.focus()
  }, [])

  const handleConvert = useCallback(async () => {
    if (!nlQuery.trim()) {
      setError('Please enter a query')
      return
    }

    setIsLoading(true)
    setError(null)
    setResult(null)

    try {
      const conversionResult = await nl2sql(nlQuery, connectionId)

      // Use startTransition for non-urgent UI updates
      startTransition(() => {
        if (conversionResult.sql) {
          setResult(conversionResult)
          if (onSQLGenerated) {
            onSQLGenerated(conversionResult.sql)
          }
        } else if (conversionResult.suggestions) {
          setError('Could not understand the query. Try rephrasing or check the suggestions.')
          setResult(conversionResult)
        } else {
          setError('Could not convert to SQL. Please try a different query.')
        }
      })
    } catch (err) {
      console.error('NL2SQL conversion failed:', err)
      setError('Failed to convert query. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }, [nlQuery, connectionId, onSQLGenerated])

  const handleKeyPress = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !isLoading) {
      handleConvert()
    }
  }, [isLoading, handleConvert])

  const handleCopy = useCallback(() => {
    if (result?.sql) {
      navigator.clipboard.writeText(result.sql)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [result?.sql])

  const handleExampleClick = useCallback((example: typeof examples[0]) => {
    setNlQuery(example.text)
    setShowExamples(false)
    // Auto-convert after setting example
    setTimeout(() => {
      startTransition(() => {
        setResult({
          sql: example.sql,
          confidence: 1.0,
          template: 'Example query',
        })
        if (onSQLGenerated) {
          onSQLGenerated(example.sql)
        }
      })
    }, 100)
  }, [examples, onSQLGenerated])

  const getConfidenceBadge = useCallback((confidence: number) => {
    if (confidence >= 0.8) {
      return <Badge className="bg-green-500">High confidence</Badge>
    } else if (confidence >= 0.5) {
      return <Badge className="bg-yellow-500">Medium confidence</Badge>
    } else {
      return <Badge className="bg-red-500">Low confidence</Badge>
    }
  }, [])

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="p-4">
          <div className="space-y-3">
            <div className="flex items-center gap-2 mb-2">
              <Sparkles className="w-5 h-5 text-purple-500" />
              <h3 className="font-medium">Natural Language to SQL</h3>
              <Popover open={showExamples} onOpenChange={setShowExamples}>
                <PopoverTrigger asChild>
                  <Button variant="ghost" size="sm">
                    <HelpCircle className="w-4 h-4" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-96">
                  <div className="space-y-2">
                    <h4 className="font-medium text-sm">Example Queries</h4>
                    <div className="space-y-1">
                      {examples.map((example, i) => (
                        <button
                          key={i}
                          onClick={() => handleExampleClick(example)}
                          className="w-full text-left p-2 hover:bg-gray-100 rounded-md transition-colors"
                        >
                          <div className="flex items-center gap-2">
                            <ChevronRight className="w-3 h-3 text-gray-400" />
                            <span className="text-sm">{example.text}</span>
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
            </div>

            <div className="flex gap-2">
              <Input
                ref={inputRef}
                placeholder="Type in plain English: show all users where status is active"
                value={nlQuery}
                onChange={(e) => setNlQuery(e.target.value)}
                onKeyPress={handleKeyPress}
                disabled={isLoading}
                className="flex-1"
              />
              <Button
                onClick={handleConvert}
                disabled={isLoading || !nlQuery.trim()}
              >
                {isLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Send className="w-4 h-4" />
                )}
                <span className="ml-2">Convert</span>
              </Button>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {result && result.sql && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">Generated SQL:</span>
                    {getConfidenceBadge(result.confidence)}
                    {result.template && (
                      <Badge variant="outline" className="text-xs">
                        {result.template}
                      </Badge>
                    )}
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCopy}
                  >
                    {copied ? (
                      <>
                        <Check className="w-4 h-4 text-green-500" />
                        <span className="ml-1">Copied!</span>
                      </>
                    ) : (
                      <>
                        <Copy className="w-4 h-4" />
                        <span className="ml-1">Copy</span>
                      </>
                    )}
                  </Button>
                </div>
                <div className="bg-gray-900 text-gray-100 p-3 rounded-md overflow-x-auto">
                  <pre className="text-sm">
                    <code>{result.sql}</code>
                  </pre>
                </div>
              </div>
            )}

            {result && result.suggestions && result.suggestions.length > 0 && (
              <div className="mt-3">
                <p className="text-sm text-gray-600 mb-2">Suggestions:</p>
                <div className="space-y-1">
                  {result.suggestions.map((suggestion, i) => (
                    <p key={i} className="text-xs text-gray-500">
                      {suggestion}
                    </p>
                  ))}
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Quick suggestions */}
      <div className="flex flex-wrap gap-2">
        <span className="text-xs text-gray-500">Try:</span>
        {quickSuggestions.map((suggestion) => (
          <Button
            key={suggestion}
            variant="outline"
            size="sm"
            className="text-xs"
            onClick={() => {
              setNlQuery(suggestion)
              inputRef.current?.focus()
            }}
          >
            {suggestion}
          </Button>
        ))}
      </div>
    </div>
  )
}