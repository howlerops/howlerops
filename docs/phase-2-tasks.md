# Phase 2: Individual Tier Backend Tasks (Weeks 5-12)

## Overview
Phase 2 builds the Individual Tier backend infrastructure, including authentication, Turso cloud sync, payment processing, and beta launch. This phase enables users to sync their data across devices and subscribe to the Individual tier.

**Timeline:** Weeks 5-12 (8 weeks)
**Status:** Not Started
**Overall Progress:** 0/40 tasks completed (0%)
**Budget Estimate:** $500-1000 (auth provider, Turso, Stripe setup)

---

## Week 5: Auth Foundation (Days 29-35)

### P2-T1: Auth Provider Selection & Setup
**Priority:** Critical
**Estimated Hours:** 8
**Assignee:** Backend Developer
**Dependencies:** Phase 1 Complete
**Status:** Not Started

**Description:**
Evaluate and set up authentication provider (Supabase vs Clerk vs Auth0).

**Decision Criteria:**
- Cost: Under $25/mo for <1000 users
- Features: Email/password, OAuth (GitHub, Google), JWT tokens
- Integration: Easy SDK for Wails Go backend
- Security: Industry-standard best practices

**Tasks:**
- Evaluate Supabase Auth (Recommended: $25/mo)
- Evaluate Clerk ($25/mo base)
- Evaluate Auth0 (free tier limited)
- Create decision matrix with pros/cons
- Set up chosen provider account
- Configure OAuth providers (GitHub, Google)
- Document setup process

**Acceptance Criteria:**
- [ ] Auth provider selected with documented rationale
- [ ] OAuth providers configured (GitHub, Google)
- [ ] Test accounts created
- [ ] API keys stored securely

**Technical Notes:**
```bash
# Recommended: Supabase
# - $25/mo for 100K MAU
# - Built-in Postgres database (bonus)
# - Simple JWT integration
# - Good Go SDK support
```

---

### P2-T2: User Registration Flow
**Priority:** Critical
**Estimated Hours:** 10
**Assignee:** Full Stack Developer
**Dependencies:** P2-T1
**Status:** Not Started

**Description:**
Implement user registration with email verification.

**Implementation Files:**
- `/backend/services/auth-service.go` - Auth service
- `/frontend/src/pages/auth/register.tsx` - Registration UI
- `/frontend/src/services/auth-api.ts` - Auth API client

**Tasks:**
- Create registration UI (email, password, name)
- Implement backend signup endpoint
- Add email verification flow
- Create verification email template
- Add password strength validation
- Implement rate limiting (5 attempts/hour)
- Add CAPTCHA for bot protection
- Handle duplicate email errors

**Acceptance Criteria:**
- [ ] User can register with email/password
- [ ] Email verification sent and works
- [ ] Password validation enforced (8+ chars, number, special char)
- [ ] Duplicate email shows friendly error
- [ ] Rate limiting prevents abuse
- [ ] UI shows loading/error states
- [ ] Mobile responsive

**API Endpoint:**
```go
// POST /api/auth/register
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name" binding:"required"`
}

type RegisterResponse struct {
    UserID           string `json:"user_id"`
    Email            string `json:"email"`
    VerificationSent bool   `json:"verification_sent"`
}
```

---

### P2-T3: Login & OAuth Flow
**Priority:** Critical
**Estimated Hours:** 12
**Assignee:** Full Stack Developer
**Dependencies:** P2-T2
**Status:** Not Started

**Description:**
Implement login with email/password and OAuth (GitHub, Google).

**Implementation Files:**
- `/frontend/src/pages/auth/login.tsx` - Login UI
- `/backend/services/auth-service.go` - Auth service (extend)
- `/frontend/src/lib/auth/oauth.ts` - OAuth helpers

**Tasks:**
- Create login UI
- Implement email/password login endpoint
- Add OAuth login buttons (GitHub, Google)
- Implement OAuth callback handling
- Generate and store JWT tokens
- Implement "Remember me" functionality
- Add "Forgot password" link
- Handle auth errors gracefully

**Acceptance Criteria:**
- [ ] Email/password login works
- [ ] GitHub OAuth login works
- [ ] Google OAuth login works
- [ ] JWT token stored securely (httpOnly cookie or secure storage)
- [ ] "Remember me" extends token TTL
- [ ] Error messages are user-friendly
- [ ] Loading states during auth

**API Endpoints:**
```go
// POST /api/auth/login
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    User         User   `json:"user"`
    ExpiresIn    int    `json:"expires_in"`
}

// GET /api/auth/oauth/github
// Redirects to GitHub OAuth

// GET /api/auth/oauth/callback
// Handles OAuth callback
```

---

### P2-T4: Session Management & Token Refresh
**Priority:** High
**Estimated Hours:** 8
**Assignee:** Backend Developer
**Dependencies:** P2-T3
**Status:** Not Started

**Description:**
Implement secure session management with token refresh.

**Tasks:**
- Implement JWT token generation
- Add refresh token rotation
- Create middleware for auth verification
- Implement token expiry (15min access, 7day refresh)
- Add automatic token refresh before expiry
- Implement logout (token revocation)
- Add session tracking in database

**Acceptance Criteria:**
- [ ] Access tokens expire after 15 minutes
- [ ] Refresh tokens work and rotate
- [ ] Middleware blocks unauthenticated requests
- [ ] Logout invalidates tokens
- [ ] Multiple sessions supported (desktop + web)
- [ ] Session list visible to user

**Implementation:**
```go
// Middleware
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        claims, err := validateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

---

