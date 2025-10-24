# Authentication System

Complete JWT-based authentication system for SQL Studio frontend that integrates with the Go backend.

## Overview

This authentication system provides:

- **JWT Token Management**: Automatic token refresh before expiration
- **Secure Storage**: Persistent authentication state with Zustand
- **User-Friendly UI**: Beautiful forms with validation and error handling
- **Tier Integration**: Automatic tier updates based on user license
- **Production Ready**: Comprehensive error handling and loading states

## Architecture

### Store (`auth-store.ts`)

Central state management using Zustand with persistence:

```typescript
import { useAuthStore } from '@/store/auth-store'

// Access auth state
const { user, isAuthenticated, isLoading, error } = useAuthStore()

// Sign up
await signUp(username, email, password)

// Sign in
await signIn(username, password)

// Sign out
await signOut()

// Check auth status
if (isAuthenticated) {
  console.log('User is logged in:', user)
}
```

**Features:**
- Automatic token refresh (5 minutes before expiration)
- Persistent storage in localStorage
- Integration with tier-store
- Error handling with user-friendly messages

### Components

#### `<AuthModal />`
Main authentication modal with tabbed interface for login and signup.

```tsx
import { AuthModal } from '@/components/auth'

function MyComponent() {
  const [showAuth, setShowAuth] = useState(false)

  return (
    <AuthModal
      open={showAuth}
      onOpenChange={setShowAuth}
      defaultTab="login" // or "signup"
    />
  )
}
```

#### `<LoginForm />`
Login form with validation and error handling.

**Fields:**
- Username or email
- Password

**Features:**
- Auto-complete support
- Loading states
- Error messages
- Success callback

#### `<SignupForm />`
Signup form with password strength validation.

**Fields:**
- Username
- Email
- Password (with strength indicators)
- Confirm password

**Password Requirements:**
- At least 8 characters
- One uppercase letter
- One number

**Features:**
- Real-time validation
- Visual password strength indicators
- Password match checking
- Loading states

#### `<AuthButton />`
Header button component that shows:
- "Sign In" button for unauthenticated users
- User menu dropdown for authenticated users

**Menu Items:**
- User info (username, email)
- Settings
- Subscription management
- Sign out

### API Client (`auth-client.ts`)

Centralized HTTP client for authenticated requests:

```typescript
import { authFetch, authApi } from '@/lib/api/auth-client'

// Use pre-built endpoints
const user = await authApi.getProfile()

// Make custom authenticated requests
const data = await authFetch('/api/custom-endpoint', {
  method: 'POST',
  body: JSON.stringify({ foo: 'bar' })
})

// Skip authentication (for public endpoints)
const publicData = await authFetch('/api/public', {
  skipAuth: true
})
```

**Features:**
- Automatic token injection
- Automatic token refresh on 401
- Error handling with typed errors
- Support for custom endpoints

## Usage

### 1. Initialize Auth Store

In your main app component:

```tsx
import { useEffect } from 'react'
import { initializeAuthStore } from '@/store/auth-store'

function App() {
  useEffect(() => {
    initializeAuthStore()
  }, [])

  return <YourApp />
}
```

### 2. Add Auth Button to Header

```tsx
import { AuthButton } from '@/components/auth/auth-button'

function Header() {
  return (
    <header>
      <AuthButton />
    </header>
  )
}
```

### 3. Protect Routes

```tsx
import { useAuthStore } from '@/store/auth-store'
import { Navigate } from 'react-router-dom'

function ProtectedRoute({ children }) {
  const isAuthenticated = useAuthStore(state => state.isAuthenticated)

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return children
}
```

### 4. Make Authenticated Requests

```tsx
import { authFetch } from '@/lib/api/auth-client'

async function fetchUserData() {
  try {
    const data = await authFetch('/api/user/data')
    console.log('User data:', data)
  } catch (error) {
    console.error('Failed to fetch:', error)
  }
}
```

## Configuration

### Environment Variables

