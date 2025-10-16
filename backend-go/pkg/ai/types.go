package ai

import "time"

// Config represents the AI service configuration
type Config struct {
	DefaultProvider string            `yaml:"default_provider"`
	OpenAI          OpenAIConfig      `yaml:"openai"`
	Anthropic       AnthropicConfig   `yaml:"anthropic"`
	Ollama          OllamaConfig      `yaml:"ollama"`
	HuggingFace     HuggingFaceConfig `yaml:"huggingface"`
	ClaudeCode      ClaudeCodeConfig  `yaml:"claudecode"`
	Codex           CodexConfig       `yaml:"codex"`
}

// OpenAIConfig represents OpenAI provider configuration
type OpenAIConfig struct {
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// AnthropicConfig represents Anthropic provider configuration
type AnthropicConfig struct {
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// OllamaConfig represents Ollama provider configuration
type OllamaConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Model    string        `yaml:"model"`
	Timeout  time.Duration `yaml:"timeout"`
}

// HuggingFaceConfig represents HuggingFace provider configuration
type HuggingFaceConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Model    string        `yaml:"model"`
	Timeout  time.Duration `yaml:"timeout"`
}

// ClaudeCodeConfig represents Claude Code provider configuration
type ClaudeCodeConfig struct {
	BinaryPath string        `yaml:"binary_path"`
	Model      string        `yaml:"model"`
	Timeout    time.Duration `yaml:"timeout"`
}

// CodexConfig represents Codex provider configuration
type CodexConfig struct {
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// SQLRequest represents an AI SQL generation or fix request
type SQLRequest struct {
	Prompt      string  `json:"prompt"`
	Query       string  `json:"query,omitempty"`
	Error       string  `json:"error,omitempty"`
	Schema      string  `json:"schema,omitempty"`
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// ChatRequest represents a generic chat interaction request
type ChatRequest struct {
	Prompt      string            `json:"prompt"`
	Context     string            `json:"context,omitempty"`
	System      string            `json:"system,omitempty"`
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
	MaxTokens   int               `json:"maxTokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SQLResponse represents a generated SQL query response
type SQLResponse struct {
	SQL         string  `json:"sql"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
	TokensUsed  int     `json:"tokensUsed"`
}

// ChatResponse represents a chat interaction response
type ChatResponse struct {
	Content    string            `json:"content"`
	Provider   string            `json:"provider"`
	Model      string            `json:"model"`
	TokensUsed int               `json:"tokensUsed"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// OptimizationResponse represents a query optimization response
type OptimizationResponse struct {
	OptimizedSQL string `json:"optimizedSql"`
	Explanation  string `json:"explanation"`
	Impact       string `json:"impact"`
	TokensUsed   int    `json:"tokensUsed"`
}

// ExplanationResponse represents a query explanation response
type ExplanationResponse struct {
	Explanation string `json:"explanation"`
	TokensUsed  int    `json:"tokensUsed"`
}

// ProviderStatus represents the status of an AI provider
type ProviderStatus struct {
	Name         string  `json:"name"`
	Available    bool    `json:"available"`
	RequestCount int64   `json:"requestCount"`
	SuccessRate  float64 `json:"successRate"`
}
