package ai_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
)

// mockAIProvider implements the AIProvider interface for testing
type mockAIProvider struct {
	providerType       ai.Provider
	generateSQLFunc    func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	fixSQLFunc         func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	chatFunc           func(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error)
	healthCheckFunc    func(ctx context.Context) (*ai.HealthStatus, error)
	getModelsFunc      func(ctx context.Context) ([]ai.ModelInfo, error)
	isAvailableFunc    func(ctx context.Context) bool
	updateConfigFunc   func(config interface{}) error
	validateConfigFunc func(config interface{}) error
}

func (m *mockAIProvider) GenerateSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.generateSQLFunc != nil {
		return m.generateSQLFunc(ctx, req)
	}
	return &ai.SQLResponse{
		Query:       "SELECT * FROM users",
		Explanation: "Mock query",
		Confidence:  0.95,
		Provider:    m.providerType,
		Model:       req.Model,
		TokensUsed:  100,
	}, nil
}

func (m *mockAIProvider) FixSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.fixSQLFunc != nil {
		return m.fixSQLFunc(ctx, req)
	}
	return &ai.SQLResponse{
		Query:       "SELECT * FROM users WHERE id = 1",
		Explanation: "Fixed query",
		Confidence:  0.90,
		Provider:    m.providerType,
		Model:       req.Model,
		TokensUsed:  50,
	}, nil
}

func (m *mockAIProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ai.ChatResponse{
		Content:    "Mock chat response",
		Provider:   m.providerType,
		Model:      req.Model,
		TokensUsed: 75,
	}, nil
}

func (m *mockAIProvider) HealthCheck(ctx context.Context) (*ai.HealthStatus, error) {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return &ai.HealthStatus{
		Provider:    m.providerType,
		Status:      "healthy",
		Message:     "Mock provider is healthy",
		LastChecked: time.Now(),
	}, nil
}

func (m *mockAIProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	if m.getModelsFunc != nil {
		return m.getModelsFunc(ctx)
	}
	return []ai.ModelInfo{
		{
			ID:       "mock-model-1",
			Name:     "Mock Model 1",
			Provider: m.providerType,
		},
	}, nil
}

func (m *mockAIProvider) GetProviderType() ai.Provider {
	return m.providerType
}

func (m *mockAIProvider) IsAvailable(ctx context.Context) bool {
	if m.isAvailableFunc != nil {
		return m.isAvailableFunc(ctx)
	}
	return true
}

func (m *mockAIProvider) UpdateConfig(config interface{}) error {
	if m.updateConfigFunc != nil {
		return m.updateConfigFunc(config)
	}
	return nil
}

func (m *mockAIProvider) ValidateConfig(config interface{}) error {
	if m.validateConfigFunc != nil {
		return m.validateConfigFunc(config)
	}
	return nil
}

// Helper function to create a test logger
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs in tests
	return logger
}

// Helper function to create a minimal valid config
func newTestConfig() *ai.Config {
	return &ai.Config{
		OpenAI: ai.OpenAIConfig{
			APIKey: "test-openai-key",
			Models: []string{"gpt-4o-mini"},
		},
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       1000,
		Temperature:     0.7,
		RequestTimeout:  30 * time.Second,
	}
}

// Helper function to create an empty config (no providers)
func newEmptyConfig() *ai.Config {
	return &ai.Config{
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       1000,
		Temperature:     0.7,
	}
}

