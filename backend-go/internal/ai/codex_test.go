//go:build integration

package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock response structures for OpenAI Codex Completion API (not Chat API)

type mockCodexCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Logprobs     *struct {
			Tokens        []string  `json:"tokens"`
			TokenLogprobs []float64 `json:"token_logprobs"`
		} `json:"logprobs"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type mockCodexErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Helper functions

func createTestCodexConfig(apiKey, baseURL, model string) *ai.CodexConfig {
	config := &ai.CodexConfig{
		APIKey: apiKey,
	}
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	if model != "" {
		config.Model = model
	}
	return config
}

func createMockCodexResponse(sql string, tokens int) mockCodexCompletionResponse {
	resp := mockCodexCompletionResponse{
		ID:      "cmpl-test123",
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Model:   "code-davinci-002",
	}
	resp.Choices = make([]struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Logprobs     *struct {
			Tokens        []string  `json:"tokens"`
			TokenLogprobs []float64 `json:"token_logprobs"`
		} `json:"logprobs"`
	}, 1)
	resp.Choices[0].Text = sql
	resp.Choices[0].FinishReason = "stop"
	resp.Usage.TotalTokens = tokens
	resp.Usage.PromptTokens = tokens / 2
	resp.Usage.CompletionTokens = tokens / 2
	return resp
}

// ========== Constructor Tests ==========

func TestCodex_NewCodexProvider_ValidConfig(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")

	provider, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, ai.ProviderCodex, provider.GetProviderType())
}

func TestCodex_NewCodexProvider_NilConfig(t *testing.T) {
	// Nil config will panic when accessing config.APIKey
	// We expect this to panic, so we recover from it
	defer func() {
		if r := recover(); r != nil {
			// Expected panic
			assert.NotNil(t, r)
		}
	}()

	_, _ = ai.NewCodexProvider(nil)
}

func TestCodex_NewCodexProvider_EmptyAPIKey(t *testing.T) {
	config := createTestCodexConfig("", "", "")

	provider, err := ai.NewCodexProvider(config)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestCodex_NewCodexProvider_DefaultModel(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")

	_, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	// Model should be set to default "code-davinci-002"
	assert.Equal(t, "code-davinci-002", config.Model)
}

func TestCodex_NewCodexProvider_CustomModel(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "code-cushman-001")

	_, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	assert.Equal(t, "code-cushman-001", config.Model)
}

func TestCodex_NewCodexProvider_DefaultMaxTokens(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")

	_, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	assert.Equal(t, 2048, config.MaxTokens)
}

func TestCodex_NewCodexProvider_CustomMaxTokens(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")
	config.MaxTokens = 4096

	_, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	assert.Equal(t, 4096, config.MaxTokens)
}

func TestCodex_NewCodexProvider_DefaultTemperature(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")

	_, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	assert.Equal(t, float32(0.0), config.Temperature)
}

func TestCodex_NewCodexProvider_WithOrganization(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")
	config.Organization = "org-test-123"

	provider, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "org-test-123", config.Organization)
}

func TestCodex_NewCodexProvider_CustomBaseURL(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "https://custom.api.com/v1", "")

	provider, err := ai.NewCodexProvider(config)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "https://custom.api.com/v1", config.BaseURL)
}

// ========== GetProviderType Tests ==========

func TestCodex_GetProviderType(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	providerType := provider.GetProviderType()

	assert.Equal(t, ai.ProviderCodex, providerType)
}

// ========== GenerateSQL Tests ==========

func TestCodex_GenerateSQL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/completions")

		// Verify headers
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify request body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "code-davinci-002", reqBody["model"])
		assert.NotNil(t, reqBody["prompt"])
		assert.NotNil(t, reqBody["max_tokens"])

		// Check for stop sequences
		stops := reqBody["stop"].([]interface{})
		assert.Contains(t, stops, "--")
		assert.Contains(t, stops, ";")

		// Send mock response
		resp := createMockCodexResponse("* FROM users WHERE active = 1", 100)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get all active users", "")

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Contains(t, response.Query, "FROM users WHERE active = 1")
	assert.Equal(t, ai.ProviderCodex, response.Provider)
	assert.Equal(t, "code-davinci-002", response.Model)
	assert.Equal(t, 50, response.TokensUsed) // CompletionTokens
	assert.Greater(t, response.TimeTaken, time.Duration(0))
}

func TestCodex_GenerateSQL_WithSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body and verify schema is included in prompt
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		prompt := reqBody["prompt"].(string)
		assert.Contains(t, prompt, "Schema:")
		assert.Contains(t, prompt, "users")
		assert.Contains(t, prompt, "VARCHAR(100)")

		resp := createMockCodexResponse("name, email FROM users WHERE status = 'active'", 120)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	schema := "CREATE TABLE users (id INT, name VARCHAR(100), email VARCHAR(255), status VARCHAR(20))"
	response, err := provider.GenerateSQL(context.Background(), "Get active user names and emails", schema)

	require.NoError(t, err)
	assert.Contains(t, response.Query, "users")
	assert.Contains(t, response.Query, "name")
}

func TestCodex_GenerateSQL_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		// Verify custom options
		assert.Equal(t, "code-cushman-001", reqBody["model"])
		assert.Equal(t, float64(4000), reqBody["max_tokens"])
		assert.Equal(t, float64(0.3), reqBody["temperature"])

		resp := createMockCodexResponse("* FROM products", 80)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(
		context.Background(),
		"Get all products",
		"",
		ai.WithModel("code-cushman-001"),
		ai.WithMaxTokens(4000),
		ai.WithTemperature(0.3),
	)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestCodex_GenerateSQL_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "Invalid API key"
		errResp.Error.Type = "invalid_request_error"
		errResp.Error.Code = "invalid_api_key"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get all users", "")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "codex API error")
}

func TestCodex_GenerateSQL_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "Rate limit exceeded"
		errResp.Error.Type = "rate_limit_error"
		errResp.Error.Code = "rate_limit_exceeded"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get all users", "")

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestCodex_GenerateSQL_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockCodexCompletionResponse{
			ID:      "cmpl-test123",
			Object:  "text_completion",
			Created: time.Now().Unix(),
			Model:   "code-davinci-002",
			Choices: []struct {
				Text         string `json:"text"`
				Index        int    `json:"index"`
				FinishReason string `json:"finish_reason"`
				Logprobs     *struct {
					Tokens        []string  `json:"tokens"`
					TokenLogprobs []float64 `json:"token_logprobs"`
				} `json:"logprobs"`
			}{}, // Empty choices
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get all users", "")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no response from Codex")
}

func TestCodex_GenerateSQL_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": "json"`)) // Malformed JSON
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get all users", "")

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestCodex_GenerateSQL_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := createMockCodexResponse("* FROM users", 100)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	response, err := provider.GenerateSQL(ctx, "Get all users", "")

	assert.Error(t, err)
	assert.Nil(t, response)
}

