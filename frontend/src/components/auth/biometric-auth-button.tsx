/**
 * Biometric Auth Button Component
 *
 * Displays biometric authentication option (Touch ID, Face ID, Windows Hello).
 * Only shows when biometric hardware is available on the platform.
 * Integrates with Wails Go backend for WebAuthn authentication.
 */

import { Button } from '@/components/ui/button'
import { Fingerprint, Loader2 } from 'lucide-react'
import { useState, useEffect } from 'react'
import { parsePublicKeyRequestOptions, serializeCredentialAssertion } from '@/lib/utils/webauthn'
import * as authApi from '@/lib/auth-api'

interface BiometricAuthButtonProps {
  onSuccess?: () => void
}

export function BiometricAuthButton({ onSuccess }: BiometricAuthButtonProps) {
  const [available, setAvailable] = useState(false)
  const [loading, setLoading] = useState(false)
  const [biometricType, setBiometricType] = useState<string>('Biometric')
  const [error, setError] = useState<string | null>(null)

  const checkBiometricAvailability = async () => {
    try {
      // Check browser support for WebAuthn
      if (!window.PublicKeyCredential) {
        setAvailable(false)
        return
      }

      // Check backend for platform biometric capabilities (works in both modes)
      const result = await authApi.checkBiometricAvailability()
      setAvailable(result.available)
      setBiometricType(result.type || 'Biometric')
    } catch (err) {
      console.error('Failed to check biometric availability:', err)
      setAvailable(false)
    }
  }

  useEffect(() => {
    checkBiometricAvailability()
  }, [])

  const handleBiometricAuth = async () => {
    setLoading(true)
    setError(null)

    try {
      // Get WebAuthn challenge from backend (works in both modes)
      const optionsJSON = await authApi.startWebAuthnAuthentication()
      const options = parsePublicKeyRequestOptions(optionsJSON)

      // Trigger browser's native biometric prompt
      const credential = await navigator.credentials.get({
        publicKey: options,
      })

      if (!credential || credential.type !== 'public-key') {
        throw new Error('Invalid credential type')
      }

      // Send credential to backend for verification (works in both modes)
      const assertionJSON = serializeCredentialAssertion(credential as PublicKeyCredential)
      const token = await authApi.finishWebAuthnAuthentication(assertionJSON)

      if (token) {
        console.log('Biometric authentication successful')
        onSuccess?.()
      } else {
        throw new Error('Authentication failed')
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Biometric authentication failed'
      console.error('Biometric auth error:', err)
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  // Don't show button if biometric auth not available
  if (!available) {
    return null
  }

  return (
    <div className="space-y-2">
      {/* Error message */}
      {error && <div className="text-sm text-red-600 dark:text-red-400 text-center">{error}</div>}

      <Button
        variant="outline"
        className="w-full border-dashed hover:border-solid"
        onClick={handleBiometricAuth}
        disabled={loading}
      >
        {loading ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin mr-2" />
            Authenticating...
          </>
        ) : (
          <>
            <Fingerprint className="h-4 w-4 mr-2" />
            Sign in with {biometricType}
          </>
        )}
      </Button>
    </div>
  )
}