### P2-T5: Password Reset Flow
**Priority:** Medium
**Estimated Hours:** 6
**Assignee:** Full Stack Developer
**Dependencies:** P2-T3
**Status:** Not Started

**Description:**
Implement secure password reset via email.

**Implementation Files:**
- `/frontend/src/pages/auth/forgot-password.tsx`
- `/frontend/src/pages/auth/reset-password.tsx`
- `/backend/services/auth-service.go` (extend)

**Tasks:**
- Create "Forgot Password" UI
- Implement reset email sending
- Create password reset email template
- Create password reset form
- Implement reset token validation (1-hour expiry)
- Update password in database
- Invalidate all existing sessions on reset

**Acceptance Criteria:**
- [ ] User receives reset email within 1 minute
- [ ] Reset link expires after 1 hour
- [ ] Reset link works only once
- [ ] Password successfully updated
- [ ] All sessions logged out after reset
- [ ] UI handles expired links gracefully

---

### P2-T6: Auth Testing & Security Audit
**Priority:** High
**Estimated Hours:** 6
**Assignee:** Security Specialist / QA
**Dependencies:** P2-T5
**Status:** Not Started

**Description:**
Comprehensive auth testing and security audit.

**Test Coverage:**
- Registration with valid/invalid inputs
- Login with correct/incorrect credentials
- OAuth flow (GitHub, Google)
- Token refresh
- Token expiry
- Password reset
- Rate limiting
- CSRF protection
- XSS prevention
- SQL injection attempts

**Tasks:**
- Write integration tests for all auth flows
- Test rate limiting effectiveness
- Verify JWT signature validation
- Test token expiry scenarios
- Attempt common attacks (SQL injection, XSS)
- Review OWASP Top 10 compliance
- Document security measures

**Acceptance Criteria:**
- [ ] All auth tests pass
- [ ] No critical security vulnerabilities
- [ ] Rate limiting prevents brute force
- [ ] Tokens properly validated
- [ ] OWASP Top 10 addressed
- [ ] Security audit document created

---

## Week 6: Turso Setup & Schema Migration (Days 36-42)

### P2-T7: Turso Database Provisioning
**Priority:** Critical
**Estimated Hours:** 4
**Assignee:** Backend Developer / DevOps
**Dependencies:** P2-T1
**Status:** Not Started

**Description:**
Set up Turso database with edge replication.

**Tasks:**
- Create Turso account
- Provision primary database (closest region)
- Enable edge replicas (3 locations)
- Generate database auth tokens
- Configure connection strings
- Set up database backup policy
- Document access credentials

**Acceptance Criteria:**
- [ ] Turso database created
- [ ] Edge replicas in 3 regions
- [ ] Auth tokens generated
- [ ] Connection successful from backend
- [ ] Backup policy configured

**CLI Commands:**
```bash
# Create database
turso db create sql-studio-sync --location iad

# Enable replicas
turso db replicate sql-studio-sync add lax
turso db replicate sql-studio-sync add lhr
turso db replicate sql-studio-sync add syd

# Show status
turso db show sql-studio-sync

# Generate auth token
turso db tokens create sql-studio-sync
```

---

### P2-T8: Turso Schema Implementation
**Priority:** Critical
**Estimated Hours:** 10
**Assignee:** Database Specialist
**Dependencies:** P2-T7
**Status:** Not Started

**Description:**
Implement Turso database schema based on design document.

**Implementation File:** `/backend/migrations/001_initial_schema.sql`

**Tables to Create:**
- schema_migrations
- users
- connections
- query_tabs
- query_history
- saved_queries
- ai_sessions
- ai_messages
- ui_preferences
- sync_conflicts
- devices

**Tasks:**
- Create migration file with full schema
- Add all indexes from design doc
- Create FTS5 virtual tables for search
- Test schema creation locally
- Apply migration to Turso
- Verify schema with turso db shell

**Acceptance Criteria:**
- [ ] All tables created successfully
- [ ] Indexes created on key columns
- [ ] FTS5 search indexes work
- [ ] Foreign keys enforced
- [ ] Schema matches design document
- [ ] Migration is idempotent

**Schema Verification:**
```sql
-- Verify tables
SELECT name FROM sqlite_master WHERE type='table';

-- Verify indexes
SELECT name FROM sqlite_master WHERE type='index';

-- Test query
SELECT * FROM users LIMIT 1;
```

---

### P2-T9: Turso Connection Library
**Priority:** Critical
**Estimated Hours:** 8
**Assignee:** Backend Developer
**Dependencies:** P2-T8
**Status:** Not Started

**Description:**
Create Go library for Turso database access.

**Implementation File:** `/backend/pkg/turso/client.go`

**Features:**
- Connection pooling
- Query execution
- Batch operations
- Transaction support
- Error handling
- Retry logic
- Metrics/logging

**Tasks:**
- Install libSQL Go driver
- Create TursoClient wrapper
- Implement connection pooling
- Add query methods (Execute, Query, Batch)
- Add transaction support
- Implement retry with exponential backoff
- Add logging for all queries
- Create helper methods

**Acceptance Criteria:**
- [ ] Connection established successfully
- [ ] CRUD operations work
- [ ] Batch inserts perform well
- [ ] Transactions roll back on error
- [ ] Retry logic handles transient errors
- [ ] Query logging enabled
- [ ] Unit tests for all methods

