# SQL Studio Authentication System Documentation

## Overview

Complete JWT-based authentication system for SQL Studio frontend using **shadcn/ui components exclusively**. Integrates with the Go backend for user management, token refresh, and tier-based features.

## Architecture

### Components

```
frontend/src/
├── store/
│   └── auth-store.ts              # Zustand store for auth state
├── components/
│   └── auth/
│       ├── auth-button.tsx        # Header auth button + user menu
│       ├── auth-modal.tsx         # Main auth dialog
│       ├── login-form.tsx         # Login form with validation
│       ├── signup-form.tsx        # Signup form with password strength
│       ├── protected-route.tsx    # Route protection component
│       ├── index.ts               # Barrel exports
│       └── __tests__/
│           └── auth-integration.test.tsx
└── .env.development               # Environment configuration
```

## Features

### 1. User Authentication
- **Sign Up**: New user registration with validation
- **Sign In**: Existing user authentication
- **Sign Out**: Clean logout with backend notification
- **Auto Token Refresh**: Refreshes 5 minutes before expiration
- **Persistent Sessions**: Survives app restarts

### 2. UI Components (shadcn/ui only)

All components use shadcn/ui primitives:
- `Dialog` - Modal container
- `Tabs` - Login/Signup tabs
- `Input` - Form inputs
- `Button` - Actions
- `Label` - Form labels
- `Alert` - Error messages
- `DropdownMenu` - User menu

### 3. Security Features
- Password strength validation (8+ chars, uppercase, number)
- Password confirmation matching
- JWT token management
- Automatic token refresh
- Secure token storage
- HTTPS-only in production

## Installation

### Required shadcn Components

All components are **already installed**:

```bash
# Already installed - no action needed
✓ dialog
✓ tabs
✓ input
✓ button
✓ label
✓ alert
✓ dropdown-menu
```

## Configuration

### Environment Variables

**Development** (`.env.development`):
```env
VITE_API_URL=http://localhost:8080
```

**Production** (`.env.production`):
```env
VITE_API_URL=https://your-production-backend.com
```

## Usage

### 1. Header Integration

The `AuthButton` is already integrated in the header:

```tsx
// frontend/src/components/layout/header.tsx
import { AuthButton } from '@/components/auth/auth-button'

export function Header() {
  return (
    <header>
      <div className="ml-auto flex items-center space-x-4">
        <AuthButton />
        <TierBadge />
        {/* other header items */}
      </div>
    </header>
  )
}
```

### 2. Auth Store Usage

```typescript
import { useAuthStore } from '@/store/auth-store'

function MyComponent() {
  const {
    isAuthenticated,
    user,
    signIn,
    signOut
  } = useAuthStore()

  if (!isAuthenticated) {
    return <div>Please sign in</div>
  }

  return (
    <div>
      Welcome, {user?.username}!
      <button onClick={signOut}>Sign Out</button>
    </div>
  )
}
```

### 3. Protected Routes

```typescript
import { ProtectedRoute } from '@/components/auth'

function App() {
  return (
    <Routes>
      <Route path="/public" element={<PublicPage />} />
      <Route
        path="/protected"
        element={
          <ProtectedRoute>
            <ProtectedPage />
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}
```

### 4. API Integration

Get auth headers for API calls:

```typescript
import { getAuthHeader } from '@/store/auth-store'

// Make authenticated API call
const response = await fetch('/api/user/data', {
  headers: {
    ...getAuthHeader(),
    'Content-Type': 'application/json',
  },
})
```

## API Endpoints

### Backend Endpoints (Go)

**Base URL**: `VITE_API_URL` (default: `http://localhost:8080`)

#### Sign Up
```http
POST /api/auth/signup
Content-Type: application/json

{
  "username": "string",
  "email": "string",
  "password": "string"
}
```

**Response**:
```json
{
  "user": {
    "id": "string",
    "username": "string",
    "email": "string",
    "role": "string",
    "created_at": "ISO8601"
  },
  "token": "string",
  "refresh_token": "string",
  "expires_at": "ISO8601"
}
```

#### Sign In
```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "string",
  "password": "string"
}
```

**Response**: Same as Sign Up

#### Refresh Token
```http
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "string"
}
```

**Response**:
```json
{
  "token": "string",
  "refresh_token": "string",
  "expires_at": "ISO8601"
}
```

#### Sign Out
```http
POST /api/auth/logout
Authorization: Bearer {access_token}
```

## Component Details

### AuthButton

**Location**: `/Users/jacob_1/projects/sql-studio/frontend/src/components/auth/auth-button.tsx`

**Features**:
- Shows "Sign In" button when not authenticated
- Shows user menu dropdown when authenticated
- Displays username and email
- Links to Settings and Subscription
- Sign out action

**Usage**:
```tsx
<AuthButton />
```

### AuthModal

**Location**: `/Users/jacob_1/projects/sql-studio/frontend/src/components/auth/auth-modal.tsx`

**Features**:
- Tabbed interface (Login/Signup)
- Controlled open state
- Default tab selection

**Props**:
```typescript
interface AuthModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  defaultTab?: 'login' | 'signup'
}
```

### LoginForm

**Location**: `/Users/jacob_1/projects/sql-studio/frontend/src/components/auth/login-form.tsx`

**Features**:
- Username or email input
- Password input
- Error display
- Loading states
- Success callback

**Props**:
```typescript
interface LoginFormProps {
  onSuccess?: () => void
}
```

### SignupForm

**Location**: `/Users/jacob_1/projects/sql-studio/frontend/src/components/auth/signup-form.tsx`

