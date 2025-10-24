# SQL Studio On-Call Guide

## Welcome to On-Call!

This guide will help you prepare for and handle your on-call shift effectively.

## Pre-Shift Checklist

### Access Verification
Before your shift starts, verify you have access to:

- [ ] **PagerDuty** - Can you receive and acknowledge alerts?
- [ ] **Slack** - Joined #incidents and #alerts channels
- [ ] **Kubernetes Cluster** - Can you run `kubectl get pods -n sql-studio`?
- [ ] **Grafana** - Can you view dashboards at https://grafana.sqlstudio.io?
- [ ] **Prometheus** - Can you access https://prometheus.sqlstudio.io?
- [ ] **Jaeger** - Can you view traces at https://jaeger.sqlstudio.io?
- [ ] **AWS Console** (or relevant cloud provider)
- [ ] **GitHub** - Can you create emergency PRs?
- [ ] **VPN** - Connected and working
- [ ] **Status Page Admin** - Can you post updates at status.sqlstudio.io?

### Tools Setup
Ensure these tools are installed and configured:

```bash
# Kubernetes CLI
kubectl version

# AWS CLI (if using AWS)
aws sts get-caller-identity

# Helm (for deployments)
helm version

# jq (for JSON parsing)
jq --version

# curl (for API testing)
curl --version
```

### Knowledge Review
Familiarize yourself with:
- [ ] Recent deployments (last 7 days)
- [ ] Open production issues
- [ ] Current system status
- [ ] Recent post-mortems
- [ ] Common issues in runbook

## Daily Routine

### Morning
- [ ] Check PagerDuty for any alerts from night
- [ ] Review monitoring dashboards
- [ ] Check Slack for any reports
- [ ] Review system health metrics

### Evening
- [ ] Ensure laptop is charged
- [ ] Phone volume up and working
- [ ] Internet connection stable
- [ ] No planned activities that prevent response

## Monitoring Dashboards

### Primary Dashboard: Application Overview
**URL:** https://grafana.sqlstudio.io/d/sql-studio-app-overview

**Key Metrics to Watch:**
- Request Rate (normal: 100-500 req/s)
- Error Rate (normal: <0.1%)
- P95 Latency (normal: <200ms)
- Active Users (varies by time of day)

**When to be concerned:**
- Error rate >1%
- P95 latency >500ms
- Sudden drop in request rate (possible outage)
- Spike in request rate (possible attack)

### Infrastructure Dashboard
**URL:** https://grafana.sqlstudio.io/d/sql-studio-infrastructure

**Key Metrics:**
- CPU Usage (warn at 80%, critical at 95%)
- Memory Usage (warn at 80%, critical at 90%)
- Pod Count (should match desired replicas)
- Node Status (all should be Ready)

### Database Dashboard
**URL:** https://grafana.sqlstudio.io/d/sql-studio-database

**Key Metrics:**
- Connection Pool Utilization (warn at 80%)
- Query Duration P95 (normal: <100ms)
- Query Error Rate (normal: <0.01%)

## Alert Response SLA

| Severity | Acknowledgment | Initial Response | Resolution Target |
|----------|---------------|------------------|-------------------|
| SEV-1    | 5 minutes     | 5 minutes        | 1 hour            |
| SEV-2    | 15 minutes    | 15 minutes       | 4 hours           |
| SEV-3    | 1 hour        | 1 hour           | 1 business day    |
| SEV-4    | 4 hours       | 4 hours          | 1 week            |

## Common Alert Responses

### HighErrorRate
**First Steps:**
1. Acknowledge in PagerDuty
2. Check Application Overview dashboard
3. Identify which endpoint is failing
4. Check recent deployments

**Quick Fix:**
```bash
# If recent deployment, rollback
kubectl rollout undo deployment/sql-studio-backend -n sql-studio

# Watch for recovery
kubectl rollout status deployment/sql-studio-backend -n sql-studio
```

