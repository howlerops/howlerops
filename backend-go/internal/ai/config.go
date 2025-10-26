package ai

import (
	"fmt"
	"os"
	"time"
)

// DefaultConfig returns the baseline AI configuration without reading environment variables.
func DefaultConfig() *Config {
	config := &Config{
		DefaultProvider: ProviderOpenAI,
		MaxTokens:       2048,
		Temperature:     0.1,
		RequestTimeout:  60 * time.Second,
		RateLimitPerMin: 60,
	}

	config.OpenAI = OpenAIConfig{
		APIKey:  "",
		BaseURL: "https://api.openai.com/v1",
		Models: []string{
			"gpt-4o-mini",
			"gpt-4o",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
		},
		OrgID: "",
	}

	config.Anthropic = AnthropicConfig{
		APIKey:  "",
		BaseURL: "https://api.anthropic.com",
		Version: "2023-06-01",
		Models: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
		},
	}

	config.Ollama = OllamaConfig{
		Endpoint:        "",
		PullTimeout:     10 * time.Minute,
		GenerateTimeout: 2 * time.Minute,
		AutoPullModels:  true,
		Models: []string{
			"sqlcoder:7b",
			"codellama:7b",
			"llama3.1:8b",
			"mistral:7b",
		},
	}

	config.HuggingFace = HuggingFaceConfig{
		Endpoint:         "",
		PullTimeout:      10 * time.Minute,
		GenerateTimeout:  2 * time.Minute,
		AutoPullModels:   true,
		RecommendedModel: "sqlcoder:7b",
		Models: []string{
			"sqlcoder:7b",
			"codellama:7b",
			"llama3.1:8b",
			"mistral:7b",
		},
	}

	config.ClaudeCode = ClaudeCodeConfig{
		ClaudePath:  "",
		Model:       "opus",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	config.Codex = CodexConfig{
		APIKey:       "",
		Organization: "",
		BaseURL:      "https://api.openai.com/v1",
		Model:        "code-davinci-002",
		MaxTokens:    2048,
		Temperature:  0.0,
	}

	return config
}

// LoadConfig loads AI configuration from environment variables and defaults
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	if value := os.Getenv("OPENAI_API_KEY"); value != "" {
		config.OpenAI.APIKey = value
	}
	config.OpenAI.BaseURL = getEnvWithDefault("OPENAI_BASE_URL", config.OpenAI.BaseURL)
	if value := os.Getenv("OPENAI_ORG_ID"); value != "" {
		config.OpenAI.OrgID = value
	}

	if value := os.Getenv("ANTHROPIC_API_KEY"); value != "" {
		config.Anthropic.APIKey = value
	}
	config.Anthropic.BaseURL = getEnvWithDefault("ANTHROPIC_BASE_URL", config.Anthropic.BaseURL)
	config.Anthropic.Version = getEnvWithDefault("ANTHROPIC_VERSION", config.Anthropic.Version)

	config.Ollama.Endpoint = getEnvWithDefault("OLLAMA_ENDPOINT", config.Ollama.Endpoint)
	config.Ollama.AutoPullModels = getEnvBool("OLLAMA_AUTO_PULL", config.Ollama.AutoPullModels)

	config.HuggingFace.Endpoint = getEnvWithDefault("HUGGINGFACE_ENDPOINT", config.HuggingFace.Endpoint)
	config.HuggingFace.AutoPullModels = getEnvBool("HUGGINGFACE_AUTO_PULL", config.HuggingFace.AutoPullModels)

	if value := os.Getenv("CLAUDE_CLI_PATH"); value != "" {
		config.ClaudeCode.ClaudePath = value
	}
	config.ClaudeCode.Model = getEnvWithDefault("CLAUDE_CODE_MODEL", config.ClaudeCode.Model)
	config.ClaudeCode.MaxTokens = getEnvIntWithDefault("CLAUDE_CODE_MAX_TOKENS", config.ClaudeCode.MaxTokens)
	config.ClaudeCode.Temperature = getEnvFloatWithDefault("CLAUDE_CODE_TEMPERATURE", config.ClaudeCode.Temperature)

	if value := os.Getenv("CODEX_API_KEY"); value != "" {
		config.Codex.APIKey = value
	}
	if value := os.Getenv("CODEX_ORGANIZATION"); value != "" {
		config.Codex.Organization = value
	}
	config.Codex.BaseURL = getEnvWithDefault("CODEX_BASE_URL", config.Codex.BaseURL)
	config.Codex.Model = getEnvWithDefault("CODEX_MODEL", config.Codex.Model)
	config.Codex.MaxTokens = getEnvIntWithDefault("CODEX_MAX_TOKENS", config.Codex.MaxTokens)

	// Override default provider if specified
	if provider := os.Getenv("AI_DEFAULT_PROVIDER"); provider != "" {
		switch Provider(provider) {
		case ProviderOpenAI, ProviderAnthropic, ProviderOllama, ProviderHuggingFace:
			config.DefaultProvider = Provider(provider)
		default:
			return nil, fmt.Errorf("invalid AI_DEFAULT_PROVIDER: %s", provider)
		}
	}

	// Override defaults from environment
	if maxTokens := os.Getenv("AI_MAX_TOKENS"); maxTokens != "" {
		if parsed, err := parseIntEnv("AI_MAX_TOKENS", maxTokens); err == nil {
			config.MaxTokens = parsed
		}
	}

	if temperature := os.Getenv("AI_TEMPERATURE"); temperature != "" {
		if parsed, err := parseFloatEnv("AI_TEMPERATURE", temperature); err == nil {
			config.Temperature = parsed
		}
	}

	if timeout := os.Getenv("AI_REQUEST_TIMEOUT"); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			config.RequestTimeout = parsed
		}
	}

	if rateLimit := os.Getenv("AI_RATE_LIMIT_PER_MIN"); rateLimit != "" {
		if parsed, err := parseIntEnv("AI_RATE_LIMIT_PER_MIN", rateLimit); err == nil {
			config.RateLimitPerMin = parsed
		}
	}
	return config, nil
}

