import { AlertTriangle, Bug } from 'lucide-react'
import React, { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

/**
 * Development component to test error boundaries
 * Only renders in development mode
 */
export function ErrorBoundaryTester() {
  const [shouldThrow, setShouldThrow] = useState(false)

  if (process.env.NODE_ENV !== 'development') {
    return null
  }

  if (shouldThrow) {
    throw new Error('Test error for error boundary - this is intentional!')
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bug className="h-4 w-4" />
          Error Boundary Tester
        </CardTitle>
        <CardDescription>
          Development tool to test error boundaries
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <Button
            variant="destructive"
            onClick={() => setShouldThrow(true)}
            className="w-full"
          >
            <AlertTriangle className="h-4 w-4 mr-2" />
            Trigger Error
          </Button>
          <p className="text-xs text-muted-foreground">
            This will cause an error to test the error boundary
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
