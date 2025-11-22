/**
 * Dual-Mode Authentication API
 *
 * Provides a unified interface for authentication that works in both:
 * 1. Wails desktop mode - Uses direct Go backend calls via window.go.main.App
 * 2. Web deployment mode - Uses HTTP API endpoints via fetch
 *
 * The caller doesn't need to know which mode is active - this module handles it transparently.
 */

import { getApiBaseUrl,isWailsApp } from './platform'
import { callWails } from './wails-guard'

// ============================================================================
// Types
// ============================================================================

export interface OAuthInitiateResponse {
  authUrl: string
  state?: string
}

export interface OAuthExchangeResponse {
  token: string
  user: {
    id: string
    name: string
    email: string
  }
}

export interface BiometricAvailability {
  available: boolean
  type?: string
}

// ============================================================================
// OAuth Methods
// ============================================================================

/**
 * Get OAuth authorization URL for the specified provider
 */
export async function getOAuthURL(provider: 'google' | 'github'): Promise<OAuthInitiateResponse> {
  if (isWailsApp()) {
    // Desktop mode: Use Wails direct call
    return callWails((app) => app.GetOAuthURL!(provider))
  } else {
    // Web mode: Use HTTP API
    const apiBaseUrl = getApiBaseUrl()
    const response = await fetch(`${apiBaseUrl}/api/auth/oauth/initiate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider, platform: 'web' }),
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Failed to initiate OAuth flow' }))
      throw new Error(error.message || 'Failed to initiate OAuth flow')
    }

    const data = await response.json()
    return {
      authUrl: data.auth_url,
      state: data.state,
    }
  }
}

/**
 * Exchange OAuth authorization code for access token
 * Only used in web mode - desktop mode handles callback via OS deep link
 */
export async function exchangeOAuthCode(
  code: string,
  state: string
): Promise<OAuthExchangeResponse> {
  if (isWailsApp()) {
    throw new Error('OAuth code exchange not supported in desktop mode')
  }

  // Web mode: Exchange code for token
  const apiBaseUrl = getApiBaseUrl()
  const response = await fetch(`${apiBaseUrl}/api/auth/oauth/exchange`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code, state }),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'OAuth exchange failed' }))
    throw new Error(error.message || 'OAuth exchange failed')
  }

  return response.json()
}

/**
 * Check for stored OAuth token
 * Only used in desktop mode - web mode uses session cookies
 */
export async function checkStoredToken(provider: 'google' | 'github'): Promise<boolean> {
  if (isWailsApp()) {
    // Desktop mode: Check keychain for stored token
    return callWails((app) => app.CheckStoredToken!(provider))
  } else {
    // Web mode: Not applicable (uses session cookies)
    return false
  }
}

// ============================================================================
// WebAuthn / Biometric Methods
// ============================================================================

/**
 * Check if biometric authentication is available on this platform
 */
export async function checkBiometricAvailability(): Promise<BiometricAvailability> {
  // Check browser support first
  if (!window.PublicKeyCredential) {
    return { available: false }
  }

  if (isWailsApp()) {
    // Desktop mode: Check via Wails
    return callWails((app) => app.CheckBiometricAvailability!())
  } else {
    // Web mode: Check via HTTP API
    const apiBaseUrl = getApiBaseUrl()
    try {
      const response = await fetch(`${apiBaseUrl}/api/auth/webauthn/available`)

      if (!response.ok) {
        return { available: false }
      }

      return response.json()
    } catch (error) {
      console.error('Failed to check biometric availability:', error)
      return { available: false }
    }
  }
}

/**
 * Start WebAuthn authentication flow (get challenge)
 */
export async function startWebAuthnAuthentication(): Promise<string> {
  if (isWailsApp()) {
    // Desktop mode: Use Wails
    return callWails((app) => app.StartWebAuthnAuthentication!())
  } else {
    // Web mode: Use HTTP API
    const apiBaseUrl = getApiBaseUrl()
    const response = await fetch(`${apiBaseUrl}/api/auth/webauthn/login/begin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Failed to start authentication' }))
      throw new Error(error.message || 'Failed to start authentication')
    }

    const data = await response.json()
    return data.options_json
  }
}

/**
 * Finish WebAuthn authentication (verify assertion)
 */
export async function finishWebAuthnAuthentication(assertionJSON: string): Promise<string> {
  if (isWailsApp()) {
    // Desktop mode: Use Wails
    return callWails((app) => app.FinishWebAuthnAuthentication!(assertionJSON))
  } else {
    // Web mode: Use HTTP API
    const apiBaseUrl = getApiBaseUrl()
    const response = await fetch(`${apiBaseUrl}/api/auth/webauthn/login/finish`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assertion_json: assertionJSON }),
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Authentication failed' }))
      throw new Error(error.message || 'Authentication failed')
    }

    const data = await response.json()
    return data.token
  }
}

/**
 * Start WebAuthn registration flow (for new credential)
 */
export async function startWebAuthnRegistration(userId: string, username: string): Promise<string> {
  if (isWailsApp()) {
    // Desktop mode: Use Wails
    return callWails((app) => app.StartWebAuthnRegistration!(userId, username))
  } else {
    // Web mode: Use HTTP API
    const apiBaseUrl = getApiBaseUrl()
    const response = await fetch(`${apiBaseUrl}/api/auth/webauthn/register/begin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, username }),
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Failed to start registration' }))
      throw new Error(error.message || 'Failed to start registration')
    }

    const data = await response.json()
    return data.options_json
  }
}

/**
 * Finish WebAuthn registration (store credential)
 */
export async function finishWebAuthnRegistration(userId: string, credentialJSON: string): Promise<boolean> {
  if (isWailsApp()) {
    // Desktop mode: Use Wails
    return callWails((app) => app.FinishWebAuthnRegistration!(userId, credentialJSON))
  } else {
    // Web mode: Use HTTP API
    const apiBaseUrl = getApiBaseUrl()
    const response = await fetch(`${apiBaseUrl}/api/auth/webauthn/register/finish`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, credential_json: credentialJSON }),
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Registration failed' }))
      throw new Error(error.message || 'Registration failed')
    }

    const data = await response.json()
    return data.success
  }
}
