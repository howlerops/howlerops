# SQL Studio Sync System Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         SQL Studio Frontend                             │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────────┐ │
│  │   Desktop    │  │     Web      │  │   Mobile   │  │    Tablet    │ │
│  │   Electron   │  │   Browser    │  │   Native   │  │     PWA      │ │
│  └──────┬───────┘  └──────┬───────┘  └─────┬──────┘  └──────┬───────┘ │
└─────────┼──────────────────┼─────────────────┼────────────────┼─────────┘
          │                  │                 │                │
          └──────────────────┴─────────────────┴────────────────┘
                                      │
                                      │ HTTPS/REST
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      API Gateway / Load Balancer                        │
│                         (Rate Limiting & CORS)                          │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Backend Go Server                               │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    HTTP/REST API Layer                           │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │  │
│  │  │   Auth      │  │    Sync     │  │   Health    │            │  │
│  │  │  Endpoints  │  │  Endpoints  │  │   Check     │            │  │
│  │  └──────┬──────┘  └──────┬──────┘  └─────────────┘            │  │
│  └─────────┼─────────────────┼──────────────────────────────────────┘  │
│            │                 │                                          │
│  ┌─────────▼─────────────────▼──────────────────────────────────────┐  │
│  │                   Middleware Layer                                │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │  │
│  │  │     Auth     │  │ Rate Limiter │  │    Request Logger      │ │  │
│  │  │  Middleware  │  │  Middleware  │  │      Middleware        │ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────────────┘ │  │
│  └──────────────────────────┬──────────────────────────────────────┘  │
│                             │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐  │
│  │                   Business Logic Layer                           │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │  │
│  │  │  Auth Service   │  │  Sync Service   │  │  Email Service  │ │  │
│  │  │                 │  │                 │  │                 │ │  │
│  │  │ • Login         │  │ • Upload        │  │ • Verification  │ │  │
│  │  │ • Register      │  │ • Download      │  │ • Reset         │ │  │
│  │  │ • Verify Email  │  │ • Conflicts     │  │ • Welcome       │ │  │
│  │  │ • Reset Pass    │  │ • Resolution    │  │                 │ │  │
│  │  │ • Tokens        │  │ • Sanitization  │  │                 │ │  │
│  │  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘ │  │
│  └───────────┼──────────────────────┼──────────────────────┼───────┘  │
│              │                      │                      │           │
│  ┌───────────▼──────────────────────▼──────────────────────▼───────┐  │
│  │                       Storage Layer                              │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │  │
│  │  │   User       │  │     Sync     │  │      Token           │  │  │
│  │  │   Store      │  │    Store     │  │      Store           │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────────────┘  │  │
│  └────────┬────────────────────┬────────────────────┬──────────────┘  │
└───────────┼────────────────────┼────────────────────┼─────────────────┘
            │                    │                    │
            ▼                    ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   PostgreSQL     │  │   Turso DB       │  │   Redis          │
│   (Users/Auth)   │  │   (Sync Data)    │  │   (Sessions)     │
└──────────────────┘  └──────────────────┘  └──────────────────┘

            │
            ▼
