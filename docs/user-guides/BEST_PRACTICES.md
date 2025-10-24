# Best Practices

Tips and techniques for getting the most out of SQL Studio.

## Query Writing

### Use Meaningful Names

**Good:**
```sql
SELECT
  customer_id,
  total_purchase_amount,
  last_order_date
FROM customer_summary;
```

**Bad:**
```sql
SELECT a, b, c FROM t1;
```

### Format for Readability

- Use consistent indentation (2 or 4 spaces)
- One clause per line for complex queries
- Align keywords vertically
- Use comments to explain complex logic

**Example:**
```sql
-- Get high-value customers from last quarter
SELECT
  c.customer_id,
  c.name,
  c.email,
  SUM(o.total) as total_spent,
  COUNT(o.order_id) as order_count
FROM customers c
  INNER JOIN orders o
    ON c.customer_id = o.customer_id
WHERE o.created_at >= DATE_TRUNC('quarter', CURRENT_DATE - INTERVAL '3 months')
  AND o.created_at < DATE_TRUNC('quarter', CURRENT_DATE)
GROUP BY c.customer_id, c.name, c.email
HAVING SUM(o.total) > 1000
ORDER BY total_spent DESC;
```

### Avoid SELECT *

Specify only the columns you need:
- Faster queries
- Clearer intent
- Less data transferred

### Use Appropriate Limits

When exploring data, use `LIMIT`:
```sql
SELECT * FROM large_table LIMIT 100;
```

### Parameterize Repeated Queries

Instead of changing values manually, create a template with parameters.

---

## Organization

### Folder Structure

Organize saved queries logically:

```
📁 Analytics
  📁 Sales
    - Daily Sales Report
    - Monthly Revenue
  📁 Users
    - Active Users
    - User Growth
📁 Operations
  📁 Monitoring
    - System Health
    - Error Logs
📁 Data Quality
  - Duplicate Check
  - Missing Data
```

### Naming Conventions

**Queries:**
- Descriptive and specific
- Include time period if relevant
- Use consistent casing

Examples:
- "Active Users - Last 30 Days"
- "Sales by Region - Monthly"
- "Inventory Low Stock Alert"

**Templates:**
- Start with category
- Include "Template" suffix

Examples:
- "Analytics - User Cohort Template"
- "Report - Sales Summary Template"

### Use Tags

Tag queries for easy filtering:
- `#monthly-report`
- `#data-quality`
- `#high-priority`
- `#needs-review`

---

## Performance

### Index Awareness

Know which columns are indexed:
- Filter (WHERE) on indexed columns
- Join on indexed columns
- Avoid functions on indexed columns

### Avoid N+1 Queries

**Bad:**
```sql
-- Run this 100 times for each user
SELECT * FROM orders WHERE user_id = ?
```

**Good:**
```sql
-- Run once
SELECT * FROM orders WHERE user_id IN (1, 2, 3, ...)
```

### Use JOINs Instead of Subqueries

When possible, JOINs are often faster:

**Slower:**
```sql
SELECT *
FROM users
WHERE id IN (SELECT user_id FROM orders);
```

**Faster:**
```sql
SELECT DISTINCT u.*
FROM users u
INNER JOIN orders o ON u.id = o.user_id;
```

### Monitor Query Execution Time

- Review slow query alerts
- Optimize frequently-run queries
- Use EXPLAIN to understand execution plans

---

## Team Collaboration

### Document Complex Queries

Add comments explaining:
- What the query does
- Why specific logic is used
- Known limitations
- Expected results

```sql
-- Monthly Active Users (MAU) Report
-- Counts unique users who performed any action in the last 30 days
-- Note: Does not include API-only users
-- Expected: ~50,000 users
SELECT COUNT(DISTINCT user_id) as monthly_active_users
FROM user_activities
WHERE activity_date >= CURRENT_DATE - INTERVAL '30 days'
  AND user_type != 'api';
```

### Share Instead of Duplicate

- Share queries with your team
- Use templates for common patterns
- Update shared queries instead of creating new versions

### Use Version Control for Critical Queries