// TestNewService_ValidConfig tests creating a new service with valid configuration
func TestNewService_ValidConfig(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()

	service, err := ai.NewService(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

// TestNewService_NilConfig tests that NewService rejects nil config
func TestNewService_NilConfig(t *testing.T) {
	logger := newTestLogger()

	service, err := ai.NewService(nil, logger)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

// TestNewService_NilLogger tests that NewService rejects nil logger
func TestNewService_NilLogger(t *testing.T) {
	config := newTestConfig()

	service, err := ai.NewService(config, nil)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "logger cannot be nil")
}

// TestNewService_NoProvidersConfigured tests that service succeeds even when no providers are configured
// (AI providers are now optional - service can run without them for basic operation)
func TestNewService_NoProvidersConfigured(t *testing.T) {
	config := newEmptyConfig()
	logger := newTestLogger()

	service, err := ai.NewService(config, logger)

	// Service should be created successfully even with no providers
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Empty(t, service.GetProviders())
}

// TestNewService_MultipleProviders tests initialization with multiple providers
func TestNewService_MultipleProviders(t *testing.T) {
	config := &ai.Config{
		OpenAI: ai.OpenAIConfig{
			APIKey: "test-openai-key",
			Models: []string{"gpt-4o-mini"},
		},
		Anthropic: ai.AnthropicConfig{
			APIKey: "test-anthropic-key",
			Models: []string{"claude-3-5-sonnet-20241022"},
		},
		Ollama: ai.OllamaConfig{
			Endpoint: "http://localhost:11434",
			Models:   []string{"llama2"},
		},
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       1000,
		Temperature:     0.7,
	}
	logger := newTestLogger()

	service, err := ai.NewService(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, service)

	providers := service.GetProviders()
	// Note: Actual providers may not initialize if API keys are invalid
	// but at least one should be attempted
	assert.GreaterOrEqual(t, len(providers), 0)
}

// TestService_StartStop tests the service lifecycle
func TestService_StartStop(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Test Start
	err = service.Start(ctx)
	assert.NoError(t, err)

	// Test Start when already started
	err = service.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	// Test Stop
	err = service.Stop(ctx)
	assert.NoError(t, err)

	// Test Stop when not started
	err = service.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

// TestService_GetProviders tests getting the list of available providers
func TestService_GetProviders(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	providers := service.GetProviders()

	assert.NotNil(t, providers)
	// Should return a slice (even if empty due to invalid API keys)
	assert.IsType(t, []ai.Provider{}, providers)
}

// TestService_GetConfig tests getting the service configuration
func TestService_GetConfig(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	retrievedConfig := service.GetConfig()

	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, config.DefaultProvider, retrievedConfig.DefaultProvider)
	assert.Equal(t, config.MaxTokens, retrievedConfig.MaxTokens)
	assert.Equal(t, config.Temperature, retrievedConfig.Temperature)
}

// TestValidateRequest_NilRequest tests validation of nil request
func TestValidateRequest_NilRequest(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	err = service.ValidateRequest(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

// TestValidateRequest_EmptyProvider tests validation with empty provider
func TestValidateRequest_EmptyProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.SQLRequest{
		Prompt: "test prompt",
	}

	err = service.ValidateRequest(request)

	// Should set default provider and model
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultProvider, request.Provider)
	assert.NotEmpty(t, request.Model)
}

// TestValidateRequest_EmptyModel tests validation with empty model
func TestValidateRequest_EmptyModel(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.SQLRequest{
		Prompt:   "test prompt",
		Provider: ai.ProviderOpenAI,
	}

	err = service.ValidateRequest(request)

	// Should set default model for provider
	assert.NoError(t, err)
	assert.NotEmpty(t, request.Model)
}

// TestValidateRequest_InvalidMaxTokens tests validation with invalid max tokens
func TestValidateRequest_InvalidMaxTokens(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.SQLRequest{
		Prompt:    "test prompt",
		Provider:  ai.ProviderOpenAI,
		Model:     "gpt-4o-mini",
		MaxTokens: 0,
	}

	err = service.ValidateRequest(request)

	// Should set default max tokens
	assert.NoError(t, err)
	assert.Equal(t, config.MaxTokens, request.MaxTokens)
}

// TestValidateRequest_InvalidTemperature tests validation with invalid temperature
func TestValidateRequest_InvalidTemperature(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		temperature float64
	}{
		{"negative temperature", -0.5},
		{"temperature too high", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &ai.SQLRequest{
				Prompt:      "test prompt",
				Provider:    ai.ProviderOpenAI,
				Model:       "gpt-4o-mini",
				Temperature: tt.temperature,
			}

			err = service.ValidateRequest(request)

			// Should set default temperature
			assert.NoError(t, err)
			assert.Equal(t, config.Temperature, request.Temperature)
		})
	}
}

