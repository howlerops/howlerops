/**
 * OAuth Button Group Component
 *
 * Displays OAuth provider buttons (Google, GitHub, etc.) for alternative login.
 * Integrates with Wails Go backend for OAuth flow via system browser.
 */

import { Github, Loader2 } from 'lucide-react'
import { useEffect,useState } from 'react'

import { Button } from '@/components/ui/button'
import * as authApi from '@/lib/auth-api'
import { isWailsApp } from '@/lib/platform'
import { subscribeToWailsEvent } from '@/lib/wails-guard'
import type { AuthSuccessEvent } from '@/types/wails-auth'

interface OAuthButtonGroupProps {
  onSuccess?: () => void
}

export function OAuthButtonGroup({ onSuccess }: OAuthButtonGroupProps) {
  const [loading, setLoading] = useState<'google' | 'github' | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Listen for auth events from backend (desktop mode only)
  useEffect(() => {
    // Only set up event listeners in Wails desktop mode
    if (!isWailsApp()) {
      return
    }

    try {
      const unsubscribeSuccess = subscribeToWailsEvent('auth:success', (data: AuthSuccessEvent) => {
        console.log('OAuth authentication successful:', data.user)
        setLoading(null)
        setError(null)
        onSuccess?.()
      })

      const unsubscribeError = subscribeToWailsEvent('auth:error', (message: string) => {
        console.error('OAuth authentication failed:', message)
        setError(message)
        setLoading(null)
      })

      return () => {
        unsubscribeSuccess()
        unsubscribeError()
      }
    } catch (err) {
      console.error('Failed to subscribe to auth events:', err)
      return () => {}
    }
  }, [onSuccess])

  const handleOAuthLogin = async (provider: 'google' | 'github') => {
    setLoading(provider)
    setError(null)

    try {
      // Get OAuth URL from backend (works in both modes)
      const { authUrl, state } = await authApi.getOAuthURL(provider)

      if (isWailsApp()) {
        // Desktop mode: Open in new window
        // Callback handled by OS custom protocol handler → auth:success event
        window.open(authUrl, '_blank')
        // Loading state will be cleared by auth:success or auth:error event
      } else {
        // Web mode: Redirect current window
        // Store state in sessionStorage for verification after redirect
        if (state) {
          sessionStorage.setItem('oauth_state', state)
        }
        window.location.href = authUrl
        // Loading state preserved across redirect
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to initiate OAuth flow'
      console.error('OAuth error:', err)
      setError(message)
      setLoading(null)
    }
  }

  return (
    <div className="space-y-3">
      {/* Error message */}
      {error && (
        <div className="text-sm text-red-600 dark:text-red-400 text-center">{error}</div>
      )}

      <div className="grid grid-cols-2 gap-3">
        {/* Google OAuth Button */}
        <Button
          variant="outline"
          className="w-full bg-white hover:bg-gray-50 border-gray-300 dark:bg-gray-800 dark:hover:bg-gray-700 dark:border-gray-600"
          onClick={() => handleOAuthLogin('google')}
          disabled={loading !== null}
        >
          {loading === 'google' ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <>
              <GoogleIcon className="h-4 w-4 mr-2" />
              <span className="text-gray-700 dark:text-gray-200">Google</span>
            </>
          )}
        </Button>

        {/* GitHub OAuth Button */}
        <Button
          variant="outline"
          className="w-full bg-[#24292e] hover:bg-[#1b1f23] text-white border-gray-700"
          onClick={() => handleOAuthLogin('github')}
          disabled={loading !== null}
        >
          {loading === 'github' ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <>
              <Github className="h-4 w-4 mr-2" />
              GitHub
            </>
          )}
        </Button>
      </div>
    </div>
  )
}

/**
 * Google Icon Component
 * Official Google logo for OAuth button
 */
function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24">
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
      />
    </svg>
  )
}