**Features**:
- Username input
- Email input with validation
- Password with strength indicators:
  - ✓ At least 8 characters
  - ✓ One uppercase letter
  - ✓ One number
- Password confirmation
- Error display
- Loading states
- Success callback

**Props**:
```typescript
interface SignupFormProps {
  onSuccess?: () => void
}
```

## State Management

### Auth Store (Zustand)

**Location**: `/Users/jacob_1/projects/sql-studio/frontend/src/store/auth-store.ts`

**State**:
```typescript
interface AuthState {
  user: User | null
  tokens: AuthTokens | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
}
```

**Actions**:
- `signUp(username, email, password)` - Register new user
- `signIn(username, password)` - Authenticate user
- `signOut()` - Logout user
- `refreshToken()` - Refresh access token
- `clearError()` - Clear error state

**Helpers**:
- `getAuthHeader()` - Get Authorization header
- `initializeAuthStore()` - Initialize on app startup

### Persistence

Authentication state is persisted to `localStorage`:

**Key**: `auth-storage`

**Persisted Fields**:
- `user`
- `tokens`
- `isAuthenticated`

## Token Refresh Strategy

### Automatic Refresh

Token refresh happens automatically **5 minutes before expiration**:

```typescript
// Refresh schedule
const REFRESH_BUFFER_MS = 5 * 60 * 1000  // 5 minutes

// Timer started after successful sign in/sign up
// Timer stopped on sign out
```

### Manual Refresh

You can also manually trigger refresh:

```typescript
const { refreshToken } = useAuthStore()

const success = await refreshToken()
if (success) {
  console.log('Token refreshed')
} else {
  console.log('Refresh failed, user signed out')
}
```

## Tier Integration

The auth system integrates with the tier store:

### On Sign In/Sign Up
```typescript
// If user has a license key in backend
if (data.user.license_key) {
  useTierStore.getState().activateLicense(data.user.license_key)
}
```

### On Sign Out
```typescript
// Reset to local tier
useTierStore.getState().setTier('local')
```

## Error Handling

### Display Errors

All forms show errors in an Alert component:

```tsx
{error && (
  <Alert variant="destructive">
    <AlertDescription>{error}</AlertDescription>
  </Alert>
)}
```

### Common Errors

**Sign Up**:
- "Username already exists"
- "Email already registered"
- "Password too weak"

**Sign In**:
- "Invalid credentials"
- "User not found"

**Refresh**:
- "Invalid refresh token" → Auto sign out
- "Token expired" → Auto sign out

## Testing

### Run Tests

```bash
cd frontend
npm run test src/components/auth/__tests__/auth-integration.test.tsx
```

### Test Coverage

- ✓ Sign up flow
- ✓ Sign in flow
- ✓ Token refresh
- ✓ Sign out
- ✓ Error handling
- ✓ Persistence

## Security Considerations

### Password Requirements

Enforced in UI and should be validated on backend:
- Minimum 8 characters
- At least one uppercase letter
- At least one number

### Token Storage

- Access token: Short-lived (1 hour typical)
- Refresh token: Longer-lived (7 days typical)
- Stored in localStorage (persisted Zustand store)
- Cleared on sign out

### HTTPS

**Development**: HTTP allowed
**Production**: HTTPS required (configure in backend)

### CORS

Backend must allow requests from frontend origin:
```go
// backend CORS config
AllowedOrigins: []string{"http://localhost:5173", "https://your-domain.com"}
```

## Troubleshooting

### "Network Error" on Sign In

Check:
1. Backend is running: `curl http://localhost:8080/health`
2. VITE_API_URL is correct in `.env.development`
3. CORS is configured on backend

### Tokens Not Refreshing

Check:
1. `expires_at` is in the future
2. Refresh timer is started (check console logs)
3. Refresh endpoint is working

### User Signed Out Automatically

This happens when:
1. Refresh token is invalid/expired
2. Backend rejects refresh request
3. User manually clears localStorage

Check backend logs and verify refresh token validity.

### "Sign In" Button Not Showing

The `AuthButton` is already integrated in the header. If not visible:
1. Check header component imports
2. Verify `AuthButton` is rendered
3. Check z-index and positioning

## Development Tips

### Enable Tier Dev Mode

Unlock all features during development:

```env
# .env.development
VITE_TIER_DEV_MODE=true
```

### Mock Auth in Tests

```typescript
import { useAuthStore } from '@/store/auth-store'

beforeEach(() => {
  useAuthStore.getState().reset()
})

// Mock sign in
useAuthStore.setState({
  isAuthenticated: true,
  user: {
    id: '1',
    username: 'testuser',
    email: 'test@example.com',
    role: 'user',
    created_at: new Date().toISOString()
  }
})
```

### Debug Auth State

```typescript
// Add to your component
const authState = useAuthStore()
console.log('Auth State:', authState)
```

## Next Steps

### Optional Enhancements

1. **Email Verification**
   - Add email verification flow
   - Resend verification email

2. **Password Reset**
   - Forgot password flow
   - Reset password with token

3. **Social Auth**
   - Google OAuth
   - GitHub OAuth

4. **Two-Factor Authentication**
   - TOTP setup
   - Backup codes

5. **Session Management**
   - View active sessions
   - Revoke sessions

## Support

For issues or questions:
1. Check backend logs
2. Check browser console
3. Verify environment variables
4. Review API response formats

## Summary

✅ **Complete authentication system implemented**
✅ **Using shadcn/ui components exclusively**
✅ **JWT-based with auto-refresh**
✅ **Integrated with tier store**
✅ **Production-ready with tests**

All components are fully functional and ready to use!
