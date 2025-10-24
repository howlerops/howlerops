# Organization API Integration Guide

This document describes how the organization API endpoints are integrated into the SQL Studio backend server and provides examples for testing and using the API.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Authentication Flow](#authentication-flow)
- [API Endpoints](#api-endpoints)
- [Request Examples](#request-examples)
- [Response Examples](#response-examples)
- [Error Codes](#error-codes)
- [Testing Instructions](#testing-instructions)

## Architecture Overview

### Component Structure

```
cmd/server/main.go
├── Creates OrganizationStore (Turso)
├── Creates OrganizationService
├── Creates AuthMiddleware
└── Initializes HTTPServer

internal/server/http.go
├── Creates HTTP router (gorilla/mux)
├── Registers organization routes
└── Applies auth middleware

internal/server/organization_routes.go
├── Creates OrganizationHandler
├── Wraps routes with HTTPAuthMiddleware
└── Calls handler.RegisterRoutes()

internal/organization/handlers.go
└── Implements 15 HTTP endpoints

internal/middleware/http_auth.go
└── Validates JWT and adds user context
```

### Request Flow

1. **Client** → HTTP Request with `Authorization: Bearer <JWT>`
2. **HTTPAuthMiddleware** → Extract & validate JWT token
3. **Context** → Add user_id, username, role to request context
4. **Handler** → Extract user info from context
5. **Service** → Business logic & permission checks
6. **Repository** → Database operations
7. **Response** → JSON response to client

## Authentication Flow

### 1. Get JWT Token

All organization endpoints require authentication. First, obtain a JWT token:

```bash
# Register a new user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "username": "johndoe"
  }'

# Login to get JWT token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'

# Response:
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

### 2. Use JWT Token

Include the JWT token in the `Authorization` header for all subsequent requests:

```bash
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## API Endpoints

### Organization Management

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/organizations` | Create organization | Yes |
| GET | `/api/organizations` | List user's organizations | Yes |
| GET | `/api/organizations/:id` | Get organization details | Yes (member) |
| PUT | `/api/organizations/:id` | Update organization | Yes (owner/admin) |
| DELETE | `/api/organizations/:id` | Delete organization | Yes (owner) |

### Member Management

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/organizations/:id/members` | List members | Yes (member) |
| PUT | `/api/organizations/:id/members/:userId` | Update member role | Yes (owner/admin) |
| DELETE | `/api/organizations/:id/members/:userId` | Remove member | Yes (owner/admin) |

### Invitation Management

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/organizations/:id/invitations` | Create invitation | Yes (owner/admin) |
| GET | `/api/organizations/:id/invitations` | List org invitations | Yes (owner/admin) |
| DELETE | `/api/organizations/:id/invitations/:inviteId` | Revoke invitation | Yes (owner/admin) |
| GET | `/api/invitations?email=` | List user invitations | Yes |
| POST | `/api/invitations/:token/accept` | Accept invitation | Yes |
| POST | `/api/invitations/:token/decline` | Decline invitation | Optional |

### Audit Logs

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/organizations/:id/audit-logs` | Get audit logs | Yes (owner/admin) |

## Request Examples

### Create Organization

```bash
curl -X POST http://localhost:8080/api/organizations \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "description": "We make great products"
  }'
```

### List Organizations

```bash
curl -X GET http://localhost:8080/api/organizations \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### Get Organization

```bash
curl -X GET http://localhost:8080/api/organizations/org_123 \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### Update Organization

```bash
curl -X PUT http://localhost:8080/api/organizations/org_123 \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp (Updated)",
    "description": "Updated description",
    "max_members": 20
  }'
```

### Create Invitation

```bash
curl -X POST http://localhost:8080/api/organizations/org_123/invitations \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newmember@example.com",
    "role": "member"
  }'
```

### List Members

```bash
curl -X GET http://localhost:8080/api/organizations/org_123/members \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### Update Member Role

```bash
curl -X PUT http://localhost:8080/api/organizations/org_123/members/user_456 \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "admin"
  }'
```

### Get Audit Logs

```bash
curl -X GET "http://localhost:8080/api/organizations/org_123/audit-logs?limit=50&offset=0" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

## Response Examples

### Success Response (Create Organization)

```json
{
  "id": "org_abc123",
  "name": "Acme Corporation",
  "description": "We make great products",
  "owner_id": "user_xyz789",
  "max_members": 10,
  "settings": {},
  "created_at": "2025-10-23T10:30:00Z",
  "updated_at": "2025-10-23T10:30:00Z"
}
```

### Success Response (List Organizations)

```json
{
  "organizations": [
    {
      "id": "org_abc123",
      "name": "Acme Corporation",
      "description": "We make great products",
      "owner_id": "user_xyz789",
      "max_members": 10,
      "member_count": 3,
      "settings": {},
      "created_at": "2025-10-23T10:30:00Z",
      "updated_at": "2025-10-23T10:30:00Z"
    }
  ],
  "count": 1
}
```

### Success Response (List Members)

```json
{
  "members": [
    {
      "organization_id": "org_abc123",
      "user_id": "user_xyz789",
      "role": "owner",
      "joined_at": "2025-10-23T10:30:00Z"
    },
    {
      "organization_id": "org_abc123",
      "user_id": "user_def456",
      "role": "member",
      "invited_by": "user_xyz789",
      "joined_at": "2025-10-23T11:00:00Z"
    }
  ],
  "count": 2
}
```

### Error Response

```json
{
  "error": true,
  "message": "insufficient permissions"
}
```

## Error Codes

| HTTP Code | Description | Common Causes |
|-----------|-------------|---------------|
| 400 | Bad Request | Invalid request body, missing required fields |
| 401 | Unauthorized | Missing or invalid JWT token |
| 403 | Forbidden | User doesn't have required permissions |
| 404 | Not Found | Organization, invitation, or member not found |
| 409 | Conflict | Duplicate organization name, already a member |
| 500 | Internal Server Error | Database error, unexpected server error |

### Common Error Messages

- `"missing authorization header"` - No JWT token provided
- `"invalid or expired token"` - JWT token is invalid or expired
- `"not a member of this organization"` - User is not a member
- `"insufficient permissions"` - User doesn't have required role
- `"organization has reached maximum member limit"` - Cannot add more members
- `"invitation already exists for this email"` - Duplicate invitation
- `"cannot remove owner from organization"` - Owner cannot be removed

## Testing Instructions

### Automated Testing

Run the comprehensive API test suite:

```bash
# Make the test script executable
chmod +x backend-go/scripts/test-organization-api.sh

# Run all tests
cd backend-go
./scripts/test-organization-api.sh

# View test results
cat test-results/test_*.log
```

The test script will:
1. Register and login a test user
2. Create an organization
3. Test all 15 endpoints
4. Verify authentication is required
5. Generate a detailed test report

### Database Verification

Verify the database schema and data:

```bash
# Make the verification script executable
chmod +x backend-go/scripts/check-org-tables.sh

# Set environment variables (or use .env file)
export TURSO_URL="libsql://your-database.turso.io"
export TURSO_AUTH_TOKEN="your-auth-token"

# Run verification
./scripts/check-org-tables.sh
```

### Manual Testing

1. **Start the server:**
   ```bash
   cd backend-go
   make dev
   ```

2. **Register and login:**
   ```bash
   # Register
   curl -X POST http://localhost:8080/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"Test123!","username":"testuser"}'

   # Login and save token
   JWT_TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"Test123!"}' | \
     jq -r '.access_token')

   echo "JWT Token: $JWT_TOKEN"
   ```

3. **Test organization endpoints:**
   ```bash
   # Create organization
   ORG_ID=$(curl -s -X POST http://localhost:8080/api/organizations \
     -H "Authorization: Bearer $JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"name":"Test Org","description":"Testing"}' | \
     jq -r '.id')

   echo "Organization ID: $ORG_ID"

   # List organizations
   curl -X GET http://localhost:8080/api/organizations \
     -H "Authorization: Bearer $JWT_TOKEN" | jq

   # Get organization
   curl -X GET http://localhost:8080/api/organizations/$ORG_ID \
     -H "Authorization: Bearer $JWT_TOKEN" | jq
   ```

## Permission Matrix

| Endpoint | Owner | Admin | Member | Non-Member |
|----------|-------|-------|--------|------------|
| Create Organization | ✓ | ✓ | ✓ | ✓ |
| List Organizations | ✓ | ✓ | ✓ | ✓ |
| Get Organization | ✓ | ✓ | ✓ | ✗ |
| Update Organization | ✓ | ✓ | ✗ | ✗ |
| Delete Organization | ✓ | ✗ | ✗ | ✗ |
| List Members | ✓ | ✓ | ✓ | ✗ |
| Update Member Role | ✓ | ✓* | ✗ | ✗ |
| Remove Member | ✓ | ✓* | ✗ | ✗ |
| Create Invitation | ✓ | ✓* | ✗ | ✗ |
| List Invitations | ✓ | ✓ | ✗ | ✗ |
| Revoke Invitation | ✓ | ✓ | ✗ | ✗ |
| Accept Invitation | ✓ | ✓ | ✓ | ✓ |
| Decline Invitation | ✓ | ✓ | ✓ | ✓ |
| Get Audit Logs | ✓ | ✓ | ✗ | ✗ |

\* Admins have restricted permissions (e.g., cannot promote to admin, cannot remove admins)

## Integration Checklist

- [x] HTTP auth middleware created (`internal/middleware/http_auth.go`)
- [x] Organization routes registered (`internal/server/organization_routes.go`)
- [x] HTTP server updated to include auth middleware
- [x] Organization service added to Services struct
- [x] Organization store initialized in main.go
- [x] All 15 endpoints implemented and registered
- [x] Authentication required for all endpoints
- [x] Permission checks implemented
- [x] Audit logging integrated
- [x] Error handling implemented
- [x] CORS support enabled
- [x] API testing script created
- [x] Database verification script created
- [x] Integration documentation created

## Troubleshooting

### Server won't start

1. Check if port 8080 is available: `lsof -i :8080`
2. Verify Turso credentials in `.env` file
3. Check logs for specific errors

### Authentication fails

1. Verify JWT token is not expired
2. Check if token is properly formatted: `Bearer <token>`
3. Verify JWT secret matches between registration and validation

### Database errors

1. Run schema initialization: `make migrate` or check `turso.InitializeSchema()`
2. Verify Turso connection: `turso db shell <database-name>`
3. Check if required tables exist: `./scripts/check-org-tables.sh`

### Permission denied errors

1. Verify user is a member of the organization
2. Check user's role in the organization
3. Review permission matrix above

## Next Steps

1. **Frontend Integration**: Use these endpoints in your React/TypeScript frontend
2. **WebSocket Support**: Add real-time organization updates via WebSocket
3. **Email Notifications**: Send invitation emails via Resend
4. **Rate Limiting**: Add rate limiting for invitation creation
5. **Analytics**: Track organization metrics and usage
6. **Webhooks**: Add webhook support for organization events

## Support

For issues or questions:
- Check the test results in `test-results/` directory
- Review the server logs
- Run the database verification script
- Consult the main API documentation
