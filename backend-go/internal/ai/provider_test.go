package ai_test

import (
	"testing"

	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithModel verifies the WithModel option sets the model field
func TestWithModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{
			name:     "valid model name",
			model:    "gpt-4",
			expected: "gpt-4",
		},
		{
			name:     "claude model",
			model:    "claude-3-opus",
			expected: "claude-3-opus",
		},
		{
			name:     "empty string",
			model:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			ai.WithModel(tt.model)(opts)
			assert.Equal(t, tt.expected, opts.Model)
		})
	}
}

// TestWithMaxTokens verifies the WithMaxTokens option sets the max tokens field
func TestWithMaxTokens(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		expected  int
	}{
		{
			name:      "positive value",
			maxTokens: 1000,
			expected:  1000,
		},
		{
			name:      "zero value",
			maxTokens: 0,
			expected:  0,
		},
		{
			name:      "negative value",
			maxTokens: -100,
			expected:  -100,
		},
		{
			name:      "large value",
			maxTokens: 100000,
			expected:  100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			ai.WithMaxTokens(tt.maxTokens)(opts)
			assert.Equal(t, tt.expected, opts.MaxTokens)
		})
	}
}

// TestWithTemperature verifies the WithTemperature option sets the temperature field
func TestWithTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		expected    float64
	}{
		{
			name:        "zero temperature",
			temperature: 0.0,
			expected:    0.0,
		},
		{
			name:        "low temperature",
			temperature: 0.3,
			expected:    0.3,
		},
		{
			name:        "medium temperature",
			temperature: 0.7,
			expected:    0.7,
		},
		{
			name:        "high temperature",
			temperature: 1.0,
			expected:    1.0,
		},
		{
			name:        "very high temperature",
			temperature: 2.0,
			expected:    2.0,
		},
		{
			name:        "negative temperature",
			temperature: -0.5,
			expected:    -0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			ai.WithTemperature(tt.temperature)(opts)
			assert.Equal(t, tt.expected, opts.Temperature)
		})
	}
}

// TestWithTopP verifies the WithTopP option sets the top-p field
func TestWithTopP(t *testing.T) {
	tests := []struct {
		name     string
		topP     float64
		expected float64
	}{
		{
			name:     "zero value",
			topP:     0.0,
			expected: 0.0,
		},
		{
			name:     "low value",
			topP:     0.1,
			expected: 0.1,
		},
		{
			name:     "medium value",
			topP:     0.5,
			expected: 0.5,
		},
		{
			name:     "high value",
			topP:     0.95,
			expected: 0.95,
		},
		{
			name:     "maximum value",
			topP:     1.0,
			expected: 1.0,
		},
		{
			name:     "negative value",
			topP:     -0.5,
			expected: -0.5,
		},
		{
			name:     "above maximum",
			topP:     1.5,
			expected: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			ai.WithTopP(tt.topP)(opts)
			assert.Equal(t, tt.expected, opts.TopP)
		})
	}
}

// TestWithStream verifies the WithStream option sets the stream field
func TestWithStream(t *testing.T) {
	tests := []struct {
		name     string
		stream   bool
		expected bool
	}{
		{
			name:     "enabled",
			stream:   true,
			expected: true,
		},
		{
			name:     "disabled",
			stream:   false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			ai.WithStream(tt.stream)(opts)
			assert.Equal(t, tt.expected, opts.Stream)
		})
	}
}

// TestWithContext verifies the WithContext option sets the context field
func TestWithContext(t *testing.T) {
	tests := []struct {
		name     string
		context  map[string]string
		expected map[string]string
	}{
		{
			name: "single entry",
			context: map[string]string{
				"key": "value",
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name: "multiple entries",
			context: map[string]string{
				"schema":   "public",
				"database": "postgres",
				"user":     "admin",
			},
			expected: map[string]string{
				"schema":   "public",
				"database": "postgres",
				"user":     "admin",
			},
		},
		{
			name:     "empty map",
			context:  map[string]string{},
			expected: map[string]string{},
		},
		{
			name:     "nil map",
			context:  nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			ai.WithContext(tt.context)(opts)
			assert.Equal(t, tt.expected, opts.Context)
		})
	}
}

