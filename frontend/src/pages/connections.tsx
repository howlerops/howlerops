import { ConnectionManager } from "@/components/connection-manager"
import { PageErrorBoundary } from "@/components/page-error-boundary"

export function Connections() {
  return (
    <PageErrorBoundary pageName="Connections">
      <ConnectionManager />
    </PageErrorBoundary>
  )
}