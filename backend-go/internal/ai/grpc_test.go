package ai_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
	aipb "github.com/jbeck018/howlerops/backend-go/pkg/pb/ai"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGRPCService is a mock implementation of the ai.Service interface for testing gRPC server
type mockGRPCService struct {
	generateSQLFunc        func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	fixSQLFunc             func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	chatFunc               func(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error)
	getProvidersFunc       func() []ai.Provider
	getProviderHealthFunc  func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error)
	getAvailableModelsFunc func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error)
	getAllUsageStatsFunc   func(ctx context.Context) (map[ai.Provider]*ai.Usage, error)
	testProviderFunc       func(ctx context.Context, provider ai.Provider, config interface{}) error
	getConfigFunc          func() *ai.Config
}

func (m *mockGRPCService) GenerateSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.generateSQLFunc != nil {
		return m.generateSQLFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) FixSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.fixSQLFunc != nil {
		return m.fixSQLFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) GetProviders() []ai.Provider {
	if m.getProvidersFunc != nil {
		return m.getProvidersFunc()
	}
	return []ai.Provider{}
}

func (m *mockGRPCService) GetProviderHealth(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
	if m.getProviderHealthFunc != nil {
		return m.getProviderHealthFunc(ctx, provider)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) GetAllProvidersHealth(ctx context.Context) (map[ai.Provider]*ai.HealthStatus, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) GetAvailableModels(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
	if m.getAvailableModelsFunc != nil {
		return m.getAvailableModelsFunc(ctx, provider)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) GetAllAvailableModels(ctx context.Context) (map[ai.Provider][]ai.ModelInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) UpdateProviderConfig(provider ai.Provider, config interface{}) error {
	return errors.New("not implemented")
}

func (m *mockGRPCService) GetConfig() *ai.Config {
	if m.getConfigFunc != nil {
		return m.getConfigFunc()
	}
	return &ai.Config{}
}

func (m *mockGRPCService) GetUsageStats(ctx context.Context, provider ai.Provider) (*ai.Usage, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) GetAllUsageStats(ctx context.Context) (map[ai.Provider]*ai.Usage, error) {
	if m.getAllUsageStatsFunc != nil {
		return m.getAllUsageStatsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGRPCService) TestProvider(ctx context.Context, provider ai.Provider, config interface{}) error {
	if m.testProviderFunc != nil {
		return m.testProviderFunc(ctx, provider, config)
	}
	return errors.New("not implemented")
}

func (m *mockGRPCService) ValidateRequest(req *ai.SQLRequest) error {
	return nil
}

func (m *mockGRPCService) ValidateChatRequest(req *ai.ChatRequest) error {
	return nil
}

func (m *mockGRPCService) Start(ctx context.Context) error {
	return nil
}

func (m *mockGRPCService) Stop(ctx context.Context) error {
	return nil
}

// Helper to create test logger for gRPC tests
func createGRPCTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}

// Tests for NewGRPCServer

func TestNewGRPCServer_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{}

	server := ai.NewGRPCServer(service, logger)

	assert.NotNil(t, server)
}

func TestNewGRPCServer_WithNilLogger(t *testing.T) {
	service := &mockGRPCService{}

	// Should not panic even with nil logger
	server := ai.NewGRPCServer(service, nil)

	assert.NotNil(t, server)
}

func TestNewGRPCServer_WithNilService(t *testing.T) {
	logger := createGRPCTestLogger()

	// Should not panic even with nil service
	server := ai.NewGRPCServer(nil, logger)

	assert.NotNil(t, server)
}

// Tests for GenerateSQL

func TestGenerateSQL_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:       "SELECT * FROM users",
				Explanation: "Retrieve all users",
				Confidence:  0.95,
				Suggestions: []string{"Add WHERE clause"},
				Provider:    ai.ProviderOpenAI,
				Model:       "gpt-4",
				TokensUsed:  150,
				Metadata:    map[string]string{"version": "1"},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider:    aipb.Provider_PROVIDER_OPENAI,
			Model:       "gpt-4",
			Prompt:      "Get all users",
			Schema:      "CREATE TABLE users (id INT, name TEXT)",
			MaxTokens:   1000,
			Temperature: 0.7,
		},
	}

	resp, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Response)
	assert.Equal(t, "SELECT * FROM users", resp.Response.Query)
	assert.Equal(t, "Retrieve all users", resp.Response.Explanation)
	assert.Equal(t, float64(0.95), resp.Response.Confidence)
	assert.Equal(t, []string{"Add WHERE clause"}, resp.Response.Suggestions)
	assert.Equal(t, aipb.Provider_PROVIDER_OPENAI, resp.Response.Provider)
	assert.Equal(t, "gpt-4", resp.Response.Model)
	assert.Equal(t, int32(150), resp.Response.TokensUsed)
	assert.NotNil(t, resp.Response.TimeTaken)
	assert.Equal(t, map[string]string{"version": "1"}, resp.Response.Metadata)
	assert.Nil(t, resp.Error)
}

func TestGenerateSQL_ServiceError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, errors.New("API rate limit exceeded")
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Prompt:   "Get all users",
		},
	}

	resp, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err) // gRPC method doesn't return error, error is in response
	require.NotNil(t, resp)
	assert.Nil(t, resp.Response)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "API rate limit exceeded", resp.Error.Message)
}

