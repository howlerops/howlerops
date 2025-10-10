package ai

import (
	"context"
	"time"
)

// Provider represents the available AI providers
type Provider string

const (
	ProviderOpenAI      Provider = "openai"
	ProviderAnthropic   Provider = "anthropic"
	ProviderOllama      Provider = "ollama"
	ProviderHuggingFace Provider = "huggingface"
	ProviderClaudeCode  Provider = "claudecode"
	ProviderCodex       Provider = "codex"
)

// Config holds the configuration for the AI service
type Config struct {
	OpenAI      OpenAIConfig      `json:"openai" yaml:"openai"`
	Anthropic   AnthropicConfig   `json:"anthropic" yaml:"anthropic"`
	Ollama      OllamaConfig      `json:"ollama" yaml:"ollama"`
	HuggingFace HuggingFaceConfig `json:"huggingface" yaml:"huggingface"`
	ClaudeCode  ClaudeCodeConfig  `json:"claudecode" yaml:"claudecode"`
	Codex       CodexConfig       `json:"codex" yaml:"codex"`

	// Global settings
	DefaultProvider Provider      `json:"default_provider" yaml:"default_provider"`
	MaxTokens       int           `json:"max_tokens" yaml:"max_tokens"`
	Temperature     float64       `json:"temperature" yaml:"temperature"`
	RequestTimeout  time.Duration `json:"request_timeout" yaml:"request_timeout"`
	RateLimitPerMin int           `json:"rate_limit_per_min" yaml:"rate_limit_per_min"`
}

// OpenAIConfig holds OpenAI specific configuration
type OpenAIConfig struct {
	APIKey  string   `json:"api_key" yaml:"api_key"`
	BaseURL string   `json:"base_url" yaml:"base_url"`
	Models  []string `json:"models" yaml:"models"`
	OrgID   string   `json:"org_id" yaml:"org_id"`
}

// AnthropicConfig holds Anthropic specific configuration
type AnthropicConfig struct {
	APIKey  string   `json:"api_key" yaml:"api_key"`
	BaseURL string   `json:"base_url" yaml:"base_url"`
	Models  []string `json:"models" yaml:"models"`
	Version string   `json:"version" yaml:"version"`
}

// OllamaConfig holds Ollama specific configuration
type OllamaConfig struct {
	Endpoint        string        `json:"endpoint" yaml:"endpoint"`
	Models          []string      `json:"models" yaml:"models"`
	PullTimeout     time.Duration `json:"pull_timeout" yaml:"pull_timeout"`
	GenerateTimeout time.Duration `json:"generate_timeout" yaml:"generate_timeout"`
	AutoPullModels  bool          `json:"auto_pull_models" yaml:"auto_pull_models"`
}

// HuggingFaceConfig holds Hugging Face specific configuration
type HuggingFaceConfig struct {
	Endpoint         string        `json:"endpoint" yaml:"endpoint"`
	Models           []string      `json:"models" yaml:"models"`
	PullTimeout      time.Duration `json:"pull_timeout" yaml:"pull_timeout"`
	GenerateTimeout  time.Duration `json:"generate_timeout" yaml:"generate_timeout"`
	AutoPullModels   bool          `json:"auto_pull_models" yaml:"auto_pull_models"`
	RecommendedModel string        `json:"recommended_model" yaml:"recommended_model"`
}

// SQLRequest represents a request to generate or fix SQL
type SQLRequest struct {
	Prompt      string            `json:"prompt"`
	Query       string            `json:"query,omitempty"`  // For fixing existing queries
	Error       string            `json:"error,omitempty"`  // Error message to fix
	Schema      string            `json:"schema,omitempty"` // Database schema context
	Provider    Provider          `json:"provider"`
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Context     map[string]string `json:"context,omitempty"` // Additional context
}

// SQLResponse represents the response from the AI service
type SQLResponse struct {
	Query       string            `json:"query"`
	Explanation string            `json:"explanation"`
	Confidence  float64           `json:"confidence"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	Provider    Provider          `json:"provider"`
	Model       string            `json:"model"`
	TokensUsed  int               `json:"tokens_used,omitempty"`
	TimeTaken   time.Duration     `json:"time_taken"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	Provider     Provider      `json:"provider"`
	Status       string        `json:"status"` // "healthy", "unhealthy", "unknown"
	Message      string        `json:"message,omitempty"`
	LastChecked  time.Time     `json:"last_checked"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
}

// ModelInfo represents information about available models
type ModelInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Provider     Provider          `json:"provider"`
	Description  string            `json:"description,omitempty"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Usage represents usage statistics
type Usage struct {
	Provider        Provider      `json:"provider"`
	Model           string        `json:"model"`
	RequestCount    int64         `json:"request_count"`
	TokensUsed      int64         `json:"tokens_used"`
	SuccessRate     float64       `json:"success_rate"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	LastUsed        time.Time     `json:"last_used"`
}

// AIProvider defines the interface that all AI providers must implement
type AIProvider interface {
	// Core functionality
	GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error)
	FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error)

	// Health and status
	HealthCheck(ctx context.Context) (*HealthStatus, error)
	GetModels(ctx context.Context) ([]ModelInfo, error)

	// Provider info
	GetProviderType() Provider
	IsAvailable(ctx context.Context) bool

	// Configuration
	UpdateConfig(config interface{}) error
	ValidateConfig(config interface{}) error
}

// Service defines the main AI service interface
type Service interface {
	// SQL operations
	GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error)
	FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error)

	// Provider management
	GetProviders() []Provider
	GetProviderHealth(ctx context.Context, provider Provider) (*HealthStatus, error)
	GetAllProvidersHealth(ctx context.Context) (map[Provider]*HealthStatus, error)

	// Model management
	GetAvailableModels(ctx context.Context, provider Provider) ([]ModelInfo, error)
	GetAllAvailableModels(ctx context.Context) (map[Provider][]ModelInfo, error)

	// Configuration
	UpdateProviderConfig(provider Provider, config interface{}) error
	GetConfig() *Config

	// Usage and analytics
	GetUsageStats(ctx context.Context, provider Provider) (*Usage, error)
	GetAllUsageStats(ctx context.Context) (map[Provider]*Usage, error)

	// Test and validation
	TestProvider(ctx context.Context, provider Provider, config interface{}) error
	ValidateRequest(req *SQLRequest) error

	// Management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Error types
type ErrorType string

const (
	ErrorTypeInvalidRequest ErrorType = "invalid_request"
	ErrorTypeProviderError  ErrorType = "provider_error"
	ErrorTypeConfigError    ErrorType = "config_error"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeInternalError  ErrorType = "internal_error"
)

// AIError represents an error from the AI service
type AIError struct {
	Type      ErrorType              `json:"type"`
	Message   string                 `json:"message"`
	Provider  Provider               `json:"provider,omitempty"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Retryable bool                   `json:"retryable"`
}

func (e *AIError) Error() string {
	return e.Message
}

// NewAIError creates a new AI error
func NewAIError(errorType ErrorType, message string, provider Provider) *AIError {
	return &AIError{
		Type:      errorType,
		Message:   message,
		Provider:  provider,
		Retryable: errorType == ErrorTypeTimeout || errorType == ErrorTypeRateLimit,
	}
}
