# Rollback Procedures for SQL Studio Backend

## Table of Contents

1. [Quick Rollback](#quick-rollback)
2. [Rollback Scenarios](#rollback-scenarios)
3. [Step-by-Step Rollback](#step-by-step-rollback)
4. [Database Rollback](#database-rollback)
5. [Traffic Management](#traffic-management)
6. [Emergency Procedures](#emergency-procedures)
7. [Post-Rollback Actions](#post-rollback-actions)

## Quick Rollback

### Immediate Rollback Command

```bash
# Fastest way to rollback to previous version
gcloud run services update-traffic sql-studio-backend \
  --to-revisions=LATEST=0 \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID
```

### Automated Rollback Script

```bash
#!/bin/bash
# Save as scripts/rollback.sh

SERVICE_NAME="sql-studio-backend"
REGION="us-central1"
PROJECT_ID="${GCP_PROJECT_ID}"

# Get the second-to-last revision (previous stable version)
PREVIOUS_REVISION=$(gcloud run revisions list \
  --service=$SERVICE_NAME \
  --region=$REGION \
  --project=$PROJECT_ID \
  --format="value(metadata.name)" \
  --limit=2 | tail -n 1)

if [ -z "$PREVIOUS_REVISION" ]; then
  echo "ERROR: No previous revision found"
  exit 1
fi

echo "Rolling back to: $PREVIOUS_REVISION"

# Route all traffic to previous revision
gcloud run services update-traffic $SERVICE_NAME \
  --region=$REGION \
  --to-revisions="$PREVIOUS_REVISION=100" \
  --project=$PROJECT_ID

echo "Rollback completed"
```

## Rollback Scenarios

### Scenario 1: High Error Rate

**Symptoms:**
- Error rate > 5% for more than 5 minutes
- 5xx status codes increasing
- Application logs show critical errors

**Actions:**
1. Immediate rollback to previous version
2. Investigate error logs
3. Fix issues in development
4. Re-deploy after testing

### Scenario 2: Performance Degradation

**Symptoms:**
- P95 latency > 3 seconds
- CPU/Memory usage abnormally high
- Cold start frequency increased

**Actions:**
1. Check if gradual rollback is possible
2. Monitor metrics during rollback
3. Investigate performance bottlenecks
4. Optimize before re-deployment

### Scenario 3: Database Schema Issues

**Symptoms:**
- Database queries failing
- Schema mismatch errors
- Data integrity issues

**Actions:**
1. Stop traffic to new version immediately
2. Assess database state
3. Apply rollback migrations if needed
4. Restore from backup if necessary

### Scenario 4: Authentication Broken

**Symptoms:**
- Users cannot login
- JWT validation failures
- Session management issues

**Actions:**
1. Immediate rollback
2. Clear any corrupted sessions
3. Verify JWT secret consistency
4. Test authentication thoroughly before re-deploy

## Step-by-Step Rollback

### 1. Assess the Situation

```bash
# Check current service status
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format="value(status.conditions[0])"

# View recent errors
gcloud logging read "severity>=ERROR AND resource.type=cloud_run_revision" \
  --limit=50 \
  --format=json

# Check metrics
gcloud monitoring time-series list \
  --filter='metric.type="run.googleapis.com/request_count"'
```

### 2. Identify Target Revision

```bash
# List all revisions with traffic percentages
gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1 \
  --format="table(metadata.name,status.traffic.percent)"

# Get details of specific revision
gcloud run revisions describe REVISION_NAME \
  --region=us-central1
```

### 3. Perform Gradual Rollback (if time permits)

```bash
# Route 50% traffic to previous version
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-revisions="NEW_REVISION=50,OLD_REVISION=50"

# Monitor for 5 minutes
sleep 300

# If stable, complete rollback
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-revisions="OLD_REVISION=100"
```

### 4. Immediate Full Rollback (emergency)

```bash
# Route 100% traffic to previous version immediately
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-revisions="OLD_REVISION=100"
```

### 5. Verify Rollback

```bash
# Check traffic distribution
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format="value(status.traffic[].percent)"

# Test health endpoint
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format="value(status.url)")

curl -f "$SERVICE_URL/health"

# Run verification script
./scripts/verify-deployment.sh $SERVICE_URL
```

## Database Rollback

### Check Database State

```bash
# Connect to Turso database
turso db shell sql-studio-production

# Check schema version
SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;

# Verify data integrity
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM workspaces;
```

### Rollback Migrations

```bash
# If using migration tool
cd backend-go
go run cmd/migrate/main.go down --steps=1

# Or manually apply rollback SQL
turso db shell sql-studio-production < migrations/rollback_v1.0.1.sql
```

### Restore from Backup

```bash
# List available backups
turso db backup list sql-studio-production

# Create new backup before restore (safety)
turso db backup create sql-studio-production --name pre-restore-backup

# Restore from specific backup
turso db backup restore sql-studio-production --backup-id=BACKUP_ID

# Verify restore
turso db shell sql-studio-production "SELECT COUNT(*) FROM users;"
```

## Traffic Management

### Gradual Traffic Shifting

```bash
# Start with 10% on new version
gcloud run services update-traffic sql-studio-backend \
  --to-revisions="NEW=10,OLD=90"

# Increase to 25%
gcloud run services update-traffic sql-studio-backend \
  --to-revisions="NEW=25,OLD=75"

# Increase to 50%
gcloud run services update-traffic sql-studio-backend \
  --to-revisions="NEW=50,OLD=50"

# Full deployment
gcloud run services update-traffic sql-studio-backend \
  --to-revisions="NEW=100"
```

### Blue-Green Deployment Rollback

```bash
# Tag revisions for clarity
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --set-tags="blue=OLD_REVISION,green=NEW_REVISION"

# Switch from green to blue
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-tags="blue=100"
```

### Canary Rollback

```bash
# If canary deployment fails
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --remove-tags="canary" \
  --to-revisions="STABLE=100"
```

## Emergency Procedures

### Complete Service Shutdown

```bash
# Stop all traffic (emergency only)
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --min-instances=0 \
  --max-instances=0

# Display maintenance page (if configured)
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --set-env-vars="MAINTENANCE_MODE=true"
```

### Secret Rotation During Rollback

```bash
# If secrets are compromised
NEW_JWT_SECRET=$(openssl rand -base64 64)

# Update secret
echo -n "$NEW_JWT_SECRET" | gcloud secrets versions add jwt-secret \
  --data-file=-

# Force service to use new secret
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --update-secrets="JWT_SECRET=jwt-secret:latest"
```

### Clear Cache and State

```bash
# If using Redis
redis-cli FLUSHALL

# Clear CDN cache (if using Cloud CDN)
gcloud compute url-maps invalidate-cdn-cache URL_MAP_NAME \
  --path="/*"

# Reset rate limiting counters
# (Implementation specific)
```

## Post-Rollback Actions

### 1. Incident Documentation

Create an incident report with:
- Timestamp of issue detection
- Symptoms observed
- Rollback decision rationale
- Actions taken
- Time to resolution
- Impact assessment

### 2. Root Cause Analysis

```bash
# Collect logs from failed deployment
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.revision_name=FAILED_REVISION" \
  --format=json > incident_logs.json

# Export metrics
gcloud monitoring time-series list \
  --filter='metric.type="run.googleapis.com/request_count"' \
  --interval-start-time="2024-01-01T00:00:00Z" \
  --format=json > incident_metrics.json
```

### 3. Communication

```markdown
# Status Update Template

## Service Incident Report

**Service:** SQL Studio Backend
**Date:** [DATE]
**Duration:** [START] - [END]
**Impact:** [DESCRIPTION]

### Summary
Brief description of the issue and resolution.

### Timeline
- HH:MM - Issue detected
- HH:MM - Rollback initiated
- HH:MM - Service restored

### Root Cause
[Identified root cause]

### Action Items
- [ ] Fix identified issue
- [ ] Add monitoring for this scenario
- [ ] Update runbook
```

### 4. Monitoring Improvements

```bash
# Add alert for the issue that caused rollback
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="Alert Name" \
  --condition='[CONDITION]'
```

### 5. Testing Improvements

Add tests for the scenario that caused the rollback:
- Unit tests
- Integration tests
- Load tests
- Chaos engineering tests

## Rollback Verification Checklist

After completing rollback:

- [ ] Traffic fully routed to stable version
- [ ] Error rates returned to normal
- [ ] Latency within acceptable range
- [ ] All health checks passing
- [ ] User authentication working
- [ ] Database queries successful
- [ ] No data loss or corruption
- [ ] Monitoring alerts cleared
- [ ] Team notified of resolution
- [ ] Incident report drafted

## Common Issues and Solutions

### Issue: Revision won't accept traffic

```bash
# Check revision status
gcloud run revisions describe REVISION_NAME \
  --region=us-central1 \
  --format="value(status.conditions[].message)"

# Force traffic update
gcloud run services replace service.yaml --region=us-central1
```

### Issue: Database connection errors after rollback

```bash
# Verify secrets are accessible
gcloud secrets versions list turso-auth-token

# Update service with latest secret version
gcloud run services update sql-studio-backend \
  --update-secrets="TURSO_AUTH_TOKEN=turso-auth-token:latest"
```

### Issue: Users experiencing cache issues

```bash
# Add cache-busting headers
gcloud run services update sql-studio-backend \
  --update-env-vars="CACHE_VERSION=$(date +%s)"
```

## Preventive Measures

1. **Always test in staging first**
2. **Use feature flags for risky changes**
3. **Implement comprehensive health checks**
4. **Set up automated rollback triggers**
5. **Maintain up-to-date runbooks**
6. **Practice rollback procedures regularly**
7. **Keep previous versions for at least 7 days**
8. **Document all configuration changes**

## Support Contacts

| Severity | Contact | Method |
|----------|---------|--------|
| Critical | On-call Engineer | PagerDuty |
| High | Team Lead | Slack + Phone |
| Medium | Dev Team | Slack |
| Low | Dev Team | Email |

## Additional Resources

- [Cloud Run Rollback Documentation](https://cloud.google.com/run/docs/rollbacks)
- [Turso Backup Documentation](https://docs.turso.tech/features/backups)
- [Incident Response Playbook](./INCIDENT_RESPONSE.md)
- [Monitoring Dashboard](https://console.cloud.google.com/monitoring)