**Example Usage:**
```go
client, err := turso.NewClient(turso.Config{
    URL:       os.Getenv("TURSO_URL"),
    AuthToken: os.Getenv("TURSO_TOKEN"),
})

// Simple query
rows, err := client.Query(ctx, "SELECT * FROM users WHERE user_id = ?", userID)

// Batch insert
err = client.Batch(ctx, []turso.Statement{
    {SQL: "INSERT INTO query_tabs (...) VALUES (?)", Args: []any{...}},
    {SQL: "INSERT INTO query_tabs (...) VALUES (?)", Args: []any{...}},
})
```

---

### P2-T10: Data Migration Tools
**Priority:** Medium
**Estimated Hours:** 8
**Assignee:** Backend Developer
**Dependencies:** P2-T9
**Status:** Not Started

**Description:**
Build tools to migrate user data from local IndexedDB to Turso.

**Implementation Files:**
- `/frontend/src/lib/migration/local-to-turso.ts`
- `/backend/services/migration-service.go`

**Features:**
- Export data from IndexedDB
- Upload to Turso via API
- Progress tracking
- Error recovery
- Duplicate detection
- Rollback capability

**Tasks:**
- Create export function for IndexedDB
- Create import API endpoint
- Implement batch upload (50 records at a time)
- Add progress bar UI
- Handle errors gracefully
- Add "Undo migration" option
- Test with large datasets (1000+ records)

**Acceptance Criteria:**
- [ ] Migration exports all local data
- [ ] Migration uploads to Turso successfully
- [ ] Progress visible to user
- [ ] Errors don't stop entire migration
- [ ] Duplicate records handled (skip or update)
- [ ] Migration tested with 1000+ records
- [ ] Rollback works if needed

**Migration Flow:**
```
1. User clicks "Enable Cloud Sync"
2. Export connections, tabs, history, queries from IndexedDB
3. Send to backend API in batches
4. Backend inserts into Turso
5. Show progress: "Migrating... 150/200 records"
6. Complete: "Migration successful! Sync enabled."
```

---

### P2-T11: Schema Versioning System
**Priority:** Medium
**Estimated Hours:** 6
**Assignee:** Backend Developer
**Dependencies:** P2-T8
**Status:** Not Started

**Description:**
Implement database migration system for schema changes.

**Implementation File:** `/backend/pkg/migrations/manager.go`

**Tasks:**
- Create migration file structure
- Implement up/down migrations
- Add migration tracking table
- Create CLI command for migrations
- Add automatic migration on startup
- Document migration process

**Acceptance Criteria:**
- [ ] Migrations run in order
- [ ] Migration state tracked
- [ ] Rollback works (down migrations)
- [ ] Idempotent (safe to re-run)
- [ ] CLI command for manual migration
- [ ] Auto-migration on app startup

---

## Week 7: Upload Sync (Client to Cloud) (Days 43-49)

### P2-T12: Sync Manager Architecture
**Priority:** Critical
**Estimated Hours:** 10
**Assignee:** Senior Developer
**Dependencies:** P2-T9
**Status:** Not Started

**Description:**
Design and implement core sync manager architecture.

**Implementation File:** `/frontend/src/lib/sync/sync-manager.ts`

**Features:**
- Push changes to Turso
- Pull changes from Turso
- Offline queue
- Conflict detection
- Debouncing
- Background sync

**Tasks:**
- Design sync manager class structure
- Implement SyncVector for tracking state
- Create offline queue with persistence
- Add debounce for frequent updates
- Implement push method (local -> cloud)
- Add sync status observables
- Create sync configuration

**Acceptance Criteria:**
- [ ] Singleton pattern for sync manager
- [ ] Offline queue persists across restarts
- [ ] Debouncing reduces unnecessary syncs
- [ ] Sync status updates in real-time
- [ ] Gracefully handles network errors
- [ ] Unit tests for core logic

**Architecture:**
```typescript
class SyncManager {
  private syncVector: SyncVector
  private offlineQueue: PendingChange[]
  private syncInterval: number

  async initialize(userId: string)
  async pushChange(change: LocalChange)
  async pullChanges(since?: string)
  async syncNow()
  onSyncStatusChange(callback: (status: SyncStatus) => void)
}
```

---

### P2-T13: Connection Sync Implementation
**Priority:** High
**Estimated Hours:** 8
**Assignee:** Full Stack Developer
**Dependencies:** P2-T12
**Status:** Not Started

**Description:**
Implement sync for database connections.

**Tasks:**
- Detect connection changes (create, update, delete)
- Push connection to Turso (without credentials)
- Handle connection updates
- Implement soft delete for connections
- Add sync status indicator in UI
- Test cross-device sync

**Acceptance Criteria:**
- [ ] New connections sync to cloud
- [ ] Updates sync to cloud
- [ ] Deletes sync to cloud (soft delete)
- [ ] Credentials never synced
- [ ] Sync status visible in UI
- [ ] Cross-device sync tested

**Security Check:**
```typescript
// Sanitize before sync
function sanitizeConnection(conn: Connection): SyncableConnection {
  return {
    ...conn,
    password: undefined,       // NEVER sync
    sshPassword: undefined,    // NEVER sync
    sshPrivateKey: undefined,  // NEVER sync
  }
}
```

---

### P2-T14: Query Tab Sync Implementation
**Priority:** High
**Estimated Hours:** 10
**Assignee:** Full Stack Developer
**Dependencies:** P2-T12
**Status:** Not Started

**Description:**
Implement sync for query tabs with debouncing.

**Tasks:**
- Detect tab changes (create, update, delete, reorder)
- Debounce content changes (2 second delay)
- Push tab state to Turso
- Handle tab reordering
- Sync tab positions
- Test rapid typing scenario

