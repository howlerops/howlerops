package ai

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	aipb "github.com/sql-studio/backend-go/pkg/pb/ai"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCServer implements the AI gRPC service.
type GRPCServer struct {
	aipb.UnimplementedAIServiceServer
	service Service
	logger  *logrus.Logger
}

// NewGRPCServer creates a new gRPC server for the AI service.
func NewGRPCServer(service Service, logger *logrus.Logger) *GRPCServer {
	return &GRPCServer{
		service: service,
		logger:  logger,
	}
}

// GenerateSQL implements the GenerateSQL gRPC method.
func (s *GRPCServer) GenerateSQL(ctx context.Context, req *aipb.GenerateSQLRequest) (*aipb.GenerateSQLResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"provider": req.Request.Provider,
		"model":    req.Request.Model,
		"prompt":   truncateString(req.Request.Prompt, 100),
	}).Info("Received GenerateSQL request")

	internalReq := &SQLRequest{
		Provider:    protoToProvider(req.Request.Provider),
		Model:       req.Request.Model,
		Prompt:      req.Request.Prompt,
		Schema:      req.Request.Schema,
		MaxTokens:   int(req.Request.MaxTokens),
		Temperature: req.Request.Temperature,
	}

	start := time.Now()
	resp, err := s.service.GenerateSQL(ctx, internalReq)
	timeTaken := time.Since(start)

	if err != nil {
		s.logger.WithError(err).Error("GenerateSQL failed")
		return &aipb.GenerateSQLResponse{
			Error: errorToProto(err),
		}, nil
	}

	return &aipb.GenerateSQLResponse{
		Response: &aipb.SQLResponse{
			Query:       resp.Query,
			Explanation: resp.Explanation,
			Confidence:  resp.Confidence,
			Suggestions: resp.Suggestions,
			Provider:    providerToProto(resp.Provider),
			Model:       resp.Model,
			// #nosec G115 - token counts from LLMs are reasonable (<100k), well within int32 range
			TokensUsed: int32(resp.TokensUsed),
			TimeTaken:  durationpb.New(timeTaken),
			Metadata:   resp.Metadata,
		},
	}, nil
}

// FixSQL implements the FixSQL gRPC method.
func (s *GRPCServer) FixSQL(ctx context.Context, req *aipb.FixSQLRequest) (*aipb.FixSQLResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"provider": req.Request.Provider,
		"model":    req.Request.Model,
		"query":    truncateString(req.Request.Query, 100),
		"error":    truncateString(req.Request.Error, 100),
	}).Info("Received FixSQL request")

	internalReq := &SQLRequest{
		Provider:    protoToProvider(req.Request.Provider),
		Model:       req.Request.Model,
		Query:       req.Request.Query,
		Error:       req.Request.Error,
		Schema:      req.Request.Schema,
		MaxTokens:   int(req.Request.MaxTokens),
		Temperature: req.Request.Temperature,
	}

	start := time.Now()
	resp, err := s.service.FixSQL(ctx, internalReq)
	timeTaken := time.Since(start)

	if err != nil {
		s.logger.WithError(err).Error("FixSQL failed")
		return &aipb.FixSQLResponse{
			Error: errorToProto(err),
		}, nil
	}

	return &aipb.FixSQLResponse{
		Response: &aipb.SQLResponse{
			Query:       resp.Query,
			Explanation: resp.Explanation,
			Confidence:  resp.Confidence,
			Suggestions: resp.Suggestions,
			Provider:    providerToProto(resp.Provider),
			Model:       resp.Model,
			// #nosec G115 - token counts from LLMs are reasonable (<100k), well within int32 range
			TokensUsed: int32(resp.TokensUsed),
			TimeTaken:  durationpb.New(timeTaken),
			Metadata:   resp.Metadata,
		},
	}, nil
}

// GetProviderHealth implements the GetProviderHealth gRPC method.
func (s *GRPCServer) GetProviderHealth(ctx context.Context, req *aipb.GetProviderHealthRequest) (*aipb.ProviderHealthResponse, error) {
	s.logger.WithField("provider", req.Provider).Info("Received GetProviderHealth request")

	provider := protoToProvider(req.Provider)
	health, err := s.service.GetProviderHealth(ctx, provider)

	if err != nil {
		s.logger.WithError(err).Error("GetProviderHealth failed")
		return &aipb.ProviderHealthResponse{
			Error: errorToProto(err),
		}, nil
	}

	return &aipb.ProviderHealthResponse{
		Health: healthStatusToProto(health),
	}, nil
}

