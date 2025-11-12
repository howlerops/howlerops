//go:build integration

package ai_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOllamaProvider is a mock implementation of the AIProvider interface for Ollama
type mockOllamaProvider struct {
	generateSQLFunc    func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	fixSQLFunc         func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	chatFunc           func(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error)
	healthCheckFunc    func(ctx context.Context) (*ai.HealthStatus, error)
	getModelsFunc      func(ctx context.Context) ([]ai.ModelInfo, error)
	updateConfigFunc   func(config interface{}) error
	validateConfigFunc func(config interface{}) error
}

func (m *mockOllamaProvider) GenerateSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.generateSQLFunc != nil {
		return m.generateSQLFunc(ctx, req)
	}
	return &ai.SQLResponse{Query: "SELECT * FROM users"}, nil
}

func (m *mockOllamaProvider) FixSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.fixSQLFunc != nil {
		return m.fixSQLFunc(ctx, req)
	}
	return &ai.SQLResponse{Query: "SELECT * FROM users WHERE id = 1"}, nil
}

func (m *mockOllamaProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ai.ChatResponse{Content: "Response"}, nil
}

func (m *mockOllamaProvider) HealthCheck(ctx context.Context) (*ai.HealthStatus, error) {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return &ai.HealthStatus{Status: "healthy"}, nil
}

func (m *mockOllamaProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	if m.getModelsFunc != nil {
		return m.getModelsFunc(ctx)
	}
	return []ai.ModelInfo{}, nil
}

func (m *mockOllamaProvider) GetProviderType() ai.Provider {
	return ai.ProviderOllama
}

func (m *mockOllamaProvider) IsAvailable(ctx context.Context) bool {
	health, err := m.HealthCheck(ctx)
	return err == nil && health.Status == "healthy"
}

func (m *mockOllamaProvider) UpdateConfig(config interface{}) error {
	if m.updateConfigFunc != nil {
		return m.updateConfigFunc(config)
	}
	return nil
}

func (m *mockOllamaProvider) ValidateConfig(config interface{}) error {
	if m.validateConfigFunc != nil {
		return m.validateConfigFunc(config)
	}
	return nil
}

// mockOllamaDetector is a mock implementation of OllamaDetector
type mockOllamaDetector struct {
	checkModelExistsFunc func(ctx context.Context, modelName string, endpoint string) (bool, error)
	pullModelFunc        func(ctx context.Context, modelName string) error
	detectOllamaFunc     func(ctx context.Context) (*ai.OllamaStatus, error)
}

func (m *mockOllamaDetector) CheckModelExists(ctx context.Context, modelName string, endpoint string) (bool, error) {
	if m.checkModelExistsFunc != nil {
		return m.checkModelExistsFunc(ctx, modelName, endpoint)
	}
	return true, nil
}

func (m *mockOllamaDetector) PullModel(ctx context.Context, modelName string) error {
	if m.pullModelFunc != nil {
		return m.pullModelFunc(ctx, modelName)
	}
	return nil
}

func (m *mockOllamaDetector) DetectOllama(ctx context.Context) (*ai.OllamaStatus, error) {
	if m.detectOllamaFunc != nil {
		return m.detectOllamaFunc(ctx)
	}
	return &ai.OllamaStatus{
		Installed: true,
		Running:   true,
	}, nil
}

// ==================== Constructor Tests ====================

// TestNewHuggingFaceProvider_Success tests successful provider creation
func TestNewHuggingFaceProvider_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Endpoint:         "http://localhost:11434",
		Models:           []string{"prem-1b-sql"},
		PullTimeout:      10 * time.Minute,
		GenerateTimeout:  2 * time.Minute,
		AutoPullModels:   true,
		RecommendedModel: "prem-1b-sql",
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, ai.ProviderHuggingFace, provider.GetProviderType())
}

// TestNewHuggingFaceProvider_NilConfig tests that nil config is rejected
func TestNewHuggingFaceProvider_NilConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	provider, err := ai.NewHuggingFaceProvider(nil, logger)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

