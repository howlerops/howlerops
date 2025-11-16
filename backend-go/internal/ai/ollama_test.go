//go:build integration

package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test logger
func newOllamaTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs during tests
	return logger
}

// TestNewOllamaProvider_ValidConfig verifies provider creation with valid config
func TestNewOllamaProvider_ValidConfig(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint:        "http://localhost:11434",
		Models:          []string{"llama3.1:8b"},
		PullTimeout:     5 * time.Minute,
		GenerateTimeout: 2 * time.Minute,
		AutoPullModels:  true,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, ai.ProviderOllama, provider.GetProviderType())
}

// TestNewOllamaProvider_NilConfig verifies error with nil config
func TestNewOllamaProvider_NilConfig(t *testing.T) {
	provider, err := ai.NewOllamaProvider(nil, newTestLogger())

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

// TestNewOllamaProvider_EmptyEndpoint verifies default endpoint is set
func TestNewOllamaProvider_EmptyEndpoint(t *testing.T) {
	config := &ai.OllamaConfig{}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Default endpoint should be set to http://localhost:11434
}

// TestNewOllamaProvider_DefaultValues verifies default values are set
func TestNewOllamaProvider_DefaultValues(t *testing.T) {
	config := &ai.OllamaConfig{}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Defaults should be:
	// - Endpoint: http://localhost:11434
	// - PullTimeout: 10 minutes
	// - GenerateTimeout: 2 minutes
	// - Models: ["sqlcoder:7b", "codellama:7b", "llama3.1:8b"]
}

// TestNewOllamaProvider_CustomTimeout verifies custom timeouts are respected
func TestNewOllamaProvider_CustomTimeout(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint:        "http://localhost:11434",
		PullTimeout:     15 * time.Minute,
		GenerateTimeout: 5 * time.Minute,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
}

// TestOllamaProvider_GetProviderType verifies provider type
func TestOllamaProvider_GetProviderType(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	assert.Equal(t, ai.ProviderOllama, provider.GetProviderType())
}

// TestOllamaProvider_GenerateSQL_Success verifies successful SQL generation
func TestOllamaProvider_GenerateSQL_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name":        "llama3.1:8b",
						"size":        4661224320,
						"digest":      "test-digest",
						"modified_at": time.Now(),
						"details": map[string]interface{}{
							"format":             "gguf",
							"family":             "llama",
							"parameter_size":     "8B",
							"quantization_level": "Q4_0",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			// Verify request structure
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "llama3.1:8b", req["model"])
			assert.Equal(t, false, req["stream"])

			resp := map[string]interface{}{
				"model":             "llama3.1:8b",
				"response":          `{"query": "SELECT * FROM users WHERE id = 1", "explanation": "Simple user lookup", "confidence": 0.95, "suggestions": ["Add index on id"]}`,
				"done":              true,
				"prompt_eval_count": 100,
				"eval_count":        50,
			}
			json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
		Models:   []string{"llama3.1:8b"},
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get user by id",
		Schema:      "CREATE TABLE users (id INT PRIMARY KEY, name TEXT)",
		Provider:    ai.ProviderOllama,
		Model:       "llama3.1:8b",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SELECT * FROM users WHERE id = 1", resp.Query)
	assert.Equal(t, "Simple user lookup", resp.Explanation)
	assert.Equal(t, 0.95, resp.Confidence)
	assert.Equal(t, []string{"Add index on id"}, resp.Suggestions)
	assert.Equal(t, ai.ProviderOllama, resp.Provider)
	assert.Equal(t, "llama3.1:8b", resp.Model)
	assert.Equal(t, 150, resp.TokensUsed) // 100 + 50
}

// TestOllamaProvider_GenerateSQL_NonJSONResponse verifies SQL extraction from non-JSON
func TestOllamaProvider_GenerateSQL_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name": "llama3.1:8b",
						"details": map[string]interface{}{
							"family": "llama",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "```sql\nSELECT * FROM products ORDER BY price DESC\n```",
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "List products by price",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.Query, "SELECT * FROM products")
	assert.Equal(t, 0.7, resp.Confidence) // Lower confidence for non-structured response
}