**Acceptance Criteria:**
- [ ] Tab creation syncs immediately
- [ ] Tab content debounced (2s)
- [ ] Tab deletion syncs immediately
- [ ] Tab reordering syncs
- [ ] Rapid typing doesn't overwhelm sync
- [ ] Sync indicator shows pending state

**Debouncing Implementation:**
```typescript
const debouncedTabSync = debounce(async (tabId: string, content: string) => {
  await syncManager.pushChange({
    entityType: 'query_tabs',
    entityId: tabId,
    data: { content, updated_at: new Date().toISOString() },
  })
}, 2000)
```

---

### P2-T15: Query History Sync
**Priority:** Medium
**Estimated Hours:** 6
**Assignee:** Frontend Developer
**Dependencies:** P2-T12
**Status:** Not Started

**Description:**
Implement sync for query execution history.

**Tasks:**
- Push executed queries to Turso
- Implement optional redaction (privacy feature)
- Batch history uploads (every 10 queries)
- Add privacy toggle in settings
- Test with high query volume

**Acceptance Criteria:**
- [ ] Query history syncs to cloud
- [ ] Redaction option works
- [ ] Batching reduces sync overhead
- [ ] Privacy setting respected
- [ ] High volume handled gracefully

**Privacy Implementation:**
```typescript
function shouldSyncHistory(settings: PrivacySettings): boolean {
  return settings.syncQueryHistory
}

function redactQuery(query: string, keywords: string[]): string {
  if (!settings.redactSensitiveQueries) return query
  return redactSensitiveData(query, keywords)
}
```

---

### P2-T16: Saved Queries Sync
**Priority:** Medium
**Estimated Hours:** 6
**Assignee:** Frontend Developer
**Dependencies:** P2-T12
**Status:** Not Started

**Description:**
Implement sync for user's saved query library.

**Tasks:**
- Push saved queries to Turso
- Handle updates to saved queries
- Sync query tags
- Sync favorite status
- Test with large libraries (100+ queries)

**Acceptance Criteria:**
- [ ] New saved queries sync
- [ ] Updates to queries sync
- [ ] Tags sync correctly
- [ ] Favorite status syncs
- [ ] Large libraries sync efficiently

---

### P2-T17: Offline Queue Implementation
**Priority:** High
**Estimated Hours:** 8
**Assignee:** Senior Developer
**Dependencies:** P2-T12
**Status:** Not Started

**Description:**
Robust offline queue with retry logic.

**Tasks:**
- Create persistent queue (IndexedDB)
- Implement retry with exponential backoff
- Add max retry limit (5 attempts)
- Detect online/offline status
- Flush queue when back online
- Show offline indicator in UI
- Add manual "Retry Now" button

**Acceptance Criteria:**
- [ ] Queue persists across app restarts
- [ ] Retries with exponential backoff
- [ ] Max retries respected
- [ ] Online status detection works
- [ ] Queue flushes automatically when online
- [ ] UI shows offline state
- [ ] Manual retry works

**Retry Logic:**
```typescript
async function retryWithBackoff(
  fn: () => Promise<void>,
  maxRetries: number = 5
): Promise<void> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      await fn()
      return
    } catch (error) {
      if (i === maxRetries - 1) throw error
      const delay = Math.pow(2, i) * 1000
      await sleep(delay)
    }
  }
}
```

---

## Week 8: Download Sync (Cloud to Client) (Days 50-56)

### P2-T18: Initial Sync on Login
**Priority:** Critical
**Estimated Hours:** 10
**Assignee:** Full Stack Developer
**Dependencies:** P2-T17
**Status:** Not Started

**Description:**
Implement full data sync when user logs in.

**Tasks:**
- Fetch all user data from Turso on login
- Merge with local data (prefer cloud)
- Show sync progress UI
- Handle large datasets (paginate if needed)
- Cache sync timestamp
- Test with 1000+ records

**Acceptance Criteria:**
- [ ] Full sync completes on login
- [ ] Local data merged with cloud
- [ ] Progress shown to user
- [ ] Large datasets handled efficiently
- [ ] Sync timestamp cached
- [ ] Works with 1000+ records

**Sync Flow:**
```
1. User logs in
2. Show: "Syncing your data..."
3. Fetch connections, tabs, history, queries from Turso
4. Merge with local IndexedDB (cloud wins by default)
5. Update local sync vector
6. Show: "Sync complete! ✓"
7. Enable real-time sync
```

---

### P2-T19: Incremental Sync (Pull Changes)
**Priority:** High
**Estimated Hours:** 8
**Assignee:** Backend Developer
**Dependencies:** P2-T18
**Status:** Not Started

**Description:**
Implement efficient incremental sync pulling only changes.

**Tasks:**
- Query Turso for changes since last sync
- Use updated_at timestamps for filtering
- Implement cursor-based pagination
- Apply changes to local IndexedDB
- Update sync vector
- Test with concurrent changes

**Acceptance Criteria:**
- [ ] Only changed records fetched
- [ ] Pagination works for large change sets
- [ ] Local database updated correctly
- [ ] Sync vector updated
- [ ] Concurrent changes handled
- [ ] Performance acceptable (<2s for 100 changes)

**API Endpoint:**
```go
// GET /api/sync/changes?since=2024-10-23T10:00:00Z
type SyncChangesResponse struct {
    Connections   []Connection   `json:"connections"`
    QueryTabs     []QueryTab     `json:"query_tabs"`
    QueryHistory  []QueryHistory `json:"query_history"`
    SavedQueries  []SavedQuery   `json:"saved_queries"`
    Timestamp     time.Time      `json:"timestamp"`
    HasMore       bool           `json:"has_more"`
}
```

---

