# Dual-Mode Authentication Architecture

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Frontend Application                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              UI Components Layer                         │  │
│  │  • AuthPage                                              │  │
│  │  • OAuthButtonGroup                                      │  │
│  │  • BiometricAuthButton                                   │  │
│  │  • AuthCallback (web only)                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Auth Store (Zustand)                        │  │
│  │  • signInWithOAuth()                                     │  │
│  │  • signInWithBiometric()                                 │  │
│  │  • User state management                                 │  │
│  │  • Token management                                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Auth API Wrapper                            │  │
│  │  • Platform detection                                    │  │
│  │  • Mode-aware method routing                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│         ┌────────────────┴────────────────┐                     │
│         ↓                                 ↓                     │
│  ┌──────────────┐                ┌──────────────────┐          │
│  │ Platform.ts  │                │   Auth-API.ts    │          │
│  │ Detection    │                │   Dual Routing   │          │
│  └──────────────┘                └──────────────────┘          │
│         ↓                                 ↓                     │
│    if Desktop?                       if Desktop?                │
│         │                                 │                     │
└─────────┼─────────────────────────────────┼─────────────────────┘
          │                                 │
          │                                 │
    ┌─────┴─────┐                     ┌─────┴─────┐
    │   YES     │                     │    NO     │
    │ (Wails)   │                     │   (Web)   │
    └─────┬─────┘                     └─────┬─────┘
          ↓                                 ↓
