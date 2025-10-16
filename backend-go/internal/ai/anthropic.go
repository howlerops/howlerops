package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// anthropicProvider implements the AIProvider interface for Anthropic Claude
type anthropicProvider struct {
	config *AnthropicConfig
	client *http.Client
	logger *logrus.Logger
}

// Anthropic API structures
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	System    string             `json:"system,omitempty"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config *AnthropicConfig, logger *logrus.Logger) (AIProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}

	if config.Version == "" {
		config.Version = "2023-06-01"
	}

	if len(config.Models) == 0 {
		config.Models = []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
		}
	}

	return &anthropicProvider{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}, nil
}

// GenerateSQL generates SQL from natural language using Anthropic Claude
func (p *anthropicProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	prompt := p.buildGeneratePrompt(req)

	response, err := p.callAnthropic(ctx, req.Model, prompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(response, req)
}

// FixSQL fixes SQL based on error message using Anthropic Claude
func (p *anthropicProvider) FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	prompt := p.buildFixPrompt(req)

	response, err := p.callAnthropic(ctx, req.Model, prompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(response, req)
}

// HealthCheck checks if the Anthropic service is available
func (p *anthropicProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	// Create a simple test request
	testRequest := anthropicRequest{
		Model:     "claude-3-5-haiku-20241022",
		MaxTokens: 10,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	jsonBody, err := json.Marshal(testRequest)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderAnthropic,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Failed to create test request: %v", err),
			LastChecked: time.Now(),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderAnthropic,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Failed to create request: %v", err),
			LastChecked: time.Now(),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", p.config.Version)

	resp, err := p.client.Do(req)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderAnthropic,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Request failed: %v", err),
			LastChecked: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	responseTime := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &HealthStatus{
			Provider:     ProviderAnthropic,
			Status:       "unhealthy",
			Message:      fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			LastChecked:  time.Now(),
			ResponseTime: responseTime,
		}, nil
	}

	return &HealthStatus{
		Provider:     ProviderAnthropic,
		Status:       "healthy",
		Message:      "Service is operational",
		LastChecked:  time.Now(),
		ResponseTime: responseTime,
	}, nil
}

// GetModels returns available models from Anthropic
func (p *anthropicProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	// Anthropic doesn't have a models endpoint, so we return the configured models
	var models []ModelInfo
	for _, modelID := range p.config.Models {
		var description string
		var maxTokens int

		switch {
		case strings.Contains(modelID, "claude-3-5-sonnet"):
			description = "Claude 3.5 Sonnet - Most intelligent model, best for complex analysis"
			maxTokens = 200000
		case strings.Contains(modelID, "claude-3-5-haiku"):
			description = "Claude 3.5 Haiku - Fastest and most cost-effective model"
			maxTokens = 200000
		case strings.Contains(modelID, "claude-3-opus"):
			description = "Claude 3 Opus - Most powerful model for highly complex tasks"
			maxTokens = 200000
		default:
			description = fmt.Sprintf("Anthropic %s model", modelID)
			maxTokens = 100000
		}

		models = append(models, ModelInfo{
			ID:          modelID,
			Name:        modelID,
			Provider:    ProviderAnthropic,
			Description: description,
			MaxTokens:   maxTokens,
			Capabilities: []string{"text-to-sql", "sql-fixing", "explanation", "analysis"},
		})
	}

	return models, nil
}

// GetProviderType returns the provider type
func (p *anthropicProvider) GetProviderType() Provider {
	return ProviderAnthropic
}

// IsAvailable checks if the provider is available
func (p *anthropicProvider) IsAvailable(ctx context.Context) bool {
	health, err := p.HealthCheck(ctx)
	return err == nil && health.Status == "healthy"
}

// UpdateConfig updates the provider configuration
func (p *anthropicProvider) UpdateConfig(config interface{}) error {
	anthropicConfig, ok := config.(*AnthropicConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Anthropic provider")
	}

	if err := p.ValidateConfig(anthropicConfig); err != nil {
		return err
	}

	p.config = anthropicConfig
	return nil
}

// ValidateConfig validates the provider configuration
func (p *anthropicProvider) ValidateConfig(config interface{}) error {
	anthropicConfig, ok := config.(*AnthropicConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Anthropic provider")
	}

	if anthropicConfig.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if anthropicConfig.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	return nil
}

// callAnthropic makes a request to the Anthropic API
func (p *anthropicProvider) callAnthropic(ctx context.Context, model, prompt string, maxTokens int, temperature float64) (*anthropicResponse, error) {
	messages := []anthropicMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}
	systemPrompt := "You are an expert SQL developer. Generate clean, efficient SQL queries and provide clear explanations. Always format your responses as valid JSON when requested."

	return p.callAnthropicWithMessages(ctx, model, systemPrompt, messages, maxTokens, temperature)
}

// callAnthropicWithMessages makes a request to the Anthropic API with explicit messages
func (p *anthropicProvider) callAnthropicWithMessages(ctx context.Context, model, systemPrompt string, messages []anthropicMessage, maxTokens int, temperature float64) (*anthropicResponse, error) {
	requestBody := anthropicRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Messages:    messages,
		Temperature: temperature,
		System:      systemPrompt,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", p.config.Version)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp anthropicErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("Anthropic API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &anthropicResp, nil
}

// Chat handles generic conversational interactions
func (p *anthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	systemPrompt := req.System
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant for SQL Studio. Provide thoughtful, concise answers and include actionable guidance when relevant."
	}

	if req.Context != "" {
		systemPrompt = fmt.Sprintf("%s\n\nAdditional context:\n%s", systemPrompt, req.Context)
	}

	messages := []anthropicMessage{
		{
			Role:    "user",
			Content: req.Prompt,
		},
	}

	response, err := p.callAnthropicWithMessages(ctx, req.Model, systemPrompt, messages, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	if len(response.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	content := strings.TrimSpace(response.Content[0].Text)

	return &ChatResponse{
		Content:    content,
		Provider:   ProviderAnthropic,
		Model:      req.Model,
		TokensUsed: response.Usage.InputTokens + response.Usage.OutputTokens,
		Metadata: map[string]string{
			"stop_reason": response.StopReason,
		},
	}, nil
}

// buildGeneratePrompt builds a prompt for SQL generation
func (p *anthropicProvider) buildGeneratePrompt(req *SQLRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Generate a SQL query for the following request:\n\n")
	prompt.WriteString(fmt.Sprintf("Request: %s\n\n", req.Prompt))

	if req.Schema != "" {
		prompt.WriteString("Database Schema:\n")
		prompt.WriteString(req.Schema)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide your response as JSON with the following structure:\n")
	prompt.WriteString(`{
  "query": "SELECT ...",
  "explanation": "This query...",
  "confidence": 0.95,
  "suggestions": ["Consider adding an index on...", "..."],
  "warnings": ["This query might be slow on large tables", "..."]
}`)

	prompt.WriteString("\n\nRequirements:\n")
	prompt.WriteString("- The query should be syntactically correct\n")
	prompt.WriteString("- Use best practices for performance\n")
	prompt.WriteString("- Include helpful comments if the query is complex\n")
	prompt.WriteString("- Provide actionable suggestions for optimization\n")

	return prompt.String()
}

// buildFixPrompt builds a prompt for SQL fixing
func (p *anthropicProvider) buildFixPrompt(req *SQLRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Fix the following SQL query that has an error:\n\n")
	prompt.WriteString(fmt.Sprintf("Original Query:\n```sql\n%s\n```\n\n", req.Query))
	prompt.WriteString(fmt.Sprintf("Error Message:\n%s\n\n", req.Error))

	if req.Schema != "" {
		prompt.WriteString("Database Schema:\n")
		prompt.WriteString(req.Schema)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide your response as JSON with the following structure:\n")
	prompt.WriteString(`{
  "query": "SELECT ...",
  "explanation": "The error was caused by... Fixed by...",
  "confidence": 0.90,
  "suggestions": ["To prevent this error...", "..."],
  "warnings": ["Be careful with...", "..."]
}`)

	prompt.WriteString("\n\nRequirements:\n")
	prompt.WriteString("- Fix the syntax or logical error\n")
	prompt.WriteString("- Explain what caused the error\n")
	prompt.WriteString("- Provide suggestions to prevent similar errors\n")
	prompt.WriteString("- Ensure the fixed query is optimized\n")

	return prompt.String()
}

// parseResponse parses the Anthropic response into SQLResponse
func (p *anthropicProvider) parseResponse(response *anthropicResponse, req *SQLRequest) (*SQLResponse, error) {
	if len(response.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	content := response.Content[0].Text

	// Try to parse as JSON first
	var jsonResp struct {
		Query       string   `json:"query"`
		Explanation string   `json:"explanation"`
		Confidence  float64  `json:"confidence"`
		Suggestions []string `json:"suggestions"`
		Warnings    []string `json:"warnings"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResp); err == nil {
		return &SQLResponse{
			Query:       jsonResp.Query,
			Explanation: jsonResp.Explanation,
			Confidence:  jsonResp.Confidence,
			Suggestions: jsonResp.Suggestions,
			Warnings:    jsonResp.Warnings,
			Provider:    ProviderAnthropic,
			Model:       req.Model,
			TokensUsed:  response.Usage.InputTokens + response.Usage.OutputTokens,
		}, nil
	}

	// If JSON parsing fails, try to extract SQL from the content
	query := p.extractSQL(content)
	if query == "" {
		return nil, fmt.Errorf("could not extract SQL from response")
	}

	return &SQLResponse{
		Query:       query,
		Explanation: content,
		Confidence:  0.8, // Lower confidence for non-structured response
		Provider:    ProviderAnthropic,
		Model:       req.Model,
		TokensUsed:  response.Usage.InputTokens + response.Usage.OutputTokens,
	}, nil
}

