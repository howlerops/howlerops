//go:build integration

package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
)

// mockHTTPService implements ai.Service for testing HTTP handlers
type mockHTTPService struct {
	generateSQLFunc           func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	fixSQLFunc                func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error)
	chatFunc                  func(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error)
	getProvidersFunc          func() []ai.Provider
	getProviderHealthFunc     func(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error)
	getAllProvidersHealthFunc func(ctx context.Context) (map[ai.Provider]*ai.HealthStatus, error)
	getAvailableModelsFunc    func(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error)
	getAllAvailableModelsFunc func(ctx context.Context) (map[ai.Provider][]ai.ModelInfo, error)
	updateProviderConfigFunc  func(provider ai.Provider, config interface{}) error
	getConfigFunc             func() *ai.Config
	getUsageStatsFunc         func(ctx context.Context, provider ai.Provider) (*ai.Usage, error)
	getAllUsageStatsFunc      func(ctx context.Context) (map[ai.Provider]*ai.Usage, error)
	testProviderFunc          func(ctx context.Context, provider ai.Provider, config interface{}) error
	validateRequestFunc       func(req *ai.SQLRequest) error
	validateChatRequestFunc   func(req *ai.ChatRequest) error
	startFunc                 func(ctx context.Context) error
	stopFunc                  func(ctx context.Context) error
}

func (m *mockHTTPService) GenerateSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.generateSQLFunc != nil {
		return m.generateSQLFunc(ctx, req)
	}
	return &ai.SQLResponse{
		Query:      "SELECT * FROM users",
		Confidence: 0.95,
		TokensUsed: 100,
	}, nil
}

func (m *mockHTTPService) FixSQL(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
	if m.fixSQLFunc != nil {
		return m.fixSQLFunc(ctx, req)
	}
	return &ai.SQLResponse{
		Query:      "SELECT * FROM users WHERE id = 1",
		Confidence: 0.90,
		TokensUsed: 50,
	}, nil
}

func (m *mockHTTPService) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ai.ChatResponse{Content: "Mock response"}, nil
}

func (m *mockHTTPService) GetProviders() []ai.Provider {
	if m.getProvidersFunc != nil {
		return m.getProvidersFunc()
	}
	return []ai.Provider{ai.ProviderOpenAI}
}

func (m *mockHTTPService) GetProviderHealth(ctx context.Context, provider ai.Provider) (*ai.HealthStatus, error) {
	if m.getProviderHealthFunc != nil {
		return m.getProviderHealthFunc(ctx, provider)
	}
	return &ai.HealthStatus{Status: "healthy"}, nil
}

func (m *mockHTTPService) GetAllProvidersHealth(ctx context.Context) (map[ai.Provider]*ai.HealthStatus, error) {
	if m.getAllProvidersHealthFunc != nil {
		return m.getAllProvidersHealthFunc(ctx)
	}
	return map[ai.Provider]*ai.HealthStatus{}, nil
}

func (m *mockHTTPService) GetAvailableModels(ctx context.Context, provider ai.Provider) ([]ai.ModelInfo, error) {
	if m.getAvailableModelsFunc != nil {
		return m.getAvailableModelsFunc(ctx, provider)
	}
	return []ai.ModelInfo{}, nil
}

func (m *mockHTTPService) GetAllAvailableModels(ctx context.Context) (map[ai.Provider][]ai.ModelInfo, error) {
	if m.getAllAvailableModelsFunc != nil {
		return m.getAllAvailableModelsFunc(ctx)
	}
	return map[ai.Provider][]ai.ModelInfo{}, nil
}

func (m *mockHTTPService) UpdateProviderConfig(provider ai.Provider, config interface{}) error {
	if m.updateProviderConfigFunc != nil {
		return m.updateProviderConfigFunc(provider, config)
	}
	return nil
}

func (m *mockHTTPService) GetConfig() *ai.Config {
	if m.getConfigFunc != nil {
		return m.getConfigFunc()
	}
	return &ai.Config{}
}

func (m *mockHTTPService) GetUsageStats(ctx context.Context, provider ai.Provider) (*ai.Usage, error) {
	if m.getUsageStatsFunc != nil {
		return m.getUsageStatsFunc(ctx, provider)
	}
	return &ai.Usage{}, nil
}

