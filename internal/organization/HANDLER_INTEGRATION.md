# Organization HTTP Handlers Integration Guide

## Overview
This document explains how to integrate the organization HTTP handlers into the main server.

## Files Created
1. `/Users/jacob_1/projects/sql-studio/backend-go/internal/organization/handlers.go` - HTTP handler implementations
2. `/Users/jacob_1/projects/sql-studio/backend-go/internal/organization/handlers_test.go` - Comprehensive test suite

## API Endpoints Implemented

### Organization Management
- `POST /api/organizations` - Create organization
- `GET /api/organizations` - List user's organizations
- `GET /api/organizations/:id` - Get organization details
- `PUT /api/organizations/:id` - Update organization
- `DELETE /api/organizations/:id` - Delete organization (soft delete)

### Member Management
- `GET /api/organizations/:id/members` - List members
- `PUT /api/organizations/:id/members/:userId` - Update member role
- `DELETE /api/organizations/:id/members/:userId` - Remove member

### Invitation Management
- `POST /api/organizations/:id/invitations` - Create invitation
- `GET /api/organizations/:id/invitations` - List org invitations
- `DELETE /api/organizations/:id/invitations/:inviteId` - Revoke invitation
- `GET /api/invitations` - List user's pending invitations (requires ?email= param)
- `POST /api/invitations/:id/accept` - Accept invitation (by token)
- `POST /api/invitations/:id/decline` - Decline invitation

### Audit Logs
- `GET /api/organizations/:id/audit-logs` - Get audit logs (owner/admin only)
  - Query params: `limit` (default 50, max 100), `offset` (default 0)

## Integration Steps

### 1. Initialize Organization Service
In your `cmd/server/main.go` or server initialization:

```go
import (
    "github.com/sql-studio/backend-go/internal/organization"
    "github.com/sql-studio/backend-go/pkg/storage/turso"
)

// After creating tursoClient and logger...

// Create organization repository
orgRepo := turso.NewOrganizationStore(tursoClient, appLogger)

// Create organization service
orgService := organization.NewService(orgRepo, appLogger)
```

### 2. Create HTTP Handler
```go
// Create organization handler
orgHandler := organization.NewHandler(orgService, appLogger)
```

### 3. Register Routes
Using gorilla/mux router (already in your project):

```go
// Create router (or use existing one)
router := mux.NewRouter()

// Apply auth middleware to protect routes
authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret, appLogger)

// Create subrouter for organization endpoints
orgRouter := router.PathPrefix("/api").Subrouter()

// Apply authentication middleware
// Note: You'll need to adapt the gRPC middleware to HTTP
// or create an HTTP-specific auth middleware

// Register organization routes
orgHandler.RegisterRoutes(orgRouter)
```

### 4. HTTP Auth Middleware Adapter (Required)

You'll need to create an HTTP middleware adapter for authentication since the current middleware is gRPC-focused. Create a file `internal/middleware/http_auth.go`:

```go
package middleware

import (
    "context"
    "net/http"
    "strings"
)

// HTTPAuthMiddleware wraps the existing auth middleware for HTTP requests
func (a *AuthMiddleware) HTTPAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "missing authorization header", http.StatusUnauthorized)
            return
        }

        // Extract bearer token
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
            return
        }

        token := parts[1]

        // Validate token
        claims, err := a.validateToken(token)
        if err != nil {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }

        // Add user info to context
        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "username", claims.Username)
        ctx = context.WithValue(ctx, "role", claims.Role)

        // Call next handler with updated context
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Then apply it:

```go
// Apply auth middleware to organization routes
orgRouter.Use(authMiddleware.HTTPAuthMiddleware)
orgHandler.RegisterRoutes(orgRouter)
```

### 5. Public Routes

Some routes should be accessible without authentication:
- `POST /api/invitations/:id/decline` - Can be called with or without auth

You may want to create conditional middleware or skip auth for specific routes.

## Request/Response Examples

### Create Organization
```bash
POST /api/organizations
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "name": "My Team",
  "description": "Our awesome team workspace"
}