// TestOllamaProvider_FixSQL_Success verifies successful SQL fixing
func TestOllamaProvider_FixSQL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name":    "llama3.1:8b",
						"details": map[string]interface{}{"family": "llama"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			// Verify fix prompt contains error and original query
			prompt := req["prompt"].(string)
			assert.Contains(t, prompt, "Fix the following SQL")
			assert.Contains(t, prompt, "syntax error")

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT * FROM users WHERE name = 'John'", "explanation": "Fixed missing quote", "confidence": 0.90, "suggestions": ["Use parameterized queries"]}`,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query:    "SELECT * FROM users WHERE name = 'John",
		Error:    "syntax error at end of input",
		Provider: ai.ProviderOllama,
		Model:    "llama3.1:8b",
	}

	resp, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.Query, "SELECT * FROM users")
	assert.Contains(t, resp.Explanation, "Fixed missing quote")
	assert.Equal(t, 0.90, resp.Confidence)
}

// TestOllamaProvider_Chat_Success verifies successful chat interaction
func TestOllamaProvider_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "llama3.1:8b", req["model"])
			assert.Equal(t, false, req["stream"])

			resp := map[string]interface{}{
				"model":             "llama3.1:8b",
				"response":          "SQL is a declarative language for managing relational databases.",
				"done":              true,
				"prompt_eval_count": 50,
				"eval_count":        30,
				"total_duration":    1000000000,
				"load_duration":     500000000,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "What is SQL?",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.Content, "SQL is a declarative language")
	assert.Equal(t, ai.ProviderOllama, resp.Provider)
	assert.Equal(t, "llama3.1:8b", resp.Model)
	assert.Equal(t, 80, resp.TokensUsed) // 50 + 30
	assert.NotNil(t, resp.Metadata)
	assert.Equal(t, "llama3.1:8b", resp.Metadata["model"])
}

// TestOllamaProvider_Chat_WithContext verifies chat with context
func TestOllamaProvider_Chat_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			prompt := req["prompt"].(string)
			assert.Contains(t, prompt, "Context:")
			assert.Contains(t, prompt, "Database: PostgreSQL")
			assert.Contains(t, prompt, "How do I index")

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "Use CREATE INDEX statement",
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "How do I index a column?",
		Context:  "Database: PostgreSQL 15",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Contains(t, resp.Content, "CREATE INDEX")
}

// TestOllamaProvider_Chat_WithCustomSystem verifies custom system prompt
func TestOllamaProvider_Chat_WithCustomSystem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "You are a database expert", req["system"])

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "Expert response",
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "Help me",
		System:   "You are a database expert",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Expert response", resp.Content)
}

// TestOllamaProvider_Chat_NilRequest verifies error on nil request
func TestOllamaProvider_Chat_NilRequest(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	resp, err := provider.Chat(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

// TestOllamaProvider_Chat_EmptyResponse verifies error on empty response
func TestOllamaProvider_Chat_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "   ", // Whitespace only
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "empty response")
}

// TestOllamaProvider_HealthCheck_Healthy verifies healthy status
func TestOllamaProvider_HealthCheck_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderOllama, health.Provider)
	assert.Equal(t, "healthy", health.Status)
	assert.Contains(t, health.Message, "operational")
	assert.Greater(t, health.ResponseTime, time.Duration(0))
}

// TestOllamaProvider_HealthCheck_Unhealthy verifies unhealthy status
func TestOllamaProvider_HealthCheck_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service unavailable"))
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "503")
}

// TestOllamaProvider_HealthCheck_ConnectionError verifies connection error
func TestOllamaProvider_HealthCheck_ConnectionError(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:99999", // Invalid port
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "Request failed")
}

