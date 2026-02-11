# Database Migration Scripts

This directory contains standalone migration and maintenance scripts for the HowlerOps backend.

## Shared Credentials Migration Script

### Purpose

`migrate_shared_credentials.go` is a diagnostic and reporting tool that analyzes database connections to identify which shared connections need migration to the Organization Envelope Key (OEK) system introduced in migration 009.

**Important**: This script is **read-only** and **cannot** automatically migrate credentials. This is by design due to the zero-knowledge architecture.

### Why Automatic Migration Is Not Possible

The application uses a zero-knowledge encryption architecture where:
- User master keys are **never** stored on the server
- Database passwords are encrypted client-side before being sent to the server
- The server only stores encrypted ciphertext

To migrate from personal encryption to organization encryption:
1. Password must be **decrypted** with the user's master key (client-side only)
2. Password must be **re-encrypted** with the organization's OEK (client-side only)
3. New encrypted password is stored in the `shared_credentials` table

This requires active user participation and cannot be done automatically.

### What This Script Does

The script performs a comprehensive analysis:

1. **Connects** to the Turso database
2. **Identifies** all shared connections (`visibility='shared'` AND `organization_id IS NOT NULL`)
3. **Checks** which connections have OEK-encrypted credentials in `shared_credentials` table
4. **Reports** which connections still need user action to complete migration
5. **Generates** detailed statistics and migration instructions

### Prerequisites

- Go 1.21 or later
- Access to Turso database
- `TURSO_URL` and `TURSO_AUTH_TOKEN` environment variables

### Installation

No installation required - this is a standalone script.

### Usage

#### Basic Analysis

```bash
cd backend-go
TURSO_URL="libsql://your-db.turso.io" \
TURSO_AUTH_TOKEN="your-auth-token" \
go run scripts/migrate_shared_credentials.go
```

#### Dry Run Mode

Preview what would be analyzed (same as normal mode since this is read-only):

```bash
TURSO_URL="libsql://your-db.turso.io" \
TURSO_AUTH_TOKEN="your-auth-token" \
go run scripts/migrate_shared_credentials.go --dry-run
```

#### Verbose Mode

Show detailed progress and debug information:

```bash
TURSO_URL="libsql://your-db.turso.io" \
TURSO_AUTH_TOKEN="your-auth-token" \
go run scripts/migrate_shared_credentials.go --verbose
```

#### Combined Flags

```bash
TURSO_URL="libsql://your-db.turso.io" \
TURSO_AUTH_TOKEN="your-auth-token" \
go run scripts/migrate_shared_credentials.go --dry-run --verbose
```

### Output

The script generates a detailed report including:

#### Summary Statistics
- Total connections (personal + shared)
- Personal connections count
- Shared connections count
- Already migrated count and percentage
- Needs migration count and percentage
- Analysis duration

#### Connections Requiring Migration
Grouped by organization, showing:
- Connection name, ID, and type
- Created by (user ID)
- Creation timestamp
- Personal credential status
- Shared credential status

#### Migration Instructions
Step-by-step guide for users to complete the migration via the UI.

### Example Output

```
================================================================================
         SHARED CREDENTIALS MIGRATION ANALYSIS REPORT
================================================================================

SUMMARY:
  Total Connections:           47
  - Personal Connections:      35
  - Shared Connections:        12

  Already Migrated:            8 (66.7%)
  Needs Migration:             4 (33.3%)

  Analysis Duration:           142ms

CONNECTIONS REQUIRING MIGRATION:
--------------------------------------------------------------------------------

Organization: org_abc123 (2 connections)
  • Production PostgreSQL
    ID:           conn_xyz789
    Type:         postgresql
    Created By:   user_456
    Created At:   2024-12-15T10:30:00Z
    Personal Cred: true
    Shared Cred:   false

  • Staging MySQL
    ID:           conn_def456
    Type:         mysql
    Created By:   user_456
    Created At:   2024-12-16T14:20:00Z
    Personal Cred: true
    Shared Cred:   false

================================================================================
MIGRATION INSTRUCTIONS:
================================================================================

These shared connections are using the OLD encryption model (personal master
keys) and need to be migrated to the NEW model (organization envelope keys).

WHY AUTOMATIC MIGRATION IS NOT POSSIBLE:
  - User master keys are NOT stored on the server (zero-knowledge architecture)
  - Passwords must be decrypted client-side with the user's master key
  - Then re-encrypted client-side with the organization envelope key (OEK)

WHAT USERS NEED TO DO:
  1. Log in to the application
  2. Navigate to Organization Connections
  3. For each connection listed above:
     a. Click on the connection
     b. Click 'Re-share with Organization' or 'Migrate to OEK'
     c. The app will handle client-side re-encryption automatically
```

### Exit Codes

- `0`: Success (with or without migrations needed)
- `1`: Error occurred during analysis

### Troubleshooting

#### Error: "TURSO_URL environment variable is required"
Set the `TURSO_URL` environment variable to your Turso database URL.

#### Error: "TURSO_AUTH_TOKEN environment variable is required"
Set the `TURSO_AUTH_TOKEN` environment variable to your Turso auth token.

#### Error: "Migration 009 has not been applied"
Run the database migrations first:
```bash
go run cmd/migrate/main.go
```

#### Error: "Failed to ping database"
Check that:
- Your Turso database is running
- The `TURSO_URL` is correct
- The `TURSO_AUTH_TOKEN` is valid
- Network connectivity is available

### Security Considerations

This script:
- ✅ Only performs **read** operations on the database
- ✅ Does **not** access or expose plaintext passwords
- ✅ Does **not** access or expose encryption keys
- ✅ Does **not** modify any data
- ✅ Can be safely run in production environments

### Integration with Application

After running this script, users should be notified through the application UI to complete the migration. Consider:

1. Adding a banner/notification for users with connections needing migration
2. Providing a "Migrate All" button in the organization settings
3. Showing migration status on each connection card
4. Sending email notifications to organization admins
5. Tracking migration progress over time

### Related Documentation

- [Migration 009: Shared Credentials](../pkg/storage/turso/migrations/009_shared_credentials.sql)
- [Organization Envelope Key Architecture](../docs/oek-architecture.md)
- [Zero-Knowledge Encryption](../docs/encryption.md)

### Development

To modify this script:

```bash
# Run tests
go test ./scripts/...

# Check for issues
go vet ./scripts/...

# Format code
go fmt ./scripts/...
```

### Future Enhancements

Potential improvements for future versions:

- [ ] Export report to CSV or JSON
- [ ] Email report to administrators
- [ ] Integration with monitoring systems (Prometheus, etc.)
- [ ] Automatic retry logic for failed checks
- [ ] Parallel analysis for large databases
- [ ] Historical migration tracking
- [ ] Estimated time to complete migration
- [ ] UI generation for migration dashboard
