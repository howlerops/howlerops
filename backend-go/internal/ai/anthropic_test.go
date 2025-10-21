//go:build integration
// +build integration

package ai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAnthropicProvider_Success tests successful provider creation
func TestNewAnthropicProvider_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "https://api.anthropic.com",
		Version: "2023-06-01",
		Models:  []string{"claude-3-5-sonnet-20241022"},
	}

	provider, err := ai.NewAnthropicProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, ai.ProviderAnthropic, provider.GetProviderType())
}

// TestNewAnthropicProvider_NilConfig tests that nil config is rejected
func TestNewAnthropicProvider_NilConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	provider, err := ai.NewAnthropicProvider(nil, logger)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

// TestNewAnthropicProvider_EmptyAPIKey tests that empty API key is rejected
func TestNewAnthropicProvider_EmptyAPIKey(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "API key is required")
}

// TestNewAnthropicProvider_DefaultBaseURL tests default BaseURL is set
func TestNewAnthropicProvider_DefaultBaseURL(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "https://api.anthropic.com", config.BaseURL)
}

// TestNewAnthropicProvider_DefaultVersion tests default version is set
func TestNewAnthropicProvider_DefaultVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		Version: "",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "2023-06-01", config.Version)
}

// TestNewAnthropicProvider_DefaultModels tests default models are set
func TestNewAnthropicProvider_DefaultModels(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
		Models: []string{},
	}

	provider, err := ai.NewAnthropicProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotEmpty(t, config.Models)
	assert.Contains(t, config.Models, "claude-3-5-sonnet-20241022")
	assert.Contains(t, config.Models, "claude-3-5-haiku-20241022")
	assert.Contains(t, config.Models, "claude-3-opus-20240229")
}

// TestNewAnthropicProvider_CustomConfig tests custom configuration is preserved
func TestNewAnthropicProvider_CustomConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "custom-key",
		BaseURL: "https://custom.api.com",
		Version: "2024-01-01",
		Models:  []string{"custom-model"},
	}

	provider, err := ai.NewAnthropicProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "https://custom.api.com", config.BaseURL)
	assert.Equal(t, "2024-01-01", config.Version)
	assert.Equal(t, []string{"custom-model"}, config.Models)
}

// TestNewAnthropicProvider_NilLogger tests that nil logger works
func TestNewAnthropicProvider_NilLogger(t *testing.T) {
	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, nil)

	require.NoError(t, err)
	require.NotNil(t, provider)
}

// TestAnthropicGenerateSQL_Success tests successful SQL generation
func TestAnthropicGenerateSQL_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock response
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": `{"query":"SELECT * FROM users","explanation":"This query selects all users","confidence":0.95,"suggestions":["Add WHERE clause"],"warnings":["May return large results"]}`,
				},
			},
			"model":       "claude-3-5-sonnet-20241022",
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Version: "2023-06-01",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Schema:      "users (id, name, email)",
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   2048,
		Temperature: 0.1,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SELECT * FROM users", response.Query)
	assert.Equal(t, "This query selects all users", response.Explanation)
	assert.Equal(t, 0.95, response.Confidence)
	assert.Equal(t, []string{"Add WHERE clause"}, response.Suggestions)
	assert.Equal(t, []string{"May return large results"}, response.Warnings)
	assert.Equal(t, ai.ProviderAnthropic, response.Provider)
	assert.Equal(t, "claude-3-5-sonnet-20241022", response.Model)
	assert.Equal(t, 150, response.TokensUsed)
}

// TestAnthropicGenerateSQL_APIError tests API error handling
func TestAnthropicGenerateSQL_APIError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "Invalid API key provided",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Get all users",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Invalid API key provided")
}

// TestGenerateSQL_NetworkError tests network error handling
func TestAnthropicGenerateSQL_NetworkError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "http://invalid-url-that-does-not-exist.local",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Get all users",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
}

// TestGenerateSQL_EmptyResponse tests handling of empty response
func TestAnthropicGenerateSQL_EmptyResponse(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning empty content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "msg_123",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]string{},
			"model":   "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 0,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Get all users",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no content in response")
}

// TestGenerateSQL_NonJSONResponse tests handling of non-JSON response
func TestAnthropicGenerateSQL_NonJSONResponse(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning SQL in code block
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "```sql\nSELECT * FROM users\n```",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 20,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Get all users",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SELECT * FROM users", response.Query)
	assert.Equal(t, 0.8, response.Confidence) // Lower confidence for non-JSON
}

