package database

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ElasticsearchDatabase implements the Database interface for Elasticsearch and OpenSearch
type ElasticsearchDatabase struct {
	config     ConnectionConfig
	logger     *logrus.Logger
	httpClient *http.Client
	baseURL    string
	authHeader string
	stats      connectionStats
}

// connectionStats tracks connection statistics for non-pool connections
type connectionStats struct {
	requestCount  int64
	errorCount    int64
	lastRequestAt time.Time
}

// NewElasticsearchDatabase creates a new Elasticsearch database instance
func NewElasticsearchDatabase(config ConnectionConfig, logger *logrus.Logger) (*ElasticsearchDatabase, error) {
	es := &ElasticsearchDatabase{
		config: config,
		logger: logger,
	}

	if err := es.Connect(context.Background(), config); err != nil {
		return nil, err
	}

	return es, nil
}

// Connect establishes a connection to Elasticsearch
func (es *ElasticsearchDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	es.config = config

	// Set default port if not specified
	if config.Port == 0 {
		config.Port = 9200
	}

	// Build base URL
	scheme := "http"
	if config.SSLMode == "require" || config.SSLMode == "verify-full" {
		scheme = "https"
	}
	es.baseURL = fmt.Sprintf("%s://%s:%d", scheme, config.Host, config.Port)

	// Setup authentication
	if config.Parameters != nil {
		// Check for API key authentication
		if apiKey, ok := config.Parameters["api_key"]; ok {
			es.authHeader = "ApiKey " + apiKey
		}
	}
	// Fall back to basic auth if no API key
	if es.authHeader == "" && config.Username != "" {
		es.authHeader = "Basic " + basicAuth(config.Username, config.Password)
	}

	// Create HTTP client with timeout and TLS configuration
	timeout := 30 * time.Second
	if config.ConnectionTimeout > 0 {
		timeout = config.ConnectionTimeout
	}

	// #nosec G402 - InsecureSkipVerify controlled by user config for dev/test environments
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.SSLMode == "skip-verify" || config.SSLMode == "disable",
	}

	es.httpClient = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        25,
			MaxIdleConnsPerHost: 25,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Test connection
	if err := es.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}

	es.logger.WithFields(logrus.Fields{
		"host": config.Host,
		"port": config.Port,
		"type": config.Type,
	}).Info("Elasticsearch connection established successfully")

	return nil
}

// basicAuth creates a basic auth header value
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64Encode([]byte(auth))
}

// base64Encode encodes bytes to base64
func base64Encode(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var buf bytes.Buffer

	for i := 0; i < len(data); i += 3 {
		b := make([]byte, 4)
		n := 3
		if i+2 >= len(data) {
			n = len(data) - i
		}

		var val uint32
		for j := 0; j < n; j++ {
			// #nosec G115 - base64 encoding: j is 0-2, shift values (16,8,0) are safe
			val |= uint32(data[i+j]) << uint(16-j*8)
		}

		for j := 0; j < 4; j++ {
			if i+j*3/4 < len(data) || j < (n*8+5)/6 {
				// #nosec G115 - base64 encoding: j is 0-3, shift values (18,12,6,0) are safe
				idx := (val >> uint(18-j*6)) & 0x3F
				b[j] = base64Table[idx]
			} else {
				b[j] = '='
			}
		}
		buf.Write(b)
	}

	return buf.String()
}

// Disconnect closes the Elasticsearch connection
func (es *ElasticsearchDatabase) Disconnect() error {
	if es.httpClient != nil {
		es.httpClient.CloseIdleConnections()
	}
	es.logger.Info("Elasticsearch connection closed")
	return nil
}

// Ping tests the Elasticsearch connection
func (es *ElasticsearchDatabase) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", es.baseURL, nil)
	if err != nil {
		return err
	}

	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ping failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetConnectionInfo returns Elasticsearch connection information
func (es *ElasticsearchDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", es.baseURL, nil)
	if err != nil {
		return nil, err
	}

	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get connection info: status %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return info, nil
}

// Execute runs a SQL query using Elasticsearch SQL API
func (es *ElasticsearchDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	return es.ExecuteWithOptions(ctx, query, nil, args...)
}

