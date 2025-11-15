package ai

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	claudecode "github.com/humanlayer/humanlayer/claudecode-go"
)

// ClaudeCodeConfig contains configuration for Claude Code provider
type ClaudeCodeConfig struct {
	ClaudePath  string  `json:"claude_path" yaml:"claude_path"`
	Model       string  `json:"model" yaml:"model"`
	MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`
	Temperature float64 `json:"temperature" yaml:"temperature"`
}

// ClaudeCodeProvider implements the Provider interface for Claude Code
type ClaudeCodeProvider struct {
	config *ClaudeCodeConfig
	client *claudecode.Client
}

// NewClaudeCodeProvider creates a new Claude Code provider
func NewClaudeCodeProvider(config *ClaudeCodeConfig) (*ClaudeCodeProvider, error) {
	if config.Model == "" {
		config.Model = "opus" // Default Claude Code model
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	if config.Temperature == 0 {
		config.Temperature = 0.7
	}

	var client *claudecode.Client
	var err error

	if config.ClaudePath != "" {
		client = claudecode.NewClientWithPath(config.ClaudePath)
	} else {
		client, err = claudecode.NewClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create Claude Code client: %w", err)
		}
	}

	return &ClaudeCodeProvider{
		config: config,
		client: client,
	}, nil
}

// GenerateSQL generates SQL from natural language using Claude Code
func (p *ClaudeCodeProvider) GenerateSQL(_ context.Context, prompt, schema string, options ...GenerateOption) (*SQLResponse, error) {
	opts := &GenerateOptions{
		Model:       p.config.Model,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	for _, opt := range options {
		opt(opts)
	}

	// Build the prompt for SQL generation
	fullPrompt := p.buildSQLPrompt(prompt, schema)

	startTime := time.Now()

	// For now, only Opus model is available in the claudecode library
	model := claudecode.ModelOpus

	// Launch Claude Code session and wait for completion
	result, err := p.client.LaunchAndWait(claudecode.SessionConfig{
		Query:        fullPrompt,
		Model:        model,
		OutputFormat: claudecode.OutputText,
		SystemPrompt: "You are an expert SQL developer. Generate optimized SQL queries based on natural language requests. Return only valid, executable SQL code.",
	})

	if err != nil {
		return nil, fmt.Errorf("claude code error: %w", err)
	}

	// Extract SQL from response
	query := p.extractSQL(result.Result)

	tokensUsed := 0
	if result.Usage != nil {
		tokensUsed = result.Usage.OutputTokens
	}

	return &SQLResponse{
		Query:       query,
		Explanation: result.Result,
		Confidence:  0.95, // Claude typically has high confidence
		Provider:    ProviderClaudeCode,
		Model:       opts.Model,
		TokensUsed:  tokensUsed,
		TimeTaken:   time.Since(startTime),
		Metadata: map[string]string{
			"model": string(model),
		},
	}, nil
}

// FixSQL fixes SQL based on error message using Claude Code
func (p *ClaudeCodeProvider) FixSQL(_ context.Context, query, errorMsg, schema string, options ...GenerateOption) (*SQLResponse, error) {
	opts := &GenerateOptions{
		Model:       p.config.Model,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	for _, opt := range options {
		opt(opts)
	}

	// Build the prompt for SQL fixing
	fullPrompt := p.buildFixPrompt(query, errorMsg, schema)

	startTime := time.Now()

	// For now, only Opus model is available in the claudecode library
	model := claudecode.ModelOpus

	// Launch Claude Code session and wait for completion
	result, err := p.client.LaunchAndWait(claudecode.SessionConfig{
		Query:        fullPrompt,
		Model:        model,
		OutputFormat: claudecode.OutputText,
		SystemPrompt: "You are an expert SQL developer. Fix SQL queries based on error messages. Return only the corrected SQL code.",
	})

	if err != nil {
		return nil, fmt.Errorf("claude code error: %w", err)
	}

	// Extract fixed SQL from response
	fixedQuery := p.extractSQL(result.Result)

	tokensUsed := 0
	if result.Usage != nil {
		tokensUsed = result.Usage.OutputTokens
	}

	return &SQLResponse{
		Query:       fixedQuery,
		Explanation: result.Result,
		Confidence:  0.95,
		Provider:    ProviderClaudeCode,
		Model:       opts.Model,
		TokensUsed:  tokensUsed,
		TimeTaken:   time.Since(startTime),
		Metadata: map[string]string{
			"model": string(model),
		},
	}, nil
}

// Chat handles generic conversational interactions using Claude Code
func (p *ClaudeCodeProvider) Chat(_ context.Context, prompt string, options ...GenerateOption) (*ChatResponse, error) {
	opts := &GenerateOptions{
		Model:       p.config.Model,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	for _, opt := range options {
		opt(opts)
	}

	var systemPrompt string
	if opts.Context != nil {
		systemPrompt = opts.Context["system"]
	}
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant for Howlerops. Provide concise, accurate answers and actionable guidance."
	}

	var fullPrompt strings.Builder
	if opts.Context != nil {
		if ctxText := opts.Context["context"]; ctxText != "" {
			fullPrompt.WriteString("Context:\n")
			fullPrompt.WriteString(ctxText)
			fullPrompt.WriteString("\n\n")
		}
	}
	fullPrompt.WriteString(prompt)

	startTime := time.Now()

	model := claudecode.ModelOpus
	result, err := p.client.LaunchAndWait(claudecode.SessionConfig{
		Query:        fullPrompt.String(),
		Model:        model,
		OutputFormat: claudecode.OutputText,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("claude code error: %w", err)
	}

	tokensUsed := 0
	if result.Usage != nil {
		tokensUsed = result.Usage.OutputTokens
	}

	return &ChatResponse{
		Content:    strings.TrimSpace(result.Result),
		Provider:   ProviderClaudeCode,
		Model:      opts.Model,
		TokensUsed: tokensUsed,
		TimeTaken:  time.Since(startTime),
		Metadata: map[string]string{
			"model": string(model),
		},
	}, nil
}

// GetHealth returns the health status of the Claude Code provider
func (p *ClaudeCodeProvider) GetHealth(_ context.Context) (*HealthStatus, error) {
	startTime := time.Now()

	// First check if the claude binary exists
	_, lookupErr := exec.LookPath(p.config.ClaudePath)
	if lookupErr != nil {
		return &HealthStatus{
			Provider:     ProviderClaudeCode,
			Status:       "unhealthy",
			Message:      fmt.Sprintf("Claude CLI not found at path '%s'. Please install Claude CLI or update the path.", p.config.ClaudePath),
			LastChecked:  time.Now(),
			ResponseTime: time.Since(startTime),
		}, lookupErr
	}

	// For now, if the binary exists, we assume it's healthy
	// We can't actually test it without potentially hanging
	// TODO: Implement a better health check once the claudecode library supports timeouts
	return &HealthStatus{
		Provider:     ProviderClaudeCode,
		Status:       "healthy",
		Message:      fmt.Sprintf("Claude CLI found at path '%s' (functionality not tested)", p.config.ClaudePath),
		LastChecked:  time.Now(),
		ResponseTime: time.Since(startTime),
	}, nil
}

// ListModels returns available models for Claude Code
func (p *ClaudeCodeProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	// Claude Code models available through the CLI
	models := []ModelInfo{
		{
			ID:           "opus",
			Name:         "Claude Opus (via Claude Code)",
			Provider:     ProviderClaudeCode,
			Description:  "Most capable Claude model for complex tasks, accessed through Claude Code CLI",
			MaxTokens:    200000,
			Capabilities: []string{"sql_generation", "code_generation", "reasoning", "analysis"},
		},
	}

	return models, nil
}

// GetProviderType returns the type of provider
func (p *ClaudeCodeProvider) GetProviderType() Provider {
	return ProviderClaudeCode
}

// Close cleans up provider resources
func (p *ClaudeCodeProvider) Close() error {
	// Claude Code client doesn't require cleanup
	return nil
}

// buildSQLPrompt builds a prompt for SQL generation
func (p *ClaudeCodeProvider) buildSQLPrompt(prompt, schema string) string {
	var sb strings.Builder

	sb.WriteString("Generate a SQL query for the following request.\n\n")

	if schema != "" {
		sb.WriteString("Database Schema:\n")
		sb.WriteString("```sql\n")
		sb.WriteString(schema)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("Request: ")
	sb.WriteString(prompt)
	sb.WriteString("\n\n")

	sb.WriteString("Requirements:\n")
	sb.WriteString("1. Generate only the SQL query\n")
	sb.WriteString("2. Use proper SQL syntax and formatting\n")
	sb.WriteString("3. The query should be optimized and efficient\n")
	sb.WriteString("4. Include comments for complex parts\n")
	sb.WriteString("5. Return the query in a SQL code block\n")

	return sb.String()
}

