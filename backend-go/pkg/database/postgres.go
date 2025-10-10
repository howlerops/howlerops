package database

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "sort"
    "strings"
    "time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// PostgresDatabase implements the Database interface for PostgreSQL
type PostgresDatabase struct {
	pool   *ConnectionPool
	config ConnectionConfig
	logger *logrus.Logger
}

// NewPostgresDatabase creates a new PostgreSQL database instance
func NewPostgresDatabase(config ConnectionConfig, logger *logrus.Logger) (*PostgresDatabase, error) {
	pool, err := NewConnectionPool(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	return &PostgresDatabase{
		pool:   pool,
		config: config,
		logger: logger,
	}, nil
}

// Connect establishes a connection to PostgreSQL
func (p *PostgresDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	p.config = config
	pool, err := NewConnectionPool(config, p.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if p.pool != nil {
		p.pool.Close()
	}
	p.pool = pool

	return nil
}

// Disconnect closes the PostgreSQL connection
func (p *PostgresDatabase) Disconnect() error {
	if p.pool != nil {
		return p.pool.Close()
	}
	return nil
}

// Ping tests the PostgreSQL connection
func (p *PostgresDatabase) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// GetConnectionInfo returns PostgreSQL connection information
func (p *PostgresDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})

	// Get version
	var version string
	err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to get PostgreSQL version: %w", err)
	}
	info["version"] = version

	// Get current database
	var database string
	err = db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database)
	if err == nil {
		info["database"] = database
	}

	// Get current user
	var user string
	err = db.QueryRowContext(ctx, "SELECT current_user").Scan(&user)
	if err == nil {
		info["user"] = user
	}

	// Get server stats
	var connectionsQuery = `
		SELECT count(*) as total_connections,
			   count(*) FILTER (WHERE state = 'active') as active_connections,
			   count(*) FILTER (WHERE state = 'idle') as idle_connections
		FROM pg_stat_activity`

	var totalConns, activeConns, idleConns int
	err = db.QueryRowContext(ctx, connectionsQuery).Scan(&totalConns, &activeConns, &idleConns)
	if err == nil {
		info["total_connections"] = totalConns
		info["active_connections"] = activeConns
		info["idle_connections"] = idleConns
	}

	return info, nil
}

// Execute runs a SQL query and returns the results
func (p *PostgresDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH")

	if isSelect {
		return p.executeSelect(ctx, db, query, args...)
	} else {
		return p.executeNonSelect(ctx, db, query, args...)
	}
}

// executeSelect handles SELECT queries
func (p *PostgresDatabase) executeSelect(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}
	defer rows.Close()

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

	// Read all rows
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

		// Convert byte arrays to strings for PostgreSQL
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		result.Rows = append(result.Rows, values)
	}

	if err := rows.Err(); err != nil {
		result.Error = err
		return result, err
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = time.Since(start)
	result.Editable = p.buildEditableMetadata(ctx, query, columns)

	return result, nil
}

