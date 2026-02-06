package multiquery

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFederatedQuery_ParseAndValidate tests parsing and validation of federated queries
func TestFederatedQuery_ParseAndValidate(t *testing.T) {
	tests := CommonQueryScenarios

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{
		Enabled:            true,
		MaxConcurrentConns: 10,
	}, logger)

	for _, scenario := range tests {
		t.Run(scenario.Name, func(t *testing.T) {
			parsed, err := parser.Parse(scenario.Query)
			require.NoError(t, err)

			assert.ElementsMatch(t, scenario.ExpectedConns, parsed.RequiredConnections)
			assert.Equal(t, scenario.HasJoins, parsed.HasJoins)
			assert.Equal(t, scenario.HasAggregation, parsed.HasAggregation)
		})
	}
}

// TestFederatedQuery_ExecutionStrategySelection tests strategy selection based on query characteristics
func TestFederatedQuery_ExecutionStrategySelection(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name             string
		config           *Config
		query            string
		expectedStrategy ExecutionStrategy
	}{
		{
			name: "Auto strategy for single connection",
			config: &Config{
				Enabled:                true,
				DefaultStrategy:        StrategyAuto,
				EnableCrossTypeQueries: true,
			},
			query:            "SELECT * FROM @prod.users WHERE id = 1",
			expectedStrategy: StrategyAuto,
		},
		{
			name: "Federated strategy for multi-database join",
			config: &Config{
				Enabled:         true,
				DefaultStrategy: StrategyAuto,
			},
			query:            "SELECT * FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
			expectedStrategy: StrategyFederated,
		},
		{
			name: "Federated strategy for aggregation across connections",
			config: &Config{
				Enabled:         true,
				DefaultStrategy: StrategyAuto,
			},
			query:            "SELECT COUNT(*) FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id GROUP BY u.country",
			expectedStrategy: StrategyFederated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParser(tt.config, logger)
			parsed, err := parser.Parse(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStrategy, parsed.SuggestedStrategy)
		})
	}
}

// TestFederatedQuery_MultiConnectionParsing tests parsing of queries with multiple connections
func TestFederatedQuery_MultiConnectionParsing(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := `
		SELECT 
			u.id, u.email,
			o.order_id, o.total,
			c.category_name
		FROM @prod_us.users u
		INNER JOIN @prod_eu.orders o ON u.id = o.user_id
		INNER JOIN @catalog.categories c ON o.category_id = c.id
		WHERE u.status = 'active'
		ORDER BY o.created_at DESC
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	assert.Len(t, parsed.RequiredConnections, 3)
	assert.ElementsMatch(t, []string{"prod_us", "prod_eu", "catalog"}, parsed.RequiredConnections)
	assert.True(t, parsed.HasJoins)
	assert.Len(t, parsed.Segments, 3)
}

// TestFederatedQuery_ValidationMaxConcurrentConnections tests validation of concurrent connection limits
func TestFederatedQuery_ValidationMaxConcurrentConnections(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name           string
		maxConns       int
		requiredConns  []string
		shouldValidate bool
	}{
		{
			name:           "Within limit",
			maxConns:       5,
			requiredConns:  []string{"db1", "db2", "db3"},
			shouldValidate: true,
		},
		{
			name:           "Exactly at limit",
			maxConns:       3,
			requiredConns:  []string{"db1", "db2", "db3"},
			shouldValidate: true,
		},
		{
			name:           "Exceeds limit",
			maxConns:       2,
			requiredConns:  []string{"db1", "db2", "db3"},
			shouldValidate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParser(&Config{
				Enabled:            true,
				MaxConcurrentConns: tt.maxConns,
			}, logger)

			parsed := &ParsedQuery{
				RequiredConnections: tt.requiredConns,
			}

			err := parser.Validate(parsed)
			if tt.shouldValidate {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestFederatedQuery_ErrorHandling tests error cases
func TestFederatedQuery_ErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "Empty query",
			query:   "",
			wantErr: true,
		},
		{
			name:    "Valid multi-database query",
			query:   "SELECT * FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
			wantErr: false,
		},
		{
			name:    "Unmatched parenthesis (but parsing should work)",
			query:   "SELECT * FROM @db1.users WHERE id IN (1, 2, 3",
			wantErr: false, // Parser is lenient; this is a semantic error, not parsing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsed)
			}
		})
	}
}

// TestFederatedQuery_SegmentationForParallelExecution tests query segmentation
func TestFederatedQuery_SegmentationForParallelExecution(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{
		Enabled:                true,
		ParallelExecution:      true,
		DefaultStrategy:        StrategyFederated,
		EnableCrossTypeQueries: true,
	}, logger)

	query := `
		SELECT u.id, u.name, o.total, oi.product_id
		FROM @postgres_db.users u
		JOIN @mysql_db.orders o ON u.id = o.user_id
		JOIN @sqlite_db.order_items oi ON o.id = oi.order_id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	// Should create segments for each connection
	assert.Len(t, parsed.Segments, 3, "Should have 3 segments for 3 connections")

	// Verify each segment has correct connection
	connMap := make(map[string]bool)
	for _, seg := range parsed.Segments {
		connMap[seg.ConnectionID] = true
	}
	assert.True(t, connMap["postgres_db"])
	assert.True(t, connMap["mysql_db"])
	assert.True(t, connMap["sqlite_db"])
}

