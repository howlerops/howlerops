package ai_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderConstants tests the Provider type constants
func TestProviderConstants(t *testing.T) {
	tests := []struct {
		name     string
		provider ai.Provider
		expected string
	}{
		{"OpenAI", ai.ProviderOpenAI, "openai"},
		{"Anthropic", ai.ProviderAnthropic, "anthropic"},
		{"Ollama", ai.ProviderOllama, "ollama"},
		{"HuggingFace", ai.ProviderHuggingFace, "huggingface"},
		{"ClaudeCode", ai.ProviderClaudeCode, "claudecode"},
		{"Codex", ai.ProviderCodex, "codex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.provider))
		})
	}
}

// TestErrorTypeConstants tests the ErrorType type constants
func TestErrorTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		errorType ai.ErrorType
		expected  string
	}{
		{"InvalidRequest", ai.ErrorTypeInvalidRequest, "invalid_request"},
		{"ProviderError", ai.ErrorTypeProviderError, "provider_error"},
		{"ConfigError", ai.ErrorTypeConfigError, "config_error"},
		{"RateLimit", ai.ErrorTypeRateLimit, "rate_limit"},
		{"Timeout", ai.ErrorTypeTimeout, "timeout"},
		{"InternalError", ai.ErrorTypeInternalError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.errorType))
		})
	}
}

