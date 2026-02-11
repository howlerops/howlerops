package multiquery

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// QueryParser parses multi-database queries with @connection syntax
type QueryParser struct {
	config *Config
	logger *logrus.Logger
}

// NewQueryParser creates a new query parser
func NewQueryParser(config *Config, logger *logrus.Logger) *QueryParser {
	if config == nil {
		config = &Config{
			Enabled:         true,
			DefaultStrategy: StrategyAuto,
		}
	}
	return &QueryParser{
		config: config,
		logger: logger,
	}
}

// Parse parses a SQL query and extracts connection references
func (p *QueryParser) Parse(query string) (*ParsedQuery, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	parsed := &ParsedQuery{
		OriginalSQL:         query,
		RequiredConnections: []string{},
		Segments:            []QuerySegment{},
		Tables:              []string{},
		SuggestedStrategy:   p.config.DefaultStrategy,
	}

	// Extract @connection references
	refs, err := p.extractConnectionRefs(query)
	if err != nil {
		return nil, fmt.Errorf("failed to extract connections: %w", err)
	}

	// If no @connections found, this is a single-database query
	if len(refs) == 0 {
		parsed.SuggestedStrategy = StrategyAuto
		return parsed, nil
	}

	// Extract unique connection IDs
	connMap := make(map[string]bool)
	for _, ref := range refs {
		connMap[ref.Alias] = true

		// Build table reference
		tableRef := TableRef{
			ConnectionID: ref.Alias,
			Schema:       ref.Schema,
			Table:        ref.Table,
		}

		// Add to segments
		found := false
		for i := range parsed.Segments {
			if parsed.Segments[i].ConnectionID == ref.Alias {
				parsed.Segments[i].Tables = append(parsed.Segments[i].Tables, tableRef)
				found = true
				break
			}
		}
		if !found {
			parsed.Segments = append(parsed.Segments, QuerySegment{
				ConnectionID: ref.Alias,
				Tables:       []TableRef{tableRef},
			})
		}

		// Track unique tables
		tableName := ref.Table
		if ref.Schema != "" {
			tableName = ref.Schema + "." + ref.Table
		}
		parsed.Tables = append(parsed.Tables, tableName)
	}

	// Convert map to slice
	for conn := range connMap {
		parsed.RequiredConnections = append(parsed.RequiredConnections, conn)
	}

	// Detect query characteristics
	parsed.HasJoins = p.detectJoins(query)
	parsed.HasAggregation = p.detectAggregation(query)

	// Suggest execution strategy
	parsed.SuggestedStrategy = p.suggestStrategy(parsed)

	return parsed, nil
}

// extractConnectionRefs extracts @connection.schema.table references from SQL
func (p *QueryParser) extractConnectionRefs(query string) ([]ConnectionRef, error) {
	// Pattern: @connection_alias.schema.table or @connection_alias.table
	// Note: [\w-]+ allows hyphens in connection names (e.g., @Prod-Leviosa)
	pattern := regexp.MustCompile(`@([\w-]+)\.(?:([\w-]+)\.)?([\w-]+)`)

	matches := pattern.FindAllStringSubmatch(query, -1)
	refs := make([]ConnectionRef, 0, len(matches))

	for _, match := range matches {
		ref := ConnectionRef{
			Alias: match[1],
		}

		// If we have 4 groups, it's @conn.schema.table
		if match[2] != "" {
			ref.Schema = match[2]
			ref.Table = match[3]
		} else {
			// Otherwise it's @conn.table
			ref.Table = match[3]
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

// detectJoins checks if the query contains JOIN operations
func (p *QueryParser) detectJoins(query string) bool {
	// Normalize whitespace: replace tabs and newlines with spaces, collapse multiple spaces
	normalized := strings.Join(strings.Fields(query), " ")
	upper := strings.ToUpper(normalized)

	joinKeywords := []string{" JOIN ", " LEFT JOIN ", " RIGHT JOIN ", " INNER JOIN ", " OUTER JOIN ", " CROSS JOIN "}
	for _, keyword := range joinKeywords {
		if strings.Contains(upper, keyword) {
			return true
		}
	}
	return false
}

// detectAggregation checks if the query contains aggregation
func (p *QueryParser) detectAggregation(query string) bool {
	upper := strings.ToUpper(query)
	aggregateKeywords := []string{" GROUP BY ", " HAVING ", "COUNT(", "SUM(", "AVG(", "MAX(", "MIN("}
	for _, keyword := range aggregateKeywords {
		if strings.Contains(upper, keyword) {
			return true
		}
	}
	return false
}

// suggestStrategy suggests the best execution strategy based on query characteristics
func (p *QueryParser) suggestStrategy(parsed *ParsedQuery) ExecutionStrategy {
	// If only one connection, no special strategy needed
	if len(parsed.RequiredConnections) <= 1 {
		return StrategyAuto
	}

	// If query has joins between connections, use federated
	if parsed.HasJoins {
		return StrategyFederated
	}

	// If query has aggregation, use federated to ensure correct results
	if parsed.HasAggregation {
		return StrategyFederated
	}

	// Default to federated for multi-connection queries
	return StrategyFederated
}

// Validate checks if a parsed query is valid
func (p *QueryParser) Validate(parsed *ParsedQuery) error {
	if parsed == nil {
		return fmt.Errorf("parsed query cannot be nil")
	}

	// Check if multi-query is enabled
	if len(parsed.RequiredConnections) > 1 && !p.config.Enabled {
		return fmt.Errorf("multi-database queries are not enabled")
	}

	// Check max connections
	if p.config.MaxConcurrentConns > 0 && len(parsed.RequiredConnections) > p.config.MaxConcurrentConns {
		return fmt.Errorf("query requires %d connections, but maximum allowed is %d",
			len(parsed.RequiredConnections), p.config.MaxConcurrentConns)
	}

	// Validate operation type if restrictions are configured
	if len(p.config.AllowedOperations) > 0 {
		if err := p.validateOperation(parsed.OriginalSQL); err != nil {
			return err
		}
	}

	return nil
}

// validateOperation checks if the query operation is allowed
func (p *QueryParser) validateOperation(query string) error {
	upper := strings.TrimSpace(strings.ToUpper(query))

	// Extract the first keyword
	parts := strings.Fields(upper)
	if len(parts) == 0 {
		return fmt.Errorf("empty query")
	}

	operation := parts[0]

	// Check if operation is in allowed list
	allowed := false
	for _, allowedOp := range p.config.AllowedOperations {
		if strings.ToUpper(allowedOp) == operation {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("operation %s is not allowed in multi-database queries", operation)
	}

	return nil
}
