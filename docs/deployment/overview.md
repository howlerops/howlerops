# HowlerOps - Complete Deployment Overview

Production deployment guide for the complete HowlerOps stack.

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        FRONTEND                              │
│  Platform: Vercel (Free)                                     │
│  URL: https://howlerops.vercel.app                          │
│  Cost: $0/month                                              │
└─────────────────┬────────────────────────────────────────────┘
                  │ API Calls (HTTPS)
                  ↓
┌──────────────────────────────────────────────────────────────┐
│                        BACKEND                               │
│  Platform: Google Cloud Run                                  │
│  URL: https://sql-studio-backend-xyz.run.app                │
│  Cost: $0-3/month                                            │
└──────────────────┬───────────────────┬───────────────────────┘
                   │                   │
        ┌──────────▼────────┐    ┌────▼──────────┐
        │  Turso Database   │    │  Resend Email │
        │  Cost: $0/month   │    │  Cost: $0/mo  │
        └───────────────────┘    └───────────────┘
```

**Total Monthly Cost:** $0-3 (with free tiers)

---

## 📋 Deployment Steps

### 1️⃣ Backend Deployment (30-45 min)

**Location:** `backend-go/`

**Quick Start:**
```bash
cd backend-go

# Follow the checklist
open SETUP_CHECKLIST.md

# Or read comprehensive guide
open DEPLOYMENT_GUIDE.md
```

**What you need:**
- Google Cloud account
- Turso database
- Resend email account
- JWT secret

**Deploy commands:**
```bash
make deploy-full      # Production
make deploy-staging   # Staging
```

**Result:** Backend API running at Cloud Run URL

---

### 2️⃣ Frontend Deployment (15 min)

**Location:** `frontend/`

**Quick Start:**
```bash
cd frontend

# Follow the guide
open DEPLOYMENT.md
```

**What you need:**
- Vercel account (free)
- Backend URL (from step 1)

**Deploy:**
1. Connect GitHub repo to Vercel
2. Set `VITE_API_URL` environment variable
3. Deploy (automatic)

**Result:** Frontend live at Vercel URL

---

## 🤖 Automated Deployments

### Backend (GitHub Actions)

**File:** `.github/workflows/deploy-cloud-run.yml`

**Triggers:**
- `git tag v1.0.0` → Production
- `git push origin main` → Staging
- Pull request → Preview environment

**Setup:**
1. Add GitHub secrets (see `backend-go/SETUP_CHECKLIST.md`)
2. Push tag to deploy

### Frontend (Vercel)

**Auto-configured** when you connect to Vercel:
- Push to `main` → Production
- Pull request → Preview
- Every commit → Deployment preview

---

## 🌍 Environments

| Environment | Backend | Frontend | Trigger |
|------------|---------|----------|---------|
| **Production** | `sql-studio-backend` | `howlerops.vercel.app` | Git tag |
| **Staging** | `sql-studio-backend-staging` | `howlerops-staging.vercel.app` | Push to main |
| **Preview** | `sql-studio-preview-{branch}` | `howlerops-git-{branch}.vercel.app` | Pull request |

---

## 📊 Monitoring

### Backend
```bash
cd backend-go
make prod-logs        # View logs
make prod-status      # Check health
```

**Dashboard:** https://console.cloud.google.com/run

### Frontend

**Dashboard:** https://vercel.com/dashboard
- Analytics
- Deployment logs
- Error tracking

---

## 💰 Cost Breakdown

### Free Tier Limits

**Google Cloud Run:**
- 2M requests/month
- 180K vCPU-seconds
- 360K GiB-seconds

**Turso:**
- 500MB storage
- 1B row reads/month

**Resend:**
- 100 emails/day
- 3K emails/month

**Vercel:**
- 100GB bandwidth
- Unlimited requests

### Expected Monthly Cost

| Service | Expected Usage | Cost |
|---------|---------------|------|
| Cloud Run | <1M requests | $0-2 |
| Turso | <100MB | $0 |
| Resend | <100 emails/day | $0 |
| Vercel | <10GB bandwidth | $0 |
| **TOTAL** | | **$0-3/month** |

---

## ✅ Complete Deployment Checklist

### Prerequisites
- [ ] GCP account with billing enabled
- [ ] Turso account and database
- [ ] Resend account and API key
- [ ] Vercel account
- [ ] GitHub repo

### Backend Setup
- [ ] Configure `.env.production`
- [ ] Setup GCP secrets (`make setup-gcp-secrets`)
- [ ] Deploy backend (`make deploy-prod`)
- [ ] Verify health endpoint
- [ ] Configure GitHub secrets for CI/CD

### Frontend Setup
- [ ] Connect repo to Vercel
- [ ] Set `VITE_API_URL` environment variable
- [ ] Deploy frontend
- [ ] Test API connectivity

### Post-Deployment
- [ ] Update CORS in backend with Vercel domain
- [ ] Test complete user flow
- [ ] Setup monitoring alerts
- [ ] Configure custom domains (optional)
- [ ] Document URLs for team

---

## 🚀 Deploy Commands Quick Reference

### Backend
```bash
cd backend-go