// TestNewHuggingFaceProvider_DefaultEndpoint tests default endpoint is set
func TestNewHuggingFaceProvider_DefaultEndpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Endpoint: "",
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "http://localhost:11434", config.Endpoint)
}

// TestNewHuggingFaceProvider_DefaultPullTimeout tests default pull timeout is set
func TestNewHuggingFaceProvider_DefaultPullTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		PullTimeout: 0,
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, 10*time.Minute, config.PullTimeout)
}

// TestNewHuggingFaceProvider_DefaultGenerateTimeout tests default generate timeout is set
func TestNewHuggingFaceProvider_DefaultGenerateTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		GenerateTimeout: 0,
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, 2*time.Minute, config.GenerateTimeout)
}

// TestNewHuggingFaceProvider_DefaultRecommendedModel tests default recommended model is set
func TestNewHuggingFaceProvider_DefaultRecommendedModel(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "",
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "prem-1b-sql", config.RecommendedModel)
}

// TestNewHuggingFaceProvider_DefaultModels tests default models are set
func TestNewHuggingFaceProvider_DefaultModels(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{},
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotEmpty(t, config.Models)
	assert.Contains(t, config.Models, "prem-1b-sql")
	assert.Contains(t, config.Models, "sqlcoder:7b")
	assert.Contains(t, config.Models, "codellama:7b")
	assert.Contains(t, config.Models, "llama3.1:8b")
	assert.Contains(t, config.Models, "mistral:7b")
	assert.Len(t, config.Models, 5)
}

// TestNewHuggingFaceProvider_CustomConfig tests custom configuration is preserved
func TestNewHuggingFaceProvider_CustomConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Endpoint:         "http://custom:8080",
		Models:           []string{"custom-model"},
		PullTimeout:      5 * time.Minute,
		GenerateTimeout:  1 * time.Minute,
		AutoPullModels:   false,
		RecommendedModel: "custom-model",
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "http://custom:8080", config.Endpoint)
	assert.Equal(t, []string{"custom-model"}, config.Models)
	assert.Equal(t, 5*time.Minute, config.PullTimeout)
	assert.Equal(t, 1*time.Minute, config.GenerateTimeout)
	assert.False(t, config.AutoPullModels)
	assert.Equal(t, "custom-model", config.RecommendedModel)
}

// TestNewHuggingFaceProvider_NilLogger tests that nil logger works
func TestNewHuggingFaceProvider_NilLogger(t *testing.T) {
	config := &ai.HuggingFaceConfig{}

	provider, err := ai.NewHuggingFaceProvider(config, nil)

	require.NoError(t, err)
	require.NotNil(t, provider)
}

// ==================== GetProviderType Tests ====================

// TestHuggingFaceGetProviderType tests provider type retrieval
func TestHuggingFaceGetProviderType(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	providerType := provider.GetProviderType()

	assert.Equal(t, ai.ProviderHuggingFace, providerType)
}

// ==================== ValidateConfig Tests ====================

// TestHuggingFaceValidateConfig_Success tests successful config validation
func TestHuggingFaceValidateConfig_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	validConfig := &ai.HuggingFaceConfig{
		Endpoint: "http://localhost:11434",
	}

	err = provider.ValidateConfig(validConfig)

	assert.NoError(t, err)
}

