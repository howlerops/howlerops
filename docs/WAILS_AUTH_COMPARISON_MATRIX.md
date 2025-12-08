# Wails Authentication Options: Comparison Matrix

Quick reference guide comparing all authentication approaches for Wails applications.

---

## 1. Authentication Methods Comparison

### OAuth2 (Browser-Based)

| Aspect | Rating | Details |
|--------|--------|---------|
| **Status** | ✅ Production Ready | Fully supported, widely used |
| **Cross-Platform** | ✅ Yes | Works on macOS, Windows, Linux |
| **Implementation Difficulty** | ⭐⭐ (Easy) | 2-3 days for first provider |
| **Security** | ⭐⭐⭐⭐⭐ | Industry standard with PKCE |
| **User Experience** | ⭐⭐⭐⭐ | Uses native browser, familiar flow |
| **Setup Requirements** | Medium | Need OAuth app credentials |
| **Token Storage** | OS Keyring | Secure, OS-native |
| **Best For** | Primary login | Most applications should use this |
| **Providers Available** | 100+ | Google, GitHub, Microsoft, etc. |
| **Mobile App Support** | ✅ Yes | Native SDK support on most platforms |

**Verdict:** **RECOMMENDED - Use for primary authentication**

---

### WebAuthn/FIDO2 (Biometric)

| Aspect | Rating | Details |
|--------|--------|---------|
| **Status** | ✅ Production Ready | W3C standard, widely supported |
| **Cross-Platform** | ✅ Yes | Touch ID, Face ID, Windows Hello, Keys |
| **Implementation Difficulty** | ⭐⭐⭐ (Medium) | 3-5 days, JS API straightforward |
| **Security** | ⭐⭐⭐⭐⭐ | Phishing-resistant, strongest option |
| **User Experience** | ⭐⭐⭐⭐⭐ | Fastest login (biometric), very secure |
| **Setup Requirements** | Medium | Backend credential storage needed |
| **Token Storage** | OS Keyring | Secure, OS-native |
| **Best For** | Passwordless auth | Secondary or primary auth (with fallback) |
| **Platform Support** | macOS: ✅✅, Windows: ✅✅, Linux: ⚠️ | Built-in biometrics available |
| **Hardware Keys** | ✅ Yes | YubiKey, Google Titan, etc. supported |

**Verdict:** **RECOMMENDED - Use as primary auth with password fallback**

---

### Platform-Specific Biometrics

#### macOS Touch ID/Face ID

| Aspect | Rating | Details |
|--------|--------|---------|
| **Status** | ✅ Available | Native macOS API |
| **Cross-Platform** | ❌ No | macOS only |
| **Implementation Difficulty** | ⭐⭐⭐⭐⭐ (Very Hard) | Requires Security.framework, complex |
| **Security** | ⭐⭐⭐⭐⭐ | Hardware-backed, very secure |
| **User Experience** | ⭐⭐⭐⭐⭐ | Native, very fast |
| **Setup Requirements** | High | Complex C bindings with cgo |
| **Code Complexity** | 500+ lines of C bindings |  |
| **Best For** | macOS-only apps | Not recommended - use WebAuthn |

**Verdict:** **NOT RECOMMENDED - Use WebAuthn instead**

---

#### Windows Hello

| Aspect | Rating | Details |
|--------|--------|---------|
| **Status** | ✅ Available | Native Windows API |
| **Cross-Platform** | ❌ No | Windows only |
| **Implementation Difficulty** | ⭐⭐⭐⭐⭐ (Very Hard) | Requires Windows SDK, complex |
| **Security** | ⭐⭐⭐⭐⭐ | Hardware-backed, very secure |
| **User Experience** | ⭐⭐⭐⭐⭐ | Native, very fast |
| **Setup Requirements** | Very High | Windows SDK + cgo required |
| **Code Complexity** | 700+ lines of C/C++ bindings |  |
| **Best For** | Enterprise Windows apps | Not recommended - use WebAuthn |

**Verdict:** **NOT RECOMMENDED - Use WebAuthn instead**

---

## 2. Implementation Complexity Comparison

```
Easiest to Hardest:

1. OAuth2 (Browser)
   └─ Open browser → Handle callback → Exchange code → Done
   └─ Implementation: 2-3 days
   └─ Code: ~300 lines (backend) + ~200 lines (frontend)

2. WebAuthn (Biometric)
   └─ Registration: Challenge → Create credential → Verify
   └─ Authentication: Challenge → Get assertion → Verify
   └─ Implementation: 3-5 days
   └─ Code: ~400 lines (backend) + ~250 lines (frontend)

3. Platform-Specific Biometrics
   └─ Windows Hello: Windows SDK + cgo + C++ bindings
   └─ Touch ID: Security.framework + cgo + Objective-C bindings
   └─ Implementation: 1-2 weeks per platform
   └─ Code: 500+ lines per platform
   └─ Result: Not cross-platform, redundant with WebAuthn
```