- Save versions before major changes
- Review version history regularly
- Restore previous versions if needed

### Communicate Changes

When modifying shared queries:
- Leave a comment explaining changes
- Notify team members if behavior changes
- Use descriptive version names

---

## Security

### Never Hardcode Credentials

**Bad:**
```sql
COPY data FROM 's3://bucket/file' CREDENTIALS 'aws_access_key_id=AKIA...;aws_secret_access_key=...'
```

**Good:**
Use SQL Studio's secure credential storage or environment variables.

### Limit Access Appropriately

- Use read-only connections for reporting
- Grant minimum necessary permissions
- Review access regularly

### Sanitize User Input

When using query parameters:
- Use typed parameters (not raw strings)
- Validate input ranges
- Escape special characters

### Encrypt Sensitive Data

- Enable connection encryption (SSL/TLS)
- Use encrypted connections for cloud databases
- Enable cloud sync encryption

---

## Workflow Efficiency

### Keyboard-First Approach

Learn keyboard shortcuts:
- `Cmd/Ctrl + K` for quick search
- `Cmd/Ctrl + Enter` to run queries
- `Cmd/Ctrl + /` to comment
- `?` for help

### Use Command Palette

Press `Cmd/Ctrl + Shift + P` to access any action:
- Run query
- Save query
- Switch connections
- Format SQL

### Leverage Auto-Completion

- Let the editor suggest table names
- Use snippets for common patterns
- Accept suggestions with Tab

### Create Snippets

Save frequently-used patterns as snippets:
- Window functions
- Date calculations
- Common JOINs
- CTEs

---

## Data Exploration

### Start Broad, Then Narrow

```sql
-- 1. See table structure
DESCRIBE users;

-- 2. Get sample data
SELECT * FROM users LIMIT 10;

-- 3. Check data distribution
SELECT status, COUNT(*) FROM users GROUP BY status;

-- 4. Dive deeper
SELECT * FROM users WHERE status = 'active' AND created_at > '2024-01-01';
```

### Use Aggregates to Understand Data

```sql
SELECT
  COUNT(*) as total_rows,
  COUNT(DISTINCT email) as unique_emails,
  MIN(created_at) as earliest,
  MAX(created_at) as latest,
  AVG(age) as avg_age
FROM users;
```

### Sample Large Tables

```sql
-- PostgreSQL
SELECT * FROM large_table TABLESAMPLE SYSTEM (1);

-- MySQL
SELECT * FROM large_table ORDER BY RAND() LIMIT 1000;
```

---

## Maintenance

### Regular Cleanup

- Archive old queries you no longer use
- Delete duplicate queries
- Update outdated documentation
- Review and update templates

### Review Performance

Monthly review:
- Identify slow queries
- Optimize or cache results
- Update indexes if needed
- Clean up unused connections

### Update Connection Details

When database credentials change:
- Update connections immediately
- Test connections
- Notify team members if shared

---

## Learning & Growth

### Explore the Template Library

Browse pre-built templates to learn patterns.

### Study Query Execution Plans

Understand how databases execute your queries:
- Use EXPLAIN
- Identify table scans
- Look for index usage

### Join the Community

- Share your templates
- Learn from others
- Ask questions
- Contribute improvements

### Watch Video Tutorials

Regular video series on:
- SQL techniques
- Performance optimization
- New features
- Advanced workflows

---

## Anti-Patterns to Avoid

### Don't Use SELECT * in Production

Specify columns explicitly for:
- Performance
- Maintainability
- Future compatibility

### Don't Ignore Warnings

SQL Studio warns you about:
- Potential performance issues
- Syntax deprecations
- Security concerns

### Don't Share Passwords

Use SQL Studio's secure sharing:
- Share connections, not credentials
- Use read-only access when appropriate
- Revoke access when team members leave

### Don't Run Untested Queries on Production

- Test on development databases first
- Use transactions for data modifications
- Back up before major changes

---

## Remember

> "Code is read more often than it is written."

Write queries that:
- You can understand in 6 months
- Your colleagues can understand
- Are easy to modify
- Perform efficiently

Happy querying! 🚀
