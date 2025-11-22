package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUniversalSQLPrompt(t *testing.T) {
	tests := []struct {
		name            string
		dialect         SQLDialect
		expectedContent []string
	}{
		{
			name:    "PostgreSQL dialect",
			dialect: DialectPostgreSQL,
			expectedContent: []string{
				"PostgreSQL",
				"$1, $2, $3",
				"ILIKE",
				"INTERVAL",
				"jsonb",
			},
		},
		{
			name:    "MySQL dialect",
			dialect: DialectMySQL,
			expectedContent: []string{
				"MySQL",
				"backticks",
				"CONCAT()",
				"AUTO_INCREMENT",
				"InnoDB",
			},
		},
		{
			name:    "SQLite dialect",
			dialect: DialectSQLite,
			expectedContent: []string{
				"SQLite",
				"datetime('now')",
				"AUTOINCREMENT",
				"Dynamic typing",
			},
		},
		{
			name:    "SQL Server dialect",
			dialect: DialectMSSQL,
			expectedContent: []string{
				"SQL Server",
				"square brackets",
				"GETDATE()",
				"Identity columns",
			},
		},
		{
			name:    "Oracle dialect",
			dialect: DialectOracle,
			expectedContent: []string{
				"Oracle",
				"SYSDATE",
				"VARCHAR2",
				"DUAL",
			},
		},
		{
			name:    "Generic SQL",
			dialect: DialectGeneric,
			expectedContent: []string{
				"Generic SQL",
				"standard SQL",
				"ANSI",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := GetUniversalSQLPrompt(tt.dialect)

			// Check that prompt is substantial
			assert.Greater(t, len(prompt), 1000, "Prompt should be comprehensive")

			// Check for expected dialect-specific content
			for _, expected := range tt.expectedContent {
				assert.Contains(t, prompt, expected,
					"Prompt should contain dialect-specific information about: %s", expected)
			}

			// Check for common sections
			commonSections := []string{
				"Core Principles",
				"Generation Guidelines",
				"Response Format",
				"Common Patterns",
				"Error Prevention",
			}

			for _, section := range commonSections {
				assert.Contains(t, prompt, section,
					"Prompt should contain section: %s", section)
			}

			// Check for confidence scoring guidance
			assert.Contains(t, prompt, "Confidence Scoring")
			assert.Contains(t, prompt, "0.95-1.0")

			// Check for JSON format example
			assert.Contains(t, prompt, `"query":`)
			assert.Contains(t, prompt, `"explanation":`)
			assert.Contains(t, prompt, `"confidence":`)
		})
	}
}

func TestGetSQLFixPrompt(t *testing.T) {
	tests := []struct {
		name            string
		dialect         SQLDialect
		errorCategory   ErrorCategory
		expectedContent []string
	}{
		{
			name:          "PostgreSQL syntax error",
			dialect:       DialectPostgreSQL,
			errorCategory: ErrorCategorySyntax,
			expectedContent: []string{
				"PostgreSQL",
				"Syntax Error Guidance",
				"Keyword spelling",
				"Comma placement",
			},
		},
		{
			name:          "MySQL reference error",
			dialect:       DialectMySQL,
			errorCategory: ErrorCategoryReference,
			expectedContent: []string{
				"MySQL",
				"Reference Error Guidance",
				"Table name spelling",
				"Ambiguous column",
			},
		},
		{
			name:          "SQLite type error",
			dialect:       DialectSQLite,
			errorCategory: ErrorCategoryType,
			expectedContent: []string{
				"SQLite",
				"Type Error Guidance",
				"Data type compatibility",
				"Type casting",
			},
		},
		{
			name:          "Generic constraint error",
			dialect:       DialectGeneric,
			errorCategory: ErrorCategoryConstraint,
			expectedContent: []string{
				"Constraint Error Guidance",
				"Primary key",
				"Foreign key",
				"NOT NULL",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := GetSQLFixPrompt(tt.dialect, tt.errorCategory)

			// Check that prompt is substantial
			assert.Greater(t, len(prompt), 1000, "Fix prompt should be comprehensive")

			// Check for expected content
			for _, expected := range tt.expectedContent {
				assert.Contains(t, prompt, expected,
					"Fix prompt should contain: %s", expected)
			}

			// Check for common sections
			commonSections := []string{
				"Debugging Process",
				"Common Error Types",
				"Fix Guidelines",
				"Response Format",
			}

			for _, section := range commonSections {
				assert.Contains(t, prompt, section,
					"Fix prompt should contain section: %s", section)
			}

			// Check for error category specific guidance
			assert.Contains(t, prompt, "Error Category:")

			// Check for fix examples
			assert.Contains(t, prompt, "BEFORE:")
			assert.Contains(t, prompt, "AFTER:")
		})
	}
}

