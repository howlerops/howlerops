import { HashRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ThemeProvider } from './components/theme-provider'
import { NavigationProvider } from './components/navigation-provider'
import { MainLayout } from './components/layout/main-layout'
import { Dashboard } from './pages/dashboard'
import { Connections } from './pages/connections'
import { Settings } from './pages/settings'
import { queryClient } from './lib/api'
import { Toaster } from './components/ui/toaster'

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="system" storageKey="sql-studio-theme">
        <Router>
          <NavigationProvider>
            <MainLayout>
              <div className="flex flex-1 min-h-0 flex-col">
                <Routes>
                  <Route path="/" element={<Navigate to="/dashboard" replace />} />
                  <Route path="/dashboard" element={<Dashboard />} />
                  <Route path="/connections" element={<Connections />} />
                  <Route path="/settings" element={<Settings />} />
                </Routes>
              </div>
            </MainLayout>
          </NavigationProvider>
        </Router>
        <ReactQueryDevtools initialIsOpen={false} />
        <Toaster />
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export default App