Create `.env.development` and `.env.production`:

```env
# Development
VITE_API_URL=http://localhost:8080

# Production
VITE_API_URL=https://your-production-backend.com
```

### Backend Endpoints

The system expects these endpoints on the backend:

- `POST /api/auth/signup` - User registration
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout
- `POST /api/auth/refresh` - Token refresh
- `GET /api/auth/me` - Get current user (optional)

### Response Format

**Signup/Login Response:**
```json
{
  "user": {
    "id": "user-uuid",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "user",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": "2024-01-01T01:00:00Z"
}
```

**Refresh Response:**
```json
{
  "token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": "2024-01-01T01:00:00Z"
}
```

## Integration with Tier System

The auth store automatically integrates with the tier store:

1. **On Login/Signup**: Checks if user has a license key and activates it
2. **On Logout**: Resets tier to 'local'
3. **Token Refresh**: Maintains tier state

```typescript
// This happens automatically
signIn(username, password)
  .then(() => {
    // Tier store is updated based on user license
    const currentTier = useTierStore.getState().currentTier
    console.log('Current tier:', currentTier)
  })
```

## Security Considerations

1. **Token Storage**: Tokens are stored in localStorage (persisted)
   - Consider using httpOnly cookies for production
   - Implement CSRF protection if using cookies

2. **Token Refresh**: Automatic refresh 5 minutes before expiration
   - Prevents sudden logouts
   - Maintains user session seamlessly

3. **Error Handling**: Failed requests trigger re-authentication
   - 401 errors automatically attempt token refresh
   - Failed refresh triggers logout

4. **HTTPS Only**: Production should use HTTPS
   - Update `.env.production` with HTTPS URLs
   - Configure backend for secure cookies

## Testing

### Manual Testing Checklist

- [ ] Sign up with new account
- [ ] Sign in with existing account
- [ ] Sign out successfully
- [ ] Token refresh works automatically
- [ ] Error messages display correctly
- [ ] Password validation works
- [ ] Loading states appear
- [ ] Tier integration works
- [ ] Persistent auth across page reload

### Common Issues

**Issue**: "Network Error" on signup/login
- **Solution**: Check that backend is running on correct port
- **Check**: `.env` file has correct `VITE_API_URL`

**Issue**: Token refresh not working
- **Solution**: Check token expiration times
- **Check**: Backend returns valid `expires_at` timestamp

**Issue**: Tier not updating after login
- **Solution**: Ensure backend returns license info
- **Check**: User object includes `license_key` field

## Future Enhancements

- [ ] Add password reset flow
- [ ] Add email verification
- [ ] Add OAuth providers (Google, GitHub)
- [ ] Add session management (view/revoke sessions)
- [ ] Add 2FA support
- [ ] Add password strength meter
- [ ] Add "Remember me" option
- [ ] Add rate limiting feedback
- [ ] Add biometric authentication (for desktop)
- [ ] Add SSO support for team tier

## API Reference

### Auth Store Hooks

```typescript
// Get full state
const auth = useAuthStore()

// Get specific values
const user = useAuthStore(state => state.user)
const isAuthenticated = useAuthStore(state => state.isAuthenticated)
const error = useAuthStore(state => state.error)

// Call actions
const { signIn, signOut, clearError } = useAuthStore()
```

### Auth Helper Functions

```typescript
import { getAuthHeader, initializeAuthStore } from '@/store/auth-store'

// Get auth header for manual requests
const headers = getAuthHeader()
// Returns: { Authorization: 'Bearer token' } or {}

// Initialize on app startup
initializeAuthStore()
```

### Error Types

```typescript
import { AuthApiError } from '@/lib/api/auth-client'

try {
  await authApi.login(username, password)
} catch (error) {
  if (error instanceof AuthApiError) {
    console.log(error.message) // User-friendly message
    console.log(error.status)  // HTTP status code
    console.log(error.code)    // Error code from backend
  }
}
```

## License

Part of SQL Studio - see main LICENSE file.
