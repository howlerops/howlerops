/**
 * OAuth Callback Handler Page
 *
 * Handles the OAuth callback redirect in web deployment mode.
 * Extracts the authorization code and state from URL parameters,
 * exchanges them for an access token, and updates the auth store.
 *
 * Desktop mode doesn't use this page - it handles callbacks via OS deep links.
 */

import { AlertCircle,Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { exchangeOAuthCode } from '@/lib/auth-api'
import { useAuthStore } from '@/store/auth-store'

export function AuthCallback() {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // Extract code and state from URL
        const params = new URLSearchParams(window.location.search)
        const code = params.get('code')
        const state = params.get('state')

        if (!code) {
          throw new Error('No authorization code received')
        }

        // Verify state matches what we stored (CSRF protection)
        const storedState = sessionStorage.getItem('oauth_state')
        if (state && storedState && state !== storedState) {
          throw new Error('Invalid state parameter - possible CSRF attack')
        }

        // Clean up stored state
        sessionStorage.removeItem('oauth_state')

        // Exchange code for token
        const data = await exchangeOAuthCode(code, state || '')

        // Update auth store with user and token
        useAuthStore.setState({
          user: {
            id: data.user.id,
            username: data.user.name || data.user.email,
            email: data.user.email,
            role: 'user',
            created_at: new Date().toISOString(),
          },
          tokens: {
            access_token: data.token,
            refresh_token: '', // OAuth tokens managed by backend
            expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // 24 hours
          },
          isAuthenticated: true,
          isLoading: false,
          error: null,
        })

        // Redirect to main app
        navigate('/dashboard', { replace: true })
      } catch (err) {
        const message = err instanceof Error ? err.message : 'OAuth callback failed'
        console.error('OAuth callback error:', err)
        setError(message)
      }
    }

    handleCallback()
  }, [navigate])

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="max-w-md w-full mx-4">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8 space-y-6">
            {/* Error Icon */}
            <div className="flex justify-center">
              <div className="rounded-full bg-red-100 dark:bg-red-900/20 p-3">
                <AlertCircle className="h-8 w-8 text-red-600 dark:text-red-400" />
              </div>
            </div>

            {/* Error Message */}
            <div className="text-center space-y-2">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Authentication Failed
              </h2>
              <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
            </div>

            {/* Actions */}
            <div className="space-y-3">
              <Button
                className="w-full"
                onClick={() => navigate('/auth', { replace: true })}
              >
                Back to Sign In
              </Button>
              <Button
                variant="outline"
                className="w-full"
                onClick={() => window.location.reload()}
              >
                Try Again
              </Button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Processing state
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="max-w-md w-full mx-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8 space-y-6">
          {/* Loading Icon */}
          <div className="flex justify-center">
            <div className="rounded-full bg-blue-100 dark:bg-blue-900/20 p-3">
              <Loader2 className="h-8 w-8 text-blue-600 dark:text-blue-400 animate-spin" />
            </div>
          </div>

          {/* Loading Message */}
          <div className="text-center space-y-2">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              Completing Sign In
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Please wait while we complete your authentication...
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
