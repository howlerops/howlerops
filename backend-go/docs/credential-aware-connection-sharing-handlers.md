# Credential-Aware Connection Sharing API Handlers

## Overview

This document describes the new API handlers added to `/Users/jacob_1/projects/howlerops/backend-go/internal/connections/handler.go` for credential-aware connection sharing with Organization Encryption Key (OEK) support.

## New Request Types

### ShareConnectionWithCredentialRequest

```go
type ShareConnectionWithCredentialRequest struct {
    OrganizationID     string `json:"organization_id" validate:"required"`
    EncryptedMasterKey string `json:"encrypted_master_key"` // Base64-encoded, encrypted with session key
    Password           string `json:"password"`             // Already decrypted by frontend (temporary in memory)
}
```

Used for sharing connections with password re-encryption for organizational access.

## New API Endpoints

### 1. POST /api/connections/{id}/share-with-credential

**Handler:** `ShareConnectionWithCredential`

**Purpose:** Share a connection with an organization and re-encrypt the password with the Organization Encryption Key (OEK).

**Request:**
- Path parameter: `id` (connection ID)
- Body: `ShareConnectionWithCredentialRequest`
- Context: `userID` from auth middleware

**Response:**
- Success (200): `{"success": true, "message": "Connection shared successfully with credential"}`
- Error codes:
  - 400: Invalid request (missing ID, invalid body, invalid base64 encoding)
  - 401: User not authenticated
  - 403: Insufficient permissions or not organization member
  - 404: Connection not found
  - 500: Internal server error

**Security Features:**
- Decodes master key from base64
- Uses `defer crypto.ClearBytes()` to clear sensitive data from memory
- Validates organization membership and permissions
- Calls `service.ShareConnectionWithCredential()` for business logic

### 2. GET /api/connections/{id}/password

**Handler:** `GetConnectionPassword`

**Purpose:** Retrieve the decrypted password for a connection (personal or shared).

**Request:**
- Path parameter: `id` (connection ID)
- Query parameter: `org_id` (optional, for shared connections)
- Header: `X-Master-Key` (Base64-encoded user master key)
- Context: `userID` from auth middleware

**Response:**
- Success (200): `{"password": "<base64-encoded-password>"}`
- Error codes:
  - 400: Invalid request (missing ID, missing/invalid master key header)
  - 401: User not authenticated
  - 403: Insufficient permissions or not organization member
  - 404: Connection not found
  - 500: Internal server error

**Security Features:**
- Decodes master key from base64 header
- Uses `defer crypto.ClearBytes()` to clear sensitive data from memory
- Returns password as base64-encoded for secure transmission
- Validates organization membership for shared connections

## Route Registration

A new `RegisterRoutes` method has been added to properly register all connection endpoints:

```go
func (h *Handler) RegisterRoutes(r *mux.Router) {
    // Connection CRUD
    r.HandleFunc("/api/connections", h.CreateConnection).Methods("POST")
    r.HandleFunc("/api/connections", h.GetAccessibleConnections).Methods("GET")
    r.HandleFunc("/api/connections/{id}", h.UpdateConnection).Methods("PUT")
    r.HandleFunc("/api/connections/{id}", h.DeleteConnection).Methods("DELETE")

    // Connection sharing
    r.HandleFunc("/api/connections/{id}/share", h.ShareConnection).Methods("POST")
    r.HandleFunc("/api/connections/{id}/share-with-credential", h.ShareConnectionWithCredential).Methods("POST")
    r.HandleFunc("/api/connections/{id}/unshare", h.UnshareConnection).Methods("POST")
    r.HandleFunc("/api/connections/{id}/password", h.GetConnectionPassword).Methods("GET")

    // Organization connections
    r.HandleFunc("/api/organizations/{org_id}/connections", h.GetOrganizationConnections).Methods("GET")
}
```

## Dependencies

### New Imports Added

```go
import (
    "encoding/base64"  // For encoding/decoding master keys and passwords
    "github.com/jbeck018/howlerops/backend-go/pkg/crypto"  // For ClearBytes() security
)
```

### Service Layer Integration

Both handlers rely on existing service methods:

1. **ShareConnectionWithCredential handler** calls:
   - `h.service.ShareConnectionWithCredential(ctx, connID, userID, orgID, masterKey, password)`

2. **GetConnectionPassword handler** calls:
   - `h.service.GetConnectionPassword(ctx, connID, userID, orgID, masterKey)`

These service methods are already implemented in `/Users/jacob_1/projects/howlerops/backend-go/internal/connections/service.go`.

## Security Considerations

### Memory Safety
- All sensitive data (master keys, passwords) are cleared from memory using `defer crypto.ClearBytes()`
- Ensures no sensitive data remains in memory after request completion

### Authentication & Authorization
- User authentication verified via middleware context
- Organization membership validated for shared connections
- Permission checks enforce organization RBAC

### Data Encoding
- Master keys transmitted as Base64 in headers/request body
- Passwords returned as Base64 for secure transmission
- Proper validation of Base64 encoding with error handling

## Error Handling

All handlers follow the existing error handling patterns:

1. Extract error messages from service layer
2. Map to appropriate HTTP status codes
3. Log errors with context (connection ID, user ID, org ID)
4. Return consistent JSON error responses using `respondError()`

## Testing

The handlers follow the existing pattern from `/Users/jacob_1/projects/howlerops/backend-go/internal/organization/handlers.go`:

- Use `middleware.UserIDKey` for user context
- Follow DRY principles with `respondJSON()` and `respondError()` helpers
- Match error status code patterns from existing handlers
- Include proper field validation

## Files Modified

- `/Users/jacob_1/projects/howlerops/backend-go/internal/connections/handler.go`
  - Added `ShareConnectionWithCredentialRequest` type
  - Added `ShareConnectionWithCredential` handler
  - Added `GetConnectionPassword` handler
  - Added `RegisterRoutes` method
  - Added imports for `encoding/base64` and `pkg/crypto`

## Integration Notes

To integrate these handlers into the application:

1. Add a `Connections` field to the `Services` struct in `/Users/jacob_1/projects/howlerops/backend-go/internal/services/services.go`
2. Initialize the connections service in the services constructor
3. Register the routes in `/Users/jacob_1/projects/howlerops/backend-go/internal/server/http.go`:

```go
if svc.Connections != nil {
    logger.Info("Registering Connection HTTP routes")
    connHandler := connections.NewHandler(svc.Connections, logger)
    connHandler.RegisterRoutes(mainRouter)
    logger.Info("Connection HTTP routes registered successfully")
}
```

## Compliance

These handlers implement the credential-aware connection sharing design as specified in:
- `/Users/jacob_1/projects/howlerops/backend-go/docs/credential-sharing-design.md`
- `/Users/jacob_1/projects/howlerops/backend-go/docs/credential-sharing-implementation-checklist.md`