# Local deployment
make deploy-full                        # Production
make deploy-staging                     # Staging
make deploy-preview BRANCH=feature-x    # Preview

# Via GitHub
git tag v1.0.0 && git push origin v1.0.0    # Production
git push origin main                         # Staging
gh pr create                                 # Preview
```

### Frontend
```bash
# Automatic via Vercel
git push origin main                    # Production
gh pr create                            # Preview

# Manual via CLI
vercel --prod                           # Production
vercel                                  # Preview
```

---

## 📚 Documentation

### Complete Guides
- **Backend:** `backend-go/DEPLOYMENT_GUIDE.md`
- **Frontend:** `frontend/DEPLOYMENT.md`

### Quick Reference
- **Backend Setup:** `backend-go/SETUP_CHECKLIST.md`
- **Makefile Help:** `make prod-help`

### Configuration
- **Backend Env:** `backend-go/.env.production`
- **GitHub Workflow:** `.github/workflows/deploy-cloud-run.yml`
- **Vercel:** Configure in dashboard

---

## 🔒 Security Notes

**Never commit:**
- `.env.production`
- Service account keys (`*.json`)
- API keys or tokens

**Always use:**
- GCP Secret Manager (backend secrets)
- GitHub Secrets (CI/CD credentials)
- Vercel Environment Variables (frontend config)

**Best practices:**
- Rotate JWT secret if compromised
- Review IAM permissions quarterly
- Enable 2FA on all accounts
- Monitor access logs

---

## 🆘 Getting Help

### Documentation
1. Start with deployment guides in each directory
2. Check troubleshooting sections
3. Review error logs (Cloud Run & Vercel dashboards)

### Platform Docs
- [Cloud Run](https://cloud.google.com/run/docs)
- [Vercel](https://vercel.com/docs)
- [Turso](https://docs.turso.tech)
- [GitHub Actions](https://docs.github.com/en/actions)

### Common Issues
- **CORS errors:** Update `ALLOWED_ORIGINS` in backend
- **API not found:** Check `VITE_API_URL` in Vercel
- **Build fails:** Check logs in respective dashboards
- **Secrets errors:** Re-run `make setup-gcp-secrets`

---

## 🎯 Next Steps

1. **Backend first:** Follow `backend-go/SETUP_CHECKLIST.md`
2. **Then frontend:** Follow `frontend/DEPLOYMENT.md`
3. **Test everything:** Verify complete user flow
4. **Setup monitoring:** Configure alerts and dashboards
5. **Custom domains:** Add your domains (optional)

**Ready to deploy?** Start with the backend checklist! 🚀

---

**Last Updated:** 2025-10-26
**Architecture:** Vercel (Frontend) + Cloud Run (Backend)
**Target Cost:** $0-3/month