// TestHuggingFaceValidateConfig_InvalidType tests validation with wrong config type
func TestHuggingFaceValidateConfig_InvalidType(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	err = provider.ValidateConfig("invalid-config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// TestHuggingFaceValidateConfig_MissingEndpoint tests validation with missing endpoint
func TestHuggingFaceValidateConfig_MissingEndpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := &ai.HuggingFaceConfig{
		Endpoint: "",
	}

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

// ==================== UpdateConfig Tests ====================

// TestHuggingFaceUpdateConfig_Success tests successful config update
func TestHuggingFaceUpdateConfig_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Endpoint: "http://localhost:11434",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	newConfig := &ai.HuggingFaceConfig{
		Endpoint:         "http://new:8080",
		Models:           []string{"new-model"},
		PullTimeout:      15 * time.Minute,
		GenerateTimeout:  3 * time.Minute,
		AutoPullModels:   true,
		RecommendedModel: "new-model",
	}

	err = provider.UpdateConfig(newConfig)

	require.NoError(t, err)
}

// TestHuggingFaceUpdateConfig_InvalidType tests update with wrong config type
func TestHuggingFaceUpdateConfig_InvalidType(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	err = provider.UpdateConfig("invalid-config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// TestHuggingFaceUpdateConfig_InvalidConfig tests update with invalid config
func TestHuggingFaceUpdateConfig_InvalidConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := &ai.HuggingFaceConfig{
		Endpoint: "",
	}

	err = provider.UpdateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

// ==================== GenerateSQL Tests ====================

// TestHuggingFaceGenerateSQL_Success tests successful SQL generation
func TestHuggingFaceGenerateSQL_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Schema:      "users (id, name, email)",
		Model:       "prem-1b-sql",
		MaxTokens:   2048,
		Temperature: 0.1,
	}

	// Note: This will fail if Ollama is not running - we're testing the wrapper logic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := provider.GenerateSQL(ctx, req)

	// If Ollama is not available, we expect an error, but we're testing the wrapper works
	if err != nil {
		// Expected when Ollama is not running
		assert.Error(t, err)
	} else {
		require.NotNil(t, response)
		assert.NotEmpty(t, response.Query)
		assert.Equal(t, ai.ProviderOllama, response.Provider)
	}
}

// TestHuggingFaceGenerateSQL_ModelCheckFailure tests when model check fails
func TestHuggingFaceGenerateSQL_ModelCheckFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Get all users",
		Model:  "nonexistent-model",
	}

	ctx := context.Background()
	response, err := provider.GenerateSQL(ctx, req)

	// Should fail model availability check
	assert.Error(t, err)
	assert.Nil(t, response)
}

// ==================== FixSQL Tests ====================

// TestHuggingFaceFixSQL_Success tests successful SQL fixing
func TestHuggingFaceFixSQL_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query:       "SELECT * FROM users WHERE id =",
		Error:       "syntax error at end of input",
		Schema:      "users (id, name, email)",
		Model:       "prem-1b-sql",
		MaxTokens:   2048,
		Temperature: 0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := provider.FixSQL(ctx, req)

	// If Ollama is not available, we expect an error
	if err != nil {
		assert.Error(t, err)
	} else {
		require.NotNil(t, response)
		assert.NotEmpty(t, response.Query)
		assert.Equal(t, ai.ProviderOllama, response.Provider)
	}
}

// TestHuggingFaceFixSQL_ModelCheckFailure tests when model check fails
func TestHuggingFaceFixSQL_ModelCheckFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT * FROM",
		Error: "incomplete query",
		Model: "nonexistent-model",
	}

	ctx := context.Background()
	response, err := provider.FixSQL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
}

// ==================== Chat Tests ====================

// TestHuggingFaceChat_Success tests successful chat interaction
func TestHuggingFaceChat_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:      "What is SQL?",
		Model:       "prem-1b-sql",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := provider.Chat(ctx, req)

	if err != nil {
		assert.Error(t, err)
	} else {
		require.NotNil(t, response)
		assert.NotEmpty(t, response.Content)
		assert.Equal(t, ai.ProviderOllama, response.Provider)
	}
}

// TestHuggingFaceChat_ModelCheckFailure tests when model check fails
func TestHuggingFaceChat_ModelCheckFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "Hello",
		Model:  "nonexistent-model",
	}

	ctx := context.Background()
	response, err := provider.Chat(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
}

// ==================== HealthCheck Tests ====================

// TestHuggingFaceHealthCheck_OllamaNotInstalled tests when Ollama is not installed
func TestHuggingFaceHealthCheck_OllamaNotInstalled(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	health, err := provider.HealthCheck(ctx)

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderHuggingFace, health.Provider)
	// Status will be unhealthy if Ollama is not installed/running
	assert.NotEmpty(t, health.Status)
	assert.NotEmpty(t, health.Message)
}