### P2-T20: Conflict Detection Logic
**Priority:** High
**Estimated Hours:** 10
**Assignee:** Senior Developer
**Dependencies:** P2-T19
**Status:** Not Started

**Description:**
Detect conflicts when local and cloud versions differ.

**Tasks:**
- Compare updated_at timestamps
- Detect concurrent modifications
- Identify conflicting fields
- Log conflicts to sync_conflicts table
- Notify user of conflicts
- Provide conflict resolution UI

**Acceptance Criteria:**
- [ ] Conflicts detected accurately
- [ ] Timestamps compared correctly
- [ ] Conflicts logged to database
- [ ] User notified of conflicts
- [ ] Conflict details visible in UI
- [ ] Unit tests for conflict scenarios

**Conflict Detection:**
```typescript
function detectConflict(
  local: QueryTab,
  remote: QueryTab
): Conflict | null {
  // No conflict if timestamps match
  if (local.updated_at === remote.updated_at) return null

  // Conflict if both modified since last sync
  if (local.updated_at > lastSync && remote.updated_at > lastSync) {
    return {
      entityType: 'query_tabs',
      entityId: local.tab_id,
      localVersion: local,
      remoteVersion: remote,
      conflictingFields: diffFields(local, remote),
    }
  }

  return null
}
```

---

### P2-T21: Conflict Resolution Strategy
**Priority:** High
**Estimated Hours:** 8
**Assignee:** Senior Developer
**Dependencies:** P2-T20
**Status:** Not Started

**Description:**
Implement Last-Write-Wins (LWW) conflict resolution.

**Tasks:**
- Implement LWW algorithm
- Compare timestamps for winner
- Apply winning version
- Log resolution to database
- Notify user of auto-resolution
- Provide manual resolution option

**Acceptance Criteria:**
- [ ] LWW algorithm works correctly
- [ ] Newer version always wins
- [ ] Resolution logged
- [ ] User notified
- [ ] Manual override available
- [ ] Works for all entity types

**LWW Implementation:**
```typescript
function resolveConflict(conflict: Conflict): ResolvedChange {
  const { localVersion, remoteVersion } = conflict

  // Last Write Wins
  const winner = localVersion.updated_at > remoteVersion.updated_at
    ? localVersion
    : remoteVersion

  const resolution = localVersion === winner ? 'local_wins' : 'remote_wins'

  // Log conflict
  logConflictResolution(conflict, resolution)

  return {
    entityType: conflict.entityType,
    entityId: conflict.entityId,
    data: winner,
    resolution,
  }
}
```

---

### P2-T22: Multi-Device Sync Testing
**Priority:** High
**Estimated Hours:** 6
**Assignee:** QA Engineer
**Dependencies:** P2-T21
**Status:** Not Started

**Description:**
Test sync across multiple devices/browsers.

**Test Scenarios:**
- Create connection on Device A, see on Device B
- Update tab on Device A, see on Device B
- Delete query on Device A, see on Device B
- Offline on Device A, modify on Device B, sync when A online
- Concurrent modification on both devices
- Large dataset sync

**Tasks:**
- Set up test environment (2+ browsers)
- Test all sync scenarios
- Verify sync latency (<5 seconds)
- Test conflict resolution
- Test with poor network
- Document findings

**Acceptance Criteria:**
- [ ] All scenarios tested
- [ ] Sync latency acceptable
- [ ] Conflicts resolved correctly
- [ ] Poor network handled
- [ ] Test report created
- [ ] Bugs filed for issues

---

## Week 9: Background Sync & Optimization (Days 57-63)

### P2-T23: Background Sync Worker
**Priority:** Medium
**Estimated Hours:** 8
**Assignee:** Frontend Developer
**Dependencies:** P2-T19
**Status:** Not Started

**Description:**
Implement background sync for non-critical data.

**Tasks:**
- Create background sync worker
- Sync query history every 5 minutes
- Sync AI sessions every 5 minutes
- Sync UI preferences on change
- Respect battery/network constraints
- Add sync scheduler

**Acceptance Criteria:**
- [ ] Background worker runs independently
- [ ] Non-critical data syncs periodically
- [ ] Respects network conditions
- [ ] Doesn't drain battery
- [ ] Can be paused/resumed
- [ ] Logs background activity

**Implementation:**
```typescript
class BackgroundSyncWorker {
  private intervalId: NodeJS.Timeout | null = null

  start() {
    this.intervalId = setInterval(async () => {
      if (!navigator.onLine) return
      if (isBatteryLow()) return

      await this.syncNonCriticalData()
    }, 5 * 60 * 1000) // 5 minutes
  }

  private async syncNonCriticalData() {
    await Promise.all([
      syncManager.syncQueryHistory(),
      syncManager.syncAISessions(),
      syncManager.syncPreferences(),
    ])
  }
}
```

---

### P2-T24: Sync Performance Optimization
**Priority:** High
**Estimated Hours:** 8
**Assignee:** Performance Engineer
**Dependencies:** P2-T23
**Status:** Not Started

**Description:**
Optimize sync performance for speed and efficiency.

**Tasks:**
- Profile sync operations
- Optimize batch sizes (test 10, 50, 100)
- Reduce payload sizes (compression)
- Cache frequently accessed data
- Lazy load non-critical data
- Benchmark improvements

**Acceptance Criteria:**
- [ ] Sync latency <500ms for incremental
- [ ] Full sync <2s for typical user
- [ ] Payload sizes reduced >30%
- [ ] Network requests reduced >50%
- [ ] Benchmarks documented
- [ ] Performance regression tests

