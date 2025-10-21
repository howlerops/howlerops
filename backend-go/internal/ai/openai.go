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

// openaiProvider implements the AIProvider interface for OpenAI
type openaiProvider struct {
	config *OpenAIConfig
	client *http.Client
	logger *logrus.Logger
}

// OpenAI API structures
type openaiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openaiChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream"`
}

type openaiChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

type openaiModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config *OpenAIConfig, logger *logrus.Logger) (AIProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	if len(config.Models) == 0 {
		config.Models = []string{"gpt-4o-mini", "gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}
	}

	return &openaiProvider{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}, nil
}

func (p *openaiProvider) setHTTPClient(client *http.Client) {
	if client != nil {
		p.client = client
	}
}

// GenerateSQL generates SQL from natural language using OpenAI
func (p *openaiProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	prompt := p.buildGeneratePrompt(req)

	response, err := p.callOpenAI(ctx, req.Model, prompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(response, req)
}

// FixSQL fixes SQL based on error message using OpenAI
func (p *openaiProvider) FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	prompt := p.buildFixPrompt(req)

	response, err := p.callOpenAI(ctx, req.Model, prompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(response, req)
}

// HealthCheck checks if the OpenAI service is available
func (p *openaiProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	// Try to list models as a health check
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/models", nil)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderOpenAI,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Failed to create request: %v", err),
			LastChecked: time.Now(),
		}, nil
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.OrgID != "" {
		req.Header.Set("OpenAI-Organization", p.config.OrgID)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderOpenAI,
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
			Provider:     ProviderOpenAI,
			Status:       "unhealthy",
			Message:      fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			LastChecked:  time.Now(),
			ResponseTime: responseTime,
		}, nil
	}

	return &HealthStatus{
		Provider:     ProviderOpenAI,
		Status:       "healthy",
		Message:      "Service is operational",
		LastChecked:  time.Now(),
		ResponseTime: responseTime,
	}, nil
}

