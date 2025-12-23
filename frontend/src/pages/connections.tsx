/**
 * Connections Page
 *
 * Main page for managing database connections. Shows tabs for personal
 * and team connections only when user has an organization.
 */

import { Database, Users } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

import { ConnectionManager } from '@/components/connection-manager'
import { PageErrorBoundary } from '@/components/page-error-boundary'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useOrganizationStore } from '@/store/organization-store'

import { TeamConnectionsTab } from './connections-team-tab'

type ConnectionTab = 'my' | 'team'

export function Connections() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { currentOrgId, organizations } = useOrganizationStore()

  // Get active tab from URL or default to 'my'
  const activeTab = useMemo(() => {
    const tab = searchParams.get('tab')
    if (tab === 'team' && currentOrgId) return 'team'
    return 'my'
  }, [searchParams, currentOrgId])

  // Update URL when tab changes
  const handleTabChange = (value: string) => {
    const newTab = value as ConnectionTab
    if (newTab === 'my') {
      searchParams.delete('tab')
    } else {
      searchParams.set('tab', newTab)
    }
    setSearchParams(searchParams, { replace: true })
  }

  // If user navigates to ?tab=team but has no org, reset to 'my'
  useEffect(() => {
    if (searchParams.get('tab') === 'team' && !currentOrgId) {
      searchParams.delete('tab')
      setSearchParams(searchParams, { replace: true })
    }
  }, [currentOrgId, searchParams, setSearchParams])

  // Get current org name for tab label
  const currentOrg = organizations.find((o) => o.id === currentOrgId)
  const hasOrganization = !!currentOrgId

  // No organization - show ConnectionManager directly without tabs
  if (!hasOrganization) {
    return (
      <PageErrorBoundary pageName="Connections">
        <div className="flex flex-col h-full overflow-auto">
          <ConnectionManager />
        </div>
      </PageErrorBoundary>
    )
  }

  // Has organization - show tabs for My Connections and Team
  return (
    <PageErrorBoundary pageName="Connections">
      <div className="flex flex-col h-full overflow-auto">
        <div className="p-6">
          {/* Header */}
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between mb-6">
            <div>
              <h1 className="text-2xl font-bold">Database Connections</h1>
              <p className="text-muted-foreground">Manage your database connections</p>
            </div>
          </div>

          {/* Tabs */}
          <Tabs value={activeTab} onValueChange={handleTabChange} className="w-full">
            <TabsList className="mb-6">
              <TabsTrigger value="my" className="gap-2">
                <Database className="h-4 w-4" />
                My Connections
              </TabsTrigger>
              <TabsTrigger value="team" className="gap-2">
                <Users className="h-4 w-4" />
                Team
                {currentOrg && (
                  <span className="text-xs text-muted-foreground ml-1">
                    ({currentOrg.name})
                  </span>
                )}
              </TabsTrigger>
            </TabsList>

            <TabsContent value="my" className="mt-0">
              <ConnectionManager hideHeader />
            </TabsContent>

            <TabsContent value="team" className="mt-0">
              <TeamConnectionsTab hideHeader />
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </PageErrorBoundary>
  )
}