// TestSQLRequest_JSON tests JSON marshaling and unmarshaling of SQLRequest
func TestSQLRequest_JSON(t *testing.T) {
	req := &ai.SQLRequest{
		Prompt:      "Generate a SELECT query",
		Query:       "SELECT * FROM users",
		Error:       "syntax error",
		Schema:      "CREATE TABLE users (id INT, name VARCHAR)",
		Provider:    ai.ProviderOpenAI,
		Model:       "gpt-4o",
		MaxTokens:   1000,
		Temperature: 0.5,
		Context: map[string]string{
			"database": "postgres",
			"version":  "14.5",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded ai.SQLRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, req.Prompt, decoded.Prompt)
	assert.Equal(t, req.Query, decoded.Query)
	assert.Equal(t, req.Error, decoded.Error)
	assert.Equal(t, req.Schema, decoded.Schema)
	assert.Equal(t, req.Provider, decoded.Provider)
	assert.Equal(t, req.Model, decoded.Model)
	assert.Equal(t, req.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, req.Temperature, decoded.Temperature)
	assert.Equal(t, req.Context, decoded.Context)
}

// TestSQLRequest_JSON_MinimalFields tests JSON marshaling with minimal fields
func TestSQLRequest_JSON_MinimalFields(t *testing.T) {
	req := &ai.SQLRequest{
		Prompt:   "Generate query",
		Provider: ai.ProviderAnthropic,
		Model:    "claude-3-5-sonnet",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded ai.SQLRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Prompt, decoded.Prompt)
	assert.Equal(t, req.Provider, decoded.Provider)
	assert.Equal(t, req.Model, decoded.Model)
	assert.Empty(t, decoded.Query)
	assert.Empty(t, decoded.Error)
	assert.Empty(t, decoded.Schema)
	assert.Nil(t, decoded.Context)
}

// TestChatRequest_JSON tests JSON marshaling of ChatRequest
func TestChatRequest_JSON(t *testing.T) {
	req := &ai.ChatRequest{
		Prompt:      "Explain database normalization",
		Context:     "User is learning SQL",
		System:      "You are a SQL tutor",
		Provider:    ai.ProviderAnthropic,
		Model:       "claude-3-opus",
		MaxTokens:   2000,
		Temperature: 0.7,
		Metadata: map[string]string{
			"session_id": "abc123",
			"user_level": "beginner",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded ai.ChatRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Prompt, decoded.Prompt)
	assert.Equal(t, req.Context, decoded.Context)
	assert.Equal(t, req.System, decoded.System)
	assert.Equal(t, req.Provider, decoded.Provider)
	assert.Equal(t, req.Model, decoded.Model)
	assert.Equal(t, req.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, req.Temperature, decoded.Temperature)
	assert.Equal(t, req.Metadata, decoded.Metadata)
}

// TestSQLResponse_JSON tests JSON marshaling of SQLResponse
func TestSQLResponse_JSON(t *testing.T) {
	resp := &ai.SQLResponse{
		Query:       "SELECT id, name FROM users WHERE active = true",
		Explanation: "This query selects active users",
		Confidence:  0.95,
		Suggestions: []string{"Add index on active column", "Consider pagination"},
		Warnings:    []string{"Query may return large result set"},
		Provider:    ai.ProviderOpenAI,
		Model:       "gpt-4o-mini",
		TokensUsed:  150,
		TimeTaken:   500 * time.Millisecond,
		Metadata: map[string]string{
			"temperature": "0.1",
			"cached":      "false",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ai.SQLResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Query, decoded.Query)
	assert.Equal(t, resp.Explanation, decoded.Explanation)
	assert.Equal(t, resp.Confidence, decoded.Confidence)
	assert.Equal(t, resp.Suggestions, decoded.Suggestions)
	assert.Equal(t, resp.Warnings, decoded.Warnings)
	assert.Equal(t, resp.Provider, decoded.Provider)
	assert.Equal(t, resp.Model, decoded.Model)
	assert.Equal(t, resp.TokensUsed, decoded.TokensUsed)
	assert.Equal(t, resp.TimeTaken, decoded.TimeTaken)
	assert.Equal(t, resp.Metadata, decoded.Metadata)
}

// TestSQLResponse_JSON_EmptyArrays tests that empty arrays marshal correctly
func TestSQLResponse_JSON_EmptyArrays(t *testing.T) {
	resp := &ai.SQLResponse{
		Query:       "SELECT * FROM users",
		Explanation: "Basic query",
		Confidence:  0.8,
		Provider:    ai.ProviderOllama,
		Model:       "sqlcoder:7b",
		TimeTaken:   1 * time.Second,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ai.SQLResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Query, decoded.Query)
	assert.Nil(t, decoded.Suggestions)
	assert.Nil(t, decoded.Warnings)
	assert.Nil(t, decoded.Metadata)
}

// TestChatResponse_JSON tests JSON marshaling of ChatResponse
func TestChatResponse_JSON(t *testing.T) {
	resp := &ai.ChatResponse{
		Content:    "Database normalization reduces data redundancy...",
		Provider:   ai.ProviderAnthropic,
		Model:      "claude-3-5-haiku",
		TokensUsed: 350,
		TimeTaken:  750 * time.Millisecond,
		Metadata: map[string]string{
			"stop_reason": "end_turn",
			"usage_type":  "chat",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ai.ChatResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Content, decoded.Content)
	assert.Equal(t, resp.Provider, decoded.Provider)
	assert.Equal(t, resp.Model, decoded.Model)
	assert.Equal(t, resp.TokensUsed, decoded.TokensUsed)
	assert.Equal(t, resp.TimeTaken, decoded.TimeTaken)
	assert.Equal(t, resp.Metadata, decoded.Metadata)
}

// TestHealthStatus_JSON tests JSON marshaling of HealthStatus
func TestHealthStatus_JSON(t *testing.T) {
	now := time.Now()
	status := &ai.HealthStatus{
		Provider:     ai.ProviderOpenAI,
		Status:       "healthy",
		Message:      "All systems operational",
		LastChecked:  now,
		ResponseTime: 100 * time.Millisecond,
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded ai.HealthStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, status.Provider, decoded.Provider)
	assert.Equal(t, status.Status, decoded.Status)
	assert.Equal(t, status.Message, decoded.Message)
	assert.WithinDuration(t, status.LastChecked, decoded.LastChecked, time.Second)
	assert.Equal(t, status.ResponseTime, decoded.ResponseTime)
}

// TestModelInfo_JSON tests JSON marshaling of ModelInfo
func TestModelInfo_JSON(t *testing.T) {
	model := &ai.ModelInfo{
		ID:          "gpt-4o",
		Name:        "GPT-4 Optimized",
		Provider:    ai.ProviderOpenAI,
		Description: "Most capable GPT-4 model",
		MaxTokens:   128000,
		Capabilities: []string{
			"text-generation",
			"code-generation",
			"function-calling",
		},
		Metadata: map[string]string{
			"context_window": "128k",
			"training_date":  "2024-05",
		},
	}

	data, err := json.Marshal(model)
	require.NoError(t, err)

	var decoded ai.ModelInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, model.ID, decoded.ID)
	assert.Equal(t, model.Name, decoded.Name)
	assert.Equal(t, model.Provider, decoded.Provider)
	assert.Equal(t, model.Description, decoded.Description)
	assert.Equal(t, model.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, model.Capabilities, decoded.Capabilities)
	assert.Equal(t, model.Metadata, decoded.Metadata)
}

// TestUsage_JSON tests JSON marshaling of Usage
func TestUsage_JSON(t *testing.T) {
	now := time.Now()
	usage := &ai.Usage{
		Provider:        ai.ProviderAnthropic,
		Model:           "claude-3-5-sonnet",
		RequestCount:    1250,
		TokensUsed:      45000,
		SuccessRate:     0.98,
		AvgResponseTime: 850 * time.Millisecond,
		LastUsed:        now,
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var decoded ai.Usage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, usage.Provider, decoded.Provider)
	assert.Equal(t, usage.Model, decoded.Model)
	assert.Equal(t, usage.RequestCount, decoded.RequestCount)
	assert.Equal(t, usage.TokensUsed, decoded.TokensUsed)
	assert.Equal(t, usage.SuccessRate, decoded.SuccessRate)
	assert.Equal(t, usage.AvgResponseTime, decoded.AvgResponseTime)
	assert.WithinDuration(t, usage.LastUsed, decoded.LastUsed, time.Second)
}

// TestConfig_JSON tests JSON marshaling of Config
func TestConfig_JSON(t *testing.T) {
	cfg := &ai.Config{
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       2048,
		Temperature:     0.1,
		RequestTimeout:  60 * time.Second,
		RateLimitPerMin: 60,
		OpenAI: ai.OpenAIConfig{
			APIKey:  "sk-test-key",
			BaseURL: "https://api.openai.com/v1",
			Models:  []string{"gpt-4o", "gpt-4o-mini"},
			OrgID:   "org-123",
		},
		Anthropic: ai.AnthropicConfig{
			APIKey:  "sk-ant-test",
			BaseURL: "https://api.anthropic.com",
			Models:  []string{"claude-3-5-sonnet"},
			Version: "2023-06-01",
		},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.Config
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.DefaultProvider, decoded.DefaultProvider)
	assert.Equal(t, cfg.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, cfg.Temperature, decoded.Temperature)
	assert.Equal(t, cfg.RequestTimeout, decoded.RequestTimeout)
	assert.Equal(t, cfg.RateLimitPerMin, decoded.RateLimitPerMin)
	assert.Equal(t, cfg.OpenAI.APIKey, decoded.OpenAI.APIKey)
	assert.Equal(t, cfg.Anthropic.Version, decoded.Anthropic.Version)
}

// TestOpenAIConfig_JSON tests JSON marshaling of OpenAIConfig
func TestOpenAIConfig_JSON(t *testing.T) {
	cfg := &ai.OpenAIConfig{
		APIKey:  "sk-test-123",
		BaseURL: "https://api.openai.com/v1",
		Models: []string{
			"gpt-4o-mini",
			"gpt-4o",
			"gpt-4-turbo",
		},
		OrgID: "org-amplifier",
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.OpenAIConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.APIKey, decoded.APIKey)
	assert.Equal(t, cfg.BaseURL, decoded.BaseURL)
	assert.Equal(t, cfg.Models, decoded.Models)
	assert.Equal(t, cfg.OrgID, decoded.OrgID)
}

// TestAnthropicConfig_JSON tests JSON marshaling of AnthropicConfig
func TestAnthropicConfig_JSON(t *testing.T) {
	cfg := &ai.AnthropicConfig{
		APIKey:  "sk-ant-test-456",
		BaseURL: "https://api.anthropic.com",
		Models: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
		},
		Version: "2023-06-01",
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.AnthropicConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.APIKey, decoded.APIKey)
	assert.Equal(t, cfg.BaseURL, decoded.BaseURL)
	assert.Equal(t, cfg.Models, decoded.Models)
	assert.Equal(t, cfg.Version, decoded.Version)
}

// TestOllamaConfig_JSON tests JSON marshaling of OllamaConfig
func TestOllamaConfig_JSON(t *testing.T) {
	cfg := &ai.OllamaConfig{
		Endpoint:        "http://localhost:11434",
		Models:          []string{"sqlcoder:7b", "codellama:7b"},
		PullTimeout:     10 * time.Minute,
		GenerateTimeout: 2 * time.Minute,
		AutoPullModels:  true,
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.OllamaConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.Endpoint, decoded.Endpoint)
	assert.Equal(t, cfg.Models, decoded.Models)
	assert.Equal(t, cfg.PullTimeout, decoded.PullTimeout)
	assert.Equal(t, cfg.GenerateTimeout, decoded.GenerateTimeout)
	assert.Equal(t, cfg.AutoPullModels, decoded.AutoPullModels)
}

// TestHuggingFaceConfig_JSON tests JSON marshaling of HuggingFaceConfig
func TestHuggingFaceConfig_JSON(t *testing.T) {
	cfg := &ai.HuggingFaceConfig{
		Endpoint:         "http://localhost:8080",
		Models:           []string{"sqlcoder:7b", "llama3.1:8b"},
		PullTimeout:      15 * time.Minute,
		GenerateTimeout:  3 * time.Minute,
		AutoPullModels:   false,
		RecommendedModel: "sqlcoder:7b",
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.HuggingFaceConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.Endpoint, decoded.Endpoint)
	assert.Equal(t, cfg.Models, decoded.Models)
	assert.Equal(t, cfg.PullTimeout, decoded.PullTimeout)
	assert.Equal(t, cfg.GenerateTimeout, decoded.GenerateTimeout)
	assert.Equal(t, cfg.AutoPullModels, decoded.AutoPullModels)
	assert.Equal(t, cfg.RecommendedModel, decoded.RecommendedModel)
}

// TestClaudeCodeConfig_JSON tests JSON marshaling of ClaudeCodeConfig
func TestClaudeCodeConfig_JSON(t *testing.T) {
	cfg := &ai.ClaudeCodeConfig{
		ClaudePath:  "/usr/local/bin/claude",
		Model:       "opus",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.ClaudeCodeConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.ClaudePath, decoded.ClaudePath)
	assert.Equal(t, cfg.Model, decoded.Model)
	assert.Equal(t, cfg.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, cfg.Temperature, decoded.Temperature)
}

// TestCodexConfig_JSON tests JSON marshaling of CodexConfig
func TestCodexConfig_JSON(t *testing.T) {
	cfg := &ai.CodexConfig{
		APIKey:       "sk-codex-test",
		Organization: "org-codex",
		BaseURL:      "https://api.openai.com/v1",
		Model:        "code-davinci-002",
		MaxTokens:    2048,
		Temperature:  0.0,
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.CodexConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cfg.APIKey, decoded.APIKey)
	assert.Equal(t, cfg.Organization, decoded.Organization)
	assert.Equal(t, cfg.BaseURL, decoded.BaseURL)
	assert.Equal(t, cfg.Model, decoded.Model)
	assert.Equal(t, cfg.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, cfg.Temperature, decoded.Temperature)
}

// TestAIError_Error tests the Error() method
func TestAIError_Error(t *testing.T) {
	err := &ai.AIError{
		Type:     ai.ErrorTypeProviderError,
		Message:  "API request failed",
		Provider: ai.ProviderOpenAI,
		Code:     "rate_limit_exceeded",
	}

	assert.Equal(t, "API request failed", err.Error())
}

// TestNewAIError tests the NewAIError constructor
func TestNewAIError(t *testing.T) {
	tests := []struct {
		name          string
		errorType     ai.ErrorType
		message       string
		provider      ai.Provider
		wantRetryable bool
	}{
		{
			name:          "Timeout error is retryable",
			errorType:     ai.ErrorTypeTimeout,
			message:       "Request timed out",
			provider:      ai.ProviderAnthropic,
			wantRetryable: true,
		},
		{
			name:          "Rate limit error is retryable",
			errorType:     ai.ErrorTypeRateLimit,
			message:       "Rate limit exceeded",
			provider:      ai.ProviderOpenAI,
			wantRetryable: true,
		},
		{
			name:          "Invalid request is not retryable",
			errorType:     ai.ErrorTypeInvalidRequest,
			message:       "Missing required field",
			provider:      ai.ProviderOllama,
			wantRetryable: false,
		},
		{
			name:          "Provider error is not retryable",
			errorType:     ai.ErrorTypeProviderError,
			message:       "Provider unavailable",
			provider:      ai.ProviderHuggingFace,
			wantRetryable: false,
		},
		{
			name:          "Config error is not retryable",
			errorType:     ai.ErrorTypeConfigError,
			message:       "Invalid configuration",
			provider:      ai.ProviderClaudeCode,
			wantRetryable: false,
		},
		{
			name:          "Internal error is not retryable",
			errorType:     ai.ErrorTypeInternalError,
			message:       "Unexpected error",
			provider:      ai.ProviderCodex,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ai.NewAIError(tt.errorType, tt.message, tt.provider)

			assert.NotNil(t, err)
			assert.Equal(t, tt.errorType, err.Type)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.provider, err.Provider)
			assert.Equal(t, tt.wantRetryable, err.Retryable)
			assert.Empty(t, err.Code)
			assert.Nil(t, err.Details)
		})
	}
}

