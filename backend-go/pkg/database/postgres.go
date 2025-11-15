package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// PostgresDatabase implements the Database interface for PostgreSQL
type PostgresDatabase struct {
	pool           *ConnectionPool
	config         ConnectionConfig
	logger         *logrus.Logger
	structureCache *tableStructureCache
}

// NewPostgresDatabase creates a new PostgreSQL database instance
func NewPostgresDatabase(config ConnectionConfig, logger *logrus.Logger) (*PostgresDatabase, error) {
	pool, err := NewConnectionPool(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	return &PostgresDatabase{
		pool:           pool,
		config:         config,
		logger:         logger,
		structureCache: newTableStructureCache(10 * time.Minute),
	}, nil
}

// Connect establishes a connection to PostgreSQL
func (p *PostgresDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	p.config = config
	pool, err := NewConnectionPool(config, nil, p.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if p.pool != nil {
		if err := p.pool.Close(); err != nil {
			log.Printf("Failed to close existing PostgreSQL pool: %v", err)
		}
	}
	p.pool = pool
	p.structureCache = newTableStructureCache(10 * time.Minute)

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
		return p.executeSelect(ctx, db, query, nil, args...)
	} else {
		return p.executeNonSelect(ctx, db, query, args...)
	}
}

// ExecuteWithOptions runs a SQL query with options and returns the results
func (p *PostgresDatabase) ExecuteWithOptions(ctx context.Context, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "WITH")

	if isSelect {
		return p.executeSelect(ctx, db, query, opts, args...)
	} else {
		return p.executeNonSelect(ctx, db, query, args...)
	}
}

// executeSelect handles SELECT queries
func (p *PostgresDatabase) executeSelect(ctx context.Context, db *sql.DB, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
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
				p.logger.WithError(err).Warn("Failed to get total count for pagination")
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
			p.logger.WithError(err).Error("Failed to close rows")
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

		// Convert byte arrays to strings for PostgreSQL (existing functionality)
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

	// PRESERVED: Editable metadata detection logic
	if metadata, ready, err := p.computeEditableMetadata(ctx, query, columns, false); err == nil {
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
		// Fall back to disabled metadata with error reason
		metadata := newEditableMetadata(columns)
		metadata.Reason = "Failed to prepare editable metadata"
		result.Editable = metadata
	}

	return result, nil
}

func (p *PostgresDatabase) computeEditableMetadata(ctx context.Context, query string, columns []string, allowFetch bool) (*EditableQueryMetadata, bool, error) {
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
		if !allowFetch {
			metadata.Pending = true
			metadata.Reason = "Resolving schema"
			return metadata, false, nil
		}

		currentSchema, err := p.getCurrentSchema(ctx)
		if err != nil {
			metadata.Reason = "Unable to determine current schema"
			return metadata, true, err
		}
		schema = currentSchema
	}

	metadata.Schema = schema
	metadata.Table = table

	structure, ok, err := p.ensureTableStructure(ctx, schema, table, allowFetch)
	if err != nil {
		metadata.Reason = "Unable to load table metadata"
		return metadata, true, err
	}

	if !ok {
		metadata.Pending = true
		if metadata.Reason == "" {
			metadata.Reason = "Loading table metadata"
		}
		return metadata, false, nil
	}

	if err := populateEditableMetadataFromStructure(metadata, columns, structure); err != nil {
		metadata.Reason = err.Error()
		return metadata, true, err
	}

	return metadata, true, nil
}

func (p *PostgresDatabase) ensureTableStructure(ctx context.Context, schema, table string, allowFetch bool) (*TableStructure, bool, error) {
	if structure, ok := p.structureCache.get(schema, table); ok {
		return structure, true, nil
	}

	if !allowFetch {
		return nil, false, nil
	}

	structure, err := p.loadTableStructure(ctx, schema, table)
	if err != nil {
		return nil, false, err
	}

	p.structureCache.set(schema, table, structure)
	return cloneTableStructure(structure), true, nil
}