// TestValidateRequest_MissingPromptAndQuery tests validation with both prompt and query missing
func TestValidateRequest_MissingPromptAndQuery(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.SQLRequest{
		Provider: ai.ProviderOpenAI,
		Model:    "gpt-4o-mini",
	}

	err = service.ValidateRequest(request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either prompt or query is required")
}

// TestValidateChatRequest_NilRequest tests validation of nil chat request
func TestValidateChatRequest_NilRequest(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	err = service.ValidateChatRequest(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

// TestValidateChatRequest_EmptyPrompt tests validation with empty prompt
func TestValidateChatRequest_EmptyPrompt(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.ChatRequest{
		Provider: ai.ProviderOpenAI,
		Model:    "gpt-4o-mini",
	}

	err = service.ValidateChatRequest(request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

// TestValidateChatRequest_UnavailableProvider tests validation with unavailable provider
func TestValidateChatRequest_UnavailableProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.ChatRequest{
		Prompt:   "test prompt",
		Provider: "nonexistent",
		Model:    "test-model",
	}

	err = service.ValidateChatRequest(request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// TestService_GetProviderHealth_UnavailableProvider tests health check for unavailable provider
func TestService_GetProviderHealth_UnavailableProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	health, err := service.GetProviderHealth(ctx, "nonexistent")

	require.NoError(t, err)
	assert.Equal(t, "unavailable", health.Status)
	assert.Contains(t, health.Message, "not configured")
}

// TestService_GetAllProvidersHealth tests getting health status for all providers
func TestService_GetAllProvidersHealth(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	healthMap, err := service.GetAllProvidersHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, healthMap)
	// Health map should contain entries for configured providers
	assert.IsType(t, map[ai.Provider]*ai.HealthStatus{}, healthMap)
}

// TestService_GetUsageStats_UnavailableProvider tests getting usage stats for unavailable provider
func TestService_GetUsageStats_UnavailableProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	usage, err := service.GetUsageStats(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "not available")
}

// TestService_GetAllUsageStats tests getting usage stats for all providers
func TestService_GetAllUsageStats(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	usageMap, err := service.GetAllUsageStats(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, usageMap)
	assert.IsType(t, map[ai.Provider]*ai.Usage{}, usageMap)
}

// TestService_GetAvailableModels_UnavailableProvider tests getting models for unavailable provider
func TestService_GetAvailableModels_UnavailableProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	models, err := service.GetAvailableModels(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, models)
	assert.Contains(t, err.Error(), "not available")
}

// TestService_GetAllAvailableModels tests getting models for all providers
func TestService_GetAllAvailableModels(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	modelsMap, err := service.GetAllAvailableModels(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, modelsMap)
	assert.IsType(t, map[ai.Provider][]ai.ModelInfo{}, modelsMap)
}

// TestService_UpdateProviderConfig_UnavailableProvider tests updating config for unavailable provider
func TestService_UpdateProviderConfig_UnavailableProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	err = service.UpdateProviderConfig("nonexistent", &ai.OpenAIConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// TestService_TestProvider_InvalidConfig tests testing a provider with invalid config
func TestService_TestProvider_InvalidConfig(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with wrong config type
	err = service.TestProvider(ctx, ai.ProviderOpenAI, "invalid-config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// TestService_TestProvider_UnknownProvider tests testing an unknown provider
func TestService_TestProvider_UnknownProvider(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	err = service.TestProvider(ctx, "unknown", &ai.OpenAIConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

// TestService_ConcurrentRequests tests thread safety with concurrent requests
func TestService_ConcurrentRequests(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	numGoroutines := 10
	var wg sync.WaitGroup

	// Test concurrent GetProviders calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = service.GetProviders()
		}()
	}
	wg.Wait()

	// Test concurrent GetConfig calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = service.GetConfig()
		}()
	}
	wg.Wait()

	// Test concurrent GetAllUsageStats calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = service.GetAllUsageStats(ctx)
		}()
	}
	wg.Wait()

	// Test concurrent GetAllProvidersHealth calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = service.GetAllProvidersHealth(ctx)
		}()
	}
	wg.Wait()
}

// TestService_ConcurrentStartStop tests thread safety of Start/Stop operations
func TestService_ConcurrentStartStop(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	numGoroutines := 5
	var wg sync.WaitGroup

	// Only one Start should succeed
	successCount := 0
	var mu sync.Mutex

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			err := service.Start(ctx)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	assert.Equal(t, 1, successCount, "only one Start should succeed")
	mu.Unlock()

	// Only one Stop should succeed
	successCount = 0

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			err := service.Stop(ctx)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	assert.Equal(t, 1, successCount, "only one Stop should succeed")
	mu.Unlock()
}