func TestGenerateSQL_ProviderConversion(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name             string
		protoProvider    aipb.Provider
		expectedProvider ai.Provider
	}{
		{
			name:             "OpenAI provider",
			protoProvider:    aipb.Provider_PROVIDER_OPENAI,
			expectedProvider: ai.ProviderOpenAI,
		},
		{
			name:             "Anthropic provider",
			protoProvider:    aipb.Provider_PROVIDER_ANTHROPIC,
			expectedProvider: ai.ProviderAnthropic,
		},
		{
			name:             "Ollama provider",
			protoProvider:    aipb.Provider_PROVIDER_OLLAMA,
			expectedProvider: ai.ProviderOllama,
		},
		{
			name:             "HuggingFace provider",
			protoProvider:    aipb.Provider_PROVIDER_HUGGINGFACE,
			expectedProvider: ai.ProviderHuggingFace,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedProvider ai.Provider
			service := &mockGRPCService{
				generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
					capturedProvider = req.Provider
					return &ai.SQLResponse{
						Query:    "SELECT 1",
						Provider: req.Provider,
						Model:    req.Model,
					}, nil
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GenerateSQLRequest{
				Request: &aipb.SQLRequest{
					Provider: tc.protoProvider,
					Model:    "test-model",
					Prompt:   "test prompt",
				},
			}

			resp, err := server.GenerateSQL(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tc.expectedProvider, capturedProvider)
		})
	}
}

func TestGenerateSQL_RequestFieldValidation(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedRequest *ai.SQLRequest
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			capturedRequest = req
			return &ai.SQLResponse{
				Query: "SELECT 1",
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider:    aipb.Provider_PROVIDER_OPENAI,
			Model:       "gpt-4",
			Prompt:      "Get all users from database",
			Schema:      "CREATE TABLE users (id INT, name TEXT)",
			MaxTokens:   2000,
			Temperature: 0.5,
		},
	}

	_, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, capturedRequest)
	assert.Equal(t, ai.ProviderOpenAI, capturedRequest.Provider)
	assert.Equal(t, "gpt-4", capturedRequest.Model)
	assert.Equal(t, "Get all users from database", capturedRequest.Prompt)
	assert.Equal(t, "CREATE TABLE users (id INT, name TEXT)", capturedRequest.Schema)
	assert.Equal(t, 2000, capturedRequest.MaxTokens)
	assert.Equal(t, 0.5, capturedRequest.Temperature)
}

func TestGenerateSQL_EmptyResponse(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Prompt:   "test",
		},
	}

	resp, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Response)
	assert.Equal(t, "", resp.Response.Query)
	assert.Nil(t, resp.Error)
}

func TestGenerateSQL_WithMetadata(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:    "SELECT * FROM orders",
				Metadata: map[string]string{"key1": "value1", "key2": "value2"},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Prompt:   "Get orders",
		},
	}

	resp, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Response)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, resp.Response.Metadata)
}

func TestGenerateSQL_LongPromptTruncation(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	// Create a prompt longer than 100 characters (truncation limit in logs)
	longPrompt := "This is a very long prompt that exceeds the truncation limit of 100 characters and should be truncated in the logs but not in the actual request to the service"

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Prompt:   longPrompt,
		},
	}

	// Should succeed without error
	_, err := server.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
}

// Tests for FixSQL

func TestFixSQL_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:       "SELECT * FROM users WHERE id = 1",
				Explanation: "Fixed syntax error",
				Confidence:  0.90,
				Suggestions: []string{"Consider using prepared statements"},
				Provider:    ai.ProviderAnthropic,
				Model:       "claude-3",
				TokensUsed:  200,
				Metadata:    map[string]string{"fix_type": "syntax"},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider:    aipb.Provider_PROVIDER_ANTHROPIC,
			Model:       "claude-3",
			Query:       "SELECT * FROM users WHERE id = ",
			Error:       "syntax error near end of query",
			Schema:      "CREATE TABLE users (id INT, name TEXT)",
			MaxTokens:   1500,
			Temperature: 0.3,
		},
	}

	resp, err := server.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Response)
	assert.Equal(t, "SELECT * FROM users WHERE id = 1", resp.Response.Query)
	assert.Equal(t, "Fixed syntax error", resp.Response.Explanation)
	assert.Equal(t, float64(0.90), resp.Response.Confidence)
	assert.Equal(t, []string{"Consider using prepared statements"}, resp.Response.Suggestions)
	assert.Equal(t, aipb.Provider_PROVIDER_ANTHROPIC, resp.Response.Provider)
	assert.Equal(t, "claude-3", resp.Response.Model)
	assert.Equal(t, int32(200), resp.Response.TokensUsed)
	assert.NotNil(t, resp.Response.TimeTaken)
	assert.Equal(t, map[string]string{"fix_type": "syntax"}, resp.Response.Metadata)
	assert.Nil(t, resp.Error)
}

func TestFixSQL_ServiceError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, errors.New("provider unavailable")
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_ANTHROPIC,
			Model:    "claude-3",
			Query:    "SELEC * FROM users",
			Error:    "unknown command: SELEC",
		},
	}

	resp, err := server.FixSQL(context.Background(), req)

	require.NoError(t, err) // gRPC method doesn't return error
	require.NotNil(t, resp)
	assert.Nil(t, resp.Response)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "provider unavailable", resp.Error.Message)
}

