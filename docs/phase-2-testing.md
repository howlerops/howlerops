# Phase 2: Individual Tier Backend - Testing Checklist

## Overview
Comprehensive testing requirements for Phase 2, covering authentication, sync, payments, and overall system integration. This checklist ensures quality and reliability before beta launch.

**Phase:** Phase 2 - Weeks 5-12
**Last Updated:** 2025-10-23
**Status:** Active

---

## Testing Philosophy

### Testing Pyramid
```
        /\
       /E2E\       10% - End-to-end tests
      /------\
     /  INT   \    30% - Integration tests
    /----------\
   /   UNIT     \  60% - Unit tests
  /--------------\
```

### Coverage Targets
- Unit Tests: >80% coverage
- Integration Tests: All critical paths
- E2E Tests: All user flows
- Security Tests: OWASP Top 10
- Performance Tests: Key metrics met

---

## 1. Authentication Testing

### 1.1 User Registration

#### Test Cases
- [ ] **REG-001:** Register with valid email and password
  - Input: Valid email, strong password
  - Expected: User created, verification email sent
  - Test Type: E2E
  - Priority: Critical

- [ ] **REG-002:** Register with existing email
  - Input: Email already in database
  - Expected: Error message "Email already registered"
  - Test Type: Integration
  - Priority: High

- [ ] **REG-003:** Register with weak password
  - Input: Password < 8 chars or missing requirements
  - Expected: Validation error with requirements
  - Test Type: Unit
  - Priority: High

- [ ] **REG-004:** Register with invalid email format
  - Input: "notanemail"
  - Expected: Validation error
  - Test Type: Unit
  - Priority: Medium

- [ ] **REG-005:** Email verification flow
  - Input: Click verification link
  - Expected: Email verified, can log in
  - Test Type: E2E
  - Priority: Critical

- [ ] **REG-006:** Expired verification link
  - Input: Verification link >24h old
  - Expected: Error, option to resend
  - Test Type: Integration
  - Priority: Medium

- [ ] **REG-007:** Rate limiting on registration
  - Input: 10 registration attempts in 1 minute
  - Expected: Rate limit error after 5 attempts
  - Test Type: Integration
  - Priority: High

### 1.2 User Login

#### Test Cases
- [ ] **LOGIN-001:** Login with correct credentials
  - Input: Registered email + correct password
  - Expected: JWT token returned, user logged in
  - Test Type: E2E
  - Priority: Critical

- [ ] **LOGIN-002:** Login with incorrect password
  - Input: Registered email + wrong password
  - Expected: Error "Invalid credentials"
  - Test Type: Integration
  - Priority: High

- [ ] **LOGIN-003:** Login with unverified email
  - Input: Email not verified
  - Expected: Error "Please verify your email"
  - Test Type: Integration
  - Priority: High

- [ ] **LOGIN-004:** Login rate limiting
  - Input: 10 failed login attempts
  - Expected: Account locked for 15 minutes
  - Test Type: Integration
  - Priority: High

- [ ] **LOGIN-005:** "Remember Me" functionality
  - Input: Check "Remember Me" box
  - Expected: Token TTL extended to 30 days
  - Test Type: Integration
  - Priority: Medium

### 1.3 OAuth Login

#### Test Cases
- [ ] **OAUTH-001:** GitHub OAuth login (new user)
  - Input: Click "Login with GitHub"
  - Expected: User created, logged in
  - Test Type: E2E
  - Priority: Critical

- [ ] **OAUTH-002:** GitHub OAuth login (existing user)
  - Input: GitHub account linked to existing user
  - Expected: User logged in
  - Test Type: E2E
  - Priority: Critical

- [ ] **OAUTH-003:** Google OAuth login (new user)
  - Input: Click "Login with Google"
  - Expected: User created, logged in
  - Test Type: E2E
  - Priority: Critical

- [ ] **OAUTH-004:** OAuth callback failure
  - Input: OAuth provider error
  - Expected: Error message, fallback to email login
  - Test Type: Integration
  - Priority: Medium

- [ ] **OAUTH-005:** OAuth account linking
  - Input: Link GitHub to existing email account
  - Expected: Accounts linked successfully
  - Test Type: E2E
  - Priority: Medium