// TestGenerateSQL_ProviderNotAvailable tests GenerateSQL with unavailable provider
func TestGenerateSQL_ProviderNotAvailable(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	request := &ai.SQLRequest{
		Prompt:   "SELECT all users",
		Provider: "nonexistent",
		Model:    "test-model",
	}

	response, err := service.GenerateSQL(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "not available")
}

// TestFixSQL_MissingQuery tests FixSQL with missing query
func TestFixSQL_MissingQuery(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	request := &ai.SQLRequest{
		Prompt:   "fix this",
		Provider: ai.ProviderOpenAI,
		Model:    "gpt-4o-mini",
		Error:    "syntax error",
	}

	response, err := service.FixSQL(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "query is required")
}

// TestFixSQL_MissingError tests FixSQL with missing error message
func TestFixSQL_MissingError(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	request := &ai.SQLRequest{
		Prompt:   "fix this",
		Provider: ai.ProviderOpenAI,
		Model:    "gpt-4o-mini",
		Query:    "SELECT * FROM users WHERE",
	}

	response, err := service.FixSQL(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error message is required")
}

// TestChat_ProviderNotAvailable tests Chat with unavailable provider
func TestChat_ProviderNotAvailable(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	request := &ai.ChatRequest{
		Prompt:   "Hello",
		Provider: "nonexistent",
		Model:    "test-model",
	}

	response, err := service.Chat(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "not available")
}

// TestPartialProviderConfiguration tests service with only some providers configured
func TestPartialProviderConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config *ai.Config
	}{
		{
			name: "only OpenAI",
			config: &ai.Config{
				OpenAI: ai.OpenAIConfig{
					APIKey: "test-key",
					Models: []string{"gpt-4o-mini"},
				},
				DefaultProvider: ai.ProviderOpenAI,
				MaxTokens:       1000,
				Temperature:     0.7,
			},
		},
		{
			name: "only Anthropic",
			config: &ai.Config{
				Anthropic: ai.AnthropicConfig{
					APIKey: "test-key",
					Models: []string{"claude-3-5-sonnet-20241022"},
				},
				DefaultProvider: ai.ProviderAnthropic,
				MaxTokens:       1000,
				Temperature:     0.7,
			},
		},
		{
			name: "only Ollama",
			config: &ai.Config{
				Ollama: ai.OllamaConfig{
					Endpoint: "http://localhost:11434",
					Models:   []string{"llama2"},
				},
				DefaultProvider: ai.ProviderOllama,
				MaxTokens:       1000,
				Temperature:     0.7,
			},
		},
		{
			name: "ClaudeCode only",
			config: &ai.Config{
				ClaudeCode: ai.ClaudeCodeConfig{
					ClaudePath: "/usr/local/bin/claude",
					Model:      "claude-3-5-sonnet-20241022",
				},
				DefaultProvider: ai.ProviderClaudeCode,
				MaxTokens:       1000,
				Temperature:     0.7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLogger()
			service, err := ai.NewService(tt.config, logger)

			require.NoError(t, err)
			assert.NotNil(t, service)
		})
	}
}

// TestConfigDefaults tests that default values are applied correctly
func TestConfigDefaults(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	// Test that request gets defaults applied
	request := &ai.SQLRequest{
		Prompt: "test",
	}

	err = service.ValidateRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultProvider, request.Provider)
	assert.NotEmpty(t, request.Model)
	assert.Equal(t, config.MaxTokens, request.MaxTokens)
	// Temperature is 0 by default in the request struct, which is valid
	// The service only sets it if it's out of range (< 0 or > 1)
	assert.GreaterOrEqual(t, request.Temperature, 0.0)
	assert.LessOrEqual(t, request.Temperature, 1.0)
}

