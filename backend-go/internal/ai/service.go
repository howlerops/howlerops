package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// serviceImpl implements the Service interface
type serviceImpl struct {
	config    *Config
	providers map[Provider]AIProvider
	logger    *logrus.Logger
	usage     map[Provider]*Usage
	usageMu   sync.RWMutex
	started   bool
	mu        sync.RWMutex
}

// NewService creates a new AI service instance
func NewService(config *Config, logger *logrus.Logger) (Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	service := &serviceImpl{
		config:    config,
		providers: make(map[Provider]AIProvider),
		logger:    logger,
		usage:     make(map[Provider]*Usage),
	}

	// Initialize providers based on configuration
	if err := service.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	return service, nil
}

// initializeProviders initializes all configured providers
func (s *serviceImpl) initializeProviders() error {
	// Initialize OpenAI provider if configured
	if s.config.OpenAI.APIKey != "" {
		provider, err := NewOpenAIProvider(&s.config.OpenAI, s.logger)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to initialize OpenAI provider")
		} else {
			s.providers[ProviderOpenAI] = provider
			s.usage[ProviderOpenAI] = &Usage{
				Provider:     ProviderOpenAI,
				RequestCount: 0,
				TokensUsed:   0,
				SuccessRate:  1.0,
			}
		}
	}

	// Initialize Anthropic provider if configured
	if s.config.Anthropic.APIKey != "" {
		provider, err := NewAnthropicProvider(&s.config.Anthropic, s.logger)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to initialize Anthropic provider")
		} else {
			s.providers[ProviderAnthropic] = provider
			s.usage[ProviderAnthropic] = &Usage{
				Provider:     ProviderAnthropic,
				RequestCount: 0,
				TokensUsed:   0,
				SuccessRate:  1.0,
			}
		}
	}

	// Initialize Ollama provider if configured
	if s.config.Ollama.Endpoint != "" {
		provider, err := NewOllamaProvider(&s.config.Ollama, s.logger)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to initialize Ollama provider")
		} else {
			s.providers[ProviderOllama] = provider
			s.usage[ProviderOllama] = &Usage{
				Provider:     ProviderOllama,
				RequestCount: 0,
				TokensUsed:   0,
				SuccessRate:  1.0,
			}
		}
	}

	// Initialize Hugging Face provider if configured
	if s.config.HuggingFace.Endpoint != "" {
		provider, err := NewHuggingFaceProvider(&s.config.HuggingFace, s.logger)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to initialize Hugging Face provider")
		} else {
			s.providers[ProviderHuggingFace] = provider
			s.usage[ProviderHuggingFace] = &Usage{
				Provider:     ProviderHuggingFace,
				RequestCount: 0,
				TokensUsed:   0,
				SuccessRate:  1.0,
			}
		}
	}

	// Initialize Claude Code provider if configured
	if s.config.ClaudeCode.ClaudePath != "" || s.config.ClaudeCode.Model != "" {
		adapter, err := NewClaudeCodeProvider(&s.config.ClaudeCode)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to initialize Claude Code provider")
		} else {
			// Wrap adapter to conform to AIProvider interface
			provider := &providerAdapterWrapper{
				adapter: adapter,
				logger:  s.logger,
			}
			s.providers[ProviderClaudeCode] = provider
			s.usage[ProviderClaudeCode] = &Usage{
				Provider:     ProviderClaudeCode,
				RequestCount: 0,
				TokensUsed:   0,
				SuccessRate:  1.0,
			}
		}
	}

	// Initialize Codex provider if configured
	if s.config.Codex.APIKey != "" {
		adapter, err := NewCodexProvider(&s.config.Codex)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to initialize Codex provider")
		} else {
			// Wrap adapter to conform to AIProvider interface
			provider := &providerAdapterWrapper{
				adapter: adapter,
				logger:  s.logger,
			}
			s.providers[ProviderCodex] = provider
			s.usage[ProviderCodex] = &Usage{
				Provider:     ProviderCodex,
				RequestCount: 0,
				TokensUsed:   0,
				SuccessRate:  1.0,
			}
		}
	}

	if len(s.providers) == 0 {
		return fmt.Errorf("no AI providers configured")
	}

	s.logger.WithField("providers", len(s.providers)).Info("AI providers initialized")
	return nil
}