### 1.4 Session Management

#### Test Cases
- [ ] **SESSION-001:** Token refresh before expiry
  - Input: Token expires in 1 minute
  - Expected: Automatic refresh, no logout
  - Test Type: Integration
  - Priority: Critical

- [ ] **SESSION-002:** Token expiry handling
  - Input: Token expired, no refresh
  - Expected: Redirect to login
  - Test Type: Integration
  - Priority: High

- [ ] **SESSION-003:** Multiple device sessions
  - Input: Log in on 2 devices
  - Expected: Both sessions active
  - Test Type: E2E
  - Priority: High

- [ ] **SESSION-004:** Logout single session
  - Input: Click logout on Device A
  - Expected: Device A logged out, Device B still active
  - Test Type: E2E
  - Priority: Medium

- [ ] **SESSION-005:** Logout all sessions
  - Input: Click "Logout everywhere"
  - Expected: All sessions invalidated
  - Test Type: E2E
  - Priority: Medium

### 1.5 Password Reset

#### Test Cases
- [ ] **RESET-001:** Request password reset
  - Input: Valid email address
  - Expected: Reset email sent
  - Test Type: E2E
  - Priority: High

- [ ] **RESET-002:** Reset with valid token
  - Input: Click reset link, enter new password
  - Expected: Password updated, logged in
  - Test Type: E2E
  - Priority: Critical

- [ ] **RESET-003:** Reset with expired token
  - Input: Token >1 hour old
  - Expected: Error, option to request new link
  - Test Type: Integration
  - Priority: Medium

- [ ] **RESET-004:** Reset token used twice
  - Input: Use same reset link twice
  - Expected: Second attempt fails
  - Test Type: Integration
  - Priority: High

- [ ] **RESET-005:** All sessions logged out after reset
  - Input: Reset password
  - Expected: All existing sessions invalidated
  - Test Type: Integration
  - Priority: High

---

## 2. Sync Testing

### 2.1 Initial Sync

#### Test Cases
- [ ] **SYNC-001:** Initial sync on first login
  - Input: New user logs in
  - Expected: Empty data synced, ready for use
  - Test Type: E2E
  - Priority: Critical

- [ ] **SYNC-002:** Initial sync with local data
  - Input: User with local data logs in
  - Expected: Local data uploaded to cloud
  - Test Type: E2E
  - Priority: Critical

- [ ] **SYNC-003:** Initial sync with cloud data
  - Input: User logs in on new device
  - Expected: Cloud data downloaded to device
  - Test Type: E2E
  - Priority: Critical

- [ ] **SYNC-004:** Initial sync progress indicator
  - Input: Large dataset (1000+ records)
  - Expected: Progress shown: "Syncing... 500/1000"
  - Test Type: E2E
  - Priority: High

### 2.2 Connection Sync

#### Test Cases
- [ ] **CONN-001:** Create connection, sync to cloud
  - Input: Create new database connection
  - Expected: Connection in Turso (without password)
  - Test Type: E2E
  - Priority: Critical

- [ ] **CONN-002:** Update connection, sync changes
  - Input: Edit connection name
  - Expected: Updated in Turso
  - Test Type: E2E
  - Priority: High

- [ ] **CONN-003:** Delete connection, soft delete sync
  - Input: Delete connection
  - Expected: deleted_at set in Turso
  - Test Type: E2E
  - Priority: High

- [ ] **CONN-004:** Credentials not synced
  - Input: Connection with password
  - Expected: Password field null in Turso
  - Test Type: Integration
  - Priority: Critical

- [ ] **CONN-005:** Connection appears on other device
  - Input: Create connection on Device A
  - Expected: Visible on Device B within 5 seconds
  - Test Type: E2E
  - Priority: Critical

### 2.3 Query Tab Sync

#### Test Cases
- [ ] **TAB-001:** Create tab, sync immediately
  - Input: Create new query tab
  - Expected: Tab in Turso within 1 second
  - Test Type: E2E
  - Priority: Critical

- [ ] **TAB-002:** Edit tab content, debounced sync
  - Input: Type in tab, pause 2+ seconds
  - Expected: Content synced after 2s debounce
  - Test Type: E2E
  - Priority: Critical

