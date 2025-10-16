package ai

import (
	"fmt"
	"os"
	"time"
)

// LoadConfig loads AI configuration from environment variables and defaults
func LoadConfig() (*Config, error) {
	config := &Config{
		DefaultProvider: ProviderOpenAI,
		MaxTokens:       2048,
		Temperature:     0.1,
		RequestTimeout:  60 * time.Second,
		RateLimitPerMin: 60,
	}

	// OpenAI Configuration
	config.OpenAI = OpenAIConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: getEnvWithDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		Models: []string{
			"gpt-4o-mini",
			"gpt-4o",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
		},
		OrgID: os.Getenv("OPENAI_ORG_ID"),
	}

	// Anthropic Configuration
	config.Anthropic = AnthropicConfig{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL: getEnvWithDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
		Version: getEnvWithDefault("ANTHROPIC_VERSION", "2023-06-01"),
		Models: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
		},
	}

	// Ollama Configuration
	config.Ollama = OllamaConfig{
		Endpoint:        getEnvWithDefault("OLLAMA_ENDPOINT", "http://localhost:11434"),
		PullTimeout:     10 * time.Minute,
		GenerateTimeout: 2 * time.Minute,
		AutoPullModels:  getEnvBool("OLLAMA_AUTO_PULL", true),
		Models: []string{
			"sqlcoder:7b",
			"codellama:7b",
			"llama3.1:8b",
			"mistral:7b",
		},
	}

	// Hugging Face Configuration
	config.HuggingFace = HuggingFaceConfig{
		Endpoint:         getEnvWithDefault("HUGGINGFACE_ENDPOINT", "http://localhost:11434"),
		PullTimeout:      10 * time.Minute,
		GenerateTimeout:  2 * time.Minute,
		AutoPullModels:   getEnvBool("HUGGINGFACE_AUTO_PULL", true),
		RecommendedModel: "sqlcoder:7b",
		Models: []string{
			"sqlcoder:7b",
			"codellama:7b",
			"llama3.1:8b",
			"mistral:7b",
		},
	}

	// Claude Code Configuration
	claudePath := os.Getenv("CLAUDE_CLI_PATH")
	if claudePath == "" {
		claudePath = "claude"
	}

	config.ClaudeCode = ClaudeCodeConfig{
		ClaudePath:  claudePath,
		Model:       getEnvWithDefault("CLAUDE_CODE_MODEL", "opus"),
		MaxTokens:   getEnvIntWithDefault("CLAUDE_CODE_MAX_TOKENS", 4096),
		Temperature: getEnvFloatWithDefault("CLAUDE_CODE_TEMPERATURE", 0.7),
	}

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

	if !hasProvider {
		return fmt.Errorf("at least one AI provider must be configured")
	}

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