func TestDetectErrorCategory(t *testing.T) {
	tests := []struct {
		errorMessage string
		expected     ErrorCategory
	}{
		{
			errorMessage: "syntax error at or near \"SELCT\"",
			expected:     ErrorCategorySyntax,
		},
		{
			errorMessage: "column \"user_id\" does not exist",
			expected:     ErrorCategoryReference,
		},
		{
			errorMessage: "table \"orders\" not found",
			expected:     ErrorCategoryReference,
		},
		{
			errorMessage: "cannot cast type integer to text",
			expected:     ErrorCategoryType,
		},
		{
			errorMessage: "permission denied for table users",
			expected:     ErrorCategoryPermission,
		},
		{
			errorMessage: "unique constraint violation",
			expected:     ErrorCategoryConstraint,
		},
		{
			errorMessage: "foreign key constraint failed",
			expected:     ErrorCategoryConstraint,
		},
		{
			errorMessage: "query execution timeout",
			expected:     ErrorCategoryPerformance,
		},
		{
			errorMessage: "some unknown database error",
			expected:     ErrorCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.errorMessage, func(t *testing.T) {
			category := DetectErrorCategory(tt.errorMessage)
			assert.Equal(t, tt.expected, category,
				"Error message should be categorized correctly")
		})
	}
}

func TestDetectDialect(t *testing.T) {
	tests := []struct {
		connectionType string
		expected       SQLDialect
	}{
		{"postgres", DialectPostgreSQL},
		{"postgresql", DialectPostgreSQL},
		{"pg", DialectPostgreSQL},
		{"mysql", DialectMySQL},
		{"mariadb", DialectMySQL},
		{"sqlite", DialectSQLite},
		{"sqlite3", DialectSQLite},
		{"mssql", DialectMSSQL},
		{"sqlserver", DialectMSSQL},
		{"oracle", DialectOracle},
		{"unknown_db", DialectGeneric},
	}

	for _, tt := range tests {
		t.Run(tt.connectionType, func(t *testing.T) {
			dialect := DetectDialect(tt.connectionType)
			assert.Equal(t, tt.expected, dialect,
				"Connection type should map to correct dialect")
		})
	}
}

func TestPromptTemplate(t *testing.T) {
	t.Run("variable replacement", func(t *testing.T) {
		template := &PromptTemplate{
			UserPrefix: "Hello {{name}}, your role is {{role}}",
			Variables: map[string]string{
				"role": "admin",
			},
		}

		context := map[string]interface{}{
			"name": "Alice",
		}

		result := template.BuildPrompt("Main content", context)

		assert.Contains(t, result, "Hello Alice")
		assert.Contains(t, result, "your role is admin")
		assert.Contains(t, result, "Main content")
	})

	t.Run("build complete prompt", func(t *testing.T) {
		template := &PromptTemplate{
			UserPrefix: "## Context\n{{context}}",
			UserSuffix: "Please respond in JSON.",
			Variables:  map[string]string{},
		}

		context := map[string]interface{}{
			"context": "Database schema information",
		}

		result := template.BuildPrompt("Generate SQL", context)

		assert.Contains(t, result, "## Context")
		assert.Contains(t, result, "Database schema information")
		assert.Contains(t, result, "Generate SQL")
		assert.Contains(t, result, "Please respond in JSON")
	})
}

func TestGetSQLGenerationTemplate(t *testing.T) {
	template := GetSQLGenerationTemplate(DialectPostgreSQL)

	assert.NotNil(t, template)
	assert.NotEmpty(t, template.System)
	assert.Contains(t, template.System, "PostgreSQL")
	assert.Greater(t, template.MaxTokens, 1000)
	assert.Less(t, template.Temperature, 1.0)
}

func TestGetSQLFixTemplate(t *testing.T) {
	template := GetSQLFixTemplate(DialectMySQL, ErrorCategorySyntax)

	assert.NotNil(t, template)
	assert.NotEmpty(t, template.System)
	assert.Contains(t, template.System, "MySQL")
	assert.Contains(t, template.System, "Syntax")
	assert.Greater(t, template.MaxTokens, 1000)
}

