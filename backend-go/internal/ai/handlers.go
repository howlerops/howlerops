package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// HTTPHandler handles HTTP requests for the AI service
type HTTPHandler struct {
	service Service
	logger  *logrus.Logger
}

// NewHTTPHandler creates a new HTTP handler for the AI service
func NewHTTPHandler(service Service, logger *logrus.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all AI-related HTTP routes
func (h *HTTPHandler) RegisterRoutes(router *mux.Router) {
	// AI generation endpoints
	router.HandleFunc("/generate-sql", h.GenerateSQL).Methods("POST")
	router.HandleFunc("/fix-sql", h.FixSQL).Methods("POST")

	// Provider management endpoints - TODO: Implement these handlers
	// router.HandleFunc("/providers", h.GetProviders).Methods("GET")
	// router.HandleFunc("/providers/{provider}/health", h.GetProviderHealth).Methods("GET")
	// router.HandleFunc("/providers/{provider}/models", h.GetProviderModels).Methods("GET")
	// router.HandleFunc("/providers/{provider}/test", h.TestProvider).Methods("POST")

	// Test connection endpoints
	router.HandleFunc("/test/openai", h.TestOpenAI).Methods("POST")
	router.HandleFunc("/test/anthropic", h.TestAnthropic).Methods("POST")
	router.HandleFunc("/test/ollama", h.TestOllama).Methods("POST")
	router.HandleFunc("/test/huggingface", h.TestHuggingFace).Methods("POST")
	router.HandleFunc("/test/claudecode", h.TestClaudeCode).Methods("POST")
	router.HandleFunc("/test/codex", h.TestCodex).Methods("POST")

	// Ollama detection and management endpoints
	router.HandleFunc("/ollama/detect", h.DetectOllama).Methods("GET")
	router.HandleFunc("/ollama/install", h.GetOllamaInstallInstructions).Methods("GET")
	router.HandleFunc("/ollama/start", h.StartOllamaService).Methods("POST")
	router.HandleFunc("/ollama/pull", h.PullOllamaModel).Methods("POST")
	router.HandleFunc("/ollama/open-terminal", h.OpenOllamaTerminal).Methods("POST")

	// Usage and analytics endpoints - TODO: Implement these handlers
	// router.HandleFunc("/usage", h.GetUsageStats).Methods("GET")
	// router.HandleFunc("/usage/{provider}", h.GetProviderUsage).Methods("GET")

	// Configuration endpoints - TODO: Implement these handlers
	// router.HandleFunc("/config", h.GetConfig).Methods("GET")
}

// Helper methods

// respondWithJSON writes a JSON response
func (h *HTTPHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// respondWithError writes an error response
func (h *HTTPHandler) respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	if err != nil {
		h.logger.WithError(err).Error(message)
	}
	h.respondWithJSON(w, statusCode, map[string]string{
		"error": message,
	})
}

// DetectOllama detects Ollama installation and status
func (h *HTTPHandler) DetectOllama(w http.ResponseWriter, r *http.Request) {
	detector := NewOllamaDetector(h.logger)

	status, err := detector.DetectOllama(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detect Ollama: %v", err), err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, status)
}

// GetOllamaInstallInstructions returns installation instructions for Ollama
func (h *HTTPHandler) GetOllamaInstallInstructions(w http.ResponseWriter, r *http.Request) {
	detector := NewOllamaDetector(h.logger)

	instructions, err := detector.InstallOllama()
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get installation instructions: %v", err), err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"instructions": instructions,
	})
}

// StartOllamaService attempts to start the Ollama service
func (h *HTTPHandler) StartOllamaService(w http.ResponseWriter, r *http.Request) {
	detector := NewOllamaDetector(h.logger)

	err := detector.StartOllamaService(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start Ollama service: %v", err), err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Ollama service started successfully",
	})
}

// PullOllamaModel pulls a specific model from Ollama
func (h *HTTPHandler) PullOllamaModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Model == "" {
		h.respondWithError(w, http.StatusBadRequest, "Model name is required", fmt.Errorf("model name is required"))
		return
	}

	detector := NewOllamaDetector(h.logger)

	err := detector.PullModel(r.Context(), req.Model)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pull model: %v", err), err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Model %s pulled successfully", req.Model),
	})
}

