package templates

import (
	"testing"
	"time"

	"github.com/sql-studio/backend-go/pkg/storage/turso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstituteParameters(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		params      map[string]interface{}
		paramDefs   []turso.TemplateParameter
		expected    string
		expectError bool
	}{
		{
			name:     "Simple string substitution",
			template: "SELECT * FROM users WHERE email = {{email}}",
			params: map[string]interface{}{
				"email": "test@example.com",
			},
			paramDefs: []turso.TemplateParameter{
				{Name: "email", Type: "string", Required: true},
			},
			expected:    "SELECT * FROM users WHERE email = 'test@example.com'",
			expectError: false,
		},
		{
			name:     "Multiple parameters",
			template: "SELECT * FROM orders WHERE user_id = {{user_id}} AND status = {{status}}",
			params: map[string]interface{}{
				"user_id": 123,
				"status":  "active",
			},
			paramDefs: []turso.TemplateParameter{
				{Name: "user_id", Type: "number", Required: true},
				{Name: "status", Type: "string", Required: true},
			},
			expected:    "SELECT * FROM orders WHERE user_id = 123 AND status = 'active'",
			expectError: false,
		},
		{
			name:     "Date parameter",
			template: "SELECT * FROM events WHERE created_at > {{start_date}}",
			params: map[string]interface{}{
				"start_date": "2024-01-01",
			},
			paramDefs: []turso.TemplateParameter{
				{Name: "start_date", Type: "date", Required: true},
			},
			expected:    "SELECT * FROM events WHERE created_at > '2024-01-01 00:00:00'",
			expectError: false,
		},
		{
			name:     "Boolean parameter",
			template: "SELECT * FROM users WHERE active = {{active}}",
			params: map[string]interface{}{
				"active": true,
			},
			paramDefs: []turso.TemplateParameter{
				{Name: "active", Type: "boolean", Required: true},
			},
			expected:    "SELECT * FROM users WHERE active = 1",
			expectError: false,
		},
		{
			name:     "Default value used",
			template: "SELECT * FROM users WHERE status = {{status}}",
			params:   map[string]interface{}{},
			paramDefs: []turso.TemplateParameter{
				{Name: "status", Type: "string", Required: false, DefaultValue: "active"},
			},
			expected:    "SELECT * FROM users WHERE status = 'active'",
			expectError: false,
		},
		{
			name:     "Missing required parameter",
			template: "SELECT * FROM users WHERE email = {{email}}",
			params:   map[string]interface{}{},
			paramDefs: []turso.TemplateParameter{
				{Name: "email", Type: "string", Required: true},
			},
			expectError: true,
		},
		{
			name:     "SQL injection attempt - single quote",
			template: "SELECT * FROM users WHERE email = {{email}}",
			params: map[string]interface{}{
				"email": "test'; DROP TABLE users; --",
			},
			paramDefs: []turso.TemplateParameter{
				{Name: "email", Type: "string", Required: true},
			},
			expectError: true,
		},
		{
			name:     "SQL injection attempt - union",
			template: "SELECT * FROM users WHERE id = {{user_id}}",
			params: map[string]interface{}{
				"user_id": "1 UNION SELECT * FROM passwords",
			},
			paramDefs: []turso.TemplateParameter{
				{Name: "user_id", Type: "string", Required: true},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SubstituteParameters(tt.template, tt.params, tt.paramDefs)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		validation  string
		expected    string
		expectError bool
	}{
		{
			name:        "Simple string",
			value:       "hello",
			validation:  "",
			expected:    "'hello'",
			expectError: false,
		},
		{
			name:        "String with single quote",
			value:       "O'Brien",
			validation:  "",
			expected:    "'O''Brien'",
			expectError: false,
		},
		{
			name:        "Email validation",
			value:       "test@example.com",
			validation:  "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
			expected:    "'test@example.com'",
			expectError: false,
		},
		{
			name:        "Invalid email",
			value:       "not-an-email",
			validation:  "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
			expectError: true,
		},
		{
			name:        "SQL injection - DROP TABLE",
			value:       "'; DROP TABLE users; --",
			validation:  "",
			expectError: true,
		},
		{
			name:        "SQL injection - UNION SELECT",
			value:       "' UNION SELECT * FROM passwords --",
			validation:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeString(tt.value, tt.validation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeNumber(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		validation  string
		expected    string
		expectError bool
	}{
		{
			name:        "Integer",
			value:       123,
			validation:  "",
			expected:    "123",
			expectError: false,
		},
		{
			name:        "Float",
			value:       123.45,
			validation:  "",
			expected:    "123.45",
			expectError: false,
		},
		{
			name:        "String number",
			value:       "456",
			validation:  "",
			expected:    "456",
			expectError: false,
		},
		{
			name:        "Positive validation",
			value:       10,
			validation:  ">=0",
			expected:    "10",
			expectError: false,
		},
		{
			name:        "Negative fails validation",
			value:       -5,
			validation:  ">=0",
			expectError: true,
		},
		{
			name:        "Range validation",
			value:       50,
			validation:  "0-100",
			expected:    "50",
			expectError: false,
		},
		{
			name:        "Out of range",
			value:       150,
			validation:  "0-100",
			expectError: true,
		},
		{
			name:        "Invalid string",
			value:       "not-a-number",
			validation:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeNumber(tt.value, tt.validation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeDate(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		validation  string
		expectError bool
	}{
		{
			name:        "ISO date string",
			value:       "2024-01-15",
			validation:  "",
			expectError: false,
		},
		{
			name:        "RFC3339 date",
			value:       "2024-01-15T10:30:00Z",
			validation:  "",
			expectError: false,
		},
		{
			name:        "Unix timestamp",
			value:       int64(1705315800),
			validation:  "",
			expectError: false,
		},
		{
			name:        "time.Time",
			value:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			validation:  "",
			expectError: false,
		},
		{
			name:        "Invalid date format",
			value:       "not-a-date",
			validation:  "",
			expectError: true,
		},
		{
			name:        "Date too far in past",
			value:       "1800-01-01",
			validation:  "",
			expectError: true,
		},
		{
			name:        "Date too far in future",
			value:       "2200-01-01",
			validation:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sanitizeDate(tt.value, tt.validation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeBoolean(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "Boolean true",
			value:       true,
			expected:    "1",
			expectError: false,
		},
		{
			name:        "Boolean false",
			value:       false,
			expected:    "0",
			expectError: false,
		},
		{
			name:        "String 'true'",
			value:       "true",
			expected:    "1",
			expectError: false,
		},
		{
			name:        "String 'false'",
			value:       "false",
			expected:    "0",
			expectError: false,
		},
		{
			name:        "String '1'",
			value:       "1",
			expected:    "1",
			expectError: false,
		},
		{
			name:        "String '0'",
			value:       "0",
			expected:    "0",
			expectError: false,
		},
		{
			name:        "Integer 1",
			value:       1,
			expected:    "1",
			expectError: false,
		},
		{
			name:        "Integer 0",
			value:       0,
			expected:    "0",
			expectError: false,
		},
		{
			name:        "Invalid boolean",
			value:       "maybe",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeBoolean(tt.value)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestContainsSQLInjection(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "Clean string",
			value:    "hello world",
			expected: false,
		},
		{
			name:     "Email address",
			value:    "test@example.com",
			expected: false,
		},
		{
			name:     "DROP TABLE",
			value:    "'; DROP TABLE users; --",
			expected: true,
		},
		{
			name:     "UNION SELECT",
			value:    "' UNION SELECT * FROM passwords",
			expected: true,
		},
		{
			name:     "Comment injection",
			value:    "test-- comment",
			expected: true,
		},
		{
			name:     "xp_cmdshell",
			value:    "'; EXEC xp_cmdshell('dir'); --",
			expected: true,
		},
		{
			name:     "Hex encoding",
			value:    "0x7365637265743a20706173733132333435",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSQLInjection(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNumber(t *testing.T) {
	tests := []struct {
		name        string
		num         float64
		validation  string
		expectError bool
	}{
		{
			name:        "No validation",
			num:         42,
			validation:  "",
			expectError: false,
		},
		{
			name:        "Greater than or equal",
			num:         10,
			validation:  ">=0",
			expectError: false,
		},
		{
			name:        "Less than",
			num:         50,
			validation:  "<100",
			expectError: false,
		},
		{
			name:        "Range valid",
			num:         50,
			validation:  "0-100",
			expectError: false,
		},
		{
			name:        "Range invalid - below",
			num:         -5,
			validation:  "0-100",
			expectError: true,
		},
		{
			name:        "Range invalid - above",
			num:         150,
			validation:  "0-100",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNumber(tt.num, tt.validation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractParameterReferences(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "Single parameter",
			template: "SELECT * FROM users WHERE id = {{user_id}}",
			expected: []string{"user_id"},
		},
		{
			name:     "Multiple parameters",
			template: "SELECT * FROM orders WHERE user_id = {{user_id}} AND status = {{status}}",
			expected: []string{"user_id", "status"},
		},
		{
			name:     "Duplicate parameters",
			template: "SELECT * FROM logs WHERE id = {{id}} OR parent_id = {{id}}",
			expected: []string{"id"},
		},
		{
			name:     "No parameters",
			template: "SELECT * FROM users",
			expected: []string{},
		},
		{
			name:     "Parameters with underscores",
			template: "SELECT * FROM events WHERE start_date = {{start_date}} AND end_date = {{end_date}}",
			expected: []string{"start_date", "end_date"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractParameterReferences(tt.template)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
