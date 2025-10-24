# Phase 2: Individual Tier Backend - Technical Specifications

## Overview
Detailed technical specifications for Phase 2 implementation, covering authentication architecture, sync protocol, Turso schema, API endpoints, and webhook specifications.

**Phase:** Phase 2 - Weeks 5-12
**Last Updated:** 2025-10-23
**Status:** Draft

---

## Table of Contents
1. [Authentication Architecture](#1-authentication-architecture)
2. [Sync Protocol Specification](#2-sync-protocol-specification)
3. [Turso Schema Definition](#3-turso-schema-definition)
4. [API Endpoints](#4-api-endpoints)
5. [Webhook Specifications](#5-webhook-specifications)
6. [Error Handling Strategy](#6-error-handling-strategy)

---

## 1. Authentication Architecture

### 1.1 Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                       SQL Studio Client                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Frontend (React)                            │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │   │
│  │  │ Login Page │  │  Register  │  │   OAuth    │        │   │
│  │  │            │  │    Page    │  │  Buttons   │        │   │
│  │  └──────┬─────┘  └──────┬─────┘  └──────┬─────┘        │   │
│  │         │                │                │              │   │
│  │         └────────────────┴────────────────┘              │   │
│  │                          │                                │   │
│  │                  ┌───────▼────────┐                      │   │
│  │                  │  Auth Service  │                      │   │
│  │                  │   (Frontend)   │                      │   │
│  │                  └───────┬────────┘                      │   │
│  └──────────────────────────┼───────────────────────────────┘   │
└────────────────────────────┼───────────────────────────────────┘
                             │ HTTPS
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                  SQL Studio Backend (Go)                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Auth Service                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │   │
│  │  │  Register  │  │   Login    │  │   Token    │        │   │
│  │  │  Handler   │  │  Handler   │  │  Manager   │        │   │
│  │  └──────┬─────┘  └──────┬─────┘  └──────┬─────┘        │   │
│  │         │                │                │              │   │
│  │         └────────────────┴────────────────┘              │   │
│  │                          │                                │   │
│  └──────────────────────────┼───────────────────────────────┘   │
└────────────────────────────┼───────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
      ┌─────────────┐ ┌───────────┐ ┌──────────┐
      │  Supabase   │ │   Turso   │ │  Email   │
      │    Auth     │ │  (users)  │ │ Service  │
      └─────────────┘ └───────────┘ └──────────┘
```

### 1.2 Auth Provider: Supabase

**Why Supabase:**
- Cost: $25/mo for 100K MAU (sufficient for Phase 2)
- Features: Email/password, OAuth (GitHub, Google), JWT
- Security: Industry-standard, SOC2 compliant
- Integration: Good Go SDK, excellent documentation
- Bonus: Postgres database (could use for metadata)

**Configuration:**
```bash
# Environment variables
SUPABASE_URL=https://xxxxx.supabase.co
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
SUPABASE_SERVICE_ROLE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
JWT_SECRET=<from Supabase project settings>
```

### 1.3 JWT Token Structure

**Access Token (15 min TTL):**
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user-uuid-here",
    "email": "user@example.com",
    "role": "authenticated",
    "tier": "individual",
    "aud": "authenticated",
    "exp": 1730000000,
    "iat": 1729999100
  },
  "signature": "..."
}
```

**Refresh Token (7 day TTL):**
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user-uuid-here",
    "session_id": "session-uuid",
    "exp": 1730604800,
    "iat": 1729999100
  },
  "signature": "..."
}
```

### 1.4 OAuth Flow Sequence

```
User                  Frontend              Backend           Supabase
  │                       │                    │                 │
  │ Click "Login GitHub"  │                    │                 │
  ├──────────────────────>│                    │                 │
  │                       │ GET /auth/github   │                 │
  │                       ├───────────────────>│                 │
  │                       │                    │ Redirect to     │
  │                       │                    │ GitHub OAuth    │
  │                       │<───────────────────┤                 │
  │                       │                    │                 │
  │<──────────────────────┤                    │                 │
  │                       │                    │                 │
  │   GitHub Login Page   │                    │                 │
  │<──────────────────────┼────────────────────┼─────────────────┤
  │                       │                    │                 │
  │  Authorize SQL Studio │                    │                 │
  ├──────────────────────>│                    │                 │
  │                       │ Callback + code    │                 │
  │                       │<───────────────────┼─────────────────┤
  │                       │                    │                 │
  │                       │ GET /auth/callback │                 │
  │                       ├───────────────────>│ Exchange code   │
  │                       │                    ├────────────────>│
  │                       │                    │ User + tokens   │
  │                       │                    │<────────────────┤
  │                       │                    │                 │
  │                       │   JWT + User       │                 │
  │                       │<───────────────────┤                 │
  │<──────────────────────┤                    │                 │
  │                       │                    │                 │
```

---

## 2. Sync Protocol Specification

### 2.1 Sync Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Client Device                             │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Zustand    │──│ Sync Manager │──│ Offline Queue│         │
│  │    Store     │  │              │  │              │         │
│  └──────────────┘  └──────┬───────┘  └──────────────┘         │
│                           │                                    │
│  ┌──────────────┐  ┌──────▼───────┐                           │
│  │  IndexedDB   │──│ Turso Client │                           │
│  │   (Local)    │  │   (libSQL)   │                           │
│  └──────────────┘  └──────┬───────┘                           │
└────────────────────────────┼──────────────────────────────────┘
                             │ HTTPS / WSS
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Sync Backend (Go)                            │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  REST API    │  │  WebSocket   │  │ Conflict     │         │
│  │  (Push/Pull) │  │  (Real-time) │  │ Resolver     │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                 │                 │                  │
│         └─────────────────┴─────────────────┘                  │
│                           │                                    │
│                    ┌──────▼───────┐                            │
│                    │ Turso Client │                            │
│                    └──────┬───────┘                            │
└────────────────────────────┼──────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Turso Cloud                                 │
│                                                                 │
│  ┌──────────────┐      ┌──────────────┐                        │
│  │   Primary    │──────│  Edge Replica│                        │
│  │   Database   │      │  (us-west)   │                        │
│  └──────────────┘      └──────────────┘                        │
│                                                                 │
│                        ┌──────────────┐                        │
│                        │  Edge Replica│                        │
│                        │  (eu-west)   │                        │
│                        └──────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Sync Protocol

**Protocol Version:** 1.0

**Sync Modes:**
1. **Full Sync** - Initial sync, pull all user data
2. **Incremental Sync** - Pull changes since last sync
3. **Push Sync** - Push local changes to cloud
4. **Real-time Sync** - WebSocket for instant updates

### 2.3 Sync Vector

**Purpose:** Track sync state per device

**Structure:**
```typescript
interface SyncVector {
  user_id: string
  device_id: string
  last_sync_at: string  // ISO 8601 timestamp
  entity_versions: {
    connections: number
    query_tabs: number
    query_history: number
    saved_queries: number
    ai_sessions: number
    ui_preferences: number
  }
}
```

**Storage:** IndexedDB on client, Turso on server

### 2.4 Change Record Format

```typescript
interface SyncChange {
  change_id: string      // UUID
  entity_type: string    // 'connections' | 'query_tabs' | ...
  entity_id: string      // UUID of entity
  operation: string      // 'create' | 'update' | 'delete'
  data: any             // Entity data (full or partial)
  updated_at: string     // ISO 8601 timestamp
  user_id: string
  device_id: string
}
```

### 2.5 Sync API Protocol

**Push Changes (Client → Server):**
```http
POST /api/sync/push
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "device_id": "device-uuid",
  "changes": [
    {
      "entity_type": "query_tabs",
      "entity_id": "tab-uuid",
      "operation": "update",
      "data": {
        "content": "SELECT * FROM users",
        "updated_at": "2024-10-23T10:30:00Z"
      }
    }
  ]
}

Response 200 OK:
{
  "synced_count": 1,
  "conflicts": [],
  "timestamp": "2024-10-23T10:30:05Z"
}
```

**Pull Changes (Server → Client):**
```http
GET /api/sync/pull?since=2024-10-23T10:00:00Z
Authorization: Bearer <jwt_token>

Response 200 OK:
{
  "changes": [
    {
      "entity_type": "connections",
      "entity_id": "conn-uuid",
      "operation": "create",
      "data": {
        "name": "Production DB",
        "db_type": "postgresql",
        ...
      },
      "updated_at": "2024-10-23T10:15:00Z"
    }
  ],
  "timestamp": "2024-10-23T10:30:05Z",
  "has_more": false
}
```

### 2.6 WebSocket Protocol

**Connection:**
```
WSS wss://sync.sqlstudio.app/ws?token=<jwt_token>
```

**Message Format:**
```json
{
  "type": "change",
  "entity_type": "query_tabs",
  "entity_id": "tab-uuid",
  "operation": "update",
  "data": { ... },
  "updated_at": "2024-10-23T10:30:00Z"
}
```

**Heartbeat:**
```json
{
  "type": "ping"
}

Response:
{
  "type": "pong",
  "timestamp": "2024-10-23T10:30:00Z"
}
```

---

## 3. Turso Schema Definition

**(Full schema in turso-integration-design.md, key tables excerpted here)**

### 3.1 Core Tables

```sql
-- Users table
CREATE TABLE users (
    user_id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    display_name TEXT,
    tier TEXT NOT NULL DEFAULT 'free',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Connections table (NO PASSWORDS)
CREATE TABLE connections (
    connection_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    db_type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT NOT NULL,
    username TEXT,
    ssl_mode TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_connections_user ON connections(user_id, deleted_at);

-- Query tabs table
CREATE TABLE query_tabs (
    tab_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    connection_id TEXT,
    position INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_query_tabs_user ON query_tabs(user_id, deleted_at, position);
```

### 3.2 Indexes Strategy

**Index Naming Convention:**
- `idx_{table}_{columns}` for single table
- `idx_{table}_{purpose}` for complex queries

**Performance Considerations:**
- All user_id queries indexed
- Soft delete (deleted_at) included in indexes
- Composite indexes for common query patterns

---

## 4. API Endpoints

### 4.1 Authentication Endpoints

```
POST   /api/auth/register
POST   /api/auth/login
POST   /api/auth/logout
POST   /api/auth/refresh
POST   /api/auth/forgot-password
POST   /api/auth/reset-password
GET    /api/auth/oauth/github
GET    /api/auth/oauth/google
GET    /api/auth/oauth/callback
GET    /api/auth/me
```

### 4.2 Sync Endpoints

```
POST   /api/sync/push
GET    /api/sync/pull
POST   /api/sync/full
GET    /api/sync/status
POST   /api/sync/resolve-conflict
GET    /api/sync/conflicts
```

### 4.3 Billing Endpoints

```
POST   /api/billing/create-checkout
GET    /api/billing/portal
GET    /api/billing/subscription
POST   /api/billing/cancel
POST   /api/webhooks/stripe
```

### 4.4 API Authentication

**Authorization Header:**
```
Authorization: Bearer <jwt_access_token>
```

**Middleware:**
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.JSON(401, gin.H{"error": "Missing authorization"})
            c.Abort()
            return
        }

        token := strings.TrimPrefix(tokenString, "Bearer ")
        claims, err := validateJWT(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

---

## 5. Webhook Specifications

### 5.1 Stripe Webhook Events

**Webhook URL:** `https://api.sqlstudio.app/api/webhooks/stripe`

**Events to Handle:**
- `checkout.session.completed`
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.payment_succeeded`
- `invoice.payment_failed`

### 5.2 Webhook Payload Examples

**checkout.session.completed:**
```json
{
  "id": "evt_xxx",
  "type": "checkout.session.completed",
  "data": {
    "object": {
      "id": "cs_xxx",
      "customer": "cus_xxx",
      "subscription": "sub_xxx",
      "mode": "subscription",
      "metadata": {
        "user_id": "user-uuid"
      }
    }
  }
}
```

**Handler Logic:**
```go
func HandleCheckoutComplete(event stripe.Event) error {
    var session stripe.CheckoutSession
    json.Unmarshal(event.Data.Raw, &session)

    userID := session.Metadata["user_id"]
    subscriptionID := session.Subscription.ID

    // Create subscription in database
    err := createSubscription(userID, subscriptionID)
    if err != nil {
        return err
    }

    // Upgrade user tier
    err = upgradeUserTier(userID, "individual")
    if err != nil {
        return err
    }

    // Send confirmation email
    sendSubscriptionConfirmationEmail(userID)

    return nil
}
```

### 5.3 Webhook Security

**Signature Verification:**
```go
func VerifyStripeWebhook(payload []byte, signature string) error {
    webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
    _, err := webhook.ConstructEvent(payload, signature, webhookSecret)
    return err
}
```

**Idempotency:**
```go
func HandleWebhook(eventID string, handler func() error) error {
    // Check if already processed
    processed, err := isEventProcessed(eventID)
    if err != nil {
        return err
    }
    if processed {
        return nil // Already handled
    }

    // Process event
    err = handler()
    if err != nil {
        return err
    }

    // Mark as processed
    return markEventProcessed(eventID)
}
```

---

## 6. Error Handling Strategy

### 6.1 Error Categories

**Client Errors (4xx):**
- 400 Bad Request - Invalid input
- 401 Unauthorized - Missing/invalid auth
- 403 Forbidden - Insufficient permissions
- 404 Not Found - Resource not found
- 409 Conflict - Sync conflict
- 429 Too Many Requests - Rate limited

**Server Errors (5xx):**
- 500 Internal Server Error - Unexpected error
- 502 Bad Gateway - Upstream service error
- 503 Service Unavailable - Maintenance/overload
- 504 Gateway Timeout - Request timeout

### 6.2 Error Response Format

```json
{
  "error": {
    "code": "SYNC_CONFLICT",
    "message": "Conflict detected while syncing query tab",
    "details": {
      "entity_type": "query_tabs",
      "entity_id": "tab-uuid",
      "local_version": "2024-10-23T10:30:00Z",
      "remote_version": "2024-10-23T10:31:00Z"
    },
    "request_id": "req-uuid"
  }
}
```

### 6.3 Error Codes

```
AUTH_INVALID_CREDENTIALS
AUTH_EMAIL_NOT_VERIFIED
AUTH_TOKEN_EXPIRED
AUTH_RATE_LIMITED

SYNC_CONFLICT
SYNC_OFFLINE_QUEUE_FULL
SYNC_RATE_LIMITED

BILLING_PAYMENT_FAILED
BILLING_SUBSCRIPTION_NOT_FOUND
BILLING_WEBHOOK_SIGNATURE_INVALID

INTERNAL_DATABASE_ERROR
INTERNAL_SERVICE_UNAVAILABLE
```

### 6.4 Retry Strategy

```typescript
interface RetryConfig {
  maxRetries: number        // 5
  initialDelay: number      // 1000ms
  maxDelay: number          // 30000ms
  backoffMultiplier: number // 2
}

async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  config: RetryConfig
): Promise<T> {
  let delay = config.initialDelay

  for (let i = 0; i < config.maxRetries; i++) {
    try {
      return await fn()
    } catch (error) {
      if (i === config.maxRetries - 1) throw error

      // Don't retry client errors
      if (error.status >= 400 && error.status < 500) throw error

      await sleep(delay)
      delay = Math.min(delay * config.backoffMultiplier, config.maxDelay)
    }
  }

  throw new Error('Max retries exceeded')
}
```

---

## 7. Performance Specifications

### 7.1 Latency Targets

| Operation | Target (p95) | Max Acceptable |
|-----------|--------------|----------------|
| Auth login | <200ms | <500ms |
| Token refresh | <100ms | <300ms |
| Sync push | <500ms | <1s |
| Sync pull | <500ms | <1s |
| Full sync | <2s | <5s |
| Payment checkout | <2s | <5s |

### 7.2 Throughput Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| API requests | 1000 req/s | Per instance |
| WebSocket connections | 10,000 | Per instance |
| Sync operations | 500/s | Across all users |
| Database writes | 5,000 rows/s | Turso limit |

### 7.3 Resource Limits

| Resource | Limit | Notes |
|----------|-------|-------|
| Max request size | 10 MB | For large queries |
| Max response size | 10 MB | Paginate if larger |
| WebSocket message | 1 MB | Per message |
| Offline queue | 1,000 items | Per device |
| JWT token TTL | 15 min | Access token |
| Refresh token TTL | 7 days | Refresh token |

---

## 8. Security Specifications

### 8.1 Data Classification

**Sensitive (NEVER sync):**
- Database passwords
- SSH private keys
- API tokens/secrets
- Credit card numbers

**Personal (Sync with encryption):**
- Email addresses
- Display names
- User preferences

**Application (Sync plaintext):**
- Query tab content
- Connection metadata (no passwords)
- Query history (optional redaction)
- Saved queries

### 8.2 Encryption Standards

**In Transit:**
- TLS 1.3 minimum
- Strong cipher suites only
- Certificate pinning (mobile apps)

**At Rest:**
- Turso: AES-256 (automatic)
- Passwords: Argon2id hashing
- Tokens: Secure random generation

### 8.3 Compliance

**GDPR:**
- User data export API
- User data deletion API
- Privacy policy consent
- Data retention policies

**PCI DSS:**
- No card data stored
- Stripe handles all payments
- Webhook signature verification

---

## Document Metadata

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-10-23
**Next Review:** 2025-11-06 (Week 7)
**Approved By:** Pending

**Change Log:**
| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-10-23 | 1.0 | Initial draft | PM Agent |

---

## References

- [Turso Integration Design](./turso-integration-design.md)
- [Phase 2 Tasks](./phase-2-tasks.md)
- [Phase 2 Risk Register](./phase-2-risks.md)
- [Supabase Auth Documentation](https://supabase.com/docs/guides/auth)
- [Stripe Webhook Guide](https://stripe.com/docs/webhooks)
