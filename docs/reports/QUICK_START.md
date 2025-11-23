# Reports Features: Quick Start Guide

This guide provides quick code snippets to get started with the three new report features.

## Export

### Basic CSV Export

```go
import "github.com/jbeck018/howlerops/backend-go/pkg/export"

result, err := reportService.ExportReport(
    reportID,
    export.FormatCSV,
    nil, // All components
    nil, // No filters
    export.ExportOptions{},
)
if err != nil {
    return err
}

// Serve as download
w.Header().Set("Content-Type", result.MimeType)
w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
w.Write(result.Data)
```

### Excel Export with Options

```go
result, err := reportService.ExportReport(
    reportID,
    export.FormatExcel,
    []string{"component1", "component2"},
    map[string]interface{}{"startDate": "2024-01-01"},
    export.ExportOptions{
        Title:  "Monthly Sales Report",
        Author: "Analytics Team",
    },
)
```

### PDF Export

```go
result, err := reportService.ExportReport(
    reportID,
    export.FormatPDF,
    nil,
    nil,
    export.ExportOptions{
        PageSize:    "Letter",
        Orientation: "landscape",
        Title:       "Quarterly Performance",
    },
)
```

---

## Alerts

### Create Alert Rule

```go
import "github.com/jbeck018/howlerops/backend-go/pkg/alerts"

rule := &alerts.AlertRule{
    ReportID:    "report-123",
    ComponentID: "component-456",
    Name:        "High Error Rate Alert",
    Description: "Triggers when error rate exceeds 5%",
    Condition: alerts.AlertCondition{
        Metric:    "error_percentage",
        Operator:  ">",
        Threshold: 5.0,
    },
    Actions: []alerts.AlertAction{
        {
            Type:   "email",
            Target: "ops-team@company.com",
        },
    },
    Schedule: "*/15 * * * *", // Every 15 minutes
    Enabled:  true,
}

err := reportService.SaveAlertRule(rule)
```

### Test Alert

```go
result, err := reportService.TestAlert(ruleID)
if err != nil {
    return err
}

if result.Triggered {
    fmt.Printf("Alert would trigger! Value: %.2f, Threshold: %.2f\n",
        result.ActualValue, result.Threshold)
    fmt.Println(result.Message)
}
```

### Get Alert History

```go
history, err := reportService.GetAlertHistory(ruleID, 100)
for _, alert := range history {
    fmt.Printf("%s - Value: %.2f - %s\n",
        alert.TriggeredAt.Format("2006-01-02 15:04"),
        alert.ActualValue,
        alert.Message)
}
```

---

## Materialization

### Materialize Report

```go
import "github.com/jbeck018/howlerops/backend-go/pkg/materialization"

snapshot, err := reportService.MaterializeReport(
    reportID,
    map[string]interface{}{"date": "2024-01-01"},
    24 * time.Hour, // TTL
)
if err != nil {
    return err
}

fmt.Printf("Snapshot created: %s (%.2f KB)\n",
    snapshot.ID, float64(snapshot.SizeBytes)/1024)
```

### Get Report with Automatic Caching

```go
// This will use a cached snapshot if available and fresh,
// otherwise it will run the report and return fresh results
resp, err := reportService.GetReportWithCache(
    reportID,
    map[string]interface{}{"date": "2024-01-01"},
    30 * time.Minute, // Max age
)
if err != nil {
    return err
}

// Check if served from cache
for _, result := range resp.Results {
    if result.CacheHit {
        fmt.Println("Served from cache!")
    }
}
```

### Schedule Periodic Materialization

```go
// Materialize report daily at 6 AM
err := reportService.ScheduleMaterialization(
    reportID,
    "0 6 * * *", // 6 AM daily
    map[string]interface{}{},
    24 * time.Hour,
)
```

### Invalidate Cache

```go
// Invalidate all snapshots for a report (e.g., after data update)
err := reportService.InvalidateCache(reportID)
```

---

## Initialization (in app.go)

```go
// After database initialization
sqlDB := manager.GetDB()

// Initialize alert engine
alertEngine := alerts.NewAlertEngine(sqlDB, logger)
if err := alertEngine.EnsureSchema(); err != nil {
    logger.WithError(err).Warn("Failed to ensure alert schema")
}

// Initialize materializer
materializer := materialization.NewMaterializer(sqlDB, logger)
if err := materializer.EnsureSchema(); err != nil {
    logger.WithError(err).Warn("Failed to ensure materialization schema")
}

// Inject into report service
reportService.SetAlertEngine(alertEngine)
reportService.SetMaterializer(materializer)

// Optional: Background cleanup
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    for range ticker.C {
        if err := materializer.CleanupExpiredSnapshots(); err != nil {
            logger.WithError(err).Warn("Snapshot cleanup failed")
        }
    }
}()
```

---

## HTTP Endpoints (Example)

### Export Endpoint

