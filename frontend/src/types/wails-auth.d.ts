/**
 * TypeScript definitions for Wails v3 authentication backend bindings
 *
 * In Wails v3, backend methods are imported directly from generated bindings.
 * This file provides type definitions for authentication-related events and data.
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

// In v3, we use the Events module from @wailsio/runtime for events
// No need for window.go or window.runtime declarations

export {}
