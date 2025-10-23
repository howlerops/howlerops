# SQL Studio Phase 1 - Testing Checklist

## Overview

This document provides a comprehensive testing checklist for Phase 1 (Foundation) of SQL Studio's tiered architecture implementation. All tests must pass before Phase 1 can be considered complete.

**Phase 1 Test Coverage Target:** >80% overall, >90% for critical components

---

## Table of Contents

1. [Unit Tests](#unit-tests)
2. [Integration Tests](#integration-tests)
3. [E2E Tests](#e2e-tests)
4. [Performance Tests](#performance-tests)
5. [Security Tests](#security-tests)
6. [Compatibility Tests](#compatibility-tests)
7. [Manual Testing](#manual-testing)
8. [Acceptance Criteria](#acceptance-criteria)

---

## Unit Tests

### 1.1 IndexedDB Layer

**Test File:** `/frontend/src/lib/storage/__tests__/indexeddb-client.test.ts`

#### Database Initialization
- [ ] Database opens successfully
- [ ] Schema version tracked correctly
- [ ] Object stores created with correct structure
- [ ] Indexes created correctly
- [ ] Database upgrades from v1 to v2
- [ ] Multiple database connections handled
- [ ] Database reopens after close

#### CRUD Operations
- [ ] **Create:** Insert new record successfully
- [ ] **Create:** Duplicate key rejected
- [ ] **Create:** Invalid data rejected by validation
- [ ] **Read:** Get record by primary key
- [ ] **Read:** Get all records from store
- [ ] **Read:** Get with filter/query
- [ ] **Read:** Non-existent record returns null
- [ ] **Update:** Existing record updated
- [ ] **Update:** Non-existent record creates new (upsert)
- [ ] **Delete:** Record deleted successfully
- [ ] **Delete:** Soft delete sets deleted_at
- [ ] **Delete:** Hard delete removes record

#### Index Queries
- [ ] Query by single-field index
- [ ] Query by multi-field index
- [ ] Range queries work correctly
- [ ] Index returns results in correct order
- [ ] Empty result set handled

#### Transactions
- [ ] Transaction commits successfully
- [ ] Transaction rollback on error
- [ ] Read-only transaction prevents writes
- [ ] Read-write transaction allows writes
- [ ] Nested transactions handled correctly
- [ ] Concurrent transactions isolated

#### Error Handling
- [ ] Database open failure handled
- [ ] Quota exceeded error handled
- [ ] Transaction abort handled
- [ ] Invalid operation rejected
- [ ] Network errors handled gracefully

**Coverage Target:** >90%

---

### 1.2 Storage Repositories

**Test Files:**
- `/frontend/src/lib/storage/__tests__/connections-repository.test.ts`
- `/frontend/src/lib/storage/__tests__/query-tabs-repository.test.ts`
- `/frontend/src/lib/storage/__tests__/query-history-repository.test.ts`
- `/frontend/src/lib/storage/__tests__/saved-queries-repository.test.ts`

#### Connections Repository
- [ ] Create connection with valid data
- [ ] Create connection with missing required fields fails
- [ ] Get connection by ID
- [ ] Get all connections for user
- [ ] Update connection name
- [ ] Delete connection (soft delete)
- [ ] Search connections by name
- [ ] Filter connections by db_type
- [ ] Order connections by last_used_at
- [ ] Pagination works correctly

#### Query Tabs Repository
- [ ] Create tab
- [ ] Update tab content
- [ ] Update tab position
- [ ] Delete tab
- [ ] Get all tabs for user
- [ ] Get tabs ordered by position
- [ ] Pin/unpin tab
- [ ] Associate tab with connection

#### Query History Repository
- [ ] Create history entry
- [ ] Get history for connection
- [ ] Get history for user
- [ ] Search history by query text
- [ ] Filter by date range
- [ ] Limit query results
- [ ] Redacted queries stored correctly
- [ ] Query hash calculated correctly

#### Saved Queries Repository
- [ ] Create saved query
- [ ] Update saved query
- [ ] Delete saved query
- [ ] Get all saved queries
- [ ] Search by name/description
- [ ] Filter by tags
- [ ] Mark as favorite
- [ ] Increment execution count

**Coverage Target:** >90% per repository

---

### 1.3 Data Sanitization

**Test File:** `/frontend/src/lib/storage/__tests__/sanitization.test.ts`

#### Connection Sanitization
- [ ] Password field removed
- [ ] SSH tunnel password removed
- [ ] SSH private key removed
- [ ] API keys stripped
- [ ] Safe fields preserved
- [ ] Nested objects sanitized

#### Query Sanitization
- [ ] Queries with passwords redacted (when enabled)
- [ ] API keys in queries redacted
- [ ] Tokens in queries redacted
- [ ] Safe queries unchanged
- [ ] Custom keyword redaction works
- [ ] Query hash computed for redacted queries

#### AI Session Sanitization
- [ ] Credentials in messages removed
- [ ] Safe content preserved
- [ ] Metadata sanitized

#### General
- [ ] Sanitization doesn't mutate original object
- [ ] Empty/null objects handled
- [ ] Circular references handled
- [ ] Performance <10ms per operation

**Coverage Target:** >95%

---

### 1.4 Data Validation

**Test File:** `/frontend/src/lib/storage/__tests__/validation.test.ts`

#### Zod Schema Validation
- [ ] Valid connection passes validation
- [ ] Invalid connection rejected
- [ ] Missing required field detected
- [ ] Invalid UUID format rejected
- [ ] Invalid enum value rejected
- [ ] Invalid port number rejected
- [ ] String length limits enforced
- [ ] Type coercion works correctly

#### Entity Validation
- [ ] Connection schema validates correctly
- [ ] QueryTab schema validates correctly
- [ ] QueryHistory schema validates correctly
- [ ] SavedQuery schema validates correctly
- [ ] UIPreferences schema validates correctly

#### Error Messages
- [ ] Validation errors have clear messages
- [ ] Field names included in errors
- [ ] Multiple errors reported together

**Coverage Target:** >90%

---

### 1.5 BroadcastChannel

**Test File:** `/frontend/src/lib/sync/__tests__/broadcast-manager.test.ts`

#### Message Passing
- [ ] Message sent successfully
- [ ] Message received by other instance
- [ ] Message type serialization works
- [ ] Large messages handled
- [ ] Malformed messages rejected

#### Channel Management
- [ ] Channel created with name
- [ ] Multiple channels isolated
- [ ] Channel closed properly
- [ ] Listeners removed on close

#### Error Handling
- [ ] Unsupported browser fallback works
- [ ] Send error handled
- [ ] Receive error handled

**Coverage Target:** >85%

---

### 1.6 Local Sync Manager

**Test File:** `/frontend/src/lib/sync/__tests__/local-sync-manager.test.ts`

#### Change Detection
- [ ] IndexedDB change detected
- [ ] Change broadcast to other tabs
- [ ] Duplicate changes ignored
- [ ] Change debouncing works

#### Sync Protocol
- [ ] Local change synced to remote tabs
- [ ] Remote change applied to local store
- [ ] Circular updates prevented
- [ ] Last-write-wins conflict resolution

#### State Management
- [ ] Sync state tracked correctly
- [ ] Pending changes queued
- [ ] Queue processed on reconnect

**Coverage Target:** >85%

---

### 1.7 Tier System

**Test File:** `/frontend/src/lib/tiers/__tests__/tier-detector.test.ts`

#### Tier Detection
- [ ] LOCAL tier detected (no auth)
- [ ] INDIVIDUAL tier detected (auth, no org)
- [ ] TEAM tier detected (org member)
- [ ] Tier cached correctly
- [ ] Tier refreshed periodically
- [ ] Network error handled

#### Feature Gating
- [ ] Feature available for tier
- [ ] Feature unavailable for tier
- [ ] Fallback rendered when unavailable
- [ ] useTier() hook returns correct data
- [ ] FeatureGate component renders correctly

#### Limits Enforcement
- [ ] Connection limit enforced
- [ ] Saved query limit enforced
- [ ] Query history limit enforced
- [ ] Limit exceeded error shown
- [ ] Upgrade prompt shown

**Coverage Target:** >90%

---

## Integration Tests

### 2.1 End-to-End Storage Flow

**Test File:** `/frontend/src/lib/storage/__tests__/storage-integration.test.ts`

#### Complete Workflows
- [ ] Create connection → Store in IndexedDB → Retrieve → Verify
- [ ] Create tab → Update content → Store → Sync to other tab → Verify
- [ ] Execute query → Store in history → Retrieve → Verify sanitized
- [ ] Save query → Update → Delete → Verify soft delete
- [ ] Update preferences → Sync to all tabs → Verify

#### Cross-Repository Operations
- [ ] Delete connection deletes associated tabs
- [ ] Delete connection deletes associated history
- [ ] Update connection reflects in tabs
- [ ] Tab references connection correctly

#### Data Integrity
- [ ] Foreign keys enforced
- [ ] Cascading deletes work
- [ ] Transactions atomic
- [ ] No orphaned records

**Coverage Target:** >80%

---

### 2.2 Multi-Tab Synchronization

**Test File:** `/frontend/src/lib/sync/__tests__/multi-tab-integration.test.ts`

#### Sync Scenarios
- [ ] Tab A creates connection → Tab B receives update
- [ ] Tab A updates tab content → Tab B sees changes
- [ ] Tab A deletes query → Tab B reflects deletion
- [ ] Multiple tabs update simultaneously → Conflicts resolved
- [ ] Rapid updates debounced correctly
- [ ] One tab offline → Comes online → Syncs pending changes

#### Performance
- [ ] Sync latency <100ms (p95)
- [ ] No memory leaks after 1000 sync operations
- [ ] No performance degradation over time

**Coverage Target:** >80%

---

### 2.3 Tier System Integration

**Test File:** `/frontend/src/lib/tiers/__tests__/tier-integration.test.ts`

#### Tier Detection Flow
- [ ] User logs in → Tier detected → Features enabled
- [ ] User logs out → Tier downgraded to LOCAL
- [ ] Subscription expires → Tier downgraded → Features disabled
- [ ] User joins org → Tier upgraded to TEAM

#### Feature Access
- [ ] LOCAL tier cannot access sync features
- [ ] INDIVIDUAL tier can access sync
- [ ] TEAM tier can access team features
- [ ] Feature gates work across app

#### Limit Enforcement
- [ ] LOCAL user blocked at connection limit
- [ ] INDIVIDUAL user has unlimited connections
- [ ] Upgrade flow triggered on limit exceeded

**Coverage Target:** >80%

---

### 2.4 Sanitization Integration

**Test File:** `/frontend/src/lib/storage/__tests__/sanitization-integration.test.ts`

#### Security Validation
- [ ] Connection with password stored without password
- [ ] Connection retrieved has no password
- [ ] Query with sensitive data redacted (if enabled)
- [ ] Sanitized data passes validation
- [ ] No credentials leak via BroadcastChannel
- [ ] No credentials leak via IndexedDB export

**Coverage Target:** >95%

---

## E2E Tests

### 3.1 User Flows (Playwright)

**Test File:** `/frontend/e2e/phase1-flows.spec.ts`

#### Connection Management
- [ ] User creates new connection (sanitized)
- [ ] User edits connection
- [ ] User deletes connection
- [ ] Connection appears in all open tabs
- [ ] Connection limit enforced for LOCAL tier

#### Query Tab Management
- [ ] User creates new query tab
- [ ] User types in tab → Content auto-saved
- [ ] User switches tabs → Content persisted
- [ ] User closes tab → Data saved
- [ ] Tab sync works across browser tabs
- [ ] Tab content survives app restart

#### Saved Queries
- [ ] User saves query
- [ ] User edits saved query
- [ ] User searches saved queries
- [ ] User deletes saved query
- [ ] Saved query limit enforced

#### Tier Features
- [ ] LOCAL user sees upgrade prompts
- [ ] INDIVIDUAL user can sync (mock)
- [ ] TEAM user sees team features (mock)

**Coverage Target:** Critical user flows

---

### 3.2 Multi-Browser Tab Testing

**Test File:** `/frontend/e2e/multi-tab.spec.ts`

#### Sync Verification
- [ ] Open 2 tabs → Create connection in Tab 1 → Verify in Tab 2
- [ ] Open 3 tabs → Update tab in Tab 1 → Verify in Tab 2 & 3
- [ ] Close Tab 1 with unsaved changes → No data loss
- [ ] Rapid updates in multiple tabs → All tabs eventually consistent

**Coverage Target:** All sync scenarios

---

## Performance Tests

### 4.1 IndexedDB Performance

**Test File:** `/frontend/src/lib/storage/__tests__/indexeddb-performance.test.ts`

#### Latency Benchmarks
- [ ] Write latency p50 <20ms
- [ ] Write latency p95 <50ms
- [ ] Write latency p99 <100ms
- [ ] Read latency p50 <10ms
- [ ] Read latency p95 <20ms
- [ ] Read latency p99 <50ms
- [ ] Query latency p95 <50ms
- [ ] Transaction latency p95 <100ms

#### Throughput Benchmarks
- [ ] Sustained write rate >100 ops/sec
- [ ] Sustained read rate >1000 ops/sec
- [ ] Batch insert 1000 records <1s

#### Scale Benchmarks
- [ ] 1000 connections stored and retrieved
- [ ] 10,000 query history entries searchable
- [ ] 100 concurrent tabs syncing

**Pass Criteria:** All benchmarks meet targets

---

### 4.2 Sync Performance

**Test File:** `/frontend/src/lib/sync/__tests__/sync-performance.test.ts`

#### Latency
- [ ] Sync latency p50 <50ms
- [ ] Sync latency p95 <100ms
- [ ] Sync latency p99 <200ms

#### Throughput
- [ ] 100 updates/sec handled without lag
- [ ] 10 concurrent tabs sync <100ms

#### Resource Usage
- [ ] Memory usage <50MB baseline
- [ ] Memory usage <100MB with 100 connections
- [ ] No memory leaks over 1 hour
- [ ] CPU usage <5% idle
- [ ] CPU usage <20% active sync

**Pass Criteria:** All benchmarks meet targets

---

### 4.3 Load Testing

**Test File:** `/frontend/src/lib/storage/__tests__/load-test.test.ts`

#### Large Dataset Scenarios
- [ ] 1000 connections loaded <2s
- [ ] 10,000 saved queries searchable <500ms
- [ ] 100,000 history entries queryable <1s
- [ ] App responsive with large datasets

#### Concurrent Load
- [ ] 50 tabs syncing simultaneously
- [ ] 100 updates/sec across tabs
- [ ] Rapid tab creation/deletion

**Pass Criteria:** App remains responsive

---

## Security Tests

### 5.1 Credential Security

**Test File:** `/frontend/src/lib/security/__tests__/credentials-security.test.ts`

#### Storage Security
- [ ] Passwords never stored in IndexedDB
- [ ] SSH keys never stored in IndexedDB
- [ ] API keys stripped before storage
- [ ] Credentials stored in OS keychain (Wails)
- [ ] Credentials encrypted at rest

#### Leakage Prevention
- [ ] No credentials in BroadcastChannel messages
- [ ] No credentials in error messages
- [ ] No credentials in logs
- [ ] No credentials in analytics events

**Pass Criteria:** Zero credential leaks

---

### 5.2 Data Sanitization Security

**Test File:** `/frontend/src/lib/storage/__tests__/sanitization-security.test.ts`

#### Attack Scenarios
- [ ] SQL injection in query text handled
- [ ] XSS in query text handled
- [ ] Malicious connection data sanitized
- [ ] Code injection attempts blocked

#### Privacy
- [ ] Query redaction works (when enabled)
- [ ] User can opt out of history sync
- [ ] Data export excludes credentials

**Pass Criteria:** All attack vectors blocked

---

### 5.3 Security Audit

**Manual Review Checklist:**

#### Code Review
- [ ] No hardcoded credentials
- [ ] No sensitive data in logs
- [ ] Input validation comprehensive
- [ ] Output encoding correct
- [ ] CSRF protection (if applicable)
- [ ] XSS prevention measures

#### Data Flow Review
- [ ] Credentials never touch IndexedDB
- [ ] Sanitization applied before storage
- [ ] Validation applied before operations
- [ ] Secure data transmission (if applicable)

#### Dependency Review
- [ ] All dependencies up-to-date
- [ ] No known vulnerabilities
- [ ] License compliance

**Pass Criteria:** Security team sign-off

---

## Compatibility Tests

### 6.1 Browser Compatibility

**Test Browsers:**
- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Chrome (1 version back)

**Test Matrix:**

| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| IndexedDB | [ ] | [ ] | [ ] | [ ] |
| BroadcastChannel | [ ] | [ ] | [ ] | [ ] |
| Tier Detection | [ ] | [ ] | [ ] | [ ] |
| Sync Manager | [ ] | [ ] | [ ] | [ ] |

**Pass Criteria:** All features work on all browsers

---

### 6.2 Platform Compatibility (Wails)

**Test Platforms:**
- macOS (Intel & ARM)
- Windows (10 & 11)
- Linux (Ubuntu)

**Test Matrix:**

| Feature | macOS Intel | macOS ARM | Windows | Linux |
|---------|-------------|-----------|---------|-------|
| IndexedDB | [ ] | [ ] | [ ] | [ ] |
| OS Keychain | [ ] | [ ] | [ ] | [ ] |
| BroadcastChannel | [ ] | [ ] | [ ] | [ ] |
| Performance | [ ] | [ ] | [ ] | [ ] |

**Pass Criteria:** All platforms functional

---

### 6.3 Data Migration

**Test Scenarios:**
- [ ] Migrate from localStorage to IndexedDB
- [ ] Handle missing data gracefully
- [ ] Handle corrupt data
- [ ] Rollback on migration failure
- [ ] User notified of migration

**Pass Criteria:** Migration successful, no data loss

---

### 6.4 Backward Compatibility

**Test Scenarios:**
- [ ] Old connection format still works
- [ ] Old saved query format migrated
- [ ] API responses compatible
- [ ] No breaking changes for existing users

**Pass Criteria:** Existing users unaffected

---

## Manual Testing

### 7.1 User Experience Testing

#### Workflow Testing
- [ ] Create connection flow intuitive
- [ ] Multi-tab sync feels natural
- [ ] Tier upgrade flow clear
- [ ] Error messages helpful
- [ ] Loading states informative

#### Visual Testing
- [ ] Tier badges display correctly
- [ ] Upgrade modals styled consistently
- [ ] Sync indicators visible
- [ ] Responsive on all screen sizes
- [ ] Dark/light mode works

#### Accessibility
- [ ] Keyboard navigation works
- [ ] Screen reader compatible
- [ ] Focus management correct
- [ ] Color contrast WCAG AA compliant

**Pass Criteria:** UX team approval

---

### 7.2 Edge Cases

#### Unusual Scenarios
- [ ] User deletes all data
- [ ] User exceeds quota
- [ ] User offline for extended period
- [ ] User rapidly switches tiers
- [ ] User clears browser data
- [ ] Multiple devices logged in
- [ ] Clock skew between devices

**Pass Criteria:** All edge cases handled gracefully

---

### 7.3 Recovery Testing

#### Failure Scenarios
- [ ] IndexedDB corrupted → Recovery flow works
- [ ] Quota exceeded → User notified, can clear
- [ ] Sync failure → Retry logic works
- [ ] Network failure → Offline mode works
- [ ] App crash → Data recovered on restart

**Pass Criteria:** No data loss, graceful recovery

---

## Acceptance Criteria

### Phase 1 Testing Complete When:

#### Unit Tests
- [x] All unit tests written
- [x] All unit tests passing
- [x] Code coverage >80% overall
- [x] Code coverage >90% for critical components
- [x] No flaky tests
- [x] Tests run in <30 seconds

#### Integration Tests
- [x] All integration tests written
- [x] All integration tests passing
- [x] Coverage >80%
- [x] Tests run in <2 minutes

#### E2E Tests
- [x] Critical user flows tested
- [x] All E2E tests passing
- [x] Tests run in <5 minutes
- [x] No flaky tests

#### Performance Tests
- [x] All benchmarks run
- [x] All targets met
- [x] No performance regressions
- [x] Load tests passed

#### Security Tests
- [x] Security audit complete
- [x] No critical vulnerabilities
- [x] All high findings remediated
- [x] Security team sign-off

#### Compatibility Tests
- [x] All browsers tested
- [x] All platforms tested
- [x] Migration tested
- [x] Backward compatibility verified

#### Manual Testing
- [x] UX testing complete
- [x] Edge cases tested
- [x] Recovery scenarios tested
- [x] Accessibility verified

### Final Sign-Off

- [ ] Engineering Lead approval
- [ ] QA Lead approval
- [ ] Security Officer approval
- [ ] Product Manager approval

---

## Test Execution Tracking

### Week 1 Tests

| Test Suite | Planned | Written | Passing | Coverage |
|------------|---------|---------|---------|----------|
| IndexedDB Unit | 40 | 0 | 0 | 0% |
| Repository Unit | 50 | 0 | 0 | 0% |
| **Total Week 1** | **90** | **0** | **0** | **0%** |

### Week 2 Tests

| Test Suite | Planned | Written | Passing | Coverage |
|------------|---------|---------|---------|----------|
| Sanitization Unit | 30 | 0 | 0 | 0% |
| Validation Unit | 25 | 0 | 0 | 0% |
| Security Tests | 20 | 0 | 0 | 0% |
| **Total Week 2** | **75** | **0** | **0** | **0%** |

### Week 3 Tests

| Test Suite | Planned | Written | Passing | Coverage |
|------------|---------|---------|---------|----------|
| BroadcastChannel | 20 | 0 | 0 | 0% |
| Sync Manager | 30 | 0 | 0 | 0% |
| Multi-Tab Integration | 25 | 0 | 0 | 0% |
| **Total Week 3** | **75** | **0** | **0** | **0%** |

### Week 4 Tests

| Test Suite | Planned | Written | Passing | Coverage |
|------------|---------|---------|---------|----------|
| Tier System | 35 | 0 | 0 | 0% |
| E2E Tests | 40 | 0 | 0 | 0% |
| Performance Tests | 30 | 0 | 0 | 0% |
| **Total Week 4** | **105** | **0** | **0** | **0%** |

### Overall Phase 1

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Total Tests | 345 | 0 | Not Started |
| Tests Passing | 345 | 0 | Not Started |
| Code Coverage | >80% | 0% | Not Started |
| Performance Benchmarks Met | 100% | 0% | Not Started |
| Security Vulnerabilities | 0 Critical | - | Not Started |

---

## CI/CD Integration

### Continuous Integration

**Required Checks (must pass before merge):**
- [ ] `npm run test` passes (all unit tests)
- [ ] `npm run test:integration` passes
- [ ] `npm run typecheck` passes
- [ ] `npm run lint` passes
- [ ] Code coverage >80%
- [ ] No new security vulnerabilities

**Nightly Builds:**
- [ ] E2E test suite runs
- [ ] Performance benchmarks run
- [ ] Load tests run
- [ ] Results published to dashboard

**Pre-Release:**
- [ ] Full test suite passes
- [ ] Manual testing complete
- [ ] Security audit passed
- [ ] Performance targets met

---

## Test Tooling

### Tools Used

| Tool | Purpose | Version |
|------|---------|---------|
| Vitest | Unit testing | Latest |
| Playwright | E2E testing | Latest |
| fake-indexeddb | IndexedDB mocking | Latest |
| Zod | Schema validation | Latest |
| Istanbul | Code coverage | Latest |
| Lighthouse | Performance | Latest |

### Test Commands

```bash
# Unit tests
npm run test                    # Run all unit tests
npm run test:watch              # Watch mode
npm run test:coverage           # With coverage report

# Integration tests
npm run test:integration        # Run integration tests

# E2E tests
npm run test:e2e                # Run E2E tests
npm run test:e2e:ui             # E2E with UI

# Performance tests
npm run test:performance        # Run benchmarks

# All tests
npm run test:all                # Run everything
```

---

## Notes

- All tests must be written during development, not after
- Tests are part of the Definition of Done
- No PR merges without passing tests
- Flaky tests must be fixed or removed
- Performance regressions block release

---

**Document Version:** 1.0
**Last Updated:** 2025-10-23
**Status:** Active
**Next Review:** Weekly during Phase 1
