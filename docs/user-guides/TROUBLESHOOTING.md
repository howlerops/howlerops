# Troubleshooting Guide

Common issues and how to resolve them.

## Connection Issues

### Cannot connect to database

**Symptoms**: "Connection failed" or timeout errors

**Solutions**:

1. **Verify credentials**
   ```
   - Double-check username and password
   - Ensure database name is correct
   - Verify host and port
   ```

2. **Check network connectivity**
   ```bash
   # Test if database host is reachable
   ping your-database-host.com

   # Test if port is open
   telnet your-database-host.com 5432
   ```

3. **Firewall settings**
   - Whitelist Howlerops IP addresses
   - Check your organization's firewall rules
   - Verify cloud provider security groups

4. **SSL/TLS requirements**
   - Enable "Require SSL" in connection settings
   - Provide SSL certificates if required
   - Verify SSL mode (prefer, require, verify-ca, verify-full)

5. **Database permissions**
   ```sql
   -- PostgreSQL: Grant connection permission
   GRANT CONNECT ON DATABASE your_db TO your_user;

   -- MySQL: Grant access from specific host
   GRANT ALL PRIVILEGES ON your_db.* TO 'user'@'%';
   FLUSH PRIVILEGES;
   ```

### Connection drops frequently

**Solutions**:

1. **Increase connection timeout**
   - Settings > Connections > Advanced > Connection Timeout

2. **Enable keep-alive**
   - Settings > Connections > Advanced > Keep-Alive

3. **Check proxy settings**
   - Disable proxies if not needed
   - Configure proxy authentication

### "Too many connections" error

**Solutions**:

1. **Close unused connections**
   - Click Connections > Active Connections
   - Close idle connections

2. **Increase max connections on database**
   ```sql
   -- PostgreSQL
   ALTER SYSTEM SET max_connections = 200;

   -- MySQL
   SET GLOBAL max_connections = 200;
   ```

3. **Use connection pooling**
   - Settings > Connections > Enable Connection Pooling

---

## Query Issues

### Query runs indefinitely

**Solutions**:

1. **Cancel the query**
   - Click "Cancel" button
   - Or press `Cmd/Ctrl + .`

2. **Add LIMIT clause**
   ```sql
   SELECT * FROM large_table LIMIT 1000;
   ```

3. **Optimize query**
   - Add indexes to filtered columns
   - Use EXPLAIN to analyze execution plan
   - Avoid SELECT * on large tables

4. **Set query timeout**
   - Settings > Query > Execution Timeout

### Syntax errors

**Common fixes**:

```sql
-- Missing comma
SELECT name, email FROM users; -- ✓
SELECT name email FROM users;  -- ✗

-- Unclosed quote
SELECT * FROM users WHERE name = 'John'; -- ✓
SELECT * FROM users WHERE name = 'John;  -- ✗

-- Wrong keyword order
SELECT * FROM users WHERE age > 18 ORDER BY name; -- ✓
SELECT * FROM users ORDER BY name WHERE age > 18; -- ✗

-- Missing FROM clause
SELECT name, email FROM users; -- ✓
SELECT name, email;           -- ✗
```

### Out of memory errors

**Solutions**:

1. **Reduce result size**
   ```sql
   -- Add LIMIT
   SELECT * FROM large_table LIMIT 10000;

   -- Select specific columns
   SELECT id, name FROM large_table;
   ```

2. **Use pagination**
   ```sql
   SELECT * FROM large_table
   LIMIT 1000 OFFSET 0;
   ```

3. **Export large results**
   - Instead of viewing, export to file
   - Use streaming export for very large datasets

4. **Increase memory limit**
   - Settings > Advanced > Memory Limit

### Permission denied errors

**Solutions**:

```sql
-- PostgreSQL: Grant SELECT permission
GRANT SELECT ON table_name TO user_name;

-- PostgreSQL: Grant all permissions
GRANT ALL PRIVILEGES ON TABLE table_name TO user_name;

-- MySQL: Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON database.table TO 'user'@'host';

-- Check current permissions
-- PostgreSQL
\dp table_name

-- MySQL
SHOW GRANTS FOR 'user'@'host';
```

---

## Performance Issues

### Slow query execution

**Diagnosis**:

