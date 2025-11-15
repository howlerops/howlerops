# Howlerops Backend - Quick Deployment Guide

Fast-track deployment guide for Howlerops Go backend.

## Quick Start (5 minutes)

### Option 1: Deploy to GCP Cloud Run

```bash
# 1. Set environment variables
export GCP_PROJECT_ID="your-project-id"
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-token"
export RESEND_API_KEY="re_your-key"
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
export JWT_SECRET=$(openssl rand -base64 32)

# 2. Deploy
cd backend-go
./scripts/deploy-cloudrun.sh

# 3. Test
curl https://your-service-url/health
```

### Option 2: Deploy to Fly.io

```bash
# 1. Install Fly CLI
curl -L https://fly.io/install.sh | sh

# 2. Set environment variables
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your-token"
export RESEND_API_KEY="re_your-key"
export RESEND_FROM_EMAIL="noreply@yourdomain.com"
export JWT_SECRET=$(openssl rand -base64 32)

# 3. Deploy
cd backend-go
./scripts/deploy-fly.sh

# 4. Test
curl https://sql-studio-backend.fly.dev/health
```

## Files Included

```
backend-go/
├── Dockerfile                  # Multi-stage production build
├── cloudbuild.yaml            # GCP Cloud Build config
├── cloudrun.yaml              # GCP Cloud Run service definition
├── fly.toml                   # Fly.io configuration
├── scripts/
│   ├── deploy-cloudrun.sh    # GCP deployment script
│   └── deploy-fly.sh         # Fly.io deployment script
├── DEPLOYMENT.md              # Complete deployment guide
├── COST_ANALYSIS.md           # Cost comparison
└── README.deployment.md       # This file
```

## GitHub Actions Setup

1. Add these secrets to your GitHub repository:
   - `GCP_PROJECT_ID`
   - `GCP_SA_KEY` (service account JSON)
   - `FLY_API_TOKEN` (optional)
   - `TURSO_URL`
   - `TURSO_AUTH_TOKEN`
   - `RESEND_API_KEY`
   - `RESEND_FROM_EMAIL`
   - `JWT_SECRET`

2. Push to main:
   ```bash
   git push origin main
   ```

3. Automatic deployment starts!

## What's Included

### Production-Ready Features

- ✅ Multi-stage Docker build (optimized size)
- ✅ Health checks and monitoring
- ✅ Auto-scaling (0 to 1000+ instances)
- ✅ Zero-downtime deployments
- ✅ Secret management (no hardcoded credentials)
- ✅ CI/CD with GitHub Actions
- ✅ Comprehensive logging
- ✅ Cost optimization (scale-to-zero)

### Security Best Practices

- ✅ Non-root container user
- ✅ Secrets in Secret Manager/Vault
- ✅ HTTPS only (enforced)
- ✅ Minimal attack surface (Alpine Linux)
- ✅ No hardcoded credentials
- ✅ JWT token authentication

### Monitoring & Observability

- ✅ Health check endpoint (`/health`)
- ✅ Prometheus metrics (`/metrics` on port 9100)
- ✅ Structured JSON logging
- ✅ Request tracing
- ✅ Error alerting (configurable)

## Cost Estimates

| Scenario | GCP Cloud Run | Fly.io |
|----------|---------------|---------|
| Hobby (< 100k req/mo) | $0-5/mo | $0-2/mo |
| Small Business (1M req/mo) | $5-15/mo | $5-10/mo |
| Growing Startup (10M req/mo) | $50-100/mo | $30-50/mo |
| Enterprise (100M req/mo) | $500-1500/mo | $100-200/mo |

See [COST_ANALYSIS.md](./COST_ANALYSIS.md) for detailed breakdown.

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTPS
       ▼
┌─────────────────────────┐
│  Load Balancer / CDN    │
│  (GCP/Fly.io managed)   │
└──────────┬──────────────┘
           │
           ▼
┌──────────────────────────┐
│   Howlerops Backend     │
│  ┌────────────────────┐  │
│  │  HTTP API (8500)   │  │
│  │  gRPC API (9500)   │  │
│  │  Metrics (9100)    │  │
│  └────────────────────┘  │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐
│   External Services      │
│  ┌────────────────────┐  │
│  │  Turso (Database)  │  │
│  │  Resend (Email)    │  │
│  └────────────────────┘  │
└──────────────────────────┘
```

## API Endpoints

- `GET /health` - Health check (returns 200 OK)
- `GET /metrics` - Prometheus metrics
- `POST /api/v1/*` - REST API endpoints
- `gRPC :9500` - gRPC services

## Environment Variables Reference

### Required

```bash
TURSO_URL=libsql://your-db.turso.io
TURSO_AUTH_TOKEN=your-auth-token
RESEND_API_KEY=re_your-api-key
RESEND_FROM_EMAIL=noreply@yourdomain.com
JWT_SECRET=your-32-char-minimum-secret
```

### Optional

```bash
ENVIRONMENT=production        # Environment name
LOG_LEVEL=info               # debug, info, warn, error
LOG_FORMAT=json              # json or text
SERVER_HTTP_PORT=8500        # HTTP server port
SERVER_GRPC_PORT=9500        # gRPC server port
METRICS_PORT=9100            # Metrics port
```

## Common Commands

### GCP Cloud Run

```bash
# View logs
gcloud run services logs tail sql-studio-backend --region us-central1

# Scale
gcloud run services update sql-studio-backend --min-instances 1 --max-instances 20

# Rollback
gcloud run services update-traffic sql-studio-backend --to-revisions REVISION_NAME=100

# Update secret
echo -n "new-value" | gcloud secrets versions add secret-name --data-file=-
```

### Fly.io

```bash
# View logs
flyctl logs --follow

# Scale
flyctl scale count 3
flyctl scale vm shared-cpu-1x --memory 1024

# Rollback
flyctl releases rollback

# Update secret
flyctl secrets set KEY=value
```

## Troubleshooting

### Service Not Starting

1. Check logs:
   ```bash
   # GCP
   gcloud run services logs tail sql-studio-backend

   # Fly.io
   flyctl logs
   ```

2. Verify secrets are set correctly
3. Test locally:
   ```bash
   docker build -t test .
   docker run -p 8500:8500 -e TURSO_URL="..." test
   ```

### Health Check Failing

- Verify port 8500 is exposed and accessible
- Check database connection (TURSO_URL and token)
- Ensure JWT_SECRET is at least 32 characters

### High Costs

1. Enable scale-to-zero (if not already)
2. Reduce memory allocation if overprovisioned
3. Optimize response times to reduce CPU usage
4. Check egress bandwidth (optimize payload sizes)

## Support

- Full docs: [DEPLOYMENT.md](./DEPLOYMENT.md)
- Cost analysis: [COST_ANALYSIS.md](./COST_ANALYSIS.md)
- Issues: https://github.com/sql-studio/sql-studio/issues

## Next Steps

After deployment:

1. Set up custom domain
2. Configure monitoring alerts
3. Set up backup/disaster recovery
4. Implement rate limiting (if needed)
5. Add authentication/authorization
6. Set up CI/CD for automated deployments

## Production Checklist

Before going to production:

- [ ] Secrets are properly configured (not hardcoded)
- [ ] JWT_SECRET is at least 32 characters
- [ ] HTTPS is enabled and enforced
- [ ] Health checks are passing
- [ ] Monitoring is set up
- [ ] Logs are being collected
- [ ] Backup strategy is in place
- [ ] Rate limiting is configured
- [ ] Cost alerts are set
- [ ] Rollback procedure is tested

## License

MIT License - See LICENSE file for details
