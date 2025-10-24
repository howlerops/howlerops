/**
 * Integration Examples
 *
 * Example usage of shared resources components in various contexts.
 * These examples demonstrate common integration patterns.
 *
 * @module components/sharing/INTEGRATION_EXAMPLES
 */

/* eslint-disable @typescript-eslint/no-unused-vars */
import { useState } from 'react'
import { VisibilityToggle } from './VisibilityToggle'
import { SharedResourceCard } from './SharedResourceCard'
import { ConflictResolutionDialog } from './ConflictResolutionDialog'
import { useConnectionsStore } from '@/store/connections-store'
import { useQueriesStore } from '@/store/queries-store'
import type { Connection } from '@/lib/api/connections'
import type { SavedQuery } from '@/lib/api/queries'
import type { Conflict } from '@/types/sync'

// ============================================================================
// Example 1: Connection Card with Visibility Toggle
// ============================================================================

export function ConnectionCardExample() {
  const { shareConnection, unshareConnection } = useConnectionsStore()

  const connection: Connection = {
    id: 'conn_123',
    user_id: 'user_456',
    organization_id: 'org_789',
    name: 'Production Database',
    description: 'Main PostgreSQL database for production environment',
    database_type: 'postgresql',
    host: 'db.example.com',
    port: 5432,
    database_name: 'myapp_prod',
    username: 'app_user',
    ssl_enabled: true,
    visibility: 'shared',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    created_by_email: 'john@example.com',
    last_used: new Date().toISOString(),
  }

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-bold">Connection with Visibility Toggle</h2>

      {/* Visibility Toggle */}
      <VisibilityToggle
        resourceId={connection.id}
        resourceType="connection"
        currentVisibility={connection.visibility}
        currentOrgId={connection.organization_id}
        ownerId={connection.user_id}
        onShare={shareConnection}
        onUnshare={unshareConnection}
        onUpdate={() => console.log('Connection visibility updated')}
      />

      {/* Connection Card */}
      <SharedResourceCard
        resource={connection}
        type="connection"
        onView={(resource) => {
          console.log('View connection:', resource)
          // Open connection details modal
        }}
        onEdit={(resource) => {
          console.log('Edit connection:', resource)
          // Open connection edit form
        }}
        onUnshare={(resource) => {
          console.log('Unshare connection:', resource)
          unshareConnection(resource.id)
        }}
        onDelete={(resource) => {
          console.log('Delete connection:', resource)
          // Show confirmation dialog, then delete
        }}
        onUse={(resource) => {
          console.log('Connect to database:', resource)
          // Initiate database connection
        }}
      />
    </div>
  )
}

// ============================================================================
// Example 2: Query Card with Tags and Actions
// ============================================================================

export function QueryCardExample() {
  const { shareQuery, unshareQuery } = useQueriesStore()

  const query: SavedQuery = {
    id: 'query_123',
    user_id: 'user_456',
    organization_id: 'org_789',
    title: 'Monthly Sales Report',
    description: 'Aggregates sales data by region and product category',
    sql_content: `SELECT
  region,
  category,
  SUM(amount) as total_sales,
  COUNT(*) as order_count
FROM sales
WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE)
GROUP BY region, category
ORDER BY total_sales DESC`,
    database_type: 'postgresql',
    tags: ['sales', 'reporting', 'monthly'],
    visibility: 'shared',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    created_by_email: 'jane@example.com',
    last_executed: new Date().toISOString(),
  }

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-bold">Query with Tags and Actions</h2>

      {/* Compact Visibility Badge */}
      <VisibilityToggle
        resourceId={query.id}
        resourceType="query"
        currentVisibility={query.visibility}
        currentOrgId={query.organization_id}
        ownerId={query.user_id}
        onShare={shareQuery}
        onUnshare={unshareQuery}
        onUpdate={() => console.log('Query visibility updated')}
        compact={true}
      />

      {/* Query Card */}
      <SharedResourceCard
        resource={query}
        type="query"
        onView={(resource) => {
          console.log('View query:', resource)
          // Open query details modal
        }}
        onEdit={(resource) => {
          console.log('Edit query:', resource)
          // Open query editor
        }}
        onUnshare={(resource) => {
          console.log('Unshare query:', resource)
          unshareQuery(resource.id)
        }}
        onDelete={(resource) => {
          console.log('Delete query:', resource)
          // Show confirmation, then delete
        }}
        onUse={(resource) => {
          console.log('Execute query:', resource)
          // Run query in active connection
        }}
      />
    </div>
  )
}

// ============================================================================
// Example 3: Conflict Resolution Flow
// ============================================================================