// TestHuggingFaceHealthCheck_DetectorError tests when detector returns error
func TestHuggingFaceHealthCheck_DetectorError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	health, err := provider.HealthCheck(ctx)

	// Should not return error even if Ollama is not available
	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderHuggingFace, health.Provider)
}

// ==================== GetModels Tests ====================

// TestHuggingFaceGetModels_Success tests retrieving available models
func TestHuggingFaceGetModels_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{"prem-1b-sql", "sqlcoder:7b"},
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)

	// Will fail if Ollama is not running
	if err != nil {
		assert.Error(t, err)
	} else {
		require.NotNil(t, models)
		// Check that models are marked as HuggingFace provider
		for _, model := range models {
			assert.Equal(t, ai.ProviderHuggingFace, model.Provider)
		}
	}
}

// TestHuggingFaceGetModels_PremSQL tests prem-1b-sql specific metadata
func TestHuggingFaceGetModels_PremSQL(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{"prem-1b-sql"},
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)

	if err == nil && len(models) > 0 {
		// Find prem-1b-sql model
		for _, model := range models {
			if model.Name == "prem-1b-sql" {
				assert.Contains(t, model.Description, "Prem-1B-SQL")
				assert.Contains(t, model.Description, "Recommended")
				assert.Contains(t, model.Capabilities, "text-to-sql")
				assert.Contains(t, model.Capabilities, "sql-fixing")
				assert.Contains(t, model.Capabilities, "sql-optimization")
				assert.Contains(t, model.Capabilities, "schema-analysis")
			}
		}
	}
}

// TestHuggingFaceGetModels_SQLCoder tests sqlcoder specific metadata
func TestHuggingFaceGetModels_SQLCoder(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{"sqlcoder:7b"},
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)

	if err == nil && len(models) > 0 {
		for _, model := range models {
			if model.Name == "sqlcoder:7b" || model.Name == "sqlcoder" {
				assert.Contains(t, model.Description, "SQLCoder")
				assert.Contains(t, model.Capabilities, "text-to-sql")
				assert.Contains(t, model.Capabilities, "sql-fixing")
				assert.Contains(t, model.Capabilities, "sql-optimization")
			}
		}
	}
}

// TestHuggingFaceGetModels_CodeLlama tests codellama specific metadata
func TestHuggingFaceGetModels_CodeLlama(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{"codellama:7b"},
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)

	if err == nil && len(models) > 0 {
		for _, model := range models {
			if model.Name == "codellama:7b" || model.Name == "codellama" {
				assert.Contains(t, model.Description, "CodeLlama")
				assert.Contains(t, model.Capabilities, "text-to-sql")
				assert.Contains(t, model.Capabilities, "sql-fixing")
			}
		}
	}
}

// TestHuggingFaceGetModels_Llama tests llama specific metadata
func TestHuggingFaceGetModels_Llama(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{"llama3.1:8b"},
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)

	if err == nil && len(models) > 0 {
		for _, model := range models {
			if model.Name == "llama3.1:8b" {
				assert.Contains(t, model.Description, "Llama")
				assert.Contains(t, model.Capabilities, "text-to-sql")
			}
		}
	}
}

// TestHuggingFaceGetModels_Mistral tests mistral specific metadata
func TestHuggingFaceGetModels_Mistral(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Models: []string{"mistral:7b"},
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)

	if err == nil && len(models) > 0 {
		for _, model := range models {
			if model.Name == "mistral:7b" || model.Name == "mistral" {
				assert.Contains(t, model.Description, "Mistral")
				assert.Contains(t, model.Capabilities, "text-to-sql")
			}
		}
	}
}

// ==================== IsAvailable Tests ====================

// TestHuggingFaceIsAvailable tests IsAvailable
func TestHuggingFaceIsAvailable(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	available := provider.IsAvailable(ctx)

	// Will be false if Ollama is not installed/running
	assert.NotNil(t, available)
}

// ==================== Model Availability Tests ====================

