# SQL Studio Backend - Production Deployment Guide

Complete guide for deploying SQL Studio backend to Google Cloud Run with Turso database.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Step-by-Step Deployment](#step-by-step-deployment)
4. [Post-Deployment](#post-deployment)
5. [Monitoring & Maintenance](#monitoring--maintenance)
6. [Troubleshooting](#troubleshooting)
7. [Cost Management](#cost-management)

---

## Prerequisites

### Required Accounts & Services

- **Google Cloud Platform Account** (free tier available)
  - Create at: https://console.cloud.google.com
  - Enable billing (required for Cloud Run, but free tier covers most usage)

- **Turso Database** (free tier: 500MB storage, 1B row reads/month)
  - Sign up at: https://turso.tech
  - Create a database for production

- **Resend Email Service** (free tier: 100 emails/day)
  - Sign up at: https://resend.com
  - Verify your domain for sending emails

### Required Tools

```bash
# Install gcloud CLI
# macOS
brew install google-cloud-sdk

# Linux
curl https://sdk.cloud.google.com | bash

# Verify installation
gcloud version

# Authenticate
gcloud auth login
```

### Required Environment Variables

```bash
# Create .env.production file
cp .env.production.example .env.production

# Edit with your values
# Required:
export GCP_PROJECT_ID="your-gcp-project-id"
export TURSO_URL="libsql://your-database.turso.io"
export TURSO_AUTH_TOKEN="your-turso-auth-token"
export JWT_SECRET="$(openssl rand -base64 32)"
export RESEND_API_KEY="re_your-resend-api-key"

# Optional:
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
export ALLOWED_ORIGINS="https://yourdomain.com,https://www.yourdomain.com"
```

---

## Quick Start

### One-Command Deployment

```bash
# 1. Set environment variables
source .env.production

# 2. Deploy everything
make deploy-prod
```

That's it! The script will:
1. Check production readiness
2. Setup GCP secrets
3. Deploy to Cloud Run
4. Verify the deployment

---

## Step-by-Step Deployment

### Step 1: Prepare Your Environment

```bash
# Navigate to backend directory
cd backend-go

# Copy and configure production environment
cp .env.production.example .env.production

# Edit .env.production with your actual values
# IMPORTANT: Never commit this file to git!
nano .env.production

# Source the environment
source .env.production
```

### Step 2: Setup Turso Database

```bash
# Install Turso CLI (if not already installed)
brew install tursodatabase/tap/turso  # macOS
# or
curl -sSfL https://get.tur.so/install.sh | bash  # Linux

# Login to Turso
turso auth login

# Create production database
turso db create sql-studio-prod

# Get database URL
turso db show sql-studio-prod --url

# Create auth token
turso db tokens create sql-studio-prod

# Add these to your .env.production:
# TURSO_URL=libsql://sql-studio-prod-your-org.turso.io
# TURSO_AUTH_TOKEN=your-token-here
```

### Step 3: Setup Resend Email

```bash
# 1. Go to https://resend.com/api-keys
# 2. Create a new API key
# 3. Add to .env.production:
#    RESEND_API_KEY=re_your_key_here

# 4. Verify your domain at https://resend.com/domains
# 5. Add DNS records (SPF, DKIM)
# 6. Set RESEND_FROM_EMAIL=noreply@yourdomain.com
```

### Step 4: Setup GCP Project

```bash
# Create a new GCP project (or use existing)
gcloud projects create sql-studio-prod --name="SQL Studio Production"

# Set as default project
gcloud config set project sql-studio-prod

# Enable billing (required for Cloud Run)
# Visit: https://console.cloud.google.com/billing

# Export project ID
export GCP_PROJECT_ID=sql-studio-prod
```

### Step 5: Generate Secrets

```bash
# Generate a strong JWT secret (NEVER reuse dev secrets!)
export JWT_SECRET=$(openssl rand -base64 32)

# Verify all secrets are set
echo "Checking environment variables..."
echo "GCP_PROJECT_ID: ${GCP_PROJECT_ID:0:20}..."
echo "TURSO_URL: ${TURSO_URL:0:30}..."
echo "TURSO_AUTH_TOKEN: ${TURSO_AUTH_TOKEN:0:20}..."
echo "JWT_SECRET length: ${#JWT_SECRET} chars"
echo "RESEND_API_KEY: ${RESEND_API_KEY:0:10}..."
```

### Step 6: Pre-Deployment Checks

```bash
# Run production readiness check
make prod-check

# This will verify:
# - All environment variables are set
# - JWT secret is strong enough
# - Tests pass
# - Code is formatted
# - gcloud is installed and authenticated
# - GCP project is set correctly
```

### Step 7: Setup GCP Secrets

```bash
# Create secrets in GCP Secret Manager
make setup-gcp-secrets

# This will:
# - Enable required GCP APIs
# - Create secrets in Secret Manager
# - Grant Cloud Run access to secrets
# - Verify secrets are accessible
```

### Step 8: Deploy to Cloud Run

```bash
# Option 1: Full automated deployment
make deploy-prod

# Option 2: Manual deployment
gcloud run deploy sql-studio-backend \
  --source . \
  --region=us-central1 \
  --platform=managed \
  --allow-unauthenticated \
  --min-instances=0 \
  --max-instances=10 \
  --memory=512Mi \
  --timeout=300 \
  --set-env-vars="ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json" \
  --set-secrets="TURSO_URL=turso-url:latest,TURSO_AUTH_TOKEN=turso-auth-token:latest,JWT_SECRET=jwt-secret:latest,RESEND_API_KEY=resend-api-key:latest"
```

### Step 9: Verify Deployment

```bash
# Get service URL
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format='value(status.url)')

# Run verification tests
make verify-prod SERVICE_URL=$SERVICE_URL

# Or manually test
curl $SERVICE_URL/health
```

---

## Post-Deployment

### Configure Frontend

Update your frontend configuration to use the production API:

```typescript
// frontend/src/lib/config.ts
export const API_URL = 'https://sql-studio-backend-abc123-uc.a.run.app'
```

### Setup Custom Domain (Optional)

```bash
# Map custom domain to Cloud Run
gcloud run domain-mappings create \
  --service=sql-studio-backend \
  --domain=api.yourdomain.com \
  --region=us-central1

# Add DNS records as instructed by GCP
# Typically:
# api.yourdomain.com CNAME ghs.googlehosted.com
```

### Configure CORS for Production

Update allowed origins in your deployment:

```bash
# Redeploy with production CORS settings
gcloud run services update sql-studio-backend \
  --update-env-vars="ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com" \
  --region=us-central1
```

### Initial Smoke Tests

```bash
# 1. Health check
curl https://your-service-url.run.app/health

# 2. Create test user
curl -X POST https://your-service-url.run.app/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"TestPass123"}'

# 3. Login
curl -X POST https://your-service-url.run.app/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"TestPass123"}'

# 4. Test protected endpoint (use token from login)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://your-service-url.run.app/api/sync/download
```

---

## Monitoring & Maintenance

### View Logs

```bash
# Tail logs in real-time
make prod-logs

# Or directly
gcloud run services logs tail sql-studio-backend

# View specific error logs
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
  --limit=50 \
  --format=json
```

### Check Service Status

```bash
# Get service status
make prod-status

# List all revisions
make prod-revisions

# Check request metrics
gcloud monitoring time-series list \
  --filter='metric.type="run.googleapis.com/request_count"'
```

### Cost Monitoring

```bash
# Check current costs and usage
make check-costs

# View detailed billing
gcloud billing accounts list
```

### Setup Monitoring Alerts

```bash
# Create alert for high error rate
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="SQL Studio High Error Rate" \
  --condition-display-name="Error rate > 1%" \
  --condition-threshold-value=0.01 \
  --condition-threshold-duration=300s
```

### Setup Uptime Checks

```bash
# Create uptime check
gcloud monitoring uptime create sql-studio-health \
  --resource-type=uptime-url \
  --host=your-service-url.run.app \
  --path=/health \
  --check-interval=300s
```

---

## Troubleshooting

### Deployment Fails

```bash
# Check build logs
gcloud builds list --limit=5
gcloud builds log BUILD_ID

# Common issues:
# 1. Missing dependencies in go.mod
go mod tidy && git commit -am "Update dependencies"

# 2. Dockerfile errors
docker build -t test . --no-cache

# 3. Missing secrets
make setup-gcp-secrets
```

### Service Won't Start

```bash
# Check recent logs
gcloud run services logs tail sql-studio-backend --limit=100

# Check service configuration
gcloud run services describe sql-studio-backend --region=us-central1

# Common issues:
# 1. Secrets not accessible
gcloud secrets list
gcloud secrets get-iam-policy secret-name

# 2. Database connection fails
# Test locally first:
source .env.production
make dev
```

### High Error Rate

```bash
# View recent errors
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
  --limit=20 \
  --format=json

# Check error distribution
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
  --format="value(jsonPayload.message)" | sort | uniq -c
```

### Slow Performance

```bash
# Check latency metrics
gcloud monitoring time-series list \
  --filter='metric.type="run.googleapis.com/request_latencies"'

# Increase memory if needed
gcloud run services update sql-studio-backend \
  --memory=1Gi \
  --region=us-central1

# Increase max instances
gcloud run services update sql-studio-backend \
  --max-instances=20 \
  --region=us-central1
```

### Rollback Deployment

```bash
# List revisions
make prod-revisions

# Rollback to previous revision
make prod-rollback REVISION=sql-studio-backend-00001-abc

# Or automatically rollback to previous
PREV_REVISION=$(gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1 \
  --format="value(metadata.name)" \
  --limit=2 | tail -n 1)
make prod-rollback REVISION=$PREV_REVISION
```

---

## Cost Management

### Free Tier Limits

Google Cloud Run Free Tier (per month):
- 2M requests
- 180,000 vCPU-seconds
- 360,000 GiB-seconds
- 1 GB network egress (North America)

Turso Free Tier:
- 500 MB total storage
- 1 billion row reads
- 25 million row writes
- Up to 3 databases

Resend Free Tier:
- 100 emails per day
- 3,000 emails per month

### Staying Under $3/month

Configuration for minimal cost:

```yaml
# cloudrun.yaml settings
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "0"  # Scale to zero
        autoscaling.knative.dev/maxScale: "10" # Prevent runaway
    spec:
      containers:
      - resources:
          limits:
            memory: 512Mi  # Don't overprovision
            cpu: "1"       # 1 vCPU
      containerConcurrency: 80
      timeoutSeconds: 300
```

### Cost Optimization Tips

1. **Scale to Zero**: Set min instances to 0
2. **Limit Max Instances**: Set max instances to 10
3. **Right-size Memory**: Start with 512Mi, increase only if needed
4. **CPU Allocation**: Use "CPU is only allocated during request processing"
5. **Clean Up Old Revisions**: Keep only 2-3 recent revisions
6. **Monitor Usage**: Run `make check-costs` weekly
7. **Set Budget Alerts**: Alert at $5/month threshold

### Budget Alerts

```bash
# Set up budget alert
gcloud billing budgets create \
  --billing-account=YOUR_BILLING_ACCOUNT_ID \
  --display-name="SQL Studio Backend Budget" \
  --budget-amount=5USD \
  --threshold-rule=percent=80 \
  --threshold-rule=percent=100
```

---

## Production Checklist Reference

For a comprehensive deployment checklist, see [PRODUCTION_CHECKLIST.md](./PRODUCTION_CHECKLIST.md)

## Available Commands

```bash
# Deployment
make deploy-prod              # Full deployment (recommended)
make prod-check               # Check readiness
make setup-gcp-secrets        # Setup secrets
make verify-prod SERVICE_URL=https://... # Verify deployment

# Monitoring
make prod-status              # Service status
make prod-logs                # Tail logs
make check-costs              # Cost report
make prod-revisions           # List revisions

# Rollback
make prod-rollback REVISION=sql-studio-backend-00001-abc

# Help
make prod-help                # Production help
```

---

## Additional Resources

- [Production Checklist](./PRODUCTION_CHECKLIST.md) - Comprehensive deployment checklist
- [Architecture Documentation](./ARCHITECTURE.md) - System architecture
- [API Documentation](./API_DOCUMENTATION.md) - API reference
- [Deployment Summary](./DEPLOYMENT.md) - Deployment overview

## Support

- **Cloud Run Documentation**: https://cloud.google.com/run/docs
- **Turso Documentation**: https://docs.turso.tech
- **Resend Documentation**: https://resend.com/docs

---

**Last Updated**: October 2024
