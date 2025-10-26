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

// Helper functions for creating test data

func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests
	return logger
}

func createTestConfig(apiKey, baseURL string, models []string) *ai.OpenAIConfig {
	return &ai.OpenAIConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Models:  models,
	}
}

// Mock response structures matching OpenAI API

type mockOpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type mockOpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

type mockOpenAIModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// Constructor Tests

func TestOpenAI_NewOpenAIProvider_ValidConfig(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})

	provider, err := ai.NewOpenAIProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, ai.ProviderOpenAI, provider.GetProviderType())
}

func TestOpenAI_NewOpenAIProvider_NilConfig(t *testing.T) {
	logger := createTestLogger()

	provider, err := ai.NewOpenAIProvider(nil, logger)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestOpenAI_NewOpenAIProvider_EmptyAPIKey(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("", "https://api.openai.com/v1", []string{"gpt-4o-mini"})

	provider, err := ai.NewOpenAIProvider(config, logger)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestOpenAI_NewOpenAIProvider_DefaultBaseURL(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "", []string{"gpt-4o-mini"})

	provider, err := ai.NewOpenAIProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	// BaseURL should be set to default - verify via health check endpoint
}

func TestOpenAI_NewOpenAIProvider_DefaultModels(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", nil)

	provider, err := ai.NewOpenAIProvider(config, logger)

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Models should be set to defaults
}

// GetProviderType Tests

func TestOpenAI_GetProviderType(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	providerType := provider.GetProviderType()

	assert.Equal(t, ai.ProviderOpenAI, providerType)
}

// GenerateSQL Tests

func TestOpenAI_GenerateSQL_Success(t *testing.T) {
	logger := createTestLogger()

	// Create mock server
	server := testServer()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		// Verify headers
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify request body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "gpt-4o-mini", reqBody["model"])
		assert.Equal(t, false, reqBody["stream"])
		assert.NotNil(t, reqBody["messages"])

		// Send mock response
		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Role = "assistant"
		resp.Choices[0].Message.Content = `{"query":"SELECT * FROM users","explanation":"This query selects all users","confidence":0.95,"suggestions":["Consider adding a WHERE clause"]}`
		resp.Choices[0].FinishReason = "stop"
		resp.Usage.TotalTokens = 100

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "SELECT * FROM users", response.Query)
	assert.Equal(t, "This query selects all users", response.Explanation)
	assert.Equal(t, 0.95, response.Confidence)
	assert.Equal(t, ai.ProviderOpenAI, response.Provider)
	assert.Equal(t, "gpt-4o-mini", response.Model)
	assert.Equal(t, 100, response.TokensUsed)
}

func TestOpenAI_GenerateSQL_WithSchema(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body and verify schema is included
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		userMessage := messages[len(messages)-1].(map[string]interface{})
		content := userMessage["content"].(string)
		assert.Contains(t, content, "Database Schema:")

		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT name FROM users","explanation":"Query with schema","confidence":0.90}`
		resp.Usage.TotalTokens = 120

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get user names",
		Schema:      "CREATE TABLE users (id INT, name VARCHAR(100))",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT name FROM users", response.Query)
}

func TestOpenAI_GenerateSQL_NonJSONResponse(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "```sql\nSELECT * FROM users\n```"
		resp.Usage.TotalTokens = 50

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM users", response.Query)
	assert.Equal(t, 0.8, response.Confidence) // Lower confidence for non-JSON
}

