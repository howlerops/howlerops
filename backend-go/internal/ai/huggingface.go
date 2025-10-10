package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// huggingFaceProvider implements the AIProvider interface for Hugging Face models via Ollama
type huggingFaceProvider struct {
	config     *HuggingFaceConfig
	ollama     *OllamaProvider
	detector   *OllamaDetector
	logger     *logrus.Logger
}

// NewHuggingFaceProvider creates a new Hugging Face provider that uses Ollama
func NewHuggingFaceProvider(config *HuggingFaceConfig, logger *logrus.Logger) (AIProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set defaults
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:11434"
	}

	if config.PullTimeout == 0 {
		config.PullTimeout = 10 * time.Minute
	}

	if config.GenerateTimeout == 0 {
		config.GenerateTimeout = 2 * time.Minute
	}

	if config.RecommendedModel == "" {
		config.RecommendedModel = "prem-1b-sql"
	}

	if len(config.Models) == 0 {
		config.Models = []string{
			"prem-1b-sql",
			"sqlcoder:7b",
			"codellama:7b",
			"llama3.1:8b",
			"mistral:7b",
		}
	}

	// Create Ollama provider for actual model execution
	ollamaConfig := &OllamaConfig{
		Endpoint:         config.Endpoint,
		Models:           config.Models,
		PullTimeout:      config.PullTimeout,
		GenerateTimeout:  config.GenerateTimeout,
		AutoPullModels:   config.AutoPullModels,
	}

	ollamaProvider, err := NewOllamaProvider(ollamaConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama provider: %w", err)
	}

	ollama, ok := ollamaProvider.(*OllamaProvider)
	if !ok {
		return nil, fmt.Errorf("unexpected Ollama provider type")
	}

	detector := NewOllamaDetector(logger)

	return &huggingFaceProvider{
		config:   config,
		ollama:   ollama,
		detector: detector,
		logger:   logger,
	}, nil
}

// GenerateSQL generates SQL from natural language using Hugging Face models via Ollama
func (p *huggingFaceProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	// Ensure the recommended model is available
	if err := p.ensureModelAvailable(ctx, req.Model); err != nil {
		return nil, fmt.Errorf("model availability check failed: %w", err)
	}

	// Use Ollama provider for actual generation
	return p.ollama.GenerateSQL(ctx, req)
}

// FixSQL fixes SQL based on error message using Hugging Face models via Ollama
func (p *huggingFaceProvider) FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	// Ensure the recommended model is available
	if err := p.ensureModelAvailable(ctx, req.Model); err != nil {
		return nil, fmt.Errorf("model availability check failed: %w", err)
	}

	// Use Ollama provider for actual fixing
	return p.ollama.FixSQL(ctx, req)
}

// HealthCheck checks if the Hugging Face provider (via Ollama) is available
func (p *huggingFaceProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	// Check Ollama status first
	status, err := p.detector.DetectOllama(ctx)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderHuggingFace,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Failed to detect Ollama: %v", err),
			LastChecked: time.Now(),
		}, nil
	}

	if !status.Installed {
		return &HealthStatus{
			Provider:    ProviderHuggingFace,
			Status:      "unhealthy",
			Message:     "Ollama is not installed. Please install Ollama to use Hugging Face models.",
			LastChecked: time.Now(),
		}, nil
	}

	if !status.Running {
		return &HealthStatus{
			Provider:    ProviderHuggingFace,
			Status:      "unhealthy",
			Message:     "Ollama service is not running. Please start Ollama service.",
			LastChecked: time.Now(),
		}, nil
	}

	// Check if recommended model is available
	modelExists, err := p.detector.CheckModelExists(ctx, p.config.RecommendedModel)
	if err != nil {
		return &HealthStatus{
			Provider:    ProviderHuggingFace,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Failed to check model availability: %v", err),
			LastChecked: time.Now(),
		}, nil
	}

	if !modelExists {
		return &HealthStatus{
			Provider:    ProviderHuggingFace,
			Status:      "unhealthy",
			Message:     fmt.Sprintf("Recommended model '%s' is not available. Please pull it: ollama pull %s", p.config.RecommendedModel, p.config.RecommendedModel),
			LastChecked: time.Now(),
		}, nil
	}

	return &HealthStatus{
		Provider:     ProviderHuggingFace,
		Status:       "healthy",
		Message:      fmt.Sprintf("Hugging Face provider ready with Ollama. Recommended model '%s' available.", p.config.RecommendedModel),
		LastChecked:  time.Now(),
		ResponseTime: 0, // Will be set by actual health check
	}, nil
}