---

## 3. Security Comparison

### OAuth2 with PKCE

```
Threat Vector        | Status | Mitigation
─────────────────────|────────|──────────────────────────
Authorization Code   | Protected | PKCE challenge/verifier
Interception         |          | S256 hashing
                     |          |
Phishing             | Vulnerable | User must verify OAuth
                     |          | provider's domain
                     |          |
Token Theft          | Protected | Token stored in OS keyring
                     |          | Not accessible to other apps
                     |          |
Client Secret        | Not used | PKCE replaces client secret
Exposure             | ✅ Secure | No secrets in binary
                     |          |
CSRF Attack          | Protected | State parameter validation
                     |          | One-time use enforcement
                     |          |
Man-in-Middle        | Protected | Custom protocol handler
(malicious app       |          | is native OS mechanism
intercepting)        |          | Other apps can't intercept
```

### WebAuthn

```
Threat Vector        | Status | Mitigation
─────────────────────|────────|──────────────────────────
Phishing             | IMMUNE | Server validates origin
                     | ✅ | Challenge-response proves
                     |    | server identity
                     |    |
Stolen Credentials   | Resilient | Private key never leaves
                     | ✅ | device (for platform auth)
                     |    |
Replay Attack        | Protected | Challenge expires/changes
                     | ✅ | Signature only valid once
                     |    |
Account Enumeration  | Protected | No password hint-checking
                     | ✅ |
Brute Force          | Immune | No password to bruteforce
                     | ✅ |
Server Database      | Protected | Only public keys stored
Breach               | ✅ | Private keys never sent
```

---

## 4. Platform Support Matrix

### WebAuthn Support

```
Feature              | macOS | Windows | Linux | Browser Support
─────────────────────|────---|---------|-------|─────────────────
Touch ID             | ✅    | -       | -     | Safari 13+
                     |       |         |       | Chrome (via NFC)
─────────────────────|-------|---------|-------|─────────────────
Face ID              | ✅    | -       | -     | Safari 14+
─────────────────────|-------|---------|-------|─────────────────
Windows Hello        | -     | ✅      | -     | Edge (native)
(Fingerprint)        |       |         |       | Chrome (via NFC)
─────────────────────|-------|---------|-------|─────────────────
Windows Hello        | -     | ✅      | -     | Edge (native)
(Facial)             |       |         |       | Chrome (via NFC)
─────────────────────|-------|---------|-------|─────────────────
Hardware Keys        | ✅    | ✅      | ✅    | Chrome 46+
(FIDO2)              |       |         |       | Firefox 60+
                     |       |         |       | Edge 18+
                     |       |         |       | Safari 13+
─────────────────────|-------|---------|-------|─────────────────
Platform Auth        | ✅    | ✅      | ⚠️    | All platforms
(fallback)           | (Touch| (Hello) | (slow)| support
                     | ID)   |         |      |
```

### Wails WebView Support

```
Component            | macOS    | Windows  | Linux
─────────────────────|----------|----------|──────────
WebView Engine       | WebKit   | WebView2 | WebKitGTK
─────────────────────|----------|----------|──────────
WebAuthn Support     | ✅ Full  | ✅ Full  | ✅ Full
OAuth2 Support       | ✅ Full  | ✅ Full  | ✅ Full
Custom Protocol      | ✅       | ✅       | ✅
Cookie Support       | ⚠️ Limited | ⚠️ Limited | ⚠️ Limited
─────────────────────|----------|----------|──────────
TLS 1.3              | ✅       | ✅       | ✅
Secure Context       | ✅       | ✅       | ✅
(HTTPS/localhost)    |          |          |
```

---

## 5. Token Storage Options

### Storage Method Comparison

```
Method               | Security | Persistence | Access | Best For
─────────────────────|----------|-------------|--------|──────────────
OS Keyring           | ⭐⭐⭐⭐⭐ | ✅ Persist | Go     | Long-term storage
(zalando/keyring)    | Hardware | across     | only   | RECOMMENDED
                     | backed   | restarts   |        |
─────────────────────|----------|-------------|--------|──────────────
sessionStorage       | ⭐⭐⭐   | ❌ Clear   | React  | Single session
                     | Medium   | on app     | only   | Development
                     |          | close      |        |
─────────────────────|----------|-------------|--------|──────────────
localStorage         | ⭐⭐    | ✅ Persist | React  | NOT RECOMMENDED
                     | Weak     | indefinitely| only  | XSS-vulnerable
                     |          |            |        |
─────────────────────|----------|-------------|--------|──────────────
In-Memory (Map)      | ⭐      | ❌ Lost    | Go     | Temporary use
                     | Weak     | on restart | only   | Development
                     |          |            |        |
─────────────────────|----------|-------------|--------|──────────────
Plain File           | ⭐      | ✅ Persist | Both   | NEVER USE
(config directory)   | Exposed  | indefinitely| both   | Readable by
                     |          |            |        | other processes
```