func TestOpenAI_GenerateSQL_APIError(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		errResp := mockOpenAIErrorResponse{}
		errResp.Error.Message = "Invalid API key"
		errResp.Error.Type = "invalid_request_error"
		errResp.Error.Code = "invalid_api_key"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestOpenAI_GenerateSQL_RateLimitError(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		errResp := mockOpenAIErrorResponse{}
		errResp.Error.Message = "Rate limit exceeded"
		errResp.Error.Type = "rate_limit_error"
		errResp.Error.Code = "rate_limit_exceeded"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Rate limit exceeded")
}

func TestOpenAI_GenerateSQL_MalformedResponse(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": "json"`)) // Malformed JSON
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestOpenAI_GenerateSQL_EmptyChoices(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{}, // Empty choices
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no response choices returned")
}

// FixSQL Tests

func TestOpenAI_FixSQL_Success(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request includes error context
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		userMessage := messages[len(messages)-1].(map[string]interface{})
		content := userMessage["content"].(string)
		assert.Contains(t, content, "Error Message:")
		assert.Contains(t, content, "Original Query:")

		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT * FROM users WHERE id = 1","explanation":"Fixed syntax error","confidence":0.92}`
		resp.Usage.TotalTokens = 150

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query:       "SELECT * FROM users WHERE id =",
		Error:       "Syntax error near '='",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM users WHERE id = 1", response.Query)
	assert.Equal(t, "Fixed syntax error", response.Explanation)
	assert.Equal(t, 0.92, response.Confidence)
}

func TestOpenAI_FixSQL_WithSchema(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		userMessage := messages[len(messages)-1].(map[string]interface{})
		content := userMessage["content"].(string)
		assert.Contains(t, content, "Database Schema:")

		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT name FROM users","explanation":"Fixed with schema","confidence":0.95}`
		resp.Usage.TotalTokens = 130

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Query:       "SELECT username FROM users",
		Error:       "Column 'username' not found",
		Schema:      "CREATE TABLE users (id INT, name VARCHAR(100))",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.FixSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT name FROM users", response.Query)
}

// Chat Tests

func TestOpenAI_Chat_Success(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		assert.GreaterOrEqual(t, len(messages), 2) // At least system + user

		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "This is a helpful response"
		resp.Choices[0].FinishReason = "stop"
		resp.Usage.TotalTokens = 75

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:      "What is SQL?",
		Model:       "gpt-4o-mini",
		MaxTokens:   500,
		Temperature: 0.7,
	}

	response, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "This is a helpful response", response.Content)
	assert.Equal(t, ai.ProviderOpenAI, response.Provider)
	assert.Equal(t, "gpt-4o-mini", response.Model)
	assert.Equal(t, 75, response.TokensUsed)
	assert.Equal(t, "stop", response.Metadata["finish_reason"])
}

func TestOpenAI_Chat_WithCustomSystem(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		systemMsg := messages[0].(map[string]interface{})
		assert.Equal(t, "system", systemMsg["role"])
		assert.Equal(t, "You are a database expert", systemMsg["content"])

		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "Expert response"
		resp.Usage.TotalTokens = 60

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:      "Explain indexes",
		System:      "You are a database expert",
		Model:       "gpt-4o-mini",
		MaxTokens:   500,
		Temperature: 0.7,
	}

	response, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Expert response", response.Content)
}

func TestOpenAI_Chat_WithContext(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		// Should have system, context, and user messages
		assert.Equal(t, 3, len(messages))

		contextMsg := messages[1].(map[string]interface{})
		assert.Equal(t, "system", contextMsg["role"])
		assert.Contains(t, contextMsg["content"], "Additional context:")

		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
		}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "Contextual response"
		resp.Usage.TotalTokens = 80

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:      "What should I do?",
		Context:     "User is working on optimizing queries",
		Model:       "gpt-4o-mini",
		MaxTokens:   500,
		Temperature: 0.7,
	}

	response, err := provider.Chat(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Contextual response", response.Content)
}

func TestOpenAI_Chat_NilRequest(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	response, err := provider.Chat(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

func TestOpenAI_Chat_EmptyChoices(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.ChatRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   500,
		Temperature: 0.7,
	}

	response, err := provider.Chat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no response choices returned")
}

// HealthCheck Tests

func TestOpenAI_HealthCheck_Healthy(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		resp := mockOpenAIModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				{ID: "gpt-4o-mini", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderOpenAI, health.Provider)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "Service is operational", health.Message)
	assert.True(t, health.ResponseTime > 0)
}

func TestOpenAI_HealthCheck_Unhealthy_BadStatusCode(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderOpenAI, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "HTTP 500")
}

func TestOpenAI_HealthCheck_Unhealthy_NetworkError(t *testing.T) {
	logger := createTestLogger()

	// Use invalid URL to trigger network error
	config := createTestConfig("test-api-key", "http://invalid-url-that-does-not-exist:9999", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	health, err := provider.HealthCheck(ctx)

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, ai.ProviderOpenAI, health.Provider)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "Request failed")
}