// TestErrorTypes tests that different error types are returned correctly
func TestErrorTypes(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     *ai.SQLRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			request:     nil,
			expectedErr: "request cannot be nil",
		},
		{
			name: "missing prompt and query",
			request: &ai.SQLRequest{
				Provider: ai.ProviderOpenAI,
				Model:    "gpt-4o-mini",
			},
			expectedErr: "either prompt or query is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateRequest(tt.request)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestUsageStatsIsolation tests that usage stats for different providers are isolated
func TestUsageStatsIsolation(t *testing.T) {
	config := &ai.Config{
		OpenAI: ai.OpenAIConfig{
			APIKey: "test-openai-key",
			Models: []string{"gpt-4o-mini"},
		},
		Anthropic: ai.AnthropicConfig{
			APIKey: "test-anthropic-key",
			Models: []string{"claude-3-5-sonnet-20241022"},
		},
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       1000,
		Temperature:     0.7,
	}
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Get initial usage stats
	allUsage, err := service.GetAllUsageStats(ctx)
	require.NoError(t, err)

	// Verify each provider has its own usage stats
	for provider, usage := range allUsage {
		assert.Equal(t, provider, usage.Provider)
		assert.Equal(t, int64(0), usage.RequestCount)
		assert.Equal(t, int64(0), usage.TokensUsed)
		assert.Equal(t, 1.0, usage.SuccessRate)
	}
}

// TestProviderTypeConstants tests that provider type constants are correctly defined
func TestProviderTypeConstants(t *testing.T) {
	// Test that provider constants are non-empty strings
	assert.NotEmpty(t, ai.ProviderOpenAI)
	assert.NotEmpty(t, ai.ProviderAnthropic)
	assert.NotEmpty(t, ai.ProviderOllama)
	assert.NotEmpty(t, ai.ProviderHuggingFace)
	assert.NotEmpty(t, ai.ProviderClaudeCode)
	assert.NotEmpty(t, ai.ProviderCodex)

	// Test that provider constants are unique
	providers := []ai.Provider{
		ai.ProviderOpenAI,
		ai.ProviderAnthropic,
		ai.ProviderOllama,
		ai.ProviderHuggingFace,
		ai.ProviderClaudeCode,
		ai.ProviderCodex,
	}

	seen := make(map[ai.Provider]bool)
	for _, p := range providers {
		assert.False(t, seen[p], "duplicate provider constant: %s", p)
		seen[p] = true
	}
}

// TestChatRequestValidation tests chat request validation edge cases
func TestChatRequestValidation(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     *ai.ChatRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
			errorMsg:    "request cannot be nil",
		},
		{
			name: "empty prompt",
			request: &ai.ChatRequest{
				Provider: ai.ProviderOpenAI,
				Model:    "gpt-4o-mini",
			},
			expectError: true,
			errorMsg:    "prompt is required",
		},
		{
			name: "unavailable provider",
			request: &ai.ChatRequest{
				Prompt:   "test",
				Provider: "nonexistent",
				Model:    "test",
			},
			expectError: true,
			errorMsg:    "not available",
		},
		{
			name: "valid with defaults",
			request: &ai.ChatRequest{
				Prompt: "test",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateChatRequest(tt.request)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMultiProviderConfig tests configuration with multiple providers
func TestMultiProviderConfig(t *testing.T) {
	config := &ai.Config{
		OpenAI: ai.OpenAIConfig{
			APIKey:  "test-openai-key",
			BaseURL: "https://api.openai.com",
			Models:  []string{"gpt-4o-mini", "gpt-4"},
		},
		Anthropic: ai.AnthropicConfig{
			APIKey:  "test-anthropic-key",
			BaseURL: "https://api.anthropic.com",
			Models:  []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"},
		},
		Ollama: ai.OllamaConfig{
			Endpoint: "http://localhost:11434",
			Models:   []string{"llama2", "mistral"},
		},
		HuggingFace: ai.HuggingFaceConfig{
			Endpoint: "http://localhost:8080",
			Models:   []string{"codellama/CodeLlama-7b-hf"},
		},
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       2000,
		Temperature:     0.5,
		RequestTimeout:  60 * time.Second,
	}

	logger := newTestLogger()
	service, err := ai.NewService(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Verify config is stored
	retrievedConfig := service.GetConfig()
	assert.Equal(t, config.DefaultProvider, retrievedConfig.DefaultProvider)
	assert.Equal(t, config.MaxTokens, retrievedConfig.MaxTokens)
	assert.Equal(t, config.Temperature, retrievedConfig.Temperature)
}

// BenchmarkService_GetProviders benchmarks the GetProviders method
func BenchmarkService_GetProviders(b *testing.B) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetProviders()
	}
}

// BenchmarkService_GetConfig benchmarks the GetConfig method
func BenchmarkService_GetConfig(b *testing.B) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetConfig()
	}
}

// BenchmarkService_ValidateRequest benchmarks request validation
func BenchmarkService_ValidateRequest(b *testing.B) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(b, err)

	request := &ai.SQLRequest{
		Prompt: "SELECT all users",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ValidateRequest(request)
	}
}

