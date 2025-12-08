# Query Performance Dashboard - Product Requirements Document

**Version:** 1.0
**Status:** Draft
**Author:** Engineering Team
**Date:** December 2024

---

## 1. Executive Summary

The Query Performance Dashboard enhances HowlerOps with comprehensive query performance monitoring capabilities. This feature provides users with actionable insights into query execution patterns, identifies performance bottlenecks, and enables data-driven optimization decisions across all supported database types.

**Key Value Proposition:**
- Identify slow queries before they impact production
- Understand query patterns across connections and databases
- Track performance trends over time
- Enable proactive database optimization

---

## 2. Problem Statement

### Current State
HowlerOps currently tracks basic query execution through `QueryHistory` (storage/types.go:71-85) and has foundational analytics in `internal/analytics/` with `QueryMetrics` and `DashboardService`. However:

1. Query metrics are collected but lack a dedicated performance-focused view
2. Slow query identification exists but lacks ranking and historical context
3. Per-connection performance breakdowns are incomplete
4. No trend visualization for identifying degradation over time
5. Existing analytics page (analytics-page.tsx) mixes general analytics with performance concerns

### User Pain Points
- **DBAs:** Cannot quickly identify queries consuming the most resources
- **Developers:** Lack visibility into how their queries perform over time
- **Operations:** No early warning for performance degradation
- **Enterprise teams:** Cannot attribute performance issues to specific connections or users

### Business Impact
- Slow queries go undetected until they cause outages
- Performance tuning is reactive rather than proactive
- Time wasted manually analyzing query logs
- No data to justify database optimization investments

---

## 3. Goals & Success Metrics

### Primary Goals
1. Surface slow queries within 5 clicks from any screen
2. Provide actionable performance insights without requiring DBA expertise
3. Scale to handle 10,000+ queries/day per organization

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Dashboard load time | < 2 seconds | P95 latency |
| Query identification accuracy | > 95% | Slow query detection vs manual analysis |
| User adoption | 60% of active users | Weekly active users visiting dashboard |
| Performance issue detection | < 5 min | Time from slow query to alert |
| Data retention | 90 days | Full performance history |

### Non-Goals (v1)
- Query plan analysis / EXPLAIN integration
- Automated query optimization suggestions
- Cross-organization benchmarking
- Real-time query monitoring (streaming)

---

## 4. User Stories

### Epic: Performance Visibility

**US-1: Slow Query Identification**
> As a developer, I want to see the slowest queries in the last 24 hours so I can prioritize optimization efforts.

Acceptance Criteria:
- [ ] List queries ordered by execution time (descending)
- [ ] Show execution time, rows returned, timestamp, connection
- [ ] Filter by time range (1h, 24h, 7d, 30d)
- [ ] Filter by connection/database
- [ ] Click to view full SQL and execution context

**US-2: Query Performance Trends**
> As a DBA, I want to see how query performance changes over time so I can identify degradation.

Acceptance Criteria:
- [ ] Line chart showing avg execution time over selected period
- [ ] Overlay P50, P95, P99 percentiles
- [ ] Highlight anomalies (> 2 standard deviations)
- [ ] Compare current period to previous period

**US-3: Per-Connection Analysis**
> As an operations engineer, I want to see performance metrics per database connection so I can identify problematic connections.

Acceptance Criteria:
- [ ] Table showing all connections with performance summary
- [ ] Columns: connection name, query count, avg time, error rate, P95
- [ ] Sort by any column
- [ ] Click to drill into connection-specific metrics

**US-4: Query Frequency Analysis**
> As a developer, I want to identify the most frequently executed queries so I can focus caching efforts.

Acceptance Criteria:
- [ ] List queries grouped by SQL hash
- [ ] Show execution count, avg time, success rate
- [ ] Time-based frequency chart (hourly/daily buckets)
- [ ] Filter by connection

**US-5: Error Analysis**
> As a developer, I want to see failed queries grouped by error type so I can fix common issues.