func TestFixSQL_ProviderConversion(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name             string
		protoProvider    aipb.Provider
		expectedProvider ai.Provider
	}{
		{
			name:             "OpenAI provider",
			protoProvider:    aipb.Provider_PROVIDER_OPENAI,
			expectedProvider: ai.ProviderOpenAI,
		},
		{
			name:             "Ollama provider",
			protoProvider:    aipb.Provider_PROVIDER_OLLAMA,
			expectedProvider: ai.ProviderOllama,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedProvider ai.Provider
			service := &mockGRPCService{
				fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
					capturedProvider = req.Provider
					return &ai.SQLResponse{
						Query:    "SELECT 1",
						Provider: req.Provider,
						Model:    req.Model,
					}, nil
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.FixSQLRequest{
				Request: &aipb.SQLRequest{
					Provider: tc.protoProvider,
					Model:    "test-model",
					Query:    "broken query",
					Error:    "syntax error",
				},
			}

			resp, err := server.FixSQL(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tc.expectedProvider, capturedProvider)
		})
	}
}

func TestFixSQL_RequestFieldValidation(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedRequest *ai.SQLRequest
	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			capturedRequest = req
			return &ai.SQLResponse{
				Query: "SELECT 1",
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider:    aipb.Provider_PROVIDER_ANTHROPIC,
			Model:       "claude-3-opus",
			Query:       "SELECT * FROM users WHERE",
			Error:       "incomplete WHERE clause",
			Schema:      "CREATE TABLE users (id INT)",
			MaxTokens:   500,
			Temperature: 0.2,
		},
	}

	_, err := server.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, capturedRequest)
	assert.Equal(t, ai.ProviderAnthropic, capturedRequest.Provider)
	assert.Equal(t, "claude-3-opus", capturedRequest.Model)
	assert.Equal(t, "SELECT * FROM users WHERE", capturedRequest.Query)
	assert.Equal(t, "incomplete WHERE clause", capturedRequest.Error)
	assert.Equal(t, "CREATE TABLE users (id INT)", capturedRequest.Schema)
	assert.Equal(t, 500, capturedRequest.MaxTokens)
	assert.Equal(t, 0.2, capturedRequest.Temperature)
}

func TestFixSQL_EmptyResponse(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Query:    "broken",
			Error:    "error",
		},
	}

	resp, err := server.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Response)
	assert.Equal(t, "", resp.Response.Query)
	assert.Nil(t, resp.Error)
}

func TestFixSQL_LongErrorTruncation(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	// Create a long error message and query longer than 100 characters
	longError := "This is a very long error message that exceeds the truncation limit of 100 characters and should be truncated in the logs but not in the actual request"
	longQuery := "SELECT * FROM users WHERE name LIKE '%very_long_pattern_that_exceeds_the_truncation_limit_of_100_characters%'"

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Query:    longQuery,
			Error:    longError,
		},
	}

	// Should succeed without error
	_, err := server.FixSQL(context.Background(), req)
	require.NoError(t, err)
}

func TestFixSQL_WithMetadata(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:    "SELECT * FROM products",
				Metadata: map[string]string{"attempt": "1", "category": "fix"},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "gpt-4",
			Query:    "SELECT * FROM product",
			Error:    "table product not found",
		},
	}

	resp, err := server.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Response)
	assert.Equal(t, map[string]string{"attempt": "1", "category": "fix"}, resp.Response.Metadata)
}

// Tests for GetProviderHealth

func TestGetProviderHealth_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getProviderHealthFunc: func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
			return &ai.HealthStatus{
				Provider:     provider,
				Status:       "healthy",
				Message:      "All systems operational",
				LastChecked:  time.Now(),
				ResponseTime: 50 * time.Millisecond,
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderHealthRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
	}

	resp, err := server.GetProviderHealth(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Health)
	assert.Equal(t, aipb.Provider_PROVIDER_OPENAI, resp.Health.Provider)
	assert.Equal(t, aipb.HealthStatus_HEALTH_HEALTHY, resp.Health.Status)
	assert.Equal(t, "All systems operational", resp.Health.Message)
	assert.NotNil(t, resp.Health.LastChecked)
	assert.NotNil(t, resp.Health.ResponseTime)
	assert.Nil(t, resp.Error)
}

func TestGetProviderHealth_ServiceError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getProviderHealthFunc: func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
			return nil, errors.New("health check failed")
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderHealthRequest{
		Provider: aipb.Provider_PROVIDER_ANTHROPIC,
	}

	resp, err := server.GetProviderHealth(context.Background(), req)

	require.NoError(t, err) // gRPC method doesn't return error
	require.NotNil(t, resp)
	assert.Nil(t, resp.Health)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "health check failed", resp.Error.Message)
}

