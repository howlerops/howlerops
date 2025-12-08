# Wails Authentication Research - Complete Guide

This directory contains comprehensive research and implementation guidance for authentication in Wails applications.

## Documents Overview

### 1. **WAILS_AUTH_SUMMARY.md** - START HERE
- **Best for:** Quick overview and decision-making
- **Read time:** 10-15 minutes
- **Contents:**
  - Executive summary of all auth options
  - Key findings and recommendations
  - Security best practices
  - Implementation roadmap
  - Common pitfalls and solutions

**When to read:** First thing - gives you the bird's eye view

---

### 2. **WAILS_AUTH_COMPARISON_MATRIX.md** - DECISION GUIDE
- **Best for:** Comparing options and making trade-offs
- **Read time:** 15-20 minutes
- **Contents:**
  - Detailed comparison tables (security, complexity, platform support)
  - Implementation difficulty ratings
  - Timeline estimates
  - Platform support matrix
  - Decision trees and scoring

**When to read:** When deciding which auth method to implement

---

### 3. **WAILS_AUTHENTICATION_RESEARCH.md** - DEEP DIVE
- **Best for:** Understanding technical details
- **Read time:** 30-45 minutes
- **Contents:**
  - Detailed explanation of each auth method
  - Complete architecture overviews
  - Security analysis
  - Library recommendations
  - Production-ready code patterns (high-level)

**When to read:** Before starting implementation, or when you need deep context

---

### 4. **WAILS_AUTH_QUICK_START.md** - IMPLEMENTATION GUIDE
- **Best for:** Copy-paste code and getting started fast
- **Read time:** 20-30 minutes (reference, not linear)
- **Contents:**
  - Complete OAuth2 implementation with GitHub
  - Complete WebAuthn implementation
  - React Context setup
  - Go backend setup
  - Environment configuration
  - Token management patterns

**When to read:** When you're ready to start coding

---

## Quick Start Path (30 minutes)

1. **Read:** WAILS_AUTH_SUMMARY.md (10 min)
   - Get overview of all options
   - Understand key differences

2. **Review:** WAILS_AUTH_COMPARISON_MATRIX.md sections 1-3 (10 min)
   - Compare implementation difficulty
   - Check platform support
   - See security comparison

3. **Scan:** WAILS_AUTH_QUICK_START.md (10 min)
   - Review code structure
   - Understand patterns
   - Note key functions

4. **Decide:** Which auth to implement first
   - OAuth2 (recommended for primary auth)
   - WebAuthn (recommended as secondary)
   - Or both in parallel

---

## Implementation Path (Week 1-3)

### Week 1: OAuth2 with PKCE

**Day 1-2:**
1. Read WAILS_AUTHENTICATION_RESEARCH.md section "Browser-Based OAuth"
2. Review WAILS_AUTH_QUICK_START.md OAuth2 examples
3. Start with Go backend (OAuth2Manager)
4. Set up GitHub OAuth app

**Day 3:**
1. Build React frontend (AuthContext)
2. Implement custom protocol handler
3. Test locally with `wails dev`
4. Handle errors and edge cases

**Deliverable:** Working OAuth login with GitHub

---

### Week 2: WebAuthn Biometric

**Day 1-2:**
1. Read WAILS_AUTHENTICATION_RESEARCH.md section "WebAuthn/FIDO2 Support"
2. Review WAILS_AUTH_QUICK_START.md WebAuthn examples
3. Build Go backend (WebAuthnManager)
4. Build React registration flow

**Day 3-4:**
1. Build React authentication flow
2. Test on all platforms (macOS, Windows, Linux)
3. Add fallback to password
4. Test with hardware keys if available

**Deliverable:** Working biometric authentication

---

### Week 3: Polish & Production

**Day 1-2:**
1. Read WAILS_AUTHENTICATION_RESEARCH.md section "Security Best Practices"
2. Implement refresh token rotation
3. Add rate limiting to auth endpoints
4. Add audit logging

**Day 3:**
1. Security review and testing
2. Documentation
3. Setup CI/CD for testing
4. Prepare for deployment

**Deliverable:** Production-ready authentication system

---

## Which Document to Read When

| Situation | Read |
|-----------|------|
| "What auth options exist for Wails?" | WAILS_AUTH_SUMMARY.md |
| "Should I use OAuth2 or WebAuthn?" | WAILS_AUTH_COMPARISON_MATRIX.md |
| "How does OAuth2 work with Wails?" | WAILS_AUTHENTICATION_RESEARCH.md |
| "Show me code examples" | WAILS_AUTH_QUICK_START.md |
| "What's the security concern?" | WAILS_AUTH_COMPARISON_MATRIX.md + WAILS_AUTHENTICATION_RESEARCH.md |
| "How long will this take?" | WAILS_AUTH_COMPARISON_MATRIX.md |
| "I'm stuck implementing X" | WAILS_AUTH_QUICK_START.md (or search in WAILS_AUTHENTICATION_RESEARCH.md) |