// TestOptionsDefaults verifies default values when no options are applied
func TestOptionsDefaults(t *testing.T) {
	opts := &ai.GenerateOptions{}

	assert.Equal(t, "", opts.Model, "Model should default to empty string")
	assert.Equal(t, 0, opts.MaxTokens, "MaxTokens should default to 0")
	assert.Equal(t, 0.0, opts.Temperature, "Temperature should default to 0.0")
	assert.Equal(t, 0.0, opts.TopP, "TopP should default to 0.0")
	assert.False(t, opts.Stream, "Stream should default to false")
	assert.Nil(t, opts.Context, "Context should default to nil")
}

// TestOptionComposition verifies multiple options can be composed
func TestOptionComposition(t *testing.T) {
	tests := []struct {
		name     string
		options  []ai.GenerateOption
		expected ai.GenerateOptions
	}{
		{
			name: "all options",
			options: []ai.GenerateOption{
				ai.WithModel("gpt-4"),
				ai.WithMaxTokens(2000),
				ai.WithTemperature(0.7),
				ai.WithTopP(0.9),
				ai.WithStream(true),
				ai.WithContext(map[string]string{"key": "value"}),
			},
			expected: ai.GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   2000,
				Temperature: 0.7,
				TopP:        0.9,
				Stream:      true,
				Context:     map[string]string{"key": "value"},
			},
		},
		{
			name: "partial options",
			options: []ai.GenerateOption{
				ai.WithModel("claude-3"),
				ai.WithTemperature(0.5),
			},
			expected: ai.GenerateOptions{
				Model:       "claude-3",
				MaxTokens:   0,
				Temperature: 0.5,
				TopP:        0.0,
				Stream:      false,
				Context:     nil,
			},
		},
		{
			name:    "no options",
			options: []ai.GenerateOption{},
			expected: ai.GenerateOptions{
				Model:       "",
				MaxTokens:   0,
				Temperature: 0.0,
				TopP:        0.0,
				Stream:      false,
				Context:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			for _, opt := range tt.options {
				opt(opts)
			}
			assert.Equal(t, tt.expected, *opts)
		})
	}
}

// TestOptionOverride verifies later options override earlier ones
func TestOptionOverride(t *testing.T) {
	tests := []struct {
		name     string
		options  []ai.GenerateOption
		expected ai.GenerateOptions
	}{
		{
			name: "model override",
			options: []ai.GenerateOption{
				ai.WithModel("gpt-3.5"),
				ai.WithModel("gpt-4"),
			},
			expected: ai.GenerateOptions{
				Model: "gpt-4",
			},
		},
		{
			name: "max tokens override",
			options: []ai.GenerateOption{
				ai.WithMaxTokens(1000),
				ai.WithMaxTokens(2000),
			},
			expected: ai.GenerateOptions{
				MaxTokens: 2000,
			},
		},
		{
			name: "temperature override",
			options: []ai.GenerateOption{
				ai.WithTemperature(0.5),
				ai.WithTemperature(0.8),
			},
			expected: ai.GenerateOptions{
				Temperature: 0.8,
			},
		},
		{
			name: "top-p override",
			options: []ai.GenerateOption{
				ai.WithTopP(0.9),
				ai.WithTopP(0.95),
			},
			expected: ai.GenerateOptions{
				TopP: 0.95,
			},
		},
		{
			name: "stream override",
			options: []ai.GenerateOption{
				ai.WithStream(true),
				ai.WithStream(false),
			},
			expected: ai.GenerateOptions{
				Stream: false,
			},
		},
		{
			name: "context override",
			options: []ai.GenerateOption{
				ai.WithContext(map[string]string{"key1": "value1"}),
				ai.WithContext(map[string]string{"key2": "value2"}),
			},
			expected: ai.GenerateOptions{
				Context: map[string]string{"key2": "value2"},
			},
		},
		{
			name: "multiple field overrides",
			options: []ai.GenerateOption{
				ai.WithModel("gpt-3.5"),
				ai.WithMaxTokens(1000),
				ai.WithModel("gpt-4"),
				ai.WithMaxTokens(2000),
				ai.WithTemperature(0.5),
				ai.WithTemperature(0.7),
			},
			expected: ai.GenerateOptions{
				Model:       "gpt-4",
				MaxTokens:   2000,
				Temperature: 0.7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ai.GenerateOptions{}
			for _, opt := range tt.options {
				opt(opts)
			}
			assert.Equal(t, tt.expected, *opts)
		})
	}
}

