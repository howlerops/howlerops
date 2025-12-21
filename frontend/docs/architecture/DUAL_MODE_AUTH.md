# Dual-Mode Authentication System

## Overview

The HowlerOps frontend supports **both Wails desktop mode and web deployment mode** with a unified authentication API. The system automatically detects the runtime environment and uses the appropriate authentication flow without requiring code changes.

## Architecture

### Platform Detection

**File: `src/lib/platform.ts`**

Provides utilities to detect whether the app is running in Wails desktop mode or web mode:

```typescript
import { isWailsApp, isWebApp, getPlatformType } from '@/lib/platform'

if (isWailsApp()) {
  // Running in desktop mode
} else {
  // Running in web mode
}
```

### Dual-Mode Auth API

**File: `src/lib/auth-api.ts`**

A unified authentication API that works transparently in both modes:

- **Desktop mode**: Uses Wails Go backend methods via `window.go.main.App.*`
- **Web mode**: Uses HTTP API endpoints via `fetch()`

The caller doesn't need to know which mode is active - the auth API handles it automatically.

## Authentication Flows

### OAuth Authentication

#### Desktop Mode (Wails)

1. User clicks OAuth button
2. Frontend calls `authApi.getOAuthURL(provider)`
   - Uses Wails method: `window.go.main.App.GetOAuthURL()`
3. Opens OAuth URL in **new window** via `window.open()`
4. User authenticates with provider
5. Provider redirects to custom OS protocol handler (e.g., `howlerops://`)
6. Wails backend receives callback and emits `auth:success` event
7. Frontend receives event and updates auth state

**Flow diagram:**
```
User Click → Get OAuth URL (Wails) → Open New Window → OS Deep Link Callback → Wails Event → Auth Store Updated
```

#### Web Mode (HTTP)

1. User clicks OAuth button
2. Frontend calls `authApi.getOAuthURL(provider)`
   - Uses HTTP: `POST /api/auth/oauth/initiate`
   - Returns `{auth_url, state}`
3. Stores `state` in `sessionStorage` for CSRF protection
4. Redirects **current window** via `window.location.href = authUrl`
5. User authenticates with provider
6. Provider redirects back to `https://app.howlerops.com/auth/callback?code=...&state=...`
7. `AuthCallback` page extracts code and state
8. Calls `POST /api/auth/oauth/exchange` with code and state
9. Receives access token and user data
10. Updates auth store and redirects to `/dashboard`

**Flow diagram:**
```
User Click → Get OAuth URL (HTTP) → Store State → Redirect Window → Provider Callback → /auth/callback Page → Exchange Code (HTTP) → Auth Store Updated → Redirect to Dashboard
```

### WebAuthn (Biometric) Authentication

WebAuthn works **the same way in both modes** because it's a browser-based API:

1. User clicks biometric button
2. Frontend calls `authApi.startWebAuthnAuthentication()`
   - Desktop: Uses Wails method
   - Web: Uses HTTP `POST /api/auth/webauthn/login/begin`
3. Receives challenge from backend
4. Calls browser's `navigator.credentials.get()` to trigger biometric prompt
5. Browser shows native biometric UI (Touch ID, Face ID, Windows Hello)
6. User authenticates with biometric
7. Browser returns credential assertion
8. Frontend calls `authApi.finishWebAuthnAuthentication(assertionJSON)`
   - Desktop: Uses Wails method
   - Web: Uses HTTP `POST /api/auth/webauthn/login/finish`
9. Receives access token
10. Updates auth store

**Flow diagram (same for both modes):**
```
User Click → Get Challenge → Browser WebAuthn API → User Biometric → Verify Assertion → Receive Token → Auth Store Updated
```

## Key Files

### Core Authentication

- **`src/lib/platform.ts`** - Platform detection utilities
- **`src/lib/auth-api.ts`** - Dual-mode auth API wrapper
- **`src/store/auth-store.ts`** - Authentication state management
- **`src/pages/AuthCallback.tsx`** - OAuth callback handler (web mode only)

### Components

- **`src/components/auth/oauth-button-group.tsx`** - OAuth provider buttons
- **`src/components/auth/biometric-auth-button.tsx`** - Biometric auth button
- **`src/components/auth/protected-route.tsx`** - Route protection

### Router

- **`src/app.tsx`** - Route configuration including `/auth/callback`

## API Endpoints (Web Mode)

### OAuth Endpoints

**POST `/api/auth/oauth/initiate`**
```json
Request:
{
  "provider": "google" | "github",
  "platform": "web"
}

Response:
{
  "auth_url": "https://accounts.google.com/...",
  "state": "random-state-token"
}
```

**GET `/api/auth/oauth/callback`**
- Handles OAuth provider redirect
- Query params: `code`, `state`

**POST `/api/auth/oauth/exchange`**
```json
Request:
{
  "code": "authorization-code",
  "state": "state-token"
}

Response:
{
  "token": "jwt-access-token",
  "user": {
    "id": "user-id",
    "name": "User Name",
    "email": "user@example.com"
  }
}
```

### WebAuthn Endpoints

**GET `/api/auth/webauthn/available`**
```json
Response:
{
  "available": true,
  "type": "Touch ID" | "Face ID" | "Windows Hello"
}
```

