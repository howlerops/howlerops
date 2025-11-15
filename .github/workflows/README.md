# Howlerops GitHub Actions Workflows

This directory contains production-ready CI/CD workflows for Howlerops's Go backend. The workflows automate testing, building, releasing, and deploying the application.

## Overview

| Workflow | Purpose | Triggers |
|----------|---------|----------|
| **ci.yml** | Continuous Integration | PR, push to main |
| **release.yml** | Build & release binaries | Tag push (v*) |
| **deploy-backend.yml** | Deploy to Cloud Run | Release, manual |

## Workflows

### 1. CI Workflow (`ci.yml`)

Fast feedback for developers on every code change.

**Features:**
- Parallel job execution for speed
- Comprehensive test coverage (60% threshold)
- Race condition detection
- Multi-platform build verification
- Security scanning with gosec and govulncheck
- Code quality checks with golangci-lint

**Jobs:**
1. **test** - Run tests with coverage reporting
2. **lint** - Code quality and style checks
3. **build** - Multi-platform compilation verification
4. **security** - Vulnerability scanning

**Triggers:**
- Pull requests to main
- Pushes to main
- Manual workflow dispatch

**Typical Runtime:** 3-5 minutes

**Configuration:**
```yaml
env:
  GO_VERSION: '1.24'
  GOLANGCI_LINT_VERSION: 'v1.61'
  CGO_ENABLED: 1
```

### 2. Release Workflow (`release.yml`)

Automated creation of production releases with cross-platform binaries.

**Features:**
- Cross-platform builds (macOS, Linux, Windows)
- Multi-architecture support (amd64, arm64)
- Version embedding via ldflags
- SHA256 checksums for security
- Automated GitHub Release creation
- Docker image builds and scanning

**Build Matrix:**
```
darwin-amd64   (macOS Intel)
darwin-arm64   (macOS Apple Silicon)
linux-amd64    (Linux x86_64)
linux-arm64    (Linux ARM64)
windows-amd64  (Windows x86_64)
```

**Jobs:**
1. **create-release** - Create GitHub Release with changelog
2. **build-binaries** - Build cross-platform binaries
3. **generate-checksums** - Create combined checksum file
4. **build-docker** - Build and scan Docker images
5. **validate-release** - Verify all artifacts

**Triggers:**
- Git tag push matching `v*` pattern
- Manual workflow dispatch with custom version

**Usage:**
```bash
# Create and push a version tag
git tag v1.0.0
git push origin v1.0.0

# Release is created automatically with all artifacts
```

**Artifacts:**
- `sql-studio-{platform}-{arch}.tar.gz` - Binary archives
- `checksums.txt` - Combined SHA256 checksums
- Docker images in GCR with version tags

### 3. Deploy Backend Workflow (`deploy-backend.yml`)

Production deployment to GCP Cloud Run with zero-downtime strategy.

**Features:**
- Zero-downtime rolling updates
- Pre-deployment validation
- Secret management via GCP Secret Manager
- Comprehensive smoke tests
- Automatic rollback on failure
- Multi-platform deployment (Cloud Run, Fly.io)

**Jobs:**
1. **validate** - Pre-deployment checks and secret validation
2. **build-docker** - Build and push Docker image
3. **manage-secrets** - Create/update GCP secrets
4. **deploy-cloudrun** - Deploy to Cloud Run with health checks
5. **deploy-fly** - Optional Fly.io deployment
6. **deployment-summary** - Generate deployment report
7. **rollback-on-failure** - Automatic rollback safety net

**Triggers:**
- Release published (recommended)
- Manual workflow dispatch

**Zero-Downtime Strategy:**
1. Deploy new revision without traffic (`--no-traffic`)
2. Run smoke tests on new revision
3. Route traffic only after validation
4. Rollback automatically if tests fail

**Usage:**

**Option 1: Automated (via Release)**
```bash
# Create a release (triggers deploy-backend automatically)
gh release create v1.0.0 --generate-notes
```

**Option 2: Manual Deployment**
```
1. Go to Actions > Deploy Backend
2. Click "Run workflow"
3. Select environment (production/staging)
4. Select target (cloudrun/fly/both)
5. Click "Run workflow"
```