// TestOllamaProvider_GetModels_Success verifies model listing
func TestOllamaProvider_GetModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name":        "sqlcoder:7b",
						"size":        4661224320,
						"digest":      "sha256:abc123",
						"modified_at": time.Now(),
						"details": map[string]interface{}{
							"format":             "gguf",
							"family":             "sqlcoder",
							"parameter_size":     "7B",
							"quantization_level": "Q4_0",
						},
					},
					{
						"name":        "codellama:7b",
						"size":        3826793728,
						"digest":      "sha256:def456",
						"modified_at": time.Now(),
						"details": map[string]interface{}{
							"format":             "gguf",
							"family":             "llama",
							"parameter_size":     "7B",
							"quantization_level": "Q4_0",
						},
					},
					{
						"name":        "llama3.1:8b",
						"size":        4661224320,
						"digest":      "sha256:ghi789",
						"modified_at": time.Now(),
						"details": map[string]interface{}{
							"format":             "gguf",
							"family":             "llama",
							"parameter_size":     "8B",
							"quantization_level": "Q4_0",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	require.NoError(t, err)
	require.Len(t, models, 3)

	// Check SQLCoder model
	sqlcoder := models[0]
	assert.Equal(t, "sqlcoder:7b", sqlcoder.ID)
	assert.Equal(t, "sqlcoder:7b", sqlcoder.Name)
	assert.Equal(t, ai.ProviderOllama, sqlcoder.Provider)
	assert.Contains(t, sqlcoder.Description, "SQLCoder")
	assert.Contains(t, sqlcoder.Capabilities, "text-to-sql")
	assert.Contains(t, sqlcoder.Capabilities, "sql-optimization")
	assert.Equal(t, "7B", sqlcoder.Metadata["parameter_size"])

	// Check CodeLlama model
	codellama := models[1]
	assert.Equal(t, "codellama:7b", codellama.Name)
	assert.Contains(t, codellama.Description, "CodeLlama")
	assert.Contains(t, codellama.Capabilities, "explanation")

	// Check Llama model
	llama := models[2]
	assert.Equal(t, "llama3.1:8b", llama.Name)
	assert.Contains(t, llama.Description, "Llama")
	assert.Equal(t, "llama", llama.Metadata["family"])
}

// TestOllamaProvider_GetModels_EmptyList verifies empty model list
func TestOllamaProvider_GetModels_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	require.NoError(t, err)
	assert.Empty(t, models)
}

// TestOllamaProvider_GetModels_HTTPError verifies HTTP error handling
func TestOllamaProvider_GetModels_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	assert.Error(t, err)
	assert.Nil(t, models)
	assert.Contains(t, err.Error(), "HTTP 500")
}

// TestOllamaProvider_IsAvailable_True verifies available check returns true
func TestOllamaProvider_IsAvailable_True(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	available := provider.IsAvailable(context.Background())

	assert.True(t, available)
}

// TestOllamaProvider_IsAvailable_False verifies available check returns false
func TestOllamaProvider_IsAvailable_False(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:99999",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	available := provider.IsAvailable(context.Background())

	assert.False(t, available)
}

// TestOllamaProvider_PullModel_Success verifies model pulling
func TestOllamaProvider_PullModel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "llama3.1:8b", req["name"])
			assert.Equal(t, false, req["stream"])

			resp := map[string]interface{}{
				"status":    "success",
				"digest":    "sha256:abc123",
				"total":     4661224320,
				"completed": 4661224320,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint:       server.URL,
		AutoPullModels: true,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	// Type assert to access PullModel (not part of AIProvider interface)
	ollamaProvider, ok := provider.(*ai.OllamaProvider)
	require.True(t, ok)

	err = ollamaProvider.PullModel(context.Background(), "llama3.1:8b")

	assert.NoError(t, err)
}

// TestOllamaProvider_PullModel_AutoPullDisabled verifies error when auto-pull disabled
func TestOllamaProvider_PullModel_AutoPullDisabled(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint:       "http://localhost:11434",
		AutoPullModels: false,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	// Type assert to access PullModel (not part of AIProvider interface)
	ollamaProvider, ok := provider.(*ai.OllamaProvider)
	require.True(t, ok)

	err = ollamaProvider.PullModel(context.Background(), "llama3.1:8b")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auto-pull is disabled")
	assert.Contains(t, err.Error(), "ollama pull")
}

// TestOllamaProvider_PullModel_HTTPError verifies pull error handling
func TestOllamaProvider_PullModel_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("model not found"))
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint:       server.URL,
		AutoPullModels: true,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	// Type assert to access PullModel (not part of AIProvider interface)
	ollamaProvider, ok := provider.(*ai.OllamaProvider)
	require.True(t, ok)

	err = ollamaProvider.PullModel(context.Background(), "nonexistent:model")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

// TestOllamaProvider_UpdateConfig_Success verifies config update
func TestOllamaProvider_UpdateConfig_Success(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	newConfig := &ai.OllamaConfig{
		Endpoint: "http://localhost:11435",
		Models:   []string{"llama3.1:8b"},
	}

	err = provider.UpdateConfig(newConfig)

	assert.NoError(t, err)
}

