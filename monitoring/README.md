# Howlerops Monitoring & Observability Infrastructure

## Overview

This directory contains comprehensive monitoring, logging, and observability configurations for Howlerops. The monitoring stack provides full visibility into application performance, infrastructure health, business metrics, and user experience.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Howlerops Application                   │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │  Frontend  │  │   Backend  │  │  Database (Turso)   │  │
│  └─────┬──────┘  └──────┬─────┘  └──────────┬──────────┘  │
│        │                │                    │              │
│        │                │  Metrics (/metrics endpoint)      │
│        │                ├────────────────────┘              │
│        │                │                                   │
└────────┼────────────────┼───────────────────────────────────┘
         │                │
         │                │  Logs (stdout/stderr)
         │                ├────────────────────┐
         │                │                    │
    ┌────▼────────────────▼─────┐         ┌───▼────────┐
    │      Prometheus            │         │  Fluentd   │
    │  (Metrics Collection)      │         │ (Log Agg)  │
    └────────┬───────────────────┘         └─────┬──────┘
             │                                   │
             │                                   │
    ┌────────▼────────────┐              ┌──────▼──────────┐
    │   AlertManager      │              │  Elasticsearch  │
    │ (Alert Routing)     │              │  (Log Storage)  │
    └────────┬────────────┘              └─────────────────┘
             │
             │
    ┌────────▼─────────────────────────────────┐
    │            Grafana                        │
    │  ┌──────────────┐  ┌──────────────────┐  │
    │  │  Dashboards  │  │  Alert Mgmt UI   │  │
    │  └──────────────┘  └──────────────────┘  │
    └───────────────────────────────────────────┘
             │
    ┌────────▼─────────────────┐
    │  Notification Channels   │
    │  ├─ Slack                │
    │  ├─ Email                │
    │  ├─ PagerDuty            │
    │  └─ Webhooks             │
    └──────────────────────────┘
```

## Components

### 1. Metrics Collection (Prometheus)

**Location:** `/monitoring/prometheus/`

- **prometheus.yml** - Main Prometheus configuration with scrape configs
- **alerts.yml** - Alert rule definitions (error rates, latency, crashes, etc.)
- **recording-rules.yml** - Pre-computed metrics for faster queries

**Metrics Exposed:**
- HTTP request metrics (rate, duration, errors)
- Database query metrics (duration, errors, connection pool)
- Authentication metrics (attempts, failures, lockouts)
- Sync operation metrics (success rate, duration)
- Business metrics (active users, queries executed, signups)
- Infrastructure metrics (CPU, memory, network, disk)

### 2. Visualization (Grafana)

**Location:** `/monitoring/grafana/`

**Dashboards:**
- **Application Overview** - Request rate, error rate, latency, active users
- **Infrastructure** - CPU, memory, pods, nodes
- **Database** - Query performance, connection pool, slow queries
- **Business Metrics** - DAU, queries/day, signups, feature adoption
- **SLO Tracking** - SLI compliance, error budget, burn rate

**Configuration:**
- `provisioning/datasources.yml` - Auto-provision Prometheus
- `provisioning/dashboards.yml` - Auto-load dashboard JSONs
- `DASHBOARDS_README.md` - PromQL queries and dashboard guide

### 3. Logging (Fluentd + Elasticsearch/Loki)

**Location:** `/monitoring/logging/`

- **fluentd-config.yaml** - Log collection from Kubernetes pods
- **structured.go** - Structured logging implementation for Go

**Features:**
- JSON-formatted logs with context (trace ID, user ID, request ID)
- Automatic Kubernetes metadata injection
- Sensitive data redaction (passwords, tokens, API keys)
- Log aggregation from all pods
- Searchable log index in Elasticsearch

**Log Types:**
- HTTP requests (method, path, duration, status)
- Database queries (query, duration, row count)
- Authentication events (login attempts, failures)
- Sync operations (operation type, data size, duration)
- Security events (severity levels, detailed context)
- Business events (signups, feature usage)

### 4. Distributed Tracing (Jaeger)

**Location:** `/monitoring/tracing/`

- **jaeger-config.yaml** - Jaeger all-in-one deployment
- **tracer.go** - OpenTelemetry instrumentation

**Sampling Strategy:**
- Production: 1% of traces (probabilistic sampling)
- Development: 100% of traces

**Trace Data:**
- End-to-end request flow
- Database query execution
- External API calls
- Sync operations
- Performance bottleneck identification

### 5. Alerting (AlertManager)

**Location:** `/monitoring/alerting/`

- **alertmanager.yml** - Alert routing and notification configuration
- **notification-templates/** - Email and Slack templates

**Alert Routing:**
- **SEV-1 (Critical)** → PagerDuty + Slack (#critical-alerts)
- **SEV-2 (High)** → Slack (#alerts) + Email
- **SEV-3 (Medium)** → Slack (#warnings)
- **Security** → Security team + Slack (#security-alerts)

**Alert Types:**
- High error rate (>5%)
- High latency (p95 >500ms)
- Pod crashes (>3 restarts in 10min)
- High memory/CPU usage (>80%)
- Connection pool exhaustion
- Failed auth attempts (>100/min)
- Sync failures (>10%)
- SSL certificate expiry (<7 days)

### 6. Health Checks

**Location:** `/backend-go/internal/health/`

**Endpoints:**
- `GET /health` - Overall health (for load balancer)
- `GET /health/ready` - Readiness probe (for Kubernetes)
- `GET /health/live` - Liveness probe (for Kubernetes)
- `GET /health/detailed` - Full health report (authenticated)

**Checks:**
- Database connectivity
- Redis connectivity (if enabled)
- External API availability
- Disk space
- Memory pressure

### 7. Synthetic Monitoring

**Location:** `/monitoring/synthetic/`

- **uptime-checks.yaml** - External health checks via Blackbox Exporter
- **smoke-tests.sh** - Post-deployment validation script

**Checks:**
- Homepage availability (every 1 minute)
- API health endpoint (every 30 seconds)
- Critical user flows (every 5 minutes)
- Multi-region availability
- SSL certificate validity

### 8. SLO Tracking

**Location:** `/monitoring/slo/`

**Defined SLOs:**
- **Availability:** 99.9% uptime (43 minutes downtime/month)
- **Latency:** 95% of requests <200ms
- **Query Success:** 99.5% of queries succeed
- **Sync Success:** 99.5% of sync operations succeed

**Burn Rate Alerts:**
- Fast burn: 2% error budget in 1 hour (critical)
- Moderate burn: 5% error budget in 6 hours (warning)
- Slow burn: 10% error budget in 3 days (info)

### 9. Performance Profiling

**Location:** `/backend-go/internal/profiling/`

**Enabled Profiles:**
- CPU profiling (`/debug/pprof/profile`)
- Heap profiling (`/debug/pprof/heap`)
- Goroutine profiling (`/debug/pprof/goroutine`)
- Block profiling (`/debug/pprof/block`)
- Mutex profiling (`/debug/pprof/mutex`)

**Usage:**
```bash
# Capture 30-second CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Analyze heap
go tool pprof http://localhost:8080/debug/pprof/heap