// GetModels returns available models from OpenAI
func (p *openaiProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.OrgID != "" {
		req.Header.Set("OpenAI-Organization", p.config.OrgID)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var modelsResp openaiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	var models []ModelInfo
	for _, model := range modelsResp.Data {
		// Filter for chat/completion models
		if strings.Contains(model.ID, "gpt") {
			models = append(models, ModelInfo{
				ID:       model.ID,
				Name:     model.ID,
				Provider: ProviderOpenAI,
				Description: fmt.Sprintf("OpenAI %s model", model.ID),
				Capabilities: []string{"text-to-sql", "sql-fixing", "explanation"},
			})
		}
	}

	return models, nil
}

// GetProviderType returns the provider type
func (p *openaiProvider) GetProviderType() Provider {
	return ProviderOpenAI
}

// IsAvailable checks if the provider is available
func (p *openaiProvider) IsAvailable(ctx context.Context) bool {
	health, err := p.HealthCheck(ctx)
	return err == nil && health.Status == "healthy"
}

// UpdateConfig updates the provider configuration
func (p *openaiProvider) UpdateConfig(config interface{}) error {
	openaiConfig, ok := config.(*OpenAIConfig)
	if !ok {
		return fmt.Errorf("invalid config type for OpenAI provider")
	}

	if err := p.ValidateConfig(openaiConfig); err != nil {
		return err
	}

	p.config = openaiConfig
	return nil
}

// ValidateConfig validates the provider configuration
func (p *openaiProvider) ValidateConfig(config interface{}) error {
	openaiConfig, ok := config.(*OpenAIConfig)
	if !ok {
		return fmt.Errorf("invalid config type for OpenAI provider")
	}

	if openaiConfig.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if openaiConfig.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	return nil
}

// callOpenAI makes a request to the OpenAI API
func (p *openaiProvider) callOpenAI(ctx context.Context, model, prompt string, maxTokens int, temperature float64) (*openaiChatResponse, error) {
	messages := []openaiChatMessage{
		{
			Role:    "system",
			Content: "You are an expert SQL developer. Generate clean, efficient SQL queries and provide clear explanations.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return p.callOpenAIWithMessages(ctx, model, messages, maxTokens, temperature)
}

// callOpenAIWithMessages makes a chat completion request with explicit messages
func (p *openaiProvider) callOpenAIWithMessages(ctx context.Context, model string, messages []openaiChatMessage, maxTokens int, temperature float64) (*openaiChatResponse, error) {
	requestBody := openaiChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.OrgID != "" {
		req.Header.Set("OpenAI-Organization", p.config.OrgID)
	}

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
		var errorResp openaiErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("OpenAI API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var chatResp openaiChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chatResp, nil
}

// Chat handles generic conversational interactions
func (p *openaiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	systemPrompt := req.System
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant for SQL Studio. Provide concise, accurate answers. Use Markdown formatting when it improves clarity."
	}

	messages := []openaiChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	if req.Context != "" {
		messages = append(messages, openaiChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("Additional context:\n%s", req.Context),
		})
	}

	messages = append(messages, openaiChatMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	response, err := p.callOpenAIWithMessages(ctx, req.Model, messages, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	metadata := map[string]string{}
	if response.Choices[0].FinishReason != "" {
		metadata["finish_reason"] = response.Choices[0].FinishReason
	}

	return &ChatResponse{
		Content:    content,
		Provider:   ProviderOpenAI,
		Model:      req.Model,
		TokensUsed: response.Usage.TotalTokens,
		Metadata:   metadata,
	}, nil
}

// buildGeneratePrompt builds a prompt for SQL generation
func (p *openaiProvider) buildGeneratePrompt(req *SQLRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Generate a SQL query for the following request:\n\n")
	prompt.WriteString(fmt.Sprintf("Request: %s\n\n", req.Prompt))

	if req.Schema != "" {
		prompt.WriteString("Database Schema:\n")
		prompt.WriteString(req.Schema)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. The SQL query\n")
	prompt.WriteString("2. An explanation of what the query does\n")
	prompt.WriteString("3. Any important notes or considerations\n\n")
	prompt.WriteString("Format your response as JSON with the following structure:\n")
	prompt.WriteString(`{
  "query": "SELECT ...",
  "explanation": "This query...",
  "confidence": 0.95,
  "suggestions": ["Consider adding an index on...", "..."]
}`)

	return prompt.String()
}

// buildFixPrompt builds a prompt for SQL fixing
func (p *openaiProvider) buildFixPrompt(req *SQLRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Fix the following SQL query that has an error:\n\n")
	prompt.WriteString(fmt.Sprintf("Original Query:\n%s\n\n", req.Query))
	prompt.WriteString(fmt.Sprintf("Error Message:\n%s\n\n", req.Error))

	if req.Schema != "" {
		prompt.WriteString("Database Schema:\n")
		prompt.WriteString(req.Schema)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. The corrected SQL query\n")
	prompt.WriteString("2. An explanation of what was wrong and how it was fixed\n")
	prompt.WriteString("3. Any suggestions to prevent similar errors\n\n")
	prompt.WriteString("Format your response as JSON with the following structure:\n")
	prompt.WriteString(`{
  "query": "SELECT ...",
  "explanation": "The error was caused by... Fixed by...",
  "confidence": 0.90,
  "suggestions": ["To prevent this error...", "..."]
}`)

	return prompt.String()
}

// parseResponse parses the OpenAI response into SQLResponse
func (p *openaiProvider) parseResponse(response *openaiChatResponse, req *SQLRequest) (*SQLResponse, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	content := response.Choices[0].Message.Content

	// Try to parse as JSON first
	var jsonResp struct {
		Query       string   `json:"query"`
		Explanation string   `json:"explanation"`
		Confidence  float64  `json:"confidence"`
		Suggestions []string `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResp); err == nil {
		return &SQLResponse{
			Query:       jsonResp.Query,
			Explanation: jsonResp.Explanation,
			Confidence:  jsonResp.Confidence,
			Suggestions: jsonResp.Suggestions,
			Provider:    ProviderOpenAI,
			Model:       req.Model,
			TokensUsed:  response.Usage.TotalTokens,
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
		Provider:    ProviderOpenAI,
		Model:       req.Model,
		TokensUsed:  response.Usage.TotalTokens,
	}, nil
}

// extractSQL attempts to extract SQL from unstructured text
func (p *openaiProvider) extractSQL(content string) string {
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
func (p *openaiProvider) looksLikeSQL(text string) bool {
	text = strings.ToUpper(strings.TrimSpace(text))
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER"}

	for _, keyword := range sqlKeywords {
		if strings.HasPrefix(text, keyword) {
			return true
		}
	}
	return false
}
