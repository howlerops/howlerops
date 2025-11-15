# Howlerops Backend - Deployment Documentation

**Complete guide to deploying Howlerops backend to production**

---

## Quick Links

| Document | For | Time | Description |
|----------|-----|------|-------------|
| **[DEPLOYMENT_COMPLETE_GUIDE.md](./DEPLOYMENT_COMPLETE_GUIDE.md)** | Beginners | 30-60 min | Step-by-step guide with zero GCP/Turso experience required |
| **[DEPLOYMENT_QUICK_START.md](./DEPLOYMENT_QUICK_START.md)** | Experienced | 5-10 min | Commands-only guide for those who know GCP/Docker |
| **[DEPLOYMENT_FAQ.md](./DEPLOYMENT_FAQ.md)** | Everyone | As needed | Common questions and troubleshooting |

---

## Which Guide Should I Use?

### Choose the Complete Guide if:
- You've never deployed to GCP before
- You've never used Turso database
- You want screenshots and detailed explanations
- You want to understand each step
- You're deploying for the first time
- **Time: 30-60 minutes**

### Choose the Quick Start if:
- You're comfortable with GCP and gcloud CLI
- You know Docker and containerization
- You just need the commands
- You've deployed similar projects before
- **Time: 5-10 minutes**

### Use the FAQ when:
- Something went wrong during deployment
- You need to update configuration
- You have specific questions
- You're troubleshooting issues
- **Time: Find your answer in 1-2 minutes**

---

## What You're Deploying

**Howlerops Backend** is a Go-based API service that provides:
- REST API for Howlerops frontend
- User authentication (JWT)
- Query history and saved queries
- Cloud sync functionality
- Email notifications (via Resend)
- Multi-tenant data isolation

**Production stack:**
- **Platform:** Google Cloud Run (or Fly.io)
- **Database:** Turso (edge SQLite)
- **Email:** Resend
- **Deployment:** GitHub Actions (automated)
- **Monitoring:** GCP Cloud Monitoring

---

## Prerequisites Summary

**Accounts needed:**
- Google Cloud Platform (free $300 credit)
- Turso (free tier: 1B row reads/month)
- Resend (optional, free tier: 3,000 emails/month)
- GitHub (for automated deployment)

**Tools needed:**
- Git
- Google Cloud SDK (gcloud)
- OpenSSL (for generating secrets)

**Costs:**
- Hobby project: **$0-5/month** (likely $0)
- Small business: **$5-20/month**
- Growing startup: **$50-100/month**

