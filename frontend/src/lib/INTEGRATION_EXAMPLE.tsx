/**
 * Integration Example for Credential Migration
 *
 * This file shows how to integrate the credential migration utility
 * into your React application. Copy the relevant parts into your app.tsx.
 */

import { useEffect } from 'react'

import { useMigrateCredentials } from './migrate-credentials'

/**
 * Example 1: Simple integration in App component
 */
export function AppWithMigration() {
  // This hook runs migration automatically on mount
  useMigrateCredentials()

  return (
    <div>
      {/* Your app components */}
    </div>
  )
}

/**
 * Example 2: Integration with custom logic
 */
export function AppWithCustomMigration() {
  useEffect(() => {
    // Custom migration logic with logging
    import('./migrate-credentials').then(({ migrateCredentialsToKeychain }) => {
      migrateCredentialsToKeychain()
        .then((result) => {
          if (result.success) {
            console.log(`✓ Successfully migrated ${result.migratedCount} credentials`)
          } else if (result.skipped) {
            console.log(`○ Migration skipped: ${result.reason}`)
          } else {
            console.warn(`⚠ Migration completed with errors:`, result.errors)
          }
        })
        .catch((error) => {
          console.error('Unexpected migration error:', error)
        })
    })
  }, [])

  return (
    <div>
      {/* Your app components */}
    </div>
  )
}

/**
 * Example 3: Integration with user notification
 */
export function AppWithNotification() {
  const [migrationStatus, setMigrationStatus] = useState<string | null>(null)

  useEffect(() => {
    import('./migrate-credentials').then(({ migrateCredentialsToKeychain }) => {
      migrateCredentialsToKeychain()
        .then((result) => {
          if (result.success && result.migratedCount > 0) {
            setMigrationStatus(
              `Your credentials have been securely migrated to the system keychain.`
            )
          }
        })
    })
  }, [])

  return (
    <div>
      {migrationStatus && (
        <div className="notification">
          {migrationStatus}
        </div>
      )}
      {/* Your app components */}
    </div>
  )
}

/**
 * Example 4: Integration with error boundary
 */
export function AppWithErrorBoundary() {
  useMigrateCredentials()

  // Migration errors are handled internally and don't throw,
  // so ErrorBoundary won't catch them. This is intentional
  // to prevent blocking app startup.

  return (
    <ErrorBoundary>
      <div>
        {/* Your app components */}
      </div>
    </ErrorBoundary>
  )
}

/**
 * Example 5: Recommended integration for app.tsx
 *
 * Add this to your existing app.tsx file:
 */
/*
import { useMigrateCredentials } from './lib/migrate-credentials'

function App() {
  // Initialize stores on app startup
  useEffect(() => {
    initializeAuthStore()
    initializeTierStore()
    initializeOrganizationStore()
  }, [])

  // Run credential migration (non-blocking)
  useMigrateCredentials()

  return (
    // ... rest of your app
  )
}
*/

/**
 * Example 6: Manual migration trigger (for settings page)
 */
export function MigrationSettings() {
  const [status, setStatus] = useState<string>('Checking...')
  const [canRetry, setCanRetry] = useState(false)

  useEffect(() => {
    import('./migrate-credentials').then(({ getMigrationStatus }) => {
      const status = getMigrationStatus()
      if (status.migrated) {
        setStatus('Credentials are stored securely in your system keychain.')
      } else if (!status.keychainAvailable) {
        setStatus('Keychain not available. Using session storage.')
        setCanRetry(false)
      } else if (status.hasCredentials) {
        setStatus('Credentials pending migration.')
        setCanRetry(true)
      } else {
        setStatus('No credentials to migrate.')
      }
    })
  }, [])

  const handleRetry = async () => {
    setStatus('Migrating...')
    const { retryMigration } = await import('./migrate-credentials')
    const result = await retryMigration()

    if (result.success) {
      setStatus('Migration successful!')
      setCanRetry(false)
    } else {
      setStatus(`Migration failed: ${result.errors[0]?.error || 'Unknown error'}`)
    }
  }

  return (
    <div>
      <h3>Credential Storage</h3>
      <p>{status}</p>
      {canRetry && (
        <button onClick={handleRetry}>
          Retry Migration
        </button>
      )}
    </div>
  )
}
