import { AlertTriangle, RefreshCw, Sparkles } from 'lucide-react'
import React, { Component, ErrorInfo, ReactNode } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface Props {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error, errorInfo: ErrorInfo) => void
  featureName?: string
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
}

/**
 * AI Error Boundary - Specialized error boundary for AI-powered features
 *
 * Provides graceful degradation for AI components:
 * - Shows user-friendly fallback UI
 * - Allows retry without full page reload
 * - Logs errors for debugging
 * - Preserves surrounding UI functionality
 */
export class AIErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null
    }
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error
    }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('AI Component Error:', error, errorInfo)

    this.setState({
      error,
      errorInfo
    })

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(error, errorInfo)
    }

    // Log to service in production
    if (process.env.NODE_ENV === 'production') {
      this.logErrorToService(error, errorInfo)
    }
  }

  private logErrorToService = (error: Error, errorInfo: ErrorInfo) => {
    try {
      const errorData = {
        feature: this.props.featureName || 'AI Component',
        message: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
        timestamp: new Date().toISOString(),
        userAgent: navigator.userAgent,
        url: window.location.href
      }

      console.log('AI Error logged:', errorData)
    } catch (loggingError) {
      console.error('Failed to log AI error:', loggingError)
    }
  }

  private handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null
    })
  }

  render() {
    if (this.state.hasError) {
      // Custom fallback UI
      if (this.props.fallback) {
        return this.props.fallback
      }

      const featureName = this.props.featureName || 'AI Assistant'

      // Default AI error UI - more compact and inline
      return (
        <Card className="border-destructive/50 bg-destructive/5">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10">
                <AlertTriangle className="h-5 w-5 text-destructive" />
              </div>
              <div>
                <CardTitle className="text-lg">{featureName} Unavailable</CardTitle>
                <CardDescription>
                  The AI feature encountered an error and needs to be restarted.
                </CardDescription>
              </div>
            </div>
          </CardHeader>

          <CardContent className="space-y-4">
            {/* Error Details */}
            <Alert variant="destructive">
              <AlertDescription>
                <div className="space-y-1">
                  <div className="text-sm">
                    <strong>Error:</strong> {this.state.error?.message || 'Unknown error'}
                  </div>
                </div>
              </AlertDescription>
            </Alert>

            {/* Stack Trace (Development Only) */}
            {process.env.NODE_ENV === 'development' && this.state.error?.stack && (
              <details className="space-y-2">
                <summary className="cursor-pointer text-sm font-medium text-muted-foreground hover:text-foreground">
                  Technical Details (Development)
                </summary>
                <pre className="mt-2 max-h-32 overflow-auto rounded bg-muted p-3 text-xs">
                  {this.state.error.stack}
                </pre>
              </details>
            )}

            {/* Action Buttons */}
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button onClick={this.handleRetry} className="flex items-center gap-2">
                <RefreshCw className="h-4 w-4" />
                Retry {featureName}
              </Button>
            </div>

            {/* Help Text */}
            <div className="text-sm text-muted-foreground">
              <p className="flex items-center gap-2">
                <Sparkles className="h-4 w-4" />
                Don't worry - the rest of your workspace is still working normally.
              </p>
            </div>
          </CardContent>
        </Card>
      )
    }

    return this.props.children
  }
}

export default AIErrorBoundary