Acceptance Criteria:
- [ ] List recent errors with SQL, error message, timestamp
- [ ] Group by error pattern/type
- [ ] Show error rate trend over time
- [ ] Filter by connection

---

## 5. Technical Requirements

### 5.1 Backend (Go)

#### Data Model Extension

Existing `QueryExecution` (analytics/query_metrics.go:22-35) already captures needed fields. Add:

```go
// QueryPerformanceSummary aggregates performance for a time period
type QueryPerformanceSummary struct {
    Period           TimeRange         `json:"period"`
    TotalQueries     int64             `json:"total_queries"`
    UniqueQueries    int64             `json:"unique_queries"`
    AvgExecutionMs   float64           `json:"avg_execution_ms"`
    P50ExecutionMs   float64           `json:"p50_execution_ms"`
    P95ExecutionMs   float64           `json:"p95_execution_ms"`
    P99ExecutionMs   float64           `json:"p99_execution_ms"`
    ErrorRate        float64           `json:"error_rate"`
    TimeoutRate      float64           `json:"timeout_rate"`
    SlowQueryCount   int64             `json:"slow_query_count"`
    TopConnections   []ConnectionPerf  `json:"top_connections"`
}

// ConnectionPerf represents per-connection performance
type ConnectionPerf struct {
    ConnectionID   string  `json:"connection_id"`
    ConnectionName string  `json:"connection_name"`
    DatabaseType   string  `json:"database_type"`
    QueryCount     int64   `json:"query_count"`
    AvgTimeMs      float64 `json:"avg_time_ms"`
    P95TimeMs      float64 `json:"p95_time_ms"`
    ErrorRate      float64 `json:"error_rate"`
    SlowQueryCount int64   `json:"slow_query_count"`
}

// SlowQueryRanking ranks queries by impact
type SlowQueryRanking struct {
    SQLHash         string  `json:"sql_hash"`
    SQL             string  `json:"sql"`
    ConnectionID    string  `json:"connection_id"`
    ExecutionCount  int64   `json:"execution_count"`
    TotalTimeMs     int64   `json:"total_time_ms"`     // Sum of all executions
    AvgTimeMs       float64 `json:"avg_time_ms"`
    MaxTimeMs       int64   `json:"max_time_ms"`
    ImpactScore     float64 `json:"impact_score"`      // count * avg_time
}
```

#### New Service Methods

Extend `QueryMetrics` (analytics/query_metrics.go:17-20):

```go
// GetPerformanceSummary returns aggregated performance metrics
func (m *QueryMetrics) GetPerformanceSummary(ctx context.Context, orgID *string, period TimeRange) (*QueryPerformanceSummary, error)

// GetSlowQueryRanking returns queries ranked by performance impact
func (m *QueryMetrics) GetSlowQueryRanking(ctx context.Context, orgID *string, period TimeRange, limit int) ([]*SlowQueryRanking, error)

// GetConnectionPerformance returns performance metrics per connection
func (m *QueryMetrics) GetConnectionPerformance(ctx context.Context, orgID *string, period TimeRange) ([]*ConnectionPerf, error)

// GetPerformanceTrend returns time-series performance data
func (m *QueryMetrics) GetPerformanceTrend(ctx context.Context, orgID *string, period TimeRange, bucketSize time.Duration) ([]*TimeSeriesPoint, error)

// GetSlowQueryThreshold returns dynamic threshold based on percentiles
func (m *QueryMetrics) GetSlowQueryThreshold(ctx context.Context, orgID *string, period TimeRange) (int64, error)
```

#### Database Schema

Leverage existing `query_metrics` table (analytics/query_metrics.go:70-92). Add materialized views for performance:

