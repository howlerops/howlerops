// Package ai provides AI-powered text-to-SQL generation and query assistance
// for HowlerOps. It supports multiple AI providers including OpenAI, Anthropic,
// and local Ollama models.
//
// The package provides:
//   - Text-to-SQL generation from natural language prompts
//   - SQL query error fixing and optimization suggestions
//   - Multiple AI provider support with fallback capabilities
//   - Usage analytics and health monitoring
//   - Secure configuration management
//
// Example usage:
//
//	config, err := ai.LoadConfig()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	service, err := ai.NewService(config, logger)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	request := &ai.SQLRequest{
//		Prompt:   "Show all users who signed up last month",
//		Provider: ai.ProviderOpenAI,
//		Model:    "gpt-4o-mini",
//	}
//
//	response, err := service.GenerateSQL(ctx, request)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println("Generated SQL:", response.Query)
package ai

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Module represents the AI module for HowlerOps
type Module struct {
	service Service
	handler *HTTPHandler
	config  *Config
	logger  *logrus.Logger
}

// NewModule creates a new AI module
func NewModule(logger *logrus.Logger) (*Module, error) {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	// Create service
	service, err := NewService(config, logger)
	if err != nil {
		return nil, err
	}

	// Create HTTP handler
	handler := NewHTTPHandler(service, logger)

	return &Module{
		service: service,
		handler: handler,
		config:  config,
		logger:  logger,
	}, nil
}

// Start starts the AI module
func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("Starting AI module")
	return m.service.Start(ctx)
}

// Stop stops the AI module
func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("Stopping AI module")
	return m.service.Stop(ctx)
}

// GetService returns the AI service
func (m *Module) GetService() Service {
	return m.service
}

// GetHTTPHandler returns the HTTP handler
func (m *Module) GetHTTPHandler() *HTTPHandler {
	return m.handler
}

// GetConfig returns the configuration
func (m *Module) GetConfig() *Config {
	return m.config
}

// UpdateConfig updates the module configuration
func (m *Module) UpdateConfig(newConfig *Config) error {
	if err := ValidateConfig(newConfig); err != nil {
		return err
	}

	m.config = newConfig
	m.logger.Info("AI module configuration updated")
	return nil
}

// HealthCheck performs a health check on all providers
func (m *Module) HealthCheck(ctx context.Context) map[Provider]*HealthStatus {
	health, err := m.service.GetAllProvidersHealth(ctx)
	if err != nil {
		m.logger.WithError(err).Error("Failed to get provider health")
		return make(map[Provider]*HealthStatus)
	}
	return health
}

// GetUsageReport returns a usage report for all providers
func (m *Module) GetUsageReport(ctx context.Context) map[Provider]*Usage {
	usage, err := m.service.GetAllUsageStats(ctx)
	if err != nil {
		m.logger.WithError(err).Error("Failed to get usage stats")
		return make(map[Provider]*Usage)
	}
	return usage
}