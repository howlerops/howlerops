# Wails Authentication: Quick Start Implementation Guide

This document contains copy-paste-ready code examples for implementing authentication in Wails applications.

---

## Quick Start: OAuth2 with GitHub

### Step 1: Go Backend Setup

```go
// backend/auth.go
package main

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "math/rand"
    "net/http"
    "strings"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/github"
    "github.com/zalando/go-keyring"
)

type OAuth2Manager struct {
    githubConfig *oauth2.Config
    pendingTokens map[string]pendingToken
}

type pendingToken struct {
    AccessToken string
    State       string
    ExpiresAt   int64
}

type GitHubUser struct {
    ID    int    `json:"id"`
    Login string `json:"login"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

func NewOAuth2Manager(clientID, clientSecret string) *OAuth2Manager {
    return &OAuth2Manager{
        githubConfig: &oauth2.Config{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            RedirectURL:  "myapp://oauth/callback",
            Scopes:       []string{"user:email"},
            Endpoint:     github.Endpoint,
        },
        pendingTokens: make(map[string]pendingToken),
    }
}

// Generate PKCE parameters
func generatePKCEPair() (verifier, challenge string, err error) {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
    verifier = ""
    for i := 0; i < 128; i++ {
        verifier += string(chars[rand.Intn(len(chars))])
    }

    h := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(h[:])
    return
}

// Generate random state
func generateRandomState() string {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    state := ""
    for i := 0; i < 32; i++ {
        state += string(chars[rand.Intn(len(chars))])
    }
    return state
}

func (om *OAuth2Manager) GetAuthURL() (map[string]string, error) {
    verifier, challenge, err := generatePKCEPair()
    if err != nil {
        return nil, err
    }

    state := generateRandomState()

    // Store PKCE verifier for later
    om.pendingTokens[state] = pendingToken{
        State: state,
    }

    authURL := om.githubConfig.AuthCodeURL(state,
        oauth2.SetAuthURLParam("code_challenge", challenge),
        oauth2.SetAuthURLParam("code_challenge_method", "S256"),
    )

    return map[string]string{
        "authUrl": authURL,
        "state":   state,
    }, nil
}

func (om *OAuth2Manager) ExchangeCodeForToken(code, state string) (string, *GitHubUser, error) {
    // Verify state
    pending, exists := om.pendingTokens[state]
    if !exists {
        return "", nil, errors.New("invalid state parameter")
    }
    defer delete(om.pendingTokens, state)

    // Exchange code for token
    token, err := om.githubConfig.Exchange(context.Background(), code)
    if err != nil {
        return "", nil, fmt.Errorf("failed to exchange code: %w", err)
    }

    // Get user info
    client := om.githubConfig.Client(context.Background(), token)
    resp, err := client.Get("https://api.github.com/user")
    if err != nil {
        return "", nil, fmt.Errorf("failed to get user info: %w", err)
    }
    defer resp.Body.Close()

    var user GitHubUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return "", nil, fmt.Errorf("failed to decode user: %w", err)
    }

    return token.AccessToken, &user, nil
}

// Secure token storage
type SecureStorage struct{}

func (ss *SecureStorage) StoreToken(token string) error {
    return keyring.Set("myapp", "github_token", token)
}

func (ss *SecureStorage) RetrieveToken() (string, error) {
    return keyring.Get("myapp", "github_token")
}

func (ss *SecureStorage) DeleteToken() error {
    return keyring.Delete("myapp", "github_token")
}
```

```go
// backend/main.go
package main

import (
    "context"
    "log"

    "github.com/wailsapp/wails/v2/pkg/runtime"
    "github.com/wailsapp/wails/v2"
    "github.com/wailsapp/wails/v2/pkg/options"
)

type App struct {
    ctx     context.Context
    auth    *OAuth2Manager
    storage *SecureStorage
}

func NewApp() *App {
    return &App{
        auth:    NewOAuth2Manager(os.Getenv("GH_CLIENT_ID"), os.Getenv("GH_CLIENT_SECRET")),
        storage: &SecureStorage{},
    }
}

func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx

    // Restore stored token if available
    if token, err := a.storage.RetrieveToken(); err == nil && token != "" {
        runtime.EventsEmit(ctx, "auth:restored", map[string]string{
            "token": token,
        })
    }
}

