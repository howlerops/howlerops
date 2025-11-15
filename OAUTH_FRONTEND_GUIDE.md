# OAuth Frontend Integration Guide

Quick reference for integrating OAuth authentication in the HowlerOps React frontend.

## Backend API

### Methods

```typescript
// Get OAuth authorization URL
window.go.main.App.GetOAuthURL(provider: string): Promise<{authUrl: string, state: string}>

// Check if token exists (without retrieving it)
window.go.main.App.CheckStoredToken(provider: string): Promise<boolean>

// Get stored user info (without access token)
window.go.main.App.GetStoredUserInfo(provider: string): Promise<{
  provider: string,
  userId: string,
  email: string,
  expiresAt?: string
} | null>

// Logout (delete stored token)
window.go.main.App.Logout(provider: string): Promise<void>
```

### Events

```typescript
// Success event - fired when OAuth callback completes
'auth:success': {
  provider: string,    // 'github' or 'google'
  id: string,
  login: string,
  email: string,
  name: string,
  avatarUrl?: string
}

// Error event - fired when OAuth fails
'auth:error': string  // Error message
```

## React Implementation

### Auth Context (Recommended)

```tsx
// src/contexts/AuthContext.tsx
import React, { createContext, useContext, useEffect, useState } from 'react';

interface User {
  provider: string;
  id: string;
  login: string;
  email: string;
  name: string;
  avatarUrl?: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  error: string | null;
  login: (provider: 'github' | 'google') => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Check for stored token on mount
    checkStoredAuth();

    // Listen for OAuth success
    const unsubSuccess = window.runtime.EventsOn('auth:success', (userData: User) => {
      setUser(userData);
      setIsAuthenticated(true);
      setError(null);
      setLoading(false);
    });

    // Listen for OAuth errors
    const unsubError = window.runtime.EventsOn('auth:error', (message: string) => {
      setError(message);
      setLoading(false);
    });

    return () => {
      unsubSuccess();
      unsubError();
    };
  }, []);

  const checkStoredAuth = async () => {
    try {
      // Check both providers
      const hasGithub = await window.go.main.App.CheckStoredToken('github');
      const hasGoogle = await window.go.main.App.CheckStoredToken('google');

      if (hasGithub) {
        const info = await window.go.main.App.GetStoredUserInfo('github');
        if (info) {
          setUser(info as User);
          setIsAuthenticated(true);
        }
      } else if (hasGoogle) {
        const info = await window.go.main.App.GetStoredUserInfo('google');
        if (info) {
          setUser(info as User);
          setIsAuthenticated(true);
        }
      }
    } catch (err) {
      console.error('Failed to check stored auth:', err);
    } finally {
      setLoading(false);
    }
  };

  const login = async (provider: 'github' | 'google') => {
    try {
      setLoading(true);
      setError(null);

      const { authUrl } = await window.go.main.App.GetOAuthURL(provider);

      // Open browser for OAuth flow
      window.open(authUrl, '_blank');

      // Loading will be stopped by event listeners
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
      setLoading(false);
    }
  };

  const logout = async () => {
    if (!user) return;

    try {
      setLoading(true);
      await window.go.main.App.Logout(user.provider);
      setUser(null);
      setIsAuthenticated(false);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Logout failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated,
      loading,
      error,
      login,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
```

### Login Component

```tsx
// src/components/Login.tsx
import React from 'react';
import { useAuth } from '../contexts/AuthContext';

export function Login() {
  const { user, isAuthenticated, loading, error, login, logout } = useAuth();

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  if (isAuthenticated && user) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="bg-white shadow-md rounded-lg p-8 max-w-md">
          <div className="flex items-center space-x-4 mb-6">
            {user.avatarUrl && (
              <img
                src={user.avatarUrl}
                alt={user.name}
                className="w-16 h-16 rounded-full"
              />
            )}
            <div>
              <h2 className="text-2xl font-bold">{user.name}</h2>
              <p className="text-gray-600">{user.email}</p>
              <p className="text-sm text-gray-500">
                Signed in with {user.provider}
              </p>
            </div>
          </div>
          <button
            onClick={logout}
            className="w-full bg-red-500 hover:bg-red-600 text-white font-semibold py-2 px-4 rounded transition-colors"
          >
            Logout
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <div className="bg-white shadow-md rounded-lg p-8 max-w-md w-full">
        <h1 className="text-3xl font-bold text-center mb-8">Sign In</h1>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-6">
            {error}
          </div>
        )}

        <div className="space-y-4">
          <button
            onClick={() => login('github')}
            disabled={loading}
            className="w-full bg-gray-800 hover:bg-gray-900 text-white font-semibold py-3 px-4 rounded flex items-center justify-center space-x-2 transition-colors disabled:opacity-50"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z" clipRule="evenodd" />
            </svg>
            <span>Sign in with GitHub</span>
          </button>

          <button
            onClick={() => login('google')}
            disabled={loading}
            className="w-full bg-white hover:bg-gray-50 text-gray-700 font-semibold py-3 px-4 rounded border border-gray-300 flex items-center justify-center space-x-2 transition-colors disabled:opacity-50"
          >
            <svg className="w-5 h-5" viewBox="0 0 24 24">
              <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
              <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
              <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
              <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
            </svg>
            <span>Sign in with Google</span>
          </button>
        </div>
      </div>
    </div>
  );
}
```

### App Setup

```tsx
// src/App.tsx
import { AuthProvider } from './contexts/AuthContext';
import { Login } from './components/Login';

function App() {
  return (
    <AuthProvider>
      <Login />
    </AuthProvider>
  );
}

export default App;
```

## TypeScript Types

```typescript
// src/types/auth.ts
export interface OAuthUser {
  provider: 'github' | 'google';
  id: string;
  login: string;
  email: string;
  name: string;
  avatarUrl?: string;
}

export interface StoredUserInfo {
  provider: string;
  userId: string;
  email: string;
  expiresAt?: string;
}
```

## Best Practices

1. **Always use event listeners** - The OAuth callback happens asynchronously via custom protocol handler
2. **Check stored tokens on mount** - Users may already be authenticated
3. **Handle errors gracefully** - Network issues, user cancellation, etc.
4. **Show loading states** - OAuth flow takes several seconds
5. **Don't store tokens in frontend** - Backend handles all token storage securely
6. **Use HTTPS in production** - OAuth providers require secure origins

## Common Issues

### OAuth window doesn't close automatically
- This is expected behavior - user manually closes browser tab after authorization
- Alternative: Use `window.focus()` to bring app to foreground after auth success

### "Invalid state parameter" error
- State expires after 10 minutes - user must complete OAuth flow quickly
- Restart the flow if this occurs

### Token not persisting after app restart
- Check that OS keyring is accessible
- macOS: Keychain must be unlocked
- Windows: Credential Manager must be enabled
- Linux: GNOME Keyring or similar must be installed

## Testing

```typescript
// Test login flow
await login('github');
// Browser opens, user authorizes
// Event 'auth:success' fires
// User state updates automatically

// Test stored token
const hasToken = await window.go.main.App.CheckStoredToken('github');
console.log('Has token:', hasToken);

// Test logout
await logout();
const stillHasToken = await window.go.main.App.CheckStoredToken('github');
console.log('Still has token:', stillHasToken); // Should be false
```

## Environment Setup

Before testing, ensure OAuth credentials are configured:

```bash
# .env file in project root
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
```

See `OAUTH_IMPLEMENTATION.md` for detailed OAuth app setup instructions.