func (p *PostgresDatabase) buildEditableMetadata(ctx context.Context, query string, columns []string) *EditableQueryMetadata {
	metadata := &EditableQueryMetadata{
		Enabled: false,
		Columns: make([]EditableColumn, 0, len(columns)),
	}

	// Default columns (read-only) in case we early-return
	for _, col := range columns {
		metadata.Columns = append(metadata.Columns, EditableColumn{
			Name:       col,
			ResultName: col,
			Editable:   false,
			PrimaryKey: false,
		})
	}

	query = strings.TrimSpace(query)
	if query == "" {
		metadata.Reason = "Empty query"
		return metadata
	}

	schema, table, reason, ok := parseSimpleSelect(query)
	if !ok {
		metadata.Reason = reason
		return metadata
	}

	if schema == "" {
		currentSchema, err := p.getCurrentSchema(ctx)
		if err != nil {
			metadata.Reason = "Unable to determine current schema"
			return metadata
		}
		schema = currentSchema
	}

	if table == "" {
		metadata.Reason = "Unable to identify target table"
		return metadata
	}

	structure, err := p.GetTableStructure(ctx, schema, table)
	if err != nil {
		metadata.Reason = "Unable to load table metadata"
		return metadata
	}

	columnMap := make(map[string]ColumnInfo)
	primaryKeys := make([]string, 0)
	for _, col := range structure.Columns {
		key := strings.ToLower(col.Name)
		columnMap[key] = col
		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, col.Name)
		}
	}

	if len(primaryKeys) == 0 {
		metadata.Reason = "Table does not have a primary key"
		return metadata
	}

	resultColumnSet := make(map[string]struct{})
	for _, col := range columns {
		resultColumnSet[strings.ToLower(col)] = struct{}{}
	}

	missingPK := make([]string, 0)
	for _, pk := range primaryKeys {
		if _, ok := resultColumnSet[strings.ToLower(pk)]; !ok {
			missingPK = append(missingPK, pk)
		}
	}
	if len(missingPK) > 0 {
		metadata.Reason = fmt.Sprintf("Result set is missing primary key columns: %s", strings.Join(missingPK, ", "))
		return metadata
	}

	editableColumns := make([]EditableColumn, 0, len(columns))
	editableCount := 0
	for _, resultCol := range columns {
		colInfo, exists := columnMap[strings.ToLower(resultCol)]
		columnMeta := EditableColumn{
			Name:       resultCol,
			ResultName: resultCol,
			Editable:   false,
			PrimaryKey: false,
		}

		if exists {
			columnMeta.Name = colInfo.Name
			columnMeta.DataType = colInfo.DataType
			columnMeta.PrimaryKey = colInfo.PrimaryKey
			if !colInfo.PrimaryKey {
				columnMeta.Editable = true
				editableCount++
			}
		} else {
			// Column not part of the base table (computed/alias)
			columnMeta.Editable = false
		}

		editableColumns = append(editableColumns, columnMeta)
	}

	if editableCount == 0 {
		metadata.Reason = "No editable columns found in result set"
		return metadata
	}

	metadata.Enabled = true
	metadata.Reason = ""
	metadata.Schema = schema
	metadata.Table = table
	metadata.PrimaryKeys = primaryKeys
	metadata.Columns = editableColumns

	return metadata
}


func (p *PostgresDatabase) getCurrentSchema(ctx context.Context) (string, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return "", err
	}

	var schema string
	if err := db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
		return "", err
	}

	return schema, nil
}

// executeNonSelect handles INSERT, UPDATE, DELETE, DDL queries
func (p *PostgresDatabase) executeNonSelect(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*QueryResult, error) {
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
		// Some operations don't return affected rows
		affected = 0
	}

	return &QueryResult{
		Affected: affected,
		Duration: time.Since(start),
	}, nil
}