// GetModels returns available Hugging Face models via Ollama
func (p *huggingFaceProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	// Get models from Ollama
	ollamaModels, err := p.ollama.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	// Transform Ollama models to Hugging Face format
	var models []ModelInfo
	for _, model := range ollamaModels {
		// Mark as Hugging Face provider
		model.Provider = ProviderHuggingFace
		
		// Add Hugging Face specific descriptions
		switch {
		case strings.Contains(model.Name, "prem-1b-sql"):
			model.Description = "Prem-1B-SQL - Specialized 1B parameter model for SQL generation (Recommended)"
			model.Capabilities = []string{"text-to-sql", "sql-fixing", "sql-optimization", "schema-analysis"}
		case strings.Contains(model.Name, "sqlcoder"):
			model.Description = "SQLCoder - Specialized model for SQL code generation"
			model.Capabilities = []string{"text-to-sql", "sql-fixing", "sql-optimization", "schema-analysis"}
		case strings.Contains(model.Name, "codellama"):
			model.Description = "CodeLlama - General purpose code generation model"
			model.Capabilities = []string{"text-to-sql", "sql-fixing", "explanation", "debugging"}
		case strings.Contains(model.Name, "llama"):
			model.Description = "Llama - General purpose language model"
			model.Capabilities = []string{"text-to-sql", "sql-fixing", "explanation", "analysis"}
		case strings.Contains(model.Name, "mistral"):
			model.Description = "Mistral - General purpose language model"
			model.Capabilities = []string{"text-to-sql", "sql-fixing", "explanation", "analysis"}
		}

		models = append(models, model)
	}

	return models, nil
}

// GetProviderType returns the provider type
func (p *huggingFaceProvider) GetProviderType() Provider {
	return ProviderHuggingFace
}

// IsAvailable checks if the provider is available
func (p *huggingFaceProvider) IsAvailable(ctx context.Context) bool {
	health, err := p.HealthCheck(ctx)
	return err == nil && health.Status == "healthy"
}

// UpdateConfig updates the provider configuration
func (p *huggingFaceProvider) UpdateConfig(config interface{}) error {
	hfConfig, ok := config.(*HuggingFaceConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Hugging Face provider")
	}

	if err := p.ValidateConfig(hfConfig); err != nil {
		return err
	}

	p.config = hfConfig

	// Update underlying Ollama config
	ollamaConfig := &OllamaConfig{
		Endpoint:         hfConfig.Endpoint,
		Models:           hfConfig.Models,
		PullTimeout:      hfConfig.PullTimeout,
		GenerateTimeout:  hfConfig.GenerateTimeout,
		AutoPullModels:   hfConfig.AutoPullModels,
	}

	return p.ollama.UpdateConfig(ollamaConfig)
}

// ValidateConfig validates the provider configuration
func (p *huggingFaceProvider) ValidateConfig(config interface{}) error {
	hfConfig, ok := config.(*HuggingFaceConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Hugging Face provider")
	}

	if hfConfig.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	return nil
}

// ensureModelAvailable ensures the specified model is available, pulling it if necessary
func (p *huggingFaceProvider) ensureModelAvailable(ctx context.Context, modelName string) error {
	// Check if model exists
	exists, err := p.detector.CheckModelExists(ctx, modelName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Model doesn't exist, try to pull it
	if p.config.AutoPullModels {
		p.logger.WithField("model", modelName).Info("Model not found, attempting to pull")
		if err := p.detector.PullModel(ctx, modelName); err != nil {
			return fmt.Errorf("failed to pull model %s: %w", modelName, err)
		}
		return nil
	}

	return fmt.Errorf("model %s not found and auto-pull is disabled. Please pull it manually: ollama pull %s", modelName, modelName)
}

// GetRecommendedModel returns the recommended model for this provider
func (p *huggingFaceProvider) GetRecommendedModel() string {
	return p.config.RecommendedModel
}

// GetInstallationInstructions returns instructions for setting up the provider
func (p *huggingFaceProvider) GetInstallationInstructions() (string, error) {
	status, err := p.detector.DetectOllama(context.Background())
	if err != nil {
		return "", err
	}

	if !status.Installed {
		instructions, err := p.detector.InstallOllama()
		if err != nil {
			return "", err
		}
		return instructions, nil
	}

	if !status.Running {
		return "Ollama is installed but not running. Please start the Ollama service.", nil
	}

	// Check if recommended model is available
	exists, err := p.detector.CheckModelExists(context.Background(), p.config.RecommendedModel)
	if err != nil {
		return "", err
	}

	if !exists {
		return fmt.Sprintf("Ollama is running but the recommended model '%s' is not available. Please pull it:\n\nollama pull %s", p.config.RecommendedModel, p.config.RecommendedModel), nil
	}

	return "Hugging Face provider is ready to use!", nil
}