┌──────────────────────────────────┐
│        Resend API                │
│     (Email Delivery)             │
└──────────────────────────────────┘
```

## Component Architecture

### 1. Email Service Layer

```
┌────────────────────────────────────────────────────────────────┐
│                      Email Service                             │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Interface: EmailService                                       │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ • SendVerificationEmail(email, token, url)               │ │
│  │ • SendPasswordResetEmail(email, token, url)              │ │
│  │ • SendWelcomeEmail(email, name)                          │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  Implementations:                                              │
│  ┌─────────────────────┐      ┌─────────────────────────┐    │
│  │ ResendEmailService  │      │  MockEmailService       │    │
│  │ (Production)        │      │  (Testing)              │    │
│  │                     │      │                         │    │
│  │ • Resend API        │      │ • In-memory storage     │    │
│  │ • HTML templates    │      │ • No external calls     │    │
│  │ • Error handling    │      │ • Verification helpers  │    │
│  │ • Retry logic       │      │                         │    │
│  └─────────────────────┘      └─────────────────────────┘    │
│                                                                │
│  Templates:                                                    │
│  • verification.html - Email verification                      │
│  • password_reset.html - Password reset                        │
│  • welcome.html - Welcome message                              │
└────────────────────────────────────────────────────────────────┘
```

### 2. Sync Service Layer

```
┌────────────────────────────────────────────────────────────────┐
│                       Sync Service                             │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Core Components:                                              │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Sync Engine                                             │ │
│  │  ┌────────────┐  ┌────────────┐  ┌──────────────────┐  │ │
│  │  │  Upload    │  │  Download  │  │  Conflict        │  │ │
│  │  │  Handler   │  │  Handler   │  │  Resolver        │  │ │
│  │  └────────────┘  └────────────┘  └──────────────────┘  │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  Data Flow:                                                    │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  1. Client Upload                                        │ │
│  │     ↓                                                    │ │
│  │  2. Validate Request (size, credentials, format)        │ │
│  │     ↓                                                    │ │
│  │  3. Detect Conflicts (sync_version, updated_at)         │ │
│  │     ↓                                                    │ │
│  │  4. Apply Changes (connections, queries, history)       │ │
│  │     ↓                                                    │ │
│  │  5. Update Metadata (last_sync, counts)                 │ │
│  │     ↓                                                    │ │
│  │  6. Return Response (success, conflicts, rejected)      │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  Conflict Resolution Strategies:                               │
│  ┌────────────────────┐  ┌─────────────────┐                 │
│  │  Last Write Wins   │  │   Keep Both     │                 │
│  │  (Default)         │  │                 │                 │
│  │                    │  │  • Create copy  │                 │
│  │  • Compare time    │  │  • New ID       │                 │
│  │  • Use latest      │  │  • Keep both    │                 │
│  └────────────────────┘  └─────────────────┘                 │
│            ┌─────────────────────┐                            │
│            │   User Choice       │                            │
│            │                     │                            │
│            │  • Show UI          │                            │
│            │  • User decides     │                            │
│            │  • Apply choice     │                            │
│            └─────────────────────┘                            │
└────────────────────────────────────────────────────────────────┘
```

### 3. Data Models

```
┌─────────────────────────────────────────────────────────────────┐
│                        Data Models                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ConnectionTemplate                                             │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ • id: string                                              │ │
│  │ • name: string                                            │ │
│  │ • type: postgres|mysql|sqlite|mongodb                    │ │
│  │ • host: string (optional)                                 │ │
│  │ • port: int (optional)                                    │ │
│  │ • database: string                                        │ │
│  │ • username: string (optional)                             │ │
│  │ • use_ssh: bool                                           │ │
│  │ • color: string                                           │ │
│  │ • icon: string                                            │ │
│  │ • metadata: map[string]string                             │ │
│  │ • created_at: timestamp                                   │ │
│  │ • updated_at: timestamp                                   │ │
│  │ • sync_version: int                                       │ │
│  │                                                           │ │
│  │ NOTE: password and ssh_key NEVER synced                  │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  SavedQuery                                                     │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ • id: string                                              │ │
│  │ • name: string                                            │ │
│  │ • description: string                                     │ │
│  │ • query: string                                           │ │
│  │ • connection_id: string                                   │ │
│  │ • tags: []string                                          │ │
│  │ • favorite: bool                                          │ │
│  │ • metadata: map[string]string                             │ │
│  │ • created_at: timestamp                                   │ │
│  │ • updated_at: timestamp                                   │ │
│  │ • sync_version: int                                       │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  QueryHistory                                                   │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ • id: string                                              │ │
│  │ • query: string                                           │ │
│  │ • connection_id: string                                   │ │
│  │ • executed_at: timestamp                                  │ │
│  │ • duration_ms: int64                                      │ │
│  │ • rows_affected: int64                                    │ │
│  │ • success: bool                                           │ │
│  │ • error: string (optional)                                │ │
│  │ • metadata: map[string]string                             │ │
│  │ • sync_version: int                                       │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Conflict                                                       │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ • id: string                                              │ │
│  │ • item_type: connection|saved_query|query_history        │ │
│  │ • item_id: string                                         │ │
│  │ • local_version: ConflictVersion                          │ │
│  │ • remote_version: ConflictVersion                         │ │
│  │ • detected_at: timestamp                                  │ │
│  │ • resolved_at: timestamp (optional)                       │ │
│  │ • resolution: strategy (optional)                         │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Sync Flow Diagram