┌─────────────────────┐           ┌─────────────────────┐
│  Wails Go Backend   │           │   HTTP API Server   │
│  (Desktop Mode)     │           │   (Web Mode)        │
├─────────────────────┤           ├─────────────────────┤
│ • Direct Go calls   │           │ • REST endpoints    │
│ • window.go.main    │           │ • POST /api/auth/*  │
│ • OS keychain       │           │ • Session cookies   │
│ • Deep link handler │           │ • HTTP callbacks    │
└─────────────────────┘           └─────────────────────┘
```

## OAuth Flow Comparison

### Desktop Mode (Wails)

```
┌──────────┐
│   User   │
└────┬─────┘
     │ 1. Clicks OAuth button
     ↓
┌────────────────────┐
│  OAuth Component   │
└────────┬───────────┘
     │ 2. authApi.getOAuthURL(provider)
     ↓
┌────────────────────┐
│   Auth API Wrapper │
└────────┬───────────┘
     │ 3. Detects Wails mode
     │ 4. Calls window.go.main.App.GetOAuthURL()
     ↓
┌────────────────────┐
│  Wails Go Backend  │
└────────┬───────────┘
     │ 5. Returns {authUrl, state}
     ↓
┌────────────────────┐
│  OAuth Component   │
└────────┬───────────┘
     │ 6. window.open(authUrl, '_blank')
     ↓
┌────────────────────┐
│  System Browser    │  ← Opens in new window
│  (OAuth Provider)  │
└────────┬───────────┘
     │ 7. User authenticates
     │ 8. Redirects to: howlerops://callback?code=...
     ↓
┌────────────────────┐
│  OS Deep Link      │  ← Custom protocol handler
│  Handler           │
└────────┬───────────┘
     │ 9. Wails backend receives callback
     ↓
┌────────────────────┐
│  Wails Go Backend  │
└────────┬───────────┘
     │ 10. Emits auth:success event
     ↓
┌────────────────────┐
│   Auth Store       │  ← Event listener
└────────┬───────────┘
     │ 11. Updates user state
     │ 12. Stores token
     ↓
┌────────────────────┐
│   Dashboard        │  ← Authenticated!
└────────────────────┘
```

### Web Mode (HTTP)

```
┌──────────┐
│   User   │
└────┬─────┘
     │ 1. Clicks OAuth button
     ↓
┌────────────────────┐
│  OAuth Component   │
└────────┬───────────┘
     │ 2. authApi.getOAuthURL(provider)
     ↓
┌────────────────────┐
│   Auth API Wrapper │
└────────┬───────────┘
     │ 3. Detects Web mode
     │ 4. POST /api/auth/oauth/initiate
     ↓
┌────────────────────┐
│  HTTP API Server   │
└────────┬───────────┘
     │ 5. Returns {auth_url, state}
     ↓
┌────────────────────┐
│  OAuth Component   │
└────────┬───────────┘
     │ 6. sessionStorage.setItem('oauth_state', state)
     │ 7. window.location.href = authUrl
     ↓
┌────────────────────┐
│  OAuth Provider    │  ← Current window redirects
│  (Google/GitHub)   │
└────────┬───────────┘
     │ 8. User authenticates
     │ 9. Redirects to: https://app.howlerops.com/auth/callback?code=...&state=...
     ↓
┌────────────────────┐
│  AuthCallback Page │  ← Route: /auth/callback
└────────┬───────────┘
     │ 10. Extract code and state from URL
     │ 11. Verify state matches sessionStorage
     │ 12. authApi.exchangeOAuthCode(code, state)
     ↓
┌────────────────────┐
│   Auth API Wrapper │
└────────┬───────────┘
     │ 13. POST /api/auth/oauth/exchange
     ↓
┌────────────────────┐
│  HTTP API Server   │
└────────┬───────────┘
     │ 14. Returns {token, user}
     ↓
┌────────────────────┐
│  AuthCallback Page │
└────────┬───────────┘
     │ 15. Update auth store with user and token
     │ 16. navigate('/dashboard')
     ↓
┌────────────────────┐
│   Dashboard        │  ← Authenticated!
└────────────────────┘
```

## WebAuthn Flow (Same for Both Modes)

```
┌──────────┐
│   User   │
└────┬─────┘
     │ 1. Clicks biometric button
     ↓
┌───────────────────────┐
│ Biometric Component   │
└────────┬──────────────┘
     │ 2. authApi.startWebAuthnAuthentication()
     ↓
┌────────────────────┐
│   Auth API Wrapper │  ← Mode detection happens here
└────────┬───────────┘
     │ 3a. Desktop: Wails call
     │ 3b. Web: POST /api/auth/webauthn/login/begin
     ↓
┌────────────────────┐
│   Backend          │  ← Wails or HTTP
└────────┬───────────┘
     │ 4. Returns challenge (options_json)
     ↓
┌───────────────────────┐
│ Biometric Component   │
└────────┬──────────────┘
     │ 5. navigator.credentials.get(options)
     ↓
┌────────────────────┐
│   Browser WebAuthn │  ← Native browser API
│   Secure Enclave   │
└────────┬───────────┘
     │ 6. Shows biometric prompt
     │ 7. User authenticates (Touch ID, Face ID, etc.)
     │ 8. Returns credential assertion
     ↓
┌───────────────────────┐
│ Biometric Component   │
└────────┬──────────────┘
     │ 9. authApi.finishWebAuthnAuthentication(assertion)
     ↓
┌────────────────────┐
│   Auth API Wrapper │
└────────┬───────────┘
     │ 10a. Desktop: Wails call
     │ 10b. Web: POST /api/auth/webauthn/login/finish
     ↓
┌────────────────────┐
│   Backend          │
└────────┬───────────┘
     │ 11. Verifies assertion
     │ 12. Returns access token
     ↓
┌───────────────────────┐
│ Biometric Component   │
└────────┬──────────────┘
     │ 13. Update auth store
     ↓
┌────────────────────┐
│   Dashboard        │  ← Authenticated!
└────────────────────┘
```

## File Structure

```
frontend/
├── src/
│   ├── lib/
│   │   ├── platform.ts           ← NEW: Platform detection
│   │   ├── auth-api.ts           ← NEW: Dual-mode API wrapper
│   │   ├── wails-guard.ts        ← Existing: Wails call wrapper
│   │   └── environment.ts        ← Existing: Environment detection
│   │
│   ├── store/
│   │   └── auth-store.ts         ← MODIFIED: Uses auth-api wrapper
│   │
│   ├── components/auth/
│   │   ├── oauth-button-group.tsx      ← MODIFIED: Dual-mode OAuth
│   │   └── biometric-auth-button.tsx   ← MODIFIED: Dual-mode WebAuthn
│   │
│   ├── pages/
│   │   ├── AuthPage.tsx          ← Existing: Login page
│   │   └── AuthCallback.tsx      ← NEW: OAuth callback handler (web only)
│   │
│   └── app.tsx                   ← MODIFIED: Added /auth/callback route
│
├── DUAL_MODE_AUTH.md             ← NEW: Complete documentation
├── DUAL_MODE_IMPLEMENTATION_SUMMARY.md  ← NEW: Implementation summary
└── DUAL_MODE_ARCHITECTURE.md     ← NEW: This file
```

## Decision Flow in Auth API

```
                    ┌─────────────────────┐
                    │  Auth API Method    │
                    │  (e.g., getOAuthURL)│
                    └──────────┬──────────┘
                               │
                               ↓
                    ┌─────────────────────┐
                    │  isWailsApp()?      │
                    └──────┬──────┬───────┘
                          YES    NO
                           │      │
                ┌──────────┘      └──────────┐
                ↓                            ↓
    ┌──────────────────────┐    ┌──────────────────────┐
    │  Desktop Mode        │    │  Web Mode            │
    ├──────────────────────┤    ├──────────────────────┤
    │ • Use Wails calls    │    │ • Use HTTP fetch()   │
    │ • window.go.main.App │    │ • POST /api/auth/*   │
    │ • Returns Promise    │    │ • Returns Promise    │
    └──────────┬───────────┘    └──────────┬───────────┘
               │                           │
               └───────────┬───────────────┘
                           │
                           ↓
                    ┌─────────────────────┐
                    │  Unified Response   │
                    │  Same interface!    │
                    └─────────────────────┘
```

## Security Architecture

### Desktop Mode Security

```
┌─────────────────────┐
│  Wails App          │
├─────────────────────┤
│ • Sandboxed runtime │
│ • Direct Go calls   │
│ • No HTTP exposure  │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  OS Keychain        │  ← OAuth tokens stored here
├─────────────────────┤
│ • Encrypted         │
│ • OS-level security │
│ • Per-user isolated │
└─────────────────────┘
```

### Web Mode Security

```
┌─────────────────────┐
│  Browser            │
├─────────────────────┤
│ • HTTPS required    │
│ • CORS protection   │
│ • CSP headers       │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  Security Layers    │
├─────────────────────┤
│ 1. OAuth State      │  ← CSRF protection
│ 2. Token storage    │  ← localStorage encrypted
│ 3. WebAuthn origin  │  ← Origin validation
│ 4. HTTPS transport  │  ← TLS encryption
└─────────────────────┘
```

## Platform Detection Logic

```typescript
// Simple detection
export function isWailsApp(): boolean {
  return typeof window !== 'undefined' && !!window.go?.main?.App
}

// Why it works:
// - Desktop: Wails injects window.go.main.App
// - Web: window.go is undefined
// - SSR: window is undefined (returns false)
```

## Key Architectural Principles

1. **Single Responsibility**: Each module has one clear purpose
   - `platform.ts` - Detect environment
   - `auth-api.ts` - Route API calls
   - `auth-store.ts` - Manage state

2. **Abstraction**: Components don't know about modes
   - UI components → Auth store → Auth API → Platform-specific implementation
   - Clean separation of concerns

3. **Type Safety**: Full TypeScript coverage
   - Interfaces for all responses
   - Proper error types
   - Mode-specific types where needed

4. **Error Handling**: Graceful degradation
   - Desktop: Falls back to manual login
   - Web: Shows error UI with retry
   - Both: Clear error messages

5. **Security First**:
   - CSRF protection in web mode
   - Secure token storage in both modes
   - WebAuthn challenge/response
   - Origin validation

## Integration Points

### With Backend

**Desktop Mode:**
- Wails bindings in Go backend
- OS keychain integration
- Deep link URL handlers

**Web Mode:**
- HTTP REST API endpoints
- Session management
- OAuth callback endpoints

### With OAuth Providers

**Desktop:**
- Custom URL scheme: `howlerops://callback`
- Registered as OS protocol handler

**Web:**
- Standard HTTP callback: `https://app.howlerops.com/auth/callback`
- Configured in provider dashboard

### With Browser APIs

**Both Modes:**
- WebAuthn API (`navigator.credentials`)
- SessionStorage (web CSRF protection)
- LocalStorage (token persistence)
