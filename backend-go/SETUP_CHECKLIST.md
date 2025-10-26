# SQL Studio Deployment - Setup Checklist

Quick reference for setting up production deployment. Check off each item as you complete it.

---

## 📋 One-Time Setup Tasks

### ☐ 1. Google Cloud Platform

```bash
# Create project
gcloud projects create howlerops-prod --name="HowlerOps"
gcloud config set project howlerops-prod

# Enable billing
# Visit: https://console.cloud.google.com/billing

# Save project ID
export GCP_PROJECT_ID="howlerops-prod"
```

**What you need:** GCP account (free tier available)
**Time:** 5 minutes

---

### ☐ 2. Turso Database

```bash
# Install CLI
brew install tursodatabase/tap/turso

# Login and create database
turso auth login
turso db create howlerops-prod

# Get credentials
turso db show howlerops-prod --url        # Save as TURSO_URL
turso db tokens create howlerops-prod     # Save as TURSO_AUTH_TOKEN
```

**What you need:** Turso account (free tier: 500MB)
**Time:** 5 minutes

---

### ☐ 3. Resend Email

1. Sign up at https://resend.com
2. Verify your domain (add DNS records)
3. Create API key at https://resend.com/api-keys
4. Save API key as `RESEND_API_KEY`

**What you need:** Domain name, email account
**Time:** 10 minutes (waiting for DNS propagation)

---

### ☐ 4. Generate JWT Secret

```bash
# Generate secret
openssl rand -base64 32    # Save as JWT_SECRET
```

**Time:** 1 minute

---

### ☐ 5. Configure Local Environment

```bash
cd backend-go
cp .env.production.example .env.production
nano .env.production    # Fill in all values
```

Required in `.env.production`:
- `GCP_PROJECT_ID`
- `TURSO_URL`
- `TURSO_AUTH_TOKEN`
- `RESEND_API_KEY`
- `JWT_SECRET`

**Time:** 2 minutes

---

### ☐ 6. Setup GCP Secrets

```bash
source .env.production
make setup-gcp-secrets
```

**Time:** 2 minutes

---

### ☐ 7. Deploy to Production

```bash
make deploy-prod
```

**Time:** 3-5 minutes

---

### ☐ 8. Verify Deployment

```bash
# Get URL from deployment output
curl https://your-service-url.run.app/health
```

**Expected:** `{"status":"healthy"}`

---

## 🤖 GitHub Actions Setup (Optional)

### ☐ 1. Create Service Account

```bash
# Create service account
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions CI/CD"

# Grant permissions
for role in run.admin iam.serviceAccountUser secretmanager.secretAccessor storage.admin; do
  gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
    --member="serviceAccount:github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/$role"
done

# Create key
gcloud iam service-accounts keys create github-actions-key.json \
  --iam-account=github-actions@$GCP_PROJECT_ID.iam.gserviceaccount.com
```

---

### ☐ 2. Add GitHub Secrets

Go to: `GitHub → Settings → Secrets and variables → Actions`

Add these secrets (copy from your `.env.production`):
- [ ] `GCP_PROJECT_ID`
- [ ] `GCP_SA_KEY` (contents of `github-actions-key.json`)
- [ ] `TURSO_URL`
- [ ] `TURSO_AUTH_TOKEN`
- [ ] `JWT_SECRET`
- [ ] `RESEND_API_KEY`

---

### ☐ 3. Clean Up

```bash
# Delete local service account key
rm github-actions-key.json

# Verify .env.production is in .gitignore
git check-ignore .env.production    # Should output: .env.production
```

---

### ☐ 4. Test GitHub Actions

```bash
# Create and push a tag
git tag v1.0.0
git push origin v1.0.0

# Watch deployment at:
# https://github.com/your-org/your-repo/actions
```

---

## ✅ Verification Checklist

After deployment, verify:

- [ ] Service is accessible via Cloud Run URL
- [ ] Health endpoint returns 200 OK
- [ ] Logs show no errors: `make prod-logs`
- [ ] Can create a test user (test signup endpoint)
- [ ] Emails are being sent (check Resend dashboard)
- [ ] GitHub Actions workflow runs successfully
- [ ] Preview environments work on pull requests

---

## 🔒 Security Checklist

Before going live:

- [ ] `.env.production` is NOT committed to git
- [ ] Service account key JSON is NOT committed to git
- [ ] JWT secret is at least 32 characters
- [ ] Turso token is kept secret
- [ ] Resend API key is kept secret
- [ ] All secrets are in GCP Secret Manager
- [ ] All GitHub secrets are configured
- [ ] CORS origins are restricted to your domain
- [ ] Rate limiting is enabled

---

## 💰 Cost Monitoring Checklist

Setup budget alerts:

- [ ] Enable billing export to BigQuery (optional)
- [ ] Set up budget alert at $5/month
- [ ] Review costs weekly for first month
- [ ] Monitor Cloud Run metrics
- [ ] Check Turso usage dashboard
- [ ] Monitor Resend email usage

---

## 📚 Quick Commands Reference

```bash
# Local development
make dev                          # Run locally

# Deployment
make deploy-prod                  # Deploy to production
make deploy-staging               # Deploy to staging
make deploy-preview BRANCH=feat   # Deploy preview

# Monitoring
make prod-logs                    # View logs
make prod-status                  # Check status

# Troubleshooting
make prod-check                   # Pre-deployment checks
make setup-gcp-secrets            # Refresh secrets
```

---

## 🆘 Need Help?

- **Deployment Guide:** `DEPLOYMENT_GUIDE.md` (comprehensive)
- **Makefile help:** `make prod-help`
- **Cloud Run docs:** https://cloud.google.com/run/docs
- **Turso docs:** https://docs.turso.tech

---

**Total Setup Time:** ~30-45 minutes (including DNS propagation wait)
**Monthly Cost:** $0-3 (with free tiers)
**Deployment Time:** 3-5 minutes per deploy