```
┌──────────────┐                                    ┌──────────────┐
│  Device A    │                                    │  Device B    │
└──────┬───────┘                                    └──────┬───────┘
       │                                                   │
       │ 1. Make local changes                            │
       │    (create connection)                           │
       │                                                   │
       │ 2. Upload changes                                │
       ├─────────────────────────┐                        │
       │                         │                        │
       │                    ┌────▼─────┐                  │
       │                    │  Server  │                  │
       │                    │          │                  │
       │                    │ • Validate                  │
       │                    │ • Detect conflicts          │
       │                    │ • Save changes              │
       │                    │ • Update metadata           │
       │                    └────┬─────┘                  │
       │                         │                        │
       │◄────────────────────────┘                        │
       │ 3. Response                                      │
       │    (success, no conflicts)                       │
       │                                                   │
       │                                                   │
       │                                                   │ 4. Download changes
       │                                                   │    (periodic sync)
       │                    ┌─────────┐                   │
       │                    │ Server  │◄──────────────────┤
       │                    │         │                   │
       │                    │ • Query changes            │
       │                    │ • Filter by timestamp      │
       │                    │ • Return delta             │
       │                    └────┬────┘                   │
       │                         │                        │
       │                         └───────────────────────►│
       │                                                   │ 5. Apply changes
       │                                                   │    (add connection)
       │                                                   │
       │ 6. Make conflicting change                       │
       │    (update same connection)                      │
       │                                                   │
       │ 7. Upload changes                                │
       ├─────────────────────────┐                        │
       │                         │                        │
       │                    ┌────▼─────┐                  │
       │                    │  Server  │                  │
       │                    │          │                  │
       │                    │ • Compare sync_version     │
       │                    │ • Detect CONFLICT!         │
       │                    │ • Save conflict            │
       │                    └────┬─────┘                  │
       │                         │                        │
       │◄────────────────────────┘                        │
       │ 8. Response with conflict                        │
       │                                                   │
       │ 9. Show conflict UI                              │
       │    User chooses resolution                       │
       │                                                   │
       │ 10. Resolve conflict                             │
       ├─────────────────────────┐                        │
       │                         │                        │
       │                    ┌────▼─────┐                  │
       │                    │  Server  │                  │
       │                    │          │                  │
       │                    │ • Apply resolution         │
       │                    │ • Mark resolved            │
       │                    └────┬─────┘                  │
       │                         │                        │
       │◄────────────────────────┘                        │
       │ 11. Conflict resolved                            │
       │                                                   │
```

## Database Schema