func TestParseSQLResponse(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantQuery string
		wantConf  float64
		wantErr   bool
	}{
		{
			name: "clean JSON response",
			content: `{
				"query": "SELECT * FROM users",
				"explanation": "Fetches all users",
				"confidence": 0.95,
				"suggestions": ["Add WHERE clause"],
				"warnings": []
			}`,
			wantQuery: "SELECT * FROM users",
			wantConf:  0.95,
			wantErr:   false,
		},
		{
			name: "JSON in markdown code block",
			content: "```json\n{\n" +
				`"query": "SELECT id FROM users",` + "\n" +
				`"explanation": "Gets user IDs",` + "\n" +
				`"confidence": 0.90` + "\n" +
				"}\n```",
			wantQuery: "SELECT id FROM users",
			wantConf:  0.90,
			wantErr:   false,
		},
		{
			name: "JSON with explanation before",
			content: "Here's the query:\n\n{\n" +
				`"query": "SELECT name FROM users WHERE active = true",` + "\n" +
				`"explanation": "Active users",` + "\n" +
				`"confidence": 0.85` + "\n" +
				"}",
			wantQuery: "SELECT name FROM users WHERE active = true",
			wantConf:  0.85,
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			content:   "This is not JSON at all",
			wantQuery: "",
			wantConf:  0,
			wantErr:   true,
		},
		{
			name:      "JSON missing query field",
			content:   `{"explanation": "Some text", "confidence": 0.5}`,
			wantQuery: "",
			wantConf:  0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := ParseSQLResponse(tt.content, ProviderOpenAI, "gpt-4")

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, response)

			assert.Equal(t, tt.wantQuery, response.Query)
			assert.Equal(t, tt.wantConf, response.Confidence)
			assert.Equal(t, ProviderOpenAI, response.Provider)
			assert.Equal(t, "gpt-4", response.Model)
			assert.NotNil(t, response.Suggestions)
			assert.NotNil(t, response.Warnings)
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "JSON in markdown block",
			text:     "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in plain code block",
			text:     "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "raw JSON",
			text:     `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with surrounding text",
			text:     "Here is the result: {\"key\": \"value\"} that's it",
			expected: `{"key": "value"}`,
		},
		{
			name:     "no JSON",
			text:     "This text contains no JSON",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.text)
			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.JSONEq(t, tt.expected, result)
			}
		})
	}
}

func TestGetRecommendedTokenBudget(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"claude-3-opus-20240229", 200000},
		{"claude-3-sonnet-20240229", 200000},
		{"claude-3-haiku-20240307", 200000},
		{"gpt-4-turbo-preview", 128000},
		{"gpt-4-32k", 32000},
		{"gpt-4", 8192},
		{"gpt-3.5-turbo-16k", 16384},
		{"gpt-3.5-turbo", 4096},
		{"unknown-model", 4096},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			budget := GetRecommendedTokenBudget(tt.model)
			assert.Equal(t, tt.expected, budget)
		})
	}
}

func TestDetectDialectFromConnection(t *testing.T) {
	tests := []struct {
		name     string
		connType string
		metadata map[string]string
		expected SQLDialect
	}{
		{
			name:     "PostgreSQL from connection type",
			connType: "postgresql",
			expected: DialectPostgreSQL,
		},
		{
			name:     "MySQL from metadata",
			connType: "unknown",
			metadata: map[string]string{
				"database_type": "mysql",
			},
			expected: DialectMySQL,
		},
		{
			name:     "SQLite from driver",
			connType: "unknown",
			metadata: map[string]string{
				"driver": "sqlite3",
			},
			expected: DialectSQLite,
		},
		{
			name:     "Generic fallback",
			connType: "unknown",
			metadata: map[string]string{},
			expected: DialectGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialect := DetectDialectFromConnection(tt.connType, tt.metadata)
			assert.Equal(t, tt.expected, dialect)
		})
	}
}

func TestPromptCompleteness(t *testing.T) {
	t.Run("SQL generation prompt covers all dialects", func(t *testing.T) {
		dialects := []SQLDialect{
			DialectPostgreSQL,
			DialectMySQL,
			DialectSQLite,
			DialectMSSQL,
			DialectOracle,
			DialectGeneric,
		}

		for _, dialect := range dialects {
			prompt := GetUniversalSQLPrompt(dialect)
			assert.Greater(t, len(prompt), 100,
				"Prompt for %s should be substantial", dialect)
		}
	})

	t.Run("SQL fix prompt covers all error categories", func(t *testing.T) {
		categories := []ErrorCategory{
			ErrorCategorySyntax,
			ErrorCategoryReference,
			ErrorCategoryType,
			ErrorCategoryPermission,
			ErrorCategoryConstraint,
			ErrorCategoryPerformance,
			ErrorCategoryUnknown,
		}

		for _, category := range categories {
			prompt := GetSQLFixPrompt(DialectPostgreSQL, category)
			assert.Greater(t, len(prompt), 100,
				"Fix prompt for %s should be substantial", category)

			// Should mention the error category
			assert.True(t,
				strings.Contains(prompt, string(category)) ||
					strings.Contains(prompt, "Error Guidance"),
				"Fix prompt should reference error category: %s", category)
		}
	})
}

func BenchmarkGetUniversalSQLPrompt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetUniversalSQLPrompt(DialectPostgreSQL)
	}
}

func BenchmarkDetectErrorCategory(b *testing.B) {
	errorMsg := "ERROR: column \"user_id\" does not exist at character 45"
	for i := 0; i < b.N; i++ {
		_ = DetectErrorCategory(errorMsg)
	}
}

func BenchmarkParseSQLResponse(b *testing.B) {
	content := `{
		"query": "SELECT id, name FROM users WHERE active = true",
		"explanation": "Retrieves active users",
		"confidence": 0.95,
		"suggestions": ["Add index on active column"],
		"warnings": []
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSQLResponse(content, ProviderOpenAI, "gpt-4")
	}
}
