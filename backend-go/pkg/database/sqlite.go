package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// SQLiteDatabase implements the Database interface for SQLite
type SQLiteDatabase struct {
	pool           *ConnectionPool
	config         ConnectionConfig
	logger         *logrus.Logger
	structureCache *tableStructureCache
}

// NewSQLiteDatabase creates a new SQLite database instance
func NewSQLiteDatabase(config ConnectionConfig, logger *logrus.Logger) (*SQLiteDatabase, error) {
	pool, err := NewConnectionPool(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite connection pool: %w", err)
	}

	return &SQLiteDatabase{
		pool:           pool,
		config:         config,
		logger:         logger,
		structureCache: newTableStructureCache(10 * time.Minute),
	}, nil
}

// Connect establishes a connection to SQLite
func (s *SQLiteDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	s.config = config
	pool, err := NewConnectionPool(config, nil, s.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	if s.pool != nil {
		if err := s.pool.Close(); err != nil {
			log.Printf("Failed to close existing SQLite pool: %v", err)
		}
	}
	s.pool = pool
	s.structureCache = newTableStructureCache(10 * time.Minute)

	return nil
}

// Disconnect closes the SQLite connection
func (s *SQLiteDatabase) Disconnect() error {
	if s.pool != nil {
		return s.pool.Close()
	}
	return nil
}

// Ping tests the SQLite connection
func (s *SQLiteDatabase) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// GetConnectionInfo returns SQLite connection information
func (s *SQLiteDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})

	// Get SQLite version
	var version string
	err = db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQLite version: %w", err)
	}
	info["version"] = version

	// Get database file path
	info["database"] = s.config.Database

	// Get database size
	var pageCount, pageSize int64
	err = db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
	if err == nil {
		err = db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
		if err == nil {
			info["size_bytes"] = pageCount * pageSize
		}
	}

	// Get user version
	var userVersion int
	err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&userVersion)
	if err == nil {
		info["user_version"] = userVersion
	}

	// Get encoding
	var encoding string
	err = db.QueryRowContext(ctx, "PRAGMA encoding").Scan(&encoding)
	if err == nil {
		info["encoding"] = encoding
	}

	return info, nil
}

// Execute runs a SQL query and returns the results
func (s *SQLiteDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH") ||
		strings.HasPrefix(strings.ToUpper(query), "PRAGMA")

	if isSelect {
		return s.executeSelect(ctx, db, query, args...)
	} else {
		return s.executeNonSelect(ctx, db, query, args...)
	}
}

// executeSelect handles SELECT queries
func (s *SQLiteDatabase) executeSelect(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
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

		result.Rows = append(result.Rows, values)
	}

	if err := rows.Err(); err != nil {
		result.Error = err
		return result, err
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = time.Since(start)
	if metadata, ready, err := s.computeEditableMetadata(ctx, query, columns, false); err == nil {
		if metadata != nil {
			if !ready {
				metadata.Pending = true
				if metadata.Reason == "" {
					metadata.Reason = "Loading editable metadata"
				}
			}
			result.Editable = metadata
		}
	} else {
		meta := newEditableMetadata(columns)
		meta.Reason = "Failed to prepare editable metadata"
		result.Editable = meta
	}

	return result, nil
}

