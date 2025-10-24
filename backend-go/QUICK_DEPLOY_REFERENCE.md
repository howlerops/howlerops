# Quick Deployment Reference

Fast reference for common deployment operations.

## First Time Setup

```bash
# 1. Setup environment
cp .env.production.example .env.production
nano .env.production  # Add your values

# 2. Source environment
source .env.production

# 3. Deploy
make deploy-prod
```

## Required Environment Variables

```bash
export GCP_PROJECT_ID="your-project-id"
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-token"
export JWT_SECRET="$(openssl rand -base64 32)"
export RESEND_API_KEY="re_your-key"
```

## Common Commands

### Deployment

```bash
# Full deployment (recommended)
make deploy-prod

# Step-by-step
make prod-check               # 1. Check readiness
make setup-gcp-secrets        # 2. Setup secrets
gcloud run deploy...          # 3. Deploy
make verify-prod SERVICE_URL=... # 4. Verify
```

### Monitoring

```bash
# View logs
make prod-logs

# Check status
make prod-status

# Check costs
make check-costs

# List revisions
make prod-revisions
```

### Troubleshooting

```bash
# View recent errors
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" --limit=20

# Get service URL
gcloud run services describe sql-studio-backend --region=us-central1 --format='value(status.url)'

# Test health endpoint
curl $(gcloud run services describe sql-studio-backend --region=us-central1 --format='value(status.url)')/health
```

### Rollback

```bash
# List revisions
make prod-revisions

# Rollback to specific revision
make prod-rollback REVISION=sql-studio-backend-00001-abc
```

## Quick Health Check

```bash
SERVICE_URL=$(gcloud run services describe sql-studio-backend --region=us-central1 --format='value(status.url)')
curl $SERVICE_URL/health
```

## Update Secrets

```bash
# Update environment variables
nano .env.production
source .env.production

# Re-run setup
make setup-gcp-secrets

# Redeploy to pick up new secrets
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --update-secrets=JWT_SECRET=jwt-secret:latest
```

## Scale Configuration

```bash
# Scale to zero when idle (cost optimization)
gcloud run services update sql-studio-backend \
  --min-instances=0 \
  --max-instances=10 \
  --region=us-central1

# Always-on (faster response, higher cost)
gcloud run services update sql-studio-backend \
  --min-instances=1 \
  --max-instances=10 \
  --region=us-central1
```

## Memory Configuration

```bash
# Increase memory (if seeing OOM errors)
gcloud run services update sql-studio-backend \
  --memory=1Gi \
  --region=us-central1

# Reduce memory (cost optimization)
gcloud run services update sql-studio-backend \
  --memory=512Mi \
  --region=us-central1
```

## Cost Checks

```bash
# Quick cost check
make check-costs

# Detailed billing
gcloud billing accounts list

# View Cloud Run pricing
# https://cloud.google.com/run/pricing
```

## Emergency Procedures

### Service is Down

```bash
# 1. Check logs
make prod-logs

# 2. Check status
make prod-status

# 3. Rollback if needed
make prod-revisions
make prod-rollback REVISION=LAST_WORKING_REVISION
```

### High Error Rate

```bash
# 1. View errors
gcloud logging read "severity>=ERROR" --limit=50

# 2. Identify issue
# Check for common errors: database connection, secrets, rate limits

# 3. Fix and redeploy
# Or rollback temporarily
```

### Unexpected Costs

```bash
# 1. Check usage
make check-costs

# 2. Scale down immediately
gcloud run services update sql-studio-backend \
  --max-instances=1 \
  --region=us-central1

# 3. Investigate
gcloud monitoring time-series list \
  --filter='metric.type="run.googleapis.com/request_count"'
```

## URLs & Consoles

```bash
# Cloud Run Console
echo "https://console.cloud.google.com/run?project=$GCP_PROJECT_ID"

# Logs Console
echo "https://console.cloud.google.com/logs?project=$GCP_PROJECT_ID"

# Billing Console
echo "https://console.cloud.google.com/billing?project=$GCP_PROJECT_ID"

# Service URL
gcloud run services describe sql-studio-backend --region=us-central1 --format='value(status.url)'
```

## Help

```bash
# Production deployment help
make prod-help

# All available commands
make help
```

---

**Pro Tip**: Save these commands in your shell history for quick access!

```bash
# Add to ~/.bashrc or ~/.zshrc
alias sql-deploy='cd ~/projects/sql-studio/backend-go && source .env.production && make deploy-prod'
alias sql-logs='gcloud run services logs tail sql-studio-backend'
alias sql-status='make prod-status'
```
