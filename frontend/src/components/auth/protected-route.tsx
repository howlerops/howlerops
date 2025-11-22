/**
 * Protected Route Component
 *
 * Wrapper component that requires authentication to access.
 * Redirects to home if user is not authenticated.
 */

import { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { useAuthStore } from '@/store/auth-store'

interface ProtectedRouteProps {
  children: ReactNode
  redirectTo?: string
}

export function ProtectedRoute({ children, redirectTo = '/' }: ProtectedRouteProps) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const location = useLocation()

  if (!isAuthenticated) {
    // Redirect to home but save the attempted location
    return <Navigate to={redirectTo} state={{ from: location }} replace />
  }

  return <>{children}</>
}