func (s *SQLiteDatabase) computeEditableMetadata(ctx context.Context, query string, columns []string, allowFetch bool) (*EditableQueryMetadata, bool, error) {
	metadata := newEditableMetadata(columns)

	query = strings.TrimSpace(query)
	if query == "" {
		metadata.Reason = "Empty query"
		return metadata, true, nil
	}

	schema, table, reason, ok := parseSimpleSelect(query)
	if !ok {
		metadata.Reason = reason
		return metadata, true, nil
	}

	if table == "" {
		metadata.Reason = "Unable to identify target table"
		return metadata, true, nil
	}

	if schema == "" {
		schema = "main"
	}

	metadata.Schema = schema
	metadata.Table = table

	structure, ok, err := s.ensureTableStructure(ctx, schema, table, allowFetch)
	if err != nil {
		metadata.Reason = fmt.Sprintf("Failed to get table structure: %v", err)
		return metadata, true, err
	}

	if !ok {
		metadata.Pending = true
		if metadata.Reason == "" {
			metadata.Reason = "Loading table metadata"
		}
		return metadata, false, nil
	}

	columnMap := make(map[string]ColumnInfo, len(structure.Columns))
	primaryKeys := make([]string, 0)
	for _, col := range structure.Columns {
		columnMap[strings.ToLower(col.Name)] = col
		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, col.Name)
		}
	}

	if len(primaryKeys) == 0 {
		metadata.Reason = "Table does not have a primary key"
		return metadata, true, nil
	}

	// Create foreign key lookup map
	fkMap := make(map[string]ForeignKeyRef)
	for _, fk := range structure.ForeignKeys {
		for i, col := range fk.Columns {
			if i < len(fk.ReferencedColumns) {
				fkMap[strings.ToLower(col)] = ForeignKeyRef{
					Table:  fk.ReferencedTable,
					Column: fk.ReferencedColumns[i],
					Schema: fk.ReferencedSchema,
				}
			}
		}
	}

	editableColumns := make([]EditableColumn, 0, len(columns))
	for _, col := range columns {
		columnMeta := EditableColumn{
			Name:       col,
			ResultName: col,
			Editable:   false,
			PrimaryKey: false,
		}

		if colInfo, exists := columnMap[strings.ToLower(col)]; exists {
			columnMeta.Name = colInfo.Name
			columnMeta.DataType = colInfo.DataType
			columnMeta.Editable = true
			columnMeta.PrimaryKey = colInfo.PrimaryKey
			if colInfo.DefaultValue != nil {
				columnMeta.HasDefault = true
				columnMeta.DefaultVal = *colInfo.DefaultValue
				columnMeta.DefaultExp = *colInfo.DefaultValue
			}
			dataTypeLower := strings.ToLower(colInfo.DataType)
			if strings.Contains(dataTypeLower, "timestamp") || strings.Contains(dataTypeLower, "datetime") {
				columnMeta.TimeZone = true
			}
			if colInfo.NumericPrecision != nil {
				precision := *colInfo.NumericPrecision
				columnMeta.Precision = &precision
			}
		}

		// Add foreign key information if available
		if fkRef, hasFK := fkMap[strings.ToLower(col)]; hasFK {
			columnMeta.ForeignKey = &fkRef
		}

		editableColumns = append(editableColumns, columnMeta)
	}

	metadata.Enabled = true
	metadata.PrimaryKeys = primaryKeys
	metadata.Columns = editableColumns
	metadata.Pending = false
	metadata.Reason = ""
	metadata.Capabilities = &MutationCapabilities{
		CanInsert: false,
		CanUpdate: false,
		CanDelete: false,
		Reason:    "Row editing is not yet supported for SQLite connections",
	}

	return metadata, true, nil
}

func (s *SQLiteDatabase) ensureTableStructure(ctx context.Context, schema, table string, allowFetch bool) (*TableStructure, bool, error) {
	if structure, ok := s.structureCache.get(schema, table); ok {
		return structure, true, nil
	}

	if !allowFetch {
		return nil, false, nil
	}

	structure, err := s.loadTableStructure(ctx, schema, table)
	if err != nil {
		return nil, false, err
	}

	s.structureCache.set(schema, table, structure)
	return cloneTableStructure(structure), true, nil
}

func (s *SQLiteDatabase) loadTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	structure := &TableStructure{}

	tableInfo, err := s.getTableInfo(ctx, db, table)
	if err != nil {
		return nil, err
	}
	structure.Table = *tableInfo

	columns, err := s.getTableColumns(ctx, db, table)
	if err != nil {
		return nil, err
	}
	structure.Columns = columns

	indexes, err := s.getTableIndexes(ctx, db, table)
	if err != nil {
		return nil, err
	}
	structure.Indexes = indexes

	foreignKeys, err := s.getTableForeignKeys(ctx, db, table)
	if err != nil {
		return nil, err
	}
	structure.ForeignKeys = foreignKeys

	return structure, nil
}

func (s *SQLiteDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error) {
	metadata, _, err := s.computeEditableMetadata(ctx, query, columns, true)
	if metadata != nil {
		metadata.Pending = false
	}
	return metadata, err
}

// executeNonSelect handles INSERT, UPDATE, DELETE, DDL queries
func (s *SQLiteDatabase) executeNonSelect(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*QueryResult, error) {
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
		affected = 0
	}

	return &QueryResult{
		Affected: affected,
		Duration: time.Since(start),
	}, nil
}

