package database

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestElasticsearchDatabase_GetDatabaseType(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := ConnectionConfig{
		Type:     Elasticsearch,
		Host:     "localhost",
		Port:     9200,
		Database: "test",
	}

	// Create without connecting (we don't want to require a running ES instance)
	es := &ElasticsearchDatabase{
		config: config,
		logger: logger,
	}

	if es.GetDatabaseType() != Elasticsearch {
		t.Errorf("Expected database type %s, got %s", Elasticsearch, es.GetDatabaseType())
	}
}

func TestElasticsearchDatabase_QuoteIdentifier(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	es := &ElasticsearchDatabase{
		logger: logger,
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"users", "`users`"},
		{"my_index", "`my_index`"},
		{"index`with`backticks", "`index``with``backticks`"},
	}

	for _, test := range tests {
		result := es.QuoteIdentifier(test.input)
		if result != test.expected {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestElasticsearchDatabase_GetDataTypeMappings(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	es := &ElasticsearchDatabase{
		logger: logger,
	}

	mappings := es.GetDataTypeMappings()

	expectedMappings := map[string]string{
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

	for key, expected := range expectedMappings {
		if value, ok := mappings[key]; !ok || value != expected {
			t.Errorf("DataTypeMapping[%q] = %q, want %q", key, value, expected)
		}
	}
}

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1kb", 1024},
		{"1.5kb", 1536},
		{"1mb", 1024 * 1024},
		{"2.5mb", 2621440},
		{"1gb", 1024 * 1024 * 1024},
		{"1tb", 1024 * 1024 * 1024 * 1024},
		{"512b", 512},
		{"0kb", 0},
	}

	for _, test := range tests {
		result := parseSizeString(test.input)
		if result != test.expected {
			t.Errorf("parseSizeString(%q) = %d, want %d", test.input, result, test.expected)
		}
	}
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		username string
		password string
	}{
		{"elastic", "password123"},
		{"admin", "secret"},
		{"user@domain.com", "p@ssw0rd!"},
	}

	for _, test := range tests {
		result := basicAuth(test.username, test.password)
		// Just verify it returns a non-empty string
		if result == "" {
			t.Errorf("basicAuth(%q, %q) returned empty string", test.username, test.password)
		}
		// Verify it doesn't contain the plaintext password
		if result == test.password {
			t.Errorf("basicAuth returned plaintext password")
		}
	}
}

func TestElasticsearchDatabase_ComputeEditableMetadata(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	es := &ElasticsearchDatabase{
		logger: logger,
	}

	columns := []string{"id", "name", "email"}
	metadata, err := es.ComputeEditableMetadata(context.TODO(), "SELECT * FROM users", columns)
	if err != nil {
		t.Fatalf("ComputeEditableMetadata failed: %v", err)
	}

	if metadata.Enabled {
		t.Error("Expected Enabled to be false for Elasticsearch")
	}

	if metadata.Reason == "" {
		t.Error("Expected a reason why editing is not enabled")
	}

	if len(metadata.Columns) != len(columns) {
		t.Errorf("Expected %d columns, got %d", len(columns), len(metadata.Columns))
	}
}