**Optimizations:**
- Batch operations (50 records per request)
- Compress large text fields (gzip)
- Debounce frequent updates (2s)
- Use IndexedDB transactions
- Cache Turso responses (5 minutes)
- Lazy load query history

---

### P2-T25: Sync UI Polish
**Priority:** Medium
**Estimated Hours:** 6
**Assignee:** Frontend Developer / UI Designer
**Dependencies:** P2-T24
**Status:** Not Started

**Description:**
Polish sync UI/UX for better user experience.

**Tasks:**
- Add sync status indicator (always visible)
- Show sync progress for initial sync
- Add "Last synced" timestamp
- Create sync settings panel
- Add manual "Sync Now" button
- Show sync errors gracefully
- Add sync activity log

**Acceptance Criteria:**
- [ ] Sync status always visible
- [ ] Progress shown during sync
- [ ] Last sync time displayed
- [ ] Settings panel intuitive
- [ ] Manual sync works
- [ ] Errors shown gracefully
- [ ] Activity log helpful

**UI Components:**
- Sync indicator icon (header)
- Sync progress modal (initial sync)
- Sync settings page
- Sync activity drawer
- Error toast notifications

---

### P2-T26: Sync Metrics & Monitoring
**Priority:** Medium
**Estimated Hours:** 6
**Assignee:** DevOps Engineer
**Dependencies:** P2-T25
**Status:** Not Started

**Description:**
Add monitoring and metrics for sync operations.

**Tasks:**
- Track sync success/failure rate
- Measure sync latency (p50, p95, p99)
- Count sync operations per user
- Monitor offline queue size
- Track conflict rate
- Set up alerts for anomalies

**Acceptance Criteria:**
- [ ] Metrics collected for all sync ops
- [ ] Latency percentiles tracked
- [ ] Dashboards created
- [ ] Alerts configured
- [ ] Metrics exportable
- [ ] Historical data retained

**Metrics to Track:**
- sync_duration_ms
- sync_records_count
- sync_errors_total
- offline_queue_size
- conflict_rate
- turso_request_latency

---

## Week 10: Testing & Polish (Days 64-70)

### P2-T27: End-to-End Sync Testing
**Priority:** Critical
**Estimated Hours:** 10
**Assignee:** QA Engineer
**Dependencies:** P2-T26
**Status:** Not Started

**Description:**
Comprehensive E2E testing of sync flows.

**Test Suites:**
1. Happy path sync
2. Offline/online transitions
3. Conflict scenarios
4. Large dataset sync
5. Performance tests
6. Security tests
7. Edge cases

**Tasks:**
- Write Playwright E2E tests
- Test all sync entity types
- Test offline queue
- Test conflict resolution
- Test multi-device sync
- Test performance under load
- Generate test report

**Acceptance Criteria:**
- [ ] All E2E tests pass
- [ ] Coverage >80% of sync flows
- [ ] Performance tests pass
- [ ] Security tests pass
- [ ] Test report generated
- [ ] CI/CD integration complete

---

### P2-T28: Sync Error Recovery
**Priority:** High
**Estimated Hours:** 6
**Assignee:** Backend Developer
**Dependencies:** P2-T27
**Status:** Not Started

**Description:**
Robust error handling and recovery for sync.

**Tasks:**
- Handle network timeouts
- Handle Turso rate limits
- Handle auth token expiry
- Handle quota exceeded
- Retry failed operations
- Provide user feedback
- Add debug mode

**Acceptance Criteria:**
- [ ] All error types handled
- [ ] Retries work correctly
- [ ] User notified of errors
- [ ] Recovery automatic when possible
- [ ] Debug logs available
- [ ] Error tracking integrated

---

### P2-T29: Data Integrity Validation
**Priority:** High
**Estimated Hours:** 6
**Assignee:** QA Engineer
**Dependencies:** P2-T28
**Status:** Not Started

**Description:**
Ensure data integrity across sync operations.

**Tasks:**
- Verify data consistency (local vs cloud)
- Test referential integrity
- Test foreign key constraints
- Verify no data loss scenarios
- Test rollback on error
- Add integrity check command

**Acceptance Criteria:**
- [ ] Data consistency verified
- [ ] No data loss in any scenario
- [ ] Referential integrity maintained
- [ ] Constraints enforced
- [ ] Rollback works correctly
- [ ] Integrity check tool created

---

### P2-T30: Sync Documentation
**Priority:** Medium
**Estimated Hours:** 4
**Assignee:** Technical Writer / Developer
**Dependencies:** P2-T29
**Status:** Not Started

**Description:**
Document sync architecture and usage.

**Documents:**
- Sync architecture diagram
- API documentation
- User guide (how sync works)
- Troubleshooting guide
- Developer guide (extending sync)

**Acceptance Criteria:**
- [ ] Architecture documented
- [ ] API documented
- [ ] User guide complete
- [ ] Troubleshooting guide helpful
- [ ] Developer guide clear

---

## Week 11: Payment Integration (Days 71-77)

### P2-T31: Stripe Account Setup
**Priority:** Critical
**Estimated Hours:** 4
**Assignee:** Backend Developer / Finance
**Dependencies:** P2-T1
**Status:** Not Started

**Description:**
Set up Stripe account for payment processing.

**Tasks:**
- Create Stripe account
- Complete business verification
- Configure tax settings
- Set up bank account
- Generate API keys
- Enable webhook endpoints
- Test mode setup

**Acceptance Criteria:**
- [ ] Stripe account verified
- [ ] API keys generated
- [ ] Webhooks configured
- [ ] Test mode works
- [ ] Production mode ready
- [ ] Documentation updated

