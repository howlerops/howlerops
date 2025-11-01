import React, { ErrorInfo } from 'react'
import ErrorBoundary from './error-boundary'

export function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  errorBoundaryProps?: Omit<React.ComponentProps<typeof ErrorBoundary>, 'children'>
) {
  const WrappedComponent = (props: P) => (
    <ErrorBoundary {...errorBoundaryProps}>
      <Component {...props} />
    </ErrorBoundary>
  )

  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`

  return WrappedComponent
}

export function useErrorHandler() {
  return (error: Error, errorInfo?: ErrorInfo) => {
    console.error('Manual error report:', error, errorInfo)

    if (process.env.NODE_ENV === 'production') {
      console.log('Error reported to service:', { error, errorInfo })
    }
  }
}