- [ ] **TAB-003:** Rapid typing doesn't overwhelm sync
  - Input: Type continuously for 10 seconds
  - Expected: Only 1 sync after stopping
  - Test Type: E2E
  - Priority: High

- [ ] **TAB-004:** Reorder tabs, sync positions
  - Input: Drag tab to new position
  - Expected: Positions synced
  - Test Type: E2E
  - Priority: Medium

- [ ] **TAB-005:** Delete tab, sync deletion
  - Input: Close tab
  - Expected: Soft deleted in Turso
  - Test Type: E2E
  - Priority: High

- [ ] **TAB-006:** Tab content appears on other device
  - Input: Type query on Device A
  - Expected: Appears on Device B after debounce
  - Test Type: E2E
  - Priority: Critical

### 2.4 Query History Sync

#### Test Cases
- [ ] **HIST-001:** Execute query, add to history
  - Input: Run SELECT query
  - Expected: Query in history, synced to Turso
  - Test Type: E2E
  - Priority: High

- [ ] **HIST-002:** History batching (10 queries)
  - Input: Execute 10 queries
  - Expected: Batched into 1 sync request
  - Test Type: Integration
  - Priority: Medium

- [ ] **HIST-003:** Query redaction if enabled
  - Input: Execute query with "password" keyword
  - Expected: Value redacted in sync
  - Test Type: Integration
  - Priority: High

- [ ] **HIST-004:** Disable history sync (privacy)
  - Input: Toggle "Sync history" off
  - Expected: No history synced
  - Test Type: Integration
  - Priority: Medium

### 2.5 Offline Sync

#### Test Cases
- [ ] **OFFLINE-001:** Queue changes while offline
  - Input: Go offline, create connection
  - Expected: Queued locally
  - Test Type: E2E
  - Priority: Critical

- [ ] **OFFLINE-002:** Flush queue when back online
  - Input: Go back online
  - Expected: Queue flushed automatically
  - Test Type: E2E
  - Priority: Critical

- [ ] **OFFLINE-003:** Offline indicator visible
  - Input: Go offline
  - Expected: "Offline" badge shown
  - Test Type: E2E
  - Priority: High

- [ ] **OFFLINE-004:** Manual retry button
  - Input: Click "Retry sync"
  - Expected: Queue flush attempted
  - Test Type: E2E
  - Priority: Medium

- [ ] **OFFLINE-005:** Offline queue persists across restart
  - Input: Queue changes, close app, reopen
  - Expected: Queue still present
  - Test Type: E2E
  - Priority: High

- [ ] **OFFLINE-006:** Retry with exponential backoff
  - Input: Sync fails 3 times
  - Expected: Delays: 1s, 2s, 4s
  - Test Type: Integration
  - Priority: Medium

### 2.6 Conflict Resolution

#### Test Cases
- [ ] **CONFLICT-001:** Detect concurrent modification
  - Input: Edit tab on Device A and B simultaneously
  - Expected: Conflict detected
  - Test Type: E2E
  - Priority: Critical

- [ ] **CONFLICT-002:** Last-Write-Wins resolution
  - Input: Conflict with timestamps
  - Expected: Newer timestamp wins
  - Test Type: Integration
  - Priority: Critical

- [ ] **CONFLICT-003:** Conflict logged
  - Input: Conflict resolved
  - Expected: Entry in sync_conflicts table
  - Test Type: Integration
  - Priority: Medium

- [ ] **CONFLICT-004:** User notified of conflict
  - Input: Conflict auto-resolved
  - Expected: Notification shown
  - Test Type: E2E
  - Priority: High

- [ ] **CONFLICT-005:** Manual conflict resolution
  - Input: User chooses version manually
  - Expected: Chosen version applied
  - Test Type: E2E
  - Priority: Medium

---

## 3. Payment Testing

### 3.1 Subscription Checkout

#### Test Cases
- [ ] **PAY-001:** Successful checkout (monthly)
  - Input: Stripe test card 4242424242424242
  - Expected: Subscription created, tier upgraded
  - Test Type: E2E
  - Priority: Critical