// UpdateRow persists changes to a single row identified by its primary key
func (p *PostgresDatabase) UpdateRow(ctx context.Context, params UpdateRowParams) error {
	if len(params.Columns) == 0 {
		return errors.New("result column metadata is required for updates")
	}

	metadata := p.buildEditableMetadata(ctx, params.OriginalQuery, params.Columns)
	if metadata == nil || !metadata.Enabled {
		reason := "query is not editable"
		if metadata != nil && metadata.Reason != "" {
			reason = metadata.Reason
		}
		return errors.New(reason)
	}

	schema := metadata.Schema
	if schema == "" {
		schema = params.Schema
	}
	table := metadata.Table
	if table == "" {
		table = params.Table
	}
	if table == "" {
		return errors.New("target table not specified")
	}

	if len(metadata.PrimaryKeys) == 0 {
		return errors.New("table does not have a primary key")
	}

	// Ensure all primary key values are present
	pkValues := make(map[string]interface{})
	for _, pk := range metadata.PrimaryKeys {
		found := false
		for key, value := range params.PrimaryKey {
			if strings.EqualFold(key, pk) {
				pkValues[pk] = value
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing primary key value for column %s", pk)
		}
	}

	if len(params.Values) == 0 {
		return errors.New("no values provided for update")
	}

	editableColumns := make(map[string]EditableColumn)
	for _, col := range metadata.Columns {
		editableColumns[strings.ToLower(col.ResultName)] = col
	}

	valueKeys := make([]string, 0, len(params.Values))
	for key := range params.Values {
		valueKeys = append(valueKeys, key)
	}
	sort.Slice(valueKeys, func(i, j int) bool {
		return strings.ToLower(valueKeys[i]) < strings.ToLower(valueKeys[j])
	})

	setClauses := make([]string, 0, len(valueKeys))
	args := make([]interface{}, 0, len(valueKeys)+len(pkValues))
	argIndex := 1
	for _, key := range valueKeys {
		colMeta, ok := editableColumns[strings.ToLower(key)]
		if !ok || !colMeta.Editable || colMeta.PrimaryKey {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", p.QuoteIdentifier(colMeta.Name), argIndex))
		args = append(args, params.Values[key])
		argIndex++
	}

	if len(setClauses) == 0 {
		return errors.New("no editable columns provided for update")
	}

	pkNames := make([]string, len(metadata.PrimaryKeys))
	copy(pkNames, metadata.PrimaryKeys)
	sort.Slice(pkNames, func(i, j int) bool {
		return strings.ToLower(pkNames[i]) < strings.ToLower(pkNames[j])
	})

	whereClauses := make([]string, 0, len(pkNames))
	for _, pk := range pkNames {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", p.QuoteIdentifier(pk), argIndex))
		args = append(args, pkValues[pk])
		argIndex++
	}

	tableIdentifier := p.QuoteIdentifier(table)
	if schema != "" {
		tableIdentifier = fmt.Sprintf("%s.%s", p.QuoteIdentifier(schema), tableIdentifier)
	}

	updateSQL := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableIdentifier,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	db, err := p.pool.Get(ctx)
	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, updateSQL, args...)
	if err != nil {
		return err
	}

	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return errors.New("no rows were updated; data may have changed or no modifications detected")
	}

	return nil
}

// ExecuteStream executes a query and streams results in batches
func (p *PostgresDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

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

		// Convert byte arrays to strings for PostgreSQL
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
			batch = batch[:0] // Reset slice
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
func (p *PostgresDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return "", err
	}

	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + query

	var plan string
	err = db.QueryRowContext(ctx, explainQuery, args...).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("failed to explain query: %w", err)
	}

	return plan, nil
}