// ExecuteStream executes a query and streams results in batches
func (s *SQLiteDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
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
func (s *SQLiteDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return "", err
	}

	explainQuery := "EXPLAIN QUERY PLAN " + query

	rows, err := db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", fmt.Errorf("failed to explain query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var plan strings.Builder
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			return "", err
		}
		plan.WriteString(fmt.Sprintf("%d|%d|%d|%s\n", id, parent, notused, detail))
	}

	return plan.String(), rows.Err()
}

// GetSchemas returns list of schemas (always 'main' for SQLite)
func (s *SQLiteDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	// SQLite has a simple schema structure
	return []string{"main"}, nil
}

// GetTables returns list of tables in the database
func (s *SQLiteDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			name,
			type,
			sql
		FROM sqlite_master
		WHERE type IN ('table', 'view')
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var sql sql.NullString

		err := rows.Scan(
			&table.Name,
			&table.Type,
			&sql,
		)
		if err != nil {
			return nil, err
		}

		table.Schema = "main"

		// Convert type to uppercase
		table.Type = strings.ToUpper(table.Type)

		// Get row count for tables (not views)
		if table.Type == "TABLE" {
			var count int64
			// #nosec G201 - table name from quoted identifier, safe for SQL formatting
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.QuoteIdentifier(table.Name))
			err = db.QueryRowContext(ctx, countQuery).Scan(&count)
			if err == nil {
				table.RowCount = count
			}
		}

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// GetTableStructure returns detailed structure information for a table
func (s *SQLiteDatabase) GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	if schema == "" {
		schema = "main"
	}

	if structure, ok := s.structureCache.get(schema, table); ok {
		return structure, nil
	}

	structure, err := s.loadTableStructure(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	s.structureCache.set(schema, table, structure)
	return cloneTableStructure(structure), nil
}

// Helper methods for getting table structure details
func (s *SQLiteDatabase) getTableInfo(ctx context.Context, db *sql.DB, table string) (*TableInfo, error) {
	query := `
		SELECT
			name,
			type,
			sql
		FROM sqlite_master
		WHERE name = ? AND type IN ('table', 'view')`

	var tableInfo TableInfo
	var sql sql.NullString

	err := db.QueryRowContext(ctx, query, table).Scan(
		&tableInfo.Name,
		&tableInfo.Type,
		&sql,
	)
	if err != nil {
		return nil, err
	}

	tableInfo.Schema = "main"
	tableInfo.Type = strings.ToUpper(tableInfo.Type)

	// Get row count for tables
	if tableInfo.Type == "TABLE" {
		var count int64
		// #nosec G201 - table name from quoted identifier, safe for SQL formatting
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.QuoteIdentifier(table))
		err = db.QueryRowContext(ctx, countQuery).Scan(&count)
		if err == nil {
			tableInfo.RowCount = count
		}
	}

	return &tableInfo, nil
}

