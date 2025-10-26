package database

import (
	"regexp"
	"strings"
)

var quotedIdentifierPattern = regexp.MustCompile(`^"(.*)"$`)

// parseSimpleSelect inspects a query and attempts to determine whether it targets a single table
// that can be updated safely. It returns the schema, table, optional reason, and a boolean flag
// indicating whether editing is permitted.
func parseSimpleSelect(query string) (string, string, string, bool) {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return "", "", "Empty query", false
	}

	upper := strings.ToUpper(normalized)
	if strings.HasPrefix(upper, "WITH ") {
		return "", "", "Common table expressions are read-only", false
	}

	disallowed := []struct {
		token  string
		reason string
	}{
		{" UNION ", "Query contains UNION operations"},
		{" INTERSECT ", "Query contains INTERSECT operations"},
		{" EXCEPT ", "Query contains EXCEPT operations"},
		{" GROUP BY", "Query contains GROUP BY"},
		{" HAVING ", "Query contains HAVING"},
		{" DISTINCT ", "Query uses DISTINCT"},
		{" RETURNING ", "Query contains RETURNING clause"},
		{" FOR UPDATE", "Query contains FOR UPDATE"},
	}

	for _, item := range disallowed {
		if strings.Contains(upper, item.token) {
			return "", "", item.reason, false
		}
	}

	if strings.Contains(upper, " JOIN ") {
		return parseJoinQuery(normalized)
	}

	fromClause := extractFromClause(normalized)
	if fromClause == "" {
		return "", "", "Unable to determine target table", false
	}

	trimmedFrom := strings.TrimSpace(fromClause)
	if strings.HasPrefix(trimmedFrom, "(") {
		return "", "", "Query selects from a subquery", false
	}

	if strings.Contains(strings.ToUpper(trimmedFrom), " JOIN ") {
		return parseJoinQuery(normalized)
	}

	if strings.Contains(trimmedFrom, ",") {
		return "", "", "Query targets multiple tables", false
	}

	parts := strings.Fields(trimmedFrom)
	if len(parts) == 0 {
		return "", "", "Unable to determine target table", false
	}

	tableToken := parts[0]
	if strings.EqualFold(tableToken, "ONLY") {
		if len(parts) < 2 {
			return "", "", "Unable to determine target table", false
		}
		tableToken = parts[1]
	}

	tableToken = strings.Trim(tableToken, ";")
	schema, table := splitIdentifier(tableToken)

	return schema, table, "", true
}

// parseJoinQuery handles JOIN queries for editing support, currently restricting editing to the first table.
func parseJoinQuery(query string) (string, string, string, bool) {
	upper := strings.ToUpper(query)
	fromIndex := strings.Index(upper, " FROM ")
	if fromIndex == -1 {
		return "", "", "No FROM clause found", false
	}

	fromClause := query[fromIndex+6:]
	joinIndex := strings.Index(strings.ToUpper(fromClause), " JOIN ")
	if joinIndex == -1 {
		return "", "", "No JOIN found in FROM clause", false
	}

	mainTableClause := strings.TrimSpace(fromClause[:joinIndex])
	parts := strings.Fields(mainTableClause)
	if len(parts) == 0 {
		return "", "", "Unable to determine main table", false
	}

	tableToken := parts[0]
	tableToken = strings.Trim(tableToken, ";")
	schema, table := splitIdentifier(tableToken)

	return schema, table, "", true
}

func extractFromClause(query string) string {
	upper := strings.ToUpper(query)
	index := strings.Index(upper, "FROM")
	if index == -1 {
		return ""
	}

	rest := query[index+4:]
	upperRest := strings.ToUpper(rest)
	end := len(rest)

	keywords := []string{
		" WHERE ",
		" GROUP ",
		" ORDER ",
		" LIMIT ",
		" OFFSET ",
		" RETURNING ",
		" FOR ",
		" UNION ",
		" INTERSECT ",
		" EXCEPT ",
	}

	for _, keyword := range keywords {
		if idx := strings.Index(upperRest, keyword); idx != -1 && idx < end {
			end = idx
		}
	}

	return strings.TrimSpace(rest[:end])
}

func splitIdentifier(identifier string) (string, string) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return "", ""
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) == 1 {
		return "", unquoteIdentifierPart(parts[0])
	}

	schemaPart := strings.Join(parts[:len(parts)-1], ".")
	tablePart := parts[len(parts)-1]
	return unquoteIdentifierPart(schemaPart), unquoteIdentifierPart(tablePart)
}

func unquoteIdentifierPart(value string) string {
	value = strings.TrimSpace(value)
	if quotedIdentifierPattern.MatchString(value) {
		matches := quotedIdentifierPattern.FindStringSubmatch(value)
		if len(matches) == 2 {
			return strings.ReplaceAll(matches[1], `""`, `"`)
		}
	}
	return strings.ToLower(value)
}
