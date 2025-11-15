package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/sirupsen/logrus"
)

// ClickHouseDatabase implements the Database interface for ClickHouse
type ClickHouseDatabase struct {
	pool   *ConnectionPool
	config ConnectionConfig
	logger *logrus.Logger
}

// NewClickHouseDatabase creates a new ClickHouse database instance
func NewClickHouseDatabase(config ConnectionConfig, logger *logrus.Logger) (*ClickHouseDatabase, error) {
	pool, err := NewConnectionPool(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClickHouse connection pool: %w", err)
	}

	return &ClickHouseDatabase{
		pool:   pool,
		config: config,
		logger: logger,
	}, nil
}

// Connect establishes a connection to ClickHouse
func (c *ClickHouseDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	c.config = config
	pool, err := NewConnectionPool(config, nil, c.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	if c.pool != nil {
		if err := c.pool.Close(); err != nil {
			log.Printf("Failed to close existing ClickHouse pool: %v", err)
		}
	}
	c.pool = pool

	return nil
}

// Disconnect closes the ClickHouse connection
func (c *ClickHouseDatabase) Disconnect() error {
	if c.pool != nil {
		return c.pool.Close()
	}
	return nil
}

// Ping tests the ClickHouse connection
func (c *ClickHouseDatabase) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// GetConnectionInfo returns ClickHouse connection information
func (c *ClickHouseDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})

	// Get version
	var version string
	err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to get ClickHouse version: %w", err)
	}
	info["version"] = version

	// Get current database
	var database string
	err = db.QueryRowContext(ctx, "SELECT currentDatabase()").Scan(&database)
	if err == nil {
		info["database"] = database
	}

	// Get uptime
	var uptime uint64
	err = db.QueryRowContext(ctx, "SELECT uptime()").Scan(&uptime)
	if err == nil {
		info["uptime_seconds"] = uptime
	}

	return info, nil
}

// Execute runs a SQL query and returns the results
func (c *ClickHouseDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH") ||
		strings.HasPrefix(strings.ToUpper(query), "SHOW") ||
		strings.HasPrefix(strings.ToUpper(query), "DESCRIBE")

	if isSelect {
		return c.executeSelect(ctx, db, query, nil, args...)
	} else {
		return c.executeNonSelect(ctx, db, query, args...)
	}
}

// ExecuteWithOptions runs a SQL query with options and returns the results
func (c *ClickHouseDatabase) ExecuteWithOptions(ctx context.Context, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH") ||
		strings.HasPrefix(strings.ToUpper(query), "SHOW") ||
		strings.HasPrefix(strings.ToUpper(query), "DESCRIBE")

	if isSelect {
		return c.executeSelect(ctx, db, query, opts, args...)
	} else {
		return c.executeNonSelect(ctx, db, query, args...)
	}
}

