package multiquery

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseSimpleMultiQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(nil, logger)

	query := `
		SELECT u.name, o.total
		FROM @prod.users u
		JOIN @staging.orders o ON u.id = o.user_id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Len(t, parsed.RequiredConnections, 2)
	assert.Contains(t, parsed.RequiredConnections, "prod")
	assert.Contains(t, parsed.RequiredConnections, "staging")
	assert.True(t, parsed.HasJoins)
}

func TestParser_ParseConnectionSyntax(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name        string
		query       string
		wantErr     bool
		connections []string
	}{
		{
			name:        "Single connection",
			query:       "SELECT * FROM @db1.users",
			wantErr:     false,
			connections: []string{"db1"},
		},
		{
			name:        "Multiple connections",
			query:       "SELECT * FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
			wantErr:     false,
			connections: []string{"db1", "db2"},
		},
		{
			name:        "With schema",
			query:       "SELECT * FROM @db1.public.users",
			wantErr:     false,
			connections: []string{"db1"},
		},
		{
			name:        "No @connections",
			query:       "SELECT * FROM users",
			wantErr:     false,
			connections: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParser(nil, logger)
			parsed, err := parser.Parse(tt.query)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.connections, parsed.RequiredConnections)
			}
		})
	}
}

func TestParser_ExtractConnectionRefs(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(nil, logger)

	tests := []struct {
		name     string
		query    string
		wantRefs int
	}{
		{
			name:     "Simple table reference",
			query:    "SELECT * FROM @prod.users",
			wantRefs: 1,
		},
		{
			name:     "With schema",
			query:    "SELECT * FROM @prod.public.users",
			wantRefs: 1,
		},
		{
			name:     "Multiple tables",
			query:    "SELECT * FROM @prod.users, @staging.orders",
			wantRefs: 2,
		},
		{
			name:     "With joins",
			query:    "SELECT * FROM @prod.users u JOIN @staging.orders o ON u.id = o.user_id",
			wantRefs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := parser.extractConnectionRefs(tt.query)
			assert.NoError(t, err)
			assert.Len(t, refs, tt.wantRefs)
		})
	}
}

func TestParser_DetectJoins(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(nil, logger)

	tests := []struct {
		name     string
		query    string
		wantJoin bool
	}{
		{
			name:     "Has JOIN",
			query:    "SELECT * FROM users u JOIN orders o ON u.id = o.user_id",
			wantJoin: true,
		},
		{
			name:     "Has LEFT JOIN",
			query:    "SELECT * FROM users u LEFT JOIN orders o ON u.id = o.user_id",
			wantJoin: true,
		},
		{
			name:     "No JOIN",
			query:    "SELECT * FROM users",
			wantJoin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasJoin := parser.detectJoins(tt.query)
			assert.Equal(t, tt.wantJoin, hasJoin)
		})
	}
}

func TestParser_DetectAggregation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(nil, logger)

	tests := []struct {
		name    string
		query   string
		wantAgg bool
	}{
		{
			name:    "Has GROUP BY",
			query:   "SELECT COUNT(*) FROM users GROUP BY country",
			wantAgg: true,
		},
		{
			name:    "Has COUNT",
			query:   "SELECT COUNT(*) FROM users",
			wantAgg: true,
		},
		{
			name:    "Has SUM",
			query:   "SELECT SUM(amount) FROM orders",
			wantAgg: true,
		},
		{
			name:    "No aggregation",
			query:   "SELECT * FROM users",
			wantAgg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAgg := parser.detectAggregation(tt.query)
			assert.Equal(t, tt.wantAgg, hasAgg)
		})
	}
}

func TestParser_Validate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name    string
		config  *Config
		parsed  *ParsedQuery
		wantErr bool
	}{
		{
			name: "Valid single connection",
			config: &Config{
				Enabled:            true,
				MaxConcurrentConns: 5,
			},
			parsed: &ParsedQuery{
				RequiredConnections: []string{"db1"},
			},
			wantErr: false,
		},
		{
			name: "Exceeds max connections",
			config: &Config{
				Enabled:            true,
				MaxConcurrentConns: 2,
			},
			parsed: &ParsedQuery{
				RequiredConnections: []string{"db1", "db2", "db3"},
			},
			wantErr: true,
		},
		{
			name: "Multi-query disabled",
			config: &Config{
				Enabled: false,
			},
			parsed: &ParsedQuery{
				RequiredConnections: []string{"db1", "db2"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParser(tt.config, logger)
			err := parser.Validate(tt.parsed)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