## Setup Instructions

### 1. Required GitHub Secrets

Configure these in: **Settings > Secrets and variables > Actions**

#### For All Workflows:
```
GITHUB_TOKEN - Automatically provided by GitHub
```

#### For Release Workflow:
```
GCP_PROJECT_ID - Google Cloud project ID
GCP_SA_KEY - Service account key JSON
```

#### For Deploy Workflow:
```
GCP_PROJECT_ID - Google Cloud project ID
GCP_SA_KEY - Service account key JSON
TURSO_URL - Turso database URL (e.g., libsql://db.turso.io)
TURSO_AUTH_TOKEN - Turso authentication token
JWT_SECRET - JWT signing secret (64+ chars recommended)
RESEND_API_KEY - Resend email API key (optional)
RESEND_FROM_EMAIL - Sender email (optional, e.g., noreply@sqlstudio.com)
FLY_API_TOKEN - Fly.io API token (optional)
```

#### Optional for CI:
```
CODECOV_TOKEN - Codecov token for private repos (optional)
```

### 2. GCP Service Account Setup

**Create Service Account:**
```bash
# Set your project ID
PROJECT_ID="your-project-id"

# Create service account
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions CI/CD" \
  --project="$PROJECT_ID"
```

**Grant Required Roles:**
```bash
# Service account email
SA_EMAIL="github-actions@${PROJECT_ID}.iam.gserviceaccount.com"

# Grant Cloud Run Admin
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/run.admin"

# Grant Service Account User
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/iam.serviceAccountUser"

# Grant Secret Manager Admin
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/secretmanager.admin"

# Grant Storage Admin (for Container Registry)
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/storage.admin"
```

**Create and Download Key:**
```bash
# Create key
gcloud iam service-accounts keys create github-actions-key.json \
  --iam-account="${SA_EMAIL}"

# Copy contents to GitHub secret GCP_SA_KEY
cat github-actions-key.json

# Securely delete local key
shred -u github-actions-key.json
```

### 3. Enable Required GCP APIs

```bash
gcloud services enable \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com \
  --project="$PROJECT_ID"
```

### 4. Turso Database Setup

**Create Database:**
```bash
# Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# Login
turso auth login

# Create database
turso db create sql-studio-prod

# Get database URL
turso db show sql-studio-prod --url

# Create auth token
turso db tokens create sql-studio-prod
```

**Add to GitHub Secrets:**
- `TURSO_URL`: Output from `--url` command
- `TURSO_AUTH_TOKEN`: Output from tokens create

### 5. Resend Email Setup (Optional)

**Create API Key:**
```
1. Go to https://resend.com/
2. Sign up / login
3. Create API key with "Sending access"
4. Verify your domain
```

**Add to GitHub Secrets:**
- `RESEND_API_KEY`: Your API key
- `RESEND_FROM_EMAIL`: Verified sender (e.g., noreply@yourdomain.com)

### 6. JWT Secret Generation

**Generate Strong Secret:**
```bash
# Generate 64-character random secret
openssl rand -base64 48

# Or use this one-liner
head -c 48 /dev/urandom | base64
```

**Add to GitHub Secrets:**
- `JWT_SECRET`: Generated secret (minimum 32 chars, recommend 64)

## Workflow Usage Examples

### Example 1: Release a New Version

```bash
# 1. Ensure all changes are committed and pushed
git add .
git commit -m "feat: new feature"
git push origin main

# 2. Create and push version tag
git tag v1.0.0
git push origin v1.0.0

# 3. Workflow automatically:
#    - Runs CI tests
#    - Builds binaries for all platforms
#    - Creates GitHub Release
#    - Builds Docker images
#    - Deploys to Cloud Run (if configured)

# 4. Check release
gh release view v1.0.0
```

### Example 2: Manual Deployment

```bash
# Deploy specific version to production
gh workflow run deploy-backend.yml \
  -f environment=production \
  -f deploy_target=cloudrun \
  -f image_tag=v1.0.0

# Deploy latest main to staging
gh workflow run deploy-backend.yml \
  -f environment=staging \
  -f deploy_target=cloudrun
```

