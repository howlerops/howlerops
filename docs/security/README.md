# Security Documentation

Security architecture and implementation details for HowlerOps.

## Components

### [Keychain Integration](keychain/)
✅ **Complete** - OS-native keychain integration for secure credential storage.

**Platforms Supported:**
- macOS: Keychain Access
- Windows: Credential Manager
- Linux: Secret Service API (libsecret)

**Security Properties:**
- Credentials never stored in plaintext
- OS-level encryption and access control
- Per-user credential isolation
- Secure transmission over Wails bridge

**Documents:** 5 comprehensive guides covering API, implementation, and integration

---

### [Password Encryption](encryption/)
🚧 **In Progress** - Zero-knowledge encryption architecture for password storage.

**Status:**
- ✅ Frontend encryption utilities (Web Crypto API)
- ✅ Backend encryption utilities (Go crypto)
- ✅ Database migration for encrypted storage
- ✅ Automatic migration deployment
- ⏳ Auth handler integration
- ⏳ Connection storage/retrieval integration

**Security Properties:**
- Zero-knowledge architecture (server never sees plaintext)
- PBKDF2-SHA256 with 600,000 iterations (OWASP 2023)
- AES-256-GCM authenticated encryption
- Unique salts per user, unique IVs per encryption
- Master key system (1Password/Bitwarden pattern)

**Documents:** 2 detailed design and implementation guides

---

## Security Principles

### Defense in Depth

HowlerOps implements multiple security layers:

1. **Transport Security**: HTTPS/TLS for all network communication
2. **Authentication**: JWT-based user authentication
3. **Authorization**: User-scoped data access
4. **Credential Storage**: OS keychain + encrypted storage
5. **Session Security**: Secure session management
6. **Input Validation**: Server-side validation for all inputs

### Zero-Knowledge Architecture

For password storage, the server operates with zero knowledge of plaintext passwords:
- User password derives encryption key (client-side)
- Master key encrypted with derived key
- Database passwords encrypted with master key
- Server only stores encrypted data

### Secure Development Practices

- Regular security audits
- Dependency vulnerability scanning
- Principle of least privilege
- Secure defaults
- Regular updates to crypto libraries

## Related Documentation

- [Architecture Documentation](../architecture/) - System architecture
- [Deployment Documentation](../deployment/) - Secure deployment practices
