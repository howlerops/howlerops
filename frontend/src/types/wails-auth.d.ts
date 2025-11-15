/**
 * TypeScript definitions for Wails authentication backend bindings
 *
 * These types extend the window interface to include authentication-related
 * methods exposed by the Go backend via Wails runtime.
 */

export interface OAuthURLResponse {
  authUrl: string
  state: string
}

export interface BiometricAvailability {
  available: boolean
  type: string // "Touch ID", "Windows Hello", "Face ID", etc.
}

export interface AuthSuccessEvent {
  token: string
  user: {
    id: string
    email: string
    name?: string
    avatar_url?: string
  }
}

export interface AuthRestoredEvent {
  token: string
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          // Existing methods
          GetTableStructure?: () => Promise<unknown>

          // OAuth methods
          GetOAuthURL?: (provider: 'google' | 'github') => Promise<OAuthURLResponse>
          CheckStoredToken?: (provider: 'google' | 'github') => Promise<boolean>
          Logout?: (provider: 'google' | 'github') => Promise<void>

          // WebAuthn/Biometric methods
          CheckBiometricAvailability?: () => Promise<BiometricAvailability>
          StartWebAuthnRegistration?: (userID: string, userName: string) => Promise<string>
          FinishWebAuthnRegistration?: (userID: string, credentialJSON: string) => Promise<boolean>
          StartWebAuthnAuthentication?: () => Promise<string>
          FinishWebAuthnAuthentication?: (assertionJSON: string) => Promise<string>

          [key: string]: unknown
        }
        [key: string]: unknown
      }
      version?: string
      [key: string]: unknown
    }
    runtime?: {
      EventsOn?: (eventName: string, callback: (data: any) => void) => () => void
      EventsOff?: (eventName: string) => void
    }
  }
}

export {}
