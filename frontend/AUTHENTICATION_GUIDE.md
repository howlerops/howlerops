# Howlerops Authentication System - Integration Guide

Complete JWT-based authentication system for the frontend, integrated with the Go backend.

## What's Been Implemented

### 1. Core Store (`src/store/auth-store.ts`)

**Features:**
- JWT token management with automatic refresh
- Persistent authentication state
- Integration with tier system
- Comprehensive error handling
- TypeScript types for User and AuthTokens

**Key Functions:**
- `signUp(username, email, password)` - Register new user
- `signIn(username, password)` - Login existing user
- `signOut()` - Logout and clear session
- `refreshToken()` - Refresh JWT token automatically
- `initializeAuthStore()` - Initialize on app startup

**Token Refresh:**
- Automatically refreshes token 5 minutes before expiration
- Handles token refresh failures gracefully
- Logs user out if refresh fails

### 2. UI Components (`src/components/auth/`)

**Components Created:**

#### `AuthModal`
Main authentication dialog with tabs for login/signup
- Responsive design matching app theme
- Tab switching between login and signup
- Success callbacks on authentication

#### `LoginForm`
User login interface
- Username or email input
- Password field with autocomplete
- Loading states
- Error message display
- Form validation

#### `SignupForm`
User registration interface
- Username, email, password inputs
- Password confirmation
- Real-time password strength validation:
  - Minimum 8 characters
  - At least 1 uppercase letter
  - At least 1 number
- Visual validation indicators
- Match checking for password confirmation

#### `AuthButton`
Header component showing auth status
- **Logged Out:** "Sign In" button
- **Logged In:** User menu dropdown with:
  - Username and email display
  - Settings link
  - Subscription management
  - Sign out option

#### `ProtectedRoute`
Route wrapper requiring authentication
- Redirects to home if not authenticated
- Preserves attempted location for redirect after login

### 3. API Client (`src/lib/api/auth-client.ts`)

**Features:**
- Centralized authenticated HTTP client
- Automatic JWT token injection
- Automatic token refresh on 401 errors
- Typed error handling
- Pre-built auth endpoints

**Usage Examples:**

```typescript
// Use pre-built endpoints
import { authApi } from '@/lib/api/auth-client'

await authApi.login(username, password)
await authApi.signup(username, email, password)
await authApi.logout()
await authApi.getProfile()

// Make custom authenticated requests
import { authFetch } from '@/lib/api/auth-client'

const data = await authFetch('/api/custom-endpoint', {
  method: 'POST',
  body: JSON.stringify({ foo: 'bar' })
})
```

### 4. Environment Configuration

Created environment files:
- `.env` - Updated with VITE_API_URL
- `.env.development` - Development config (localhost:8080)
- `.env.production` - Production config template
- `.env.example` - Example configuration

### 5. App Integration

Updated files:
- `src/app.tsx` - Added store initialization
- `src/components/layout/header.tsx` - Added AuthButton

## File Structure

```
frontend/
├── src/
│   ├── store/
│   │   └── auth-store.ts          # Auth state management
│   ├── components/
│   │   └── auth/
│   │       ├── auth-modal.tsx     # Main auth modal
│   │       ├── auth-button.tsx    # Header auth button
│   │       ├── login-form.tsx     # Login form
│   │       ├── signup-form.tsx    # Signup form
│   │       ├── protected-route.tsx # Route protection
│   │       ├── index.ts           # Barrel exports
│   │       └── README.md          # Component docs
│   ├── lib/
│   │   └── api/
│   │       └── auth-client.ts     # HTTP client
│   └── app.tsx                    # App initialization
├── .env                           # Environment config
├── .env.development              # Dev environment
├── .env.production               # Prod environment
├── .env.example                  # Example config
└── AUTHENTICATION_GUIDE.md       # This file
```

## Backend Integration

### Required Endpoints

Your Go backend must implement these endpoints:

#### `POST /api/auth/signup`
Register new user

