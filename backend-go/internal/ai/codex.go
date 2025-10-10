package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// CodexConfig contains configuration for OpenAI Codex provider
type CodexConfig struct {
	APIKey       string  `json:"api_key" yaml:"api_key"`
	Organization string  `json:"organization" yaml:"organization"`
	BaseURL      string  `json:"base_url" yaml:"base_url"`
	Model        string  `json:"model" yaml:"model"`
	MaxTokens    int     `json:"max_tokens" yaml:"max_tokens"`
	Temperature  float32 `json:"temperature" yaml:"temperature"`
}

// CodexProvider implements the Provider interface for OpenAI Codex
type CodexProvider struct {
	config *CodexConfig
	client *openai.Client
}

// NewCodexProvider creates a new Codex provider
func NewCodexProvider(config *CodexConfig) (*CodexProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if config.Model == "" {
		config.Model = "code-davinci-002" // Default Codex model
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 2048
	}

	if config.Temperature == 0 {
		config.Temperature = 0.0 // Deterministic for code generation
	}

	// Configure OpenAI client
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.Organization != "" {
		clientConfig.OrgID = config.Organization
	}
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &CodexProvider{
		config: config,
		client: client,
	}, nil
}

// GenerateSQL generates SQL from natural language using Codex
func (p *CodexProvider) GenerateSQL(ctx context.Context, prompt string, schema string, options ...GenerateOption) (*SQLResponse, error) {
	opts := &GenerateOptions{
		Model:       p.config.Model,
		MaxTokens:   p.config.MaxTokens,
		Temperature: float64(p.config.Temperature),
	}

	for _, opt := range options {
		opt(opts)
	}

	// Build the prompt for SQL generation
	fullPrompt := p.buildSQLPrompt(prompt, schema)

	startTime := time.Now()

	// Create completion request
	req := openai.CompletionRequest{
		Model:       opts.Model,
		Prompt:      fullPrompt,
		MaxTokens:   opts.MaxTokens,
		Temperature: float32(opts.Temperature),
		Stop:        []string{"--", ";", "\n\n"},
		N:           1,
	}

	resp, err := p.client.CreateCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("codex API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Codex")
	}

	// Extract SQL from response
	query := p.extractSQL(resp.Choices[0].Text)

	return &SQLResponse{
		Query:       query,
		Explanation: resp.Choices[0].Text,
		Confidence:  0.90, // Codex typically has high confidence for SQL
		Provider:    ProviderCodex,
		Model:       opts.Model,
		TokensUsed:  resp.Usage.CompletionTokens,
		TimeTaken:   time.Since(startTime),
		Metadata: map[string]string{
			"request_id":        resp.ID,
			"prompt_tokens":     fmt.Sprintf("%d", resp.Usage.PromptTokens),
			"completion_tokens": fmt.Sprintf("%d", resp.Usage.CompletionTokens),
			"total_tokens":      fmt.Sprintf("%d", resp.Usage.TotalTokens),
		},
	}, nil
}

// FixSQL fixes SQL based on error message using Codex
func (p *CodexProvider) FixSQL(ctx context.Context, query string, errorMsg string, schema string, options ...GenerateOption) (*SQLResponse, error) {
	opts := &GenerateOptions{
		Model:       p.config.Model,
		MaxTokens:   p.config.MaxTokens,
		Temperature: float64(p.config.Temperature),
	}

	for _, opt := range options {
		opt(opts)
	}

	// Build the prompt for SQL fixing
	fullPrompt := p.buildFixPrompt(query, errorMsg, schema)

	startTime := time.Now()

	// Create completion request
	req := openai.CompletionRequest{
		Model:       opts.Model,
		Prompt:      fullPrompt,
		MaxTokens:   opts.MaxTokens,
		Temperature: float32(opts.Temperature),
		Stop:        []string{"--", ";", "\n\n"},
		N:           1,
	}

	resp, err := p.client.CreateCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("codex API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Codex")
	}

	// Extract fixed SQL from response
	fixedQuery := p.extractSQL(resp.Choices[0].Text)

	return &SQLResponse{
		Query:       fixedQuery,
		Explanation: resp.Choices[0].Text,
		Confidence:  0.90,
		Provider:    ProviderCodex,
		Model:       opts.Model,
		TokensUsed:  resp.Usage.CompletionTokens,
		TimeTaken:   time.Since(startTime),
		Metadata: map[string]string{
			"request_id":        resp.ID,
			"prompt_tokens":     fmt.Sprintf("%d", resp.Usage.PromptTokens),
			"completion_tokens": fmt.Sprintf("%d", resp.Usage.CompletionTokens),
			"total_tokens":      fmt.Sprintf("%d", resp.Usage.TotalTokens),
		},
	}, nil
}

