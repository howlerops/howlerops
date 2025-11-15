# Howlerops Backend - Production Deployment Checklist

Use this checklist to ensure all prerequisites are met before deploying to production.

## Pre-Deployment Requirements

### GCP Setup

- [ ] **GCP Account Created**
  - Account verified with valid payment method
  - Billing alerts configured
  - Organization/folder structure set up (if applicable)

- [ ] **GCP Project Created**
  ```bash
  gcloud projects create PROJECT_ID --name="Howlerops Production"
  ```

- [ ] **Billing Enabled**
  ```bash
  gcloud billing projects link PROJECT_ID --billing-account=BILLING_ACCOUNT_ID
  ```

- [ ] **Required APIs Enabled**
  ```bash
  gcloud services enable \
    run.googleapis.com \
    secretmanager.googleapis.com \
    containerregistry.googleapis.com \
    cloudbuild.googleapis.com \
    logging.googleapis.com \
    monitoring.googleapis.com \
    --project=PROJECT_ID
  ```

- [ ] **Service Account Created**
  ```bash
  gcloud iam service-accounts create sql-studio-backend \
    --display-name="Howlerops Backend Service"
  ```

- [ ] **IAM Permissions Configured**
  - [ ] Secret Manager Secret Accessor
  - [ ] Cloud Run Invoker (if needed)
  - [ ] Cloud Logging Writer
  - [ ] Cloud Monitoring Metric Writer

### Database Setup