### Recommended Storage Pattern

```
┌─────────────────────────────────────┐
│   Frontend (React)                  │
│   ┌───────────────────────────────┐ │
│   │ SessionStorage               │ │
│   │ (runtime, cleared on exit)   │ │
│   └───────────────────────────────┘ │
└─────────────────────────────────────┘
           ↓
┌─────────────────────────────────────┐
│   Backend (Go)                      │
│   ┌───────────────────────────────┐ │
│   │ Function: StoreTokenSecurely()│ │
│   │ Function: GetTokenSecurely()  │ │
│   │ Uses: go-keyring              │ │
│   └───────────────────────────────┘ │
└─────────────────────────────────────┘
           ↓
┌─────────────────────────────────────┐
│   OS Native                         │
│   • macOS: Keychain                 │
│   • Windows: Credential Manager     │
│   • Linux: GNOME Keyring            │
└─────────────────────────────────────┘

NEVER: Token in localStorage
NEVER: Token in plaintext file
AVOID: Long-lived access tokens
GOOD: Short-lived + refresh tokens
BEST: OS keyring storage
```

---

## 6. Decision Tree

```
What type of authentication do you need?

1. Primary user login?
   ├─ YES → Use OAuth2 (PKCE flow)
   │       └─ Supports: Google, GitHub, Microsoft, etc.
   │
   └─ NO → Skip to password/alternative

2. Do you want passwordless login?
   ├─ YES → Use WebAuthn (biometric + hardware keys)
   │       └─ Touch ID, Face ID, Windows Hello, YubiKey
   │       └─ Set as primary or secondary auth
   │
   └─ NO → Use password + 2FA (if needed)

3. Need biometric support on all platforms?
   ├─ YES → Use WebAuthn (built into all WebViews)
   │       └─ Simpler than platform-specific code
   │       └─ Works on macOS, Windows, Linux, mobile
   │
   └─ NO → Platform-specific APIs (NOT RECOMMENDED)
           └─ Very complex: Touch ID/Face ID/Windows Hello
           └─ Consider WebAuthn instead

4. Need to support older platforms?
   ├─ YES → Use OAuth2 as primary
   │       └─ Add WebAuthn as optional secondary
   │
   └─ NO → Use WebAuthn as primary
           └─ Add password fallback for accessibility

FINAL RECOMMENDATION:
━━━━━━━━━━━━━━━━━━━━━━━━
Primary:    OAuth2 (PKCE) + WebAuthn
Backup:     Password + Email recovery
Storage:    OS Keyring (go-keyring)
Never Use:  Platform-specific biometric APIs
```

---

## 7. Implementation Timeline Estimates

### OAuth2 with PKCE

```
Task                          | Time | Difficulty
──────────────────────────────|------|───────────
1. Set up OAuth app           | 30m  | Easy
   (GitHub, Google, etc.)     |      |
                              |      |
2. Backend: OAuth2Manager     | 4-6h | Medium
   - AuthCodeURL generation   |      |
   - Code exchange            |      |
   - Token storage            |      |
   - Error handling           |      |
                              |      |
3. Frontend: AuthContext      | 3-4h | Easy
   - Login button             |      |
   - Event listeners          |      |
   - Token management         |      |
                              |      |
4. Custom Protocol Handler    | 2-3h | Easy
   - Callback URL handling    |      |
   - OAuth validation         |      |
                              |      |
5. Testing & Debug           | 3-4h | Medium
   - All platforms            |      |
   - Token persistence        |      |
   - Error cases              |      |
                              |      |
TOTAL FIRST PROVIDER          | 1-2 days
Additional providers          | 4-6h each
──────────────────────────────|------|───────────
```

### WebAuthn Biometric