// Start starts the AI service
func (s *serviceImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("service already started")
	}

	// Test all providers
	for providerType, provider := range s.providers {
		if !provider.IsAvailable(ctx) {
			s.logger.WithField("provider", providerType).Warn("Provider not available")
		}
	}

	s.started = true
	s.logger.Info("AI service started")
	return nil
}

// Stop stops the AI service
func (s *serviceImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("service not started")
	}

	s.started = false
	s.logger.Info("AI service stopped")
	return nil
}

// GenerateSQL generates SQL from natural language
func (s *serviceImpl) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	if err := s.ValidateRequest(req); err != nil {
		return nil, err
	}

	provider, exists := s.providers[req.Provider]
	if !exists {
		return nil, NewAIError(ErrorTypeProviderError, fmt.Sprintf("provider %s not available", req.Provider), req.Provider)
	}

	start := time.Now()
	response, err := provider.GenerateSQL(ctx, req)
	duration := time.Since(start)

	// Update usage statistics
	s.updateUsage(req.Provider, req.Model, err == nil, duration, response)

	if err != nil {
		s.logger.WithError(err).WithField("provider", req.Provider).Error("Failed to generate SQL")
		return nil, err
	}

	response.TimeTaken = duration
	s.logger.WithFields(logrus.Fields{
		"provider":   req.Provider,
		"model":      req.Model,
		"duration":   duration,
		"confidence": response.Confidence,
	}).Info("SQL generated successfully")

	return response, nil
}

// FixSQL fixes SQL based on error message
func (s *serviceImpl) FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	if err := s.ValidateRequest(req); err != nil {
		return nil, err
	}

	if req.Query == "" {
		return nil, NewAIError(ErrorTypeInvalidRequest, "query is required for fixing SQL", req.Provider)
	}

	if req.Error == "" {
		return nil, NewAIError(ErrorTypeInvalidRequest, "error message is required for fixing SQL", req.Provider)
	}

	provider, exists := s.providers[req.Provider]
	if !exists {
		return nil, NewAIError(ErrorTypeProviderError, fmt.Sprintf("provider %s not available", req.Provider), req.Provider)
	}

	start := time.Now()
	response, err := provider.FixSQL(ctx, req)
	duration := time.Since(start)

	// Update usage statistics
	s.updateUsage(req.Provider, req.Model, err == nil, duration, response)

	if err != nil {
		s.logger.WithError(err).WithField("provider", req.Provider).Error("Failed to fix SQL")
		return nil, err
	}

	response.TimeTaken = duration
	s.logger.WithFields(logrus.Fields{
		"provider":   req.Provider,
		"model":      req.Model,
		"duration":   duration,
		"confidence": response.Confidence,
	}).Info("SQL fixed successfully")

	return response, nil
}

// GetProviders returns list of available providers
func (s *serviceImpl) GetProviders() []Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]Provider, 0, len(s.providers))
	for provider := range s.providers {
		providers = append(providers, provider)
	}
	return providers
}

// GetProviderHealth returns health status of a specific provider
func (s *serviceImpl) GetProviderHealth(ctx context.Context, provider Provider) (*HealthStatus, error) {
	s.mu.RLock()
	p, exists := s.providers[provider]
	s.mu.RUnlock()

	if !exists {
		return &HealthStatus{
			Provider:    provider,
			Status:      "unavailable",
			Message:     "Provider not configured",
			LastChecked: time.Now(),
		}, nil
	}

	return p.HealthCheck(ctx)
}