export function ConflictResolutionExample() {
  const [showConflict, setShowConflict] = useState(false)
  const [isResolving, setIsResolving] = useState(false)

  const conflict: Conflict<Connection> = {
    id: 'conflict_123',
    entityType: 'connection',
    entityId: 'conn_123',
    localVersion: {
      id: 'conn_123',
      user_id: 'user_456',
      name: 'Production DB (Local)',
      description: 'Updated locally',
      database_type: 'postgresql',
      host: 'db.example.com',
      port: 5432,
      database_name: 'myapp_prod',
      username: 'app_user',
      ssl_enabled: true,
      visibility: 'shared',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
    remoteVersion: {
      id: 'conn_123',
      user_id: 'user_456',
      name: 'Production DB (Remote)',
      description: 'Updated on server',
      database_type: 'postgresql',
      host: 'db2.example.com', // Different host
      port: 5432,
      database_name: 'myapp_prod',
      username: 'app_user',
      ssl_enabled: true,
      visibility: 'shared',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
    localSyncVersion: 5,
    remoteSyncVersion: 6,
    localUpdatedAt: new Date(Date.now() - 3600000), // 1 hour ago
    remoteUpdatedAt: new Date(Date.now() - 1800000), // 30 minutes ago
    recommendedResolution: 'remote', // Remote is newer
    reason: 'Connection was modified on another device',
  }

  const handleResolve = async (resolution: 'local' | 'remote') => {
    setIsResolving(true)

    try {
      // Simulate API call to resolve conflict
      await new Promise((resolve) => setTimeout(resolve, 1000))

      if (resolution === 'local') {
        console.log('Applying local version:', conflict.localVersion)
        // Upload local version to server
      } else {
        console.log('Applying remote version:', conflict.remoteVersion)
        // Update local database with remote version
      }

      console.log(`Conflict resolved using ${resolution} version`)
      setShowConflict(false)
    } catch (error) {
      console.error('Failed to resolve conflict:', error)
    } finally {
      setIsResolving(false)
    }
  }

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-bold">Conflict Resolution</h2>

      <button
        onClick={() => setShowConflict(true)}
        className="px-4 py-2 bg-amber-500 text-white rounded hover:bg-amber-600"
      >
        Simulate Conflict
      </button>

      <ConflictResolutionDialog
        open={showConflict}
        onOpenChange={setShowConflict}
        conflict={conflict}
        onResolve={handleResolve}
        loading={isResolving}
      />
    </div>
  )
}

// ============================================================================
// Example 4: Grid Layout with Multiple Resources
// ============================================================================

export function ResourceGridExample() {
  const connections: Connection[] = [
    {
      id: 'conn_1',
      user_id: 'user_1',
      organization_id: 'org_1',
      name: 'Production PostgreSQL',
      database_type: 'postgresql',
      host: 'db1.example.com',
      port: 5432,
      database_name: 'prod_db',
      username: 'app',
      ssl_enabled: true,
      visibility: 'shared',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      created_by_email: 'admin@example.com',
    },
    {
      id: 'conn_2',
      user_id: 'user_2',
      organization_id: 'org_1',
      name: 'Analytics MySQL',
      database_type: 'mysql',
      host: 'analytics.example.com',
      port: 3306,
      database_name: 'analytics',
      username: 'readonly',
      ssl_enabled: true,
      visibility: 'shared',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      created_by_email: 'data@example.com',
    },
    {
      id: 'conn_3',
      user_id: 'user_3',
      organization_id: 'org_1',
      name: 'Development MongoDB',
      database_type: 'mongodb',
      host: 'dev-mongo.example.com',
      port: 27017,
      database_name: 'dev_app',
      username: 'dev_user',
      ssl_enabled: false,
      visibility: 'shared',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      created_by_email: 'dev@example.com',
    },
  ]

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-bold">Resource Grid</h2>

      {/* Responsive Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {connections.map((connection) => (
          <SharedResourceCard
            key={connection.id}
            resource={connection}
            type="connection"
            onView={(r) => console.log('View:', r)}
            onEdit={(r) => console.log('Edit:', r)}
            onUnshare={(r) => console.log('Unshare:', r)}
            onDelete={(r) => console.log('Delete:', r)}
            onUse={(r) => console.log('Use:', r)}
          />
        ))}
      </div>
    </div>
  )
}

// ============================================================================
// Example 5: Integration in Connection Form
// ============================================================================

export function ConnectionFormIntegrationExample() {
  const [formData, setFormData] = useState<Partial<Connection>>({
    name: '',
    database_type: 'postgresql',
    host: '',
    port: 5432,
    database_name: '',
    username: '',
    ssl_enabled: false,
    visibility: 'personal',
  })

  const { shareConnection, unshareConnection, createConnection } =
    useConnectionsStore()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      // Create connection
      const newConnection = await createConnection(formData as any)
      console.log('Connection created:', newConnection)

      // If visibility is 'shared', share with current organization
      if (formData.visibility === 'shared' && formData.organization_id) {
        await shareConnection(newConnection.id, formData.organization_id)
      }
    } catch (error) {
      console.error('Failed to create connection:', error)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 max-w-md">
      <h2 className="text-2xl font-bold">Create Connection</h2>

      {/* Standard form fields... */}
      <div>
        <label className="block text-sm font-medium mb-1">Name</label>
        <input
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full px-3 py-2 border rounded"
          required
        />
      </div>

      {/* ... more fields ... */}

      {/* Visibility Toggle Integration */}
      <div className="pt-4 border-t">
        <VisibilityToggle
          resourceId="new" // Will be set after creation
          resourceType="connection"
          currentVisibility={formData.visibility as any}
          currentOrgId={formData.organization_id}
          onShare={(id, orgId) => {
            setFormData({ ...formData, visibility: 'shared', organization_id: orgId })
          }}
          onUnshare={(id) => {
            setFormData({ ...formData, visibility: 'personal', organization_id: undefined })
          }}
          onUpdate={() => {}}
          disabled={!formData.name} // Disable until connection is created
        />
      </div>

      <button
        type="submit"
        className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
      >
        Create Connection
      </button>
    </form>
  )
}
