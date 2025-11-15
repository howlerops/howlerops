# Howlerops Operational Runbook

## Table of Contents
1. [Common Issues](#common-issues)
2. [Service Recovery](#service-recovery)
3. [Performance Troubleshooting](#performance-troubleshooting)
4. [Database Issues](#database-issues)
5. [Authentication Issues](#authentication-issues)
6. [Deployment Issues](#deployment-issues)

## Common Issues

### High Error Rate (>5%)

**Alert:** `HighErrorRate`
**Severity:** Critical
**Symptoms:**
- Error rate gauge showing >5%
- Increased 5xx responses in logs
- Users reporting failures

**Diagnosis:**
```bash
# Check error distribution by endpoint
kubectl logs -n sql-studio -l app=sql-studio-backend | grep "status\":5" | tail -100

# View Grafana dashboard
# Navigate to: Application Overview → Error Rate by Endpoint

# Check Prometheus for error breakdown
# Query: sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) by (endpoint, status)
```

**Resolution:**
1. **Identify failing endpoint(s)**
   - Check which endpoints have highest error rates
   - Look for recent code changes to those endpoints

2. **Check recent deployments**
   ```bash
   kubectl rollout history deployment/sql-studio-backend -n sql-studio
   ```

3. **Rollback if recent deployment**
   ```bash
   kubectl rollout undo deployment/sql-studio-backend -n sql-studio
   ```

4. **Check dependencies**
   - Database connectivity
   - External API availability
   - Redis/cache availability

5. **Monitor recovery**
   - Error rate should drop within 2-3 minutes
   - Check Grafana dashboard

---

### High API Latency (P95 >500ms)

**Alert:** `HighAPILatency`
**Severity:** Warning
**Symptoms:**
- Slow response times
- Users reporting sluggish UI
- Increased request duration

**Diagnosis:**
```bash
# Check slow endpoints
# Prometheus query: topk(10, histogram_quantile(0.95, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (endpoint, le)))

# Check database query performance
# Query: histogram_quantile(0.95, sum(rate(sql_studio_database_query_duration_seconds_bucket[5m])) by (operation, le))

# View traces in Jaeger
# https://jaeger.sqlstudio.io
# Search for slow requests (duration > 500ms)
```

**Resolution:**
1. **Identify slow endpoint**
   - Check Grafana "Top 10 Slowest Endpoints" panel

2. **Check for slow database queries**
   ```bash
   # View slow query logs
   kubectl logs -n sql-studio -l app=sql-studio-backend | grep "duration_ms\":[0-9]\{4,\}"
   ```

3. **Scale horizontally if needed**
   ```bash
   # Increase replicas
   kubectl scale deployment/sql-studio-backend --replicas=6 -n sql-studio
   ```

4. **Check resource utilization**
   - CPU/Memory at limits?
   - Network I/O saturated?

5. **Investigate specific slow query**
   - Missing index?
   - Full table scan?
   - N+1 query problem?

**Prevention:**
- Add database indexes
- Implement caching
- Optimize queries
- Add pagination

---

### Pod Crash Looping

**Alert:** `PodCrashLooping`
**Severity:** Critical
**Symptoms:**
- Pods constantly restarting
- CrashLoopBackOff status
- Service degradation

**Diagnosis:**
```bash
# Check pod status
kubectl get pods -n sql-studio -l app=sql-studio-backend

# View pod events
kubectl describe pod POD_NAME -n sql-studio

# Check logs (including previous crashes)
kubectl logs POD_NAME -n sql-studio --previous
```

**Common Causes:**

1. **OOM (Out of Memory)**
   ```bash
   # Check for OOMKilled in pod events
   kubectl describe pod POD_NAME -n sql-studio | grep -A 5 "State:"

   # Solution: Increase memory limits
   kubectl edit deployment sql-studio-backend -n sql-studio
   # Update resources.limits.memory
   ```

2. **Failed Health Checks**
   ```bash
   # Check liveness/readiness probe failures
   kubectl describe pod POD_NAME -n sql-studio | grep -A 10 "Liveness:"

   # Solution: Fix health check endpoint or increase timeout
   ```

3. **Application Panic/Crash**
   ```bash
   # View panic stack trace
   kubectl logs POD_NAME -n sql-studio --previous | grep -A 50 "panic:"

   # Solution: Fix bug causing panic, deploy hotfix
   ```

4. **Missing Configuration**
   ```bash
   # Check if env vars/secrets are present
   kubectl exec POD_NAME -n sql-studio -- env | grep SQL_STUDIO

   # Solution: Update ConfigMap/Secret
   kubectl edit configmap sql-studio-config -n sql-studio
   ```

**Resolution:**
1. Identify root cause from logs
2. Fix configuration or code issue
3. If critical, rollback deployment
4. Monitor pod stability

---

### Database Connection Pool Exhaustion

**Alert:** `DatabaseConnectionPoolExhaustion`
**Severity:** Critical
**Symptoms:**
- Errors: "too many connections"
- High latency on database operations
- Timeouts acquiring connections

**Diagnosis:**
```bash
# Check connection pool metrics
# Prometheus query: sql_studio_database_connections_active / sql_studio_database_connections_max

# Check active connections
kubectl exec -it POD_NAME -n sql-studio -- /bin/sh
# Then check internal metrics or health endpoint
```

**Resolution:**

**Immediate (Stop the bleeding):**
1. **Scale up connection pool**
   ```bash
   # Update environment variable
   kubectl set env deployment/sql-studio-backend -n sql-studio \
     DB_MAX_CONNECTIONS=50
   ```

2. **Scale horizontally** (more pods, same total connections)
   ```bash
   kubectl scale deployment/sql-studio-backend --replicas=3 -n sql-studio
   ```

3. **Restart leaking pods** (if connection leak suspected)
   ```bash
   kubectl rollout restart deployment/sql-studio-backend -n sql-studio
   ```

**Long-term:**
1. **Find connection leaks**
   - Check for unclosed DB connections in code
   - Look for long-running transactions
   - Review defer close() patterns

2. **Optimize connection usage**
   - Use connection pooling effectively
   - Close connections promptly
   - Set appropriate idle timeout

3. **Right-size pool**
   - Formula: `max_connections = (num_pods * connections_per_pod)`
   - Leave headroom for spikes

---

### High Memory Usage (>80%)

**Alert:** `HighMemoryUsage`
**Severity:** Warning
**Symptoms:**
- Memory gauge showing >80%
- Slow performance
- Risk of OOM kill

**Diagnosis:**
```bash
# Check memory usage
kubectl top pods -n sql-studio

# Get detailed metrics
kubectl describe pod POD_NAME -n sql-studio | grep -A 5 "Limits:"

# Profile memory usage (requires pprof)
kubectl port-forward POD_NAME 8080:8080 -n sql-studio
go tool pprof http://localhost:8080/debug/pprof/heap
```

**Resolution:**

1. **Immediate: Scale up memory**
   ```bash
   kubectl edit deployment sql-studio-backend -n sql-studio
   # Increase resources.limits.memory
   ```

2. **Investigate memory leak**
   - Capture heap profile
   - Look for goroutine leaks
   - Check for unbounded caches

3. **Optimize memory usage**
   - Implement cache eviction
   - Reduce batch sizes
   - Stream large responses instead of buffering

---

### Failed Authentication Attempts (>100/min)

**Alert:** `HighFailedAuthenticationRate`
**Severity:** Critical (potential security issue)
**Symptoms:**
- Spike in failed login attempts
- Possible brute force attack
- Account lockouts

**Diagnosis:**
```bash
# Check failed auth rate
# Prometheus query: rate(sql_studio_auth_attempts_total{status="failed"}[1m]) * 60

# View failed attempts by IP
kubectl logs -n sql-studio -l app=sql-studio-backend | grep "authentication failed" | awk '{print $NF}' | sort | uniq -c | sort -rn | head -20
```

**Resolution:**

1. **Block malicious IPs** (if concentrated attack)
   ```bash
   # Add to deny list in ingress controller
   kubectl edit configmap nginx-configuration -n ingress-nginx
   ```

2. **Enable rate limiting**
   - Ensure rate limiting is active on auth endpoints
   - Reduce limits temporarily

3. **Check for credential stuffing**
   - Review list of attempted usernames
   - Alert affected users to reset passwords

4. **Enable CAPTCHA** (if not already enabled)

5. **Monitor for escalation**

**Follow-up:**
- Review security logs
- Check if any accounts were compromised
- Update security policies

---

### Sync Failures (>10% of operations)

**Alert:** `HighSyncFailureRate`
**Severity:** Warning
**Symptoms:**
- Users unable to sync data
- Conflict resolution issues
- Data inconsistency

**Diagnosis:**
```bash
# Check sync failure rate
# Prometheus query: sum(rate(sql_studio_sync_operations_total{status="failed"}[5m])) / sum(rate(sql_studio_sync_operations_total[5m]))

# View sync errors
kubectl logs -n sql-studio -l app=sql-studio-backend | grep "sync operation failed"

# Check Turso connectivity
kubectl exec -it POD_NAME -n sql-studio -- nc -zv turso.io 443
```

**Resolution:**

1. **Check Turso service status**
   - Visit Turso status page
   - Test connectivity from pods

2. **Review conflict resolution**
   - Check if conflicts are being handled properly
   - Verify conflict strategy configuration

3. **Check for data corruption**
   - Validate sync payloads
   - Look for schema mismatches

4. **Temporary: Disable sync** (if critical)
   ```bash
   kubectl set env deployment/sql-studio-backend -n sql-studio \
     SYNC_ENABLED=false
   ```

---

### SSL Certificate Expiring Soon (<7 days)

**Alert:** `SSLCertificateExpiringSoon`
**Severity:** Warning
**Symptoms:**
- Certificate expiration warning
- Users may see security warnings soon

**Diagnosis:**
```bash
# Check certificate expiration
kubectl get certificate -n sql-studio

# View cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager
```

**Resolution:**

1. **Trigger cert renewal**
   ```bash
   # Delete cert to force renewal
   kubectl delete certificate sql-studio-tls -n sql-studio

   # cert-manager should automatically request new cert
   ```

2. **Check cert-manager**
   ```bash
   # Verify cert-manager is running
   kubectl get pods -n cert-manager

   # Check for errors
   kubectl logs -n cert-manager deployment/cert-manager
   ```

3. **Manual cert update** (if cert-manager failing)
   - Obtain new certificate manually
   - Update secret

---

## Service Recovery

### Complete Service Outage

**Checklist:**
- [ ] Check Kubernetes cluster health
- [ ] Verify all pods are running
- [ ] Check database connectivity
- [ ] Verify ingress controller
- [ ] Check DNS resolution
- [ ] Review recent changes
- [ ] Check cloud provider status

**Recovery Steps:**

1. **Check cluster**
   ```bash
   kubectl cluster-info
   kubectl get nodes
   kubectl get pods --all-namespaces
   ```

2. **Check deployments**
   ```bash
   kubectl get deployments -n sql-studio
   kubectl get services -n sql-studio
   ```

3. **Check ingress**
   ```bash
   kubectl get ingress -n sql-studio
   kubectl describe ingress sql-studio -n sql-studio
   ```

4. **Rollback if recent deployment**
   ```bash
   kubectl rollout undo deployment/sql-studio-backend -n sql-studio
   ```

5. **Scale up if needed**
   ```bash
   kubectl scale deployment/sql-studio-backend --replicas=3 -n sql-studio
   ```

---

## Performance Troubleshooting

### Profiling Go Application

**CPU Profile:**
```bash
# Capture 30-second CPU profile
kubectl port-forward POD_NAME 8080:8080 -n sql-studio
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze
go tool pprof cpu.prof
# Commands: top, list, web
```

**Memory Profile:**
```bash
# Capture heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# Analyze
go tool pprof heap.prof
```

**Goroutine Profile:**
```bash
# Check for goroutine leaks
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

---

## Quick Command Reference

```bash
# View all pods
kubectl get pods -n sql-studio

# View pod logs (live)
kubectl logs -f POD_NAME -n sql-studio

# Execute command in pod
kubectl exec -it POD_NAME -n sql-studio -- /bin/sh

# Port forward to pod
kubectl port-forward POD_NAME 8080:8080 -n sql-studio

# Describe pod (events, status)
kubectl describe pod POD_NAME -n sql-studio

# Rollback deployment
kubectl rollout undo deployment/sql-studio-backend -n sql-studio

# Scale deployment
kubectl scale deployment/sql-studio-backend --replicas=5 -n sql-studio

# Restart deployment
kubectl rollout restart deployment/sql-studio-backend -n sql-studio

# View deployment history
kubectl rollout history deployment/sql-studio-backend -n sql-studio

# Check resource usage
kubectl top pods -n sql-studio
kubectl top nodes
```

## Additional Resources

- [Incident Response Guide](./INCIDENT_RESPONSE_GUIDE.md)
- [On-Call Guide](./ON_CALL_GUIDE.md)
- [Monitoring Dashboards](https://grafana.sqlstudio.io)
- [Trace Viewer](https://jaeger.sqlstudio.io)
