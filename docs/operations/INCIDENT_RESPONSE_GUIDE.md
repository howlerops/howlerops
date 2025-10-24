# SQL Studio Incident Response Guide

## Table of Contents
1. [Severity Levels](#severity-levels)
2. [On-Call Procedures](#on-call-procedures)
3. [Incident Response Process](#incident-response-process)
4. [Communication](#communication)
5. [Escalation](#escalation)
6. [Post-Incident](#post-incident)

## Severity Levels

### SEV-1: Critical
**Impact:** Service completely down or major functionality unavailable
**Response Time:** Immediate (< 5 minutes)
**Examples:**
- Complete service outage
- Data loss or corruption
- Security breach
- Multiple critical systems failing

**Actions:**
- Page on-call engineer immediately
- Create incident channel in Slack (#incident-YYYYMMDD-NNN)
- Notify leadership within 15 minutes
- Update status page immediately
- All hands on deck until resolved

### SEV-2: High
**Impact:** Significant degradation affecting multiple users
**Response Time:** < 15 minutes
**Examples:**
- Database connection pool exhausted
- High error rates (>5%)
- Major feature unavailable
- Performance severely degraded

**Actions:**
- Alert on-call engineer
- Create incident channel
- Update status page
- Notify stakeholders within 30 minutes

### SEV-3: Medium
**Impact:** Minor degradation or single feature affected
**Response Time:** < 1 hour
**Examples:**
- Single service degraded
- Non-critical feature failing
- Elevated error rates (1-5%)

**Actions:**
- Create ticket
- Monitor for escalation
- Fix during business hours
- Update users if needed

### SEV-4: Low
**Impact:** Minor issues with workarounds available
**Response Time:** < 24 hours
**Examples:**
- UI glitches
- Performance slightly degraded
- Low-impact bugs

**Actions:**
- Create ticket
- Schedule fix in next sprint
- Document workaround

## On-Call Procedures

### On-Call Rotation
- **Primary On-Call:** First responder for all incidents
- **Secondary On-Call:** Backup if primary doesn't respond in 5 minutes
- **Rotation:** Weekly rotation, Monday 9am to Monday 9am

### On-Call Responsibilities
1. Acknowledge alerts within 5 minutes
2. Assess severity and escalate if needed
3. Lead incident response
4. Update stakeholders regularly
5. Write post-mortem after SEV-1/SEV-2 incidents

### Handoff Process
**Monday 9am:**
1. Outgoing on-call reviews open incidents
2. Shares any ongoing issues
3. Updates runbooks with any new learnings
4. Incoming on-call confirms receipt

### On-Call Tools Access
Ensure you have access to:
- [ ] PagerDuty
- [ ] Slack incident channels
- [ ] Kubernetes cluster (kubectl configured)
- [ ] Grafana dashboards
- [ ] Prometheus
- [ ] Jaeger tracing
- [ ] Cloud provider console
- [ ] VPN (if required)

## Incident Response Process

### Step 1: Acknowledge (< 5 minutes)
1. **Acknowledge alert in PagerDuty**
2. **Initial assessment:**
   - Is service down or degraded?
   - How many users affected?
   - Is data at risk?
3. **Determine severity** (use table above)
4. **Create incident channel:** `#incident-YYYYMMDD-NNN`

### Step 2: Triage (< 10 minutes)
1. **Check monitoring dashboards:**
   - Application Overview: Request rate, error rate, latency
   - Infrastructure: CPU, memory, pods status
   - Database: Connection pool, query performance

2. **Check recent changes:**
   ```bash
   # Check recent deployments
   kubectl rollout history deployment/sql-studio-backend -n sql-studio

   # Check recent pod events
   kubectl get events -n sql-studio --sort-by='.lastTimestamp'
   ```

3. **Check logs:**
   ```bash
   # View recent errors
   kubectl logs -n sql-studio -l app=sql-studio-backend --tail=100 | grep ERROR

   # Or use log aggregation
   # Elasticsearch/Kibana or Loki/Grafana
   ```

### Step 3: Communicate (< 15 minutes for SEV-1/SEV-2)
1. **Update incident channel:**
   ```
   INCIDENT: [SEV-X] Brief description
   STATUS: Investigating
   IMPACT: X users affected, Y functionality down
   NEXT UPDATE: In 15 minutes
   ```

2. **Update status page** (for user-facing issues)
   - Go to status.sqlstudio.io admin
   - Create incident
   - Post initial update

3. **Notify stakeholders:**
   - SEV-1: CEO, CTO, VP Engineering (immediately)
   - SEV-2: Engineering Manager, Product Manager (within 30 min)
   - SEV-3: Engineering Manager (within 1 hour)

### Step 4: Mitigate
**Priority: Stop the bleeding**

Common mitigation actions:
- Rollback recent deployment
- Scale up resources
- Restart failing pods
- Enable feature flags to disable failing features
- Redirect traffic to backup systems

**DO NOT:**
- Make changes without documenting them
- Skip telling the team what you're doing
- Delete data without backup
- Restart services randomly without understanding root cause

### Step 5: Resolve
1. **Implement fix**
2. **Verify resolution:**
   - Error rates back to normal
   - Latency acceptable
   - All health checks passing
   - User verification (if possible)

3. **Monitor for 15-30 minutes** before declaring resolved

### Step 6: Close
1. **Update status page:** "Resolved"
2. **Update incident channel:**
   ```
   RESOLVED: Brief description of fix
   ROOT CAUSE: One-line summary
   DURATION: Start time - End time
   POST-MORTEM: Will be posted by [date]
   ```

3. **Close PagerDuty incident**
4. **Thank the team**

## Communication

### Update Frequency
- **SEV-1:** Every 15 minutes minimum
- **SEV-2:** Every 30 minutes minimum
- **SEV-3:** Every hour or when status changes

### Update Template
```
UPDATE [HH:MM UTC]:
STATUS: [Investigating/Identified/Monitoring/Resolved]
WHAT WE KNOW: Brief summary
WHAT WE'RE DOING: Current actions
IMPACT: Users/features affected
NEXT UPDATE: Timeframe
```

### Slack Incident Channel
Create for SEV-1 and SEV-2:
```
/incident create sev-1 "Brief description"
```

Channel topic should include:
- Severity
- Impact summary
- Incident commander
- Status page link

### Status Page Updates
**When to update:**
- All user-facing SEV-1 and SEV-2 incidents
- SEV-3 if users are reporting issues

**What to include:**
- What's affected
- What we're doing
- When we expect resolution (if known)
- Workarounds (if available)

**What NOT to include:**
- Technical implementation details
- Blame or finger-pointing
- Speculation about causes

## Escalation

### When to Escalate
- You don't know how to fix it within 15 minutes
- Issue is getting worse despite mitigation
- Multiple systems failing
- Data integrity concerns
- Security implications
- Need additional access/permissions

### How to Escalate

**To Secondary On-Call:**
1. Page in PagerDuty
2. Call them directly
3. Brief them in incident channel

**To Engineering Manager:**
- Slack DM + @mention in incident channel
- If no response in 5 min, call

**To Leadership:**
- For SEV-1, notify immediately
- Use emergency contact list

### Subject Matter Experts

| Area | Primary | Secondary | When to Page |
|------|---------|-----------|--------------|
| Database | @db-team | @senior-backend | Connection issues, slow queries, data corruption |
| Infrastructure | @devops-team | @platform-lead | Kubernetes, networking, cloud provider |
| Security | @security-team | @cto | Breaches, authentication issues, suspicious activity |
| Frontend | @frontend-team | @frontend-lead | UI issues, client-side errors |
| Backend | @backend-team | @backend-lead | API issues, business logic bugs |

## Post-Incident

### Post-Mortem Required For
- All SEV-1 incidents
- All SEV-2 incidents
- Any incident with customer impact
- Repeated SEV-3 incidents (pattern)

### Post-Mortem Timeline
- **24 hours:** Draft post-mortem document
- **48 hours:** Review with team
- **72 hours:** Publish internally
- **1 week:** Complete all action items or schedule them

### Post-Mortem Template

```markdown
# Post-Mortem: [Incident Title]

## Incident Summary
- **Date:** YYYY-MM-DD
- **Duration:** HH:MM
- **Severity:** SEV-X
- **Incident Commander:** Name
- **Responders:** Names

## Impact
- **Users Affected:** Number or percentage
- **Revenue Impact:** $ (if applicable)
- **Functionality Impacted:** Description

## Timeline (all times UTC)
- HH:MM - First alert fired
- HH:MM - On-call acknowledged
- HH:MM - Root cause identified
- HH:MM - Fix deployed
- HH:MM - Incident resolved

## Root Cause
Detailed description of what went wrong and why.

## Resolution
What we did to fix it.

## What Went Well
- Quick detection
- Clear communication
- Fast resolution
- etc.

## What Went Wrong
- Late detection
- Unclear runbooks
- Slow deployment
- etc.

## Action Items
- [ ] Item 1 - Owner - Due Date
- [ ] Item 2 - Owner - Due Date
- [ ] Item 3 - Owner - Due Date

## Lessons Learned
Key takeaways to prevent recurrence.
```

### Post-Mortem Review Meeting
- **When:** Within 48 hours of resolution
- **Who:** Incident responders, engineering manager, relevant stakeholders
- **Duration:** 30-60 minutes
- **Goal:** Blameless review, identify improvements

### Follow-up
- Track action items in project management tool
- Review in weekly team meetings
- Update runbooks and documentation
- Share learnings in engineering all-hands

## Quick Reference

### Emergency Contacts
```
Primary On-Call: PagerDuty
Secondary On-Call: PagerDuty
Engineering Manager: [Phone]
CTO: [Phone]
Security Team: security@sqlstudio.io
```

### Key Links
- Status Page: https://status.sqlstudio.io
- Grafana: https://grafana.sqlstudio.io
- Prometheus: https://prometheus.sqlstudio.io
- Jaeger: https://jaeger.sqlstudio.io
- Runbooks: https://docs.sqlstudio.io/runbooks
- Kubernetes Dashboard: https://k8s.sqlstudio.io

### Common Commands
```bash
# Get pod status
kubectl get pods -n sql-studio

# View logs
kubectl logs -n sql-studio -l app=sql-studio-backend --tail=100 -f

# Describe pod
kubectl describe pod POD_NAME -n sql-studio

# Rollback deployment
kubectl rollout undo deployment/sql-studio-backend -n sql-studio

# Scale deployment
kubectl scale deployment/sql-studio-backend --replicas=5 -n sql-studio

# Check events
kubectl get events -n sql-studio --sort-by='.lastTimestamp'
```

## Remember

1. **Stay Calm:** Panic doesn't help
2. **Communicate:** Over-communicate rather than under-communicate
3. **Document:** Everything you do and observe
4. **Ask for Help:** Don't be a hero, escalate when needed
5. **Blameless:** Focus on systems, not people
6. **Learn:** Every incident is an opportunity to improve