// GetProviderModels implements the GetProviderModels gRPC method.
func (s *GRPCServer) GetProviderModels(ctx context.Context, req *aipb.GetProviderModelsRequest) (*aipb.GetProviderModelsResponse, error) {
	s.logger.WithField("provider", req.Provider).Info("Received GetProviderModels request")

	provider := protoToProvider(req.Provider)
	models, err := s.service.GetAvailableModels(ctx, provider)

	if err != nil {
		s.logger.WithError(err).Error("GetProviderModels failed")
		return &aipb.GetProviderModelsResponse{
			Error: errorToProto(err),
		}, nil
	}

	protoModels := make([]*aipb.ModelInfo, 0, len(models))
	for _, model := range models {
		protoModels = append(protoModels, &aipb.ModelInfo{
			Id:          model.ID,
			Name:        model.Name,
			Provider:    providerToProto(model.Provider),
			Description: model.Description,
			// #nosec G115 - model max tokens are configured values (<1M), well within int32 range
			MaxTokens:    int32(model.MaxTokens),
			Capabilities: model.Capabilities,
			Metadata:     model.Metadata,
		})
	}

	return &aipb.GetProviderModelsResponse{
		Models: protoModels,
	}, nil
}

// TestProvider implements the TestProvider gRPC method.
func (s *GRPCServer) TestProvider(ctx context.Context, req *aipb.TestProviderRequest) (*aipb.TestProviderResponse, error) {
	s.logger.WithField("provider", req.Provider).Info("Received TestProvider request")

	var config interface{}
	switch req.Provider {
	case aipb.Provider_PROVIDER_OPENAI:
		if req.GetOpenaiConfig() != nil {
			config = &OpenAIConfig{
				APIKey: req.GetOpenaiConfig().ApiKey,
			}
		}
	case aipb.Provider_PROVIDER_ANTHROPIC:
		if req.GetAnthropicConfig() != nil {
			config = &AnthropicConfig{
				APIKey: req.GetAnthropicConfig().ApiKey,
			}
		}
	case aipb.Provider_PROVIDER_OLLAMA:
		if req.GetOllamaConfig() != nil {
			config = &OllamaConfig{
				Endpoint: req.GetOllamaConfig().Endpoint,
			}
		}
	case aipb.Provider_PROVIDER_HUGGINGFACE:
		if req.GetHuggingfaceConfig() != nil {
			config = &HuggingFaceConfig{
				Endpoint: req.GetHuggingfaceConfig().Endpoint,
			}
		}
	}

	provider := protoToProvider(req.Provider)
	err := s.service.TestProvider(ctx, provider, config)

	if err != nil {
		s.logger.WithError(err).Error("TestProvider failed")
		return &aipb.TestProviderResponse{
			Success: false,
			Error:   errorToProto(err),
		}, nil
	}

	return &aipb.TestProviderResponse{
		Success: true,
	}, nil
}

// GetUsageStats implements the GetUsageStats gRPC method.
func (s *GRPCServer) GetUsageStats(ctx context.Context, req *aipb.GetUsageStatsRequest) (*aipb.GetUsageStatsResponse, error) {
	s.logger.Info("Received GetUsageStats request")

	stats, err := s.service.GetAllUsageStats(ctx)

	if err != nil {
		s.logger.WithError(err).Error("GetUsageStats failed")
		return &aipb.GetUsageStatsResponse{
			Error: errorToProto(err),
		}, nil
	}

	protoStats := make(map[string]*aipb.Usage, len(stats))
	for provider, usage := range stats {
		protoStats[string(provider)] = &aipb.Usage{
			Provider:        providerToProto(usage.Provider),
			Model:           usage.Model,
			RequestCount:    usage.RequestCount,
			TokensUsed:      usage.TokensUsed,
			SuccessRate:     usage.SuccessRate,
			LastUsed:        timestamppb.New(usage.LastUsed),
			AvgResponseTime: durationpb.New(usage.AvgResponseTime),
		}
	}

	return &aipb.GetUsageStatsResponse{
		UsageStats: protoStats,
	}, nil
}

