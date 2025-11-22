package rag

import (
	"fmt"
	"math"
)

// TokenBudget manages the allocation of tokens across different context components
type TokenBudget struct {
	TotalTokens      int     // Total available tokens in context window
	SystemPrompt     int     // Tokens allocated to system prompt
	UserQuery        int     // Tokens allocated to user query
	OutputBuffer     int     // Reserved tokens for model output
	ContextAvailable int     // Remaining tokens for RAG context
	SafetyMargin     float64 // Safety margin percentage (0.0-1.0)
}

// BudgetAllocation represents token allocation across components
type BudgetAllocation struct {
	SystemPrompt int                     // Tokens for system prompt
	UserQuery    int                     // Tokens for user query
	Schema       int                     // Tokens for schema context
	Examples     int                     // Tokens for example queries
	Business     int                     // Tokens for business rules
	Performance  int                     // Tokens for performance hints
	OutputBuffer int                     // Reserved for output
	Total        int                     // Total allocated tokens
	Details      map[string]TokenDetails // Detailed breakdown
}

// TokenDetails provides detailed token information for a component
type TokenDetails struct {
	Allocated int     // Tokens allocated to this component
	Used      int     // Tokens actually used
	Priority  int     // Priority level (1-10, higher = more important)
	Ratio     float64 // Ratio of total context budget
}

// DefaultTokenBudget creates a token budget with sensible defaults
func DefaultTokenBudget(totalTokens int) *TokenBudget {
	// Reserve tokens for system prompt (typically 1000-2000 tokens)
	systemPrompt := estimateSystemPromptTokens()

	// Reserve tokens for output (25% of total, capped at 2000)
	outputBuffer := int(math.Min(float64(totalTokens)/4.0, 2000))

	// User query is typically small (100-500 tokens)
	userQuery := 500

	// Safety margin to avoid hitting exact token limits
	safetyMargin := 0.05 // 5% safety margin

	// Calculate available context tokens
	contextAvailable := totalTokens - systemPrompt - outputBuffer - userQuery
	contextAvailable = int(float64(contextAvailable) * (1.0 - safetyMargin))

	if contextAvailable < 0 {
		contextAvailable = 0
	}

	return &TokenBudget{
		TotalTokens:      totalTokens,
		SystemPrompt:     systemPrompt,
		UserQuery:        userQuery,
		OutputBuffer:     outputBuffer,
		ContextAvailable: contextAvailable,
		SafetyMargin:     safetyMargin,
	}
}

// AllocateContextBudget allocates available context tokens across RAG components
func (tb *TokenBudget) AllocateContextBudget(priorities map[string]int) *BudgetAllocation {
	// Default priorities if not specified
	if priorities == nil {
		priorities = map[string]int{
			"schema":      10, // Highest priority - schema is critical
			"examples":    7,  // High priority - examples show patterns
			"business":    5,  // Medium priority - business rules
			"performance": 3,  // Lower priority - nice to have
		}
	}

	// Calculate total priority weight
	totalPriority := 0
	for _, priority := range priorities {
		totalPriority += priority
	}

	// Allocate tokens proportionally based on priority
	allocation := &BudgetAllocation{
		SystemPrompt: tb.SystemPrompt,
		UserQuery:    tb.UserQuery,
		OutputBuffer: tb.OutputBuffer,
		Details:      make(map[string]TokenDetails),
	}

	if totalPriority > 0 {
		for component, priority := range priorities {
			ratio := float64(priority) / float64(totalPriority)
			tokens := int(float64(tb.ContextAvailable) * ratio)

			details := TokenDetails{
				Allocated: tokens,
				Priority:  priority,
				Ratio:     ratio,
			}

			allocation.Details[component] = details

			// Set component-specific allocations
			switch component {
			case "schema":
				allocation.Schema = tokens
			case "examples":
				allocation.Examples = tokens
			case "business":
				allocation.Business = tokens
			case "performance":
				allocation.Performance = tokens
			}
		}
	}

	allocation.Total = allocation.SystemPrompt + allocation.UserQuery +
		allocation.Schema + allocation.Examples +
		allocation.Business + allocation.Performance +
		allocation.OutputBuffer

	return allocation
}