```
Task                          | Time | Difficulty
──────────────────────────────|------|───────────
1. Backend: WebAuthnManager   | 6-8h | Hard
   - Challenge generation     |      |
   - Credential verification  |      |
   - Assertion validation     |      |
   - Database schema          |      |
                              |      |
2. Frontend: Registration     | 4-5h | Medium
   - Challenge request        |      |
   - Credential creation      |      |
   - Verification call        |      |
                              |      |
3. Frontend: Authentication   | 3-4h | Medium
   - Challenge request        |      |
   - Credential get           |      |
   - Assertion verification   |      |
                              |      |
4. Testing Biometrics        | 3-4h | Hard
   - macOS Touch ID test      |      |
   - Windows Hello test       |      |
   - Hardware key test        |      |
   - Error handling           |      |
                              |      |
5. Fallback Auth (password)   | 3-4h | Easy
   - Backup authentication    |      |
   - Account recovery         |      |
                              |      |
TOTAL WEBAUTHN               | 3-5 days
──────────────────────────────|------|───────────
```

### Recommended Phasing

```
Week 1: OAuth2
├─ Day 1: Setup, backend OAuth2Manager
├─ Day 2: Frontend, custom protocol handling
├─ Day 3: Testing, polishing, error handling
└─ Deliverable: Working OAuth login

Week 2: WebAuthn
├─ Day 1: Backend credential storage
├─ Day 2-3: Registration + Authentication flows
├─ Day 4: Testing all platforms
└─ Deliverable: Working biometric auth

Week 3: Polish
├─ Day 1-2: Refresh tokens, session management
├─ Day 3: Security review, rate limiting
├─ Day 4: Documentation, deployment
└─ Deliverable: Production-ready auth
```

---

## 8. Common Pitfalls Quick Reference

| Pitfall | Detection | Fix |
|---------|-----------|-----|
| **No PKCE in OAuth** | Authorization code in URL | Add PKCE: `oauth2.SetAuthURLParam("code_challenge", ...)` |
| **Storing tokens in localStorage** | XSS = token theft | Use OS keyring via Go backend |
| **Client secret in code** | App binary exposed | Remove secret, use PKCE flow |
| **No state validation** | CSRF possible | Store state, validate on callback |
| **Cookie-based auth** | Cookies don't work in WebView | Use tokens + sessionStorage |
| **Very long token lifetime** | Compromised = long access | 1-hour access tokens, rotate refresh |
| **Testing in browser only** | Works in browser, fails in app | Test with `wails dev` |
| **Platform-specific biometrics** | Can't deploy to all platforms | Use WebAuthn instead |
| **No error handling** | Silent failures | Handle 401s, network errors |

---

## 9. Quick Decision Matrix

```
Question                          | Answer | Action
─────────────────────────────────|--------|──────────────
Need to support traditional       | YES    | Add password
login?                            |        | + email recovery
                                  |        |
                                  | NO     | OAuth + WebAuthn
                                  |        | only
                                  |        |
─────────────────────────────────|--------|──────────────
Users on older devices?           | YES    | OAuth primary
                                  |        | WebAuthn fallback
                                  |        |
                                  | NO     | WebAuthn primary
                                  |        | Password fallback
                                  |        |
─────────────────────────────────|--------|──────────────
Targeting specific platform       | YES    | OK to use
(Windows enterprise)?             |        | Windows Hello
                                  |        |
                                  | NO     | Use WebAuthn
                                  |        |
─────────────────────────────────|--------|──────────────
First time building Wails app?    | YES    | Start with OAuth2
                                  |        | (simplest)
                                  |        |
                                  | NO     | OAuth2 + WebAuthn
                                  |        | together
                                  |        |
─────────────────────────────────|--------|──────────────
Have external auth provider?      | YES    | Use OAuth2 +
(Auth0, etc)                      |        | that provider's
                                  |        | SDK for API
                                  |        |
                                  | NO     | Implement OAuth2
                                  |        | yourself
                                  |        |
─────────────────────────────────|--------|──────────────
Ready to deploy?                  | YES    | OAuth2 + OS
                                  |        | keyring storage
                                  |        |
                                  | NO     | sessionStorage
                                  |        | OK for dev
```

---

## Summary Scorecard

### Recommendation Score (out of 5 stars)

| Option | Score | Recommendation |
|--------|-------|-----------------|
| **OAuth2 (PKCE)** | ⭐⭐⭐⭐⭐ | USE IMMEDIATELY |
| **WebAuthn** | ⭐⭐⭐⭐⭐ | USE WITH OAUTH2 |
| **OS Keyring Storage** | ⭐⭐⭐⭐⭐ | REQUIRED |
| **Platform Biometrics** | ⭐ | AVOID |
| **Cookie-based auth** | ⭐ | AVOID |
| **Token in localStorage** | ⭐ | AVOID |

---

**Start with:** OAuth2 (PKCE) + OS Keyring Storage
**Add next:** WebAuthn biometric authentication
**Never use:** Platform-specific biometric APIs