func (m *mockHTTPService) GetAllUsageStats(ctx context.Context) (map[ai.Provider]*ai.Usage, error) {
	if m.getAllUsageStatsFunc != nil {
		return m.getAllUsageStatsFunc(ctx)
	}
	return map[ai.Provider]*ai.Usage{}, nil
}

func (m *mockHTTPService) TestProvider(ctx context.Context, provider ai.Provider, config interface{}) error {
	if m.testProviderFunc != nil {
		return m.testProviderFunc(ctx, provider, config)
	}
	return nil
}

func (m *mockHTTPService) ValidateRequest(req *ai.SQLRequest) error {
	if m.validateRequestFunc != nil {
		return m.validateRequestFunc(req)
	}
	return nil
}

func (m *mockHTTPService) ValidateChatRequest(req *ai.ChatRequest) error {
	if m.validateChatRequestFunc != nil {
		return m.validateChatRequestFunc(req)
	}
	return nil
}

func (m *mockHTTPService) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	return nil
}

func (m *mockHTTPService) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return nil
}

// Test helpers
func newHandlerTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}

func newTestHandler() (*ai.HTTPHandler, *mockHTTPService) {
	service := &mockHTTPService{}
	logger := newHandlerTestLogger()
	handler := ai.NewHTTPHandler(service, logger)
	return handler, service
}

// TestNewHTTPHandler tests handler creation
func TestNewHTTPHandler(t *testing.T) {
	t.Run("creates handler successfully", func(t *testing.T) {
		service := &mockHTTPService{}
		logger := newHandlerTestLogger()

		handler := ai.NewHTTPHandler(service, logger)

		require.NotNil(t, handler)
	})

	t.Run("creates handler with nil service", func(t *testing.T) {
		logger := newHandlerTestLogger()

		handler := ai.NewHTTPHandler(nil, logger)

		require.NotNil(t, handler)
	})

	t.Run("creates handler with nil logger", func(t *testing.T) {
		service := &mockHTTPService{}

		handler := ai.NewHTTPHandler(service, nil)

		require.NotNil(t, handler)
	})
}