**Request:**
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePass123"
}
```

**Response (200):**
```json
{
  "user": {
    "id": "uuid-here",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "user",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-01T01:00:00Z"
}
```

#### `POST /api/auth/login`
Login existing user

**Request:**
```json
{
  "username": "johndoe",
  "password": "SecurePass123"
}
```

**Response (200):** Same as signup

#### `POST /api/auth/logout`
Logout user (requires auth header)

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200):**
```json
{
  "message": "Logged out successfully"
}
```

#### `POST /api/auth/refresh`
Refresh access token

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-01T01:00:00Z"
}
```

#### `GET /api/auth/me` (Optional)
Get current user profile

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200):**
```json
{
  "id": "uuid-here",
  "username": "johndoe",
  "email": "john@example.com",
  "role": "user",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Error Responses

All endpoints should return errors in this format:

```json
{
  "message": "Invalid credentials",
  "code": "INVALID_CREDENTIALS"
}
```

Common error codes:
- `INVALID_CREDENTIALS` - Wrong username/password
- `USER_EXISTS` - Username/email already taken
- `VALIDATION_ERROR` - Invalid input data
- `AUTH_REQUIRED` - Missing/invalid token
- `TOKEN_EXPIRED` - Token has expired

## Usage Examples

### 1. Basic Login Flow

```tsx
import { useAuthStore } from '@/store/auth-store'
import { AuthButton } from '@/components/auth'

function MyComponent() {
  return (
    <div>
      <AuthButton />
    </div>
  )
}
```

### 2. Protected Routes

```tsx
import { Routes, Route } from 'react-router-dom'
import { ProtectedRoute } from '@/components/auth'