func populateEditableMetadataFromStructure(metadata *EditableQueryMetadata, columns []string, structure *TableStructure) error {
	if metadata == nil || structure == nil {
		return fmt.Errorf("unable to load table metadata")
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
		return fmt.Errorf("table does not have a primary key")
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
		return fmt.Errorf("result set is missing primary key columns: %s", strings.Join(missingPK, ", "))
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
			if colInfo.DefaultValue != nil {
				columnMeta.HasDefault = true
				columnMeta.DefaultVal = *colInfo.DefaultValue
				columnMeta.DefaultExp = *colInfo.DefaultValue
				lowerDefault := strings.ToLower(*colInfo.DefaultValue)
				if strings.HasPrefix(lowerDefault, "nextval(") || strings.Contains(lowerDefault, "identity") {
					columnMeta.AutoNumber = true
				}
			}
			if strings.Contains(strings.ToLower(colInfo.DataType), "with time zone") {
				columnMeta.TimeZone = true
			}
			if colInfo.NumericPrecision != nil {
				precision := *colInfo.NumericPrecision
				columnMeta.Precision = &precision
			}
			if !colInfo.PrimaryKey {
				columnMeta.Editable = true
				editableCount++
			}
		}

		// Add foreign key information if available
		if fkRef, hasFK := fkMap[strings.ToLower(resultCol)]; hasFK {
			columnMeta.ForeignKey = &fkRef
		}

		editableColumns = append(editableColumns, columnMeta)
	}

	if editableCount == 0 {
		return fmt.Errorf("no editable columns found in result set")
	}

	metadata.Enabled = true
	metadata.Reason = ""
	metadata.PrimaryKeys = primaryKeys
	metadata.Columns = editableColumns
	metadata.Capabilities = &MutationCapabilities{
		CanInsert: true,
		CanUpdate: editableCount > 0,
		CanDelete: len(primaryKeys) > 0,
	}

	return nil
}

func (p *PostgresDatabase) loadTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
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