// AdjustForActualUsage adjusts allocation based on actual token usage
func (ba *BudgetAllocation) AdjustForActualUsage(component string, tokensUsed int) {
	if details, exists := ba.Details[component]; exists {
		details.Used = tokensUsed
		ba.Details[component] = details

		// If we used fewer tokens than allocated, redistribute the surplus
		if tokensUsed < details.Allocated {
			surplus := details.Allocated - tokensUsed
			ba.redistributeSurplus(component, surplus)
		}
	}
}

// redistributeSurplus redistributes unused tokens to other components
func (ba *BudgetAllocation) redistributeSurplus(sourceComponent string, surplus int) {
	// Find components that haven't been filled yet, ordered by priority
	type priorityComponent struct {
		name     string
		priority int
		deficit  int
	}

	var needMore []priorityComponent
	for name, details := range ba.Details {
		if name != sourceComponent && details.Used < details.Allocated {
			deficit := details.Allocated - details.Used
			needMore = append(needMore, priorityComponent{
				name:     name,
				priority: details.Priority,
				deficit:  deficit,
			})
		}
	}

	// Sort by priority (higher first)
	for i := 0; i < len(needMore); i++ {
		for j := i + 1; j < len(needMore); j++ {
			if needMore[j].priority > needMore[i].priority {
				needMore[i], needMore[j] = needMore[j], needMore[i]
			}
		}
	}

	// Redistribute surplus to higher priority components first
	remaining := surplus
	for _, pc := range needMore {
		if remaining <= 0 {
			break
		}

		allocation := pc.deficit
		if allocation > remaining {
			allocation = remaining
		}

		if details, exists := ba.Details[pc.name]; exists {
			details.Allocated += allocation
			ba.Details[pc.name] = details
			remaining -= allocation
		}
	}
}

// GetComponentBudget returns the allocated budget for a specific component
func (ba *BudgetAllocation) GetComponentBudget(component string) int {
	if details, exists := ba.Details[component]; exists {
		return details.Allocated
	}
	return 0
}

// RemainingBudget returns remaining unused tokens
func (ba *BudgetAllocation) RemainingBudget() int {
	totalUsed := ba.SystemPrompt + ba.UserQuery + ba.OutputBuffer
	for _, details := range ba.Details {
		totalUsed += details.Used
	}
	return ba.Total - totalUsed
}

// Validate checks if allocation is within budget constraints
func (ba *BudgetAllocation) Validate(maxTokens int) error {
	if ba.Total > maxTokens {
		return fmt.Errorf("total allocation (%d) exceeds maximum tokens (%d)", ba.Total, maxTokens)
	}

	totalAllocated := ba.SystemPrompt + ba.UserQuery + ba.OutputBuffer
	for _, details := range ba.Details {
		totalAllocated += details.Allocated
	}

	if totalAllocated > maxTokens {
		return fmt.Errorf("total allocated tokens (%d) exceeds maximum tokens (%d)", totalAllocated, maxTokens)
	}

	return nil
}

