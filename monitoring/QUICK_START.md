# Monitoring Quick Start Guide

Get SQL Studio monitoring up and running in 15 minutes.

## Prerequisites

- Kubernetes cluster running
- kubectl configured
- Helm 3 installed
- SQL Studio backend deployed

## Step 1: Deploy Monitoring Stack (5 minutes)

```bash
# Create monitoring namespace
kubectl create namespace monitoring

# Add Helm repositories
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install Prometheus Stack (includes Grafana, AlertManager)
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.retention=15d \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus -n monitoring --timeout=300s
```

## Step 2: Apply Custom Configurations (2 minutes)

```bash
# Navigate to project directory
cd /Users/jacob_1/projects/sql-studio

# Apply Prometheus alerts
kubectl create configmap prometheus-alerts \
  --from-file=monitoring/prometheus/alerts.yml \
  --namespace monitoring

# Apply recording rules
kubectl create configmap prometheus-recording-rules \
  --from-file=monitoring/prometheus/recording-rules.yml \
  --namespace monitoring

# Apply AlertManager configuration
kubectl create secret generic alertmanager-config \
  --from-file=monitoring/alerting/alertmanager.yml \
  --namespace monitoring

# Restart Prometheus to pick up changes
kubectl rollout restart statefulset/prometheus-prometheus-kube-prometheus-prometheus -n monitoring
```

## Step 3: Deploy Logging (3 minutes)

```bash
# Deploy Fluentd DaemonSet
kubectl apply -f monitoring/logging/fluentd-config.yaml

# Verify Fluentd is running on all nodes
kubectl get pods -n kube-system -l app=fluentd
```

## Step 4: Deploy Tracing (2 minutes)

```bash
# Deploy Jaeger
kubectl apply -f monitoring/tracing/jaeger-config.yaml

# Wait for Jaeger to be ready
kubectl wait --for=condition=ready pod -l app=jaeger -n monitoring --timeout=180s
```

## Step 5: Deploy Synthetic Monitoring (2 minutes)

```bash
# Deploy Blackbox Exporter
kubectl apply -f monitoring/synthetic/uptime-checks.yaml

# Make smoke tests executable
chmod +x monitoring/synthetic/smoke-tests.sh

# Run smoke tests
./monitoring/synthetic/smoke-tests.sh
```

## Step 6: Access Dashboards (1 minute)

```bash
# Get Grafana admin password
kubectl get secret --namespace monitoring prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo

# Port forward Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80 &

# Port forward Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090 &

# Port forward Jaeger
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686 &

# Port forward AlertManager
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-alertmanager 9093:9093 &
```

**Access URLs:**
- Grafana: http://localhost:3000 (admin / <password from above>)
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686
- AlertManager: http://localhost:9093

## Step 7: Integrate Application (Backend)

Add to `/backend-go/cmd/server/main.go`:

```go
import (
    "github.com/sql-studio/backend-go/internal/middleware"
    "github.com/sql-studio/backend-go/internal/health"
    "github.com/sql-studio/backend-go/internal/tracing"
)

// Add metrics middleware
r.Use(middleware.MetricsMiddleware)

// Initialize health checker
healthChecker := health.NewHealthChecker(Version)
healthChecker.RegisterChecker(health.NewDatabaseChecker("turso", tursoClient.DB()))

// Add health endpoints
r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    check := healthChecker.Check(r.Context())
    status := http.StatusOK
    if check.Status == health.StatusUnhealthy {
        status = http.StatusServiceUnavailable
    }
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(check)
})

r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
    if healthChecker.Ready(r.Context()) {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
})

r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
    if healthChecker.Live() {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
})

// Initialize tracing
tracerProvider, err := tracing.InitTracer(tracing.Config{
    Enabled:     true,
    ServiceName: "sql-studio-backend",
    Environment: cfg.GetEnv(),
    JaegerURL:   "http://jaeger-collector.monitoring.svc.cluster.local:14268/api/traces",
    SampleRate:  0.01, // 1% in production
})
if err != nil {
    logger.WithError(err).Fatal("Failed to initialize tracer")
}
defer tracerProvider.Shutdown(context.Background())
```

Update Kubernetes deployment to add Prometheus annotations:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sql-studio-backend
  namespace: sql-studio
spec:
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9100"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: backend
        # ... rest of container spec
```

## Step 8: Verify Monitoring (2 minutes)

### Check Metrics

```bash
# Check that Prometheus is scraping the backend
# Navigate to: http://localhost:9090/targets
# Look for job "sql-studio-backend" - should be UP

# Test a query
# Navigate to: http://localhost:9090/graph
# Query: sql_studio_http_requests_total
# Should see data
```

### Check Logs

```bash
# View application logs
kubectl logs -n sql-studio -l app=sql-studio-backend --tail=50

