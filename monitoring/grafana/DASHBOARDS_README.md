# Grafana Dashboards for Howlerops

This directory contains Grafana dashboard configurations for comprehensive monitoring of Howlerops.

## Dashboard Overview

### 1. Application Overview Dashboard
**File:** `dashboards/application-overview.json`
**Purpose:** High-level application performance and health metrics

**Key Panels:**

#### Request Rate (Gauge)
```promql
sum(rate(sql_studio_http_requests_total[5m]))
```

#### Error Rate (Gauge with thresholds: 1% yellow, 5% red)
```promql
sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) / sum(rate(sql_studio_http_requests_total[5m]))
```

#### P95 Latency (Graph)
```promql
histogram_quantile(0.95, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (le))
```

#### P99 Latency (Graph)
```promql
histogram_quantile(0.99, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (le))
```

#### Requests by Endpoint (Graph)
```promql
sum(rate(sql_studio_http_requests_total[5m])) by (endpoint)
```

#### Errors by Endpoint (Table)
```promql
topk(10, sum(rate(sql_studio_http_requests_total{status=~"5.."}[5m])) by (endpoint))
```

#### Active Users (Gauge)
```promql
sql_studio_business_active_users
```

#### Requests in Flight (Graph)
```promql
sql_studio_http_requests_in_flight
```

#### Top Slowest Endpoints (Table)
```promql
topk(10, histogram_quantile(0.95, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (endpoint, le)))
```

### 2. Infrastructure Dashboard
**File:** `dashboards/infrastructure.json`
**Purpose:** Kubernetes cluster and container metrics

**Key Panels:**

#### CPU Usage by Pod (Graph)
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="sql-studio"}[5m])) by (pod)
```

#### CPU Usage Percentage (Graph with threshold at 80%)
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="sql-studio"}[5m])) by (pod) /
sum(container_spec_cpu_quota{namespace="sql-studio"}) by (pod) * 100000
```

#### Memory Usage by Pod (Graph)
```promql
sum(container_memory_working_set_bytes{namespace="sql-studio"}) by (pod)
```

#### Memory Usage Percentage (Graph with threshold at 80%)
```promql
sum(container_memory_working_set_bytes{namespace="sql-studio"}) by (pod) /
sum(container_spec_memory_limit_bytes{namespace="sql-studio"}) by (pod)
```

#### Network I/O (Graph)
```promql
# Receive
sum(rate(container_network_receive_bytes_total{namespace="sql-studio"}[5m])) by (pod)

# Transmit
sum(rate(container_network_transmit_bytes_total{namespace="sql-studio"}[5m])) by (pod)
```

#### Pod Status (Stat panel)
```promql
count(kube_pod_status_phase{namespace="sql-studio", phase="Running"})
count(kube_pod_status_phase{namespace="sql-studio", phase!="Running"})
```

#### Pod Restarts (Graph)
```promql
rate(kube_pod_container_status_restarts_total{namespace="sql-studio"}[5m])
```

#### Node Status (Table)
```promql
kube_node_status_condition{condition="Ready", status="true"}
```

### 3. Database Dashboard
**File:** `dashboards/database.json`
**Purpose:** Database performance and connection pool metrics

**Key Panels:**

#### Query Rate (Graph)
```promql
sum(rate(sql_studio_database_queries_total[5m])) by (operation)
```

#### Query Duration P95 (Graph)
```promql
histogram_quantile(0.95, sum(rate(sql_studio_database_query_duration_seconds_bucket[5m])) by (operation, le))
```

#### Slow Queries (>1s) (Counter)
```promql
sum(rate(sql_studio_database_query_duration_seconds_count{operation=~".*"}[5m])) by (operation) > 1
```

#### Connection Pool Utilization (Gauge with 90% threshold)
```promql
sql_studio_database_connections_active / sql_studio_database_connections_max
```

#### Active Connections (Graph)
```promql
sql_studio_database_connections_active
```

#### Connection Churn (Graph)
```promql
# Connections opened
rate(sql_studio_database_connections_opened_total[5m])

# Connections closed
rate(sql_studio_database_connections_closed_total[5m])
```

#### Query Errors (Graph)
```promql
sum(rate(sql_studio_database_queries_total{status="error"}[5m])) by (operation)
```

#### Rows Returned Distribution (Heatmap)
```promql
sum(rate(sql_studio_database_rows_returned_bucket[5m])) by (le)
```

### 4. Business Metrics Dashboard
**File:** `dashboards/business-metrics.json`
**Purpose:** Business KPIs and user engagement metrics

**Key Panels:**

#### Daily Active Users (Graph)
```promql
sql_studio_business_active_users
```

#### Queries Executed per Minute (Graph)
```promql
sum(rate(sql_studio_database_queries_total[1m]))
```

#### New User Signups (Counter)
```promql
increase(sql_studio_business_user_registrations_total[1h])
```

#### Organizations Created (Counter)
```promql
increase(sql_studio_business_organizations_created_total[1d])
```

#### Feature Usage (Bar chart)
```promql
topk(10, sum(rate(sql_studio_business_feature_usage_total[5m])) by (feature))
```

#### Auth Success Rate (Gauge)
```promql
sum(rate(sql_studio_auth_attempts_total{status="success"}[5m])) /
sum(rate(sql_studio_auth_attempts_total[5m]))
```