# View in browser
go tool pprof -http=:8081 profile.prof
```

## Operational Documentation

**Location:** `/docs/operations/`

- **INCIDENT_RESPONSE_GUIDE.md** - Incident severity levels, response procedures
- **RUNBOOK.md** - Common issues and resolution steps
- **ON_CALL_GUIDE.md** - On-call rotation, responsibilities, quick reference

## Deployment

### Prerequisites

1. Kubernetes cluster running
2. Helm installed
3. kubectl configured

### Install Monitoring Stack

```bash
# Create monitoring namespace
kubectl create namespace monitoring

# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values monitoring/prometheus/values.yaml

# Deploy custom configurations
kubectl apply -f monitoring/prometheus/prometheus.yml
kubectl apply -f monitoring/prometheus/alerts.yml
kubectl apply -f monitoring/prometheus/recording-rules.yml

# Deploy Grafana dashboards
kubectl create configmap grafana-dashboards \
  --from-file=monitoring/grafana/dashboards/ \
  --namespace monitoring

# Deploy Fluentd
kubectl apply -f monitoring/logging/fluentd-config.yaml

# Deploy Jaeger
kubectl apply -f monitoring/tracing/jaeger-config.yaml

# Deploy Blackbox Exporter
kubectl apply -f monitoring/synthetic/uptime-checks.yaml

# Deploy AlertManager config
kubectl apply -f monitoring/alerting/alertmanager.yml
```

### Verify Deployment

```bash
# Check all monitoring components are running
kubectl get pods -n monitoring

# Access Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# Open http://localhost:3000 (admin/prom-operator)

# Access Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Open http://localhost:9090

# Access AlertManager
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-alertmanager 9093:9093
# Open http://localhost:9093

# Access Jaeger
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686
# Open http://localhost:16686
```

## Application Integration

### 1. Add Metrics Middleware

In `/backend-go/cmd/server/main.go`:

```go
import (
    "github.com/sql-studio/backend-go/internal/middleware"
)

// Add metrics middleware to your Chi router
r.Use(middleware.MetricsMiddleware)
```

### 2. Add Health Checks

```go
import (
    "github.com/sql-studio/backend-go/internal/health"
)

// Initialize health checker
healthChecker := health.NewHealthChecker(Version)

// Register health checks
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
```

### 3. Add Tracing

```go
import (
    "github.com/sql-studio/backend-go/internal/tracing"
)

// Initialize tracer
tracerProvider, err := tracing.InitTracer(tracing.Config{
    Enabled:     cfg.Tracing.Enabled,
    ServiceName: "sql-studio-backend",
    Environment: cfg.GetEnv(),
    JaegerURL:   "http://jaeger-collector:14268/api/traces",
    SampleRate:  0.01, // 1% sampling in production
})
if err != nil {
    logger.WithError(err).Fatal("Failed to initialize tracer")
}
defer tracerProvider.Shutdown(context.Background())