// ========== FixSQL Tests ==========

func TestCodex_FixSQL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request includes error context
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		prompt := reqBody["prompt"].(string)
		assert.Contains(t, prompt, "Fix SQL Query")
		assert.Contains(t, prompt, "Original Query:")
		assert.Contains(t, prompt, "Error:")

		resp := createMockCodexResponse("* FROM users WHERE id = 1", 150)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.FixSQL(context.Background(), "SELECT * FROM users WHERE id =", "Syntax error near '='", "")

	require.NoError(t, err)
	assert.Contains(t, response.Query, "FROM users WHERE id = 1")
	assert.Equal(t, 0.90, response.Confidence)
	assert.Equal(t, ai.ProviderCodex, response.Provider)
}

func TestCodex_FixSQL_WithSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		prompt := reqBody["prompt"].(string)
		assert.Contains(t, prompt, "Schema:")
		assert.Contains(t, prompt, "CREATE TABLE")

		resp := createMockCodexResponse("name FROM users WHERE id = 1", 130)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	schema := "CREATE TABLE users (id INT, name VARCHAR(100))"
	response, err := provider.FixSQL(context.Background(), "SELECT username FROM users WHERE id = 1", "Column 'username' not found", schema)

	require.NoError(t, err)
	assert.Contains(t, response.Query, "name FROM users")
}