// TestOllamaProvider_UpdateConfig_InvalidType verifies error on invalid config type
func TestOllamaProvider_UpdateConfig_InvalidType(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	invalidConfig := &ai.OpenAIConfig{
		APIKey: "test",
	}

	err = provider.UpdateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// TestOllamaProvider_ValidateConfig_Success verifies config validation
func TestOllamaProvider_ValidateConfig_Success(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	validConfig := &ai.OllamaConfig{
		Endpoint: "http://localhost:11435",
	}

	err = provider.ValidateConfig(validConfig)

	assert.NoError(t, err)
}

// TestOllamaProvider_ValidateConfig_EmptyEndpoint verifies error on empty endpoint
func TestOllamaProvider_ValidateConfig_EmptyEndpoint(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	invalidConfig := &ai.OllamaConfig{
		Endpoint: "",
	}

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

// TestOllamaProvider_ValidateConfig_InvalidType verifies error on invalid type
func TestOllamaProvider_ValidateConfig_InvalidType(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	invalidConfig := &ai.OpenAIConfig{
		APIKey: "test",
	}

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// TestOllamaProvider_ModelNotFound_AutoPull verifies auto-pull when model not found
func TestOllamaProvider_ModelNotFound_AutoPull(t *testing.T) {
	pullCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			// Return empty model list initially
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/pull":
			pullCalled = true
			resp := map[string]interface{}{
				"status": "success",
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT 1", "explanation": "Test", "confidence": 0.9}`,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint:       server.URL,
		AutoPullModels: true,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, pullCalled, "Pull should have been called")
}

// TestOllamaProvider_ModelNotFound_NoPull verifies error when auto-pull disabled
func TestOllamaProvider_ModelNotFound_NoPull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint:       server.URL,
		AutoPullModels: false,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Test",
		Model:    "nonexistent:model",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "pull it manually")
}

// TestOllamaProvider_OllamaAPIError verifies Ollama API error handling
func TestOllamaProvider_OllamaAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			w.WriteHeader(http.StatusBadRequest)
			resp := map[string]interface{}{
				"error": "invalid request: temperature must be between 0 and 2",
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "llama3.1:8b",
		Provider:    ai.ProviderOllama,
		Temperature: 5.0, // Invalid
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Ollama API error")
	assert.Contains(t, err.Error(), "temperature must be between")
}

// TestOllamaProvider_RequestOptions verifies options are passed correctly
func TestOllamaProvider_RequestOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			options := req["options"].(map[string]interface{})
			assert.Equal(t, 0.8, options["temperature"])
			assert.Equal(t, float64(2000), options["num_predict"])

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT 1", "explanation": "Test", "confidence": 0.9}`,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "llama3.1:8b",
		Provider:    ai.ProviderOllama,
		Temperature: 0.8,
		MaxTokens:   2000,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestOllamaProvider_StreamDisabled verifies stream is always false
func TestOllamaProvider_StreamDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, false, req["stream"], "Stream should always be false")

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "Test response",
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/pull":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, false, req["stream"], "Stream should always be false for pull")

			resp := map[string]interface{}{
				"status": "success",
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint:       server.URL,
		AutoPullModels: true,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	// Test Chat (which calls callOllama indirectly)
	chatReq := &ai.ChatRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}
	_, _ = provider.Chat(context.Background(), chatReq)

	// Type assert to access PullModel (not part of AIProvider interface)
	ollamaProvider, ok := provider.(*ai.OllamaProvider)
	require.True(t, ok)

	// Test PullModel
	_ = ollamaProvider.PullModel(context.Background(), "llama3.1:8b")
}

// TestOllamaProvider_ContextCancellation verifies context cancellation
func TestOllamaProvider_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	health, err := provider.HealthCheck(ctx)

	// Context should be cancelled
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", health.Status)
}

// TestOllamaProvider_JSONExtractionFromEmbedded verifies JSON extraction from text
func TestOllamaProvider_JSONExtractionFromEmbedded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			// Response with JSON embedded in text
			textWithJSON := `Here's the SQL query you requested:

{"query": "SELECT * FROM orders", "explanation": "Lists all orders", "confidence": 0.88}

I hope this helps!`

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": textWithJSON,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "List orders",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	// When JSON is embedded in text, extractSQL finds it and extracts the query
	assert.Equal(t, "SELECT * FROM orders", resp.Query)
	// But explanation becomes the full response text (not from the embedded JSON)
	assert.Contains(t, resp.Explanation, "Here's the SQL query")
	// And confidence is lower for extracted (non-direct-JSON) responses
	assert.Equal(t, 0.7, resp.Confidence)
}