func (p *PostgresDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error) {
	metadata, _, err := p.computeEditableMetadata(ctx, query, columns, true)
	if metadata != nil {
		metadata.Pending = false
	}
	return metadata, err
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

func normalizeDriverValue(value interface{}) interface{} {
	if b, ok := value.([]byte); ok {
		return string(b)
	}
	return value
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

	metadata, metaErr := p.ComputeEditableMetadata(ctx, params.OriginalQuery, params.Columns)
	if metaErr != nil && (metadata == nil || metadata.Reason == "") {
		return metaErr
	}
	if metadata == nil || !metadata.Enabled {
		reason := "query is not editable"
		if metadata != nil && metadata.Reason != "" {
			reason = metadata.Reason
		} else if metaErr != nil {
			reason = metaErr.Error()
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

	// #nosec G201 - uses parameterized WHERE clauses with quoted identifiers
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

// InsertRow inserts a new row and returns the persisted values (including defaults)
func (p *PostgresDatabase) InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error) {
	if len(params.Values) == 0 {
		return nil, errors.New("no column values provided for insert")
	}

	metadata, metaErr := p.ComputeEditableMetadata(ctx, params.OriginalQuery, params.Columns)
	if metaErr != nil && (metadata == nil || metadata.Reason == "") {
		return nil, metaErr
	}
	if metadata == nil || !metadata.Enabled {
		reason := "query is not editable"
		if metadata != nil && metadata.Reason != "" {
			reason = metadata.Reason
		} else if metaErr != nil {
			reason = metaErr.Error()
		}
		return nil, errors.New(reason)
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
		return nil, errors.New("target table not specified")
	}

	columnLookup := make(map[string]EditableColumn, len(metadata.Columns)*2)
	for _, col := range metadata.Columns {
		if col.ResultName != "" {
			columnLookup[strings.ToLower(col.ResultName)] = col
		}
		if col.Name != "" {
			columnLookup[strings.ToLower(col.Name)] = col
		}
	}

	valueKeys := make([]string, 0, len(params.Values))
	for key := range params.Values {
		valueKeys = append(valueKeys, key)
	}
	sort.Slice(valueKeys, func(i, j int) bool {
		return strings.ToLower(valueKeys[i]) < strings.ToLower(valueKeys[j])
	})

	insertColumns := make([]string, 0, len(valueKeys))
	args := make([]interface{}, 0, len(valueKeys))
	argIndex := 1

	for _, key := range valueKeys {
		colMeta, ok := columnLookup[strings.ToLower(key)]
		if !ok || colMeta.Name == "" {
			return nil, fmt.Errorf("column %s is not editable in this result set", key)
		}
		insertColumns = append(insertColumns, p.QuoteIdentifier(colMeta.Name))
		args = append(args, params.Values[key])
		argIndex++
	}

	if len(insertColumns) == 0 {
		return nil, errors.New("no valid columns provided for insert")
	}

	tableIdentifier := p.QuoteIdentifier(table)
	if schema != "" {
		tableIdentifier = fmt.Sprintf("%s.%s", p.QuoteIdentifier(schema), tableIdentifier)
	}

	returningKeys := make([]string, 0, len(params.Columns))
	if len(params.Columns) == 0 {
		for _, col := range metadata.Columns {
			if col.ResultName != "" {
				returningKeys = append(returningKeys, col.ResultName)
			} else if col.Name != "" {
				returningKeys = append(returningKeys, col.Name)
			}
		}
	} else {
		returningKeys = append(returningKeys, params.Columns...)
	}

	returningColumns := make([]string, 0, len(returningKeys))
	for _, key := range returningKeys {
		if colMeta, ok := columnLookup[strings.ToLower(key)]; ok && colMeta.Name != "" {
			returningColumns = append(returningColumns, p.QuoteIdentifier(colMeta.Name))
		} else {
			returningColumns = append(returningColumns, p.QuoteIdentifier(key))
		}
	}
	if len(returningColumns) == 0 {
		returningColumns = []string{"*"}
	}

	valuesPlaceholders := make([]string, len(insertColumns))
	for i := range insertColumns {
		valuesPlaceholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// #nosec G201 - uses parameterized placeholders with quoted identifiers
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		tableIdentifier,
		strings.Join(insertColumns, ", "),
		strings.Join(valuesPlaceholders, ", "),
		strings.Join(returningColumns, ", "),
	)

	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	row := db.QueryRowContext(ctx, insertSQL, args...)
	resultValues := make([]interface{}, len(returningColumns))
	scanArgs := make([]interface{}, len(returningColumns))
	for i := range resultValues {
		scanArgs[i] = &resultValues[i]
	}

	if err := row.Scan(scanArgs...); err != nil {
		return nil, err
	}

	result := make(map[string]interface{}, len(returningColumns))
	for idx, key := range returningKeys {
		if idx >= len(resultValues) {
			break
		}
		result[key] = normalizeDriverValue(resultValues[idx])
	}
	// In case returningColumns fallback to *, ensure map not empty
	if len(result) == 0 {
		for idx, val := range resultValues {
			colName := fmt.Sprintf("column_%d", idx+1)
			result[colName] = normalizeDriverValue(val)
		}
	}

	return result, nil
}

// DeleteRow removes a row identified by the provided primary key values
func (p *PostgresDatabase) DeleteRow(ctx context.Context, params DeleteRowParams) error {
	if len(params.PrimaryKey) == 0 {
		return errors.New("primary key values are required for delete")
	}

	metadata, metaErr := p.ComputeEditableMetadata(ctx, params.OriginalQuery, params.Columns)
	if metaErr != nil && (metadata == nil || metadata.Reason == "") {
		return metaErr
	}
	if metadata == nil || !metadata.Enabled {
		reason := "query is not editable"
		if metadata != nil && metadata.Reason != "" {
			reason = metadata.Reason
		} else if metaErr != nil {
			reason = metaErr.Error()
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

	pkNames := make([]string, len(metadata.PrimaryKeys))
	copy(pkNames, metadata.PrimaryKeys)
	sort.Slice(pkNames, func(i, j int) bool {
		return strings.ToLower(pkNames[i]) < strings.ToLower(pkNames[j])
	})

	whereClauses := make([]string, 0, len(pkNames))
	args := make([]interface{}, 0, len(pkNames))
	argIndex := 1
	for _, pk := range pkNames {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", p.QuoteIdentifier(pk), argIndex))
		args = append(args, pkValues[pk])
		argIndex++
	}

	tableIdentifier := p.QuoteIdentifier(table)
	if schema != "" {
		tableIdentifier = fmt.Sprintf("%s.%s", p.QuoteIdentifier(schema), tableIdentifier)
	}

	// #nosec G201 - uses parameterized WHERE clauses with quoted identifiers
	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE %s", tableIdentifier, strings.Join(whereClauses, " AND "))

	db, err := p.pool.Get(ctx)
	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, deleteSQL, args...)
	if err != nil {
		return err
	}

	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return errors.New("no rows were deleted; data may have changed or the row no longer exists")
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
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
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

// ListDatabases returns all non-template databases
func (p *PostgresDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	db, err := p.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT datname
		FROM pg_database
		WHERE datallowconn = TRUE
		AND datistemplate = FALSE
		ORDER BY datname`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

// SwitchDatabase requires a reconnect for PostgreSQL connections
func (p *PostgresDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	if strings.TrimSpace(databaseName) == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	return ErrDatabaseSwitchRequiresReconnect
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
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
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
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
		}
	}()

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
	if structure, ok := p.structureCache.get(schema, table); ok {
		return structure, nil
	}

	structure, err := p.loadTableStructure(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	p.structureCache.set(schema, table, structure)
	return cloneTableStructure(structure), nil
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
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
		}
	}()

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
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
		}
	}()

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
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close rows")
		}
	}()

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
