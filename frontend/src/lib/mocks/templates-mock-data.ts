/**
 * Mock Data for Templates & Schedules
 * Use this for testing and development
 */

import type {
  QueryResult,
  QuerySchedule,
  QueryTemplate,
  ScheduleExecution} from '@/types/templates'

export const mockTemplates: QueryTemplate[] = [
  {
    id: 'tpl-1',
    name: 'Daily Sales Report',
    description: 'Generate a comprehensive sales report for a given date range',
    sql_template: `
SELECT
  DATE(order_date) as date,
  COUNT(*) as total_orders,
  SUM(total_amount) as revenue,
  AVG(total_amount) as avg_order_value,
  COUNT(DISTINCT customer_id) as unique_customers
FROM orders
WHERE order_date BETWEEN {{startDate}} AND {{endDate}}
  AND status = {{status}}
GROUP BY DATE(order_date)
ORDER BY date DESC
    `.trim(),
    parameters: [
      {
        name: 'startDate',
        type: 'date',
        required: true,
        description: 'Start date for the report',
      },
      {
        name: 'endDate',
        type: 'date',
        required: true,
        description: 'End date for the report',
      },
      {
        name: 'status',
        type: 'string',
        required: false,
        default: 'completed',
        description: 'Order status to filter',
        validation: {
          options: ['pending', 'completed', 'cancelled', 'refunded'],
        },
      },
    ],
    tags: ['sales', 'reporting', 'daily'],
    category: 'reporting',
    created_by: 'user-1',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-20T14:30:00Z',
    is_public: true,
    usage_count: 145,
  },
  {
    id: 'tpl-2',
    name: 'User Activity Analysis',
    description: 'Analyze user engagement metrics and activity patterns',
    sql_template: `
SELECT
  user_id,
  COUNT(DISTINCT DATE(event_timestamp)) as active_days,
  COUNT(*) as total_events,
  MAX(event_timestamp) as last_activity,
  array_agg(DISTINCT event_type) as event_types
FROM user_events
WHERE event_timestamp >= NOW() - INTERVAL '{{days}} days'
  AND event_type != 'page_view'
GROUP BY user_id
HAVING COUNT(*) >= {{minEvents}}
ORDER BY total_events DESC
LIMIT {{limit}}
    `.trim(),
    parameters: [
      {
        name: 'days',
        type: 'number',
        required: false,
        default: 30,
        description: 'Number of days to look back',
        validation: { min: 1, max: 365 },
      },
      {
        name: 'minEvents',
        type: 'number',
        required: false,
        default: 10,
        description: 'Minimum events to include user',
        validation: { min: 1 },
      },
      {
        name: 'limit',
        type: 'number',
        required: false,
        default: 100,
        description: 'Maximum users to return',
        validation: { min: 1, max: 1000 },
      },
    ],
    tags: ['analytics', 'users', 'engagement'],
    category: 'analytics',
    created_by: 'user-2',
    created_at: '2024-01-10T09:00:00Z',
    updated_at: '2024-01-18T11:00:00Z',
    is_public: true,
    usage_count: 89,
  },
  {
    id: 'tpl-3',
    name: 'Database Table Statistics',
    description: 'Get size and row count statistics for all tables',
    sql_template: `
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
  pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) as index_size,
  n_live_tup as row_count
FROM pg_stat_user_tables
WHERE schemaname = {{schema}}
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    `.trim(),
    parameters: [
      {
        name: 'schema',
        type: 'string',
        required: false,
        default: 'public',
        description: 'Database schema to analyze',
      },
    ],
    tags: ['maintenance', 'database', 'monitoring'],
    category: 'maintenance',
    created_by: 'user-1',
    created_at: '2024-01-05T08:00:00Z',
    updated_at: '2024-01-05T08:00:00Z',
    is_public: true,
    usage_count: 234,
  },
  {
    id: 'tpl-4',
    name: 'Customer Cohort Analysis',
    description: 'Analyze customer retention by signup cohort',
    sql_template: `
WITH cohorts AS (
  SELECT
    user_id,
    DATE_TRUNC('month', signup_date) as cohort_month
  FROM users
  WHERE signup_date >= {{startDate}}
)
SELECT
  c.cohort_month,
  COUNT(DISTINCT c.user_id) as cohort_size,
  COUNT(DISTINCT CASE WHEN o.order_date >= c.cohort_month + INTERVAL '1 month' THEN c.user_id END) as month_1_retained,
  COUNT(DISTINCT CASE WHEN o.order_date >= c.cohort_month + INTERVAL '3 months' THEN c.user_id END) as month_3_retained,
  COUNT(DISTINCT CASE WHEN o.order_date >= c.cohort_month + INTERVAL '6 months' THEN c.user_id END) as month_6_retained
FROM cohorts c
LEFT JOIN orders o ON c.user_id = o.user_id
GROUP BY c.cohort_month
ORDER BY c.cohort_month DESC
    `.trim(),
    parameters: [
      {
        name: 'startDate',
        type: 'date',
        required: true,
        description: 'Earliest cohort to include',
      },
    ],
    tags: ['analytics', 'retention', 'cohort'],
    category: 'analytics',
    created_by: 'user-2',
    created_at: '2024-01-12T15:00:00Z',
    updated_at: '2024-01-19T16:00:00Z',
    is_public: true,
    usage_count: 67,
  },
  {
    id: 'tpl-5',
    name: 'Slow Query Detector',
    description: 'Find queries that are taking longer than expected',
    sql_template: `
SELECT
  query,
  calls,
  mean_exec_time,
  max_exec_time,
  total_exec_time,
  stddev_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > {{threshold}}
  AND calls > {{minCalls}}
ORDER BY mean_exec_time DESC
LIMIT {{limit}}
    `.trim(),
    parameters: [
      {
        name: 'threshold',
        type: 'number',
        required: false,
        default: 1000,
        description: 'Minimum avg execution time (ms)',
        validation: { min: 0 },
      },
      {
        name: 'minCalls',
        type: 'number',
        required: false,
        default: 10,
        description: 'Minimum number of calls',
        validation: { min: 1 },
      },
      {
        name: 'limit',
        type: 'number',
        required: false,
        default: 50,
        description: 'Maximum queries to return',
        validation: { min: 1, max: 100 },
      },
    ],
    tags: ['maintenance', 'performance', 'monitoring'],
    category: 'maintenance',
    created_by: 'user-1',
    created_at: '2024-01-08T12:00:00Z',
    updated_at: '2024-01-16T13:00:00Z',
    is_public: true,
    usage_count: 178,
  },
  {
    id: 'tpl-6',
    name: 'Revenue Forecast',
    description: 'Project revenue based on historical trends',
    sql_template: `
SELECT
  DATE_TRUNC('month', order_date) as month,
  SUM(total_amount) as revenue,
  COUNT(*) as orders,
  AVG(total_amount) as avg_order_value
FROM orders
WHERE order_date >= NOW() - INTERVAL '{{months}} months'
  AND status = 'completed'
GROUP BY DATE_TRUNC('month', order_date)
ORDER BY month DESC
    `.trim(),
    parameters: [
      {
        name: 'months',
        type: 'number',
        required: false,
        default: 12,
        description: 'Months of history to analyze',
        validation: { min: 1, max: 36 },
      },
    ],
    tags: ['reporting', 'revenue', 'forecast'],
    category: 'reporting',
    created_by: 'user-2',
    created_at: '2024-01-14T11:00:00Z',
    updated_at: '2024-01-14T11:00:00Z',
    is_public: false,
    usage_count: 42,
  },
]

