# Howlerops Backend - Complete Deployment Guide

**Platform:** Google Cloud Run
**Cost:** ~$0-3/month (with free tier)
**Deployment Time:** ~5 minutes
**Difficulty:** Beginner-friendly

---

## 📋 Table of Contents

1. [Quick Start](#quick-start) (Deploy in 5 minutes)
2. [Prerequisites](#prerequisites) (One-time setup)
3. [Local Deployment](#local-deployment)
4. [GitHub Actions Deployment](#github-actions-deployment)
5. [Environment Management](#environment-management)
6. [Monitoring & Troubleshooting](#monitoring--troubleshooting)
7. [Cost Management](#cost-management)

---

## 🚀 Quick Start

**Already have everything set up?** Deploy in one command:

```bash
# Navigate to backend directory
cd backend-go

# Set your environment variables (one-time)
export GCP_PROJECT_ID="your-project-id"
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-token"
export RESEND_API_KEY="re_your_key"
export JWT_SECRET="$(openssl rand -base64 32)"

# Deploy to production
make deploy-full
```

That's it! Skip to [Post-Deployment](#post-deployment) to verify your deployment.

---

## 📦 Prerequisites

### What You Need to Provide

#### 1. **Google Cloud Platform Account** (Free tier available)

```bash
# Create a new project
gcloud projects create howlerops-prod --name="HowlerOps"

# Set as default
gcloud config set project howlerops-prod
export GCP_PROJECT_ID="howlerops-prod"

# Enable billing (required for Cloud Run)
# Visit: https://console.cloud.google.com/billing
```

**Cost:** $0/month for first 2M requests + 180K vCPU-seconds (covers most small apps)

#### 2. **Turso Database** (Free tier: 500MB, 1B row reads/month)

```bash
# Install Turso CLI
brew install tursodatabase/tap/turso

# Login
turso auth login

# Create database
turso db create howlerops-prod

# Get credentials (save these!)
turso db show howlerops-prod --url
turso db tokens create howlerops-prod
```

#### 3. **Resend Email Service** (Free tier: 100 emails/day)

1. Sign up at https://resend.com
2. Verify your domain (add DNS records)
3. Create API key at https://resend.com/api-keys

#### 4. **JWT Secret** (Generate locally)

```bash
# Generate a strong secret
openssl rand -base64 32
```

---

## 💻 Local Deployment

### Step 1: Configure Environment

```bash
# Copy example configuration
cp .env.production.example .env.production

# Edit with your values
nano .env.production
```

Required values in `.env.production`:
```bash
GCP_PROJECT_ID=howlerops-prod
TURSO_URL=libsql://howlerops-prod-yourorg.turso.io
TURSO_AUTH_TOKEN=eyJhbG...your-token-here
RESEND_API_KEY=re_abc123...your-key-here
JWT_SECRET=your-generated-secret-here
```

### Step 2: Load Environment

```bash
# Load environment variables
source .env.production
```

### Step 3: Run Pre-Deployment Checks

```bash
# Verify everything is configured correctly
make prod-check
```

This checks:
- ✅ All environment variables are set
- ✅ JWT secret is strong enough
- ✅ Code passes tests
- ✅ gcloud CLI is installed and authenticated
- ✅ GCP project is set correctly

### Step 4: Setup GCP Secrets (One-time)

```bash
# Store secrets in GCP Secret Manager
make setup-gcp-secrets
```

This creates secrets that Cloud Run will access securely.

### Step 5: Deploy

```bash
# Deploy to production
make deploy-prod

# Or deploy to staging
make deploy-staging

# Or deploy a preview environment
make deploy-preview BRANCH=feature-auth
```

### Step 6: Verify Deployment

```bash
# Get the service URL from the deployment output, then:
curl https://your-service-url.run.app/health

# Should return: {"status":"healthy"}
```

---

## 🤖 GitHub Actions Deployment

### Setup (One-time)

#### 1. Create GCP Service Account

```bash
# Create service account for GitHub Actions
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions CI/CD"

# Grant necessary permissions
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.admin"

# Create and download key
gcloud iam service-accounts keys create github-actions-key.json \
  --iam-account=github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com
```

#### 2. Configure GitHub Secrets

Go to: **GitHub Repository → Settings → Secrets and variables → Actions**

Add these secrets:

| Secret Name | Value | How to Get It |
|------------|-------|---------------|
| `GCP_PROJECT_ID` | `howlerops-prod` | Your GCP project ID |
| `GCP_SA_KEY` | `{"type":"service_account",...}` | Contents of `github-actions-key.json` |
| `TURSO_URL` | `libsql://...` | `turso db show howlerops-prod --url` |
| `TURSO_AUTH_TOKEN` | `eyJ...` | `turso db tokens create howlerops-prod` |
| `JWT_SECRET` | `abc123...` | `openssl rand -base64 32` |
| `RESEND_API_KEY` | `re_...` | From https://resend.com/api-keys |

#### 3. Clean up local key (important!)

```bash
# Delete the local key file (it's now in GitHub Secrets)
rm github-actions-key.json

# NEVER commit this file to git!
```

### Deployment Triggers

The GitHub Actions workflow automatically deploys based on these triggers:

#### Production Deployment (Git Tags)
```bash
# Create and push a version tag
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions will:
# 1. Run all tests
# 2. Build Docker image
# 3. Scan for vulnerabilities
# 4. Deploy to production
# 5. Run smoke tests
```

#### Staging Deployment (Push to main)
```bash
# Push to main branch
git push origin main

# GitHub Actions will deploy to staging environment
```

#### Preview Deployment (Pull Requests)
```bash
# Create a pull request
gh pr create

# GitHub Actions will:
# 1. Create a preview environment
# 2. Add a comment with the preview URL
# 3. Delete the environment when PR is closed
```

---

## 🌍 Environment Management

### Three Environments

| Environment | Trigger | Service Name | Resources | Cost |
|------------|---------|--------------|-----------|------|
| **Production** | Git tag (`v1.0.0`) | `sql-studio-backend` | 512Mi, 1-10 instances | ~$2-5/mo |
| **Staging** | Push to `main` | `sql-studio-backend-staging` | 512Mi, 0-5 instances | ~$0-2/mo |
| **Preview** | Pull request | `sql-studio-preview-{branch}` | 256Mi, 0-2 instances | ~$0-1/mo |

### Accessing Environments

```bash
# List all services
gcloud run services list --region=us-central1

# Get production URL
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format='value(status.url)'

# Get staging URL
gcloud run services describe sql-studio-backend-staging \
  --region=us-central1 \
  --format='value(status.url)'
```

### Cleanup Preview Environments

Preview environments are automatically deleted when pull requests are closed. To manually delete:

```bash
# List preview environments
gcloud run services list --region=us-central1 | grep preview

# Delete a specific preview
gcloud run services delete sql-studio-preview-feature-name \
  --region=us-central1 \
  --quiet
```

---

## 📊 Monitoring & Troubleshooting

### View Logs

```bash
# Tail production logs
gcloud run services logs tail sql-studio-backend --region=us-central1

# View recent errors
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
  --limit=50 \
  --format=json
```

### Check Service Status

```bash
# Get service details
gcloud run services describe sql-studio-backend --region=us-central1

# Check recent revisions
gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1 \
  --limit=5
```

### Common Issues

#### ❌ "Service unavailable" errors

**Cause:** Service still starting up or crashed

**Fix:**
```bash
# Check logs for errors
gcloud run services logs tail sql-studio-backend

# Check if secrets are accessible
gcloud secrets list
```

#### ❌ "Permission denied" errors

**Cause:** Service account doesn't have access to secrets

**Fix:**
```bash
# Re-run secret setup
make setup-gcp-secrets
```

#### ❌ "Database connection failed"

**Cause:** Invalid Turso credentials

**Fix:**
```bash
# Verify Turso URL and token
turso db show howlerops-prod

# Update secrets
make setup-gcp-secrets
```

### Rollback

```bash
# List revisions
gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1

# Rollback to previous revision
gcloud run services update-traffic sql-studio-backend \
  --to-revisions=sql-studio-backend-00001-abc=100 \
  --region=us-central1
```

---

## 💰 Cost Management

### Free Tier Limits

**Cloud Run Free Tier (per month):**
- 2M requests
- 180,000 vCPU-seconds
- 360,000 GiB-seconds
- 1 GB network egress (North America)

**Turso Free Tier:**
- 500 MB storage
- 1 billion row reads
- 25 million row writes

**Resend Free Tier:**
- 100 emails/day
- 3,000 emails/month

### Staying Under $3/month

```yaml
# Production configuration (in deploy script)
Min Instances: 1          # $1-2/month for always-on
Max Instances: 10         # Prevents runaway costs
Memory: 512Mi             # Don't overprovision
CPU Allocation: Throttled # Only use CPU during requests
```

### Monitor Costs

```bash
# View current month's costs
gcloud billing accounts list

# Set up budget alert
gcloud billing budgets create \
  --billing-account=YOUR_BILLING_ACCOUNT_ID \
  --display-name="Howlerops Budget" \
  --budget-amount=5USD \
  --threshold-rule=percent=80 \
  --threshold-rule=percent=100
```

---

## 🎯 Makefilecommands Reference

```bash
# Development
make dev                    # Run locally
make test                   # Run tests
make prod-check             # Pre-deployment validation

# Secrets
make setup-gcp-secrets      # Store secrets in GCP

# Deployment
make deploy-full            # Full production deploy (check + secrets + deploy)
make deploy-prod            # Deploy to production
make deploy-staging         # Deploy to staging
make deploy-preview BRANCH=feat  # Deploy preview environment

# Monitoring
make prod-logs              # Tail production logs
make prod-status            # Check service status

# Help
make prod-help              # Show production commands
```

---

## 📚 Additional Resources

- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Turso Documentation](https://docs.turso.tech)
- [Resend Documentation](https://resend.com/docs)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

---

## ✅ Post-Deployment Checklist

After deploying:

- [ ] Service is accessible at the Cloud Run URL
- [ ] Health check passes: `curl https://your-url.run.app/health`
- [ ] Logs show no errors: `make prod-logs`
- [ ] Update frontend with backend URL
- [ ] Setup custom domain (optional)
- [ ] Configure monitoring alerts
- [ ] Test critical endpoints
- [ ] Document service URL for team

---

## 🔒 Security Notes

- **Never** commit `.env.production` or service account keys
- **Always** use GCP Secret Manager for production secrets
- **Rotate** JWT secret if compromised
- **Enable** Cloud Armor for DDoS protection (when needed)
- **Review** IAM permissions regularly
- **Update** dependencies regularly

---

**Last Updated:** $(date +%Y-%m-%d)
**Deployment Platform:** Google Cloud Run
**Target Cost:** $0-3/month