# Should see JSON-formatted logs with trace_id, user_id, etc.
```

### Check Traces

```bash
# Navigate to: http://localhost:16686
# Select "sql-studio-backend" service
# Click "Find Traces"
# Should see traces (if there has been traffic)
```

### Check Alerts

```bash
# Navigate to: http://localhost:9090/alerts
# Should see all alert rules listed
# Check that rules are evaluating (not showing errors)
```

## Step 9: Import Dashboards (Optional)

The full dashboard JSONs would be large files. For now, create basic dashboards manually:

1. Go to Grafana (http://localhost:3000)
2. Create → Dashboard
3. Add Panel
4. Use queries from `/monitoring/grafana/DASHBOARDS_README.md`

Key panels to create:
- Request Rate: `sum(rate(sql_studio_http_requests_total[5m]))`
- Error Rate: `sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) / sum(rate(sql_studio_http_requests_total[5m]))`
- P95 Latency: `histogram_quantile(0.95, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (le))`

## Step 10: Test Alerts

Trigger an alert to verify the system works:

```bash
# Temporarily set a very low threshold for error rate
kubectl edit prometheusrule prometheus-kube-prometheus-alertmanager.rules -n monitoring

# Or manually create a test alert
curl -X POST http://localhost:9093/api/v1/alerts -d '[
  {
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning"
    },
    "annotations": {
      "summary": "This is a test alert"
    }
  }
]'

# Check AlertManager: http://localhost:9093
# Should see the alert
```

## Troubleshooting

### Metrics not appearing

```bash
# Check if application is exposing metrics
kubectl port-forward -n sql-studio POD_NAME 9100:9100
curl http://localhost:9100/metrics

# Should see Prometheus-formatted metrics
```

### Prometheus not scraping

```bash
# Check ServiceMonitor or PodMonitor
kubectl get servicemonitor -n sql-studio
kubectl get podmonitor -n sql-studio

# If missing, check that prometheus.io annotations are set
kubectl get deployment sql-studio-backend -n sql-studio -o yaml | grep -A 5 annotations
```

### Logs not appearing in Elasticsearch

```bash
# Check Fluentd is running
kubectl get pods -n kube-system -l app=fluentd

# Check Fluentd logs for errors
kubectl logs -n kube-system -l app=fluentd | grep ERROR
```

### Traces not appearing in Jaeger

```bash
# Check Jaeger is running
kubectl get pods -n monitoring -l app=jaeger

# Check if backend can reach Jaeger
kubectl exec -it -n sql-studio POD_NAME -- nc -zv jaeger-collector.monitoring.svc.cluster.local 14268

# Check sampling rate (should be >0 for development)
# In production with 1% sampling, traces may be sparse
```

## Next Steps

1. Read the [Monitoring README](/Users/jacob_1/projects/sql-studio/monitoring/README.md)
2. Review [Incident Response Guide](/Users/jacob_1/projects/sql-studio/docs/operations/INCIDENT_RESPONSE_GUIDE.md)
3. Study the [Runbook](/Users/jacob_1/projects/sql-studio/docs/operations/RUNBOOK.md)
4. Set up alerting channels (Slack, PagerDuty, email)
5. Run load tests to establish baselines
6. Tune alert thresholds based on actual traffic
7. Create custom dashboards for your team's needs

## Production Checklist

Before going to production:

- [ ] Prometheus has persistent storage configured
- [ ] Grafana has persistent storage configured
- [ ] AlertManager notification channels configured
- [ ] On-call rotation set up in PagerDuty
- [ ] Runbooks reviewed and updated
- [ ] Team trained on monitoring tools
- [ ] Smoke tests run successfully
- [ ] Load testing completed
- [ ] Alert thresholds tuned to avoid false positives
- [ ] Backup/restore procedures tested

## Quick Reference

### Important Metrics

```promql
# Request rate
sum(rate(sql_studio_http_requests_total[5m]))

# Error rate
sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) / sum(rate(sql_studio_http_requests_total[5m]))

# P95 latency
histogram_quantile(0.95, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (le))

# Database connections
sql_studio_database_connections_active / sql_studio_database_connections_max

# Memory usage
container_memory_working_set_bytes{namespace="sql-studio"} / container_spec_memory_limit_bytes{namespace="sql-studio"}
```

### Useful Commands

```bash
# View logs
kubectl logs -n sql-studio -l app=sql-studio-backend -f

# Check pod health
kubectl get pods -n sql-studio

# Restart deployment
kubectl rollout restart deployment/sql-studio-backend -n sql-studio

# Scale deployment
kubectl scale deployment/sql-studio-backend --replicas=3 -n sql-studio

# Run smoke tests
./monitoring/synthetic/smoke-tests.sh
```

## Support

- Documentation: `/monitoring/README.md`
- Runbook: `/docs/operations/RUNBOOK.md`
- Incident Response: `/docs/operations/INCIDENT_RESPONSE_GUIDE.md`
- On-Call Guide: `/docs/operations/ON_CALL_GUIDE.md`