// GetAllProvidersHealth returns health status of all providers
func (s *serviceImpl) GetAllProvidersHealth(ctx context.Context) (map[Provider]*HealthStatus, error) {
	s.mu.RLock()
	providers := make(map[Provider]AIProvider)
	for k, v := range s.providers {
		providers[k] = v
	}
	s.mu.RUnlock()

	result := make(map[Provider]*HealthStatus)
	for providerType, provider := range providers {
		health, err := provider.HealthCheck(ctx)
		if err != nil {
			health = &HealthStatus{
				Provider:    providerType,
				Status:      "error",
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
		}
		result[providerType] = health
	}

	return result, nil
}

// GetAvailableModels returns available models for a provider
func (s *serviceImpl) GetAvailableModels(ctx context.Context, provider Provider) ([]ModelInfo, error) {
	s.mu.RLock()
	p, exists := s.providers[provider]
	s.mu.RUnlock()

	if !exists {
		return nil, NewAIError(ErrorTypeProviderError, fmt.Sprintf("provider %s not available", provider), provider)
	}

	return p.GetModels(ctx)
}

// GetAllAvailableModels returns available models for all providers
func (s *serviceImpl) GetAllAvailableModels(ctx context.Context) (map[Provider][]ModelInfo, error) {
	s.mu.RLock()
	providers := make(map[Provider]AIProvider)
	for k, v := range s.providers {
		providers[k] = v
	}
	s.mu.RUnlock()

	result := make(map[Provider][]ModelInfo)
	for providerType, provider := range providers {
		models, err := provider.GetModels(ctx)
		if err != nil {
			s.logger.WithError(err).WithField("provider", providerType).Warn("Failed to get models")
			continue
		}
		result[providerType] = models
	}

	return result, nil
}

func (s *serviceImpl) defaultModelFor(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		if len(s.config.OpenAI.Models) > 0 {
			return s.config.OpenAI.Models[0]
		}
	case ProviderAnthropic:
		if len(s.config.Anthropic.Models) > 0 {
			return s.config.Anthropic.Models[0]
		}
	case ProviderOllama:
		if len(s.config.Ollama.Models) > 0 {
			return s.config.Ollama.Models[0]
		}
	case ProviderHuggingFace:
		if len(s.config.HuggingFace.Models) > 0 {
			return s.config.HuggingFace.Models[0]
		}
	case ProviderClaudeCode:
		if s.config.ClaudeCode.Model != "" {
			return s.config.ClaudeCode.Model
		}
	case ProviderCodex:
		if s.config.Codex.Model != "" {
			return s.config.Codex.Model
		}
	}
	return ""
}

// UpdateProviderConfig updates configuration for a specific provider
func (s *serviceImpl) UpdateProviderConfig(provider Provider, config interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, exists := s.providers[provider]
	if !exists {
		return NewAIError(ErrorTypeProviderError, fmt.Sprintf("provider %s not available", provider), provider)
	}

	return p.UpdateConfig(config)
}

// GetConfig returns the current configuration
func (s *serviceImpl) GetConfig() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetUsageStats returns usage statistics for a provider
func (s *serviceImpl) GetUsageStats(ctx context.Context, provider Provider) (*Usage, error) {
	s.usageMu.RLock()
	defer s.usageMu.RUnlock()

	usage, exists := s.usage[provider]
	if !exists {
		return nil, NewAIError(ErrorTypeProviderError, fmt.Sprintf("provider %s not available", provider), provider)
	}

	// Return a copy to prevent modification
	return &Usage{
		Provider:        usage.Provider,
		Model:           usage.Model,
		RequestCount:    usage.RequestCount,
		TokensUsed:      usage.TokensUsed,
		SuccessRate:     usage.SuccessRate,
		AvgResponseTime: usage.AvgResponseTime,
		LastUsed:        usage.LastUsed,
	}, nil
}