**Stripe Configuration:**
- API keys stored securely
- Webhook secret configured
- Product created: "Individual Tier" ($9/month)
- Test cards documented

---

### P2-T32: Subscription Product Setup
**Priority:** High
**Estimated Hours:** 6
**Assignee:** Backend Developer
**Dependencies:** P2-T31
**Status:** Not Started

**Description:**
Create subscription products in Stripe.

**Products to Create:**
- Individual Tier: $9/month
- Individual Tier (Annual): $90/year (17% discount)

**Tasks:**
- Create products in Stripe Dashboard
- Set pricing tiers
- Configure billing intervals
- Set up free trial (14 days)
- Configure proration settings
- Test product creation

**Acceptance Criteria:**
- [ ] Products created
- [ ] Pricing correct ($9/mo, $90/yr)
- [ ] Free trial configured (14 days)
- [ ] Proration enabled
- [ ] Test purchases work

---

### P2-T33: Checkout Flow Implementation
**Priority:** Critical
**Estimated Hours:** 10
**Assignee:** Full Stack Developer
**Dependencies:** P2-T32
**Status:** Not Started

**Description:**
Implement Stripe checkout flow for subscriptions.

**Implementation Files:**
- `/frontend/src/pages/billing/checkout.tsx`
- `/backend/services/billing-service.go`

**Tasks:**
- Create checkout page UI
- Integrate Stripe Checkout (embedded or redirect)
- Handle payment success
- Handle payment failure
- Create subscription in database
- Send confirmation email
- Upgrade user tier

**Acceptance Criteria:**
- [ ] Checkout page functional
- [ ] Payment methods accepted (card, ACH)
- [ ] Success page shown after payment
- [ ] Failure handled gracefully
- [ ] Subscription created in DB
- [ ] Confirmation email sent
- [ ] Tier upgraded immediately

**API Endpoint:**
```go
// POST /api/billing/create-checkout
type CreateCheckoutRequest struct {
    PriceID string `json:"price_id"` // Stripe Price ID
}

type CreateCheckoutResponse struct {
    CheckoutURL string `json:"checkout_url"`
    SessionID   string `json:"session_id"`
}
```

---

### P2-T34: Webhook Handler Implementation
**Priority:** Critical
**Estimated Hours:** 8
**Assignee:** Backend Developer
**Dependencies:** P2-T33
**Status:** Not Started

**Description:**
Handle Stripe webhook events for subscriptions.

**Webhook Events to Handle:**
- checkout.session.completed
- customer.subscription.created
- customer.subscription.updated
- customer.subscription.deleted
- invoice.payment_succeeded
- invoice.payment_failed

**Tasks:**
- Create webhook endpoint
- Verify webhook signatures
- Handle subscription created
- Handle subscription updated
- Handle subscription canceled
- Handle payment failed
- Update user tier accordingly
- Send notification emails

**Acceptance Criteria:**
- [ ] Webhook endpoint secure (signature verified)
- [ ] All events handled correctly
- [ ] User tier updated in real-time
- [ ] Failed payments handled
- [ ] Cancellations handled
- [ ] Emails sent on events
- [ ] Idempotent (duplicate events handled)

**Webhook Endpoint:**
```go
// POST /api/webhooks/stripe
func HandleStripeWebhook(c *gin.Context) {
    payload := c.Request.Body
    signature := c.GetHeader("Stripe-Signature")

    event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid signature"})
        return
    }

    switch event.Type {
    case "checkout.session.completed":
        handleCheckoutComplete(event)
    case "customer.subscription.deleted":
        handleSubscriptionCanceled(event)
    // ...
    }

    c.JSON(200, gin.H{"received": true})
}
```

---

### P2-T35: Billing Portal Integration
**Priority:** High
**Estimated Hours:** 6
**Assignee:** Full Stack Developer
**Dependencies:** P2-T34
**Status:** Not Started

**Description:**
Integrate Stripe Customer Portal for self-service.

**Features:**
- View subscription details
- Update payment method
- View invoices
- Cancel subscription
- Reactivate subscription

**Tasks:**
- Create portal redirect endpoint
- Add "Manage Billing" button in app
- Test all portal features
- Configure portal settings in Stripe
- Handle portal return URL

**Acceptance Criteria:**
- [ ] Portal accessible from app
- [ ] User can update payment method
- [ ] User can cancel subscription
- [ ] Invoices downloadable
- [ ] Return URL works
- [ ] All features tested

**Implementation:**
```go
// GET /api/billing/portal
func CreateBillingPortalSession(c *gin.Context) {
    userID := c.GetString("user_id")
    customer, _ := getStripeCustomer(userID)

    session, err := billingportal.Session.New(&stripe.BillingPortalSessionParams{
        Customer:  stripe.String(customer.ID),
        ReturnURL: stripe.String("https://sqlstudio.app/settings/billing"),
    })

    c.JSON(200, gin.H{"url": session.URL})
}
```

---

### P2-T36: Subscription State Management
**Priority:** High
**Estimated Hours:** 6
**Assignee:** Backend Developer
**Dependencies:** P2-T35
**Status:** Not Started

**Description:**
Track subscription state in application database.

**Database Table:**
```sql
CREATE TABLE subscriptions (
    subscription_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    stripe_subscription_id TEXT UNIQUE,
    stripe_customer_id TEXT,
    status TEXT NOT NULL, -- active, canceled, past_due
    current_period_start DATETIME,
    current_period_end DATETIME,
    cancel_at_period_end BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);
```