- [ ] **PAY-002:** Successful checkout (annual)
  - Input: Annual plan selected
  - Expected: $90 charged, tier upgraded
  - Test Type: E2E
  - Priority: High

- [ ] **PAY-003:** Failed payment (declined card)
  - Input: Test card 4000000000000002
  - Expected: Error message, no subscription
  - Test Type: E2E
  - Priority: High

- [ ] **PAY-004:** Failed payment (insufficient funds)
  - Input: Test card 4000000000009995
  - Expected: Error message with retry option
  - Test Type: E2E
  - Priority: Medium

- [ ] **PAY-005:** 3D Secure authentication
  - Input: Test card 4000002500003155
  - Expected: 3DS prompt, successful after auth
  - Test Type: E2E
  - Priority: High

### 3.2 Subscription Management

#### Test Cases
- [ ] **SUB-001:** View subscription details
  - Input: Navigate to billing page
  - Expected: Current plan, next billing date shown
  - Test Type: E2E
  - Priority: High

- [ ] **SUB-002:** Update payment method
  - Input: Add new card via Stripe portal
  - Expected: New card saved
  - Test Type: E2E
  - Priority: High

- [ ] **SUB-003:** Cancel subscription
  - Input: Click "Cancel subscription"
  - Expected: Canceled at end of period
  - Test Type: E2E
  - Priority: Critical

- [ ] **SUB-004:** Reactivate canceled subscription
  - Input: Reactivate before period end
  - Expected: Subscription active again
  - Test Type: E2E
  - Priority: Medium

- [ ] **SUB-005:** Download invoices
  - Input: Click "Download invoice"
  - Expected: PDF downloaded
  - Test Type: E2E
  - Priority: Low

### 3.3 Webhook Testing

#### Test Cases
- [ ] **WEBHOOK-001:** checkout.session.completed
  - Input: Successful checkout
  - Expected: User tier upgraded, webhook logged
  - Test Type: Integration
  - Priority: Critical

- [ ] **WEBHOOK-002:** customer.subscription.updated
  - Input: Payment method updated
  - Expected: Subscription record updated
  - Test Type: Integration
  - Priority: High

- [ ] **WEBHOOK-003:** customer.subscription.deleted
  - Input: Subscription canceled
  - Expected: User tier downgraded
  - Test Type: Integration
  - Priority: Critical

- [ ] **WEBHOOK-004:** invoice.payment_failed
  - Input: Payment fails
  - Expected: Email sent, grace period started
  - Test Type: Integration
  - Priority: High

- [ ] **WEBHOOK-005:** invoice.payment_succeeded
  - Input: Recurring payment succeeds
  - Expected: Subscription extended
  - Test Type: Integration
  - Priority: High

- [ ] **WEBHOOK-006:** Webhook signature verification
  - Input: Webhook with invalid signature
  - Expected: Rejected (400 error)
  - Test Type: Integration
  - Priority: Critical

- [ ] **WEBHOOK-007:** Idempotent webhook handling
  - Input: Same webhook received twice
  - Expected: Processed only once
  - Test Type: Integration
  - Priority: High

### 3.4 Trial Period

#### Test Cases
- [ ] **TRIAL-001:** 14-day trial starts correctly
  - Input: New subscription
  - Expected: Trial end date = now + 14 days
  - Test Type: Integration
  - Priority: High

- [ ] **TRIAL-002:** Access during trial
  - Input: User in trial period
  - Expected: Full Individual tier access
  - Test Type: E2E
  - Priority: Critical

- [ ] **TRIAL-003:** Trial expiry (convert)
  - Input: Trial period ends, payment succeeds
  - Expected: Converted to paid
  - Test Type: Integration
  - Priority: Critical

- [ ] **TRIAL-004:** Trial expiry (no payment)
  - Input: Trial ends, no payment method
  - Expected: Downgraded to Free tier
  - Test Type: Integration
  - Priority: High

- [ ] **TRIAL-005:** Trial cancellation
  - Input: Cancel during trial
  - Expected: No charge, downgraded
  - Test Type: E2E
  - Priority: Medium

---

## 4. Security Testing

