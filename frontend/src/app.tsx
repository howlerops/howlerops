import { useEffect, lazy, Suspense, useMemo } from 'react'
import { HashRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ThemeProvider } from './components/theme-provider'
import { NavigationProvider } from './components/navigation-provider'
import { MainLayout } from './components/layout/main-layout'
import { ErrorBoundary } from './components/error-boundary'
import { queryClient } from './lib/api'
import { Toaster } from './components/ui/toaster'
import { initializeAuthStore } from './store/auth-store'
import { initializeTierStore } from './store/tier-store'
import { initializeOrganizationStore } from './store/organization-store'
import { initializeConnectionStore } from './store/connection-store'
import { Loader2 } from 'lucide-react'
import { ProtectedRoute } from './components/auth/protected-route'
import { shouldEnforceHostedAuth } from './lib/environment'

// Lazy load pages for code splitting
const Dashboard = lazy(() => import('./pages/dashboard').then(m => ({ default: m.Dashboard })))
const Connections = lazy(() => import('./pages/connections').then(m => ({ default: m.Connections })))
const Settings = lazy(() => import('./pages/settings').then(m => ({ default: m.Settings })))
const InviteAcceptPage = lazy(() => import('./pages/InviteAcceptPage').then(m => ({ default: m.InviteAcceptPage })))
const PendingInvitationsPage = lazy(() => import('./pages/PendingInvitationsPage').then(m => ({ default: m.PendingInvitationsPage })))
const AnalyticsPage = lazy(() => import('./pages/AnalyticsPage'))
const AuthPage = lazy(() => import('./pages/AuthPage').then(m => ({ default: m.AuthPage })))

// Loading component
function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center h-full">
      <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
    </div>
  )
}

function App() {
  const enforceAuth = useMemo(() => shouldEnforceHostedAuth(), [])

  // Initialize stores and migrate credentials on app startup
  useEffect(() => {
    // Initialize stores
    initializeAuthStore()
    initializeTierStore()

    // Initialize organization store (only works with Individual/Team tiers with backend API)
    initializeOrganizationStore().catch(err => {
      // Silently ignore - organization features require backend API (not available in local tier)
      console.debug('Organization features not available:', err.message)
    })

    // Migrate credentials from localStorage to OS keychain (one-time)
    import('./lib/migrate-credentials').then(({ migrateCredentialsToKeychain }) => {
      migrateCredentialsToKeychain().catch(err => {
        console.error('Credential migration failed:', err)
        // App continues normally even if migration fails
      })
    })

    // Auto-connect to last active connection
    // Add small delay to ensure store hydration is complete
    const autoConnectTimer = setTimeout(() => {
      initializeConnectionStore().catch(err => {
        console.error('Auto-connect failed:', err)
        // App continues normally even if auto-connect fails
      })
    }, 100)

    return () => clearTimeout(autoConnectTimer)
  }, [])

  const mainAppRoutes = (
    <MainLayout>
      <div className="flex flex-1 min-h-0 flex-col">
        <Suspense fallback={<LoadingSpinner />}>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/connections" element={<Connections />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/invitations" element={<PendingInvitationsPage />} />
            <Route path="/analytics" element={<AnalyticsPage />} />
          </Routes>
        </Suspense>
      </div>
    </MainLayout>
  )

  return (
    <ErrorBoundary
      onError={(error, errorInfo) => {
        console.error('App-level error caught:', error, errorInfo)
        // You could integrate with error reporting services here
      }}
    >
      <QueryClientProvider client={queryClient}>
        <ThemeProvider defaultTheme="system" storageKey="sql-studio-theme">
          <Router>
            <NavigationProvider>
              <Suspense fallback={<LoadingSpinner />}>
                <Routes>
                  <Route path="/auth" element={<AuthPage />} />

                  {/* Public invitation route - no MainLayout wrapper */}
                  <Route path="/invite/:token" element={<InviteAcceptPage />} />

                  {/* Protected routes with MainLayout */}
                  <Route
                    path="/*"
                    element={
                      enforceAuth ? (
                        <ProtectedRoute redirectTo="/auth">{mainAppRoutes}</ProtectedRoute>
                      ) : (
                        mainAppRoutes
                      )
                    }
                  />
                </Routes>
              </Suspense>
            </NavigationProvider>
          </Router>
          <ReactQueryDevtools initialIsOpen={false} />
          <Toaster />
        </ThemeProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}

export default App