// GetConfig implements the GetConfig gRPC method.
func (s *GRPCServer) GetConfig(ctx context.Context, req *aipb.GetConfigRequest) (*aipb.GetConfigResponse, error) {
	s.logger.Info("Received GetConfig request")

	config := s.service.GetConfig()

	protoConfig := &aipb.AIConfig{
		DefaultProvider: providerToProto(config.DefaultProvider),
		// #nosec G115 - config max tokens are reasonable (<1M), well within int32 range
		MaxTokens:      int32(config.MaxTokens),
		Temperature:    config.Temperature,
		RequestTimeout: durationpb.New(config.RequestTimeout),
		// #nosec G115 - rate limit is config value (<10000), well within int32 range
		RateLimitPerMin: int32(config.RateLimitPerMin),
	}

	if config.OpenAI.APIKey != "" {
		protoConfig.Openai = &aipb.OpenAIConfig{
			ApiKey: config.OpenAI.APIKey,
		}
	}

	if config.Anthropic.APIKey != "" {
		protoConfig.Anthropic = &aipb.AnthropicConfig{
			ApiKey: config.Anthropic.APIKey,
		}
	}

	if config.Ollama.Endpoint != "" {
		protoConfig.Ollama = &aipb.OllamaConfig{
			Endpoint: config.Ollama.Endpoint,
		}
	}

	if config.HuggingFace.Endpoint != "" {
		protoConfig.Huggingface = &aipb.HuggingFaceConfig{
			Endpoint:         config.HuggingFace.Endpoint,
			Models:           config.HuggingFace.Models,
			PullTimeout:      durationpb.New(config.HuggingFace.PullTimeout),
			GenerateTimeout:  durationpb.New(config.HuggingFace.GenerateTimeout),
			AutoPullModels:   config.HuggingFace.AutoPullModels,
			RecommendedModel: config.HuggingFace.RecommendedModel,
			Configured:       true,
		}
	}

	return &aipb.GetConfigResponse{
		Config: protoConfig,
	}, nil
}

// Helper functions for converting between protobuf and internal types.

func protoToProvider(protoProv aipb.Provider) Provider {
	switch protoProv {
	case aipb.Provider_PROVIDER_OPENAI:
		return ProviderOpenAI
	case aipb.Provider_PROVIDER_ANTHROPIC:
		return ProviderAnthropic
	case aipb.Provider_PROVIDER_OLLAMA:
		return ProviderOllama
	case aipb.Provider_PROVIDER_HUGGINGFACE:
		return ProviderHuggingFace
	default:
		return ProviderOpenAI
	}
}

func providerToProto(p Provider) aipb.Provider {
	switch p {
	case ProviderOpenAI:
		return aipb.Provider_PROVIDER_OPENAI
	case ProviderAnthropic:
		return aipb.Provider_PROVIDER_ANTHROPIC
	case ProviderOllama:
		return aipb.Provider_PROVIDER_OLLAMA
	case ProviderHuggingFace:
		return aipb.Provider_PROVIDER_HUGGINGFACE
	default:
		return aipb.Provider_PROVIDER_UNSPECIFIED
	}
}

func healthStatusToProto(h *HealthStatus) *aipb.ProviderHealth {
	var status aipb.HealthStatus
	switch h.Status {
	case "healthy":
		status = aipb.HealthStatus_HEALTH_HEALTHY
	case "unhealthy":
		status = aipb.HealthStatus_HEALTH_UNHEALTHY
	case "error":
		status = aipb.HealthStatus_HEALTH_ERROR
	default:
		status = aipb.HealthStatus_HEALTH_UNKNOWN
	}

	return &aipb.ProviderHealth{
		Provider:     providerToProto(h.Provider),
		Status:       status,
		Message:      h.Message,
		LastChecked:  timestamppb.New(h.LastChecked),
		ResponseTime: durationpb.New(h.ResponseTime),
	}
}

func errorToProto(err error) *aipb.AIError {
	if err == nil {
		return nil
	}

	return &aipb.AIError{
		Type:      aipb.ErrorType_ERROR_UNKNOWN,
		Message:   err.Error(),
		Retryable: false,
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