func TestGetProviderHealth_UnhealthyStatus(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getProviderHealthFunc: func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
			return &ai.HealthStatus{
				Provider:     provider,
				Status:       "unhealthy",
				Message:      "API endpoint not responding",
				LastChecked:  time.Now(),
				ResponseTime: 5 * time.Second,
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderHealthRequest{
		Provider: aipb.Provider_PROVIDER_OLLAMA,
	}

	resp, err := server.GetProviderHealth(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Health)
	assert.Equal(t, aipb.HealthStatus_HEALTH_UNHEALTHY, resp.Health.Status)
	assert.Equal(t, "API endpoint not responding", resp.Health.Message)
	assert.Nil(t, resp.Error)
}

func TestGetProviderHealth_ErrorStatus(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getProviderHealthFunc: func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
			return &ai.HealthStatus{
				Provider:    provider,
				Status:      "error",
				Message:     "Connection refused",
				LastChecked: time.Now(),
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderHealthRequest{
		Provider: aipb.Provider_PROVIDER_HUGGINGFACE,
	}

	resp, err := server.GetProviderHealth(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Health)
	assert.Equal(t, aipb.HealthStatus_HEALTH_ERROR, resp.Health.Status)
	assert.Equal(t, "Connection refused", resp.Health.Message)
	assert.Nil(t, resp.Error)
}

func TestGetProviderHealth_UnknownStatus(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getProviderHealthFunc: func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
			return &ai.HealthStatus{
				Provider:    provider,
				Status:      "initializing",
				Message:     "Provider starting up",
				LastChecked: time.Now(),
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderHealthRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
	}

	resp, err := server.GetProviderHealth(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Health)
	assert.Equal(t, aipb.HealthStatus_HEALTH_UNKNOWN, resp.Health.Status)
	assert.Nil(t, resp.Error)
}

// Tests for GetProviderModels

func TestGetProviderModels_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAvailableModelsFunc: func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
			return []ai.ModelInfo{
				{
					ID:           "gpt-4",
					Name:         "GPT-4",
					Provider:     ai.ProviderOpenAI,
					Description:  "Most capable model",
					MaxTokens:    8192,
					Capabilities: []string{"sql", "chat"},
					Metadata:     map[string]string{"version": "0314"},
				},
				{
					ID:           "gpt-3.5-turbo",
					Name:         "GPT-3.5 Turbo",
					Provider:     ai.ProviderOpenAI,
					Description:  "Fast and efficient",
					MaxTokens:    4096,
					Capabilities: []string{"sql", "chat"},
					Metadata:     map[string]string{"version": "0125"},
				},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderModelsRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
	}

	resp, err := server.GetProviderModels(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Models, 2)

	// Check first model
	assert.Equal(t, "gpt-4", resp.Models[0].Id)
	assert.Equal(t, "GPT-4", resp.Models[0].Name)
	assert.Equal(t, aipb.Provider_PROVIDER_OPENAI, resp.Models[0].Provider)
	assert.Equal(t, "Most capable model", resp.Models[0].Description)
	assert.Equal(t, int32(8192), resp.Models[0].MaxTokens)
	assert.Equal(t, []string{"sql", "chat"}, resp.Models[0].Capabilities)
	assert.Equal(t, map[string]string{"version": "0314"}, resp.Models[0].Metadata)

	// Check second model
	assert.Equal(t, "gpt-3.5-turbo", resp.Models[1].Id)
	assert.Equal(t, "GPT-3.5 Turbo", resp.Models[1].Name)

	assert.Nil(t, resp.Error)
}

func TestGetProviderModels_ServiceError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAvailableModelsFunc: func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
			return nil, errors.New("provider not configured")
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderModelsRequest{
		Provider: aipb.Provider_PROVIDER_ANTHROPIC,
	}

	resp, err := server.GetProviderModels(context.Background(), req)

	require.NoError(t, err) // gRPC method doesn't return error
	require.NotNil(t, resp)
	assert.Nil(t, resp.Models)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "provider not configured", resp.Error.Message)
}

func TestGetProviderModels_EmptyList(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAvailableModelsFunc: func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
			return []ai.ModelInfo{}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderModelsRequest{
		Provider: aipb.Provider_PROVIDER_OLLAMA,
	}

	resp, err := server.GetProviderModels(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Models, 0)
	assert.Nil(t, resp.Error)
}

func TestGetProviderModels_MultipleProviders(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name           string
		provider       aipb.Provider
		expectedModels int
	}{
		{
			name:           "OpenAI models",
			provider:       aipb.Provider_PROVIDER_OPENAI,
			expectedModels: 2,
		},
		{
			name:           "Anthropic models",
			provider:       aipb.Provider_PROVIDER_ANTHROPIC,
			expectedModels: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &mockGRPCService{
				getAvailableModelsFunc: func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
					switch provider {
					case ai.ProviderOpenAI:
						return []ai.ModelInfo{
							{ID: "gpt-4", Name: "GPT-4"},
							{ID: "gpt-3.5", Name: "GPT-3.5"},
						}, nil
					case ai.ProviderAnthropic:
						return []ai.ModelInfo{
							{ID: "claude-3", Name: "Claude 3"},
						}, nil
					default:
						return []ai.ModelInfo{}, nil
					}
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GetProviderModelsRequest{
				Provider: tc.provider,
			}

			resp, err := server.GetProviderModels(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Len(t, resp.Models, tc.expectedModels)
		})
	}
}

