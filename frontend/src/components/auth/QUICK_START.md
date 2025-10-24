# Authentication System - Quick Start

## Installation Complete ✅

The authentication system is already installed and integrated!

## Quick Reference

### 1. Use Auth in Components

```tsx
import { useAuthStore } from '@/store/auth-store'

function MyComponent() {
  const { isAuthenticated, user, signIn, signOut } = useAuthStore()

  return (
    <div>
      {isAuthenticated ? (
        <div>
          Welcome {user?.username}!
          <button onClick={signOut}>Sign Out</button>
        </div>
      ) : (
        <div>Please sign in</div>
      )}
    </div>
  )
}
```

### 2. Make Authenticated Requests

```tsx
import { authFetch } from '@/lib/api/auth-client'

// Automatic token injection and refresh
const data = await authFetch('/api/endpoint', {
  method: 'POST',
  body: JSON.stringify({ foo: 'bar' })
})
```

### 3. Protect Routes

```tsx
import { ProtectedRoute } from '@/components/auth'

<Route
  path="/private"
  element={
    <ProtectedRoute>
      <PrivatePage />
    </ProtectedRoute>
  }
/>
```

### 4. Use Auth Button

```tsx
import { AuthButton } from '@/components/auth'

// Already added to header!
<header>
  <AuthButton />
</header>
```

## Environment Setup

Make sure `.env` has:

```env
VITE_API_URL=http://localhost:8080
```

## Backend Requirements

Your Go backend needs these endpoints:

- `POST /api/auth/signup` - Create account
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `POST /api/auth/refresh` - Refresh token

## Testing

1. Start backend: `go run cmd/server/main.go` (on port 8080)
2. Start frontend: `npm run dev`
3. Open browser and click "Sign In" button
4. Create account or login

## Store State

```tsx
{
  user: {
    id: string
    username: string
    email: string
    role: string
    created_at: string
  } | null
  tokens: {
    access_token: string
    refresh_token: string
    expires_at: string
  } | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
}
```

## Store Actions

```tsx
const {
  // Auth actions
  signUp,      // (username, email, password) => Promise<void>
  signIn,      // (username, password) => Promise<void>
  signOut,     // () => Promise<void>
  refreshToken, // () => Promise<boolean>
  clearError,  // () => void

  // State
  user,
  tokens,
  isAuthenticated,
  isLoading,
  error
} = useAuthStore()
```

## Helper Functions

```tsx
import { getAuthHeader } from '@/store/auth-store'

// Get auth header for manual fetch
const headers = getAuthHeader()
// Returns: { Authorization: 'Bearer token' } or {}
```

## Common Patterns

### Check if user is logged in

```tsx
const isAuthenticated = useAuthStore(state => state.isAuthenticated)
```

### Get current user

```tsx
const user = useAuthStore(state => state.user)
```

### Handle login

```tsx
const { signIn, isLoading, error } = useAuthStore()

const handleLogin = async () => {
  try {
    await signIn(username, password)
    // Success - user is now logged in
  } catch (err) {
    // Error is in the store's error property
    console.error(error)
  }
}
```

### Show loading state

```tsx
const isLoading = useAuthStore(state => state.isLoading)

return isLoading ? <Spinner /> : <Content />
```

## Troubleshooting

**Issue**: "Network Error"
- Check backend is running on port 8080
- Check `.env` has correct `VITE_API_URL`

**Issue**: CORS Error
- Add frontend URL to backend CORS whitelist

**Issue**: Token not refreshing
- Check backend returns valid `expires_at` timestamp

## Documentation

- Full guide: `/frontend/AUTHENTICATION_GUIDE.md`
- Component docs: `/frontend/src/components/auth/README.md`
- Summary: `/frontend/AUTH_IMPLEMENTATION_SUMMARY.md`

## Support

Check browser console for errors and backend logs for API issues.