func (a *App) OnUrlOpen(url string) {
    // Handle myapp://oauth/callback?code=xxx&state=yyy
    if !strings.Contains(url, "oauth/callback") {
        return
    }

    query := url[strings.Index(url, "?")+1:]
    params := parseQuery(query)

    code, ok := params["code"]
    if !ok {
        runtime.EventsEmit(a.ctx, "auth:error", "no authorization code")
        return
    }

    state, ok := params["state"]
    if !ok {
        runtime.EventsEmit(a.ctx, "auth:error", "no state parameter")
        return
    }

    token, user, err := a.auth.ExchangeCodeForToken(code, state)
    if err != nil {
        runtime.EventsEmit(a.ctx, "auth:error", err.Error())
        return
    }

    // Store token securely
    if err := a.storage.StoreToken(token); err != nil {
        runtime.EventsEmit(a.ctx, "auth:error", "failed to store token")
        return
    }

    runtime.EventsEmit(a.ctx, "auth:success", map[string]interface{}{
        "token": token,
        "user": user,
    })
}

func (a *App) GetAuthURL() (map[string]string, error) {
    return a.auth.GetAuthURL()
}

func (a *App) Logout() error {
    return a.storage.DeleteToken()
}

func main() {
    app := NewApp()

    err := wails.Run(&options.App{
        Title:      "Wails Auth Demo",
        Width:      1024,
        Height:     768,
        OnStartup:  app.OnStartup,
        OnUrlOpen:  app.OnUrlOpen,
        Bind: []interface{}{
            app,
        },
    })

    if err != nil {
        log.Fatal(err)
    }
}

// Helper function to parse query string
func parseQuery(query string) map[string]string {
    params := make(map[string]string)
    for _, pair := range strings.Split(query, "&") {
        parts := strings.Split(pair, "=")
        if len(parts) == 2 {
            params[parts[0]] = parts[1]
        }
    }
    return params
}
```

### Step 2: React Frontend

```tsx
// frontend/src/contexts/AuthContext.tsx
import React, { createContext, useCallback, useEffect, useState } from 'react';

interface AuthContextType {
    isAuthenticated: boolean;
    user: any | null;
    token: string | null;
    loading: boolean;
    error: string | null;
    initiateLogin: () => Promise<void>;
    logout: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextType>(null!);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [user, setUser] = useState<any>(null);
    const [token, setToken] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        // Listen for auth restored event (from OnStartup)
        const unsubscribeRestored = window.runtime.EventsOn('auth:restored', (data: any) => {
            setToken(data.token);
            setIsAuthenticated(true);
            setLoading(false);
        });

        // Listen for successful OAuth callback
        const unsubscribeSuccess = window.runtime.EventsOn('auth:success', (data: any) => {
            setUser(data.user);
            setToken(data.token);
            setIsAuthenticated(true);
            setError(null);
        });

        // Listen for auth errors
        const unsubscribeError = window.runtime.EventsOn('auth:error', (message: string) => {
            setError(message);
            setIsAuthenticated(false);
        });

        // Check if auth was restored, or no stored token available
        const checkAuth = async () => {
            // Wait a bit for events to fire
            setTimeout(() => {
                if (!isAuthenticated) {
                    setLoading(false);
                }
            }, 100);
        };

        checkAuth();

        return () => {
            unsubscribeRestored();
            unsubscribeSuccess();
            unsubscribeError();
        };
    }, []);

    const initiateLogin = useCallback(async () => {
        setLoading(true);
        setError(null);

        try {
            const { authUrl } = await window.go.main.App.GetAuthURL();
            // Open browser - the callback will be handled by OnUrlOpen
            window.open(authUrl, '_blank');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Login failed');
            setLoading(false);
        }
    }, []);

    const logout = useCallback(async () => {
        setLoading(true);
        try {
            await window.go.main.App.Logout();
            setToken(null);
            setUser(null);
            setIsAuthenticated(false);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Logout failed');
        } finally {
            setLoading(false);
        }
    }, []);

    return (
        <AuthContext.Provider value={{
            isAuthenticated,
            user,
            token,
            loading,
            error,
            initiateLogin,
            logout,
        }}>
            {children}
        </AuthContext.Provider>
    );
}

// Custom hook for using auth
export function useAuth() {
    const context = React.useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within AuthProvider');
    }
    return context;
}
```

```tsx
// frontend/src/pages/Login.tsx
import React from 'react';
import { useAuth } from '../contexts/AuthContext';