**POST `/api/auth/webauthn/login/begin`**
```json
Response:
{
  "options_json": "{...publicKeyCredentialRequestOptions...}"
}
```

**POST `/api/auth/webauthn/login/finish`**
```json
Request:
{
  "assertion_json": "{...publicKeyCredential...}"
}

Response:
{
  "token": "jwt-access-token"
}
```

**POST `/api/auth/webauthn/register/begin`**
```json
Request:
{
  "user_id": "user-id",
  "username": "username"
}

Response:
{
  "options_json": "{...publicKeyCredentialCreationOptions...}"
}
```

**POST `/api/auth/webauthn/register/finish`**
```json
Request:
{
  "credential_json": "{...publicKeyCredential...}"
}

Response:
{
  "success": true
}
```

## Environment Variables

### Web Deployment

Create a `.env.production` file:

```bash
# Backend API URL
VITE_API_URL=https://api.howlerops.com

# OAuth Callback URL (must match provider configuration)
VITE_OAUTH_CALLBACK_URL=https://app.howlerops.com/auth/callback
```

### Desktop Mode

No environment variables needed - uses direct Wails calls.

## Testing Checklist

### Desktop Mode (Wails)

- [ ] OAuth buttons open new window with provider login
- [ ] OS deep link callback works after OAuth success
- [ ] `auth:success` event updates auth state
- [ ] WebAuthn shows native biometric prompt
- [ ] Biometric auth completes successfully
- [ ] Stored tokens are retrieved from OS keychain on app restart

### Web Mode (HTTP)

- [ ] OAuth buttons redirect current window to provider
- [ ] Provider redirects back to `/auth/callback`
- [ ] State parameter is verified (CSRF protection)
- [ ] Code exchange returns valid token
- [ ] User is redirected to `/dashboard` after success
- [ ] WebAuthn shows native biometric prompt
- [ ] Biometric auth completes successfully
- [ ] Error states show appropriate messages

### Error Handling

- [ ] Invalid OAuth code shows error
- [ ] State mismatch shows CSRF warning
- [ ] Network errors show retry option
- [ ] Biometric unavailable hides button
- [ ] Token expiration triggers re-authentication

## Security Considerations

### CSRF Protection (Web Mode)

OAuth flow uses state parameter to prevent CSRF attacks:
1. Generate random state before redirect
2. Store in `sessionStorage`
3. Verify matches on callback
4. Clear from storage after use

### Token Storage

- **Desktop mode**: OAuth tokens stored in OS keychain (secure)
- **Web mode**: Access tokens in Zustand persisted storage (encrypted at rest)
- **Master keys**: Never persisted, only in memory during session

### WebAuthn Security

- Challenge/response prevents replay attacks
- Credentials tied to origin (phishing protection)
- Private keys never leave secure enclave
- User verification required (biometric/PIN)

## Common Issues

### Issue: "Wails runtime not available"

**Cause**: Trying to use Wails methods in web mode

**Solution**: Ensure using `authApi.*` methods instead of direct `callWails()`. The auth API handles mode detection.

### Issue: OAuth callback shows "No authorization code received"

**Cause**: Provider didn't include `code` in callback URL

**Solution**:
1. Check OAuth provider configuration
2. Verify redirect URI matches exactly
3. Check for provider-specific errors in URL (`error` parameter)

### Issue: "Invalid state parameter - possible CSRF attack"

**Cause**: State mismatch between initiation and callback

**Solution**:
1. Ensure not blocking `sessionStorage`
2. Check for multiple browser tabs interfering
3. Verify not clearing storage between redirect

### Issue: Biometric button doesn't appear

**Cause**: WebAuthn not available

**Solution**:
1. Check browser supports WebAuthn (modern browsers only)
2. Verify HTTPS (required for WebAuthn in production)
3. Check backend reports biometric hardware available

## Migration Guide

### From Wails-Only to Dual-Mode

If you have existing Wails-only code:

1. **Replace direct Wails calls** with auth API:
   ```typescript
   // Before
   const { authUrl } = await callWails((app) => app.GetOAuthURL(provider))

   // After
   const { authUrl, state } = await authApi.getOAuthURL(provider)
   ```

2. **Update OAuth button logic**:
   ```typescript
   // Before
   window.open(authUrl, '_blank')

   // After
   if (isWailsApp()) {
     window.open(authUrl, '_blank')
   } else {
     if (state) sessionStorage.setItem('oauth_state', state)
     window.location.href = authUrl
   }
   ```

3. **Add callback route** to handle web OAuth redirects

4. **Update event listeners** to only run in Wails mode:
   ```typescript
   if (isWailsApp() && window.runtime) {
     subscribeToWailsEvent('auth:success', handler)
   }
   ```

## Future Enhancements

Potential improvements to consider:

1. **Token Refresh**: Automatic background token refresh
2. **Multi-Factor Auth**: Additional verification step
3. **Social Login Providers**: Add Microsoft, Apple, etc.
4. **Session Management**: Multi-device session tracking
5. **Audit Logging**: Track authentication events
6. **Rate Limiting**: Prevent brute force attacks

## Support

For issues or questions:
- Check console for detailed error messages
- Verify environment variables are set correctly
- Test in both desktop and web modes
- Review backend logs for API errors