// ValidateConfig validates the AI configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	if config.Temperature < 0 || config.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1")
	}

	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}

	if config.RateLimitPerMin <= 0 {
		return fmt.Errorf("rate_limit_per_min must be positive")
	}

	// Check if at least one provider is configured
	hasProvider := false

	if config.OpenAI.APIKey != "" {
		hasProvider = true
		if config.OpenAI.BaseURL == "" {
			return fmt.Errorf("openai base_url is required when api_key is provided")
		}
	}

	if config.Anthropic.APIKey != "" {
		hasProvider = true
		if config.Anthropic.BaseURL == "" {
			return fmt.Errorf("anthropic base_url is required when api_key is provided")
		}
		if config.Anthropic.Version == "" {
			return fmt.Errorf("anthropic version is required when api_key is provided")
		}
	}

	if config.Ollama.Endpoint != "" {
		hasProvider = true
		if config.Ollama.PullTimeout <= 0 {
			return fmt.Errorf("ollama pull_timeout must be positive")
		}
		if config.Ollama.GenerateTimeout <= 0 {
			return fmt.Errorf("ollama generate_timeout must be positive")
		}
	}

	if config.HuggingFace.Endpoint != "" {
		hasProvider = true
		if config.HuggingFace.PullTimeout <= 0 {
			return fmt.Errorf("huggingface pull_timeout must be positive")
		}
		if config.HuggingFace.GenerateTimeout <= 0 {
			return fmt.Errorf("huggingface generate_timeout must be positive")
		}
	}

	if config.ClaudeCode.ClaudePath != "" {
		hasProvider = true
		if config.ClaudeCode.Model == "" {
			return fmt.Errorf("claudecode model is required when claude path is configured")
		}
	}

	if config.Codex.APIKey != "" {
		hasProvider = true
		if config.Codex.Model == "" {
			return fmt.Errorf("codex model is required when api_key is provided")
		}
		if config.Codex.BaseURL == "" {
			return fmt.Errorf("codex base_url is required when api_key is provided")
		}
	}

	// AI providers are optional - users can configure them later through the app
	// No error if no providers are configured at startup

	return nil
}

// Helper functions

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := parseIntEnv(key, value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloatWithDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := parseFloatEnv(key, value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

func parseIntEnv(key, value string) (int, error) {
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return 0, fmt.Errorf("invalid %s: %s", key, value)
	}
	return result, nil
}

func parseFloatEnv(key, value string) (float64, error) {
	var result float64
	if _, err := fmt.Sscanf(value, "%f", &result); err != nil {
		return 0, fmt.Errorf("invalid %s: %s", key, value)
	}
	return result, nil
}

// ConfigFromUserSettings creates a config from user settings (for testing)
func ConfigFromUserSettings(settings map[string]interface{}) (*Config, error) {
	config := &Config{
		DefaultProvider: ProviderOpenAI,
		MaxTokens:       2048,
		Temperature:     0.1,
		RequestTimeout:  60 * time.Second,
		RateLimitPerMin: 60,
	}

	if provider, ok := settings["provider"].(string); ok {
		config.DefaultProvider = Provider(provider)
	}

	if maxTokens, ok := settings["maxTokens"].(float64); ok {
		config.MaxTokens = int(maxTokens)
	}

	if temperature, ok := settings["temperature"].(float64); ok {
		config.Temperature = temperature
	}

	// OpenAI settings
	if openaiKey, ok := settings["openaiApiKey"].(string); ok {
		config.OpenAI.APIKey = openaiKey
		config.OpenAI.BaseURL = "https://api.openai.com/v1"
		config.OpenAI.Models = []string{
			"gpt-4o-mini", "gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo",
		}
	}

	// Anthropic settings
	if anthropicKey, ok := settings["anthropicApiKey"].(string); ok {
		config.Anthropic.APIKey = anthropicKey
		config.Anthropic.BaseURL = "https://api.anthropic.com"
		config.Anthropic.Version = "2023-06-01"
		config.Anthropic.Models = []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
		}
	}

	// Ollama settings
	if ollamaEndpoint, ok := settings["ollamaEndpoint"].(string); ok {
		config.Ollama.Endpoint = ollamaEndpoint
		config.Ollama.PullTimeout = 10 * time.Minute
		config.Ollama.GenerateTimeout = 2 * time.Minute
		config.Ollama.AutoPullModels = true
		config.Ollama.Models = []string{
			"sqlcoder:7b", "codellama:7b", "llama3.1:8b", "mistral:7b",
		}
	}

	// Hugging Face settings
	if huggingfaceEndpoint, ok := settings["huggingfaceEndpoint"].(string); ok {
		config.HuggingFace.Endpoint = huggingfaceEndpoint
		config.HuggingFace.PullTimeout = 10 * time.Minute
		config.HuggingFace.GenerateTimeout = 2 * time.Minute
		config.HuggingFace.AutoPullModels = true
		config.HuggingFace.RecommendedModel = "sqlcoder:7b"
		config.HuggingFace.Models = []string{
			"sqlcoder:7b", "codellama:7b", "llama3.1:8b", "mistral:7b",
		}
	}

	return config, nil
}
