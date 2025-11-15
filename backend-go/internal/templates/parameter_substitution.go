package templates

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

// SubstituteParameters replaces {{param_name}} placeholders with actual values
// This function sanitizes inputs to prevent SQL injection
func SubstituteParameters(sqlTemplate string, params map[string]interface{}, paramDefs []turso.TemplateParameter) (string, error) {
	// Create a map of parameter definitions for easy lookup
	paramDefMap := make(map[string]turso.TemplateParameter)
	for _, def := range paramDefs {
		paramDefMap[def.Name] = def
	}

	// Find all parameter references
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)

	// Track which parameters have been substituted
	substituted := make(map[string]bool)

	result := re.ReplaceAllStringFunc(sqlTemplate, func(match string) string {
		// Extract parameter name from {{param_name}}
		paramName := strings.Trim(match, "{}")
		paramName = strings.TrimSpace(paramName)

		substituted[paramName] = true

		// Get parameter definition
		def, hasDef := paramDefMap[paramName]
		if !hasDef {
			// This shouldn't happen if validation passed, but handle it
			return fmt.Sprintf("{{ERROR:undefined_param_%s}}", paramName)
		}

		// Get parameter value
		value, hasValue := params[paramName]

		// Use default if not provided
		if !hasValue {
			if def.DefaultValue != nil {
				value = def.DefaultValue
			} else if def.Required {
				return fmt.Sprintf("{{ERROR:missing_required_param_%s}}", paramName)
			} else {
				// Optional parameter without default or value
				return "NULL"
			}
		}

		// Convert and sanitize value based on type
		sanitized, err := sanitizeParameter(value, def.Type, def.Validation)
		if err != nil {
			return fmt.Sprintf("{{ERROR:invalid_param_%s_%s}}", paramName, err.Error())
		}

		return sanitized
	})

	// Check for any errors in substitution
	if strings.Contains(result, "{{ERROR:") {
		errorRe := regexp.MustCompile(`\{\{ERROR:([^}]+)\}\}`)
		matches := errorRe.FindAllStringSubmatch(result, -1)
		if len(matches) > 0 {
			return "", fmt.Errorf("parameter substitution failed: %s", matches[0][1])
		}
	}

	// Validate all required parameters were used
	for _, def := range paramDefs {
		if def.Required && !substituted[def.Name] {
			return "", fmt.Errorf("required parameter not found in template: %s", def.Name)
		}
	}

	return result, nil
}

// sanitizeParameter converts and sanitizes a parameter value based on its type
func sanitizeParameter(value interface{}, paramType string, validation string) (string, error) {
	switch paramType {
	case "string":
		return sanitizeString(value, validation)
	case "number":
		return sanitizeNumber(value, validation)
	case "date":
		return sanitizeDate(value, validation)
	case "boolean":
		return sanitizeBoolean(value)
	default:
		return "", fmt.Errorf("unsupported parameter type: %s", paramType)
	}
}

// sanitizeString sanitizes string values
func sanitizeString(value interface{}, validation string) (string, error) {
	var str string

	switch v := value.(type) {
	case string:
		str = v
	case fmt.Stringer:
		str = v.String()
	default:
		str = fmt.Sprintf("%v", v)
	}

	// Apply validation regex if provided
	if validation != "" {
		matched, err := regexp.MatchString(validation, str)
		if err != nil {
			return "", fmt.Errorf("invalid validation pattern: %w", err)
		}
		if !matched {
			return "", fmt.Errorf("value does not match validation pattern")
		}
	}

	// Check for SQL injection attempts
	if containsSQLInjection(str) {
		return "", fmt.Errorf("value contains potentially unsafe characters")
	}

	// Escape single quotes by doubling them (SQL standard)
	escaped := strings.ReplaceAll(str, "'", "''")

	// Return as quoted string
	return fmt.Sprintf("'%s'", escaped), nil
}

// sanitizeNumber sanitizes numeric values
func sanitizeNumber(value interface{}, validation string) (string, error) {
	var num float64

	switch v := value.(type) {
	case int:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	case float32:
		num = float64(v)
	case float64:
		num = v
	case string:
		// Try to parse string as number
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", fmt.Errorf("invalid number format: %w", err)
		}
		num = parsed
	default:
		return "", fmt.Errorf("cannot convert to number")
	}

	// Apply validation if provided (e.g., ">=0", "<100")
	if validation != "" {
		if err := validateNumber(num, validation); err != nil {
			return "", err
		}
	}

	// Return as unquoted number
	return fmt.Sprintf("%v", num), nil
}