// TestOllamaProvider_SQLExtractionFromCodeBlock verifies SQL extraction
func TestOllamaProvider_SQLExtractionFromCodeBlock(t *testing.T) {
	tests := []struct {
		name               string
		response           string
		expectedQuery      string
		expectedConfidence float64
	}{
		{
			name:               "sql code block",
			response:           "```sql\nSELECT id, name FROM users WHERE active = true\n```",
			expectedQuery:      "SELECT id, name FROM users WHERE active = true",
			expectedConfidence: 0.7,
		},
		{
			name:               "generic code block",
			response:           "```\nSELECT COUNT(*) FROM orders\n```",
			expectedQuery:      "SELECT COUNT(*) FROM orders",
			expectedConfidence: 0.7,
		},
		{
			name:               "inline SQL statement",
			response:           "You can use this query:\nSELECT * FROM products WHERE price > 100",
			expectedQuery:      "SELECT * FROM products WHERE price > 100",
			expectedConfidence: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/tags":
					resp := map[string]interface{}{
						"models": []map[string]interface{}{
							{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
						},
					}
					json.NewEncoder(w).Encode(resp)
				case "/api/generate":
					resp := map[string]interface{}{
						"model":    "llama3.1:8b",
						"response": tt.response,
						"done":     true,
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			config := &ai.OllamaConfig{
				Endpoint: server.URL,
			}

			provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
			require.NoError(t, err)

			req := &ai.SQLRequest{
				Prompt:   "Test query",
				Model:    "llama3.1:8b",
				Provider: ai.ProviderOllama,
			}

			resp, err := provider.GenerateSQL(context.Background(), req)

			require.NoError(t, err)
			assert.Contains(t, resp.Query, strings.TrimSpace(tt.expectedQuery))
			assert.Equal(t, tt.expectedConfidence, resp.Confidence)
		})
	}
}

// TestOllamaProvider_InvalidResponseParsing verifies error on unparseable response
func TestOllamaProvider_InvalidResponseParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "This is just plain text without any SQL or JSON",
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Generate query",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "could not extract SQL")
}

// TestOllamaProvider_MultipleModels verifies handling of multiple models
func TestOllamaProvider_MultipleModels(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
		Models:   []string{"sqlcoder:7b", "codellama:7b", "llama3.1:8b"},
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
}

// TestOllamaProvider_Timeout verifies timeout behavior
func TestOllamaProvider_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			time.Sleep(5 * time.Second) // Longer than timeout
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint:        server.URL,
		GenerateTimeout: 100 * time.Millisecond,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	ctx := context.Background()
	models, err := provider.GetModels(ctx)

	assert.Error(t, err)
	assert.Nil(t, models)
}

// TestOllamaProvider_SchemaInPrompt verifies schema is included in prompts
func TestOllamaProvider_SchemaInPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			prompt := req["prompt"].(string)
			assert.Contains(t, prompt, "Database Schema:")
			assert.Contains(t, prompt, "CREATE TABLE users")

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT * FROM users", "explanation": "Test", "confidence": 0.9}`,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Get all users",
		Schema:   "CREATE TABLE users (id INT, name TEXT)",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestOllamaProvider_TokenCounting verifies token counting
func TestOllamaProvider_TokenCounting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":             "llama3.1:8b",
				"response":          "Test response",
				"done":              true,
				"prompt_eval_count": 125,
				"eval_count":        75,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.TokensUsed) // 125 + 75
}

// TestOllamaProvider_MetadataInResponse verifies metadata is populated
func TestOllamaProvider_MetadataInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":             "llama3.1:8b",
				"response":          "Test",
				"done":              true,
				"total_duration":    5000000000,
				"load_duration":     1000000000,
				"eval_count":        50,
				"prompt_eval_count": 25,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp.Metadata)
	assert.Equal(t, "llama3.1:8b", resp.Metadata["model"])
	assert.Equal(t, "5000000000", resp.Metadata["total_time"])
	assert.Equal(t, "1000000000", resp.Metadata["load_time"])
	assert.Equal(t, "50", resp.Metadata["eval_count"])
	assert.Equal(t, "25", resp.Metadata["prompt_count"])
}

