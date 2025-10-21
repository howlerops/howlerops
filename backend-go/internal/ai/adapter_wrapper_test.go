package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProviderAdapter is a mock implementation of ProviderAdapter for testing
type mockProviderAdapter struct {
	generateSQLFunc   func(ctx context.Context, prompt, schema string, opts ...GenerateOption) (*SQLResponse, error)
	fixSQLFunc        func(ctx context.Context, query, errorMsg, schema string, opts ...GenerateOption) (*SQLResponse, error)
	chatFunc          func(ctx context.Context, prompt string, opts ...GenerateOption) (*ChatResponse, error)
	getHealthFunc     func(ctx context.Context) (*HealthStatus, error)
	listModelsFunc    func(ctx context.Context) ([]ModelInfo, error)
	getProviderFunc   func() Provider
	closeFunc         func() error
	capturedOptions   []GenerateOptions // Stores all options received
}

func (m *mockProviderAdapter) GenerateSQL(ctx context.Context, prompt, schema string, opts ...GenerateOption) (*SQLResponse, error) {
	// Capture options for verification
	options := &GenerateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	m.capturedOptions = append(m.capturedOptions, *options)

	if m.generateSQLFunc != nil {
		return m.generateSQLFunc(ctx, prompt, schema, opts...)
	}
	return &SQLResponse{Query: "SELECT 1"}, nil
}

func (m *mockProviderAdapter) FixSQL(ctx context.Context, query, errorMsg, schema string, opts ...GenerateOption) (*SQLResponse, error) {
	// Capture options for verification
	options := &GenerateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	m.capturedOptions = append(m.capturedOptions, *options)

	if m.fixSQLFunc != nil {
		return m.fixSQLFunc(ctx, query, errorMsg, schema, opts...)
	}
	return &SQLResponse{Query: "SELECT 1 FROM fixed"}, nil
}

func (m *mockProviderAdapter) Chat(ctx context.Context, prompt string, opts ...GenerateOption) (*ChatResponse, error) {
	// Capture options for verification
	options := &GenerateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	m.capturedOptions = append(m.capturedOptions, *options)

	if m.chatFunc != nil {
		return m.chatFunc(ctx, prompt, opts...)
	}
	return &ChatResponse{Content: "test response"}, nil
}

func (m *mockProviderAdapter) GetHealth(ctx context.Context) (*HealthStatus, error) {
	if m.getHealthFunc != nil {
		return m.getHealthFunc(ctx)
	}
	return &HealthStatus{Status: "healthy", Provider: ProviderOpenAI}, nil
}

func (m *mockProviderAdapter) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if m.listModelsFunc != nil {
		return m.listModelsFunc(ctx)
	}
	return []ModelInfo{{ID: "model-1", Provider: ProviderOpenAI}}, nil
}

func (m *mockProviderAdapter) GetProviderType() Provider {
	if m.getProviderFunc != nil {
		return m.getProviderFunc()
	}
	return ProviderOpenAI
}