### Example 3: Verify CI on Pull Request

```bash
# Create PR
git checkout -b feature/new-feature
git push origin feature/new-feature
gh pr create --title "New feature" --body "Description"

# CI automatically runs:
# - Tests with coverage
# - Linting
# - Build verification
# - Security scans

# View PR checks
gh pr checks

# Merge when all checks pass
gh pr merge --squash
```

## Monitoring and Troubleshooting

### View Workflow Runs

```bash
# List recent workflow runs
gh run list

# View specific run
gh run view <run-id>

# Watch live run
gh run watch
```

### Check Deployment Status

```bash
# Cloud Run service info
gcloud run services describe sql-studio-backend \
  --region=us-central1 \
  --format=yaml

# View logs
gcloud run services logs read sql-studio-backend \
  --region=us-central1 \
  --limit=100

# Check health
curl https://your-service.run.app/health
```

### Rollback Deployment

**Automatic Rollback:**
- Workflow automatically rolls back on deployment failure
- Previous revision receives 100% traffic
- Failed revision is kept for debugging

**Manual Rollback:**
```bash
# List revisions
gcloud run revisions list \
  --service=sql-studio-backend \
  --region=us-central1

# Route to specific revision
gcloud run services update-traffic sql-studio-backend \
  --region=us-central1 \
  --to-revisions=REVISION-NAME=100
```

### Common Issues

**1. Secret Not Found**
```
Error: Secret "turso-url" not found
```
**Solution:** Ensure secret is created in GCP Secret Manager and service account has access.

**2. Permission Denied**
```
Error: Permission denied on resource
```
**Solution:** Verify service account has all required roles (see setup section).

**3. Build Timeout**
```
Error: Build exceeded timeout
```
**Solution:** Increase timeout in workflow or optimize build (use cache).

**4. Health Check Failed**
```
Error: Health check failed after 30 attempts
```
**Solution:** Check application logs, verify environment variables, ensure database is accessible.

## Best Practices

### Version Tagging
- Use semantic versioning (v1.2.3)
- Tag format: `v{major}.{minor}.{patch}`
- Optional pre-release: `v1.0.0-beta.1`
- Never delete/move tags

### Deployment Strategy
- Always deploy via releases (not manual)
- Test in staging before production
- Use feature flags for risky changes
- Monitor metrics after deployment

### Secret Management
- Rotate secrets regularly (quarterly)
- Never commit secrets to git
- Use Secret Manager for all sensitive data
- Audit secret access regularly

### Testing
- Maintain >60% code coverage
- Fix race conditions immediately
- Run security scans before release
- Test binaries on all platforms

## Workflow Dependencies

### Actions Used
```yaml
actions/checkout@v4
actions/setup-go@v5
golangci/golangci-lint-action@v6
codecov/codecov-action@v4
docker/setup-buildx-action@v3
docker/build-push-action@v5
google-github-actions/auth@v2
google-github-actions/setup-gcloud@v2
aquasecurity/trivy-action@master
superfly/flyctl-actions/setup-flyctl@master
```

### External Services
- GitHub Actions (CI/CD)
- Google Cloud Platform (Cloud Run, Secret Manager, Container Registry)
- Turso (Database)
- Resend (Email)
- Codecov (Coverage reporting)
- Trivy (Security scanning)

## Cost Optimization

### GitHub Actions
- Uses GitHub-hosted runners (included in free tier)
- Caching reduces build times (faster = cheaper)
- Parallel jobs reduce total runtime

### GCP Cloud Run
- Scales to zero (pay only for requests)
- 2M requests/month free tier
- Estimated cost: $5-20/month for small apps

### Container Registry
- 0.5GB free storage
- Minimal egress costs
- Use image cleanup to reduce storage

## Support

### Documentation
- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Cloud Run Docs](https://cloud.google.com/run/docs)
- [Turso Docs](https://docs.turso.tech/)

### Getting Help
- Check workflow logs first
- Review this README
- Open issue with workflow run URL
- Include error messages and secrets status (redact values)

---

**Last Updated:** 2025-10-23
**Maintained By:** Howlerops Team