- [ ] **Turso Account Created**
  - Account verified at [turso.tech](https://turso.tech)
  - Billing configured if needed

- [ ] **Production Database Created**
  ```bash
  turso db create sql-studio-production --region [closest-region]
  ```

- [ ] **Database Schema Applied**
  ```bash
  turso db shell sql-studio-production < schema.sql
  ```

- [ ] **Database URL Retrieved**
  ```bash
  turso db show sql-studio-production --url
  ```

- [ ] **Auth Token Generated**
  ```bash
  turso db tokens create sql-studio-production --expiration never
  ```

- [ ] **Backup Strategy Configured**
  - [ ] Automatic backups enabled
  - [ ] Backup retention period set
  - [ ] Restore procedure documented

### Security Configuration

- [ ] **JWT Secret Generated (64+ characters)**
  ```bash
  openssl rand -base64 64
  ```
  - Stored securely
  - Not committed to Git
  - Added to Secret Manager

- [ ] **API Keys Secured**
  - [ ] Resend API key obtained
  - [ ] All keys stored in Secret Manager
  - [ ] Keys not hardcoded anywhere

- [ ] **CORS Configuration**
  - [ ] Allowed origins defined
  - [ ] Credentials handling configured
  - [ ] Methods and headers specified

- [ ] **Rate Limiting Configured**
  - [ ] Limits defined per endpoint
  - [ ] DDoS protection enabled (Cloud Armor)

### Email Service

- [ ] **Resend Account Setup** (optional but recommended)
  - Account created at [resend.com](https://resend.com)
  - Payment method added if needed

- [ ] **API Key Generated**
  - Key created with appropriate permissions
  - Key stored in Secret Manager

- [ ] **Domain Verified**
  - [ ] SPF records added
  - [ ] DKIM configured
  - [ ] Domain verification completed

- [ ] **From Email Configured**
  - Sender email address set
  - Reply-to address configured

### GitHub Configuration

- [ ] **Repository Secrets Added**
  ```
  GCP_PROJECT_ID
  GCP_SA_KEY (service account JSON)
  TURSO_URL
  TURSO_AUTH_TOKEN
  JWT_SECRET
  RESEND_API_KEY
  RESEND_FROM_EMAIL
  ```

- [ ] **GitHub Actions Enabled**
  - Workflows reviewed and tested
  - Branch protection rules configured
  - Required status checks enabled

### Local Development Environment

- [ ] **Required Tools Installed**
  - [ ] gcloud CLI
  - [ ] Docker
  - [ ] Go 1.24+
  - [ ] git
  - [ ] make
  - [ ] jq (optional)
  - [ ] curl

- [ ] **Authentication Configured**
  ```bash
  gcloud auth login
  gcloud config set project PROJECT_ID
  ```

- [ ] **Docker Configured**
  ```bash
  gcloud auth configure-docker gcr.io
  ```

### Code Preparation

- [ ] **Code Review Completed**
  - All PRs reviewed and approved
  - No pending security issues
  - Performance bottlenecks addressed

- [ ] **Tests Passing**
  ```bash
  go test ./... -race -cover
  ```

- [ ] **Linting Passed**
  ```bash
  golangci-lint run
  ```

- [ ] **Dependencies Updated**
  ```bash
  go mod tidy
  go list -u -m all
  ```

- [ ] **Build Successful**
  ```bash
  docker build -t sql-studio-backend .
  ```

### Domain and SSL (if using custom domain)

- [ ] **Domain Registered**
  - Domain ownership verified
  - DNS management access available

- [ ] **DNS Provider Configured**
  - Access to DNS settings
  - CAA records configured (optional)

- [ ] **SSL Certificate Strategy**
  - [ ] Using Cloud Run managed certificates (recommended)
  - [ ] Or custom certificates uploaded

### Monitoring and Alerts

- [ ] **Logging Strategy Defined**
  - Log retention period set
  - Log-based metrics created

- [ ] **Alert Policies Created**
  - [ ] High error rate alert
  - [ ] High latency alert
  - [ ] Service down alert
  - [ ] Budget alert

- [ ] **Notification Channels Configured**
  - [ ] Email notifications
  - [ ] Slack integration (optional)
  - [ ] PagerDuty (optional)

- [ ] **Dashboard Created**
  - Key metrics identified
  - Custom dashboard configured
  - SLI/SLO defined

### Backup and Recovery

- [ ] **Database Backup Plan**
  - Automated backups scheduled
  - Manual backup procedure documented
  - Restore procedure tested

- [ ] **Code Backup**
  - Git repository backed up
  - CI/CD pipeline configuration backed up
  - Secrets backup strategy defined

- [ ] **Rollback Plan Documented**
  - Rollback procedure written
  - Previous versions retained
  - Rollback tested in staging

### Documentation

- [ ] **API Documentation Updated**
  - Endpoints documented
  - Authentication described
  - Rate limits specified

- [ ] **Runbooks Created**
  - [ ] Deployment runbook
  - [ ] Rollback runbook
  - [ ] Incident response runbook
  - [ ] Secret rotation runbook

- [ ] **Team Training Completed**
  - Deployment process understood
  - Monitoring tools familiar
  - Emergency procedures known

### Legal and Compliance

- [ ] **Privacy Policy Updated**
  - Data handling described
  - Third-party services listed
  - User rights explained

- [ ] **Terms of Service Updated**
  - Service limitations defined
  - Liability limitations stated
  - Governing law specified

- [ ] **GDPR Compliance** (if applicable)
  - [ ] Data processing agreements signed
  - [ ] Right to deletion implemented
  - [ ] Data export functionality

### Performance Testing

- [ ] **Load Testing Completed**
  ```bash
  # Example with k6
  k6 run load-test.js
  ```

- [ ] **Stress Testing Done**
  - Maximum capacity identified
  - Breaking points documented
  - Recovery behavior verified

- [ ] **Latency Benchmarks**
  - P50, P95, P99 latencies measured
  - Acceptable thresholds defined
  - Cold start time measured

## Deployment Day Checklist

### Pre-Deployment (Day Before)

- [ ] Final code freeze
- [ ] All tests passing
- [ ] Team notified of deployment window
- [ ] Rollback plan reviewed
- [ ] Monitoring dashboards ready
- [ ] Communication channels open

### Deployment Steps

1. [ ] **Run Production Readiness Check**
   ```bash
   ./scripts/prod-readiness-check.sh
   ```

2. [ ] **Create Git Tag**
   ```bash
   git tag -a v1.0.0 -m "Production release v1.0.0"
   git push origin v1.0.0
   ```

3. [ ] **Trigger Deployment**
   ```bash
   # Via GitHub Actions (recommended)
   gh release create v1.0.0 --generate-notes

   # Or manually
   ./scripts/deploy-gcp.sh --project PROJECT_ID --use-cloudbuild
   ```

4. [ ] **Monitor Deployment**
   - Watch GitHub Actions progress
   - Monitor Cloud Build logs
   - Check Cloud Run console

5. [ ] **Verify Deployment**
   ```bash
   ./scripts/verify-deployment.sh SERVICE_URL
   ```

6. [ ] **Smoke Tests**
   - [ ] Health endpoint responding
   - [ ] Authentication working
   - [ ] Database queries successful
   - [ ] Email sending (if applicable)

### Post-Deployment

- [ ] **Monitor for 30 minutes**
  - Check error rates
  - Monitor latency
  - Watch for anomalies

- [ ] **Update Status Page**
  - Mark service as operational
  - Note any known issues

- [ ] **Send Deployment Report**
  - Deployment successful/failed
  - Version deployed
  - Any issues encountered
  - Next steps

- [ ] **Update Documentation**
  - Update version numbers
  - Document any configuration changes
  - Update runbooks if needed

## Rollback Criteria

Initiate rollback if:

- [ ] Error rate > 5% for 5 minutes
- [ ] P99 latency > 3 seconds for 5 minutes
- [ ] Health checks failing consistently
- [ ] Database connection errors
- [ ] Authentication completely broken
- [ ] Data corruption detected

## Emergency Contacts

| Role | Name | Contact | When to Contact |
|------|------|---------|----------------|
| Lead Engineer | [Name] | [Email/Phone] | Deployment issues, rollback decisions |
| Database Admin | [Name] | [Email/Phone] | Database issues, data corruption |
| Security Lead | [Name] | [Email/Phone] | Security incidents, auth issues |
| Product Owner | [Name] | [Email/Phone] | Major outages, user impact |
| On-Call SRE | [Rotation] | [PagerDuty] | After-hours emergencies |

## Quick Reference

### Useful Commands

```bash
# Check service status
gcloud run services describe sql-studio-backend --region=us-central1

# View recent logs
gcloud logging read "resource.type=cloud_run_revision" --limit=50

# Quick rollback
gcloud run services update-traffic sql-studio-backend --to-revisions=LATEST=0

# Check database status
turso db show sql-studio-production

# Monitor in real-time
gcloud alpha run services logs tail sql-studio-backend --region=us-central1
```

### Important URLs

- Cloud Run Console: https://console.cloud.google.com/run
- Cloud Logging: https://console.cloud.google.com/logs
- Cloud Monitoring: https://console.cloud.google.com/monitoring
- Secret Manager: https://console.cloud.google.com/security/secret-manager
- Turso Dashboard: https://turso.tech/dashboard
- GitHub Actions: https://github.com/[org]/sql-studio/actions

## Sign-off

- [ ] Technical Lead Approval
- [ ] Security Review Completed
- [ ] Product Owner Approval
- [ ] Deployment Scheduled

---

**Last Updated:** [Date]
**Next Review:** [Date + 3 months]