// TestGenerateSQL_ContextCancellation tests context cancellation
func TestAnthropicGenerateSQL_ContextCancellation(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Get all users",
		Model:  "claude-3-5-sonnet-20241022",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	response, err := provider.GenerateSQL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
}

// TestFixSQL_Success tests successful SQL fixing
func TestAnthropicFixSQL_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)

		// Read and verify body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		message := messages[0].(map[string]interface{})
		content := message["content"].(string)
		assert.Contains(t, content, "Original Query")
		assert.Contains(t, content, "Error Message")

		// Return fixed SQL
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": `{"query":"SELECT * FROM users WHERE id = 1","explanation":"Fixed syntax error","confidence":0.90}`,
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query:       "SELECT * FROM users WHERE id =",
		Error:       "syntax error at end of input",
		Schema:      "users (id, name, email)",
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   2048,
		Temperature: 0.1,
	}

	response, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SELECT * FROM users WHERE id = 1", response.Query)
	assert.Equal(t, "Fixed syntax error", response.Explanation)
	assert.Equal(t, 0.90, response.Confidence)
}

// TestFixSQL_APIError tests API error handling in FixSQL
func TestAnthropicFixSQL_APIError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "Invalid authentication",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT * FROM",
		Error: "incomplete query",
		Model: "claude-3-5-sonnet-20241022",
	}

	response, err := provider.FixSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Invalid authentication")
}

// TestFixSQL_NetworkError tests network error handling in FixSQL
func TestAnthropicFixSQL_NetworkError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "http://nonexistent-server.local",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT * FROM",
		Error: "incomplete query",
		Model: "claude-3-5-sonnet-20241022",
	}

	response, err := provider.FixSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
}

// TestFixSQL_EmptyResponse tests empty response handling in FixSQL
func TestAnthropicFixSQL_EmptyResponse(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning empty content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "msg_123",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]string{},
			"model":   "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 0,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT * FROM",
		Error: "incomplete query",
		Model: "claude-3-5-sonnet-20241022",
	}

	response, err := provider.FixSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no content in response")
}

// TestFixSQL_ExtractFromCodeBlock tests SQL extraction from code blocks
func TestAnthropicFixSQL_ExtractFromCodeBlock(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning SQL in code block
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Here's the fixed query:\n```sql\nSELECT * FROM users WHERE active = true\n```",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 30,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT * FROM users WHERE",
		Error: "incomplete WHERE clause",
		Model: "claude-3-5-sonnet-20241022",
	}

	response, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SELECT * FROM users WHERE active = true", response.Query)
}

// TestFixSQL_MalformedJSON tests handling of malformed JSON responses
func TestAnthropicFixSQL_MalformedJSON(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning plain SQL text
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "SELECT * FROM users",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  100,
				"output_tokens": 10,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query: "SELECT * FROM",
		Error: "incomplete query",
		Model: "claude-3-5-sonnet-20241022",
	}

	response, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SELECT * FROM users", response.Query)
	assert.Equal(t, 0.8, response.Confidence) // Lower confidence for non-JSON
}

// TestChat_Success tests successful chat interaction
func TestAnthropicChat_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		// Verify body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "claude-3-5-sonnet-20241022", reqBody["model"])
		assert.Contains(t, reqBody["system"], "helpful assistant")

		messages := reqBody["messages"].([]interface{})
		message := messages[0].(map[string]interface{})
		assert.Equal(t, "user", message["role"])
		assert.Equal(t, "What is SQL?", message["content"])

		// Return response
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "SQL stands for Structured Query Language.",
				},
			},
			"model":       "claude-3-5-sonnet-20241022",
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  50,
				"output_tokens": 20,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Version: "2023-06-01",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:      "What is SQL?",
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SQL stands for Structured Query Language.", response.Content)
	assert.Equal(t, ai.ProviderAnthropic, response.Provider)
	assert.Equal(t, "claude-3-5-sonnet-20241022", response.Model)
	assert.Equal(t, 70, response.TokensUsed)
	assert.Equal(t, "end_turn", response.Metadata["stop_reason"])
}

// TestChat_NilRequest tests nil request handling
func TestAnthropicChat_NilRequest(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	response, err := provider.Chat(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

// TestChat_WithCustomSystem tests custom system prompt
func TestAnthropicChat_WithCustomSystem(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom system prompt
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Contains(t, reqBody["system"], "database expert")

		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Expert database response",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  50,
				"output_tokens": 20,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "Explain indexing",
		System: "You are a database expert",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "Expert database response", response.Content)
}

// TestChat_WithContext tests context in system prompt
func TestAnthropicChat_WithContext(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context in system prompt
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		systemPrompt := reqBody["system"].(string)
		assert.Contains(t, systemPrompt, "Additional context")
		assert.Contains(t, systemPrompt, "PostgreSQL database")

		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "PostgreSQL-specific response",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  50,
				"output_tokens": 20,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:  "Explain JSONB",
		Context: "PostgreSQL database",
		Model:   "claude-3-5-sonnet-20241022",
	}

	response, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "PostgreSQL-specific response", response.Content)
}

