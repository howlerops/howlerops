# Security Best Practices Guide

**Version**: 1.0
**Last Updated**: 2025-10-23
**Audience**: Development Team

## Table of Contents
1. [Adding New Permissions](#adding-new-permissions)
2. [Checking Permissions Correctly](#checking-permissions-correctly)
3. [Common Security Pitfalls](#common-security-pitfalls)
4. [Testing Requirements](#testing-requirements)
5. [Code Review Checklist](#code-review-checklist)
6. [Security Patterns](#security-patterns)
7. [Emergency Procedures](#emergency-procedures)

---

## 1. Adding New Permissions

### Step 1: Define the Permission Constant
```go
// In internal/organization/permissions.go
const (
    // Add your new permission with clear naming
    PermYourNewAction Permission = "resource:action"
    // Example: PermExportData Permission = "data:export"
)
```

### Step 2: Assign Permission to Roles
```go
// In internal/organization/permissions.go
var RolePermissions = map[OrganizationRole][]Permission{
    RoleOwner: {
        // ... existing permissions ...
        PermYourNewAction, // Owners usually get all permissions
    },
    RoleAdmin: {
        // ... existing permissions ...
        PermYourNewAction, // Decide if admins need this
    },
    RoleMember: {
        // ... existing permissions ...
        // Members typically don't get admin permissions
    },
}
```

### Step 3: Add Permission Check in Service Layer
```go
// In your service method
func (s *Service) YourNewAction(ctx context.Context, orgID, userID string) error {
    // ALWAYS check membership first
    member, err := s.repo.GetMember(ctx, orgID, userID)
    if err != nil || member == nil {
        return fmt.Errorf("not a member of this organization")
    }

    // Then check specific permission
    if !HasPermission(member.Role, PermYourNewAction) {
        // Log permission denial for audit
        s.CreateAuditLog(ctx, &AuditLog{
            OrganizationID: &orgID,
            UserID:         userID,
            Action:         "permission_denied",
            ResourceType:   "your_resource",
            Details: map[string]interface{}{
                "permission": string(PermYourNewAction),
                "role":       string(member.Role),
                "attempted":  "your_action",
            },
        })
        return fmt.Errorf("insufficient permissions")
    }

    // Perform the action
    // ...
}
```

### Step 4: Update Handler
```go
// In your handler
func (h *Handler) YourNewEndpoint(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Extract user from context (set by auth middleware)
    userID, ok := ctx.Value("user_id").(string)
    if !ok || userID == "" {
        h.respondError(w, http.StatusUnauthorized, "unauthorized")
        return
    }

    // Call service method (permission check happens there)
    err := h.service.YourNewAction(ctx, orgID, userID)
    if err != nil {
        if strings.Contains(err.Error(), "insufficient permissions") {
            h.respondError(w, http.StatusForbidden, err.Error())
            return
        }
        // Handle other errors
    }
}
```

### Step 5: Add Tests
```go
// In security_test.go or your_feature_test.go
func TestYourNewPermission(t *testing.T) {
    // Test owner can perform action
    t.Run("Owner can perform action", func(t *testing.T) {
        // Setup owner scenario
        // Assert success
    })

    // Test admin permissions (if applicable)
    t.Run("Admin permission check", func(t *testing.T) {
        // Setup admin scenario
        // Assert based on permission design
    })

    // Test member cannot perform action
    t.Run("Member cannot perform action", func(t *testing.T) {
        // Setup member scenario
        // Assert returns "insufficient permissions"
    })

    // Test non-member cannot perform action
    t.Run("Non-member cannot perform action", func(t *testing.T) {
        // Setup non-member scenario
        // Assert returns "not a member"
    })
}
```

---

## 2. Checking Permissions Correctly

### ✅ DO: Check at Service Layer
```go
// CORRECT - Permission check in service layer
func (s *Service) DeleteResource(ctx context.Context, orgID, userID, resourceID string) error {
    member, err := s.repo.GetMember(ctx, orgID, userID)
    if err != nil || member == nil {
        return fmt.Errorf("not a member of this organization")
    }

    if !HasPermission(member.Role, PermDeleteResource) {
        return fmt.Errorf("insufficient permissions")
    }

    return s.repo.DeleteResource(ctx, resourceID)
}
```

### ❌ DON'T: Check Only at Handler Layer
```go
// WRONG - Permission check only in handler
func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
    // This is insufficient! Service methods might be called from other places
    if !h.hasPermission(r, PermDeleteResource) {
        h.respondError(w, http.StatusForbidden, "forbidden")
        return
    }

    // Service method has no permission check - DANGEROUS!
    h.service.DeleteResourceUnsafe(ctx, resourceID)
}
```

### ✅ DO: Use Consistent Error Messages
```go
// CORRECT - Consistent error messages
if member == nil {
    return fmt.Errorf("not a member of this organization")
}

if !HasPermission(member.Role, permission) {
    return fmt.Errorf("insufficient permissions")
}
```

### ❌ DON'T: Leak Information in Errors
```go
// WRONG - Reveals too much information
if member == nil {
    return fmt.Errorf("user %s is not in organization %s database table", userID, orgID)
}
```

### ✅ DO: Check Resource Ownership
```go
// CORRECT - Check if user owns the resource
func (s *Service) UpdateQuery(ctx context.Context, userID, queryID string) error {
    query, err := s.repo.GetQuery(ctx, queryID)
    if err != nil {
        return fmt.Errorf("query not found")
    }

    // Members can only update their own queries
    member, _ := s.repo.GetMember(ctx, query.OrganizationID, userID)
    if !CanUpdateResource(member.Role, query.CreatedBy, userID) {
        return fmt.Errorf("insufficient permissions")
    }

    // Proceed with update
}
```

---

## 3. Common Security Pitfalls to Avoid

### Pitfall 1: Forgetting Permission Checks
```go
// ❌ WRONG - No permission check
func (s *Service) GetSensitiveData(ctx context.Context, orgID string) (*Data, error) {
    return s.repo.GetSensitiveData(ctx, orgID) // Direct access!
}

// ✅ CORRECT - With permission check
func (s *Service) GetSensitiveData(ctx context.Context, orgID, userID string) (*Data, error) {
    if err := s.checkMembership(ctx, orgID, userID); err != nil {
        return nil, err
    }
    return s.repo.GetSensitiveData(ctx, orgID)
}
```

### Pitfall 2: Checking Wrong Permission Level
```go
// ❌ WRONG - Admin trying to delete org
if member.Role == RoleAdmin || member.Role == RoleOwner {
    // Allows admin to delete - WRONG!
    return s.repo.DeleteOrganization(ctx, orgID)
}

// ✅ CORRECT - Only owner can delete
if !HasPermission(member.Role, PermDeleteOrganization) {
    return fmt.Errorf("insufficient permissions")
}
```

### Pitfall 3: SQL Injection via String Concatenation
```go
// ❌ WRONG - SQL Injection vulnerability
query := fmt.Sprintf("SELECT * FROM orgs WHERE name = '%s'", userInput)

// ✅ CORRECT - Use parameterized queries
query := "SELECT * FROM orgs WHERE name = ?"
rows, err := db.Query(query, userInput)
```

### Pitfall 4: Storing Sensitive Data in Logs
```go
// ❌ WRONG - Password in logs
logger.Info("User login", "email", email, "password", password)

// ✅ CORRECT - Never log sensitive data
logger.Info("User login attempt", "email", email)
```

### Pitfall 5: Missing Rate Limiting
```go
// ❌ WRONG - No rate limiting
func (s *Service) SendInvitation(ctx context.Context, ...) error {
    // Directly send invitation
    return s.emailService.Send(...)
}

// ✅ CORRECT - With rate limiting
func (s *Service) SendInvitation(ctx context.Context, ...) error {
    if s.rateLimiter != nil {
        if allowed, reason := s.rateLimiter.Check(userID); !allowed {
            return fmt.Errorf("rate limit exceeded: %s", reason)
        }
    }
    return s.emailService.Send(...)
}
```

### Pitfall 6: Improper Error Handling
```go
// ❌ WRONG - Exposes internal details
if err != nil {
    return fmt.Errorf("database error: %v", err)
}

// ✅ CORRECT - Generic error to user, detailed log
if err != nil {
    logger.Error("Database query failed", "error", err)
    return fmt.Errorf("internal server error")
}
```

---

## 4. Testing Requirements for New Features

### Minimum Test Coverage Required

#### 1. Permission Matrix Tests
```go
// Test all role/action combinations
roles := []OrganizationRole{RoleOwner, RoleAdmin, RoleMember}
for _, role := range roles {
    t.Run(fmt.Sprintf("%s role", role), func(t *testing.T) {
        // Test what this role can and cannot do
    })
}
```

#### 2. Boundary Tests
```go
// Test edge cases
- Empty inputs
- Maximum length inputs
- Special characters
- Null/nil values
- Duplicate entries
```

#### 3. Security Tests
```go
// Required security test cases
- SQL injection attempts
- XSS payloads
- Permission bypass attempts
- Token manipulation
- Rate limit testing
```

#### 4. Integration Tests
```go
// Test full flow
- Create resource as owner
- Access resource as member
- Try to modify as non-member
- Verify audit logs created
```

### Test Checklist Template
```go
// For each new endpoint/feature, ensure:
// [ ] Owner can perform action
// [ ] Admin permissions correct
// [ ] Member permissions correct
// [ ] Non-member gets 403
// [ ] Invalid input returns 400
// [ ] Missing auth returns 401
// [ ] Rate limits enforced
// [ ] Audit logs created
// [ ] Error messages don't leak info
// [ ] SQL injection prevented
```

---

## 5. Code Review Checklist for Security

### Authentication & Authorization
- [ ] All endpoints require authentication (except public ones)
- [ ] Permission checks at service layer, not just handlers
- [ ] Consistent use of `HasPermission()` function
- [ ] Resource ownership verified where applicable
- [ ] No hardcoded credentials or secrets

### Input Validation
- [ ] All user inputs validated
- [ ] Length limits enforced
- [ ] Special characters handled
- [ ] Email addresses validated
- [ ] Enum values checked

### Data Protection
- [ ] Parameterized queries used (no string concatenation)
- [ ] Sensitive data not logged
- [ ] Passwords hashed (never stored plain)
- [ ] Tokens generated securely (crypto/rand)
- [ ] HTTPS enforced in production

### Error Handling
- [ ] Generic errors returned to users
- [ ] Detailed errors logged internally
- [ ] No stack traces in responses
- [ ] 403 vs 404 used appropriately

### Audit & Monitoring
- [ ] Audit logs for sensitive operations
- [ ] Permission denials logged
- [ ] Rate limiting on sensitive endpoints
- [ ] Failed attempts tracked

### Testing
- [ ] Security tests included
- [ ] Permission matrix tested
- [ ] Edge cases covered
- [ ] Integration tests present

---

## 6. Security Patterns

### Pattern: Defense in Depth
```go
// Multiple layers of security
func (s *Service) SensitiveOperation(ctx context.Context, ...) error {
    // Layer 1: Authentication (handled by middleware)

    // Layer 2: Membership check
    if err := s.checkMembership(ctx, orgID, userID); err != nil {
        return err
    }

    // Layer 3: Permission check
    if !HasPermission(member.Role, PermSensitiveOp) {
        return fmt.Errorf("insufficient permissions")
    }

    // Layer 4: Rate limiting
    if !s.rateLimiter.Allow(userID) {
        return fmt.Errorf("rate limit exceeded")
    }

    // Layer 5: Input validation
    if err := validateInput(input); err != nil {
        return err
    }

    // Layer 6: Audit logging
    defer s.CreateAuditLog(ctx, ...)

    // Perform operation
}
```

### Pattern: Fail Secure
```go
// Default to deny
func HasPermission(role OrganizationRole, perm Permission) bool {
    perms, ok := RolePermissions[role]
    if !ok {
        return false // Unknown role = no permissions
    }

    for _, p := range perms {
        if p == perm {
            return true
        }
    }

    return false // Default deny
}
```

### Pattern: Least Privilege
```go
// Give minimum required permissions
var RolePermissions = map[OrganizationRole][]Permission{
    RoleMember: {
        PermViewOrganization, // Only what they need
        PermViewQueries,      // Nothing more
    },
}
```

### Pattern: Secure by Default
```go
type Organization struct {
    // Sensitive fields are omitempty by default
    APIKeys []string `json:"api_keys,omitempty"`

    // Public fields are explicit
    Name string `json:"name"`
}
```

---

## 7. Emergency Procedures

### Suspected Security Breach
1. **Immediate Actions**:
   ```bash
   # Revoke all tokens (implement token blacklist)
   # Rotate all secrets
   # Enable enhanced logging
   ```

2. **Investigation**:
   ```sql
   -- Check audit logs for suspicious activity
   SELECT * FROM audit_logs
   WHERE created_at > NOW() - INTERVAL '24 hours'
   AND action LIKE '%failed%' OR action LIKE '%denied%';
   ```

3. **Notification**:
   - Security team
   - Engineering lead
   - Affected users (if data exposed)

### Fixing a Security Vulnerability

1. **Assess Impact**:
   - Determine affected versions
   - Identify exposed data
   - Check if actively exploited

2. **Develop Fix**:
   ```go
   // Fix the vulnerability
   // Add test to prevent regression
   // Update documentation
   ```

3. **Deploy**:
   - Deploy to staging first
   - Run security tests
   - Deploy to production
   - Monitor for issues

4. **Post-Mortem**:
   - Document what happened
   - How it was fixed
   - Preventive measures

---

## Security Resources

### Internal Documentation
- [SECURITY_AUDIT_REPORT.md](./SECURITY_AUDIT_REPORT.md)
- [VULNERABILITY_ASSESSMENT.md](./VULNERABILITY_ASSESSMENT.md)
- [API_DOCUMENTATION.md](./API_DOCUMENTATION.md)

### External Resources
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://golang.org/doc/security)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

### Security Tools
```bash
# Static analysis
go get -u github.com/securego/gosec/v2/cmd/gosec
gosec ./...

# Dependency scanning
go list -m all | nancy sleuth

# Fuzzing
go test -fuzz=FuzzInputValidation

# Load testing with rate limits
hey -n 1000 -c 100 -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/organizations
```

---

## Summary

Security is everyone's responsibility. Follow these practices:

1. **Never skip permission checks**
2. **Validate all inputs**
3. **Use parameterized queries**
4. **Log security events**
5. **Test security scenarios**
6. **Review code for security**
7. **Keep dependencies updated**
8. **Follow the principle of least privilege**

When in doubt, ask the security team or err on the side of caution.

---

**Document Version**: 1.0
**Last Review**: 2025-10-23
**Next Review**: 2025-11-23
**Owner**: Security Team