```go
func (s *Server) handleExportReport(w http.ResponseWriter, r *http.Request) {
    reportID := chi.URLParam(r, "id")
    format := r.URL.Query().Get("format") // csv, xlsx, pdf

    var req struct {
        ComponentIDs []string               `json:"componentIds"`
        Filters      map[string]interface{} `json:"filters"`
        Options      export.ExportOptions   `json:"options"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    result, err := s.reportService.ExportReport(
        reportID,
        export.ExportFormat(format),
        req.ComponentIDs,
        req.Filters,
        req.Options,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", result.MimeType)
    w.Header().Set("Content-Disposition",
        fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
    w.Write(result.Data)
}
```

### Alert Endpoints

```go
// Create/Update alert
func (s *Server) handleSaveAlert(w http.ResponseWriter, r *http.Request) {
    var rule alerts.AlertRule
    if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := s.reportService.SaveAlertRule(&rule); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(rule)
}

// Test alert
func (s *Server) handleTestAlert(w http.ResponseWriter, r *http.Request) {
    ruleID := chi.URLParam(r, "id")

    result, err := s.reportService.TestAlert(ruleID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(result)
}
```

### Materialization Endpoints

```go
// Get with cache
func (s *Server) handleGetReportCached(w http.ResponseWriter, r *http.Request) {
    reportID := chi.URLParam(r, "id")
    maxAgeStr := r.URL.Query().Get("maxAge") // in minutes

    maxAge := 30 * time.Minute
    if maxAgeStr != "" {
        minutes, _ := strconv.Atoi(maxAgeStr)
        maxAge = time.Duration(minutes) * time.Minute
    }

    var req struct {
        Filters map[string]interface{} `json:"filters"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    resp, err := s.reportService.GetReportWithCache(
        reportID,
        req.Filters,
        maxAge,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(resp)
}

// Invalidate cache
func (s *Server) handleInvalidateCache(w http.ResponseWriter, r *http.Request) {
    reportID := chi.URLParam(r, "id")

    if err := s.reportService.InvalidateCache(reportID); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

---

## Frontend Integration Examples

### TypeScript API Client

```typescript
// Export
export async function exportReport(
  reportId: string,
  format: 'csv' | 'xlsx' | 'pdf',
  options?: {
    componentIds?: string[]
    filters?: Record<string, any>
    title?: string
    author?: string
  }
): Promise<Blob> {
  const response = await fetch(`/api/reports/${reportId}/export?format=${format}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options),
  })

  if (!response.ok) throw new Error('Export failed')
  return response.blob()
}

// Download helper
function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

// Usage
const blob = await exportReport('report-123', 'csv')
downloadBlob(blob, 'report.csv')
```

```typescript
// Alerts
export async function createAlert(rule: AlertRule): Promise<AlertRule> {
  const response = await fetch('/api/alerts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  })
  return response.json()
}

export async function testAlert(ruleId: string): Promise<AlertResult> {
  const response = await fetch(`/api/alerts/${ruleId}/test`)
  return response.json()
}

// Usage
const rule: AlertRule = {
  reportId: 'report-123',
  componentId: 'comp-456',
  name: 'High Error Rate',
  condition: {
    metric: 'error_rate',
    operator: '>',
    threshold: 5,
  },
  actions: [{ type: 'email', target: 'team@company.com' }],
  schedule: '*/15 * * * *',
  enabled: true,
}
await createAlert(rule)
```

```typescript
// Materialization
export async function getReportWithCache(
  reportId: string,
  maxAgeMinutes?: number
): Promise<ReportResult> {
  const params = new URLSearchParams()
  if (maxAgeMinutes) params.set('maxAge', maxAgeMinutes.toString())

  const response = await fetch(`/api/reports/${reportId}/cached?${params}`)
  return response.json()
}

export async function invalidateCache(reportId: string): Promise<void> {
  await fetch(`/api/reports/${reportId}/cache`, { method: 'DELETE' })
}

// Usage
const report = await getReportWithCache('report-123', 30) // Max 30 min old
```

---

## Cron Schedule Examples

```
"*/5 * * * *"     // Every 5 minutes
"0 * * * *"       // Every hour
"0 */6 * * *"     // Every 6 hours
"0 0 * * *"       // Daily at midnight
"0 6 * * *"       // Daily at 6 AM
"0 0 * * 0"       // Weekly on Sunday
"0 0 1 * *"       // Monthly on the 1st
"0 9 * * 1-5"     // Weekdays at 9 AM
```

---

## Common Patterns

### Export on Schedule

```go
// In app initialization, schedule daily export
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        result, err := reportService.ExportReport(
            "daily-report",
            export.FormatPDF,
            nil, nil,
            export.ExportOptions{Title: "Daily Report"},
        )
        if err == nil {
            saveToS3(result.Data, result.Filename)
        }
    }
}()
```

### Alert with Email

```go
rule := &alerts.AlertRule{
    Name: "Critical System Error",
    Condition: alerts.AlertCondition{
        Metric:    "error_count",
        Operator:  ">",
        Threshold: 100,
    },
    Actions: []alerts.AlertAction{
        {Type: "email", Target: "oncall@company.com"},
    },
    Schedule: "*/5 * * * *",
}
```

### Pre-warm Cache on Startup

```go
// In app startup
popularReports := []string{"dashboard", "sales", "performance"}
for _, reportID := range popularReports {
    go func(id string) {
        reportService.MaterializeReport(id, nil, 4*time.Hour)
    }(reportID)
}
```

---

## Troubleshooting

### Export fails with "too large"
- Reduce component count
- Add filters to limit rows
- Use CSV instead of PDF

### Alert not triggering
- Check schedule syntax (use https://crontab.guru)
- Verify component query returns data
- Check metric column exists

### Snapshot not being used
- Verify filter values match exactly
- Check maxAge parameter
- Confirm snapshot hasn't expired

---

## Performance Tips

1. **Export**: Use CSV for > 10k rows
2. **Alerts**: Cache component results for 5 minutes
3. **Materialization**: Schedule before peak hours (e.g., 6 AM)
4. **All**: Monitor query duration and optimize slow components

---

That's it! You now have all three features ready to use. 🚀