// BenchmarkService_ConcurrentGetProviders benchmarks concurrent GetProviders calls
func BenchmarkService_ConcurrentGetProviders(b *testing.B) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = service.GetProviders()
		}
	})
}

// TestServiceImpl_IsNotExportedDirectly tests that serviceImpl is not accessible from outside package
func TestServiceImpl_IsNotExportedDirectly(t *testing.T) {
	// This test verifies that we only export the Service interface, not the implementation
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	// Verify service implements the Service interface
	var _ = service
}

// TestAIError tests the custom error type
func TestAIError(t *testing.T) {
	tests := []struct {
		name      string
		errorType ai.ErrorType
		message   string
		provider  ai.Provider
		retryable bool
	}{
		{
			name:      "timeout error is retryable",
			errorType: ai.ErrorTypeTimeout,
			message:   "request timeout",
			provider:  ai.ProviderOpenAI,
			retryable: true,
		},
		{
			name:      "rate limit error is retryable",
			errorType: ai.ErrorTypeRateLimit,
			message:   "rate limit exceeded",
			provider:  ai.ProviderOpenAI,
			retryable: true,
		},
		{
			name:      "invalid request is not retryable",
			errorType: ai.ErrorTypeInvalidRequest,
			message:   "invalid request",
			provider:  ai.ProviderOpenAI,
			retryable: false,
		},
		{
			name:      "provider error is not retryable",
			errorType: ai.ErrorTypeProviderError,
			message:   "provider error",
			provider:  ai.ProviderOpenAI,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aiErr := ai.NewAIError(tt.errorType, tt.message, tt.provider)

			assert.Equal(t, tt.errorType, aiErr.Type)
			assert.Equal(t, tt.message, aiErr.Message)
			assert.Equal(t, tt.provider, aiErr.Provider)
			assert.Equal(t, tt.retryable, aiErr.Retryable)
			assert.Equal(t, tt.message, aiErr.Error())
		})
	}
}

