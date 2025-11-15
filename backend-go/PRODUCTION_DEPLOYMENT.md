# Production Deployment Guide for Howlerops Backend

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Setup](#environment-setup)
3. [First-Time Deployment](#first-time-deployment)
4. [Updating an Existing Deployment](#updating-an-existing-deployment)
5. [Domain and SSL Configuration](#domain-and-ssl-configuration)
6. [Monitoring and Logging](#monitoring-and-logging)
7. [Cost Estimates](#cost-estimates)
8. [Troubleshooting](#troubleshooting)
9. [Emergency Procedures](#emergency-procedures)

## Prerequisites

### Required Accounts and Services

- **Google Cloud Platform (GCP) Account**
  - Billing enabled
  - Project created
  - APIs enabled: Cloud Run, Secret Manager, Container Registry, Cloud Build

- **Turso Database**
  - Account created at [turso.tech](https://turso.tech)
  - Database created and configured
  - Auth token generated

- **Resend Email Service** (optional but recommended)
  - Account at [resend.com](https://resend.com)
  - API key generated
  - Domain verified

### Local Development Tools

```bash
# Install gcloud CLI
# macOS
brew install google-cloud-sdk

# Linux/WSL
curl https://sdk.cloud.google.com | bash
exec -l $SHELL

# Windows
# Download installer from https://cloud.google.com/sdk/docs/install

# Verify installation
gcloud --version

# Install Docker (for local builds)
# Visit https://docs.docker.com/get-docker/

# Install other tools
brew install jq curl git
```

### Authentication and Access

```bash
# Authenticate with Google Cloud
gcloud auth login

# Set default project
gcloud config set project YOUR_PROJECT_ID

# Verify authentication
gcloud auth list
```

## Environment Setup

### 1. GCP Project Configuration

```bash
# Set your project ID
export GCP_PROJECT_ID="your-project-id"

# Enable required APIs
gcloud services enable \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com \
  cloudbuild.googleapis.com \
  --project=$GCP_PROJECT_ID

# Create service account for Cloud Run
gcloud iam service-accounts create sql-studio-backend \
  --display-name="Howlerops Backend Service" \
  --project=$GCP_PROJECT_ID

# Grant necessary permissions
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:sql-studio-backend@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### 2. Turso Database Setup

```bash
# Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# Login to Turso
turso auth login

# Create a database (if not already created)
turso db create sql-studio-production --region sjc

# Get your database URL
turso db show sql-studio-production --url
# Save this as TURSO_URL

# Generate auth token
turso db tokens create sql-studio-production
# Save this as TURSO_AUTH_TOKEN
```

### 3. Secret Generation

```bash
# Generate a strong JWT secret (64+ characters recommended)
export JWT_SECRET=$(openssl rand -base64 64)
echo "JWT_SECRET: $JWT_SECRET"

# Set your Resend API key
export RESEND_API_KEY="re_your_api_key_here"

# Set your sender email
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
```

## First-Time Deployment

### Step 1: Configure Secrets in GCP Secret Manager

```bash
# Navigate to backend directory
cd backend-go

# Run the setup-secrets script
./scripts/deploy-gcp.sh --project $GCP_PROJECT_ID --setup-secrets

# Or manually create secrets
echo -n "$TURSO_URL" | gcloud secrets create turso-url --data-file=- --project=$GCP_PROJECT_ID
echo -n "$TURSO_AUTH_TOKEN" | gcloud secrets create turso-auth-token --data-file=- --project=$GCP_PROJECT_ID
echo -n "$JWT_SECRET" | gcloud secrets create jwt-secret --data-file=- --project=$GCP_PROJECT_ID
echo -n "$RESEND_API_KEY" | gcloud secrets create resend-api-key --data-file=- --project=$GCP_PROJECT_ID
```

### Step 2: Build and Deploy

```bash
# Option 1: Deploy with Cloud Build (Recommended for CI/CD)
./scripts/deploy-gcp.sh \
  --project $GCP_PROJECT_ID \
  --region us-central1 \
  --use-cloudbuild

# Option 2: Deploy with local Docker build
./scripts/deploy-gcp.sh \
  --project $GCP_PROJECT_ID \
  --region us-central1

# Option 3: Full deployment with all checks
./scripts/deploy-full.sh
```

### Step 3: Verify Deployment

```bash
# Get service URL
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID \
  --format='value(status.url)')

# Run verification script
./scripts/verify-deployment.sh $SERVICE_URL

# Or manually test
curl $SERVICE_URL/health
```

## Updating an Existing Deployment

### GitHub Actions (Recommended)

```bash
# Create a release
git tag v1.0.1
git push origin v1.0.1

# Create GitHub release
gh release create v1.0.1 --generate-notes

# This will trigger automatic deployment via GitHub Actions
```

### Manual Update

```bash
# Pull latest changes
git pull origin main

# Deploy update
./scripts/deploy-gcp.sh \
  --project $GCP_PROJECT_ID \
  --use-cloudbuild

# Monitor rollout
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID
```

### Rolling Back

```bash
# List available revisions
gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID

# Route traffic to specific revision
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-revisions=sql-studio-backend-00003-xyz=100 \
  --project=$GCP_PROJECT_ID
```

## Domain and SSL Configuration

### 1. Map Custom Domain

```bash
# Add custom domain mapping
gcloud run domain-mappings create \
  --service=sql-studio-backend \
  --domain=api.yourdomain.com \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID

# Get DNS records to configure
gcloud run domain-mappings describe \
  --domain=api.yourdomain.com \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID
```

### 2. Configure DNS

Add the following records to your DNS provider:

```
Type: A
Name: api
Value: [IP provided by Cloud Run]

Type: AAAA
Name: api
Value: [IPv6 provided by Cloud Run]

Type: CNAME (alternative)
Name: api
Value: ghs.googlehosted.com
```

### 3. SSL Certificate

Cloud Run automatically provisions and manages SSL certificates for custom domains. No additional configuration required.

## Monitoring and Logging

### 1. Cloud Logging

```bash
# View recent logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=sql-studio-backend" \
  --limit=50 \
  --project=$GCP_PROJECT_ID \
  --format=json

# Stream logs in real-time
gcloud alpha run services logs tail sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID

# Search for errors
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
  --limit=20 \
  --project=$GCP_PROJECT_ID
```

### 2. Cloud Monitoring Setup

```bash
# Create alerting policy for high error rate
gcloud alpha monitoring policies create \
  --notification-channels=YOUR_CHANNEL_ID \
  --display-name="High Error Rate - Howlerops Backend" \
  --condition='{"displayName":"Error rate > 1%","conditionThreshold":{"filter":"resource.type=\"cloud_run_revision\" AND metric.type=\"run.googleapis.com/request_count\" AND metric.label.response_code_class=\"5xx\"","comparison":"COMPARISON_GT","thresholdValue":0.01,"duration":"60s"}}' \
  --project=$GCP_PROJECT_ID
```

### 3. Metrics Dashboard

Visit the [Cloud Console](https://console.cloud.google.com/run) to view:
- Request count and latency
- CPU and memory usage
- Error rates
- Cold start frequency
- Billing estimates

### 4. Custom Monitoring

```bash
# Check application metrics endpoint
curl $SERVICE_URL/metrics

# Set up Prometheus/Grafana (optional)
# See monitoring/prometheus-setup.md for details
```

## Cost Estimates

### Cloud Run Pricing (as of 2024)

| Resource | Free Tier | Price After Free Tier | Monthly Estimate |
|----------|-----------|----------------------|------------------|
| Requests | 2M/month | $0.40 per million | $0.40 (3M requests) |
| CPU | 180,000 vCPU-seconds | $0.00002400/vCPU-second | $2.88 (500K seconds) |
| Memory | 360,000 GiB-seconds | $0.00000250/GiB-second | $0.90 (1M GiB-seconds) |
| Networking | 1 GiB North America | $0.085/GiB | $0.85 (10 GiB) |

**Estimated monthly cost: $5-10** for low to moderate traffic

### Additional Services

| Service | Monthly Cost |
|---------|-------------|
| Turso Database (Starter) | $0-29 |
| Secret Manager | ~$0.06 per secret |
| Container Registry | ~$0.10 per GB |
| Cloud Build | 120 min free, then $0.003/min |

### Cost Optimization Tips

1. **Set minimum instances to 0** for development/staging
2. **Use regional endpoints** to minimize networking costs
3. **Enable CPU throttling** when idle
4. **Set appropriate concurrency limits** (80 recommended)
5. **Use Cloud Build caching** to reduce build times
6. **Monitor and set budget alerts**

```bash
# Create budget alert
gcloud billing budgets create \
  --billing-account=YOUR_BILLING_ACCOUNT \
  --display-name="Howlerops Backend Budget" \
  --budget-amount=50 \
  --threshold-rule=percent=0.5 \
  --threshold-rule=percent=0.9 \
  --threshold-rule=percent=1.0
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Service Won't Start

```bash
# Check logs for startup errors
gcloud logging read "resource.type=cloud_run_revision" \
  --limit=100 \
  --project=$GCP_PROJECT_ID

# Common causes:
# - Missing secrets
# - Database connection issues
# - Port binding problems
```

#### 2. Authentication Errors

```bash
# Verify secrets are accessible
gcloud secrets versions list turso-url --project=$GCP_PROJECT_ID
gcloud secrets versions list jwt-secret --project=$GCP_PROJECT_ID

# Check service account permissions
gcloud projects get-iam-policy $GCP_PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:sql-studio-backend@$GCP_PROJECT_ID.iam.gserviceaccount.com"
```

#### 3. High Latency

```bash
# Check cold start frequency
gcloud monitoring time-series list \
  --filter='metric.type="run.googleapis.com/request_latencies"' \
  --project=$GCP_PROJECT_ID

# Solutions:
# - Set minimum instances to 1
# - Optimize container startup time
# - Use regional endpoints
```

#### 4. Database Connection Issues

```bash
# Test Turso connection locally
export TURSO_URL="your-url"
export TURSO_AUTH_TOKEN="your-token"
go run cmd/server/main.go

# Check Turso status
turso db show sql-studio-production
```

### Debug Commands

```bash
# Get detailed service description
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID \
  --format=yaml

# List recent revisions
gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID

# Check resource limits
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format="value(spec.template.spec.containers[0].resources)" \
  --project=$GCP_PROJECT_ID
```

## Emergency Procedures

### 1. Immediate Rollback

```bash
# Quick rollback to previous revision
./scripts/rollback.sh

# Or manually
gcloud run services update-traffic sql-studio-backend \
  --to-revisions=LATEST=0 \
  --to-revisions=sql-studio-backend-00002-abc=100 \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID
```

### 2. Emergency Shutdown

```bash
# Temporarily disable the service
gcloud run services update sql-studio-backend \
  --min-instances=0 \
  --max-instances=0 \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID

# Or delete the service entirely
gcloud run services delete sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID
```

### 3. Database Recovery

```bash
# Create database backup
turso db backup create sql-studio-production

# List backups
turso db backup list sql-studio-production

# Restore from backup
turso db backup restore sql-studio-production --backup-id=BACKUP_ID
```

### 4. Secret Rotation

```bash
# Generate new JWT secret
NEW_JWT_SECRET=$(openssl rand -base64 64)

# Update secret in GCP
echo -n "$NEW_JWT_SECRET" | gcloud secrets versions add jwt-secret \
  --data-file=- \
  --project=$GCP_PROJECT_ID

# Redeploy service to pick up new secret
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --project=$GCP_PROJECT_ID
```

### Emergency Contacts

Configure alerts to notify:
- On-call engineer: Create PagerDuty integration
- Team Slack channel: Use Cloud Monitoring webhooks
- Email notifications: Configure in Cloud Monitoring

```bash
# Create notification channel
gcloud alpha monitoring channels create \
  --display-name="Howlerops Alerts" \
  --type=email \
  --channel-labels=email_address=alerts@yourdomain.com \
  --project=$GCP_PROJECT_ID
```

## Security Best Practices

1. **Never commit secrets to Git**
2. **Use Secret Manager for all sensitive data**
3. **Enable audit logging**
4. **Regularly rotate secrets**
5. **Use least-privilege IAM roles**
6. **Enable VPC Service Controls** (for enterprise)
7. **Configure Web Application Firewall** (Cloud Armor)
8. **Enable Binary Authorization** (for container security)

## Next Steps

After successful deployment:

1. Configure frontend to use production API URL
2. Set up monitoring dashboards
3. Configure backup automation
4. Document runbooks for common operations
5. Schedule regular security audits
6. Plan for disaster recovery

## Support and Resources

- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Turso Documentation](https://docs.turso.tech)
- [Resend Documentation](https://resend.com/docs)
- [GCP Pricing Calculator](https://cloud.google.com/products/calculator)
- [Howlerops Repository](https://github.com/yourusername/sql-studio)

For urgent production issues, check:
1. Cloud Run logs
2. Turso database status
3. GCP Status page
4. GitHub Actions runs