// executeSelect handles SELECT queries
func (c *ClickHouseDatabase) executeSelect(ctx context.Context, db *sql.DB, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	// Check if query already has LIMIT clause
	trimmedQuery := strings.TrimSpace(query)
	trimmedQuery = strings.TrimSuffix(trimmedQuery, ";")

	// Parse existing LIMIT clause (handles "LIMIT 1000" and "LIMIT 1000 OFFSET 500")
	limitRegex := regexp.MustCompile(`(?i)\s+LIMIT\s+(\d+)(?:\s+OFFSET\s+(\d+))?`)
	matches := limitRegex.FindStringSubmatch(trimmedQuery)

	var userLimit int64
	var userOffset int64
	var hasLimit bool
	var queryWithoutLimit string

	if len(matches) > 0 {
		hasLimit = true
		userLimit, _ = strconv.ParseInt(matches[1], 10, 64)
		if len(matches) > 2 && matches[2] != "" {
			userOffset, _ = strconv.ParseInt(matches[2], 10, 64)
		}
		// Remove LIMIT/OFFSET from query
		queryWithoutLimit = limitRegex.ReplaceAllString(trimmedQuery, "")
	} else {
		queryWithoutLimit = trimmedQuery
	}

	// Step 1: Determine total rows and pagination strategy
	var totalRows int64
	modifiedQuery := query

	if opts != nil && opts.Limit > 0 {
		if hasLimit {
			// User specified LIMIT - use that as total, but paginate through it
			totalRows = userLimit

			// Apply our pagination on top of user's limit
			effectiveLimit := opts.Limit
			effectiveOffset := opts.Offset + int(userOffset)

			// Don't exceed user's limit
			if int64(effectiveOffset) >= userLimit {
				effectiveLimit = 0 // No more rows to fetch
			} else if int64(effectiveOffset)+int64(effectiveLimit) > userLimit {
				effectiveLimit = int(userLimit - int64(effectiveOffset))
			}

			if effectiveLimit > 0 {
				modifiedQuery = fmt.Sprintf("%s LIMIT %d OFFSET %d", queryWithoutLimit, effectiveLimit, effectiveOffset)
			} else {
				modifiedQuery = fmt.Sprintf("%s LIMIT 0", queryWithoutLimit)
			}
		} else {
			// No user LIMIT - get total count and apply pagination
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_subquery", queryWithoutLimit)
			err := db.QueryRowContext(ctx, countQuery, args...).Scan(&totalRows)
			if err != nil {
				c.logger.WithError(err).Warn("Failed to get total count for pagination")
				totalRows = 0
			}

			modifiedQuery = fmt.Sprintf("%s LIMIT %d", queryWithoutLimit, opts.Limit)
			if opts.Offset > 0 {
				modifiedQuery = fmt.Sprintf("%s OFFSET %d", modifiedQuery, opts.Offset)
			}
		}
	} else {
		// No pagination requested - use original query
		modifiedQuery = trimmedQuery
	}

	// Step 3: Execute modified query
	rows, err := db.QueryContext(ctx, modifiedQuery, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the query - data already retrieved
			_ = err
		}
	}()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	result := &QueryResult{
		Columns:  columns,
		Rows:     make([][]interface{}, 0),
		Duration: time.Since(start),
	}

	// Step 4: Read and normalize all rows
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return &QueryResult{
				Error:    err,
				Duration: time.Since(start),
			}, err
		}

		// Convert byte arrays to strings for ClickHouse (existing functionality)
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		// NEW: Normalize each value
		normalizedRow := make([]interface{}, len(values))
		for i, val := range values {
			normalizedRow[i] = NormalizeValue(val)
		}

		result.Rows = append(result.Rows, normalizedRow)
	}

	if err := rows.Err(); err != nil {
		result.Error = err
		return result, err
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = time.Since(start)

	// Step 5: Set pagination metadata
	if opts != nil && opts.Limit > 0 {
		result.TotalRows = totalRows
		result.PagedRows = int64(len(result.Rows))
		result.Offset = opts.Offset
		result.HasMore = (int64(opts.Offset) + result.PagedRows) < totalRows
	}

	// PRESERVED: Editable metadata logic (ClickHouse tables are not directly editable)
	metadata, err := c.ComputeEditableMetadata(ctx, query, columns)
	if err == nil && metadata != nil {
		result.Editable = metadata
	}

	return result, nil
}

// executeNonSelect handles INSERT, UPDATE, DELETE, DDL queries
func (c *ClickHouseDatabase) executeNonSelect(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	sqlResult, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	affected, err := sqlResult.RowsAffected()
	if err != nil {
		// ClickHouse doesn't always return affected rows for all operations
		affected = 0
	}

	return &QueryResult{
		Affected: affected,
		Duration: time.Since(start),
	}, nil
}

// ExecuteStream executes a query and streams results in batches
func (c *ClickHouseDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the stream - data already retrieved
			_ = err
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	batch := make([][]interface{}, 0, batchSize)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		// Convert byte arrays to strings
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		batch = append(batch, values)

		if len(batch) >= batchSize {
			if err := callback(batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}

	// Send remaining rows
	if len(batch) > 0 {
		if err := callback(batch); err != nil {
			return err
		}
	}

	return rows.Err()
}

// ExplainQuery returns the execution plan for a query
func (c *ClickHouseDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return "", err
	}

	explainQuery := "EXPLAIN " + query

	rows, err := db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", fmt.Errorf("failed to explain query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail - data already retrieved
			_ = err
		}
	}()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		plan.WriteString(line)
		plan.WriteString("\n")
	}

	return plan.String(), nil
}

