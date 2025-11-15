/**
 * Biometric Auth Button Component
 *
 * Displays biometric authentication option (Touch ID, Face ID, Windows Hello).
 * Only shows when biometric hardware is available on the platform.
 * Currently shows UI placeholder - backend WebAuthn integration pending.
 */

import { Button } from '@/components/ui/button'
import { Fingerprint, Loader2 } from 'lucide-react'
import { useState, useEffect } from 'react'

interface BiometricAuthButtonProps {
  onSuccess?: () => void
}

export function BiometricAuthButton({ onSuccess: _onSuccess }: BiometricAuthButtonProps) {
  const [available, setAvailable] = useState(false)
  const [loading, setLoading] = useState(false)
  const [biometricType, setBiometricType] = useState<string>('Biometric')

  const checkBiometricAvailability = async () => {
    try {
      // TODO: Check with Wails backend for platform biometric capabilities
      // For now, show the button as a placeholder on all platforms

      // Check if browser supports WebAuthn (basic check)
      if (window.PublicKeyCredential) {
        setAvailable(true)

        // Detect platform for better labeling (rough detection)
        const platform = navigator.userAgent
        if (platform.includes('Mac')) {
          setBiometricType('Touch ID')
        } else if (platform.includes('Windows')) {
          setBiometricType('Windows Hello')
        } else {
          setBiometricType('Biometric')
        }
      }
    } catch {
      setAvailable(false)
    }
  }

  useEffect(() => {
    checkBiometricAvailability()
  }, [])

  const handleBiometricAuth = async () => {
    setLoading(true)

    // TODO: Implement WebAuthn biometric authentication
    // This will:
    // 1. Call Wails backend to create WebAuthn challenge
    // 2. Trigger browser's native biometric prompt
    // 3. Validate credential with backend
    // 4. Complete authentication

    // Placeholder: Show coming soon message
    setTimeout(() => {
      console.log('Biometric auth - Coming soon!')
      alert(`${biometricType} authentication coming soon!\n\nThis will use your device's biometric authentication.`)
      setLoading(false)
    }, 500)
  }

  // Don't show button if biometric auth not available
  if (!available) {
    return null
  }

  return (
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
  )
}