function AppRoutes() {
  return (
    <Routes>
      <Route path="/public" element={<PublicPage />} />
      <Route
        path="/private"
        element={
          <ProtectedRoute>
            <PrivatePage />
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}
```

### 3. Check Auth Status

```tsx
import { useAuthStore } from '@/store/auth-store'

function MyComponent() {
  const { isAuthenticated, user } = useAuthStore()

  if (isAuthenticated) {
    return <div>Welcome {user?.username}!</div>
  }

  return <div>Please sign in</div>
}
```

### 4. Manual Auth Actions

```tsx
import { useAuthStore } from '@/store/auth-store'

function LoginPage() {
  const { signIn, isLoading, error } = useAuthStore()

  const handleLogin = async () => {
    try {
      await signIn('username', 'password')
      // Success! User is logged in
    } catch (err) {
      // Error is available in store
      console.error('Login failed:', error)
    }
  }

  return (
    <button onClick={handleLogin} disabled={isLoading}>
      {isLoading ? 'Signing in...' : 'Sign In'}
    </button>
  )
}
```

### 5. Authenticated API Requests

```tsx
import { authFetch } from '@/lib/api/auth-client'

async function fetchUserData() {
  try {
    const data = await authFetch('/api/user/data')
    console.log('Data:', data)
  } catch (error) {
    console.error('Request failed:', error)
  }
}
```

## Tier System Integration

The auth system automatically integrates with the tier system:

### On Login/Signup
```typescript
// Automatic tier update based on user license
signIn(username, password).then(() => {
  const tier = useTierStore.getState().currentTier
  console.log('User tier:', tier) // 'local', 'individual', or 'team'
})
```

### On Logout
```typescript
// Automatic tier reset to 'local'
signOut().then(() => {
  const tier = useTierStore.getState().currentTier
  console.log('Tier after logout:', tier) // 'local'
})
```

### Backend Requirements for Tier Integration

Include license information in user object:

```json
{
  "user": {
    "id": "uuid",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "user",
    "license_key": "SQL-INDIVIDUAL-...",  // Optional
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

## Testing Checklist

### Manual Testing

- [ ] **Signup Flow**
  - [ ] Can create new account
  - [ ] Password validation works
  - [ ] Error messages display correctly
  - [ ] Success redirects properly
  - [ ] User is logged in after signup

- [ ] **Login Flow**
  - [ ] Can login with username
  - [ ] Can login with email
  - [ ] Wrong password shows error
  - [ ] Non-existent user shows error
  - [ ] Success logs user in

- [ ] **Logout Flow**
  - [ ] Sign out button works
  - [ ] Auth state clears
  - [ ] Tier resets to 'local'
  - [ ] Redirects to home

- [ ] **Token Refresh**
  - [ ] Tokens refresh automatically
  - [ ] User stays logged in
  - [ ] Failed refresh logs user out

- [ ] **Persistence**
  - [ ] Auth state persists on reload
  - [ ] User stays logged in after closing app
  - [ ] Logout clears persisted state

- [ ] **UI/UX**
  - [ ] Loading states show correctly
  - [ ] Error messages are user-friendly
  - [ ] Form validation is responsive
  - [ ] Password strength indicators work
  - [ ] Auth button shows correct state

### Backend Testing

- [ ] All endpoints return correct response format
- [ ] JWT tokens are valid and include correct expiration
- [ ] Refresh tokens work correctly
- [ ] Error responses follow expected format
- [ ] CORS is configured for frontend origin

## Security Considerations

### Current Implementation

1. **Token Storage**: Tokens stored in localStorage
   - ✅ Survives page reloads
   - ⚠️ Vulnerable to XSS attacks
   - 💡 Consider httpOnly cookies for production

2. **Token Refresh**: Automatic refresh 5 min before expiration
   - ✅ Prevents sudden logouts
   - ✅ Maintains seamless user experience

3. **Error Handling**: 401 errors trigger re-auth
   - ✅ Automatic token refresh attempt
   - ✅ Logout on failed refresh

### Production Recommendations

1. **Use HTTPS Only**
   - Update `.env.production` with HTTPS URLs
   - Configure backend for secure cookies

2. **Implement CSRF Protection**
   - If using cookies instead of localStorage
   - Add CSRF tokens to forms

3. **Rate Limiting**
   - Implement on backend for login/signup
   - Show user-friendly messages on rate limit

4. **Password Requirements**
   - Current: 8 chars, 1 uppercase, 1 number
   - Consider: Special characters, common password check

5. **Session Management**
   - Implement session listing
   - Allow users to revoke sessions
   - Track login locations/devices

## Troubleshooting

### Common Issues

**Issue**: "Network Error" on login
- **Check**: Backend is running on port 8080
- **Check**: `.env` has correct `VITE_API_URL`
- **Fix**: Update environment variable

**Issue**: Token refresh not working
- **Check**: Backend returns valid `expires_at`
- **Check**: Token expiration time is reasonable
- **Fix**: Verify backend token generation

**Issue**: CORS errors
- **Check**: Backend CORS config allows frontend origin
- **Fix**: Add frontend URL to backend CORS whitelist

**Issue**: Tier not updating after login
- **Check**: Backend returns `license_key` in user object
- **Fix**: Update backend response to include license info

**Issue**: Auth state lost on reload
- **Check**: localStorage is enabled
- **Check**: Store persistence is working
- **Fix**: Check browser console for errors

## Next Steps

### Immediate

1. **Start Backend**: Ensure Go backend is running on port 8080
2. **Test Signup**: Create a new account
3. **Test Login**: Login with credentials
4. **Verify Token Refresh**: Leave app open for 60+ minutes
5. **Test Logout**: Sign out and verify state clears

### Future Enhancements

1. **Password Reset Flow**
   - Forgot password link
   - Email verification
   - Token-based reset

2. **Email Verification**
   - Send verification email on signup
   - Verify email before full access

3. **OAuth Providers**
   - Google Sign-In
   - GitHub Sign-In
   - Enterprise SSO

4. **Two-Factor Authentication**
   - TOTP-based 2FA
   - SMS verification
   - Backup codes

5. **Session Management**
   - View active sessions
   - Revoke sessions remotely
   - Device tracking

6. **Enhanced Security**
   - Biometric authentication (desktop)
   - Hardware key support
   - IP-based restrictions

## Support

For questions or issues:
1. Check the component README: `src/components/auth/README.md`
2. Review the auth store implementation
3. Check browser console for errors
4. Verify backend logs for API errors

## License

Part of Howlerops - see main LICENSE file.