1. **Use EXPLAIN**
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM users WHERE email = 'user@example.com';
   ```

2. **Check execution plan**
   - Look for "Seq Scan" (table scan)
   - Should use "Index Scan" when possible

**Optimizations**:

1. **Add indexes**
   ```sql
   -- PostgreSQL
   CREATE INDEX idx_users_email ON users(email);

   -- MySQL
   CREATE INDEX idx_users_email ON users(email);
   ```

2. **Avoid SELECT ***
   ```sql
   -- Slow
   SELECT * FROM large_table WHERE id = 1;

   -- Fast
   SELECT id, name, email FROM large_table WHERE id = 1;
   ```

3. **Use appropriate JOINs**
   ```sql
   -- Slow: Multiple subqueries
   SELECT * FROM users
   WHERE id IN (SELECT user_id FROM orders)
     AND id IN (SELECT user_id FROM payments);

   -- Fast: Single JOIN
   SELECT DISTINCT u.*
   FROM users u
   JOIN orders o ON u.id = o.user_id
   JOIN payments p ON u.id = p.user_id;
   ```

4. **Limit result size**
   ```sql
   SELECT * FROM large_table
   WHERE created_at > CURRENT_DATE - INTERVAL '7 days'
   LIMIT 1000;
   ```

### Howlerops running slow

**Solutions**:

1. **Clear cache**
   - Settings > Advanced > Clear Cache
   - Restart Howlerops

2. **Reduce active connections**
   - Close unused database connections
   - Limit concurrent queries

3. **Update to latest version**
   - Help > Check for Updates

4. **Check system resources**
   - Close other applications
   - Free up disk space
   - Increase RAM if needed

---

## Sync Issues

### Cloud sync not working

**Solutions**:

1. **Check internet connection**
   - Verify you're online
   - Test connection to sync.sqlstudio.com

2. **Re-authenticate**
   - Settings > Cloud Sync > Sign Out
   - Sign back in

3. **Force sync**
   - Settings > Cloud Sync > Force Sync

4. **Check sync status**
   - Look for sync icon in status bar
   - Click for detailed sync status

### Sync conflicts

**What are conflicts?**
When the same query is modified on multiple devices, a conflict occurs.

**Resolution**:

1. **Automatic (default)**
   - Latest change wins
   - Previous version saved in history

2. **Manual resolution**
   - Settings > Cloud Sync > Conflict Resolution
   - Review conflicted items
   - Choose which version to keep

### Synced items not appearing

**Solutions**:

1. **Wait for sync to complete**
   - Check sync progress in status bar

2. **Refresh**
   - Click Refresh in sidebar
   - Or restart Howlerops

3. **Check sync settings**
   - Ensure item type is enabled for sync
   - Verify not excluded by filters

---

## UI Issues

### Interface not responding

**Solutions**:

1. **Force refresh**
   - Press `Cmd/Ctrl + R`

2. **Clear cache**
   - Settings > Advanced > Clear Cache

3. **Restart application**
   - Quit and reopen Howlerops

### Missing or broken layout

**Solutions**:

1. **Reset layout**
   - View > Reset Layout
   - Or Settings > Appearance > Reset to Defaults

2. **Reset window size**
   - Close Howlerops
   - Delete window state file:
     - macOS: `~/Library/Application Support/Howlerops/window-state.json`
     - Windows: `%APPDATA%/Howlerops/window-state.json`
     - Linux: `~/.config/sql-studio/window-state.json`

### Dark mode issues

**Solutions**:

1. **Toggle theme**
   - Settings > Appearance > Theme
   - Try "Auto", "Light", or "Dark"

2. **Update application**
   - Theme improvements in newer versions

3. **Check OS theme**
   - Howlerops respects OS theme in "Auto" mode

---

## Export/Import Issues

### Cannot export results

**Solutions**:

1. **Check file permissions**
   - Verify write access to destination folder

2. **Reduce export size**
   - Limit number of rows
   - Select specific columns

3. **Try different format**
   - CSV usually most compatible
   - JSON for nested data
   - Excel for formatted output

### Import failing

**Common issues**:

1. **Encoding problems**
   - Try UTF-8 encoding
   - Check for special characters

2. **Delimiter mismatch**
   - Verify CSV delimiter (comma vs. semicolon)
   - Check quote character

3. **Schema mismatch**
   - Ensure columns match table structure
   - Check data types

---

## Installation Issues

### Cannot install on macOS

**"App is damaged" error**:
```bash
# Remove quarantine attribute
xattr -cr /Applications/SQL\ Studio.app
```

**"Unidentified developer" error**:
1. System Preferences > Security & Privacy
2. Click "Open Anyway"

### Cannot install on Windows

**SmartScreen warning**:
1. Click "More info"
2. Click "Run anyway"

**Permission denied**:
- Run installer as Administrator
- Right-click > Run as Administrator

### Linux AppImage not running

```bash
# Make executable
chmod +x SQL-Studio-*.AppImage

# Run with FUSE
./SQL-Studio-*.AppImage

# Or extract and run
./SQL-Studio-*.AppImage --appimage-extract
./squashfs-root/AppRun
```

---

## Account Issues

### Cannot log in

**Solutions**:

1. **Reset password**
   - Click "Forgot password?" on login screen
   - Check email for reset link

2. **Clear browser cache**
   - For web version only
   - Or try incognito mode

3. **Verify email**
   - Check spam folder for verification email
   - Resend verification from login screen

4. **Check account status**
   - Contact support if account suspended

### Two-factor authentication issues

**Lost device**:
- Use backup codes provided during 2FA setup
- Contact support to disable 2FA

**Code not working**:
- Verify time sync on device
- Check you're using the correct account
- Try backup codes

---

## Keyboard Shortcuts Not Working

**Solutions**:

1. **Check for conflicts**
   - macOS: System Preferences > Keyboard > Shortcuts
   - Windows: Check other applications

2. **Reset shortcuts**
   - Settings > Keyboard > Reset to Defaults

3. **Verify keyboard layout**
   - Some shortcuts depend on keyboard layout

---

## Still Having Issues?

### Collect diagnostic information

1. **Error logs**
   - Help > Show Logs
   - Copy recent errors

2. **System information**
   - Help > About > Copy System Info

3. **Screenshot**
   - Capture the issue if visual

### Contact support

**Email**: support@sqlstudio.com

Include:
- Detailed description of issue
- Steps to reproduce
- Error messages
- System information
- Screenshots

**Expected response time**:
- Free: 2-3 business days
- Pro: 24 hours
- Enterprise: 4 hours

### Community resources

- **Forum**: [community.sqlstudio.com](https://community.sqlstudio.com)
- **Discord**: [discord.gg/sqlstudio](https://discord.gg/sqlstudio)
- **GitHub Issues**: For bug reports
- **Stack Overflow**: Tag with `sql-studio`