```
┌─────────────────────────────────────────────────────────────────┐
│                      Turso Database                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  connections                                                    │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ id              TEXT PRIMARY KEY                          │ │
│  │ user_id         TEXT NOT NULL                             │ │
│  │ name            TEXT NOT NULL                             │ │
│  │ type            TEXT NOT NULL                             │ │
│  │ host            TEXT                                      │ │
│  │ port            INTEGER                                   │ │
│  │ database_name   TEXT NOT NULL                             │ │
│  │ username        TEXT                                      │ │
│  │ use_ssh         BOOLEAN DEFAULT 0                         │ │
│  │ ssh_host        TEXT                                      │ │
│  │ ssh_port        INTEGER                                   │ │
│  │ ssh_user        TEXT                                      │ │
│  │ color           TEXT                                      │ │
│  │ icon            TEXT                                      │ │
│  │ metadata        TEXT (JSON)                               │ │
│  │ created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ sync_version    INTEGER DEFAULT 1                         │ │
│  │ deleted_at      TIMESTAMP                                 │ │
│  │                                                           │ │
│  │ UNIQUE(user_id, id)                                       │ │
│  │ INDEX idx_connections_user_id (user_id)                  │ │
│  │ INDEX idx_connections_updated_at (updated_at)            │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  saved_queries                                                  │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ id              TEXT PRIMARY KEY                          │ │
│  │ user_id         TEXT NOT NULL                             │ │
│  │ name            TEXT NOT NULL                             │ │
│  │ description     TEXT                                      │ │
│  │ query           TEXT NOT NULL                             │ │
│  │ connection_id   TEXT                                      │ │
│  │ tags            TEXT (JSON array)                         │ │
│  │ favorite        BOOLEAN DEFAULT 0                         │ │
│  │ metadata        TEXT (JSON)                               │ │
│  │ created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ sync_version    INTEGER DEFAULT 1                         │ │
│  │ deleted_at      TIMESTAMP                                 │ │
│  │                                                           │ │
│  │ UNIQUE(user_id, id)                                       │ │
│  │ INDEX idx_saved_queries_user_id (user_id)                │ │
│  │ INDEX idx_saved_queries_updated_at (updated_at)          │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  query_history                                                  │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ id              TEXT PRIMARY KEY                          │ │
│  │ user_id         TEXT NOT NULL                             │ │
│  │ query           TEXT NOT NULL                             │ │
│  │ connection_id   TEXT                                      │ │
│  │ executed_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ duration_ms     INTEGER                                   │ │
│  │ rows_affected   INTEGER                                   │ │
│  │ success         BOOLEAN DEFAULT 1                         │ │
│  │ error           TEXT                                      │ │
│  │ metadata        TEXT (JSON)                               │ │
│  │ sync_version    INTEGER DEFAULT 1                         │ │
│  │                                                           │ │
│  │ UNIQUE(user_id, id)                                       │ │
│  │ INDEX idx_query_history_user_id (user_id)                │ │
│  │ INDEX idx_query_history_executed_at (executed_at)        │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  conflicts                                                      │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ id              TEXT PRIMARY KEY                          │ │
│  │ user_id         TEXT NOT NULL                             │ │
│  │ item_type       TEXT NOT NULL                             │ │
│  │ item_id         TEXT NOT NULL                             │ │
│  │ local_version   TEXT (JSON)                               │ │
│  │ remote_version  TEXT (JSON)                               │ │
│  │ detected_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ resolved_at     TIMESTAMP                                 │ │
│  │ resolution      TEXT                                      │ │
│  │                                                           │ │
│  │ UNIQUE(user_id, id)                                       │ │
│  │ INDEX idx_conflicts_user_id (user_id)                    │ │
│  │ INDEX idx_conflicts_resolved_at (resolved_at)            │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  sync_metadata                                                  │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ user_id         TEXT NOT NULL                             │ │
│  │ device_id       TEXT NOT NULL                             │ │
│  │ last_sync_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP       │ │
│  │ total_synced    INTEGER DEFAULT 0                         │ │
│  │ conflicts_count INTEGER DEFAULT 0                         │ │
│  │ version         TEXT                                      │ │
│  │                                                           │ │
│  │ PRIMARY KEY(user_id, device_id)                           │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Security Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Security Layers                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Transport Security                                          │
│     • TLS 1.3 encryption                                        │
│     • HTTPS only in production                                  │
│     • Certificate pinning (optional)                            │
│                                                                 │
│  2. Authentication                                              │
│     • JWT tokens (HS256)                                        │
│     • Token expiration (24 hours)                               │
│     • Refresh tokens (7 days)                                   │
│     • Session management                                        │
│                                                                 │
│  3. Authorization                                               │
│     • User ID scoped queries                                    │
│     • Device ID tracking                                        │
│     • Role-based access (future)                                │
│                                                                 │
│  4. Data Security                                               │
│     • Credential sanitization                                   │
│     • No passwords in sync                                      │
│     • Encrypted at rest (Turso)                                 │
│     • Soft deletes (30-day retention)                           │
│                                                                 │
│  5. Rate Limiting                                               │
│     • 10 uploads/min per user                                   │
│     • 20 downloads/min per user                                 │
│     • Token bucket algorithm                                    │
│     • IP-based throttling                                       │
│                                                                 │
│  6. Input Validation                                            │
│     • Request size limits (10MB)                                │
│     • JSON schema validation                                    │
│     • SQL injection prevention                                  │
│     • XSS protection                                            │
│                                                                 │
│  7. Email Security                                              │
│     • Secure token generation (32 bytes)                        │
│     • Token expiration                                          │
│     • One-time use tokens                                       │
│     • DKIM/SPF/DMARC (Resend)                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Deployment Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Production Deployment                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  CDN/Edge (Cloudflare)                                          │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ • SSL/TLS termination                                     │ │
│  │ • DDoS protection                                         │ │
│  │ • Rate limiting                                           │ │
│  │ • Geographic routing                                      │ │
│  └─────────────────────┬─────────────────────────────────────┘ │
│                        │                                        │
│                        ▼                                        │
│  Load Balancer (AWS ALB)                                        │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ • Health checks                                           │ │
│  │ • Auto-scaling                                            │ │
│  │ • SSL certificates                                        │ │
│  │ • Request routing                                         │ │
│  └──────┬──────────────────┬──────────────────┬──────────────┘ │
│         │                  │                  │                 │
│         ▼                  ▼                  ▼                 │
│  Backend Instances (ECS/Fargate)                                │
│  ┌────────────┐      ┌────────────┐     ┌────────────┐        │
│  │ Instance 1 │      │ Instance 2 │     │ Instance N │        │
│  │            │      │            │     │            │        │
│  │ Go Server  │      │ Go Server  │     │ Go Server  │        │
│  │ Port 8500  │      │ Port 8500  │     │ Port 8500  │        │
│  └─────┬──────┘      └─────┬──────┘     └─────┬──────┘        │
│        │                   │                   │                │
│        └───────────────────┴───────────────────┘                │
│                            │                                    │
│                            ▼                                    │
│  Data Layer                                                     │
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────────┐  │
│  │  RDS Postgres  │  │   Turso DB     │  │  Redis Cluster  │  │
│  │  (Auth/Users)  │  │  (Sync Data)   │  │   (Sessions)    │  │
│  │                │  │                │  │                 │  │
│  │  • Primary     │  │  • Edge DB     │  │  • Cache        │  │
│  │  • Read replica│  │  • Global      │  │  • Rate limits  │  │
│  │  • Backup      │  │  • Fast        │  │  • Locks        │  │
│  └────────────────┘  └────────────────┘  └─────────────────┘  │
│                                                                 │
│  External Services                                              │
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────────┐  │
│  │  Resend API    │  │   DataDog      │  │   Sentry        │  │
│  │  (Email)       │  │   (Metrics)    │  │   (Errors)      │  │
│  └────────────────┘  └────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Monitoring & Observability

```
┌─────────────────────────────────────────────────────────────────┐
│                 Monitoring Stack                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Metrics (Prometheus/DataDog)                                   │
│  • Request rate (req/sec)                                       │
│  • Response time (p50, p95, p99)                                │
│  • Error rate (%)                                               │
│  • Upload size distribution                                     │
│  • Conflict rate                                                │
│  • Active users                                                 │
│  • Sync success rate                                            │
│                                                                 │
│  Logs (CloudWatch/Loki)                                         │
│  • Structured JSON logs                                         │
│  • Request/response logs                                        │
│  • Error stack traces                                           │
│  • Performance metrics                                          │
│  • Security events                                              │
│                                                                 │
│  Traces (OpenTelemetry/Jaeger)                                  │
│  • Request tracing                                              │
│  • Service dependencies                                         │
│  • Slow query detection                                         │
│  • Bottleneck identification                                    │
│                                                                 │
│  Alerts                                                         │
│  • Error rate > 1%                                              │
│  • Response time p95 > 1s                                       │
│  • Sync failure rate > 5%                                       │
│  • Database connection issues                                   │
│  • Email delivery failures                                      │
└─────────────────────────────────────────────────────────────────┘
```

This architecture provides a solid foundation for SQL Studio's sync system with emphasis on:
- **Scalability**: Horizontal scaling, load balancing
- **Reliability**: Multiple replicas, health checks, retries
- **Security**: Multiple layers, encryption, sanitization
- **Performance**: Caching, indexing, connection pooling
- **Observability**: Comprehensive monitoring and logging