func TestGetProviderModels_ModelWithoutMetadata(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAvailableModelsFunc: func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
			return []ai.ModelInfo{
				{
					ID:       "simple-model",
					Name:     "Simple Model",
					Provider: ai.ProviderOllama,
					// No description, capabilities, or metadata
				},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderModelsRequest{
		Provider: aipb.Provider_PROVIDER_OLLAMA,
	}

	resp, err := server.GetProviderModels(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Models, 1)
	assert.Equal(t, "simple-model", resp.Models[0].Id)
	assert.Equal(t, "", resp.Models[0].Description)
	assert.Nil(t, resp.Models[0].Capabilities)
	assert.Nil(t, resp.Models[0].Metadata)
}

// Tests for TestProvider

func TestTestProvider_OpenAI_Success(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedConfig interface{}
	service := &mockGRPCService{
		testProviderFunc: func(ctx context.Context, provider ai.Provider, config interface{}) error {
			capturedConfig = config
			return nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.TestProviderRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
		Config: &aipb.TestProviderRequest_OpenaiConfig{
			OpenaiConfig: &aipb.OpenAIConfig{
				ApiKey: "test-api-key-123",
			},
		},
	}

	resp, err := server.TestProvider(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)

	// Verify config was passed correctly
	require.NotNil(t, capturedConfig)
	openaiConfig, ok := capturedConfig.(*ai.OpenAIConfig)
	require.True(t, ok)
	assert.Equal(t, "test-api-key-123", openaiConfig.APIKey)
}

func TestTestProvider_Anthropic_Success(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedConfig interface{}
	service := &mockGRPCService{
		testProviderFunc: func(ctx context.Context, provider ai.Provider, config interface{}) error {
			capturedConfig = config
			return nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.TestProviderRequest{
		Provider: aipb.Provider_PROVIDER_ANTHROPIC,
		Config: &aipb.TestProviderRequest_AnthropicConfig{
			AnthropicConfig: &aipb.AnthropicConfig{
				ApiKey: "sk-ant-test-key",
			},
		},
	}

	resp, err := server.TestProvider(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)

	// Verify config was passed correctly
	require.NotNil(t, capturedConfig)
	anthropicConfig, ok := capturedConfig.(*ai.AnthropicConfig)
	require.True(t, ok)
	assert.Equal(t, "sk-ant-test-key", anthropicConfig.APIKey)
}

func TestTestProvider_Ollama_Success(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedConfig interface{}
	service := &mockGRPCService{
		testProviderFunc: func(ctx context.Context, provider ai.Provider, config interface{}) error {
			capturedConfig = config
			return nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.TestProviderRequest{
		Provider: aipb.Provider_PROVIDER_OLLAMA,
		Config: &aipb.TestProviderRequest_OllamaConfig{
			OllamaConfig: &aipb.OllamaConfig{
				Endpoint: "http://localhost:11434",
			},
		},
	}

	resp, err := server.TestProvider(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)

	// Verify config was passed correctly
	require.NotNil(t, capturedConfig)
	ollamaConfig, ok := capturedConfig.(*ai.OllamaConfig)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:11434", ollamaConfig.Endpoint)
}

func TestTestProvider_HuggingFace_Success(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedConfig interface{}
	service := &mockGRPCService{
		testProviderFunc: func(ctx context.Context, provider ai.Provider, config interface{}) error {
			capturedConfig = config
			return nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.TestProviderRequest{
		Provider: aipb.Provider_PROVIDER_HUGGINGFACE,
		Config: &aipb.TestProviderRequest_HuggingfaceConfig{
			HuggingfaceConfig: &aipb.HuggingFaceConfig{
				Endpoint: "http://localhost:8080",
			},
		},
	}

	resp, err := server.TestProvider(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)

	// Verify config was passed correctly
	require.NotNil(t, capturedConfig)
	hfConfig, ok := capturedConfig.(*ai.HuggingFaceConfig)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:8080", hfConfig.Endpoint)
}

func TestTestProvider_ServiceError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		testProviderFunc: func(ctx context.Context, provider ai.Provider, config interface{}) error {
			return errors.New("authentication failed")
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.TestProviderRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
		Config: &aipb.TestProviderRequest_OpenaiConfig{
			OpenaiConfig: &aipb.OpenAIConfig{
				ApiKey: "invalid-key",
			},
		},
	}

	resp, err := server.TestProvider(context.Background(), req)

	require.NoError(t, err) // gRPC method doesn't return error
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "authentication failed", resp.Error.Message)
}

func TestTestProvider_WithoutConfig(t *testing.T) {
	logger := createGRPCTestLogger()

	var capturedConfig interface{}
	service := &mockGRPCService{
		testProviderFunc: func(ctx context.Context, provider ai.Provider, config interface{}) error {
			capturedConfig = config
			return nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.TestProviderRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
		// No config provided
	}

	resp, err := server.TestProvider(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)

	// Should receive nil config
	assert.Nil(t, capturedConfig)
}

// Tests for GetUsageStats

func TestGetUsageStats_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAllUsageStatsFunc: func(ctx context.Context) (map[ai.Provider]*ai.Usage, error) {
			return map[ai.Provider]*ai.Usage{
				ai.ProviderOpenAI: {
					Provider:        ai.ProviderOpenAI,
					Model:           "gpt-4",
					RequestCount:    100,
					TokensUsed:      50000,
					SuccessRate:     0.98,
					LastUsed:        time.Now(),
					AvgResponseTime: 500 * time.Millisecond,
				},
				ai.ProviderAnthropic: {
					Provider:        ai.ProviderAnthropic,
					Model:           "claude-3",
					RequestCount:    50,
					TokensUsed:      25000,
					SuccessRate:     0.99,
					LastUsed:        time.Now(),
					AvgResponseTime: 300 * time.Millisecond,
				},
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetUsageStatsRequest{}

	resp, err := server.GetUsageStats(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.UsageStats, 2)

	// Check OpenAI stats
	openaiStats := resp.UsageStats["openai"]
	require.NotNil(t, openaiStats)
	assert.Equal(t, aipb.Provider_PROVIDER_OPENAI, openaiStats.Provider)
	assert.Equal(t, "gpt-4", openaiStats.Model)
	assert.Equal(t, int64(100), openaiStats.RequestCount)
	assert.Equal(t, int64(50000), openaiStats.TokensUsed)
	assert.Equal(t, float64(0.98), openaiStats.SuccessRate)
	assert.NotNil(t, openaiStats.LastUsed)
	assert.NotNil(t, openaiStats.AvgResponseTime)

	// Check Anthropic stats
	anthropicStats := resp.UsageStats["anthropic"]
	require.NotNil(t, anthropicStats)
	assert.Equal(t, aipb.Provider_PROVIDER_ANTHROPIC, anthropicStats.Provider)
	assert.Equal(t, "claude-3", anthropicStats.Model)

	assert.Nil(t, resp.Error)
}

func TestGetUsageStats_ServiceError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAllUsageStatsFunc: func(ctx context.Context) (map[ai.Provider]*ai.Usage, error) {
			return nil, errors.New("database connection failed")
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetUsageStatsRequest{}

	resp, err := server.GetUsageStats(context.Background(), req)

	require.NoError(t, err) // gRPC method doesn't return error
	require.NotNil(t, resp)
	assert.Nil(t, resp.UsageStats)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "database connection failed", resp.Error.Message)
}

func TestGetUsageStats_EmptyStats(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getAllUsageStatsFunc: func(ctx context.Context) (map[ai.Provider]*ai.Usage, error) {
			return map[ai.Provider]*ai.Usage{}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetUsageStatsRequest{}

	resp, err := server.GetUsageStats(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.UsageStats, 0)
	assert.Nil(t, resp.Error)
}

// Tests for GetConfig

func TestGetConfig_Success(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getConfigFunc: func() *ai.Config {
			return &ai.Config{
				DefaultProvider: ai.ProviderOpenAI,
				MaxTokens:       2000,
				Temperature:     0.7,
				RequestTimeout:  30 * time.Second,
				RateLimitPerMin: 60,
				OpenAI: ai.OpenAIConfig{
					APIKey: "test-openai-key",
				},
				Anthropic: ai.AnthropicConfig{
					APIKey: "test-anthropic-key",
				},
				Ollama: ai.OllamaConfig{
					Endpoint: "http://localhost:11434",
				},
				HuggingFace: ai.HuggingFaceConfig{
					Endpoint:         "http://localhost:8080",
					Models:           []string{"model1", "model2"},
					PullTimeout:      5 * time.Minute,
					GenerateTimeout:  2 * time.Minute,
					AutoPullModels:   true,
					RecommendedModel: "model1",
				},
			}
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetConfigRequest{}

	resp, err := server.GetConfig(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Config)
	assert.Equal(t, aipb.Provider_PROVIDER_OPENAI, resp.Config.DefaultProvider)
	assert.Equal(t, int32(2000), resp.Config.MaxTokens)
	assert.Equal(t, float64(0.7), resp.Config.Temperature)
	assert.NotNil(t, resp.Config.RequestTimeout)
	assert.Equal(t, int32(60), resp.Config.RateLimitPerMin)

	// Check provider configs
	assert.NotNil(t, resp.Config.Openai)
	assert.Equal(t, "test-openai-key", resp.Config.Openai.ApiKey)

	assert.NotNil(t, resp.Config.Anthropic)
	assert.Equal(t, "test-anthropic-key", resp.Config.Anthropic.ApiKey)

	assert.NotNil(t, resp.Config.Ollama)
	assert.Equal(t, "http://localhost:11434", resp.Config.Ollama.Endpoint)

	assert.NotNil(t, resp.Config.Huggingface)
	assert.Equal(t, "http://localhost:8080", resp.Config.Huggingface.Endpoint)
	assert.Equal(t, []string{"model1", "model2"}, resp.Config.Huggingface.Models)
	assert.True(t, resp.Config.Huggingface.AutoPullModels)
	assert.Equal(t, "model1", resp.Config.Huggingface.RecommendedModel)
	assert.True(t, resp.Config.Huggingface.Configured)
}

func TestGetConfig_EmptyProviderConfigs(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getConfigFunc: func() *ai.Config {
			return &ai.Config{
				DefaultProvider: ai.ProviderOpenAI,
				MaxTokens:       1000,
				Temperature:     0.5,
				RequestTimeout:  10 * time.Second,
				RateLimitPerMin: 30,
				// Empty provider configs
			}
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetConfigRequest{}

	resp, err := server.GetConfig(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Config)

	// Provider configs should be nil when API keys/endpoints are empty
	assert.Nil(t, resp.Config.Openai)
	assert.Nil(t, resp.Config.Anthropic)
	assert.Nil(t, resp.Config.Ollama)
	assert.Nil(t, resp.Config.Huggingface)
}

func TestGetConfig_PartialProviderConfigs(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		getConfigFunc: func() *ai.Config {
			return &ai.Config{
				DefaultProvider: ai.ProviderAnthropic,
				MaxTokens:       1500,
				Temperature:     0.3,
				RequestTimeout:  20 * time.Second,
				RateLimitPerMin: 40,
				Anthropic: ai.AnthropicConfig{
					APIKey: "sk-ant-key",
				},
				// Only Anthropic configured
			}
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetConfigRequest{}

	resp, err := server.GetConfig(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Config)

	// Only Anthropic should be present
	assert.NotNil(t, resp.Config.Anthropic)
	assert.Equal(t, "sk-ant-key", resp.Config.Anthropic.ApiKey)

	// Others should be nil
	assert.Nil(t, resp.Config.Openai)
	assert.Nil(t, resp.Config.Ollama)
	assert.Nil(t, resp.Config.Huggingface)
}

// Tests for conversion helper functions
// Note: These tests use the internal conversion functions through the gRPC methods
// since the conversion functions are not exported

func TestProtoToProvider_AllProviders(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name             string
		protoProvider    aipb.Provider
		expectedProvider ai.Provider
	}{
		{
			name:             "OpenAI",
			protoProvider:    aipb.Provider_PROVIDER_OPENAI,
			expectedProvider: ai.ProviderOpenAI,
		},
		{
			name:             "Anthropic",
			protoProvider:    aipb.Provider_PROVIDER_ANTHROPIC,
			expectedProvider: ai.ProviderAnthropic,
		},
		{
			name:             "Ollama",
			protoProvider:    aipb.Provider_PROVIDER_OLLAMA,
			expectedProvider: ai.ProviderOllama,
		},
		{
			name:             "HuggingFace",
			protoProvider:    aipb.Provider_PROVIDER_HUGGINGFACE,
			expectedProvider: ai.ProviderHuggingFace,
		},
		{
			name:             "Unspecified defaults to OpenAI",
			protoProvider:    aipb.Provider_PROVIDER_UNSPECIFIED,
			expectedProvider: ai.ProviderOpenAI,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedProvider ai.Provider
			service := &mockGRPCService{
				generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
					capturedProvider = req.Provider
					return &ai.SQLResponse{Query: "SELECT 1"}, nil
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GenerateSQLRequest{
				Request: &aipb.SQLRequest{
					Provider: tc.protoProvider,
					Model:    "test-model",
					Prompt:   "test",
				},
			}

			_, err := server.GenerateSQL(context.Background(), req)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedProvider, capturedProvider)
		})
	}
}

func TestProviderToProto_AllProviders(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name          string
		provider      ai.Provider
		expectedProto aipb.Provider
	}{
		{
			name:          "OpenAI",
			provider:      ai.ProviderOpenAI,
			expectedProto: aipb.Provider_PROVIDER_OPENAI,
		},
		{
			name:          "Anthropic",
			provider:      ai.ProviderAnthropic,
			expectedProto: aipb.Provider_PROVIDER_ANTHROPIC,
		},
		{
			name:          "Ollama",
			provider:      ai.ProviderOllama,
			expectedProto: aipb.Provider_PROVIDER_OLLAMA,
		},
		{
			name:          "HuggingFace",
			provider:      ai.ProviderHuggingFace,
			expectedProto: aipb.Provider_PROVIDER_HUGGINGFACE,
		},
		{
			name:          "Unknown provider defaults to unspecified",
			provider:      ai.Provider("unknown"),
			expectedProto: aipb.Provider_PROVIDER_UNSPECIFIED,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &mockGRPCService{
				generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
					return &ai.SQLResponse{
						Query:    "SELECT 1",
						Provider: tc.provider,
						Model:    "test-model",
					}, nil
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GenerateSQLRequest{
				Request: &aipb.SQLRequest{
					Provider: aipb.Provider_PROVIDER_OPENAI,
					Model:    "test-model",
					Prompt:   "test",
				},
			}

			resp, err := server.GenerateSQL(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Response)
			assert.Equal(t, tc.expectedProto, resp.Response.Provider)
		})
	}
}

func TestHealthStatusToProto_AllStatuses(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name          string
		status        string
		expectedProto aipb.HealthStatus
	}{
		{
			name:          "Healthy",
			status:        "healthy",
			expectedProto: aipb.HealthStatus_HEALTH_HEALTHY,
		},
		{
			name:          "Unhealthy",
			status:        "unhealthy",
			expectedProto: aipb.HealthStatus_HEALTH_UNHEALTHY,
		},
		{
			name:          "Error",
			status:        "error",
			expectedProto: aipb.HealthStatus_HEALTH_ERROR,
		},
		{
			name:          "Unknown",
			status:        "unknown",
			expectedProto: aipb.HealthStatus_HEALTH_UNKNOWN,
		},
		{
			name:          "Invalid status defaults to unknown",
			status:        "invalid",
			expectedProto: aipb.HealthStatus_HEALTH_UNKNOWN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &mockGRPCService{
				getProviderHealthFunc: func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
					return &ai.HealthStatus{
						Provider:     provider,
						Status:       tc.status,
						Message:      "Test message",
						LastChecked:  time.Now(),
						ResponseTime: 100 * time.Millisecond,
					}, nil
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GetProviderHealthRequest{
				Provider: aipb.Provider_PROVIDER_OPENAI,
			}

			resp, err := server.GetProviderHealth(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Health)
			assert.Equal(t, tc.expectedProto, resp.Health.Status)
		})
	}
}

func TestErrorToProto_NilError(t *testing.T) {
	logger := createGRPCTestLogger()
	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   "test",
		},
	}

	resp, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestErrorToProto_WithError(t *testing.T) {
	logger := createGRPCTestLogger()

	testCases := []struct {
		name         string
		errorMessage string
	}{
		{
			name:         "Simple error",
			errorMessage: "simple error",
		},
		{
			name:         "Long error message",
			errorMessage: "This is a very long error message that contains detailed information about what went wrong in the system",
		},
		{
			name:         "Error with special characters",
			errorMessage: "Error: connection failed @ 127.0.0.1:8080 (errno: 111)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &mockGRPCService{
				generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
					return nil, errors.New(tc.errorMessage)
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GenerateSQLRequest{
				Request: &aipb.SQLRequest{
					Provider: aipb.Provider_PROVIDER_OPENAI,
					Model:    "test-model",
					Prompt:   "test",
				},
			}

			resp, err := server.GenerateSQL(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tc.errorMessage, resp.Error.Message)
			assert.Equal(t, aipb.ErrorType_ERROR_UNKNOWN, resp.Error.Type)
			assert.False(t, resp.Error.Retryable)
		})
	}
}

func TestTruncateString_BelowLimit(t *testing.T) {
	logger := createGRPCTestLogger()

	shortPrompt := "Short prompt"

	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			// Verify the full prompt is received (not truncated)
			assert.Equal(t, shortPrompt, req.Prompt)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   shortPrompt,
		},
	}

	_, err := server.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
}

func TestTruncateString_AtLimit(t *testing.T) {
	logger := createGRPCTestLogger()

	// Exactly 100 characters (truncation limit in logs)
	exactPrompt := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"

	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			// Verify the full prompt is received (not truncated)
			assert.Equal(t, exactPrompt, req.Prompt)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   exactPrompt,
		},
	}

	_, err := server.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
}

func TestTruncateString_AboveLimit(t *testing.T) {
	logger := createGRPCTestLogger()

	// More than 100 characters (truncation limit in logs)
	longPrompt := "This is a very long prompt that exceeds 100 characters and will be truncated in the logs but the service should still receive the full prompt without any truncation applied"

	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			// Verify the FULL prompt is received (truncation only happens in logs)
			assert.Equal(t, longPrompt, req.Prompt)
			assert.Greater(t, len(req.Prompt), 100)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   longPrompt,
		},
	}

	_, err := server.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
}