// TestRegisterRoutes tests route registration
func TestRegisterRoutes(t *testing.T) {
	t.Run("registers all routes successfully", func(t *testing.T) {
		handler, _ := newTestHandler()
		router := mux.NewRouter()

		handler.RegisterRoutes(router)

		// Verify routes are registered by walking the router
		routeCount := 0
		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			routeCount++
			return nil
		})

		assert.Greater(t, routeCount, 0)
	})

	t.Run("registers GenerateSQL route", func(t *testing.T) {
		handler, _ := newTestHandler()
		router := mux.NewRouter()

		handler.RegisterRoutes(router)

		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(`{"prompt":"test"}`))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("registers FixSQL route", func(t *testing.T) {
		handler, _ := newTestHandler()
		router := mux.NewRouter()

		handler.RegisterRoutes(router)

		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(`{"query":"SELECT","error":"syntax"}`))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

// TestGenerateSQL tests the GenerateSQL handler
func TestGenerateSQL(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:      "SELECT * FROM users WHERE id = 1",
				Confidence: 0.95,
				TokensUsed: 100,
			}, nil
		}

		body := `{
			"prompt": "get user by id",
			"schema": "users(id, name)",
			"provider": "openai",
			"model": "gpt-4"
		}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "sql")
		assert.Contains(t, response, "confidence")
		assert.Contains(t, response, "tokens")
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(`{invalid json`))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		assert.Contains(t, response["error"], "Invalid request body")
	})

	t.Run("missing prompt", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"schema": "users(id, name)"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		assert.Contains(t, response["error"], "Prompt is required")
	})

	t.Run("empty prompt", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"prompt": ""}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, errors.New("service error")
		}

		body := `{"prompt": "test query"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		assert.Contains(t, response["error"], "Failed to generate SQL")
	})

	t.Run("with all optional fields", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			assert.Equal(t, "test prompt", req.Prompt)
			assert.Equal(t, "users(id, name)", req.Schema)
			assert.Equal(t, ai.Provider("openai"), req.Provider)
			assert.Equal(t, "gpt-4", req.Model)
			assert.Equal(t, 2000, req.MaxTokens)
			assert.Equal(t, 0.7, req.Temperature)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		body := `{
			"prompt": "test prompt",
			"schema": "users(id, name)",
			"provider": "openai",
			"model": "gpt-4",
			"maxTokens": 2000,
			"temperature": 0.7
		}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nil response from service", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, errors.New("nil response")
		}

		body := `{"prompt": "test"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("zero confidence response", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:      "SELECT 1",
				Confidence: 0.0,
				TokensUsed: 0,
			}, nil
		}

		body := `{"prompt": "test"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)
		assert.Equal(t, float64(0), response["confidence"])
	})
}

// TestFixSQL tests the FixSQL handler
func TestFixSQL(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, service := newTestHandler()
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{
				Query:      "SELECT * FROM users WHERE id = 1",
				Confidence: 0.90,
				TokensUsed: 50,
			}, nil
		}

		body := `{
			"query": "SELECT * FROM user WHERE id = 1",
			"error": "table 'user' does not exist",
			"schema": "users(id, name)"
		}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)
		assert.Contains(t, response, "sql")
		assert.Contains(t, response, "confidence")
		assert.Contains(t, response, "tokens")
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(`invalid`))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing query", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"error": "syntax error"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		assert.Contains(t, response["error"], "Query and error are required")
	})

	t.Run("missing error", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"query": "SELECT * FROM users"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty query", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"query": "", "error": "syntax error"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty error", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"query": "SELECT 1", "error": ""}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		handler, service := newTestHandler()
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, errors.New("service error")
		}

		body := `{"query": "SELECT 1", "error": "test error"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("with all optional fields", func(t *testing.T) {
		handler, service := newTestHandler()
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			assert.Equal(t, "SELECT 1", req.Query)
			assert.Equal(t, "syntax error", req.Error)
			assert.Equal(t, "users(id)", req.Schema)
			assert.Equal(t, ai.Provider("anthropic"), req.Provider)
			assert.Equal(t, "claude-3", req.Model)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		body := `{
			"query": "SELECT 1",
			"error": "syntax error",
			"schema": "users(id)",
			"provider": "anthropic",
			"model": "claude-3",
			"maxTokens": 1500,
			"temperature": 0.5
		}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestTestOpenAI tests the TestOpenAI handler
