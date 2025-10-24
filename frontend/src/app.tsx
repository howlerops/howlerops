import { useEffect, lazy, Suspense } from 'react'
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
import { Loader2 } from 'lucide-react'

// Lazy load pages for code splitting
const Dashboard = lazy(() => import('./pages/dashboard').then(m => ({ default: m.Dashboard })))
const Connections = lazy(() => import('./pages/connections').then(m => ({ default: m.Connections })))
const Settings = lazy(() => import('./pages/settings').then(m => ({ default: m.Settings })))
const InviteAcceptPage = lazy(() => import('./pages/InviteAcceptPage').then(m => ({ default: m.InviteAcceptPage })))
const PendingInvitationsPage = lazy(() => import('./pages/PendingInvitationsPage').then(m => ({ default: m.PendingInvitationsPage })))
const AnalyticsPage = lazy(() => import('./pages/AnalyticsPage'))

// Loading component
function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center h-full">
      <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
    </div>
  )
}

function App() {
  // Initialize stores and migrate credentials on app startup
  useEffect(() => {
    // Initialize stores
    initializeAuthStore()
    initializeTierStore()
    initializeOrganizationStore()

    // Migrate credentials from localStorage to OS keychain (one-time)
    import('./lib/migrate-credentials').then(({ migrateCredentialsToKeychain }) => {
      migrateCredentialsToKeychain().catch(err => {
        console.error('Credential migration failed:', err)
        // App continues normally even if migration fails
      })
    })
  }, [])

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
                  {/* Public invitation route - no MainLayout wrapper */}
                  <Route path="/invite/:token" element={<InviteAcceptPage />} />

                  {/* Protected routes with MainLayout */}
                  <Route
                    path="/*"
                    element={
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