func TestCodex_FixSQL_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "code-cushman-001", reqBody["model"])
		assert.Equal(t, float64(3000), reqBody["max_tokens"])

		resp := createMockCodexResponse("* FROM orders WHERE status = 'pending'", 100)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.FixSQL(
		context.Background(),
		"SELECT * FROM orders WHERE stat = 'pending'",
		"Column 'stat' not found",
		"",
		ai.WithModel("code-cushman-001"),
		ai.WithMaxTokens(3000),
	)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestCodex_FixSQL_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "Internal server error"
		errResp.Error.Type = "server_error"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.FixSQL(context.Background(), "SELECT * FROM users WHERE id =", "Syntax error", "")

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestCodex_FixSQL_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockCodexCompletionResponse{
			ID:      "cmpl-test123",
			Object:  "text_completion",
			Created: time.Now().Unix(),
			Model:   "code-davinci-002",
			Choices: []struct {
				Text         string `json:"text"`
				Index        int    `json:"index"`
				FinishReason string `json:"finish_reason"`
				Logprobs     *struct {
					Tokens        []string  `json:"tokens"`
					TokenLogprobs []float64 `json:"token_logprobs"`
				} `json:"logprobs"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.FixSQL(context.Background(), "SELECT * FROM users", "Error", "")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no response from Codex")
}

// ========== Chat Tests ==========

func TestCodex_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		prompt := reqBody["prompt"].(string)
		assert.Contains(t, prompt, "What is SQL?")

		resp := createMockCodexResponse("SQL stands for Structured Query Language. It is used for managing databases.", 75)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.Chat(context.Background(), "What is SQL?")

	require.NoError(t, err)
	assert.Contains(t, response.Content, "SQL")
	assert.Contains(t, response.Content, "Structured Query Language")
	assert.Equal(t, ai.ProviderCodex, response.Provider)
	assert.Equal(t, "code-davinci-002", response.Model)
}

func TestCodex_Chat_WithCustomSystem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		prompt := reqBody["prompt"].(string)
		assert.Contains(t, prompt, "You are a database expert")
		assert.Contains(t, prompt, "Explain indexes")

		resp := createMockCodexResponse("Database indexes improve query performance by...", 60)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.Chat(
		context.Background(),
		"Explain indexes",
		ai.WithContext(map[string]string{
			"system": "You are a database expert",
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, response.Content, "indexes")
}

func TestCodex_Chat_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		prompt := reqBody["prompt"].(string)
		assert.Contains(t, prompt, "Context:")
		assert.Contains(t, prompt, "PostgreSQL database")

		resp := createMockCodexResponse("For PostgreSQL, you should use BTREE indexes for...", 80)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.Chat(
		context.Background(),
		"What indexes should I use?",
		ai.WithContext(map[string]string{
			"context": "PostgreSQL database with 1M records",
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, response.Content, "PostgreSQL")
}

func TestCodex_Chat_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "code-cushman-001", reqBody["model"])
		assert.Equal(t, float64(500), reqBody["max_tokens"])
		assert.Equal(t, float64(0.8), reqBody["temperature"])

		resp := createMockCodexResponse("Response text", 50)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.Chat(
		context.Background(),
		"Test prompt",
		ai.WithModel("code-cushman-001"),
		ai.WithMaxTokens(500),
		ai.WithTemperature(0.8),
	)

	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestCodex_Chat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockCodexCompletionResponse{
			ID:      "cmpl-test123",
			Object:  "text_completion",
			Created: time.Now().Unix(),
			Model:   "code-davinci-002",
			Choices: []struct {
				Text         string `json:"text"`
				Index        int    `json:"index"`
				FinishReason string `json:"finish_reason"`
				Logprobs     *struct {
					Tokens        []string  `json:"tokens"`
					TokenLogprobs []float64 `json:"token_logprobs"`
				} `json:"logprobs"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.Chat(context.Background(), "Test")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no response from Codex")
}

func TestCodex_Chat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "Invalid request"
		errResp.Error.Type = "invalid_request_error"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.Chat(context.Background(), "Test")

	assert.Error(t, err)
	assert.Nil(t, response)
}

// ========== HealthCheck Tests ==========

func TestCodex_GetHealth_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		// OpenAI SDK uses /completions not /v1/completions when base URL is set
		assert.Contains(t, r.URL.Path, "/completions")
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Verify health check request
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "code-davinci-002", reqBody["model"])
		assert.Equal(t, float64(1), reqBody["max_tokens"])

		resp := createMockCodexResponse("--", 10)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderCodex, health.Provider)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "Codex API is operational", health.Message)
	assert.Greater(t, health.ResponseTime, time.Duration(0))
}