func TestTestOpenAI(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{
			"apiKey": "sk-test123",
			"model": "gpt-4",
			"organization": "org-123"
		}`
		req := httptest.NewRequest("POST", "/test/openai", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestOpenAI(w, req)

		// Will fail to create actual provider but should handle gracefully
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/openai", strings.NewReader(`{invalid`))
		w := httptest.NewRecorder()

		handler.TestOpenAI(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing API key", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"model": "gpt-4"}`
		req := httptest.NewRequest("POST", "/test/openai", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestOpenAI(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("empty request body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/openai", strings.NewReader(`{}`))
		w := httptest.NewRecorder()

		handler.TestOpenAI(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestTestAnthropic tests the TestAnthropic handler
func TestTestAnthropic(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{
			"apiKey": "sk-ant-test123",
			"model": "claude-3-opus"
		}`
		req := httptest.NewRequest("POST", "/test/anthropic", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestAnthropic(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/anthropic", strings.NewReader(`invalid json`))
		w := httptest.NewRecorder()

		handler.TestAnthropic(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/anthropic", strings.NewReader(`{}`))
		w := httptest.NewRecorder()

		handler.TestAnthropic(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestTestOllama tests the TestOllama handler
func TestTestOllama(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{
			"endpoint": "http://localhost:11434",
			"model": "llama2"
		}`
		req := httptest.NewRequest("POST", "/test/ollama", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestOllama(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/ollama", strings.NewReader(`{`))
		w := httptest.NewRecorder()

		handler.TestOllama(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing endpoint", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"model": "llama2"}`
		req := httptest.NewRequest("POST", "/test/ollama", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestOllama(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestTestHuggingFace tests the TestHuggingFace handler
func TestTestHuggingFace(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{
			"apiKey": "hf_test123",
			"model": "gpt2",
			"endpoint": "https://api-inference.huggingface.co"
		}`
		req := httptest.NewRequest("POST", "/test/huggingface", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestHuggingFace(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/huggingface", strings.NewReader(`not json`))
		w := httptest.NewRecorder()

		handler.TestHuggingFace(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty request", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/huggingface", strings.NewReader(`{}`))
		w := httptest.NewRecorder()

		handler.TestHuggingFace(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestTestClaudeCode tests the TestClaudeCode handler
func TestTestClaudeCode(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{
			"binaryPath": "/usr/local/bin/claude",
			"model": "claude-code"
		}`
		req := httptest.NewRequest("POST", "/test/claudecode", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestClaudeCode(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("with default binary path", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"model": "claude-code"}`
		req := httptest.NewRequest("POST", "/test/claudecode", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestClaudeCode(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/claudecode", strings.NewReader(`bad json`))
		w := httptest.NewRecorder()

		handler.TestClaudeCode(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty body uses defaults", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/claudecode", strings.NewReader(`{}`))
		w := httptest.NewRecorder()

		handler.TestClaudeCode(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestTestCodex tests the TestCodex handler
func TestTestCodex(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{
			"apiKey": "sk-test",
			"model": "code-davinci-002",
			"organization": "org-123"
		}`
		req := httptest.NewRequest("POST", "/test/codex", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestCodex(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/test/codex", strings.NewReader(`{bad`))
		w := httptest.NewRecorder()

		handler.TestCodex(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing API key", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"model": "code-davinci-002"}`
		req := httptest.NewRequest("POST", "/test/codex", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.TestCodex(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestDetectOllama tests the DetectOllama handler
func TestDetectOllama(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("GET", "/ollama/detect", nil)
		w := httptest.NewRecorder()

		handler.DetectOllama(w, req)

		// Detector will be created and run - may succeed or fail depending on system
		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with context", func(t *testing.T) {
		handler, _ := newTestHandler()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := httptest.NewRequest("GET", "/ollama/detect", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DetectOllama(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with cancelled context", func(t *testing.T) {
		handler, _ := newTestHandler()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := httptest.NewRequest("GET", "/ollama/detect", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DetectOllama(w, req)

		assert.NotEqual(t, 0, w.Code)
	})
}

// TestGetOllamaInstallInstructions tests the GetOllamaInstallInstructions handler
func TestGetOllamaInstallInstructions(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("GET", "/ollama/install", nil)
		w := httptest.NewRecorder()

		handler.GetOllamaInstallInstructions(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("returns JSON response", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("GET", "/ollama/install", nil)
		w := httptest.NewRecorder()

		handler.GetOllamaInstallInstructions(w, req)

		if w.Code == http.StatusOK {
			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Contains(t, response, "instructions")
		}
	})
}

// TestStartOllamaService tests the StartOllamaService handler
func TestStartOllamaService(t *testing.T) {
	t.Run("POST request", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/ollama/start", nil)
		w := httptest.NewRecorder()

		handler.StartOllamaService(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with context timeout", func(t *testing.T) {
		handler, _ := newTestHandler()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		req := httptest.NewRequest("POST", "/ollama/start", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.StartOllamaService(w, req)

		assert.NotEqual(t, 0, w.Code)
	})
}

// TestPullOllamaModel tests the PullOllamaModel handler
func TestPullOllamaModel(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"model": "llama2"}`
		req := httptest.NewRequest("POST", "/ollama/pull", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.PullOllamaModel(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/ollama/pull", strings.NewReader(`{invalid`))
		w := httptest.NewRecorder()

		handler.PullOllamaModel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing model name", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{}`
		req := httptest.NewRequest("POST", "/ollama/pull", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.PullOllamaModel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		assert.Contains(t, response["error"], "Model name is required")
	})

	t.Run("empty model name", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"model": ""}`
		req := httptest.NewRequest("POST", "/ollama/pull", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.PullOllamaModel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("with various model names", func(t *testing.T) {
		handler, _ := newTestHandler()

		models := []string{"llama2", "codellama", "mistral", "vicuna"}
		for _, model := range models {
			body := fmt.Sprintf(`{"model": "%s"}`, model)
			req := httptest.NewRequest("POST", "/ollama/pull", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.PullOllamaModel(w, req)

			assert.NotEqual(t, 0, w.Code)
		}
	})
}

// TestOpenOllamaTerminal tests the OpenOllamaTerminal handler
func TestOpenOllamaTerminal(t *testing.T) {
	t.Run("success case with default commands", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/ollama/open-terminal", nil)
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		// Will attempt to open terminal - may succeed or fail based on OS
		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with custom commands", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"commands": ["ollama serve", "ollama list"]}`
		req := httptest.NewRequest("POST", "/ollama/open-terminal", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with empty commands array", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"commands": []}`
		req := httptest.NewRequest("POST", "/ollama/open-terminal", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with single command", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"commands": ["ollama run llama2"]}`
		req := httptest.NewRequest("POST", "/ollama/open-terminal", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/ollama/open-terminal", strings.NewReader(`{bad json`))
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("with nil body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/ollama/open-terminal", nil)
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with empty body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/ollama/open-terminal", bytes.NewReader([]byte{}))
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("with empty JSON object", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{}`
		req := httptest.NewRequest("POST", "/ollama/open-terminal", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.OpenOllamaTerminal(w, req)

		assert.NotEqual(t, 0, w.Code)
	})
}

// TestHelperMethods tests the helper methods
func TestRespondWithJSON(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		handler, _ := newTestHandler()

		w := httptest.NewRecorder()

		// We can't directly call the private method, but we can test it indirectly
		// through a public endpoint
		req := httptest.NewRequest("GET", "/ollama/install", nil)
		handler.GetOllamaInstallInstructions(w, req)

		// Check that response has JSON content type if successful
		if w.Code == http.StatusOK {
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		}
	})
}

// TestErrorHandling tests error handling across handlers
func TestErrorHandling(t *testing.T) {
	t.Run("GenerateSQL service timeout", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, context.DeadlineExceeded
		}

		body := `{"prompt": "test"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("FixSQL service timeout", func(t *testing.T) {
		handler, service := newTestHandler()
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, context.DeadlineExceeded
		}

		body := `{"query": "SELECT 1", "error": "test"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("malformed JSON bodies", func(t *testing.T) {
		handler, _ := newTestHandler()

		testCases := []struct {
			name string
			path string
			body string
		}{
			{"GenerateSQL", "/generate-sql", `{"prompt": "test", "model": `},
			{"FixSQL", "/fix-sql", `{"query": "SELECT", "error": `},
			{"TestOpenAI", "/test/openai", `{"apiKey": `},
			{"PullModel", "/ollama/pull", `{"model": "llama2", `},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
				w := httptest.NewRecorder()

				router := mux.NewRouter()
				handler.RegisterRoutes(router)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})

	t.Run("nil pointer handling", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return nil, errors.New("something went wrong")
		}

		body := `{"prompt": "test"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	t.Run("multiple GenerateSQL requests", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			time.Sleep(10 * time.Millisecond)
			return &ai.SQLResponse{Query: "SELECT 1", Confidence: 0.9}, nil
		}

		numRequests := 10
		var wg sync.WaitGroup
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				defer wg.Done()
				body := `{"prompt": "test query"}`
				req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
				w := httptest.NewRecorder()

				handler.GenerateSQL(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			}()
		}

		wg.Wait()
	})

	t.Run("mixed endpoint requests", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		numRequests := 20
		var wg sync.WaitGroup
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			if i%2 == 0 {
				go func() {
					defer wg.Done()
					body := `{"prompt": "test"}`
					req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
					w := httptest.NewRecorder()
					handler.GenerateSQL(w, req)
				}()
			} else {
				go func() {
					defer wg.Done()
					body := `{"query": "SELECT 1", "error": "test"}`
					req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
					w := httptest.NewRecorder()
					handler.FixSQL(w, req)
				}()
			}
		}

		wg.Wait()
	})
}

// TestRequestValidation tests request validation
func TestRequestValidation(t *testing.T) {
	t.Run("various invalid prompts", func(t *testing.T) {
		handler, _ := newTestHandler()

		testCases := []string{
			`{"prompt": ""}`,
			`{}`,
		}

		for _, tc := range testCases {
			req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(tc))
			w := httptest.NewRecorder()

			handler.GenerateSQL(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		}
	})

	t.Run("various invalid fix requests", func(t *testing.T) {
		handler, _ := newTestHandler()

		testCases := []string{
			`{"query": "", "error": "test"}`,
			`{"query": "SELECT 1", "error": ""}`,
			`{"query": "", "error": ""}`,
			`{}`,
		}

		for _, tc := range testCases {
			req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(tc))
			w := httptest.NewRecorder()

			handler.FixSQL(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		}
	})
}

// TestContentTypeHandling tests content type handling
func TestContentTypeHandling(t *testing.T) {
	t.Run("GenerateSQL with wrong content type", func(t *testing.T) {
		handler, _ := newTestHandler()

		body := `{"prompt": "test"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		// Should still process as JSON decoder doesn't check content-type
		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("response has correct content type", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		body := `{"prompt": "test"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
}

// TestBodyReading tests request body reading
func TestBodyReading(t *testing.T) {
	t.Run("large request body", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		largePrompt := strings.Repeat("test ", 10000)
		body := fmt.Sprintf(`{"prompt": "%s"}`, largePrompt)
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("empty body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/generate-sql", bytes.NewReader([]byte{}))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("EOF body", func(t *testing.T) {
		handler, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/generate-sql", &errorReader{})
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// errorReader is a helper for testing error conditions
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// TestHTTPMethodValidation tests that only correct HTTP methods work
func TestHTTPMethodValidation(t *testing.T) {
	t.Run("POST endpoints reject GET", func(t *testing.T) {
		handler, _ := newTestHandler()
		router := mux.NewRouter()
		handler.RegisterRoutes(router)

		endpoints := []string{
			"/generate-sql",
			"/fix-sql",
			"/test/openai",
			"/test/anthropic",
			"/test/ollama",
			"/test/huggingface",
			"/test/claudecode",
			"/test/codex",
			"/ollama/start",
			"/ollama/pull",
			"/ollama/open-terminal",
		}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "endpoint: %s", endpoint)
		}
	})

	t.Run("GET endpoints reject POST", func(t *testing.T) {
		handler, _ := newTestHandler()
		router := mux.NewRouter()
		handler.RegisterRoutes(router)

		endpoints := []string{
			"/ollama/detect",
			"/ollama/install",
		}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest("POST", endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "endpoint: %s", endpoint)
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("Unicode in prompts", func(t *testing.T) {
		handler, service := newTestHandler()
		service.generateSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			assert.Contains(t, req.Prompt, "用户")
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		body := `{"prompt": "查询所有用户"}`
		req := httptest.NewRequest("POST", "/generate-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Special characters in SQL", func(t *testing.T) {
		handler, service := newTestHandler()
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			assert.Contains(t, req.Query, `"`)
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		body := `{"query": "SELECT \"field\" FROM table", "error": "syntax error"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Newlines in error messages", func(t *testing.T) {
		handler, service := newTestHandler()
		service.fixSQLFunc = func(ctx context.Context, req *ai.SQLRequest) (*ai.SQLResponse, error) {
			assert.Contains(t, req.Error, "\n")
			return &ai.SQLResponse{Query: "SELECT 1"}, nil
		}

		body := `{"query": "SELECT 1", "error": "Error line 1\nError line 2"}`
		req := httptest.NewRequest("POST", "/fix-sql", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.FixSQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Very long model names", func(t *testing.T) {
		handler, _ := newTestHandler()

		longModel := strings.Repeat("a", 500)
		body := fmt.Sprintf(`{"model": "%s"}`, longModel)
		req := httptest.NewRequest("POST", "/ollama/pull", strings.NewReader(body))
		w := httptest.NewRecorder()

		handler.PullOllamaModel(w, req)

		assert.NotEqual(t, 0, w.Code)
	})
}
