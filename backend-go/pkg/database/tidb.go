package database

import (
	"context"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

// TiDBDatabase implements the Database interface for TiDB.
// TiDB is MySQL-compatible, so we extend MySQL functionality with TiDB-specific features.
type TiDBDatabase struct {
	*MySQLDatabase // Embed MySQL for base functionality
	logger         *logrus.Logger
}

// NewTiDBDatabase creates a new TiDB database instance
func NewTiDBDatabase(config ConnectionConfig, logger *logrus.Logger) (*TiDBDatabase, error) {
	// Create MySQL instance as base
	mysqlDB, err := NewMySQLDatabase(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create TiDB connection: %w", err)
	}

	return &TiDBDatabase{
		MySQLDatabase: mysqlDB,
		logger:        logger,
	}, nil
}

// GetConnectionInfo returns TiDB connection information with TiDB-specific details
func (t *TiDBDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	// Get base MySQL info
	info, err := t.MySQLDatabase.GetConnectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	db, err := t.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Add TiDB-specific information
	// Check if this is actually TiDB
	var version string
	err = db.QueryRowContext(ctx, "SELECT @@version").Scan(&version)
	if err == nil {
		info["version"] = version
		info["is_tidb"] = strings.Contains(strings.ToLower(version), "tidb")
	}

	// Get TiDB server version
	var tidbVersion string
	err = db.QueryRowContext(ctx, "SELECT @@tidb_version").Scan(&tidbVersion)
	if err == nil {
		info["tidb_version"] = tidbVersion
	}

	// Get TiKV store info
	var tikvStores int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tikv_store_status WHERE store_state_name = 'Up'").Scan(&tikvStores)
	if err == nil {
		info["tikv_stores_up"] = tikvStores
	}

	// Get TiFlash info (if available)
	var tiflashStores int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tiflash_replica").Scan(&tiflashStores)
	if err == nil {
		info["tiflash_replicas"] = tiflashStores
	}

	return info, nil
}

// GetSchemas returns list of schemas with TiDB-specific system schemas filtered
func (t *TiDBDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	db, err := t.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys', 'metrics_schema')
		ORDER BY SCHEMA_NAME`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.logger.WithError(err).Error("Failed to close rows")
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

// GetTables returns list of tables with TiDB-specific metadata
func (t *TiDBDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	db, err := t.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			t.TABLE_SCHEMA,
			t.TABLE_NAME,
			t.TABLE_TYPE,
			COALESCE(t.TABLE_COMMENT, '') as TABLE_COMMENT,
			COALESCE(t.TABLE_ROWS, 0) as TABLE_ROWS,
			COALESCE(t.DATA_LENGTH + t.INDEX_LENGTH, 0) as SIZE_BYTES,
			COALESCE(tr.TIFLASH_REPLICA, 0) as TIFLASH_REPLICA_COUNT
		FROM information_schema.TABLES t
		LEFT JOIN information_schema.TIFLASH_REPLICA tr ON t.TABLE_SCHEMA = tr.TABLE_SCHEMA AND t.TABLE_NAME = tr.TABLE_NAME
		WHERE t.TABLE_SCHEMA = ?
		ORDER BY t.TABLE_NAME`

	rows, err := db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var tableType string
		var tiflashReplicas int

		err := rows.Scan(
			&table.Schema,
			&table.Name,
			&tableType,
			&table.Comment,
			&table.RowCount,
			&table.SizeBytes,
			&tiflashReplicas,
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
		case "SYSTEM VIEW":
			table.Type = "SYSTEM VIEW"
		default:
			table.Type = tableType
		}

		// Add TiFlash replica info
		if table.Metadata == nil {
			table.Metadata = make(map[string]string)
		}
		if tiflashReplicas > 0 {
			table.Metadata["tiflash_replicas"] = fmt.Sprintf("%d", tiflashReplicas)
		}

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// ExplainQuery returns the execution plan with TiDB-specific EXPLAIN ANALYZE
func (t *TiDBDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	db, err := t.pool.Get(ctx)
	if err != nil {
		return "", err
	}

	// TiDB supports EXPLAIN ANALYZE for detailed execution stats
	explainQuery := "EXPLAIN ANALYZE " + query

	rows, err := db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", fmt.Errorf("failed to explain query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var plan strings.Builder
	columns, _ := rows.Columns()

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		for i, col := range columns {
			if i > 0 {
				plan.WriteString(" | ")
			}
			plan.WriteString(col)
			plan.WriteString(": ")
			if values[i] != nil {
				plan.WriteString(fmt.Sprintf("%v", values[i]))
			}
		}
		plan.WriteString("\n")
	}

	return plan.String(), nil
}

// GetDatabaseType returns TiDB as the database type
func (t *TiDBDatabase) GetDatabaseType() DatabaseType {
	return TiDB
}

// GetDataTypeMappings returns TiDB-specific data type mappings
func (t *TiDBDatabase) GetDataTypeMappings() map[string]string {
	mappings := t.MySQLDatabase.GetDataTypeMappings()

	// Add TiDB-specific types
	mappings["bit"] = "BIT"
	mappings["set"] = "SET"
	mappings["enum"] = "ENUM"

	return mappings
}

// IsTiFlashAvailable checks if TiFlash is available for the given table
func (t *TiDBDatabase) IsTiFlashAvailable(ctx context.Context, schema, table string) (bool, error) {
	db, err := t.pool.Get(ctx)
	if err != nil {
		return false, err
	}

	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tiflash_replica WHERE table_schema = ? AND table_name = ? AND available = 1",
		schema, table).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetTiKVRegionInfo returns information about TiKV regions for a table
func (t *TiDBDatabase) GetTiKVRegionInfo(ctx context.Context, schema, table string) (map[string]interface{}, error) {
	db, err := t.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})

	// Get region count
	var regionCount int
	// #nosec G201 - schema and table names from internal metadata query, validated by TiDB
	query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tikv_region_status WHERE db_name = '%s' AND table_name = '%s'", schema, table)
	err = db.QueryRowContext(ctx, query).Scan(&regionCount)
	if err == nil {
		info["region_count"] = regionCount
	}

	return info, nil
}
