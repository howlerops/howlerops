package ai

import (
	"context"

	"github.com/sirupsen/logrus"
)

const (
	healthyStatus = "healthy"
)

// providerAdapterWrapper wraps the new ProviderAdapter to work with the existing AIProvider interface
type providerAdapterWrapper struct {
	adapter ProviderAdapter
	logger  *logrus.Logger
}

// GenerateSQL implements AIProvider
func (w *providerAdapterWrapper) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	options := []GenerateOption{
		WithModel(req.Model),
		WithMaxTokens(req.MaxTokens),
		WithTemperature(req.Temperature),
		WithContext(req.Context),
	}

	return w.adapter.GenerateSQL(ctx, req.Prompt, req.Schema, options...)
}

// FixSQL implements AIProvider
func (w *providerAdapterWrapper) FixSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	options := []GenerateOption{
		WithModel(req.Model),
		WithMaxTokens(req.MaxTokens),
		WithTemperature(req.Temperature),
		WithContext(req.Context),
	}

	return w.adapter.FixSQL(ctx, req.Query, req.Error, req.Schema, options...)
}

// Chat implements AIProvider
func (w *providerAdapterWrapper) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	options := []GenerateOption{
		WithModel(req.Model),
		WithMaxTokens(req.MaxTokens),
		WithTemperature(req.Temperature),
	}

	contextMap := map[string]string{}
	if req.Context != "" {
		contextMap["context"] = req.Context
	}
	if req.System != "" {
		contextMap["system"] = req.System
	}
	if len(req.Metadata) > 0 {
		for key, value := range req.Metadata {
			contextMap[key] = value
		}
	}
	if len(contextMap) > 0 {
		options = append(options, WithContext(contextMap))
	}

	return w.adapter.Chat(ctx, req.Prompt, options...)
}

// HealthCheck implements AIProvider
func (w *providerAdapterWrapper) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	return w.adapter.GetHealth(ctx)
}

// GetModels implements AIProvider
func (w *providerAdapterWrapper) GetModels(ctx context.Context) ([]ModelInfo, error) {
	return w.adapter.ListModels(ctx)
}

// GetProviderType implements AIProvider
func (w *providerAdapterWrapper) GetProviderType() Provider {
	return w.adapter.GetProviderType()
}

// IsAvailable implements AIProvider
func (w *providerAdapterWrapper) IsAvailable(ctx context.Context) bool {
	health, err := w.adapter.GetHealth(ctx)
	if err != nil {
		return false
	}
	return health.Status == healthyStatus
}

// UpdateConfig implements AIProvider
func (w *providerAdapterWrapper) UpdateConfig(_ interface{}) error {
	// Not implemented for now
	return nil
}

// ValidateConfig implements AIProvider
func (w *providerAdapterWrapper) ValidateConfig(_ interface{}) error {
	// Not implemented for now
	return nil
}