```sql
-- Hourly aggregates for trend analysis (materialized hourly)
CREATE TABLE IF NOT EXISTS query_metrics_hourly (
    hour_bucket INTEGER NOT NULL,  -- Unix timestamp truncated to hour
    organization_id TEXT,
    connection_id TEXT NOT NULL,
    query_count INTEGER NOT NULL,
    success_count INTEGER NOT NULL,
    error_count INTEGER NOT NULL,
    timeout_count INTEGER NOT NULL,
    total_execution_ms INTEGER NOT NULL,
    min_execution_ms INTEGER NOT NULL,
    max_execution_ms INTEGER NOT NULL,
    PRIMARY KEY (hour_bucket, organization_id, connection_id)
);

CREATE INDEX IF NOT EXISTS idx_hourly_org ON query_metrics_hourly(organization_id);
CREATE INDEX IF NOT EXISTS idx_hourly_bucket ON query_metrics_hourly(hour_bucket);

-- Daily aggregates per SQL hash for ranking
CREATE TABLE IF NOT EXISTS query_metrics_daily (
    day_bucket INTEGER NOT NULL,  -- Unix timestamp truncated to day
    organization_id TEXT,
    sql_hash TEXT NOT NULL,
    sample_sql TEXT NOT NULL,  -- Store one sample
    connection_id TEXT NOT NULL,
    query_count INTEGER NOT NULL,
    success_count INTEGER NOT NULL,
    total_execution_ms INTEGER NOT NULL,
    max_execution_ms INTEGER NOT NULL,
    PRIMARY KEY (day_bucket, organization_id, sql_hash)
);

CREATE INDEX IF NOT EXISTS idx_daily_org ON query_metrics_daily(organization_id);
CREATE INDEX IF NOT EXISTS idx_daily_bucket ON query_metrics_daily(day_bucket);
CREATE INDEX IF NOT EXISTS idx_daily_total_time ON query_metrics_daily(total_execution_ms DESC);
```

### 5.2 API Design

Follow existing REST pattern from `dashboard.go:809-811`. Add dedicated performance endpoints:

```
GET /api/analytics/performance/summary
    Query: range=24h|7d|30d, connection_id?, org_id?
    Response: QueryPerformanceSummary

GET /api/analytics/performance/slow-queries
    Query: range=24h|7d|30d, threshold_ms?, limit=20, connection_id?
    Response: { queries: SlowQueryRanking[], threshold_ms: int }

GET /api/analytics/performance/connections
    Query: range=24h|7d|30d, sort=avg_time|query_count|error_rate
    Response: { connections: ConnectionPerf[] }

GET /api/analytics/performance/trends
    Query: range=24h|7d|30d, metric=avg_time|p95|error_rate, bucket=hour|day
    Response: { points: TimeSeriesPoint[] }

GET /api/analytics/performance/query/{sql_hash}
    Query: range=7d
    Response: { stats: QueryStats, executions: QueryExecution[], trend: TimeSeriesPoint[] }
```

### 5.3 Frontend (React/TypeScript)

#### New Components

```
frontend/src/pages/
  query-performance.tsx          # Main performance dashboard page

frontend/src/components/performance/
  PerformanceSummaryCards.tsx    # Overview stat cards (reuse StatCard pattern)
  SlowQueryTable.tsx             # Slow query ranking table
  ConnectionPerfTable.tsx        # Per-connection performance table
  PerformanceTrendChart.tsx      # Time-series trend visualization
  QueryDetailModal.tsx           # Drill-down modal for specific queries
```

#### Type Definitions

```typescript
// frontend/src/types/performance.ts
export interface QueryPerformanceSummary {
  period: { start: string; end: string }
  total_queries: number
  unique_queries: number
  avg_execution_ms: number
  p50_execution_ms: number
  p95_execution_ms: number
  p99_execution_ms: number
  error_rate: number
  timeout_rate: number
  slow_query_count: number
  top_connections: ConnectionPerf[]
}

export interface ConnectionPerf {
  connection_id: string
  connection_name: string
  database_type: string
  query_count: number
  avg_time_ms: number
  p95_time_ms: number
  error_rate: number
  slow_query_count: number
}

export interface SlowQueryRanking {
  sql_hash: string
  sql: string
  connection_id: string
  execution_count: number
  total_time_ms: number
  avg_time_ms: number
  max_time_ms: number
  impact_score: number
}

export type PerformanceTimeRange = '1h' | '24h' | '7d' | '30d'
```