// TestOptionOrder verifies that option application order matters
func TestOptionOrder(t *testing.T) {
	// First order: low temp, then high temp
	opts1 := &ai.GenerateOptions{}
	ai.WithTemperature(0.1)(opts1)
	ai.WithTemperature(0.9)(opts1)

	// Second order: high temp, then low temp
	opts2 := &ai.GenerateOptions{}
	ai.WithTemperature(0.9)(opts2)
	ai.WithTemperature(0.1)(opts2)

	assert.Equal(t, 0.9, opts1.Temperature, "First order should result in 0.9")
	assert.Equal(t, 0.1, opts2.Temperature, "Second order should result in 0.1")
	assert.NotEqual(t, opts1.Temperature, opts2.Temperature, "Order should matter")
}

// TestMultipleOptionApplications verifies options can be applied multiple times
func TestMultipleOptionApplications(t *testing.T) {
	opts := &ai.GenerateOptions{}

	// Apply model option
	ai.WithModel("gpt-3.5")(opts)
	assert.Equal(t, "gpt-3.5", opts.Model)

	// Apply another option
	ai.WithMaxTokens(1000)(opts)
	assert.Equal(t, "gpt-3.5", opts.Model, "Previous option should persist")
	assert.Equal(t, 1000, opts.MaxTokens)

	// Override first option
	ai.WithModel("gpt-4")(opts)
	assert.Equal(t, "gpt-4", opts.Model, "Model should be overridden")
	assert.Equal(t, 1000, opts.MaxTokens, "Other options should persist")
}

// TestContextMapMutation verifies that context map can be mutated
func TestContextMapMutation(t *testing.T) {
	ctx := map[string]string{
		"key1": "value1",
	}

	opts := &ai.GenerateOptions{}
	ai.WithContext(ctx)(opts)

	assert.Equal(t, "value1", opts.Context["key1"])

	// Mutate the original map
	ctx["key2"] = "value2"

	// The option should reflect the mutation (since maps are references)
	assert.Equal(t, "value2", opts.Context["key2"])
}

// TestNilOptionsStruct verifies options work with initialized struct
func TestNilOptionsStruct(t *testing.T) {
	// This test verifies that we can't call options on nil pointer
	// (which would panic), but we always initialize the struct first
	opts := &ai.GenerateOptions{}
	require.NotNil(t, opts)

	ai.WithModel("test")(opts)
	assert.Equal(t, "test", opts.Model)
}

