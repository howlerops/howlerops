# Example Output: Shared Credentials Migration Analysis

This document shows example output from the `migrate_shared_credentials.go` script.

## Scenario 1: Some Connections Need Migration

```
Analyzing shared credentials migration status...

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

Organization: org_abc123def456 (2 connections)
  • Production PostgreSQL
    ID:           conn_xyz789abc
    Type:         postgresql
    Created By:   user_456def
    Created At:   2024-12-15T10:30:00Z
    Personal Cred: true
    Shared Cred:   false

  • Staging MySQL
    ID:           conn_def456ghi
    Type:         mysql
    Created By:   user_456def
    Created At:   2024-12-16T14:20:00Z
    Personal Cred: true
    Shared Cred:   false

Organization: org_xyz789ghi012 (2 connections)
  • Analytics Snowflake
    ID:           conn_ghi789jkl
    Type:         snowflake
    Created By:   user_789ghi
    Created At:   2024-12-18T09:15:00Z
    Personal Cred: false
    Shared Cred:   false

  • Customer DuckDB
    ID:           conn_jkl012mno
    Type:         duckdb
    Created By:   user_789ghi
    Created At:   2024-12-19T16:45:00Z
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

WHAT HAPPENS DURING MIGRATION:
  1. User's browser decrypts the password using their master key
  2. User's browser encrypts the password using the organization's OEK
  3. New encrypted password is stored in the shared_credentials table
  4. Connection's encryption_type is updated to 'shared'
  5. All organization members can now access the connection

================================================================================
End of Report
================================================================================
```

## Scenario 2: All Connections Already Migrated

```
Analyzing shared credentials migration status...

================================================================================
         SHARED CREDENTIALS MIGRATION ANALYSIS REPORT
================================================================================

SUMMARY:
  Total Connections:           32
  - Personal Connections:      28
  - Shared Connections:        4

  Already Migrated:            4 (100.0%)
  Needs Migration:             0 (0.0%)

  Analysis Duration:           98ms

================================================================================
✓ ALL SHARED CONNECTIONS HAVE BEEN MIGRATED!
================================================================================

================================================================================
End of Report
================================================================================
```

## Scenario 3: Migration 009 Not Applied Yet

```
Analyzing shared credentials migration status...

================================================================================
         SHARED CREDENTIALS MIGRATION ANALYSIS REPORT
================================================================================

SUMMARY:
  Total Connections:           0
  - Personal Connections:      0
  - Shared Connections:        0

  Already Migrated:            0 (0.0%)
  Needs Migration:             0 (0.0%)

  Analysis Duration:           12ms

================================================================================
ERRORS ENCOUNTERED:
================================================================================
1. Migration 009 has not been applied - shared_credentials tables do not exist

================================================================================
End of Report
================================================================================

Migration 009 has not been applied yet
```

## Verbose Mode Example

Run with `--verbose` flag:

```bash
TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." \
go run scripts/migrate_shared_credentials.go --verbose
```

Output:
```
Connecting to database: libsql://my-db-abc123.turso.io
Database connection established
Checking if shared credentials tables exist...
Querying all connections...
  ✓ Production PostgreSQL: Already migrated (has shared_credentials)
  ✓ Staging PostgreSQL: Already migrated (has shared_credentials)
  ✗ Analytics Snowflake: Needs migration
  ✓ Customer MySQL: Already migrated (has shared_credentials)
  ✗ Legacy Oracle: Needs migration

[... rest of report ...]
```

## Dry Run Mode

Run with `--dry-run` flag:

```bash
TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." \
go run scripts/migrate_shared_credentials.go --dry-run
```

Output:
```
================================================================================
         SHARED CREDENTIALS MIGRATION ANALYSIS REPORT
================================================================================

MODE: DRY RUN (Read-only analysis)

[... rest of report identical to normal mode ...]
```

Note: Since this script is read-only by design, dry-run mode produces the same output as normal mode.

## Integration with Makefile

Using the Makefile shortcuts:

```bash
# Basic analysis
TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." make migrate-analyze-shared

# Verbose analysis
TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." make migrate-analyze-shared-verbose
```

## Exit Codes

The script uses standard exit codes:

- **0**: Success (even if migrations are pending)
- **1**: Error occurred (database connection failed, tables missing, etc.)

Example error handling in a script:

```bash
#!/bin/bash

TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." \
go run scripts/migrate_shared_credentials.go

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo "Analysis completed successfully"
else
  echo "Analysis failed with exit code $EXIT_CODE"
  exit 1
fi
```

## Programmatic Output Parsing

The report uses consistent formatting for parsing:

### Key Patterns

```regex
Total Connections:\s+(\d+)
Shared Connections:\s+(\d+)
Already Migrated:\s+(\d+)\s+\(([\d.]+)%\)
Needs Migration:\s+(\d+)\s+\(([\d.]+)%\)
```

### Example Parser (Python)

```python
import re
import subprocess

# Run the script
result = subprocess.run(
    ['go', 'run', 'scripts/migrate_shared_credentials.go'],
    capture_output=True,
    text=True,
    env={'TURSO_URL': '...', 'TURSO_AUTH_TOKEN': '...'}
)

# Parse output
output = result.stdout

total = int(re.search(r'Total Connections:\s+(\d+)', output).group(1))
shared = int(re.search(r'Shared Connections:\s+(\d+)', output).group(1))
migrated = int(re.search(r'Already Migrated:\s+(\d+)', output).group(1))
pending = int(re.search(r'Needs Migration:\s+(\d+)', output).group(1))

print(f"Total: {total}, Shared: {shared}, Migrated: {migrated}, Pending: {pending}")

# Check if action needed
if pending > 0:
    print(f"⚠ {pending} connections need migration")
    # Send notification to admins
    notify_admins(pending)
```

## Monitoring and Alerting

### Cron Job Example

Monitor migration progress daily:

```cron
# Run analysis daily at 9 AM
0 9 * * * cd /path/to/backend-go && TURSO_URL=$TURSO_URL TURSO_AUTH_TOKEN=$TURSO_AUTH_TOKEN go run scripts/migrate_shared_credentials.go >> /var/log/migration-analysis.log 2>&1
```

### Prometheus Metrics Example

Convert to metrics format:

```bash
#!/bin/bash

OUTPUT=$(TURSO_URL="..." TURSO_AUTH_TOKEN="..." \
  go run scripts/migrate_shared_credentials.go 2>&1)

# Extract metrics
TOTAL=$(echo "$OUTPUT" | grep "Total Connections:" | awk '{print $3}')
SHARED=$(echo "$OUTPUT" | grep "Shared Connections:" | awk '{print $4}')
MIGRATED=$(echo "$OUTPUT" | grep "Already Migrated:" | awk '{print $3}')
PENDING=$(echo "$OUTPUT" | grep "Needs Migration:" | awk '{print $3}')

# Write to metrics file
cat > /var/lib/prometheus/node_exporter/migration_metrics.prom <<EOF
# HELP shared_credentials_total Total number of shared connections
# TYPE shared_credentials_total gauge
shared_credentials_total $SHARED

# HELP shared_credentials_migrated Number of migrated connections
# TYPE shared_credentials_migrated gauge
shared_credentials_migrated $MIGRATED

# HELP shared_credentials_pending Number of connections pending migration
# TYPE shared_credentials_pending gauge
shared_credentials_pending $PENDING
EOF
```

## Dashboard Integration

Use the script output to build admin dashboards showing:

- Total connections by type (personal vs shared)
- Migration progress (percentage)
- Connections per organization needing migration
- Historical migration trends
- Estimated completion time
