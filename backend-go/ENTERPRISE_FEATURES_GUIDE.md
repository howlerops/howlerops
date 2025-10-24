# Enterprise Features Guide

This guide covers the enterprise-grade multi-tenancy and white-labeling features in SQL Studio.

## Table of Contents

1. [Multi-tenancy Architecture](#multi-tenancy-architecture)
2. [White-labeling Setup](#white-labeling-setup)
3. [Custom Domain Configuration](#custom-domain-configuration)
4. [Quota Management](#quota-management)
5. [SLA Monitoring](#sla-monitoring)
6. [API Reference](#api-reference)
7. [Security Best Practices](#security-best-practices)

---

## Multi-tenancy Architecture

### Overview

SQL Studio implements strict tenant isolation to ensure complete data separation between organizations. Every request is scoped to the user's organizations, preventing any data leakage.

### Key Components

1. **Tenant Isolation Middleware** (`internal/middleware/tenant_isolation.go`)
   - Automatically loads user's organizations on each request
   - Adds organization context to all downstream handlers
   - Provides helper functions for query filtering

2. **Organization Context**
   ```go
   // Get current organization ID
   orgID := middleware.GetCurrentOrgID(ctx)

   // Get all user's organizations
   orgs := middleware.GetUserOrganizationIDs(ctx)

   // Verify access to specific organization
   err := middleware.VerifyOrgAccess(ctx, orgID)
   ```

3. **Query Filtering**
   ```go
   // Automatically filter queries by organization
   whereClause, args := middleware.BuildOrgFilterQuery(ctx, "organization_id")
   query := "SELECT * FROM connections WHERE " + whereClause
   ```

### Database Schema

All tenant-scoped tables must include an `organization_id` column with a foreign key:

```sql
CREATE TABLE connections (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    -- ... other fields
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX idx_connections_org ON connections(organization_id);
```

### Switching Organizations

Users can switch between organizations using:
- `X-Organization-ID` HTTP header
- `org_id` query parameter

```bash
# Using header
curl -H "X-Organization-ID: org-123" https://api.example.com/connections

# Using query parameter
curl https://api.example.com/connections?org_id=org-123
```

---

## White-labeling Setup

### Configuration

White-labeling allows organizations to customize branding, colors, and appearance.

### API Endpoints

#### Get White-label Configuration
```http
GET /api/organizations/{id}/white-label
```

Response:
```json
{
  "organization_id": "org-123",
  "custom_domain": "app.customer.com",
  "logo_url": "https://cdn.customer.com/logo.png",
  "favicon_url": "https://cdn.customer.com/favicon.ico",
  "primary_color": "#1E40AF",
  "secondary_color": "#64748B",
  "accent_color": "#8B5CF6",
  "company_name": "Acme Corp",
  "support_email": "support@acme.com",
  "hide_branding": false
}
```

#### Update White-label Configuration
```http
PUT /api/organizations/{id}/white-label
Content-Type: application/json

{
  "primary_color": "#FF5733",
  "company_name": "Acme Corp",
  "hide_branding": true
}
```

#### Get Branded CSS
```http
GET /api/white-label/css?domain=app.customer.com
```

Returns dynamically generated CSS based on organization's branding settings.

### Frontend Integration

```typescript
// Load white-label config
const response = await fetch('/api/white-label/css?domain=' + window.location.hostname);
const css = await response.text();

// Inject CSS
const style = document.createElement('style');
style.textContent = css;
document.head.appendChild(style);
```

### Customization Options

| Field | Description | Format |
|-------|-------------|--------|
| `logo_url` | Company logo | PNG/SVG URL |
| `favicon_url` | Browser favicon | ICO/PNG URL (32x32px) |
| `primary_color` | Primary brand color | Hex (#RRGGBB) |
| `secondary_color` | Secondary color | Hex (#RRGGBB) |
| `accent_color` | Accent/highlight color | Hex (#RRGGBB) |
| `company_name` | Company name displayed | Text |
| `support_email` | Support contact email | Email address |
| `custom_css` | Additional CSS overrides | CSS text |
| `hide_branding` | Hide "Powered by SQL Studio" | Boolean |

---

## Custom Domain Configuration

### Adding a Custom Domain

1. **Initiate Domain Verification**
   ```http
   POST /api/organizations/{id}/domains
   Content-Type: application/json

   {
     "domain": "app.customer.com"
   }
   ```

2. **Add DNS Record**

   Response includes DNS verification instructions:
   ```json
   {
     "verification": {
       "domain": "app.customer.com",
       "dns_record_type": "TXT",
       "dns_record_name": "_sql-studio-verification.app.customer.com",
       "dns_record_value": "abc123def456..."
     },
     "instructions": "To verify ownership of app.customer.com..."
   }
   ```

3. **Configure DNS**

   Add the TXT record to your DNS provider:
   ```
   Type: TXT
   Name: _sql-studio-verification.app.customer.com
   Value: abc123def456...
   ```

4. **Verify Domain**
   ```http
   POST /api/organizations/{id}/domains/{domain}/verify
   ```

### DNS Provider Examples

#### Cloudflare
1. Go to DNS settings
2. Add record: Type=TXT, Name=`_sql-studio-verification`, Value=`<token>`
3. Save and wait 1-5 minutes

#### AWS Route53
```bash
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch '{
    "Changes": [{
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "_sql-studio-verification.app.customer.com",
        "Type": "TXT",
        "TTL": 300,
        "ResourceRecords": [{"Value": "\"<token>\""}]
      }
    }]
  }'
```

#### GoDaddy
1. DNS Management
2. Add TXT record
3. Host: `_sql-studio-verification`
4. TXT Value: `<token>`

### List Domains
```http
GET /api/organizations/{id}/domains
```

### Remove Domain
```http
DELETE /api/organizations/{id}/domains/{domain}
```

---

## Quota Management

### Resource Types

- **Connections**: Maximum database connections
- **Queries**: Daily query limit
- **Storage**: Maximum storage in MB
- **API Calls**: Hourly API request limit
- **Concurrent Queries**: Maximum simultaneous queries
- **Team Members**: Maximum organization members

### Get Quotas
```http
GET /api/organizations/{id}/quotas
```

Response:
```json
{
  "organization_id": "org-123",
  "max_connections": 20,
  "max_queries_per_day": 5000,
  "max_storage_mb": 500,
  "max_api_calls_per_hour": 1000,
  "max_concurrent_queries": 10,
  "max_team_members": 10,
  "features_enabled": "enterprise"
}
```

### Update Quotas
```http
PUT /api/organizations/{id}/quotas
Content-Type: application/json

{
  "max_connections": 50,
  "max_queries_per_day": 10000,
  "features_enabled": "enterprise"
}
```

### Usage Statistics
```http
GET /api/organizations/{id}/usage?days=30
```

Response:
```json
{
  "organization_id": "org-123",
  "period": "last_30_days",
  "total_queries": 45000,
  "total_api_calls": 12000,
  "average_queries_per_day": 1500,
  "average_api_calls_per_day": 400,
  "peak_concurrent_queries": 8,
  "current_storage_used_mb": 250.5,
  "daily_usage": [
    {
      "date": "2025-10-24",
      "queries_count": 1600,
      "api_calls_count": 420,
      "storage_used_mb": 250.5
    }
  ],
  "quota_limits": {
    "max_connections": 50,
    "max_queries_per_day": 10000
  }
}
```

### Export Usage Data
```http
GET /api/organizations/{id}/usage/export?days=30
```

Returns CSV file with daily usage data.

### Quota Enforcement

Quotas are automatically enforced by:
1. **Middleware**: Per-organization rate limiting
2. **Service Layer**: Pre-flight quota checks
3. **Background Jobs**: Usage tracking

When quota is exceeded, API returns:
```json
{
  "error": "Rate limit exceeded",
  "message": "Daily query quota exceeded (10000/10000)",
  "retry_after": 3600
}
```

HTTP Headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 245
X-RateLimit-Reset: 1698235200
Retry-After: 3600
```

---

## SLA Monitoring

### Overview

SLA monitoring tracks uptime, response times, and error rates per organization.

### Metrics Tracked

- **Uptime Percentage**: % of successful requests
- **Average Response Time**: Mean response time in milliseconds
- **Error Rate**: % of failed requests (5xx errors)
- **P95 Response Time**: 95th percentile response time
- **P99 Response Time**: 99th percentile response time

### Get SLA Metrics
```http
GET /api/organizations/{id}/sla?days=30
```

Response:
```json
[
  {
    "organization_id": "org-123",
    "metric_date": "2025-10-24T00:00:00Z",
    "uptime_percentage": 99.95,
    "avg_response_time_ms": 245.5,
    "error_rate": 0.05,
    "p95_response_time_ms": 450.0,
    "p99_response_time_ms": 800.0,
    "total_requests": 10000,
    "failed_requests": 5
  }
]
```

### SLA Report
```http
GET /api/organizations/{id}/sla/report?days=30&target_uptime=99.9&target_response_time=500
```

Response:
```json
{
  "organization_id": "org-123",
  "start_date": "2025-09-24T00:00:00Z",
  "end_date": "2025-10-24T00:00:00Z",
  "overall_uptime": 99.92,
  "avg_response_time": 235.8,
  "overall_error_rate": 0.08,
  "total_requests": 300000,
  "failed_requests": 240,
  "sla_compliance": {
    "target_uptime": 99.9,
    "actual_uptime": 99.92,
    "compliant_days": 28,
    "non_compliant_days": 2,
    "compliance_rate": 93.33,
    "target_response_time": 500,
    "actual_response_time": 235.8,
    "response_time_ok": true
  },
  "worst_days": [
    {
      "metric_date": "2025-10-15T00:00:00Z",
      "uptime_percentage": 98.5,
      "avg_response_time_ms": 650.0
    }
  ]
}
```

### Automatic Tracking

SLA tracking is automatic via middleware:

```go
// In your main router setup
router.Use(slaTracking.Track)
```

All requests are logged with:
- Response time
- Status code
- Success/failure
- Organization ID

### Daily Aggregation

SLA metrics are calculated daily at 1 AM via background scheduler:

```go
// Start SLA scheduler
slaMonitor.StartScheduler(ctx)
slaMonitor.StartCleanupScheduler(ctx, 30) // Keep 30 days of logs
```

---

## API Reference

### Authentication

All endpoints require authentication via JWT token:

```
Authorization: Bearer <token>
```

### Organization Context

Specify organization using header or query parameter:

```
X-Organization-ID: org-123
```

or

```
?org_id=org-123
```

### Complete Endpoint List

#### White-labeling
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/organizations/{id}/white-label` | Get config |
| PUT | `/api/organizations/{id}/white-label` | Update config |
| GET | `/api/white-label/css` | Get branded CSS |

#### Custom Domains
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/organizations/{id}/domains` | List domains |
| POST | `/api/organizations/{id}/domains` | Add domain |
| POST | `/api/organizations/{id}/domains/{domain}/verify` | Verify domain |
| DELETE | `/api/organizations/{id}/domains/{domain}` | Remove domain |

#### Quotas & Usage
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/organizations/{id}/quotas` | Get quotas |
| PUT | `/api/organizations/{id}/quotas` | Update quotas |
| GET | `/api/organizations/{id}/usage` | Get usage stats |
| GET | `/api/organizations/{id}/usage/export` | Export CSV |

#### SLA
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/organizations/{id}/sla` | Get metrics |
| GET | `/api/organizations/{id}/sla/report` | Download report |

---

## Security Best Practices

### 1. Tenant Isolation

- **Always use middleware**: Never bypass tenant isolation middleware
- **Filter all queries**: Use `BuildOrgFilterQuery()` for all database queries
- **Verify access**: Call `VerifyOrgAccess()` before accessing resources

```go
// Bad - no organization filter
rows, err := db.Query("SELECT * FROM connections WHERE id = ?", connID)

// Good - with organization filter
if err := middleware.VerifyOrgAccess(ctx, orgID); err != nil {
    return err
}
whereClause, args := middleware.BuildOrgFilterQuery(ctx, "organization_id")
query := "SELECT * FROM connections WHERE id = ? AND " + whereClause
rows, err := db.Query(query, append([]interface{}{connID}, args...)...)
```

### 2. Rate Limiting

- Per-organization rate limiting is automatic
- Quotas are enforced at both middleware and service layers
- Monitor rate limit headers in responses

### 3. Custom Domains

- Require DNS verification before activation
- Validate all domain inputs
- Block common test/localhost domains
- Support SSL/TLS certificates

### 4. White-labeling Security

- Sanitize all custom CSS inputs
- Validate color hex codes
- Validate URLs (must be https://)
- Validate email addresses
- Prevent XSS via CSS injection

### 5. Audit Logging

All enterprise operations are logged:
- Organization changes
- Quota modifications
- Domain verifications
- White-label updates

### 6. Database Security

- All tables have foreign key constraints
- Cascade deletes ensure data cleanup
- Indexes on organization_id for performance
- Prepared statements prevent SQL injection

---

## Support

For enterprise support, contact: enterprise@sqlstudio.com

## License

Enterprise features are available under the Enterprise license.