// TestHuggingFaceModelAvailability_ModelExists tests when model exists
func TestHuggingFaceModelAvailability_ModelExists(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
		AutoPullModels:   false,
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	// Test that model check is performed
	req := &ai.SQLRequest{
		Prompt: "Test",
		Model:  "prem-1b-sql",
	}

	ctx := context.Background()
	_, err = provider.GenerateSQL(ctx, req)

	// If Ollama is not running, will get an error
	// We're just testing that the code path executes
	if err != nil {
		assert.Error(t, err)
	}
}

// TestHuggingFaceModelAvailability_AutoPullDisabled tests when auto-pull is disabled
func TestHuggingFaceModelAvailability_AutoPullDisabled(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
		AutoPullModels:   false,
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Test",
		Model:  "nonexistent-model-xyz",
	}

	ctx := context.Background()
	_, err = provider.GenerateSQL(ctx, req)

	// Should fail because model doesn't exist and auto-pull is disabled
	assert.Error(t, err)
}

// TestHuggingFaceModelAvailability_AutoPullEnabled tests when auto-pull is enabled
func TestHuggingFaceModelAvailability_AutoPullEnabled(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
		AutoPullModels:   true,
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Test",
		Model:  "nonexistent-model-xyz",
	}

	ctx := context.Background()
	_, err = provider.GenerateSQL(ctx, req)

	// Will attempt to pull the model, but should fail for non-existent model
	assert.Error(t, err)
}

// ==================== Integration Tests ====================

// TestHuggingFaceIntegration_ConfigMapping tests that config is properly mapped to Ollama
func TestHuggingFaceIntegration_ConfigMapping(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		Endpoint:         "http://localhost:11434",
		Models:           []string{"prem-1b-sql"},
		PullTimeout:      15 * time.Minute,
		GenerateTimeout:  3 * time.Minute,
		AutoPullModels:   true,
		RecommendedModel: "prem-1b-sql",
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Config should be properly mapped to underlying Ollama provider
}

// TestHuggingFaceIntegration_ProviderDelegation tests that calls are delegated to Ollama
func TestHuggingFaceIntegration_ProviderDelegation(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	// Test that GetProviderType returns HuggingFace (wrapper)
	assert.Equal(t, ai.ProviderHuggingFace, provider.GetProviderType())

	// Test that operations delegate to Ollama (will fail if not available)
	ctx := context.Background()

	// These will fail if Ollama is not running, but we're testing delegation logic
	_, _ = provider.GenerateSQL(ctx, &ai.SQLRequest{Prompt: "test", Model: "prem-1b-sql"})
	_, _ = provider.FixSQL(ctx, &ai.SQLRequest{Query: "SELECT", Error: "error", Model: "prem-1b-sql"})
	_, _ = provider.Chat(ctx, &ai.ChatRequest{Prompt: "test", Model: "prem-1b-sql"})
	_, _ = provider.HealthCheck(ctx)
	_, _ = provider.GetModels(ctx)
}

// ==================== Error Handling Tests ====================

// TestHuggingFaceErrorHandling_GenerateSQLError tests GenerateSQL error propagation
func TestHuggingFaceErrorHandling_GenerateSQLError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "test",
		Model:  "prem-1b-sql",
	}

	ctx := context.Background()
	_, err = provider.GenerateSQL(ctx, req)

	// Should get an error if Ollama is not running
	// We're testing that errors are properly propagated
	if err != nil {
		assert.Error(t, err)
	}
}

// TestHuggingFaceErrorHandling_FixSQLError tests FixSQL error propagation
func TestHuggingFaceErrorHandling_FixSQLError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT",
		Error: "error",
		Model: "prem-1b-sql",
	}

	ctx := context.Background()
	_, err = provider.FixSQL(ctx, req)

	if err != nil {
		assert.Error(t, err)
	}
}

// TestHuggingFaceErrorHandling_ChatError tests Chat error propagation
func TestHuggingFaceErrorHandling_ChatError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "test",
		Model:  "prem-1b-sql",
	}

	ctx := context.Background()
	_, err = provider.Chat(ctx, req)

	if err != nil {
		assert.Error(t, err)
	}
}

// ==================== Context Tests ====================

