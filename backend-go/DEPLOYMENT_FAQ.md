# Howlerops Backend - Deployment FAQ

**Frequently Asked Questions about deploying and operating Howlerops backend**

Last Updated: 2025-10-23

---

## Table of Contents

- [General Questions](#general-questions)
- [Deployment Issues](#deployment-issues)
- [Configuration & Secrets](#configuration--secrets)
- [Database & Storage](#database--storage)
- [Performance & Scaling](#performance--scaling)
- [Costs & Billing](#costs--billing)
- [Security & Authentication](#security--authentication)
- [Monitoring & Debugging](#monitoring--debugging)
- [Platform-Specific Questions](#platform-specific-questions)

---

## General Questions

### Q: Which deployment platform should I choose?

**A:** Depends on your needs:

**Choose GCP Cloud Run if:**
- You need enterprise-grade SLAs and compliance
- You want mature monitoring/logging (Cloud Operations)
- You need VPC networking or private services
- You have other services on GCP
- Budget: $0-5/month for hobby, $10-100/month for production

**Choose Fly.io if:**
- You want the cheapest option (true scale-to-zero)
- You need global edge deployment (multi-region)
- You prefer simpler setup and CLI
- You're building a hobby project or MVP
- Budget: $0-2/month for hobby, $5-30/month for production

**Both are excellent choices.** We recommend starting with GCP Cloud Run for production (better docs, support) and Fly.io for side projects (cheaper).

### Q: How long does first deployment take?

**A:**
- **With the guide:** 30-60 minutes (if you've never used GCP/Turso)
- **Experienced users:** 5-10 minutes (using Quick Start guide)
- **Actual deployment time:** 5-8 minutes (GitHub Actions workflow)

### Q: Do I need a credit card?

**A:**
- **GCP:** Yes (required for billing, but won't charge unless you upgrade)
- **Turso:** No (free tier doesn't require payment)
- **Resend:** No (free tier doesn't require payment)
- **GitHub:** No (Actions free for public repos, 2000 min/month for private)

### Q: What are the actual costs?

**A:** For a typical hobby/small project:
- **GCP Cloud Run:** $0/month (free tier covers ~2M requests)
- **Turso:** $0/month (free tier: 1B row reads, 9GB storage)
- **Resend:** $0/month (free tier: 3,000 emails/month)
- **Total:** $0-5/month

See [Cost Breakdown](#costs--billing) section for detailed breakdown.

### Q: Can I deploy without GitHub Actions?

**A:** Yes, multiple options:

**Option 1: Manual gcloud deployment**
```bash
cd backend-go
gcloud builds submit --tag gcr.io/$PROJECT_ID/sql-studio-backend
gcloud run deploy sql-studio-backend \
  --image gcr.io/$PROJECT_ID/sql-studio-backend \
  --region us-central1
```

**Option 2: Deployment script**
```bash
cd backend-go
./scripts/deploy-cloudrun.sh
```

**Option 3: Cloud Build trigger**
- Set up in GCP Console
- Triggers on git push

**But we recommend GitHub Actions** for:
- Automated testing before deployment
- Zero-downtime rolling updates
- Automatic rollback on failures
- Deployment history in GitHub

### Q: Is this production-ready?

**A:** Yes! The deployment includes:

- Multi-stage Docker build (security hardened)
- Non-root user in container
- Secret management (GCP Secret Manager / Fly.io Secrets)
- Health checks and auto-recovery
- Zero-downtime rolling updates
- Auto-scaling (0 to 100+ instances)
- Comprehensive logging (JSON structured logs)
- Metrics collection (Prometheus)
- Automatic rollback on failures

Hundreds of production apps run on this stack.

---

## Deployment Issues

### Q: Why is my deployment failing?

**A:** Check which step is failing:

**"Pre-deployment Validation" fails:**
- Missing GitHub Secrets (go to Settings → Secrets)
- Typo in secret names (case-sensitive!)
- Invalid GCP_SA_KEY (must be full JSON)

**"Build Docker Image" fails:**
- Go compilation error (test locally: `go build cmd/server/main.go`)
- Missing dependency (run `go mod tidy`)
- Dockerfile syntax error (test: `docker build .`)

**"Deploy to Cloud Run" fails:**
- Insufficient GCP permissions (re-run Step 1.4 from guide)
- GCP APIs not enabled (re-run Step 1.3)
- Invalid service account key

**"Smoke tests" fail:**
- Database connection error (check TURSO_URL and token)
- Application crash on startup (check logs)
- Port mismatch (ensure port 8500 is exposed)

**Quick diagnosis:**
```bash
# Check GitHub Actions logs
# Click on failed job → expand failed step

# Check Cloud Run logs
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  --limit 50
```

### Q: Deployment succeeded but service returns 502/503

**A:** Service is deployed but not healthy. Common causes:

**1. Database connection failure:**
```bash
# Verify Turso credentials
turso db shell sql-studio-prod "SELECT 1"

# Check secret values in GCP
gcloud secrets versions access latest --secret=turso-url
gcloud secrets versions access latest --secret=turso-auth-token
```

**2. Application crash:**
```bash
# Check logs for panic/error
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  | grep -E "panic|fatal|error"
```

**3. Port misconfiguration:**
- Cloud Run expects port 8500 (set in deployment)
- Dockerfile exposes port 8500
- App listens on SERVER_HTTP_PORT=8500
- If mismatch, update in workflow: `--port=8500`

**4. Health check timeout:**
```bash
# Test health endpoint directly
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region us-central1 --format 'value(status.url)')
curl -v $SERVICE_URL/health
```

### Q: How do I rollback a bad deployment?

**A:** Three methods:

**Method 1: Automatic rollback**
- GitHub Actions automatically rolls back if smoke tests fail
- No action needed!

**Method 2: GCP Console (easiest)**
1. Go to Cloud Run → sql-studio-backend
2. Click "REVISIONS" tab
3. Find previous working revision
4. Click "..." → "Manage traffic"
5. Route 100% to previous revision
6. Click "SAVE"

**Method 3: gcloud CLI (fastest)**
```bash
# List revisions
gcloud run revisions list \
  --service sql-studio-backend \
  --region us-central1

# Rollback to specific revision
gcloud run services update-traffic sql-studio-backend \
  --region us-central1 \
  --to-revisions sql-studio-backend-00042-abc=100
```

**Method 4: Re-deploy old version**
```bash
# Delete bad tag
git tag -d v1.2.0
git push origin :refs/tags/v1.2.0

# Re-tag old working version
git checkout v1.1.0
git tag v1.2.1
git push origin v1.2.1
```

### Q: Deployment is stuck/hanging for 15+ minutes

**A:** Likely Cloud Build timeout. Solutions:

**Cancel and retry:**
```bash
# In GitHub Actions: Click "Cancel workflow"

# In GCP Console: Cloud Build → Cancel stuck build

# Retry deployment
git tag v1.0.1  # New tag
git push origin v1.0.1
```

**Increase timeout (if needed):**

Edit `.github/workflows/deploy-backend.yml`:
```yaml
timeout-minutes: 30  # Default is 20
```

**Or increase Cloud Build timeout:**
Edit `backend-go/cloudbuild.yaml`:
```yaml
timeout: '3600s'  # 60 minutes
```

---

## Configuration & Secrets

### Q: How do I update my JWT secret?

**A:** Update in GCP Secret Manager, service auto-picks up on restart:

```bash
# Generate new secret
NEW_SECRET=$(openssl rand -base64 64)

# Update in GCP Secret Manager
echo -n "$NEW_SECRET" | gcloud secrets versions add jwt-secret \
  --data-file=-

# Also update in GitHub Secrets (for future deployments)
# Go to GitHub: Settings → Secrets → Edit JWT_SECRET

# Force service restart to pick up new secret
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --update-env-vars RESTART=$(date +%s)
```

**Warning:** Updating JWT_SECRET will invalidate all existing user sessions. Users will need to log in again.

### Q: How do I change environment variables?

**A:** Update in Cloud Run deployment:

```bash
# Update a single variable
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --update-env-vars LOG_LEVEL=debug

# Update multiple variables
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --set-env-vars="LOG_LEVEL=debug,LOG_FORMAT=text"

# Remove a variable
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --remove-env-vars LOG_LEVEL
```

**Or edit workflow file** for permanent changes:

Edit `.github/workflows/deploy-backend.yml`:
```yaml
--set-env-vars="ENVIRONMENT=production,LOG_LEVEL=info,NEW_VAR=value"
```

### Q: Where are my secrets stored?

**A:** Secrets are stored securely:

**GCP Cloud Run:**
- Stored in: GCP Secret Manager (encrypted at rest)
- Access: Only Cloud Run service account has access
- Rotation: Create new version, old versions retained
- View: `gcloud secrets list` (shows names only, not values)

**Fly.io:**
- Stored in: Fly.io Secrets vault (encrypted)
- Access: Only your app can access
- View: `flyctl secrets list` (shows names only)

**GitHub:**
- Stored in: GitHub Secrets (encrypted)
- Access: Only GitHub Actions workflows
- View: Cannot view after creation (must re-create to change)

**Never commit secrets to Git!**

### Q: How do I add a new secret?

**A:**

**For GCP Cloud Run:**

```bash
# 1. Create secret in GCP Secret Manager
echo -n "secret-value" | gcloud secrets create new-secret-name \
  --replication-policy="automatic" \
  --data-file=-

# 2. Grant Cloud Run access
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID \
  --format="value(projectNumber)")

gcloud secrets add-iam-policy-binding new-secret-name \
  --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# 3. Update Cloud Run deployment to use secret
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --update-secrets=NEW_SECRET_ENV_VAR=new-secret-name:latest

# 4. Update GitHub workflow for future deployments
# Edit .github/workflows/deploy-backend.yml:
# --set-secrets="...,NEW_SECRET_ENV_VAR=new-secret-name:latest"
```

**For Fly.io:**

```bash
flyctl secrets set NEW_SECRET_NAME="secret-value"
# App automatically restarts with new secret
```

---

## Database & Storage

### Q: Can I use a different database instead of Turso?

**A:** Yes, but requires code changes. Turso uses libSQL (SQLite-compatible), so you can use:

**Option 1: Local SQLite**
```bash
# In .env
TURSO_URL=file:./data/production.db
TURSO_AUTH_TOKEN=  # Leave empty
```

**Option 2: PostgreSQL**
- Requires code changes (different database driver)
- Replace `github.com/tursodatabase/libsql-client-go` with `github.com/lib/pq`
- Update connection logic in `internal/storage/`

**Option 3: MySQL**
- Similar to PostgreSQL (use `github.com/go-sql-driver/mysql`)

**We recommend sticking with Turso** because:
- Free tier is generous (1B row reads/month)
- Edge replication for low latency globally
- SQLite-compatible (easy migration)
- Automatic backups included

### Q: How do I run database migrations?

**A:** Two options:

**Option 1: Manual (via Turso CLI)**
```bash
# Create migration SQL file
cat > migration.sql << EOF
CREATE TABLE IF NOT EXISTS new_table (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL
);
EOF

# Apply migration
turso db shell sql-studio-prod < migration.sql

# Or interactively
turso db shell sql-studio-prod
```

**Option 2: Automated (via migration tool)**

Howlerops includes a migration command (future enhancement):
```bash
# Run migrations on deployment
# Edit cloudbuild.yaml or workflow to add:
# - ./sql-studio-backend migrate
```

**Best practice:**
1. Test migration on dev database first
2. Backup production before migration: `turso db shell sql-studio-prod .dump > backup.sql`
3. Apply migration
4. Verify data integrity

### Q: How do I backup my database?

**A:** Turso provides automatic backups:

**View backup settings:**
```bash
turso db show sql-studio-prod
# Shows: Point-in-time recovery window (24 hours on free tier)
```

**Manual export:**
```bash
# Export entire database to SQL
turso db shell sql-studio-prod .dump > backup-$(date +%Y%m%d).sql

# Compress for storage
gzip backup-$(date +%Y%m%d).sql

# Upload to cloud storage
gsutil cp backup-*.sql.gz gs://your-backup-bucket/
```

**Automated backups (recommended for production):**

```bash
# Create script: scripts/backup-database.sh
#!/bin/bash
DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="sql-studio-backup-$DATE.sql"

turso db shell sql-studio-prod .dump > $BACKUP_FILE
gzip $BACKUP_FILE
gsutil cp $BACKUP_FILE.gz gs://sql-studio-backups/

# Keep only last 30 days
gsutil -m rm gs://sql-studio-backups/*$(date -d '30 days ago' +%Y%m%d)*.sql.gz

# Schedule via cron or Cloud Scheduler
```

### Q: What happens if Turso goes down?

**A:** Turso has high availability:

- **Multi-region replication:** Data replicated across multiple regions
- **99.9% SLA:** On paid plans (free tier: best effort)
- **Automatic failover:** Switches to healthy replicas automatically

**Your app:**
- Will return database errors while Turso is down
- Health check will fail (Cloud Run won't route traffic to unhealthy instances)
- Automatically recovers when Turso comes back

**Mitigation strategies:**
1. **Add retry logic** (already included in libSQL driver)
2. **Cache frequently accessed data** (optional, add Redis)
3. **Implement circuit breaker** (optional, for graceful degradation)
4. **Monitor Turso status:** https://status.turso.tech

**For critical production apps:**
- Upgrade to Turso paid plan (99.9% SLA)
- Set up status page monitoring
- Consider database fallback strategy

---

## Performance & Scaling

### Q: How do I scale up to handle more traffic?

**A:** Cloud Run auto-scales, but you can tune:

**Horizontal scaling (more instances):**
```bash
# Increase max instances
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --max-instances 50

# Set minimum instances (eliminates cold starts but costs more)
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --min-instances 2
```

**Vertical scaling (more resources per instance):**
```bash
# Increase CPU and memory
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --cpu 4 \
  --memory 2Gi
```

**Concurrency (more requests per instance):**
```bash
# Increase concurrent requests per instance
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --concurrency 200
```

**Typical configurations:**

| Traffic Level | CPU | Memory | Min Instances | Max Instances | Concurrency |
|---------------|-----|--------|---------------|---------------|-------------|
| Low (< 10 req/s) | 1 | 512Mi | 0 | 10 | 80 |
| Medium (10-100 req/s) | 2 | 1Gi | 1 | 50 | 100 |
| High (100-1000 req/s) | 4 | 2Gi | 5 | 200 | 150 |

### Q: How do I reduce cold start time?

**A:** Several strategies:

**1. Set minimum instances (costs more)**
```bash
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --min-instances 1
```

**2. Optimize Docker image size**
- Already optimized! (~30MB final image)
- Multi-stage build with Alpine
- Binary stripped of debug symbols

**3. Use CPU boost (startup CPU allocation)**
```bash
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --cpu-boost  # Allocates more CPU during startup
```

**4. Implement lazy initialization**
- Move heavy initialization to first request
- Already implemented in Howlerops

**Current cold start time:** ~1-2 seconds (very good for Go apps)

**If < 3 second cold starts are critical:** Set `min-instances=1` (costs ~$3-5/month)

### Q: My API is slow. How do I diagnose?

**A:** Check multiple layers:

**1. Application performance:**
```bash
# Check Cloud Run metrics
# Go to: Cloud Run → sql-studio-backend → METRICS
# Look at: Request latency, CPU utilization

# Check for slow queries in logs
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  | grep -E "duration|latency" \
  | grep -E "[0-9]{3,}ms"  # Queries > 100ms
```

**2. Database performance:**
```bash
# Check Turso latency
# Visit: https://turso.tech/dashboard
# Look at: Query latency metrics

# Test direct database connection
time turso db shell sql-studio-prod "SELECT COUNT(*) FROM users"
```

**3. Network latency:**
```bash
# Test from different locations
curl -w "@-" -o /dev/null -s $SERVICE_URL/health << EOF
time_namelookup:  %{time_namelookup}
time_connect:  %{time_connect}
time_appconnect:  %{time_appconnect}
time_starttransfer:  %{time_starttransfer}
time_total:  %{time_total}
EOF
```

**Common fixes:**
- **Slow queries:** Add database indexes
- **High CPU:** Optimize algorithms, add caching
- **Network latency:** Use CDN, deploy closer to users
- **Cold starts:** Set min-instances=1

### Q: Can I deploy to multiple regions?

**A:** Yes, for global low latency:

**GCP Cloud Run (multiple regions):**

```bash
# Deploy to Europe
gcloud run deploy sql-studio-backend-eu \
  --image gcr.io/$PROJECT_ID/sql-studio-backend \
  --region europe-west1 \
  --platform managed

# Deploy to Asia
gcloud run deploy sql-studio-backend-asia \
  --image gcr.io/$PROJECT_ID/sql-studio-backend \
  --region asia-northeast1 \
  --platform managed

# Set up global load balancer
# Follow: https://cloud.google.com/load-balancing/docs/https/setup-global-ext-https-serverless
```

**Fly.io (simpler multi-region):**

```bash
# Add regions
flyctl regions add lhr  # London
flyctl regions add nrt  # Tokyo
flyctl regions add syd  # Sydney

# Scale each region
flyctl scale count 3  # Total across all regions

# Fly automatically routes to nearest region!
```

**Cost implications:**
- GCP: ~2-3x cost (separate deployments per region)
- Fly.io: ~2-3x cost (machines in each region)

**Recommendation:** Start single-region, add regions when you have users globally.

---

## Costs & Billing

### Q: Why am I being charged when I thought it was free?

**A:** Check what's using resources:

```bash
# View billing breakdown
# Go to: https://console.cloud.google.com/billing

# Check Cloud Run metrics
gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format json | jq '.status.traffic'

# Common causes:
# 1. min-instances > 0 (always-on instances)
# 2. High request volume (> 2M/month)
# 3. High network egress (> 1GB/month)
# 4. Other GCP services running
```

**Verify free tier settings:**
```bash
# Should be 0 for scale-to-zero
gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format 'value(spec.template.spec.autoscaling.minScale)'
```

**Set budget alert:**
```bash
# Go to: Billing → Budgets & alerts
# Create alert at $5, $10, $20
```

### Q: How can I reduce my costs?

**A:** Multiple strategies:

**1. Enable scale-to-zero (biggest savings)**
```bash
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --min-instances 0
```

**2. Enable CPU throttling**
```bash
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --cpu-throttling
```

**3. Right-size resources**
```bash
# Start small, scale up only if needed
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --memory 256Mi \
  --cpu 1
```

**4. Reduce max instances**
```bash
# Prevent runaway costs from traffic spikes
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --max-instances 10
```

**5. Optimize cold starts vs. cost**
- Cold starts (min=0): $0-2/month, 1-2s startup
- Always-on (min=1): $3-5/month, 0s startup
- Choose based on your needs

**6. Monitor and optimize**
```bash
# Check what's using resources
gcloud monitoring time-series list \
  --filter 'metric.type="run.googleapis.com/request_count"'
```

### Q: What's included in the free tier?

**A:** Very generous limits:

**GCP Cloud Run (always free tier):**
- 2 million requests per month
- 360,000 GB-seconds of memory per month
- 180,000 vCPU-seconds per month
- 1 GB network egress per month (North America)
- Unlimited inbound network traffic

**Example: What fits in free tier?**
- 2M requests/month
- Each request: 512MB RAM, 500ms duration
- = 1M GB-seconds (well within 360K limit!)
- **Typical hobby app:** 10-100K requests/month = $0

**Turso (free tier, always free):**
- 500 databases
- 9 GB total storage across all databases
- 1 billion row reads per month
- Point-in-time recovery (24 hours)

**Resend (free tier, always free):**
- 3,000 emails per month
- 100 emails per day
- 1 custom domain
- Email API access

**Total cost for small project:** $0/month

### Q: When should I upgrade from free tier?

**A:** Upgrade when you consistently hit limits:

**Cloud Run:**
- Consistently > 2M requests/month → ~$5-15/month
- Need more than 10 instances → Enterprise traffic
- Need SLA/support → Paid support plan

**Turso:**
- > 9GB storage → Starter plan ($29/month, 50GB)
- > 1B row reads/month → Starter plan
- Need > 24hr recovery → Pro plan ($99/month)
- Need dedicated support → Enterprise plan

**Resend:**
- > 3,000 emails/month → Pro plan ($20/month, 50K emails)
- > 100 emails/day → Pro plan (1,500/day)
- Need dedicated IPs → Business plan

**Most projects stay free tier for months/years!**

---

## Security & Authentication

### Q: Is my deployment secure?

**A:** Yes, if you followed the guide. Security features:

**Container security:**
- Non-root user (UID 1001)
- Minimal base image (Alpine Linux)
- No shell access to container
- Binary stripped (no debug symbols)

**Network security:**
- HTTPS enforced (automatic SSL via Cloud Run/Fly.io)
- TLS 1.2+ only
- Private networking for internal services

**Secret management:**
- Secrets in Secret Manager (encrypted at rest and in transit)
- No secrets in environment variables or code
- Service account with minimal permissions

**Application security:**
- JWT token authentication
- Bcrypt password hashing (cost 12)
- SQL injection protection (parameterized queries)
- XSS protection (input sanitization)
- CORS configured

**Additional hardening (optional):**

**1. Enable Cloud Armor (DDoS protection)**
```bash
# For high-traffic production apps
# Follow: https://cloud.google.com/armor/docs
```

**2. Rotate secrets regularly**
```bash
# JWT secret every 90 days
# Database tokens every 90 days
# API keys every 90 days
```

**3. Enable audit logging**
```bash
# Track all access to resources
# Follow: https://cloud.google.com/logging/docs/audit
```

### Q: How do I rotate secrets without downtime?

**A:** Use versioned secrets:

**1. Create new secret version:**
```bash
NEW_SECRET=$(openssl rand -base64 64)
echo -n "$NEW_SECRET" | gcloud secrets versions add jwt-secret \
  --data-file=-
```

**2. Cloud Run automatically uses latest version**
- No need to redeploy!
- New instances use new secret
- Old instances continue with old secret

**3. Graceful transition:**
```bash
# Wait for old instances to drain (5-10 minutes)
# Or force restart:
gcloud run services update sql-studio-backend \
  --region us-central1
```

**For JWT secrets:** You may want to support both old and new for transition period (requires code changes).

### Q: How do I enable authentication on my API?

**A:** API already includes JWT authentication:

**Register user:**
```bash
curl -X POST $SERVICE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'
```

**Login:**
```bash
TOKEN=$(curl -X POST $SERVICE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }' | jq -r '.token')
```

**Use token:**
```bash
curl $SERVICE_URL/api/v1/protected-endpoint \
  -H "Authorization: Bearer $TOKEN"
```

**To require auth on Cloud Run level (all requests):**
```bash
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --no-allow-unauthenticated
```

**To add API key authentication** (requires code changes):
- Add API key generation endpoint
- Store API keys in database (hashed)
- Validate API key in middleware

---

## Monitoring & Debugging

### Q: How do I view logs?

**A:** Multiple options:

**Option 1: gcloud CLI (fastest)**
```bash
# Real-time logs
gcloud run services logs tail sql-studio-backend \
  --region us-central1

# Last 100 lines
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  --limit 100

# Filter by severity
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  --limit 50 \
  | grep ERROR
```

**Option 2: GCP Console (best for exploration)**
1. Go to: https://console.cloud.google.com/logs
2. Filter: `resource.type="cloud_run_revision"`
3. Click on log entries to expand

**Option 3: Cloud Logging query language**
```bash
gcloud logging read 'resource.type="cloud_run_revision"
  AND resource.labels.service_name="sql-studio-backend"
  AND severity="ERROR"' \
  --limit 50 \
  --format json
```

**Logs are JSON structured:**
```json
{
  "timestamp": "2025-10-23T12:34:56Z",
  "severity": "INFO",
  "message": "Request completed",
  "method": "GET",
  "path": "/api/v1/users",
  "status": 200,
  "duration": "15ms"
}
```

### Q: How do I debug a production issue?

**A:** Systematic approach:

**1. Check service health:**
```bash
curl $SERVICE_URL/health

# Expected: {"status":"healthy",...}
# If not healthy, check logs
```

**2. Check recent logs for errors:**
```bash
gcloud run services logs read sql-studio-backend \
  --region us-central1 \
  --limit 100 \
  | grep -E "ERROR|FATAL|panic"
```

**3. Check metrics:**
```bash
# Go to Cloud Run → sql-studio-backend → METRICS
# Look for:
# - Error rate spike
# - Latency increase
# - CPU/memory saturation
```

**4. Reproduce locally:**
```bash
# Run same code locally with production config
cd backend-go
cp .env.production.example .env.debug

# Use production database (read-only)
TURSO_URL="libsql://..." \
TURSO_AUTH_TOKEN="..." \
go run cmd/server/main.go

# Test problematic endpoint
curl http://localhost:8080/problematic-endpoint
```

**5. Enable debug logging (temporarily):**
```bash
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --update-env-vars LOG_LEVEL=debug

# Check logs
gcloud run services logs tail sql-studio-backend \
  --region us-central1

# Revert after debugging
gcloud run services update sql-studio-backend \
  --region us-central1 \
  --update-env-vars LOG_LEVEL=info
```

**6. Check dependencies:**
```bash
# Test Turso database
turso db shell sql-studio-prod "SELECT 1"

# Test Resend API
curl https://api.resend.com/emails \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"from":"onboarding@resend.dev","to":"test@test.com","subject":"Test","html":"<p>Test</p>"}'
```

### Q: How do I set up alerts?

**A:** Use Cloud Monitoring:

**Create uptime check:**
1. Go to: Monitoring → Uptime checks
2. Create check for: `$SERVICE_URL/health`
3. Frequency: 5 minutes
4. Regions: Global

**Create alert policy:**
```bash
# Via gcloud (example)
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="Backend Down Alert" \
  --condition-threshold-value=1 \
  --condition-filter='resource.type="cloud_run_revision" AND metric.type="run.googleapis.com/request_count"'
```

**Or use external monitoring (easier):**
- **UptimeRobot:** https://uptimerobot.com (free: 50 monitors)
- **Pingdom:** https://www.pingdom.com
- **Better Uptime:** https://betteruptime.com

**Recommended alerts:**
- Health endpoint down (critical)
- Error rate > 5% (warning)
- Response time > 2s (warning)
- CPU > 80% (info)

---

## Platform-Specific Questions

### Q: GCP vs Fly.io - What are the key differences?

**A:**

| Feature | GCP Cloud Run | Fly.io |
|---------|---------------|--------|
| **Pricing (low traffic)** | $0-2/month | $0/month |
| **Pricing (high traffic)** | More expensive | Cheaper |
| **Free tier** | 2M requests/month | 3 VMs + 160GB egress |
| **Cold start** | ~1-2s | ~1-2s |
| **Max instances** | 1000 | 250 |
| **Regions** | 30+ (manual setup) | 30+ (auto-routing) |
| **SLA** | 99.5% (no SLA on free tier) | Best effort |
| **Monitoring** | Built-in (Cloud Monitoring) | Basic (external needed) |
| **Support** | Extensive docs, paid support | Community, Discord |
| **Compliance** | SOC 2, HIPAA, ISO | Limited |
| **Best for** | Enterprise, compliance-heavy | Hobby, startups, global apps |

**Our recommendation:**
- **Hobby/MVP:** Fly.io (cheaper, simpler)
- **Production/Business:** GCP Cloud Run (better reliability, support)

### Q: Can I deploy to both GCP and Fly.io?

**A:** Yes! Multi-cloud for redundancy:

```bash
# Deploy to GCP (primary)
git tag v1.0.0
git push origin v1.0.0
# GitHub Actions deploys to Cloud Run

# Deploy to Fly.io (backup)
cd backend-go
flyctl deploy
```

**Use DNS failover:**
- Primary: GCP Cloud Run
- Backup: Fly.io (if GCP fails)
- Use health checks to auto-switch

**Cost:** Running both doubles costs (but provides redundancy)

### Q: How do I migrate from Fly.io to GCP (or vice versa)?

**A:** Both use same Docker image:

**Fly.io → GCP:**
```bash
# Already set up for GCP in GitHub Actions
# Just push a tag:
git tag v1.0.0
git push origin v1.0.0

# Update DNS to point to Cloud Run URL
```

**GCP → Fly.io:**
```bash
cd backend-go

# Deploy to Fly.io
flyctl deploy

# Set secrets
flyctl secrets set \
  TURSO_URL="$TURSO_URL" \
  TURSO_AUTH_TOKEN="$TURSO_AUTH_TOKEN" \
  JWT_SECRET="$JWT_SECRET"

# Update DNS to point to Fly.io URL
```

**No code changes needed!** Just different deployment commands.

---

## Additional Resources

- **Complete Deployment Guide:** `/backend-go/DEPLOYMENT_COMPLETE_GUIDE.md`
- **Quick Start Guide:** `/backend-go/DEPLOYMENT_QUICK_START.md`
- **Original Deployment Docs:** `/backend-go/DEPLOYMENT.md`
- **Cost Analysis:** `/backend-go/COST_ANALYSIS.md`
- **API Documentation:** `/backend-go/API_DOCUMENTATION.md`

**External Resources:**
- GCP Cloud Run: https://cloud.google.com/run/docs
- Fly.io: https://fly.io/docs
- Turso: https://docs.turso.tech
- Resend: https://resend.com/docs
- GitHub Actions: https://docs.github.com/en/actions

**Community Support:**
- GitHub Issues: https://github.com/sql-studio/sql-studio/issues
- GitHub Discussions: https://github.com/sql-studio/sql-studio/discussions
- Turso Discord: https://discord.gg/turso

---

**Document Version:** 1.0.0
**Last Updated:** 2025-10-23
**Maintainer:** Howlerops Team

**Have a question not answered here?** Open a GitHub Discussion!
