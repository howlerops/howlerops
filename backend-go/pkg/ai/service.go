package ai

import (
	"context"
	"fmt"

	"github.com/jbeck018/howlerops/backend-go/internal/ai"
	"github.com/sirupsen/logrus"
)

// Service is a public wrapper around the internal AI service
type Service struct {
	internal ai.Service
	logger   *logrus.Logger
}

// RuntimeConfig re-exports the internal AI configuration type for callers.
type RuntimeConfig = ai.Config

// Provider re-exports the provider identifier type.
type Provider = ai.Provider

const (
	ProviderOpenAI      = ai.ProviderOpenAI
	ProviderAnthropic   = ai.ProviderAnthropic
	ProviderOllama      = ai.ProviderOllama
	ProviderHuggingFace = ai.ProviderHuggingFace
	ProviderClaudeCode  = ai.ProviderClaudeCode
	ProviderCodex       = ai.ProviderCodex
)

// DefaultRuntimeConfig provides the baseline AI configuration without environment overrides.
func DefaultRuntimeConfig() *RuntimeConfig {
	return ai.DefaultConfig()
}

// NewService creates a new AI service using environment variable configuration
func NewService(logger *logrus.Logger) (*Service, error) {
	// Load config from environment variables (handled by internal package)
	internalConfig, err := ai.LoadConfig()
	if err != nil {
		return nil, err
	}

	internal, err := ai.NewService(internalConfig, logger)
	if err != nil {
		return nil, err
	}

	return &Service{
		internal: internal,
		logger:   logger,
	}, nil
}

// NewServiceWithConfig creates a new AI service using the provided configuration.
func NewServiceWithConfig(config *RuntimeConfig, logger *logrus.Logger) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := ai.ValidateConfig(config); err != nil {
		return nil, err
	}

	internal, err := ai.NewService(config, logger)
	if err != nil {
		return nil, err
	}

	return &Service{
		internal: internal,
		logger:   logger,
	}, nil
}

// GenerateSQL generates SQL from a natural language prompt
func (s *Service) GenerateSQL(ctx context.Context, prompt string, schema string) (*SQLResponse, error) {
	return s.GenerateSQLWithRequest(ctx, &SQLRequest{
		Prompt: prompt,
		Schema: schema,
	})
}

// GenerateSQLWithRequest generates SQL using a detailed request payload
func (s *Service) GenerateSQLWithRequest(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	internalReq := s.toInternalRequest(req)
	resp, err := s.internal.GenerateSQL(ctx, internalReq)
	if err != nil {
		return nil, err
	}

	return &SQLResponse{
		SQL:         resp.Query,
		Confidence:  resp.Confidence,
		Explanation: resp.Explanation,
		TokensUsed:  resp.TokensUsed,
	}, nil
}

// FixQuery attempts to fix a SQL error
func (s *Service) FixQuery(ctx context.Context, query string, errorMsg string) (*SQLResponse, error) {
	return s.FixQueryWithRequest(ctx, &SQLRequest{
		Query: query,
		Error: errorMsg,
	})
}

// FixQueryWithRequest attempts to fix SQL using a detailed request payload
func (s *Service) FixQueryWithRequest(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
	internalReq := s.toInternalRequest(req)
	resp, err := s.internal.FixSQL(ctx, internalReq)
	if err != nil {
		return nil, err
	}

	return &SQLResponse{
		SQL:         resp.Query,
		Confidence:  resp.Confidence,
		Explanation: resp.Explanation,
		TokensUsed:  resp.TokensUsed,
	}, nil
}

// Chat handles a generic conversational request
func (s *Service) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	internalReq := s.toInternalChatRequest(req)
	resp, err := s.internal.Chat(ctx, internalReq)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content:    resp.Content,
		Provider:   string(resp.Provider),
		Model:      resp.Model,
		TokensUsed: resp.TokensUsed,
		Metadata:   resp.Metadata,
	}, nil
}

func (s *Service) toInternalRequest(req *SQLRequest) *ai.SQLRequest {
	if req == nil {
		return &ai.SQLRequest{}
	}

	return &ai.SQLRequest{
		Prompt:      req.Prompt,
		Query:       req.Query,
		Error:       req.Error,
		Schema:      req.Schema,
		Provider:    ai.Provider(req.Provider),
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
}

func (s *Service) toInternalChatRequest(req *ChatRequest) *ai.ChatRequest {
	if req == nil {
		return &ai.ChatRequest{}
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return &ai.ChatRequest{
		Prompt:      req.Prompt,
		Context:     req.Context,
		System:      req.System,
		Provider:    ai.Provider(req.Provider),
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Metadata:    metadata,
	}
}

// OptimizeQuery suggests optimizations for a query
func (s *Service) OptimizeQuery(ctx context.Context, query string) (*OptimizationResponse, error) {
	req := &ai.SQLRequest{
		Prompt: "Optimize this SQL query: " + query,
		Query:  query,
	}

	resp, err := s.internal.GenerateSQL(ctx, req)
	if err != nil {
		return nil, err
	}

	return &OptimizationResponse{
		OptimizedSQL: resp.Query,
		Explanation:  resp.Explanation,
		Impact:       "Medium-High", // TODO: Parse from response
		TokensUsed:   resp.TokensUsed,
	}, nil
}

// ExplainQuery provides a natural language explanation of a query
func (s *Service) ExplainQuery(ctx context.Context, query string) (*ExplanationResponse, error) {
	req := &ai.SQLRequest{
		Prompt: "Explain what this SQL query does in simple terms: " + query,
		Query:  query,
	}

	resp, err := s.internal.GenerateSQL(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ExplanationResponse{
		Explanation: resp.Explanation,
		TokensUsed:  resp.TokensUsed,
	}, nil
}

// GetProviders returns the status of all available AI providers
func (s *Service) GetProviders(ctx context.Context) ([]ProviderStatus, error) {
	providers := s.internal.GetProviders()

	statuses := make([]ProviderStatus, 0, len(providers))
	for _, p := range providers {
		health, err := s.internal.GetProviderHealth(ctx, p)
		available := err == nil && health != nil && health.Status == "healthy"

		statuses = append(statuses, ProviderStatus{
			Name:         string(p),
			Available:    available,
			RequestCount: 0, // Not tracked in HealthStatus
			SuccessRate:  1.0,
		})
	}

	return statuses, nil
}

// Start starts the AI service
func (s *Service) Start(ctx context.Context) error {
	return s.internal.Start(ctx)
}

// Stop stops the AI service
func (s *Service) Stop(ctx context.Context) error {
	return s.internal.Stop(ctx)
}