// GetAllUsageStats returns usage statistics for all providers
func (s *serviceImpl) GetAllUsageStats(ctx context.Context) (map[Provider]*Usage, error) {
	s.usageMu.RLock()
	defer s.usageMu.RUnlock()

	result := make(map[Provider]*Usage)
	for provider, usage := range s.usage {
		result[provider] = &Usage{
			Provider:        usage.Provider,
			Model:           usage.Model,
			RequestCount:    usage.RequestCount,
			TokensUsed:      usage.TokensUsed,
			SuccessRate:     usage.SuccessRate,
			AvgResponseTime: usage.AvgResponseTime,
			LastUsed:        usage.LastUsed,
		}
	}

	return result, nil
}

// TestProvider tests a provider with given configuration
func (s *serviceImpl) TestProvider(ctx context.Context, provider Provider, config interface{}) error {
	var testProvider AIProvider
	var err error

	switch provider {
	case ProviderOpenAI:
		cfg, ok := config.(*OpenAIConfig)
		if !ok {
			return NewAIError(ErrorTypeConfigError, "invalid OpenAI config", provider)
		}
		testProvider, err = NewOpenAIProvider(cfg, s.logger)
	case ProviderAnthropic:
		cfg, ok := config.(*AnthropicConfig)
		if !ok {
			return NewAIError(ErrorTypeConfigError, "invalid Anthropic config", provider)
		}
		testProvider, err = NewAnthropicProvider(cfg, s.logger)
	case ProviderOllama:
		cfg, ok := config.(*OllamaConfig)
		if !ok {
			return NewAIError(ErrorTypeConfigError, "invalid Ollama config", provider)
		}
		testProvider, err = NewOllamaProvider(cfg, s.logger)
	default:
		return NewAIError(ErrorTypeProviderError, fmt.Sprintf("unknown provider: %s", provider), provider)
	}

	if err != nil {
		return err
	}

	// Test with a simple health check
	_, err = testProvider.HealthCheck(ctx)
	return err
}

// ValidateRequest validates a SQL request
func (s *serviceImpl) ValidateRequest(req *SQLRequest) error {
	if req == nil {
		return NewAIError(ErrorTypeInvalidRequest, "request cannot be nil", "")
	}

	if req.Provider == "" {
		req.Provider = s.config.DefaultProvider
	}

	if req.Model == "" {
		req.Model = s.defaultModelFor(req.Provider)
	}

	if req.Model == "" {
		return NewAIError(ErrorTypeInvalidRequest, "model is required", req.Provider)
	}

	if req.MaxTokens <= 0 {
		req.MaxTokens = s.config.MaxTokens
	}

	if req.Temperature < 0 || req.Temperature > 1 {
		req.Temperature = s.config.Temperature
	}

	if req.Prompt == "" && req.Query == "" {
		return NewAIError(ErrorTypeInvalidRequest, "either prompt or query is required", req.Provider)
	}

	return nil
}

// updateUsage updates usage statistics
func (s *serviceImpl) updateUsage(provider Provider, model string, success bool, duration time.Duration, response *SQLResponse) {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	usage, exists := s.usage[provider]
	if !exists {
		usage = &Usage{
			Provider:     provider,
			RequestCount: 0,
			TokensUsed:   0,
			SuccessRate:  1.0,
		}
		s.usage[provider] = usage
	}

	usage.RequestCount++
	usage.LastUsed = time.Now()
	usage.Model = model

	if response != nil && response.TokensUsed > 0 {
		usage.TokensUsed += int64(response.TokensUsed)
	}

	// Update average response time
	if usage.AvgResponseTime == 0 {
		usage.AvgResponseTime = duration
	} else {
		usage.AvgResponseTime = (usage.AvgResponseTime + duration) / 2
	}

	// Update success rate (simple moving average)
	if success {
		usage.SuccessRate = (usage.SuccessRate*float64(usage.RequestCount-1) + 1.0) / float64(usage.RequestCount)
	} else {
		usage.SuccessRate = (usage.SuccessRate*float64(usage.RequestCount-1) + 0.0) / float64(usage.RequestCount)
	}
}
