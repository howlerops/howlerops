# SQL Studio Backend - Quick Start Guide

**For experienced developers: Deploy in 5-10 minutes**

Assumes you know: GCP, Docker, Git, command line.

---

## Prerequisites Checklist

- [ ] GCP account with billing enabled
- [ ] Turso account (https://turso.tech)
- [ ] Resend account (https://resend.com) - optional
- [ ] gcloud CLI installed and authenticated
- [ ] Git installed
- [ ] OpenSSL installed

---

## 1. GCP Setup (2 minutes)

```bash
# Set project ID (replace with your project ID)
export PROJECT_ID="sql-studio-prod"

# Create project (if needed)
gcloud projects create $PROJECT_ID
gcloud config set project $PROJECT_ID

# Enable billing (via console: https://console.cloud.google.com/billing)
# Required but can't be done via CLI

# Enable APIs
gcloud services enable \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com

# Create service account for GitHub Actions
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions"

# Grant roles
for ROLE in run.admin iam.serviceAccountUser secretmanager.admin storage.admin; do
  gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/$ROLE"
done

# Create service account key
gcloud iam service-accounts keys create ~/gcp-sa-key.json \
  --iam-account=github-actions@$PROJECT_ID.iam.gserviceaccount.com

# Copy key to clipboard (macOS)
cat ~/gcp-sa-key.json | pbcopy
# Linux: cat ~/gcp-sa-key.json | xclip -selection clipboard
# Windows: type gcp-sa-key.json | clip
```

---

## 2. Turso Setup (1 minute)

```bash
# Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# Login
turso auth login

# Create database
turso db create sql-studio-prod --location iad

# Get credentials
export TURSO_URL=$(turso db show sql-studio-prod --url)
export TURSO_AUTH_TOKEN=$(turso db tokens create sql-studio-prod)

# Save to clipboard for GitHub Secrets
echo "TURSO_URL=$TURSO_URL"
echo "TURSO_AUTH_TOKEN=$TURSO_AUTH_TOKEN"
```

---

## 3. Generate Secrets (30 seconds)

```bash
# Generate JWT secret
export JWT_SECRET=$(openssl rand -base64 64)
echo "JWT_SECRET=$JWT_SECRET"

# Optional: Get Resend API key from https://resend.com/api-keys
export RESEND_API_KEY="re_your_key_here"
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
```

---

## 4. GitHub Setup (2 minutes)

**Fork/Clone Repository:**

```bash
git clone https://github.com/sql-studio/sql-studio.git
cd sql-studio
```

**Add GitHub Secrets:**

Go to: Settings → Secrets and variables → Actions → New repository secret

Add each:

```bash
GCP_PROJECT_ID            # Your GCP project ID
GCP_SA_KEY               # Full JSON from step 1 (cat ~/gcp-sa-key.json)
TURSO_URL                # From step 2
TURSO_AUTH_TOKEN         # From step 2
JWT_SECRET               # From step 3
RESEND_API_KEY           # Optional, from step 3
RESEND_FROM_EMAIL        # Optional, from step 3
```

**Or use GitHub CLI (faster):**

```bash
# Install GitHub CLI: https://cli.github.com/
gh auth login

# Set secrets
gh secret set GCP_PROJECT_ID --body "$PROJECT_ID"
gh secret set GCP_SA_KEY < ~/gcp-sa-key.json
gh secret set TURSO_URL --body "$TURSO_URL"
gh secret set TURSO_AUTH_TOKEN --body "$TURSO_AUTH_TOKEN"
gh secret set JWT_SECRET --body "$JWT_SECRET"
gh secret set RESEND_API_KEY --body "$RESEND_API_KEY"
gh secret set RESEND_FROM_EMAIL --body "$RESEND_FROM_EMAIL"
```

---

## 5. Deploy (1 minute)

**Option A: Git Tag (Recommended)**

```bash
git tag v1.0.0
git push origin v1.0.0
```

**Option B: Manual Trigger**

```bash
# Via GitHub CLI
gh workflow run deploy-backend.yml

# Or via web: Actions → Deploy Backend → Run workflow
```

**Monitor:**

```bash
# Watch workflow
gh run watch

# Or view in browser: https://github.com/YOUR-REPO/actions
```

---

## 6. Verify (30 seconds)

```bash
# Get service URL
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format 'value(status.url)')

# Test health endpoint
curl $SERVICE_URL/health

# Expected: {"status":"healthy","version":"1.0.0","uptime":"...","database":"connected"}
```

---

## Configuration Reference

### Environment Variables

All set via Cloud Run deployment (automated by GitHub Actions):

```bash
ENVIRONMENT=production
LOG_LEVEL=info
LOG_FORMAT=json
SERVER_HTTP_PORT=8500
SERVER_GRPC_PORT=9500
METRICS_PORT=9100
```

### Resource Limits

Default (adjust in `.github/workflows/deploy-backend.yml` if needed):

```yaml
--memory=512Mi
--cpu=2
--min-instances=0
--max-instances=10
--concurrency=80
--timeout=300
```

### Cost Optimization

```bash
# Verify scale-to-zero is enabled
gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format 'value(spec.template.spec.containers[0].resources.limits)'

# Should show: min-instances: 0
```

---

## Common Commands

### View Logs

```bash
# Real-time logs
gcloud run services logs tail sql-studio-backend --region us-central1

# Search logs
gcloud logging read "resource.type=cloud_run_revision" \
  --limit 50 \
  --format json | jq '.[] | select(.severity=="ERROR")'
```

### Update Secrets

```bash
# Update secret in GCP Secret Manager
echo -n "new-value" | gcloud secrets versions add secret-name --data-file=-

# Service automatically picks up new version on next cold start
# Or force redeploy:
gcloud run services update sql-studio-backend --region us-central1
```

### Manual Deployment

```bash
# Build and deploy manually (no GitHub Actions)
cd backend-go

# Build image
gcloud builds submit --tag gcr.io/$PROJECT_ID/sql-studio-backend

# Deploy
gcloud run deploy sql-studio-backend \
  --image gcr.io/$PROJECT_ID/sql-studio-backend \
  --region us-central1 \
  --platform managed \
  --allow-unauthenticated \
  --port 8500 \
  --memory 512Mi \
  --cpu 2 \
  --min-instances 0 \
  --max-instances 10 \
  --set-env-vars="ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json" \
  --set-secrets="TURSO_URL=turso-url:latest,TURSO_AUTH_TOKEN=turso-auth-token:latest,JWT_SECRET=jwt-secret:latest"
```

### Scale Resources

```bash
# Increase memory/CPU
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --memory 1Gi \
  --cpu 4

# Increase max instances
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --max-instances 50

# Set minimum instances (no cold starts, but costs more)
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --min-instances 1
```

### Rollback

```bash
# List revisions
gcloud run revisions list \
  --service sql-studio-backend \
  --region us-central1

# Route to specific revision
gcloud run services update-traffic sql-studio-backend \
  --region us-central1 \
  --to-revisions REVISION-NAME=100
```

### Custom Domain

```bash
# Map domain
gcloud run domain-mappings create \
  --service sql-studio-backend \
  --domain api.yourdomain.com \
  --region us-central1

# Get DNS records to add
gcloud run domain-mappings describe \
  --domain api.yourdomain.com \
  --region us-central1
```

---

## Alternative: Fly.io Deployment

**Why?** Cheaper for low-traffic apps, global edge network, simpler setup.

**Setup (2 minutes):**

```bash
# Install Fly CLI
curl -L https://fly.io/install.sh | sh

# Login
flyctl auth login

# Create app (from backend-go directory)
cd backend-go
flyctl apps create sql-studio-backend

# Set secrets
flyctl secrets set \
  TURSO_URL="$TURSO_URL" \
  TURSO_AUTH_TOKEN="$TURSO_AUTH_TOKEN" \
  JWT_SECRET="$JWT_SECRET" \
  ENVIRONMENT="production" \
  LOG_LEVEL="info" \
  LOG_FORMAT="json"

# Deploy
flyctl deploy --ha=false

# Get URL
flyctl info
```

**fly.toml already configured** with production settings:
- Scale-to-zero
- Health checks
- 512MB RAM, 1 shared CPU
- Auto-start on requests

---

## Local Development

**Test locally before deploying:**

```bash
cd backend-go

# Copy environment template
cp .env.example .env.development

# Edit .env.development (use local SQLite)
cat > .env.development << EOF
ENVIRONMENT=development
TURSO_URL=file:./data/development.db
TURSO_AUTH_TOKEN=
JWT_SECRET=$(openssl rand -base64 64)
SERVER_HTTP_PORT=8080
LOG_LEVEL=debug
LOG_FORMAT=text
EOF

# Run locally
go run cmd/server/main.go

# Or use Docker
docker build -t sql-studio-backend .
docker run -p 8080:8080 --env-file .env.development sql-studio-backend
```

**Test endpoints:**

```bash
# Health
curl http://localhost:8080/health

# Version
curl http://localhost:8080/api/v1/version

# Metrics
curl http://localhost:9100/metrics
```

---

## CI/CD Workflow

**Automated deployment flow:**

1. Push tag: `git tag v1.0.0 && git push origin v1.0.0`
2. GitHub Actions triggers
3. Validates code and secrets
4. Builds Docker image (multi-arch: amd64/arm64)
5. Pushes to GCP Container Registry
6. Creates/updates secrets in GCP Secret Manager
7. Deploys to Cloud Run (zero-downtime rolling update)
8. Runs smoke tests
9. Routes traffic if healthy
10. Auto-rollback on failure

**Workflow file:** `.github/workflows/deploy-backend.yml`

**Disable auto-deploy:**

Edit workflow, comment out:

```yaml
# on:
#   release:
#     types: [published]
```

---

## Monitoring

### Uptime Monitoring

**Option 1: GCP Uptime Checks**

```bash
# Create via console: https://console.cloud.google.com/monitoring/uptime
# Or use gcloud alpha commands
```

**Option 2: External (Better)**

- UptimeRobot: https://uptimerobot.com (free tier: 50 monitors)
- Pingdom: https://www.pingdom.com
- StatusCake: https://www.statuscake.com

### Application Metrics

**Prometheus metrics available at:**

```bash
curl $SERVICE_URL:9100/metrics
```

**Integrate with:**
- Grafana Cloud (free tier)
- Datadog
- New Relic
- GCP Cloud Monitoring (built-in)

### Log Aggregation

**GCP Cloud Logging (built-in):**

```bash
# View in console
open "https://console.cloud.google.com/logs/query?project=$PROJECT_ID"
```

**Export to:**
- BigQuery (for analysis)
- Cloud Storage (for archival)
- Pub/Sub (for streaming)

---

## Security Hardening

### Recommendations

**Enable Cloud Armor (DDoS protection):**

```bash
# Create policy
gcloud compute security-policies create sql-studio-policy \
  --description "DDoS and WAF protection"

# Add rate limiting rule
gcloud compute security-policies rules create 1000 \
  --security-policy sql-studio-policy \
  --expression "origin.region_code == 'CN'" \
  --action "deny-403"

# Attach to Cloud Run (requires Load Balancer)
```

**Rotate secrets regularly:**

```bash
# JWT secret (every 90 days)
NEW_JWT=$(openssl rand -base64 64)
echo -n "$NEW_JWT" | gcloud secrets versions add jwt-secret --data-file=-

# Turso token (every 90 days)
NEW_TOKEN=$(turso db tokens create sql-studio-prod --expiration 90d)
echo -n "$NEW_TOKEN" | gcloud secrets versions add turso-auth-token --data-file=-

# Redeploy to pick up new secrets
gcloud run services update sql-studio-backend --region us-central1
```

**Enable audit logging:**

```bash
# Enable Data Access audit logs
gcloud projects set-iam-policy $PROJECT_ID policy.yaml

# policy.yaml:
# auditConfigs:
# - auditLogConfigs:
#   - logType: DATA_READ
#   - logType: DATA_WRITE
#   service: run.googleapis.com
```

---

## Performance Tuning

### Database Optimization

```bash
# Create indexes in Turso
turso db shell sql-studio-prod << EOF
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_queries_user_id ON queries(user_id);
CREATE INDEX idx_queries_created_at ON queries(created_at);
EOF
```

### Caching (Optional)

**Add Redis for session/query caching:**

```bash
# Deploy Redis on Cloud Run
gcloud run deploy redis \
  --image redis:7-alpine \
  --region us-central1 \
  --allow-unauthenticated \
  --port 6379

# Or use Cloud Memorystore (managed Redis)
gcloud redis instances create sql-studio-cache \
  --size=1 \
  --region=us-central1 \
  --redis-version=redis_7_0
```

### CDN (Optional)

**Enable Cloud CDN for static assets:**

```bash
# Requires Load Balancer setup
# Follow: https://cloud.google.com/cdn/docs/setting-up-cdn-with-serverless
```

---

## Cost Monitoring

### Set Budget Alerts

```bash
# Create budget (via console)
open "https://console.cloud.google.com/billing/budgets?project=$PROJECT_ID"

# Set alerts at:
# - $5 (50% of $10 budget)
# - $10 (100% of $10 budget)
# - $15 (150% of $10 budget)
```

### View Current Costs

```bash
# View billing
open "https://console.cloud.google.com/billing?project=$PROJECT_ID"

# Export to BigQuery for analysis
gcloud beta billing accounts export create \
  --billing-account=YOUR-BILLING-ACCOUNT-ID \
  --dataset-id=billing_export \
  --project=$PROJECT_ID
```

---

## Troubleshooting

### Quick Diagnostics

```bash
# Service status
gcloud run services describe sql-studio-backend --region us-central1

# Recent logs
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  --limit 20

# Test health endpoint
curl -v $(gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format 'value(status.url)')/health

# Check secrets exist
gcloud secrets list

# Verify Turso connection
turso db shell sql-studio-prod "SELECT 1"
```

### Common Issues

| Issue | Quick Fix |
|-------|-----------|
| 502/503 errors | Check logs for database connection errors |
| High latency | Increase CPU/memory resources |
| Out of memory | Increase `--memory` limit |
| Secrets not found | Verify secret names and IAM permissions |
| Cold starts slow | Set `--min-instances=1` (costs more) |
| High costs | Verify `--min-instances=0` and `--cpu-throttling` |

---

## Production Checklist

Before going live:

- [ ] All GitHub Secrets configured correctly
- [ ] JWT_SECRET is 64+ characters (generated with openssl)
- [ ] Turso database has required schema/indexes
- [ ] Health endpoint returns 200 OK
- [ ] Custom domain configured (optional)
- [ ] Uptime monitoring set up
- [ ] Billing alerts configured ($5, $10, $20)
- [ ] Backup strategy documented
- [ ] Rollback procedure tested
- [ ] Load tested (optional: `backend-go/scripts/load-test.sh`)
- [ ] Security scan passed (optional: `docker scan`)

---

## Resources

- **Full deployment guide:** `DEPLOYMENT_COMPLETE_GUIDE.md`
- **FAQ:** `DEPLOYMENT_FAQ.md`
- **GCP Cloud Run docs:** https://cloud.google.com/run/docs
- **Turso docs:** https://docs.turso.tech
- **GitHub Actions docs:** https://docs.github.com/en/actions

---

**Document Version:** 1.0.0
**Last Updated:** 2025-10-23
**Target Audience:** Experienced developers
**Estimated Time:** 5-10 minutes