// TestDefaultProviderFactory_CreateProvider_ClaudeCode verifies ClaudeCode provider creation
func TestDefaultProviderFactory_CreateProvider_ClaudeCode(t *testing.T) {
	factory := &ai.DefaultProviderFactory{}

	t.Run("valid config", func(t *testing.T) {
		config := &ai.ClaudeCodeConfig{
			ClaudePath: "/nonexistent/path", // Will fail but validates factory logic
		}
		provider, err := factory.CreateProvider(ai.ProviderClaudeCode, config)

		// The factory should attempt creation, but the provider itself may fail
		// due to missing Claude Code installation. Both outcomes are valid.
		if err != nil {
			assert.Nil(t, provider)
			assert.Error(t, err)
		} else {
			assert.NotNil(t, provider)
			defer func() { _ = provider.Close() }() // Best-effort close in test
		}
	})

	t.Run("invalid config type", func(t *testing.T) {
		config := &ai.CodexConfig{} // Wrong config type
		provider, err := factory.CreateProvider(ai.ProviderClaudeCode, config)

		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config type")
	})

	t.Run("nil config", func(t *testing.T) {
		provider, err := factory.CreateProvider(ai.ProviderClaudeCode, nil)

		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config type")
	})
}

// TestDefaultProviderFactory_CreateProvider_Codex verifies Codex provider creation
func TestDefaultProviderFactory_CreateProvider_Codex(t *testing.T) {
	factory := &ai.DefaultProviderFactory{}

	t.Run("valid config", func(t *testing.T) {
		config := &ai.CodexConfig{
			APIKey: "test-api-key", // Minimal valid config
		}
		provider, err := factory.CreateProvider(ai.ProviderCodex, config)

		// The factory should attempt creation, but the provider itself may fail
		// due to invalid API key. Both outcomes are valid.
		if err != nil {
			assert.Nil(t, provider)
			assert.Error(t, err)
		} else {
			assert.NotNil(t, provider)
			defer func() { _ = provider.Close() }() // Best-effort close in test
		}
	})

	t.Run("invalid config type", func(t *testing.T) {
		config := &ai.ClaudeCodeConfig{} // Wrong config type
		provider, err := factory.CreateProvider(ai.ProviderCodex, config)

		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config type")
	})

	t.Run("nil config", func(t *testing.T) {
		provider, err := factory.CreateProvider(ai.ProviderCodex, nil)

		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config type")
	})
}

// TestDefaultProviderFactory_CreateProvider_UnsupportedProvider verifies error for unsupported providers
func TestDefaultProviderFactory_CreateProvider_UnsupportedProvider(t *testing.T) {
	factory := &ai.DefaultProviderFactory{}

	tests := []struct {
		name         string
		providerType ai.Provider
	}{
		{
			name:         "OpenAI",
			providerType: ai.ProviderOpenAI,
		},
		{
			name:         "Anthropic",
			providerType: ai.ProviderAnthropic,
		},
		{
			name:         "Ollama",
			providerType: ai.ProviderOllama,
		},
		{
			name:         "HuggingFace",
			providerType: ai.ProviderHuggingFace,
		},
		{
			name:         "Unknown",
			providerType: ai.Provider("unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.providerType, nil)

			assert.Nil(t, provider)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported provider type")
		})
	}
}

// TestDefaultProviderFactory_CreateProvider_AllProviderTypes verifies all provider types
func TestDefaultProviderFactory_CreateProvider_AllProviderTypes(t *testing.T) {
	factory := &ai.DefaultProviderFactory{}

	tests := []struct {
		name              string
		providerType      ai.Provider
		config            interface{}
		shouldError       bool
		errorMsg          string
		allowEitherResult bool // Some tests can succeed or fail depending on environment
	}{
		{
			name:              "ClaudeCode with valid config",
			providerType:      ai.ProviderClaudeCode,
			config:            &ai.ClaudeCodeConfig{ClaudePath: "/nonexistent/path"},
			allowEitherResult: true, // Can fail if Claude Code not installed
		},
		{
			name:              "Codex with valid config",
			providerType:      ai.ProviderCodex,
			config:            &ai.CodexConfig{APIKey: "test-api-key"},
			allowEitherResult: true, // Can fail if API key invalid
		},
		{
			name:         "OpenAI not supported",
			providerType: ai.ProviderOpenAI,
			config:       &ai.OpenAIConfig{},
			shouldError:  true,
			errorMsg:     "unsupported provider type",
		},
		{
			name:         "Invalid provider",
			providerType: ai.Provider("invalid"),
			config:       nil,
			shouldError:  true,
			errorMsg:     "unsupported provider type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.providerType, tt.config)

			if tt.allowEitherResult {
				// Either success or failure is acceptable
				if err != nil {
					assert.Nil(t, provider)
				} else {
					assert.NotNil(t, provider)
					defer func() { _ = provider.Close() }() // Best-effort close in test
				}
			} else if tt.shouldError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				if provider != nil {
					defer func() { _ = provider.Close() }() // Best-effort close in test
				}
			}
		})
	}
}

