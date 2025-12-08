# Wails Authentication Research: Comprehensive Guide

## Executive Summary

Wails applications can implement multiple authentication methods, but each has specific considerations due to the embedded WebView architecture. This guide covers browser-based OAuth, biometric authentication, WebAuthn/FIDO2, platform-specific patterns, and production-ready libraries.

---

## 1. Browser-Based OAuth

### Status & Viability: **Production-Ready**

Wails applications can implement OAuth 2.0 with Google OAuth, GitHub OAuth, and other OAuth providers. The approach differs from traditional web apps due to the desktop context and security model.

### Architecture Overview

The typical OAuth flow in Wails involves:

1. **Initiate OAuth flow** - Frontend calls Go backend
2. **Open browser window** - Backend opens the user's default browser to the OAuth provider
3. **Handle redirect** - App registers a custom protocol handler to catch the OAuth callback
4. **Token exchange** - Backend completes the token exchange and stores the token securely

### Best Practices & Patterns

#### Pattern 1: Custom Protocol Handler (Recommended)

```go
// Go Backend (main.go)
type App struct {
    ctx context.Context
}

func (a *App) OnUrlOpen(url string) {
    // OAuth callback URL arrives here (e.g., myapp://oauth?code=xxx&state=yyy)
    // Extract code/state, complete token exchange with backend
}

// In your wails.Run() setup:
options := &options.App{
    OnUrlOpen: app.OnUrlOpen,
}
```

```tsx
// React Frontend (AuthContext.tsx)
const initiateOAuth = async (provider: 'google' | 'github') => {
    const { authUrl, state } = await window.go.main.App.GetOAuthURL(provider);
    // Store state in sessionStorage for verification
    sessionStorage.setItem('oauth_state', state);
    // Open user's default browser
    window.open(authUrl, '_blank');
    // Your Go backend will handle the callback via custom protocol
};
```

**Advantages:**
- Uses native browser (better security, more cookies/extensions)
- Clean separation of concerns
- User sees OAuth provider's actual website