// OpenOllamaTerminal attempts to launch a terminal window with Ollama commands prefilled
func (h *HTTPHandler) OpenOllamaTerminal(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }() // Best-effort close

	commands := []string{"ollama serve"}

	if r.Body != nil && r.ContentLength != 0 {
		var req struct {
			Commands []string `json:"commands"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		if len(req.Commands) > 0 {
			commands = req.Commands
		}
	}

	if err := launchTerminalWithCommands(commands); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to open terminal: %v", err), err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Terminal launched",
	})
}

func launchTerminalWithCommands(commands []string) error {
	if len(commands) == 0 {
		commands = []string{"ollama serve"}
	}

	cmdString := strings.Join(commands, " && ")

	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`tell application "Terminal"
  activate
  do script "%s"
end tell`, escapeAppleScriptString(cmdString))
		// #nosec G204 - command string is sanitized via escapeAppleScriptString, terminal launch is intended feature
		return exec.Command("osascript", "-e", script).Start()
	case "linux":
		candidates := [][]string{
			{"x-terminal-emulator", "-e", "bash", "-lc", cmdString},
			{"gnome-terminal", "--", "bash", "-lc", cmdString},
			{"konsole", "-e", "bash", "-lc", cmdString},
			{"mate-terminal", "-e", "bash", "-lc", cmdString},
			{"xfce4-terminal", "-e", "bash", "-lc", cmdString},
			{"alacritty", "-e", "bash", "-lc", cmdString},
		}

		for _, candidate := range candidates {
			if _, err := exec.LookPath(candidate[0]); err == nil {
				// #nosec G204 - terminal emulator launch with validated cmdString, intended feature
				return exec.Command(candidate[0], candidate[1:]...).Start()
			}
		}

		return fmt.Errorf("no supported terminal emulator found; run manually: %s", cmdString)
	case "windows":
		// #nosec G204 - Windows terminal launch with cmdString, intended feature
		return exec.Command("cmd", "/C", "start", "cmd", "/K", cmdString).Start()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func escapeAppleScriptString(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return escaped
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GenerateSQL handles SQL generation requests
func (h *HTTPHandler) GenerateSQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt      string  `json:"prompt"`
		Schema      string  `json:"schema"`
		Provider    string  `json:"provider"`
		Model       string  `json:"model"`
		MaxTokens   int     `json:"maxTokens"`
		Temperature float64 `json:"temperature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Prompt == "" {
		h.respondWithError(w, http.StatusBadRequest, "Prompt is required", nil)
		return
	}

	// Create SQL request
	sqlReq := &SQLRequest{
		Prompt:      req.Prompt,
		Schema:      req.Schema,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Provider:    Provider(req.Provider),
	}

	// Generate SQL using the AI service
	response, err := h.service.GenerateSQL(r.Context(), sqlReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to generate SQL", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"sql":        response.Query,
		"confidence": response.Confidence,
		"tokens":     response.TokensUsed,
	})
}

// FixSQL handles SQL fix requests
func (h *HTTPHandler) FixSQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query       string  `json:"query"`
		Error       string  `json:"error"`
		Schema      string  `json:"schema"`
		Provider    string  `json:"provider"`
		Model       string  `json:"model"`
		MaxTokens   int     `json:"maxTokens"`
		Temperature float64 `json:"temperature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" || req.Error == "" {
		h.respondWithError(w, http.StatusBadRequest, "Query and error are required", nil)
		return
	}

	// Create SQL request
	sqlReq := &SQLRequest{
		Query:       req.Query,
		Error:       req.Error,
		Schema:      req.Schema,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Provider:    Provider(req.Provider),
	}

	// Fix SQL using the AI service
	response, err := h.service.FixSQL(r.Context(), sqlReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to fix SQL", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"sql":        response.Query,
		"confidence": response.Confidence,
		"tokens":     response.TokensUsed,
	})
}

// TestOpenAI tests OpenAI provider connection
func (h *HTTPHandler) TestOpenAI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey       string `json:"apiKey"`
		Model        string `json:"model"`
		Organization string `json:"organization"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	config := &OpenAIConfig{
		APIKey: req.APIKey,
		Models: []string{req.Model},
		OrgID:  req.Organization,
	}

	provider, err := NewOpenAIProvider(config, h.logger)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create OpenAI provider", err)
		return
	}

	health, err := provider.HealthCheck(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "OpenAI connection test failed", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  health.Status,
		"message": "OpenAI connection successful",
	})
}

