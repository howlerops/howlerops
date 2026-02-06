package ai

import (
	"context"
	"fmt"
)

// ProviderAdapter wraps the new provider implementations to work with the existing AIProvider interface
type ProviderAdapter interface {
	// GenerateSQL generates SQL from natural language
	GenerateSQL(ctx context.Context, prompt string, schema string, options ...GenerateOption) (*SQLResponse, error)

	// FixSQL fixes SQL based on error message
	FixSQL(ctx context.Context, query string, errorMsg string, schema string, options ...GenerateOption) (*SQLResponse, error)

	// Chat handles generic conversational requests
	Chat(ctx context.Context, prompt string, options ...GenerateOption) (*ChatResponse, error)

	// GetHealth returns the health status of the provider
	GetHealth(ctx context.Context) (*HealthStatus, error)

	// ListModels returns available models for the provider
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// GetProviderType returns the type of provider
	GetProviderType() Provider

	// Close cleans up provider resources
	Close() error
}

// GenerateOption is used to configure generation options
type GenerateOption func(*GenerateOptions)

// GenerateOptions contains options for generation
type GenerateOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Stream      bool
	Context     map[string]string
}

// WithModel sets the model to use
func WithModel(model string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Model = model
	}
}

// WithMaxTokens sets the maximum tokens
func WithMaxTokens(maxTokens int) GenerateOption {
	return func(o *GenerateOptions) {
		o.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature
func WithTemperature(temperature float64) GenerateOption {
	return func(o *GenerateOptions) {
		o.Temperature = temperature
	}
}

// WithTopP sets the top-p value
func WithTopP(topP float64) GenerateOption {
	return func(o *GenerateOptions) {
		o.TopP = topP
	}
}

// WithStream enables streaming
func WithStream(stream bool) GenerateOption {
	return func(o *GenerateOptions) {
		o.Stream = stream
	}
}

// WithContext adds context information
func WithContext(context map[string]string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Context = context
	}
}

// ProviderFactory creates a provider based on configuration
type ProviderFactory interface {
	CreateProvider(providerType Provider, config interface{}) (ProviderAdapter, error)
}

// DefaultProviderFactory is the default implementation of ProviderFactory
type DefaultProviderFactory struct{}

// CreateProvider creates a provider based on type and configuration
// Note: This factory only handles the new adapter-based providers (ClaudeCode, Codex)
// The existing providers (OpenAI, Anthropic, Ollama, HuggingFace) are handled directly in the service
func (f *DefaultProviderFactory) CreateProvider(providerType Provider, config interface{}) (ProviderAdapter, error) {
	switch providerType {
	case ProviderClaudeCode:
		cfg, ok := config.(*ClaudeCodeConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for ClaudeCode provider")
		}
		return NewClaudeCodeProvider(cfg)
	case ProviderCodex:
		cfg, ok := config.(*CodexConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for Codex provider")
		}
		return NewCodexProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type in factory: %s", providerType)
	}
}