### 4.1 Authentication Security

#### Test Cases
- [ ] **SEC-001:** SQL injection in login
  - Input: `'; DROP TABLE users; --`
  - Expected: Sanitized, no SQL injection
  - Test Type: Security
  - Priority: Critical

- [ ] **SEC-002:** XSS in registration
  - Input: `<script>alert('xss')</script>` in name
  - Expected: Escaped, no XSS
  - Test Type: Security
  - Priority: Critical

- [ ] **SEC-003:** CSRF protection
  - Input: Forge login request from external site
  - Expected: CSRF token validation fails
  - Test Type: Security
  - Priority: High

- [ ] **SEC-004:** JWT token tampering
  - Input: Modify JWT payload
  - Expected: Signature validation fails
  - Test Type: Security
  - Priority: Critical

- [ ] **SEC-005:** Brute force protection
  - Input: 100 login attempts
  - Expected: Rate limited after 10 attempts
  - Test Type: Security
  - Priority: High

### 4.2 Data Security

#### Test Cases
- [ ] **SEC-006:** Credentials not in Turso
  - Input: Inspect Turso database
  - Expected: No password fields populated
  - Test Type: Security
  - Priority: Critical

- [ ] **SEC-007:** User data isolation
  - Input: User A requests User B's data
  - Expected: 403 Forbidden
  - Test Type: Security
  - Priority: Critical

- [ ] **SEC-008:** HTTPS enforced
  - Input: HTTP request
  - Expected: Redirected to HTTPS
  - Test Type: Security
  - Priority: High

- [ ] **SEC-009:** Sensitive data in logs
  - Input: Check application logs
  - Expected: No passwords, tokens in logs
  - Test Type: Security
  - Priority: High

---

## 5. Performance Testing

### 5.1 Sync Performance

#### Test Cases
- [ ] **PERF-001:** Initial sync latency
  - Dataset: 100 connections, 50 tabs, 1000 history
  - Expected: <2 seconds
  - Test Type: Performance
  - Priority: High

- [ ] **PERF-002:** Incremental sync latency
  - Dataset: 10 changed records
  - Expected: <500ms (p95)
  - Test Type: Performance
  - Priority: High

- [ ] **PERF-003:** Large tab content sync
  - Dataset: 1MB query content
  - Expected: <1 second
  - Test Type: Performance
  - Priority: Medium

- [ ] **PERF-004:** Concurrent sync operations
  - Load: 100 concurrent users syncing
  - Expected: No rate limiting, <1s latency
  - Test Type: Performance
  - Priority: Medium

### 5.2 Database Performance

#### Test Cases
- [ ] **PERF-005:** Turso query latency
  - Query: Get user's tabs
  - Expected: <50ms (p95)
  - Test Type: Performance
  - Priority: High

- [ ] **PERF-006:** IndexedDB read latency
  - Query: Get all connections
  - Expected: <20ms
  - Test Type: Performance
  - Priority: Medium

- [ ] **PERF-007:** Batch insert performance
  - Dataset: 100 records
  - Expected: <500ms
  - Test Type: Performance
  - Priority: Medium

---

## 6. Integration Testing

### 6.1 End-to-End User Flows

#### Test Cases
- [ ] **E2E-001:** New user onboarding flow
  - Steps: Register → Verify email → Login → Create connection → Create tab → Enable sync → Subscribe
  - Expected: Complete flow successful
  - Test Type: E2E
  - Priority: Critical

- [ ] **E2E-002:** Multi-device sync flow
  - Steps: Login Device A → Create data → Login Device B → Verify data present
  - Expected: Data synced across devices
  - Test Type: E2E
  - Priority: Critical

- [ ] **E2E-003:** Offline → Online flow
  - Steps: Go offline → Make changes → Go online → Verify sync
  - Expected: Changes synced after reconnect
  - Test Type: E2E
  - Priority: Critical

- [ ] **E2E-004:** Subscription lifecycle
  - Steps: Subscribe → Use for month → Renew → Cancel → Downgrade
  - Expected: All transitions smooth
  - Test Type: E2E
  - Priority: High

---

## 7. Compatibility Testing

### 7.1 Browser Compatibility