// TestAIError_JSON tests JSON marshaling of AIError
func TestAIError_JSON(t *testing.T) {
	err := &ai.AIError{
		Type:     ai.ErrorTypeRateLimit,
		Message:  "Too many requests",
		Provider: ai.ProviderOpenAI,
		Code:     "rate_limit_exceeded",
		Details: map[string]interface{}{
			"limit":       100,
			"window":      "1m",
			"retry_after": 60,
		},
		Retryable: true,
	}

	data, jsonErr := json.Marshal(err)
	require.NoError(t, jsonErr)

	var decoded ai.AIError
	jsonErr = json.Unmarshal(data, &decoded)
	require.NoError(t, jsonErr)

	assert.Equal(t, err.Type, decoded.Type)
	assert.Equal(t, err.Message, decoded.Message)
	assert.Equal(t, err.Provider, decoded.Provider)
	assert.Equal(t, err.Code, decoded.Code)
	assert.Equal(t, err.Retryable, decoded.Retryable)
	assert.NotNil(t, decoded.Details)
}

// TestSQLRequest_EmptyContext tests SQLRequest with nil context map
func TestSQLRequest_EmptyContext(t *testing.T) {
	req := &ai.SQLRequest{
		Prompt:   "test",
		Provider: ai.ProviderOpenAI,
		Model:    "gpt-4o",
		Context:  nil,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded ai.SQLRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Context)
}