// GetSchemas returns list of databases in ClickHouse
func (c *ClickHouseDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT name
		FROM system.databases
		WHERE name NOT IN ('system', 'INFORMATION_SCHEMA', 'information_schema')
		ORDER BY name`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail - data already retrieved
			_ = err
		}
	}()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}

	return schemas, rows.Err()
}

// GetTables returns list of tables in a database
func (c *ClickHouseDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			database,
			name,
			engine,
			'',
			total_rows,
			total_bytes
		FROM system.tables
		WHERE database = ?
		ORDER BY name`

	rows, err := db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail - data already retrieved
			_ = err
		}
	}()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var engine string

		err := rows.Scan(
			&table.Schema,
			&table.Name,
			&engine,
			&table.Comment,
			&table.RowCount,
			&table.SizeBytes,
		)
		if err != nil {
			return nil, err
		}

		table.Type = "TABLE"
		if table.Metadata == nil {
			table.Metadata = make(map[string]string)
		}
		table.Metadata["engine"] = engine

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// GetTableStructure returns detailed structure information for a table
func (c *ClickHouseDatabase) GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	db, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	structure := &TableStructure{}

	// Get table info
	tableInfo := TableInfo{
		Schema: schema,
		Name:   table,
		Type:   "TABLE",
	}

	var engine string
	var totalRows, totalBytes int64
	err = db.QueryRowContext(ctx,
		"SELECT engine, total_rows, total_bytes FROM system.tables WHERE database = ? AND name = ?",
		schema, table).Scan(&engine, &totalRows, &totalBytes)
	if err != nil {
		return nil, err
	}

	tableInfo.RowCount = totalRows
	tableInfo.SizeBytes = totalBytes
	tableInfo.Metadata = map[string]string{"engine": engine}
	structure.Table = tableInfo

	// Get columns
	columnsQuery := `
		SELECT
			name,
			type,
			default_kind,
			default_expression,
			position
		FROM system.columns
		WHERE database = ? AND table = ?
		ORDER BY position`

	rows, err := db.QueryContext(ctx, columnsQuery, schema, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail - data already retrieved
			_ = err
		}
	}()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultKind sql.NullString
		var defaultExpr sql.NullString
		var position int

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&defaultKind,
			&defaultExpr,
			&position,
		)
		if err != nil {
			return nil, err
		}

		col.OrdinalPosition = position
		col.Nullable = strings.Contains(strings.ToLower(col.DataType), "nullable")

		if defaultExpr.Valid && defaultExpr.String != "" {
			col.DefaultValue = &defaultExpr.String
		}

		columns = append(columns, col)
	}
	structure.Columns = columns

	return structure, nil
}

// BeginTransaction starts a new transaction (ClickHouse has limited transaction support)
func (c *ClickHouseDatabase) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("ClickHouse does not support traditional transactions")
}

// UpdateRow is not supported for ClickHouse (immutable by design)
func (c *ClickHouseDatabase) UpdateRow(ctx context.Context, params UpdateRowParams) error {
	return fmt.Errorf("direct row updates are not supported in ClickHouse (use ALTER TABLE UPDATE)")
}

// InsertRow is not supported for ClickHouse
func (c *ClickHouseDatabase) InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error) {
	return nil, fmt.Errorf("direct row inserts via editor are not supported in ClickHouse (use INSERT statements)")
}

// DeleteRow is not supported for ClickHouse
func (c *ClickHouseDatabase) DeleteRow(ctx context.Context, params DeleteRowParams) error {
	return fmt.Errorf("direct row deletes are not supported in ClickHouse (use ALTER TABLE DELETE)")
}

// ComputeEditableMetadata returns metadata indicating ClickHouse tables are not directly editable
func (c *ClickHouseDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error) {
	metadata := &EditableQueryMetadata{
		Enabled: false,
		Reason:  "ClickHouse tables are immutable and not directly editable",
	}
	return metadata, nil
}

// ListDatabases returns an error as quick switching is not supported
func (c *ClickHouseDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	return nil, ErrDatabaseSwitchNotSupported
}

// SwitchDatabase is not supported for ClickHouse via this interface
func (c *ClickHouseDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	return ErrDatabaseSwitchNotSupported
}

// GetDatabaseType returns the database type
func (c *ClickHouseDatabase) GetDatabaseType() DatabaseType {
	return ClickHouse
}

// GetConnectionStats returns connection pool statistics
func (c *ClickHouseDatabase) GetConnectionStats() PoolStats {
	return c.pool.Stats()
}

// QuoteIdentifier quotes an identifier for ClickHouse
func (c *ClickHouseDatabase) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(identifier, "`", "``"))
}

// GetDataTypeMappings returns ClickHouse-specific data type mappings
func (c *ClickHouseDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{
		"string":  "String",
		"int":     "Int32",
		"int64":   "Int64",
		"float":   "Float32",
		"float64": "Float64",
		"bool":    "UInt8",
		"time":    "DateTime",
		"date":    "Date",
		"json":    "String",
		"uuid":    "UUID",
		"array":   "Array",
		"decimal": "Decimal",
	}
}