func (m *mockProviderAdapter) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestProviderAdapterWrapper_GenerateSQL_OptionConversion(t *testing.T) {
	tests := []struct {
		name     string
		request  *SQLRequest
		expected GenerateOptions
	}{
		{
			name: "all options set",
			request: &SQLRequest{
				Prompt:      "generate query",
				Schema:      "CREATE TABLE users",
				Model:       "gpt-4",
				MaxTokens:   1000,
				Temperature: 0.7,
				Context: map[string]string{
					"db_type": "postgres",
					"version": "14",
				},
			},
			expected: GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   1000,
				Temperature: 0.7,
				Context: map[string]string{
					"db_type": "postgres",
					"version": "14",
				},
			},
		},
		{
			name: "minimal options",
			request: &SQLRequest{
				Prompt: "simple query",
				Schema: "CREATE TABLE items",
			},
			expected: GenerateOptions{
				Model:       "",
				MaxTokens:   0,
				Temperature: 0,
				Context:     nil, // nil context when not provided
			},
		},
		{
			name: "only temperature set",
			request: &SQLRequest{
				Prompt:      "query with temperature",
				Temperature: 0.9,
			},
			expected: GenerateOptions{
				Temperature: 0.9,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProviderAdapter{}
			wrapper := &providerAdapterWrapper{
				adapter: mock,
				logger:  logrus.New(),
			}

			ctx := context.Background()
			_, err := wrapper.GenerateSQL(ctx, tt.request)
			require.NoError(t, err)

			// Verify captured options
			require.Len(t, mock.capturedOptions, 1)
			captured := mock.capturedOptions[0]

			assert.Equal(t, tt.expected.Model, captured.Model, "model mismatch")
			assert.Equal(t, tt.expected.MaxTokens, captured.MaxTokens, "maxTokens mismatch")
			assert.Equal(t, tt.expected.Temperature, captured.Temperature, "temperature mismatch")

			// Verify context
			if tt.expected.Context != nil {
				assert.Equal(t, tt.expected.Context, captured.Context, "context mismatch")
			} else if len(captured.Context) == 0 {
				// Both nil and empty map are acceptable for empty context
				assert.True(t, captured.Context == nil || len(captured.Context) == 0, "context should be nil or empty")
			}
		})
	}
}

