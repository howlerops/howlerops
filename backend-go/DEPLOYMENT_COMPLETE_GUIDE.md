# SQL Studio Backend - Complete Deployment Guide

**A beginner-friendly, step-by-step guide to deploy SQL Studio backend to production**

This guide will walk you through deploying SQL Studio backend from scratch, even if you have zero experience with GCP or Turso. By following this guide, you'll have a production-ready API running in 30-60 minutes.

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Step 1: GCP Account Setup](#step-1-gcp-account-setup)
- [Step 2: Turso Database Setup](#step-2-turso-database-setup)
- [Step 3: Environment Preparation](#step-3-environment-preparation)
- [Step 4: GitHub Repository Setup](#step-4-github-repository-setup)
- [Step 5: First Deployment](#step-5-first-deployment)
- [Step 6: Post-Deployment Setup](#step-6-post-deployment-setup)
- [Step 7: Updating the Deployment](#step-7-updating-the-deployment)
- [Cost Breakdown](#cost-breakdown)
- [Troubleshooting](#troubleshooting)

---

## Overview

**What you're deploying:**
- Go backend API for SQL Studio
- Runs on Google Cloud Run (serverless, auto-scaling)
- Uses Turso for database (global edge database)
- Sends emails via Resend (optional but recommended)
- Completely automated deployment via GitHub Actions

**What you'll need:**
- 30-60 minutes of your time
- A credit card (for GCP billing, but you'll likely stay in free tier)
- Basic command line knowledge
- A GitHub account

**Expected costs:**
- GCP Cloud Run: $0-5/month for hobby projects (free tier covers most usage)
- Turso Database: $0/month (free tier: 500 databases, 9GB total storage, 1B row reads)
- Resend Email: $0/month (free tier: 100 emails/day, 3,000/month)
- **Total: $0-5/month for small projects**

---

## Prerequisites

### Required Accounts

Before you begin, create these accounts (all free to start):

#### 1. Google Cloud Platform (GCP)

**Sign up:** https://console.cloud.google.com

- Click "Get started for free" or "Console" (top right)
- Sign in with a Google account (or create one)
- You'll get $300 in free credits for 90 days
- A credit card is required (for identity verification, won't be charged unless you explicitly upgrade)

#### 2. Turso Database

**Sign up:** https://turso.tech

- Click "Get Started" or "Sign Up"
- Sign in with GitHub (recommended) or email
- Completely free tier: 500 databases, 9GB storage, 1B row reads/month
- No credit card required

#### 3. Resend Email Service (Optional but Recommended)

**Sign up:** https://resend.com

- Click "Get Started" or "Sign Up"
- Sign in with GitHub or email
- Free tier: 3,000 emails/month, 100/day
- No credit card required

#### 4. GitHub Account

**Sign up:** https://github.com/join

- You'll use this to fork the SQL Studio repository
- GitHub Actions (free for public repos, 2,000 minutes/month for private)

### Required Tools

Install these on your local computer:

#### 1. Git

**Check if installed:**
```bash
git --version
```

**If not installed:**
- **macOS:** Install Xcode Command Line Tools: `xcode-select --install`
- **Windows:** https://git-scm.com/download/win
- **Linux:** `sudo apt-get install git` (Ubuntu/Debian) or `sudo yum install git` (Fedora/CentOS)

#### 2. Google Cloud SDK (gcloud CLI)

**Install:**

**macOS:**
```bash
# Using Homebrew (recommended)
brew install --cask google-cloud-sdk

# Or download installer
curl https://sdk.cloud.google.com | bash
```

**Linux:**
```bash
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
```

**Windows:**
Download and run the installer: https://cloud.google.com/sdk/docs/install

**Verify installation:**
```bash
gcloud --version
```

Expected output:
```
Google Cloud SDK 456.0.0
bq 2.0.101
core 2024.01.15
gcloud-crc32c 1.0.0
```

#### 3. OpenSSL (for generating secrets)

**Check if installed:**
```bash
openssl version
```

**If not installed:**
- **macOS:** Pre-installed
- **Windows:** Comes with Git for Windows
- **Linux:** `sudo apt-get install openssl`

---

## Step 1: GCP Account Setup

### 1.1: Create a New GCP Project

**Why?** A project organizes all your GCP resources (like Cloud Run services, secrets, etc.)

**Steps:**

1. Go to https://console.cloud.google.com
2. Click the project dropdown at the top (next to "Google Cloud")
3. Click "NEW PROJECT"
4. Fill in:
   - **Project name:** `sql-studio-prod` (or any name you like)
   - **Organization:** Leave as "No organization" (unless you have one)
   - **Location:** Leave as-is
5. Click "CREATE"
6. Wait 10-20 seconds for project creation
7. Select your new project from the dropdown

**Save your Project ID:**
```bash
# Your project ID will be something like: sql-studio-prod-123456
# Note it down - you'll need it later!
```

### 1.2: Enable Billing

**Why?** GCP requires billing enabled (but won't charge you unless you manually upgrade from free tier)

**Steps:**

1. In GCP Console, open the menu (three horizontal lines, top left)
2. Click "Billing"
3. Click "LINK A BILLING ACCOUNT"
4. Follow the prompts to add a credit card
5. Accept the free trial ($300 credits for 90 days)

**Tip:** Set up a billing alert to notify you if costs exceed $10/month:
1. In Billing, click "Budgets & alerts"
2. Click "CREATE BUDGET"
3. Set budget to $10, alert threshold at 50% and 100%

### 1.3: Enable Required APIs

**Why?** GCP APIs are disabled by default to save resources

**Open Cloud Shell (easiest method):**

1. Click the Cloud Shell icon (terminal symbol) in the top-right corner of GCP Console
2. Wait for Cloud Shell to open (a terminal at the bottom of the screen)
3. Run these commands:

```bash
# Set your project (replace with your actual project ID)
gcloud config set project sql-studio-prod-123456

# Enable required APIs
gcloud services enable \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com
```

**Expected output:**
```
Operation "operations/..." finished successfully.
```

**Wait 1-2 minutes** for APIs to be fully enabled.

### 1.4: Create Service Account for GitHub Actions

**Why?** GitHub Actions needs permission to deploy to your GCP project

**In Cloud Shell, run:**

```bash
# Create service account
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions Deployment"

# Get your project ID
export PROJECT_ID=$(gcloud config get-value project)
echo "Project ID: $PROJECT_ID"

# Grant necessary permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.admin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.admin"
```

### 1.5: Generate and Download Service Account Key

**In Cloud Shell:**

```bash
# Create key file
gcloud iam service-accounts keys create ~/github-actions-key.json \
  --iam-account=github-actions@$PROJECT_ID.iam.gserviceaccount.com

# Display the key (you'll copy this to GitHub Secrets)
cat ~/github-actions-key.json
```

**Copy the entire JSON output** (starts with `{` and ends with `}`) - you'll need this for GitHub Secrets.

**Warning:** This key grants access to your GCP project. Never commit it to Git or share it publicly!

---

## Step 2: Turso Database Setup

### 2.1: Sign Up for Turso

1. Go to https://turso.tech
2. Click "Get Started" or "Sign Up"
3. Choose "Continue with GitHub" (recommended)
4. Authorize Turso to access your GitHub account

### 2.2: Install Turso CLI

**Why?** The CLI makes it easy to create databases and get credentials

**macOS/Linux:**
```bash
curl -sSfL https://get.tur.so/install.sh | bash
```

**Windows (PowerShell):**
```powershell
iwr -useb https://get.tur.so/install.ps1 | iex
```

**Verify installation:**
```bash
turso --version
```

### 2.3: Login to Turso

```bash
turso auth login
```

This will open a browser window - click "Authorize" to grant access.

### 2.4: Create Production Database

```bash
# Create database named "sql-studio-prod"
turso db create sql-studio-prod

# Wait for creation (5-10 seconds)
```

**Expected output:**
```
Created database sql-studio-prod in [region]
URL: libsql://sql-studio-prod-[your-org].turso.io
```

**Tip:** Choose your primary region based on where most users are:
- Run `turso db create sql-studio-prod --location iad` for US East (Virginia)
- Run `turso db create sql-studio-prod --location sjc` for US West (California)
- Run `turso db create sql-studio-prod --location lhr` for Europe (London)
- Run `turso db create sql-studio-prod --location nrt` for Asia (Tokyo)

Full list: https://turso.tech/docs/reference/locations

### 2.5: Get Database Credentials

**Get database URL:**
```bash
turso db show sql-studio-prod --url
```

**Example output:**
```
libsql://sql-studio-prod-yourorg.turso.io
```

**Copy this URL** - you'll need it for GitHub Secrets.

**Generate authentication token:**
```bash
turso db tokens create sql-studio-prod
```

**Example output:**
```
eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJh...
```

**Copy this token** - you'll need it for GitHub Secrets.

**Warning:** This token grants full access to your database. Never commit it to Git!

### 2.6: Test Database Connection

```bash
# Open database shell
turso db shell sql-studio-prod
```

**In the shell, run:**
```sql
-- Create a test table
CREATE TABLE test (id INTEGER PRIMARY KEY, message TEXT);

-- Insert test data
INSERT INTO test (message) VALUES ('Hello from Turso!');

-- Query test data
SELECT * FROM test;

-- Exit shell
.quit
```

**Expected output:**
```
id | message
---|------------------
1  | Hello from Turso!
```

**Success!** Your Turso database is ready.

---

## Step 3: Environment Preparation

### 3.1: Generate JWT Secret

**Why?** JWT secret is used to sign authentication tokens. Must be strong and unique.

**Run this command:**
```bash
openssl rand -base64 64
```

**Example output:**
```
6K8vN2xQpL9mR4tY1wE3uI5oP7aS9dF0gH2jK4lZ6xC8vB1nM3qW5eR7tY9uI0oP1aS2dF3gH4jK5lZ6xC7vB8n
```

**Copy this output** - you'll use it as JWT_SECRET in GitHub Secrets.

**Tip:** Store this in a password manager - you'll need it if you ever want to update secrets.

### 3.2: Set Up Resend Account (Optional but Recommended)

**Why?** Email verification and password reset features require email sending.

**Skip this if:** You don't need email features (you can add it later)

**Steps:**

1. Go to https://resend.com
2. Sign up (with GitHub or email)
3. Verify your email address
4. Click "API Keys" in the left sidebar
5. Click "Create API Key"
6. Name: `sql-studio-production`
7. Permission: "Full access" (or "Sending access" only)
8. Click "Create"
9. **Copy the API key** (starts with `re_`) - you can only see it once!

**Set up sender domain (optional but recommended):**

1. Click "Domains" in left sidebar
2. Click "Add Domain"
3. Enter your domain (e.g., `yourdomain.com`)
4. Add the DNS records shown to your domain provider
5. Wait for verification (5-60 minutes)
6. Use `noreply@yourdomain.com` as your from address

**Or use Resend's test domain:**
- From address: `onboarding@resend.dev`
- Works for testing, but emails might go to spam

### 3.3: Deployment Values Checklist

**Copy this checklist and fill in your values:**

```bash
# Save this securely - you'll need it for GitHub Secrets!

# GCP Configuration
GCP_PROJECT_ID=sql-studio-prod-123456
GCP_SA_KEY={ "type": "service_account", ... }  # (the full JSON from Step 1.5)

# Turso Database
TURSO_URL=libsql://sql-studio-prod-yourorg.turso.io
TURSO_AUTH_TOKEN=eyJhbGciOiJFZERTQSIs...

# Authentication
JWT_SECRET=6K8vN2xQpL9mR4tY1wE3uI5oP7aS9dF0gH2jK4lZ6xC8vB1nM3qW5eR7tY9uI0oP1aS2dF3gH4jK5lZ6xC7vB8n

# Email (Optional - set to empty if not using)
RESEND_API_KEY=re_123456789abcdef...
RESEND_FROM_EMAIL=noreply@yourdomain.com
```

**Tip:** Save this in a password manager (1Password, Bitwarden, etc.) for safekeeping!

---

## Step 4: GitHub Repository Setup

### 4.1: Fork/Clone the Repository

**Option A: Fork (if you want to contribute back)**

1. Go to https://github.com/sql-studio/sql-studio
2. Click "Fork" (top right)
3. Select your account
4. Wait for fork to complete
5. Clone your fork:

```bash
git clone https://github.com/YOUR-USERNAME/sql-studio.git
cd sql-studio
```

**Option B: Clone directly**

```bash
git clone https://github.com/sql-studio/sql-studio.git
cd sql-studio
```

### 4.2: Configure GitHub Secrets

**Why?** GitHub Secrets store sensitive values securely (encrypted)

**Steps:**

1. Go to your repository on GitHub
2. Click "Settings" (top menu)
3. In left sidebar, click "Secrets and variables" → "Actions"
4. Click "New repository secret"

**Add each secret one by one:**

| Secret Name | Value | Notes |
|-------------|-------|-------|
| `GCP_PROJECT_ID` | `sql-studio-prod-123456` | Your GCP project ID |
| `GCP_SA_KEY` | `{ "type": "service_account", ... }` | Full JSON from Step 1.5 |
| `TURSO_URL` | `libsql://sql-studio-prod-yourorg.turso.io` | From Step 2.5 |
| `TURSO_AUTH_TOKEN` | `eyJhbGciOiJFZERTQSIs...` | From Step 2.5 |
| `JWT_SECRET` | `6K8vN2xQpL9mR4tY...` | From Step 3.1 |
| `RESEND_API_KEY` | `re_123456789...` | From Step 3.2 (optional) |
| `RESEND_FROM_EMAIL` | `noreply@yourdomain.com` | From Step 3.2 (optional) |

**For each secret:**
1. Click "New repository secret"
2. Enter "Name" (exactly as shown above, case-sensitive!)
3. Paste "Value"
4. Click "Add secret"

**Tip:** Double-check each secret name for typos!

### 4.3: Verify GitHub Actions is Enabled

**Steps:**

1. In your repository, click "Actions" (top menu)
2. If you see "Workflows", you're good!
3. If you see "Actions are disabled", click "I understand my workflows, go ahead and enable them"

---

## Step 5: First Deployment

### 5.1: Trigger Deployment via Git Tag

**Why?** The GitHub Actions workflow is triggered by release tags (best practice)

**In your local repository:**

```bash
# Make sure you're on main branch
git checkout main

# Pull latest changes
git pull origin main

# Create and push a release tag
git tag v1.0.0
git push origin v1.0.0
```

**Alternative: Manual Trigger**

1. Go to your repository on GitHub
2. Click "Actions" tab
3. Click "Deploy Backend" workflow (left sidebar)
4. Click "Run workflow" button (right side)
5. Select branch: `main`
6. Click green "Run workflow" button

### 5.2: Monitor Deployment

**Watch GitHub Actions:**

1. Click "Actions" tab
2. Click on the running workflow (top of list, yellow dot)
3. Watch each job complete:
   - Pre-deployment Validation
   - Build and Push Docker Image
   - Manage GCP Secrets
   - Deploy to Cloud Run
   - Deployment Summary

**Expected time:** 5-10 minutes

**What happens during deployment:**

1. **Validation (30 seconds)**
   - Checks all secrets are configured
   - Validates Dockerfile
   - Runs quick tests

2. **Docker Build (2-4 minutes)**
   - Builds production Docker image
   - Pushes to GCP Container Registry
   - Scans for security vulnerabilities

3. **Secret Management (30 seconds)**
   - Creates/updates secrets in GCP Secret Manager
   - Grants Cloud Run service account access

4. **Cloud Run Deployment (1-3 minutes)**
   - Deploys new revision (no traffic yet)
   - Runs smoke tests
   - Routes 100% traffic if tests pass
   - Verifies deployment

5. **Summary (5 seconds)**
   - Shows deployment URLs
   - Records deployment in logs

### 5.3: Get Your Service URL

**Option A: From GitHub Actions**

1. In the workflow run, scroll to "Deployment Summary" job
2. Copy the URL shown (e.g., `https://sql-studio-backend-xyz-uc.a.run.app`)

**Option B: From GCP Console**

1. Go to https://console.cloud.google.com/run
2. Click on "sql-studio-backend" service
3. Copy the URL at the top

**Option C: Using gcloud CLI**

```bash
gcloud run services describe sql-studio-backend \
  --region us-central1 \
  --format 'value(status.url)'
```

### 5.4: Verify Deployment

**Test health endpoint:**

```bash
# Replace URL with your actual service URL
curl https://sql-studio-backend-xyz-uc.a.run.app/health
```

**Expected response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "30s",
  "database": "connected"
}
```

**Test API version endpoint:**

```bash
curl https://sql-studio-backend-xyz-uc.a.run.app/api/v1/version
```

**Expected response:**
```json
{
  "version": "1.0.0",
  "build_time": "2025-10-23T12:00:00Z",
  "git_commit": "abc1234"
}
```

**Success!** Your backend is now deployed and running in production!

---

## Step 6: Post-Deployment Setup

### 6.1: Configure Custom Domain (Optional)

**Why?** Use your own domain instead of `*.run.app`

**Prerequisites:**
- Own a domain (from Namecheap, GoDaddy, Cloudflare, etc.)
- Access to DNS settings

**Steps:**

1. In GCP Console, go to Cloud Run
2. Click "sql-studio-backend" service
3. Click "MANAGE CUSTOM DOMAINS" (top)
4. Click "ADD MAPPING"
5. Select "sql-studio-backend" service
6. Enter domain: `api.yourdomain.com` (or subdomain of choice)
7. Click "CONTINUE"
8. Copy DNS records shown
9. Add DNS records to your domain provider:
   - Type: `CNAME`
   - Name: `api` (or your subdomain)
   - Value: `ghs.googlehosted.com`
10. Wait for DNS propagation (5-60 minutes)
11. GCP will automatically provision SSL certificate

**Verify:**
```bash
curl https://api.yourdomain.com/health
```

### 6.2: Set Up Monitoring Alerts

**Why?** Get notified if your service goes down or has errors

**Create Uptime Check:**

1. Go to https://console.cloud.google.com/monitoring
2. Click "Uptime checks" (left sidebar)
3. Click "+ CREATE UPTIME CHECK"
4. Fill in:
   - **Title:** `SQL Studio Backend Health`
   - **Protocol:** `HTTPS`
   - **Resource Type:** `URL`
   - **Hostname:** Your service URL (without https://)
   - **Path:** `/health`
   - **Check frequency:** `5 minutes`
5. Click "CONTINUE"
6. Click "CONTINUE" (accept defaults)
7. Click "CONTINUE" (skip alert policy for now)
8. Click "CREATE"

**Create Alert Policy:**

1. In Monitoring, click "Alerting" (left sidebar)
2. Click "+ CREATE POLICY"
3. Click "ADD CONDITION"
4. Select "Uptime Health Check"
5. Configure:
   - **Target:** `SQL Studio Backend Health`
   - **Condition:** `Uptime check failed`
6. Click "ADD"
7. Click "NEXT"
8. Click "ADD NOTIFICATION CHANNEL"
9. Select "Email" and add your email
10. Click "ADD"
11. Name: `Backend Down Alert`
12. Click "CREATE POLICY"

### 6.3: Review Security Settings

**Check service is properly secured:**

1. Go to Cloud Run service page
2. Click "SECURITY" tab
3. Verify:
   - Authentication: "Allow unauthenticated invocations" (for public API)
   - HTTPS: "Redirect HTTP to HTTPS" (enabled by default)
   - Service account: Has minimal permissions

**Enable Cloud Armor (optional, for DDoS protection):**

Only needed for high-traffic production sites:
1. Go to https://console.cloud.google.com/security/armor
2. Follow setup wizard
3. Create policy
4. Attach to Cloud Run service

**Tip:** Cloud Armor adds significant cost ($0.75/policy/month + per-request fees). Skip unless you have high traffic or DDoS concerns.

### 6.4: Set Up Database Backups

**Turso automatic backups:**

Turso automatically backs up your database every day (free tier: 24 hours retention, paid plans: 30 days).

**View backups:**
```bash
turso db show sql-studio-prod
```

**Restore from backup (if needed):**
```bash
# List available backups
turso db restore sql-studio-prod --list

# Restore from specific backup
turso db restore sql-studio-prod --from <backup-id>
```

**Manual export (recommended for critical data):**

```bash
# Export to local SQLite file
turso db shell sql-studio-prod ".dump" > backup-$(date +%Y%m%d).sql

# Store in cloud storage (GCS, S3, etc.)
```

---

## Step 7: Updating the Deployment

### 7.1: Deploy New Version

**When you make code changes:**

```bash
# Commit your changes
git add .
git commit -m "feat: add new feature"
git push origin main

# Create new version tag
git tag v1.1.0
git push origin v1.1.0
```

**GitHub Actions will automatically:**
1. Build new Docker image
2. Deploy to Cloud Run
3. Run smoke tests
4. Route traffic to new version

### 7.2: Zero-Downtime Updates

**How it works:**

1. New revision deploys without traffic
2. Health checks run on new revision
3. If healthy, traffic gradually shifts (0% → 100%)
4. Old revision stays running briefly
5. Old revision shut down after traffic shift

**No downtime!** Users never experience interruption.

### 7.3: Rollback if Needed

**If deployment has issues:**

**Automatic rollback:**
- GitHub Actions automatically rolls back if smoke tests fail

**Manual rollback:**

**Option A: Via GCP Console**

1. Go to Cloud Run service page
2. Click "REVISIONS" tab
3. Find previous working revision
4. Click "..." menu → "Manage traffic"
5. Set previous revision to 100%
6. Click "SAVE"

**Option B: Via gcloud CLI**

```bash
# List revisions
gcloud run revisions list \
  --service sql-studio-backend \
  --region us-central1

# Rollback to specific revision
gcloud run services update-traffic sql-studio-backend \
  --region us-central1 \
  --to-revisions REVISION_NAME=100
```

**Option C: Revert Git tag**

```bash
# Delete problematic tag
git tag -d v1.1.0
git push origin :refs/tags/v1.1.0

# Redeploy previous version
git tag v1.1.1  # Use new tag
git push origin v1.1.1
```

---

## Cost Breakdown

### Monthly Cost Estimates

**GCP Cloud Run:**

| Usage Level | Requests/Month | Monthly Cost |
|-------------|----------------|--------------|
| Hobby (few users) | 10,000 | **$0** (free tier) |
| Small project | 100,000 | **$0** (free tier) |
| Side project | 500,000 | **$0-2** |
| Small business | 2,000,000 | **$3-8** |
| Growing startup | 10,000,000 | **$30-60** |

**GCP Free Tier (always free):**
- 2 million requests/month
- 360,000 GB-seconds memory
- 180,000 vCPU-seconds
- 1 GB network egress (North America)

**Pricing factors:**
- **Requests:** First 2M free, then $0.40 per million
- **CPU time:** First 180K vCPU-seconds free, then $0.00002400/vCPU-second
- **Memory:** First 360K GB-seconds free, then $0.00000250/GB-second
- **Network egress:** First 1GB free, then $0.12/GB

**Tip:** With min_instances=0 (scale to zero), you're only charged while processing requests!

**Turso Database:**

| Plan | Databases | Storage | Row Reads | Monthly Cost |
|------|-----------|---------|-----------|--------------|
| **Free** | 500 | 9 GB total | 1 billion | **$0** |
| Starter | 10,000 | 50 GB total | 1 billion | $29 |
| Pro | Unlimited | 1 TB | Unlimited | $99 |

**Most hobby/small projects fit comfortably in free tier!**

**Resend Email:**

| Plan | Emails/Month | Emails/Day | Monthly Cost |
|------|--------------|------------|--------------|
| **Free** | 3,000 | 100 | **$0** |
| Pro | 50,000 | 1,500 | $20 |
| Business | 100,000 | 5,000 | $80 |

**Total Estimated Costs:**

| Project Size | GCP | Turso | Resend | **Total** |
|--------------|-----|-------|--------|-----------|
| Hobby project (< 100K req/mo) | $0 | $0 | $0 | **$0/mo** |
| Side project (500K req/mo) | $0-2 | $0 | $0 | **$0-2/mo** |
| Small business (2M req/mo) | $3-8 | $0 | $0 | **$3-8/mo** |
| Growing startup (10M req/mo) | $30-60 | $29 | $20 | **$79-109/mo** |

**Cost Optimization Tips:**

1. **Use scale-to-zero** (current config)
   - min_instances=0 in Cloud Run
   - Only pay when handling requests
   - ~1-2 second cold start (acceptable for most APIs)

2. **Right-size resources**
   - Start with 512MB RAM, 1 vCPU (current config)
   - Monitor usage, scale up only if needed
   - Most Go apps run fine with these resources

3. **Enable CPU throttling** (current config)
   - Cloud Run only charges CPU during request processing
   - Saves significant costs vs. always-on CPU

4. **Use request-based pricing**
   - Cloud Run charges per request, not per hour
   - Much cheaper than traditional VPS for bursty traffic

5. **Monitor and set alerts**
   - Set billing alerts at $5, $10, $20 thresholds
   - Review Cloud Run metrics monthly
   - Optimize queries/code to reduce CPU time

6. **Stay in free tiers**
   - Turso: 1B row reads/month is generous
   - Resend: 100 emails/day for user verification
   - Cloud Run: 2M requests/month covers small projects

**View your current costs:**
- GCP: https://console.cloud.google.com/billing
- Turso: https://turso.tech/pricing (check usage in dashboard)
- Resend: https://resend.com/settings/usage

---

## Troubleshooting

### Deployment Fails at "Pre-deployment Validation"

**Symptom:** GitHub Actions fails with "Missing required secrets"

**Solution:**

1. Go to repository Settings → Secrets and variables → Actions
2. Verify all required secrets are present:
   - GCP_PROJECT_ID
   - GCP_SA_KEY
   - TURSO_URL
   - TURSO_AUTH_TOKEN
   - JWT_SECRET
3. Check for typos in secret names (case-sensitive!)
4. Re-run workflow: Actions → Failed workflow → "Re-run all jobs"

### Deployment Fails at "Build Docker Image"

**Symptom:** Error "failed to build image" or "CGO compilation error"

**Solution:**

1. Check Dockerfile exists: `backend-go/Dockerfile`
2. Verify Go code compiles locally:
   ```bash
   cd backend-go
   go build -o test cmd/server/main.go
   ```
3. Check GitHub Actions logs for specific error
4. Common fixes:
   - Missing dependency in go.mod: `go mod tidy`
   - Syntax error in Go code: Fix and commit

### Deployment Fails at "Deploy to Cloud Run"

**Symptom:** Error "Permission denied" or "Forbidden"

**Solution:**

1. Verify GCP_SA_KEY secret is the full JSON (starts with `{`, ends with `}`)
2. Check service account has all required roles:
   ```bash
   gcloud projects get-iam-policy $PROJECT_ID \
     --flatten="bindings[].members" \
     --filter="bindings.members:github-actions@*"
   ```
3. Should show roles:
   - `roles/run.admin`
   - `roles/iam.serviceAccountUser`
   - `roles/secretmanager.admin`
   - `roles/storage.admin`
4. If missing, re-run commands from Step 1.4

### Health Check Fails

**Symptom:** Deployment succeeds but health check returns 502 or 503

**Solution:**

1. **Check service logs:**
   ```bash
   gcloud run services logs read sql-studio-backend \
     --region us-central1 \
     --limit 50
   ```

2. **Common issues:**

   **Database connection error:**
   - Verify TURSO_URL and TURSO_AUTH_TOKEN in GCP Secret Manager
   - Test Turso connection: `turso db shell sql-studio-prod`
   - Check database is accessible: `turso db show sql-studio-prod`

   **Port mismatch:**
   - Ensure Cloud Run `--port=8500` matches Dockerfile `EXPOSE 8500`
   - Check `SERVER_HTTP_PORT` environment variable

   **Application crash:**
   - Review logs for panic/error messages
   - Test Docker image locally:
     ```bash
     docker run -p 8500:8500 \
       -e TURSO_URL="$TURSO_URL" \
       -e TURSO_AUTH_TOKEN="$TURSO_AUTH_TOKEN" \
       -e JWT_SECRET="test-secret" \
       gcr.io/$PROJECT_ID/sql-studio-backend:latest
     ```

3. **Test health endpoint:**
   ```bash
   curl https://YOUR-SERVICE-URL/health
   ```

### High Costs (Unexpected Billing)

**Symptom:** GCP billing higher than expected

**Solution:**

1. **Check Cloud Run metrics:**
   - Go to Cloud Run → sql-studio-backend → METRICS
   - Review: Request count, instance count, CPU/memory usage

2. **Common causes:**

   **Not scaling to zero:**
   - Check min_instances=0:
     ```bash
     gcloud run services describe sql-studio-backend \
       --region us-central1 \
       --format 'value(spec.template.spec.containerConcurrency)'
     ```
   - Fix:
     ```bash
     gcloud run services update sql-studio-backend \
       --region us-central1 \
       --min-instances 0
     ```

   **High request volume:**
   - Check request count in metrics
   - Investigate unexpected traffic (possible bot/attack)
   - Enable Cloud Armor if DDoS suspected

   **Memory/CPU too high:**
   - Reduce if underutilized:
     ```bash
     gcloud run services update sql-studio-backend \
       --region us-central1 \
       --memory 256Mi \
       --cpu 1
     ```

3. **Set up billing alerts:**
   - Go to Billing → Budgets & alerts
   - Create alert at $5, $10, $20 thresholds

### Secrets Not Found

**Symptom:** Error "Secret not found" in logs

**Solution:**

1. **Check secrets exist in GCP Secret Manager:**
   ```bash
   gcloud secrets list
   ```

2. **Should see:**
   - turso-url
   - turso-auth-token
   - jwt-secret
   - resend-api-key (if using email)
   - resend-from-email (if using email)

3. **If missing, create manually:**
   ```bash
   echo -n "YOUR_VALUE" | gcloud secrets create SECRET_NAME \
     --replication-policy="automatic" \
     --data-file=-
   ```

4. **Grant Cloud Run access:**
   ```bash
   PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID \
     --format="value(projectNumber)")

   gcloud secrets add-iam-policy-binding SECRET_NAME \
     --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
     --role="roles/secretmanager.secretAccessor"
   ```

### Turso Connection Timeout

**Symptom:** Error "failed to connect to database" or "timeout"

**Solution:**

1. **Verify database exists:**
   ```bash
   turso db show sql-studio-prod
   ```

2. **Check auth token is valid:**
   ```bash
   turso db tokens list sql-studio-prod
   ```

3. **Rotate token if expired:**
   ```bash
   # Create new token
   NEW_TOKEN=$(turso db tokens create sql-studio-prod)

   # Update in GCP Secret Manager
   echo -n "$NEW_TOKEN" | gcloud secrets versions add turso-auth-token \
     --data-file=-

   # Redeploy service (picks up new secret)
   gcloud run services update sql-studio-backend \
     --region us-central1
   ```

4. **Test connection manually:**
   ```bash
   turso db shell sql-studio-prod "SELECT 1"
   ```

### Email Not Sending

**Symptom:** Emails not arriving, errors in logs

**Solution:**

1. **Verify Resend API key:**
   - Go to https://resend.com/api-keys
   - Check key is active
   - Create new key if needed

2. **Check from address:**
   - If using custom domain, verify DNS records
   - Or use Resend test domain: `onboarding@resend.dev`

3. **Test email sending:**
   ```bash
   curl -X POST 'https://api.resend.com/emails' \
     -H 'Authorization: Bearer YOUR_RESEND_API_KEY' \
     -H 'Content-Type: application/json' \
     -d '{
       "from": "onboarding@resend.dev",
       "to": "your-email@example.com",
       "subject": "Test Email",
       "html": "<p>Test</p>"
     }'
   ```

4. **Update secret in GCP:**
   ```bash
   echo -n "re_new_key" | gcloud secrets versions add resend-api-key \
     --data-file=-
   ```

### Deployment Stuck/Hanging

**Symptom:** GitHub Actions workflow running for > 15 minutes

**Solution:**

1. **Cancel workflow:**
   - In GitHub Actions, click "Cancel workflow"

2. **Check GCP Cloud Build:**
   - Go to https://console.cloud.google.com/cloud-build
   - Look for stuck builds
   - Cancel if hung

3. **Re-run deployment:**
   - Delete and recreate tag:
     ```bash
     git tag -d v1.0.0
     git push origin :refs/tags/v1.0.0
     git tag v1.0.0
     git push origin v1.0.0
     ```

4. **Or trigger manually:**
   - GitHub → Actions → Deploy Backend → Run workflow

### Getting Help

**If you're still stuck:**

1. **Check GitHub Discussions:**
   - https://github.com/sql-studio/sql-studio/discussions
   - Search for similar issues
   - Ask for help (include logs!)

2. **File GitHub Issue:**
   - https://github.com/sql-studio/sql-studio/issues
   - Include:
     - What you were trying to do
     - What happened (error messages)
     - GitHub Actions logs
     - GCP Cloud Run logs

3. **Community Support:**
   - GCP: https://cloud.google.com/support
   - Turso: https://discord.gg/turso
   - Resend: https://resend.com/support

4. **Paid Support (if needed):**
   - GCP: Upgrade to paid support plan
   - SQL Studio: Contact maintainers for consulting

---

## Next Steps

**Congratulations!** You've successfully deployed SQL Studio backend to production.

**What to do next:**

1. **Test your API:**
   - Use Postman/Insomnia to test endpoints
   - Review API documentation: `backend-go/API_DOCUMENTATION.md`

2. **Connect frontend:**
   - Update frontend API URL to your Cloud Run URL
   - Test end-to-end user flows

3. **Set up CI/CD:**
   - Your deployment is already automated!
   - Every git tag triggers deployment
   - Monitor GitHub Actions for status

4. **Monitor performance:**
   - Check Cloud Run metrics daily
   - Set up uptime monitoring
   - Review logs for errors

5. **Optimize costs:**
   - Review billing after 1 week
   - Adjust resources if needed
   - Consider reserved capacity for consistent traffic

6. **Plan for growth:**
   - When to upgrade Turso plan (> 1B row reads/month)
   - When to upgrade Resend (> 3,000 emails/month)
   - When to add caching (Redis/Memcached)
   - When to add CDN (Cloudflare/Fastly)

**You're production-ready!** Enjoy your deployed SQL Studio backend!

---

**Document Version:** 1.0.0
**Last Updated:** 2025-10-23
**Deployment Platform:** GCP Cloud Run
**Estimated Setup Time:** 30-60 minutes