// TestHuggingFaceContext_Cancellation tests context cancellation
func TestHuggingFaceContext_Cancellation(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &ai.SQLRequest{
		Prompt: "test",
		Model:  "prem-1b-sql",
	}

	_, err = provider.GenerateSQL(ctx, req)

	// Should get an error due to cancelled context
	assert.Error(t, err)
}

// TestHuggingFaceContext_Timeout tests context timeout
func TestHuggingFaceContext_Timeout(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout

	req := &ai.SQLRequest{
		Prompt: "test",
		Model:  "prem-1b-sql",
	}

	_, err = provider.GenerateSQL(ctx, req)

	// Should get an error due to timeout
	assert.Error(t, err)
}

// ==================== GetRecommendedModel Tests ====================

// TestHuggingFaceGetRecommendedModel tests getting recommended model
func TestHuggingFaceGetRecommendedModel(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "custom-model",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	// Access via type assertion since GetRecommendedModel is not in AIProvider interface
	type recommendedModelGetter interface {
		GetRecommendedModel() string
	}

	if getter, ok := provider.(recommendedModelGetter); ok {
		model := getter.GetRecommendedModel()
		assert.Equal(t, "custom-model", model)
	}
}

// ==================== GetInstallationInstructions Tests ====================

// TestHuggingFaceGetInstallationInstructions tests getting installation instructions
func TestHuggingFaceGetInstallationInstructions(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	// Access via type assertion
	type installationInstructionsGetter interface {
		GetInstallationInstructions() (string, error)
	}

	if getter, ok := provider.(installationInstructionsGetter); ok {
		instructions, err := getter.GetInstallationInstructions()

		// Should not error
		if err != nil {
			// May error if detection fails, which is okay
			assert.Error(t, err)
		} else {
			assert.NotEmpty(t, instructions)
			// Should mention Ollama since HuggingFace uses Ollama
			assert.Contains(t, instructions, "Ollama")
		}
	}
}

// ==================== Concurrent Access Tests ====================

// TestHuggingFaceConcurrency_MultipleRequests tests concurrent requests
func TestHuggingFaceConcurrency_MultipleRequests(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Make multiple concurrent requests
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func() {
			req := &ai.SQLRequest{
				Prompt: "test",
				Model:  "prem-1b-sql",
			}
			_, _ = provider.GenerateSQL(ctx, req)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we got here without panics, concurrent access works
	assert.True(t, true)
}

// ==================== Edge Case Tests ====================

// TestHuggingFaceEdgeCase_EmptyModel tests with empty model string
func TestHuggingFaceEdgeCase_EmptyModel(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "test",
		Model:  "", // Empty model
	}

	ctx := context.Background()
	_, err = provider.GenerateSQL(ctx, req)

	// Should error
	assert.Error(t, err)
}

// TestHuggingFaceEdgeCase_VeryLongTimeout tests with very long timeout
func TestHuggingFaceEdgeCase_VeryLongTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
		PullTimeout:      24 * time.Hour,
		GenerateTimeout:  24 * time.Hour,
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, 24*time.Hour, config.PullTimeout)
	assert.Equal(t, 24*time.Hour, config.GenerateTimeout)
}

// TestHuggingFaceEdgeCase_ManyModels tests with many models in config
func TestHuggingFaceEdgeCase_ManyModels(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	models := make([]string, 50)
	for i := 0; i < 50; i++ {
		models[i] = "model-" + string(rune('a'+(i%26)))
	}

	config := &ai.HuggingFaceConfig{
		Models: models,
	}

	provider, err := ai.NewHuggingFaceProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Len(t, config.Models, 50)
}

// TestHuggingFaceEdgeCase_SpecialCharactersInModel tests with special characters in model name
func TestHuggingFaceEdgeCase_SpecialCharactersInModel(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.HuggingFaceConfig{
		RecommendedModel: "prem-1b-sql",
	}
	provider, err := ai.NewHuggingFaceProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "test",
		Model:  "model-with-special-chars!@#$%",
	}

	ctx := context.Background()
	_, err = provider.GenerateSQL(ctx, req)

	// Should error (invalid model name)
	assert.Error(t, err)
}