// GetSchemas returns list of schemas in the database
func (p *PostgresDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// GetTables returns list of tables in a schema
func (p *PostgresDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			t.table_schema,
			t.table_name,
			t.table_type,
			COALESCE(obj_description(c.oid), '') as comment,
			COALESCE(s.n_tup_ins + s.n_tup_upd + s.n_tup_del, 0) as row_count,
			COALESCE(pg_total_relation_size(c.oid), 0) as size_bytes
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name AND s.schemaname = t.table_schema
		WHERE t.table_schema = $1
		ORDER BY t.table_name`

	rows, err := db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var tableType string

		err := rows.Scan(
			&table.Schema,
			&table.Name,
			&tableType,
			&table.Comment,
			&table.RowCount,
			&table.SizeBytes,
		)
		if err != nil {
			return nil, err
		}

		// Convert table type
		switch tableType {
		case "BASE TABLE":
			table.Type = "TABLE"
		case "VIEW":
			table.Type = "VIEW"
		default:
			table.Type = tableType
		}

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// GetTableStructure returns detailed structure information for a table
func (p *PostgresDatabase) GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	structure := &TableStructure{}

	// Get table info
	tableInfo, err := p.getTableInfo(ctx, db, schema, table)
	if err != nil {
		return nil, err
	}
	structure.Table = *tableInfo

	// Get columns
	columns, err := p.getTableColumns(ctx, db, schema, table)
	if err != nil {
		return nil, err
	}
	structure.Columns = columns

	// Get indexes
	indexes, err := p.getTableIndexes(ctx, db, schema, table)
	if err != nil {
		return nil, err
	}
	structure.Indexes = indexes

	// Get foreign keys
	foreignKeys, err := p.getTableForeignKeys(ctx, db, schema, table)
	if err != nil {
		return nil, err
	}
	structure.ForeignKeys = foreignKeys

	return structure, nil
}

// Helper methods for getting table structure details
func (p *PostgresDatabase) getTableInfo(ctx context.Context, db *sql.DB, schema, table string) (*TableInfo, error) {
	query := `
		SELECT
			t.table_schema,
			t.table_name,
			t.table_type,
			COALESCE(obj_description(c.oid), '') as comment,
			COALESCE(s.n_tup_ins + s.n_tup_upd + s.n_tup_del, 0) as row_count,
			COALESCE(pg_total_relation_size(c.oid), 0) as size_bytes
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name AND s.schemaname = t.table_schema
		WHERE t.table_schema = $1 AND t.table_name = $2`

	var tableInfo TableInfo
	var tableType string

	err := db.QueryRowContext(ctx, query, schema, table).Scan(
		&tableInfo.Schema,
		&tableInfo.Name,
		&tableType,
		&tableInfo.Comment,
		&tableInfo.RowCount,
		&tableInfo.SizeBytes,
	)
	if err != nil {
		return nil, err
	}

	// Convert table type
	switch tableType {
	case "BASE TABLE":
		tableInfo.Type = "TABLE"
	case "VIEW":
		tableInfo.Type = "VIEW"
	default:
		tableInfo.Type = tableType
	}

	return &tableInfo, nil
}

func (p *PostgresDatabase) getTableColumns(ctx context.Context, db *sql.DB, schema, table string) ([]ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as nullable,
			c.column_default,
			c.ordinal_position,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			COALESCE(col_description(pgc.oid, c.ordinal_position), '') as comment,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as primary_key
		FROM information_schema.columns c
		LEFT JOIN pg_class pgc ON pgc.relname = c.table_name
		LEFT JOIN pg_namespace pgn ON pgn.oid = pgc.relnamespace AND pgn.nspname = c.table_schema
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.key_column_usage ku
			JOIN information_schema.table_constraints tc ON ku.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = $1 AND tc.table_name = $2
		) pk ON pk.column_name = c.column_name
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`

	rows, err := db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultValue sql.NullString
		var charMaxLen sql.NullInt64
		var numPrecision, numScale sql.NullInt32

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.Nullable,
			&defaultValue,
			&col.OrdinalPosition,
			&charMaxLen,
			&numPrecision,
			&numScale,
			&col.Comment,
			&col.PrimaryKey,
		)
		if err != nil {
			return nil, err
		}

		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}
		if charMaxLen.Valid {
			col.CharacterMaxLength = &charMaxLen.Int64
		}
		if numPrecision.Valid {
			precision := int(numPrecision.Int32)
			col.NumericPrecision = &precision
		}
		if numScale.Valid {
			scale := int(numScale.Int32)
			col.NumericScale = &scale
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (p *PostgresDatabase) getTableIndexes(ctx context.Context, db *sql.DB, schema, table string) ([]IndexInfo, error) {
	query := `
		SELECT
			i.relname as index_name,
			array_agg(a.attname ORDER BY c.ordinality) as columns,
			ix.indisunique as unique,
			ix.indisprimary as primary,
			am.amname as method
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		JOIN unnest(ix.indkey) WITH ORDINALITY AS c(attnum, ordinality) ON true
		JOIN pg_attribute a ON a.attrelid = ix.indrelid AND a.attnum = c.attnum
		WHERE n.nspname = $1 AND t.relname = $2
		GROUP BY i.relname, ix.indisunique, ix.indisprimary, am.amname
		ORDER BY i.relname`

	rows, err := db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var columns string

		err := rows.Scan(
			&idx.Name,
			&columns,
			&idx.Unique,
			&idx.Primary,
			&idx.Method,
		)
		if err != nil {
			return nil, err
		}

		// Parse column array from PostgreSQL
		columns = strings.Trim(columns, "{}")
		if columns != "" {
			idx.Columns = strings.Split(columns, ",")
		}

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

func (p *PostgresDatabase) getTableForeignKeys(ctx context.Context, db *sql.DB, schema, table string) ([]ForeignKeyInfo, error) {
	query := `
		SELECT
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns,
			ccu.table_name as referenced_table,
			ccu.table_schema as referenced_schema,
			array_agg(ccu.column_name ORDER BY kcu.ordinal_position) as referenced_columns,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
		JOIN information_schema.referential_constraints rc ON rc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = $1 AND tc.table_name = $2
		GROUP BY tc.constraint_name, ccu.table_name, ccu.table_schema, rc.delete_rule, rc.update_rule
		ORDER BY tc.constraint_name`

	rows, err := db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []ForeignKeyInfo
	for rows.Next() {
		var fk ForeignKeyInfo
		var columns, refColumns string

		err := rows.Scan(
			&fk.Name,
			&columns,
			&fk.ReferencedTable,
			&fk.ReferencedSchema,
			&refColumns,
			&fk.OnDelete,
			&fk.OnUpdate,
		)
		if err != nil {
			return nil, err
		}

		// Parse column arrays from PostgreSQL
		columns = strings.Trim(columns, "{}")
		if columns != "" {
			fk.Columns = strings.Split(columns, ",")
		}

		refColumns = strings.Trim(refColumns, "{}")
		if refColumns != "" {
			fk.ReferencedColumns = strings.Split(refColumns, ",")
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, rows.Err()
}

// BeginTransaction starts a new transaction
func (p *PostgresDatabase) BeginTransaction(ctx context.Context) (Transaction, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &PostgresTransaction{tx: tx}, nil
}

// GetDatabaseType returns the database type
func (p *PostgresDatabase) GetDatabaseType() DatabaseType {
	return PostgreSQL
}

// GetConnectionStats returns connection pool statistics
func (p *PostgresDatabase) GetConnectionStats() PoolStats {
	return p.pool.Stats()
}

// QuoteIdentifier quotes an identifier for PostgreSQL
func (p *PostgresDatabase) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
}

// GetDataTypeMappings returns PostgreSQL-specific data type mappings
func (p *PostgresDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{
		"string":  "TEXT",
		"int":     "INTEGER",
		"int64":   "BIGINT",
		"float":   "REAL",
		"float64": "DOUBLE PRECISION",
		"bool":    "BOOLEAN",
		"time":    "TIMESTAMP",
		"date":    "DATE",
		"json":    "JSONB",
		"uuid":    "UUID",
		"text":    "TEXT",
		"varchar": "VARCHAR",
		"decimal": "DECIMAL",
	}
}

// PostgresTransaction implements the Transaction interface for PostgreSQL
type PostgresTransaction struct {
	tx *sql.Tx
}

// Execute runs a query within the transaction
func (t *PostgresTransaction) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH")

	if isSelect {
		return t.executeSelect(ctx, query, args...)
	} else {
		return t.executeNonSelect(ctx, query, args...)
	}
}

func (t *PostgresTransaction) executeSelect(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}
	defer rows.Close()

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

		// Convert byte arrays to strings for PostgreSQL
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		result.Rows = append(result.Rows, values)
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = time.Since(start)

	return result, rows.Err()
}

func (t *PostgresTransaction) executeNonSelect(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	sqlResult, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	affected, err := sqlResult.RowsAffected()
	if err != nil {
		affected = 0
	}

	return &QueryResult{
		Affected: affected,
		Duration: time.Since(start),
	}, nil
}

// Commit commits the transaction
func (t *PostgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *PostgresTransaction) Rollback() error {
	return t.tx.Rollback()
}