// TestChat_EmptyResponse tests empty response handling
func TestAnthropicChat_EmptyResponse(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning empty content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "msg_123",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]string{},
			"model":   "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  50,
				"output_tokens": 0,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "Hello",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.Chat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no content in response")
}

// TestChat_APIError tests API error handling in Chat
func TestAnthropicChat_APIError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "rate_limit_error",
				"message": "Rate limit exceeded",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "Hello",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.Chat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Rate limit exceeded")
}

// TestHealthCheck_Healthy tests healthy service check
func TestAnthropicHealthCheck_Healthy(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify health check request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "claude-3-5-haiku-20241022", reqBody["model"])
		assert.Equal(t, float64(10), reqBody["max_tokens"])

		resp := map[string]interface{}{
			"id":   "msg_health",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Hi",
				},
			},
			"model": "claude-3-5-haiku-20241022",
			"usage": map[string]int{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Version: "2023-06-01",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderAnthropic, health.Provider)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "Service is operational", health.Message)
	assert.Greater(t, health.ResponseTime, time.Duration(0))
}

// TestHealthCheck_Unhealthy tests unhealthy service check
func TestAnthropicHealthCheck_Unhealthy(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "Service temporarily unavailable")
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderAnthropic, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "503")
	assert.Contains(t, health.Message, "Service temporarily unavailable")
}

// TestHealthCheck_NetworkError tests network error in health check
func TestAnthropicHealthCheck_NetworkError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "http://nonexistent-server-for-health.local",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderAnthropic, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "Request failed")
}

// TestHealthCheck_InvalidAPIKey tests health check with invalid API key
func TestAnthropicHealthCheck_InvalidAPIKey(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderAnthropic, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "401")
}

// TestHealthCheck_Timeout tests health check timeout
func TestAnthropicHealthCheck_Timeout(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	health, err := provider.HealthCheck(ctx)

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderAnthropic, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "Request failed")
}

// TestGetModels tests retrieving available models
func TestAnthropicGetModels(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	tests := []struct {
		name           string
		configModels   []string
		expectedCount  int
		expectedModels []string
	}{
		{
			name:           "default models",
			configModels:   []string{},
			expectedCount:  3,
			expectedModels: []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"},
		},
		{
			name:           "custom models",
			configModels:   []string{"claude-3-5-sonnet-20241022"},
			expectedCount:  1,
			expectedModels: []string{"claude-3-5-sonnet-20241022"},
		},
		{
			name:           "multiple custom models",
			configModels:   []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"},
			expectedCount:  2,
			expectedModels: []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ai.AnthropicConfig{
				APIKey: "test-api-key",
				Models: tt.configModels,
			}

			provider, err := ai.NewAnthropicProvider(config, logger)
			require.NoError(t, err)

			models, err := provider.GetModels(context.Background())

			require.NoError(t, err)
			assert.Len(t, models, tt.expectedCount)

			for _, expectedModel := range tt.expectedModels {
				found := false
				for _, model := range models {
					if model.ID == expectedModel {
						found = true
						assert.Equal(t, expectedModel, model.Name)
						assert.Equal(t, ai.ProviderAnthropic, model.Provider)
						assert.NotEmpty(t, model.Description)
						assert.Greater(t, model.MaxTokens, 0)
						assert.Contains(t, model.Capabilities, "text-to-sql")
						assert.Contains(t, model.Capabilities, "sql-fixing")
						break
					}
				}
				assert.True(t, found, "Expected model %s not found", expectedModel)
			}
		})
	}
}