func TestTruncateString_EmptyString(t *testing.T) {
	logger := createGRPCTestLogger()

	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			assert.Equal(t, "", req.Prompt)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   "",
		},
	}

	_, err := server.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
}

// Additional edge case tests

func TestGenerateSQL_ContextCancellation(t *testing.T) {
	logger := createGRPCTestLogger()

	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &ai.SQLResponse{Query: "SELECT 1"}, nil
			}
		},
	}

	server := ai.NewGRPCServer(service, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   "test",
		},
	}

	resp, err := server.GenerateSQL(ctx, req)

	require.NoError(t, err) // gRPC method doesn't return error
	require.NotNil(t, resp)
	// Context cancellation error should be in the error field
	assert.NotNil(t, resp.Error)
}

func TestFixSQL_ContextCancellation(t *testing.T) {
	logger := createGRPCTestLogger()

	service := &mockGRPCService{
		fixSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &ai.SQLResponse{Query: "SELECT 1"}, nil
			}
		},
	}

	server := ai.NewGRPCServer(service, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &aipb.FixSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Query:    "broken",
			Error:    "error",
		},
	}

	resp, err := server.FixSQL(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
}

func TestGetProviderHealth_AllProviders(t *testing.T) {
	logger := createGRPCTestLogger()

	providers := []aipb.Provider{
		aipb.Provider_PROVIDER_OPENAI,
		aipb.Provider_PROVIDER_ANTHROPIC,
		aipb.Provider_PROVIDER_OLLAMA,
		aipb.Provider_PROVIDER_HUGGINGFACE,
	}

	for _, provider := range providers {
		t.Run(provider.String(), func(t *testing.T) {
			var capturedProvider ai.Provider
			service := &mockGRPCService{
				getProviderHealthFunc: func(ctx context.Context, p ai.Provider) (*ai.HealthStatus, error) {
					capturedProvider = p
					return &ai.HealthStatus{
						Provider:     p,
						Status:       "healthy",
						Message:      "OK",
						LastChecked:  time.Now(),
						ResponseTime: 100 * time.Millisecond,
					}, nil
				},
			}

			server := ai.NewGRPCServer(service, logger)

			req := &aipb.GetProviderHealthRequest{
				Provider: provider,
			}

			resp, err := server.GetProviderHealth(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Health)
			assert.NotNil(t, capturedProvider)
		})
	}
}