**See:** [Runbook - High Error Rate](./RUNBOOK.md#high-error-rate-5)

### HighAPILatency
**First Steps:**
1. Check which endpoint is slow
2. Look at traces in Jaeger for slow requests
3. Check database query performance

**Quick Fix:**
```bash
# Scale horizontally
kubectl scale deployment/sql-studio-backend --replicas=6 -n sql-studio
```

**See:** [Runbook - High API Latency](./RUNBOOK.md#high-api-latency-p95-500ms)

### PodCrashLooping
**First Steps:**
1. Check pod status
   ```bash
   kubectl get pods -n sql-studio
   ```
2. View pod logs (including previous crash)
   ```bash
   kubectl logs POD_NAME -n sql-studio --previous
   ```
3. Check pod events
   ```bash
   kubectl describe pod POD_NAME -n sql-studio
   ```

**Common Causes:**
- OOM Kill → Increase memory limits
- Failed health check → Fix health endpoint
- Application panic → Fix bug and deploy

**See:** [Runbook - Pod Crash Looping](./RUNBOOK.md#pod-crash-looping)

### DatabaseConnectionPoolExhaustion
**First Steps:**
1. Check connection pool metrics
2. Look for connection leaks in logs

**Quick Fix:**
```bash
# Increase connection pool size
kubectl set env deployment/sql-studio-backend -n sql-studio \
  DB_MAX_CONNECTIONS=50

# Or restart to clear leaked connections
kubectl rollout restart deployment/sql-studio-backend -n sql-studio
```

**See:** [Runbook - Connection Pool](./RUNBOOK.md#database-connection-pool-exhaustion)

## Escalation Guide

### When to Escalate
- You don't know how to fix within 15 minutes
- Issue is getting worse despite mitigation
- Data integrity concerns
- Security implications
- Need permissions you don't have

### How to Escalate

**1. Secondary On-Call**
```
PagerDuty → Escalate to Secondary On-Call
```

**2. Engineering Manager**
```
Slack: @engineering-manager in #incident channel
If no response in 5 min → Call directly
```

**3. Senior Engineer (by specialty)**

| Issue Type | Contact | When |
|------------|---------|------|
| Database | @database-lead | DB performance, corruption |
| Security | @security-team | Breaches, auth issues |
| Infrastructure | @devops-lead | K8s, cloud provider |
| Backend | @backend-lead | API bugs, logic errors |

**4. Leadership (SEV-1 only)**
- CTO: Immediately for SEV-1
- CEO: If customer impact >1000 users

## Communication Templates

### Initial Alert Response (in #incident channel)
```
🚨 INCIDENT ALERT - SEV-X

WHAT: Brief description (1 sentence)
IMPACT: Number of users / features affected
STATUS: Investigating
INCIDENT COMMANDER: @your-name
NEXT UPDATE: In 15 minutes

Dashboard: [link to relevant Grafana dashboard]
```

### Status Update
```
⏱️ UPDATE [HH:MM UTC]

STATUS: Investigating / Identified / Fixing / Monitoring
PROGRESS: What we've learned/done since last update
CURRENT ACTION: What we're doing now
NEXT UPDATE: In X minutes
```

### Resolution Message
```
✅ RESOLVED [HH:MM UTC]

ISSUE: What was the problem
FIX: What we did to resolve it
IMPACT: Final count of affected users/duration
MONITORING: Watching for 30 minutes to ensure stability
POST-MORTEM: Will be posted by [date]

Thank you to everyone who helped! 🙏
```

## kubectl Cheat Sheet

```bash
# Get all resources in namespace
kubectl get all -n sql-studio

# Get pods with label
kubectl get pods -n sql-studio -l app=sql-studio-backend

# Watch pods (auto-refresh)
kubectl get pods -n sql-studio -w

# Stream logs
kubectl logs -f POD_NAME -n sql-studio

# Get logs from all pods with label
kubectl logs -n sql-studio -l app=sql-studio-backend --tail=100

# Get logs from previous pod instance (if crashed)
kubectl logs POD_NAME -n sql-studio --previous

# Execute command in pod
kubectl exec -it POD_NAME -n sql-studio -- /bin/bash

# Port forward to pod
kubectl port-forward POD_NAME 8080:8080 -n sql-studio

# Get pod details and events
kubectl describe pod POD_NAME -n sql-studio

# Get deployment status
kubectl rollout status deployment/sql-studio-backend -n sql-studio

# View deployment history
kubectl rollout history deployment/sql-studio-backend -n sql-studio

# Rollback to previous version
kubectl rollout undo deployment/sql-studio-backend -n sql-studio

# Rollback to specific revision
kubectl rollout undo deployment/sql-studio-backend --to-revision=3 -n sql-studio

# Scale deployment
kubectl scale deployment/sql-studio-backend --replicas=5 -n sql-studio

# Restart deployment (rolling restart)
kubectl rollout restart deployment/sql-studio-backend -n sql-studio

# Update environment variable
kubectl set env deployment/sql-studio-backend -n sql-studio VAR_NAME=value

# Delete pod (will be recreated)
kubectl delete pod POD_NAME -n sql-studio

# Get events sorted by time
kubectl get events -n sql-studio --sort-by='.lastTimestamp'

# Get resource usage
kubectl top pods -n sql-studio
kubectl top nodes

# Copy file from pod
kubectl cp sql-studio/POD_NAME:/path/to/file ./local-file

# Run temporary pod for debugging
kubectl run -it --rm debug --image=alpine -n sql-studio -- sh
```

## PromQL Quick Reference

```promql
# Request rate
sum(rate(sql_studio_http_requests_total[5m]))

# Error rate
sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) /
sum(rate(sql_studio_http_requests_total[5m]))

# P95 latency
histogram_quantile(0.95,
  sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (le)
)

# Top errors by endpoint
topk(10,
  sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) by (endpoint)
)

# Memory usage percentage
container_memory_working_set_bytes{namespace="sql-studio"} /
container_spec_memory_limit_bytes{namespace="sql-studio"}

# CPU usage percentage
rate(container_cpu_usage_seconds_total{namespace="sql-studio"}[5m]) /
container_spec_cpu_quota{namespace="sql-studio"} * 100000

# Connection pool utilization
sql_studio_database_connections_active / sql_studio_database_connections_max
```

## Useful URLs

**Monitoring:**
- Grafana: https://grafana.sqlstudio.io
- Prometheus: https://prometheus.sqlstudio.io
- AlertManager: https://alertmanager.sqlstudio.io
- Jaeger: https://jaeger.sqlstudio.io

**Operations:**
- Status Page: https://status.sqlstudio.io
- Status Page Admin: https://status.sqlstudio.io/admin
- Kubernetes Dashboard: https://k8s.sqlstudio.io

**Documentation:**
- Runbooks: https://docs.sqlstudio.io/operations/runbook
- Incident Response: https://docs.sqlstudio.io/operations/incident-response
- API Docs: https://docs.sqlstudio.io/api

**Tools:**
- PagerDuty: https://sqlstudio.pagerduty.com
- GitHub: https://github.com/sql-studio/backend-go

## Tips for Success

### Before Incidents
1. **Be Prepared**
   - Review this guide regularly
   - Practice common commands
   - Know where to find information

2. **Stay Current**
   - Read recent post-mortems
   - Understand recent changes
   - Know the system architecture

3. **Test Your Setup**
   - Verify all access works
   - Test alert notifications
   - Practice deployments in staging

### During Incidents
1. **Stay Calm**
   - Breathe
   - You have help available
   - Follow the runbook

2. **Communicate**
   - Update every 15-30 minutes minimum
   - Be transparent about what you know/don't know
   - Ask for help early

3. **Document**
   - Write down what you try
   - Screenshot errors
   - Save relevant logs

4. **Don't Guess**
   - Check metrics before making changes
   - Understand root cause before "fixing"
   - Rollback is often better than debugging live

### After Incidents
1. **Document**
   - Write the post-mortem promptly
   - Include timeline and actions
   - Note what helped and what didn't

2. **Improve**
   - Update runbooks
   - Add monitoring if gaps found
   - Fix root cause, not just symptoms

3. **Reflect**
   - What went well?
   - What could be better?
   - What did you learn?

## Self-Care

On-call can be stressful. Remember to:
- Get enough sleep when not being paged
- Take breaks during your shift
- Ask for help when needed
- Decompress after difficult incidents
- Talk to your manager if on-call is affecting your health

## Handoff Checklist

**Monday morning at 9am:**

### Outgoing On-Call
- [ ] Share summary of any incidents
- [ ] Highlight ongoing issues
- [ ] Mention any unusual patterns observed
- [ ] Share any runbook improvements needed
- [ ] Answer incoming on-call's questions

### Incoming On-Call
- [ ] Review outgoing's summary
- [ ] Check current system status
- [ ] Review any open incidents
- [ ] Confirm you can receive alerts
- [ ] Ask questions about anything unclear

## Emergency Contacts

```
Primary On-Call: Check PagerDuty
Secondary On-Call: Check PagerDuty
Engineering Manager: [Phone]
DevOps Lead: [Phone]
Database Lead: [Phone]
Security Team: security@sqlstudio.io
CTO: [Phone - SEV-1 only]
```

## Remember

- **You're not alone** - Help is always available
- **It's okay to not know** - Ask questions
- **Better safe than sorry** - Escalate when in doubt
- **Communication is key** - Over-communicate
- **Learn from everything** - Every incident teaches us something

Good luck with your shift! 🚀
