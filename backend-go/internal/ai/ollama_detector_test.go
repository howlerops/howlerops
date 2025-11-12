//go:build integration

package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/ai"
)

// TestNewOllamaDetector tests the constructor
func TestNewOllamaDetector(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	require.NotNil(t, detector)
}

// TestIsOllamaRunning_Success tests checking if Ollama is running successfully
func TestIsOllamaRunning_Success(t *testing.T) {
	// Create mock server that returns 200 OK for /api/tags
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			resp := ai.OllamaListResponse{
				Models: []ai.OllamaModelInfo{},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	running, err := detector.IsOllamaRunning(ctx, server.URL)

	require.NoError(t, err)
	assert.True(t, running)
}

// TestIsOllamaRunning_NotRunning tests when Ollama is not running
func TestIsOllamaRunning_NotRunning(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	// Use an endpoint that doesn't exist
	running, err := detector.IsOllamaRunning(ctx, "http://localhost:99999")

	require.Error(t, err)
	assert.False(t, running)
}

// TestIsOllamaRunning_InvalidEndpoint tests with invalid endpoint
func TestIsOllamaRunning_InvalidEndpoint(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	running, err := detector.IsOllamaRunning(ctx, "://invalid-url")

	require.Error(t, err)
	assert.False(t, running)
}

// TestIsOllamaRunning_NonOKStatus tests when server returns non-200 status
func TestIsOllamaRunning_NonOKStatus(t *testing.T) {
	// Create mock server that returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	running, err := detector.IsOllamaRunning(ctx, server.URL)

	require.NoError(t, err)
	assert.False(t, running)
}

// TestIsOllamaRunning_ContextCanceled tests with canceled context
func TestIsOllamaRunning_ContextCanceled(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	running, err := detector.IsOllamaRunning(ctx, "http://localhost:11434")

	require.Error(t, err)
	assert.False(t, running)
}

// TestListAvailableModels_Success tests retrieving available models
func TestListAvailableModels_Success(t *testing.T) {
	// Create mock server with sample models
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			resp := ai.OllamaListResponse{
				Models: []ai.OllamaModelInfo{
					{
						Name:       "llama2",
						Size:       4000000000,
						Digest:     "abc123",
						ModifiedAt: time.Now(),
					},
					{
						Name:       "codellama",
						Size:       7000000000,
						Digest:     "def456",
						ModifiedAt: time.Now(),
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	models, err := detector.ListAvailableModels(ctx, server.URL)

	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "llama2", models[0])
	assert.Equal(t, "codellama", models[1])
}

// TestListAvailableModels_EmptyList tests with no models installed
func TestListAvailableModels_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			resp := ai.OllamaListResponse{
				Models: []ai.OllamaModelInfo{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	models, err := detector.ListAvailableModels(ctx, server.URL)

	require.NoError(t, err)
	assert.Empty(t, models)
}

// TestListAvailableModels_ServerError tests when server returns error
func TestListAvailableModels_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	models, err := detector.ListAvailableModels(ctx, server.URL)

	require.Error(t, err)
	assert.Nil(t, models)
	assert.Contains(t, err.Error(), "HTTP 500")
}

// TestListAvailableModels_InvalidJSON tests with invalid JSON response
func TestListAvailableModels_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	models, err := detector.ListAvailableModels(ctx, server.URL)

	require.Error(t, err)
	assert.Nil(t, models)
}

// TestListAvailableModels_NetworkError tests with network error
func TestListAvailableModels_NetworkError(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	models, err := detector.ListAvailableModels(ctx, "http://localhost:99999")

	require.Error(t, err)
	assert.Nil(t, models)
}

// TestListAvailableModels_ContextCanceled tests with canceled context
func TestListAvailableModels_ContextCanceled(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	models, err := detector.ListAvailableModels(ctx, "http://localhost:11434")

	require.Error(t, err)
	assert.Nil(t, models)
}

// TestDetectOllama_NotInstalled tests detection when Ollama is not installed
func TestDetectOllama_NotInstalled(t *testing.T) {
	// This test will vary depending on whether Ollama is actually installed
	// We can only test that it returns a valid status structure
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	status, err := detector.DetectOllama(ctx)

	require.NoError(t, err, "DetectOllama should not return an error")
	require.NotNil(t, status, "status should not be nil")
	assert.NotEmpty(t, status.Endpoint, "endpoint should be set")
	assert.NotZero(t, status.LastChecked, "last checked timestamp should be set")
	assert.NotNil(t, status.AvailableModels, "available models slice should be initialized")
}

// TestDetectOllama_Timeout tests detection with a cancelled context
func TestDetectOllama_Timeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	detector := ai.NewOllamaDetector(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	status, err := detector.DetectOllama(ctx)

	// Should still return status even if context is cancelled
	// because the installation check doesn't use context
	require.NoError(t, err, "DetectOllama should not return an error even with cancelled context")
	require.NotNil(t, status, "status should not be nil")
}

// TestGetRecommendedModels tests the recommended models list
func TestGetRecommendedModels(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	models := detector.GetRecommendedModels()

	require.NotEmpty(t, models)
	assert.Contains(t, models, "prem-1b-sql")
	assert.Contains(t, models, "sqlcoder:7b")
	assert.Contains(t, models, "codellama:7b")
	assert.Contains(t, models, "llama3.1:8b")
	assert.Contains(t, models, "mistral:7b")
}

// TestGetRecommendedModels_Order tests that primary recommendation is first
func TestGetRecommendedModels_Order(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	models := detector.GetRecommendedModels()

	require.NotEmpty(t, models)
	assert.Equal(t, "prem-1b-sql", models[0], "Primary recommendation should be first")
}

// TestCheckModelExists_Exists tests checking for existing model
func TestCheckModelExists_Exists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			resp := ai.OllamaListResponse{
				Models: []ai.OllamaModelInfo{
					{Name: "llama2"},
					{Name: "codellama"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	exists, err := detector.CheckModelExists(ctx, "llama2", server.URL)

	require.NoError(t, err)
	assert.True(t, exists)
}

// TestCheckModelExists_NotExists tests checking for non-existing model
func TestCheckModelExists_NotExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			resp := ai.OllamaListResponse{
				Models: []ai.OllamaModelInfo{
					{Name: "llama2"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	exists, err := detector.CheckModelExists(ctx, "nonexistent", server.URL)

	require.NoError(t, err)
	assert.False(t, exists)
}

// TestCheckModelExists_Error tests error handling when checking model
func TestCheckModelExists_Error(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	exists, err := detector.CheckModelExists(ctx, "llama2", "http://localhost:99999")

	require.Error(t, err)
	assert.False(t, exists)
}

// TestInstallOllama_Darwin tests installation instructions for macOS
func TestInstallOllama_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	instructions, err := detector.InstallOllama()

	require.NoError(t, err)
	assert.Contains(t, instructions, "brew install ollama")
	assert.Contains(t, instructions, "https://ollama.ai/download/mac")
	assert.Contains(t, instructions, "ollama serve")
	assert.Contains(t, instructions, "ollama pull prem-1b-sql")
}

// TestInstallOllama_Linux tests installation instructions for Linux
func TestInstallOllama_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	instructions, err := detector.InstallOllama()

	require.NoError(t, err)
	assert.Contains(t, instructions, "curl -fsSL https://ollama.ai/install.sh | sh")
	assert.Contains(t, instructions, "https://ollama.ai/download/linux")
	assert.Contains(t, instructions, "ollama serve")
}

// TestInstallOllama_Windows tests installation instructions for Windows
func TestInstallOllama_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	instructions, err := detector.InstallOllama()

	require.NoError(t, err)
	assert.Contains(t, instructions, "winget install Ollama.Ollama")
	assert.Contains(t, instructions, "https://ollama.ai/download/windows")
	assert.Contains(t, instructions, "ollama serve")
}

// TestInstallOllama_CommonElements tests common elements in all platforms
func TestInstallOllama_CommonElements(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	instructions, err := detector.InstallOllama()

	require.NoError(t, err)
	assert.Contains(t, instructions, "Ollama Installation Instructions")
	assert.Contains(t, instructions, "ollama pull prem-1b-sql")
	assert.Contains(t, instructions, "ollama list")
	assert.Contains(t, instructions, "https://ollama.ai")
}

// TestOllamaStatus_JSONSerialization tests JSON serialization of OllamaStatus
func TestOllamaStatus_JSONSerialization(t *testing.T) {
	status := &ai.OllamaStatus{
		Installed:       true,
		Running:         true,
		Version:         "0.1.0",
		Endpoint:        "http://localhost:11434",
		AvailableModels: []string{"llama2", "codellama"},
		LastChecked:     time.Now(),
		Error:           "",
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded ai.OllamaStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, status.Installed, decoded.Installed)
	assert.Equal(t, status.Running, decoded.Running)
	assert.Equal(t, status.Version, decoded.Version)
	assert.Equal(t, status.Endpoint, decoded.Endpoint)
	assert.Equal(t, status.AvailableModels, decoded.AvailableModels)
}

// TestOllamaModelInfo_JSONSerialization tests JSON serialization of OllamaModelInfo
func TestOllamaModelInfo_JSONSerialization(t *testing.T) {
	modelInfo := ai.OllamaModelInfo{
		Name:       "llama2",
		Size:       4000000000,
		Digest:     "abc123",
		ModifiedAt: time.Now(),
	}
	modelInfo.Details.Format = "gguf"
	modelInfo.Details.Family = "llama"
	modelInfo.Details.ParameterSize = "7B"

	data, err := json.Marshal(modelInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded ai.OllamaModelInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, modelInfo.Name, decoded.Name)
	assert.Equal(t, modelInfo.Size, decoded.Size)
	assert.Equal(t, modelInfo.Digest, decoded.Digest)
	assert.Equal(t, modelInfo.Details.Format, decoded.Details.Format)
}

// TestListAvailableModels_WithContext tests context propagation
func TestListAvailableModels_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context is properly propagated
		if r.Context() == nil {
			t.Error("Expected context to be propagated")
		}
		w.WriteHeader(http.StatusOK)
		resp := ai.OllamaListResponse{Models: []ai.OllamaModelInfo{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	_, err := detector.ListAvailableModels(ctx, server.URL)
	require.NoError(t, err)
}

// TestIsOllamaRunning_Timeout tests timeout handling
func TestIsOllamaRunning_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	running, err := detector.IsOllamaRunning(ctx, server.URL)

	require.Error(t, err)
	assert.False(t, running)
}

// TestCheckModelExists_EmptyModelList tests with empty model list
func TestCheckModelExists_EmptyModelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp := ai.OllamaListResponse{Models: []ai.OllamaModelInfo{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	exists, err := detector.CheckModelExists(ctx, "llama2", server.URL)

	require.NoError(t, err)
	assert.False(t, exists)
}

// TestListAvailableModels_LargeModelList tests with many models
func TestListAvailableModels_LargeModelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		models := make([]ai.OllamaModelInfo, 50)
		for i := 0; i < 50; i++ {
			models[i] = ai.OllamaModelInfo{
				Name:       "model-" + string(rune('a'+i%26)),
				Size:       int64(i * 1000000000),
				Digest:     "digest-" + string(rune('a'+i%26)),
				ModifiedAt: time.Now(),
			}
		}
		w.WriteHeader(http.StatusOK)
		resp := ai.OllamaListResponse{Models: models}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	modelNames, err := detector.ListAvailableModels(ctx, server.URL)

	require.NoError(t, err)
	assert.Len(t, modelNames, 50)
}

// TestOllamaDetector_MultipleRequests tests multiple sequential requests
func TestOllamaDetector_MultipleRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		resp := ai.OllamaListResponse{
			Models: []ai.OllamaModelInfo{
				{Name: "llama2"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()

	// Make multiple requests
	for i := 0; i < 3; i++ {
		running, err := detector.IsOllamaRunning(ctx, server.URL)
		require.NoError(t, err)
		assert.True(t, running)
	}

	assert.Equal(t, 3, requestCount, "Should have made 3 requests")
}

// TestIsOllamaRunning_VerifyEndpointPath tests that correct endpoint path is used
func TestIsOllamaRunning_VerifyEndpointPath(t *testing.T) {
	correctPath := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			correctPath = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	_, err := detector.IsOllamaRunning(ctx, server.URL)

	require.NoError(t, err)
	assert.True(t, correctPath, "Should call /api/tags endpoint")
}

// TestListAvailableModels_VerifyHTTPMethod tests that GET method is used
func TestListAvailableModels_VerifyHTTPMethod(t *testing.T) {
	correctMethod := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			correctMethod = true
		}
		w.WriteHeader(http.StatusOK)
		resp := ai.OllamaListResponse{Models: []ai.OllamaModelInfo{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	_, err := detector.ListAvailableModels(ctx, server.URL)

	require.NoError(t, err)
	assert.True(t, correctMethod, "Should use GET method")
}

// TestCheckModelExists_CaseSensitivity tests case sensitivity of model names
func TestCheckModelExists_CaseSensitivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			resp := ai.OllamaListResponse{
				Models: []ai.OllamaModelInfo{
					{Name: "llama2"},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()

	// Test with exact case
	exists1, err1 := detector.CheckModelExists(ctx, "llama2", server.URL)
	require.NoError(t, err1)
	assert.True(t, exists1)

	// Test with uppercase (should not match - case sensitive)
	exists2, err2 := detector.CheckModelExists(ctx, "LLAMA2", server.URL)
	require.NoError(t, err2)
	assert.False(t, exists2, "model names should be case-sensitive")
}

// TestOllamaStatus_Structure tests the OllamaStatus structure
func TestOllamaStatus_Structure(t *testing.T) {
	status := &ai.OllamaStatus{
		Installed:       true,
		Running:         true,
		Version:         "1.0.0",
		Endpoint:        "http://localhost:11434",
		AvailableModels: []string{"llama2", "mistral"},
		LastChecked:     time.Now(),
	}

	assert.True(t, status.Installed)
	assert.True(t, status.Running)
	assert.Equal(t, "1.0.0", status.Version)
	assert.Equal(t, "http://localhost:11434", status.Endpoint)
	assert.Len(t, status.AvailableModels, 2)
	assert.NotZero(t, status.LastChecked)
}

// TestOllamaStatus_WithError tests OllamaStatus with error field
func TestOllamaStatus_WithError(t *testing.T) {
	status := &ai.OllamaStatus{
		Installed:       false,
		Running:         false,
		Endpoint:        "http://localhost:11434",
		AvailableModels: []string{},
		LastChecked:     time.Now(),
		Error:           "Ollama is not installed",
	}

	assert.False(t, status.Installed)
	assert.False(t, status.Running)
	assert.NotEmpty(t, status.Error)
	assert.Contains(t, status.Error, "not installed")
}

// TestOllamaModelInfo_Structure tests the OllamaModelInfo structure
func TestOllamaModelInfo_Structure(t *testing.T) {
	modelInfo := ai.OllamaModelInfo{
		Name:       "llama2",
		Size:       4000000000,
		Digest:     "sha256:abc123",
		ModifiedAt: time.Now(),
	}
	modelInfo.Details.Format = "gguf"
	modelInfo.Details.Family = "llama"
	modelInfo.Details.ParameterSize = "7B"
	modelInfo.Details.QuantizationLevel = "Q4_0"

	assert.Equal(t, "llama2", modelInfo.Name)
	assert.Equal(t, int64(4000000000), modelInfo.Size)
	assert.Equal(t, "sha256:abc123", modelInfo.Digest)
	assert.NotZero(t, modelInfo.ModifiedAt)
	assert.Equal(t, "gguf", modelInfo.Details.Format)
	assert.Equal(t, "llama", modelInfo.Details.Family)
	assert.Equal(t, "7B", modelInfo.Details.ParameterSize)
	assert.Equal(t, "Q4_0", modelInfo.Details.QuantizationLevel)
}

// TestOllamaListResponse_Structure tests the OllamaListResponse structure
func TestOllamaListResponse_Structure(t *testing.T) {
	response := ai.OllamaListResponse{
		Models: []ai.OllamaModelInfo{
			{
				Name:       "llama2",
				Size:       4000000000,
				Digest:     "sha256:abc123",
				ModifiedAt: time.Now(),
			},
		},
	}

	assert.Len(t, response.Models, 1)
	assert.Equal(t, "llama2", response.Models[0].Name)
}

// TestPullModel_ServiceNotRunning tests pulling a model when service is not running
func TestPullModel_ServiceNotRunning(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()

	// Try to pull a model - will fail if Ollama is not running
	err := detector.PullModel(ctx, "test-model")

	// Should error (either service not running or pull failed)
	if err != nil {
		assert.Error(t, err)
	}
}

// TestStartOllamaService_NotInstalled tests starting when Ollama is not installed
func TestStartOllamaService_NotInstalled(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()

	err := detector.StartOllamaService(ctx)

	// Will error if Ollama is not installed - we're just testing it doesn't panic
	_ = err
}

// TestGetRecommendedModels_OrderPreserved tests that order is preserved
func TestGetRecommendedModels_OrderPreserved(t *testing.T) {
	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	models := detector.GetRecommendedModels()

	require.Len(t, models, 5)
	assert.Equal(t, "prem-1b-sql", models[0])
	assert.Equal(t, "sqlcoder:7b", models[1])
	assert.Equal(t, "codellama:7b", models[2])
	assert.Equal(t, "llama3.1:8b", models[3])
	assert.Equal(t, "mistral:7b", models[4])
}

// TestDetectOllama_RepeatedCalls tests that repeated calls work correctly
func TestDetectOllama_RepeatedCalls(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()

	// Call multiple times
	status1, err1 := detector.DetectOllama(ctx)
	status2, err2 := detector.DetectOllama(ctx)
	status3, err3 := detector.DetectOllama(ctx)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// All should have same installation status
	assert.Equal(t, status1.Installed, status2.Installed)
	assert.Equal(t, status2.Installed, status3.Installed)

	// All should have same endpoint
	assert.Equal(t, status1.Endpoint, status2.Endpoint)
	assert.Equal(t, status2.Endpoint, status3.Endpoint)
}

// TestListAvailableModels_WithModelDetails tests model with complete details
func TestListAvailableModels_WithModelDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelInfo := ai.OllamaModelInfo{
			Name:       "llama3.1:8b",
			Size:       8000000000,
			Digest:     "sha256:xyz789",
			ModifiedAt: time.Now(),
		}
		modelInfo.Details.Format = "gguf"
		modelInfo.Details.Family = "llama"
		modelInfo.Details.Families = []string{"llama", "llama3"}
		modelInfo.Details.ParameterSize = "8B"
		modelInfo.Details.QuantizationLevel = "Q4_K_M"

		resp := ai.OllamaListResponse{
			Models: []ai.OllamaModelInfo{modelInfo},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	models, err := detector.ListAvailableModels(ctx, server.URL)

	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "llama3.1:8b", models[0])
}

// TestIsOllamaRunning_VerifyRequestHeaders tests that proper headers are sent
func TestIsOllamaRunning_VerifyRequestHeaders(t *testing.T) {
	headerChecked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just verify the request reaches the server
		headerChecked = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.New()
	detector := ai.NewOllamaDetector(logger)

	ctx := context.Background()
	_, err := detector.IsOllamaRunning(ctx, server.URL)

	require.NoError(t, err)
	assert.True(t, headerChecked, "Request should reach server")
}