Response (201 Created):
{
  "id": "org-abc123",
  "name": "My Team",
  "description": "Our awesome team workspace",
  "owner_id": "user-xyz",
  "created_at": "2025-10-23T10:00:00Z",
  "updated_at": "2025-10-23T10:00:00Z",
  "max_members": 10,
  "settings": {}
}
```

### List Organizations
```bash
GET /api/organizations
Authorization: Bearer <jwt-token>

Response (200 OK):
{
  "organizations": [
    {
      "id": "org-abc123",
      "name": "My Team",
      "owner_id": "user-xyz",
      ...
    }
  ],
  "count": 1
}
```

### Create Invitation
```bash
POST /api/organizations/org-abc123/invitations
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "email": "colleague@example.com",
  "role": "member"
}

Response (201 Created):
{
  "id": "inv-def456",
  "organization_id": "org-abc123",
  "email": "colleague@example.com",
  "role": "member",
  "invited_by": "user-xyz",
  "token": "secure-token-here",
  "expires_at": "2025-10-30T10:00:00Z",
  "created_at": "2025-10-23T10:00:00Z"
}
```

### Accept Invitation
```bash
POST /api/invitations/<token>/accept
Authorization: Bearer <jwt-token>

Response (200 OK):
{
  "success": true,
  "message": "invitation accepted successfully",
  "organization": {
    "id": "org-abc123",
    "name": "My Team",
    ...
  }
}
```

### Update Member Role
```bash
PUT /api/organizations/org-abc123/members/user-456
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "role": "admin"
}

Response (200 OK):
{
  "success": true,
  "message": "member role updated successfully"
}
```

### Get Audit Logs
```bash
GET /api/organizations/org-abc123/audit-logs?limit=20&offset=0
Authorization: Bearer <jwt-token>

Response (200 OK):
{
  "logs": [
    {
      "id": "log-123",
      "organization_id": "org-abc123",
      "user_id": "user-xyz",
      "action": "organization.created",
      "resource_type": "organization",
      "resource_id": "org-abc123",
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "details": {
        "organization_name": "My Team"
      },
      "created_at": "2025-10-23T10:00:00Z"
    }
  ],
  "count": 1,
  "limit": 20,
  "offset": 0
}
```

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": true,
  "message": "descriptive error message"
}
```

### HTTP Status Codes
- `200 OK` - Successful GET/PUT/DELETE operations
- `201 Created` - Successful POST operations
- `400 Bad Request` - Invalid input or validation errors
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Duplicate resource or conflict
- `500 Internal Server Error` - Server-side errors

## Security Features

### Authentication
- All endpoints (except decline invitation) require JWT authentication
- User ID is extracted from JWT token via middleware
- User context is stored in request context

### Authorization
- **Owner-only operations**: Delete organization
- **Owner/Admin operations**: Update organization, manage members, create/revoke invitations, view audit logs
- **Member operations**: View organization, view members
- Role-based access control is enforced in the service layer

### Audit Logging
Sensitive operations automatically create audit log entries with:
- User ID of actor
- Action performed
- Resource type and ID
- IP address (extracted from X-Forwarded-For, X-Real-IP, or RemoteAddr)
- User agent
- Operation details

Audit log actions include:
- `organization.created`
- `organization.updated`
- `organization.deleted`
- `member.role_updated`
- `member.removed`
- `invitation.created`
- `invitation.accepted`
- `invitation.declined`
- `invitation.revoked`

## Testing

Run the test suite:
```bash
# Run all organization tests
go test ./internal/organization/...

# Run only handler tests
go test -v ./internal/organization/ -run "TestCreate|TestList|TestGet|TestUpdate|TestDelete"

# Run with coverage
go test -cover ./internal/organization/...
```

## Next Steps

1. Create HTTP auth middleware adapter (see step 4 above)
2. Integrate organization handler into main server router
3. Test endpoints using Postman or curl
4. Add email notifications for invitations (optional)
5. Add rate limiting for invitation endpoints (optional)
6. Document API in OpenAPI/Swagger format (optional)

## Notes

- All operations use the service layer - handlers never call repository directly
- Audit logs are created asynchronously and failures don't block operations
- The service layer handles all business logic and authorization
- Input validation is performed at both handler and service layers
- IP address extraction handles proxy headers (X-Forwarded-For, X-Real-IP)
