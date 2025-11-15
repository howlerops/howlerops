# Howlerops Backend - Deployment Guide

Complete guide for deploying the Howlerops Go backend to production environments.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Environment Variables](#environment-variables)
- [Deployment Options](#deployment-options)
  - [Google Cloud Run](#google-cloud-run-recommended)
  - [Fly.io](#flyio)
- [CI/CD Setup](#cicd-setup)
- [Monitoring and Logging](#monitoring-and-logging)
- [Scaling](#scaling)
- [Rollback Procedures](#rollback-procedures)
- [Troubleshooting](#troubleshooting)
- [Cost Optimization](#cost-optimization)

---

## Overview

The Howlerops backend is a Go application that provides:
- **HTTP/REST API** on port 8500
- **gRPC API** on port 9500
- **Prometheus Metrics** on port 9100
- **Health checks** at `/health`

**Architecture:**
- Multi-stage Docker build for minimal image size
- Stateless design (all state in Turso database)
- Auto-scaling capabilities
- Zero-downtime deployments

---

## Prerequisites

### Required Tools

1. **Docker** (for local testing)
   ```bash
   docker --version  # Should be 20.10+
   ```

2. **For GCP Cloud Run:**
   ```bash
   # Install Google Cloud SDK
   curl https://sdk.cloud.google.com | bash
   gcloud init
   gcloud auth login
   ```

3. **For Fly.io:**
   ```bash
   # macOS
   brew install flyctl

   # Linux
   curl -L https://fly.io/install.sh | sh

   # Windows
   iwr https://fly.io/install.ps1 -useb | iex
   ```

### Required Accounts

1. **Google Cloud Platform** (for Cloud Run)
   - Create project at https://console.cloud.google.com
   - Enable billing
   - Note your PROJECT_ID

2. **Fly.io** (alternative deployment)
   - Sign up at https://fly.io
   - Add payment method (required even for free tier)

3. **Turso** (database)
   - Sign up at https://turso.tech
   - Create database
   - Get connection URL and auth token

4. **Resend** (email service)
   - Sign up at https://resend.com
   - Get API key
   - Verify sender domain

---

## Environment Variables

### Required Secrets

All deployments require these environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `TURSO_URL` | Turso database connection URL | `libsql://your-db.turso.io` |
| `TURSO_AUTH_TOKEN` | Turso authentication token | `eyJhbGc...` |
| `RESEND_API_KEY` | Resend email API key | `re_123...` |
| `RESEND_FROM_EMAIL` | Sender email address | `noreply@yourdomain.com` |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | Generate with: `openssl rand -base64 32` |

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `production` | Environment name |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |
| `LOG_FORMAT` | `json` | Log format (json/text) |
| `SERVER_HTTP_PORT` | `8500` | HTTP server port |
| `SERVER_GRPC_PORT` | `9500` | gRPC server port |

---

## Deployment Options

### Google Cloud Run (Recommended)

**Best for:** Production deployments with auto-scaling and high availability

#### Initial Setup (One-time)

1. **Set up Google Cloud project:**

```bash
# Set your project ID
export GCP_PROJECT_ID="your-project-id"

# Set as default project
gcloud config set project $GCP_PROJECT_ID

# Enable required APIs
gcloud services enable \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com
```

2. **Create service account:**

```bash
gcloud iam service-accounts create sql-studio-backend \
  --display-name="Howlerops Backend"

# Grant necessary roles
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:sql-studio-backend@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

3. **Set environment variables:**

```bash
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-token"
export RESEND_API_KEY="re_your-key"
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
export JWT_SECRET=$(openssl rand -base64 32)
```

#### Deployment Methods

**Method 1: Automated Script (Recommended)**

```bash
cd backend-go
./scripts/deploy-cloudrun.sh
```

**Method 2: Cloud Build**

```bash
cd backend-go
gcloud builds submit --config cloudbuild.yaml
```

**Method 3: Manual gcloud command**

```bash
# Build and push image
gcloud builds submit --tag gcr.io/$GCP_PROJECT_ID/sql-studio-backend

# Deploy to Cloud Run
gcloud run deploy sql-studio-backend \
  --image gcr.io/$GCP_PROJECT_ID/sql-studio-backend \
  --region us-central1 \
  --platform managed \
  --allow-unauthenticated \
  --port 8500 \
  --memory 512Mi \
  --cpu 1 \
  --min-instances 0 \
  --max-instances 10 \
  --set-secrets=TURSO_URL=turso-url:latest,TURSO_AUTH_TOKEN=turso-auth-token:latest
```

#### Verify Deployment

```bash
# Get service URL
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format 'value(status.url)')

# Test health endpoint
curl $SERVICE_URL/health

# View logs
gcloud run services logs read sql-studio-backend --region us-central1
```

#### Update Configuration

To update secrets:

```bash
# Update a secret
echo -n "new-value" | gcloud secrets versions add secret-name --data-file=-

# Cloud Run will automatically use the latest version
```

---

### Fly.io

**Best for:** Low-cost deployments with global edge network

#### Initial Setup

1. **Install and authenticate:**

```bash
flyctl auth login
```

2. **Set environment variables:**

```bash
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-token"
export RESEND_API_KEY="re_your-key"
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
export JWT_SECRET=$(openssl rand -base64 32)
```

#### Deployment

**Method 1: Automated Script (Recommended)**

```bash
cd backend-go
./scripts/deploy-fly.sh
```

**Method 2: Manual flyctl commands**

```bash
cd backend-go

# Create app (first time only)
flyctl apps create sql-studio-backend

# Set secrets
flyctl secrets set \
  TURSO_URL="$TURSO_URL" \
  TURSO_AUTH_TOKEN="$TURSO_AUTH_TOKEN" \
  RESEND_API_KEY="$RESEND_API_KEY" \
  RESEND_FROM_EMAIL="$RESEND_FROM_EMAIL" \
  JWT_SECRET="$JWT_SECRET"

# Deploy
flyctl deploy --ha=false
```

#### Verify Deployment

```bash
# Get app URL
flyctl info

# Test health endpoint
curl https://sql-studio-backend.fly.dev/health

# View logs
flyctl logs

# SSH into machine
flyctl ssh console
```

#### Multi-Region Deployment

To deploy to multiple regions:

```bash
# Add regions
flyctl regions add iad  # Virginia (East Coast)
flyctl regions add lhr  # London (Europe)

# Scale to 2 machines minimum
flyctl scale count 2

# View regions
flyctl regions list
```

---

## CI/CD Setup

### GitHub Actions (Recommended)

The repository includes a complete GitHub Actions workflow for automated deployments.

#### Setup

1. **Add GitHub Secrets:**

   Go to repository Settings > Secrets and variables > Actions:

   ```
   GCP_PROJECT_ID=your-project-id
   GCP_SA_KEY=<service-account-key-json>
   FLY_API_TOKEN=<fly-token>  # Optional, for Fly.io
   TURSO_URL=libsql://your-db.turso.io
   TURSO_AUTH_TOKEN=your-token
   RESEND_API_KEY=re_your-key
   RESEND_FROM_EMAIL=noreply@yourdomain.com
   JWT_SECRET=your-secret-min-32-chars
   ```

2. **Get service account key:**

   ```bash
   # Create service account for GitHub Actions
   gcloud iam service-accounts create github-actions \
     --display-name="GitHub Actions"

   # Grant necessary roles
   gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
     --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/cloudbuild.builds.editor"

   gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
     --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/run.admin"

   # Create and download key
   gcloud iam service-accounts keys create github-actions-key.json \
     --iam-account=github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com

   # Copy contents to GCP_SA_KEY secret
   cat github-actions-key.json
   ```

3. **Trigger deployment:**

   ```bash
   # Push to main branch
   git push origin main

   # Or manually trigger from GitHub Actions UI
   ```

#### Workflow Behavior

- **On PR:** Build and test only (no deployment)
- **On push to main:** Build, test, and deploy to Cloud Run
- **Manual trigger:** Choose deployment target (Cloud Run, Fly.io, or both)

---

## Monitoring and Logging

### Google Cloud Run

**View Logs:**

```bash
# Real-time logs
gcloud run services logs tail sql-studio-backend --region us-central1

# Search logs
gcloud logging read "resource.type=cloud_run_revision" \
  --limit 50 \
  --format json
```

**Metrics:**
- Visit: https://console.cloud.google.com/run/detail/us-central1/sql-studio-backend
- Monitor: Request count, latency, error rate, CPU/memory usage

**Alerts:**

```bash
# Create alert for high error rate
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="High Error Rate" \
  --condition-threshold-value=0.05 \
  --condition-threshold-duration=60s
```

### Fly.io

**View Logs:**

```bash
# Real-time logs
flyctl logs --follow

# Search logs
flyctl logs --search "ERROR"
```

**Metrics:**
- Dashboard: https://fly.io/apps/sql-studio-backend/metrics
- Built-in Prometheus endpoint on port 9100

**Alerts:**
- Configure in Fly.io dashboard
- Use Grafana Cloud integration (free tier)

### Application Health

**Health Check Endpoint:**

```bash
curl https://your-app-url/health
```

**Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "database": "connected"
}
```

---

## Scaling

### Google Cloud Run

**Horizontal Scaling (Instances):**

```bash
# Set autoscaling
gcloud run services update sql-studio-backend \
  --min-instances 1 \
  --max-instances 100 \
  --region us-central1

# Scale to zero for cost savings
gcloud run services update sql-studio-backend \
  --min-instances 0 \
  --region us-central1
```

**Vertical Scaling (Resources):**

```bash
# Increase memory and CPU
gcloud run services update sql-studio-backend \
  --memory 1Gi \
  --cpu 2 \
  --region us-central1
```

**Concurrency:**

```bash
# Adjust requests per instance
gcloud run services update sql-studio-backend \
  --concurrency 100 \
  --region us-central1
```

### Fly.io

**Horizontal Scaling:**

```bash
# Scale to multiple machines
flyctl scale count 3

# Auto-scale configuration in fly.toml
min_machines_running = 1
max_machines_running = 10
```

**Vertical Scaling:**

```bash
# Change VM size
flyctl scale vm shared-cpu-2x --memory 1024

# Available sizes:
# shared-cpu-1x: 1 CPU, 256/512/1024 MB
# shared-cpu-2x: 2 CPU, 512/1024/2048 MB
# performance-*: Dedicated CPU options
```

---

## Rollback Procedures

### Google Cloud Run

**Method 1: Rollback to previous revision**

```bash
# List revisions
gcloud run revisions list \
  --service sql-studio-backend \
  --region us-central1

# Rollback to specific revision
gcloud run services update-traffic sql-studio-backend \
  --to-revisions REVISION_NAME=100 \
  --region us-central1
```

**Method 2: Deploy previous image**

```bash
# List images
gcloud container images list-tags gcr.io/$GCP_PROJECT_ID/sql-studio-backend

# Deploy specific image
gcloud run deploy sql-studio-backend \
  --image gcr.io/$GCP_PROJECT_ID/sql-studio-backend:COMMIT_SHA \
  --region us-central1
```

**Method 3: Gradual rollback**

```bash
# Split traffic 50/50
gcloud run services update-traffic sql-studio-backend \
  --to-revisions OLD_REVISION=50,NEW_REVISION=50 \
  --region us-central1

# Monitor and adjust
gcloud run services update-traffic sql-studio-backend \
  --to-revisions OLD_REVISION=100 \
  --region us-central1
```

### Fly.io

**Rollback to previous release:**

```bash
# List releases
flyctl releases

# Rollback to specific version
flyctl releases rollback <version>
```

**Emergency rollback:**

```bash
# Immediate rollback without confirmation
flyctl releases rollback --force
```

---

## Troubleshooting

### Common Issues

#### 1. Health Check Failing

**Symptoms:** Deployment fails, service shows unhealthy

**Solutions:**

```bash
# Check application logs
gcloud run services logs tail sql-studio-backend --region us-central1
# or
flyctl logs

# Common causes:
# - Wrong port configuration (ensure PORT env var matches health check)
# - Database connection issues (verify TURSO_URL and token)
# - Missing environment variables
# - Application crash on startup

# Test locally
docker build -t sql-studio-backend .
docker run -p 8500:8500 \
  -e TURSO_URL="$TURSO_URL" \
  -e TURSO_AUTH_TOKEN="$TURSO_AUTH_TOKEN" \
  sql-studio-backend
```

#### 2. Secrets Not Available

**Symptoms:** Application can't access secrets

**GCP Solution:**

```bash
# Verify secret exists
gcloud secrets describe jwt-secret

# Grant access to service account
gcloud secrets add-iam-policy-binding jwt-secret \
  --member="serviceAccount:sql-studio-backend@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

**Fly.io Solution:**

```bash
# List secrets
flyctl secrets list

# Set missing secret
flyctl secrets set JWT_SECRET="your-secret"
```

#### 3. Out of Memory

**Symptoms:** Container restarts frequently, OOM errors in logs

**Solutions:**

```bash
# GCP: Increase memory
gcloud run services update sql-studio-backend \
  --memory 1Gi \
  --region us-central1

# Fly.io: Increase memory
flyctl scale vm shared-cpu-1x --memory 1024
```

#### 4. Database Connection Errors

**Symptoms:** "failed to connect to database" in logs

**Solutions:**

```bash
# Verify Turso credentials
curl -X GET "https://api.turso.tech/v1/databases" \
  -H "Authorization: Bearer $TURSO_AUTH_TOKEN"

# Test connection locally
export TURSO_URL="libsql://..."
export TURSO_AUTH_TOKEN="..."
go run cmd/server/main.go
```

#### 5. Build Failures

**Symptoms:** Docker build fails, Cloud Build errors

**Solutions:**

```bash
# Test build locally
cd backend-go
docker build -t test .

# Common issues:
# - Missing dependencies in go.mod
# - CGO compilation errors (install gcc, sqlite-dev)
# - Insufficient build timeout (increase in cloudbuild.yaml)

# Increase Cloud Build timeout
gcloud builds submit --timeout=30m
```

### Debug Commands

```bash
# GCP Cloud Run
gcloud run services describe sql-studio-backend --region us-central1
gcloud run revisions list --service sql-studio-backend --region us-central1
gcloud logging read "resource.type=cloud_run_revision" --limit 100

# Fly.io
flyctl status
flyctl logs --follow
flyctl ssh console
flyctl doctor  # Check for common issues

# Local testing
docker run -it --rm sql-studio-backend sh
docker logs <container-id>
```

---

## Cost Optimization

### Google Cloud Run

**Free Tier (as of 2025):**
- 2 million requests/month
- 360,000 GB-seconds memory
- 180,000 vCPU-seconds

**Optimization Tips:**

1. **Scale to zero when idle:**
   ```bash
   gcloud run services update sql-studio-backend \
     --min-instances 0 \
     --region us-central1
   ```

2. **CPU throttling (only charge during requests):**
   ```bash
   gcloud run services update sql-studio-backend \
     --cpu-throttling \
     --region us-central1
   ```

3. **Right-size resources:**
   ```bash
   # Start small, scale up if needed
   gcloud run services update sql-studio-backend \
     --memory 256Mi \
     --cpu 1 \
     --region us-central1
   ```

4. **Monitor costs:**
   - View billing: https://console.cloud.google.com/billing
   - Set budget alerts
   - Use cost calculator: https://cloud.google.com/products/calculator

**Estimated Monthly Cost:**
- Low traffic (< 100k requests): **$0-5** (within free tier)
- Medium traffic (1M requests): **$5-15**
- High traffic (10M requests): **$50-100**

### Fly.io

**Free Tier:**
- 3 shared-cpu-1x VMs with 256MB RAM
- 160GB outbound transfer

**Optimization Tips:**

1. **Scale to zero:**
   ```toml
   # In fly.toml
   min_machines_running = 0
   auto_stop_machines = "stop"
   auto_start_machines = true
   ```

2. **Use shared CPUs:**
   ```bash
   flyctl scale vm shared-cpu-1x --memory 256
   ```

3. **Single region for low traffic:**
   ```bash
   # Don't deploy to multiple regions unless needed
   flyctl regions set sjc
   ```

4. **Monitor usage:**
   - Dashboard: https://fly.io/dashboard/personal/usage
   - Set spending limits

**Estimated Monthly Cost:**
- Scale-to-zero (low traffic): **$0-2**
- Single instance (medium traffic): **$5-10**
- Multi-region HA (3 instances): **$15-30**

### Cost Comparison

| Scenario | GCP Cloud Run | Fly.io |
|----------|---------------|---------|
| Hobby project (low traffic) | $0-5 | $0-2 |
| Small business (10k-100k req/day) | $5-15 | $5-10 |
| Medium business (1M req/day) | $50-100 | $30-50 |
| High availability (multi-region) | $100-200 | $50-100 |

**Recommendation:**
- **Hobby/Side project:** Fly.io (better free tier)
- **Business/Production:** GCP Cloud Run (better scaling, monitoring)
- **Global app:** GCP Cloud Run (better global infrastructure)

---

## Additional Resources

### Documentation

- **Google Cloud Run:** https://cloud.google.com/run/docs
- **Fly.io:** https://fly.io/docs
- **Turso:** https://docs.turso.tech
- **Resend:** https://resend.com/docs

### Support

- **Issues:** https://github.com/sql-studio/sql-studio/issues
- **Discussions:** https://github.com/sql-studio/sql-studio/discussions

### Security

- Always use HTTPS in production
- Rotate JWT_SECRET regularly
- Keep dependencies updated: `go get -u ./...`
- Enable GCP Cloud Armor or Fly.io DDoS protection for high-traffic apps
- Use VPC/private networking for sensitive deployments

---

## Quick Reference

### Environment Setup

```bash
# Required secrets
export GCP_PROJECT_ID="your-project"
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="eyJ..."
export RESEND_API_KEY="re_..."
export RESEND_FROM_EMAIL="noreply@domain.com"
export JWT_SECRET=$(openssl rand -base64 32)
```

### Deploy Commands

```bash
# GCP Cloud Run
cd backend-go && ./scripts/deploy-cloudrun.sh

# Fly.io
cd backend-go && ./scripts/deploy-fly.sh

# GitHub Actions (push to main)
git push origin main
```

### Monitoring

```bash
# GCP logs
gcloud run services logs tail sql-studio-backend --region us-central1

# Fly.io logs
flyctl logs --follow

# Health check
curl https://your-app/health
```

---

**Last Updated:** 2025-10-23
**Version:** 1.0.0