export function LoginPage() {
    const { isAuthenticated, loading, error, initiateLogin, logout, user } = useAuth();

    if (loading) {
        return <div>Loading authentication...</div>;
    }

    if (isAuthenticated && user) {
        return (
            <div style={{ padding: '20px' }}>
                <h1>Welcome, {user.name || user.login}!</h1>
                <p>Email: {user.email}</p>
                <button onClick={logout} style={{ padding: '10px 20px', fontSize: '16px' }}>
                    Logout
                </button>
            </div>
        );
    }

    return (
        <div style={{ padding: '20px', textAlign: 'center' }}>
            <h1>Sign In</h1>
            {error && <div style={{ color: 'red', marginBottom: '20px' }}>{error}</div>}
            <button
                onClick={initiateLogin}
                style={{
                    padding: '10px 20px',
                    fontSize: '16px',
                    cursor: 'pointer',
                    backgroundColor: '#24292e',
                    color: 'white',
                    border: 'none',
                    borderRadius: '6px',
                }}
            >
                Sign in with GitHub
            </button>
        </div>
    );
}
```

### Step 3: Environment Setup

Create a `.env` file with OAuth credentials:

```bash
# .env
GH_CLIENT_ID=your_GH_CLIENT_id
GH_CLIENT_SECRET=your_GH_CLIENT_secret
```

Load in main.go:

```go
import "github.com/joho/godotenv"

func init() {
    godotenv.Load()
}
```

### Step 4: Register Custom Protocol Handler

**macOS (Info.plist):**
```xml
<key>CFBundleURLTypes</key>
<array>
    <dict>
        <key>CFBundleURLSchemes</key>
        <array>
            <string>myapp</string>
        </array>
    </dict>
</array>
```

**Windows (Wails will handle automatically if you use custom protocol in config)**

**Linux (Wails will handle automatically)**

---

## Quick Start: WebAuthn (Biometric Authentication)

### Go Backend Implementation

```go
// backend/webauthn.go
package main

import (
    "encoding/base64"
    "encoding/json"

    "github.com/duo-labs/webauthn/webauthn"
    "github.com/duo-labs/webauthn/protocol"
)

type User struct {
    ID          []byte
    Name        string
    DisplayName string
}

type WebAuthnManager struct {
    webauthn *webauthn.WebAuthn
}

func NewWebAuthnManager() *WebAuthnManager {
    wam, err := webauthn.New(&webauthn.Config{
        RPID:     "localhost",
        RPOrigin: "wails://localhost",
        RPName:   "My App",
    })
    if err != nil {
        panic(err)
    }

    return &WebAuthnManager{webauthn: wam}
}

func (wam *WebAuthnManager) GetRegistrationChallenge(userID, userName string) (map[string]string, error) {
    user := &User{
        ID:          []byte(userID),
        Name:        userName,
        DisplayName: userName,
    }

    options, session, err := wam.webauthn.BeginRegistration(user)
    if err != nil {
        return nil, err
    }

    // Store session data (in real app, use database)
    sessionStore[userName] = session

    challengeJSON, _ := json.Marshal(options)
    return map[string]string{
        "options": string(challengeJSON),
    }, nil
}

func (wam *WebAuthnManager) VerifyRegistration(userID, userName, credentialJSON string) (bool, error) {
    user := &User{
        ID:          []byte(userID),
        Name:        userName,
        DisplayName: userName,
    }

    var credential *protocol.ParsedCredentialCreationData
    json.Unmarshal([]byte(credentialJSON), &credential)

    session := sessionStore[userName]

    credential, err := wam.webauthn.ParseCredentialCreationResponse(credentialJSON)
    if err != nil {
        return false, err
    }

    _, err = wam.webauthn.ValidateRegistration(user, session, credential)
    if err != nil {
        return false, err
    }

    // Store credential public key (in real app, use database)
    credentialStore[userName] = credential.Credential

    return true, nil
}

func (wam *WebAuthnManager) GetAuthenticationChallenge() (map[string]string, error) {
    options, session, err := wam.webauthn.BeginLogin()
    if err != nil {
        return nil, err
    }

    sessionStore["auth"] = session

    challengeJSON, _ := json.Marshal(options)
    return map[string]string{
        "options": string(challengeJSON),
    }, nil
}

func (wam *WebAuthnManager) VerifyAuthentication(assertionJSON string) (string, error) {
    user := &User{} // In real app, get actual user

    var credential *protocol.ParsedAssertionResponse
    json.Unmarshal([]byte(assertionJSON), &credential)

    session := sessionStore["auth"]

    _, err := wam.webauthn.ValidateAssertion(user, session, credential)
    if err != nil {
        return "", err
    }

    // Generate auth token
    return "auth_token_here", nil
}

// Temporary session/credential stores (use database in production)
var sessionStore = make(map[string]*webauthn.SessionData)
var credentialStore = make(map[string]*protocol.Credential)
```

### React Frontend Implementation

```tsx
// frontend/src/hooks/useWebAuthn.ts
import { useCallback } from 'react';

