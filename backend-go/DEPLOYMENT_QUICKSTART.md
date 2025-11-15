# Howlerops Backend - Deployment Quick Start

Quick reference guide for deploying Howlerops backend to production.

## One-Command Deployments

### GCP Cloud Run

```bash
# Setup (first time only)
cd backend-go
./scripts/deploy-gcp.sh --project YOUR_PROJECT_ID --setup-secrets

# Deploy
./scripts/deploy-gcp.sh --project YOUR_PROJECT_ID

# Deploy with Cloud Build (for CI/CD)
./scripts/deploy-gcp.sh --project YOUR_PROJECT_ID --use-cloudbuild
```

### Fly.io

```bash
# Setup (first time only)
cd backend-go
./scripts/deploy-fly.sh --setup-app --setup-secrets

# Deploy
./scripts/deploy-fly.sh

# Deploy with remote build (no Docker needed)
./scripts/deploy-fly.sh --remote-build
```

## Prerequisites Checklist

- [ ] Docker installed (for local builds)
- [ ] Turso database created and credentials ready
- [ ] Resend API key obtained
- [ ] JWT secret generated (64+ characters)
- [ ] Platform CLI installed (gcloud or flyctl)

## Required Secrets

```bash
# Generate JWT secret
openssl rand -base64 64

# Get Turso credentials
turso db show your-database
turso db tokens create your-database

# Get Resend API key
# Visit: https://resend.com/api-keys
```

## Platform Comparison

| Feature | GCP Cloud Run | Fly.io |
|---------|--------------|--------|
| Setup Time | 15-20 min | 5-10 min |
| Free Tier | 2M requests/month | None (scale-to-zero) |
| Scale to Zero | Yes | Yes |
| Cold Start | ~2-3s | ~1-2s |
| Monthly Cost (low traffic) | $0-5 | $0-5 |
| Best For | Enterprise, high reliability | Side projects, MVPs |

## Quick Deploy Commands

### GCP Cloud Run

```bash
# Prerequisites
gcloud auth login
gcloud config set project YOUR_PROJECT_ID

# Enable APIs
gcloud services enable run.googleapis.com \
  cloudbuild.googleapis.com \
  secretmanager.googleapis.com

# Set secrets
echo -n "YOUR_TURSO_URL" | gcloud secrets create turso-url --data-file=-
echo -n "YOUR_TURSO_TOKEN" | gcloud secrets create turso-auth-token --data-file=-
echo -n "YOUR_JWT_SECRET" | gcloud secrets create jwt-secret --data-file=-
echo -n "YOUR_RESEND_KEY" | gcloud secrets create resend-api-key --data-file=-

# Deploy
cd backend-go
gcloud builds submit --config=cloudbuild.yaml

# Or use the deployment script
./scripts/deploy-gcp.sh --project YOUR_PROJECT_ID
```

### Fly.io

```bash
# Prerequisites
flyctl auth login

# Create app
cd backend-go
flyctl apps create sql-studio-backend

# Set secrets
flyctl secrets set \
  TURSO_URL="YOUR_TURSO_URL" \
  TURSO_AUTH_TOKEN="YOUR_TURSO_TOKEN" \
  JWT_SECRET="YOUR_JWT_SECRET" \
  RESEND_API_KEY="YOUR_RESEND_KEY" \
  --app sql-studio-backend

# Deploy
flyctl deploy

# Or use the deployment script
./scripts/deploy-fly.sh
```

## Post-Deployment Verification

```bash
# Get your app URL
# GCP: gcloud run services describe sql-studio-backend --region=us-central1 --format='value(status.url)'
# Fly.io: flyctl info --app sql-studio-backend

# Test health endpoint
curl https://your-app-url/health

# Expected response:
# {"status":"healthy","version":"1.0.0"}

# View logs
# GCP: gcloud run services logs tail sql-studio-backend --region=us-central1
# Fly.io: flyctl logs --app sql-studio-backend
```

## Troubleshooting

### Deployment Fails

```bash
# Check logs
# GCP
gcloud run services logs read sql-studio-backend --region=us-central1 --limit=50

# Fly.io
flyctl logs --app sql-studio-backend

# Verify secrets
# GCP
gcloud secrets list

# Fly.io
flyctl secrets list --app sql-studio-backend
```

### Health Check Fails

```bash
# Test locally first
docker build -t test .
docker run -p 8080:8080 \
  -e TURSO_URL="YOUR_URL" \
  -e TURSO_AUTH_TOKEN="YOUR_TOKEN" \
  -e JWT_SECRET="YOUR_SECRET" \
  -e RESEND_API_KEY="YOUR_KEY" \
  test

# Test health endpoint
curl http://localhost:8080/health
```

### High Costs

```bash
# GCP: Set min instances to 0
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --min-instances=0

# Fly.io: Verify scale-to-zero is enabled
# Check fly.toml:
# [http_service]
#   min_machines_running = 0
#   auto_stop_machines = "stop"
```

## Cost Estimates

### GCP Cloud Run (512MB RAM, 1 CPU)
- **Free tier:** First 2M requests/month free
- **After free tier:** ~$0.10 per 100K requests
- **Low traffic (< 100K req/month):** $0-2/month
- **Medium traffic (1M req/month):** $5-15/month
- **High traffic (10M req/month):** $50-100/month

### Fly.io (512MB RAM, 1 CPU)
- **Scale-to-zero idle:** $0/month
- **Low traffic (running 10% of time):** $2-5/month
- **Medium traffic (running 50% of time):** $10-20/month
- **High traffic (always running):** $20-30/month

## Monitoring

### GCP Cloud Run

```bash
# View metrics in Cloud Console
open https://console.cloud.google.com/run/detail/us-central1/sql-studio-backend/metrics

# Set up alerts
gcloud monitoring uptime-checks create \
  --display-name="Howlerops Health" \
  --monitored-url=https://your-app-url/health
```

### Fly.io

```bash
# View dashboard
flyctl dashboard --app sql-studio-backend

# Check status
flyctl status --app sql-studio-backend

# Monitor logs
flyctl logs --app sql-studio-backend --follow
```

## Scaling

### GCP Cloud Run

```bash
# Increase max instances
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --max-instances=20

# Increase resources
gcloud run services update sql-studio-backend \
  --region=us-central1 \
  --memory=1Gi \
  --cpu=2
```

### Fly.io

```bash
# Add more machines
flyctl scale count 3 --app sql-studio-backend

# Increase machine size
flyctl scale vm shared-cpu-2x --memory 1024 --app sql-studio-backend

# Deploy to multiple regions
flyctl regions add lhr ord sjc --app sql-studio-backend
```

## Rollback

### GCP Cloud Run

```bash
# List revisions
gcloud run revisions list --service=sql-studio-backend --region=us-central1

# Rollback to previous revision
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-revisions=REVISION_NAME=100
```

### Fly.io

```bash
# List releases
flyctl releases --app sql-studio-backend

# Rollback to previous release
flyctl releases rollback --app sql-studio-backend
```

## Support

- **Full documentation:** See `DEPLOYMENT.md`
- **GitHub Issues:** https://github.com/yourusername/sql-studio/issues
- **GCP Support:** https://cloud.google.com/support
- **Fly.io Community:** https://community.fly.io/

## Next Steps

1. [ ] Set up custom domain (optional)
2. [ ] Configure CI/CD (GitHub Actions)
3. [ ] Set up monitoring alerts
4. [ ] Configure backups (Turso handles this)
5. [ ] Review security settings
6. [ ] Load test your deployment

---

**Need help?** Open a GitHub issue or consult the full deployment guide in `DEPLOYMENT.md`.
