# Quick Start: Organization API

Get up and running with the organization API in 5 minutes.

## Prerequisites

- Go 1.21+
- Turso database configured
- `.env` file with Turso credentials

## 1. Start the Server (30 seconds)

```bash
cd backend-go
make dev
```

**Expected output:**
```
INFO Organization service initialized
INFO Organization HTTP routes registered successfully
INFO All servers started successfully
```

**Server URLs:**
- API: http://localhost:8080
- Health: http://localhost:8080/health

## 2. Test Authentication (1 minute)

```bash
# Set base URL
export API_URL="http://localhost:8080"

# Register a user
curl -X POST $API_URL/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin123!",
    "username": "admin"
  }'

# Login and get token
export JWT_TOKEN=$(curl -s -X POST $API_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin123!"
  }' | jq -r '.access_token')

echo "JWT Token: $JWT_TOKEN"
```

## 3. Create Your First Organization (30 seconds)

```bash
# Create organization
export ORG_RESPONSE=$(curl -s -X POST $API_URL/api/organizations \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Organization",
    "description": "Testing organization API"
  }')

echo $ORG_RESPONSE | jq

# Extract organization ID
export ORG_ID=$(echo $ORG_RESPONSE | jq -r '.id')
echo "Organization ID: $ORG_ID"
```

## 4. Invite a Member (1 minute)

```bash
# Create invitation
curl -X POST $API_URL/api/organizations/$ORG_ID/invitations \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newmember@example.com",
    "role": "member"
  }' | jq
```

## 5. List Members (10 seconds)

```bash
curl -X GET $API_URL/api/organizations/$ORG_ID/members \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

## 6. View Audit Logs (10 seconds)

```bash
curl -X GET "$API_URL/api/organizations/$ORG_ID/audit-logs?limit=10" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

## 7. Run Automated Tests (2 minutes)

```bash
cd backend-go
./scripts/test-organization-api.sh
```

## All 15 Endpoints at a Glance

```bash
# Organization CRUD
POST   /api/organizations                              # Create
GET    /api/organizations                              # List
GET    /api/organizations/:id                          # Get
PUT    /api/organizations/:id                          # Update
DELETE /api/organizations/:id                          # Delete

# Members
GET    /api/organizations/:id/members                  # List
PUT    /api/organizations/:id/members/:userId          # Update role
DELETE /api/organizations/:id/members/:userId          # Remove

# Invitations
POST   /api/organizations/:id/invitations              # Create
GET    /api/organizations/:id/invitations              # List (org)
DELETE /api/organizations/:id/invitations/:inviteId    # Revoke
GET    /api/invitations?email=                         # List (user)
POST   /api/invitations/:token/accept                  # Accept
POST   /api/invitations/:token/decline                 # Decline

# Audit
GET    /api/organizations/:id/audit-logs               # Get logs
```

## Common Tasks

### Update Organization

```bash
curl -X PUT $API_URL/api/organizations/$ORG_ID \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Organization Name",
    "max_members": 20
  }' | jq
```

### Update Member Role

```bash
# Promote member to admin
curl -X PUT $API_URL/api/organizations/$ORG_ID/members/$USER_ID \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}' | jq
```

### Accept Invitation

```bash
# As the invited user (with their JWT)
curl -X POST $API_URL/api/invitations/$INVITE_TOKEN/accept \
  -H "Authorization: Bearer $INVITED_USER_JWT" | jq
```

## Troubleshooting

### "Server not running"
```bash
# Check if port is in use
lsof -i :8080

# Restart server
cd backend-go && make dev
```

### "Unauthorized"
```bash
# Get a fresh token
export JWT_TOKEN=$(curl -s -X POST $API_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"Admin123!"}' | \
  jq -r '.access_token')
```

### "Table not found"
```bash
# Verify database schema
./scripts/check-org-tables.sh
```

## Next Steps

1. **Full Documentation**: Read `ORGANIZATION_API_INTEGRATION.md`
2. **Integration Summary**: See `ORGANIZATION_INTEGRATION_SUMMARY.md`
3. **Frontend Integration**: Use these endpoints in your React app
4. **Production Deploy**: Configure Turso production database

## Quick Reference

**All requests require this header:**
```
Authorization: Bearer YOUR_JWT_TOKEN
```

**Base URL:**
```
http://localhost:8080  (development)
https://api.your-domain.com  (production)
```

**Response Format:**
```json
{
  "field": "value",
  ...
}
```

**Error Format:**
```json
{
  "error": true,
  "message": "error description"
}
```

## Files Location

- **HTTP Auth Middleware**: `internal/middleware/http_auth.go`
- **Organization Routes**: `internal/server/organization_routes.go`
- **Handlers**: `internal/organization/handlers.go`
- **Service**: `internal/organization/service.go`
- **Tests**: `scripts/test-organization-api.sh`
- **DB Check**: `scripts/check-org-tables.sh`

## Support

- Check logs: Server outputs detailed logs to console
- Test endpoints: Use `test-organization-api.sh` script
- Verify database: Use `check-org-tables.sh` script
- Full docs: See `ORGANIZATION_API_INTEGRATION.md`

---

**Total setup time**: ~5 minutes
**Status**: Ready for development