export function useWebAuthn() {
    const register = useCallback(async (userID: string, userName: string) => {
        // Get registration options from backend
        const { options } = await window.go.main.App.GetWebAuthnRegistrationOptions(userID, userName);
        const parsedOptions = JSON.parse(options);

        // Create credential
        const credential = await navigator.credentials.create({
            publicKey: {
                challenge: base64ToArrayBuffer(parsedOptions.challenge),
                rp: {
                    name: "My App",
                    id: "localhost",
                },
                user: {
                    id: base64ToArrayBuffer(parsedOptions.user.id),
                    name: parsedOptions.user.name,
                    displayName: parsedOptions.user.displayName,
                },
                pubKeyCredParams: parsedOptions.pubKeyCredParams,
                timeout: 60000,
                attestation: "none",
                authenticatorSelection: {
                    authenticatorAttachment: "platform",
                    userVerification: "preferred",
                },
            },
        });

        // Send to backend for verification
        const verified = await window.go.main.App.VerifyWebAuthnRegistration(
            userID,
            userName,
            JSON.stringify(credential)
        );

        return verified;
    }, []);

    const authenticate = useCallback(async () => {
        // Get authentication options
        const { options } = await window.go.main.App.GetWebAuthnAuthenticationOptions();
        const parsedOptions = JSON.parse(options);

        // Get assertion
        const assertion = await navigator.credentials.get({
            publicKey: {
                challenge: base64ToArrayBuffer(parsedOptions.challenge),
                timeout: 60000,
                userVerification: "preferred",
            },
        });

        // Verify with backend
        const token = await window.go.main.App.VerifyWebAuthnAuthentication(
            JSON.stringify(assertion)
        );

        return token;
    }, []);

    return { register, authenticate };
}

// Helper function
function base64ToArrayBuffer(base64: string): ArrayBuffer {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes.buffer;
}
```

```tsx
// frontend/src/components/BiometricSetup.tsx
import React, { useState } from 'react';
import { useWebAuthn } from '../hooks/useWebAuthn';

export function BiometricSetup({ userID, userName }: { userID: string; userName: string }) {
    const { register } = useWebAuthn();
    const [status, setStatus] = useState<'idle' | 'registering' | 'success' | 'error'>('idle');
    const [error, setError] = useState<string | null>(null);

    const handleSetup = async () => {
        setStatus('registering');
        setError(null);

        try {
            const success = await register(userID, userName);
            if (success) {
                setStatus('success');
            } else {
                setError('Registration failed');
                setStatus('error');
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unknown error');
            setStatus('error');
        }
    };

    return (
        <div style={{ padding: '20px' }}>
            <h2>Set Up Biometric Authentication</h2>
            {status === 'success' && <p style={{ color: 'green' }}>Setup successful!</p>}
            {error && <p style={{ color: 'red' }}>{error}</p>}
            <button
                onClick={handleSetup}
                disabled={status === 'registering'}
                style={{
                    padding: '10px 20px',
                    cursor: status === 'registering' ? 'not-allowed' : 'pointer',
                    opacity: status === 'registering' ? 0.6 : 1,
                }}
            >
                {status === 'registering' ? 'Setting up...' : 'Set Up Biometric'}
            </button>
        </div>
    );
}
```

---

## Token Management with API Calls

```tsx
// frontend/src/hooks/useApi.ts
import { useAuth } from '../contexts/AuthContext';
import { useCallback } from 'react';

export function useApi() {
    const { token } = useAuth();

    const request = useCallback(async (url: string, options?: RequestInit) => {
        const headers = {
            'Content-Type': 'application/json',
            ...options?.headers,
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const response = await fetch(url, {
            ...options,
            headers,
        });

        if (response.status === 401) {
            // Token expired, redirect to login
            window.location.href = '/login';
            return;
        }

        return response;
    }, [token]);

    return { request };
}

// Usage
export function UserProfile() {
    const { request } = useApi();
    const [profile, setProfile] = React.useState(null);

    React.useEffect(() => {
        request('/api/profile')
            .then(r => r.json())
            .then(setProfile);
    }, []);

    return <div>{profile?.name}</div>;
}
```

---

## Dependency Installation

```bash
# Go dependencies
go get golang.org/x/oauth2
go get github.com/zalando/go-keyring
go get github.com/duo-labs/webauthn
go get github.com/joho/godotenv

# React dependencies
npm install react react-dom typescript
# No special auth libraries needed - use native APIs!
```

---

## Next Steps

1. **Implement OAuth first** - Simplest, industry-standard approach
2. **Add WebAuthn** - Provides biometric authentication cross-platform
3. **Secure token storage** - Use OS keyring via Go backend
4. **Add refresh tokens** - Implement token rotation for long-lived sessions
5. **Multi-provider OAuth** - Add Google, Microsoft after GitHub works