// TestGetModels_ModelDescriptions tests model-specific descriptions
func TestAnthropicGetModels_ModelDescriptions(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	tests := []struct {
		name               string
		modelID            string
		expectedDesc       string
		expectedMaxTokens  int
	}{
		{
			name:              "sonnet model",
			modelID:           "claude-3-5-sonnet-20241022",
			expectedDesc:      "Claude 3.5 Sonnet",
			expectedMaxTokens: 200000,
		},
		{
			name:              "haiku model",
			modelID:           "claude-3-5-haiku-20241022",
			expectedDesc:      "Claude 3.5 Haiku",
			expectedMaxTokens: 200000,
		},
		{
			name:              "opus model",
			modelID:           "claude-3-opus-20240229",
			expectedDesc:      "Claude 3 Opus",
			expectedMaxTokens: 200000,
		},
		{
			name:              "unknown model",
			modelID:           "claude-future-model",
			expectedDesc:      "Anthropic claude-future-model model",
			expectedMaxTokens: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ai.AnthropicConfig{
				APIKey: "test-api-key",
				Models: []string{tt.modelID},
			}

			provider, err := ai.NewAnthropicProvider(config, logger)
			require.NoError(t, err)

			models, err := provider.GetModels(context.Background())

			require.NoError(t, err)
			require.Len(t, models, 1)
			assert.Contains(t, models[0].Description, tt.expectedDesc)
			assert.Equal(t, tt.expectedMaxTokens, models[0].MaxTokens)
		})
	}
}

// TestGetProviderType tests provider type retrieval
func TestAnthropicGetProviderType(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	providerType := provider.GetProviderType()

	assert.Equal(t, ai.ProviderAnthropic, providerType)
}

// TestIsAvailable_Healthy tests IsAvailable returns true for healthy service
func TestAnthropicIsAvailable_Healthy(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_health",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Hi",
				},
			},
			"model": "claude-3-5-haiku-20241022",
			"usage": map[string]int{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	available := provider.IsAvailable(context.Background())

	assert.True(t, available)
}

// TestIsAvailable_Unhealthy tests IsAvailable returns false for unhealthy service
func TestAnthropicIsAvailable_Unhealthy(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	available := provider.IsAvailable(context.Background())

	assert.False(t, available)
}

// TestUpdateConfig_Success tests successful config update
func TestAnthropicUpdateConfig_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "https://api.anthropic.com",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	newConfig := &ai.AnthropicConfig{
		APIKey:  "new-api-key",
		BaseURL: "https://new.api.com",
		Version: "2024-01-01",
	}

	err = provider.UpdateConfig(newConfig)

	require.NoError(t, err)
}

// TestUpdateConfig_InvalidType tests update with wrong config type
func TestAnthropicUpdateConfig_InvalidType(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	// Try to update with wrong config type
	err = provider.UpdateConfig("invalid-config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// TestUpdateConfig_InvalidConfig tests update with invalid config
func TestAnthropicUpdateConfig_InvalidConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	// Try to update with empty API key
	newConfig := &ai.AnthropicConfig{
		APIKey: "",
	}

	err = provider.UpdateConfig(newConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

// TestValidateConfig_Success tests successful config validation
func TestAnthropicValidateConfig_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	validConfig := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "https://api.anthropic.com",
	}

	err = provider.ValidateConfig(validConfig)

	assert.NoError(t, err)
}

// TestValidateConfig_InvalidType tests validation with wrong config type
func TestAnthropicValidateConfig_InvalidType(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	err = provider.ValidateConfig("invalid-config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// TestValidateConfig_MissingAPIKey tests validation with missing API key
func TestAnthropicValidateConfig_MissingAPIKey(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := &ai.AnthropicConfig{
		APIKey:  "",
		BaseURL: "https://api.anthropic.com",
	}

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

// TestValidateConfig_MissingBaseURL tests validation with missing BaseURL
func TestAnthropicValidateConfig_MissingBaseURL(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	config := &ai.AnthropicConfig{
		APIKey: "test-api-key",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: "",
	}

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base URL is required")
}

// TestHTTPRequest_Headers tests that all required headers are set correctly
func TestAnthropicHTTPRequest_Headers(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server to verify headers
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()

		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Response",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-header-key",
		BaseURL: server.URL,
		Version: "2023-06-01",
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "Test",
		Model:  "claude-3-5-sonnet-20241022",
	}

	_, err = provider.Chat(context.Background(), req)
	require.NoError(t, err)

	// Verify all required headers
	assert.Equal(t, "test-header-key", capturedHeaders.Get("x-api-key"))
	assert.Equal(t, "2023-06-01", capturedHeaders.Get("anthropic-version"))
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
}

// TestHTTPRequest_BodyStructure tests that request body is correctly structured
func TestAnthropicHTTPRequest_BodyStructure(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server to verify body
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": `{"query":"SELECT 1"}`,
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   1000,
		Temperature: 0.5,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	require.NoError(t, err)

	// Verify body structure
	assert.Equal(t, "claude-3-5-sonnet-20241022", capturedBody["model"])
	assert.Equal(t, float64(1000), capturedBody["max_tokens"])
	assert.Equal(t, 0.5, capturedBody["temperature"])
	assert.NotNil(t, capturedBody["messages"])
	assert.NotEmpty(t, capturedBody["system"])

	messages := capturedBody["messages"].([]interface{})
	assert.Len(t, messages, 1)

	message := messages[0].(map[string]interface{})
	assert.Equal(t, "user", message["role"])
	assert.Contains(t, message["content"], "Get all users")
}

// TestHTTPResponse_MalformedJSON tests handling of malformed HTTP response JSON
func TestAnthropicHTTPResponse_MalformedJSON(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{invalid json")
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt: "Test",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.Chat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to parse response")
}

// TestHTTPResponse_HTTPStatusCodes tests various HTTP status code handling
func TestAnthropicHTTPResponse_HTTPStatusCodes(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	tests := []struct {
		name       string
		statusCode int
		errorBody  map[string]interface{}
		expectErr  bool
	}{
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			errorBody: map[string]interface{}{
				"type": "error",
				"error": map[string]string{
					"type":    "invalid_request_error",
					"message": "Invalid request",
				},
			},
			expectErr: true,
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			errorBody: map[string]interface{}{
				"type": "error",
				"error": map[string]string{
					"type":    "authentication_error",
					"message": "Unauthorized",
				},
			},
			expectErr: true,
		},
		{
			name:       "429 Rate Limited",
			statusCode: http.StatusTooManyRequests,
			errorBody: map[string]interface{}{
				"type": "error",
				"error": map[string]string{
					"type":    "rate_limit_error",
					"message": "Too many requests",
				},
			},
			expectErr: true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			errorBody: map[string]interface{}{
				"type": "error",
				"error": map[string]string{
					"type":    "internal_server_error",
					"message": "Server error",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.errorBody)
			}))
			defer server.Close()

			config := &ai.AnthropicConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			}

			provider, err := ai.NewAnthropicProvider(config, logger)
			require.NoError(t, err)

			req := &ai.ChatRequest{
				Prompt: "Test",
				Model:  "claude-3-5-sonnet-20241022",
			}

			response, err := provider.Chat(context.Background(), req)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
			}
		})
	}
}

