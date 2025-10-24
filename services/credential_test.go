package services

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/zalando/go-keyring"
)

func TestCredentialService(t *testing.T) {
	// Use mock keyring for testing
	keyring.MockInit()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during testing

	service := NewCredentialService(logger)

	t.Run("StorePassword", func(t *testing.T) {
		err := service.StorePassword("test-connection-1", "test-password-123")
		if err != nil {
			t.Fatalf("StorePassword failed: %v", err)
		}
	})

	t.Run("GetPassword", func(t *testing.T) {
		// Store a password first
		err := service.StorePassword("test-connection-2", "test-password-456")
		if err != nil {
			t.Fatalf("StorePassword failed: %v", err)
		}

		// Retrieve it
		password, err := service.GetPassword("test-connection-2")
		if err != nil {
			t.Fatalf("GetPassword failed: %v", err)
		}

		if password != "test-password-456" {
			t.Fatalf("Expected password 'test-password-456', got '%s'", password)
		}
	})

	t.Run("GetPassword_NotFound", func(t *testing.T) {
		_, err := service.GetPassword("non-existent-connection")
		if err == nil {
			t.Fatal("Expected error for non-existent connection, got nil")
		}
	})

	t.Run("DeletePassword", func(t *testing.T) {
		// Store a password first
		err := service.StorePassword("test-connection-3", "test-password-789")
		if err != nil {
			t.Fatalf("StorePassword failed: %v", err)
		}

		// Delete it
		err = service.DeletePassword("test-connection-3")
		if err != nil {
			t.Fatalf("DeletePassword failed: %v", err)
		}

		// Verify it's gone
		_, err = service.GetPassword("test-connection-3")
		if err == nil {
			t.Fatal("Expected error after deletion, got nil")
		}
	})

	t.Run("DeletePassword_NotFound", func(t *testing.T) {
		// Deleting non-existent password should not error
		err := service.DeletePassword("non-existent-connection")
		if err != nil {
			t.Fatalf("DeletePassword for non-existent connection should not error: %v", err)
		}
	})

	t.Run("HasPassword", func(t *testing.T) {
		// Store a password
		err := service.StorePassword("test-connection-4", "test-password-abc")
		if err != nil {
			t.Fatalf("StorePassword failed: %v", err)
		}

		// Check it exists
		if !service.HasPassword("test-connection-4") {
			t.Fatal("HasPassword should return true for existing password")
		}

		// Check non-existent
		if service.HasPassword("non-existent-connection") {
			t.Fatal("HasPassword should return false for non-existent password")
		}
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		// Store initial password
		err := service.StorePassword("test-connection-5", "initial-password")
		if err != nil {
			t.Fatalf("StorePassword failed: %v", err)
		}

		// Update it
		err = service.UpdatePassword("test-connection-5", "updated-password")
		if err != nil {
			t.Fatalf("UpdatePassword failed: %v", err)
		}

		// Verify update
		password, err := service.GetPassword("test-connection-5")
		if err != nil {
			t.Fatalf("GetPassword failed: %v", err)
		}

		if password != "updated-password" {
			t.Fatalf("Expected password 'updated-password', got '%s'", password)
		}
	})

	t.Run("StorePassword_EmptyConnectionID", func(t *testing.T) {
		err := service.StorePassword("", "test-password")
		if err == nil {
			t.Fatal("Expected error for empty connection ID, got nil")
		}
	})

	t.Run("StorePassword_EmptyPassword", func(t *testing.T) {
		err := service.StorePassword("test-connection", "")
		if err == nil {
			t.Fatal("Expected error for empty password, got nil")
		}
	})

	t.Run("GetPassword_EmptyConnectionID", func(t *testing.T) {
		_, err := service.GetPassword("")
		if err == nil {
			t.Fatal("Expected error for empty connection ID, got nil")
		}
	})

	t.Run("DeletePassword_EmptyConnectionID", func(t *testing.T) {
		err := service.DeletePassword("")
		if err == nil {
			t.Fatal("Expected error for empty connection ID, got nil")
		}
	})

	t.Run("GetPlatformInfo", func(t *testing.T) {
		info := service.GetPlatformInfo()

		if _, ok := info["platform"]; !ok {
			t.Error("Expected platform field in info")
		}
		if _, ok := info["service"]; !ok {
			t.Error("Expected service field in info")
		}
		if _, ok := info["backend"]; !ok {
			t.Error("Expected backend field in info")
		}
		if _, ok := info["supported"]; !ok {
			t.Error("Expected supported field in info")
		}

		if info["service"] != ServiceName {
			t.Errorf("Expected service name %q, got %q", ServiceName, info["service"])
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		err := service.HealthCheck()
		if err != nil {
			// Health check might fail in CI environments, which is okay
			t.Logf("Health check failed (expected in CI): %v", err)
		}

		// Verify cleanup - health check credential should be removed
		if service.HasPassword("__sqlstudio_health_check__") {
			t.Error("Expected health check credential to be cleaned up")
		}
	})
}