// Summary returns a human-readable summary of the allocation
func (ba *BudgetAllocation) Summary() string {
	summary := fmt.Sprintf("Token Budget Allocation (Total: %d)\n", ba.Total)
	summary += fmt.Sprintf("  System Prompt: %d tokens\n", ba.SystemPrompt)
	summary += fmt.Sprintf("  User Query: %d tokens\n", ba.UserQuery)
	summary += fmt.Sprintf("  Output Buffer: %d tokens\n", ba.OutputBuffer)
	summary += "  Context Components:\n"

	// Sort components by priority
	type componentInfo struct {
		name    string
		details TokenDetails
	}
	var components []componentInfo
	for name, details := range ba.Details {
		components = append(components, componentInfo{name, details})
	}

	// Sort by priority descending
	for i := 0; i < len(components); i++ {
		for j := i + 1; j < len(components); j++ {
			if components[j].details.Priority > components[i].details.Priority {
				components[i], components[j] = components[j], components[i]
			}
		}
	}

	for _, comp := range components {
		summary += fmt.Sprintf("    %s: %d tokens (priority: %d, ratio: %.2f%%)\n",
			comp.name,
			comp.details.Allocated,
			comp.details.Priority,
			comp.details.Ratio*100,
		)
		if comp.details.Used > 0 {
			summary += fmt.Sprintf("      Used: %d tokens (%.1f%% of allocated)\n",
				comp.details.Used,
				float64(comp.details.Used)/float64(comp.details.Allocated)*100,
			)
		}
	}

	remaining := ba.RemainingBudget()
	if remaining > 0 {
		summary += fmt.Sprintf("  Remaining: %d tokens\n", remaining)
	}

	return summary
}

// estimateSystemPromptTokens estimates tokens needed for system prompt
func estimateSystemPromptTokens() int {
	// Universal SQL prompt is approximately 1500-2000 tokens
	// SQL fix prompt is approximately 1200-1800 tokens
	// Use conservative estimate
	return 2000
}

// EstimateTokenCount provides a rough estimate of token count for text
// This is a simple heuristic: ~4 characters per token on average
func EstimateTokenCount(text string) int {
	if text == "" {
		return 0
	}

	// Simple heuristic: roughly 4 characters per token
	// This works reasonably well for English text and code
	charCount := len(text)
	estimatedTokens := int(math.Ceil(float64(charCount) / 4.0))

	// Account for overhead (JSON formatting, whitespace, etc.)
	overhead := int(float64(estimatedTokens) * 0.1)

	return estimatedTokens + overhead
}

// TruncateToTokenBudget truncates text to fit within a token budget
func TruncateToTokenBudget(text string, maxTokens int) string {
	estimatedTokens := EstimateTokenCount(text)

	if estimatedTokens <= maxTokens {
		return text
	}

	// Calculate roughly how many characters we can keep
	ratio := float64(maxTokens) / float64(estimatedTokens)
	targetLength := int(float64(len(text)) * ratio)

	// Ensure we don't exceed the budget
	targetLength = int(float64(targetLength) * 0.95) // 5% safety margin

	if targetLength <= 0 {
		return ""
	}

	if targetLength >= len(text) {
		return text
	}

	// Try to truncate at a reasonable boundary (newline, period, comma)
	truncated := text[:targetLength]

	// Look back for a good breaking point
	boundaries := []byte{'\n', '.', ',', ';', ' '}
	for _, boundary := range boundaries {
		for i := len(truncated) - 1; i > len(truncated)-100 && i >= 0; i-- {
			if truncated[i] == boundary {
				truncated = truncated[:i+1]
				break
			}
		}
	}

	return truncated + "...[truncated]"
}

// PrioritizeComponents adjusts priorities based on query characteristics
func PrioritizeComponents(query string, hasError bool) map[string]int {
	priorities := map[string]int{
		"schema":      10, // Always high priority
		"examples":    7,
		"business":    5,
		"performance": 3,
	}

	// If fixing an error, prioritize examples of correct patterns
	if hasError {
		priorities["examples"] = 9    // Boost examples to help understand correct patterns
		priorities["performance"] = 2 // Lower performance hints when fixing
	}

	// If query mentions specific business concepts, boost business rules
	businessKeywords := []string{"revenue", "profit", "customer", "order", "discount", "refund"}
	for _, keyword := range businessKeywords {
		if containsIgnoreCase(query, keyword) {
			priorities["business"] = 8
			break
		}
	}

	// If query mentions performance concerns, boost performance hints
	performanceKeywords := []string{"slow", "performance", "optimize", "fast", "index"}
	for _, keyword := range performanceKeywords {
		if containsIgnoreCase(query, keyword) {
			priorities["performance"] = 7
			break
		}
	}

	return priorities
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