func TestGetProviderModels_LargeModelList(t *testing.T) {
	logger := createGRPCTestLogger()

	// Create a large list of models
	largeModelList := make([]ai.ModelInfo, 100)
	for i := 0; i < 100; i++ {
		largeModelList[i] = ai.ModelInfo{
			ID:       "model-" + string(rune(i)),
			Name:     "Model " + string(rune(i)),
			Provider: ai.ProviderOpenAI,
		}
	}

	service := &mockGRPCService{
		getAvailableModelsFunc: func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
			return largeModelList, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GetProviderModelsRequest{
		Provider: aipb.Provider_PROVIDER_OPENAI,
	}

	resp, err := server.GetProviderModels(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Models, 100)
}

func TestTimeTakenDuration(t *testing.T) {
	logger := createGRPCTestLogger()

	service := &mockGRPCService{
		generateSQLFunc: func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			return &ai.SQLResponse{
				Query: "SELECT 1",
			}, nil
		},
	}

	server := ai.NewGRPCServer(service, logger)

	req := &aipb.GenerateSQLRequest{
		Request: &aipb.SQLRequest{
			Provider: aipb.Provider_PROVIDER_OPENAI,
			Model:    "test-model",
			Prompt:   "test",
		},
	}

	resp, err := server.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Response)
	assert.NotNil(t, resp.Response.TimeTaken)

	// Verify time taken is reasonable (at least 10ms since we slept)
	duration := resp.Response.TimeTaken.AsDuration()
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}