See [DEPLOYMENT_COMPLETE_GUIDE.md - Cost Breakdown](./DEPLOYMENT_COMPLETE_GUIDE.md#cost-breakdown) for details.

---

## Deployment Flowchart

```
START
  |
  v
Have you deployed to GCP before?
  |
  ├─ NO ──> Use DEPLOYMENT_COMPLETE_GUIDE.md
  |         |
  |         v
  |         Follow step-by-step guide
  |         - Create GCP project
  |         - Set up Turso database
  |         - Configure GitHub Secrets
  |         - Deploy via GitHub Actions
  |         |
  |         v
  |         SUCCESS! API deployed in 30-60 min
  |
  ├─ YES ─> Use DEPLOYMENT_QUICK_START.md
            |
            v
            Run commands
            - gcloud setup
            - turso setup
            - GitHub secrets
            - git tag v1.0.0
            |
            v
            SUCCESS! API deployed in 5-10 min

Encountered an issue?
  |
  v
Check DEPLOYMENT_FAQ.md
  - Common errors and solutions
  - Configuration questions
  - Troubleshooting steps
```

---

## Deployment Options

### Option 1: Google Cloud Run (Recommended)

**Best for:**
- Production deployments
- Enterprise/compliance needs
- Businesses requiring SLAs
- Apps needing advanced monitoring

**Pros:**
- Mature platform (Google infrastructure)
- Excellent monitoring and logging
- 99.5% SLA (on paid tier)
- Auto-scaling to 1000+ instances
- Great documentation

**Cons:**
- Slightly more complex setup
- More expensive at high scale
- Requires GCP account

**Cost:** $0-5/month (hobby), $10-100/month (production)

---

### Option 2: Fly.io (Alternative)

**Best for:**
- Hobby projects and MVPs
- Global edge deployment
- Cost-sensitive projects
- Simpler setup

**Pros:**
- Cheaper (true scale-to-zero)
- Global edge network (30+ regions)
- Simpler setup and CLI
- Great for side projects

**Cons:**
- Less mature monitoring
- Smaller ecosystem
- No enterprise SLA on free tier

**Cost:** $0-2/month (hobby), $5-30/month (production)

---

## Quick Deployment (3 Commands)

**If you have GCP account and know the basics:**

```bash
# 1. Clone repository
git clone https://github.com/sql-studio/sql-studio.git
cd sql-studio

# 2. Configure GitHub Secrets
# Go to: Settings → Secrets → Add the following:
#   GCP_PROJECT_ID, GCP_SA_KEY, TURSO_URL,
#   TURSO_AUTH_TOKEN, JWT_SECRET

# 3. Deploy via tag
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions deploys automatically!
# Check progress: https://github.com/YOUR-REPO/actions
```

**For detailed setup:** See [DEPLOYMENT_QUICK_START.md](./DEPLOYMENT_QUICK_START.md)

---

## Documentation Overview

### Core Deployment Docs (NEW - Most Comprehensive)

| File | Lines | Size | Purpose |
|------|-------|------|---------|
| **DEPLOYMENT_COMPLETE_GUIDE.md** | 1,302 | 33KB | Beginner-friendly complete walkthrough |
| **DEPLOYMENT_QUICK_START.md** | 682 | 14KB | Quick reference for experienced users |
| **DEPLOYMENT_FAQ.md** | 1,189 | 30KB | Common questions and troubleshooting |

### Additional Resources

| File | Purpose |
|------|---------|
| **DEPLOYMENT.md** | Original comprehensive deployment guide |
| **DEPLOYMENT_SUMMARY.md** | Overview of deployment setup |
| **COST_ANALYSIS.md** | Detailed cost breakdown and comparison |
| **README.deployment.md** | Legacy quick start guide |

### Supporting Documentation

| File | Purpose |
|------|---------|
| **API_DOCUMENTATION.md** | API endpoints and usage |
| **ARCHITECTURE.md** | System architecture overview |
| **DEVELOPMENT.md** | Local development setup |
| **SETUP.md** | Initial project setup |

---

## Common Tasks

### Deploy New Version

```bash
# Make your changes
git add .
git commit -m "feat: new feature"
git push origin main

# Create release tag
git tag v1.1.0
git push origin v1.1.0

# GitHub Actions automatically deploys!
```

### Rollback Bad Deployment

```bash
# List revisions
gcloud run revisions list \
  --service sql-studio-backend \
  --region us-central1

# Rollback to previous version
gcloud run services update-traffic sql-studio-backend \
  --region us-central1 \
  --to-revisions REVISION-NAME=100
```

### View Logs

```bash
# Real-time logs
gcloud run services logs tail sql-studio-backend \
  --region us-central1

# Or via console
open "https://console.cloud.google.com/logs"
```

### Update Secret

```bash
# Generate new secret
NEW_SECRET=$(openssl rand -base64 64)

# Update in GCP Secret Manager
echo -n "$NEW_SECRET" | gcloud secrets versions add jwt-secret \
  --data-file=-

# Service auto-picks up new secret on next cold start
```

---

## Deployment Architecture

```
┌─────────────────────────────────────────────────────┐
│                   GitHub Repository                 │
│  ┌───────────────────────────────────────────────┐  │
│  │  Code Push or Release Tag (v1.0.0)            │  │
│  └───────────────┬───────────────────────────────┘  │
└──────────────────┼──────────────────────────────────┘
                   │
                   v
┌─────────────────────────────────────────────────────┐
│              GitHub Actions Workflow                │
│  ┌───────────────────────────────────────────────┐  │
│  │  1. Validate code and secrets                 │  │
│  │  2. Build Docker image (multi-arch)           │  │
│  │  3. Push to GCP Container Registry            │  │
│  │  4. Update secrets in Secret Manager          │  │
│  │  5. Deploy to Cloud Run (zero-downtime)       │  │
│  │  6. Run smoke tests                           │  │
│  │  7. Route traffic if healthy                  │  │
│  │  8. Auto-rollback on failure                  │  │
│  └───────────────┬───────────────────────────────┘  │
└──────────────────┼──────────────────────────────────┘
                   │
                   v
┌─────────────────────────────────────────────────────┐
│             Google Cloud Run Service                │
│  ┌───────────────────────────────────────────────┐  │
│  │  sql-studio-backend (Auto-scaling: 0-10)      │  │
│  │  - HTTP API: Port 8500                        │  │
│  │  - gRPC API: Port 9500                        │  │
│  │  - Metrics: Port 9100                         │  │
│  │  - Health checks: /health                     │  │
│  └───────────┬───────────────┬───────────────────┘  │
└──────────────┼───────────────┼──────────────────────┘
               │               │
               v               v
    ┌──────────────┐    ┌─────────────┐
    │ Turso DB     │    │ Resend      │
    │ (libSQL)     │    │ (Email)     │
    │              │    │             │
    │ - Edge DB    │    │ - SMTP      │
    │ - Auto backup│    │ - Templates │
    └──────────────┘    └─────────────┘
```

---

## Troubleshooting Quick Reference

| Problem | Solution |
|---------|----------|
| Deployment fails at "Validation" | Check GitHub Secrets are configured |
| Deployment fails at "Build" | Test Docker build locally |
| Deployment fails at "Deploy" | Check GCP permissions |
| 502/503 errors | Check database connection in logs |
| High costs | Verify min-instances=0 (scale-to-zero) |
| Slow response | Check Cloud Run metrics, add indexes |
| Secrets not found | Verify in GCP Secret Manager |
| Cold starts | Set min-instances=1 (costs more) |

**Full troubleshooting:** [DEPLOYMENT_FAQ.md - Troubleshooting](./DEPLOYMENT_FAQ.md#deployment-issues)

---

## Support and Resources

### Documentation
- **This repository:**
  - Complete Guide: [DEPLOYMENT_COMPLETE_GUIDE.md](./DEPLOYMENT_COMPLETE_GUIDE.md)
  - Quick Start: [DEPLOYMENT_QUICK_START.md](./DEPLOYMENT_QUICK_START.md)
  - FAQ: [DEPLOYMENT_FAQ.md](./DEPLOYMENT_FAQ.md)

### Official Platform Docs
- GCP Cloud Run: https://cloud.google.com/run/docs
- Fly.io: https://fly.io/docs
- Turso: https://docs.turso.tech
- Resend: https://resend.com/docs

### Community
- GitHub Issues: https://github.com/sql-studio/sql-studio/issues
- GitHub Discussions: https://github.com/sql-studio/sql-studio/discussions
- Turso Discord: https://discord.gg/turso

### Getting Help

1. **Check FAQ first:** [DEPLOYMENT_FAQ.md](./DEPLOYMENT_FAQ.md)
2. **Search GitHub Issues:** Likely someone had the same problem
3. **Check platform status:**
   - GCP: https://status.cloud.google.com
   - Turso: https://status.turso.tech
4. **Ask in Discussions:** https://github.com/sql-studio/sql-studio/discussions
5. **File an Issue:** Include logs and steps to reproduce

---

## Next Steps After Deployment

**Immediate (Day 1):**
- [ ] Test health endpoint: `curl $SERVICE_URL/health`
- [ ] Test API endpoints
- [ ] Set up uptime monitoring (UptimeRobot, Pingdom)
- [ ] Configure billing alerts ($5, $10, $20)

**Short-term (Week 1):**
- [ ] Connect frontend to backend
- [ ] Test end-to-end user flows
- [ ] Set up custom domain (optional)
- [ ] Configure email templates

**Long-term (Month 1+):**
- [ ] Review and optimize costs
- [ ] Set up proper monitoring/alerting
- [ ] Plan database backup strategy
- [ ] Load test under expected traffic
- [ ] Security audit and hardening

---

## Deployment Checklist

**Before deploying:**
- [ ] GCP project created and billing enabled
- [ ] Turso database created and tested
- [ ] All GitHub Secrets configured
- [ ] JWT_SECRET is 64+ characters
- [ ] Resend API key obtained (if using email)
- [ ] GitHub Actions workflow file reviewed

**After deploying:**
- [ ] Health endpoint returns 200 OK
- [ ] Logs show no errors
- [ ] Database connection successful
- [ ] API endpoints respond correctly
- [ ] Monitoring/alerts configured
- [ ] Costs verified (should be $0-5/month for hobby)

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-10-23 | Initial comprehensive deployment guides |
| - | - | Added DEPLOYMENT_COMPLETE_GUIDE.md |
| - | - | Added DEPLOYMENT_QUICK_START.md |
| - | - | Added DEPLOYMENT_FAQ.md |

---

**Ready to deploy?** Start with:
- **New to GCP?** → [DEPLOYMENT_COMPLETE_GUIDE.md](./DEPLOYMENT_COMPLETE_GUIDE.md)
- **Experienced?** → [DEPLOYMENT_QUICK_START.md](./DEPLOYMENT_QUICK_START.md)
- **Got questions?** → [DEPLOYMENT_FAQ.md](./DEPLOYMENT_FAQ.md)

**Good luck with your deployment!** 🚀