// ExecuteWithOptions runs a SQL query with options using Elasticsearch SQL API
func (es *ElasticsearchDatabase) ExecuteWithOptions(ctx context.Context, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	start := time.Now()
	es.stats.requestCount++
	es.stats.lastRequestAt = start

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
			countResult, err := es.executeCountQuery(ctx, countQuery, args)
			if err != nil {
				es.logger.WithError(err).Warn("Failed to get total count for pagination")
				totalRows = 0
			} else {
				totalRows = countResult
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

	// Prepare SQL query request
	sqlURL := es.baseURL + "/_sql?format=json"

	queryBody := map[string]interface{}{
		"query": modifiedQuery,
	}

	// Add parameters if provided
	if len(args) > 0 {
		params := make([]interface{}, len(args))
		copy(params, args)
		queryBody["params"] = params
	}

	bodyBytes, err := json.Marshal(queryBody)
	if err != nil {
		es.stats.errorCount++
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sqlURL, bytes.NewReader(bodyBytes))
	if err != nil {
		es.stats.errorCount++
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		es.stats.errorCount++
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		es.stats.errorCount++
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	if resp.StatusCode >= 400 {
		es.stats.errorCount++
		var errResp map[string]interface{}
		var resultErr error
		if unmarshalErr := json.Unmarshal(body, &errResp); unmarshalErr == nil {
			if errMsg, ok := errResp["error"].(map[string]interface{}); ok {
				if reason, ok := errMsg["reason"].(string); ok {
					resultErr = fmt.Errorf("elasticsearch error: %s", reason)
				} else {
					resultErr = fmt.Errorf("elasticsearch error: status %d", resp.StatusCode)
				}
			} else {
				resultErr = fmt.Errorf("elasticsearch error: %s", string(body))
			}
		} else {
			resultErr = fmt.Errorf("elasticsearch error: status %d - %s", resp.StatusCode, string(body))
		}
		return &QueryResult{
			Error:    resultErr,
			Duration: time.Since(start),
		}, resultErr
	}

	// Parse SQL response
	var sqlResp elasticsearchSQLResponse
	if err := json.Unmarshal(body, &sqlResp); err != nil {
		es.stats.errorCount++
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	// Convert to QueryResult
	result := &QueryResult{
		Columns:  make([]string, len(sqlResp.Columns)),
		Rows:     make([][]interface{}, 0, len(sqlResp.Rows)),
		Duration: time.Since(start),
	}

	for i, col := range sqlResp.Columns {
		result.Columns[i] = col.Name
	}

	// Step 3: Normalize all rows
	for _, row := range sqlResp.Rows {
		normalizedRow := make([]interface{}, len(row))
		for i, val := range row {
			normalizedRow[i] = NormalizeValue(val)
		}
		result.Rows = append(result.Rows, normalizedRow)
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = time.Since(start)

	// Step 4: Set pagination metadata
	if opts != nil && opts.Limit > 0 {
		result.TotalRows = totalRows
		result.PagedRows = int64(len(result.Rows))
		result.Offset = opts.Offset
		result.HasMore = (int64(opts.Offset) + result.PagedRows) < totalRows
	}

	// Elasticsearch indices are not directly editable
	metadata := newEditableMetadata(result.Columns)
	metadata.Reason = "Elasticsearch indices are not directly editable"
	result.Editable = metadata

	return result, nil
}

// executeCountQuery executes a count query for pagination
func (es *ElasticsearchDatabase) executeCountQuery(ctx context.Context, countQuery string, args []interface{}) (int64, error) {
	sqlURL := es.baseURL + "/_sql?format=json"

	queryBody := map[string]interface{}{
		"query": countQuery,
	}

	if len(args) > 0 {
		params := make([]interface{}, len(args))
		copy(params, args)
		queryBody["params"] = params
	}

	bodyBytes, err := json.Marshal(queryBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sqlURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("count query failed: status %d - %s", resp.StatusCode, string(body))
	}

	var sqlResp elasticsearchSQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&sqlResp); err != nil {
		return 0, err
	}

	if len(sqlResp.Rows) > 0 && len(sqlResp.Rows[0]) > 0 {
		switch v := sqlResp.Rows[0][0].(type) {
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case int:
			return int64(v), nil
		default:
			return 0, fmt.Errorf("unexpected count type: %T", v)
		}
	}

	return 0, nil
}

// elasticsearchSQLResponse represents the response from Elasticsearch SQL API
type elasticsearchSQLResponse struct {
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"columns"`
	Rows   [][]interface{} `json:"rows"`
	Cursor string          `json:"cursor,omitempty"`
}

// ExecuteStream executes a query and streams results in batches
func (es *ElasticsearchDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	// Use cursor for pagination
	sqlURL := es.baseURL + "/_sql?format=json"

	queryBody := map[string]interface{}{
		"query":      query,
		"fetch_size": batchSize,
	}

	if len(args) > 0 {
		params := make([]interface{}, len(args))
		copy(params, args)
		queryBody["params"] = params
	}

	bodyBytes, err := json.Marshal(queryBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sqlURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stream query failed: status %d - %s", resp.StatusCode, string(body))
	}

	var sqlResp elasticsearchSQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&sqlResp); err != nil {
		return err
	}

	// Send first batch
	if len(sqlResp.Rows) > 0 {
		if err := callback(sqlResp.Rows); err != nil {
			return err
		}
	}

	// Continue fetching with cursor if available
	cursor := sqlResp.Cursor
	for cursor != "" {
		cursorBody := map[string]interface{}{
			"cursor": cursor,
		}

		bodyBytes, err := json.Marshal(cursorBody)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", sqlURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		if es.authHeader != "" {
			req.Header.Set("Authorization", es.authHeader)
		}

		resp, err := es.httpClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close() // Best-effort close
			return fmt.Errorf("cursor fetch failed: status %d - %s", resp.StatusCode, string(body))
		}

		var cursorResp elasticsearchSQLResponse
		if err := json.NewDecoder(resp.Body).Decode(&cursorResp); err != nil {
			_ = resp.Body.Close() // Best-effort close
			return err
		}
		_ = resp.Body.Close() // Best-effort close

		if len(cursorResp.Rows) > 0 {
			if err := callback(cursorResp.Rows); err != nil {
				return err
			}
		}

		cursor = cursorResp.Cursor
	}

	return nil
}

// ExplainQuery returns the execution plan for a query
func (es *ElasticsearchDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	// Use Elasticsearch EXPLAIN endpoint
	explainURL := es.baseURL + "/_sql/translate"

	queryBody := map[string]interface{}{
		"query": query,
	}

	if len(args) > 0 {
		params := make([]interface{}, len(args))
		copy(params, args)
		queryBody["params"] = params
	}

	bodyBytes, err := json.Marshal(queryBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", explainURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("explain failed: status %d - %s", resp.StatusCode, string(body))
	}

	// Pretty print the JSON response
	var explain map[string]interface{}
	if err := json.Unmarshal(body, &explain); err != nil {
		return string(body), nil
	}

	prettyJSON, err := json.MarshalIndent(explain, "", "  ")
	if err != nil {
		return string(body), nil
	}

	return string(prettyJSON), nil
}

// ComputeEditableMetadata returns metadata indicating Elasticsearch indices are not editable
func (es *ElasticsearchDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error) {
	metadata := newEditableMetadata(columns)
	metadata.Reason = "Elasticsearch indices are not directly editable"
	metadata.Capabilities = &MutationCapabilities{
		CanInsert: false,
		CanUpdate: false,
		CanDelete: false,
		Reason:    metadata.Reason,
	}
	return metadata, nil
}

// GetSchemas returns list of indices in Elasticsearch
func (es *ElasticsearchDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	// In Elasticsearch, we can use _cat/indices to list indices
	// We'll return a single "schema" called "default" as ES doesn't have schemas
	return []string{"default"}, nil
}

// GetTables returns list of indices and aliases
func (es *ElasticsearchDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	indicesURL := es.baseURL + "/_cat/indices?format=json&h=index,docs.count,store.size,health,status"

	req, err := http.NewRequestWithContext(ctx, "GET", indicesURL, nil)
	if err != nil {
		return nil, err
	}

	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list indices: status %d - %s", resp.StatusCode, string(body))
	}

	var indices []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&indices); err != nil {
		return nil, err
	}

	tables := make([]TableInfo, 0, len(indices))
	for _, idx := range indices {
		indexName, _ := idx["index"].(string)

		// Skip system indices (starting with .)
		if strings.HasPrefix(indexName, ".") {
			continue
		}

		docsCount := int64(0)
		if count, ok := idx["docs.count"].(string); ok {
			_, _ = fmt.Sscanf(count, "%d", &docsCount) // Best-effort parsing
		}

		sizeBytes := int64(0)
		if size, ok := idx["store.size"].(string); ok {
			// Parse size like "1.2kb", "5mb", etc.
			sizeBytes = parseSizeString(size)
		}

		table := TableInfo{
			Schema:    schema,
			Name:      indexName,
			Type:      "INDEX",
			RowCount:  docsCount,
			SizeBytes: sizeBytes,
			Metadata:  make(map[string]string),
		}

		if health, ok := idx["health"].(string); ok {
			table.Metadata["health"] = health
		}
		if status, ok := idx["status"].(string); ok {
			table.Metadata["status"] = status
		}

		tables = append(tables, table)
	}

	return tables, nil
}

// parseSizeString parses Elasticsearch size strings like "1.2kb", "5mb"
func parseSizeString(size string) int64 {
	size = strings.ToLower(strings.TrimSpace(size))

	var num float64
	var unit string
	_, _ = fmt.Sscanf(size, "%f%s", &num, &unit) // Best-effort parsing

	multiplier := int64(1)
	switch unit {
	case "kb":
		multiplier = 1024
	case "mb":
		multiplier = 1024 * 1024
	case "gb":
		multiplier = 1024 * 1024 * 1024
	case "tb":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "b":
		multiplier = 1
	}

	return int64(num * float64(multiplier))
}

// GetTableStructure returns detailed structure information for an index
func (es *ElasticsearchDatabase) GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	// Get index mapping
	mappingURL := es.baseURL + "/" + url.PathEscape(table) + "/_mapping"

	req, err := http.NewRequestWithContext(ctx, "GET", mappingURL, nil)
	if err != nil {
		return nil, err
	}

	if es.authHeader != "" {
		req.Header.Set("Authorization", es.authHeader)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			es.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get index mapping: status %d - %s", resp.StatusCode, string(body))
	}

	var mappingResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mappingResp); err != nil {
		return nil, err
	}

	structure := &TableStructure{
		Table: TableInfo{
			Schema: schema,
			Name:   table,
			Type:   "INDEX",
		},
		Columns:     make([]ColumnInfo, 0),
		Indexes:     make([]IndexInfo, 0),
		ForeignKeys: make([]ForeignKeyInfo, 0),
		Triggers:    make([]string, 0),
		Statistics:  make(map[string]string),
	}

	// Extract field mappings
	if indexMapping, ok := mappingResp[table].(map[string]interface{}); ok {
		if mappings, ok := indexMapping["mappings"].(map[string]interface{}); ok {
			if properties, ok := mappings["properties"].(map[string]interface{}); ok {
				position := 1
				for fieldName, fieldDef := range properties {
					col := ColumnInfo{
						Name:            fieldName,
						OrdinalPosition: position,
						Nullable:        true, // ES fields are generally nullable
						Metadata:        make(map[string]string),
					}

					if defMap, ok := fieldDef.(map[string]interface{}); ok {
						if fieldType, ok := defMap["type"].(string); ok {
							col.DataType = fieldType
						}
					}

					structure.Columns = append(structure.Columns, col)
					position++
				}
			}
		}
	}

	return structure, nil
}

// BeginTransaction is not supported for Elasticsearch
func (es *ElasticsearchDatabase) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("elasticsearch does not support transactions")
}

// UpdateRow is not supported for Elasticsearch
func (es *ElasticsearchDatabase) UpdateRow(ctx context.Context, params UpdateRowParams) error {
	return fmt.Errorf("direct row updates are not supported in Elasticsearch (use Update API)")
}

// InsertRow is not supported for Elasticsearch
func (es *ElasticsearchDatabase) InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error) {
	return nil, fmt.Errorf("direct row inserts are not supported in Elasticsearch via SQL interface (use Index API)")
}

// DeleteRow is not supported for Elasticsearch
func (es *ElasticsearchDatabase) DeleteRow(ctx context.Context, params DeleteRowParams) error {
	return fmt.Errorf("direct row deletes are not supported in Elasticsearch via SQL interface (use Delete API)")
}

// ListDatabases returns an error as Elasticsearch does not support database switching
func (es *ElasticsearchDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	return nil, ErrDatabaseSwitchNotSupported
}

// SwitchDatabase is not supported for Elasticsearch
func (es *ElasticsearchDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	return ErrDatabaseSwitchNotSupported
}

// GetDatabaseType returns the database type
func (es *ElasticsearchDatabase) GetDatabaseType() DatabaseType {
	return Elasticsearch
}

// GetConnectionStats returns connection statistics
func (es *ElasticsearchDatabase) GetConnectionStats() PoolStats {
	return PoolStats{
		OpenConnections: 1,
		InUse:           1,
		Idle:            0,
	}
}

// QuoteIdentifier quotes an identifier for Elasticsearch
func (es *ElasticsearchDatabase) QuoteIdentifier(identifier string) string {
	// Elasticsearch uses backticks for identifiers in SQL queries
	return fmt.Sprintf("`%s`", strings.ReplaceAll(identifier, "`", "``"))
}

// GetDataTypeMappings returns Elasticsearch-specific data type mappings
func (es *ElasticsearchDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{
		"string":  "text",
		"keyword": "keyword",
		"int":     "integer",
		"int64":   "long",
		"float":   "float",
		"float64": "double",
		"bool":    "boolean",
		"time":    "date",
		"date":    "date",
		"json":    "object",
		"geo":     "geo_point",
		"binary":  "binary",
		"ip":      "ip",
		"text":    "text",
		"nested":  "nested",
		"object":  "object",
	}
}
