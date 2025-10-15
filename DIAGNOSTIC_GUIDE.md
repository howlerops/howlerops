# Multi-DB Autocomplete Diagnostics Guide

## Overview

A comprehensive diagnostic panel has been added to help identify and fix multi-database autocomplete issues.

## How to Access

### Method 1: Keyboard Shortcut
Press **`Ctrl+Shift+D`** (Windows/Linux) or **`Cmd+Shift+D`** (Mac)

### Method 2: Bug Button
Click the **Bug icon (🐛)** button in the query editor tab bar (next to the "+" button)

## What the Diagnostics Show

### 1. Connection Status
- **Total Connections**: Number of configured database connections
- **Connected Status**: Which connections are currently active
- **Session IDs**: Backend session identifiers for each connection
- **Connection IDs**: Frontend UUID for each connection

**What to Look For:**
- ✅ At least one connection should be **connected** (green checkmark)
- ❌ If all connections show red X, multi-DB autocomplete won't work
- 📝 Note both the `id` and `sessionId` for each connection

### 2. Multi-DB Schemas
- **Schema Map Keys**: What keys are stored in the schema map
- **Schema Count**: Number of schemas loaded per connection
- **Table Count**: Number of tables in each schema
- **Column Count**: Number of columns loaded (should be 0 with lazy loading)

**What to Look For:**
- ✅ Schema map should have keys matching your connection **names** (e.g., "Prod-Leviosa")
- ✅ Each connection should have at least 1 schema (usually "public")
- ❌ If "No schemas loaded" appears, autocomplete won't work
- 🔍 Click on a schema key to expand and see table details

### 3. Column Cache (Lazy Loading)
- **Cached Tables**: Which tables have had their columns loaded
- **Cache Key Format**: `sessionId-schema-table`

**What to Look For:**
- 📝 Initially should be **empty** (0 cached)
- ✅ After typing `@connection.table.`, columns should appear here
- 💨 Second access should use cache (instant)

### 4. Health Check
Automated checks for common issues:

| Check | What It Means | How to Fix |
|-------|---------------|------------|
| ✅ Connections configured | At least one DB connection exists | Add connections in sidebar |
| ✅ At least one connected | A database is actively connected | Click "Connect" on a connection |
| ✅ Multi-DB schemas loaded | Schema map has data | Click "Refresh Schemas" button |
| ✅ Schema keys match connections | Autocomplete lookup will work | Verify connection names match keys |

## Diagnostic Actions

### 1. Refresh Schemas
**Button**: "Refresh Schemas"
- Manually triggers `loadMultiDBSchemas()`
- Useful if schemas didn't load on startup
- Re-fetches table lists from all connected databases

### 2. Test Autocomplete
**Button**: "Test Autocomplete"
- Simulates typing `@Prod-Leviosa.`
- Shows what the autocomplete provider sees
- Logs detailed information to console

**Console Output:**
```
🧪 === AUTOCOMPLETE TEST ===
Multi-DB Schemas Ref: Map(4) { ... }
Schema Keys: ["connection-id-1", "Prod-Leviosa", ...]
Connections: [{ id: "...", name: "Prod-Leviosa", isConnected: true }]
Test Input: @Prod-Leviosa.
Pattern Match: ["@Prod-Leviosa.", "Prod-Leviosa", ""]
  Connection Identifier: Prod-Leviosa
  Partial Table: 
  Found Schemas: YES (1 schemas)
    Schema: public, Tables: 137
========================
```

### 3. Log State
**Button**: "📋 Log State"
- Dumps complete diagnostic state to console
- Includes all connections, schemas, and cache
- Useful for debugging or reporting issues

## Common Issues & Solutions

### Issue 1: "No schemas loaded"
**Symptoms:**
- Schema count shows 0
- Yellow warning box appears
- Autocomplete shows nothing after `@`

**Solutions:**
1. Click "Refresh Schemas" button
2. Ensure at least one connection is **connected** (green checkmark)
3. Check console for errors during schema loading
4. Verify backend is running and accessible

### Issue 2: Autocomplete shows connection names but not tables
**Symptoms:**
- `@Prod` shows suggestions
- `@Prod-Leviosa.` shows nothing

**Solutions:**
1. Click "Test Autocomplete" and check console output
2. Verify "Found Schemas: YES" in test output
3. Expand the schema key in diagnostics to see if tables are present
4. Check that table count > 0 for the schema

### Issue 3: Schema keys don't match connection names
**Symptoms:**
- Health check shows "Schema keys match connections" as ❌
- Autocomplete doesn't work even though schemas are loaded

**Solutions:**
1. Check what keys are in the schema map (expand to see)
2. Verify connection names match exactly (case-sensitive!)
3. Schema map should have both `connection.id` AND `connection.name` as keys
4. If only ID is present, there's a bug in schema loading

### Issue 4: Columns not loading
**Symptoms:**
- Tables autocomplete works
- Typing `alias.` shows nothing

**Solutions:**
1. Check Column Cache section in diagnostics
2. Try typing `@conn.table t WHERE t.` and wait ~300ms
3. Check console for "⏳ Loading columns for: schema.table"
4. If "💨 Columns loaded from cache" appears, it's working!

## Performance Metrics

### Expected Behavior:

**Initial Load:**
```
🔄 Loading multi-DB schemas...
📊 Connected sessions: { connectedWithSessions: 2 }
✅ Multi-DB schemas loaded: { totalTables: 312, totalColumns: 0 }
⚡ Prod-Leviosa schemas now available (1 schemas)
```

**First Column Access:**
```
🔄 Lazy loading columns for: public.accounts
⏳ Loading columns for: public.accounts
✓ Loaded 24 columns for public.accounts
```

**Cached Column Access:**
```
💨 Columns loaded from cache: public.accounts (24 columns)
```

### Performance Targets:
- **Initial schema load**: < 1 second
- **First column load**: < 500ms
- **Cached column load**: < 10ms (instant)
- **No localStorage errors**: QuotaExceededError should NOT appear

## Debugging Workflow

1. **Open Diagnostics**: `Ctrl/Cmd+Shift+D`
2. **Check Health**: All green checkmarks?
3. **Test Autocomplete**: Click "Test Autocomplete" button
4. **Review Console**: Look for errors or "❌ Schema lookup failed!"
5. **Refresh if Needed**: Click "Refresh Schemas"
6. **Log Full State**: Click "📋 Log State" for complete dump
7. **Report Issue**: Share console output with full context

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl/Cmd+Shift+D` | Toggle diagnostics panel |
| `Ctrl/Cmd+S` | Save current query |
| `Ctrl/Cmd+Enter` | Execute query |

## Cleanup

This diagnostic panel is **temporary** for debugging purposes. Once the issue is identified and fixed, it can be:
- Hidden by default (only show on error)
- Removed from production builds
- Kept as a developer tool with a hidden toggle

## Next Steps After Diagnostics

Once you've identified the issue using diagnostics:

1. **Schema loading timing**: If schemas load too late, add auto-connect on startup
2. **Key mismatch**: Fix how schema map keys are stored/looked up
3. **Connection state**: Ensure `isConnected` and `sessionId` are properly set
4. **Pattern matching**: Verify regex correctly extracts connection identifier

The diagnostics will point you to the exact root cause!

