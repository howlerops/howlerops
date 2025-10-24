# SQL Studio Backend - Quick Deploy Guide

## Prerequisites (5 minutes)

```bash
# 1. Install gcloud CLI
brew install google-cloud-sdk  # macOS
# or download from https://cloud.google.com/sdk

# 2. Authenticate
gcloud auth login

# 3. Create/Select GCP Project
export GCP_PROJECT_ID="sql-studio-prod"
gcloud config set project $GCP_PROJECT_ID
```

## Setup Required Secrets (2 minutes)

```bash
# Set these environment variables (replace with your values)
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-turso-token"
export RESEND_API_KEY="re_your-resend-key"
export JWT_SECRET=$(openssl rand -base64 32)
```

## Deploy (One Command!)

```bash
cd /Users/jacob_1/projects/sql-studio/backend-go
./scripts/deploy-full.sh
```

This will:
1. Check production readiness
2. Setup GCP secrets
3. Build Docker container
4. Deploy to Cloud Run
5. Verify deployment
6. Show you the service URL

## Or Deploy Step-by-Step

```bash
# Step 1: Verify everything is ready
make prod-check

# Step 2: Setup secrets (one-time)
make setup-gcp-secrets

# Step 3: Deploy
make deploy-prod

# Step 4: Get service URL
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format='value(status.url)')

# Step 5: Verify
make verify-prod SERVICE_URL=$SERVICE_URL
```

## Monitor Your Deployment

```bash
# View logs
make prod-logs

# Check status
make prod-status

# Check costs
make check-costs
```

## Rollback if Needed

```bash
# List all revisions
make prod-revisions

# Rollback to previous
make prod-rollback REVISION=sql-studio-backend-00001-abc
```

## Expected Cost

**$0-3/month** (within GCP free tier for MVP usage)

## Troubleshooting

If deployment fails:

1. Check logs: `make prod-logs`
2. Verify secrets: `gcloud secrets list`
3. Check status: `make prod-status`
4. Review checklist: `PRODUCTION_CHECKLIST.md`

## Next Steps After Deployment

1. Configure monitoring alerts
2. Setup budget notifications ($5 threshold)
3. Add custom domain (optional)
4. Configure frontend to use production URL
5. Monitor for 24 hours

## Support

- Full Guide: `PRODUCTION_DEPLOYMENT_SUMMARY.md`
- Checklist: `PRODUCTION_CHECKLIST.md`
- Commands: `make prod-help`
- Deployment Guide: `DEPLOYMENT.md`

---

**That's it! Your backend is production-ready with:**
- Automated deployment
- Comprehensive verification
- Secure secret management
- Cost monitoring
- Rollback capability
- Production monitoring
