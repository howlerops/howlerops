# Frequently Asked Questions

Common questions and answers about SQL Studio.

## General

### What databases does SQL Studio support?

SQL Studio supports:
- PostgreSQL (9.6+)
- MySQL (5.7+)
- SQLite (3+)
- MariaDB (10.2+)
- Microsoft SQL Server (2016+)
- Oracle Database (12c+)
- MongoDB (4.0+)
- Amazon Redshift
- Google BigQuery
- Snowflake

### Is my data secure?

Yes! Security is our top priority:
- All connections use SSL/TLS encryption
- Passwords are encrypted at rest
- Two-factor authentication available
- SOC 2 Type II certified
- GDPR compliant
- Regular security audits

### Can I use SQL Studio offline?

The desktop application works offline for:
- Local databases (SQLite)
- Previously cached connections
- Saved queries and templates

Cloud sync requires internet connection.

### How much does SQL Studio cost?

See our [pricing page](https://sqlstudio.com/pricing) for current plans:
- **Free**: Personal use, unlimited queries
- **Pro**: $15/month - Advanced features, cloud sync
- **Team**: $35/user/month - Team collaboration
- **Enterprise**: Custom pricing - SSO, dedicated support

## Connections

### How do I connect to a remote database?

1. Ensure your database allows remote connections
2. Note the host, port, database name, username, and password
3. In SQL Studio, create a new connection
4. Enter the connection details
5. Test the connection
6. Save

For cloud databases, you may need to:
- Whitelist SQL Studio's IP addresses
- Enable SSL/TLS
- Create a dedicated database user

### Why can't I connect to my database?

Common issues:
- **Wrong credentials**: Double-check username/password
- **Network**: Firewall blocking connection
- **SSL required**: Enable SSL in connection settings
- **Wrong port**: Verify port number (PostgreSQL: 5432, MySQL: 3306)
- **Database doesn't exist**: Check database name spelling

### Can I use SSH tunneling?

Yes! SQL Studio supports SSH tunnels:
1. Enable "Use SSH Tunnel" in connection settings
2. Enter SSH host, port, username
3. Choose authentication: Password or SSH key
4. Test and save

### How do I share a connection with my team?

1. Open the connection
2. Click "Share"
3. Select team members or "Organization"
4. Choose permissions: Read-only or Full access
5. Click "Share"

Credentials are never exposed to other users.

## Queries

### How do I save a query?

1. Write your query
2. Click "Save" or press `Cmd/Ctrl + S`
3. Enter a name
4. Choose a folder (optional)
5. Click "Save"

### Can I run multiple queries at once?

Yes! Separate queries with semicolons:
```sql
SELECT * FROM users;
SELECT * FROM orders;
SELECT * FROM products;
```

All queries will execute sequentially.

### How do I export query results?

1. Run your query
2. Click the export button in results panel
3. Choose format: CSV, JSON, or Excel
4. Select options (include headers, delimiter, etc.)
5. Download

### What's the maximum result size?

- **Web**: 10,000 rows
- **Desktop**: 100,000 rows
- **Enterprise**: Unlimited

For larger datasets, use export or pagination.

### How do I format my SQL?

- **Auto-format**: Right-click > "Format SQL" or `Shift+Alt+F`
- **Settings**: Customize formatting preferences in Settings > Editor

## Templates

### What are query templates?

Templates are reusable queries with parameters. Instead of hardcoding values, use placeholders:

```sql
SELECT * FROM users WHERE status = {{status}} AND created_at > {{date}}
```

When running, you'll be prompted to enter values.

### How do I create a template?

1. Write a query with parameter placeholders: `{{parameter_name}}`
2. Click "Save as Template"
3. Configure each parameter:
   - Type (text, number, date, select, etc.)
   - Default value
   - Validation rules
4. Save

### Can I share templates?

Yes! Templates can be shared with:
- Specific team members
- Your entire organization
- The public template library (opt-in)

### Where can I find pre-built templates?

Browse the Template Library:
1. Click "Templates" in sidebar
2. Select "Library"
3. Filter by category or database type
4. Click "Use Template" to copy to your workspace

## Scheduling

### How do I schedule a query?

1. Open a saved query or template
2. Click "Schedule"
3. Set frequency using cron expression or visual builder
4. Choose output destination (email, Slack, webhook, etc.)
5. Save schedule

### What cron expression format is used?

Standard cron format: `minute hour day month weekday`

Examples:
- `0 9 * * *` - Daily at 9:00 AM
- `0 9 * * 1-5` - Weekdays at 9:00 AM
- `0 */6 * * *` - Every 6 hours
- `0 0 1 * *` - First day of month

Use the visual builder if you prefer not to write cron expressions.

### Can I receive results via email?

Yes! Configure email delivery in schedule settings:
- Send to multiple addresses
- Include results as attachment
- Format: CSV, Excel, or inline table
- Conditional sending (only if results not empty)

### What happens if a scheduled query fails?

- You'll receive an email notification
- Error details logged in schedule history
- Query will retry based on your settings
- Schedules can be paused automatically after N failures

## Teams & Collaboration

### How do I create an organization?

1. Click your profile picture
2. Select "Create Organization"
3. Enter organization name
4. Invite team members
5. Done!

### How do I invite team members?

1. Go to Organization Settings
2. Click "Invite Members"
3. Enter email addresses (comma-separated)
4. Assign roles
5. Send invitations

### What are the different roles?

- **Owner**: Full control, can delete organization
- **Admin**: Manage members, billing, and settings
- **Member**: Create and share queries, use shared resources
- **Viewer**: Read-only access to shared resources

### How do I transfer ownership?

1. Go to Organization Settings > Members
2. Click menu next to member
3. Select "Transfer Ownership"
4. Confirm transfer

Only current owner can transfer ownership.

### Can I audit team activity?

Yes! Admins and owners can view:
- Query execution history
- Resource access logs
- Permission changes
- Member activity

Go to Organization Settings > Audit Log.

## Cloud Sync

### What gets synced?

You can choose to sync:
- Saved queries
- Query templates
- Connections (credentials encrypted)
- Editor settings and preferences
- Keyboard shortcuts

### How often does sync happen?

- **Real-time**: Changes sync immediately when online
- **Offline**: Queued and synced when connection restored
- **Manual**: Force sync from Settings > Cloud Sync

### What if there's a conflict?

Conflicts are resolved using "last write wins":
- The most recent change is kept
- Previous version available in history
- Manual resolution for complex conflicts

### Can I disable sync for specific items?

Yes! In Settings > Cloud Sync, toggle sync for:
- Individual queries
- Folders
- Connections
- Settings categories

## AI Assistant

### How accurate is the AI?

The AI assistant is trained on SQL best practices and is accurate for most common queries. However:
- Always review generated queries
- Test on sample data first
- Understand what the query does
- AI may not know your specific schema

### What can the AI help with?

- Generate queries from natural language
- Explain existing queries
- Optimize slow queries
- Fix syntax errors
- Suggest improvements
- Write documentation

### Does AI see my data?

No! The AI only sees:
- Your query text
- Table and column names (schema)
- Error messages

Actual data is never sent to AI.

### Can I disable AI features?

Yes, in Settings > AI:
- Disable AI entirely
- Disable specific features
- Choose which data to share

## Troubleshooting

### Query is running slow

Try:
1. Add indexes to filtered columns
2. Use EXPLAIN to see execution plan
3. Limit result size
4. Avoid SELECT *
5. Check for missing JOIN conditions

### Can't see my tables

Possible causes:
- Wrong schema/database selected
- No permissions to view tables
- Tables in different schema

Solution:
- Check selected schema in sidebar
- Verify database permissions
- Refresh schema cache

### Results not updating

- Click refresh icon in results panel
- Clear query cache: Settings > Advanced > Clear Cache
- Restart SQL Studio

### License/billing issues

Contact billing@sqlstudio.com with:
- Your account email
- Invoice number (if applicable)
- Description of issue

## Still Have Questions?

- **Community Forum**: [community.sqlstudio.com](https://community.sqlstudio.com)
- **Email Support**: [support@sqlstudio.com](mailto:support@sqlstudio.com)
- **Live Chat**: Available in-app (Pro and above)
- **Documentation**: [docs.sqlstudio.com](https://docs.sqlstudio.com)
- **Video Tutorials**: [sqlstudio.com/videos](https://sqlstudio.com/videos)
