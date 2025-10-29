import { isWailsEnvironment } from './wails-runtime'

const LOCAL_HOSTNAMES = new Set([
  'localhost',
  '127.0.0.1',
  '0.0.0.0',
  '::1',
  '[::1]',
])

/**
 * Determines if the app is running in a hosted web environment (e.g. Vercel)
 * where authentication should be enforced before accessing the main app.
 */
export function shouldEnforceHostedAuth(): boolean {
  if (import.meta.env.SSR) {
    return false
  }

  if (typeof window === 'undefined') {
    return false
  }

  if (isWailsEnvironment()) {
    return false
  }

  if (!import.meta.env.PROD) {
    return false
  }

  const { protocol, hostname } = window.location

  if (protocol === 'file:') {
    return false
  }

  if (!hostname || LOCAL_HOSTNAMES.has(hostname)) {
    return false
  }

  return true
}