export const mockSchedules: QuerySchedule[] = [
  {
    id: 'sch-1',
    template_id: 'tpl-1',
    name: 'Daily Sales Report - Morning',
    frequency: '0 9 * * *', // Daily at 9 AM
    parameters: {
      startDate: '2024-01-01',
      endDate: '2024-12-31',
      status: 'completed',
    },
    last_run_at: '2024-01-23T09:00:00Z',
    next_run_at: '2024-01-24T09:00:00Z',
    status: 'active',
    created_by: 'user-1',
    notification_email: 'reports@company.com',
    result_storage: 'database',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 'sch-2',
    template_id: 'tpl-2',
    name: 'Weekly User Activity Analysis',
    frequency: '0 9 * * 1', // Every Monday at 9 AM
    parameters: {
      days: 7,
      minEvents: 5,
      limit: 200,
    },
    last_run_at: '2024-01-22T09:00:00Z',
    next_run_at: '2024-01-29T09:00:00Z',
    status: 'active',
    created_by: 'user-2',
    notification_email: 'analytics@company.com',
    result_storage: 's3',
    created_at: '2024-01-10T14:00:00Z',
    updated_at: '2024-01-10T14:00:00Z',
  },
  {
    id: 'sch-3',
    template_id: 'tpl-3',
    name: 'Database Stats - Monthly',
    frequency: '0 0 1 * *', // 1st of every month at midnight
    parameters: {
      schema: 'public',
    },
    last_run_at: '2024-01-01T00:00:00Z',
    next_run_at: '2024-02-01T00:00:00Z',
    status: 'paused',
    created_by: 'user-1',
    notification_email: 'devops@company.com',
    result_storage: 'database',
    created_at: '2024-01-05T08:00:00Z',
    updated_at: '2024-01-20T12:00:00Z',
  },
  {
    id: 'sch-4',
    template_id: 'tpl-5',
    name: 'Slow Query Check - Hourly',
    frequency: '0 * * * *', // Every hour
    parameters: {
      threshold: 2000,
      minCalls: 20,
      limit: 25,
    },
    last_run_at: '2024-01-23T14:00:00Z',
    next_run_at: '2024-01-23T15:00:00Z',
    status: 'active',
    created_by: 'user-1',
    notification_email: 'alerts@company.com',
    result_storage: 'none',
    created_at: '2024-01-08T12:00:00Z',
    updated_at: '2024-01-16T13:00:00Z',
  },
]

