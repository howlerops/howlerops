# Production Deployment Checklist

## Pre-Deployment

### 1. Environment Configuration
- [ ] `TURSO_URL` set to production database (libsql://your-db.turso.io)
- [ ] `TURSO_AUTH_TOKEN` set and verified
- [ ] `JWT_SECRET` generated (32+ random bytes, use: `openssl rand -base64 32`)
- [ ] `RESEND_API_KEY` configured for email
- [ ] `RESEND_FROM_EMAIL` set to verified domain
- [ ] `GCP_PROJECT_ID` set to your GCP project
- [ ] `ENVIRONMENT=production`
- [ ] `LOG_LEVEL=info` (not debug)
- [ ] `LOG_FORMAT=json` for structured logging
- [ ] `ALLOWED_ORIGINS` set to production domains only

### 2. Security
- [ ] JWT_SECRET is cryptographically random (never reuse dev secrets)
- [ ] No hardcoded credentials in code
- [ ] All secrets in GCP Secret Manager (not in env vars)
- [ ] CORS configured for production domains only
- [ ] Rate limiting enabled (configured in code)
- [ ] HTTPS enforced (automatic with Cloud Run)
- [ ] No .env files committed to git
- [ ] Security headers configured (HSTS, CSP, etc.)

### 3. Database (Turso)
- [ ] Turso database created for production
- [ ] Schema migrations tested locally
- [ ] Database backups configured (Turso auto-backup enabled)
- [ ] Connection pooling optimized (max_open_conns, max_idle_conns)
- [ ] Test connection from Cloud Run region (latency check)
- [ ] Database indexes created for performance
- [ ] Turso organization has billing enabled (if needed)

### 4. Email (Resend)
- [ ] Resend API key valid and active
- [ ] Email domain verified in Resend
- [ ] SPF/DKIM records configured
- [ ] Email templates tested (signup, login)
- [ ] Rate limits understood (100 emails/day free tier)
- [ ] From address uses verified domain
- [ ] Test email delivery to common providers (Gmail, Outlook)

### 5. Testing
- [ ] All unit tests pass: `make test`
- [ ] Integration tests pass: `make test-integration`
- [ ] Build succeeds: `make build`
- [ ] Load testing completed (use `hey` or `ab`)
- [ ] Health endpoint responds: `curl /health`
- [ ] Metrics endpoint responds: `curl /metrics`
- [ ] No race conditions: `go test -race ./...`

### 6. Monitoring & Observability
- [ ] Cloud Logging enabled (automatic with Cloud Run)
- [ ] Cloud Monitoring configured
- [ ] Error reporting set up (Cloud Error Reporting)
- [ ] Uptime checks configured
- [ ] Log-based metrics created
- [ ] Alerting policies configured (error rate, latency)
- [ ] Dashboard created for key metrics
- [ ] Log retention policy set (30 days recommended)

### 7. Cloud Run Configuration
- [ ] Service name: `sql-studio-backend`
- [ ] Region: `us-central1` (lowest cost, central location)
- [ ] Min instances: `0` (scale-to-zero for cost savings)
- [ ] Max instances: `10` (prevent runaway costs)
- [ ] CPU allocation: `CPU is only allocated during request processing`
- [ ] Memory: `512Mi` (should be sufficient, can scale up)
- [ ] Timeout: `300s` (5 minutes)
- [ ] Concurrency: `80` (default, adjust based on load testing)
- [ ] Execution environment: `gen2` (better performance)
- [ ] Ingress: `All` or `Internal and Cloud Load Balancing`
- [ ] Authentication: `Allow unauthenticated invocations` (public API)

### 8. GCP Project Setup
- [ ] Billing account linked
- [ ] APIs enabled (Cloud Run, Cloud Build, Secret Manager, Logging, Monitoring)
- [ ] IAM permissions configured
- [ ] Budget alerts configured ($5/month threshold)
- [ ] Service account has minimal permissions
- [ ] Cloud Build service account can deploy Cloud Run

## Deployment Steps

### Initial Setup (One-time)

```bash
# 1. Set GCP project
export GCP_PROJECT_ID=your-project-id
gcloud config set project $GCP_PROJECT_ID

# 2. Enable required APIs
gcloud services enable \
    cloudbuild.googleapis.com \
    run.googleapis.com \
    secretmanager.googleapis.com \
    logging.googleapis.com \
    monitoring.googleapis.com

# 3. Set environment variables
export TURSO_URL=libsql://your-db.turso.io
export TURSO_AUTH_TOKEN=your-token
export RESEND_API_KEY=re_your-key
export JWT_SECRET=$(openssl rand -base64 32)

# 4. Create secrets
./scripts/setup-secrets.sh
```

### Every Deployment

```bash
# Option 1: Full automated deployment
./scripts/deploy-full.sh

# Option 2: Step-by-step
make prod-check          # 1. Verify readiness
make setup-gcp-secrets   # 2. Update secrets (if changed)
make deploy-prod         # 3. Deploy to Cloud Run
make verify-prod         # 4. Verify deployment
```

### Manual Deployment

```bash
# 1. Check readiness
./scripts/prod-readiness-check.sh

# 2. Build and deploy
gcloud builds submit --config cloudbuild.yaml

# 3. Deploy to Cloud Run
gcloud run deploy sql-studio-backend \
    --source . \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json

# 4. Get service URL
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
    --region=us-central1 \
    --format='value(status.url)')

# 5. Verify deployment
./scripts/verify-deployment.sh $SERVICE_URL
```

## Post-Deployment Verification

### Automated Checks

```bash
# Run verification suite
./scripts/verify-deployment.sh $SERVICE_URL
```

### Manual Checks

- [ ] Health endpoint returns 200: `curl $SERVICE_URL/health`
- [ ] Metrics endpoint accessible: `curl $SERVICE_URL/metrics`
- [ ] Signup flow works: Test user registration
- [ ] Login flow works: Test user login
- [ ] JWT refresh works: Test token refresh
- [ ] Database queries work: Test sync endpoints
- [ ] Email sending works: Verify magic link delivery
- [ ] Logs are structured JSON: Check Cloud Logging
- [ ] Metrics are reporting: Check Cloud Monitoring
- [ ] CORS headers correct: Test from frontend domain
- [ ] Rate limiting working: Test excessive requests

### Endpoint Tests

```bash
# Health check
curl -f $SERVICE_URL/health

# Signup
curl -X POST $SERVICE_URL/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"TestPass123"}'

# Login
curl -X POST $SERVICE_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"TestPass123"}'

# Protected endpoint (use token from login)
curl -H "Authorization: Bearer $TOKEN" $SERVICE_URL/api/sync/download
```

## Monitoring Dashboard

### Key Metrics to Watch

1. **Request Count**: Should be < 2M/month (free tier)
2. **Error Rate**: Should be < 1%
3. **Latency**: p50 < 200ms, p95 < 500ms, p99 < 1s
4. **Instance Count**: Should scale to 0 when idle
5. **Memory Usage**: Should stay under 512Mi
6. **Cost**: Should be $0-3/month

### Access Monitoring

```bash
# View logs
make prod-logs

# Or directly
gcloud run services logs tail sql-studio-backend \
    --project=$GCP_PROJECT_ID

# Check service status
make prod-status

# Check costs
make check-costs
```

### Cloud Console Links

- **Cloud Run Dashboard**: https://console.cloud.google.com/run
- **Logs Explorer**: https://console.cloud.google.com/logs
- **Monitoring**: https://console.cloud.google.com/monitoring
- **Error Reporting**: https://console.cloud.google.com/errors
- **Billing**: https://console.cloud.google.com/billing

## Rollback Plan

### If Deployment Fails

```bash
# 1. Check recent logs
gcloud run services logs tail sql-studio-backend --limit=50

# 2. Identify last working revision
gcloud run revisions list --service=sql-studio-backend

# 3. Rollback to previous revision
gcloud run services update-traffic sql-studio-backend \
    --to-revisions=sql-studio-backend-00001-abc=100

# 4. Investigate the issue
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
    --limit=50 \
    --format=json
```

### If Service is Degraded

```bash
# 1. Scale down to prevent damage
gcloud run services update sql-studio-backend \
    --max-instances=1

# 2. Check error rate
gcloud monitoring time-series list \
    --filter='metric.type="run.googleapis.com/request_count"'

# 3. Fix and redeploy
# Fix the issue in code
./scripts/deploy-full.sh

# 4. Restore scaling
gcloud run services update sql-studio-backend \
    --max-instances=10
```

### Emergency Stop

```bash
# Stop all traffic (requires load balancer setup)
gcloud run services update-traffic sql-studio-backend \
    --to-revisions=MISSING=100

# Or delete service entirely (extreme measure)
gcloud run services delete sql-studio-backend --region=us-central1
```

## Cost Optimization

### Free Tier Limits (as of 2024)

- **Requests**: 2M requests/month
- **Compute**: 180,000 vCPU-seconds/month
- **Memory**: 360,000 GiB-seconds/month
- **Networking**: 1 GB outbound/month (within North America)

### Staying Under $3/month

- [ ] Min instances = 0 (scale-to-zero)
- [ ] Max instances = 10 (prevent runaway)
- [ ] Memory = 512Mi (not overprovisioned)
- [ ] CPU allocation = request processing only
- [ ] Region = us-central1 (lowest cost)
- [ ] Monitoring = basic (no custom metrics)
- [ ] Log retention = 30 days (not longer)

### Cost Monitoring

```bash
# Check current usage
./scripts/check-costs.sh

# Set budget alert
gcloud billing budgets create \
    --billing-account=YOUR_BILLING_ACCOUNT_ID \
    --display-name="SQL Studio Backend Budget" \
    --budget-amount=5USD \
    --threshold-rule=percent=80 \
    --threshold-rule=percent=100
```

## Troubleshooting

### Common Issues

**Issue**: Secrets not accessible
```bash
# Check secret exists
gcloud secrets list | grep turso

# Check IAM permissions
gcloud secrets get-iam-policy turso-url

# Grant access to Cloud Run service account
PROJECT_NUMBER=$(gcloud projects describe $GCP_PROJECT_ID --format="value(projectNumber)")
gcloud secrets add-iam-policy-binding turso-url \
    --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"
```

**Issue**: Cloud Build fails
```bash
# Check build logs
gcloud builds list --limit=5

# View specific build
gcloud builds log BUILD_ID

# Common fix: Enable Cloud Build API
gcloud services enable cloudbuild.googleapis.com
```

**Issue**: Service won't start
```bash
# Check recent logs
gcloud run services logs tail sql-studio-backend --limit=100

# Check service configuration
gcloud run services describe sql-studio-backend --region=us-central1

# Test locally with same environment
docker run -p 8080:8080 \
    -e TURSO_URL=$TURSO_URL \
    -e TURSO_AUTH_TOKEN=$TURSO_AUTH_TOKEN \
    gcr.io/$GCP_PROJECT_ID/sql-studio-backend:latest
```

**Issue**: High costs
```bash
# Check request count
gcloud monitoring time-series list \
    --filter='metric.type="run.googleapis.com/request_count"'

# Check instance count
gcloud monitoring time-series list \
    --filter='metric.type="run.googleapis.com/container/instance_count"'

# Reduce max instances
gcloud run services update sql-studio-backend --max-instances=5
```

## Security Checklist

- [ ] No secrets in code or git
- [ ] All secrets in Secret Manager
- [ ] JWT_SECRET never reused from development
- [ ] CORS restricted to known domains
- [ ] Rate limiting enabled
- [ ] Input validation on all endpoints
- [ ] SQL injection prevention (parameterized queries)
- [ ] XSS prevention (sanitized inputs)
- [ ] HTTPS only (enforced by Cloud Run)
- [ ] Security headers configured
- [ ] Dependency vulnerabilities scanned: `go list -json -m all | nancy sleuth`
- [ ] Container vulnerabilities scanned: `gcloud container images scan`

## Compliance & Best Practices

- [ ] Error messages don't leak sensitive data
- [ ] Logs don't contain PII or secrets
- [ ] Database connection pooling configured
- [ ] Graceful shutdown implemented
- [ ] Health checks respond quickly
- [ ] Timeouts configured appropriately
- [ ] Circuit breakers for external services
- [ ] Retry logic with exponential backoff
- [ ] Request IDs for tracing
- [ ] Structured logging throughout

## Sign-off

- [ ] **Developer**: Code reviewed and tested
- [ ] **Security**: Security checklist completed
- [ ] **Operations**: Monitoring and alerts configured
- [ ] **Product**: Smoke tests passed

**Deployed by**: _______________
**Date**: _______________
**Revision**: _______________
**Sign-off**: _______________

---

## Quick Reference

```bash
# Deploy
./scripts/deploy-full.sh

# Verify
./scripts/verify-deployment.sh $SERVICE_URL

# Monitor
make prod-logs
make prod-status
make check-costs

# Rollback
gcloud run services update-traffic sql-studio-backend \
    --to-revisions=PREVIOUS_REVISION=100
```

## Support

- **Documentation**: /backend-go/DEPLOYMENT.md
- **Architecture**: /backend-go/ARCHITECTURE.md
- **API Docs**: /backend-go/API_DOCUMENTATION.md
- **GCP Console**: https://console.cloud.google.com
- **Turso Console**: https://turso.tech/app
- **Resend Console**: https://resend.com/emails