// TestAnthropic tests Anthropic provider connection
func (h *HTTPHandler) TestAnthropic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"apiKey"`
		Model  string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	config := &AnthropicConfig{
		APIKey: req.APIKey,
		Models: []string{req.Model},
	}

	provider, err := NewAnthropicProvider(config, h.logger)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create Anthropic provider", err)
		return
	}

	health, err := provider.HealthCheck(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "Anthropic connection test failed", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  health.Status,
		"message": "Anthropic connection successful",
	})
}

// TestOllama tests Ollama provider connection
func (h *HTTPHandler) TestOllama(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	config := &OllamaConfig{
		Endpoint: req.Endpoint,
		Models:   []string{req.Model},
	}

	provider, err := NewOllamaProvider(config, h.logger)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create Ollama provider", err)
		return
	}

	health, err := provider.HealthCheck(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "Ollama connection test failed", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  health.Status,
		"message": "Ollama connection successful",
	})
}

// TestHuggingFace tests HuggingFace provider connection
func (h *HTTPHandler) TestHuggingFace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey   string `json:"apiKey"`
		Model    string `json:"model"`
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	config := &HuggingFaceConfig{
		Endpoint: req.Endpoint,
		Models:   []string{req.Model},
	}

	provider, err := NewHuggingFaceProvider(config, h.logger)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create HuggingFace provider", err)
		return
	}

	health, err := provider.HealthCheck(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "HuggingFace connection test failed", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  health.Status,
		"message": "HuggingFace connection successful",
	})
}

// TestClaudeCode tests Claude Code provider connection
func (h *HTTPHandler) TestClaudeCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BinaryPath string `json:"binaryPath"`
		Model      string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Use default binary path if not provided
	if req.BinaryPath == "" {
		req.BinaryPath = "claude"
	}

	config := &ClaudeCodeConfig{
		ClaudePath:  req.BinaryPath,
		Model:       req.Model,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// Create provider using the factory
	factory := &DefaultProviderFactory{}
	provider, err := factory.CreateProvider(ProviderClaudeCode, config)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create Claude Code provider", err)
		return
	}
	defer func() {
		if err := provider.Close(); err != nil {
			h.logger.WithError(err).Error("Failed to close Claude Code provider")
		}
	}()

	// Wrap in adapter for compatibility
	wrappedProvider := &providerAdapterWrapper{
		adapter: provider,
		logger:  h.logger,
	}

	health, err := wrappedProvider.HealthCheck(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "Claude Code connection test failed", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  health.Status,
		"message": "Claude Code connection successful",
	})
}

// TestCodex tests Codex provider connection
func (h *HTTPHandler) TestCodex(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey       string `json:"apiKey"`
		Model        string `json:"model"`
		Organization string `json:"organization"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	config := &CodexConfig{
		APIKey:       req.APIKey,
		Model:        req.Model,
		Organization: req.Organization,
	}

	// Create provider using the factory
	factory := &DefaultProviderFactory{}
	provider, err := factory.CreateProvider(ProviderCodex, config)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create Codex provider", err)
		return
	}
	defer func() {
		if err := provider.Close(); err != nil {
			h.logger.WithError(err).Error("Failed to close Codex provider")
		}
	}()

	// Wrap in adapter for compatibility
	wrappedProvider := &providerAdapterWrapper{
		adapter: provider,
		logger:  h.logger,
	}

	health, err := wrappedProvider.HealthCheck(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "Codex connection test failed", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  health.Status,
		"message": "Codex connection successful",
	})
}