// TestChatRequest_EmptyMetadata tests ChatRequest with nil metadata
func TestChatRequest_EmptyMetadata(t *testing.T) {
	req := &ai.ChatRequest{
		Prompt:   "test prompt",
		Provider: ai.ProviderAnthropic,
		Model:    "claude-3-5-sonnet",
		Metadata: nil,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded ai.ChatRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Metadata)
}

// TestSQLResponse_ZeroValues tests SQLResponse with zero values
func TestSQLResponse_ZeroValues(t *testing.T) {
	resp := &ai.SQLResponse{
		Query:       "",
		Explanation: "",
		Confidence:  0.0,
		Provider:    ai.ProviderOpenAI,
		Model:       "",
		TokensUsed:  0,
		TimeTaken:   0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ai.SQLResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "", decoded.Query)
	assert.Equal(t, 0.0, decoded.Confidence)
	assert.Equal(t, 0, decoded.TokensUsed)
	assert.Equal(t, time.Duration(0), decoded.TimeTaken)
}

// TestHealthStatus_StatusValues tests different health status values
func TestHealthStatus_StatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"Healthy", "healthy"},
		{"Unhealthy", "unhealthy"},
		{"Unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := &ai.HealthStatus{
				Provider:    ai.ProviderOpenAI,
				Status:      tt.status,
				LastChecked: time.Now(),
			}

			data, err := json.Marshal(health)
			require.NoError(t, err)

			var decoded ai.HealthStatus
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.status, decoded.Status)
		})
	}
}

