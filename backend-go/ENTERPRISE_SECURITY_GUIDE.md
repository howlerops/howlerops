# Enterprise Security Features Guide

## Overview

SQL Studio provides enterprise-grade security features to protect your data and ensure compliance with industry standards. This guide covers the implementation and configuration of SSO, IP whitelisting, two-factor authentication, API key management, and security headers.

## Table of Contents

1. [Single Sign-On (SSO)](#single-sign-on-sso)
2. [IP Whitelisting](#ip-whitelisting)
3. [Two-Factor Authentication (2FA)](#two-factor-authentication-2fa)
4. [API Key Management](#api-key-management)
5. [Security Headers](#security-headers)
6. [Security Events Auditing](#security-events-auditing)
7. [Threat Model](#threat-model)
8. [Best Practices](#best-practices)

## Single Sign-On (SSO)

### Overview

SSO enables organizations to use their existing identity provider for authentication, reducing password fatigue and improving security through centralized access control.

### Supported Providers

- **SAML 2.0**: Okta, OneLogin, Azure AD, PingIdentity
- **OAuth 2.0**: Google Workspace, GitHub Enterprise
- **OpenID Connect (OIDC)**: Auth0, Keycloak, any OIDC-compliant provider

### Configuration

#### SAML Configuration Example

```json
{
  "provider": "saml",
  "provider_name": "Okta",
  "metadata": {
    "idp_metadata_url": "https://company.okta.com/app/metadata",
    "sp_entity_id": "sql-studio-prod",
    "sp_assertion_url": "https://your-domain.com/api/auth/sso/saml/acs",
    "certificate": "-----BEGIN CERTIFICATE-----...",
    "private_key": "-----BEGIN PRIVATE KEY-----..."
  }
}
```

#### OAuth2 Configuration Example

```json
{
  "provider": "oauth2",
  "provider_name": "Google",
  "metadata": {
    "client_id": "your-client-id.apps.googleusercontent.com",
    "client_secret": "your-client-secret",
    "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
    "token_url": "https://oauth2.googleapis.com/token",
    "redirect_url": "https://your-domain.com/api/auth/sso/callback",
    "scopes": ["openid", "profile", "email"]
  }
}
```

### Implementation Status

Currently, the SSO framework includes:
- ✅ Database schema for SSO configuration
- ✅ Mock SSO provider for testing
- ✅ API endpoints for configuration and authentication flow
- ✅ Frontend components for SSO setup
- ⏳ Real provider integrations (future implementation)

### Adding a Real SSO Provider

To implement a real SSO provider:

1. Create a new provider implementation:
```go
type OktaSAMLProvider struct {
    config *SSOProviderConfig
}

func (p *OktaSAMLProvider) GetLoginURL(state string) (string, error) {
    // Implement SAML AuthnRequest generation
}

func (p *OktaSAMLProvider) ValidateAssertion(assertion string) (*SSOUser, error) {
    // Implement SAML assertion validation
}
```

2. Register the provider:
```go
ssoService.RegisterProvider("Okta", NewOktaSAMLProvider(config))
```

## IP Whitelisting

### Overview

IP whitelisting restricts access to your organization based on IP addresses, providing an additional layer of network-level security.

### Features

- Single IP address whitelisting
- CIDR range support (e.g., 192.168.1.0/24)
- Per-organization configuration
- Bypass for specific users/roles

### Configuration

#### Adding IP Addresses

```bash
# Single IP
curl -X POST /api/organizations/{org_id}/ip-whitelist \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "ip_address": "203.0.113.45",
    "description": "Office network"
  }'

# CIDR Range
curl -X POST /api/organizations/{org_id}/ip-whitelist \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "ip_range": "192.168.0.0/16",
    "description": "Corporate VPN"
  }'
```

### Security Considerations

- IP addresses are checked in order: X-Forwarded-For, X-Real-IP, CF-Connecting-IP, RemoteAddr
- Supports proxy and load balancer configurations
- Empty whitelist = no restrictions (default)
- All blocked attempts are logged in security_events

## Two-Factor Authentication (2FA)

### Overview

2FA adds an extra layer of security by requiring a second form of verification beyond passwords.

### Implementation

- **Algorithm**: TOTP (Time-based One-Time Password)
- **Standard**: RFC 6238 compliant
- **Compatible Apps**: Google Authenticator, Authy, 1Password, Microsoft Authenticator
- **Backup Codes**: 10 single-use recovery codes

### Setup Process

1. **Enable 2FA**
```bash
POST /api/auth/2fa/enable
Response: {
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "otpauth://totp/SQL%20Studio:user@example.com?secret=...",
  "backup_codes": ["ABCD1234", "EFGH5678", ...]
}
```

2. **Confirm Setup**
```bash
POST /api/auth/2fa/confirm
Body: { "code": "123456" }
```

3. **Validate on Login**
```bash
POST /api/auth/2fa/validate
Body: { "code": "123456" }
```

### Security Features

- Secrets stored encrypted in database
- Backup codes hashed with bcrypt
- Rate limiting on validation attempts
- Security events logged for all 2FA operations

## API Key Management

### Overview

API keys enable secure programmatic access to SQL Studio without exposing user credentials.

### Key Format

```
sqlstudio_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx  # Production
sqlstudio_test_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx  # Testing
```

### Permissions Model

Available permissions:
- `connections:read` - Read connection configurations
- `connections:write` - Create/update/delete connections
- `queries:read` - View query history
- `queries:write` - Execute queries
- `schemas:read` - View database schemas
- `templates:read` - Access query templates
- `templates:write` - Manage query templates
- `organizations:read` - View organization details
- `organizations:write` - Manage organization settings

Wildcard support:
- `connections:*` - All connection permissions
- `*` - Full access (admin)

### Usage

#### Creating an API Key

```bash
POST /api/api-keys
Body: {
  "name": "Production Server",
  "permissions": ["queries:read", "schemas:read"],
  "expires_in_days": 90
}

Response: {
  "key": "sqlstudio_live_abc123...",  # Only shown once!
  "prefix": "sqlstudio_live_abc123",
  "api_key": { ... }
}
```

#### Using API Keys

```bash
curl -H "Authorization: Bearer sqlstudio_live_abc123..." \
  https://api.sql-studio.com/v1/queries
```

### Security Best Practices

1. **Never commit API keys to version control**
2. **Rotate keys regularly** (90-day default recommendation)
3. **Use minimum required permissions**
4. **Store keys in secure vaults** (e.g., HashiCorp Vault, AWS Secrets Manager)
5. **Monitor key usage** via security events

## Security Headers

### Default Configuration

```go
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https://api.turso.tech
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
Strict-Transport-Security: max-age=31536000; includeSubDomains (HTTPS only)
```

### Strict Production Configuration

For maximum security in production:

```go
config := StrictSecurityConfig()
// Removes 'unsafe-inline' and 'unsafe-eval'
// Enables CSP reporting
// Adds HSTS preload
// Sets stricter Permissions-Policy
```

### CSP Violation Reporting

Configure a CSP report endpoint:
```go
CSPConfig{
    ReportURI: "/api/csp-report",
    ReportOnly: false,  // Set true for testing
}
```

## Security Events Auditing

### Event Types

All security-relevant events are logged:
- `login_success` / `login_failed`
- `2fa_enabled` / `2fa_disabled` / `2fa_validation_failed`
- `ip_blocked` / `ip_whitelist_added` / `ip_whitelist_removed`
- `api_key_created` / `api_key_revoked` / `api_key_invalid_usage`
- `sso_login_initiated` / `sso_login_success` / `sso_login_failed`

### Event Structure

```json
{
  "id": "evt_abc123",
  "event_type": "login_failed",
  "user_id": "user_123",
  "organization_id": "org_456",
  "ip_address": "203.0.113.45",
  "user_agent": "Mozilla/5.0...",
  "details": {
    "reason": "invalid_password",
    "attempts": 3
  },
  "created_at": 1699123456
}
```

### Querying Events

```bash
GET /api/security/events?user_id=xxx&event_type=login_failed&from=2024-01-01
```

## Threat Model

### Identified Threats and Mitigations

| Threat | Risk Level | Mitigation |
|--------|------------|------------|
| **Brute Force Attacks** | High | Rate limiting, account lockout, 2FA requirement |
| **Session Hijacking** | High | Secure session tokens, IP validation, short TTL |
| **SQL Injection** | Critical | Parameterized queries, input validation, least privilege |
| **XSS Attacks** | High | CSP headers, input sanitization, output encoding |
| **CSRF Attacks** | Medium | CSRF tokens, SameSite cookies, state validation |
| **Man-in-the-Middle** | High | HSTS, certificate pinning, TLS 1.2+ only |
| **Unauthorized Access** | High | IP whitelisting, 2FA, API key permissions |
| **Data Exfiltration** | Critical | Query result limits, audit logging, DLP rules |
| **Insider Threats** | Medium | Principle of least privilege, audit logs, key rotation |

### OWASP Top 10 Compliance

| OWASP Risk | Status | Implementation |
|------------|--------|---------------|
| A01: Broken Access Control | ✅ | Role-based permissions, API key scoping |
| A02: Cryptographic Failures | ✅ | bcrypt for passwords, encrypted secrets |
| A03: Injection | ✅ | Parameterized queries, input validation |
| A04: Insecure Design | ✅ | Threat modeling, security by design |
| A05: Security Misconfiguration | ✅ | Security headers, secure defaults |
| A06: Vulnerable Components | ⚠️ | Regular dependency scanning needed |
| A07: Authentication Failures | ✅ | 2FA, rate limiting, secure sessions |
| A08: Data Integrity Failures | ✅ | CSRF protection, input validation |
| A09: Security Logging | ✅ | Comprehensive security event logging |
| A10: SSRF | ✅ | URL validation, network segmentation |

## Best Practices

### For Administrators

1. **Enable 2FA for all users** - Enforce via organization policy
2. **Configure IP whitelisting** - At minimum for admin accounts
3. **Regular security audits** - Review security events weekly
4. **Key rotation policy** - 90-day rotation for API keys
5. **SSO integration** - Centralize authentication when possible

### For Developers

1. **Secure API key storage**
```bash
# Bad
API_KEY="sk_live_abc123..." # Never in code!

# Good
API_KEY=$(vault kv get -field=api_key secret/sql-studio)
```

2. **Minimum permission principle**
```json
{
  "permissions": ["queries:read"],  // Not ["*"]
  "expires_in_days": 30  // Short-lived for automation
}
```

3. **Input validation**
```go
// Validate IP format
if err := middleware.ValidateIP(ipAddress); err != nil {
    return fmt.Errorf("invalid IP: %w", err)
}

// Validate CIDR notation
if err := middleware.ValidateCIDR(ipRange); err != nil {
    return fmt.Errorf("invalid CIDR: %w", err)
}
```

4. **Secure session handling**
```go
// Always regenerate session on privilege escalation
session.Regenerate()

// Bind session to IP
if session.IPAddress != currentIP {
    return ErrSessionHijacked
}
```

### For Security Teams

1. **Monitor security events**
```sql
-- Failed login attempts by IP
SELECT ip_address, COUNT(*) as attempts
FROM security_events
WHERE event_type = 'login_failed'
  AND created_at > NOW() - INTERVAL '1 hour'
GROUP BY ip_address
HAVING COUNT(*) > 5;
```

2. **Implement alerting**
```yaml
alerts:
  - name: brute_force_detection
    condition: login_failed > 10 in 5m
    action: block_ip

  - name: api_key_abuse
    condition: api_calls > 1000 in 1m
    action: revoke_key
```

3. **Regular penetration testing**
- Quarterly external assessments
- Annual red team exercises
- Automated security scanning in CI/CD

## Testing

### Security Test Coverage

```bash
# Run security tests
go test ./internal/sso/... -v
go test ./internal/auth/... -v -run TestTwoFactor
go test ./internal/apikeys/... -v
go test ./internal/middleware/... -v -run TestIPWhitelist

# Coverage report
go test ./... -coverprofile=security.coverage
go tool cover -html=security.coverage
```

### Manual Testing Checklist

- [ ] SSO login flow with mock provider
- [ ] IP whitelist blocks unauthorized IPs
- [ ] 2FA setup and validation
- [ ] API key creation and authentication
- [ ] Security headers present in responses
- [ ] Security events logged correctly
- [ ] Rate limiting prevents brute force
- [ ] CSRF protection on state-changing operations

## Support

For security issues, please contact: security@sql-studio.com

For implementation questions, refer to:
- `/backend-go/internal/sso/` - SSO implementation
- `/backend-go/internal/auth/two_factor.go` - 2FA implementation
- `/backend-go/internal/apikeys/` - API key management
- `/backend-go/internal/middleware/` - Security middleware
- `/frontend/src/components/security/` - Frontend components