func TestOpenAI_HealthCheck_WithOrgID(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify organization header
		assert.Equal(t, "org-test-123", r.Header.Get("OpenAI-Organization"))

		resp := mockOpenAIModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	config.OrgID = "org-test-123"
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	health, err := provider.HealthCheck(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)
}

// IsAvailable Tests

func TestOpenAI_IsAvailable_True(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIModelsResponse{Object: "list"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	available := provider.IsAvailable(context.Background())

	assert.True(t, available)
}

func TestOpenAI_IsAvailable_False(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	available := provider.IsAvailable(context.Background())

	assert.False(t, available)
}

// GetModels Tests

func TestOpenAI_GetModels_Success(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		resp := mockOpenAIModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				{ID: "gpt-4o-mini", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
				{ID: "gpt-4o", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
				{ID: "text-embedding-ada-002", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	require.NoError(t, err)
	require.NotNil(t, models)
	// Should filter for only gpt models
	assert.Len(t, models, 2)
	assert.Equal(t, "gpt-4o-mini", models[0].ID)
	assert.Equal(t, ai.ProviderOpenAI, models[0].Provider)
	assert.Contains(t, models[0].Capabilities, "text-to-sql")
}

func TestOpenAI_GetModels_WithOrgID(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "org-test-123", r.Header.Get("OpenAI-Organization"))

		resp := mockOpenAIModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				{ID: "gpt-4o", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	config.OrgID = "org-test-123"
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	require.NoError(t, err)
	assert.Len(t, models, 1)
}

func TestOpenAI_GetModels_HTTPError(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	assert.Error(t, err)
	assert.Nil(t, models)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestOpenAI_GetModels_MalformedJSON(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": json}`))
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	models, err := provider.GetModels(context.Background())

	assert.Error(t, err)
	assert.Nil(t, models)
}

// UpdateConfig Tests

func TestOpenAI_UpdateConfig_Success(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	newConfig := createTestConfig("new-api-key", "https://api.openai.com/v1", []string{"gpt-4o"})

	err = provider.UpdateConfig(newConfig)

	assert.NoError(t, err)
}

func TestOpenAI_UpdateConfig_InvalidType(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	err = provider.UpdateConfig("invalid-config")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

func TestOpenAI_UpdateConfig_InvalidConfig(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := createTestConfig("", "https://api.openai.com/v1", []string{"gpt-4o"})

	err = provider.UpdateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

// ValidateConfig Tests

func TestOpenAI_ValidateConfig_Valid(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	testConfig := createTestConfig("valid-key", "https://api.openai.com/v1", []string{"gpt-4o"})

	err = provider.ValidateConfig(testConfig)

	assert.NoError(t, err)
}

func TestOpenAI_ValidateConfig_InvalidType(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	err = provider.ValidateConfig("invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

func TestOpenAI_ValidateConfig_EmptyAPIKey(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := createTestConfig("", "https://api.openai.com/v1", []string{"gpt-4o"})

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestOpenAI_ValidateConfig_EmptyBaseURL(t *testing.T) {
	logger := createTestLogger()
	config := createTestConfig("test-api-key", "https://api.openai.com/v1", []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	invalidConfig := createTestConfig("valid-key", "", []string{"gpt-4o"})

	err = provider.ValidateConfig(invalidConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base URL is required")
}

// HTTP Header Tests

func TestOpenAI_HTTPHeaders_Authorization(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader)
		assert.True(t, strings.HasPrefix(authHeader, "Bearer "))
		assert.Equal(t, "Bearer test-api-key-123", authHeader)

		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT 1","explanation":"test","confidence":0.9}`
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key-123", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	assert.NoError(t, err)
}

func TestOpenAI_HTTPHeaders_ContentType(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType)

		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT 1","explanation":"test","confidence":0.9}`
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	assert.NoError(t, err)
}

func TestOpenAI_HTTPHeaders_OrganizationID(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgHeader := r.Header.Get("OpenAI-Organization")
		assert.Equal(t, "org-abc123", orgHeader)

		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT 1","explanation":"test","confidence":0.9}`
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	config.OrgID = "org-abc123"
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	assert.NoError(t, err)
}

// Request Body Verification Tests

func TestOpenAI_RequestBody_StreamDisabled(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, false, reqBody["stream"])

		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT 1","explanation":"test","confidence":0.9}`
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	assert.NoError(t, err)
}

func TestOpenAI_RequestBody_MessagesFormat(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]interface{})
		assert.GreaterOrEqual(t, len(messages), 2)

		// Check system message
		systemMsg := messages[0].(map[string]interface{})
		assert.Equal(t, "system", systemMsg["role"])
		assert.NotEmpty(t, systemMsg["content"])

		// Check user message
		userMsg := messages[1].(map[string]interface{})
		assert.Equal(t, "user", userMsg["role"])
		assert.Contains(t, userMsg["content"], "Test prompt")

		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT 1","explanation":"test","confidence":0.9}`
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test prompt",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	assert.NoError(t, err)
}

func TestOpenAI_RequestBody_ModelAndParameters(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "gpt-4o", reqBody["model"])
		assert.Equal(t, float64(2000), reqBody["max_tokens"])
		assert.Equal(t, 0.5, reqBody["temperature"])

		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = `{"query":"SELECT 1","explanation":"test","confidence":0.9}`
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o",
		MaxTokens:   2000,
		Temperature: 0.5,
	}

	_, err = provider.GenerateSQL(context.Background(), req)
	assert.NoError(t, err)
}

// Context and Timeout Tests

func TestOpenAI_Context_Cancellation(t *testing.T) {
	logger := createTestLogger()

	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := mockOpenAIChatResponse{ID: "test"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// Error Response Tests

func TestOpenAI_ErrorResponse_ServerError(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		errResp := mockOpenAIErrorResponse{}
		errResp.Error.Message = "Internal server error"
		errResp.Error.Type = "server_error"
		errResp.Error.Code = "internal_error"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Internal server error")
}

func TestOpenAI_ErrorResponse_InvalidModel(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errResp := mockOpenAIErrorResponse{}
		errResp.Error.Message = "The model `invalid-model` does not exist"
		errResp.Error.Type = "invalid_request_error"
		errResp.Error.Code = "model_not_found"
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"invalid-model"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "invalid-model",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestOpenAI_ErrorResponse_PlainText(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service temporarily unavailable"))
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Test",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	_, err = provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 503")
}

// SQL Extraction Tests

func TestOpenAI_SQLExtraction_CodeBlock(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "Here's the query:\n```sql\nSELECT id, name FROM users WHERE active = 1\n```"
		resp.Usage.TotalTokens = 50
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get active users",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT id, name FROM users WHERE active = 1", response.Query)
}

func TestOpenAI_SQLExtraction_GenericCodeBlock(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "```\nSELECT * FROM orders\n```"
		resp.Usage.TotalTokens = 40
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Get orders",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM orders", response.Query)
}

func TestOpenAI_SQLExtraction_NoValidSQL(t *testing.T) {
	logger := createTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockOpenAIChatResponse{ID: "test"}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)
		resp.Choices[0].Message.Content = "I cannot help with that request."
		resp.Usage.TotalTokens = 20
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	req := &ai.SQLRequest{
		Prompt:      "Invalid request",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	response, err := provider.GenerateSQL(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "could not extract SQL")
}

// Integration-style Tests

func TestOpenAI_FullWorkflow_GenerateAndFix(t *testing.T) {
	logger := createTestLogger()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := mockOpenAIChatResponse{ID: fmt.Sprintf("test-%d", callCount)}
		resp.Choices = make([]struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}, 1)

		if callCount == 1 {
			// First call: Generate SQL (with error)
			resp.Choices[0].Message.Content = `{"query":"SELECT * FROM user","explanation":"Get all users","confidence":0.85}`
		} else {
			// Second call: Fix SQL
			resp.Choices[0].Message.Content = `{"query":"SELECT * FROM users","explanation":"Fixed table name","confidence":0.95}`
		}
		resp.Usage.TotalTokens = 100
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestConfig("test-api-key", server.URL, []string{"gpt-4o-mini"})
	provider, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	// Generate SQL
	genReq := &ai.SQLRequest{
		Prompt:      "Get all users",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	genResp, err := provider.GenerateSQL(context.Background(), genReq)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM user", genResp.Query)

	// Fix SQL
	fixReq := &ai.SQLRequest{
		Query:       genResp.Query,
		Error:       "Table 'user' doesn't exist",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	fixResp, err := provider.FixSQL(context.Background(), fixReq)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM users", fixResp.Query)
	assert.Equal(t, 2, callCount)
}
