package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func newHTTPClient(handler roundTripFunc) *http.Client {
	return &http.Client{Transport: handler}
}

func TestOpenAIProviderGenerateSQL(t *testing.T) {
	logger := newSilentLogger()
	config := &ai.OpenAIConfig{
		APIKey:  "secret",
		BaseURL: "https://api.test/v1",
		Models:  []string{"gpt-test"},
	}

	providerIface, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)

	resBody, err := json.Marshal(map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"content": `{"query":"SELECT 1","explanation":"demo","confidence":0.92}`,
				},
			},
		},
		"usage": map[string]interface{}{"total_tokens": 42},
	})
	require.NoError(t, err)

	client := newHTTPClient(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer secret", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(resBody))),
			Header:     make(http.Header),
		}, nil
	})

	ai.SetHTTPClient(providerIface, client)

	resp, err := providerIface.GenerateSQL(context.Background(), &ai.SQLRequest{
		Prompt:    "demo",
		Model:     "gpt-test",
		MaxTokens: 128,
	})
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1", resp.Query)
	assert.Equal(t, "demo", resp.Explanation)
	assert.Equal(t, float64(0.92), resp.Confidence)
	assert.Equal(t, 42, resp.TokensUsed)
}

func TestOpenAIProviderDefaultBaseURL(t *testing.T) {
	logger := newSilentLogger()
	config := &ai.OpenAIConfig{
		APIKey: "secret",
	}

	_, err := ai.NewOpenAIProvider(config, logger)
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1", config.BaseURL)
}

func TestAnthropicProviderRequiresAPIKey(t *testing.T) {
	logger := newSilentLogger()
	_, err := ai.NewAnthropicProvider(&ai.AnthropicConfig{}, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestCodexProviderRequiresAPIKey(t *testing.T) {
	_, err := ai.NewCodexProvider(&ai.CodexConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OpenAI API key is required")
}

func TestOllamaProviderDefaults(t *testing.T) {
	logger := newSilentLogger()
	config := &ai.OllamaConfig{}
	provider, err := ai.NewOllamaProvider(config, logger)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "http://localhost:11434", config.Endpoint)
}
