# SQL Studio Sync API Documentation

## Base URL

```
Production: https://api.sqlstudio.io
Development: http://localhost:8500
```

## Authentication

All sync endpoints require JWT authentication via Bearer token:

```
Authorization: Bearer <your_jwt_token>
```

## Rate Limits

| Endpoint | Rate Limit |
|----------|-----------|
| /api/sync/upload | 10 req/min |
| /api/sync/download | 20 req/min |
| /api/sync/conflicts | 10 req/min |
| /api/auth/* | 20 req/min |

## Sync Endpoints

### Upload Local Changes

Upload local changes to sync with the cloud.

**Endpoint:** `POST /api/sync/upload`

**Request Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "device_id": "uuid-v4-device-identifier",
  "last_sync_at": "2024-01-01T00:00:00Z",
  "changes": [
    {
      "id": "uuid-v4",
      "item_type": "connection",
      "item_id": "connection-uuid",
      "action": "create",
      "data": {
        "id": "connection-uuid",
        "name": "Production DB",
        "type": "postgres",
        "host": "db.example.com",
        "port": 5432,
        "database": "prod_db",
        "username": "dbuser",
        "color": "#3B82F6",
        "icon": "database",
        "metadata": {
          "environment": "production"
        },
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T01:00:00Z",
        "sync_version": 1
      },
      "updated_at": "2024-01-01T01:00:00Z",
      "sync_version": 1,
      "device_id": "device-uuid"
    }
  ]
}
```

**Response (Success):**
```json
{
  "success": true,
  "synced_at": "2024-01-01T02:00:00Z",
  "conflicts": [],
  "rejected": [],
  "message": "Synced 1 items, 0 conflicts, 0 rejected"
}
```

**Response (With Conflicts):**
```json
{
  "success": true,
  "synced_at": "2024-01-01T02:00:00Z",
  "conflicts": [
    {
      "id": "conflict-uuid",
      "item_type": "connection",
      "item_id": "connection-uuid",
      "local_version": {
        "data": { "name": "Local Name", ... },
        "updated_at": "2024-01-01T01:00:00Z",
        "sync_version": 1,
        "device_id": "device-uuid"
      },
      "remote_version": {
        "data": { "name": "Remote Name", ... },
        "updated_at": "2024-01-01T01:30:00Z",
        "sync_version": 2
      },
      "detected_at": "2024-01-01T02:00:00Z"
    }
  ],
  "rejected": [],
  "message": "Synced 0 items, 1 conflicts, 0 rejected"
}
```

**Response (With Rejections):**
```json
{
  "success": false,
  "synced_at": "2024-01-01T02:00:00Z",
  "conflicts": [],
  "rejected": [
    {
      "change": { ... },
      "reason": "connection data contains password - credentials should not be synced"
    }
  ],
  "message": "Synced 0 items, 0 conflicts, 1 rejected"
}
```

**Item Types:**
- `connection` - Database connections
- `saved_query` - Saved SQL queries
- `query_history` - Query execution history

**Actions:**
- `create` - Create new item
- `update` - Update existing item
- `delete` - Delete item

**Error Codes:**
- `400` - Invalid request (validation errors)
- `401` - Unauthorized (missing/invalid token)
- `413` - Payload too large (> 10MB)
- `429` - Rate limit exceeded
- `500` - Internal server error

---

### Download Remote Changes

Download all changes from the cloud since a specific timestamp.

**Endpoint:** `GET /api/sync/download`

**Request Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `since` (required) - ISO 8601 timestamp (e.g., `2024-01-01T00:00:00Z`)
- `device_id` (required) - Device identifier

**Example Request:**
```
GET /api/sync/download?since=2024-01-01T00:00:00Z&device_id=device-uuid
```

**Response:**
```json
{
  "connections": [
    {
      "id": "connection-uuid",
      "name": "Production DB",
      "type": "postgres",
      "host": "db.example.com",
      "port": 5432,
      "database": "prod_db",
      "username": "dbuser",
      "use_ssh": false,
      "color": "#3B82F6",
      "icon": "database",
      "metadata": {
        "environment": "production"
      },
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T01:00:00Z",
      "sync_version": 2
    }
  ],
  "saved_queries": [
    {
      "id": "query-uuid",
      "name": "Active Users",
      "description": "Get all active users",
      "query": "SELECT * FROM users WHERE active = true",
      "connection_id": "connection-uuid",
      "tags": ["users", "active"],
      "favorite": true,
      "metadata": {},
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T01:00:00Z",
      "sync_version": 1
    }
  ],
  "query_history": [
    {
      "id": "history-uuid",
      "query": "SELECT COUNT(*) FROM users",
      "connection_id": "connection-uuid",
      "executed_at": "2024-01-01T01:00:00Z",
      "duration_ms": 42,
      "rows_affected": 1,
      "success": true,
      "metadata": {},
      "sync_version": 1
    }
  ],
  "sync_timestamp": "2024-01-01T02:00:00Z",
  "has_more": false
}
```

**Notes:**
- History is limited to 1000 items per request
- If `has_more` is `true`, make another request with the returned `sync_timestamp`
- All items include `sync_version` for conflict detection

**Error Codes:**
- `400` - Invalid request (missing parameters)
- `401` - Unauthorized
- `429` - Rate limit exceeded
- `500` - Internal server error

---

### List Conflicts

Get all unresolved sync conflicts for the authenticated user.

**Endpoint:** `GET /api/sync/conflicts`

**Request Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "conflicts": [
    {
      "id": "conflict-uuid",
      "item_type": "connection",
      "item_id": "connection-uuid",
      "local_version": {
        "data": {
          "id": "connection-uuid",
          "name": "Local Name",
          "type": "postgres",
          "database": "mydb",
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T01:00:00Z",
          "sync_version": 1
        },
        "updated_at": "2024-01-01T01:00:00Z",
        "sync_version": 1,
        "device_id": "device-uuid"
      },
      "remote_version": {
        "data": {
          "id": "connection-uuid",
          "name": "Remote Name",
          "type": "postgres",
          "database": "mydb",
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T01:30:00Z",
          "sync_version": 2
        },
        "updated_at": "2024-01-01T01:30:00Z",
        "sync_version": 2
      },
      "detected_at": "2024-01-01T02:00:00Z"
    }
  ],
  "count": 1
}
```

**Error Codes:**
- `401` - Unauthorized
- `500` - Internal server error

---

### Resolve Conflict

Resolve a specific sync conflict using a chosen strategy.

**Endpoint:** `POST /api/sync/conflicts/{conflict_id}/resolve`

**Request Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body (Last Write Wins):**
```json
{
  "strategy": "last_write_wins"
}
```

**Request Body (Keep Both):**
```json
{
  "strategy": "keep_both"
}
```

**Request Body (User Choice):**
```json
{
  "strategy": "user_choice",
  "chosen_version": "local"
}
```

**Conflict Resolution Strategies:**
- `last_write_wins` - Use the version with the most recent `updated_at`
- `keep_both` - Create a copy of the local version with a new ID
- `user_choice` - Use explicitly chosen version (`local` or `remote`)

**Response:**
```json
{
  "success": true,
  "resolved_at": "2024-01-01T02:00:00Z",
  "message": "Conflict resolved using last_write_wins strategy"
}
```

**Error Codes:**
- `400` - Invalid strategy or missing parameters
- `401` - Unauthorized
- `404` - Conflict not found
- `500` - Internal server error

---

## Authentication Endpoints

### Verify Email

Verify a user's email address using a verification token.

**Endpoint:** `POST /api/auth/verify-email`

**Request Body:**
```json
{
  "token": "verification-token-from-email"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Email verified successfully"
}
```

**Error Codes:**
- `400` - Invalid or missing token
- `404` - Token not found or expired
- `500` - Internal server error

---

### Resend Verification Email

Request a new verification email.

**Endpoint:** `POST /api/auth/resend-verification`

**Request Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "message": "Verification email sent"
}
```

**Error Codes:**
- `400` - Email already verified
- `401` - Unauthorized
- `500` - Internal server error

---

### Request Password Reset

Request a password reset email.

**Endpoint:** `POST /api/auth/request-password-reset`

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "success": true,
  "message": "If the email exists, a password reset link has been sent"
}
```

**Notes:**
- Always returns success for security (doesn't reveal if email exists)
- Reset token expires after 1 hour

**Error Codes:**
- `400` - Invalid email format
- `429` - Rate limit exceeded
- `500` - Internal server error

---

### Reset Password

Reset password using a reset token.

**Endpoint:** `POST /api/auth/reset-password`

**Request Body:**
```json
{
  "token": "reset-token-from-email",
  "new_password": "NewSecurePassword123!"
}
```

**Password Requirements:**
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number

**Response:**
```json
{
  "success": true,
  "message": "Password reset successfully"
}
```

**Notes:**
- Invalidates all existing sessions
- User must log in again with new password

**Error Codes:**
- `400` - Invalid token or weak password
- `404` - Token not found or expired
- `500` - Internal server error

---

## Data Models

### ConnectionTemplate

```typescript
interface ConnectionTemplate {
  id: string;
  name: string;
  type: 'postgres' | 'mysql' | 'sqlite' | 'mongodb' | string;
  host?: string;
  port?: number;
  database: string;
  username?: string;
  use_ssh?: boolean;
  ssh_host?: string;
  ssh_port?: number;
  ssh_user?: string;
  color?: string;
  icon?: string;
  metadata?: Record<string, string>;
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
  sync_version: number;
}
```

**Important:** Never include `password` or `ssh_key` fields. The API will reject uploads containing credentials.

---

### SavedQuery

```typescript
interface SavedQuery {
  id: string;
  name: string;
  description?: string;
  query: string;
  connection_id?: string;
  tags?: string[];
  favorite: boolean;
  metadata?: Record<string, string>;
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
  sync_version: number;
}
```

---

### QueryHistory

```typescript
interface QueryHistory {
  id: string;
  query: string;
  connection_id: string;
  executed_at: string; // ISO 8601
  duration_ms: number;
  rows_affected: number;
  success: boolean;
  error?: string;
  metadata?: Record<string, string>;
  sync_version: number;
}
```

---

## Error Response Format

All errors follow this format:

```json
{
  "error": true,
  "message": "Human-readable error message",
  "code": "ERROR_CODE"
}
```

**Common Error Codes:**
- `INVALID_REQUEST` - Malformed request
- `UNAUTHORIZED` - Missing or invalid authentication
- `FORBIDDEN` - Insufficient permissions
- `NOT_FOUND` - Resource not found
- `CONFLICT` - Resource conflict
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `PAYLOAD_TOO_LARGE` - Request body too large
- `INTERNAL_ERROR` - Server error

---

## Best Practices

### 1. Incremental Sync

Always sync incrementally using the `last_sync_at` timestamp:

```typescript
const lastSync = localStorage.getItem('last_sync');
const since = lastSync ? new Date(lastSync) : new Date(Date.now() - 30 * 24 * 60 * 60 * 1000);

const response = await fetch(`/api/sync/download?since=${since.toISOString()}&device_id=${deviceId}`);
```

### 2. Conflict Handling

Always check for conflicts after upload:

```typescript
const uploadResponse = await syncUpload(changes);

if (uploadResponse.conflicts.length > 0) {
  // Show conflict resolution UI to user
  await handleConflicts(uploadResponse.conflicts);
}
```

### 3. Batch Changes

Batch multiple changes in a single upload request:

```typescript
const changes = [
  { item_type: 'connection', action: 'create', ... },
  { item_type: 'saved_query', action: 'update', ... },
  // ... more changes
];

await syncUpload(changes);
```

### 4. Offline Support

Store failed uploads and retry when online:

```typescript
try {
  await syncUpload(changes);
} catch (error) {
  if (error.code === 'NETWORK_ERROR') {
    // Store for later retry
    await queueForRetry(changes);
  }
}
```

### 5. Rate Limit Handling

Implement exponential backoff for rate limits:

```typescript
async function syncWithRetry(changes, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      return await syncUpload(changes);
    } catch (error) {
      if (error.status === 429 && i < retries - 1) {
        await sleep(Math.pow(2, i) * 1000); // Exponential backoff
        continue;
      }
      throw error;
    }
  }
}
```

---

## Examples

### Complete Sync Flow

```typescript
class SyncManager {
  async performSync() {
    const deviceId = this.getDeviceId();
    const lastSync = this.getLastSyncTime();

    // 1. Download remote changes
    const downloadResponse = await fetch(
      `/api/sync/download?since=${lastSync.toISOString()}&device_id=${deviceId}`,
      {
        headers: { 'Authorization': `Bearer ${this.token}` }
      }
    );
    const remoteChanges = await downloadResponse.json();

    // 2. Apply remote changes locally
    await this.applyRemoteChanges(remoteChanges);

    // 3. Collect local changes
    const localChanges = await this.collectLocalChanges();

    // 4. Upload local changes
    const uploadResponse = await fetch('/api/sync/upload', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        device_id: deviceId,
        last_sync_at: lastSync,
        changes: localChanges,
      }),
    });
    const uploadResult = await uploadResponse.json();

    // 5. Handle conflicts
    if (uploadResult.conflicts.length > 0) {
      await this.resolveConflicts(uploadResult.conflicts);
    }

    // 6. Update last sync time
    this.setLastSyncTime(uploadResult.synced_at);
  }

  async resolveConflicts(conflicts) {
    for (const conflict of conflicts) {
      // Show UI to user for resolution
      const resolution = await this.showConflictUI(conflict);

      // Resolve conflict
      await fetch(`/api/sync/conflicts/${conflict.id}/resolve`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(resolution),
      });
    }
  }
}
```

---

## Support

For API support:
- Email: api-support@sqlstudio.io
- Documentation: https://docs.sqlstudio.io
- Status Page: https://status.sqlstudio.io
