package services_test

import (
	"errors"
	"fmt"
	"log"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/sql-studio/services"
)

// ExampleCredentialService_basic demonstrates basic usage of the credential service
func ExampleCredentialService_basic() {
	// Create a credential service
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Quiet for example
	credService := services.NewCredentialService(logger)

	// Store a password
	connectionID := "my-postgres-db"
	password := "super-secret-password"

	err := credService.StorePassword(connectionID, password)
	if err != nil {
		log.Fatalf("Failed to store password: %v", err)
	}
	fmt.Println("Password stored successfully")

	// Retrieve the password
	retrievedPassword, err := credService.GetPassword(connectionID)
	if err != nil {
		log.Fatalf("Failed to get password: %v", err)
	}

	// Verify it matches
	if retrievedPassword == password {
		fmt.Println("Password retrieved successfully")
	}

	// Clean up
	_ = credService.DeletePassword(connectionID)
}

// ExampleCredentialService_errorHandling demonstrates proper error handling
func ExampleCredentialService_errorHandling() {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs for example
	credService := services.NewCredentialService(logger)

	// Try to get a non-existent password
	_, err := credService.GetPassword("non-existent-connection")
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			fmt.Println("Password not found - need to prompt user")

		case errors.Is(err, services.ErrPermissionDenied):
			fmt.Println("Access denied - check keychain permissions")

		case errors.Is(err, services.ErrUnavailable):
			fmt.Println("Keychain unavailable - use fallback storage")

		default:
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	// Output:
	// Password not found - need to prompt user
}

// ExampleCredentialService_platformInfo demonstrates getting platform information
func ExampleCredentialService_platformInfo() {
	logger := logrus.New()
	credService := services.NewCredentialService(logger)

	info := credService.GetPlatformInfo()

	fmt.Printf("Platform: %s\n", info["platform"])
	fmt.Printf("Supported: %v\n", info["supported"])

	// Note: Actual output depends on the platform
}

// ExampleCredentialService_healthCheck demonstrates health checking
func ExampleCredentialService_healthCheck() {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	credService := services.NewCredentialService(logger)

	// Check if keychain is available
	if err := credService.HealthCheck(); err != nil {
		fmt.Println("Keychain not available - using fallback")
		// Implement fallback credential storage
	} else {
		fmt.Println("Keychain available - using OS keychain")
	}
}

// ExampleCredentialService_lifecycle demonstrates complete credential lifecycle
func ExampleCredentialService_lifecycle() {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	credService := services.NewCredentialService(logger)

	connectionID := "example-connection"
	originalPassword := "original-password"
	newPassword := "updated-password"

	// 1. Store initial password
	if err := credService.StorePassword(connectionID, originalPassword); err != nil {
		log.Fatal(err)
	}
	fmt.Println("1. Password stored")

	// 2. Check if password exists
	if credService.HasPassword(connectionID) {
		fmt.Println("2. Password exists")
	}

	// 3. Update password
	if err := credService.UpdatePassword(connectionID, newPassword); err != nil {
		log.Fatal(err)
	}
	fmt.Println("3. Password updated")

	// 4. Verify new password
	retrieved, err := credService.GetPassword(connectionID)
	if err != nil {
		log.Fatal(err)
	}
	if retrieved == newPassword {
		fmt.Println("4. New password verified")
	}

	// 5. Delete password
	if err := credService.DeletePassword(connectionID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("5. Password deleted")

	// 6. Verify deletion (idempotent)
	if err := credService.DeletePassword(connectionID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("6. Deletion is idempotent")

	// Output:
	// 1. Password stored
	// 2. Password exists
	// 3. Password updated
	// 4. New password verified
	// 5. Password deleted
	// 6. Deletion is idempotent
}
