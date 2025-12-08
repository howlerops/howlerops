# Wails Authentication: Executive Summary

## Research Overview

Comprehensive research on authentication options for Wails applications (Go backend + React frontend with native WebView). This document provides key findings and recommendations.

---

## Key Findings

### 1. OAuth2 with PKCE - PRODUCTION READY

**Status:** Fully supported and recommended

**Approach:**
- Open user's default browser for authentication
- Use custom protocol handler (myapp://) for callback
- Exchange authorization code for token in Go backend
- Store token securely in OS keyring

**Why it works:**
- Browser-based OAuth is standard practice
- Desktop apps can't securely store client secrets
- PKCE protects against authorization code interception
- Custom protocol handler is native desktop pattern

**Implementation time:** 2-3 days for first provider (GitHub), 1 day per additional provider

**Code files created:**
- `WAILS_AUTH_QUICK_START.md` - Complete OAuth2 example with GitHub
- Go backend with OAuth2Manager
- React frontend with AuthContext
- Secure token storage using OS keyring

---

### 2. Biometric Authentication - CONDITIONAL

**Status:** Limited native support, WebAuthn fully supported

**Two approaches:**

**Option A: WebAuthn (Recommended)**
- Cross-platform biometric support (Touch ID, Face ID, Windows Hello, hardware keys)
- Works in embedded WebView (no platform-specific code needed)
- Industry standard (FIDO2/W3C standard)
- Implementation time: 3-5 days

**Option B: Platform-specific (Not recommended)**
- Windows Hello requires Windows API bindings (complex)
- macOS Touch ID requires Security.framework (complex)
- Linux has minimal biometric support
- Each platform needs separate implementation
- Implementation time: 1-2 weeks per platform

**Recommendation:** Use WebAuthn instead of platform-specific APIs. Benefits outweigh complexity.

**WebAuthn Status:**
- ✅ Available in all browser engines (WebKit, WebKit2, Chromium)
- ✅ Works in Wails embedded WebView
- ✅ Cross-platform biometric support via OS
- ✅ Phishing-resistant
- ✅ Hardware key support (YubiKey, etc.)

---

### 3. WebAuthn/FIDO2 - PRODUCTION READY

**Status:** Fully supported, recommended for biometric/passwordless auth

**What it provides:**
- Public key cryptography for authentication
- Built-in biometric support (Touch ID, Face ID, Windows Hello)
- Hardware security key support
- Phishing-resistant authentication
- Passwordless login

**Platform support:**
- ✅ macOS: Touch ID, Face ID
- ✅ Windows: Windows Hello
- ✅ Linux: Biometric readers where available
- ✅ All: Hardware keys (FIDO2 U2F)

**Implementation time:** 3-5 days for full setup with backup codes

---

### 4. Platform-Specific Considerations

**Critical Issues:**

1. **Cookie Support:** LIMITED
   - WebView2/WebKit have reduced cookie support
   - Workaround: Use token-based auth instead of cookies
   - Store tokens in sessionStorage or OS keyring

2. **Custom Protocol Handling:** FULLY SUPPORTED
   - macOS: Via CFBundleURLTypes in Info.plist
   - Windows: Automatic via Wails
   - Linux: Automatic via Wails
   - Use for OAuth callbacks (myapp://oauth?code=xxx)

3. **Secure Storage:** FULLY SUPPORTED
   - Use Go packages: `zalando/go-keyring` or `99designs/keyring`
   - Stores in: macOS Keychain, Windows Credential Manager, Linux GNOME Keyring
   - Prevents tokens from being readable by other apps or users

4. **Multi-Window Communication:** SUPPORTED
   - Use runtime.EventsEmit/EventsOn
   - Auth events can trigger updates in all windows

**Limitations vs Web Apps:**
- No reliance on cookies (use tokens instead)
- No localStorage/sessionStorage persistence across app restarts (use OS keyring)
- No built-in session management (implement with tokens + refresh token rotation)

---

## Recommended Authentication Stack

### For Most Applications

```
┌─────────────────────────────────────┐
│     React Frontend (with WebView)   │
├─────────────────────────────────────┤
│  OAuth2 Login                       │
│  WebAuthn Biometric                 │
│  Token Management (sessionStorage)  │
├─────────────────────────────────────┤
│     Go Backend (HTTP/Runtime)       │
├─────────────────────────────────────┤
│  OAuth2Manager (code exchange)      │
│  WebAuthnManager (credential mgmt)  │
│  SecureStorage (OS keyring)         │
├─────────────────────────────────────┤
│     OS Native Secure Storage        │
│  macOS Keychain / Windows Cred Mgr  │
│  Linux GNOME Keyring                │
└─────────────────────────────────────┘
```

### Authentication Flows

**Flow 1: OAuth2 (Primary)**
1. User clicks "Sign in with GitHub" → Frontend calls Go backend
2. Backend generates PKCE pair, returns auth URL → Frontend opens in browser
3. User authenticates with GitHub → GitHub redirects to `myapp://oauth?code=xxx`
4. Wails catches custom protocol → OnUrlOpen handler in Go backend
5. Backend exchanges code for token → Backend emits "authenticated" event
6. Frontend receives token → Stores in OS keyring via backend
7. Token included in all API requests

**Flow 2: WebAuthn Biometric**
1. User clicks "Register biometric" → Frontend requests challenge from backend
2. Backend returns challenge → Frontend calls navigator.credentials.create()
3. User provides fingerprint/face → Browser generates public key credential
4. Frontend sends credential to backend → Backend stores public key
5. Later: User clicks "Authenticate" → Frontend requests challenge
6. Backend returns challenge → Frontend calls navigator.credentials.get()
7. User provides biometric → Browser signs challenge with private key
8. Frontend sends signed assertion → Backend verifies signature
9. Backend returns auth token → Frontend stores token

**Flow 3: Token Refresh**
1. Access token expires (short-lived: 1 hour)
2. Frontend attempts API call → Gets 401 Unauthorized
3. Frontend calls refresh endpoint with refresh token
4. Backend returns new access token
5. Frontend retries original request
6. If refresh fails → Redirect to login

---

## Security Best Practices (CRITICAL)

1. **Use PKCE for OAuth2**
   - Mandatory for all desktop apps
   - Prevents authorization code interception
   - Implementation: `oauth2.SetAuthURLParam("code_challenge", challenge)`

2. **Never Store Client Secrets**
   - Desktop apps can't securely store secrets
   - Use PKCE + authorization code flow instead
   - Store only access tokens in OS keyring

3. **Use HTTPS/Secure Channels**
   - Always use HTTPS for OAuth provider communication
   - Custom protocol handlers are secure (native OS)
   - Localhost is acceptable for development

4. **Validate State Parameter**
   - Prevents CSRF attacks in OAuth flow
   - Store state in memory/session temporarily
   - Delete after use (one-time use)

5. **Short Token Lifetimes**
   - Access tokens: 1 hour
   - Refresh tokens: 30 days (with rotation)
   - Reduces damage if token leaked

6. **Secure Storage for Tokens**
   - ✅ Use OS keyring (best: hardware-backed)
   - ⚠️ sessionStorage (acceptable: cleared on app exit)
   - ❌ localStorage (risky: persists to disk)
   - ❌ Plaintext (never acceptable)

---

## Popular Libraries & Tools

### Go Backend

| Library | Purpose | Status | Size |
|---------|---------|--------|------|
| `golang.org/x/oauth2` | OAuth2 client | Stable | Small |
| `github.com/duo-labs/webauthn` | WebAuthn server | Stable | Medium |
| `github.com/zalando/go-keyring` | OS keyring access | Stable | Small |
| `github.com/golang-jwt/jwt` | JWT tokens | Stable | Small |
| `github.com/markbates/goth` | Multi-provider OAuth | Stable | Large |

**Recommended minimal stack:**
- `golang.org/x/oauth2` (required)
- `github.com/zalando/go-keyring` (required for security)
- `github.com/duo-labs/webauthn` (optional, for biometrics)

**Recommended full stack:**
- All of above plus
- `github.com/golang-jwt/jwt` (for custom JWT generation)

### React Frontend

| Library | Purpose | Status | Notes |
|---------|---------|--------|-------|
| React (native) | Context API | Stable | Built-in, no dep needed |
| - | navigator.credentials | Stable | Native browser API |
| - | window.go.main | Stable | Native Wails binding |

**Recommended approach:** Zero additional auth dependencies
- Use React Context API for state
- Use native WebAuthn API
- Use Wails runtime for Go communication

**Optional libraries (if needed):**
- `zustand` or `jotai` - Lighter than Redux
- `axios` - Simpler than fetch (for API calls)

---

## Implementation Roadmap

### Phase 1: Basic OAuth (Week 1)
- [ ] Set up GitHub OAuth app
- [ ] Implement OAuth2Manager in Go (with PKCE)
- [ ] Implement AuthContext in React
- [ ] Add custom protocol handler
- [ ] Test end-to-end OAuth flow
- [ ] Add secure token storage with go-keyring
- [ ] Implement logout

**Deliverable:** Working OAuth login with GitHub

### Phase 2: WebAuthn (Week 2)
- [ ] Set up WebAuthnManager in Go
- [ ] Create WebAuthn registration flow (React)
- [ ] Create WebAuthn authentication flow (React)
- [ ] Add fallback to password login
- [ ] Test on all platforms (macOS, Windows, Linux)

**Deliverable:** Working biometric authentication

### Phase 3: Polish & Security (Week 3)
- [ ] Implement refresh token rotation
- [ ] Add session timeout
- [ ] Add rate limiting to auth endpoints
- [ ] Add account recovery options
- [ ] Add audit logging
- [ ] Security review + testing
- [ ] Documentation

**Deliverable:** Production-ready authentication

### Phase 4: Multi-Provider (Optional)
- [ ] Add Google OAuth
- [ ] Add Microsoft OAuth
- [ ] Add account linking
- [ ] Add social login with email association

---

## Common Pitfalls & Solutions

| Pitfall | Issue | Solution |
|---------|-------|----------|
| Using cookies for auth | Limited WebView support | Use tokens instead (sessionStorage + OS keyring) |
| Not using PKCE | Authorization code can be intercepted | Always use PKCE for OAuth2 |
| Client secret in code | Exposed if app reversed | Use PKCE flow, never store secrets |
| Storing tokens in localStorage | Vulnerable if XSS | Use OS keyring via Go backend |
| No state validation | CSRF attack possible | Validate state parameter in OAuth callback |
| Long token lifetimes | Compromised token = long access | Use 1-hour access tokens, rotate refresh tokens |
| Ignoring WebView limitations | Auth breaks in production | Test OAuth flow with Wails WebView, not browser |
| No token refresh handling | Sessions fail silently | Implement 401 → refresh → retry pattern |
| Platform-specific code for biometrics | Complex, brittle | Use WebAuthn instead (cross-platform) |

---

## Testing Checklist

- [ ] OAuth flow works on macOS
- [ ] OAuth flow works on Windows
- [ ] OAuth flow works on Linux
- [ ] Token persists after app restart
- [ ] Token cleared after logout
- [ ] WebAuthn registration works
- [ ] WebAuthn authentication works
- [ ] WebAuthn works with Touch ID (macOS)
- [ ] WebAuthn works with Windows Hello (Windows)
- [ ] WebAuthn works with hardware key (all platforms)
- [ ] Token refresh works after expiration
- [ ] API calls include auth header
- [ ] Invalid/expired tokens trigger re-login
- [ ] Can't access protected pages when logged out

---

## Next Actions

1. **Read detailed research:** See `WAILS_AUTHENTICATION_RESEARCH.md`
2. **Start implementation:** Follow `WAILS_AUTH_QUICK_START.md`
3. **Choose OAuth provider:**
   - GitHub (easiest for dev)
   - Google (most users recognize)
   - Microsoft (enterprise)
4. **Test on Wails WebView:** Don't test in browser - use `wails dev`
5. **Implement refresh tokens:** After basic OAuth works
6. **Add WebAuthn:** After OAuth is solid
7. **Security review:** Before production deployment

---

## References

**Documentation:**
- Wails: https://wails.io/docs/
- OAuth2: https://oauth.net/2/oauth-best-practice/
- WebAuthn: https://www.w3.org/TR/webauthn-2/
- FIDO2: https://fidoalliance.org/fido2/

**Libraries:**
- Go OAuth2: https://pkg.go.dev/golang.org/x/oauth2
- Duo WebAuthn: https://github.com/duo-labs/webauthn
- Go Keyring: https://github.com/zalando/go-keyring

**Examples:**
- See `WAILS_AUTH_QUICK_START.md` for complete code examples

---

## Questions & Troubleshooting

**Q: Can I use Auth0/Supabase/Firebase with Wails?**
A: Yes, but implement OAuth2 flow manually. These services are designed for web apps with cookies, but their OAuth2 endpoints work fine with Wails' token-based approach.

**Q: Should I implement my own OAuth server?**
A: No. Use existing providers (GitHub, Google, Microsoft). If you need custom auth, use WebAuthn instead.

**Q: What if my users don't have biometrics enabled?**
A: Provide fallback to password login or passwordless email link authentication.

**Q: Can I use the same OAuth app for web and desktop?**
A: Yes, but add both redirect URIs: web URL AND custom protocol (myapp://oauth)

**Q: How do I handle token expiration?**
A: Catch 401 responses, use refresh token to get new access token, retry request.

**Q: Is WebAuthn safe for production?**
A: Yes, it's the modern standard. All major platforms support it. Use as primary auth method.

**Q: What about two-factor authentication?**
A: WebAuthn is phishing-resistant and inherently stronger than 2FA. For additional security, add passwordless email recovery or backup codes.

---

## Support & Questions

If implementing, refer to:
1. `WAILS_AUTHENTICATION_RESEARCH.md` - Deep dive on all options
2. `WAILS_AUTH_QUICK_START.md` - Copy-paste code examples
3. Wails GitHub discussions - Real-world implementations
4. OAuth provider documentation - Provider-specific setup

Good luck with implementation!
