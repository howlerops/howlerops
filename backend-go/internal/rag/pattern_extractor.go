package rag

// PatternExtractor turns raw queries into templates with placeholders.
type PatternExtractor struct{}

func NewPatternExtractor() *PatternExtractor { return &PatternExtractor{} }

func (pe *PatternExtractor) Extract(query string) string { return query }


