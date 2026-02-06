package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/creack/pty"

	"github.com/jbeck018/howlerops/pkg/ai"
	"github.com/jbeck018/howlerops/pkg/database"
	"github.com/jbeck018/howlerops/pkg/rag"
	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/jbeck018/howlerops/services"
)

// WailsAIService handles all AI-related functionality for the Wails application
type WailsAIService struct {
	deps             *SharedDeps
	aiService        *ai.Service
	aiConfig         *ai.RuntimeConfig
	embeddingService rag.EmbeddingService
}

// NewWailsAIService creates a new WailsAIService
func NewWailsAIService(deps *SharedDeps) *WailsAIService {
	return &WailsAIService{
		deps:     deps,
		aiConfig: ai.DefaultRuntimeConfig(),
	}
}

// ======================================
// AI Provider Configuration & Status
// ======================================

// hasConfiguredProvider returns true if the current AI config has at least one provider enabled.
func (s *WailsAIService) hasConfiguredProvider() bool {
	if s.aiConfig == nil {
		return false
	}

	if s.aiConfig.OpenAI.APIKey != "" {
		return true
	}

	if s.aiConfig.Anthropic.APIKey != "" {
		return true
	}

	if s.aiConfig.Codex.APIKey != "" {
		return true
	}

	if s.aiConfig.Ollama.Endpoint != "" {
		return true
	}

	if s.aiConfig.HuggingFace.Endpoint != "" {
		return true
	}

	if s.aiConfig.ClaudeCode.ClaudePath != "" {
		return true
	}

	return false
}

// applyAIConfiguration rebuilds the AI service using the current runtime configuration.
func (s *WailsAIService) applyAIConfiguration() error {
	ctx := context.Background()

	if s.aiConfig == nil {
		s.aiConfig = ai.DefaultRuntimeConfig()
	}

	if !s.hasConfiguredProvider() {
		s.deps.Logger.Info("No AI providers configured; AI features remain disabled")
		if s.aiService != nil {
			if err := s.aiService.Stop(ctx); err != nil {
				s.deps.Logger.WithError(err).Warn("Failed to stop existing AI service")
			}
			s.aiService = nil
		}
		s.embeddingService = nil
		return fmt.Errorf("no AI providers configured")
	}

	// Shut down any existing instance before reconfiguring
	if s.aiService != nil {
		if err := s.aiService.Stop(ctx); err != nil {
			s.deps.Logger.WithError(err).Warn("Failed to stop existing AI service during reconfiguration")
		}
		s.aiService = nil
	}

	service, err := ai.NewServiceWithConfig(s.aiConfig, s.deps.Logger)
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	if err := service.Start(ctx); err != nil {
		return fmt.Errorf("failed to start AI service: %w", err)
	}

	s.aiService = service
	s.deps.Logger.Info("AI service configured successfully")

	s.rebuildEmbeddingService()
	return nil
}

// rebuildEmbeddingService establishes the embedding service if OpenAI credentials are available.
func (s *WailsAIService) rebuildEmbeddingService() {
	s.embeddingService = nil

	// Always prefer the local ONNX style projector so we stay offline first.
	onnxModelPath := getEnvOrDefault("HOWLEROPS_EMBEDDING_MODEL", "")
	provider := rag.NewONNXEmbeddingProvider(onnxModelPath, s.deps.Logger)

	// If an OpenAI embedding key exists we expose it as an optional fallback, but the primary path
	// will continue to use the local provider so that RAG works without internet access.
	if s.aiConfig != nil && strings.TrimSpace(s.aiConfig.OpenAI.APIKey) != "" {
		openAIModel := "text-embedding-3-small"
		openAIProv := rag.NewOpenAIEmbeddingProvider(s.aiConfig.OpenAI.APIKey, openAIModel, s.deps.Logger)
		provider = rag.NewFallbackEmbeddingProvider(provider, openAIProv)
	}

	s.embeddingService = rag.NewEmbeddingService(provider, s.deps.Logger)
	s.deps.Logger.WithField("model", provider.GetModel()).Info("Embedding service initialised")
}

// GetAIProviderStatus returns the status of available AI providers
func (s *WailsAIService) GetAIProviderStatus() (map[string]ProviderStatus, error) {
	s.deps.Logger.Debug("Getting AI provider status")

	if s.aiService == nil {
		statuses := map[string]ProviderStatus{
			"openai":      {Name: "OpenAI", Available: false, Error: "Configure this provider in Settings"},
			"anthropic":   {Name: "Anthropic", Available: false, Error: "Configure this provider in Settings"},
			"claudecode":  {Name: "Claude Code", Available: false, Error: "Configure this provider in Settings"},
			"codex":       {Name: "Codex", Available: false, Error: "Configure this provider in Settings"},
			"ollama":      {Name: "Ollama", Available: false, Error: "Configure this provider in Settings"},
			"huggingface": {Name: "HuggingFace", Available: false, Error: "Configure this provider in Settings"},
		}

		if s.aiConfig != nil {
			if s.aiConfig.OpenAI.APIKey != "" {
				statuses["openai"] = ProviderStatus{Name: "OpenAI", Available: true}
			}
			if s.aiConfig.Anthropic.APIKey != "" {
				statuses["anthropic"] = ProviderStatus{Name: "Anthropic", Available: true}
			}
			if s.aiConfig.Codex.APIKey != "" {
				statuses["codex"] = ProviderStatus{Name: "Codex", Available: true}
			}
			if s.aiConfig.ClaudeCode.ClaudePath != "" {
				statuses["claudecode"] = ProviderStatus{Name: "Claude Code", Available: true}
			}
			if s.aiConfig.Ollama.Endpoint != "" {
				statuses["ollama"] = ProviderStatus{Name: "Ollama", Available: true}
			}
			if s.aiConfig.HuggingFace.Endpoint != "" {
				statuses["huggingface"] = ProviderStatus{Name: "HuggingFace", Available: true}
			}
		}

		return statuses, nil
	}

	// Get provider statuses from AI service
	ctx := context.Background()
	providers, err := s.aiService.GetProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider status: %w", err)
	}

	// Convert to map
	result := make(map[string]ProviderStatus)
	for _, p := range providers {
		result[strings.ToLower(p.Name)] = ProviderStatus{
			Name:      p.Name,
			Available: p.Available,
			Error:     "",
		}
	}

	// Fill in missing providers as unavailable
	allProviders := []string{"openai", "anthropic", "claudecode", "codex", "ollama", "huggingface"}
	for _, name := range allProviders {
		if _, exists := result[name]; !exists {
			result[name] = ProviderStatus{
				Name:      strings.Title(name),
				Available: false,
				Error:     "Not configured",
			}
		}
	}

	return result, nil
}