**Considerations:**
- Requires custom protocol registration (myapp://)
- Must validate state parameter to prevent CSRF
- Need to handle browser window closing

#### Pattern 2: Popup Window (Alternative)

```go
// Go Backend - serve callback endpoint
func (a *App) HandleOAuthCallback(code, state string) (TokenResponse, error) {
    // Validate state
    // Exchange code for token
    // Return token to frontend
}
```

```tsx
// React Frontend
const initiateOAuthPopup = async (provider: 'google' | 'github') => {
    const { authUrl, state } = await window.go.main.App.GetOAuthURL(provider);
    sessionStorage.setItem('oauth_state', state);

    // Open popup that redirects to localhost callback
    const popup = window.open(authUrl, 'oauth_popup', 'width=500,height=600');

    // Poll for token or use postMessage
    const checkForToken = setInterval(async () => {
        const token = await window.go.main.App.GetPendingToken();
        if (token) {
            clearInterval(checkForToken);
            popup?.close();
            setAuthToken(token);
        }
    }, 500);
};
```

**Advantages:**
- Doesn't require custom protocol registration
- Can work with localhost callback URL

**Disadvantages:**
- Popup handling less clean
- Localhost callback requires local HTTP server

### Security Best Practices

1. **Use PKCE (Proof Key for Code Exchange)**
   - Mandatory for all public clients (desktop apps without secure backend storage)
   - Prevents authorization code interception attacks

```go
// Generate PKCE parameters in Go backend
import "crypto/sha256"
import "encoding/base64"

func generatePKCE() (codeVerifier, codeChallenge string, err error) {
    // Generate random 128-character code verifier
    // Create S256 challenge from verifier
    return verifier, challenge, nil
}
```

2. **Validate state parameter** - Prevent CSRF attacks
```go
if returnedState != storedState {
    return nil, errors.New("state mismatch - potential CSRF attack")
}
```

3. **Use HTTPS only** - Even with localhost, avoid unencrypted channels
4. **Short token lifetimes** - Request short-lived access tokens
5. **Never expose client secrets** - Desktop apps should use client credential flow or PKCE, not client secrets

### Library Recommendations

- **golangci-lint/oauth2**: `golang.org/x/oauth2` (standard library)
  - Handles all OAuth 2.0 flows with PKCE support
  - Integrates easily with Wails backends

- **ory/fosite**: For implementing your own OAuth server
- **ory/hydra**: Complete OAuth/OIDC provider solution

### Code Example: Complete OAuth Flow

```go
// Go Backend
import (
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "golang.org/x/oauth2/github"
)

type AuthManager struct {
    googleConfig *oauth2.Config
    githubConfig *oauth2.Config
    pendingTokens map[string]string // Temporary token storage
}

func (a *AuthManager) GetOAuthURL(provider string) (map[string]string, error) {
    var config *oauth2.Config
    var endpoints oauth2.Endpoint

    switch provider {
    case "google":
        config = a.googleConfig
    case "github":
        config = a.githubConfig
    }

    // Generate PKCE
    codeVerifier := generateRandomString(128)
    codeChallenge := generateS256Challenge(codeVerifier)

    // Generate state
    state := generateRandomString(32)

    // Create OAuth URL
    authURL := config.AuthCodeURL(
        state,
        oauth2.SetAuthURLParam("code_challenge", codeChallenge),
        oauth2.SetAuthURLParam("code_challenge_method", "S256"),
    )

    // Store verifier and state for later
    a.stateMap[state] = codeVerifier

    return map[string]string{
        "authUrl": authURL,
        "state": state,
    }, nil
}

func (a *AuthManager) HandleOAuthCallback(code, state string) (string, error) {
    codeVerifier := a.stateMap[state]
    delete(a.stateMap, state) // One-time use

    token, err := a.googleConfig.Exchange(context.Background(), code,
        oauth2.SetAuthURLParam("code_verifier", codeVerifier),
    )
    if err != nil {
        return "", err
    }

    // Get user info
    resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
    // ... process user info

    return token.AccessToken, nil
}
```

```tsx
// React Frontend
import { createContext, useState, useCallback } from 'react';

interface AuthContextType {
    token: string | null;
    user: any | null;
    initiateOAuth: (provider: 'google' | 'github') => Promise<void>;
    logout: () => void;
}

export const AuthContext = createContext<AuthContextType>(null!);

export function AuthProvider({ children }) {
    const [token, setToken] = useState<string | null>(null);
    const [user, setUser] = useState<any>(null);

    const initiateOAuth = useCallback(async (provider: 'google' | 'github') => {
        const result = await window.go.main.AuthManager.GetOAuthURL(provider);
        const { authUrl, state } = result;

        // Store state for verification
        sessionStorage.setItem('oauth_state', state);

        // Open browser
        window.open(authUrl, '_blank');
    }, []);

    const logout = useCallback(() => {
        setToken(null);
        setUser(null);
        sessionStorage.removeItem('oauth_state');
    }, []);

    return (
        <AuthContext.Provider value={{ token, user, initiateOAuth, logout }}>
            {children}
        </AuthContext.Provider>
    );
}
```

---

## 2. Biometric Authentication

### Status & Viability: **Limited - Platform-Specific**

Wails does **not** have built-in biometric authentication support. However, biometrics can be implemented via:

1. **WebAuthn (recommended)** - Works in embedded WebView
2. **Platform-specific Go bindings** - Custom implementation per OS

### Option 1: WebAuthn/FIDO2 (Recommended)

WebAuthn is the most practical approach for cross-platform biometric support.

**Supported Platforms:**
- macOS: Touch ID, Face ID
- Windows: Windows Hello (fingerprint, face, PIN)
- Linux: Platform-specific biometric readers
- All platforms: Hardware security keys (YubiKey, etc.)

**How it works:**
```tsx
// React Frontend
const registerBiometric = async () => {
    const credential = await navigator.credentials.create({
        publicKey: {
            challenge: new Uint8Array(32),
            rp: { name: "My App" },
            user: {
                id: new Uint8Array(16),
                name: "user@example.com",
                displayName: "User Name"
            },
            pubKeyCredParams: [
                { alg: -7, type: "public-key" }, // ES256
                { alg: -257, type: "public-key" } // RS256
            ],
            authenticatorSelection: {
                authenticatorAttachment: "platform", // Built-in authenticators
                userVerification: "preferred"
            },
            timeout: 60000,
            attestation: "none"
        }
    });

    // Send credential to backend
    await window.go.main.AuthManager.RegisterBiometric(credential);
};

const authenticateWithBiometric = async () => {
    const assertion = await navigator.credentials.get({
        publicKey: {
            challenge: new Uint8Array(32),
            timeout: 60000,
            userVerification: "preferred"
        }
    });

    // Verify with backend
    const token = await window.go.main.AuthManager.VerifyBiometric(assertion);
    setToken(token);
};
```

**Go Backend:**
```go
import "github.com/duo-labs/webauthn/webauthn"
import "github.com/duo-labs/webauthn/protocol"

type BiometricManager struct {
    webauthn *webauthn.WebAuthn
}

func (bm *BiometricManager) RegisterBiometric(credential protocol.CredentialCreationResponse) error {
    // Verify credential was created with biometric
    // Store public key for future authentication
    return nil
}

func (bm *BiometricManager) VerifyBiometric(assertion protocol.AuthenticatorAssertionResponse) (string, error) {
    // Verify signature with stored public key
    // Return auth token
    return token, nil
}
```

**Library Recommendations:**
- **duo-labs/webauthn**: Pure Go implementation of WebAuthn standard
- **google/go-attestation**: Verify hardware attestation
- **web-auth/webauthn-framework**: Language-agnostic reference implementation

### Option 2: Platform-Specific Implementation (Advanced)

For applications requiring native biometric integration:

**macOS Touch ID:**
```go
import "os/exec"

func (a *App) AuthenticateWithTouchID() (bool, error) {
    cmd := exec.Command("security", "authorize", "-l")
    err := cmd.Run()
    return err == nil, err
}
```

**Windows Hello:**
```go
import "github.com/microsoft/go-winio"

func (a *App) AuthenticateWithWindowsHello() (bool, error) {
    // Use Windows Hello API via Windows API bindings
    // Requires cgo and Windows SDK
    return false, errors.New("Windows Hello integration requires Windows SDK")
}
```

**Practical Limitation:** Platform-specific biometric APIs are complex and require:
- Platform SDK installation
- CGo bindings
- Separate implementation per platform
- Complex build processes

**Recommendation:** Use WebAuthn instead - it provides cross-platform biometric support via the embedded WebView without platform-specific complexity.

---

## 3. WebAuthn/FIDO2 Support

### Status & Viability: **Production-Ready**

WebAuthn is fully supported in Wails applications because:
- Wails uses native WebView2 (Windows), WebKit (macOS), WebKitGTK (Linux)
- All modern browser engines support WebAuthn
- WebAuthn API is available in JavaScript

### Key Characteristics

**FIDO2 Architecture:**
- **WebAuthn** (W3C) - JavaScript API for browser
- **CTAP** (FIDO Alliance) - Protocol between browser and authenticator
- **Authenticators:**
  - Platform authenticators (built-in biometrics)
  - Roaming authenticators (hardware keys)

### Complete WebAuthn Implementation

```tsx
// React Frontend - Credential Registration
interface RegistrationOptions {
    challenge: BufferSource;
    rp: PublicKeyCredentialRpEntity;
    user: PublicKeyCredentialUserEntity;
    pubKeyCredParams: PublicKeyCredentialParameters[];
    timeout?: number;
    attestation?: AttestationConveyanceFormat;
    authenticatorSelection?: AuthenticatorSelectionCriteria;
    extensions?: AuthenticationExtensionsClientInputs;
}

const registerWebAuthn = async (email: string) => {
    // 1. Request registration challenge from backend
    const { challenge, userId } = await window.go.main.AuthManager.GetRegistrationChallenge(email);

    // 2. Create credential with WebAuthn
    const credential = await navigator.credentials.create({
        publicKey: {
            challenge: new Uint8Array(Buffer.from(challenge, 'base64')),
            rp: {
                name: "MyApp",
                id: "myapp.com"
            },
            user: {
                id: new Uint8Array(Buffer.from(userId, 'base64')),
                name: email,
                displayName: email
            },
            pubKeyCredParams: [
                { type: "public-key", alg: -7 },    // ES256
                { type: "public-key", alg: -257 }   // RS256
            ],
            timeout: 60000,
            attestation: "direct",
            authenticatorSelection: {
                authenticatorAttachment: "platform",
                residentKey: "preferred",
                userVerification: "preferred"
            }
        }
    });

    // 3. Send credential to backend for verification and storage
    const verified = await window.go.main.AuthManager.VerifyRegistration(
        JSON.stringify(credential)
    );

    return verified;
};

const authenticateWebAuthn = async () => {
    // 1. Request authentication challenge
    const { challenge } = await window.go.main.AuthManager.GetAuthenticationChallenge();

    // 2. Get assertion from WebAuthn
    const assertion = await navigator.credentials.get({
        publicKey: {
            challenge: new Uint8Array(Buffer.from(challenge, 'base64')),
            timeout: 60000,
            userVerification: "preferred"
        }
    });

    // 3. Verify with backend
    const token = await window.go.main.AuthManager.VerifyAssertion(
        JSON.stringify(assertion)
    );

    return token;
};
```

```go
// Go Backend - WebAuthn Integration
import (
    "github.com/duo-labs/webauthn/webauthn"
    "github.com/duo-labs/webauthn/protocol"
    "github.com/duo-labs/webauthn/protocol/webauthncose"
)

type WebAuthnManager struct {
    webauthn *webauthn.WebAuthn
}

func (wam *WebAuthnManager) GetRegistrationChallenge(email string) (map[string]string, error) {
    user := &User{
        ID: []byte(generateUserID()), // Must be unique
        Name: email,
        DisplayName: email,
    }

    options, _, err := wam.webauthn.BeginRegistration(user)
    if err != nil {
        return nil, err
    }

    // Store session data (in memory or database)
    storeSessionData(email, options.Response.Challenge)

    return map[string]string{
        "challenge": base64.StdEncoding.EncodeToString(options.Response.Challenge),
        "userId": base64.StdEncoding.EncodeToString(user.ID),
    }, nil
}

func (wam *WebAuthnManager) VerifyRegistration(credentialJSON string) (bool, error) {
    credential := &protocol.ParsedCredentialCreationData{}
    json.Unmarshal([]byte(credentialJSON), credential)

    // Validate credential with stored challenge
    validated, err := wam.webauthn.ValidateRegistration(user, sessionData, credential)
    if err != nil {
        return false, err
    }

    // Store public key in database
    storeCredential(user, validated.Credential)

    return true, nil
}

func (wam *WebAuthnManager) VerifyAssertion(assertionJSON string) (string, error) {
    assertion := &protocol.ParsedAssertionResponse{}
    json.Unmarshal([]byte(assertionJSON), assertion)

    // Get user's stored credentials
    credentials := getStoredCredentials(assertion.Response.UserHandle)

    // Validate assertion
    validated, err := wam.webauthn.ValidateAssertion(user, sessionData, assertion)
    if err != nil {
        return "", err
    }

    // Generate auth token
    return generateAuthToken(user), nil
}
```

### WebAuthn vs Traditional MFA

| Feature | WebAuthn | TOTP/SMS |
|---------|----------|---------|
| Biometric | Yes (Touch ID, Face ID, Windows Hello) | No |
| Hardware Keys | Yes | No |
| Phishing Resistant | Yes | No |
| User Experience | Excellent | Good |
| Setup Complexity | Medium | Low |
| Cost | Free | May require SMS service |

---

## 4. Platform-Specific Considerations

### Cookie Limitations (Critical)

**Issue:** Wails WebView2/WebKit have limited cookie support compared to full browsers.

**Workaround:** Use token-based authentication instead of cookie-based sessions:

```tsx
// GOOD: Token-based (works in Wails)
const setAuthToken = (token: string) => {
    sessionStorage.setItem('auth_token', token);
    window.go.main.App.StoreTokenSecurely(token);
};

// AVOID: Cookie-based (limited support)
// document.cookie = "auth_token=..."; // May not persist reliably
```

### Runtime Context Management

Store the Wails context in your Go backend:

```go
type App struct {
    ctx context.Context
}

func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx
    // Can now use runtime.EventsOn, runtime.Invoke, etc.
}

// Later, in authentication flow
func (a *App) EmitAuthenticatedEvent(user *User) {
    runtime.EventsEmit(a.ctx, "authenticated", user)
}
```

### Multi-Window Authentication

For apps with multiple windows:

```go
func (a *App) OnUrlOpen(url string) {
    // Parse OAuth callback
    code := getCodeFromURL(url)
    token, _ := a.completeOAuthFlow(code)

    // Emit to all windows
    runtime.EventsEmit(a.ctx, "oauth:complete", token)
}
```

```tsx
// React - Listen for auth events
useEffect(() => {
    const unsubscribe = window.runtime.EventsOn("oauth:complete", (token: string) => {
        setAuthToken(token);
    });

    return unsubscribe;
}, []);
```

### Secure Token Storage

**Desktop-specific approach:** Use platform keyrings via Go backend

```go
import "github.com/zalando/go-keyring"

func (a *App) StoreTokenSecurely(token string) error {
    return keyring.Set("myapp", "auth_token", token)
}

func (a *App) RetrieveTokenSecurely() (string, error) {
    return keyring.Get("myapp", "auth_token")
}

func (a *App) ClearTokenSecurely() error {
    return keyring.Delete("myapp", "auth_token")
}
```

This stores tokens in:
- **macOS**: Keychain
- **Windows**: Credential Manager
- **Linux**: GNOME Keyring / Secret Service

---

## 5. Popular Libraries & Solutions

### Go Backend Libraries

| Library | Purpose | Status | Notes |
|---------|---------|--------|-------|
| `golang.org/x/oauth2` | OAuth 2.0 client | Stable | Standard library, PKCE support |
| `github.com/duo-labs/webauthn` | WebAuthn server | Stable | Pure Go, actively maintained |
| `github.com/zalando/go-keyring` | Secure storage | Stable | Cross-platform keyring access |
| `github.com/golang-jwt/jwt` | JWT handling | Stable | Token generation and verification |
| `github.com/markbates/goth` | Multi-provider OAuth | Stable | 40+ OAuth providers built-in |

### React Frontend Libraries

| Library | Purpose | Status | Notes |
|---------|---------|--------|-------|
| `@react-oauth/google` | Google OAuth | Stable | Works with custom flow |
| `webauthn-json` | WebAuthn helper | Stable | Simplifies WebAuthn API |
| `axios` | HTTP client | Stable | For API calls with tokens |
| `zustand` or `jotai` | State management | Stable | Lightweight token storage |
| `react-query` | API caching | Stable | Handles auth headers |

### Recommended Architecture

```
Go Backend:
├── auth/
│   ├── oauth2.go         (OAuth2Manager)
│   ├── webauthn.go       (WebAuthnManager)
│   ├── secure_storage.go (KeyringManager)
│   └── jwt_handler.go    (TokenManager)
└── main.go

React Frontend:
├── contexts/
│   └── AuthContext.tsx   (Auth state + bindings)
├── hooks/
│   ├── useAuth.ts
│   ├── useOAuth.ts
│   └── useWebAuthn.ts
├── components/
│   ├── LoginForm.tsx
│   ├── BiometricButton.tsx
│   └── OAuthButton.tsx
└── types/
    └── auth.ts
```

---

## 6. Complete Production Example

### Project Structure

```
wails-auth-app/
├── frontend/
│   ├── src/
│   │   ├── contexts/AuthContext.tsx
│   │   ├── hooks/useAuth.ts
│   │   ├── pages/Login.tsx
│   │   └── App.tsx
│   └── package.json
├── backend/
│   ├── auth.go
│   ├── oauth.go
│   ├── webauthn.go
│   ├── storage.go
│   ├── main.go
│   └── go.mod
└── wails.json
```

### Go Implementation

```go
// backend/main.go
package main

import (
    "context"
    "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
    ctx context.Context
    auth *AuthManager
}

type AuthManager struct {
    oauth *OAuth2Manager
    webauthn *WebAuthnManager
    storage *SecureStorage
}

func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx
    a.auth = &AuthManager{
        oauth: NewOAuth2Manager(/* config */),
        webauthn: NewWebAuthnManager(/* config */),
        storage: NewSecureStorage(),
    }
}

func (a *App) GetOAuthURL(provider string) (map[string]interface{}, error) {
    return a.auth.oauth.GetAuthURL(provider)
}

func (a *App) HandleOAuthCallback(code, state string) (string, error) {
    token, user, err := a.auth.oauth.ExchangeCode(code, state)
    if err != nil {
        return "", err
    }

    // Store token securely
    a.auth.storage.StoreToken(token)

    // Emit authenticated event
    runtime.EventsEmit(a.ctx, "authenticated", map[string]interface{}{
        "user": user,
        "token": token,
    })

    return token, nil
}

func (a *App) GetWebAuthnChallenge() (map[string]string, error) {
    return a.auth.webauthn.GetRegistrationChallenge()
}

func (a *App) VerifyWebAuthnRegistration(credential string) (bool, error) {
    return a.auth.webauthn.VerifyRegistration(credential)
}

func (a *App) OnUrlOpen(url string) {
    // Handle custom protocol redirects (myapp://oauth?code=xxx)
    // Extract code and call HandleOAuthCallback
}

func main() {
    app := &App{}

    err := wails.Run(&options.App{
        Title: "Wails Auth Demo",
        OnStartup: app.OnStartup,
        OnUrlOpen: app.OnUrlOpen,
        // ... other options
    })

    if err != nil {
        log.Fatal(err)
    }
}
```

### React Implementation

```tsx
// frontend/src/contexts/AuthContext.tsx
import React, { createContext, useCallback, useEffect, useState } from 'react';

interface AuthContextType {
    isAuthenticated: boolean;
    user: any | null;
    token: string | null;
    initiateOAuth: (provider: string) => Promise<void>;
    registerBiometric: () => Promise<void>;
    authenticateBiometric: () => Promise<void>;
    logout: () => void;
}

export const AuthContext = createContext<AuthContextType>(null!);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [user, setUser] = useState<any>(null);
    const [token, setToken] = useState<string | null>(null);

    useEffect(() => {
        // Restore auth state on startup
        window.go.main.App.GetStoredToken()
            .then(token => {
                if (token) {
                    setToken(token);
                    setIsAuthenticated(true);
                }
            });

        // Listen for auth events
        window.runtime.EventsOn('authenticated', (data: any) => {
            setUser(data.user);
            setToken(data.token);
            setIsAuthenticated(true);
        });
    }, []);

    const initiateOAuth = useCallback(async (provider: string) => {
        const { authUrl, state } = await window.go.main.App.GetOAuthURL(provider);
        sessionStorage.setItem('oauth_state', state);
        window.open(authUrl, '_blank');
    }, []);

    const registerBiometric = useCallback(async () => {
        const challenge = await window.go.main.App.GetWebAuthnChallenge();
        const credential = await navigator.credentials.create({
            publicKey: {
                challenge: new Uint8Array(Buffer.from(challenge.challenge, 'base64')),
                // ... full config
            }
        });
        await window.go.main.App.VerifyWebAuthnRegistration(JSON.stringify(credential));
    }, []);

    const authenticateBiometric = useCallback(async () => {
        const challenge = await window.go.main.App.GetWebAuthnChallenge();
        const assertion = await navigator.credentials.get({
            publicKey: {
                challenge: new Uint8Array(Buffer.from(challenge.challenge, 'base64')),
            }
        });
        const token = await window.go.main.App.VerifyWebAuthnAssertion(JSON.stringify(assertion));
        setToken(token);
        setIsAuthenticated(true);
    }, []);

    const logout = useCallback(async () => {
        await window.go.main.App.ClearTokenSecurely();
        setToken(null);
        setUser(null);
        setIsAuthenticated(false);
    }, []);

    return (
        <AuthContext.Provider value={{
            isAuthenticated,
            user,
            token,
            initiateOAuth,
            registerBiometric,
            authenticateBiometric,
            logout
        }}>
            {children}
        </AuthContext.Provider>
    );
}
```

---

## Summary & Recommendations

### For New Wails Projects

**Recommended Auth Stack:**

1. **Primary:** OAuth 2.0 with PKCE + WebAuthn
   - Handles password-less authentication
   - Biometric support cross-platform
   - Industry standard security

2. **Token Storage:** Platform keyring (zalando/go-keyring)
   - Secure, OS-native storage
   - Works on macOS, Windows, Linux

3. **For API Protection:** JWT tokens
   - Store in sessionStorage (frontend)
   - Include in Authorization headers
   - Short expiration times

### Avoid

- Cookie-based sessions (limited WebView support)
- Storing tokens in localStorage (less secure than OS keyring)
- Custom authentication schemes (use OAuth2/OIDC instead)
- Storing credentials in plaintext

### Implementation Timeline

**Phase 1 (Week 1):**
- Implement basic OAuth2 with PKCE
- Add token storage in OS keyring
- Test with single provider (GitHub)

**Phase 2 (Week 2):**
- Add WebAuthn registration/authentication
- Implement refresh token rotation
- Add logout with token cleanup

**Phase 3 (Week 3):**
- Add additional OAuth providers (Google, Microsoft)
- Implement account linking
- Add biometric recovery codes

---

## References & Resources

- **Wails Documentation:** https://wails.io/
- **OAuth 2.0 Security Best Practices:** https://oauth.net/2/oauth-best-practice/
- **WebAuthn Specification:** https://www.w3.org/TR/webauthn-2/
- **FIDO2 Standards:** https://fidoalliance.org/fido2/
- **Go OAuth2 Package:** https://pkg.go.dev/golang.org/x/oauth2
- **Duo WebAuthn:** https://github.com/duo-labs/webauthn
- **Zalando go-keyring:** https://github.com/zalando/go-keyring