// sanitizeDate sanitizes date values
func sanitizeDate(value interface{}, validation string) (string, error) {
	var dateTime time.Time

	switch v := value.(type) {
	case time.Time:
		dateTime = v
	case string:
		// Try common date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02",
			"2006-01-02 15:04:05",
			"01/02/2006",
			"2006-01-02T15:04:05Z",
		}

		var err error
		parsed := false
		for _, format := range formats {
			dateTime, err = time.Parse(format, v)
			if err == nil {
				parsed = true
				break
			}
		}

		if !parsed {
			return "", fmt.Errorf("invalid date format")
		}
	case int64:
		// Unix timestamp
		dateTime = time.Unix(v, 0)
	default:
		return "", fmt.Errorf("cannot convert to date")
	}

	// Validate date is reasonable (not too far in past or future)
	now := time.Now()
	if dateTime.Before(now.AddDate(-100, 0, 0)) || dateTime.After(now.AddDate(100, 0, 0)) {
		return "", fmt.Errorf("date is out of reasonable range")
	}

	// Return in SQL-friendly format
	return fmt.Sprintf("'%s'", dateTime.Format("2006-01-02 15:04:05")), nil
}

// sanitizeBoolean sanitizes boolean values
func sanitizeBoolean(value interface{}) (string, error) {
	var boolVal bool

	switch v := value.(type) {
	case bool:
		boolVal = v
	case string:
		lower := strings.ToLower(v)
		switch lower {
		case "true", "1", "yes":
			boolVal = true
		case "false", "0", "no":
			boolVal = false
		default:
			return "", fmt.Errorf("invalid boolean value")
		}
	case int:
		boolVal = v != 0
	default:
		return "", fmt.Errorf("cannot convert to boolean")
	}

	// Return as SQL boolean (1 or 0 for compatibility)
	if boolVal {
		return "1", nil
	}
	return "0", nil
}

// containsSQLInjection checks for SQL injection patterns in string values
func containsSQLInjection(str string) bool {
	// Check for common SQL injection patterns
	dangerous := []string{
		"';",
		"\";",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
		"exec ",
		"execute ",
		"drop ",
		"delete ",
		"insert ",
		"update ",
		"union ",
		"select ",
		"into ",
		"0x", // Hex encoding
		"char(",
		"nchar(",
		"varchar(",
		"nvarchar(",
		"alter ",
		"create ",
		"truncate ",
		"script ",
		"javascript:",
		"<script",
	}

	strLower := strings.ToLower(str)

	for _, pattern := range dangerous {
		if strings.Contains(strLower, pattern) {
			return true
		}
	}

	// Check for multiple single quotes (potential escaping attempt)
	if strings.Count(str, "'") > 2 {
		return true
	}

	return false
}

// validateNumber validates a number against a validation rule
func validateNumber(num float64, validation string) error {
	// Parse validation rules like ">=0", "<100", "0-100"
	validation = strings.TrimSpace(validation)

	// Range validation: "0-100"
	if strings.Contains(validation, "-") && !strings.HasPrefix(validation, "<") && !strings.HasPrefix(validation, ">") {
		parts := strings.Split(validation, "-")
		if len(parts) == 2 {
			min, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			max, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				if num < min || num > max {
					return fmt.Errorf("number must be between %v and %v", min, max)
				}
				return nil
			}
		}
	}

	// Comparison validation: ">=0", "<100", etc.
	if strings.HasPrefix(validation, ">=") {
		min, err := strconv.ParseFloat(strings.TrimPrefix(validation, ">="), 64)
		if err == nil && num < min {
			return fmt.Errorf("number must be >= %v", min)
		}
	} else if strings.HasPrefix(validation, ">") {
		min, err := strconv.ParseFloat(strings.TrimPrefix(validation, ">"), 64)
		if err == nil && num <= min {
			return fmt.Errorf("number must be > %v", min)
		}
	} else if strings.HasPrefix(validation, "<=") {
		max, err := strconv.ParseFloat(strings.TrimPrefix(validation, "<="), 64)
		if err == nil && num > max {
			return fmt.Errorf("number must be <= %v", max)
		}
	} else if strings.HasPrefix(validation, "<") {
		max, err := strconv.ParseFloat(strings.TrimPrefix(validation, "<"), 64)
		if err == nil && num >= max {
			return fmt.Errorf("number must be < %v", max)
		}
	}

	return nil
}

// ValidateSQL performs basic SQL validation to ensure it's safe to execute
func ValidateSQL(sql string) error {
	// Check SQL is not empty
	if strings.TrimSpace(sql) == "" {
		return fmt.Errorf("SQL cannot be empty")
	}

	// Check for multiple statements (semicolon-separated)
	statements := strings.Split(sql, ";")
	if len(statements) > 2 { // Allow one statement + trailing semicolon
		return fmt.Errorf("multiple SQL statements not allowed")
	}

	// Check for dangerous keywords in certain contexts
	sqlLower := strings.ToLower(sql)

	// Don't allow certain system procedures
	systemProcs := []string{"xp_", "sp_password", "sp_addlogin"}
	for _, proc := range systemProcs {
		if strings.Contains(sqlLower, proc) {
			return fmt.Errorf("system procedures not allowed")
		}
	}

	return nil
}