func TestProviderAdapterWrapper_GenerateSQL_AdapterCall(t *testing.T) {
	expectedPrompt := "generate a query"
	expectedSchema := "CREATE TABLE users (id INT)"
	called := false

	mock := &mockProviderAdapter{
		generateSQLFunc: func(ctx context.Context, prompt, schema string, opts ...GenerateOption) (*SQLResponse, error) {
			called = true
			assert.Equal(t, expectedPrompt, prompt)
			assert.Equal(t, expectedSchema, schema)
			return &SQLResponse{
				Query:       "SELECT * FROM users",
				Explanation: "retrieves all users",
				Confidence:  0.95,
			}, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	req := &SQLRequest{
		Prompt: expectedPrompt,
		Schema: expectedSchema,
		Model:  "gpt-4",
	}

	ctx := context.Background()
	resp, err := wrapper.GenerateSQL(ctx, req)

	require.NoError(t, err)
	assert.True(t, called, "adapter method not called")
	assert.Equal(t, "SELECT * FROM users", resp.Query)
	assert.Equal(t, "retrieves all users", resp.Explanation)
	assert.Equal(t, 0.95, resp.Confidence)
}

func TestProviderAdapterWrapper_GenerateSQL_Error(t *testing.T) {
	expectedErr := errors.New("generation failed")

	mock := &mockProviderAdapter{
		generateSQLFunc: func(ctx context.Context, prompt, schema string, opts ...GenerateOption) (*SQLResponse, error) {
			return nil, expectedErr
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	req := &SQLRequest{
		Prompt: "test",
		Schema: "schema",
	}

	ctx := context.Background()
	resp, err := wrapper.GenerateSQL(ctx, req)

	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
}

func TestProviderAdapterWrapper_FixSQL_OptionConversion(t *testing.T) {
	tests := []struct {
		name     string
		request  *SQLRequest
		expected GenerateOptions
	}{
		{
			name: "fix with full options",
			request: &SQLRequest{
				Query:       "SELECT * FORM users",
				Error:       "syntax error at 'FORM'",
				Schema:      "CREATE TABLE users",
				Model:       "gpt-3.5-turbo",
				MaxTokens:   500,
				Temperature: 0.5,
				Context: map[string]string{
					"error_type": "syntax",
				},
			},
			expected: GenerateOptions{
				Model:       "gpt-3.5-turbo",
				MaxTokens:   500,
				Temperature: 0.5,
				Context: map[string]string{
					"error_type": "syntax",
				},
			},
		},
		{
			name: "fix with minimal options",
			request: &SQLRequest{
				Query: "SELECT * FORM users",
				Error: "syntax error",
			},
			expected: GenerateOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProviderAdapter{}
			wrapper := &providerAdapterWrapper{
				adapter: mock,
				logger:  logrus.New(),
			}

			ctx := context.Background()
			_, err := wrapper.FixSQL(ctx, tt.request)
			require.NoError(t, err)

			// Verify captured options
			require.Len(t, mock.capturedOptions, 1)
			captured := mock.capturedOptions[0]

			assert.Equal(t, tt.expected.Model, captured.Model)
			assert.Equal(t, tt.expected.MaxTokens, captured.MaxTokens)
			assert.Equal(t, tt.expected.Temperature, captured.Temperature)

			if tt.expected.Context != nil {
				assert.Equal(t, tt.expected.Context, captured.Context)
			}
		})
	}
}

func TestProviderAdapterWrapper_FixSQL_AdapterCall(t *testing.T) {
	expectedQuery := "SELECT * FORM users"
	expectedError := "syntax error at 'FORM'"
	expectedSchema := "CREATE TABLE users (id INT)"
	called := false

	mock := &mockProviderAdapter{
		fixSQLFunc: func(ctx context.Context, query, errorMsg, schema string, opts ...GenerateOption) (*SQLResponse, error) {
			called = true
			assert.Equal(t, expectedQuery, query)
			assert.Equal(t, expectedError, errorMsg)
			assert.Equal(t, expectedSchema, schema)
			return &SQLResponse{
				Query:       "SELECT * FROM users",
				Explanation: "fixed syntax error",
				Confidence:  0.98,
			}, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	req := &SQLRequest{
		Query:  expectedQuery,
		Error:  expectedError,
		Schema: expectedSchema,
	}

	ctx := context.Background()
	resp, err := wrapper.FixSQL(ctx, req)

	require.NoError(t, err)
	assert.True(t, called, "adapter method not called")
	assert.Equal(t, "SELECT * FROM users", resp.Query)
	assert.Equal(t, "fixed syntax error", resp.Explanation)
}

func TestProviderAdapterWrapper_Chat_OptionConversion(t *testing.T) {
	tests := []struct {
		name     string
		request  *ChatRequest
		expected GenerateOptions
	}{
		{
			name: "chat with basic options",
			request: &ChatRequest{
				Prompt:      "hello",
				Model:       "gpt-4",
				MaxTokens:   2000,
				Temperature: 0.8,
			},
			expected: GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   2000,
				Temperature: 0.8,
				Context:     nil, // nil when no context fields are set
			},
		},
		{
			name: "chat with context field",
			request: &ChatRequest{
				Prompt:      "explain this code",
				Context:     "code snippet here",
				Model:       "gpt-3.5-turbo",
				MaxTokens:   1500,
				Temperature: 0.6,
			},
			expected: GenerateOptions{
				Model:       "gpt-3.5-turbo",
				MaxTokens:   1500,
				Temperature: 0.6,
				Context: map[string]string{
					"context": "code snippet here",
				},
			},
		},
		{
			name: "chat with system field",
			request: &ChatRequest{
				Prompt:      "write a function",
				System:      "you are a Go expert",
				Model:       "gpt-4",
				MaxTokens:   3000,
				Temperature: 0.7,
			},
			expected: GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   3000,
				Temperature: 0.7,
				Context: map[string]string{
					"system": "you are a Go expert",
				},
			},
		},
		{
			name: "chat with metadata",
			request: &ChatRequest{
				Prompt:      "analyze this",
				Model:       "gpt-4",
				MaxTokens:   1000,
				Temperature: 0.5,
				Metadata: map[string]string{
					"user_id":   "123",
					"session":   "abc",
					"task_type": "analysis",
				},
			},
			expected: GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   1000,
				Temperature: 0.5,
				Context: map[string]string{
					"user_id":   "123",
					"session":   "abc",
					"task_type": "analysis",
				},
			},
		},
		{
			name: "chat with all context fields",
			request: &ChatRequest{
				Prompt:      "complex request",
				Context:     "additional context",
				System:      "system instructions",
				Model:       "gpt-4",
				MaxTokens:   2500,
				Temperature: 0.75,
				Metadata: map[string]string{
					"priority": "high",
					"category": "technical",
				},
			},
			expected: GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   2500,
				Temperature: 0.75,
				Context: map[string]string{
					"context":  "additional context",
					"system":   "system instructions",
					"priority": "high",
					"category": "technical",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProviderAdapter{}
			wrapper := &providerAdapterWrapper{
				adapter: mock,
				logger:  logrus.New(),
			}

			ctx := context.Background()
			_, err := wrapper.Chat(ctx, tt.request)
			require.NoError(t, err)

			// Verify captured options
			require.Len(t, mock.capturedOptions, 1)
			captured := mock.capturedOptions[0]

			assert.Equal(t, tt.expected.Model, captured.Model)
			assert.Equal(t, tt.expected.MaxTokens, captured.MaxTokens)
			assert.Equal(t, tt.expected.Temperature, captured.Temperature)

			// Verify context mapping
			if len(tt.expected.Context) > 0 {
				assert.Equal(t, tt.expected.Context, captured.Context, "context map mismatch")
			}
		})
	}
}

func TestProviderAdapterWrapper_Chat_EmptyContextNotAdded(t *testing.T) {
	// Test that empty context fields don't get added to the context map
	mock := &mockProviderAdapter{}
	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	req := &ChatRequest{
		Prompt:      "test",
		Context:     "", // empty
		System:      "", // empty
		Model:       "gpt-4",
		MaxTokens:   1000,
		Temperature: 0.5,
		Metadata:    nil, // nil
	}

	ctx := context.Background()
	_, err := wrapper.Chat(ctx, req)
	require.NoError(t, err)

	// Verify no context was added
	require.Len(t, mock.capturedOptions, 1)
	captured := mock.capturedOptions[0]

	// Context should be nil or empty when no context fields are set
	assert.Empty(t, captured.Context, "context should be empty when all fields are empty")
}

func TestProviderAdapterWrapper_Chat_AdapterCall(t *testing.T) {
	expectedPrompt := "explain golang concurrency"
	called := false

	mock := &mockProviderAdapter{
		chatFunc: func(ctx context.Context, prompt string, opts ...GenerateOption) (*ChatResponse, error) {
			called = true
			assert.Equal(t, expectedPrompt, prompt)
			return &ChatResponse{
				Content:    "Golang uses goroutines...",
				Provider:   ProviderOpenAI,
				Model:      "gpt-4",
				TokensUsed: 150,
			}, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	req := &ChatRequest{
		Prompt: expectedPrompt,
		Model:  "gpt-4",
	}

	ctx := context.Background()
	resp, err := wrapper.Chat(ctx, req)

	require.NoError(t, err)
	assert.True(t, called, "adapter method not called")
	assert.Equal(t, "Golang uses goroutines...", resp.Content)
	assert.Equal(t, ProviderOpenAI, resp.Provider)
	assert.Equal(t, 150, resp.TokensUsed)
}

func TestProviderAdapterWrapper_Chat_Error(t *testing.T) {
	expectedErr := errors.New("chat failed")

	mock := &mockProviderAdapter{
		chatFunc: func(ctx context.Context, prompt string, opts ...GenerateOption) (*ChatResponse, error) {
			return nil, expectedErr
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	req := &ChatRequest{
		Prompt: "test",
	}

	ctx := context.Background()
	resp, err := wrapper.Chat(ctx, req)

	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
}

func TestProviderAdapterWrapper_HealthCheck(t *testing.T) {
	expectedHealth := &HealthStatus{
		Provider:     ProviderAnthropic,
		Status:       "healthy",
		Message:      "all systems operational",
		LastChecked:  time.Now(),
		ResponseTime: 100 * time.Millisecond,
	}

	mock := &mockProviderAdapter{
		getHealthFunc: func(ctx context.Context) (*HealthStatus, error) {
			return expectedHealth, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	health, err := wrapper.HealthCheck(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedHealth.Provider, health.Provider)
	assert.Equal(t, expectedHealth.Status, health.Status)
	assert.Equal(t, expectedHealth.Message, health.Message)
	assert.Equal(t, expectedHealth.ResponseTime, health.ResponseTime)
}

func TestProviderAdapterWrapper_HealthCheck_Error(t *testing.T) {
	expectedErr := errors.New("health check failed")

	mock := &mockProviderAdapter{
		getHealthFunc: func(ctx context.Context) (*HealthStatus, error) {
			return nil, expectedErr
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	health, err := wrapper.HealthCheck(ctx)

	assert.Nil(t, health)
	assert.Equal(t, expectedErr, err)
}

func TestProviderAdapterWrapper_GetModels(t *testing.T) {
	expectedModels := []ModelInfo{
		{
			ID:           "gpt-4",
			Name:         "GPT-4",
			Provider:     ProviderOpenAI,
			Description:  "Most capable GPT-4 model",
			MaxTokens:    8192,
			Capabilities: []string{"completion", "chat", "code"},
		},
		{
			ID:           "gpt-3.5-turbo",
			Name:         "GPT-3.5 Turbo",
			Provider:     ProviderOpenAI,
			Description:  "Fast and efficient",
			MaxTokens:    4096,
			Capabilities: []string{"completion", "chat"},
		},
	}

	mock := &mockProviderAdapter{
		listModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return expectedModels, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	models, err := wrapper.GetModels(ctx)

	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, expectedModels[0].ID, models[0].ID)
	assert.Equal(t, expectedModels[0].Name, models[0].Name)
	assert.Equal(t, expectedModels[1].ID, models[1].ID)
}

func TestProviderAdapterWrapper_GetModels_Error(t *testing.T) {
	expectedErr := errors.New("failed to list models")

	mock := &mockProviderAdapter{
		listModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return nil, expectedErr
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	models, err := wrapper.GetModels(ctx)

	assert.Nil(t, models)
	assert.Equal(t, expectedErr, err)
}

func TestProviderAdapterWrapper_GetProviderType(t *testing.T) {
	tests := []struct {
		name             string
		expectedProvider Provider
	}{
		{
			name:             "OpenAI provider",
			expectedProvider: ProviderOpenAI,
		},
		{
			name:             "Anthropic provider",
			expectedProvider: ProviderAnthropic,
		},
		{
			name:             "Ollama provider",
			expectedProvider: ProviderOllama,
		},
		{
			name:             "ClaudeCode provider",
			expectedProvider: ProviderClaudeCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProviderAdapter{
				getProviderFunc: func() Provider {
					return tt.expectedProvider
				},
			}

			wrapper := &providerAdapterWrapper{
				adapter: mock,
				logger:  logrus.New(),
			}

			provider := wrapper.GetProviderType()
			assert.Equal(t, tt.expectedProvider, provider)
		})
	}
}

func TestProviderAdapterWrapper_IsAvailable_Healthy(t *testing.T) {
	mock := &mockProviderAdapter{
		getHealthFunc: func(ctx context.Context) (*HealthStatus, error) {
			return &HealthStatus{
				Provider: ProviderOpenAI,
				Status:   "healthy",
			}, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	available := wrapper.IsAvailable(ctx)

	assert.True(t, available, "provider should be available when healthy")
}

func TestProviderAdapterWrapper_IsAvailable_Unhealthy(t *testing.T) {
	mock := &mockProviderAdapter{
		getHealthFunc: func(ctx context.Context) (*HealthStatus, error) {
			return &HealthStatus{
				Provider: ProviderOpenAI,
				Status:   "unhealthy",
			}, nil
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	available := wrapper.IsAvailable(ctx)

	assert.False(t, available, "provider should not be available when unhealthy")
}

func TestProviderAdapterWrapper_IsAvailable_Error(t *testing.T) {
	mock := &mockProviderAdapter{
		getHealthFunc: func(ctx context.Context) (*HealthStatus, error) {
			return nil, errors.New("health check failed")
		},
	}

	wrapper := &providerAdapterWrapper{
		adapter: mock,
		logger:  logrus.New(),
	}

	ctx := context.Background()
	available := wrapper.IsAvailable(ctx)

	assert.False(t, available, "provider should not be available when health check fails")
}

func TestProviderAdapterWrapper_UpdateConfig(t *testing.T) {
	// UpdateConfig is not implemented yet, should return nil
	wrapper := &providerAdapterWrapper{
		adapter: &mockProviderAdapter{},
		logger:  logrus.New(),
	}

	err := wrapper.UpdateConfig(map[string]interface{}{"key": "value"})
	assert.NoError(t, err, "UpdateConfig should return nil (not implemented)")
}

func TestProviderAdapterWrapper_ValidateConfig(t *testing.T) {
	// ValidateConfig is not implemented yet, should return nil
	wrapper := &providerAdapterWrapper{
		adapter: &mockProviderAdapter{},
		logger:  logrus.New(),
	}

	err := wrapper.ValidateConfig(map[string]interface{}{"key": "value"})
	assert.NoError(t, err, "ValidateConfig should return nil (not implemented)")
}

func TestProviderAdapterWrapper_ContextPropagation(t *testing.T) {
	// Test that context is properly propagated to adapter methods
	type contextKey string
	const testKey contextKey = "test-key"
	const testValue = "test-value"

	tests := []struct {
		name string
		call func(ctx context.Context, wrapper *providerAdapterWrapper) error
	}{
		{
			name: "GenerateSQL context propagation",
			call: func(ctx context.Context, wrapper *providerAdapterWrapper) error {
				req := &SQLRequest{Prompt: "test", Schema: "schema"}
				_, err := wrapper.GenerateSQL(ctx, req)
				return err
			},
		},
		{
			name: "FixSQL context propagation",
			call: func(ctx context.Context, wrapper *providerAdapterWrapper) error {
				req := &SQLRequest{Query: "query", Error: "error"}
				_, err := wrapper.FixSQL(ctx, req)
				return err
			},
		},
		{
			name: "Chat context propagation",
			call: func(ctx context.Context, wrapper *providerAdapterWrapper) error {
				req := &ChatRequest{Prompt: "test"}
				_, err := wrapper.Chat(ctx, req)
				return err
			},
		},
		{
			name: "HealthCheck context propagation",
			call: func(ctx context.Context, wrapper *providerAdapterWrapper) error {
				_, err := wrapper.HealthCheck(ctx)
				return err
			},
		},
		{
			name: "GetModels context propagation",
			call: func(ctx context.Context, wrapper *providerAdapterWrapper) error {
				_, err := wrapper.GetModels(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextReceived := false

			mock := &mockProviderAdapter{
				generateSQLFunc: func(ctx context.Context, prompt, schema string, opts ...GenerateOption) (*SQLResponse, error) {
					if ctx.Value(testKey) == testValue {
						contextReceived = true
					}
					return &SQLResponse{}, nil
				},
				fixSQLFunc: func(ctx context.Context, query, errorMsg, schema string, opts ...GenerateOption) (*SQLResponse, error) {
					if ctx.Value(testKey) == testValue {
						contextReceived = true
					}
					return &SQLResponse{}, nil
				},
				chatFunc: func(ctx context.Context, prompt string, opts ...GenerateOption) (*ChatResponse, error) {
					if ctx.Value(testKey) == testValue {
						contextReceived = true
					}
					return &ChatResponse{}, nil
				},
				getHealthFunc: func(ctx context.Context) (*HealthStatus, error) {
					if ctx.Value(testKey) == testValue {
						contextReceived = true
					}
					return &HealthStatus{Status: "healthy"}, nil
				},
				listModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
					if ctx.Value(testKey) == testValue {
						contextReceived = true
					}
					return []ModelInfo{}, nil
				},
			}

			wrapper := &providerAdapterWrapper{
				adapter: mock,
				logger:  logrus.New(),
			}

			ctx := context.WithValue(context.Background(), testKey, testValue)
			err := tt.call(ctx, wrapper)

			require.NoError(t, err)
			assert.True(t, contextReceived, "context value should be propagated to adapter")
		})
	}
}