// TestGenerateOptionFunction verifies GenerateOption is a function type
func TestGenerateOptionFunction(t *testing.T) {
	// Verify that GenerateOption is indeed a function type
	opt := ai.WithModel("test")

	assert.NotNil(t, opt)
	assert.IsType(t, ai.GenerateOption(nil), opt)

	// Verify it can be called
	opts := &ai.GenerateOptions{}
	opt(opts)
	assert.Equal(t, "test", opts.Model)
}

// TestGenerateOptionsStructFields verifies all fields in GenerateOptions struct
func TestGenerateOptionsStructFields(t *testing.T) {
	opts := ai.GenerateOptions{
		Model:       "test-model",
		MaxTokens:   1500,
		Temperature: 0.8,
		TopP:        0.92,
		Stream:      true,
		Context: map[string]string{
			"db": "postgres",
		},
	}

	assert.Equal(t, "test-model", opts.Model)
	assert.Equal(t, 1500, opts.MaxTokens)
	assert.Equal(t, 0.8, opts.Temperature)
	assert.Equal(t, 0.92, opts.TopP)
	assert.True(t, opts.Stream)
	assert.Equal(t, "postgres", opts.Context["db"])
}

// TestOptionChaining verifies options can be chained in a single expression
func TestOptionChaining(t *testing.T) {
	// Simulate how options would be used in practice
	applyOptions := func(opts *ai.GenerateOptions, options ...ai.GenerateOption) {
		for _, opt := range options {
			opt(opts)
		}
	}

	opts := &ai.GenerateOptions{}
	applyOptions(opts,
		ai.WithModel("gpt-4"),
		ai.WithMaxTokens(2000),
		ai.WithTemperature(0.7),
		ai.WithTopP(0.9),
		ai.WithStream(true),
		ai.WithContext(map[string]string{"schema": "public"}),
	)

	assert.Equal(t, "gpt-4", opts.Model)
	assert.Equal(t, 2000, opts.MaxTokens)
	assert.Equal(t, 0.7, opts.Temperature)
	assert.Equal(t, 0.9, opts.TopP)
	assert.True(t, opts.Stream)
	assert.Equal(t, "public", opts.Context["schema"])
}

// TestEmptyContextMap verifies empty context map behavior
func TestEmptyContextMap(t *testing.T) {
	opts := &ai.GenerateOptions{}
	emptyMap := make(map[string]string)
	ai.WithContext(emptyMap)(opts)

	assert.NotNil(t, opts.Context)
	assert.Empty(t, opts.Context)
	assert.Len(t, opts.Context, 0)
}

// TestContextMapWithSpecialCharacters verifies context map with special characters
func TestContextMapWithSpecialCharacters(t *testing.T) {
	ctx := map[string]string{
		"key-with-dash":       "value",
		"key_with_underscore": "value",
		"key.with.dots":       "value",
		"key with spaces":     "value with spaces",
		"key=with=equals":     "value=with=equals",
	}

	opts := &ai.GenerateOptions{}
	ai.WithContext(ctx)(opts)

	assert.Equal(t, ctx, opts.Context)
	assert.Equal(t, "value", opts.Context["key-with-dash"])
	assert.Equal(t, "value with spaces", opts.Context["key with spaces"])
}