// In your handlers, add tracing
ctx, span := tracing.StartSpan(r.Context(), "http-handler", "HandleRequest")
defer span.End()

tracing.SetAttributes(ctx,
    tracing.HTTPMethodKey.String(r.Method),
    tracing.HTTPPathKey.String(r.URL.Path),
)
```

### 4. Add Structured Logging

```go
import (
    structuredLogger "github.com/sql-studio/backend-go/internal/logger"
)

// Create structured logger
sLogger := structuredLogger.NewStructuredLogger(appLogger)

// Log with context
sLogger.LogRequest(ctx, r.Method, r.URL.Path, duration, statusCode)
sLogger.LogDatabaseQuery(ctx, query, duration, rowCount, err)
sLogger.LogAuthentication(ctx, email, success, reason)
```

## Monitoring Best Practices

### 1. Metric Naming
- Use consistent prefixes: `sql_studio_`
- Use subsystems: `http_`, `database_`, `auth_`, `sync_`
- Use base units: seconds, bytes, not milliseconds or kilobytes

### 2. Alert Configuration
- Set meaningful thresholds based on actual baseline
- Use multi-window burn rate alerts for SLOs
- Avoid alert fatigue - only alert on actionable items
- Always include runbook link in annotations

### 3. Dashboard Design
- Show the most important metrics at the top
- Use consistent time ranges across panels
- Add threshold lines for SLOs
- Include drill-down links to traces/logs

### 4. Log Management
- Always use structured logging
- Include correlation IDs (trace ID, request ID)
- Sanitize sensitive data before logging
- Set appropriate log levels

### 5. Tracing
- Sample appropriately (1% in production is usually sufficient)
- Add meaningful span names and attributes
- Trace critical paths (auth, queries, sync)
- Use trace context propagation

## Cost Estimation

Based on typical usage patterns:

| Component | Monthly Cost | Notes |
|-----------|--------------|-------|
| Prometheus (3 replicas, 50GB storage) | $150 | Metrics retention: 15 days |
| Grafana (2 replicas) | $50 | Light compute |
| Elasticsearch (3 nodes, 500GB) | $400 | Log retention: 7 days |
| Jaeger (with ES backend) | $100 | Trace retention: 7 days |
| AlertManager (2 replicas) | $25 | Light compute |
| Blackbox Exporter | $10 | Minimal resources |
| **Total** | **~$735/month** | Scales with traffic |

**Cost Optimization Tips:**
- Use Loki instead of Elasticsearch (50% cheaper for logs)
- Reduce metric retention to 7 days
- Use remote write to cheaper long-term storage
- Sample traces more aggressively (0.1% vs 1%)

## Troubleshooting

### Metrics not appearing in Prometheus

1. Check that application pods are running:
   ```bash
   kubectl get pods -n sql-studio
   ```

2. Verify metrics endpoint is accessible:
   ```bash
   kubectl port-forward POD_NAME 9100:9100 -n sql-studio
   curl http://localhost:9100/metrics
   ```

3. Check Prometheus targets:
   - Navigate to Prometheus UI → Status → Targets
   - Look for `sql-studio-backend` job
   - Check for scrape errors

4. Verify ServiceMonitor/PodMonitor exists:
   ```bash
   kubectl get servicemonitor -n sql-studio
   ```

### Dashboards show "No data"

1. Verify Prometheus datasource in Grafana:
   - Configuration → Data Sources → Prometheus
   - Test connection

2. Check metric names in Prometheus:
   - Query: `{__name__=~"sql_studio_.*"}`

3. Verify time range selection in dashboard

### Alerts not firing

1. Check AlertManager is receiving alerts:
   - Navigate to AlertManager UI → Alerts

2. Verify alert rules in Prometheus:
   - Prometheus UI → Alerts
   - Check rule state and errors

3. Check notification configuration:
   - AlertManager UI → Status
   - Verify receiver configuration

## Next Steps

After deploying monitoring:

1. **Baseline Metrics** - Run for 1 week to establish normal baselines
2. **Tune Alerts** - Adjust thresholds based on observed patterns
3. **Create Runbooks** - Document resolution steps for each alert
4. **Load Testing** - Verify monitoring works under stress
5. **Disaster Recovery** - Test alert routing and on-call rotation
6. **Training** - Train team on dashboards and incident response

## Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [SLO Best Practices](https://sre.google/workbook/implementing-slos/)
- [Incident Response Guide](../docs/operations/INCIDENT_RESPONSE_GUIDE.md)
- [Runbook](../docs/operations/RUNBOOK.md)
- [On-Call Guide](../docs/operations/ON_CALL_GUIDE.md)

## Support

For questions or issues with monitoring:
- Slack: #observability
- Email: devops@sqlstudio.io
- Docs: https://docs.sqlstudio.io/monitoring
