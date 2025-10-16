package main

import "testing"

func TestDeduplicateSequentialSQL(t *testing.T) {
	queryBlock := `-- Sample query
SELECT id, display_name
FROM accounts
WHERE revenue > 10000;`

	duplicated := queryBlock + "\n\n" + queryBlock

	result := deduplicateSequentialSQL(duplicated)
	if result != queryBlock {
		t.Fatalf("expected duplicated query to collapse to single block, got:\n%s", result)
	}

	// Ensure non-duplicated SQL is unchanged
	unique := queryBlock + "\n\n" + "/* summary */"
	if unchanged := deduplicateSequentialSQL(unique); unchanged != unique {
		t.Fatalf("expected unique query to remain unchanged, got:\n%s", unchanged)
	}
}

func TestSanitizeExplanation(t *testing.T) {
	sql := `SELECT id FROM accounts;`
	explanation := "Here is the query:\n```sql\n" + sql + "\n```\n" + sql

	cleaned := sanitizeExplanation(explanation, sql)
	expected := "Here is the query:"

	if cleaned != expected {
		t.Fatalf("expected sanitized explanation %q, got %q", expected, cleaned)
	}

	// Explanation with only code should return empty string
	codeOnly := "```sql\n" + sql + "\n```"
	if sanitized := sanitizeExplanation(codeOnly, sql); sanitized != "" {
		t.Fatalf("expected code-only explanation to be empty, got %q", sanitized)
	}
}