#### Test Cases
- [ ] **COMPAT-001:** Chrome (latest)
  - Features: Auth, sync, payments
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: Critical

- [ ] **COMPAT-002:** Firefox (latest)
  - Features: Auth, sync, payments
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: High

- [ ] **COMPAT-003:** Safari (latest)
  - Features: Auth, sync, payments
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: High

- [ ] **COMPAT-004:** Edge (latest)
  - Features: Auth, sync, payments
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: Medium

### 7.2 Platform Compatibility

#### Test Cases
- [ ] **COMPAT-005:** macOS desktop app
  - Features: Full feature set
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: Critical

- [ ] **COMPAT-006:** Windows desktop app
  - Features: Full feature set
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: High

- [ ] **COMPAT-007:** Linux desktop app
  - Features: Full feature set
  - Expected: All features work
  - Test Type: Compatibility
  - Priority: Medium

---

## 8. User Acceptance Testing (UAT)

### 8.1 Beta User Scenarios

#### Test Cases
- [ ] **UAT-001:** Beta user signs up
  - Scenario: Real user registration
  - Expected: Smooth, no confusion
  - Test Type: UAT
  - Priority: Critical

- [ ] **UAT-002:** Beta user enables sync
  - Scenario: First-time sync setup
  - Expected: Clear instructions, works
  - Test Type: UAT
  - Priority: Critical

- [ ] **UAT-003:** Beta user subscribes
  - Scenario: Trial → paid conversion
  - Expected: Clear value, easy checkout
  - Test Type: UAT
  - Priority: Critical

- [ ] **UAT-004:** Beta user reports issue
  - Scenario: User submits feedback
  - Expected: Feedback received, acknowledged
  - Test Type: UAT
  - Priority: High

---

## Test Execution Schedule

### Week 5-6: Unit & Integration Tests
- Auth unit tests
- Sync unit tests
- Repository tests
- API integration tests

### Week 7-8: Feature Tests
- Sync feature tests
- Conflict resolution tests
- Offline queue tests

### Week 9: Performance Tests
- Load testing
- Latency testing
- Stress testing

### Week 10: Security & Compatibility
- Security audit
- OWASP testing
- Browser compatibility
- Platform compatibility

### Week 11: Payment Tests
- Stripe integration tests
- Webhook tests
- Subscription lifecycle

### Week 12: UAT & E2E
- Beta user testing
- End-to-end flows
- Regression testing
- Final QA sign-off

---

## Test Metrics

### Coverage Targets
- Unit Test Coverage: >80%
- Integration Test Coverage: >70%
- E2E Test Coverage: 100% of critical flows
- Security Test Coverage: OWASP Top 10

### Performance Targets
- Sync latency (p95): <500ms
- Auth latency (p95): <200ms
- Payment latency (p95): <2s
- App load time: <3s

### Quality Gates
- Zero critical bugs
- <5 high priority bugs
- <10 medium priority bugs
- All E2E tests passing
- Security audit passed

---

## Test Tools

### Frameworks
- **Unit Tests:** Vitest (frontend), Go testing (backend)
- **Integration Tests:** Vitest + Supertest
- **E2E Tests:** Playwright
- **Load Tests:** k6
- **Security Tests:** OWASP ZAP

### Services
- **CI/CD:** GitHub Actions
- **Test Reporting:** Allure
- **Coverage:** Codecov
- **Monitoring:** Sentry (error tracking)

---

## Sign-Off Checklist

### Phase 2 Testing Complete
- [ ] All critical tests passing
- [ ] All high priority tests passing
- [ ] >80% unit test coverage
- [ ] All E2E flows working
- [ ] Security audit passed
- [ ] Performance targets met
- [ ] UAT feedback positive (>4.0/5)
- [ ] Zero critical bugs
- [ ] QA Lead sign-off
- [ ] Product Manager sign-off
- [ ] Tech Lead sign-off

**QA Lead:** _________________ Date: _______
**Product Manager:** _________________ Date: _______
**Tech Lead:** _________________ Date: _______

---

**Document Version:** 1.0
**Last Updated:** 2025-10-23
**Next Review:** Weekly during Phase 2
**Status:** Active