// buildFixPrompt builds a prompt for SQL fixing
func (p *ClaudeCodeProvider) buildFixPrompt(query, errorMsg, schema string) string {
	var sb strings.Builder

	sb.WriteString("Fix the following SQL query based on the error message.\n\n")

	if schema != "" {
		sb.WriteString("Database Schema:\n")
		sb.WriteString("```sql\n")
		sb.WriteString(schema)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("Original Query:\n")
	sb.WriteString("```sql\n")
	sb.WriteString(query)
	sb.WriteString("\n```\n\n")

	sb.WriteString("Error Message:\n")
	sb.WriteString(errorMsg)
	sb.WriteString("\n\n")

	sb.WriteString("Requirements:\n")
	sb.WriteString("1. Fix the SQL query to resolve the error\n")
	sb.WriteString("2. Return only the corrected SQL query\n")
	sb.WriteString("3. Maintain the original intent of the query\n")
	sb.WriteString("4. Return the query in a SQL code block\n")

	return sb.String()
}

// extractSQL extracts SQL from the response
func (p *ClaudeCodeProvider) extractSQL(response string) string {
	// Try to find SQL code blocks
	if start := strings.Index(response, "```sql"); start != -1 {
		start += 6
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Try to find generic code blocks
	if start := strings.Index(response, "```"); start != -1 {
		start += 3
		// Skip language identifier if present
		if newline := strings.Index(response[start:], "\n"); newline != -1 {
			start += newline + 1
		}
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Return the whole response if no code blocks found
	return strings.TrimSpace(response)
}