#### Hooks

```typescript
// frontend/src/hooks/use-performance.ts
export function usePerformanceSummary(range: PerformanceTimeRange, connectionId?: string)
export function useSlowQueries(range: PerformanceTimeRange, limit?: number)
export function useConnectionPerformance(range: PerformanceTimeRange)
export function usePerformanceTrend(range: PerformanceTimeRange, metric: string)
```

---

## 6. UI/UX Wireframes

### 6.1 Dashboard Layout

```
+------------------------------------------------------------------+
| Query Performance                          [1h] [24h] [7d] [30d] |
+------------------------------------------------------------------+
| +------------+ +------------+ +------------+ +------------+       |
| | Total      | | Avg Time   | | P95 Time   | | Error Rate |       |
| | 15,234     | | 145ms      | | 423ms      | | 1.5%       |       |
| | +12.5%     | | -8.3%      | | +2.1%      | | -0.5%      |       |
| +------------+ +------------+ +------------+ +------------+       |
+------------------------------------------------------------------+
| Performance Trend                                                 |
| +--------------------------------------------------------------+ |
| |     /\                                                        | |
| |    /  \    /\      avg ___   p95 ---   p99 ...               | |
| |   /    \  /  \                                                | |
| |  /      \/    \___                                            | |
| +--------------------------------------------------------------+ |
| [Mon] [Tue] [Wed] [Thu] [Fri] [Sat] [Sun]                        |
+------------------------------------------------------------------+
| [Slow Queries] [By Connection] [Errors]                          |
+------------------------------------------------------------------+
| Slow Queries (ranked by impact)                                  |
| +--------------------------------------------------------------+ |
| | #  | Query                    | Count | Avg   | Max   | Impact|
| |----|--------------------------|-------|-------|-------|-------|
| | 1  | SELECT * FROM orders ... | 523   | 2.3s  | 8.1s  | HIGH  |
| | 2  | UPDATE inventory SET ... | 412   | 1.8s  | 4.2s  | MED   |
| | 3  | SELECT JOIN customers... | 387   | 1.5s  | 3.0s  | MED   |
| +--------------------------------------------------------------+ |
+------------------------------------------------------------------+
```

### 6.2 Query Detail Modal

```
+------------------------------------------------------------------+
| Query Performance Details                              [x] Close |
+------------------------------------------------------------------+
| SQL:                                                              |
| +--------------------------------------------------------------+ |
| | SELECT o.*, c.name, c.email                                   | |
| | FROM orders o                                                 | |
| | JOIN customers c ON o.customer_id = c.id                      | |
| | WHERE o.status = 'pending'                                    | |
| | ORDER BY o.created_at DESC                                    | |
| +--------------------------------------------------------------+ |
|                                                                   |
| Statistics (Last 7 days)                                          |
| +-------------------+  +-------------------+  +-------------------+|
| | Executions: 523   |  | Avg Time: 2.3s    |  | Success: 98.5%   ||
| +-------------------+  +-------------------+  +-------------------+|
|                                                                   |
| Execution History                                                 |
| +--------------------------------------------------------------+ |
| |  ^                                                            | |
| |  |    *     *                                                 | |
| |  | *    *  * *   *    *                                       | |
| +--------------------------------------------------------------+ |
|                                                                   |
| Recent Executions                                                 |
| +--------------------------------------------------------------+ |
| | Time        | Duration | Status  | Connection    | User      | |
| |-------------|----------|---------|---------------|-----------|
| | 2 min ago   | 2.1s     | success | production    | alice     | |
| | 15 min ago  | 8.1s     | success | production    | bob       | |
| +--------------------------------------------------------------+ |
+------------------------------------------------------------------+
```

