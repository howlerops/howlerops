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

// OllamaProvider implements the AIProvider interface for Ollama
type OllamaProvider struct {
	config *OllamaConfig
	client *http.Client
	logger *logrus.Logger
}

const defaultOllamaSystemPrompt = "You are an expert SQL developer. Generate clean, efficient SQL queries and provide clear explanations. Always format your responses as valid JSON when requested."

// Ollama API structures
type ollamaGenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Model              string    `json:"model"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	CreatedAt          time.Time `json:"created_at"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

type ollamaListResponse struct {
	Models []struct {
		Name       string    `json:"name"`
		Size       int64     `json:"size"`
		Digest     string    `json:"digest"`
		ModifiedAt time.Time `json:"modified_at"`
		Details    struct {
			Format            string   `json:"format"`
			Family            string   `json:"family"`
			Families          []string `json:"families"`
			ParameterSize     string   `json:"parameter_size"`
			QuantizationLevel string   `json:"quantization_level"`
		} `json:"details"`
	} `json:"models"`
}

type ollamaPullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

type ollamaPullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

type ollamaErrorResponse struct {
	Error string `json:"error"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config *OllamaConfig, logger *logrus.Logger) (AIProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:11434"
	}

	if config.PullTimeout == 0 {
		config.PullTimeout = 10 * time.Minute
	}

	if config.GenerateTimeout == 0 {
		config.GenerateTimeout = 2 * time.Minute
	}

	if len(config.Models) == 0 {
		config.Models = []string{"sqlcoder:7b", "codellama:7b", "llama3.1:8b"}
	}

	return &OllamaProvider{
		config: config,
		client: &http.Client{
			Timeout: config.GenerateTimeout,
		},
		logger: logger,
	}, nil
}

func (p *OllamaProvider) setHTTPClient(client *http.Client) {
	if client != nil {
		p.client = client
	}
}

// GenerateSQL generates SQL from natural language using Ollama
func (p *OllamaProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	prompt := p.buildGeneratePrompt(req)

	response, err := p.callOllama(ctx, req.Model, prompt, defaultOllamaSystemPrompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(response, req)
}

// FixSQL fixes SQL based on error message using Ollama
func (p *OllamaProvider) FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	prompt := p.buildFixPrompt(req)

	response, err := p.callOllama(ctx, req.Model, prompt, defaultOllamaSystemPrompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	return p.parseResponse(response, req)
}

// Chat handles generic conversational interactions using Ollama
func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	systemPrompt := req.System
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant for SQL Studio. Provide concise, accurate answers and offer practical guidance when relevant."
	}

	var promptBuilder strings.Builder
	if req.Context != "" {
		promptBuilder.WriteString("Context:\n")
		promptBuilder.WriteString(req.Context)
		promptBuilder.WriteString("\n\n")
	}
	promptBuilder.WriteString(req.Prompt)

	response, err := p.callOllama(ctx, req.Model, promptBuilder.String(), systemPrompt, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(response.Response)
	if content == "" {
		return nil, fmt.Errorf("empty response from Ollama")
	}

	tokensUsed := response.PromptEvalCount + response.EvalCount
	metadata := map[string]string{
		"model":        response.Model,
		"total_time":   fmt.Sprintf("%d", response.TotalDuration),
		"load_time":    fmt.Sprintf("%d", response.LoadDuration),
		"eval_count":   fmt.Sprintf("%d", response.EvalCount),
		"prompt_count": fmt.Sprintf("%d", response.PromptEvalCount),
	}

	return &ChatResponse{
		Content:    content,
		Provider:   ProviderOllama,
		Model:      req.Model,
		TokensUsed: tokensUsed,
		Metadata:   metadata,
	}, nil
}

// HealthCheck checks if the Ollama service is available
func (p *OllamaProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoint+"/api/tags", nil)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderOllama,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Failed to create request: %v", err),
			LastChecked: time.Now(),
		}, nil
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderOllama,
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
			Provider:     ProviderOllama,
			Status:       "unhealthy",
			Message:      fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			LastChecked:  time.Now(),
			ResponseTime: responseTime,
		}, nil
	}

	return &HealthStatus{
		Provider:     ProviderOllama,
		Status:       "healthy",
		Message:      "Service is operational",
		LastChecked:  time.Now(),
		ResponseTime: responseTime,
	}, nil
}

// GetModels returns available models from Ollama
func (p *OllamaProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var listResp ollamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	var models []ModelInfo
	for _, model := range listResp.Models {
		description := fmt.Sprintf("Ollama %s model", model.Name)
		capabilities := []string{"text-to-sql", "sql-fixing"}

		// Add specific descriptions for known models
		switch {
		case strings.Contains(model.Name, "sqlcoder"):
			description = "SQLCoder - Specialized model for SQL code generation"
			capabilities = append(capabilities, "sql-optimization", "schema-analysis")
		case strings.Contains(model.Name, "codellama"):
			description = "CodeLlama - General purpose code generation model"
			capabilities = append(capabilities, "explanation", "debugging")
		case strings.Contains(model.Name, "llama"):
			description = "Llama - General purpose language model"
			capabilities = append(capabilities, "explanation", "analysis")
		}

		models = append(models, ModelInfo{
			ID:           model.Name,
			Name:         model.Name,
			Provider:     ProviderOllama,
			Description:  description,
			Capabilities: capabilities,
			Metadata: map[string]string{
				"size":           fmt.Sprintf("%d", model.Size),
				"family":         model.Details.Family,
				"parameter_size": model.Details.ParameterSize,
				"format":         model.Details.Format,
			},
		})
	}

	return models, nil
}

// GetProviderType returns the provider type
func (p *OllamaProvider) GetProviderType() Provider {
	return ProviderOllama
}

// IsAvailable checks if the provider is available
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	health, err := p.HealthCheck(ctx)
	return err == nil && health.Status == "healthy"
}

// UpdateConfig updates the provider configuration
func (p *OllamaProvider) UpdateConfig(config interface{}) error {
	ollamaConfig, ok := config.(*OllamaConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Ollama provider")
	}

	if err := p.ValidateConfig(ollamaConfig); err != nil {
		return err
	}

	p.config = ollamaConfig
	return nil
}

// ValidateConfig validates the provider configuration
func (p *OllamaProvider) ValidateConfig(config interface{}) error {
	ollamaConfig, ok := config.(*OllamaConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Ollama provider")
	}

	if ollamaConfig.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	return nil
}

// PullModel pulls a model if it doesn't exist
func (p *OllamaProvider) PullModel(ctx context.Context, modelName string) error {
	if !p.config.AutoPullModels {
		return fmt.Errorf("auto-pull is disabled, please pull model manually: ollama pull %s", modelName)
	}

	p.logger.WithField("model", modelName).Info("Pulling Ollama model")

	requestBody := ollamaPullRequest{
		Name:   modelName,
		Stream: false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	// Use a longer timeout for pulling models
	pullClient := &http.Client{
		Timeout: p.config.PullTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/pull", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := pullClient.Do(req)
	if err != nil {
		return fmt.Errorf("pull request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed with HTTP %d: %s", resp.StatusCode, string(body))
	}

	p.logger.WithField("model", modelName).Info("Model pulled successfully")
	return nil
}

// callOllama makes a request to the Ollama API
func (p *OllamaProvider) callOllama(ctx context.Context, model, prompt string, systemPrompt string, maxTokens int, temperature float64) (*ollamaGenerateResponse, error) {
	// Check if model exists, try to pull if auto-pull is enabled
	models, err := p.GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check available models: %w", err)
	}

	modelExists := false
	for _, m := range models {
		if m.ID == model {
			modelExists = true
			break
		}
	}

	if !modelExists {
		if p.config.AutoPullModels {
			if err := p.PullModel(ctx, model); err != nil {
				return nil, fmt.Errorf("model %s not found and pull failed: %w", model, err)
			}
		} else {
			return nil, fmt.Errorf("model %s not found, please pull it manually: ollama pull %s", model, model)
		}
	}

	options := make(map[string]interface{})
	if temperature > 0 {
		options["temperature"] = temperature
	}
	if maxTokens > 0 {
		options["num_predict"] = maxTokens
	}

	requestBody := ollamaGenerateRequest{
		Model:   model,
		Prompt:  prompt,
		System:  systemPrompt,
		Stream:  false,
		Options: options,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		var errorResp ollamaErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("Ollama API error: %s", errorResp.Error)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &ollamaResp, nil
}

// buildGeneratePrompt builds a prompt for SQL generation
func (p *OllamaProvider) buildGeneratePrompt(req *SQLRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Generate a SQL query for the following request:\n\n")
	prompt.WriteString(fmt.Sprintf("Request: %s\n\n", req.Prompt))

	if req.Schema != "" {
		prompt.WriteString("Database Schema:\n")
		prompt.WriteString(req.Schema)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide your response as JSON with this exact structure:\n")
	prompt.WriteString(`{
  "query": "SELECT ...",
  "explanation": "This query...",
  "confidence": 0.95,
  "suggestions": ["Consider adding an index on...", "..."]
}`)

	prompt.WriteString("\n\nRequirements:\n")
	prompt.WriteString("- Generate syntactically correct SQL\n")
	prompt.WriteString("- Use proper formatting and indentation\n")
	prompt.WriteString("- Include helpful comments for complex queries\n")
	prompt.WriteString("- Provide optimization suggestions\n")
	prompt.WriteString("- Response must be valid JSON only\n")

	return prompt.String()
}

// buildFixPrompt builds a prompt for SQL fixing
func (p *OllamaProvider) buildFixPrompt(req *SQLRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Fix the following SQL query that has an error:\n\n")
	prompt.WriteString(fmt.Sprintf("Original Query:\n```sql\n%s\n```\n\n", req.Query))
	prompt.WriteString(fmt.Sprintf("Error Message:\n%s\n\n", req.Error))

	if req.Schema != "" {
		prompt.WriteString("Database Schema:\n")
		prompt.WriteString(req.Schema)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide your response as JSON with this exact structure:\n")
	prompt.WriteString(`{
  "query": "SELECT ...",
  "explanation": "The error was caused by... Fixed by...",
  "confidence": 0.90,
  "suggestions": ["To prevent this error...", "..."]
}`)

	prompt.WriteString("\n\nRequirements:\n")
	prompt.WriteString("- Fix the syntax or logical error\n")
	prompt.WriteString("- Explain what was wrong\n")
	prompt.WriteString("- Provide prevention suggestions\n")
	prompt.WriteString("- Ensure the fixed query is optimized\n")
	prompt.WriteString("- Response must be valid JSON only\n")

	return prompt.String()
}

// parseResponse parses the Ollama response into SQLResponse
func (p *OllamaProvider) parseResponse(response *ollamaGenerateResponse, req *SQLRequest) (*SQLResponse, error) {
	content := strings.TrimSpace(response.Response)

	// Try to parse as JSON first
	var jsonResp struct {
		Query       string   `json:"query"`
		Explanation string   `json:"explanation"`
		Confidence  float64  `json:"confidence"`
		Suggestions []string `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResp); err == nil {
		tokensUsed := 0
		if response.PromptEvalCount > 0 && response.EvalCount > 0 {
			tokensUsed = response.PromptEvalCount + response.EvalCount
		}

		return &SQLResponse{
			Query:       jsonResp.Query,
			Explanation: jsonResp.Explanation,
			Confidence:  jsonResp.Confidence,
			Suggestions: jsonResp.Suggestions,
			Provider:    ProviderOllama,
			Model:       req.Model,
			TokensUsed:  tokensUsed,
		}, nil
	}

	// If JSON parsing fails, try to extract SQL from the content
	query := p.extractSQL(content)
	if query == "" {
		return nil, fmt.Errorf("could not extract SQL from response: %s", content)
	}

	tokensUsed := 0
	if response.PromptEvalCount > 0 && response.EvalCount > 0 {
		tokensUsed = response.PromptEvalCount + response.EvalCount
	}

	return &SQLResponse{
		Query:       query,
		Explanation: content,
		Confidence:  0.7, // Lower confidence for non-structured response
		Provider:    ProviderOllama,
		Model:       req.Model,
		TokensUsed:  tokensUsed,
	}, nil
}

// extractSQL attempts to extract SQL from unstructured text
func (p *OllamaProvider) extractSQL(content string) string {
	// Look for SQL code blocks
	if start := strings.Index(content, "```sql"); start != -1 {
		start += 6
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// Look for generic code blocks
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

	// Look for SQL-like patterns
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
func (p *OllamaProvider) looksLikeSQL(text string) bool {
	text = strings.ToUpper(strings.TrimSpace(text))
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "WITH"}

	for _, keyword := range sqlKeywords {
		if strings.HasPrefix(text, keyword) {
			return true
		}
	}
	return false
}