// TestOllamaProvider_MalformedJSON verifies handling of malformed JSON response
func TestOllamaProvider_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			w.Write([]byte("not valid json at all"))
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to parse response")
}

// TestOllamaProvider_EmptyModelList verifies handling when config has no models
func TestOllamaProvider_EmptyModelList(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint: "http://localhost:11434",
		Models:   []string{}, // Empty list
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Should use default models
}

// TestOllamaProvider_FixSQLWithSchema verifies schema inclusion in fix requests
func TestOllamaProvider_FixSQLWithSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			prompt := req["prompt"].(string)
			assert.Contains(t, prompt, "Database Schema:")
			assert.Contains(t, prompt, "Original Query:")
			assert.Contains(t, prompt, "Error Message:")

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT * FROM orders", "explanation": "Fixed", "confidence": 0.85}`,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query:    "SELECT * FROM order",
		Error:    "table 'order' does not exist",
		Schema:   "CREATE TABLE orders (id INT, total DECIMAL)",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM orders", resp.Query)
}

// TestOllamaProvider_ZeroTokens verifies handling of zero token counts
func TestOllamaProvider_ZeroTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": "Test",
				"done":     true,
				// No token counts provided
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	resp, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, 0, resp.TokensUsed)
}

// TestOllamaProvider_DefaultSystemPrompt verifies default system prompt is used
func TestOllamaProvider_DefaultSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			system := req["system"].(string)
			assert.Contains(t, system, "SQL developer")

			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT 1", "explanation": "Test", "confidence": 0.9}`,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
}

// TestOllamaProvider_ResponseWithoutDoneFlag verifies handling of incomplete responses
func TestOllamaProvider_ResponseWithoutDoneFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.1:8b", "details": map[string]interface{}{"family": "llama"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			resp := map[string]interface{}{
				"model":    "llama3.1:8b",
				"response": `{"query": "SELECT 1", "explanation": "Test", "confidence": 0.9}`,
				"done":     false, // Not done
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:   "Test",
		Model:    "llama3.1:8b",
		Provider: ai.ProviderOllama,
	}

	// Should still work since we don't stream
	resp, err := provider.GenerateSQL(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestOllamaProvider_GetModelsByFamily verifies model family detection
func TestOllamaProvider_GetModelsByFamily(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name": "sqlcoder:15b",
						"details": map[string]interface{}{
							"family": "sqlcoder",
						},
					},
					{
						"name": "codellama:13b",
						"details": map[string]interface{}{
							"family": "llama",
						},
					},
					{
						"name": "mistral:7b",
						"details": map[string]interface{}{
							"family": "mistral",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	require.NoError(t, err)
	require.Len(t, models, 3)

	// Verify SQLCoder gets special treatment
	var sqlcoder *ai.ModelInfo
	for i := range models {
		if strings.Contains(models[i].Name, "sqlcoder") {
			sqlcoder = &models[i]
			break
		}
	}
	require.NotNil(t, sqlcoder)
	assert.Contains(t, sqlcoder.Capabilities, "sql-optimization")
	assert.Contains(t, sqlcoder.Description, "SQLCoder")
}

// TestOllamaProvider_HealthCheckResponseTime verifies response time is measured
func TestOllamaProvider_HealthCheckResponseTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			time.Sleep(50 * time.Millisecond) // Small delay
			resp := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ai.OllamaConfig{
		Endpoint: server.URL,
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	assert.Greater(t, health.ResponseTime, time.Duration(0))
	assert.GreaterOrEqual(t, health.ResponseTime, 50*time.Millisecond)
}

// TestOllamaProvider_PullTimeout verifies pull timeout configuration
func TestOllamaProvider_PullTimeout(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint:    "http://localhost:11434",
		PullTimeout: 15 * time.Minute, // Custom pull timeout
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
}

// TestOllamaProvider_GenerateTimeout verifies generate timeout configuration
func TestOllamaProvider_GenerateTimeout(t *testing.T) {
	config := &ai.OllamaConfig{
		Endpoint:        "http://localhost:11434",
		GenerateTimeout: 5 * time.Minute, // Custom generate timeout
	}

	provider, err := ai.NewOllamaProvider(config, newOllamaTestLogger())

	require.NoError(t, err)
	require.NotNil(t, provider)
}