func (s *SQLiteDatabase) getTableColumns(ctx context.Context, db *sql.DB, table string) ([]ColumnInfo, error) {
	query := fmt.Sprintf("PRAGMA table_info('%s')", table)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var cid int
		var notNull int
		var defaultValue sql.NullString

		err := rows.Scan(
			&cid,
			&col.Name,
			&col.DataType,
			&notNull,
			&defaultValue,
			&col.PrimaryKey,
		)
		if err != nil {
			return nil, err
		}

		col.OrdinalPosition = cid + 1 // SQLite cid is 0-based
		col.Nullable = notNull == 0

		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (s *SQLiteDatabase) getTableIndexes(ctx context.Context, db *sql.DB, table string) ([]IndexInfo, error) {
	query := fmt.Sprintf("PRAGMA index_list('%s')", table)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var seq int
		var partial int
		var origin string // PRAGMA index_list origin: "c"=CREATE INDEX, "u"=UNIQUE, "pk"=PRIMARY KEY

		err := rows.Scan(
			&seq,
			&idx.Name,
			&idx.Unique,
			&origin,
			&partial,
		)
		if err != nil {
			return nil, err
		}

		// Set Primary flag based on origin
		idx.Primary = (origin == "pk")

		// Get index columns
		colQuery := fmt.Sprintf("PRAGMA index_info('%s')", idx.Name)
		colRows, err := db.QueryContext(ctx, colQuery)
		if err != nil {
			continue
		}

		var columns []string
		for colRows.Next() {
			var seqno, cid int
			var name string
			if err := colRows.Scan(&seqno, &cid, &name); err != nil {
				continue
			}
			columns = append(columns, name)
		}
		_ = colRows.Close() // Best-effort close

		idx.Columns = columns
		idx.Type = "BTREE" // SQLite default

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

func (s *SQLiteDatabase) getTableForeignKeys(ctx context.Context, db *sql.DB, table string) ([]ForeignKeyInfo, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list('%s')", table)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	fkMap := make(map[int]*ForeignKeyInfo)

	for rows.Next() {
		var id, seq int
		var referencedTable, from, to, onUpdate, onDelete, match string

		err := rows.Scan(
			&id,
			&seq,
			&referencedTable,
			&from,
			&to,
			&onUpdate,
			&onDelete,
			&match,
		)
		if err != nil {
			return nil, err
		}

		if fkMap[id] == nil {
			fkMap[id] = &ForeignKeyInfo{
				Name:              fmt.Sprintf("fk_%s_%d", table, id),
				ReferencedTable:   referencedTable,
				ReferencedSchema:  "main",
				OnUpdate:          onUpdate,
				OnDelete:          onDelete,
				Columns:           make([]string, 0),
				ReferencedColumns: make([]string, 0),
			}
		}

		fk := fkMap[id]
		fk.Columns = append(fk.Columns, from)
		fk.ReferencedColumns = append(fk.ReferencedColumns, to)
	}

	var foreignKeys []ForeignKeyInfo
	for _, fk := range fkMap {
		foreignKeys = append(foreignKeys, *fk)
	}

	return foreignKeys, rows.Err()
}

// BeginTransaction starts a new transaction
func (s *SQLiteDatabase) BeginTransaction(ctx context.Context) (Transaction, error) {
	db, err := s.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &SQLiteTransaction{tx: tx}, nil
}

// ListDatabases returns the current SQLite database (file path)
func (s *SQLiteDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	if strings.TrimSpace(s.config.Database) == "" {
		return []string{}, nil
	}
	return []string{s.config.Database}, nil
}

// SwitchDatabase requires reconnecting with a new SQLite file
func (s *SQLiteDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	if strings.TrimSpace(databaseName) == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	return ErrDatabaseSwitchRequiresReconnect
}

// UpdateRow is currently not supported for SQLite
func (s *SQLiteDatabase) UpdateRow(ctx context.Context, params UpdateRowParams) error {
	return errors.New("row editing is not yet supported for SQLite connections")
}

// InsertRow is currently not supported for SQLite
func (s *SQLiteDatabase) InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error) {
	return nil, errors.New("row insertion is not yet supported for SQLite connections")
}

// DeleteRow is currently not supported for SQLite
func (s *SQLiteDatabase) DeleteRow(ctx context.Context, params DeleteRowParams) error {
	return errors.New("row deletion is not yet supported for SQLite connections")
}

// GetDatabaseType returns the database type
func (s *SQLiteDatabase) GetDatabaseType() DatabaseType {
	return SQLite
}

// GetConnectionStats returns connection pool statistics
func (s *SQLiteDatabase) GetConnectionStats() PoolStats {
	return s.pool.Stats()
}

// QuoteIdentifier quotes an identifier for SQLite
func (s *SQLiteDatabase) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
}

// GetDataTypeMappings returns SQLite-specific data type mappings
func (s *SQLiteDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{
		"string":  "TEXT",
		"int":     "INTEGER",
		"int64":   "INTEGER",
		"float":   "REAL",
		"float64": "REAL",
		"bool":    "INTEGER", // SQLite doesn't have native boolean
		"time":    "DATETIME",
		"date":    "DATE",
		"json":    "TEXT",
		"uuid":    "TEXT",
		"text":    "TEXT",
		"varchar": "TEXT",
		"decimal": "NUMERIC",
	}
}

// SQLiteTransaction implements the Transaction interface for SQLite
type SQLiteTransaction struct {
	tx *sql.Tx
}

// Execute runs a query within the transaction
func (t *SQLiteTransaction) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH") ||
		strings.HasPrefix(strings.ToUpper(query), "PRAGMA")

	if isSelect {
		return t.executeSelect(ctx, query, args...)
	} else {
		return t.executeNonSelect(ctx, query, args...)
	}
}

func (t *SQLiteTransaction) executeSelect(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}
	defer func() { _ = rows.Close() }() // Best-effort close in transaction

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

		result.Rows = append(result.Rows, values)
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = time.Since(start)

	return result, rows.Err()
}

func (t *SQLiteTransaction) executeNonSelect(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
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
func (t *SQLiteTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *SQLiteTransaction) Rollback() error {
	return t.tx.Rollback()
}
