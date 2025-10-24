# Feature Guides

Comprehensive guides for all SQL Studio features.

## Table of Contents

1. [Query Editor](#query-editor)
2. [Query Templates](#query-templates)
3. [Query Scheduling](#query-scheduling)
4. [Organizations](#organizations)
5. [Cloud Sync](#cloud-sync)
6. [AI Assistant](#ai-assistant)
7. [Performance Monitoring](#performance-monitoring)

---

## Query Editor

The Query Editor is where you write, execute, and analyze your SQL queries.

### Features

#### Syntax Highlighting
SQL keywords, functions, and data types are color-coded for better readability.

#### Smart Auto-Completion
As you type, the editor suggests:
- SQL keywords and functions
- Table names from your connected database
- Column names based on context
- Snippets for common patterns

**Trigger**: Start typing or press `Ctrl+Space`

#### Multiple Query Tabs
Work on multiple queries simultaneously:
- Click the `+` button to create a new tab
- Drag tabs to reorder
- Close tabs with the `×` button
- Unsaved changes are indicated with a dot

#### Run Query
Execute your SQL query:
- **Run All**: Click **Run** button or press `Cmd/Ctrl + Enter`
- **Run Selection**: Highlight specific lines and run
- **Run to Cursor**: Press `Cmd/Ctrl + Shift + Enter`

#### Results Panel
View and interact with query results:
- **Sort**: Click column headers
- **Filter**: Use the filter box above results
- **Copy**: Select cells and copy
- **Export**: Download as CSV, JSON, or Excel

#### Query History
Every executed query is saved:
- Access from sidebar
- Re-run with one click
- See execution time and date
- Filter by date or connection

---

## Query Templates

Create reusable query templates with parameters.

### Creating a Template

1. Write your query with parameters:
   ```sql
   SELECT * FROM users
   WHERE created_at >= {{start_date}}
     AND created_at <= {{end_date}}
     AND status = {{status}}
   ```

2. Click **Save as Template**

3. Configure parameters:
   - **start_date**: Type = Date, Default = "7 days ago"
   - **end_date**: Type = Date, Default = "today"
   - **status**: Type = Select, Options = "active, inactive, pending"

4. Save the template

### Using a Template

1. Open **Templates** from sidebar
2. Select a template
3. Fill in parameter values
4. Click **Run**

### Parameter Types

- **Text**: Free-form text input
- **Number**: Numeric values only
- **Date**: Date picker
- **DateTime**: Date and time picker
- **Select**: Dropdown with predefined options
- **Multi-Select**: Multiple selection dropdown
- **Boolean**: Checkbox (true/false)

### Template Library

Browse pre-built templates:
- User analytics
- Sales reports
- System monitoring
- Data quality checks

---

## Query Scheduling

Automate query execution on a schedule.

### Creating a Schedule

1. Open a saved query or template
2. Click **Schedule**
3. Configure:
   - **Frequency**: Cron expression or visual builder
   - **Time Zone**: Select your timezone
   - **Output**: Where to send results
   - **Notifications**: Email on success/failure

4. Click **Save Schedule**

### Cron Expression Examples

- `0 9 * * *` - Every day at 9:00 AM
- `0 9 * * 1-5` - Weekdays at 9:00 AM
- `0 */4 * * *` - Every 4 hours
- `0 0 1 * *` - First day of every month

### Output Options

- **Email**: Send results to specified addresses
- **Slack**: Post to a Slack channel
- **Webhook**: HTTP POST to your endpoint
- **S3 Bucket**: Upload to AWS S3
- **Google Sheets**: Append to spreadsheet

### Managing Schedules

View all scheduled queries:
- **Active/Paused**: Toggle status
- **Run Now**: Execute immediately
- **Edit**: Modify schedule
- **History**: See past executions

---

## Organizations

Collaborate with your team.

### Creating an Organization

1. Click your profile menu
2. Select **Create Organization**
3. Enter organization name
4. Click **Create**

### Inviting Members

1. Go to **Organization Settings**
2. Click **Invite Members**
3. Enter email addresses (comma-separated)
4. Select role:
   - **Owner**: Full access
   - **Admin**: Manage members and resources
   - **Member**: View and edit shared resources
   - **Viewer**: Read-only access

5. Click **Send Invitations**

### Sharing Resources

#### Sharing Queries
1. Open a saved query
2. Click **Share**
3. Select:
   - **Specific people**: Choose team members
   - **Organization**: All members can access
4. Set permissions: View or Edit
5. Click **Share**

#### Sharing Connections
1. Go to **Connections**
2. Click **Share** on a connection
3. Select members or organization
4. Set read-only or full access
5. Click **Share**

### Managing Permissions

**Owner** can:
- Everything

**Admin** can:
- Invite/remove members
- Share resources
- View audit logs

**Member** can:
- Create and share queries
- Use shared connections
- Collaborate on templates

**Viewer** can:
- View shared queries
- Run shared templates
- Cannot edit or share

---

## Cloud Sync

Access your work from anywhere.

### Enabling Cloud Sync

1. Go to **Settings** > **Cloud Sync**
2. Click **Enable Cloud Sync**
3. Choose what to sync:
   - Saved queries
   - Query templates
   - Connections (credentials encrypted)
   - Settings and preferences

4. Click **Save**

### How It Works

- **Automatic**: Changes sync in real-time
- **Offline**: Work offline, sync when reconnected
- **Conflicts**: Resolved with latest-write-wins
- **Encryption**: All data encrypted in transit and at rest

### Managing Devices

View connected devices:
- See last sync time
- Revoke access to lost devices
- Force sync across all devices

### Sync Status

Check sync status in the status bar:
- ✓ Synced
- ↻ Syncing...
- ⚠ Sync error (click to resolve)

---

## AI Assistant

Get intelligent help with your queries.

### Using AI Assistant

1. Click the AI icon in query editor
2. Choose an action:
   - **Explain Query**: Understand what a query does
   - **Optimize**: Get performance improvements
   - **Fix Errors**: Resolve syntax issues
   - **Generate**: Create query from description

### Natural Language Queries

Describe what you want in plain English:

> "Show me the top 10 customers by total purchase amount last month"

AI will generate:
```sql
SELECT
  c.customer_id,
  c.name,
  SUM(o.total) as total_purchased
FROM customers c
JOIN orders o ON c.customer_id = o.customer_id
WHERE o.created_at >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '1 month')
  AND o.created_at < DATE_TRUNC('month', CURRENT_DATE)
GROUP BY c.customer_id, c.name
ORDER BY total_purchased DESC
LIMIT 10;
```

### Query Explanations

AI explains what each part does:
- Line-by-line breakdown
- Visual diagram of JOINs
- Performance considerations
- Suggested improvements

### Optimization Tips

AI analyzes your query for:
- Missing indexes
- Inefficient JOINs
- Unnecessary subqueries
- Better alternatives

---

## Performance Monitoring

Track and optimize query performance.

### Query Execution Stats

For every query, see:
- **Execution Time**: How long it took
- **Rows Returned**: Number of results
- **Data Size**: Amount of data transferred
- **Execution Plan**: Database's query plan

### Slow Query Detection

Queries taking longer than threshold are flagged:
- View in **Slow Queries** panel
- Get optimization suggestions
- Compare with similar queries

### Performance Dashboard

Monitor overall performance:
- Average query time
- Most frequent queries
- Resource usage
- Error rates

### Explain Plan Visualization

Visual representation of query execution:
- See table scans vs. index usage
- Identify bottlenecks
- Compare estimated vs. actual costs

---

## Additional Resources

- [Best Practices](BEST_PRACTICES.md)
- [FAQ](FAQ.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [Video Tutorials](https://sqlstudio.com/videos)