// GetHealth returns the health status of the Codex provider
func (p *CodexProvider) GetHealth(ctx context.Context) (*HealthStatus, error) {
	startTime := time.Now()

	// Test the API with a simple request
	req := openai.CompletionRequest{
		Model:     "code-davinci-002",
		Prompt:    "-- SQL comment",
		MaxTokens: 1,
	}

	_, err := p.client.CreateCompletion(ctx, req)
	if err != nil {
		return &HealthStatus{
			Provider:     ProviderCodex,
			Status:       "unhealthy",
			Message:      fmt.Sprintf("Codex API error: %v", err),
			LastChecked:  time.Now(),
			ResponseTime: time.Since(startTime),
		}, nil
	}

	return &HealthStatus{
		Provider:     ProviderCodex,
		Status:       "healthy",
		Message:      "Codex API is operational",
		LastChecked:  time.Now(),
		ResponseTime: time.Since(startTime),
	}, nil
}

// ListModels returns available models for Codex
func (p *CodexProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Codex models
	models := []ModelInfo{
		{
			ID:           "code-davinci-002",
			Name:         "Codex Davinci",
			Provider:     ProviderCodex,
			Description:  "Most capable Codex model for SQL and code generation",
			MaxTokens:    8000,
			Capabilities: []string{"sql_generation", "code_generation", "code_completion"},
		},
		{
			ID:           "code-cushman-001",
			Name:         "Codex Cushman",
			Provider:     ProviderCodex,
			Description:  "Faster Codex model for simpler SQL tasks",
			MaxTokens:    2048,
			Capabilities: []string{"sql_generation", "code_generation"},
		},
	}

	return models, nil
}

// GetProviderType returns the type of provider
func (p *CodexProvider) GetProviderType() Provider {
	return ProviderCodex
}

// Close cleans up provider resources
func (p *CodexProvider) Close() error {
	// OpenAI client doesn't require cleanup
	return nil
}

// buildSQLPrompt builds a prompt for SQL generation
func (p *CodexProvider) buildSQLPrompt(prompt string, schema string) string {
	var sb strings.Builder

	// Codex works best with code comments as prompts
	sb.WriteString("-- Database SQL Generation\n")

	if schema != "" {
		sb.WriteString("-- Schema:\n")
		lines := strings.Split(schema, "\n")
		for _, line := range lines {
			sb.WriteString("-- ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("-- Task: ")
	sb.WriteString(prompt)
	sb.WriteString("\n\n")
	sb.WriteString("-- SQL Query:\n")
	sb.WriteString("SELECT ")

	return sb.String()
}

// buildFixPrompt builds a prompt for SQL fixing
func (p *CodexProvider) buildFixPrompt(query string, errorMsg string, schema string) string {
	var sb strings.Builder

	sb.WriteString("-- Fix SQL Query\n")

	if schema != "" {
		sb.WriteString("-- Schema:\n")
		lines := strings.Split(schema, "\n")
		for _, line := range lines {
			sb.WriteString("-- ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("-- Original Query:\n")
	sb.WriteString(query)
	sb.WriteString("\n\n")

	sb.WriteString("-- Error: ")
	sb.WriteString(errorMsg)
	sb.WriteString("\n\n")

	sb.WriteString("-- Fixed Query:\n")
	sb.WriteString("SELECT ")

	return sb.String()
}

// extractSQL extracts SQL from the response
func (p *CodexProvider) extractSQL(response string) string {
	// Remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Remove SQL comments if present
	lines := strings.Split(response, "\n")
	var sqlLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "--") && trimmed != "" {
			sqlLines = append(sqlLines, line)
		}
	}

	if len(sqlLines) > 0 {
		return strings.Join(sqlLines, "\n")
	}

	return response
}