---

## Key Recommendations Summary

### Primary Authentication: OAuth2
- **Why:** Industry standard, widely understood, secure with PKCE
- **How:** Open browser → User authenticates → Custom protocol callback → Exchange code
- **Time:** 2-3 days for first provider
- **Providers:** GitHub (easiest), Google (most users), Microsoft (enterprise)
- **Example:** Complete working code in WAILS_AUTH_QUICK_START.md

### Secondary Authentication: WebAuthn
- **Why:** Passwordless biometric, phishing-resistant, cross-platform
- **How:** Challenge-response with biometric, works on all platforms via native APIs
- **Time:** 3-5 days including all platforms
- **Biometrics:** Touch ID (macOS), Face ID (macOS), Windows Hello, hardware keys
- **Example:** Complete working code in WAILS_AUTH_QUICK_START.md

### Token Storage: OS Keyring
- **Why:** Secure, OS-native, can't be accessed by other apps
- **How:** Go backend exposes SecureStorage functions to frontend
- **Libraries:** `zalando/go-keyring` or `99designs/keyring`
- **Platforms:** macOS Keychain, Windows Credential Manager, Linux GNOME Keyring

### Avoid: Platform-Specific Biometrics
- **Why:** Complex, cross-platform nightmare, WebAuthn does it better
- **Complexity:** 1-2 weeks per platform
- **Code:** 500+ lines C bindings each
- **Result:** Still doesn't work everywhere

---

## Technology Stack

### Recommended Stack

```
Frontend:
├── React (with React Router)
├── TypeScript
├── Wails Runtime (for Go communication)
└── No auth-specific libraries needed!
    (Uses native navigator.credentials API)

Backend:
├── Go 1.18+
├── golang.org/x/oauth2 (OAuth2 client)
├── github.com/duo-labs/webauthn (WebAuthn)
├── github.com/zalando/go-keyring (Secure storage)
└── Standard library for HTTP/JSON

Data Storage:
├── OS Keyring (tokens)
└── Local database if needed (PostgreSQL, SQLite, etc.)
```

### Minimal Stack (OAuth2 only)

```
Backend:
├── golang.org/x/oauth2
├── github.com/zalando/go-keyring
└── Wails runtime

Frontend:
├── React
└── Standard fetch API
```

### Full Stack (OAuth2 + WebAuthn)

```
Backend:
├── golang.org/x/oauth2
├── github.com/duo-labs/webauthn
├── github.com/zalando/go-keyring
└── github.com/golang-jwt/jwt (optional, for custom JWTs)

Frontend:
├── React
└── (Optional) webauthn-json for helper utilities
```

---

## Important Files & Locations

Generated code examples location:
```
/Users/jacob/projects/amplifier/ai_working/howlerops/
├── WAILS_AUTH_SUMMARY.md (THIS FILE)
├── WAILS_AUTH_COMPARISON_MATRIX.md
├── WAILS_AUTHENTICATION_RESEARCH.md
├── WAILS_AUTH_QUICK_START.md
└── README_AUTHENTICATION.md (this index)
```

---

## Most Important Points

### OAuth2 Security

1. **Use PKCE** - Mandatory for desktop apps
2. **Validate state** - Prevent CSRF attacks
3. **Never store client secrets** - Use PKCE flow instead
4. **Use HTTPS** - All OAuth provider communication
5. **Short token lifetimes** - 1 hour for access tokens

### WebAuthn Security

1. **Phishing-resistant** - Server verifies origin
2. **No passwords** - Uses public key cryptography
3. **Hardware-backed** - Keys never leave device
4. **Cross-platform** - Touch ID, Face ID, Windows Hello, YubiKey

### Wails-Specific

1. **Cookies don't work** - Use tokens instead
2. **Custom protocols needed** - For OAuth callbacks
3. **Use Go for secrets** - Never in frontend
4. **Test with `wails dev`** - Not browser testing
5. **OS keyring available** - Use for secure storage

---

## Testing Checklist

Before deploying to production:

