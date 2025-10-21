package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// OllamaDetector handles detection and management of Ollama installation
type OllamaDetector struct {
	logger *logrus.Logger
	client *http.Client
}

// OllamaStatus represents the status of Ollama installation and service
type OllamaStatus struct {
	Installed     bool      `json:"installed"`
	Running       bool      `json:"running"`
	Version       string    `json:"version,omitempty"`
	Endpoint      string    `json:"endpoint"`
	AvailableModels []string `json:"available_models"`
	LastChecked   time.Time `json:"last_checked"`
	Error         string    `json:"error,omitempty"`
}

// OllamaModelInfo represents information about an Ollama model
type OllamaModelInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	ModifiedAt time.Time `json:"modified_at"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// OllamaListResponse represents the response from Ollama's /api/tags endpoint
type OllamaListResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

// NewOllamaDetector creates a new Ollama detector
func NewOllamaDetector(logger *logrus.Logger) *OllamaDetector {
	return &OllamaDetector{
		logger: logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// DetectOllama checks if Ollama is installed and running
func (d *OllamaDetector) DetectOllama(ctx context.Context) (*OllamaStatus, error) {
	status := &OllamaStatus{
		Endpoint:      "http://localhost:11434",
		LastChecked:   time.Now(),
		AvailableModels: []string{},
	}

	// Check if Ollama is installed
	installed, version, err := d.IsOllamaInstalled()
	if err != nil {
		status.Error = fmt.Sprintf("Failed to check Ollama installation: %v", err)
		return status, nil
	}
	status.Installed = installed
	status.Version = version

	if !installed {
		return status, nil
	}

	// Check if Ollama service is running
	running, err := d.IsOllamaRunning(ctx, status.Endpoint)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to check Ollama service: %v", err)
		return status, nil
	}
	status.Running = running

	if !running {
		return status, nil
	}

	// Get available models
	models, err := d.ListAvailableModels(ctx, status.Endpoint)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get available models")
	} else {
		status.AvailableModels = models
	}

	return status, nil
}

// IsOllamaInstalled checks if Ollama is installed on the system
func (d *OllamaDetector) IsOllamaInstalled() (bool, string, error) {
	// Try to run ollama --version
	cmd := exec.CommandContext(context.Background(), "ollama", "--version")
	output, err := cmd.Output()
	if err != nil {
		// Check if ollama command exists in PATH
		if _, err := exec.LookPath("ollama"); err != nil {
			return false, "", nil
		}
		return false, "", err
	}

	// Parse version from output (handle warnings)
	version := strings.TrimSpace(string(output))
	lines := strings.Split(version, "\n")
	for _, line := range lines {
		if strings.Contains(line, "client version") {
			parts := strings.Split(line, "client version is")
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	return true, version, nil
}

// IsOllamaRunning checks if Ollama service is running and accessible
func (d *OllamaDetector) IsOllamaRunning(ctx context.Context, endpoint string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return false, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// ListAvailableModels retrieves the list of available models from Ollama
func (d *OllamaDetector) ListAvailableModels(ctx context.Context, endpoint string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var listResp OllamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	var models []string
	for _, model := range listResp.Models {
		models = append(models, model.Name)
	}

	return models, nil
}

// InstallOllama provides instructions for installing Ollama
func (d *OllamaDetector) InstallOllama() (string, error) {
	var installCmd string
	var installURL string
	var startCmd string

	switch runtime.GOOS {
	case "darwin":
		installCmd = "brew install ollama"
		installURL = "https://ollama.ai/download/mac"
		startCmd = "ollama serve"
	case "linux":
		installCmd = "curl -fsSL https://ollama.ai/install.sh | sh"
		installURL = "https://ollama.ai/download/linux"
		startCmd = "ollama serve"
	case "windows":
		installCmd = "winget install Ollama.Ollama"
		installURL = "https://ollama.ai/download/windows"
		startCmd = "ollama serve"
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	instructions := fmt.Sprintf(`Ollama Installation Instructions

1. Install Ollama:
   Option A: Visit %s and download the installer
   Option B: Run: %s

2. Start Ollama service:
   %s

3. Install the recommended model:
   ollama pull prem-1b-sql

4. Verify installation:
   ollama list

For more information, visit: https://ollama.ai

Note: After installation, you may need to restart your terminal or add Ollama to your PATH.`, 
		installURL, installCmd, startCmd)

	return instructions, nil
}

// StartOllamaService attempts to start the Ollama service
func (d *OllamaDetector) StartOllamaService(ctx context.Context) error {
	// Check if Ollama is already running
	status, err := d.DetectOllama(ctx)
	if err != nil {
		return err
	}

	if status.Running {
		return nil // Already running
	}

	if !status.Installed {
		return fmt.Errorf("Ollama is not installed")
	}

	// Try to start Ollama service
	// Note: This might not work on all systems due to permission requirements
	d.logger.Info("Attempting to start Ollama service")
	
	cmd := exec.CommandContext(ctx, "ollama", "serve")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Ollama service: %v. Please start it manually with 'ollama serve'", err)
	}

	// Wait a moment for the service to start
	time.Sleep(3 * time.Second)

	// Verify it's running
	status, err = d.DetectOllama(ctx)
	if err != nil {
		return err
	}

	if !status.Running {
		return fmt.Errorf("Ollama service failed to start. Please start it manually with 'ollama serve'")
	}

	d.logger.Info("Ollama service started successfully")
	return nil
}

// PullModel pulls a specific model from Ollama
func (d *OllamaDetector) PullModel(ctx context.Context, modelName string) error {
	status, err := d.DetectOllama(ctx)
	if err != nil {
		return err
	}

	if !status.Running {
		return fmt.Errorf("Ollama service is not running")
	}

	d.logger.WithField("model", modelName).Info("Pulling Ollama model")

	cmd := exec.CommandContext(ctx, "ollama", "pull", modelName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull model %s: %v, output: %s", modelName, err, string(output))
	}

	d.logger.WithField("model", modelName).Info("Model pulled successfully")
	return nil
}

// GetRecommendedModels returns a list of recommended models for SQL generation
func (d *OllamaDetector) GetRecommendedModels() []string {
	return []string{
		"prem-1b-sql",      // Primary recommendation
		"sqlcoder:7b",      // Alternative SQL-focused model
		"codellama:7b",     // General code generation
		"llama3.1:8b",     // General purpose
		"mistral:7b",       // Alternative general purpose
	}
}

// CheckModelExists checks if a specific model is available
func (d *OllamaDetector) CheckModelExists(ctx context.Context, modelName string, endpoint string) (bool, error) {
	models, err := d.ListAvailableModels(ctx, endpoint)
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model == modelName {
			return true, nil
		}
	}

	return false, nil
}