// extractSQL attempts to extract SQL from unstructured text
func (p *anthropicProvider) extractSQL(content string) string {
	// Look for SQL code blocks
	if start := strings.Index(content, "```sql"); start != -1 {
		start += 6
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// Look for generic code blocks that might contain SQL
	if start := strings.Index(content, "```"); start != -1 {
		start += 3
		if end := strings.Index(content[start:], "```"); end != -1 {
			sql := strings.TrimSpace(content[start : start+end])
			if p.looksLikeSQL(sql) {
				return sql
			}
		}
	}

	// Look for JSON within the response
	if start := strings.Index(content, "{"); start != -1 {
		if end := strings.LastIndex(content, "}"); end != -1 && end > start {
			jsonStr := content[start : end+1]
			var jsonResp struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal([]byte(jsonStr), &jsonResp); err == nil && jsonResp.Query != "" {
				return jsonResp.Query
			}
		}
	}

	// Look for common SQL keywords at the start of lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if p.looksLikeSQL(line) && len(line) > 10 {
			return line
		}
	}

	return ""
}

// looksLikeSQL checks if a string looks like SQL
func (p *anthropicProvider) looksLikeSQL(text string) bool {
	text = strings.ToUpper(strings.TrimSpace(text))
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "WITH"}

	for _, keyword := range sqlKeywords {
		if strings.HasPrefix(text, keyword) {
			return true
		}
	}
	return false
}