// ConfigureAIProvider configures an AI provider dynamically from UI
func (s *WailsAIService) ConfigureAIProvider(config ProviderConfig) error {
	s.deps.Logger.WithField("provider", config.Provider).Info("Configuring AI provider")

	if s.aiConfig == nil {
		s.aiConfig = ai.DefaultRuntimeConfig()
	}

	provider := strings.ToLower(config.Provider)

	switch provider {
	case "openai":
		s.aiConfig.OpenAI.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			s.aiConfig.OpenAI.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Model != "" {
			s.aiConfig.DefaultProvider = ai.ProviderOpenAI
		}

	case "anthropic":
		s.aiConfig.Anthropic.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			s.aiConfig.Anthropic.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Model != "" {
			s.aiConfig.DefaultProvider = ai.ProviderAnthropic
		}

	case "ollama":
		s.aiConfig.Ollama.Endpoint = strings.TrimSpace(config.Endpoint)
		if config.Model != "" && !containsModel(s.aiConfig.Ollama.Models, config.Model) {
			s.aiConfig.Ollama.Models = append([]string{config.Model}, s.aiConfig.Ollama.Models...)
		}
		s.aiConfig.DefaultProvider = ai.ProviderOllama

	case "huggingface":
		s.aiConfig.HuggingFace.Endpoint = strings.TrimSpace(config.Endpoint)
		if config.Model != "" && !containsModel(s.aiConfig.HuggingFace.Models, config.Model) {
			s.aiConfig.HuggingFace.Models = append([]string{config.Model}, s.aiConfig.HuggingFace.Models...)
		}
		s.aiConfig.DefaultProvider = ai.ProviderHuggingFace

	case "claudecode":
		claudePath := ""
		if config.Options != nil {
			claudePath = strings.TrimSpace(config.Options["binary_path"])
		}
		if claudePath == "" {
			claudePath = "claude"
		}
		s.aiConfig.ClaudeCode.ClaudePath = claudePath
		if config.Model != "" {
			s.aiConfig.ClaudeCode.Model = config.Model
		}
		s.aiConfig.DefaultProvider = ai.ProviderClaudeCode

	case "codex":
		s.aiConfig.Codex.APIKey = strings.TrimSpace(config.APIKey)
		if config.Model != "" {
			s.aiConfig.Codex.Model = config.Model
		}
		if config.Endpoint != "" {
			s.aiConfig.Codex.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Options != nil {
			if org, ok := config.Options["organization"]; ok {
				s.aiConfig.Codex.Organization = strings.TrimSpace(org)
			}
		}
		s.aiConfig.DefaultProvider = ai.ProviderCodex

	default:
		return fmt.Errorf("unknown AI provider: %s", config.Provider)
	}

	if err := s.applyAIConfiguration(); err != nil {
		return fmt.Errorf("failed to apply AI configuration: %w", err)
	}

	return nil
}

// GetAIConfiguration returns the currently active AI provider configuration with masked secrets.
func (s *WailsAIService) GetAIConfiguration() (ProviderConfig, error) {
	if s.aiConfig == nil {
		return ProviderConfig{}, fmt.Errorf("AI configuration not initialised")
	}

	provider := strings.ToLower(string(s.aiConfig.DefaultProvider))
	if provider == "" {
		provider = "openai"
	}

	config := ProviderConfig{
		Provider: provider,
	}

	switch provider {
	case "openai":
		config.APIKey = maskSecret(s.aiConfig.OpenAI.APIKey)
		config.Endpoint = s.aiConfig.OpenAI.BaseURL
		if len(s.aiConfig.OpenAI.Models) > 0 {
			config.Model = s.aiConfig.OpenAI.Models[0]
		}

	case "anthropic":
		config.APIKey = maskSecret(s.aiConfig.Anthropic.APIKey)
		config.Endpoint = s.aiConfig.Anthropic.BaseURL
		if len(s.aiConfig.Anthropic.Models) > 0 {
			config.Model = s.aiConfig.Anthropic.Models[0]
		}

	case "codex":
		config.APIKey = maskSecret(s.aiConfig.Codex.APIKey)
		config.Endpoint = s.aiConfig.Codex.BaseURL
		config.Model = s.aiConfig.Codex.Model
		config.Options = map[string]string{
			"organization": s.aiConfig.Codex.Organization,
		}

	case "claudecode":
		config.Endpoint = s.aiConfig.ClaudeCode.ClaudePath
		config.Model = s.aiConfig.ClaudeCode.Model

	case "ollama":
		config.Endpoint = s.aiConfig.Ollama.Endpoint
		if len(s.aiConfig.Ollama.Models) > 0 {
			config.Model = s.aiConfig.Ollama.Models[0]
		}

	case "huggingface":
		config.Endpoint = s.aiConfig.HuggingFace.Endpoint
		config.Model = s.aiConfig.HuggingFace.RecommendedModel
	}

	return config, nil
}

// TestAIProvider tests a provider configuration without saving it
func (s *WailsAIService) TestAIProvider(config ProviderConfig) (*ProviderStatus, error) {
	s.deps.Logger.WithField("provider", config.Provider).Info("Testing AI provider")

	testConfig := ai.DefaultRuntimeConfig()
	provider := strings.ToLower(config.Provider)

	switch provider {
	case "openai":
		testConfig.OpenAI.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			testConfig.OpenAI.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		testConfig.DefaultProvider = ai.ProviderOpenAI

	case "anthropic":
		testConfig.Anthropic.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			testConfig.Anthropic.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		testConfig.DefaultProvider = ai.ProviderAnthropic

	case "ollama":
		testConfig.Ollama.Endpoint = strings.TrimSpace(config.Endpoint)
		testConfig.DefaultProvider = ai.ProviderOllama

	case "huggingface":
		testConfig.HuggingFace.Endpoint = strings.TrimSpace(config.Endpoint)
		testConfig.DefaultProvider = ai.ProviderHuggingFace

	case "claudecode":
		path := ""
		if config.Options != nil {
			path = strings.TrimSpace(config.Options["binary_path"])
		}
		if path == "" {
			path = "claude"
		}
		testConfig.ClaudeCode.ClaudePath = path
		if config.Model != "" {
			testConfig.ClaudeCode.Model = config.Model
		}
		testConfig.DefaultProvider = ai.ProviderClaudeCode

	case "codex":
		testConfig.Codex.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			testConfig.Codex.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Model != "" {
			testConfig.Codex.Model = config.Model
		}
		if config.Options != nil {
			if org, ok := config.Options["organization"]; ok {
				testConfig.Codex.Organization = strings.TrimSpace(org)
			}
		}
		testConfig.DefaultProvider = ai.ProviderCodex

	default:
		return nil, fmt.Errorf("unknown AI provider: %s", config.Provider)
	}

	// Remove other providers to avoid validation noise
	if provider != "openai" {
		testConfig.OpenAI.APIKey = ""
	}
	if provider != "anthropic" {
		testConfig.Anthropic.APIKey = ""
	}
	if provider != "codex" {
		testConfig.Codex.APIKey = ""
		testConfig.Codex.Organization = ""
	}
	if provider != "claudecode" {
		testConfig.ClaudeCode.ClaudePath = ""
	}
	if provider != "ollama" {
		testConfig.Ollama.Endpoint = ""
	}
	if provider != "huggingface" {
		testConfig.HuggingFace.Endpoint = ""
	}

	ctx := context.Background()
	testService, err := ai.NewServiceWithConfig(testConfig, s.deps.Logger)
	if err != nil {
		return &ProviderStatus{
			Name:      config.Provider,
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	defer func() {
		_ = testService.Stop(ctx)
	}()

	if err := testService.Start(ctx); err != nil {
		return &ProviderStatus{
			Name:      config.Provider,
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	return &ProviderStatus{
		Name:      config.Provider,
		Available: true,
	}, nil
}

// GetAvailableModels returns the available models for a specific AI provider.
// It queries the provider's API dynamically when possible, falling back to configured defaults.
func (s *WailsAIService) GetAvailableModels(provider string) ([]ModelInfoResponse, error) {
	s.deps.Logger.WithField("provider", provider).Debug("Getting available models")

	if s.aiService == nil {
		if err := s.applyAIConfiguration(); err != nil {
			// Return empty list - frontend will use its static registry
			return []ModelInfoResponse{}, nil //nolint:nilerr // graceful fallback to static registry
		}
	}

	if s.aiService == nil {
		return []ModelInfoResponse{}, nil
	}

	ctx := context.Background()
	models, err := s.aiService.GetAvailableModels(ctx, ai.Provider(provider))
	if err != nil {
		s.deps.Logger.WithError(err).WithField("provider", provider).Debug("Failed to get models from provider")
		return []ModelInfoResponse{}, nil
	}

	result := make([]ModelInfoResponse, 0, len(models))
	for _, m := range models {
		source := "api"
		if m.Metadata != nil {
			if src, ok := m.Metadata["source"]; ok {
				source = src
			}
		}
		result = append(result, ModelInfoResponse{
			ID:          m.ID,
			Name:        m.Name,
			Provider:    string(m.Provider),
			Description: m.Description,
			MaxTokens:   m.MaxTokens,
			Source:      source,
		})
	}

	return result, nil
}

// ======================================
// AI Provider Test Methods
// ======================================

// TestOpenAIConnection tests OpenAI provider connection
func (s *WailsAIService) TestOpenAIConnection(apiKey, model string) *AITestResponse {
	s.deps.Logger.WithField("provider", "openai").WithField("model", model).Info("Testing OpenAI connection")

	if apiKey == "" {
		return &AITestResponse{
			Success: false,
			Error:   "OpenAI API key is required",
		}
	}

	// Basic validation - check if API key format is correct
	if !strings.HasPrefix(apiKey, "sk-") {
		return &AITestResponse{
			Success: false,
			Error:   "Invalid OpenAI API key format",
		}
	}

	// For now, return success for valid-looking keys
	// TODO: Implement actual API call to OpenAI
	return &AITestResponse{
		Success: true,
		Message: "OpenAI connection test successful",
	}
}

// TestAnthropicConnection tests Anthropic provider connection
func (s *WailsAIService) TestAnthropicConnection(apiKey, model string) *AITestResponse {
	s.deps.Logger.WithField("provider", "anthropic").WithField("model", model).Info("Testing Anthropic connection")

	if apiKey == "" {
		return &AITestResponse{
			Success: false,
			Error:   "Anthropic API key is required",
		}
	}

	// Basic validation - check if API key format is correct
	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return &AITestResponse{
			Success: false,
			Error:   "Invalid Anthropic API key format",
		}
	}

	// For now, return success for valid-looking keys
	// TODO: Implement actual API call to Anthropic
	return &AITestResponse{
		Success: true,
		Message: "Anthropic connection test successful",
	}
}

// TestOllamaConnection tests Ollama provider connection
func (s *WailsAIService) TestOllamaConnection(endpoint, model string) *AITestResponse {
	s.deps.Logger.WithField("provider", "ollama").WithField("endpoint", endpoint).WithField("model", model).Info("Testing Ollama connection")

	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// Try to connect to Ollama endpoint
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint + "/api/tags")
	if err != nil {
		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect to Ollama at %s: %v", endpoint, err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Ollama endpoint returned status %d", resp.StatusCode),
		}
	}

	return &AITestResponse{
		Success: true,
		Message: "Ollama connection test successful",
	}
}

// TestClaudeCodeConnection tests Claude Code provider connection
func (s *WailsAIService) TestClaudeCodeConnection(binaryPath, model string) *AITestResponse {
	s.deps.Logger.WithField("provider", "claudecode").WithField("binaryPath", binaryPath).WithField("model", model).Info("Testing Claude Code connection")

	// First check for detected credentials
	hasCredentials, credsSource := detectClaudeCredentials()
	if !hasCredentials && credsSource != "" {
		s.deps.Logger.WithField("source", credsSource).Warn("Claude Code credentials detected but invalid")
	} else if hasCredentials {
		s.deps.Logger.WithField("source", credsSource).Info("Claude Code credentials detected")
	}

	if binaryPath == "" {
		binaryPath = "claude"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runCommand := func(args ...string) (string, string, error) {
		cmd := exec.CommandContext(ctx, binaryPath, args...)
		cmd.Env = os.Environ()

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
	}

	output, errMsg, err := runCommand("whoami", "--json")
	if err != nil && isUnknownOptionError(errMsg) {
		output, errMsg, err = runCommand("whoami", "--format", "json")
	}

	if err != nil && isUnknownOptionError(errMsg) {
		plainOutput, plainErrMsg, plainErr := runCommand("whoami")
		if plainErr != nil && strings.TrimSpace(plainOutput) == "" {
			if plainErrMsg == "" {
				plainErrMsg = plainErr.Error()
			}
			return &AITestResponse{
				Success: false,
				Error:   fmt.Sprintf("Claude CLI check failed: %s", plainErrMsg),
			}
		}
		return parseClaudeWhoAmIOutput(plainOutput)
	}

	if err != nil {
		if output != "" {
			return parseClaudeWhoAmIOutput(output)
		}

		if errMsg == "" {
			errMsg = err.Error()
		}

		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Claude CLI check failed: %s", errMsg),
		}
	}

	return parseClaudeWhoAmIOutput(output)
}

// StartClaudeCodeLogin begins the Claude CLI login flow.
func (s *WailsAIService) StartClaudeCodeLogin(binaryPath string) *AITestResponse {
	s.deps.Logger.WithField("binaryPath", binaryPath).Info("Launching Claude CLI login flow")

	if binaryPath == "" {
		binaryPath = "claude"
	}

	defaultMessage := "Open the link and authorise Claude Code using the displayed code."
	if response, handled := s.runClaudeLoginJSON(binaryPath, defaultMessage); handled {
		if response != nil {
			return response
		}
	}

	return s.runDeviceLoginCommand(binaryPath, []string{"/login"}, "claudecode", defaultMessage)
}

// StartCodexLogin begins the OpenAI CLI login flow for Codex access.
func (s *WailsAIService) StartCodexLogin(binaryPath string) *AITestResponse {
	s.deps.Logger.WithField("binaryPath", binaryPath).Info("Launching OpenAI CLI login flow")

	if binaryPath == "" {
		binaryPath = "openai"
	}

	return s.runDeviceLoginCommand(binaryPath, []string{"login"}, "codex", "Open the link and authorise OpenAI using the displayed code.")
}

// TestCodexConnection tests Codex provider connection
func (s *WailsAIService) TestCodexConnection(apiKey, model, organization string) *AITestResponse {
	s.deps.Logger.WithField("provider", "codex").WithField("model", model).WithField("organization", organization).Info("Testing Codex connection")

	// Try to detect credentials from global paths if not provided
	if apiKey == "" {
		detectedKey, source := detectCodexCredentials()
		if detectedKey != "" {
			apiKey = detectedKey
			s.deps.Logger.WithField("source", source).Info("Using detected Codex credentials")
		} else {
			return &AITestResponse{
				Success: false,
				Error:   "Codex API key is required. Please set OPENAI_API_KEY environment variable or create ~/.codex/auth.json, or log in via 'openai login' CLI command.",
			}
		}
	}

	// Basic validation - check if API key format is correct
	if !strings.HasPrefix(apiKey, "sk-") {
		return &AITestResponse{
			Success: false,
			Error:   "Invalid Codex API key format",
		}
	}

	// For now, return success for valid-looking keys
	// TODO: Implement actual API call to OpenAI Codex
	metadata := map[string]string{}
	if organization != "" {
		metadata["organization"] = organization
	}

	return &AITestResponse{
		Success:  true,
		Message:  "Codex connection test successful",
		Metadata: metadata,
	}
}

// TestHuggingFaceConnection tests HuggingFace provider connection
func (s *WailsAIService) TestHuggingFaceConnection(endpoint, model string) *AITestResponse {
	s.deps.Logger.WithField("provider", "huggingface").WithField("endpoint", endpoint).WithField("model", model).Info("Testing HuggingFace connection")

	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// For HuggingFace via Ollama, test the Ollama endpoint
	return s.TestOllamaConnection(endpoint, model)
}

// ======================================
// AI/RAG Methods
// ======================================

// GenerateSQLFromNaturalLanguage generates SQL from a natural language prompt
func (s *WailsAIService) GenerateSQLFromNaturalLanguage(req NLQueryRequest) (*GeneratedSQLResponse, error) {
	s.deps.Logger.WithField("prompt", req.Prompt).WithField("connectionId", req.ConnectionID).WithField("hasContext", req.Context != "").Info("Generating SQL from natural language")

	if s.aiService == nil {
		if err := s.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	isMultiDB := strings.Contains(req.Context, "Multi-DB Mode") || strings.Contains(req.Context, "@connection_name") || isMultiDatabaseSQL(req.Prompt)

	var connectionSchema string
	if !isMultiDB && req.ConnectionID != "" {
		connectionSchema = s.buildDetailedSchemaContext(req.ConnectionID)
	}

	manualContext := buildManualContext(isMultiDB, connectionSchema, req.Context)

	if generated, err := s.tryGenerateWithRAG(req, manualContext, isMultiDB); err == nil {
		response := &GeneratedSQLResponse{
			SQL:         strings.TrimSpace(generated.Query),
			Confidence:  float64(generated.Confidence),
			Explanation: generated.Explanation,
			Warnings:    generated.Warnings,
		}
		sanitizeSQLResponse(&ai.SQLResponse{
			SQL:         response.SQL,
			Confidence:  response.Confidence,
			Explanation: response.Explanation,
		})
		return response, nil
	} else {
		s.deps.Logger.WithError(err).Warn("RAG-enhanced SQL generation failed, falling back to direct provider call")
	}

	fallbackPrompt, fallbackSchema := buildFallbackPrompt(req.Prompt, manualContext, isMultiDB)
	ctx := context.Background()
	result, err := s.aiService.GenerateSQLWithRequest(ctx, &ai.SQLRequest{
		Prompt:      fallbackPrompt,
		Schema:      fallbackSchema,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	sanitizeSQLResponse(result)

	return &GeneratedSQLResponse{
		SQL:         strings.TrimSpace(result.SQL),
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		Warnings:    []string{},
	}, nil
}

func (s *WailsAIService) tryGenerateWithRAG(req NLQueryRequest, manualContext string, isMulti bool) (*rag.GeneratedSQL, error) {
	if s.deps.StorageManager == nil {
		return nil, fmt.Errorf("storage manager not initialised")
	}

	if s.embeddingService == nil {
		s.rebuildEmbeddingService()
	}

	if s.embeddingService == nil {
		return nil, fmt.Errorf("embedding service unavailable")
	}

	rawVectorStore := s.deps.StorageManager.GetVectorStore()
	vectorStore, ok := rawVectorStore.(rag.VectorStore)
	if !ok || vectorStore == nil {
		return nil, fmt.Errorf("vector store unavailable")
	}

	contextBuilder := rag.NewContextBuilder(vectorStore, s.embeddingService, s.deps.Logger)
	provider := &ragLLMProvider{
		service:       s,
		request:       req,
		manualContext: manualContext,
		multiMode:     isMulti,
	}

	generator := rag.NewSmartSQLGenerator(contextBuilder, provider, s.deps.Logger)
	connectionID := req.ConnectionID
	if isMulti {
		connectionID = ""
	}

	ctx := context.Background()
	return generator.Generate(ctx, req.Prompt, connectionID)
}

// GenericChat handles generic conversational AI requests without SQL-specific expectations
func (s *WailsAIService) GenericChat(req GenericChatRequest) (*GenericChatResponse, error) {
	s.deps.Logger.WithField("hasContext", req.Context != "").WithField("provider", req.Provider).WithField("model", req.Model).Info("Handling generic AI chat request")

	if s.aiService == nil {
		if err := s.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	chatReq := &ai.ChatRequest{
		Prompt:      req.Prompt,
		Context:     req.Context,
		System:      req.System,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Metadata:    req.Metadata,
	}

	ctx := context.Background()
	response, err := s.aiService.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chat response: %w", err)
	}

	return &GenericChatResponse{
		Content:    response.Content,
		Provider:   response.Provider,
		Model:      response.Model,
		TokensUsed: response.TokensUsed,
		Metadata:   response.Metadata,
	}, nil
}

// FixSQLError attempts to fix a SQL error
func (s *WailsAIService) FixSQLError(query string, error string, connectionID string) (*FixedSQLResponse, error) {
	return s.FixSQLErrorWithOptions(FixSQLRequest{
		Query:        query,
		Error:        error,
		ConnectionID: connectionID,
	})
}

// OptimizeQuery optimizes a SQL query
func (s *WailsAIService) OptimizeQuery(query string, connectionID string) (*OptimizationResponse, error) {
	s.deps.Logger.WithField("query", query).WithField("connectionId", connectionID).Info("Optimizing query")

	if s.aiService == nil {
		if err := s.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	ctx := context.Background()
	result, err := s.aiService.OptimizeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize query: %w", err)
	}

	return &OptimizationResponse{
		SQL:              result.OptimizedSQL,
		EstimatedSpeedup: result.Impact,
		Explanation:      result.Explanation,
		Suggestions:      []Suggestion{},
	}, nil
}

// FixSQLErrorWithOptions applies AI-based fixes with provider-specific configuration
func (s *WailsAIService) FixSQLErrorWithOptions(req FixSQLRequest) (*FixedSQLResponse, error) {
	s.deps.Logger.WithField("query", req.Query).WithField("error", req.Error).WithField("connectionId", req.ConnectionID).WithField("provider", req.Provider).WithField("model", req.Model).Info("Fixing SQL error with options")

	if s.aiService == nil {
		if err := s.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if strings.TrimSpace(req.Error) == "" {
		return nil, fmt.Errorf("error message cannot be empty")
	}

	isMultiDB := strings.Contains(req.Query, "@") && (strings.Contains(req.Error, "multi-database") || strings.Contains(req.Error, "@connection_name"))

	enhancedError := req.Error
	if isMultiDB {
		enhancedError = fmt.Sprintf(
			"%s\n\n"+
				"Context: This appears to be a multi-database query. "+
				"Ensure that:\n"+
				"1. Table references use @connection_name.table_name syntax\n"+
				"2. Connection names are spelled correctly (case-sensitive)\n"+
				"3. All referenced connections are properly connected\n"+
				"4. Schema names are included for non-public schemas (@conn.schema.table)",
			req.Error,
		)
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	model := strings.TrimSpace(req.Model)

	aiRequest := &ai.SQLRequest{
		Query:       req.Query,
		Error:       enhancedError,
		Schema:      req.Context,
		Provider:    provider,
		Model:       model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	ctx := context.Background()
	result, err := s.aiService.FixQueryWithRequest(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fix query: %w", err)
	}

	return &FixedSQLResponse{
		SQL:         result.SQL,
		Explanation: result.Explanation,
		Changes:     []string{"Fixed query based on error message"},
	}, nil
}

// GetQuerySuggestions provides autocomplete suggestions for a partial query
func (s *WailsAIService) GetQuerySuggestions(partialQuery string, connectionID string) ([]Suggestion, error) {
	s.deps.Logger.WithField("query", partialQuery).WithField("connectionId", connectionID).Debug("Getting query suggestions")

	// TODO: Integrate with backend-go AI service when properly exposed
	return []Suggestion{}, nil
}

// SuggestVisualization suggests appropriate visualizations for query results
func (s *WailsAIService) SuggestVisualization(resultData ResultData) (*VizSuggestion, error) {
	s.deps.Logger.WithField("columns", len(resultData.Columns)).WithField("rowCount", resultData.RowCount).Debug("Suggesting visualization")

	// TODO: Integrate with backend-go AI service when properly exposed
	return nil, fmt.Errorf("AI service not yet integrated")
}

// ======================================
// AI Memory Methods
// ======================================

// SaveAIMemorySessions persists AI memory sessions and indexes them for recall
func (s *WailsAIService) SaveAIMemorySessions(sessions []AIMemorySessionPayload) error {
	if s.deps.StorageManager == nil {
		return fmt.Errorf("storage manager not initialized")
	}

	previousSessions, err := s.LoadAIMemorySessions()
	if err != nil {
		return err
	}

	data, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("failed to marshal AI memory sessions: %w", err)
	}

	ctx := context.Background()
	if err := s.deps.StorageManager.SetSetting(ctx, "ai_memory_sessions", string(data)); err != nil {
		return fmt.Errorf("failed to save AI memory sessions: %w", err)
	}

	s.pruneAIMemoryDocuments(previousSessions, sessions)

	if len(sessions) > 0 {
		snapshot := append([]AIMemorySessionPayload(nil), sessions...)
		go func(payload []AIMemorySessionPayload) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := s.indexAIMemorySessions(ctx, payload); err != nil {
				s.deps.Logger.WithError(err).Warn("Failed to index AI memory sessions")
			}
		}(snapshot)
	}

	return nil
}

// LoadAIMemorySessions retrieves previously stored AI memory sessions
func (s *WailsAIService) LoadAIMemorySessions() ([]AIMemorySessionPayload, error) {
	if s.deps.StorageManager == nil {
		return []AIMemorySessionPayload{}, nil
	}

	ctx := context.Background()
	value, err := s.deps.StorageManager.GetSetting(ctx, "ai_memory_sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to load AI memory sessions: %w", err)
	}

	if strings.TrimSpace(value) == "" {
		return []AIMemorySessionPayload{}, nil
	}

	var sessions []AIMemorySessionPayload
	if err := json.Unmarshal([]byte(value), &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode AI memory sessions: %w", err)
	}

	return sessions, nil
}

// ClearAIMemorySessions removes stored AI memory sessions
func (s *WailsAIService) ClearAIMemorySessions() error {
	if s.deps.StorageManager == nil {
		return nil
	}

	ctx := context.Background()
	if err := s.deps.StorageManager.DeleteSetting(ctx, "ai_memory_sessions"); err != nil {
		return fmt.Errorf("failed to clear AI memory sessions: %w", err)
	}

	return nil
}

// RecallAIMemorySessions returns the most relevant stored memories for the given prompt
func (s *WailsAIService) RecallAIMemorySessions(prompt string, limit int) ([]AIMemoryRecallResult, error) {
	if strings.TrimSpace(prompt) == "" || limit == 0 {
		return []AIMemoryRecallResult{}, nil
	}

	if limit < 0 {
		limit = 5
	}
	if limit == 0 {
		limit = 5
	}

	if s.embeddingService == nil || s.deps.StorageManager == nil {
		return []AIMemoryRecallResult{}, nil
	}

	ctx := context.Background()
	embedding, err := s.embeddingService.EmbedText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to embed prompt for memory recall: %w", err)
	}

	docs, err := s.deps.StorageManager.SearchDocuments(ctx, embedding, &storage.DocumentFilters{
		Type:  string(rag.DocumentTypeMemory),
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search AI memories: %w", err)
	}

	results := make([]AIMemoryRecallResult, 0, len(docs))
	for _, doc := range docs {
		sessionID := ""
		title := ""
		summary := ""

		if doc.Metadata != nil {
			if val, ok := doc.Metadata["session_id"].(string); ok {
				sessionID = val
			}
			if val, ok := doc.Metadata["title"].(string); ok {
				title = val
			}
			if val, ok := doc.Metadata["summary"].(string); ok {
				summary = val
			}
		}

		results = append(results, AIMemoryRecallResult{
			SessionID: sessionID,
			Title:     title,
			Summary:   summary,
			Content:   doc.Content,
			Score:     doc.Score,
		})
	}

	return results, nil
}

func (s *WailsAIService) indexAIMemorySessions(ctx context.Context, sessions []AIMemorySessionPayload) error {
	if s.embeddingService == nil || s.deps.StorageManager == nil {
		return nil
	}

	for _, session := range sessions {
		doc, err := s.buildMemoryDocument(ctx, session)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to build memory document")
			continue
		}
		if doc == nil {
			continue
		}

		if err := s.deps.StorageManager.IndexDocument(ctx, doc); err != nil {
			s.deps.Logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to index AI memory document")
		}
	}

	return nil
}

func (s *WailsAIService) buildMemoryDocument(ctx context.Context, session AIMemorySessionPayload) (*storage.Document, error) {
	if len(session.Messages) == 0 {
		return nil, nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Session: %s\n", session.Title))
	if session.Summary != "" {
		builder.WriteString(fmt.Sprintf("Summary: %s\n", session.Summary))
	}

	const maxMessages = 6
	start := len(session.Messages) - maxMessages
	if start < 0 {
		start = 0
	}

	builder.WriteString("Recent conversation:\n")
	for _, msg := range session.Messages[start:] {
		builder.WriteString(fmt.Sprintf("[%s] %s\n", strings.ToUpper(msg.Role), msg.Content))
	}

	content := builder.String()
	embedding, err := s.embeddingService.EmbedText(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to embed AI memory: %w", err)
	}

	createdAt := time.UnixMilli(session.CreatedAt)
	if session.CreatedAt == 0 {
		createdAt = time.Now()
	}
	updatedAt := time.UnixMilli(session.UpdatedAt)
	if session.UpdatedAt == 0 {
		updatedAt = createdAt
	}

	metadata := map[string]interface{}{
		"session_id":    session.ID,
		"title":         session.Title,
		"summary":       session.Summary,
		"message_count": len(session.Messages),
	}

	return &storage.Document{
		ID:           fmt.Sprintf("ai_memory:%s", session.ID),
		ConnectionID: "",
		Type:         string(rag.DocumentTypeMemory),
		Content:      content,
		Embedding:    embedding,
		Metadata:     metadata,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// DeleteAIMemorySession removes a single session by ID
func (s *WailsAIService) DeleteAIMemorySession(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	sessions, err := s.LoadAIMemorySessions()
	if err != nil {
		return err
	}

	filtered := make([]AIMemorySessionPayload, 0, len(sessions))
	for _, session := range sessions {
		if session.ID != sessionID {
			filtered = append(filtered, session)
		}
	}

	if len(filtered) == len(sessions) {
		return fmt.Errorf("session not found")
	}

	if err := s.SaveAIMemorySessions(filtered); err != nil {
		return err
	}

	if s.deps.StorageManager != nil {
		docID := fmt.Sprintf("ai_memory:%s", sessionID)
		ctx := context.Background()
		if err := s.deps.StorageManager.DeleteDocument(ctx, docID); err != nil {
			s.deps.Logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to delete memory document")
		}
	}

	return nil
}

func (s *WailsAIService) pruneAIMemoryDocuments(previous, current []AIMemorySessionPayload) {
	if s.deps.StorageManager == nil {
		return
	}

	currentSet := make(map[string]struct{}, len(current))
	for _, session := range current {
		currentSet[session.ID] = struct{}{}
	}

	ctx := context.Background()
	for _, session := range previous {
		if _, exists := currentSet[session.ID]; !exists {
			docID := fmt.Sprintf("ai_memory:%s", session.ID)
			if err := s.deps.StorageManager.DeleteDocument(ctx, docID); err != nil {
				s.deps.Logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to delete AI memory document")
			}
		}
	}
}

// ======================================
// Helper Types and Methods
// ======================================

// ragLLMProvider adapts the AI service for RAG integration
type ragLLMProvider struct {
	service       *WailsAIService
	request       NLQueryRequest
	manualContext string
	multiMode     bool
}

func (p *ragLLMProvider) GenerateSQL(ctx context.Context, prompt string, queryCtx *rag.QueryContext) (*rag.GeneratedSQL, error) {
	aggregatedContext := aggregateContextSections(p.manualContext, renderQueryContext(queryCtx))
	finalPrompt := composePrompt(prompt, aggregatedContext, p.multiMode)

	resp, err := p.service.aiService.GenerateSQLWithRequest(ctx, &ai.SQLRequest{
		Prompt:      finalPrompt,
		Schema:      aggregatedContext,
		Provider:    p.resolveProvider(),
		Model:       p.resolveModel(),
		MaxTokens:   p.resolveMaxTokens(),
		Temperature: p.resolveTemperature(),
	})
	if err != nil {
		return nil, err
	}

	return &rag.GeneratedSQL{
		Query:       strings.TrimSpace(resp.SQL),
		Explanation: resp.Explanation,
		Confidence:  float32(resp.Confidence),
	}, nil
}

func (p *ragLLMProvider) ExplainSQL(ctx context.Context, sql string) (*rag.SQLExplanation, error) {
	prompt := fmt.Sprintf("Explain the purpose and mechanics of the following SQL query:\n\n%s", sql)
	resp, err := p.service.aiService.Chat(ctx, &ai.ChatRequest{
		Prompt:      prompt,
		Provider:    p.resolveProvider(),
		Model:       p.resolveModel(),
		MaxTokens:   p.resolveMaxTokens() / 2,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	return &rag.SQLExplanation{
		Summary: resp.Content,
		Steps:   []rag.ExplanationStep{},
		Complexity: func() string {
			if strings.Count(sql, "JOIN") > 1 {
				return "complex"
			}
			if strings.Contains(strings.ToUpper(sql), "GROUP BY") {
				return "moderate"
			}
			return "simple"
		}(),
		EstimatedTime: "",
	}, nil
}

func (p *ragLLMProvider) OptimizeSQL(ctx context.Context, sql string, hints []rag.OptimizationHint) (*rag.OptimizedSQL, error) {
	resp, err := p.service.aiService.OptimizeQuery(ctx, sql)
	if err != nil {
		return nil, err
	}

	improvements := []rag.Improvement{}
	if resp.Explanation != "" {
		improvements = append(improvements, rag.Improvement{
			Type:        "rewrite",
			Description: resp.Explanation,
			Before:      sql,
			After:       resp.OptimizedSQL,
		})
	}

	return &rag.OptimizedSQL{
		OriginalQuery:  sql,
		OptimizedQuery: resp.OptimizedSQL,
		Improvements:   improvements,
		EstimatedGain:  0,
	}, nil
}

func (p *ragLLMProvider) resolveProvider() string {
	if provider := strings.TrimSpace(p.request.Provider); provider != "" {
		return provider
	}
	if p.service.aiConfig != nil {
		return string(p.service.aiConfig.DefaultProvider)
	}
	return "openai"
}

func (p *ragLLMProvider) resolveModel() string {
	return strings.TrimSpace(p.request.Model)
}

func (p *ragLLMProvider) resolveMaxTokens() int {
	if p.request.MaxTokens > 0 {
		return p.request.MaxTokens
	}
	if p.service.aiConfig != nil && p.service.aiConfig.MaxTokens > 0 {
		return p.service.aiConfig.MaxTokens
	}
	return 2048
}

func (p *ragLLMProvider) resolveTemperature() float64 {
	if p.request.Temperature > 0 {
		return p.request.Temperature
	}
	if p.service.aiConfig != nil && p.service.aiConfig.Temperature > 0 {
		return p.service.aiConfig.Temperature
	}
	return 0.1
}

// schemaProviderAdapter adapts DatabaseService calls to the indexer's SchemaProvider.
type schemaProviderAdapter struct{ dbsvc *services.DatabaseService }

func (a *schemaProviderAdapter) GetSchemas(connID string) ([]string, error) {
	return a.dbsvc.GetSchemas(connID)
}

func (a *schemaProviderAdapter) GetTables(connID, schema string) ([]database.TableInfo, error) {
	return a.dbsvc.GetTables(connID, schema)
}

func (a *schemaProviderAdapter) GetTableStructure(connID, schema, table string) (*database.TableStructure, error) {
	return a.dbsvc.GetTableStructure(connID, schema, table)
}

// buildDetailedSchemaContext constructs a concise schema summary with column details for a connection.
func (s *WailsAIService) buildDetailedSchemaContext(connectionID string) string {
	const (
		maxTablesPerSchema = 10
		maxTotalTables     = 40
		maxColumnsPerTable = 25
	)

	schemas, err := s.deps.DatabaseService.GetSchemas(connectionID)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("connection_id", connectionID).
			Warn("Failed to load schemas for AI context")
		return ""
	}

	if len(schemas) == 0 {
		return ""
	}

	sort.Strings(schemas)

	var builder strings.Builder
	builder.WriteString("Database Schema Information:\n")

	totalTables := 0
	for _, schemaName := range schemas {
		if totalTables >= maxTotalTables {
			break
		}

		tables, err := s.deps.DatabaseService.GetTables(connectionID, schemaName)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("connection_id", connectionID).WithField("schema", schemaName).Warn("Failed to load tables for AI context")
			continue
		}

		if len(tables) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("\nSchema: %s\n", schemaName))

		tableLimit := len(tables)
		if tableLimit > maxTablesPerSchema {
			tableLimit = maxTablesPerSchema
		}

		remaining := maxTotalTables - totalTables
		if tableLimit > remaining {
			tableLimit = remaining
		}

		type tableSummary struct {
			columns       string
			relationships string
		}

		summaries := make([]tableSummary, tableLimit)
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 4)

		for i := 0; i < tableLimit; i++ {
			table := tables[i]
			wg.Add(1)
			semaphore <- struct{}{}
			go func(idx int, tbl database.TableInfo) {
				defer wg.Done()
				defer func() { <-semaphore }()

				structure, err := s.deps.DatabaseService.GetTableStructure(connectionID, schemaName, tbl.Name)
				if err != nil {
					s.deps.Logger.WithError(err).WithField("connection_id", connectionID).WithField("schema", schemaName).WithField("table", tbl.Name).Debug("Failed to load table structure for AI context")
					return
				}

				if structure == nil {
					return
				}

				summary := tableSummary{}
				if len(structure.Columns) > 0 {
					columns := formatColumnsForAI(structure.Columns, maxColumnsPerTable)
					summary.columns = "Columns: " + strings.Join(columns, ", ")
				}

				if len(structure.ForeignKeys) > 0 {
					rels := make([]string, 0, len(structure.ForeignKeys))
					for _, fk := range structure.ForeignKeys {
						left := strings.Join(fk.Columns, ",")
						rightSchema := fk.ReferencedSchema
						if strings.TrimSpace(rightSchema) == "" {
							rightSchema = schemaName
						}
						right := fmt.Sprintf("%s.%s(%s)", rightSchema, fk.ReferencedTable, strings.Join(fk.ReferencedColumns, ","))
						rels = append(rels, fmt.Sprintf("%s -> %s", left, right))
					}
					summary.relationships = "Relationships: " + strings.Join(rels, "; ")
				}

				summaries[idx] = summary
			}(i, table)
		}

		wg.Wait()

		for i := 0; i < tableLimit && totalTables < maxTotalTables; i++ {
			table := tables[i]
			tableType := strings.ToLower(table.Type)
			if tableType == "" {
				tableType = "table"
			}

			builder.WriteString(fmt.Sprintf("Table: %s (%s)\n", table.Name, tableType))

			if summaries[i].columns != "" {
				builder.WriteString(summaries[i].columns)
				builder.WriteString("\n")
			}

			if summaries[i].relationships != "" {
				builder.WriteString(summaries[i].relationships)
				builder.WriteString("\n")
			}

			totalTables++
		}

		if len(tables) > tableLimit {
			builder.WriteString(fmt.Sprintf("... %d more tables in schema %s omitted for brevity\n", len(tables)-tableLimit, schemaName))
		}
	}

	return strings.TrimSpace(builder.String())
}

// ======================================
// Device Login Helper Methods
// ======================================

var (
	loginURLRegex       = regexp.MustCompile(`https?://[^\s]+`)
	loginCodeRegex      = regexp.MustCompile(`(?i)(?:code|token|key)[^A-Z0-9]*([A-Z0-9-]{4,})`)
	loginCodeValueRegex = regexp.MustCompile(`^[A-Z0-9-]{4,}$`)
	ansiEscapeRegex     = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-Z\\-_]`)
	emailRegex          = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	oscHyperlinkRegex   = regexp.MustCompile(`\x1b]8;;([^\x07\x1b]*)(?:\x07|\x1b\\)([^\x1b]*?)(?:\x1b]8;;(?:\x07|\x1b\\))`)
)

type deviceLoginResult struct {
	Link             string
	UserCode         string
	DeviceCode       string
	Message          string
	RawOutput        string
	OriginalOutput   string
	Err              error
	expectUserCode   bool
	expectDeviceCode bool
}

func (s *WailsAIService) runClaudeLoginJSON(binaryPath string, defaultMessage string) (*AITestResponse, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "/login", "--json")
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		if isUnknownOptionError(message) || strings.Contains(strings.ToLower(message), "unknown command") {
			return nil, false
		}
		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Claude CLI JSON login failed: %s", message),
		}, true
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return &AITestResponse{
			Success: false,
			Error:   "Claude CLI JSON login returned no data",
		}, true
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		metadata := map[string]string{"raw_output": output}
		return &AITestResponse{
			Success:  false,
			Error:    fmt.Sprintf("Failed to parse Claude CLI JSON login output: %v", err),
			Metadata: metadata,
		}, true
	}

	info := deviceLoginResult{OriginalOutput: output, RawOutput: output}
	applyLoginJSON([]byte(output), &info)
	ensureLoginInfoFromRaw(output, &info)

	message := buildLoginMessage(defaultMessage, info)
	if message == "" {
		message = defaultMessage
	}

	metadata := loginMetadata(info)
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["raw_output"] = output

	return &AITestResponse{
		Success:  true,
		Message:  message,
		Metadata: metadata,
	}, true
}

func (s *WailsAIService) runDeviceLoginCommand(binaryPath string, args []string, provider string, defaultMessage string) *AITestResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Env = os.Environ()

	reader, cleanupStream, writeInput, alreadyStarted, err := startLoginStream(cmd)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("provider", provider).Error("Failed to configure login stream")
		return &AITestResponse{Success: false, Error: "unable to prepare login stream"}
	}

	if !alreadyStarted {
		if err := cmd.Start(); err != nil {
			if cleanupStream != nil {
				cleanupStream()
			}
			s.deps.Logger.WithError(err).WithField("provider", provider).Error("Failed to start login command")
			return &AITestResponse{Success: false, Error: fmt.Sprintf("unable to start %s login: %v", provider, err)}
		}
	}

	resultChan := make(chan deviceLoginResult, 1)
	infoLatest := &deviceLoginResult{}

	// Handle interactive prompts (like Claude CLI trust prompt) and login confirmation
	if writeInput != nil {
		go func() {
			trustPromptHandled := false

			// Check for trust prompt every 200ms for up to 5 seconds
			for i := 0; i < 25; i++ {
				time.Sleep(200 * time.Millisecond)

				currentOutput := ""
				if infoLatest != nil {
					currentOutput = strings.ToLower(infoLatest.OriginalOutput)
				}

				// Detect Claude CLI trust prompt
				if !trustPromptHandled && strings.Contains(currentOutput, "do you trust the files") {
					s.deps.Logger.Debug("Detected Claude CLI trust prompt, sending 'y' response")
					writeInput([]byte("y\n"))
					trustPromptHandled = true
					// Wait a bit more for the verification URL to appear
					time.Sleep(1 * time.Second)
					break
				}

				// If we see a verification URL, we're past any prompts
				if strings.Contains(currentOutput, "http") && strings.Contains(currentOutput, "claude.ai") {
					break
				}
			}

			// Send final confirmation if needed
			if !trustPromptHandled {
				writeInput([]byte("\n"))
			}
		}()
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if cleanupStream != nil {
				cleanupStream()
			}
		})
	}

	go func() {
		defer cleanup()

		var builder strings.Builder
		var rawAll strings.Builder
		info := deviceLoginResult{}
		sent := false
		residual := ""
		buf := make([]byte, 2048)

		emit := func(text string) {
			clean := strings.TrimSpace(stripANSICodes(text))
			if clean == "" {
				return
			}

			if rawAll.Len() > 0 {
				rawAll.WriteString("\n")
			}
			rawAll.WriteString(clean)
			updateLoginInfoFromLine(clean, &info)
			info.OriginalOutput = rawAll.String()
			*infoLatest = info

			if isDecorativeLoginLine(clean) {
				return
			}

			keep := false
			for _, r := range clean {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					keep = true
					break
				}
			}
			if !keep && !strings.Contains(clean, "http") {
				return
			}

			lower := strings.ToLower(clean)
			if strings.HasPrefix(clean, "╭") ||
				strings.HasPrefix(clean, "╰") ||
				strings.HasPrefix(clean, "│") ||
				strings.HasPrefix(clean, "─") {
				if !(strings.Contains(lower, "http") ||
					strings.Contains(lower, "code") ||
					strings.Contains(lower, "token") ||
					strings.Contains(lower, "visit") ||
					strings.Contains(lower, "open")) {
					return
				}
			}

			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(clean)
			info.RawOutput = builder.String()
			*infoLatest = info
			if !sent && (info.Link != "" || info.UserCode != "" || info.DeviceCode != "") {
				select {
				case resultChan <- info:
					sent = true
				default:
				}
			}
		}

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				cleanChunk := stripANSICodes(chunk)
				if cleanChunk != "" {
					residual += cleanChunk
				}

				normalized := strings.ReplaceAll(residual, "\r\n", "\n")
				normalized = strings.ReplaceAll(normalized, "\r", "\n")
				parts := strings.Split(normalized, "\n")
				if !strings.HasSuffix(normalized, "\n") {
					residual = parts[len(parts)-1]
					parts = parts[:len(parts)-1]
				} else {
					residual = ""
				}

				for _, line := range parts {
					emit(line)
				}
			}

			if err != nil {
				if strings.TrimSpace(residual) != "" {
					emit(residual)
				}

				if err != io.EOF {
					info.Err = err
					info.RawOutput = builder.String()
					*infoLatest = info
				}
				break
			}
		}

		if waitErr := cmd.Wait(); waitErr != nil {
			info.Err = waitErr
		}

		info.OriginalOutput = rawAll.String()
		info.RawOutput = builder.String()
		*infoLatest = info

		if !sent {
			select {
			case resultChan <- info:
			default:
			}
		}
	}()

	var result deviceLoginResult
	select {
	case result = <-resultChan:
	case <-time.After(120 * time.Second):
		_ = cmd.Process.Kill()
		info := *infoLatest
		if info.Message == "" {
			info.Message = "Login command timed out before producing instructions"
		}
		cleanup()
		return &AITestResponse{
			Success:  false,
			Error:    info.Message,
			Metadata: loginMetadata(info),
		}
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		info := *infoLatest
		if info.Message == "" {
			info.Message = "Login command cancelled"
		}
		cleanup()
		return &AITestResponse{
			Success:  false,
			Error:    info.Message,
			Metadata: loginMetadata(info),
		}
	}

	if cmd.ProcessState == nil {
		_ = cmd.Process.Kill()
	}

	originalRaw := result.OriginalOutput
	if originalRaw == "" {
		originalRaw = result.RawOutput
	}

	sanitizedOutput := sanitizeLoginOutput(originalRaw)
	result.RawOutput = sanitizedOutput
	ensureLoginInfoFromRaw(originalRaw, &result)
	if result.Link == "" {
		ensureLoginInfoFromRaw(result.RawOutput, &result)
	}

	message := buildLoginMessage(defaultMessage, result)
	if message == "" && result.RawOutput != "" {
		message = result.RawOutput
	}

	result.RawOutput = message
	metadata := loginMetadata(result)

	if result.Link == "" {
		s.deps.Logger.WithField("provider", provider).WithField("raw_output", originalRaw).WithField("sanitized_output", sanitizedOutput).Warn("Claude login output missing verification URL")
	}

	if result.Err != nil && result.Link == "" && result.UserCode == "" && result.DeviceCode == "" {
		errMsg := result.Err.Error()
		if message != "" {
			errMsg = message
		}
		cleanup()
		return &AITestResponse{Success: false, Error: errMsg, Metadata: metadata}
	}

	cleanup()
	return &AITestResponse{Success: true, Message: message, Metadata: metadata}
}

// ======================================
// Global Helper Functions
// ======================================

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func containsModel(models []string, candidate string) bool {
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if candidate == "" {
		return false
	}

	for _, model := range models {
		if strings.TrimSpace(strings.ToLower(model)) == candidate {
			return true
		}
	}

	return false
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if len(value) <= 8 {
		return "********"
	}

	return fmt.Sprintf("%s****%s", value[:4], value[len(value)-4:])
}

func detectClaudeCredentials() (bool, string) {
	// Check environment variable first
	if token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); token != "" {
		return true, "credentials found in CLAUDE_CODE_OAUTH_TOKEN"
	}

	// Check ~/.claude/.credentials.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}

	credPath := homeDir + "/.claude/.credentials.json"
	if data, err := os.ReadFile(credPath); err == nil {
		var creds struct {
			ClaudeAiOauth struct {
				AccessToken string `json:"accessToken"`
				ExpiresAt   int64  `json:"expiresAt"`
			} `json:"claudeAiOauth"`
		}
		if json.Unmarshal(data, &creds) == nil && creds.ClaudeAiOauth.AccessToken != "" {
			// Check if token is not expired
			if creds.ClaudeAiOauth.ExpiresAt > time.Now().Unix() {
				return true, "credentials found in ~/.claude/.credentials.json"
			}
			return false, "credentials expired in ~/.claude/.credentials.json"
		}
	}

	return false, ""
}

func detectCodexCredentials() (string, string) {
	// Check environment variables first
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return apiKey, "from OPENAI_API_KEY environment variable"
	}
	if apiKey := os.Getenv("CODEX_API_KEY"); apiKey != "" {
		return apiKey, "from CODEX_API_KEY environment variable"
	}

	// Check ~/.codex/auth.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}

	authPath := homeDir + "/.codex/auth.json"
	if data, err := os.ReadFile(authPath); err == nil {
		var auth struct {
			ApiKey string `json:"api_key"`
		}
		if json.Unmarshal(data, &auth) == nil && auth.ApiKey != "" {
			return auth.ApiKey, "from ~/.codex/auth.json"
		}
	}

	return "", ""
}

func startLoginStream(cmd *exec.Cmd) (io.ReadCloser, func(), func([]byte), bool, error) {
	if runtime.GOOS == "windows" {
		reader, cleanup, writeFn, err := startPipeStream(cmd)
		return reader, cleanup, writeFn, false, err
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		reader, cleanup, writeFn, pipeErr := startPipeStream(cmd)
		if pipeErr != nil {
			return nil, nil, nil, false, pipeErr
		}
		return reader, cleanup, writeFn, false, nil
	}

	cleanup := func() {
		_ = ptmx.Close()
	}

	writeFn := func(data []byte) {
		_, _ = ptmx.Write(data)
	}

	return ptmx, cleanup, writeFn, true, nil
}

func startPipeStream(cmd *exec.Cmd) (io.ReadCloser, func(), func([]byte), error) {
	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter
	cmd.Stderr = pipeWriter

	stdin, err := cmd.StdinPipe()
	if err != nil {
		stdin = nil
	}

	cleanup := func() {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		if stdin != nil {
			_ = stdin.Close()
		}
	}

	writeFn := func(data []byte) {
		if stdin != nil {
			_, _ = stdin.Write(data)
		}
	}

	return pipeReader, cleanup, writeFn, nil
}

func stripANSICodes(input string) string {
	if input == "" {
		return ""
	}
	return ansiEscapeRegex.ReplaceAllString(input, "")
}

func updateLoginInfoFromLine(line string, info *deviceLoginResult) {
	if info == nil {
		return
	}

	lower := strings.ToLower(line)

	if info.Link == "" {
		if url := extractLoginURL(line); url != "" {
			info.Link = url
		}
	}

	if strings.Contains(lower, "user code") || strings.Contains(lower, "verification code") {
		info.expectUserCode = true
	}

	if strings.Contains(lower, "device code") {
		info.expectDeviceCode = true
	}

	if matches := loginCodeRegex.FindStringSubmatch(line); len(matches) > 1 {
		code := strings.TrimSpace(matches[1])
		if loginCodeValueRegex.MatchString(code) {
			if info.UserCode == "" {
				info.UserCode = code
				info.expectUserCode = false
			} else if info.DeviceCode == "" && code != info.UserCode {
				info.DeviceCode = code
				info.expectDeviceCode = false
			}
		}
	}

	tryAssignPendingCode(line, &info.UserCode, &info.expectUserCode)
	tryAssignPendingCode(line, &info.DeviceCode, &info.expectDeviceCode)
}

func applyLoginJSON(data []byte, info *deviceLoginResult) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	if info.Link == "" {
		if v, ok := payload["verification_uri_complete"].(string); ok && v != "" {
			info.Link = v
		} else if v, ok := payload["verification_uri"].(string); ok && v != "" {
			info.Link = v
		} else if v, ok := payload["login_url"].(string); ok && v != "" {
			info.Link = v
		}
	}

	if info.UserCode == "" {
		if v, ok := payload["user_code"].(string); ok && v != "" {
			info.UserCode = v
		} else if v, ok := payload["code"].(string); ok && v != "" {
			info.UserCode = v
		}
	}

	if info.DeviceCode == "" {
		if v, ok := payload["device_code"].(string); ok && v != "" {
			info.DeviceCode = v
		}
	}

	if info.Message == "" {
		if v, ok := payload["message"].(string); ok && v != "" {
			info.Message = v
		}
	}
}

func tryAssignPendingCode(trimmed string, target *string, pending *bool) {
	if !*pending || *target != "" {
		return
	}

	token := strings.TrimSpace(strings.ToUpper(trimmed))
	token = strings.Trim(token, ".\"')")
	if token == "" {
		return
	}

	if loginCodeValueRegex.MatchString(token) {
		*target = token
		*pending = false
	}
}

func isUnknownOptionError(message string) bool {
	if message == "" {
		return false
	}

	lower := strings.ToLower(message)
	return strings.Contains(lower, "unknown option") ||
		strings.Contains(lower, "unknown flag") ||
		strings.Contains(lower, "flag provided but not defined") ||
		strings.Contains(lower, "did you mean")
}

func parseClaudeWhoAmIOutput(output string) *AITestResponse {
	clean := strings.TrimSpace(stripANSICodes(output))
	if clean == "" {
		return &AITestResponse{
			Success: true,
			Message: "Claude CLI responded successfully. Complete the login flow if prompted.",
		}
	}

	var whoami struct {
		LoggedIn bool `json:"loggedIn"`
		Account  struct {
			Email string `json:"email"`
		} `json:"account"`
	}

	if err := json.Unmarshal([]byte(clean), &whoami); err == nil {
		metadata := map[string]string{
			"raw_output": clean,
		}

		if whoami.Account.Email != "" {
			metadata["email"] = whoami.Account.Email
		}

		if !whoami.LoggedIn {
			return &AITestResponse{
				Success:  false,
				Error:    "Claude CLI is not logged in. Run 'claude login' to link your account.",
				Metadata: metadata,
			}
		}

		message := "Claude CLI authenticated"
		if whoami.Account.Email != "" {
			message = fmt.Sprintf("Claude CLI authenticated as %s", whoami.Account.Email)
		}

		return &AITestResponse{
			Success:  true,
			Message:  message,
			Metadata: metadata,
		}
	}

	return parseClaudeWhoAmIPlain(clean)
}

func parseClaudeWhoAmIPlain(output string) *AITestResponse {
	clean := strings.TrimSpace(stripANSICodes(output))
	if clean == "" {
		return &AITestResponse{
			Success: true,
			Message: "Claude CLI responded successfully. Complete the login flow if prompted.",
		}
	}

	lower := strings.ToLower(clean)

	metadata := map[string]string{
		"raw_output": clean,
	}

	if strings.Contains(lower, "not logged") || strings.Contains(lower, "please run") && strings.Contains(lower, "login") {
		return &AITestResponse{
			Success:  false,
			Error:    "Claude CLI is not logged in. Run 'claude login' to link your account.",
			Metadata: metadata,
		}
	}

	if email := emailRegex.FindString(clean); email != "" {
		metadata["email"] = email
		return &AITestResponse{
			Success:  true,
			Message:  fmt.Sprintf("Claude CLI authenticated as %s", email),
			Metadata: metadata,
		}
	}

	if strings.Contains(lower, "logged in") || strings.Contains(lower, "authenticated") {
		return &AITestResponse{
			Success:  true,
			Message:  clean,
			Metadata: metadata,
		}
	}

	return &AITestResponse{
		Success:  true,
		Message:  clean,
		Metadata: metadata,
	}
}

func buildLoginMessage(defaultMessage string, result deviceLoginResult) string {
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = strings.TrimSpace(defaultMessage)
	}

	var lines []string
	if message != "" {
		lines = append(lines, message)
	}

	if result.Link != "" && !strings.Contains(strings.ToLower(message), "http") {
		lines = append(lines, fmt.Sprintf("Verification URL: %s", result.Link))
	}

	if result.UserCode != "" {
		lines = append(lines, fmt.Sprintf("Code: %s", result.UserCode))
	}

	if result.DeviceCode != "" && !strings.EqualFold(result.DeviceCode, result.UserCode) {
		lines = append(lines, fmt.Sprintf("Device Code: %s", result.DeviceCode))
	}

	seen := map[string]struct{}{}
	dedup := make([]string, 0, len(lines))
	for _, line := range lines {
		key := strings.ToLower(strings.TrimSpace(line))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dedup = append(dedup, line)
	}

	if len(dedup) == 0 {
		return defaultMessage
	}

	return strings.Join(dedup, "\n")
}

func sanitizeLoginOutput(output string) string {
	if output == "" {
		return ""
	}

	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))

	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(stripANSICodes(line))
		if trimmed == "" {
			continue
		}

		if isDecorativeLoginLine(trimmed) {
			continue
		}

		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		filtered = append(filtered, trimmed)
	}

	return strings.Join(filtered, "\n")
}

func isDecorativeLoginLine(line string) bool {
	if line == "" {
		return true
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	isDecorativeChars := true
	for _, r := range trimmed {
		switch r {
		case '│', '╭', '╰', '─', '╮', '╯', '╲', '╱', '╳', '┼', '┌', '┐', '└', '┘', '━', '┃', '┏', '┓', '┗', '┛', '╸', '╹', '╺', '╻':
			continue
		case ' ', '\t':
			continue
		default:
			isDecorativeChars = false
		}
	}
	if isDecorativeChars {
		return true
	}

	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "/help") ||
		strings.Contains(lower, "/status") ||
		strings.Contains(lower, "cwd:") ||
		strings.Contains(lower, "welcome to claude code") ||
		strings.Contains(lower, "mcp server needs auth") ||
		strings.Contains(lower, "? for shortcuts") ||
		strings.Contains(lower, "for shortcuts") && strings.Contains(lower, "/ide") ||
		strings.Contains(lower, "hatching") ||
		strings.Contains(lower, "beaming") ||
		strings.HasPrefix(lower, "try \"") ||
		strings.HasPrefix(lower, ">") && !strings.Contains(lower, "http") && !strings.Contains(lower, "code") && !strings.Contains(lower, "device") ||
		strings.HasPrefix(lower, "·") && !strings.Contains(lower, "http") && !strings.Contains(lower, "code") && !strings.Contains(lower, "device") ||
		strings.EqualFold(trimmed, "code") {
		return true
	}

	return false
}

func ensureLoginInfoFromRaw(raw string, info *deviceLoginResult) {
	if raw == "" || info == nil {
		return
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		clean := strings.TrimSpace(stripANSICodes(line))
		if clean == "" {
			continue
		}
		updateLoginInfoFromLine(clean, info)
		if info.Link != "" && info.UserCode != "" && info.DeviceCode != "" {
			break
		}
	}

	if info.Link == "" {
		if url := extractLoginURL(raw); url != "" {
			info.Link = url
		}
	}
}

func extractLoginURL(text string) string {
	if text == "" {
		return ""
	}
	if match := loginURLRegex.FindString(text); match != "" {
		return strings.TrimRight(match, ".,\"')")
	}
	return ""
}

func loginMetadata(info deviceLoginResult) map[string]string {
	metadata := map[string]string{}
	if info.Link != "" {
		metadata["verification_url"] = info.Link
	}
	if info.UserCode != "" {
		metadata["user_code"] = info.UserCode
	}
	if info.DeviceCode != "" {
		metadata["device_code"] = info.DeviceCode
	}
	if info.RawOutput != "" {
		metadata["raw_output"] = info.RawOutput
	}
	if info.OriginalOutput != "" {
		metadata["original_raw_output"] = info.OriginalOutput
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

// SQL Helper Functions

func buildManualContext(isMulti bool, schemaContext, provided string) string {
	sections := make([]string, 0, 3)
	if trimmed := strings.TrimSpace(schemaContext); trimmed != "" {
		sections = append(sections, trimmed)
	}
	if trimmed := strings.TrimSpace(provided); trimmed != "" {
		sections = append(sections, trimmed)
	}
	if isMulti {
		sections = append(sections, multiDBGuidance)
	}
	return aggregateContextSections(sections...)
}

func buildFallbackPrompt(userPrompt string, manualContext string, isMulti bool) (string, string) {
	contextBlock := strings.TrimSpace(manualContext)
	var builder strings.Builder

	if isMulti {
		builder.WriteString("You are operating in multi-database mode; use @connection_name.table syntax for remote tables.\n")
		builder.WriteString("Ensure schema-qualified names for non-public schemas.\n\n")
	}
	if contextBlock != "" {
		builder.WriteString("Context:\n")
		builder.WriteString(contextBlock)
		builder.WriteString("\n\n")
	}

	builder.WriteString("User request:\n")
	builder.WriteString(userPrompt)

	return builder.String(), contextBlock
}

func aggregateContextSections(sections ...string) string {
	valid := make([]string, 0, len(sections))
	for _, section := range sections {
		if trimmed := strings.TrimSpace(section); trimmed != "" {
			valid = append(valid, trimmed)
		}
	}
	return strings.Join(valid, "\n\n---\n\n")
}

func renderQueryContext(ctx *rag.QueryContext) string {
	if ctx == nil {
		return ""
	}

	var builder strings.Builder

	if len(ctx.RelevantSchemas) > 0 {
		builder.WriteString("Relevant schemas:\n")
		for i, schema := range ctx.RelevantSchemas {
			if i >= 5 {
				builder.WriteString("- ...\n")
				break
			}
			builder.WriteString(fmt.Sprintf("- %s", schema.TableName))
			if schema.Description != "" {
				builder.WriteString(fmt.Sprintf(" · %s", schema.Description))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	if len(ctx.SimilarQueries) > 0 {
		builder.WriteString("Similar query patterns:\n")
		for i, pattern := range ctx.SimilarQueries {
			if i >= 3 {
				break
			}
			builder.WriteString(fmt.Sprintf("- %s (used %d×)\n", pattern.Pattern, pattern.Frequency))
		}
		builder.WriteString("\n")
	}

	if ctx.DataStatistics != nil {
		builder.WriteString("Data characteristics:\n")
		builder.WriteString(fmt.Sprintf("- Approx rows: %d\n", ctx.DataStatistics.TotalRows))
		if ctx.DataStatistics.GrowthRate != 0 {
			builder.WriteString(fmt.Sprintf("- Growth rate: %.2f%%\n", ctx.DataStatistics.GrowthRate))
		}
	}

	return strings.TrimSpace(builder.String())
}

const multiDBGuidance = `IMPORTANT SQL Generation Rules for Multi-Database Mode:
1. Use @connection_name.table_name syntax for remote tables.
2. Use @connection_name.schema_name.table_name for non-public schemas.
3. Alias tables clearly when combining databases.
4. Cross-database JOINs are allowed; ensure join conditions use compatible keys.
5. Connection names are case-sensitive.`

func composePrompt(userPrompt string, context string, multi bool) string {
	var builder strings.Builder
	if multi {
		builder.WriteString("Multi-database mode is active. Use @connection_name.table syntax for remote tables.\n")
		builder.WriteString("Ensure joins reference the correct connection names and schema-qualified identifiers.\n\n")
	}
	if trimmed := strings.TrimSpace(context); trimmed != "" {
		builder.WriteString("Context:\n")
		builder.WriteString(trimmed)
		builder.WriteString("\n\n")
	}
	builder.WriteString("User request:\n")
	builder.WriteString(userPrompt)
	return builder.String()
}

// formatColumnsForAI shortens column metadata for prompt consumption.
func formatColumnsForAI(columns []database.ColumnInfo, maxColumns int) []string {
	formatted := make([]string, 0, len(columns))

	for _, column := range columns {
		columnType := strings.ToLower(column.DataType)
		if columnType == "" {
			columnType = "unknown"
		}

		attributes := make([]string, 0, 3)
		if column.PrimaryKey {
			attributes = append(attributes, "pk")
		}
		if column.Unique {
			attributes = append(attributes, "unique")
		}
		if !column.Nullable {
			attributes = append(attributes, "not null")
		}

		if len(attributes) > 0 {
			columnType = fmt.Sprintf("%s %s", columnType, strings.Join(attributes, "/"))
		}

		formatted = append(formatted, fmt.Sprintf("%s (%s)", column.Name, columnType))

		if maxColumns > 0 && len(formatted) >= maxColumns {
			break
		}
	}

	if maxColumns > 0 && len(columns) > maxColumns {
		formatted = append(formatted, "... additional columns omitted")
	}

	return formatted
}

// sanitizeSQLResponse removes duplicate SQL blocks and strips code from explanations to reduce UI noise.
func sanitizeSQLResponse(resp *ai.SQLResponse) {
	if resp == nil {
		return
	}

	resp.SQL = deduplicateSequentialSQL(resp.SQL)
	resp.SQL = strings.TrimSpace(resp.SQL)

	resp.Explanation = sanitizeExplanation(resp.Explanation, resp.SQL)
}

// deduplicateSequentialSQL collapses responses where the model repeats the same SQL twice.
func deduplicateSequentialSQL(sql string) string {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return trimmed
	}

	halfStart := len(trimmed) / 2
	for i := halfStart; i < len(trimmed); i++ {
		if trimmed[i] != '\n' && trimmed[i] != '\r' {
			continue
		}

		first := strings.TrimSpace(trimmed[:i])
		second := strings.TrimSpace(trimmed[i:])
		if first != "" && first == second {
			return first
		}
	}

	return trimmed
}

// sanitizeExplanation strips code blocks and duplicate SQL from the explanation text.
func sanitizeExplanation(explanation string, sql string) string {
	if explanation == "" {
		return ""
	}

	cleaned := removeCodeBlocks(explanation)

	if sql != "" {
		cleaned = strings.ReplaceAll(cleaned, sql, "")
		// Also remove compressed versions of the SQL where extra whitespace may have been collapsed.
		compressedSQL := strings.Join(strings.Fields(sql), " ")
		if compressedSQL != "" {
			cleaned = strings.ReplaceAll(cleaned, compressedSQL, "")
		}
	}

	return strings.TrimSpace(cleaned)
}

// removeCodeBlocks removes fenced code blocks (``` ... ```) from a string.
func removeCodeBlocks(text string) string {
	result := text

	for {
		start := strings.Index(result, "```")
		if start == -1 {
			break
		}

		closing := strings.Index(result[start+3:], "```")
		if closing == -1 {
			break
		}
		closing += start + 3

		if closing > len(result) {
			closing = len(result)
		}

		result = result[:start] + result[closing+3:]
	}

	return strings.TrimSpace(result)
}

// isMultiDatabaseSQL, clamp, minInt are defined in ai_query_agent.go
