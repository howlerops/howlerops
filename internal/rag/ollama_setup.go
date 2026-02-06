package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// OllamaModelManager manages Ollama models
type OllamaModelManager struct {
	endpoint string
	client   *http.Client
	logger   *logrus.Logger
}

// NewOllamaModelManager creates a new Ollama model manager
func NewOllamaModelManager(endpoint string, logger *logrus.Logger) *OllamaModelManager {
	return &OllamaModelManager{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 5 * time.Minute, // Model downloads can take time
		},
		logger: logger,
	}
}

// EnsureModelAvailable checks if model exists, downloads if not
func (m *OllamaModelManager) EnsureModelAvailable(ctx context.Context, modelName string) error {
	// Check if model exists
	exists, err := m.hasModel(ctx, modelName)
	if err != nil {
		return fmt.Errorf("failed to check model: %w", err)
	}

	if exists {
		m.logger.WithField("model", modelName).Debug("Model already available")
		return nil
	}

	// Pull model
	m.logger.WithField("model", modelName).Info("Downloading embedding model (first time only)...")
	return m.pullModel(ctx, modelName)
}

// hasModel checks if a model is available locally
func (m *OllamaModelManager) hasModel(ctx context.Context, modelName string) (bool, error) {
	url := fmt.Sprintf("%s/api/tags", m.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to list models: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	for _, model := range result.Models {
		if model.Name == modelName {
			return true, nil
		}
	}

	return false, nil
}

// pullModel downloads a model from Ollama registry
func (m *OllamaModelManager) pullModel(ctx context.Context, modelName string) error {
	url := fmt.Sprintf("%s/api/pull", m.endpoint)

	reqBody := map[string]string{
		"name": modelName,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Stream progress (optional: could parse and show percentage)
	// For now, just wait for completion
	_, _ = io.Copy(io.Discard, resp.Body)

	m.logger.WithField("model", modelName).Info("Model downloaded successfully")
	return nil
}