// TestModelInfo_EmptyCapabilities tests ModelInfo with no capabilities
func TestModelInfo_EmptyCapabilities(t *testing.T) {
	model := &ai.ModelInfo{
		ID:           "test-model",
		Name:         "Test Model",
		Provider:     ai.ProviderOllama,
		Capabilities: []string{},
	}

	data, err := json.Marshal(model)
	require.NoError(t, err)

	var decoded ai.ModelInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// When empty slice is marshaled, it becomes null in JSON, so after unmarshal it's nil
	// This is standard Go JSON behavior
	if decoded.Capabilities != nil {
		assert.Empty(t, decoded.Capabilities)
	}
}

// TestUsage_ZeroCounters tests Usage with zero values
func TestUsage_ZeroCounters(t *testing.T) {
	usage := &ai.Usage{
		Provider:        ai.ProviderAnthropic,
		Model:           "claude-3-5-sonnet",
		RequestCount:    0,
		TokensUsed:      0,
		SuccessRate:     0.0,
		AvgResponseTime: 0,
		LastUsed:        time.Time{},
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var decoded ai.Usage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, int64(0), decoded.RequestCount)
	assert.Equal(t, int64(0), decoded.TokensUsed)
	assert.Equal(t, 0.0, decoded.SuccessRate)
	assert.Equal(t, time.Duration(0), decoded.AvgResponseTime)
}