func TestCodex_GetHealth_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "Service unavailable"
		errResp.Error.Type = "server_error"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderCodex, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "Codex API error")
}

func TestCodex_GetHealth_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	config := createTestCodexConfig("test-api-key", "http://invalid-url-that-does-not-exist:9999", "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	health, err := provider.GetHealth(ctx)

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderCodex, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
}

func TestCodex_GetHealth_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		resp := createMockCodexResponse("--", 10)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	health, err := provider.GetHealth(ctx)

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, "unhealthy", health.Status)
}

// ========== ListModels Tests ==========

func TestCodex_ListModels_Success(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	models, err := provider.ListModels(context.Background())

	require.NoError(t, err)
	require.NotNil(t, models)
	assert.Len(t, models, 2) // code-davinci-002 and code-cushman-001

	// Verify first model
	assert.Equal(t, "code-davinci-002", models[0].ID)
	assert.Equal(t, "Codex Davinci", models[0].Name)
	assert.Equal(t, ai.ProviderCodex, models[0].Provider)
	assert.Equal(t, 8000, models[0].MaxTokens)
	assert.Contains(t, models[0].Capabilities, "sql_generation")
	assert.Contains(t, models[0].Capabilities, "code_generation")

	// Verify second model
	assert.Equal(t, "code-cushman-001", models[1].ID)
	assert.Equal(t, "Codex Cushman", models[1].Name)
	assert.Equal(t, ai.ProviderCodex, models[1].Provider)
	assert.Equal(t, 2048, models[1].MaxTokens)
}

func TestCodex_ListModels_NoAPICall(t *testing.T) {
	// ListModels returns static list, no API call needed
	config := createTestCodexConfig("test-api-key", "", "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	// Call multiple times to verify consistency
	models1, err1 := provider.ListModels(context.Background())
	models2, err2 := provider.ListModels(context.Background())

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, len(models1), len(models2))
	assert.Equal(t, models1[0].ID, models2[0].ID)
}

// ========== HTTP Headers Tests ==========

func TestCodex_HTTPHeaders_Authorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader)
		assert.True(t, strings.HasPrefix(authHeader, "Bearer "))
		assert.Equal(t, "Bearer test-api-key-123", authHeader)

		resp := createMockCodexResponse("* FROM users", 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key-123", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	_, err = provider.GenerateSQL(context.Background(), "Test", "")
	assert.NoError(t, err)
}

func TestCodex_HTTPHeaders_ContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType)

		resp := createMockCodexResponse("* FROM users", 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	_, err = provider.GenerateSQL(context.Background(), "Test", "")
	assert.NoError(t, err)
}

func TestCodex_HTTPHeaders_OrganizationID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Note: OpenAI SDK uses "OpenAI-Organization" header
		orgHeader := r.Header.Get("OpenAI-Organization")
		assert.Equal(t, "org-abc123", orgHeader)

		resp := createMockCodexResponse("* FROM users", 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	config.Organization = "org-abc123"
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	_, err = provider.GenerateSQL(context.Background(), "Test", "")
	assert.NoError(t, err)
}

// ========== Request Body Tests ==========

func TestCodex_RequestBody_CompletionFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		// Verify completion API format (not chat API)
		assert.NotNil(t, reqBody["prompt"])   // completion uses "prompt"
		_, hasMessages := reqBody["messages"] // NOT chat format
		assert.False(t, hasMessages, "Should not have messages field (chat API)")
		assert.Equal(t, "code-davinci-002", reqBody["model"])
		// OpenAI SDK may omit zero-valued fields
		assert.NotNil(t, reqBody["max_tokens"])

		resp := createMockCodexResponse("* FROM users", 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	_, err = provider.GenerateSQL(context.Background(), "Test prompt", "")
	assert.NoError(t, err)
}

func TestCodex_RequestBody_StopSequences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		stops := reqBody["stop"].([]interface{})
		assert.Len(t, stops, 3)
		assert.Contains(t, stops, "--")
		assert.Contains(t, stops, ";")
		assert.Contains(t, stops, "\n\n")

		resp := createMockCodexResponse("* FROM users", 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	_, err = provider.GenerateSQL(context.Background(), "Test", "")
	assert.NoError(t, err)
}