#### Sync Operations (Graph)
```promql
sum(rate(sql_studio_sync_operations_total[5m])) by (operation, status)
```

#### User Retention (Calculated metric)
```promql
# Users active in last 7 days who were also active 7-14 days ago
# This requires custom recording rules
```

### 5. Security Dashboard
**Purpose:** Security monitoring and threat detection

**Key Panels:**

#### Failed Login Attempts (Graph with alert at 100/min)
```promql
sum(rate(sql_studio_auth_attempts_total{status="failed"}[1m])) * 60
```

#### Account Lockouts (Counter)
```promql
increase(sql_studio_auth_lockouts_total[5m])
```

#### Failed Logins by IP (Table)
```promql
topk(10, sum(rate(sql_studio_auth_attempts_total{status="failed"}[5m])) by (client_ip))
```

### 6. SLO Dashboard
**File:** `dashboards/slo-dashboard.json`
**Purpose:** Track SLO compliance and error budgets

**Key Panels:**

#### Availability SLI (Gauge with 99.9% SLO)
```promql
sum(rate(sql_studio_http_requests_total{status!~"5.."}[30d])) /
sum(rate(sql_studio_http_requests_total[30d]))
```

#### Error Budget Remaining (Bar chart)
```promql
# Total allowed errors per month
(1 - 0.999) * sum(increase(sql_studio_http_requests_total[30d])) -
# Actual errors
sum(increase(sql_studio_http_requests_total{status=~"5.."}[30d]))
```

#### Error Budget Burn Rate (Graph with multi-window)
```promql
# 5-minute window
(1 - sum(rate(sql_studio_http_requests_total{status!~"5.."}[5m])) / sum(rate(sql_studio_http_requests_total[5m]))) / (1 - 0.999)

# 1-hour window
(1 - sum(rate(sql_studio_http_requests_total{status!~"5.."}[1h])) / sum(rate(sql_studio_http_requests_total[1h]))) / (1 - 0.999)
```

#### Latency SLI (Gauge with 200ms P95 target)
```promql
histogram_quantile(0.95, sum(rate(sql_studio_http_request_duration_seconds_bucket[5m])) by (le))
```

## Creating Dashboards

### Option 1: Using Grafana UI

1. Log into Grafana
2. Click "+" → "Import"
3. Upload the JSON file or paste the JSON content
4. Select the Prometheus datasource
5. Click "Import"

### Option 2: Automatic Provisioning

Dashboards in the `dashboards/` directory are automatically loaded if you have:
- Mounted the directory to `/etc/grafana/dashboards` in the Grafana container
- Configured provisioning in `provisioning/dashboards.yml`

### Option 3: Using Terraform/Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards
  namespace: monitoring
data:
  application-overview.json: |
    {{ .Files.Get "dashboards/application-overview.json" | nindent 4 }}
```

## Dashboard Variables

Most dashboards support the following variables for filtering:

- **$namespace**: Kubernetes namespace (default: sql-studio)
- **$pod**: Pod name (supports multi-select)
- **$interval**: Time interval for rate calculations
- **$percentile**: Percentile for latency graphs (50, 95, 99)

### Example Variable Definitions:

**Namespace Variable:**
```promql
label_values(sql_studio_http_requests_total, namespace)
```

**Pod Variable:**
```promql
label_values(sql_studio_http_requests_total{namespace="$namespace"}, pod)
```

**Endpoint Variable:**
```promql
label_values(sql_studio_http_requests_total, endpoint)
```

## Alert Annotations

Dashboards can show alert firing times as annotations. Configure in dashboard settings:

```json
{
  "annotations": {
    "list": [
      {
        "datasource": "Prometheus",
        "enable": true,
        "expr": "ALERTS{alertstate=\"firing\", namespace=\"sql-studio\"}",
        "iconColor": "red",
        "name": "Alerts",
        "tagKeys": "alertname,severity",
        "titleFormat": "{{ alertname }}",
        "textFormat": "{{ description }}"
      }
    ]
  }
}
```

## Best Practices

1. **Use Recording Rules**: For frequently used queries, create recording rules in `prometheus/recording-rules.yml`

2. **Set Appropriate Time Ranges**:
   - Real-time monitoring: Last 15m - 1h
   - Troubleshooting: Last 6h - 24h
   - Trend analysis: Last 7d - 30d

3. **Use Thresholds**: Set warning/critical thresholds on gauges and graphs

4. **Add Links**: Link related dashboards for easier navigation

5. **Document Panels**: Add descriptions to explain what each panel shows

## Exporting Dashboards

To export a dashboard for version control:

1. Open the dashboard
2. Click the settings icon (⚙️)
3. Select "JSON Model"
4. Copy the JSON
5. Save to the `dashboards/` directory

## Common Issues

### Dashboard shows "No data"
- Check that Prometheus is scraping the backend pods
- Verify the metric names match the queries
- Check time range selection

### High cardinality warnings
- Review endpoint normalization in the metrics middleware
- Add recording rules for high-cardinality queries

### Dashboard is slow
- Use shorter time ranges
- Create recording rules for expensive queries
- Increase Prometheus resources

## Dashboard Refresh Rates

Recommended refresh rates by dashboard type:
- Real-time monitoring: 5s - 10s
- Operations: 30s - 1m
- Business metrics: 5m - 15m
- SLO tracking: 1h

## Additional Resources

- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