**Tasks:**
- Create subscriptions table
- Update on webhook events
- Query subscription status efficiently
- Handle subscription lifecycle
- Add subscription endpoints

**Acceptance Criteria:**
- [ ] Subscription state tracked
- [ ] Webhook updates work
- [ ] Status queried efficiently
- [ ] Lifecycle handled correctly
- [ ] API endpoints functional

---

### P2-T37: Payment Testing
**Priority:** Critical
**Estimated Hours:** 6
**Assignee:** QA Engineer
**Dependencies:** P2-T36
**Status:** Not Started

**Description:**
Test all payment scenarios.

**Test Cases:**
- Successful checkout
- Failed payment
- Subscription upgrade
- Subscription cancellation
- Payment method update
- Free trial expiry
- Invoice generation

**Tasks:**
- Use Stripe test cards
- Test all scenarios
- Verify webhook delivery
- Test edge cases
- Document findings

**Acceptance Criteria:**
- [ ] All scenarios tested
- [ ] Test cards work
- [ ] Webhooks received
- [ ] Edge cases handled
- [ ] Test report created

---

## Week 12: Beta Launch Preparation (Days 78-84)

### P2-T38: Beta Launch Checklist
**Priority:** Critical
**Estimated Hours:** 8
**Assignee:** Product Manager / Tech Lead
**Dependencies:** P2-T37
**Status:** Not Started

**Description:**
Prepare for beta launch with comprehensive checklist.

**Checklist Items:**
- [ ] All Phase 2 tasks complete
- [ ] Auth working (email, OAuth)
- [ ] Sync working (all entity types)
- [ ] Payments working (Stripe)
- [ ] Tests passing (unit, integration, E2E)
- [ ] Security audit complete
- [ ] Performance benchmarks met
- [ ] Documentation complete
- [ ] Monitoring set up
- [ ] Error tracking configured
- [ ] Support system ready
- [ ] Billing admin tools ready
- [ ] Legal docs ready (ToS, Privacy Policy)
- [ ] Email templates ready
- [ ] Landing page ready
- [ ] Beta invite system ready

**Tasks:**
- Create detailed launch checklist
- Assign owners to each item
- Track completion status
- Conduct launch readiness review
- Get stakeholder sign-off

**Acceptance Criteria:**
- [ ] All checklist items complete
- [ ] Launch team aligned
- [ ] Rollback plan documented
- [ ] Support team trained
- [ ] Stakeholders approved

---

### P2-T39: Beta User Onboarding
**Priority:** High
**Estimated Hours:** 6
**Assignee:** Product Manager / UI Designer
**Dependencies:** P2-T38
**Status:** Not Started

**Description:**
Create onboarding experience for beta users.

**Onboarding Steps:**
1. Welcome email
2. Account setup wizard
3. Initial sync setup
4. First connection guide
5. Feature tour
6. Upgrade prompt (14-day trial)

**Tasks:**
- Design onboarding flow
- Create welcome email template
- Build setup wizard UI
- Add feature tour (tooltips)
- Create tutorial content
- Add "What's new" modal

**Acceptance Criteria:**
- [ ] Onboarding flow smooth
- [ ] Welcome email sent
- [ ] Wizard guides users
- [ ] Tour highlights features
- [ ] Trial explained clearly
- [ ] Upgrade path clear

---

### P2-T40: Beta Launch
**Priority:** Critical
**Estimated Hours:** 4
**Assignee:** Product Manager / Tech Lead
**Dependencies:** P2-T39
**Status:** Not Started

**Description:**
Execute beta launch.

**Launch Steps:**
1. Deploy to production
2. Enable feature flags
3. Send beta invites (50 users)
4. Monitor metrics closely
5. Collect feedback
6. Fix critical issues
7. Iterate based on feedback

**Tasks:**
- Deploy to production
- Send beta invite emails
- Monitor error logs
- Track key metrics
- Set up feedback channels
- Respond to user issues
- Plan iteration based on feedback

**Acceptance Criteria:**
- [ ] Production deployment successful
- [ ] Beta users invited
- [ ] No critical errors
- [ ] Metrics tracked
- [ ] Feedback collected
- [ ] Issues triaged
- [ ] Next steps planned

**Success Metrics:**
- 50 beta users signed up
- >80% complete onboarding
- >50% enable sync
- >20% convert to paid (after trial)
- <5% critical bugs
- >4.0/5 satisfaction score

---

## Summary

### Task Breakdown by Week
- Week 5 (Auth): 6 tasks, 50 hours
- Week 6 (Turso): 5 tasks, 40 hours
- Week 7 (Upload Sync): 6 tasks, 48 hours
- Week 8 (Download Sync): 5 tasks, 42 hours
- Week 9 (Background Sync): 4 tasks, 28 hours
- Week 10 (Testing): 4 tasks, 26 hours
- Week 11 (Payments): 7 tasks, 46 hours
- Week 12 (Launch): 3 tasks, 18 hours

**Total:** 40 tasks, ~300 hours

### Critical Path
P2-T1 → P2-T2 → P2-T3 → P2-T7 → P2-T8 → P2-T9 → P2-T12 → P2-T18 → P2-T31 → P2-T33 → P2-T38 → P2-T40

### Resource Requirements
- 2 Full Stack Developers
- 1 Backend Developer
- 1 Frontend Developer
- 1 QA Engineer
- 0.5 DevOps Engineer
- 0.5 Security Specialist
- 0.5 Product Manager

**Total Team:** ~6.5 FTE

---

**Document Version:** 1.0
**Last Updated:** 2025-10-23
**Status:** Active
**Next Review:** 2025-11-20 (After Phase 1)