---

## 7. Performance Requirements

### 7.1 Scalability Targets

| Metric | Requirement |
|--------|-------------|
| Queries/day | 10,000+ per organization |
| Historical data | 90 days retention |
| Dashboard refresh | < 2s response time |
| Concurrent users | 100 per organization |

### 7.2 Optimization Strategy

1. **Pre-aggregation:** Hourly/daily rollups via background job
2. **Pagination:** All list endpoints paginated (default: 20 items)
3. **Caching:** Summary data cached for 1 minute
4. **Sampling:** For high-volume periods, sample detailed data
5. **Indexing:** Targeted indexes on query patterns (see schema above)

### 7.3 Data Retention

| Data Type | Retention |
|-----------|-----------|
| Raw query metrics | 30 days |
| Hourly aggregates | 90 days |
| Daily aggregates | 1 year |

Background cleanup job (extend existing `CleanupOldMetrics`):
```go
// Run daily at 2 AM
func (m *QueryMetrics) CleanupAndAggregate(ctx context.Context) error {
    // 1. Aggregate yesterday's data to hourly/daily tables
    // 2. Delete raw data older than 30 days
    // 3. Delete hourly data older than 90 days
}
```

---

## 8. Implementation Plan

### Phase 1: Foundation (Week 1-2)
- [ ] Extend QueryMetrics with new methods
- [ ] Create aggregation tables and background job
- [ ] Implement REST endpoints
- [ ] Unit tests for new analytics functions

### Phase 2: Core Dashboard (Week 3-4)
- [ ] Create query-performance.tsx page
- [ ] Implement PerformanceSummaryCards component
- [ ] Implement SlowQueryTable with pagination
- [ ] Add navigation link in sidebar

### Phase 3: Trend Analysis (Week 5)
- [ ] Implement PerformanceTrendChart
- [ ] Add time-series data hooks
- [ ] Implement metric comparison (current vs previous period)

### Phase 4: Drill-down & Polish (Week 6)
- [ ] Implement QueryDetailModal
- [ ] Add ConnectionPerfTable
- [ ] Export functionality (CSV)
- [ ] Documentation and user guide

### Phase 5: Testing & Optimization (Week 7)
- [ ] Load testing with 10k queries/day
- [ ] Performance optimization
- [ ] Integration tests
- [ ] User acceptance testing

---

## 9. Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| High query volume impacts dashboard performance | High | Medium | Pre-aggregation, pagination, caching |
| Existing query_metrics table grows too large | Medium | High | Implement retention policy, partitioning if needed |
| SQL hash collisions | Low | Low | Use SHA-256, store sample SQL for verification |
| Cross-database compatibility issues | Medium | Low | Abstract queries behind service layer |
| User confusion about metrics | Medium | Medium | Tooltips, documentation, contextual help |

---

## 10. Dependencies

- **Existing Systems:**
  - `internal/analytics/query_metrics.go` - Base metrics collection
  - `internal/analytics/dashboard.go` - Dashboard service pattern
  - `pkg/storage/types.go` - QueryHistory type

- **Frontend Libraries:**
  - React Query (existing) - Data fetching
  - Recharts or similar - Charts (evaluate vs existing simple charts)
  - shadcn/ui (existing) - UI components

- **External:**
  - None required

---

## 11. Open Questions

1. Should slow query threshold be configurable per organization?
2. Do we need real-time alerts for slow queries? (defer to v2?)
3. Should we integrate with external monitoring (DataDog, etc.)?
4. How should we handle multi-statement queries?

---

## Appendix A: Existing Code References

- Query execution recording: `analytics/query_metrics.go:98-147`
- Slow query detection: `analytics/query_metrics.go:149-196`
- Dashboard data model: `analytics/dashboard.go:26-118`
- Existing analytics page: `frontend/src/pages/analytics-page.tsx`
- Query history type: `storage/types.go:71-85`