export const mockExecutions: ScheduleExecution[] = [
  {
    id: 'exec-1',
    schedule_id: 'sch-1',
    executed_at: '2024-01-23T09:00:00Z',
    status: 'success',
    duration_ms: 234,
    rows_returned: 365,
  },
  {
    id: 'exec-2',
    schedule_id: 'sch-1',
    executed_at: '2024-01-22T09:00:00Z',
    status: 'success',
    duration_ms: 198,
    rows_returned: 365,
  },
  {
    id: 'exec-3',
    schedule_id: 'sch-1',
    executed_at: '2024-01-21T09:00:00Z',
    status: 'failed',
    duration_ms: 5421,
    error_message: 'Query timeout: execution exceeded 5000ms limit',
  },
  {
    id: 'exec-4',
    schedule_id: 'sch-2',
    executed_at: '2024-01-22T09:00:00Z',
    status: 'success',
    duration_ms: 1876,
    rows_returned: 1543,
  },
  {
    id: 'exec-5',
    schedule_id: 'sch-4',
    executed_at: '2024-01-23T14:00:00Z',
    status: 'success',
    duration_ms: 89,
    rows_returned: 12,
  },
]

export const mockQueryResult: QueryResult = {
  columns: ['date', 'total_orders', 'revenue', 'avg_order_value', 'unique_customers'],
  rows: [
    {
      date: '2024-01-23',
      total_orders: 156,
      revenue: 12450.00,
      avg_order_value: 79.81,
      unique_customers: 134,
    },
    {
      date: '2024-01-22',
      total_orders: 189,
      revenue: 15230.50,
      avg_order_value: 80.58,
      unique_customers: 167,
    },
    {
      date: '2024-01-21',
      total_orders: 142,
      revenue: 11890.25,
      avg_order_value: 83.73,
      unique_customers: 128,
    },
  ],
  rowCount: 3,
  executionTime: 234,
}

/**
 * Initialize mock store for testing
 */
export function initializeMockStore() {
  // This would be used in development/testing to populate the store
  return {
    templates: mockTemplates,
    schedules: mockSchedules,
    executions: new Map([
      ['sch-1', mockExecutions.filter(e => e.schedule_id === 'sch-1')],
      ['sch-2', mockExecutions.filter(e => e.schedule_id === 'sch-2')],
      ['sch-4', mockExecutions.filter(e => e.schedule_id === 'sch-4')],
    ]),
  }
}