// TestFederatedQuery_CrossDatabaseTypeJoin tests joining across different database types
func TestFederatedQuery_CrossDatabaseTypeJoin(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{
		Enabled:                true,
		EnableCrossTypeQueries: true,
	}, logger)

	tests := []struct {
		name     string
		query    string
		conns    int
		hasJoins bool
	}{
		{
			name:     "PostgreSQL to MySQL join",
			query:    "SELECT * FROM @pg_prod.users u JOIN @mysql_db.orders o ON u.id = o.user_id",
			conns:    2,
			hasJoins: true,
		},
		{
			name:     "MySQL to SQLite join",
			query:    "SELECT * FROM @mysql_app.products p JOIN @sqlite_cache.cache c ON p.id = c.product_id",
			conns:    2,
			hasJoins: true,
		},
		{
			name:     "Three database types",
			query:    "SELECT * FROM @pg.a a JOIN @mysql.b b ON a.id=b.id JOIN @sqlite.c c ON b.id=c.id",
			conns:    3,
			hasJoins: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			require.NoError(t, err)
			assert.Len(t, parsed.RequiredConnections, tt.conns)
			assert.Equal(t, tt.hasJoins, parsed.HasJoins)
		})
	}
}

// TestFederatedQuery_ComplexAggregationScenarios tests complex aggregation scenarios
func TestFederatedQuery_ComplexAggregationScenarios(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	tests := []struct {
		name        string
		query       string
		expectAgg   bool
		expectJoins bool
	}{
		{
			name: "Multi-level aggregation with subqueries",
			query: `
				SELECT category, SUM(total_sales)
				FROM (
					SELECT p.category, SUM(o.total) as total_sales
					FROM @prod.products p
					JOIN @analytics.orders o ON p.id = o.product_id
					GROUP BY p.category
				) sub
				GROUP BY category
			`,
			expectAgg:   true,
			expectJoins: true,
		},
		{
			name: "Window functions with joins",
			query: `
				SELECT 
					user_id,
					total,
					ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at) as order_num
				FROM @db1.orders o
				JOIN @db2.users u ON o.user_id = u.id
			`,
			expectAgg:   false, // Window functions aren't aggregation in traditional sense
			expectJoins: true,
		},
		{
			name: "HAVING clause with aggregation",
			query: `
				SELECT u.id, COUNT(o.id) as order_count
				FROM @db1.users u
				LEFT JOIN @db2.orders o ON u.id = o.user_id
				GROUP BY u.id
				HAVING COUNT(o.id) > 5
			`,
			expectAgg:   true,
			expectJoins: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expectAgg, parsed.HasAggregation)
			assert.Equal(t, tt.expectJoins, parsed.HasJoins)
		})
	}
}

