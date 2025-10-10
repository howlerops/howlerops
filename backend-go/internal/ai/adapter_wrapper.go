package ai

import (
	"context"

	"github.com/sirupsen/logrus"
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
	return health.Status == "healthy"
}

// UpdateConfig implements AIProvider
func (w *providerAdapterWrapper) UpdateConfig(config interface{}) error {
	// Not implemented for now
	return nil
}

// ValidateConfig implements AIProvider
func (w *providerAdapterWrapper) ValidateConfig(config interface{}) error {
	// Not implemented for now
	return nil
}