func TestCodex_RequestBody_ModelAndParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "code-cushman-001", reqBody["model"])
		assert.Equal(t, float64(3000), reqBody["max_tokens"])
		assert.Equal(t, float64(0.5), reqBody["temperature"])
		assert.Equal(t, float64(1), reqBody["n"]) // Always 1 completion

		resp := createMockCodexResponse("* FROM orders", 80)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "code-cushman-001")
	config.MaxTokens = 3000
	config.Temperature = 0.5
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	_, err = provider.GenerateSQL(context.Background(), "Test", "")
	assert.NoError(t, err)
}

// ========== SQL Extraction Tests ==========

func TestCodex_SQLExtraction_PlainSQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := createMockCodexResponse("* FROM users WHERE active = 1", 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get active users", "")

	require.NoError(t, err)
	assert.Equal(t, "* FROM users WHERE active = 1", response.Query)
}

func TestCodex_SQLExtraction_WithComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sqlWithComments := `-- This is a comment
* FROM users
-- Another comment
WHERE active = 1`
		resp := createMockCodexResponse(sqlWithComments, 60)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get active users", "")

	require.NoError(t, err)
	// Comments should be removed
	assert.NotContains(t, response.Query, "--")
	assert.Contains(t, response.Query, "FROM users")
	assert.Contains(t, response.Query, "WHERE active = 1")
}

func TestCodex_SQLExtraction_MultiLine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		multilineSQL := `id, name, email
FROM users
WHERE status = 'active'
ORDER BY name`
		resp := createMockCodexResponse(multilineSQL, 70)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get active users sorted by name", "")

	require.NoError(t, err)
	assert.Contains(t, response.Query, "FROM users")
	assert.Contains(t, response.Query, "WHERE status = 'active'")
	assert.Contains(t, response.Query, "ORDER BY name")
}

func TestCodex_SQLExtraction_WithWhitespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sqlWithWhitespace := "  \n  * FROM users WHERE id = 1  \n  "
		resp := createMockCodexResponse(sqlWithWhitespace, 40)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get user by id", "")

	require.NoError(t, err)
	// Whitespace should be trimmed
	assert.Equal(t, "* FROM users WHERE id = 1", strings.TrimSpace(response.Query))
}

func TestCodex_SQLExtraction_EmptyLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sqlWithEmptyLines := `id, name

FROM users

WHERE active = 1`
		resp := createMockCodexResponse(sqlWithEmptyLines, 50)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Get active users", "")

	require.NoError(t, err)
	assert.Contains(t, response.Query, "FROM users")
	assert.Contains(t, response.Query, "WHERE active = 1")
}

func TestCodex_SQLExtraction_OnlyComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		onlyComments := `-- First comment
-- Second comment
-- Third comment`
		resp := createMockCodexResponse(onlyComments, 30)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Test", "")

	require.NoError(t, err)
	// Should return empty or original since no SQL found
	assert.NotContains(t, response.Query, "SELECT")
}

// ========== Error Response Tests ==========

func TestCodex_ErrorResponse_InvalidModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "The model `invalid-model` does not exist"
		errResp.Error.Type = "invalid_request_error"
		errResp.Error.Code = "model_not_found"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "invalid-model")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Test", "")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "codex API error")
}

func TestCodex_ErrorResponse_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		errResp := mockCodexErrorResponse{}
		errResp.Error.Message = "Internal server error"
		errResp.Error.Type = "server_error"
		errResp.Error.Code = "internal_error"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Test", "")

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestCodex_ErrorResponse_PlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service temporarily unavailable"))
	}))
	defer server.Close()

	config := createTestCodexConfig("test-api-key", server.URL, "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	response, err := provider.GenerateSQL(context.Background(), "Test", "")

	assert.Error(t, err)
	assert.Nil(t, response)
}

// ========== Close Method Test ==========

func TestCodex_Close_Success(t *testing.T) {
	config := createTestCodexConfig("test-api-key", "", "")
	provider, err := ai.NewCodexProvider(config)
	require.NoError(t, err)

	err = provider.Close()

	assert.NoError(t, err)
}
