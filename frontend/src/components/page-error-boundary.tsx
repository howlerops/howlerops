import React from 'react'
import { ErrorBoundary } from '@/components/error-boundary'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface PageErrorBoundaryProps {
  children: React.ReactNode
  pageName?: string
}

export function PageErrorBoundary({ children, pageName = 'page' }: PageErrorBoundaryProps) {
  return (
    <ErrorBoundary
      fallback={
        <div className="flex flex-1 items-center justify-center p-8">
          <Card className="w-full max-w-md">
            <CardHeader className="text-center">
              <div className="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10">
                <AlertTriangle className="h-5 w-5 text-destructive" />
              </div>
              <CardTitle>Error in {pageName}</CardTitle>
              <CardDescription>
                Something went wrong on this page. You can try refreshing or go back to the dashboard.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-col gap-2">
                <Button 
                  onClick={() => window.location.reload()} 
                  className="flex items-center gap-2"
                >
                  <RefreshCw className="h-4 w-4" />
                  Refresh Page
                </Button>
                <Button 
                  variant="outline" 
                  onClick={() => window.location.href = '/dashboard'}
                >
                  Go to Dashboard
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      }
      onError={(error, errorInfo) => {
        console.error(`Error in ${pageName}:`, error, errorInfo)
      }}
    >
      {children}
    </ErrorBoundary>
  )
}

// Higher-order component for wrapping pages - legitimate React pattern
// eslint-disable-next-line react-refresh/only-export-components
export function withPageErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  pageName?: string
) {
  const WrappedComponent = (props: P) => (
    <PageErrorBoundary pageName={pageName}>
      <Component {...props} />
    </PageErrorBoundary>
  )
  
  WrappedComponent.displayName = `withPageErrorBoundary(${Component.displayName || Component.name})`
  
  return WrappedComponent
}
