package ai

import (
	"context"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
)

// ProviderWithPrompts wraps a provider with enhanced prompt building capabilities
type ProviderWithPrompts struct {
	provider       AIProvider
	promptBuilder  *PromptBuilder
	defaultDialect SQLDialect
}

// WrapProviderWithPrompts wraps an existing provider with enhanced prompts
func WrapProviderWithPrompts(
	provider AIProvider,
	contextBuilder *rag.ContextBuilder,
	defaultDialect SQLDialect,
) *ProviderWithPrompts {
	return &ProviderWithPrompts{
		provider:       provider,
		promptBuilder:  NewPromptBuilder(contextBuilder),
		defaultDialect: defaultDialect,
	}
}

// GenerateSQLWithEnhancedPrompts generates SQL using the new universal prompts
func (p *ProviderWithPrompts) GenerateSQLWithEnhancedPrompts(
	ctx context.Context,
	req *SQLRequest,
) (*SQLResponse, *rag.BudgetAllocation, error) {
	// Detect dialect
	dialect := p.detectDialect(req)

	// Get recommended token budget for the model
	maxTokens := GetRecommendedTokenBudget(req.Model)
	if req.MaxTokens > 0 && req.MaxTokens < maxTokens {
		maxTokens = req.MaxTokens
	}

	// Build enhanced prompts with RAG context
	systemPrompt, userPrompt, allocation, err := p.promptBuilder.BuildSQLGenerationPrompt(
		ctx,
		req,
		dialect,
		maxTokens,
	)

	if err != nil {
		// Fall back to original behavior if prompt building fails
		response, genErr := p.provider.GenerateSQL(ctx, req)
		return response, nil, genErr
	}

	// Create modified request with enhanced prompts
	enhancedReq := &SQLRequest{
		Prompt:      userPrompt,
		Schema:      "", // Schema is now in the prompt
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Context:     req.Context,
	}

	// Store system prompt in context for provider to use
	if enhancedReq.Context == nil {
		enhancedReq.Context = make(map[string]string)
	}
	enhancedReq.Context["system_prompt"] = systemPrompt

	// Call the underlying provider
	response, err := p.provider.GenerateSQL(ctx, enhancedReq)
	if err != nil {
		return nil, allocation, err
	}

	return response, allocation, nil
}

// FixSQLWithEnhancedPrompts fixes SQL using the new universal prompts
func (p *ProviderWithPrompts) FixSQLWithEnhancedPrompts(
	ctx context.Context,
	req *SQLRequest,
) (*SQLResponse, *rag.BudgetAllocation, error) {
	// Detect dialect
	dialect := p.detectDialect(req)

	// Get recommended token budget for the model
	maxTokens := GetRecommendedTokenBudget(req.Model)
	if req.MaxTokens > 0 && req.MaxTokens < maxTokens {
		maxTokens = req.MaxTokens
	}

	// Build enhanced prompts with RAG context
	systemPrompt, userPrompt, allocation, err := p.promptBuilder.BuildSQLFixPrompt(
		ctx,
		req,
		dialect,
		maxTokens,
	)

	if err != nil {
		// Fall back to original behavior if prompt building fails
		response, fixErr := p.provider.FixSQL(ctx, req)
		return response, nil, fixErr
	}

	// Create modified request with enhanced prompts
	enhancedReq := &SQLRequest{
		Prompt:      userPrompt,
		Query:       "", // Query is now in the prompt
		Error:       "", // Error is now in the prompt
		Schema:      "", // Schema is now in the prompt
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Context:     req.Context,
	}

	// Store system prompt in context for provider to use
	if enhancedReq.Context == nil {
		enhancedReq.Context = make(map[string]string)
	}
	enhancedReq.Context["system_prompt"] = systemPrompt

	// Call the underlying provider
	response, err := p.provider.FixSQL(ctx, enhancedReq)
	if err != nil {
		return nil, allocation, err
	}

	return response, allocation, nil
}

// detectDialect detects the SQL dialect from request context
func (p *ProviderWithPrompts) detectDialect(req *SQLRequest) SQLDialect {
	if req.Context == nil {
		return p.defaultDialect
	}

	// Try to detect from connection type
	if connType, ok := req.Context["connection_type"]; ok {
		dialect := DetectDialectFromConnection(connType, req.Context)
		if dialect != DialectGeneric {
			return dialect
		}
	}

	// Try explicit dialect specification
	if dialectStr, ok := req.Context["dialect"]; ok {
		return SQLDialect(dialectStr)
	}

	return p.defaultDialect
}

// GetProvider returns the underlying provider
func (p *ProviderWithPrompts) GetProvider() AIProvider {
	return p.provider
}

// BuildMessagesWithSystemPrompt builds chat messages with system prompt from context
func BuildMessagesWithSystemPrompt(systemPrompt, userPrompt string) []ChatMessage {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	return messages
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string
	Content string
}

// ProviderHelpers provides helper methods for providers to use enhanced prompts
type ProviderHelpers struct{}

// NewProviderHelpers creates a new provider helpers instance
func NewProviderHelpers() *ProviderHelpers {
	return &ProviderHelpers{}
}

// ExtractSystemPromptFromContext extracts system prompt from request context
func (h *ProviderHelpers) ExtractSystemPromptFromContext(req *SQLRequest) string {
	if req.Context == nil {
		return ""
	}

	if systemPrompt, ok := req.Context["system_prompt"]; ok {
		return systemPrompt
	}

	return ""
}

// GetDefaultSystemPrompt returns a default system prompt if no enhanced prompt is available
func (h *ProviderHelpers) GetDefaultSystemPrompt(isFixing bool) string {
	if isFixing {
		return "You are an expert SQL debugger. Fix broken SQL queries and explain the issues clearly."
	}
	return "You are an expert SQL developer. Generate clean, efficient SQL queries and provide clear explanations."
}

// BuildPromptForProvider builds appropriate prompts for a provider
func (h *ProviderHelpers) BuildPromptForProvider(req *SQLRequest, isFixing bool) (systemPrompt, userPrompt string) {
	// Try to get enhanced system prompt from context
	systemPrompt = h.ExtractSystemPromptFromContext(req)

	// Fall back to default if no enhanced prompt
	if systemPrompt == "" {
		systemPrompt = h.GetDefaultSystemPrompt(isFixing)
	}

	// User prompt is always from the request
	userPrompt = req.Prompt

	return systemPrompt, userPrompt
}

// Example usage for updating providers:
//
// In GenerateSQL method:
//   helpers := NewProviderHelpers()
//   systemPrompt, userPrompt := helpers.BuildPromptForProvider(req, false)
//   messages := BuildMessagesWithSystemPrompt(systemPrompt, userPrompt)
//
// In FixSQL method:
//   helpers := NewProviderHelpers()
//   systemPrompt, userPrompt := helpers.BuildPromptForProvider(req, true)
//   messages := BuildMessagesWithSystemPrompt(systemPrompt, userPrompt)