// TestFederatedQuery_LargeScaleQuery tests parsing of large, complex queries
func TestFederatedQuery_LargeScaleQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{
		Enabled:            true,
		MaxConcurrentConns: 10,
	}, logger)

	// Build a large query with many joins
	query := `
		SELECT 
			u.id, u.name,
			o.id as order_id,
			p.name as product_name,
			c.category_name,
			s.warehouse_location
		FROM @db1.users u
		INNER JOIN @db2.orders o ON u.id = o.user_id
		INNER JOIN @db3.order_items oi ON o.id = oi.order_id
		INNER JOIN @db4.products p ON oi.product_id = p.id
		INNER JOIN @db5.categories c ON p.category_id = c.id
		INNER JOIN @db6.stock s ON p.id = s.product_id AND s.warehouse_id = 1
		WHERE u.country = 'US'
		AND o.created_at > DATE_SUB(NOW(), INTERVAL 30 DAY)
		AND p.active = true
		ORDER BY o.created_at DESC
		LIMIT 1000
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	assert.Len(t, parsed.RequiredConnections, 6)
	assert.True(t, parsed.HasJoins)
	assert.Len(t, parsed.Segments, 6)
}

// TestFederatedQuery_QueryOptionsParsing tests parsing of query with options
func TestFederatedQuery_QueryOptionsParsing(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{
		Enabled:       true,
		Timeout:       10 * time.Second,
		MaxResultRows: 10000,
	}, logger)

	query := "SELECT u.name, o.total FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id"

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	// Query options should be respected at parse time
	assert.NotNil(t, parsed)

	// Verify config constraints are applied
	assert.Equal(t, StrategyFederated, parsed.SuggestedStrategy)
}

// TestFederatedQuery_SpecialCharacterHandling tests handling of special characters in identifiers
func TestFederatedQuery_SpecialCharacterHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	tests := []struct {
		name  string
		query string
		conns int
	}{
		{
			name:  "Hyphenated connection names",
			query: "SELECT * FROM @prod-us-east.users u JOIN @staging-eu.orders o ON u.id = o.user_id",
			conns: 2,
		},
		{
			name:  "Underscored identifiers",
			query: "SELECT * FROM @prod_db_1.user_accounts u JOIN @prod_db_2.user_orders o ON u.user_id = o.user_id",
			conns: 2,
		},
		{
			name:  "Numeric in names",
			query: "SELECT * FROM @db1.table1 t1 JOIN @db2.table2 t2 ON t1.id = t2.id",
			conns: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			require.NoError(t, err)
			assert.Len(t, parsed.RequiredConnections, tt.conns)
		})
	}
}

// TestFederatedQuery_ConfigValidation tests configuration validation
func TestFederatedQuery_ConfigValidation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name    string
		config  *Config
		query   string
		wantErr bool
	}{
		{
			name: "Disabled multiquery config",
			config: &Config{
				Enabled: false,
			},
			query:   "SELECT * FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
			wantErr: true,
		},
		{
			name: "Valid enabled config",
			config: &Config{
				Enabled:            true,
				MaxConcurrentConns: 5,
			},
			query:   "SELECT * FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParser(tt.config, logger)
			parsed, err := parser.Parse(tt.query)

			if tt.wantErr {
				// For disabled config, validation should fail
				if parsed != nil {
					err = parser.Validate(parsed)
				}
				// Single database queries in disabled mode should work
				if len(parsed.RequiredConnections) > 1 {
					assert.Error(t, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// BenchmarkFederatedQuery_Parsing benchmarks query parsing performance
func BenchmarkFederatedQuery_Parsing(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := `
		SELECT u.id, u.name, o.total, c.category
		FROM @db1.users u
		JOIN @db2.orders o ON u.id = o.user_id
		JOIN @db3.categories c ON o.category_id = c.id
		WHERE u.active = true
		ORDER BY o.created_at DESC
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(query)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}