// TestExtractSQL_VariousFormats tests SQL extraction from various text formats
func TestAnthropicExtractSQL_VariousFormats(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	tests := []struct {
		name         string
		responseText string
		expectedSQL  string
	}{
		{
			name:         "SQL in code block",
			responseText: "```sql\nSELECT * FROM users\n```",
			expectedSQL:  "SELECT * FROM users",
		},
		{
			name:         "SQL in generic code block",
			responseText: "```\nSELECT * FROM products\n```",
			expectedSQL:  "SELECT * FROM products",
		},
		{
			name:         "SQL in JSON",
			responseText: `{"query": "SELECT * FROM orders", "explanation": "..."}`,
			expectedSQL:  "SELECT * FROM orders",
		},
		{
			name:         "Plain SQL starting with SELECT",
			responseText: "SELECT id, name FROM customers WHERE active = true",
			expectedSQL:  "SELECT id, name FROM customers WHERE active = true",
		},
		{
			name:         "SQL with INSERT",
			responseText: "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
			expectedSQL:  "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
		},
		{
			name:         "SQL with UPDATE",
			responseText: "UPDATE products SET price = 99.99 WHERE id = 1",
			expectedSQL:  "UPDATE products SET price = 99.99 WHERE id = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]interface{}{
					"id":   "msg_123",
					"type": "message",
					"role": "assistant",
					"content": []map[string]string{
						{
							"type": "text",
							"text": tt.responseText,
						},
					},
					"model": "claude-3-5-sonnet-20241022",
					"usage": map[string]int{
						"input_tokens":  10,
						"output_tokens": 20,
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			config := &ai.AnthropicConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			}

			provider, err := ai.NewAnthropicProvider(config, logger)
			require.NoError(t, err)

			req := &ai.SQLRequest{
				Prompt: "Generate SQL",
				Model:  "claude-3-5-sonnet-20241022",
			}

			response, err := provider.GenerateSQL(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, response)
			assert.Equal(t, tt.expectedSQL, strings.TrimSpace(response.Query))
		})
	}
}

// TestExtractSQL_NoSQLFound tests when no SQL can be extracted
func TestAnthropicExtractSQL_NoSQLFound(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Mock server returning non-SQL text
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{
					"type": "text",
					"text": "I cannot generate SQL for this request because it's too vague.",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &ai.AnthropicConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}

	provider, err := ai.NewAnthropicProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt: "Vague request",
		Model:  "claude-3-5-sonnet-20241022",
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "could not extract SQL")
}