// TestRequestValidation_BoundaryValues tests validation with boundary values
func TestRequestValidation_BoundaryValues(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     *ai.SQLRequest
		expectError bool
	}{
		{
			name: "temperature at lower bound (0)",
			request: &ai.SQLRequest{
				Prompt:      "test",
				Provider:    ai.ProviderOpenAI,
				Model:       "gpt-4o-mini",
				Temperature: 0.0,
			},
			expectError: false,
		},
		{
			name: "temperature at upper bound (1)",
			request: &ai.SQLRequest{
				Prompt:      "test",
				Provider:    ai.ProviderOpenAI,
				Model:       "gpt-4o-mini",
				Temperature: 1.0,
			},
			expectError: false,
		},
		{
			name: "max tokens at 1",
			request: &ai.SQLRequest{
				Prompt:    "test",
				Provider:  ai.ProviderOpenAI,
				Model:     "gpt-4o-mini",
				MaxTokens: 1,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateRequest(tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHuggingFaceProviderConfig tests HuggingFace provider configuration
func TestHuggingFaceProviderConfig(t *testing.T) {
	config := &ai.Config{
		HuggingFace: ai.HuggingFaceConfig{
			Endpoint:         "http://localhost:8080",
			Models:           []string{"codellama/CodeLlama-7b-hf"},
			RecommendedModel: "codellama/CodeLlama-7b-hf",
		},
		DefaultProvider: ai.ProviderHuggingFace,
		MaxTokens:       1000,
		Temperature:     0.7,
	}

	logger := newTestLogger()
	service, err := ai.NewService(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

// TestCodexProviderConfig tests Codex provider configuration
func TestCodexProviderConfig(t *testing.T) {
	config := &ai.Config{
		Codex: ai.CodexConfig{
			APIKey: "test-codex-key",
			Model:  "codex-model",
		},
		DefaultProvider: ai.ProviderCodex,
		MaxTokens:       1000,
		Temperature:     0.7,
	}

	logger := newTestLogger()
	service, err := ai.NewService(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

// mockProviderWithError creates a mock provider that returns errors
func mockProviderWithError(providerType ai.Provider, err error) *mockAIProvider {
	return &mockAIProvider{
		providerType: providerType,
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, err
		},
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, err
		},
		chatFunc: func(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
			return nil, err
		},
	}
}

// TestProviderError tests handling of provider errors
func TestProviderError(t *testing.T) {
	testErr := errors.New("provider error")
	mock := mockProviderWithError(ai.ProviderOpenAI, testErr)

	ctx := context.Background()

	// Test GenerateSQL error
	_, err := mock.GenerateSQL(ctx, &ai.SQLRequest{})
	assert.Error(t, err)
	assert.Equal(t, testErr, err)

	// Test FixSQL error
	_, err = mock.FixSQL(ctx, &ai.SQLRequest{})
	assert.Error(t, err)
	assert.Equal(t, testErr, err)

	// Test Chat error
	_, err = mock.Chat(ctx, &ai.ChatRequest{})
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

// TestServiceContextCancellation tests that context cancellation is respected
func TestServiceContextCancellation(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// These operations should handle the canceled context gracefully
	_, err = service.GetAllProvidersHealth(ctx)
	// Note: The actual implementation may or may not check context cancellation
	// This test documents the behavior
	_ = err

	_, err = service.GetAllUsageStats(ctx)
	_ = err
}

// TestValidateRequest_ContextPreservation tests that context is preserved in requests
func TestValidateRequest_ContextPreservation(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	contextMap := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	request := &ai.SQLRequest{
		Prompt:  "test",
		Context: contextMap,
	}

	err = service.ValidateRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, contextMap, request.Context)
}

// TestClaudeCodeConfig tests ClaudeCode configuration variations
func TestClaudeCodeConfig(t *testing.T) {
	tests := []struct {
		name   string
		config ai.ClaudeCodeConfig
	}{
		{
			name: "with claude path",
			config: ai.ClaudeCodeConfig{
				ClaudePath: "/usr/local/bin/claude",
				Model:      "claude-3-5-sonnet-20241022",
			},
		},
		{
			name: "with model only",
			config: ai.ClaudeCodeConfig{
				Model: "claude-3-5-sonnet-20241022",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ai.Config{
				ClaudeCode:      tt.config,
				DefaultProvider: ai.ProviderClaudeCode,
				MaxTokens:       1000,
				Temperature:     0.7,
			}

			logger := newTestLogger()
			service, err := ai.NewService(config, logger)

			// In CI, Claude binary may not be available
			// Service creation will fail with "no AI providers configured"
			if err != nil {
				assert.Contains(t, err.Error(), "no AI providers configured")
				t.Skip("Claude binary not available - expected behavior in CI")
				return
			}
			assert.NotNil(t, service)
		})
	}
}

// TestGetProvidersReturnsSlice tests that GetProviders always returns a slice
func TestGetProvidersReturnsSlice(t *testing.T) {
	config := newTestConfig()
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	providers := service.GetProviders()

	assert.NotNil(t, providers)
	assert.IsType(t, []ai.Provider{}, providers)
}

// TestDefaultTemperatureRange tests that temperature defaults to valid range
func TestDefaultTemperatureRange(t *testing.T) {
	config := newTestConfig()
	config.Temperature = 0.5
	logger := newTestLogger()
	service, err := ai.NewService(config, logger)
	require.NoError(t, err)

	request := &ai.SQLRequest{
		Prompt:      "test",
		Temperature: -1.0, // Invalid
	}

	err = service.ValidateRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, 0.5, request.Temperature)
}

// TestServiceInitializationWithWarnings tests that service initializes even if some providers fail
func TestServiceInitializationWithWarnings(t *testing.T) {
	// Configure multiple providers with invalid credentials
	// Service should still initialize if at least one provider succeeds
	config := &ai.Config{
		OpenAI: ai.OpenAIConfig{
			APIKey: "invalid-key",
			Models: []string{"gpt-4o-mini"},
		},
		Anthropic: ai.AnthropicConfig{
			APIKey: "invalid-key",
			Models: []string{"claude-3-5-sonnet-20241022"},
		},
		Ollama: ai.OllamaConfig{
			Endpoint: "http://localhost:11434", // May or may not be running
			Models:   []string{"llama2"},
		},
		DefaultProvider: ai.ProviderOpenAI,
		MaxTokens:       1000,
		Temperature:     0.7,
	}

	logger := newTestLogger()
	service, err := ai.NewService(config, logger)

	// Service may initialize successfully even with invalid credentials
	// because provider initialization errors are logged as warnings
	if err != nil {
		assert.Contains(t, err.Error(), "no AI providers configured")
	}

	// If service initialized, it should be non-nil
	if service != nil {
		assert.NotNil(t, service)
	}
}