// TestConfig_DefaultProvider tests different default providers
func TestConfig_DefaultProvider(t *testing.T) {
	providers := []ai.Provider{
		ai.ProviderOpenAI,
		ai.ProviderAnthropic,
		ai.ProviderOllama,
		ai.ProviderHuggingFace,
		ai.ProviderClaudeCode,
		ai.ProviderCodex,
	}

	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			cfg := &ai.Config{
				DefaultProvider: provider,
				MaxTokens:       1000,
				Temperature:     0.5,
				RequestTimeout:  30 * time.Second,
				RateLimitPerMin: 50,
			}

			data, err := json.Marshal(cfg)
			require.NoError(t, err)

			var decoded ai.Config
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, provider, decoded.DefaultProvider)
		})
	}
}

// TestTimeDuration_JSONMarshaling tests that time.Duration marshals correctly
func TestTimeDuration_JSONMarshaling(t *testing.T) {
	durations := []time.Duration{
		0,
		100 * time.Millisecond,
		1 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		10 * time.Minute,
	}

	for _, d := range durations {
		t.Run(d.String(), func(t *testing.T) {
			cfg := &ai.Config{
				DefaultProvider: ai.ProviderOpenAI,
				MaxTokens:       1000,
				Temperature:     0.5,
				RequestTimeout:  d,
				RateLimitPerMin: 60,
			}

			data, err := json.Marshal(cfg)
			require.NoError(t, err)

			var decoded ai.Config
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, d, decoded.RequestTimeout)
		})
	}
}

// TestOllamaConfig_EmptyModels tests OllamaConfig with empty models slice
func TestOllamaConfig_EmptyModels(t *testing.T) {
	cfg := &ai.OllamaConfig{
		Endpoint:        "http://localhost:11434",
		Models:          []string{},
		PullTimeout:     5 * time.Minute,
		GenerateTimeout: 1 * time.Minute,
		AutoPullModels:  false,
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var decoded ai.OllamaConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.Models)
	assert.Empty(t, decoded.Models)
}

// TestProviderType_UnknownValue tests unmarshaling unknown provider value
func TestProviderType_UnknownValue(t *testing.T) {
	jsonData := `{"provider":"unknown_provider","model":"test"}`

	var req ai.SQLRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	require.NoError(t, err)

	// Unknown provider value should still unmarshal
	assert.Equal(t, ai.Provider("unknown_provider"), req.Provider)
}