- [ ] OAuth login works on macOS
- [ ] OAuth login works on Windows
- [ ] OAuth login works on Linux
- [ ] Token persists after app restart
- [ ] Token cleared after logout
- [ ] WebAuthn registration works with Touch ID/Face ID
- [ ] WebAuthn registration works with Windows Hello
- [ ] WebAuthn registration works with hardware key
- [ ] WebAuthn authentication works on all platforms
- [ ] Token refresh works when expired
- [ ] API calls include auth header
- [ ] Expired tokens redirect to login
- [ ] Error messages are helpful
- [ ] PKCE state is validated
- [ ] Rate limiting on auth endpoints

---

## FAQ

**Q: Can I use Auth0/Firebase/Supabase with Wails?**
A: Yes, via their OAuth2 endpoints. Don't use their SDKs meant for web apps (they assume cookies). Implement the OAuth2 flow manually instead.

**Q: Should I implement my own OAuth provider?**
A: No. Use existing providers (GitHub, Google, Microsoft). If you need custom auth, use WebAuthn with email recovery instead.

**Q: Why not use platform-specific biometrics?**
A: WebAuthn does it better, works across platforms, and is way less code. Platform-specific implementation is 10x more complex and still only works on one platform.

**Q: Can I use cookies for session management?**
A: Limited support in Wails WebView. Use tokens instead - more reliable and works the same across platforms.

**Q: How do I handle token expiration?**
A: 1-hour access tokens + refresh tokens. When API returns 401, use refresh token to get new access token, then retry request.

**Q: Is WebAuthn ready for production?**
A: Yes, it's the W3C standard. All modern platforms support it. Use as primary auth method.

**Q: What about 2FA/MFA?**
A: WebAuthn is phishing-resistant and stronger than 2FA. If you need additional security, add passwordless email recovery or backup codes.

**Q: Can users log in from multiple devices?**
A: Yes. Each device registers its own credentials. Users have one credential per device.

**Q: How do I revoke access after logout?**
A: Clear tokens from OS keyring. With WebAuthn, credentials are device-specific and can't be used elsewhere.

**Q: What if user switches devices?**
A: Need to re-register biometric on new device, or use recovery code sent via email.

---

## Getting Help

1. **Stuck on implementation?** → Check WAILS_AUTH_QUICK_START.md for code examples
2. **Unsure which auth method?** → Review WAILS_AUTH_COMPARISON_MATRIX.md
3. **Need deep understanding?** → Read WAILS_AUTHENTICATION_RESEARCH.md
4. **Security question?** → See "Security Best Practices" section in WAILS_AUTH_SUMMARY.md
5. **Timeline/effort estimate?** → Check WAILS_AUTH_COMPARISON_MATRIX.md section 7
6. **Want to compare all options?** → Review WAILS_AUTH_COMPARISON_MATRIX.md

---

## Document Structure

Each document is designed to be self-contained but references others when relevant.

### WAILS_AUTH_SUMMARY.md
- Executive summary
- Recommendations
- Security practices
- Roadmap

### WAILS_AUTH_COMPARISON_MATRIX.md
- Detailed comparisons
- Timeline estimates
- Decision trees
- Scoring system

### WAILS_AUTHENTICATION_RESEARCH.md
- Technical details
- Complete explanations
- Library reviews
- Full code patterns

### WAILS_AUTH_QUICK_START.md
- Copy-paste code
- Step-by-step setup
- Configuration examples
- Usage patterns

---

## Next Steps

1. **Read:** Start with WAILS_AUTH_SUMMARY.md
2. **Decide:** Pick OAuth2 as primary (mandatory)
3. **Plan:** Review WAILS_AUTH_COMPARISON_MATRIX.md for timeline
4. **Code:** Follow WAILS_AUTH_QUICK_START.md
5. **Test:** Use wails dev for local testing
6. **Deploy:** Follow production security checklist
7. **Monitor:** Track auth errors and user feedback

---

## Document Versions

- **Created:** 2025-11-15
- **Based on:** Web research + community discussions + Wails v2/v3 docs
- **Accuracy:** High (multiple sources cross-referenced)
- **Completeness:** Comprehensive (covers all practical options)
- **Audience:** Developers building Wails apps with React + Go

---

## Support Links

- **Wails Documentation:** https://wails.io/
- **OAuth2 Best Practices:** https://oauth.net/2/oauth-best-practice/
- **WebAuthn Spec:** https://www.w3.org/TR/webauthn-2/
- **FIDO2 Standard:** https://fidoalliance.org/fido2/
- **Go OAuth2:** https://pkg.go.dev/golang.org/x/oauth2
- **Duo WebAuthn:** https://github.com/duo-labs/webauthn
- **Zalando go-keyring:** https://github.com/zalando/go-keyring

---

**Happy building! Authentication is critical security - take time to implement it right.**
